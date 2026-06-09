//go:build containers

package integration

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestE2EWithContainerImage(t *testing.T) {
	SkipIfShort(t)
	RequireDocker(t)

	image := os.Getenv("SENTINELOPS_E2E_IMAGE")
	if image == "" {
		t.Skip("SENTINELOPS_E2E_IMAGE no está definido; saltando E2E de imagen completa")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{"9443/tcp", "9001/tcp"},
		Env: map[string]string{
			"APP_ENV":                  "testcontainers",
			"APP_PROFILE":              "hardened",
			"APP_TRANSPORT":            "ssh",
			"APP_CONTROL_API_ENABLED":  "true",
			"APP_CONTROL_API_ADDR":     ":9443",
			"APP_CONTROL_API_USER":     "admin",
			"APP_CONTROL_API_PASSWORD": "admin-secret-it",
			"METRICS_ADDR":             ":9001",
			"LAB_PASSWORD_STUDENT":     "student-secret-it",
			"LAB_PASSWORD_TEACHER":     "teacher-secret-it",
			"LAB_PASSWORD_AUDITOR":     "auditor-secret-it",
			"LAB_PASSWORD_ADMIN":       "admin-secret-it",
		},
		WaitingFor: wait.ForListeningPort("9443/tcp").WithStartupTimeout(time.Minute),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("no se pudo iniciar la imagen E2E: %v", err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("no se pudo terminar el contenedor E2E: %v", err)
		}
	}()

	apiPort, err := container.MappedPort(ctx, "9443/tcp")
	if err != nil {
		t.Fatalf("no se pudo obtener puerto de API: %v", err)
	}
	metricsPort, err := container.MappedPort(ctx, "9001/tcp")
	if err != nil {
		t.Fatalf("no se pudo obtener puerto de métricas: %v", err)
	}

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true}},
	}

	apiURL := fmt.Sprintf("https://127.0.0.1:%s", apiPort.Port())
	metricsURL := fmt.Sprintf("http://127.0.0.1:%s/metrics", metricsPort.Port())

	res, err := client.Get(apiURL + "/healthz")
	if err != nil {
		t.Fatalf("healthz falló: %v", err)
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("healthz devolvió %d", res.StatusCode)
	}

	reqStatus, err := http.NewRequest(http.MethodGet, apiURL+"/api/admin/status", nil)
	if err != nil {
		t.Fatalf("no se pudo crear request de status: %v", err)
	}
	reqStatus.SetBasicAuth("admin", "admin-secret-it")
	res, err = client.Do(reqStatus)
	if err != nil {
		t.Fatalf("status autenticado falló: %v", err)
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status autenticado devolvió %d", res.StatusCode)
	}

	res, err = http.Get(metricsURL)
	if err != nil {
		t.Fatalf("metrics falló: %v", err)
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("metrics devolvió %d", res.StatusCode)
	}
}
