#!/usr/bin/env bash
# Genera .env.local con credenciales aleatorias para desarrollo.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
ENV_LOCAL="${PROJECT_ROOT}/.env.local"
ENV_EXAMPLE="${PROJECT_ROOT}/.env.example"

log_info() {
  printf '[INFO] %s\n' "$1"
}

log_warn() {
  printf '[WARN] %s\n' "$1"
}

log_error() {
  printf '[ERROR] %s\n' "$1" >&2
}

generate_password() {
  local password
  password="$(LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 24 || true)"
  printf '%s' "${password}"
}

generate_lab_password() {
  local words=(alpha beta gamma delta sigma omega nova pulse core flux)
  local word1="${words[$((RANDOM % ${#words[@]}))]}"
  local word2="${words[$((RANDOM % ${#words[@]}))]}"
  local number=$((1000 + RANDOM % 9000))
  printf '%s-%s-%s' "${word1}" "${number}" "${word2}"
}

if [[ ! -f "${ENV_EXAMPLE}" ]]; then
  log_error "No se encontró ${ENV_EXAMPLE}"
  exit 1
fi

log_info "Generando credenciales locales para SentinelOps"

ADMIN_API_PASS="$(generate_password)"
STUDENT_PASS="$(generate_lab_password)"
TEACHER_PASS="$(generate_lab_password)"
AUDITOR_PASS="$(generate_lab_password)"
ADMIN_PASS="$(generate_lab_password)"

cat > "${ENV_LOCAL}" <<EOF_ENV
# SentinelOps - entorno local generado
# Generado: $(date -u +"%Y-%m-%dT%H:%M:%SZ")
# No versionar este archivo.

APP_ENV=development
APP_PROFILE=hardened
APP_TRANSPORT=ssh
APP_ADDR=:2324
APP_SSH_ADDR=:2222
APP_SSH_HOST_KEY_PATH=data/ssh/host_ed25519_key
METRICS_ADDR=:9001

OTEL_TRACES_ENABLED=false
OTEL_EXPORTER_TYPE=stdout
OTEL_EXPORTER_ENDPOINT=localhost:4317
OTEL_EXPORTER_INSECURE=true
OTEL_SAMPLE_RATE=1.0

APP_CONTROL_API_ENABLED=true
APP_CONTROL_API_ADDR=:9443
APP_CONTROL_API_USER=admin
APP_CONTROL_API_PASSWORD=${ADMIN_API_PASS}

LAB_USER_STUDENT=student
LAB_PASSWORD_STUDENT=${STUDENT_PASS}
LAB_USER_TEACHER=teacher
LAB_PASSWORD_TEACHER=${TEACHER_PASS}
LAB_USER_AUDITOR=auditor
LAB_PASSWORD_AUDITOR=${AUDITOR_PASS}
LAB_USER_ADMIN=admin
LAB_PASSWORD_ADMIN=${ADMIN_PASS}

APP_AUTH_RATE_LIMIT_ENABLED=true
APP_AUTH_RATE_LIMIT_MAX_FAILURES=5
APP_AUTH_RATE_LIMIT_WINDOW=1m
APP_AUTH_RATE_LIMIT_LOCKOUT=1m

APP_SSH_LOCAL_FORWARD_ENABLED=true
APP_SSH_REMOTE_FORWARD_ENABLED=false
APP_SSH_FORWARD_ALLOWLIST=127.0.0.1:9001,localhost:9001
APP_SSH_LOCAL_ALLOWED_ROLES=student,teacher,auditor,admin
APP_SSH_REMOTE_BIND_ALLOWLIST=127.0.0.1:10080,127.0.0.1:10443
APP_SSH_REMOTE_ALLOWED_ROLES=teacher,auditor,admin

EXTERNAL_AUDIT_ENABLED=true
EXTERNAL_AUDIT_COMMAND=python3
EXTERNAL_AUDIT_SCRIPT=tools/audit/audit.py
EXTERNAL_VALIDATOR_ENABLED=true
EXTERNAL_VALIDATOR_BINARY=rust/input-guard/target/release/input-guard
EXTERNAL_VALIDATOR_FAIL_OPEN=false
OPA_POLICY_ENABLED=true
OPA_BINARY=opa
OPA_POLICY_DIR=policies/kubernetes
OPA_POLICY_MODE=exec
OPA_POLICY_URL=http://localhost:8181
OPA_POLICY_TIMEOUT=2s
OPA_POLICY_CACHE_ENABLED=true
OPA_POLICY_CACHE_TTL=30s

APP_STATE_PERSISTENCE_ENABLED=false
APP_STATE_PERSISTENCE_DIR=data/state
APP_STATE_SESSIONS_PATH=data/state/sessions.json
APP_STATE_TUNNELS_PATH=data/state/tunnels.json
EOF_ENV

chmod 600 "${ENV_LOCAL}"

log_info "Credenciales generadas en .env.local"
printf '\nCredenciales de laboratorio\n'
printf -- '------------------------------------------------------------\n'
printf 'API de control: admin / %s\n' "${ADMIN_API_PASS}"
printf 'student: %s\n' "${STUDENT_PASS}"
printf 'teacher: %s\n' "${TEACHER_PASS}"
printf 'auditor: %s\n' "${AUDITOR_PASS}"
printf 'admin: %s\n' "${ADMIN_PASS}"
printf -- '------------------------------------------------------------\n\n'
log_warn "Guarda estas credenciales. El archivo .env.local no debe versionarse."
log_info "Uso sugerido: source .env.local && make run-ssh"
