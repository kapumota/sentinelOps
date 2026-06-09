//go:build containers

package forwarding

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestForwardingManagerIntegration(t *testing.T) {
	t.Run("abrir_listar_y_cerrar_tunel_local", func(t *testing.T) {
		var opened atomic.Int32
		var closed atomic.Int32
		manager := NewManager(Hooks{
			OnOpen: func(Tunnel) {
				opened.Add(1)
			},
			OnClose: func(Tunnel) {
				closed.Add(1)
			},
		})

		stopCalled := make(chan struct{}, 1)
		tunnel := manager.OpenLocal("sess-1", "student", "127.0.0.1:9001", "127.0.0.1:50000", func() {
			stopCalled <- struct{}{}
		})

		if tunnel.Direction != "local" {
			t.Fatalf("dirección incorrecta: %s", tunnel.Direction)
		}
		if manager.Count() != 1 {
			t.Fatalf("se esperaba un túnel activo")
		}
		if opened.Load() != 1 {
			t.Fatalf("se esperaba un evento de apertura")
		}

		if !manager.Close(tunnel.ID) {
			t.Fatalf("el túnel debería cerrarse")
		}
		select {
		case <-stopCalled:
		case <-time.After(time.Second):
			t.Fatal("la función de cierre del túnel no fue llamada")
		}
		if closed.Load() != 1 {
			t.Fatalf("se esperaba un evento de cierre")
		}
		if manager.Count() != 0 {
			t.Fatalf("no deberían quedar túneles activos")
		}
	})

	t.Run("snapshot_por_usuario", func(t *testing.T) {
		manager := NewManager(Hooks{})
		manager.OpenLocal("sess-1", "student", "127.0.0.1:9001", "127.0.0.1:51000", nil)
		manager.OpenLocal("sess-2", "teacher", "127.0.0.1:9001", "127.0.0.1:51001", nil)
		manager.OpenRemote("sess-3", "student", "127.0.0.1:10080", "127.0.0.1:51002", nil)

		studentTunnels := manager.SnapshotByUsername(" STUDENT ")
		if len(studentTunnels) != 2 {
			t.Fatalf("se esperaban dos túneles de student, got %d", len(studentTunnels))
		}
	})

	t.Run("operaciones_concurrentes", func(t *testing.T) {
		manager := NewManager(Hooks{})
		var wg sync.WaitGroup

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				manager.OpenLocal("sess-concurrente", "student", "127.0.0.1:9001", fmt.Sprintf("127.0.0.1:%d", 52000+index), nil)
			}(i)
		}

		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = manager.Snapshot()
				_ = manager.Count()
			}()
		}

		wg.Wait()
		if got := manager.Count(); got != 100 {
			t.Fatalf("se esperaban 100 túneles, got %d", got)
		}
	})
}

func TestForwardingPolicyIntegration(t *testing.T) {
	policy := NewPolicy(
		true,
		"127.0.0.1:9001,localhost:9001",
		"student,teacher,admin",
		false,
		"127.0.0.1:10080",
		"admin",
	)

	if !policy.AllowLocalTarget("student", "127.0.0.1", 9001) {
		t.Fatal("student debería poder abrir forwarding local hacia 127.0.0.1:9001")
	}
	if policy.AllowLocalTarget("auditor", "127.0.0.1", 9001) {
		t.Fatal("auditor no debería estar permitido para forwarding local")
	}
	if policy.AllowLocalTarget("student", "evil.example", 9001) {
		t.Fatal("un destino fuera de la allowlist debería ser rechazado")
	}
	if policy.AllowRemoteBind("admin", "127.0.0.1", 10080) {
		t.Fatal("forwarding remoto debería estar deshabilitado")
	}
}
