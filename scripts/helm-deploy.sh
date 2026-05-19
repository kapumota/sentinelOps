#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

PROFILE="${1:-hardened}"
NAMESPACE="${2:-sentinelops}"
RELEASE="${3:-sentinelops}"

VALUES_FILE="deploy/helm/sentinelops/values-${PROFILE}.yaml"

if [[ ! -f "${VALUES_FILE}" ]]; then
  echo "No existe ${VALUES_FILE}"
  exit 1
fi

kubectl apply -f deploy/kubernetes/namespace.yaml

helm upgrade --install "${RELEASE}" deploy/helm/sentinelops   --namespace "${NAMESPACE}"   -f deploy/helm/sentinelops/values.yaml   -f "${VALUES_FILE}"

echo "Release desplegado."
echo "namespace=${NAMESPACE}"
echo "release=${RELEASE}"
echo "profile=${PROFILE}"
