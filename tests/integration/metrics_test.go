//go:build containers

package integration

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"sentinelops/internal/metrics"
)

func TestMetricsIntegrationWithPrometheus(t *testing.T) {
	SkipIfShort(t)
	RequireDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	defer cancel()

	req := testcontainers.ContainerRequest{
		Image:        "prom/prometheus:v2.50.0",
		ExposedPorts: []string{"9090/tcp"},
		WaitingFor:   wait.ForHTTP("/-/healthy").WithPort("9090/tcp").WithStartupTimeout(20 * time.Second),
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      "testdata/prometheus.yml",
				ContainerFilePath: "/etc/prometheus/prometheus.yml",
				FileMode:          0o644,
			},
		},
	}

	prometheus, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("no se pudo iniciar Prometheus: %v", err)
	}
	defer func() {
		if err := prometheus.Terminate(ctx); err != nil {
			t.Logf("no se pudo terminar Prometheus: %v", err)
		}
	}()

	mappedPort, err := prometheus.MappedPort(ctx, "9090/tcp")
	if err != nil {
		t.Fatalf("no se pudo obtener el puerto de Prometheus: %v", err)
	}
	prometheusURL := fmt.Sprintf("http://127.0.0.1:%s", mappedPort.Port())

	assertHTTPContains(t, prometheusURL+"/-/healthy", "")
	assertHTTPContains(t, prometheusURL+"/api/v1/query?query=prometheus_build_info", "success")
}

func TestMetricsEndpointExposesSentinelOpsCounters(t *testing.T) {
	addr := FreeTCPAddr(t)
	server := metrics.New(addr, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start(ctx)
	}()

	url := "http://" + addr + "/metrics"
	waitForMetrics(t, url)

	server.ObserveSessionOpened()
	server.ObserveCommand("status", "ok")
	server.ObserveRejectedInput()

	body := assertHTTPContains(t, url, "sentinelops_sessions_total")
	if !strings.Contains(body, "sentinelops_commands_total") {
		t.Fatalf("no se encontró métrica de comandos en /metrics")
	}
	if !strings.Contains(body, "sentinelops_rejected_input_total") {
		t.Fatalf("no se encontró métrica de entradas rechazadas en /metrics")
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("el servidor de métricas devolvió error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("el servidor de métricas no se detuvo después de cancelar el contexto")
	}
}

func waitForMetrics(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("el endpoint de métricas no quedó disponible: %s", url)
}

func assertHTTPContains(t *testing.T, url, expected string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("no se pudo consultar %s: %v", url, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("no se pudo leer respuesta de %s: %v", url, err)
	}
	body := string(bodyBytes)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("respuesta HTTP inesperada en %s: %d, body: %s", url, resp.StatusCode, body)
	}
	if expected != "" && !strings.Contains(body, expected) {
		t.Fatalf("respuesta de %s no contiene %q: %s", url, expected, body)
	}
	return body
}
