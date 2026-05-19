package auth

import (
	"testing"
	"time"
)

func TestRateLimiterBlocksAfterConfiguredFailures(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	limiter := NewRateLimiter(RateLimitConfig{
		Enabled:     true,
		MaxFailures: 2,
		Window:      time.Minute,
		Lockout:     30 * time.Second,
		Now:         func() time.Time { return now },
	})

	key := Key("127.0.0.1:5555", "student")
	if !limiter.Allow(key).Allowed {
		t.Fatal("first attempt should be allowed")
	}
	if !limiter.RecordFailure(key).Allowed {
		t.Fatal("first failure should not block")
	}
	blocked := limiter.RecordFailure(key)
	if blocked.Allowed {
		t.Fatal("second failure should block")
	}
	if limiter.Allow(key).Allowed {
		t.Fatal("blocked key should not be allowed")
	}

	now = now.Add(31 * time.Second)
	if !limiter.Allow(key).Allowed {
		t.Fatal("key should be allowed after lockout expires")
	}
}

func TestRateLimiterSuccessClearsFailures(t *testing.T) {
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	limiter := NewRateLimiter(RateLimitConfig{
		Enabled:     true,
		MaxFailures: 2,
		Window:      time.Minute,
		Lockout:     30 * time.Second,
		Now:         func() time.Time { return now },
	})

	key := Key("10.0.0.5:5555", "teacher")
	limiter.RecordFailure(key)
	limiter.RecordSuccess(key)
	if !limiter.RecordFailure(key).Allowed {
		t.Fatal("failure counter should be cleared after success")
	}
}
