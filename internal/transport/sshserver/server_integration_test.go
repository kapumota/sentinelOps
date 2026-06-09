package sshserver

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"io"
	"log/slog"
	"net"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"sentinelops/internal/auth"
	"sentinelops/internal/commands"
	"sentinelops/internal/config"
	"sentinelops/internal/forwarding"
	"sentinelops/internal/metrics"
	"sentinelops/internal/security"
	"sentinelops/internal/session"
)

func TestSSHForwardingIntegration(t *testing.T) {
	t.Setenv("LAB_PASSWORD_STUDENT", "student-secret")
	sshAddr := freeTCPAddr(t)
	targetAddr, stopEcho := startEchoServer(t)
	defer stopEcho()

	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate host key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("create host signer: %v", err)
	}

	cfg := config.Config{
		Profile:                 "hardened",
		SSHListenAddr:           sshAddr,
		SSHServerVersion:        "SSH-2.0-SentinelOps-Test",
		SSHPasswordAuthEnabled:  true,
		SSHPublicKeyAuthEnable:  false,
		SSHLocalForwardEnabled:  true,
		SSHForwardAllowlist:     targetAddr,
		SSHLocalAllowedRoles:    "student,teacher,auditor,admin",
		SSHRemoteForwardEnabled: true,
		SSHRemoteBindAllowlist:  "127.0.0.1:0",
		SSHRemoteAllowedRoles:   "student,teacher,auditor,admin",
		Banner:                  "SentinelOps Test",
		ReadTimeout:             5 * time.Second,
		WriteTimeout:            5 * time.Second,
		IdleTimeout:             30 * time.Second,
	}

	tunnelManager := forwarding.NewManager(forwarding.Hooks{})
	server := New(
		cfg,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		metrics.New("127.0.0.1:0", slog.New(slog.NewTextHandler(io.Discard, nil))),
		security.NewValidator(security.Options{}),
		auth.NewDefaultService(),
		auth.NewRateLimiter(auth.RateLimitConfig{Enabled: false}),
		commands.NewRegistry(commands.NewHelpCommand()),
		signer,
		nil,
		forwarding.NewPolicy(true, targetAddr, "student,teacher,auditor,admin", true, "127.0.0.1:0", "student,teacher,auditor,admin"),
		tunnelManager,
		session.NewRegistry(),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error, 1)
	go func() { errCh <- server.Run(ctx) }()

	client := dialSSH(t, sshAddr)
	defer client.Close()

	t.Run("local direct-tcpip reaches allowed target", func(t *testing.T) {
		conn, err := client.Dial("tcp", targetAddr)
		if err != nil {
			t.Fatalf("client local forward dial failed: %v", err)
		}
		defer conn.Close()

		if _, err := conn.Write([]byte("ping")); err != nil {
			t.Fatalf("write forwarded payload: %v", err)
		}
		buf := make([]byte, 9)
		if _, err := io.ReadFull(conn, buf); err != nil {
			t.Fatalf("read forwarded payload: %v", err)
		}
		if string(buf) != "echo:ping" {
			t.Fatalf("unexpected forwarded response: %q", string(buf))
		}
	})

	t.Run("remote tcpip-forward accepts inbound connection", func(t *testing.T) {
		listener, err := client.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("remote forward listen failed: %v", err)
		}
		defer listener.Close()

		serverSide := make(chan net.Conn, 1)
		acceptErr := make(chan error, 1)
		go func() {
			conn, err := listener.Accept()
			if err != nil {
				acceptErr <- err
				return
			}
			serverSide <- conn
		}()

		inbound, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
		if err != nil {
			t.Fatalf("dial remote forwarded listener: %v", err)
		}
		defer inbound.Close()

		var forwarded net.Conn
		select {
		case forwarded = <-serverSide:
			defer forwarded.Close()
		case err := <-acceptErr:
			t.Fatalf("accept remote forwarded channel: %v", err)
		case <-time.After(2 * time.Second):
			t.Fatal("remote forwarded channel was not accepted")
		}

		if _, err := inbound.Write([]byte("hola")); err != nil {
			t.Fatalf("write inbound payload: %v", err)
		}
		buf := make([]byte, 4)
		if _, err := io.ReadFull(forwarded, buf); err != nil {
			t.Fatalf("read forwarded inbound payload: %v", err)
		}
		if string(buf) != "hola" {
			t.Fatalf("unexpected remote forwarded payload: %q", string(buf))
		}
	})

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("ssh server returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("ssh server did not stop after context cancellation")
	}
}

func dialSSH(t *testing.T, addr string) *ssh.Client {
	t.Helper()
	cfg := &ssh.ClientConfig{
		User:            "student",
		Auth:            []ssh.AuthMethod{ssh.Password("student-secret")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         time.Second,
	}
	deadline := time.Now().Add(3 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		client, err := ssh.Dial("tcp", addr, cfg)
		if err == nil {
			return client
		}
		lastErr = err
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("ssh server did not become ready: %v", lastErr)
	return nil
}

func startEchoServer(t *testing.T) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("start echo listener: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				n, err := c.Read(buf)
				if err != nil {
					return
				}
				_, _ = c.Write([]byte("echo:" + string(buf[:n])))
			}(conn)
		}
	}()
	return ln.Addr().String(), func() { cancel(); _ = ln.Close() }
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
