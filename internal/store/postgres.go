package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type PostgresStore struct {
	db *sqlx.DB
}

const postgresSchema = `
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    role TEXT NOT NULL,
    remote_addr TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    last_active TIMESTAMPTZ NOT NULL,
    closed_at TIMESTAMPTZ,
    status TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_sessions_username ON sessions(username);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_sessions_created_at ON sessions(created_at);

CREATE TABLE IF NOT EXISTS tunnels (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    local_addr TEXT,
    remote_addr TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    closed_at TIMESTAMPTZ,
    bytes_sent BIGINT NOT NULL DEFAULT 0,
    bytes_recv BIGINT NOT NULL DEFAULT 0,
    status TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_tunnels_session_id ON tunnels(session_id);
CREATE INDEX IF NOT EXISTS idx_tunnels_status ON tunnels(status);

CREATE TABLE IF NOT EXISTS audit_log (
    id TEXT PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    correlation_id TEXT,
    trace_id TEXT,
    action TEXT NOT NULL,
    username TEXT,
    role TEXT,
    resource TEXT,
    result TEXT,
    details JSONB NOT NULL DEFAULT '{}',
    source_ip TEXT
);

CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_username ON audit_log(username);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_log(action);
CREATE INDEX IF NOT EXISTS idx_audit_result ON audit_log(result);

CREATE TABLE IF NOT EXISTS rate_limits (
    name TEXT PRIMARY KEY,
    count INTEGER NOT NULL DEFAULT 0,
    first_at TIMESTAMPTZ NOT NULL,
    locked_until TIMESTAMPTZ
);
`

func NewPostgresStore(cfg PostgresConfig) (*PostgresStore, error) {
	if cfg.Port == 0 {
		cfg.Port = 5432
	}
	if cfg.SSLMode == "" {
		cfg.SSLMode = "disable"
	}
	if cfg.PoolSize <= 0 {
		cfg.PoolSize = 10
	}

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Database,
		cfg.SSLMode,
	)

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("conectando a PostgreSQL: %w", err)
	}
	db.SetMaxOpenConns(cfg.PoolSize)
	db.SetMaxIdleConns(maxInt(1, cfg.PoolSize/2))
	db.SetConnMaxLifetime(30 * time.Minute)

	if _, err := db.Exec(postgresSchema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("creando esquema PostgreSQL: %w", err)
	}
	return &PostgresStore{db: db}, nil
}

func (p *PostgresStore) CreateSession(ctx context.Context, sess *Session) error {
	if sess.ID == "" {
		sess.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	sess.CreatedAt = now
	sess.LastActive = now
	if sess.Status == "" {
		sess.Status = "active"
	}
	metadata, err := marshalJSONMap(sess.Metadata)
	if err != nil {
		return err
	}
	_, err = p.db.ExecContext(ctx, `
        INSERT INTO sessions (id, username, role, remote_addr, created_at, last_active, status, metadata)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `, sess.ID, sess.Username, sess.Role, sess.RemoteAddr, sess.CreatedAt, sess.LastActive, sess.Status, metadata)
	return translatePostgresError(err)
}

func (p *PostgresStore) GetSession(ctx context.Context, id string) (*Session, error) {
	var sess Session
	var metadata []byte
	err := p.db.QueryRowxContext(ctx, `
        SELECT id, username, role, remote_addr, created_at, last_active, closed_at, status, metadata
        FROM sessions WHERE id = $1
    `, id).Scan(&sess.ID, &sess.Username, &sess.Role, &sess.RemoteAddr, &sess.CreatedAt, &sess.LastActive, &sess.ClosedAt, &sess.Status, &metadata)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	sess.Metadata = unmarshalJSONMap(metadata)
	return &sess, nil
}

func (p *PostgresStore) UpdateSession(ctx context.Context, sess *Session) error {
	sess.LastActive = time.Now().UTC()
	metadata, err := marshalJSONMap(sess.Metadata)
	if err != nil {
		return err
	}
	result, err := p.db.ExecContext(ctx, `
        UPDATE sessions
        SET username = $2, role = $3, remote_addr = $4, last_active = $5, closed_at = $6, status = $7, metadata = $8
        WHERE id = $1
    `, sess.ID, sess.Username, sess.Role, sess.RemoteAddr, sess.LastActive, sess.ClosedAt, sess.Status, metadata)
	if err != nil {
		return err
	}
	return requireRowsAffected(result)
}

func (p *PostgresStore) DeleteSession(ctx context.Context, id string) error {
	result, err := p.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = $1`, id)
	if err != nil {
		return err
	}
	return requireRowsAffected(result)
}

func (p *PostgresStore) ListSessions(ctx context.Context, filter SessionFilter) ([]*Session, error) {
	query := `SELECT id, username, role, remote_addr, created_at, last_active, closed_at, status, metadata FROM sessions WHERE 1=1`
	args := []any{}
	index := 1
	if filter.Username != "" {
		query += fmt.Sprintf(" AND username = $%d", index)
		args = append(args, filter.Username)
		index++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", index)
		args = append(args, filter.Status)
		index++
	}
	if filter.RemoteAddr != "" {
		query += fmt.Sprintf(" AND remote_addr = $%d", index)
		args = append(args, filter.RemoteAddr)
		index++
	}
	if filter.Since != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", index)
		args = append(args, *filter.Since)
		index++
	}
	query += " ORDER BY created_at DESC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", index)
		args = append(args, filter.Limit)
		index++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", index)
		args = append(args, filter.Offset)
	}
	rows, err := p.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*Session, 0)
	for rows.Next() {
		var sess Session
		var metadata []byte
		if err := rows.Scan(&sess.ID, &sess.Username, &sess.Role, &sess.RemoteAddr, &sess.CreatedAt, &sess.LastActive, &sess.ClosedAt, &sess.Status, &metadata); err != nil {
			return nil, err
		}
		sess.Metadata = unmarshalJSONMap(metadata)
		result = append(result, &sess)
	}
	return result, rows.Err()
}

func (p *PostgresStore) CountActiveSessions(ctx context.Context, username string) (int, error) {
	var count int
	err := p.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM sessions WHERE username = $1 AND status = 'active'`, username)
	return count, err
}

func (p *PostgresStore) TouchSession(ctx context.Context, id string) error {
	result, err := p.db.ExecContext(ctx, `UPDATE sessions SET last_active = $2 WHERE id = $1`, id, time.Now().UTC())
	if err != nil {
		return err
	}
	return requireRowsAffected(result)
}

func (p *PostgresStore) CreateTunnel(ctx context.Context, tunnel *Tunnel) error {
	if tunnel.ID == "" {
		tunnel.ID = uuid.New().String()
	}
	tunnel.CreatedAt = time.Now().UTC()
	if tunnel.Status == "" {
		tunnel.Status = "active"
	}
	metadata, err := marshalJSONMap(tunnel.Metadata)
	if err != nil {
		return err
	}
	_, err = p.db.ExecContext(ctx, `
        INSERT INTO tunnels (id, session_id, type, local_addr, remote_addr, created_at, status, metadata)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `, tunnel.ID, tunnel.SessionID, tunnel.Type, tunnel.LocalAddr, tunnel.RemoteAddr, tunnel.CreatedAt, tunnel.Status, metadata)
	return translatePostgresError(err)
}

func (p *PostgresStore) GetTunnel(ctx context.Context, id string) (*Tunnel, error) {
	var tunnel Tunnel
	var metadata []byte
	err := p.db.QueryRowxContext(ctx, `
        SELECT id, session_id, type, local_addr, remote_addr, created_at, closed_at, bytes_sent, bytes_recv, status, metadata
        FROM tunnels WHERE id = $1
    `, id).Scan(&tunnel.ID, &tunnel.SessionID, &tunnel.Type, &tunnel.LocalAddr, &tunnel.RemoteAddr, &tunnel.CreatedAt, &tunnel.ClosedAt, &tunnel.BytesSent, &tunnel.BytesRecv, &tunnel.Status, &metadata)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	tunnel.Metadata = unmarshalJSONMap(metadata)
	return &tunnel, nil
}

func (p *PostgresStore) UpdateTunnel(ctx context.Context, tunnel *Tunnel) error {
	metadata, err := marshalJSONMap(tunnel.Metadata)
	if err != nil {
		return err
	}
	result, err := p.db.ExecContext(ctx, `
        UPDATE tunnels
        SET bytes_sent = $2, bytes_recv = $3, status = $4, closed_at = $5, metadata = $6
        WHERE id = $1
    `, tunnel.ID, tunnel.BytesSent, tunnel.BytesRecv, tunnel.Status, tunnel.ClosedAt, metadata)
	if err != nil {
		return err
	}
	return requireRowsAffected(result)
}

func (p *PostgresStore) DeleteTunnel(ctx context.Context, id string) error {
	result, err := p.db.ExecContext(ctx, `DELETE FROM tunnels WHERE id = $1`, id)
	if err != nil {
		return err
	}
	return requireRowsAffected(result)
}

func (p *PostgresStore) ListTunnels(ctx context.Context, filter TunnelFilter) ([]*Tunnel, error) {
	query := `SELECT id, session_id, type, local_addr, remote_addr, created_at, closed_at, bytes_sent, bytes_recv, status, metadata FROM tunnels WHERE 1=1`
	args := []any{}
	index := 1
	if filter.SessionID != "" {
		query += fmt.Sprintf(" AND session_id = $%d", index)
		args = append(args, filter.SessionID)
		index++
	}
	if filter.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", index)
		args = append(args, filter.Type)
		index++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", index)
		args = append(args, filter.Status)
		index++
	}
	query += " ORDER BY created_at DESC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", index)
		args = append(args, filter.Limit)
		index++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", index)
		args = append(args, filter.Offset)
	}
	rows, err := p.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*Tunnel, 0)
	for rows.Next() {
		var tunnel Tunnel
		var metadata []byte
		if err := rows.Scan(&tunnel.ID, &tunnel.SessionID, &tunnel.Type, &tunnel.LocalAddr, &tunnel.RemoteAddr, &tunnel.CreatedAt, &tunnel.ClosedAt, &tunnel.BytesSent, &tunnel.BytesRecv, &tunnel.Status, &metadata); err != nil {
			return nil, err
		}
		tunnel.Metadata = unmarshalJSONMap(metadata)
		result = append(result, &tunnel)
	}
	return result, rows.Err()
}

func (p *PostgresStore) DeleteTunnelsBySession(ctx context.Context, sessionID string) ([]string, error) {
	ids := []string{}
	if err := p.db.SelectContext(ctx, &ids, `SELECT id FROM tunnels WHERE session_id = $1`, sessionID); err != nil {
		return nil, err
	}
	if _, err := p.db.ExecContext(ctx, `DELETE FROM tunnels WHERE session_id = $1`, sessionID); err != nil {
		return nil, err
	}
	return ids, nil
}

func (p *PostgresStore) IncrementAttempts(ctx context.Context, key string, window time.Duration) (int, error) {
	cutoff := time.Now().UTC().Add(-window)
	_, err := p.db.ExecContext(ctx, `
        INSERT INTO rate_limits (name, count, first_at)
        VALUES ($1, 1, $2)
        ON CONFLICT (name) DO UPDATE SET
            count = CASE WHEN rate_limits.first_at < $3 THEN 1 ELSE rate_limits.count + 1 END,
            first_at = CASE WHEN rate_limits.first_at < $3 THEN $2 ELSE rate_limits.first_at END
    `, key, time.Now().UTC(), cutoff)
	if err != nil {
		return 0, err
	}
	return p.GetAttempts(ctx, key)
}

func (p *PostgresStore) GetAttempts(ctx context.Context, key string) (int, error) {
	var count int
	err := p.db.GetContext(ctx, &count, `SELECT count FROM rate_limits WHERE name = $1`, key)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return count, err
}

func (p *PostgresStore) ResetAttempts(ctx context.Context, key string) error {
	_, err := p.db.ExecContext(ctx, `DELETE FROM rate_limits WHERE name = $1`, key)
	return err
}

func (p *PostgresStore) IsLocked(ctx context.Context, key string, _ time.Duration) (bool, error) {
	var lockedUntil *time.Time
	err := p.db.GetContext(ctx, &lockedUntil, `SELECT locked_until FROM rate_limits WHERE name = $1`, key)
	if err == sql.ErrNoRows || lockedUntil == nil {
		return false, nil
	}
	return time.Now().UTC().Before(*lockedUntil), err
}

func (p *PostgresStore) Lock(ctx context.Context, key string, duration time.Duration) error {
	now := time.Now().UTC()
	_, err := p.db.ExecContext(ctx, `
        INSERT INTO rate_limits (name, count, first_at, locked_until)
        VALUES ($1, 0, $2, $3)
        ON CONFLICT (name) DO UPDATE SET locked_until = $3
    `, key, now, now.Add(duration))
	return err
}

func (p *PostgresStore) AppendAuditLog(ctx context.Context, entry *AuditEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	entry.Timestamp = time.Now().UTC()
	details, err := marshalJSONMap(entry.Details)
	if err != nil {
		return err
	}
	_, err = p.db.ExecContext(ctx, `
        INSERT INTO audit_log (id, timestamp, correlation_id, trace_id, action, username, role, resource, result, details, source_ip)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    `, entry.ID, entry.Timestamp, entry.CorrelationID, entry.TraceID, entry.Action, entry.Username, entry.Role, entry.Resource, entry.Result, details, entry.SourceIP)
	return translatePostgresError(err)
}

func (p *PostgresStore) QueryAuditLog(ctx context.Context, filter AuditFilter) ([]*AuditEntry, error) {
	query := `SELECT id, timestamp, correlation_id, trace_id, action, username, role, resource, result, details, source_ip FROM audit_log WHERE 1=1`
	args := []any{}
	index := 1
	if filter.Username != "" {
		query += fmt.Sprintf(" AND username = $%d", index)
		args = append(args, filter.Username)
		index++
	}
	if filter.Action != "" {
		query += fmt.Sprintf(" AND action = $%d", index)
		args = append(args, filter.Action)
		index++
	}
	if filter.Result != "" {
		query += fmt.Sprintf(" AND result = $%d", index)
		args = append(args, filter.Result)
		index++
	}
	if filter.Since != nil {
		query += fmt.Sprintf(" AND timestamp >= $%d", index)
		args = append(args, *filter.Since)
		index++
	}
	if filter.Until != nil {
		query += fmt.Sprintf(" AND timestamp <= $%d", index)
		args = append(args, *filter.Until)
		index++
	}
	query += " ORDER BY timestamp DESC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", index)
		args = append(args, filter.Limit)
		index++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", index)
		args = append(args, filter.Offset)
	}
	rows, err := p.db.QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]*AuditEntry, 0)
	for rows.Next() {
		var entry AuditEntry
		var details []byte
		if err := rows.Scan(&entry.ID, &entry.Timestamp, &entry.CorrelationID, &entry.TraceID, &entry.Action, &entry.Username, &entry.Role, &entry.Resource, &entry.Result, &details, &entry.SourceIP); err != nil {
			return nil, err
		}
		entry.Details = unmarshalJSONMap(details)
		result = append(result, &entry)
	}
	return result, rows.Err()
}

func (p *PostgresStore) CleanupInactiveSessions(ctx context.Context, maxInactive time.Duration) (int, error) {
	cutoff := time.Now().UTC().Add(-maxInactive)
	result, err := p.db.ExecContext(ctx, `
        UPDATE sessions SET status = 'expired', closed_at = $2
        WHERE status = 'active' AND last_active < $1
    `, cutoff, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}

func (p *PostgresStore) CleanupOldAuditLogs(ctx context.Context, maxAge time.Duration) (int, error) {
	cutoff := time.Now().UTC().Add(-maxAge)
	result, err := p.db.ExecContext(ctx, `DELETE FROM audit_log WHERE timestamp < $1`, cutoff)
	if err != nil {
		return 0, err
	}
	rows, _ := result.RowsAffected()
	return int(rows), nil
}

func (p *PostgresStore) Health(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

func (p *PostgresStore) Close() error {
	return p.db.Close()
}

func marshalJSONMap(value JSONMap) ([]byte, error) {
	if value == nil {
		value = JSONMap{}
	}
	return json.Marshal(value)
}

func unmarshalJSONMap(data []byte) JSONMap {
	result := JSONMap{}
	_ = json.Unmarshal(data, &result)
	return result
}

func requireRowsAffected(result sql.Result) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func translatePostgresError(err error) error {
	if err == nil {
		return nil
	}
	return err
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
