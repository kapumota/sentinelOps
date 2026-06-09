#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

# shellcheck source=/dev/null
source "${ROOT_DIR}/scripts/env-local.sh"
ensure_env_local "${ROOT_DIR}"
load_env_local "${ROOT_DIR}"

PROFILE="${1:-hardened}"
TRANSPORT="${2:-ssh}"
IMAGE="sentinelops:local"

SSH_PORT="${SSH_PORT:-2222}"
TCP_PORT="${TCP_PORT:-2324}"
METRICS_PORT="${METRICS_PORT:-9001}"
API_PORT="${API_PORT:-9443}"

docker build -t "${IMAGE}" .
docker rm -f sentinelops-local >/dev/null 2>&1 || true

if [[ "${TRANSPORT}" == "ssh" ]]; then
  docker run -d --rm \
    --name sentinelops-local \
    -p "${SSH_PORT}:2222" \
    -p "${METRICS_PORT}:9001" \
    -p "${API_PORT}:9443" \
    -e APP_ENV=container \
    -e APP_PROFILE="${PROFILE}" \
    -e APP_TRANSPORT=ssh \
    -e APP_SSH_ADDR=:2222 \
    -e METRICS_ADDR=:9001 \
    -e APP_PROJECT_ROOT=/app \
    -e APP_CONTROL_API_ENABLED=true \
    -e APP_CONTROL_API_ADDR=:9443 \
    -e APP_CONTROL_API_USER=admin \
    -e APP_CONTROL_API_PASSWORD="${APP_CONTROL_API_PASSWORD:-}" \
    -e LAB_PASSWORD_STUDENT="${LAB_PASSWORD_STUDENT:-}" \
    -e LAB_PASSWORD_TEACHER="${LAB_PASSWORD_TEACHER:-}" \
    -e LAB_PASSWORD_AUDITOR="${LAB_PASSWORD_AUDITOR:-}" \
    -e LAB_PASSWORD_ADMIN="${LAB_PASSWORD_ADMIN:-}" \
    -e APP_CONTROL_API_CERT_PATH=/app/data/controlplane/tls.crt \
    -e APP_CONTROL_API_KEY_PATH=/app/data/controlplane/tls.key \
    -e APP_SSH_LOCAL_FORWARD_ENABLED=true \
    -e APP_SSH_FORWARD_ALLOWLIST=127.0.0.1:9001,localhost:9001 \
    -e APP_SSH_LOCAL_ALLOWED_ROLES=student,teacher,auditor,admin \
    -e APP_SSH_REMOTE_FORWARD_ENABLED=false \
    -e APP_SSH_REMOTE_BIND_ALLOWLIST=127.0.0.1:10080,127.0.0.1:10443 \
    -e APP_SSH_REMOTE_ALLOWED_ROLES=teacher,auditor,admin \
    -e EXTERNAL_AUDIT_ENABLED=true \
    -e EXTERNAL_AUDIT_COMMAND=python3 \
    -e EXTERNAL_AUDIT_SCRIPT=/app/tools/audit/audit.py \
    -e EXTERNAL_VALIDATOR_ENABLED=true \
    -e EXTERNAL_VALIDATOR_BINARY=/app/bin/input-guard \
    -e EXTERNAL_VALIDATOR_FAIL_OPEN=false \
    -e OPA_POLICY_ENABLED=true \
    -e OPA_BINARY=/app/bin/opa \
    -e OPA_POLICY_DIR=/app/policies/kubernetes \
    "${IMAGE}"

  echo "Contenedor desplegado en modo SSH."
  echo "SSH:        localhost:${SSH_PORT}"
  echo "Metrics:    http://localhost:${METRICS_PORT}/metrics"
  echo "ControlAPI: https://localhost:${API_PORT}/api/admin/status"
else
  docker run -d --rm \
    --name sentinelops-local \
    -p "${TCP_PORT}:2323" \
    -p "${METRICS_PORT}:9001" \
    -p "${API_PORT}:9443" \
    -e APP_ENV=container \
    -e APP_PROFILE="${PROFILE}" \
    -e APP_TRANSPORT=tcp \
    -e APP_ADDR=:2323 \
    -e METRICS_ADDR=:9001 \
    -e APP_PROJECT_ROOT=/app \
    -e APP_CONTROL_API_ENABLED=true \
    -e APP_CONTROL_API_ADDR=:9443 \
    -e APP_CONTROL_API_USER=admin \
    -e APP_CONTROL_API_PASSWORD="${APP_CONTROL_API_PASSWORD:-}" \
    -e LAB_PASSWORD_STUDENT="${LAB_PASSWORD_STUDENT:-}" \
    -e LAB_PASSWORD_TEACHER="${LAB_PASSWORD_TEACHER:-}" \
    -e LAB_PASSWORD_AUDITOR="${LAB_PASSWORD_AUDITOR:-}" \
    -e LAB_PASSWORD_ADMIN="${LAB_PASSWORD_ADMIN:-}" \
    -e APP_CONTROL_API_CERT_PATH=/app/data/controlplane/tls.crt \
    -e APP_CONTROL_API_KEY_PATH=/app/data/controlplane/tls.key \
    -e EXTERNAL_AUDIT_ENABLED=true \
    -e EXTERNAL_AUDIT_COMMAND=python3 \
    -e EXTERNAL_AUDIT_SCRIPT=/app/tools/audit/audit.py \
    -e EXTERNAL_VALIDATOR_ENABLED=true \
    -e EXTERNAL_VALIDATOR_BINARY=/app/bin/input-guard \
    -e EXTERNAL_VALIDATOR_FAIL_OPEN=false \
    -e OPA_POLICY_ENABLED=true \
    -e OPA_BINARY=/app/bin/opa \
    -e OPA_POLICY_DIR=/app/policies/kubernetes \
    "${IMAGE}"

  echo "Contenedor desplegado en modo TCP."
  echo "TCP:        localhost:${TCP_PORT}"
  echo "Metrics:    http://localhost:${METRICS_PORT}/metrics"
  echo "ControlAPI: https://localhost:${API_PORT}/api/admin/status"
fi

echo "Profile:    ${PROFILE}"
