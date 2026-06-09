#!/usr/bin/env bash
# Funciones comunes para cargar .env.local en scripts de desarrollo.

load_env_local() {
  local root_dir="$1"
  local env_file="${root_dir}/.env.local"

  if [[ -f "${env_file}" ]]; then
    set -a
    # shellcheck source=/dev/null
    source "${env_file}"
    set +a
  fi
}

ensure_env_local() {
  local root_dir="$1"
  if [[ ! -f "${root_dir}/.env.local" ]]; then
    bash "${root_dir}/scripts/generate-secrets.sh"
  fi
}
