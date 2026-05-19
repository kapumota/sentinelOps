package auth

import (
	"net"
	"strings"
	"sync"
	"time"
)

type Clock func() time.Time

type RateLimitConfig struct {
	Enabled     bool
	MaxFailures int
	Window      time.Duration
	Lockout     time.Duration
	Now         Clock
}

type RateLimitDecision struct {
	Allowed    bool
	RetryAfter time.Duration
	Failures   int
}

type rateEntry struct {
	failures     []time.Time
	blockedUntil time.Time
}

type RateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rateEntry
	cfg     RateLimitConfig
	now     Clock
}

func NewRateLimiter(cfg RateLimitConfig) *RateLimiter {
	if cfg.MaxFailures <= 0 {
		cfg.MaxFailures = 5
	}
	if cfg.Window <= 0 {
		cfg.Window = time.Minute
	}
	if cfg.Lockout <= 0 {
		cfg.Lockout = time.Minute
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &RateLimiter{
		entries: make(map[string]*rateEntry),
		cfg:     cfg,
		now:     cfg.Now,
	}
}

func Key(remoteAddr, username string) string {
	host := remoteAddr
	if h, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = h
	}
	return strings.ToLower(strings.TrimSpace(host)) + "|" + strings.ToLower(strings.TrimSpace(username))
}

func (r *RateLimiter) Allow(key string) RateLimitDecision {
	if r == nil || !r.cfg.Enabled {
		return RateLimitDecision{Allowed: true}
	}

	now := r.now()
	r.mu.Lock()
	defer r.mu.Unlock()

	entry := r.entries[key]
	if entry == nil {
		return RateLimitDecision{Allowed: true}
	}

	entry.failures = keepRecent(entry.failures, now.Add(-r.cfg.Window))
	if !entry.blockedUntil.IsZero() && now.Before(entry.blockedUntil) {
		return RateLimitDecision{
			Allowed:    false,
			RetryAfter: entry.blockedUntil.Sub(now),
			Failures:   len(entry.failures),
		}
	}
	if len(entry.failures) == 0 {
		delete(r.entries, key)
	}
	return RateLimitDecision{Allowed: true, Failures: len(entry.failures)}
}

func (r *RateLimiter) RecordFailure(key string) RateLimitDecision {
	if r == nil || !r.cfg.Enabled {
		return RateLimitDecision{Allowed: true}
	}

	now := r.now()
	r.mu.Lock()
	defer r.mu.Unlock()

	entry := r.entries[key]
	if entry == nil {
		entry = &rateEntry{}
		r.entries[key] = entry
	}
	entry.failures = append(keepRecent(entry.failures, now.Add(-r.cfg.Window)), now)
	if len(entry.failures) >= r.cfg.MaxFailures {
		entry.blockedUntil = now.Add(r.cfg.Lockout)
		return RateLimitDecision{
			Allowed:    false,
			RetryAfter: r.cfg.Lockout,
			Failures:   len(entry.failures),
		}
	}
	return RateLimitDecision{Allowed: true, Failures: len(entry.failures)}
}

func (r *RateLimiter) RecordSuccess(key string) {
	if r == nil || !r.cfg.Enabled {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.entries, key)
}

func keepRecent(values []time.Time, cutoff time.Time) []time.Time {
	out := values[:0]
	for _, v := range values {
		if !v.Before(cutoff) {
			out = append(out, v)
		}
	}
	return out
}
