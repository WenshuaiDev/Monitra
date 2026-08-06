#!/usr/bin/env bash

set -euo pipefail

temporary_parent="$(cd "${TMPDIR:-/tmp}" && pwd -P)"
generated_directory="$(mktemp -d "${temporary_parent}/monitra-api-check.XXXXXX")"
cleanup() {
  if [[ -d "${generated_directory}" && "${generated_directory}" == "${temporary_parent}"/monitra-api-check.* ]]; then
    find "${generated_directory}" -type f -delete
    rmdir "${generated_directory}"
  fi
}
trap cleanup EXIT

node scripts/generate-api-client.mjs "${generated_directory}"
diff -u web/src/api/generated.ts "${generated_directory}/generated.ts"
diff -u web/src/api/generated-client.ts "${generated_directory}/generated-client.ts"
