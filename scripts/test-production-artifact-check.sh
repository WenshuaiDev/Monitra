#!/usr/bin/env bash

set -euo pipefail

repository_root="$(cd "$(dirname "$0")/.." && pwd -P)"
checker="${repository_root}/scripts/check-production-artifact.sh"
fixtures="${repository_root}/tests/production/artifacts"
temporary_parent="$(cd "${TMPDIR:-/tmp}" && pwd -P)"
test_directory="$(mktemp -d "${temporary_parent}/monitra-artifact-check.XXXXXX")"

cleanup() {
  if [[ -d "${test_directory}" && "${test_directory}" == "${temporary_parent}"/monitra-artifact-check.* ]]; then
    find "${test_directory}" -xdev -type f -delete
    rmdir "${test_directory}"
  fi
}
trap cleanup EXIT

"${checker}" "${fixtures}/clean.manifest" clean-fixture >"${test_directory}/clean.log"
grep -Fq 'PRODUCTION_ARTIFACT_CLEAN label=clean-fixture' "${test_directory}/clean.log"

set +e
"${checker}" "${fixtures}/contaminated.manifest" contaminated-fixture >"${test_directory}/contaminated.log" 2>&1
contaminated_status=$?
set -e

if (( contaminated_status == 0 )); then
  printf '%s\n' 'production artifact checker accepted the contaminated fixture' >&2
  exit 1
fi

grep -Fq 'PRODUCTION_ARTIFACT_REJECTED label=contaminated-fixture' "${test_directory}/contaminated.log"
grep -Fq 'dependencies/apk/chromium' "${test_directory}/contaminated.log"
grep -Fq 'node_modules/@playwright/test/index.js' "${test_directory}/contaminated.log"
grep -Fq 'opt/google/chrome/chrome' "${test_directory}/contaminated.log"
grep -Fq 'srv/__e2e.js' "${test_directory}/contaminated.log"
grep -Fq 'srv/e2e/startup.spec.ts' "${test_directory}/contaminated.log"
grep -Fq 'srv/playwright.config.ts' "${test_directory}/contaminated.log"
grep -Fq 'usr/local/bin/http-server' "${test_directory}/contaminated.log"
grep -Fq 'usr/bin/vite' "${test_directory}/contaminated.log"

printf 'PRODUCTION_ARTIFACT_FIXTURES_OK rejected_status=%s\n' "${contaminated_status}"
