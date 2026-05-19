#!/usr/bin/env bash
set -Eeuo pipefail
IFS=$'\n\t'

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

DRY_RUN=0
DEEP=0
DOCKER=1

usage() {
  cat <<'EOF'
Uso:
  scripts/cleanup.sh [opciones]

Opciones:
  --dry-run     Muestra qué se borraría, sin borrar nada.
  --deep        Limpieza más agresiva: borra data local y builds Rust.
  --no-docker   No intenta borrar contenedores Docker.
  -h, --help    Muestra esta ayuda.

Ejemplos:
  scripts/cleanup.sh
  scripts/cleanup.sh --dry-run
  scripts/cleanup.sh --deep
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    --deep)
      DEEP=1
      shift
      ;;
    --no-docker)
      DOCKER=0
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Opción desconocida: $1" >&2
      usage
      exit 1
      ;;
  esac
done

run() {
  if [[ "${DRY_RUN}" -eq 1 ]]; then
    printf '+'
    printf ' %q' "$@"
    printf '\n'
  else
    "$@"
  fi
}

remove_path() {
  local path="$1"

  if [[ -e "${path}" || -L "${path}" ]]; then
    run rm -rf -- "${path}"
  fi
}

remove_find_matches() {
  while IFS= read -r path; do
    remove_path "${path}"
  done
}

echo "Limpiando SentinelOps en: ${ROOT_DIR}"

# 1. Docker local
if [[ "${DOCKER}" -eq 1 ]]; then
  if command -v docker >/dev/null 2>&1; then
    run docker rm -f sentinelops-local >/dev/null 2>&1 || true
  else
    echo "Docker no está instalado; se omite limpieza de contenedor."
  fi
fi

# 2. Reportes y artefactos comunes
remove_path "coverage.out"
remove_path "coverage.html"
remove_path ".pytest_cache"
remove_path ".mypy_cache"
remove_path ".ruff_cache"
remove_path "bin"
remove_path "dist"
remove_path "tmp"
remove_path ".tmp"

# Mantener la carpeta reports, pero vaciar su contenido.
if [[ -d "reports" ]]; then
  if [[ "${DRY_RUN}" -eq 1 ]]; then
    find reports -mindepth 1 -print
  else
    find reports -mindepth 1 -exec rm -rf {} +
  fi
else
  run mkdir -p reports
fi

# 3. Cachés Python
find . \
  -path "./.git" -prune -o \
  -type d -name "__pycache__" -print | remove_find_matches

find . \
  -path "./.git" -prune -o \
  -type f \( -name "*.pyc" -o -name "*.pyo" \) -print | remove_find_matches

# 4. Logs y basura de sistema/editor
find . \
  -path "./.git" -prune -o \
  -type f \( \
    -name "*.log" -o \
    -name ".DS_Store" -o \
    -name "Thumbs.db" -o \
    -name "*~" -o \
    -name "*.swp" -o \
    -name "*.swo" \
  \) -print | remove_find_matches

# 5. Limpieza profunda opcional
if [[ "${DEEP}" -eq 1 ]]; then
  remove_path "data"
  remove_path "rust/input-guard/target"
fi

# 6. Aviso sobre archivos sensibles que NO se borran automáticamente
SENSITIVE_FILES="$(find . \
  -path "./.git" -prune -o \
  -type f \( \
    -name ".env" -o \
    -name "*.env" -o \
    -name "*.key" -o \
    -name "*.pem" -o \
    -name "*.crt" -o \
    -name "id_rsa" -o \
    -name "id_ed25519" \
  \) -print || true)"

if [[ -n "${SENSITIVE_FILES}" ]]; then
  echo
  echo "Aviso: se encontraron archivos potencialmente sensibles."
  echo "No se borraron automáticamente. Revísalos antes de subir a GitHub:"
  echo "${SENSITIVE_FILES}"
fi

echo
echo "Limpieza completada."
