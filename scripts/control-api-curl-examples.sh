#!/usr/bin/env bash
set -euo pipefail

API_URL="${API_URL:-https://localhost:9443}"
API_USER="${API_USER:-admin}"
API_PASSWORD="${API_PASSWORD:-admin123!}"
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
