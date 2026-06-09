//go:build containers

package integration

import (
	"context"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
)

const TestTimeout = 90 * time.Second

func SkipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("saltando prueba de integración en modo corto")
	}
}

func RequireDocker(t *testing.T) {
	t.Helper()
	if os.Getenv("DOCKER_HOST") != "" {
		return
	}
	if _, err := os.Stat("/var/run/docker.sock"); err == nil {
		return
	}
	t.Skip("Docker no está disponible para pruebas con testcontainers")
}

type ContainerLogConsumer struct {
	Prefix string
}

func (c *ContainerLogConsumer) Accept(log testcontainers.Log) {
	prefix := c.Prefix
	if prefix == "" {
		prefix = "contenedor"
	}
	fmt.Printf("[%s] %s", prefix, string(log.Content))
}

func ContextWithTimeout(t *testing.T, timeout time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), timeout)
}

func FreeTCPAddr(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("no se pudo reservar un puerto TCP libre: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("no se pudo cerrar el listener temporal: %v", err)
	}
	return addr
}
