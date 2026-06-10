#!/usr/bin/env bash
# Recolecta evidencia de runtime sin incluir secretos en el reporte.

set -euo pipefail

REPORT_ROOT="${REPORT_ROOT:-reports/runtime}"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
REPORT_DIR="${REPORT_ROOT}/${STAMP}"
METRICS_URL="${METRICS_URL:-http://localhost:9101/metrics}"
API_URL="${API_URL:-https://localhost:9444}"
PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9090}"
GRAFANA_URL="${GRAFANA_URL:-http://localhost:3000}"
JAEGER_URL="${JAEGER_URL:-http://localhost:16686}"
API_USER="${API_USER:-${APP_CONTROL_API_USER:-admin}}"
API_PASSWORD="${API_PASSWORD:-${APP_CONTROL_API_PASSWORD:-}}"

mkdir -p "$REPORT_DIR"

write_text() {
  local path="$1"
  local content="$2"
  printf '%s\n' "$content" > "$path"
}

capture_url() {
  local name="$1"
  local url="$2"
  local output="$3"
  if curl -kfsS --max-time 10 "$url" > "$output"; then
    printf '[PASS] %s\n' "$name"
  else
    printf '[WARN] %s no disponible\n' "$name"
    printf 'no disponible\n' > "$output"
  fi
}

capture_auth_url() {
  local name="$1"
  local url="$2"
  local output="$3"
  if [ -z "$API_PASSWORD" ]; then
    printf '[WARN] %s omitido porque APP_CONTROL_API_PASSWORD no está definido\n' "$name"
    printf 'omitido: contraseña no definida\n' > "$output"
    return 0
  fi

  if curl -kfsS --max-time 10 -u "$API_USER:$API_PASSWORD" "$url" > "$output"; then
    printf '[PASS] %s\n' "$name"
  else
    printf '[WARN] %s no disponible\n' "$name"
    printf 'no disponible\n' > "$output"
  fi
}

GIT_COMMIT="$(git rev-parse --short HEAD 2>/dev/null || printf 'desconocido')"
GIT_BRANCH="$(git branch --show-current 2>/dev/null || printf 'desconocida')"

cat > "${REPORT_DIR}/metadata.json" <<EOF_META
{
  "generado_en_utc": "${STAMP}",
  "git_commit": "${GIT_COMMIT}",
  "git_branch": "${GIT_BRANCH}",
  "metrics_url": "${METRICS_URL}",
  "api_url": "${API_URL}",
  "prometheus_url": "${PROMETHEUS_URL}",
  "grafana_url": "${GRAFANA_URL}",
  "jaeger_url": "${JAEGER_URL}"
}
EOF_META

capture_url "métricas" "$METRICS_URL" "${REPORT_DIR}/metrics.prom"
capture_url "health live" "$API_URL/healthz/live" "${REPORT_DIR}/health-live.json"
capture_url "health ready" "$API_URL/healthz/ready" "${REPORT_DIR}/health-ready.json"
capture_url "health startup" "$API_URL/healthz/startup" "${REPORT_DIR}/health-startup.json"
capture_auth_url "status autenticado" "$API_URL/api/v1/admin/status" "${REPORT_DIR}/admin-status.json"
capture_url "Prometheus health" "$PROMETHEUS_URL/-/healthy" "${REPORT_DIR}/prometheus-health.txt"
capture_url "Prometheus targets" "$PROMETHEUS_URL/api/v1/targets" "${REPORT_DIR}/prometheus-targets.json"
capture_url "Grafana health" "$GRAFANA_URL/api/health" "${REPORT_DIR}/grafana-health.json"
capture_url "Jaeger services" "$JAEGER_URL/api/services" "${REPORT_DIR}/jaeger-services.json"

if command -v docker >/dev/null 2>&1; then
  docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}' > "${REPORT_DIR}/docker-ps.txt" 2>/dev/null || true
fi

cat > "${REPORT_DIR}/README.md" <<EOF_README
### Evidencia de runtime SentinelOps

#### Contenido

- \`metadata.json\`: metadatos del entorno.
- \`metrics.prom\`: métricas Prometheus expuestas por SentinelOps.
- \`health-live.json\`: respuesta del health check live.
- \`health-ready.json\`: respuesta del health check ready.
- \`health-startup.json\`: respuesta del health check startup.
- \`admin-status.json\`: estado autenticado del Control Plane si se configuró contraseña.
- \`prometheus-targets.json\`: targets observados por Prometheus.
- \`grafana-health.json\`: estado de Grafana.
- \`jaeger-services.json\`: servicios observados por Jaeger.
- \`docker-ps.txt\`: estado de contenedores si Docker está disponible.

#### Nota

Este reporte no debe incluir secretos. Las credenciales se usan solo para consultar endpoints autenticados.
EOF_README

printf 'Evidencia generada en %s\n' "$REPORT_DIR"
