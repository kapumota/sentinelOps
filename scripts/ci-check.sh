#!/usr/bin/env bash
# Ejecuta verificaciones locales cercanas al flujo de CI.

set -euo pipefail

failed=0

run_required() {
  name="$1"
  shift
  echo "--------- ${name}"
  if "$@"; then
    echo "PASS ${name}"
  else
    echo "FAIL ${name}"
    failed=1
  fi
}

run_optional() {
  name="$1"
  shift
  echo "--------- ${name}"
  if "$@"; then
    echo "PASS ${name}"
  else
    echo "WARN ${name} no disponible o falló"
  fi
}

echo "Verificación local de SentinelOps"

run_required "secret scan" make check-secrets
run_required "OpenAPI" make docs-check
run_required "Go format" bash -c 'test -z "$(gofmt -l .)"'
run_required "Go vet" make vet
run_required "Go tests" make test
run_required "Rust validator clásico" make rust-test
run_required "Rust validator gRPC build" make validator-grpc-build
run_required "Rust validator gRPC tests" make validator-grpc-test
run_optional "OPA policies" bash -c 'make opa-test && make opa-build && make opa-clean'
run_optional "proto generation" bash -c 'make proto-go && make proto-clean'
run_optional "Docker SentinelOps" docker build -t sentinelops:ci-local .
run_optional "Docker input-guard gRPC" docker build -f rust/input-guard-grpc/Dockerfile -t sentinelops/input-guard-grpc:ci-local .

rm -f coverage.out coverage.html
make opa-clean >/dev/null 2>&1 || true
make proto-clean >/dev/null 2>&1 || true

if [ "$failed" -ne 0 ]; then
  echo "FAIL verificación local con errores obligatorios"
  exit 1
fi

echo "PASS verificación local completada"
