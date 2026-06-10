package store

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryStoreSessionTunnelAudit(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	sess := &Session{Username: "student", Role: "estudiante", RemoteAddr: "127.0.0.1:12000"}
	if err := store.CreateSession(ctx, sess); err != nil {
		t.Fatalf("crear sesión: %v", err)
	}
	if sess.ID == "" {
		t.Fatal("se esperaba ID de sesión")
	}

	count, err := store.CountActiveSessions(ctx, "student")
	if err != nil {
		t.Fatalf("contar sesiones: %v", err)
	}
	if count != 1 {
		t.Fatalf("sesiones activas inesperadas: %d", count)
	}

	tunnel := &Tunnel{SessionID: sess.ID, Type: "local", LocalAddr: "127.0.0.1:9001", RemoteAddr: "localhost:9001"}
	if err := store.CreateTunnel(ctx, tunnel); err != nil {
		t.Fatalf("crear túnel: %v", err)
	}
	tunnels, err := store.ListTunnels(ctx, TunnelFilter{SessionID: sess.ID})
	if err != nil {
		t.Fatalf("listar túneles: %v", err)
	}
	if len(tunnels) != 1 {
		t.Fatalf("túneles inesperados: %d", len(tunnels))
	}

	entry := &AuditEntry{Action: "login", Username: "student", Result: "success", Resource: "ssh"}
	if err := store.AppendAuditLog(ctx, entry); err != nil {
		t.Fatalf("agregar auditoría: %v", err)
	}
	entries, err := store.QueryAuditLog(ctx, AuditFilter{Username: "student", Action: "login"})
	if err != nil {
		t.Fatalf("consultar auditoría: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entradas de auditoría inesperadas: %d", len(entries))
	}
}

func TestMemoryStoreRateLimit(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	key := "login:student:127.0.0.1"

	for i := 1; i <= 3; i++ {
		got, err := store.IncrementAttempts(ctx, key, time.Minute)
		if err != nil {
			t.Fatalf("incrementar intento: %v", err)
		}
		if got != i {
			t.Fatalf("contador esperado %d, obtenido %d", i, got)
		}
	}

	if err := store.Lock(ctx, key, time.Minute); err != nil {
		t.Fatalf("bloquear llave: %v", err)
	}
	locked, err := store.IsLocked(ctx, key, time.Minute)
	if err != nil {
		t.Fatalf("consultar bloqueo: %v", err)
	}
	if !locked {
		t.Fatal("se esperaba bloqueo activo")
	}

	if err := store.ResetAttempts(ctx, key); err != nil {
		t.Fatalf("reiniciar intentos: %v", err)
	}
	locked, err = store.IsLocked(ctx, key, time.Minute)
	if err != nil {
		t.Fatalf("consultar bloqueo después de reset: %v", err)
	}
	if locked {
		t.Fatal("no se esperaba bloqueo después de reset")
	}
}

func TestMemoryStoreDeleteSessionCascadesTunnels(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()

	sess := &Session{Username: "teacher", Role: "docente"}
	if err := store.CreateSession(ctx, sess); err != nil {
		t.Fatalf("crear sesión: %v", err)
	}
	tunnel := &Tunnel{SessionID: sess.ID, Type: "local"}
	if err := store.CreateTunnel(ctx, tunnel); err != nil {
		t.Fatalf("crear túnel: %v", err)
	}
	if err := store.DeleteSession(ctx, sess.ID); err != nil {
		t.Fatalf("borrar sesión: %v", err)
	}
	if _, err := store.GetTunnel(ctx, tunnel.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("se esperaba ErrNotFound para túnel eliminado, obtuvo %v", err)
	}
}
