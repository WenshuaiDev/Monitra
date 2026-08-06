#!/usr/bin/env bash

set -euo pipefail

repository_root="$(cd "$(dirname "$0")/.." && pwd -P)"
taskfile="${MONITRA_CHECK_TASKFILE:-${repository_root}/Taskfile.yml}"

if [[ ! -f "${taskfile}" ]]; then
  printf 'authoritative gate Taskfile does not exist: %s\n' "${taskfile}" >&2
  exit 2
fi

gate_steps=(
  check:format
  test:authoritative-gate
  check:go
  api:check
  check:web
  test:integration
  test:production
  test:e2e
  test:production:database-lifecycle
  test:production:artifact-fixtures
  test:production:artifacts
)

cd "${repository_root}"
step_index=0
for gate_step in "${gate_steps[@]}"; do
  step_index=$((step_index + 1))
  printf 'AUTHORITATIVE_GATE_START step=%s position=%s/%s\n' \
    "${gate_step}" "${step_index}" "${#gate_steps[@]}"
  if task --taskfile "${taskfile}" "${gate_step}"; then
    printf 'AUTHORITATIVE_GATE_OK step=%s position=%s/%s\n' \
      "${gate_step}" "${step_index}" "${#gate_steps[@]}"
  else
    step_status=$?
    printf 'AUTHORITATIVE_GATE_FAILED step=%s position=%s/%s status=%s\n' \
      "${gate_step}" "${step_index}" "${#gate_steps[@]}" "${step_status}" >&2
    exit "${step_status}"
  fi
done

printf 'AUTHORITATIVE_GATE_OK steps=%s\n' "${#gate_steps[@]}"
