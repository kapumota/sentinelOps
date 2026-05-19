#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${ROOT_DIR}"

USER_NAME="${USER_NAME:-student}"
SSH_PORT="${SSH_PORT:-2222}"
METRICS_PORT="${METRICS_PORT:-9001}"
API_PORT="${API_PORT:-9443}"
FORWARD_PORT="${FORWARD_PORT:-9900}"
API_URL="https://localhost:${API_PORT}"
API_USER="${API_USER:-admin}"
API_PASSWORD="${API_PASSWORD:-admin123!}"
IDENTITY_FILE="data/ssh/client/${USER_NAME}_ed25519"
KNOWN_HOSTS_FILE="data/ssh/client/demo_known_hosts"
SERVER_LOG="reports/demo-server.log"
CLIENT_LOG="reports/demo-client.log"
TUNNEL_LOG="reports/demo-tunnel.log"

SERVER_PID=""
TUNNEL_PID=""

cleanup() {
  if [[ -n "${TUNNEL_PID}" ]] && kill -0 "${TUNNEL_PID}" >/dev/null 2>&1; then
    kill "${TUNNEL_PID}" >/dev/null 2>&1 || true
    wait "${TUNNEL_PID}" 2>/dev/null || true
  fi
  if [[ -n "${SERVER_PID}" ]] && kill -0 "${SERVER_PID}" >/dev/null 2>&1; then
    kill "${SERVER_PID}" >/dev/null 2>&1 || true
    wait "${SERVER_PID}" 2>/dev/null || true
  fi
}
trap cleanup EXIT

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || { echo "Falta comando requerido: $1"; exit 1; }
}

wait_for_tcp() {
  local host="$1"
  local port="$2"
  local attempts="${3:-40}"
  for _ in $(seq 1 "$attempts"); do
    if python3 - <<PY >/dev/null 2>&1
import socket
s=socket.socket()
s.settimeout(0.5)
try:
    s.connect(("${host}", int("${port}")))
    ok=True
except Exception:
    ok=False
finally:
    s.close()
raise SystemExit(0 if ok else 1)
PY
    then
      return 0
    fi
    sleep 0.5
  done
  return 1
}

wait_for_https() {
  local url="$1"
  local attempts="${2:-40}"
  for _ in $(seq 1 "$attempts"); do
    if curl -sk "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.5
  done
  return 1
}

mkdir -p reports data/ssh/client
: > "${KNOWN_HOSTS_FILE}"

require_cmd go
require_cmd python3
require_cmd ssh
require_cmd curl
require_cmd cargo

if ! command -v opa >/dev/null 2>&1; then
  echo "Advertencia: opa no está instalado. El comando policy puede fallar durante la demo."
fi

if ! command -v ssh-keygen >/dev/null 2>&1; then
  echo "Falta comando requerido: ssh-keygen"
  exit 1
fi

echo "[1/8] Preparando llaves SSH de laboratorio"
make ssh-lab-setup USER_NAME="${USER_NAME}"

echo "[2/8] Compilando proyecto"
go mod tidy
make build >/dev/null

echo "[3/8] Iniciando SentinelOps en modo SSH"
make run-ssh >"${SERVER_LOG}" 2>&1 &
SERVER_PID=$!

wait_for_tcp 127.0.0.1 "${SSH_PORT}" || { echo "SSH no respondió en puerto ${SSH_PORT}"; exit 1; }
wait_for_https "${API_URL}/healthz" || { echo "API HTTPS no respondió en ${API_URL}"; exit 1; }

echo "[4/8] Ejecutando comando remoto con el cliente Go"
./bin/sentinelops-client \
  --addr "localhost:${SSH_PORT}" \
  --user "${USER_NAME}" \
  --identity "${IDENTITY_FILE}" \
  --known-hosts "${KNOWN_HOSTS_FILE}" \
  --strict-host-key true \
  --accept-unknown-host true \
  --cmd status | tee "${CLIENT_LOG}"

echo "[5/8] Consultando API HTTPS de control"
printf '\n## /healthz\n'
curl -sk "${API_URL}/healthz"
printf '\n\n## /api/admin/status\n'
curl -sk -u "${API_USER}:${API_PASSWORD}" "${API_URL}/api/admin/status"
printf '\n\n## /api/admin/sessions\n'
curl -sk -u "${API_USER}:${API_PASSWORD}" "${API_URL}/api/admin/sessions"
printf '\n\n## /api/admin/tunnels (antes)\n'
curl -sk -u "${API_USER}:${API_PASSWORD}" "${API_URL}/api/admin/tunnels"
printf '\n'

echo "[6/8] Abriendo túnel local SSH hacia métricas"
ssh \
  -o StrictHostKeyChecking=accept-new \
  -o UserKnownHostsFile="${KNOWN_HOSTS_FILE}" \
  -p "${SSH_PORT}" \
  -i "${IDENTITY_FILE}" \
  -N \
  -L "${FORWARD_PORT}:127.0.0.1:${METRICS_PORT}" \
  "${USER_NAME}@localhost" >"${TUNNEL_LOG}" 2>&1 &
TUNNEL_PID=$!

wait_for_tcp 127.0.0.1 "${FORWARD_PORT}" || { echo "El túnel local no abrió en ${FORWARD_PORT}"; exit 1; }

echo "[7/8] Consumiendo métricas a través del túnel"
curl -s "http://localhost:${FORWARD_PORT}/metrics" | head -n 20

printf '\n## /api/admin/tunnels (después de abrir túnel)\n'
TUNNELS_JSON="$(curl -sk -u "${API_USER}:${API_PASSWORD}" "${API_URL}/api/admin/tunnels")"
printf '%s\n' "${TUNNELS_JSON}"

TUNNEL_ID="$(printf '%s' "${TUNNELS_JSON}" | python3 - <<'PY'
import json, sys
try:
    data = json.load(sys.stdin)
    tunnels = data.get('tunnels') or []
    print(tunnels[0]['id'] if tunnels else '')
except Exception:
    print('')
PY
)"

echo "[8/8] Cerrando túnel vía API HTTPS"
if [[ -n "${TUNNEL_ID}" ]]; then
  curl -sk -u "${API_USER}:${API_PASSWORD}" -X DELETE "${API_URL}/api/admin/tunnels/${TUNNEL_ID}"
  printf '\n\n## /api/admin/tunnels (después de cerrar)\n'
  curl -sk -u "${API_USER}:${API_PASSWORD}" "${API_URL}/api/admin/tunnels"
  printf '\n'
else
  echo "No se encontró tunnel_id para cerrar."
fi

echo

echo "Demostración completada."
echo "Logs:"
echo "  servidor: ${SERVER_LOG}"
echo "  cliente:  ${CLIENT_LOG}"
echo "  túnel:    ${TUNNEL_LOG}"
