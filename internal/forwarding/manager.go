package forwarding

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type Tunnel struct {
	ID        string    `json:"id"`
	SessionID string    `json:"session_id"`
	Username  string    `json:"username"`
	Direction string    `json:"direction"`
	Target    string    `json:"target"`
	Bind      string    `json:"bind"`
	Origin    string    `json:"origin"`
	StartedAt time.Time `json:"started_at"`
}

type Reader interface{ Snapshot() []Tunnel }

type Controller interface {
	Reader
	SnapshotByUsername(username string) []Tunnel
	Get(id string) (Tunnel, bool)
	Close(id string) bool
	Count() int
}

type Hooks struct {
	OnOpen  func(Tunnel)
	OnClose func(Tunnel)
}

type managedTunnel struct {
	Tunnel
	stop func()
}

type Manager struct {
	mu      sync.RWMutex
	seq     uint64
	tunnels map[string]managedTunnel
	hooks   Hooks
}

func NewManager(hooks Hooks) *Manager {
	return &Manager{
		tunnels: make(map[string]managedTunnel),
		hooks:   hooks,
	}
}

func (m *Manager) OpenLocal(sessionID, username, target, origin string, stop func()) Tunnel {
	return m.open("local", sessionID, username, target, "", origin, stop)
}

func (m *Manager) OpenRemote(sessionID, username, bind, origin string, stop func()) Tunnel {
	return m.open("remote", sessionID, username, "", bind, origin, stop)
}

func (m *Manager) open(direction, sessionID, username, target, bind, origin string, stop func()) Tunnel {
	m.mu.Lock()
	m.seq++
	id := fmt.Sprintf("tun-%06d", m.seq)

	t := Tunnel{
		ID:        id,
		SessionID: sessionID,
		Username:  username,
		Direction: direction,
		Target:    target,
		Bind:      bind,
		Origin:    origin,
		StartedAt: time.Now().UTC(),
	}

	m.tunnels[id] = managedTunnel{
		Tunnel: t,
		stop:   stop,
	}
	hook := m.hooks.OnOpen
	m.mu.Unlock()

	if hook != nil {
		hook(t)
	}

	return t
}

func (m *Manager) Get(id string) (Tunnel, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	item, ok := m.tunnels[id]
	if !ok {
		return Tunnel{}, false
	}

	return item.Tunnel, true
}

func (m *Manager) Close(id string) bool {
	m.mu.Lock()
	item, ok := m.tunnels[id]
	if ok {
		delete(m.tunnels, id)
	}
	m.mu.Unlock()

	if !ok {
		return false
	}

	if item.stop != nil {
		item.stop()
	}

	if m.hooks.OnClose != nil {
		m.hooks.OnClose(item.Tunnel)
	}

	return true
}

func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.tunnels)
}

func (m *Manager) Snapshot() []Tunnel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make([]Tunnel, 0, len(m.tunnels))
	for _, item := range m.tunnels {
		out = append(out, item.Tunnel)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].StartedAt.Before(out[j].StartedAt)
	})

	return out
}

func (m *Manager) SnapshotByUsername(username string) []Tunnel {
	user := strings.ToLower(strings.TrimSpace(username))
	items := m.Snapshot()

	out := make([]Tunnel, 0, len(items))
	for _, t := range items {
		if strings.ToLower(strings.TrimSpace(t.Username)) == user {
			out = append(out, t)
		}
	}

	return out
}
