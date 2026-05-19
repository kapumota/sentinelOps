#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

KEY_DIR="data/ssh/client"
AUTH_DIR="data/ssh/authorized_keys"
USER_NAME="${1:-student}"
KEY_PATH="${KEY_DIR}/${USER_NAME}_ed25519"
PUB_PATH="${KEY_PATH}.pub"
AUTHORIZED_KEYS_PATH="${AUTH_DIR}/${USER_NAME}"

mkdir -p "${KEY_DIR}" "${AUTH_DIR}"

if [[ ! -f "${KEY_PATH}" ]]; then
  ssh-keygen -t ed25519 -N "" -f "${KEY_PATH}" -C "${USER_NAME}@sentinelops-lab"
fi

touch "${AUTHORIZED_KEYS_PATH}"
chmod 600 "${AUTHORIZED_KEYS_PATH}"

if ! grep -Fqx "$(cat "${PUB_PATH}")" "${AUTHORIZED_KEYS_PATH}" 2>/dev/null; then
  cat "${PUB_PATH}" >> "${AUTHORIZED_KEYS_PATH}"
fi

echo "SSH lab key prepared."
echo "User:           ${USER_NAME}"
echo "Private key:    ${KEY_PATH}"
echo "Public key:     ${PUB_PATH}"
echo "authorized_keys ${AUTHORIZED_KEYS_PATH}"
echo
echo "Prueba sugerida:"
echo "  make run-ssh"
echo "  go run ./cmd/client --addr localhost:2222 --user ${USER_NAME} --identity ${KEY_PATH}"
