package store

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrNotFound      = errors.New("recurso no encontrado")
	ErrAlreadyExists = errors.New("recurso ya existe")
	ErrInvalidState  = errors.New("estado inválido")
)

type Store interface {
	CreateSession(ctx context.Context, sess *Session) error
	GetSession(ctx context.Context, id string) (*Session, error)
	UpdateSession(ctx context.Context, sess *Session) error
	DeleteSession(ctx context.Context, id string) error
	ListSessions(ctx context.Context, filter SessionFilter) ([]*Session, error)
	CountActiveSessions(ctx context.Context, username string) (int, error)
	TouchSession(ctx context.Context, id string) error

	CreateTunnel(ctx context.Context, tunnel *Tunnel) error
	GetTunnel(ctx context.Context, id string) (*Tunnel, error)
	UpdateTunnel(ctx context.Context, tunnel *Tunnel) error
	DeleteTunnel(ctx context.Context, id string) error
	ListTunnels(ctx context.Context, filter TunnelFilter) ([]*Tunnel, error)
	DeleteTunnelsBySession(ctx context.Context, sessionID string) ([]string, error)

	IncrementAttempts(ctx context.Context, key string, window time.Duration) (int, error)
	GetAttempts(ctx context.Context, key string) (int, error)
	ResetAttempts(ctx context.Context, key string) error
	IsLocked(ctx context.Context, key string, lockout time.Duration) (bool, error)
	Lock(ctx context.Context, key string, duration time.Duration) error

	AppendAuditLog(ctx context.Context, entry *AuditEntry) error
	QueryAuditLog(ctx context.Context, filter AuditFilter) ([]*AuditEntry, error)

	CleanupInactiveSessions(ctx context.Context, maxInactive time.Duration) (int, error)
	CleanupOldAuditLogs(ctx context.Context, maxAge time.Duration) (int, error)

	Health(ctx context.Context) error
	Close() error
}

type Session struct {
	ID         string     `json:"id" db:"id"`
	Username   string     `json:"username" db:"username"`
	Role       string     `json:"role" db:"role"`
	RemoteAddr string     `json:"remote_addr" db:"remote_addr"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	LastActive time.Time  `json:"last_active" db:"last_active"`
	ClosedAt   *time.Time `json:"closed_at,omitempty" db:"closed_at"`
	Status     string     `json:"status" db:"status"`
	Metadata   JSONMap    `json:"metadata" db:"metadata"`
}

type Tunnel struct {
	ID         string     `json:"id" db:"id"`
	SessionID  string     `json:"session_id" db:"session_id"`
	Type       string     `json:"type" db:"type"`
	LocalAddr  string     `json:"local_addr" db:"local_addr"`
	RemoteAddr string     `json:"remote_addr" db:"remote_addr"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	ClosedAt   *time.Time `json:"closed_at,omitempty" db:"closed_at"`
	BytesSent  int64      `json:"bytes_sent" db:"bytes_sent"`
	BytesRecv  int64      `json:"bytes_recv" db:"bytes_recv"`
	Status     string     `json:"status" db:"status"`
	Metadata   JSONMap    `json:"metadata" db:"metadata"`
}

type AuditEntry struct {
	ID            string    `json:"id" db:"id"`
	Timestamp     time.Time `json:"timestamp" db:"timestamp"`
	CorrelationID string    `json:"correlation_id" db:"correlation_id"`
	TraceID       string    `json:"trace_id" db:"trace_id"`
	Action        string    `json:"action" db:"action"`
	Username      string    `json:"username" db:"username"`
	Role          string    `json:"role" db:"role"`
	Resource      string    `json:"resource" db:"resource"`
	Result        string    `json:"result" db:"result"`
	Details       JSONMap   `json:"details" db:"details"`
	SourceIP      string    `json:"source_ip" db:"source_ip"`
}

type SessionFilter struct {
	Username   string
	Status     string
	Since      *time.Time
	RemoteAddr string
	Limit      int
	Offset     int
}

type TunnelFilter struct {
	SessionID string
	Type      string
	Status    string
	Limit     int
	Offset    int
}

type AuditFilter struct {
	Username string
	Action   string
	Result   string
	Since    *time.Time
	Until    *time.Time
	Limit    int
	Offset   int
}

type JSONMap map[string]any

type Config struct {
	Type     string
	Postgres PostgresConfig
	Redis    RedisConfig
}

type PostgresConfig struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSLMode  string
	PoolSize int
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	PoolSize int
}

func NewStore(cfg Config) (Store, error) {
	switch cfg.Type {
	case "", "memory":
		return NewMemoryStore(), nil
	case "postgres":
		return NewPostgresStore(cfg.Postgres)
	case "redis":
		return NewRedisStore(cfg.Redis)
	default:
		return nil, fmt.Errorf("tipo de almacenamiento desconocido: %s", cfg.Type)
	}
}
