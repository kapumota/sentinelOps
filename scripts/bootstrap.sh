#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

mkdir -p reports

echo "[1/9] Formateando código Go"
go fmt ./...

echo "[2/9] Ejecutando análisis estático Go"
go vet ./...

echo "[3/9] Ejecutando pruebas Go"
go test ./...

echo "[4/9] Ejecutando pruebas Rust"
cargo test --manifest-path rust/input-guard/Cargo.toml

echo "[5/9] Compilando validador Rust"
cargo build --release --manifest-path rust/input-guard/Cargo.toml

echo "[6/9] Ejecutando auditoría externa en Python"
python3 tools/audit/audit.py --profile hardened --project-root . > reports/bootstrap-audit.json

echo "[7/9] Evaluando políticas hardened"
bash scripts/ci-policy-check.sh hardened pass

echo "[8/9] Renderizando Helm hardened"
bash scripts/ci-helm-validate.sh hardened

echo "[9/9] Bootstrap completado"
echo "Validador Rust: rust/input-guard/target/release/input-guard"
echo "Reporte generado en reports/bootstrap-audit.json"
