#!/usr/bin/env bash
# Verifica que no queden credenciales de laboratorio versionadas.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

student_secret="student""123!"
teacher_secret="teacher""123!"
auditor_secret="auditor""123!"
admin_secret="admin""123!"
patterns="${student_secret}|${teacher_secret}|${auditor_secret}|${admin_secret}"

if grep -RInE "${patterns}" . \
  --exclude-dir=.git \
  --exclude-dir=bin \
  --exclude-dir=dist \
  --exclude-dir=rust/input-guard/target \
  --exclude='.env.local' \
  --exclude='*.patch' \
  --exclude='*.diff'; then
  echo "Credenciales hardcodeadas detectadas. Reemplaza esos valores por variables o placeholders."
  exit 1
fi

echo "No se detectaron credenciales hardcodeadas conocidas."
