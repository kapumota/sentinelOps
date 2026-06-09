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

	res, err := client.Get(baseURL + "/healthz")
	if err != nil {
		t.Fatalf("healthz request failed: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected healthz 200, got %d", res.StatusCode)
	}
	if got := res.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("expected nosniff header, got %q", got)
	}
	_ = res.Body.Close()

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
