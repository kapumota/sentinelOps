#!/bin/sh
# Prepara credenciales temporales cuando no se inyectan variables de entorno.

set -eu

generate_password() {
  LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 24 || true
}

ensure_var() {
  name="$1"
  description="$2"
  current_value="$(eval "printf '%s' \"\${${name}:-}\"")"

  if [ -z "${current_value}" ]; then
    generated="$(generate_password)"
    export "${name}=${generated}"
    echo "------------------------------------------------------------"
    echo "Credencial temporal generada para ${description}"
    echo "Variable: ${name}"
    echo "Valor: ${generated}"
    echo "------------------------------------------------------------"
  fi
}

ensure_var APP_CONTROL_API_PASSWORD "API de control"
ensure_var LAB_PASSWORD_STUDENT "usuario student"
ensure_var LAB_PASSWORD_TEACHER "usuario teacher"
ensure_var LAB_PASSWORD_AUDITOR "usuario auditor"
ensure_var LAB_PASSWORD_ADMIN "usuario admin"

exec "$@"
