#!/usr/bin/env bash
set -euo pipefail

PROFILE="${1:-hardened}"
CHART_DIR="deploy/helm/sentinelops"
BASE_VALUES="${CHART_DIR}/values.yaml"
PROFILE_VALUES="${CHART_DIR}/values-${PROFILE}.yaml"
REPORT_DIR="reports/helm"
OUTPUT_FILE="${REPORT_DIR}/rendered-${PROFILE}.yaml"

mkdir -p "${REPORT_DIR}"

if [[ ! -d "${CHART_DIR}" ]]; then
  echo "Helm chart directory not found: ${CHART_DIR}"
  exit 1
fi

if [[ ! -f "${BASE_VALUES}" ]]; then
  echo "Base values file not found: ${BASE_VALUES}"
  exit 1
fi

if [[ ! -f "${PROFILE_VALUES}" ]]; then
  echo "Profile values file not found: ${PROFILE_VALUES}"
  exit 1
fi

helm lint "${CHART_DIR}"   -f "${BASE_VALUES}"   -f "${PROFILE_VALUES}"

helm template sentinelops "${CHART_DIR}"   -f "${BASE_VALUES}"   -f "${PROFILE_VALUES}"   > "${OUTPUT_FILE}"

if [[ ! -s "${OUTPUT_FILE}" ]]; then
  echo "Rendered manifest is empty: ${OUTPUT_FILE}"
  exit 1
fi

grep -q "kind: Deployment" "${OUTPUT_FILE}" || {
  echo "Rendered manifest does not contain a Deployment"
  exit 1
}

grep -q "kind: Service" "${OUTPUT_FILE}" || {
  echo "Rendered manifest does not contain a Service"
  exit 1
}

if [[ "${PROFILE}" == "hardened" ]]; then
  grep -q "runAsNonRoot: true" "${OUTPUT_FILE}" || {
    echo "Hardened profile must render runAsNonRoot: true"
    exit 1
  }

  grep -q "allowPrivilegeEscalation: false" "${OUTPUT_FILE}" || {
    echo "Hardened profile must render allowPrivilegeEscalation: false"
    exit 1
  }

  grep -q "readOnlyRootFilesystem: true" "${OUTPUT_FILE}" || {
    echo "Hardened profile must render readOnlyRootFilesystem: true"
    exit 1
  }
fi

if [[ "${PROFILE}" == "insecure" ]]; then
  grep -q "runAsNonRoot: false" "${OUTPUT_FILE}" || {
    echo "Insecure profile must render runAsNonRoot: false"
    exit 1
  }

  grep -q "allowPrivilegeEscalation: true" "${OUTPUT_FILE}" || {
    echo "Insecure profile must render allowPrivilegeEscalation: true"
    exit 1
  }

  grep -q "privileged: true" "${OUTPUT_FILE}" || {
    echo "Insecure profile must render privileged: true"
    exit 1
  }
fi

echo "Helm validation OK for profile=${PROFILE}"
echo "Rendered manifest saved to ${OUTPUT_FILE}"
