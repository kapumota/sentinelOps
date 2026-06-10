#!/usr/bin/env bash
set -euo pipefail

BENCH_PATTERN="${BENCH_PATTERN:-.}"
BENCH_TIME="${BENCH_TIME:-3s}"
BENCH_COUNT="${BENCH_COUNT:-3}"
REPORT_DIR="${REPORT_DIR:-reports/benchmarks/$(date -u +%Y%m%dT%H%M%SZ)}"

mkdir -p "$REPORT_DIR"

metadata_file="$REPORT_DIR/metadata.txt"
result_file="$REPORT_DIR/go-benchmarks.txt"

{
    echo "Benchmark SentinelOps"
    echo "fecha_utc=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    echo "bench_pattern=$BENCH_PATTERN"
    echo "bench_time=$BENCH_TIME"
    echo "bench_count=$BENCH_COUNT"
    echo "go_version=$(go version)"
    echo "commit=$(git rev-parse --short HEAD 2>/dev/null || echo desconocido)"
} > "$metadata_file"

echo "Ejecutando benchmarks Go"

go test ./benchmarks \
    -run '^$' \
    -bench "$BENCH_PATTERN" \
    -benchmem \
    -benchtime "$BENCH_TIME" \
    -count "$BENCH_COUNT" | tee "$result_file"

echo "Resultados generados en $REPORT_DIR"
