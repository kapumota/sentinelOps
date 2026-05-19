package session

import (
	"sort"
	"sync"
	"time"
)

type Snapshot struct {
	ID              string    `json:"id"`
	RemoteAddr      string    `json:"remote_addr"`
	Transport       string    `json:"transport"`
	Authn           string    `json:"authn"`
	ConnectedAt     time.Time `json:"connected_at"`
	AuthenticatedAt time.Time `json:"authenticated_at"`
	Username        string    `json:"username"`
	Role            string    `json:"role"`
	CommandCount    int       `json:"command_count"`
}

type Registry struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	onChange func([]Snapshot)
}

func NewRegistry() *Registry {
	return &Registry{sessions: make(map[string]*Session)}
}

func (r *Registry) SetOnChange(hook func([]Snapshot)) {
	r.mu.Lock()
	r.onChange = hook
	r.mu.Unlock()
}

func (r *Registry) Add(sess *Session) {
	if sess == nil {
		return
	}
	r.mu.Lock()
	r.sessions[sess.ID] = sess
	hook := r.onChange
	r.mu.Unlock()
	r.notify(hook)
}

func (r *Registry) Remove(id string) {
	r.mu.Lock()
	delete(r.sessions, id)
	hook := r.onChange
	r.mu.Unlock()
	r.notify(hook)
}

func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.sessions)
}

func (r *Registry) Snapshot() []Snapshot {
	r.mu.RLock()
	out := make([]Snapshot, 0, len(r.sessions))
	for _, sess := range r.sessions {
		out = append(out, Snapshot{
			ID:              sess.ID,
			RemoteAddr:      sess.RemoteAddr,
			Transport:       sess.Transport,
			Authn:           sess.Authn,
			ConnectedAt:     sess.ConnectedAt,
			AuthenticatedAt: sess.AuthenticatedAt,
			Username:        sess.Username,
			Role:            sess.Role,
			CommandCount:    sess.CommandCount,
		})
	}

	r.mu.RUnlock()
	sortSnapshots(out)
	return out
}

func (r *Registry) notify(hook func([]Snapshot)) {
	if hook == nil {
		return
	}
	hook(r.Snapshot())
}

func sortSnapshots(out []Snapshot) {
	sort.Slice(out, func(i, j int) bool {
		return out[i].ConnectedAt.Before(out[j].ConnectedAt)
	})
}
