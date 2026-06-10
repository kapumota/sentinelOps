#!/usr/bin/env bash
set -euo pipefail

log_info() {
    echo "[release-clean] $1"
}

remove_path() {
    local target="$1"
    if [ -e "$target" ]; then
        rm -rf "$target"
        log_info "eliminado $target"
    fi
}

log_info "limpiando artefactos locales de release"

remove_path "coverage.out"
remove_path "coverage.html"
remove_path "reports/benchmarks"
remove_path "reports/runtime"
remove_path "reports/release"
remove_path "policies/bundle"
remove_path "gen/go"

find . -type d -name "__pycache__" -prune -exec rm -rf {} +
find . -type f -name "*.pyc" -delete

log_info "limpieza finalizada"
