package store

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MemoryStore struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	tunnels  map[string]*Tunnel
	auditLog []*AuditEntry
	attempts map[string]*attemptEntry
}

type attemptEntry struct {
	count       int
	firstAt     time.Time
	lockedUntil *time.Time
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		sessions: make(map[string]*Session),
		tunnels:  make(map[string]*Tunnel),
		auditLog: make([]*AuditEntry, 0),
		attempts: make(map[string]*attemptEntry),
	}
}

func (m *MemoryStore) CreateSession(_ context.Context, sess *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if sess.ID == "" {
		sess.ID = uuid.New().String()
	}
	if _, exists := m.sessions[sess.ID]; exists {
		return ErrAlreadyExists
	}
	now := time.Now().UTC()
	sess.CreatedAt = now
	sess.LastActive = now
	if sess.Status == "" {
		sess.Status = "active"
	}
	m.sessions[sess.ID] = cloneSession(sess)
	return nil
}

func (m *MemoryStore) GetSession(_ context.Context, id string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sess, ok := m.sessions[id]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneSession(sess), nil
}

func (m *MemoryStore) UpdateSession(_ context.Context, sess *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.sessions[sess.ID]; !ok {
		return ErrNotFound
	}
	sess.LastActive = time.Now().UTC()
	m.sessions[sess.ID] = cloneSession(sess)
	return nil
}

func (m *MemoryStore) DeleteSession(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.sessions[id]; !ok {
		return ErrNotFound
	}
	delete(m.sessions, id)
	for tid, tunnel := range m.tunnels {
		if tunnel.SessionID == id {
			delete(m.tunnels, tid)
		}
	}
	return nil
}

func (m *MemoryStore) ListSessions(_ context.Context, filter SessionFilter) ([]*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Session, 0)
	for _, sess := range m.sessions {
		if filter.Username != "" && sess.Username != filter.Username {
			continue
		}
		if filter.Status != "" && sess.Status != filter.Status {
			continue
		}
		if filter.RemoteAddr != "" && sess.RemoteAddr != filter.RemoteAddr {
			continue
		}
		if filter.Since != nil && sess.CreatedAt.Before(*filter.Since) {
			continue
		}
		result = append(result, cloneSession(sess))
	}
	return applySessionWindow(result, filter.Limit, filter.Offset), nil
}

func (m *MemoryStore) CountActiveSessions(_ context.Context, username string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, sess := range m.sessions {
		if sess.Username == username && sess.Status == "active" {
			count++
		}
	}
	return count, nil
}

func (m *MemoryStore) TouchSession(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sess, ok := m.sessions[id]
	if !ok {
		return ErrNotFound
	}
	sess.LastActive = time.Now().UTC()
	return nil
}

func (m *MemoryStore) CreateTunnel(_ context.Context, tunnel *Tunnel) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if tunnel.ID == "" {
		tunnel.ID = uuid.New().String()
	}
	if _, exists := m.tunnels[tunnel.ID]; exists {
		return ErrAlreadyExists
	}
	tunnel.CreatedAt = time.Now().UTC()
	if tunnel.Status == "" {
		tunnel.Status = "active"
	}
	m.tunnels[tunnel.ID] = cloneTunnel(tunnel)
	return nil
}

func (m *MemoryStore) GetTunnel(_ context.Context, id string) (*Tunnel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tunnel, ok := m.tunnels[id]
	if !ok {
		return nil, ErrNotFound
	}
	return cloneTunnel(tunnel), nil
}

func (m *MemoryStore) UpdateTunnel(_ context.Context, tunnel *Tunnel) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.tunnels[tunnel.ID]; !ok {
		return ErrNotFound
	}
	m.tunnels[tunnel.ID] = cloneTunnel(tunnel)
	return nil
}

func (m *MemoryStore) DeleteTunnel(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.tunnels[id]; !ok {
		return ErrNotFound
	}
	delete(m.tunnels, id)
	return nil
}

func (m *MemoryStore) ListTunnels(_ context.Context, filter TunnelFilter) ([]*Tunnel, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Tunnel, 0)
	for _, tunnel := range m.tunnels {
		if filter.SessionID != "" && tunnel.SessionID != filter.SessionID {
			continue
		}
		if filter.Type != "" && tunnel.Type != filter.Type {
			continue
		}
		if filter.Status != "" && tunnel.Status != filter.Status {
			continue
		}
		result = append(result, cloneTunnel(tunnel))
	}
	return applyTunnelWindow(result, filter.Limit, filter.Offset), nil
}

func (m *MemoryStore) DeleteTunnelsBySession(_ context.Context, sessionID string) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	deleted := make([]string, 0)
	for id, tunnel := range m.tunnels {
		if tunnel.SessionID == sessionID {
			delete(m.tunnels, id)
			deleted = append(deleted, id)
		}
	}
	return deleted, nil
}

func (m *MemoryStore) IncrementAttempts(_ context.Context, key string, window time.Duration) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	entry, exists := m.attempts[key]
	if !exists || now.Sub(entry.firstAt) > window {
		m.attempts[key] = &attemptEntry{count: 1, firstAt: now}
		return 1, nil
	}
	entry.count++
	return entry.count, nil
}

func (m *MemoryStore) GetAttempts(_ context.Context, key string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.attempts[key]
	if !ok {
		return 0, nil
	}
	return entry.count, nil
}

func (m *MemoryStore) ResetAttempts(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.attempts, key)
	return nil
}

func (m *MemoryStore) IsLocked(_ context.Context, key string, _ time.Duration) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.attempts[key]
	if !ok || entry.lockedUntil == nil {
		return false, nil
	}
	return time.Now().UTC().Before(*entry.lockedUntil), nil
}

func (m *MemoryStore) Lock(_ context.Context, key string, duration time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.attempts[key]
	if !ok {
		entry = &attemptEntry{firstAt: time.Now().UTC()}
		m.attempts[key] = entry
	}
	until := time.Now().UTC().Add(duration)
	entry.lockedUntil = &until
	return nil
}

func (m *MemoryStore) AppendAuditLog(_ context.Context, entry *AuditEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	entry.Timestamp = time.Now().UTC()
	m.auditLog = append(m.auditLog, cloneAuditEntry(entry))
	return nil
}

func (m *MemoryStore) QueryAuditLog(_ context.Context, filter AuditFilter) ([]*AuditEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*AuditEntry, 0)
	for _, entry := range m.auditLog {
		if filter.Username != "" && entry.Username != filter.Username {
			continue
		}
		if filter.Action != "" && entry.Action != filter.Action {
			continue
		}
		if filter.Result != "" && entry.Result != filter.Result {
			continue
		}
		if filter.Since != nil && entry.Timestamp.Before(*filter.Since) {
			continue
		}
		if filter.Until != nil && entry.Timestamp.After(*filter.Until) {
			continue
		}
		result = append(result, cloneAuditEntry(entry))
	}
	return applyAuditWindow(result, filter.Limit, filter.Offset), nil
}

func (m *MemoryStore) CleanupInactiveSessions(_ context.Context, maxInactive time.Duration) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().UTC().Add(-maxInactive)
	removed := 0
	now := time.Now().UTC()
	for _, sess := range m.sessions {
		if sess.Status == "active" && sess.LastActive.Before(cutoff) {
			sess.Status = "expired"
			sess.ClosedAt = &now
			removed++
		}
	}
	return removed, nil
}

func (m *MemoryStore) CleanupOldAuditLogs(_ context.Context, maxAge time.Duration) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().UTC().Add(-maxAge)
	kept := make([]*AuditEntry, 0, len(m.auditLog))
	removed := 0
	for _, entry := range m.auditLog {
		if entry.Timestamp.Before(cutoff) {
			removed++
			continue
		}
		kept = append(kept, entry)
	}
	m.auditLog = kept
	return removed, nil
}

func (m *MemoryStore) Health(_ context.Context) error {
	return nil
}

func (m *MemoryStore) Close() error {
	return nil
}

func cloneSession(sess *Session) *Session {
	if sess == nil {
		return nil
	}
	clone := *sess
	clone.Metadata = cloneJSONMap(sess.Metadata)
	return &clone
}

func cloneTunnel(tunnel *Tunnel) *Tunnel {
	if tunnel == nil {
		return nil
	}
	clone := *tunnel
	clone.Metadata = cloneJSONMap(tunnel.Metadata)
	return &clone
}

func cloneAuditEntry(entry *AuditEntry) *AuditEntry {
	if entry == nil {
		return nil
	}
	clone := *entry
	clone.Details = cloneJSONMap(entry.Details)
	return &clone
}

func cloneJSONMap(input JSONMap) JSONMap {
	if input == nil {
		return nil
	}
	output := make(JSONMap, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}

func applySessionWindow(input []*Session, limit, offset int) []*Session {
	if offset >= len(input) {
		return []*Session{}
	}
	if offset < 0 {
		offset = 0
	}
	end := len(input)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return input[offset:end]
}

func applyTunnelWindow(input []*Tunnel, limit, offset int) []*Tunnel {
	if offset >= len(input) {
		return []*Tunnel{}
	}
	if offset < 0 {
		offset = 0
	}
	end := len(input)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return input[offset:end]
}

func applyAuditWindow(input []*AuditEntry, limit, offset int) []*AuditEntry {
	if offset >= len(input) {
		return []*AuditEntry{}
	}
	if offset < 0 {
		offset = 0
	}
	end := len(input)
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	return input[offset:end]
}
