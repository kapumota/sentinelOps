//go:build containers

package integration

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go/modules/redis"

	"sentinelops/internal/auth"
)

func TestAuthIntegrationWithRedisContainer(t *testing.T) {
	SkipIfShort(t)
	RequireDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	redisContainer, err := redis.Run(ctx, "redis:7-alpine")
	if err != nil {
		t.Fatalf("no se pudo iniciar Redis con testcontainers: %v", err)
	}
	defer func() {
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Logf("no se pudo terminar Redis: %v", err)
		}
	}()

	redisURI, err := redisContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("no se pudo obtener la URI de Redis: %v", err)
	}
	if !strings.HasPrefix(redisURI, "redis://") {
		t.Fatalf("URI de Redis inesperada: %s", redisURI)
	}

	t.Setenv("LAB_PASSWORD_STUDENT", "student-secret-it")
	t.Setenv("LAB_PASSWORD_TEACHER", "teacher-secret-it")
	t.Setenv("LAB_PASSWORD_AUDITOR", "auditor-secret-it")
	t.Setenv("LAB_PASSWORD_ADMIN", "admin-secret-it")

	svc := auth.NewDefaultService()
	limiter := auth.NewRateLimiter(auth.RateLimitConfig{
		Enabled:     true,
		MaxFailures: 2,
		Window:      time.Minute,
		Lockout:     time.Minute,
	})

	t.Run("autenticacion_exitosa", func(t *testing.T) {
		identity, err := svc.Authenticate("student", "student-secret-it")
		if err != nil {
			t.Fatalf("la autenticación debió ser exitosa: %v", err)
		}
		if identity.Role != auth.RoleStudent {
			t.Fatalf("rol incorrecto: got %s, want %s", identity.Role, auth.RoleStudent)
		}
	})

	t.Run("rate_limiting_bloquea_despues_de_fallos", func(t *testing.T) {
		key := auth.Key("127.0.0.1:52000", "student")
		if !limiter.Allow(key).Allowed {
			t.Fatal("el primer intento debería estar permitido")
		}
		if !limiter.RecordFailure(key).Allowed {
			t.Fatal("el primer fallo no debería bloquear")
		}
		blocked := limiter.RecordFailure(key)
		if blocked.Allowed {
			t.Fatal("el segundo fallo debería bloquear")
		}
		if limiter.Allow(key).Allowed {
			t.Fatal("la clave bloqueada no debería estar permitida")
		}
	})

	t.Run("acceso_concurrente_sin_panico", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, 64)

		for i := 0; i < 64; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				password := "student-secret-it"
				if index%2 == 0 {
					password = "clave-incorrecta"
				}
				_, err := svc.Authenticate("student", password)
				if err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		failed := 0
		for range errors {
			failed++
		}
		if failed == 0 {
			t.Fatal("se esperaban algunos intentos fallidos")
		}
	})
}
