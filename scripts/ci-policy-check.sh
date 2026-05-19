#!/usr/bin/env bash
set -euo pipefail

PROFILE="${1:-hardened}"
EXPECTED="${2:-pass}"

mkdir -p reports

TMP_FILE="$(mktemp)"
OUT_FILE="reports/policy-${PROFILE}.json"

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

DENY_JSON="$(opa eval --format=json --data policies/kubernetes --input "${TMP_FILE}" 'data.kubernetes.security.deny')"
WARN_JSON="$(opa eval --format=json --data policies/kubernetes --input "${TMP_FILE}" 'data.kubernetes.security.warn')"

DENY_COUNT="$(printf '%s' "${DENY_JSON}" | python3 -c 'import sys, json
data = json.load(sys.stdin)
items = []
if data.get("result") and data["result"][0].get("expressions"):
    value = data["result"][0]["expressions"][0].get("value")
    if isinstance(value, list):
        items = value
    elif value is not None:
        items = [value]
print(len(items))')"

WARN_COUNT="$(printf '%s' "${WARN_JSON}" | python3 -c 'import sys, json
data = json.load(sys.stdin)
items = []
if data.get("result") and data["result"][0].get("expressions"):
    value = data["result"][0]["expressions"][0].get("value")
    if isinstance(value, list):
        items = value
    elif value is not None:
        items = [value]
print(len(items))')"

STATUS="pass"
if [[ "${DENY_COUNT}" -gt 0 ]]; then
  STATUS="fail"
elif [[ "${WARN_COUNT}" -gt 0 ]]; then
  STATUS="warn"
fi

DENY_JSON_ENV="${DENY_JSON}" WARN_JSON_ENV="${WARN_JSON}" PROFILE_ENV="${PROFILE}" EXPECTED_ENV="${EXPECTED}" STATUS_ENV="${STATUS}" DENY_COUNT_ENV="${DENY_COUNT}" WARN_COUNT_ENV="${WARN_COUNT}" python3 - <<'PY' > "${OUT_FILE}"
import json
import os

deny = json.loads(os.environ["DENY_JSON_ENV"])
warn = json.loads(os.environ["WARN_JSON_ENV"])

def extract_items(payload):
    if not payload.get("result"):
        return []
    expressions = payload["result"][0].get("expressions", [])
    if not expressions:
        return []
    value = expressions[0].get("value")
    if isinstance(value, list):
        return value
    if value is None:
        return []
    return [value]

result = {
    "profile": os.environ["PROFILE_ENV"],
    "expected": os.environ["EXPECTED_ENV"],
    "status": os.environ["STATUS_ENV"],
    "deny_count": int(os.environ["DENY_COUNT_ENV"]),
    "warn_count": int(os.environ["WARN_COUNT_ENV"]),
    "denies": extract_items(deny),
    "warnings": extract_items(warn),
}
print(json.dumps(result, indent=2))
PY

cat "${OUT_FILE}"

if [[ "${EXPECTED}" == "pass" && "${STATUS}" != "pass" ]]; then
  echo "Policy gate failed: expected pass, got ${STATUS}"
  exit 1
fi

if [[ "${EXPECTED}" == "fail" && "${STATUS}" != "fail" ]]; then
  echo "Policy gate failed: expected fail, got ${STATUS}"
  exit 1
fi

echo "Policy gate OK for profile=${PROFILE}, status=${STATUS}"
