package config

import (
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	AppName                string
	Environment            string
	Profile                string
	Transport              string
	ListenAddr             string
	SSHListenAddr          string
	SSHHostKeyPath         string
	SSHServerVersion       string
	SSHAuthorizedKeysDir   string
	SSHPasswordAuthEnabled bool
	SSHPublicKeyAuthEnable bool

	SSHLocalForwardEnabled bool
	SSHForwardAllowlist    string
	SSHLocalAllowedRoles   string

	SSHRemoteForwardEnabled bool
	SSHRemoteBindAllowlist  string
	SSHRemoteAllowedRoles   string

	ControlAPIEnabled   bool
	ControlAPIAddr      string
	ControlAPICertPath  string
	ControlAPIKeyPath   string
	ControlAPIUser      string
	ControlAPIPassword  string
	ControlAPICertHosts string

	MetricsAddr              string
	LogLevel                 string
	Banner                   string
	ReadTimeout              time.Duration
	IdleTimeout              time.Duration
	WriteTimeout             time.Duration
	AuthEnabled              bool
	AuthMaxAttempts          int
	AuthRateLimitEnabled     bool
	AuthRateLimitMaxFailures int
	AuthRateLimitWindow      time.Duration
	AuthRateLimitLockout     time.Duration
	ProjectRoot              string
	StatePersistenceEnabled  bool
	StatePersistenceDir      string
	StateSessionsPath        string
	StateTunnelsPath         string
	ExternalAuditEnabled     bool
	ExternalAuditCommand     string
	ExternalAuditScript      string
	ExternalValidatorOn      bool
	ExternalValidatorBin     string
	ExternalValidatorOpen    bool
	PolicyEnabled            bool
	PolicyBinary             string
	PolicyDir                string
}

func Load() Config {
	stateDir := getEnv("APP_STATE_PERSISTENCE_DIR", "data/state")

	return Config{
		AppName:                getEnv("APP_NAME", "sentinelops"),
		Environment:            getEnv("APP_ENV", "dev"),
		Profile:                getEnv("APP_PROFILE", "hardened"),
		Transport:              getEnv("APP_TRANSPORT", "tcp"),
		ListenAddr:             getEnv("APP_ADDR", ":2323"),
		SSHListenAddr:          getEnv("APP_SSH_ADDR", ":2222"),
		SSHHostKeyPath:         getEnv("APP_SSH_HOST_KEY_PATH", "data/ssh/host_ed25519_key"),
		SSHServerVersion:       getEnv("APP_SSH_SERVER_VERSION", "SSH-2.0-SentinelOps"),
		SSHAuthorizedKeysDir:   getEnv("APP_SSH_AUTHORIZED_KEYS_DIR", "data/ssh/authorized_keys"),
		SSHPasswordAuthEnabled: getBool("APP_SSH_PASSWORD_AUTH_ENABLED", true),
		SSHPublicKeyAuthEnable: getBool("APP_SSH_PUBLICKEY_AUTH_ENABLED", true),

		SSHLocalForwardEnabled: getBool("APP_SSH_LOCAL_FORWARD_ENABLED", true),
		SSHForwardAllowlist:    getEnv("APP_SSH_FORWARD_ALLOWLIST", "127.0.0.1:9000,localhost:9000"),
		SSHLocalAllowedRoles:   getEnv("APP_SSH_LOCAL_ALLOWED_ROLES", "student,teacher,auditor,admin"),

		SSHRemoteForwardEnabled: getBool("APP_SSH_REMOTE_FORWARD_ENABLED", false),
		SSHRemoteBindAllowlist:  getEnv("APP_SSH_REMOTE_BIND_ALLOWLIST", "127.0.0.1:10080,127.0.0.1:10443"),
		SSHRemoteAllowedRoles:   getEnv("APP_SSH_REMOTE_ALLOWED_ROLES", "teacher,auditor,admin"),

		ControlAPIEnabled:   getBool("APP_CONTROL_API_ENABLED", true),
		ControlAPIAddr:      getEnv("APP_CONTROL_API_ADDR", ":9443"),
		ControlAPICertPath:  getEnv("APP_CONTROL_API_CERT_PATH", "data/controlplane/tls.crt"),
		ControlAPIKeyPath:   getEnv("APP_CONTROL_API_KEY_PATH", "data/controlplane/tls.key"),
		ControlAPIUser:      getEnv("APP_CONTROL_API_USER", "admin"),
		ControlAPIPassword:  getEnv("APP_CONTROL_API_PASSWORD", "admin123!"),
		ControlAPICertHosts: getEnv("APP_CONTROL_API_CERT_HOSTS", "localhost,127.0.0.1"),

		MetricsAddr:              getEnv("METRICS_ADDR", ":9000"),
		LogLevel:                 getEnv("LOG_LEVEL", "info"),
		Banner:                   getEnv("APP_BANNER", "SentinelOps - Laboratorio Seguro de Acceso Remoto"),
		ReadTimeout:              getDuration("READ_TIMEOUT", 30*time.Second),
		IdleTimeout:              getDuration("IDLE_TIMEOUT", 5*time.Minute),
		WriteTimeout:             getDuration("WRITE_TIMEOUT", 10*time.Second),
		AuthEnabled:              getBool("APP_AUTH_ENABLED", true),
		AuthMaxAttempts:          getInt("APP_AUTH_MAX_ATTEMPTS", 3),
		AuthRateLimitEnabled:     getBool("APP_AUTH_RATE_LIMIT_ENABLED", true),
		AuthRateLimitMaxFailures: getInt("APP_AUTH_RATE_LIMIT_MAX_FAILURES", 5),
		AuthRateLimitWindow:      getDuration("APP_AUTH_RATE_LIMIT_WINDOW", 1*time.Minute),
		AuthRateLimitLockout:     getDuration("APP_AUTH_RATE_LIMIT_LOCKOUT", 1*time.Minute),
		ProjectRoot:              getEnv("APP_PROJECT_ROOT", "."),
		StatePersistenceEnabled:  getBool("APP_STATE_PERSISTENCE_ENABLED", false),
		StatePersistenceDir:      stateDir,
		StateSessionsPath:        getEnv("APP_STATE_SESSIONS_PATH", filepath.Join(stateDir, "sessions.json")),
		StateTunnelsPath:         getEnv("APP_STATE_TUNNELS_PATH", filepath.Join(stateDir, "tunnels.json")),
		ExternalAuditEnabled:     getBool("EXTERNAL_AUDIT_ENABLED", true),
		ExternalAuditCommand:     getEnv("EXTERNAL_AUDIT_COMMAND", "python3"),
		ExternalAuditScript:      getEnv("EXTERNAL_AUDIT_SCRIPT", "tools/audit/audit.py"),
		ExternalValidatorOn:      getBool("EXTERNAL_VALIDATOR_ENABLED", true),
		ExternalValidatorBin:     getEnv("EXTERNAL_VALIDATOR_BINARY", "rust/input-guard/target/release/input-guard"),
		ExternalValidatorOpen:    getBool("EXTERNAL_VALIDATOR_FAIL_OPEN", false),
		PolicyEnabled:            getBool("OPA_POLICY_ENABLED", true),
		PolicyBinary:             getEnv("OPA_BINARY", "opa"),
		PolicyDir:                getEnv("OPA_POLICY_DIR", "policies/kubernetes"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		d, err := time.ParseDuration(value)
		if err == nil {
			return d
		}
	}
	return fallback
}

func getBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func getInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
	}
	return fallback
}
