#!/usr/bin/env bash

set -euo pipefail

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

  artifact_directory=${artifact_path%/*}
  if [[ "${artifact_directory}" == "${artifact_path}" ]]; then
    artifact_directory=
  fi
  case "/${artifact_directory}/" in
    */node_modules/* | */ms-playwright/* | */playwright-report/* | */test-results/*)
      reason=test-dependency
      ;;
    */e2e/* | */test/* | */tests/*)
      reason=test-source
      ;;
  esac

  artifact_name=${artifact_path##*/}
  case "${artifact_name}" in
    *.spec.js | *.spec.jsx | *.spec.ts | *.spec.tsx | *.test.js | *.test.jsx | *.test.ts | *.test.tsx | *_test.go)
      reason=${reason:-test-source}
      ;;
  esac

  case "${artifact_path}" in
    dependencies/*/playwright* | dependencies/*/chromium* | dependencies/*/google-chrome* | dependencies/*/firefox* | dependencies/*/webkit* | dependencies/*/nodejs* | dependencies/*/npm | dependencies/*/pnpm | dependencies/*/vite* | dependencies/*/webpack*)
      reason=${reason:-test-or-development-dependency}
      ;;
    */bin/playwright | */bin/chromium | */bin/chrome | */bin/firefox | */bin/webkit | */bin/vite | */bin/vite-node | */bin/webpack | */bin/webpack-dev-server | */bin/node | */bin/npm | */bin/pnpm | */bin/yarn)
      reason=${reason:-test-or-development-runtime}
      ;;
    *e2e-control* | *e2e_control* | *test-control* | *test_control*)
      reason=${reason:-test-control}
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
