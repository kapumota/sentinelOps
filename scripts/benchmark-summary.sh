#!/usr/bin/env bash
set -euo pipefail

REPORT_DIR="${1:-reports/benchmarks}"

if [ ! -d "$REPORT_DIR" ]; then
    echo "No existe el directorio de reportes: $REPORT_DIR"
    exit 1
fi

find "$REPORT_DIR" -name "go-benchmarks.txt" -print | sort | while read -r file; do
    echo "Reporte: $file"
    grep -E '^Benchmark' "$file" || true
    echo ""
done
