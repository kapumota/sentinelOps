#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

PROFILE="${1:-hardened}"
mkdir -p reports

python3 tools/audit/audit.py   --profile "${PROFILE}"   --project-root "." | tee "reports/audit-${PROFILE}.json"
