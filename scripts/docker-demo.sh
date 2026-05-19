#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.demo.yml}"
API_URL="${API_URL:-https://localhost:9443}"
API_USER="${API_USER:-admin}"
API_PASSWORD="${API_PASSWORD:-admin123!}"
FORWARD_PORT="${FORWARD_PORT:-9900}"
LOCAL_USER="${LOCAL_USER:-student}"
REMOTE_USER="${REMOTE_USER:-teacher}"
KNOWN_HOSTS="/work/data/ssh/client/docker_known_hosts"
LOCAL_KEY="/work/data/ssh/client/${LOCAL_USER}_ed25519"
REMOTE_KEY="/work/data/ssh/client/${REMOTE_USER}_ed25519"
LOCAL_LOG="reports/docker-demo-local-tunnel.log"
REMOTE_LOG="reports/docker-demo-remote-tunnel.log"

cleanup() {
  docker compose -f "${COMPOSE_FILE}" down >/dev/null 2>&1 || true
}
trap cleanup EXIT

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || { echo "Falta comando requerido: $1"; exit 1; }
}

wait_for_url() {
  local url="$1"
  for _ in $(seq 1 50); do
    if curl -sk "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 1
  done
  return 1
}

extract_first_tunnel_id() {
  python3 - <<'PY'
import json, sys
raw = sys.stdin.read().strip()
if not raw:
    sys.exit(0)
try:
    data = json.loads(raw)
except Exception:
    sys.exit(0)
items = []
if isinstance(data, list):
    items = data
elif isinstance(data, dict):
    if isinstance(data.get("tuneles"), list):
        items = data["tuneles"]
    elif isinstance(data.get("tunnels"), list):
        items = data["tunnels"]
if items:
    first = items[0]
    if isinstance(first, dict):
        print(first.get("id", ""))
PY
}

require_cmd docker
require_cmd curl
require_cmd ssh-keygen

mkdir -p reports data/ssh/client data/ssh/authorized_keys
: > "${LOCAL_LOG}"
: > "${REMOTE_LOG}"

printf '[1/8] Preparando llaves SSH de laboratorio
'
bash scripts/setup-ssh-lab.sh "${LOCAL_USER}" >/dev/null
bash scripts/setup-ssh-lab.sh "${REMOTE_USER}" >/dev/null

printf '[2/8] Levantando stack de Docker Compose
'
docker compose -f "${COMPOSE_FILE}" up --build -d

wait_for_url "${API_URL}/healthz" || { echo "La API HTTPS no respondió"; exit 1; }

printf '[3/8] Verificando API HTTPS
'
curl -sk "${API_URL}/healthz"
printf '
'
curl -sk -u "${API_USER}:${API_PASSWORD}" "${API_URL}/api/admin/status"
printf '
'

printf '[4/8] Ejecutando comando remoto por SSH (modo exec)
'
docker exec sentinelops-tester sh -lc   "rm -f ${KNOWN_HOSTS} && ssh -T -o StrictHostKeyChecking=accept-new -o UserKnownHostsFile=${KNOWN_HOSTS} -p 2222 -i ${LOCAL_KEY} ${LOCAL_USER}@sentinelops status"

printf '[5/8] Abriendo túnel local SSH hacia métricas
'
docker exec -d sentinelops-tester sh -lc   "ssh -T -o StrictHostKeyChecking=accept-new -o UserKnownHostsFile=${KNOWN_HOSTS} -p 2222 -i ${LOCAL_KEY} -N -L ${FORWARD_PORT}:127.0.0.1:9001 ${LOCAL_USER}@sentinelops > /work/${LOCAL_LOG} 2>&1"
sleep 3
docker exec sentinelops-tester sh -lc "curl -s http://127.0.0.1:${FORWARD_PORT}/metrics | head -n 20"

printf '[6/8] Listando túneles por API
'
TUNNELS_JSON="$(curl -sk -u "${API_USER}:${API_PASSWORD}" "${API_URL}/api/admin/tunnels")"
printf '%s
' "${TUNNELS_JSON}"

printf '[7/8] Abriendo reenvío remoto con rol permitido (%s)\n' "${REMOTE_USER}"
docker exec -d sentinelops-tester sh -lc   "ssh -T -o StrictHostKeyChecking=accept-new -o UserKnownHostsFile=${KNOWN_HOSTS} -p 2222 -i ${REMOTE_KEY} -N -R 10080:127.0.0.1:9001 ${REMOTE_USER}@sentinelops > /work/${REMOTE_LOG} 2>&1"
sleep 3
docker exec sentinelops sh -lc "curl -s http://127.0.0.1:10080/metrics | head -n 20"

printf '[8/8] Cerrando el primer túnel vía API HTTPS
'
TUNNEL_ID="$(printf '%s' "${TUNNELS_JSON}" | extract_first_tunnel_id)"
if [[ -n "${TUNNEL_ID}" ]]; then
  curl -sk -u "${API_USER}:${API_PASSWORD}" -X DELETE "${API_URL}/api/admin/tunnels/${TUNNEL_ID}"
  printf '
'
fi

printf '
Demostración Docker completada.
'
printf 'Comandos útiles:
'
printf '  docker compose -f %s ps
' "${COMPOSE_FILE}"
printf '  docker logs sentinelops
'
printf '  docker exec -it sentinelops-tester sh
'
printf '  docker exec -it sentinelops sh
'
