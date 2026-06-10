package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
	memory *MemoryStore
}

func NewRedisStore(cfg RedisConfig) (*RedisStore, error) {
	if cfg.Addr == "" {
		cfg.Addr = "localhost:6379"
	}
	if cfg.PoolSize <= 0 {
		cfg.PoolSize = 10
	}
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("conectando a Redis: %w", err)
	}
	return &RedisStore{client: client, memory: NewMemoryStore()}, nil
}

func (r *RedisStore) CreateSession(ctx context.Context, sess *Session) error {
	if sess.ID == "" {
		sess.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	sess.CreatedAt = now
	sess.LastActive = now
	if sess.Status == "" {
		sess.Status = "active"
	}
	data, err := json.Marshal(sess)
	if err != nil {
		return err
	}
	pipe := r.client.TxPipeline()
	pipe.Set(ctx, sessionKey(sess.ID), data, 24*time.Hour)
	pipe.SAdd(ctx, userSessionsKey(sess.Username), sess.ID)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *RedisStore) GetSession(ctx context.Context, id string) (*Session, error) {
	data, err := r.client.Get(ctx, sessionKey(id)).Bytes()
	if err == redis.Nil {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func (r *RedisStore) UpdateSession(ctx context.Context, sess *Session) error {
	sess.LastActive = time.Now().UTC()
	data, err := json.Marshal(sess)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, sessionKey(sess.ID), data, 24*time.Hour).Err()
}

func (r *RedisStore) DeleteSession(ctx context.Context, id string) error {
	sess, err := r.GetSession(ctx, id)
	if err != nil {
		return err
	}
	pipe := r.client.TxPipeline()
	pipe.Del(ctx, sessionKey(id))
	pipe.SRem(ctx, userSessionsKey(sess.Username), id)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *RedisStore) ListSessions(ctx context.Context, filter SessionFilter) ([]*Session, error) {
	if filter.Username == "" {
		return r.memory.ListSessions(ctx, filter)
	}
	ids, err := r.client.SMembers(ctx, userSessionsKey(filter.Username)).Result()
	if err != nil {
		return nil, err
	}
	result := make([]*Session, 0, len(ids))
	for _, id := range ids {
		sess, err := r.GetSession(ctx, id)
		if err != nil {
			continue
		}
		if filter.Status != "" && sess.Status != filter.Status {
			continue
		}
		result = append(result, sess)
	}
	return applySessionWindow(result, filter.Limit, filter.Offset), nil
}

func (r *RedisStore) CountActiveSessions(ctx context.Context, username string) (int, error) {
	sessions, err := r.ListSessions(ctx, SessionFilter{Username: username, Status: "active"})
	if err != nil {
		return 0, err
	}
	return len(sessions), nil
}

func (r *RedisStore) TouchSession(ctx context.Context, id string) error {
	sess, err := r.GetSession(ctx, id)
	if err != nil {
		return err
	}
	return r.UpdateSession(ctx, sess)
}

func (r *RedisStore) CreateTunnel(ctx context.Context, tunnel *Tunnel) error {
	if tunnel.ID == "" {
		tunnel.ID = uuid.New().String()
	}
	tunnel.CreatedAt = time.Now().UTC()
	if tunnel.Status == "" {
		tunnel.Status = "active"
	}
	data, err := json.Marshal(tunnel)
	if err != nil {
		return err
	}
	pipe := r.client.TxPipeline()
	pipe.Set(ctx, tunnelKey(tunnel.ID), data, 24*time.Hour)
	pipe.SAdd(ctx, sessionTunnelsKey(tunnel.SessionID), tunnel.ID)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *RedisStore) GetTunnel(ctx context.Context, id string) (*Tunnel, error) {
	data, err := r.client.Get(ctx, tunnelKey(id)).Bytes()
	if err == redis.Nil {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var tunnel Tunnel
	if err := json.Unmarshal(data, &tunnel); err != nil {
		return nil, err
	}
	return &tunnel, nil
}

func (r *RedisStore) UpdateTunnel(ctx context.Context, tunnel *Tunnel) error {
	data, err := json.Marshal(tunnel)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, tunnelKey(tunnel.ID), data, 24*time.Hour).Err()
}

func (r *RedisStore) DeleteTunnel(ctx context.Context, id string) error {
	tunnel, err := r.GetTunnel(ctx, id)
	if err != nil {
		return err
	}
	pipe := r.client.TxPipeline()
	pipe.Del(ctx, tunnelKey(id))
	pipe.SRem(ctx, sessionTunnelsKey(tunnel.SessionID), id)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *RedisStore) ListTunnels(ctx context.Context, filter TunnelFilter) ([]*Tunnel, error) {
	if filter.SessionID == "" {
		return r.memory.ListTunnels(ctx, filter)
	}
	ids, err := r.client.SMembers(ctx, sessionTunnelsKey(filter.SessionID)).Result()
	if err != nil {
		return nil, err
	}
	result := make([]*Tunnel, 0, len(ids))
	for _, id := range ids {
		tunnel, err := r.GetTunnel(ctx, id)
		if err != nil {
			continue
		}
		if filter.Type != "" && tunnel.Type != filter.Type {
			continue
		}
		if filter.Status != "" && tunnel.Status != filter.Status {
			continue
		}
		result = append(result, tunnel)
	}
	return applyTunnelWindow(result, filter.Limit, filter.Offset), nil
}

func (r *RedisStore) DeleteTunnelsBySession(ctx context.Context, sessionID string) ([]string, error) {
	ids, err := r.client.SMembers(ctx, sessionTunnelsKey(sessionID)).Result()
	if err != nil {
		return nil, err
	}
	pipe := r.client.TxPipeline()
	for _, id := range ids {
		pipe.Del(ctx, tunnelKey(id))
	}
	pipe.Del(ctx, sessionTunnelsKey(sessionID))
	_, err = pipe.Exec(ctx)
	return ids, err
}

func (r *RedisStore) IncrementAttempts(ctx context.Context, key string, window time.Duration) (int, error) {
	redisKey := rateLimitKey(key)
	script := redis.NewScript(`
local key = KEYS[1]
local ttl = tonumber(ARGV[1])
local current = redis.call('GET', key)
if current == false then
  redis.call('SET', key, 1, 'EX', ttl)
  return 1
end
return redis.call('INCR', key)
`)
	value, err := script.Run(ctx, r.client, []string{redisKey}, int(window.Seconds())).Int()
	if err != nil {
		return 0, err
	}
	return value, nil
}

func (r *RedisStore) GetAttempts(ctx context.Context, key string) (int, error) {
	value, err := r.client.Get(ctx, rateLimitKey(key)).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return value, err
}

func (r *RedisStore) ResetAttempts(ctx context.Context, key string) error {
	pipe := r.client.TxPipeline()
	pipe.Del(ctx, rateLimitKey(key))
	pipe.Del(ctx, rateLimitLockKey(key))
	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisStore) IsLocked(ctx context.Context, key string, _ time.Duration) (bool, error) {
	ttl, err := r.client.TTL(ctx, rateLimitLockKey(key)).Result()
	if err != nil {
		return false, err
	}
	return ttl > 0, nil
}

func (r *RedisStore) Lock(ctx context.Context, key string, duration time.Duration) error {
	return r.client.Set(ctx, rateLimitLockKey(key), "locked", duration).Err()
}

func (r *RedisStore) AppendAuditLog(ctx context.Context, entry *AuditEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	entry.Timestamp = time.Now().UTC()
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	return r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: "audit:log",
		Values: map[string]any{"data": string(data)},
		MaxLen: 10000,
	}).Err()
}

func (r *RedisStore) QueryAuditLog(ctx context.Context, filter AuditFilter) ([]*AuditEntry, error) {
	entries, err := r.client.XRevRangeN(ctx, "audit:log", "+", "-", int64(limitOrDefault(filter.Limit, 1000))).Result()
	if err != nil {
		return nil, err
	}
	result := make([]*AuditEntry, 0)
	for _, streamEntry := range entries {
		raw, ok := streamEntry.Values["data"].(string)
		if !ok {
			continue
		}
		var entry AuditEntry
		if err := json.Unmarshal([]byte(raw), &entry); err != nil {
			continue
		}
		if filter.Username != "" && entry.Username != filter.Username {
			continue
		}
		if filter.Action != "" && entry.Action != filter.Action {
			continue
		}
		if filter.Result != "" && entry.Result != filter.Result {
			continue
		}
		result = append(result, &entry)
	}
	return applyAuditWindow(result, filter.Limit, filter.Offset), nil
}

func (r *RedisStore) CleanupInactiveSessions(_ context.Context, _ time.Duration) (int, error) {
	return 0, nil
}

func (r *RedisStore) CleanupOldAuditLogs(_ context.Context, _ time.Duration) (int, error) {
	return 0, nil
}

func (r *RedisStore) Health(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *RedisStore) Close() error {
	return r.client.Close()
}

func sessionKey(id string) string            { return fmt.Sprintf("session:%s", id) }
func userSessionsKey(username string) string { return fmt.Sprintf("user:sessions:%s", username) }
func tunnelKey(id string) string             { return fmt.Sprintf("tunnel:%s", id) }
func sessionTunnelsKey(id string) string     { return fmt.Sprintf("session:tunnels:%s", id) }
func rateLimitKey(key string) string         { return fmt.Sprintf("ratelimit:%s", key) }
func rateLimitLockKey(key string) string     { return fmt.Sprintf("ratelimit:lock:%s", key) }

func limitOrDefault(limit, fallback int) int {
	if limit > 0 {
		return limit
	}
	return fallback
}
