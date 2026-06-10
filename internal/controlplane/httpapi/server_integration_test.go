package httpapi

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"sentinelops/internal/config"
	"sentinelops/internal/forwarding"
	"sentinelops/internal/session"
)

func TestControlAPIHTTPSIntegration(t *testing.T) {
	t.Parallel()

	addr := freeTCPAddr(t)
	dir := t.TempDir()
	cfg := config.Config{
		AppName:             "sentinelops",
		Profile:             "hardened",
		Transport:           "ssh",
		ControlAPIEnabled:   true,
		ControlAPIAddr:      addr,
		ControlAPICertPath:  filepath.Join(dir, "tls.crt"),
		ControlAPIKeyPath:   filepath.Join(dir, "tls.key"),
		ControlAPIUser:      "admin",
		ControlAPIPassword:  "admin-secret",
		ControlAPICertHosts: "localhost,127.0.0.1",
		ReadTimeout:         5 * time.Second,
		WriteTimeout:        5 * time.Second,
		IdleTimeout:         30 * time.Second,
	}

	registry := session.NewRegistry()
	registry.Add(&session.Session{ID: "sess-it", RemoteAddr: "127.0.0.1:1111", Transport: "ssh", Username: "student", Role: "student", ConnectedAt: time.Now().UTC()})
	tunnels := forwarding.NewManager(forwarding.Hooks{})
	api := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), registry, tunnels)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() { errCh <- api.Start(ctx) }()

	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: true}}}
	baseURL := "https://" + addr
	waitForHTTPS(t, client, baseURL+"/healthz")

	var res *http.Response
	var err error

	for _, path := range []string{"/healthz", "/healthz/live", "/healthz/ready", "/healthz/startup"} {
		res, err = client.Get(baseURL + path)
		if err != nil {
			t.Fatalf("health request failed for %s: %v", path, err)
		}
		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected health 200 for %s, got %d", path, res.StatusCode)
		}
		if got := res.Header.Get("X-Content-Type-Options"); got != "nosniff" {
			t.Fatalf("expected nosniff header for %s, got %q", path, got)
		}
		_ = res.Body.Close()
	}

	res, err = client.Get(baseURL + "/api/admin/status")
	if err != nil {
		t.Fatalf("unauthorized status request failed: %v", err)
	}
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized status 401, got %d", res.StatusCode)
	}
	_ = res.Body.Close()

	req, err := http.NewRequest(http.MethodGet, baseURL+"/api/admin/status", nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.SetBasicAuth("admin", "admin-secret")
	res, err = client.Do(req)
	if err != nil {
		t.Fatalf("authorized status request failed: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected authorized status 200, got %d", res.StatusCode)
	}
	var payload map[string]any
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("decode status payload: %v", err)
	}
	if payload["sesiones_activas"].(float64) != 1 {
		t.Fatalf("expected one active session, got %#v", payload["sesiones_activas"])
	}

	req, err = http.NewRequest(http.MethodGet, baseURL+"/api/v1/admin/status", nil)
	if err != nil {
		t.Fatalf("build v1 status request: %v", err)
	}
	req.SetBasicAuth("admin", "admin-secret")
	res, err = client.Do(req)
	if err != nil {
		t.Fatalf("authorized v1 status request failed: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected authorized v1 status 200, got %d", res.StatusCode)
	}
	_ = res.Body.Close()

	res, err = client.Get(baseURL + "/api/v1/docs/swagger.json")
	if err != nil {
		t.Fatalf("openapi request failed: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected openapi 200, got %d", res.StatusCode)
	}
	if got := res.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected openapi json content type, got %q", got)
	}
	var spec map[string]any
	if err := json.NewDecoder(res.Body).Decode(&spec); err != nil {
		t.Fatalf("decode openapi: %v", err)
	}
	_ = res.Body.Close()
	if spec["openapi"] != "3.0.3" {
		t.Fatalf("expected openapi 3.0.3, got %#v", spec["openapi"])
	}

	res, err = client.Get(baseURL + "/api/v1/docs/swagger/")
	if err != nil {
		t.Fatalf("swagger ui request failed: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected swagger ui 200, got %d", res.StatusCode)
	}
	_ = res.Body.Close()

	tunnel := tunnels.OpenLocal("sess-it", "student", "127.0.0.1:9001", "127.0.0.1:1234", nil)
	req, err = http.NewRequest(http.MethodPost, baseURL+"/api/v1/admin/tunnels/"+tunnel.ID+"/close", nil)
	if err != nil {
		t.Fatalf("build close tunnel request: %v", err)
	}
	req.SetBasicAuth("admin", "admin-secret")
	res, err = client.Do(req)
	if err != nil {
		t.Fatalf("close tunnel request failed: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected close tunnel 200, got %d", res.StatusCode)
	}
	_ = res.Body.Close()
	if _, ok := tunnels.Get(tunnel.ID); ok {
		t.Fatalf("expected tunnel %s to be closed", tunnel.ID)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("api returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("api did not stop after context cancellation")
	}
}

func freeTCPAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen for free addr: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	return addr
}

func waitForHTTPS(t *testing.T, client *http.Client, url string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		res, err := client.Get(url)
		if err == nil {
			_ = res.Body.Close()
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("server did not become ready at %s", url)
}
