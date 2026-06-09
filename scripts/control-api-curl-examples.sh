#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

# shellcheck source=/dev/null
source "${ROOT_DIR}/scripts/env-local.sh"
load_env_local "${ROOT_DIR}"

API_URL="${API_URL:-https://localhost:9443}"
API_USER="${API_USER:-admin}"
API_PASSWORD="${API_PASSWORD:-${APP_CONTROL_API_PASSWORD:-}}"
if [[ -z "${API_PASSWORD}" ]]; then
  echo "Falta API_PASSWORD o APP_CONTROL_API_PASSWORD. Ejecuta make generate-secrets." >&2
  exit 1
fi
TUNNEL_ID="${TUNNEL_ID:-}"

curl_json() {
  local method="$1"
  local path="$2"
  curl -sk \
    -u "${API_USER}:${API_PASSWORD}" \
    -H 'Accept: application/json' \
    -X "$method" \
    "${API_URL}${path}"
  printf '\n'
}

echo "# healthz"
curl -sk "${API_URL}/healthz"
printf '\n\n'

echo "# estado administrativo"
curl_json GET "/api/admin/status"
printf '\n'

echo "# sesiones activas"
curl_json GET "/api/admin/sessions"
printf '\n'

echo "# túneles activos"
curl_json GET "/api/admin/tunnels"
printf '\n'

echo "# snapshot persistido de sesiones"
curl_json GET "/api/admin/state/sessions"
printf '\n'

echo "# snapshot persistido de túneles"
curl_json GET "/api/admin/state/tunnels"
printf '\n'

if [[ -n "${TUNNEL_ID}" ]]; then
  echo "# cerrar túnel ${TUNNEL_ID}"
  curl_json DELETE "/api/admin/tunnels/${TUNNEL_ID}"
fi
