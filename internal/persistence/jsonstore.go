package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"sentinelops/internal/forwarding"
	"sentinelops/internal/session"
)

type SnapshotFile[T any] struct {
	Version   string    `json:"version"`
	Kind      string    `json:"kind"`
	Generated time.Time `json:"generated_at"`
	Items     []T       `json:"items"`
}

type Store struct {
	enabled      bool
	sessionsPath string
	tunnelsPath  string
	now          func() time.Time
}

type Options struct {
	Enabled      bool
	SessionsPath string
	TunnelsPath  string
	Now          func() time.Time
}

func NewStore(opts Options) *Store {
	if opts.Now == nil {
		opts.Now = func() time.Time { return time.Now().UTC() }
	}
	return &Store{
		enabled:      opts.Enabled,
		sessionsPath: opts.SessionsPath,
		tunnelsPath:  opts.TunnelsPath,
		now:          opts.Now,
	}
}

func (s *Store) Enabled() bool { return s != nil && s.enabled }
func (s *Store) SessionsPath() string {
	if s == nil {
		return ""
	}
	return s.sessionsPath
}
func (s *Store) TunnelsPath() string {
	if s == nil {
		return ""
	}
	return s.tunnelsPath
}

func (s *Store) SaveSessions(items []session.Snapshot) error {
	if !s.Enabled() {
		return nil
	}
	return writeSnapshot(s.sessionsPath, SnapshotFile[session.Snapshot]{
		Version:   "2.4.1",
		Kind:      "sessions",
		Generated: s.now(),
		Items:     items,
	})
}

func (s *Store) SaveTunnels(items []forwarding.Tunnel) error {
	if !s.Enabled() {
		return nil
	}
	return writeSnapshot(s.tunnelsPath, SnapshotFile[forwarding.Tunnel]{
		Version:   "2.4.1",
		Kind:      "tunnels",
		Generated: s.now(),
		Items:     items,
	})
}

func LoadSessions(path string) (SnapshotFile[session.Snapshot], error) {
	return loadSnapshot[session.Snapshot](path)
}

func LoadTunnels(path string) (SnapshotFile[forwarding.Tunnel], error) {
	return loadSnapshot[forwarding.Tunnel](path)
}

func writeSnapshot[T any](path string, snapshot SnapshotFile[T]) error {
	if path == "" {
		return fmt.Errorf("ruta de persistencia vacía")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func loadSnapshot[T any](path string) (SnapshotFile[T], error) {
	var out SnapshotFile[T]
	data, err := os.ReadFile(path)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, err
	}
	return out, nil
}
