#!/usr/bin/env bash
set -euo pipefail

PROFILE="${1:-hardened}"
TMP_FILE="$(mktemp)"

cleanup() {
  rm -f "${TMP_FILE}"
}
trap cleanup EXIT

if [[ "${PROFILE}" == "insecure" ]]; then
cat > "${TMP_FILE}" <<'JSON'
{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {
    "name": "sentinelops",
    "labels": {
      "app": "sentinelops",
      "profile": "insecure"
    }
  },
  "spec": {
    "template": {
      "spec": {
        "containers": [
          {
            "name": "server",
            "image": "sentinelops:latest",
            "securityContext": {
              "privileged": true,
              "runAsNonRoot": false,
              "allowPrivilegeEscalation": true,
              "readOnlyRootFilesystem": false
            }
          }
        ]
      }
    }
  }
}
JSON
else
cat > "${TMP_FILE}" <<'JSON'
{
  "apiVersion": "apps/v1",
  "kind": "Deployment",
  "metadata": {
    "name": "sentinelops",
    "labels": {
      "app": "sentinelops",
      "profile": "hardened"
    }
  },
  "spec": {
    "template": {
      "spec": {
        "containers": [
          {
            "name": "server",
            "image": "sentinelops:1.0.0",
            "securityContext": {
              "privileged": false,
              "runAsNonRoot": true,
              "allowPrivilegeEscalation": false,
              "readOnlyRootFilesystem": true
            }
          }
        ]
      }
    }
  }
}
JSON
fi

echo "deny"
opa eval --format=pretty --data policies/kubernetes --input "${TMP_FILE}" 'data.kubernetes.security.deny'
echo
echo "warn"
opa eval --format=pretty --data policies/kubernetes --input "${TMP_FILE}" 'data.kubernetes.security.warn'
