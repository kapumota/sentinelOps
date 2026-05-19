package persistence

import (
	"path/filepath"
	"testing"
	"time"

	"sentinelops/internal/session"
)

func TestStoreSavesAndLoadsSessions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")
	now := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	store := NewStore(Options{
		Enabled:      true,
		SessionsPath: path,
		TunnelsPath:  filepath.Join(dir, "tunnels.json"),
		Now:          func() time.Time { return now },
	})

	items := []session.Snapshot{{ID: "sess-000001", Username: "student", Role: "student"}}
	if err := store.SaveSessions(items); err != nil {
		t.Fatalf("SaveSessions failed: %v", err)
	}

	loaded, err := LoadSessions(path)
	if err != nil {
		t.Fatalf("LoadSessions failed: %v", err)
	}
	if loaded.Version != "2.4.1" || loaded.Kind != "sessions" {
		t.Fatalf("unexpected metadata: %#v", loaded)
	}
	if len(loaded.Items) != 1 || loaded.Items[0].ID != "sess-000001" {
		t.Fatalf("unexpected items: %#v", loaded.Items)
	}
}

func TestDisabledStoreDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.json")
	store := NewStore(Options{Enabled: false, SessionsPath: path})
	if err := store.SaveSessions(nil); err != nil {
		t.Fatalf("disabled store should not fail: %v", err)
	}
}
