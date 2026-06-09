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
