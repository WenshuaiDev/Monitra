#!/usr/bin/env bash

set -euo pipefail
shopt -s nocasematch

if (( $# != 2 )); then
  printf 'usage: %s <filesystem-manifest> <label>\n' "$0" >&2
  exit 2
fi

manifest=$1
label=$2

if [[ ! -f "${manifest}" ]]; then
  printf 'production artifact manifest does not exist: %s\n' "${manifest}" >&2
  exit 2
fi
if [[ -z "${label}" || "${label}" == *[!A-Za-z0-9._-]* ]]; then
  printf 'production artifact label is invalid: %s\n' "${label}" >&2
  exit 2
fi

entries=0
forbidden=0

while IFS= read -r artifact_path || [[ -n "${artifact_path}" ]]; do
  artifact_path=${artifact_path#./}
  artifact_path=${artifact_path#/}
  [[ -n "${artifact_path}" ]] || continue
  entries=$((entries + 1))
  reason=

  case "${artifact_path}" in
    usr/bin/test)
      ;;
    *playwright* | node_modules/* | */node_modules/* | */ms-playwright/* | */playwright-report/* | */test-results/*)
      reason=test-dependency
      ;;
    *chromium* | *chrome* | *firefox* | *webkit* | *browser-binary*)
      reason=browser-runtime
      ;;
    *e2e*)
      reason=test-control
      ;;
    dependencies/*/nodejs* | dependencies/*/npm | dependencies/*/pnpm | dependencies/*/vite* | dependencies/*/webpack* | */bin/node | */bin/npm | */bin/pnpm | */bin/yarn | */bin/serve | */bin/http-server | */bin/*dev-server* | */bin/vite* | */bin/webpack* | */bin/parcel | */bin/react-scripts | */bin/next)
      reason=test-or-development-runtime
      ;;
    *test* | *.spec.js | *.spec.jsx | *.spec.ts | *.spec.tsx)
      reason=test-source-or-control
      ;;
  esac

  if [[ -n "${reason}" ]]; then
    forbidden=$((forbidden + 1))
    printf 'FORBIDDEN_PRODUCTION_ARTIFACT label=%s reason=%s path=%s\n' \
      "${label}" "${reason}" "${artifact_path}" >&2
  fi
done <"${manifest}"

if (( forbidden > 0 )); then
  printf 'PRODUCTION_ARTIFACT_REJECTED label=%s entries=%s forbidden=%s\n' \
    "${label}" "${entries}" "${forbidden}" >&2
  exit 1
fi

printf 'PRODUCTION_ARTIFACT_CLEAN label=%s entries=%s forbidden=0\n' "${label}" "${entries}"
