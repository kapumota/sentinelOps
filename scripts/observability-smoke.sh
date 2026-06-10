#!/usr/bin/env bash
# Verifica endpoints operacionales de SentinelOps, Prometheus, Grafana y Jaeger.

set -euo pipefail

METRICS_URL="${METRICS_URL:-http://localhost:9101/metrics}"
API_URL="${API_URL:-https://localhost:9444}"
PROMETHEUS_URL="${PROMETHEUS_URL:-http://localhost:9090}"
GRAFANA_URL="${GRAFANA_URL:-http://localhost:3000}"
JAEGER_URL="${JAEGER_URL:-http://localhost:16686}"
API_USER="${API_USER:-${APP_CONTROL_API_USER:-admin}}"
API_PASSWORD="${API_PASSWORD:-${APP_CONTROL_API_PASSWORD:-}}"

log_info() {
  printf '[INFO] %s\n' "$1"
}

log_pass() {
  printf '[PASS] %s\n' "$1"
}

log_warn() {
  printf '[WARN] %s\n' "$1"
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf '[FAIL] comando requerido no disponible: %s\n' "$1" >&2
    exit 1
  fi
}

check_http() {
  local name="$1"
  local url="$2"
  if curl -fsS --max-time 5 "$url" >/dev/null; then
    log_pass "$name disponible en $url"
  else
    log_warn "$name no respondió en $url"
  fi
}

check_https() {
  local name="$1"
  local url="$2"
  if curl -kfsS --max-time 5 "$url" >/dev/null; then
    log_pass "$name disponible en $url"
  else
    log_warn "$name no respondió en $url"
  fi
}

require_command curl

log_info "verificando métricas de SentinelOps"
if curl -fsS --max-time 5 "$METRICS_URL" | grep -q 'sentinelops_'; then
  log_pass "métricas SentinelOps expuestas"
else
  log_warn "no se encontraron métricas SentinelOps en $METRICS_URL"
fi

log_info "verificando health checks de Control Plane"
check_https "health live" "$API_URL/healthz/live"
check_https "health ready" "$API_URL/healthz/ready"
check_https "health startup" "$API_URL/healthz/startup"

if [ -n "$API_PASSWORD" ]; then
  if curl -kfsS --max-time 5 -u "$API_USER:$API_PASSWORD" "$API_URL/api/v1/admin/status" >/dev/null; then
    log_pass "status autenticado disponible"
  else
    log_warn "status autenticado no respondió"
  fi
else
  log_warn "APP_CONTROL_API_PASSWORD no está definido; se omite status autenticado"
fi

log_info "verificando componentes de observabilidad"
check_http "Prometheus" "$PROMETHEUS_URL/-/healthy"
check_http "Grafana" "$GRAFANA_URL/api/health"
check_http "Jaeger" "$JAEGER_URL/api/services"

log_pass "verificación de observabilidad completada"
