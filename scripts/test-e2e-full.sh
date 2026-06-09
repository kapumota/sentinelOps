#!/usr/bin/env bash
set -Eeuo pipefail
IFS=$'\n\t'

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

# shellcheck source=/dev/null
source "${ROOT_DIR}/scripts/env-local.sh"
ensure_env_local "${ROOT_DIR}"
load_env_local "${ROOT_DIR}"

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.demo.yml}"

# Puertos externos en tu máquina host.
SSH_PORT="${SSH_PORT:-2223}"
API_PORT="${API_PORT:-9444}"
METRICS_PORT="${METRICS_PORT:-9101}"

# Puertos internos dentro del contenedor sentinelops.
INTERNAL_SSH_PORT="${INTERNAL_SSH_PORT:-2222}"
INTERNAL_API_PORT="${INTERNAL_API_PORT:-9443}"
INTERNAL_METRICS_PORT="${INTERNAL_METRICS_PORT:-9001}"

# Puertos para túneles.
LOCAL_FORWARD_PORT="${LOCAL_FORWARD_PORT:-9901}"

# Importante: 10080 coincide con la allowlist por defecto:
# 127.0.0.1:10080,127.0.0.1:10443
REMOTE_BIND_PORT="${REMOTE_BIND_PORT:-10080}"

API_USER="${API_USER:-admin}"
API_PASSWORD="${API_PASSWORD:-${APP_CONTROL_API_PASSWORD:-}}"
if [[ -z "${API_PASSWORD}" ]]; then
  echo "Falta API_PASSWORD o APP_CONTROL_API_PASSWORD. Ejecuta make generate-secrets." >&2
  exit 1
fi

export SSH_PORT
export API_PORT
export METRICS_PORT

STUDENT_KEY="data/ssh/client/student_ed25519"
TEACHER_KEY="data/ssh/client/teacher_ed25519"

TIMESTAMP="$(date +%Y%m%d_%H%M%S)"
REPORT_BASE="reports/e2e"
REPORT_DIR="${REPORT_BASE}/${TIMESTAMP}"
LATEST_LINK="${REPORT_BASE}/latest"

mkdir -p "${REPORT_DIR}"

RUN_LOG="${REPORT_DIR}/run.log"
ACTA_FILE="${REPORT_DIR}/acta-validacion.txt"
RESULTS_JSON="${REPORT_DIR}/resultados.json"

exec > >(tee -a "${RUN_LOG}") 2>&1

PASS_COUNT=0
FAIL_COUNT=0

SSH_OPTS=(
  -o StrictHostKeyChecking=no
  -o UserKnownHostsFile=/dev/null
  -o LogLevel=ERROR
)

SSH_TUNNEL_PID=""
SSH_REMOTE_PID=""
LOCAL_KEEPALIVE_PID=""
LOCAL_TUNNEL_ID=""
REMOTE_TUNNEL_ID=""

record_result() {
  local step="$1"
  local status="$2"
  local detail="$3"

  python3 - "$RESULTS_JSON" "$step" "$status" "$detail" <<'PY'
import json
import os
import sys

path, step, status, detail = sys.argv[1:]

data = {"results": []}

if os.path.exists(path):
    try:
        with open(path, "r", encoding="utf-8") as f:
            data = json.load(f)
    except Exception:
        data = {"results": []}

data.setdefault("results", []).append({
    "step": step,
    "status": status,
    "detail": detail,
})

with open(path, "w", encoding="utf-8") as f:
    json.dump(data, f, indent=2, ensure_ascii=False)
PY

  if [[ "${status}" == "OK" ]]; then
    PASS_COUNT=$((PASS_COUNT + 1))
  else
    FAIL_COUNT=$((FAIL_COUNT + 1))
  fi
}

prepare_host_permissions() {
  mkdir -p data/ssh/client
  mkdir -p data/ssh/authorized_keys
  mkdir -p data/controlplane
  mkdir -p data/state
  mkdir -p reports

  # El contenedor necesita escribir en data/ssh para crear/leer host keys.
  # Esto es para laboratorio local. No usar como política de producción.
  chmod 755 data
  chmod 777 data/ssh

  # authorized_keys debe ser legible por el contenedor.
  chmod 755 data/ssh/authorized_keys
  find data/ssh/authorized_keys -type f -exec chmod 644 {} \; 2>/dev/null || true

  # Las claves privadas del cliente deben quedar protegidas para OpenSSH.
  chmod 700 data/ssh/client
  find data/ssh/client -type f -name "*_ed25519" -exec chmod 600 {} \; 2>/dev/null || true
  find data/ssh/client -type f -name "*.pub" -exec chmod 644 {} \; 2>/dev/null || true

  # El contenedor necesita escribir certificados, estado y reportes.
  chmod -R a+rwX data/controlplane
  chmod -R a+rwX data/state
  chmod -R a+rwX reports

  # Si ya existen host keys creadas con permisos incompatibles, hacerlas legibles/escribibles
  # para el contenedor de laboratorio.
  find data/ssh -maxdepth 1 -type f -name "host_*" -exec chmod a+rw {} \; 2>/dev/null || true
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Falta el comando requerido: $1"
    record_result "require_cmd:${1}" "FALLO" "Comando no encontrado"
    exit 1
  }

  record_result "require_cmd:${1}" "OK" "Comando disponible"
}

require_docker_compose() {
  if ! docker compose version >/dev/null 2>&1; then
    echo "Falta Docker Compose v2: docker compose"
    record_result "require_cmd:docker-compose" "FALLO" "docker compose no disponible"
    exit 1
  fi

  record_result "require_cmd:docker-compose" "OK" "docker compose disponible"
}

wait_for_http() {
  local url="$1"
  local max_attempts="${2:-30}"
  local sleep_seconds="${3:-2}"

  for _ in $(seq 1 "${max_attempts}"); do
    if curl -ksf "${url}" >/dev/null 2>&1; then
      return 0
    fi

    sleep "${sleep_seconds}"
  done

  return 1
}

wait_for_api() {
  local path="$1"
  local max_attempts="${2:-40}"
  local sleep_seconds="${3:-2}"

  for _ in $(seq 1 "${max_attempts}"); do
    if curl -ksf -u "${API_USER}:${API_PASSWORD}" "https://localhost:${API_PORT}${path}" >/dev/null 2>&1; then
      return 0
    fi

    sleep "${sleep_seconds}"
  done

  return 1
}

wait_for_tcp() {
  local host="$1"
  local port="$2"
  local max_attempts="${3:-30}"
  local sleep_seconds="${4:-2}"

  for _ in $(seq 1 "${max_attempts}"); do
    if nc -z "${host}" "${port}" >/dev/null 2>&1; then
      return 0
    fi

    sleep "${sleep_seconds}"
  done

  return 1
}

api_get() {
  local path="$1"
  curl -ksf -u "${API_USER}:${API_PASSWORD}" "https://localhost:${API_PORT}${path}"
}

api_delete() {
  local path="$1"
  curl -ksf -u "${API_USER}:${API_PASSWORD}" -X DELETE "https://localhost:${API_PORT}${path}"
}

extract_first_tunnel_id_by_direction() {
  local direction="$1"

  python3 -c '
import json
import sys

direction = sys.argv[1]
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

for item in items:
    if isinstance(item, dict) and item.get("direction") == direction:
        tunnel_id = item.get("id", "")
        if tunnel_id:
            print(tunnel_id)
        break
' "$direction"
}

contains_tunnel_id() {
  local tunnel_id="$1"

  python3 -c '
import json
import sys

wanted = sys.argv[1]
raw = sys.stdin.read().strip()

if not raw:
    sys.exit(1)

try:
    data = json.loads(raw)
except Exception:
    sys.exit(1)

items = []

if isinstance(data, list):
    items = data
elif isinstance(data, dict):
    if isinstance(data.get("tuneles"), list):
        items = data["tuneles"]
    elif isinstance(data.get("tunnels"), list):
        items = data["tunnels"]

for item in items:
    if isinstance(item, dict) and item.get("id") == wanted:
        sys.exit(0)

sys.exit(1)
' "$tunnel_id"
}

save_docker_logs() {
  docker logs sentinelops > "${REPORT_DIR}/docker-sentinelops.log" 2>&1 || true
  docker logs sentinelops-tester > "${REPORT_DIR}/docker-sentinelops-tester.log" 2>&1 || true
  docker compose -f "${COMPOSE_FILE}" ps > "${REPORT_DIR}/docker-compose-ps.txt" 2>&1 || true
}

write_acta() {
  cat > "${ACTA_FILE}" <<EOF
ACTA DE VALIDACIÓN E2E - SENTINELOPS v2.4.1
Fecha: $(date -Iseconds)
Directorio de evidencia: ${REPORT_DIR}

Configuración de puertos:
- SSH externo: ${SSH_PORT}
- SSH interno: ${INTERNAL_SSH_PORT}
- API externa: ${API_PORT}
- API interna: ${INTERNAL_API_PORT}
- Métricas externas: ${METRICS_PORT}
- Métricas internas: ${INTERNAL_METRICS_PORT}
- Túnel local: ${LOCAL_FORWARD_PORT}
- Bind remoto: ${REMOTE_BIND_PORT}

Resumen:
- Pasos exitosos: ${PASS_COUNT}
- Pasos fallidos: ${FAIL_COUNT}

Artefactos generados:
- ${RUN_LOG}
- ${RESULTS_JSON}
- ${REPORT_DIR}/api-healthz.txt
- ${REPORT_DIR}/api-status.json
- ${REPORT_DIR}/api-sessions.json
- ${REPORT_DIR}/api-sessions-final.json
- ${REPORT_DIR}/api-state-sessions.json
- ${REPORT_DIR}/api-state-tunnels.json
- ${REPORT_DIR}/api-tunnels-inicial.json
- ${REPORT_DIR}/api-tunnels-local.json
- ${REPORT_DIR}/api-tunnels-local-after-delete.json
- ${REPORT_DIR}/api-tunnels-remote.json
- ${REPORT_DIR}/api-tunnels-remote-after-delete.json
- ${REPORT_DIR}/api-delete-local.txt
- ${REPORT_DIR}/api-delete-remote.txt
- ${REPORT_DIR}/ssh-help.txt
- ${REPORT_DIR}/ssh-status.txt
- ${REPORT_DIR}/ssh-whoami.txt
- ${REPORT_DIR}/ssh-audit.txt
- ${REPORT_DIR}/ssh-policy.txt
- ${REPORT_DIR}/metrics-direct.txt
- ${REPORT_DIR}/metrics-local-forward.txt
- ${REPORT_DIR}/metrics-remote-forward.txt
- ${REPORT_DIR}/docker-sentinelops.log
- ${REPORT_DIR}/docker-sentinelops-tester.log
- ${REPORT_DIR}/docker-compose-ps.txt

Veredicto:
EOF

  if [[ "${FAIL_COUNT}" -eq 0 ]]; then
    echo "VALIDACIÓN E2E EXITOSA" >> "${ACTA_FILE}"
  else
    echo "VALIDACIÓN E2E CON OBSERVACIONES" >> "${ACTA_FILE}"
  fi
}

cleanup() {
  set +e

  if [[ -n "${LOCAL_KEEPALIVE_PID}" ]]; then
    kill "${LOCAL_KEEPALIVE_PID}" >/dev/null 2>&1 || true
  fi

  if [[ -n "${SSH_TUNNEL_PID}" ]]; then
    kill "${SSH_TUNNEL_PID}" >/dev/null 2>&1 || true
  fi

  if [[ -n "${SSH_REMOTE_PID}" ]]; then
    kill "${SSH_REMOTE_PID}" >/dev/null 2>&1 || true
  fi

  save_docker_logs
  write_acta

  rm -rf "${LATEST_LINK}"
  ln -s "${TIMESTAMP}" "${LATEST_LINK}"

  docker compose -f "${COMPOSE_FILE}" down >/dev/null 2>&1 || true
}

trap cleanup EXIT

echo "[1/14] Verificando dependencias"
require_cmd docker
require_docker_compose
require_cmd curl
require_cmd ssh
require_cmd nc
require_cmd python3
require_cmd make

echo "[2/14] Preparando llaves SSH"
make ssh-lab-setup USER_NAME=student >/dev/null
make ssh-lab-setup USER_NAME=teacher >/dev/null

prepare_host_permissions

if [[ ! -f "${STUDENT_KEY}" ]]; then
  record_result "ssh-key-student" "FALLO" "No existe ${STUDENT_KEY}"
  exit 1
fi

if [[ ! -f "${TEACHER_KEY}" ]]; then
  record_result "ssh-key-teacher" "FALLO" "No existe ${TEACHER_KEY}"
  exit 1
fi

record_result "ssh-keys" "OK" "Llaves de student y teacher disponibles"

echo "[3/14] Levantando Docker Compose"
docker compose -f "${COMPOSE_FILE}" down >/dev/null 2>&1 || true
docker compose -f "${COMPOSE_FILE}" up --build -d
record_result "docker-compose-up" "OK" "Stack levantado"

echo "[4/14] Esperando servicios"

if wait_for_tcp localhost "${SSH_PORT}" 40 2; then
  record_result "wait-ssh" "OK" "SSH disponible en localhost:${SSH_PORT}"
else
  record_result "wait-ssh" "FALLO" "SSH no disponible en localhost:${SSH_PORT}"
  exit 1
fi

if wait_for_http "https://localhost:${API_PORT}/healthz" 40 2; then
  record_result "wait-api-healthz" "OK" "API HTTPS disponible en https://localhost:${API_PORT}/healthz"
else
  record_result "wait-api-healthz" "FALLO" "Healthz no disponible en https://localhost:${API_PORT}/healthz"
  exit 1
fi

if wait_for_api "/api/admin/status" 40 2; then
  record_result "wait-api-status" "OK" "API administrativa disponible con autenticación"
else
  record_result "wait-api-status" "FALLO" "API administrativa no disponible con autenticación"
  exit 1
fi

if wait_for_tcp localhost "${METRICS_PORT}" 40 2; then
  record_result "wait-metrics" "OK" "Métricas disponibles en localhost:${METRICS_PORT}"
else
  record_result "wait-metrics" "FALLO" "Métricas no disponibles en localhost:${METRICS_PORT}"
  exit 1
fi

echo "[5/14] Guardando healthz y estado"
curl -ksf "https://localhost:${API_PORT}/healthz" | tee "${REPORT_DIR}/api-healthz.txt" >/dev/null
api_get "/api/admin/status" | tee "${REPORT_DIR}/api-status.json" >/dev/null
api_get "/api/admin/sessions" | tee "${REPORT_DIR}/api-sessions.json" >/dev/null
api_get "/api/admin/tunnels" | tee "${REPORT_DIR}/api-tunnels-inicial.json" >/dev/null
api_get "/api/admin/state/sessions" | tee "${REPORT_DIR}/api-state-sessions.json" >/dev/null
api_get "/api/admin/state/tunnels" | tee "${REPORT_DIR}/api-state-tunnels.json" >/dev/null
record_result "api-status-sessions-tunnels-inicial" "OK" "Estado, sesiones, túneles y snapshots de estado exportados"

echo "[6/14] Validando comandos SSH"
ssh "${SSH_OPTS[@]}" -T -p "${SSH_PORT}" -i "${STUDENT_KEY}" student@localhost help   | tee "${REPORT_DIR}/ssh-help.txt" >/dev/null
ssh "${SSH_OPTS[@]}" -T -p "${SSH_PORT}" -i "${STUDENT_KEY}" student@localhost status | tee "${REPORT_DIR}/ssh-status.txt" >/dev/null
ssh "${SSH_OPTS[@]}" -T -p "${SSH_PORT}" -i "${STUDENT_KEY}" student@localhost whoami | tee "${REPORT_DIR}/ssh-whoami.txt" >/dev/null
ssh "${SSH_OPTS[@]}" -T -p "${SSH_PORT}" -i "${STUDENT_KEY}" student@localhost audit  | tee "${REPORT_DIR}/ssh-audit.txt" >/dev/null
ssh "${SSH_OPTS[@]}" -T -p "${SSH_PORT}" -i "${STUDENT_KEY}" student@localhost policy | tee "${REPORT_DIR}/ssh-policy.txt" >/dev/null
record_result "ssh-commands" "OK" "help/status/whoami/audit/policy ejecutados"

echo "[7/14] Exportando métricas directas"
curl -sf "http://localhost:${METRICS_PORT}/metrics" | tee "${REPORT_DIR}/metrics-direct.txt" >/dev/null
record_result "metrics-direct" "OK" "Métricas directas exportadas"

echo "[8/14] Abriendo túnel local"
ssh "${SSH_OPTS[@]}" -N -T \
  -p "${SSH_PORT}" \
  -i "${STUDENT_KEY}" \
  -L "${LOCAL_FORWARD_PORT}:127.0.0.1:${INTERNAL_METRICS_PORT}" \
  student@localhost &

SSH_TUNNEL_PID=$!

if wait_for_tcp localhost "${LOCAL_FORWARD_PORT}" 20 1; then
  record_result "local-forward-listener" "OK" "Listener local disponible en localhost:${LOCAL_FORWARD_PORT}"
else
  record_result "local-forward-listener" "FALLO" "No se abrió el listener local en localhost:${LOCAL_FORWARD_PORT}"
  exit 1
fi

# Mantiene una conexión viva para que SentinelOps registre el túnel local.
nc 127.0.0.1 "${LOCAL_FORWARD_PORT}" >/dev/null 2>&1 &
LOCAL_KEEPALIVE_PID=$!

sleep 1

curl -sf "http://localhost:${LOCAL_FORWARD_PORT}/metrics" | tee "${REPORT_DIR}/metrics-local-forward.txt" >/dev/null
record_result "local-forward" "OK" "Túnel local operativo"

echo "[9/14] Exportando túneles con local forwarding activo"
api_get "/api/admin/tunnels" | tee "${REPORT_DIR}/api-tunnels-local.json" >/dev/null

LOCAL_TUNNEL_ID="$(extract_first_tunnel_id_by_direction "local" < "${REPORT_DIR}/api-tunnels-local.json")"

if [[ -z "${LOCAL_TUNNEL_ID}" ]]; then
  record_result "local-forward-id" "FALLO" "No se encontró ID del túnel local"
  exit 1
fi

record_result "local-forward-id" "OK" "ID detectado: ${LOCAL_TUNNEL_ID}"

echo "[10/14] Cerrando túnel local por API"
api_delete "/api/admin/tunnels/${LOCAL_TUNNEL_ID}" > "${REPORT_DIR}/api-delete-local.txt"

sleep 2

api_get "/api/admin/tunnels" | tee "${REPORT_DIR}/api-tunnels-local-after-delete.json" >/dev/null

if contains_tunnel_id "${LOCAL_TUNNEL_ID}" < "${REPORT_DIR}/api-tunnels-local-after-delete.json"; then
  record_result "local-forward-delete" "FALLO" "El túnel local sigue apareciendo"
  exit 1
fi

record_result "local-forward-delete" "OK" "Túnel local eliminado por API"

if [[ -n "${LOCAL_KEEPALIVE_PID}" ]]; then
  kill "${LOCAL_KEEPALIVE_PID}" >/dev/null 2>&1 || true
  LOCAL_KEEPALIVE_PID=""
fi

if [[ -n "${SSH_TUNNEL_PID}" ]]; then
  kill "${SSH_TUNNEL_PID}" >/dev/null 2>&1 || true
  SSH_TUNNEL_PID=""
fi

echo "[11/14] Abriendo reenvío remoto con teacher"
ssh "${SSH_OPTS[@]}" -N -T \
  -p "${SSH_PORT}" \
  -i "${TEACHER_KEY}" \
  -R "127.0.0.1:${REMOTE_BIND_PORT}:127.0.0.1:${METRICS_PORT}" \
  teacher@localhost &

SSH_REMOTE_PID=$!

sleep 3

# Mantiene una conexión viva desde el contenedor hacia el bind remoto.
docker exec -d sentinelops sh -lc "nc 127.0.0.1 ${REMOTE_BIND_PORT} >/dev/null 2>&1"

sleep 1

docker exec sentinelops sh -lc "curl -sf http://127.0.0.1:${REMOTE_BIND_PORT}/metrics" | tee "${REPORT_DIR}/metrics-remote-forward.txt" >/dev/null
record_result "remote-forward" "OK" "Reenvío remoto operativo"

echo "[12/14] Exportando túneles con reenvío remoto activo"
api_get "/api/admin/tunnels" | tee "${REPORT_DIR}/api-tunnels-remote.json" >/dev/null

REMOTE_TUNNEL_ID="$(extract_first_tunnel_id_by_direction "remote" < "${REPORT_DIR}/api-tunnels-remote.json")"

if [[ -z "${REMOTE_TUNNEL_ID}" ]]; then
  record_result "remote-forward-id" "FALLO" "No se encontró ID del túnel remoto"
  exit 1
fi

record_result "remote-forward-id" "OK" "ID detectado: ${REMOTE_TUNNEL_ID}"

echo "[13/14] Cerrando túnel remoto por API"
api_delete "/api/admin/tunnels/${REMOTE_TUNNEL_ID}" > "${REPORT_DIR}/api-delete-remote.txt"

sleep 2

api_get "/api/admin/tunnels" | tee "${REPORT_DIR}/api-tunnels-remote-after-delete.json" >/dev/null

if contains_tunnel_id "${REMOTE_TUNNEL_ID}" < "${REPORT_DIR}/api-tunnels-remote-after-delete.json"; then
  record_result "remote-forward-delete" "FALLO" "El túnel remoto sigue apareciendo"
  exit 1
fi

record_result "remote-forward-delete" "OK" "Túnel remoto eliminado por API"

if [[ -n "${SSH_REMOTE_PID}" ]]; then
  kill "${SSH_REMOTE_PID}" >/dev/null 2>&1 || true
  SSH_REMOTE_PID=""
fi

echo "[14/14] Exportando sesiones finales"
api_get "/api/admin/sessions" | tee "${REPORT_DIR}/api-sessions-final.json" >/dev/null
record_result "sessions-final" "OK" "Sesiones finales exportadas"

echo
echo "VALIDACIÓN E2E COMPLETADA"
echo "Resultados: ${RESULTS_JSON}"
echo "Acta:       ${ACTA_FILE}"
echo "Evidencia:  ${REPORT_DIR}"