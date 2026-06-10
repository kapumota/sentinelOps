package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func isolateEnvFile(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV_FILE", filepath.Join(t.TempDir(), "missing.env"))
	t.Setenv("APP_CONTROL_API_PASSWORD", "")
	t.Setenv("APP_CONTROL_API_USER", "")
}

func TestLoadSecureDefaults(t *testing.T) {
	isolateEnvFile(t)
	t.Setenv("APP_SSH_REMOTE_FORWARD_ENABLED", "")
	t.Setenv("EXTERNAL_VALIDATOR_FAIL_OPEN", "")
	t.Setenv("APP_STATE_PERSISTENCE_ENABLED", "")

	cfg := Load()
	if cfg.SSHRemoteForwardEnabled {
		t.Fatal("remote SSH forwarding must be disabled by default")
	}
	if cfg.ExternalValidatorOpen {
		t.Fatal("external validator must fail closed by default")
	}
	if !cfg.AuthRateLimitEnabled {
		t.Fatal("auth rate limiting must be enabled by default")
	}
	if cfg.StatePersistenceEnabled {
		t.Fatal("state persistence must be opt-in by default")
	}
}

func TestLoadBooleanOverrides(t *testing.T) {
	isolateEnvFile(t)
	t.Setenv("APP_SSH_REMOTE_FORWARD_ENABLED", "true")
	t.Setenv("EXTERNAL_VALIDATOR_FAIL_OPEN", "true")
	t.Setenv("APP_STATE_PERSISTENCE_ENABLED", "true")

	cfg := Load()
	if !cfg.SSHRemoteForwardEnabled {
		t.Fatal("expected remote SSH forwarding override to be honored")
	}
	if !cfg.ExternalValidatorOpen {
		t.Fatal("expected validator fail-open override to be honored")
	}
	if !cfg.StatePersistenceEnabled {
		t.Fatal("expected persistence override to be honored")
	}
}

func TestLoadRateLimitOverrides(t *testing.T) {
	isolateEnvFile(t)
	t.Setenv("APP_AUTH_RATE_LIMIT_ENABLED", "false")
	t.Setenv("APP_AUTH_RATE_LIMIT_MAX_FAILURES", "7")
	t.Setenv("APP_AUTH_RATE_LIMIT_WINDOW", "2m")
	t.Setenv("APP_AUTH_RATE_LIMIT_LOCKOUT", "30s")

	cfg := Load()
	if cfg.AuthRateLimitEnabled {
		t.Fatal("expected rate limit enabled override to be honored")
	}
	if cfg.AuthRateLimitMaxFailures != 7 {
		t.Fatalf("expected max failures 7, got %d", cfg.AuthRateLimitMaxFailures)
	}
	if cfg.AuthRateLimitWindow != 2*time.Minute {
		t.Fatalf("expected 2m window, got %s", cfg.AuthRateLimitWindow)
	}
	if cfg.AuthRateLimitLockout != 30*time.Second {
		t.Fatalf("expected 30s lockout, got %s", cfg.AuthRateLimitLockout)
	}
}

func TestLoadStatePathDerivedFromDir(t *testing.T) {
	isolateEnvFile(t)
	t.Setenv("APP_STATE_PERSISTENCE_DIR", "/tmp/sentinelops-state")
	t.Setenv("APP_STATE_SESSIONS_PATH", "")
	t.Setenv("APP_STATE_TUNNELS_PATH", "")

	cfg := Load()
	if cfg.StateSessionsPath != filepath.Join("/tmp/sentinelops-state", "sessions.json") {
		t.Fatalf("unexpected sessions path: %s", cfg.StateSessionsPath)
	}
	if cfg.StateTunnelsPath != filepath.Join("/tmp/sentinelops-state", "tunnels.json") {
		t.Fatalf("unexpected tunnels path: %s", cfg.StateTunnelsPath)
	}
}

func TestLoadEnvFile(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env.local")
	content := "APP_CONTROL_API_PASSWORD=control-secret\nAPP_CONTROL_API_USER=operator\n"
	if err := os.WriteFile(envFile, []byte(content), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	t.Setenv("APP_ENV_FILE", envFile)
	t.Setenv("APP_CONTROL_API_PASSWORD", "")
	t.Setenv("APP_CONTROL_API_USER", "")

	cfg := Load()
	if cfg.ControlAPIUser != "operator" {
		t.Fatalf("expected control user from env file, got %s", cfg.ControlAPIUser)
	}
	if cfg.ControlAPIPassword != "control-secret" {
		t.Fatal("expected control password from env file")
	}
}

func TestLoadGeneratesRandomControlPassword(t *testing.T) {
	isolateEnvFile(t)
	t.Setenv("APP_CONTROL_API_PASSWORD", "")

	cfg1 := Load()
	cfg2 := Load()

	if cfg1.ControlAPIPassword == "" || cfg2.ControlAPIPassword == "" {
		t.Fatal("expected generated control API passwords")
	}
	if cfg1.ControlAPIPassword == cfg2.ControlAPIPassword {
		t.Fatal("expected generated control API passwords to differ")
	}
}

func TestLoadPolicySidecarDefaults(t *testing.T) {
	isolateEnvFile(t)
	t.Setenv("OPA_POLICY_MODE", "")
	t.Setenv("OPA_POLICY_URL", "")
	t.Setenv("OPA_POLICY_TIMEOUT", "")
	t.Setenv("OPA_POLICY_CACHE_ENABLED", "")
	t.Setenv("OPA_POLICY_CACHE_TTL", "")

	cfg := Load()
	if cfg.PolicyMode != "exec" {
		t.Fatalf("expected default OPA policy mode exec, got %s", cfg.PolicyMode)
	}
	if cfg.PolicyURL != "http://localhost:8181" {
		t.Fatalf("unexpected OPA policy URL: %s", cfg.PolicyURL)
	}
	if cfg.PolicyTimeout != 2*time.Second {
		t.Fatalf("unexpected OPA policy timeout: %s", cfg.PolicyTimeout)
	}
	if !cfg.PolicyCacheEnabled {
		t.Fatal("expected OPA policy cache enabled by default")
	}
	if cfg.PolicyCacheTTL != 30*time.Second {
		t.Fatalf("unexpected OPA policy cache TTL: %s", cfg.PolicyCacheTTL)
	}
}

func TestLoadPolicySidecarOverrides(t *testing.T) {
	isolateEnvFile(t)
	t.Setenv("OPA_POLICY_MODE", "http")
	t.Setenv("OPA_POLICY_URL", "http://opa:8181")
	t.Setenv("OPA_POLICY_TIMEOUT", "5s")
	t.Setenv("OPA_POLICY_CACHE_ENABLED", "false")
	t.Setenv("OPA_POLICY_CACHE_TTL", "2m")

	cfg := Load()
	if cfg.PolicyMode != "http" {
		t.Fatalf("expected OPA policy mode http, got %s", cfg.PolicyMode)
	}
	if cfg.PolicyURL != "http://opa:8181" {
		t.Fatalf("unexpected OPA policy URL: %s", cfg.PolicyURL)
	}
	if cfg.PolicyTimeout != 5*time.Second {
		t.Fatalf("unexpected OPA policy timeout: %s", cfg.PolicyTimeout)
	}
	if cfg.PolicyCacheEnabled {
		t.Fatal("expected OPA policy cache override to disable cache")
	}
	if cfg.PolicyCacheTTL != 2*time.Minute {
		t.Fatalf("unexpected OPA policy cache TTL: %s", cfg.PolicyCacheTTL)
	}
}

func TestLoadValidatorGRPCOverrides(t *testing.T) {
	isolateEnvFile(t)
	t.Setenv("VALIDATOR_MODE", "grpc")
	t.Setenv("VALIDATOR_GRPC_ADDR", "input-guard:50051")
	t.Setenv("VALIDATOR_GRPC_TIMEOUT", "3s")
	t.Setenv("VALIDATOR_GRPC_FAIL_OPEN", "true")

	cfg := Load()
	if cfg.ValidatorMode != "grpc" {
		t.Fatalf("expected grpc validator mode, got %s", cfg.ValidatorMode)
	}
	if cfg.ValidatorGRPCAddr != "input-guard:50051" {
		t.Fatalf("unexpected grpc addr: %s", cfg.ValidatorGRPCAddr)
	}
	if cfg.ValidatorGRPCTimeout != 3*time.Second {
		t.Fatalf("unexpected grpc timeout: %s", cfg.ValidatorGRPCTimeout)
	}
	if !cfg.ValidatorGRPCFailOpen {
		t.Fatal("expected grpc fail-open override to be honored")
	}
}

func TestLoadStorageDefaults(t *testing.T) {
	isolateEnvFile(t)
	t.Setenv("STORE_TYPE", "")
	t.Setenv("POSTGRES_PORT", "")
	t.Setenv("REDIS_DB", "")

	cfg := Load()
	if cfg.StoreType != "memory" {
		t.Fatalf("expected memory store by default, got %s", cfg.StoreType)
	}
	if cfg.PostgresPort != 5432 {
		t.Fatalf("unexpected postgres port: %d", cfg.PostgresPort)
	}
	if cfg.RedisAddr != "localhost:6379" {
		t.Fatalf("unexpected redis addr: %s", cfg.RedisAddr)
	}
}

func TestLoadStorageOverrides(t *testing.T) {
	isolateEnvFile(t)
	t.Setenv("STORE_TYPE", "postgres")
	t.Setenv("POSTGRES_HOST", "postgres")
	t.Setenv("POSTGRES_PORT", "55432")
	t.Setenv("POSTGRES_DB", "sentinelops_test")
	t.Setenv("POSTGRES_USER", "operator")
	t.Setenv("POSTGRES_PASSWORD", "secret")
	t.Setenv("POSTGRES_SSLMODE", "require")
	t.Setenv("POSTGRES_POOL_SIZE", "20")
	t.Setenv("REDIS_ADDR", "redis:6379")
	t.Setenv("REDIS_PASSWORD", "redis-secret")
	t.Setenv("REDIS_DB", "2")
	t.Setenv("REDIS_POOL_SIZE", "30")

	cfg := Load()
	if cfg.StoreType != "postgres" {
		t.Fatalf("expected postgres store, got %s", cfg.StoreType)
	}
	if cfg.PostgresHost != "postgres" || cfg.PostgresPort != 55432 || cfg.PostgresDB != "sentinelops_test" {
		t.Fatal("postgres overrides not honored")
	}
	if cfg.PostgresUser != "operator" || cfg.PostgresPassword != "secret" || cfg.PostgresSSLMode != "require" || cfg.PostgresPoolSize != 20 {
		t.Fatal("postgres security overrides not honored")
	}
	if cfg.RedisAddr != "redis:6379" || cfg.RedisPassword != "redis-secret" || cfg.RedisDB != 2 || cfg.RedisPoolSize != 30 {
		t.Fatal("redis overrides not honored")
	}
}
