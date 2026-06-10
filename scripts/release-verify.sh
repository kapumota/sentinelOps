#!/usr/bin/env bash
set -euo pipefail

run_step() {
    local name="$1"
    shift
    echo "[release-verify] $name"
    "$@"
}

run_step "verificar secretos" make check-secrets
run_step "verificar scripts de benchmarks" bash -n scripts/run-benchmarks.sh
run_step "verificar resumen de benchmarks" bash -n scripts/benchmark-summary.sh
run_step "verificar limpieza de release" bash -n scripts/release-clean.sh
run_step "verificar go vet" make vet
run_step "ejecutar pruebas Go" make test
run_step "ejecutar pruebas de storage" make storage-test
run_step "ejecutar pruebas de integración" env TESTCONTAINERS_RYUK_DISABLED=true make test-integration
run_step "ejecutar pruebas Rust" make rust-test
run_step "compilar validador gRPC" make validator-grpc-build
run_step "probar validador gRPC" make validator-grpc-test
run_step "verificar diferencias" git diff --check

echo "[release-verify] validación final completada"
