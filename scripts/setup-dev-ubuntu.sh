#!/usr/bin/env bash
set -Eeuo pipefail
IFS=$'\n\t'

GO_VERSION="1.25.0"
OPA_VERSION="0.67.1"
HELM_VERSION="v3.14.4"

SCRIPT_NAME="$(basename "$0")"
TMP_DIR="$(mktemp -d)"

cleanup() {
  rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

log() {
  echo
  echo "==> $*"
}

die() {
  echo "Error: $*" >&2
  exit 1
}

confirm() {
  echo "Este script instalará herramientas del sistema usando sudo:"
  echo "- Go ${GO_VERSION}"
  echo "- Rust"
  echo "- OPA ${OPA_VERSION}"
  echo "- Helm ${HELM_VERSION}"
  echo "- Docker Engine"
  echo "- Python 3.11 y herramientas base"
  echo
  read -rp "¿Continuar? [y/N]: " answer

  case "${answer}" in
    y|Y|yes|YES)
      ;;
    *)
      echo "Cancelado."
      exit 0
      ;;
  esac
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "Comando requerido no encontrado: $1"
}

check_os() {
  [[ -f /etc/os-release ]] || die "No se encontró /etc/os-release."

  # shellcheck source=/dev/null
  . /etc/os-release

  if [[ "${ID:-}" != "ubuntu" ]]; then
    die "Este script está diseñado para Ubuntu. Sistema detectado: ${PRETTY_NAME:-desconocido}"
  fi

  UBUNTU_CODENAME_DETECTED="${UBUNTU_CODENAME:-${VERSION_CODENAME:-}}"
  [[ -n "${UBUNTU_CODENAME_DETECTED}" ]] || die "No se pudo detectar el codename de Ubuntu."
}

detect_architecture() {
  ARCH="$(dpkg --print-architecture)"

  case "${ARCH}" in
    amd64)
      GO_ARCH="amd64"
      OPA_ARCH="amd64"
      HELM_ARCH="amd64"
      ;;
    arm64)
      GO_ARCH="arm64"
      OPA_ARCH="arm64"
      HELM_ARCH="arm64"
      ;;
    *)
      die "Arquitectura no soportada automáticamente: ${ARCH}"
      ;;
  esac
}

install_apt_dependencies() {
  log "Instalando dependencias base con apt"

  sudo apt update

  sudo apt install -y \
    build-essential \
    ca-certificates \
    curl \
    wget \
    git \
    make \
    jq \
    unzip \
    tar \
    gzip \
    gnupg \
    lsb-release \
    openssh-client \
    netcat-openbsd \
    pkg-config \
    software-properties-common \
    python3.11 \
    python3.11-venv \
    python3.11-dev
}

install_go() {
  log "Instalando Go ${GO_VERSION}"

  local archive="go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
  local url="https://go.dev/dl/${archive}"

  cd "${TMP_DIR}"

  curl -fL --retry 3 --connect-timeout 15 -o "${archive}" "${url}"

  sudo rm -rf /usr/local/go
  sudo tar -C /usr/local -xzf "${archive}"

  if ! grep -q "/usr/local/go/bin" "${HOME}/.profile" 2>/dev/null; then
    echo 'export PATH=$PATH:/usr/local/go/bin' >> "${HOME}/.profile"
  fi

  export PATH="${PATH}:/usr/local/go/bin"
}

install_rust() {
  log "Instalando Rust"

  if command -v cargo >/dev/null 2>&1 && command -v rustc >/dev/null 2>&1; then
    echo "Rust ya está instalado."
    return
  fi

  curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y

  if [[ -f "${HOME}/.cargo/env" ]]; then
    # shellcheck source=/dev/null
    source "${HOME}/.cargo/env"
  else
    die "Rustup terminó, pero no se encontró ${HOME}/.cargo/env"
  fi
}

install_opa() {
  log "Instalando OPA ${OPA_VERSION}"

  local url="https://openpolicyagent.org/downloads/v${OPA_VERSION}/opa_linux_${OPA_ARCH}"
  local output="${TMP_DIR}/opa"

  curl -fL --retry 3 --connect-timeout 15 -o "${output}" "${url}"
  chmod 755 "${output}"
  sudo mv "${output}" /usr/local/bin/opa
}

install_helm() {
  log "Instalando Helm ${HELM_VERSION}"

  local archive="helm-${HELM_VERSION}-linux-${HELM_ARCH}.tar.gz"
  local url="https://get.helm.sh/${archive}"

  cd "${TMP_DIR}"

  curl -fL --retry 3 --connect-timeout 15 -o "${archive}" "${url}"
  tar -xzf "${archive}"

  sudo mv "linux-${HELM_ARCH}/helm" /usr/local/bin/helm
}

install_docker() {
  log "Instalando Docker Engine"

  sudo apt remove -y \
    docker.io \
    docker-compose \
    docker-compose-v2 \
    docker-doc \
    podman-docker \
    containerd \
    runc || true

  sudo install -m 0755 -d /etc/apt/keyrings

  sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
    -o /etc/apt/keyrings/docker.asc

  sudo chmod a+r /etc/apt/keyrings/docker.asc

  sudo tee /etc/apt/sources.list.d/docker.sources >/dev/null <<EOF
Types: deb
URIs: https://download.docker.com/linux/ubuntu
Suites: ${UBUNTU_CODENAME_DETECTED}
Components: stable
Architectures: ${ARCH}
Signed-By: /etc/apt/keyrings/docker.asc
EOF

  sudo apt update

  sudo apt install -y \
    docker-ce \
    docker-ce-cli \
    containerd.io \
    docker-buildx-plugin \
    docker-compose-plugin

  sudo systemctl enable --now docker
}

print_versions() {
  log "Versiones instaladas"

  go version
  python3.11 --version
  rustc --version
  cargo --version
  opa version
  helm version
  docker version
  docker compose version
}

print_next_steps() {
  echo
  echo "Listo."
  echo
  echo "Para usar Docker sin sudo, ejecuta:"
  echo
  echo "  sudo usermod -aG docker \$USER"
  echo
  echo "Luego cierra sesión y vuelve a entrar."
  echo
  echo "También puedes recargar tu perfil con:"
  echo
  echo "  source ~/.profile"
}

main() {
  echo "${SCRIPT_NAME}: preparación de entorno de desarrollo para SentinelOps"
  echo

  require_command sudo
  require_command dpkg

  check_os
  detect_architecture
  confirm

  install_apt_dependencies
  install_go
  install_rust
  install_opa
  install_helm
  install_docker
  print_versions
  print_next_steps
}

main "$@"