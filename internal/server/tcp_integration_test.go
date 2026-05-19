package server

import (
	"bufio"
	"context"
	"io"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"

	"sentinelops/internal/auth"
	"sentinelops/internal/commands"
	"sentinelops/internal/config"
	"sentinelops/internal/metrics"
	"sentinelops/internal/security"
	"sentinelops/internal/session"
)

func TestTCPLoginRateLimitIntegration(t *testing.T) {
	addr := freeTCPAddr(t)
	cfg := config.Config{
		ListenAddr:      addr,
		Profile:         "hardened",
		Banner:          "SentinelOps Test",
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
		IdleTimeout:     5 * time.Second,
		AuthEnabled:     true,
		AuthMaxAttempts: 1,
	}

	rateLimiter := auth.NewRateLimiter(auth.RateLimitConfig{
		Enabled:     true,
		MaxFailures: 2,
		Window:      time.Minute,
		Lockout:     time.Minute,
	})

	srv := NewTCPServer(
		cfg,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		metrics.New("127.0.0.1:0", slog.New(slog.NewTextHandler(io.Discard, nil))),
		security.NewValidator(security.Options{}),
		auth.NewDefaultService(),
		rateLimiter,
		commands.NewRegistry(commands.NewHelpCommand()),
		session.NewRegistry(),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Run(ctx)
	}()

	waitForTCP(t, addr)

	first := attemptLogin(t, addr, "student", "wrong")
	if !strings.Contains(first, "Invalid credentials") {
		t.Fatalf("se esperaba respuesta de credenciales inválidas en el primer intento, se obtuvo %q", first)
	}

	second := attemptLogin(t, addr, "student", "still-wrong")
	if !strings.Contains(second, "Login temporarily locked") {
		t.Fatalf("se esperaba bloqueo temporal de la cuenta en el segundo intento, se obtuvo %q", second)
	}

	third := attemptLogin(t, addr, "student", "student123!")
	if !strings.Contains(third, "Too many failed login attempts") {
		t.Fatalf("se esperaba que la contraseña válida fuera bloqueada durante el periodo de bloqueo, se obtuvo %q", third)
	}

	cancel()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("el servidor TCP devolvió un error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("el servidor TCP no se detuvo después de cancelar el contexto")
	}
}

func attemptLogin(t *testing.T, addr, username, password string) string {
	t.Helper()

	conn, err := net.DialTimeout("tcp", addr, time.Second)
	if err != nil {
		t.Fatalf("no se pudo conectar al servidor TCP: %v", err)
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))

	reader := bufio.NewReader(conn)

	_, _ = reader.ReadString('\n')
	_, _ = reader.ReadString('\n')

	if _, err := reader.ReadString(' '); err != nil {
		t.Fatalf("no se pudo leer el prompt de usuario: %v", err)
	}

	_, _ = io.WriteString(conn, username+"\n")

	if _, err := reader.ReadString(' '); err != nil {
		t.Fatalf("no se pudo leer el prompt de contraseña: %v", err)
	}

	_, _ = io.WriteString(conn, password+"\n")

	body, _ := io.ReadAll(reader)
	return string(body)
}

func waitForTCP(t *testing.T, addr string) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	var lastErr error

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}

		lastErr = err
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("el servidor TCP no quedó listo a tiempo: %v", lastErr)
}

func freeTCPAddr(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("no se pudo reservar un puerto TCP libre: %v", err)
	}

	addr := ln.Addr().String()
	_ = ln.Close()

	return addr
}
