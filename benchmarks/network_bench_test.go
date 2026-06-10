package benchmarks

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

func BenchmarkTCPConnectionThroughput(b *testing.B) {
	addr, stop := startTCPAcceptCloseServer(b)
	defer stop()

	start := time.Now()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err != nil {
			b.Fatalf("falló conexión TCP: %v", err)
		}
		_ = conn.Close()
	}

	b.StopTimer()
	reportConnectionsPerSecond(b, start)
}

func BenchmarkSSHConnectionThroughput(b *testing.B) {
	addr, hostPublicKey, stop := startSSHAcceptCloseServer(b)
	defer stop()

	config := &ssh.ClientConfig{
		User:            "bench",
		Auth:            []ssh.AuthMethod{},
		HostKeyCallback: strictHostKeyCallback(hostPublicKey),
		Timeout:         2 * time.Second,
	}

	start := time.Now()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		client, err := ssh.Dial("tcp", addr, config)
		if err != nil {
			b.Fatalf("falló conexión SSH: %v", err)
		}
		_ = client.Close()
	}

	b.StopTimer()
	reportConnectionsPerSecond(b, start)
}

func reportConnectionsPerSecond(b *testing.B, start time.Time) {
	elapsed := time.Since(start).Seconds()
	if elapsed > 0 {
		b.ReportMetric(float64(b.N)/elapsed, "conn/s")
	}
}

func startTCPAcceptCloseServer(b *testing.B) (string, func()) {
	b.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("no se pudo iniciar TCP benchmark: %v", err)
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-stop:
					return
				default:
					continue
				}
			}
			_ = conn.Close()
		}
	}()

	return listener.Addr().String(), func() {
		close(stop)
		_ = listener.Close()
		wg.Wait()
	}
}

func startSSHAcceptCloseServer(b *testing.B) (string, ssh.PublicKey, func()) {
	b.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		b.Fatalf("no se pudo generar clave SSH efímera: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		b.Fatalf("no se pudo crear signer SSH: %v", err)
	}

	config := &ssh.ServerConfig{NoClientAuth: true}
	config.AddHostKey(signer)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("no se pudo iniciar SSH benchmark: %v", err)
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-stop:
					return
				default:
					continue
				}
			}
			go handleSSHBenchmarkConn(conn, config)
		}
	}()

	return listener.Addr().String(), signer.PublicKey(), func() {
		close(stop)
		_ = listener.Close()
		wg.Wait()
	}
}

func handleSSHBenchmarkConn(conn net.Conn, config *ssh.ServerConfig) {
	serverConn, channels, requests, err := ssh.NewServerConn(conn, config)
	if err != nil {
		_ = conn.Close()
		return
	}
	go ssh.DiscardRequests(requests)
	go ssh.DiscardRequests(requests)
	for newChannel := range channels {
		_ = newChannel.Reject(ssh.UnknownChannelType, "benchmark sin canales")
	}
	_ = serverConn.Close()
}

func strictHostKeyCallback(expected ssh.PublicKey) ssh.HostKeyCallback {
	return func(_ string, _ net.Addr, key ssh.PublicKey) error {
		if bytes.Equal(expected.Marshal(), key.Marshal()) {
			return nil
		}
		return fmt.Errorf("host key no coincide")
	}
}
