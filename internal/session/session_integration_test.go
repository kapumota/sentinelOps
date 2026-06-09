//go:build containers

package session

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestRegistryIntegrationConcurrentAccess(t *testing.T) {
	t.Run("agregar_y_obtener_snapshot", func(t *testing.T) {
		registry := NewRegistry()
		sess := New("127.0.0.1:54000")
		sess.SetTransport("ssh")
		sess.SetAuthn("password")
		sess.SetIdentity("student", "student")

		registry.Add(sess)
		snapshots := registry.Snapshot()
		if len(snapshots) != 1 {
			t.Fatalf("se esperaba una sesión, got %d", len(snapshots))
		}
		if snapshots[0].Username != "student" {
			t.Fatalf("usuario incorrecto: %s", snapshots[0].Username)
		}
	})

	t.Run("notificar_cambios", func(t *testing.T) {
		registry := NewRegistry()
		notifications := make(chan []Snapshot, 2)
		registry.SetOnChange(func(items []Snapshot) {
			copyItems := append([]Snapshot(nil), items...)
			notifications <- copyItems
		})

		sess := New("127.0.0.1:54001")
		registry.Add(sess)
		registry.Remove(sess.ID)

		first := <-notifications
		second := <-notifications
		if len(first) != 1 {
			t.Fatalf("la primera notificación debería tener una sesión, got %d", len(first))
		}
		if len(second) != 0 {
			t.Fatalf("la segunda notificación debería quedar vacía, got %d", len(second))
		}
	})

	t.Run("acceso_concurrente", func(t *testing.T) {
		registry := NewRegistry()
		var wg sync.WaitGroup

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				sess := New(fmt.Sprintf("127.0.0.1:%d", 55000+index))
				sess.SetIdentity("student", "student")
				registry.Add(sess)
			}(i)
		}

		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = registry.Snapshot()
				_ = registry.Count()
			}()
		}

		wg.Wait()
		if got := registry.Count(); got != 100 {
			t.Fatalf("se esperaban 100 sesiones, got %d", got)
		}
	})

	t.Run("snapshot_ordenado_por_conexion", func(t *testing.T) {
		registry := NewRegistry()
		first := New("127.0.0.1:56001")
		time.Sleep(time.Millisecond)
		second := New("127.0.0.1:56002")

		registry.Add(second)
		registry.Add(first)

		snapshots := registry.Snapshot()
		if snapshots[0].ID != first.ID || snapshots[1].ID != second.ID {
			t.Fatalf("el snapshot no está ordenado por fecha de conexión: %#v", snapshots)
		}
	})
}
