#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

# shellcheck source=/dev/null
source "${ROOT_DIR}/scripts/env-local.sh"
ensure_env_local "${ROOT_DIR}"
load_env_local "${ROOT_DIR}"

mkdir -p reports
LOG_FILE="reports/e2e-server.log"
OUT_FILE="reports/e2e-session.txt"
RUST_BIN="rust/input-guard/target/release/input-guard"

if [[ -z "${LAB_PASSWORD_STUDENT:-}" ]]; then
  echo "Falta LAB_PASSWORD_STUDENT. Ejecuta make generate-secrets." >&2
  exit 1
fi

cleanup() {
  if [[ -n "${SERVER_PID:-}" ]]; then
    kill "${SERVER_PID}" >/dev/null 2>&1 || true
    wait "${SERVER_PID}" 2>/dev/null || true
  fi
}
trap cleanup EXIT

cargo build --release --manifest-path rust/input-guard/Cargo.toml

APP_PROFILE=hardened APP_PROJECT_ROOT=. EXTERNAL_AUDIT_ENABLED=true EXTERNAL_AUDIT_COMMAND=python3 EXTERNAL_AUDIT_SCRIPT=tools/audit/audit.py EXTERNAL_VALIDATOR_ENABLED=true EXTERNAL_VALIDATOR_BINARY="${RUST_BIN}" EXTERNAL_VALIDATOR_FAIL_OPEN=false OPA_POLICY_ENABLED=true OPA_BINARY=opa OPA_POLICY_DIR=policies/kubernetes go run ./cmd/server >"${LOG_FILE}" 2>&1 &
SERVER_PID=$!

sleep 2

{
  sleep 0.5; echo "student"
  sleep 0.5; echo "${LAB_PASSWORD_STUDENT}"
  sleep 0.5; echo "whoami"
  sleep 0.5; echo "audit json"
  sleep 0.5; echo "policy json"
  sleep 0.5; echo "quit"
} | nc 127.0.0.1 2323 > "${OUT_FILE}"

grep -q "username=student" "${OUT_FILE}"
grep -q '"status"' "${OUT_FILE}"

echo "Prueba e2e completada."
echo "Salida: ${OUT_FILE}"
echo "Logs:   ${LOG_FILE}"
