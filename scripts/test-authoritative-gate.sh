#!/usr/bin/env bash

set -euo pipefail

repository_root="$(cd "$(dirname "$0")/.." && pwd -P)"
runner="${repository_root}/scripts/run-authoritative-gate.sh"
fixture="${repository_root}/tests/production/authoritative-gate.Taskfile.yml"
temporary_parent="$(cd "${TMPDIR:-/tmp}" && pwd -P)"
test_directory="$(mktemp -d "${temporary_parent}/monitra-authoritative-gate.XXXXXX")"

cleanup() {
  if [[ -d "${test_directory}" && "${test_directory}" == "${temporary_parent}"/monitra-authoritative-gate.* ]]; then
    find "${test_directory}" -xdev -type f -delete
    rmdir "${test_directory}"
  fi
}
trap cleanup EXIT

expected_all="${test_directory}/expected-all.log"
expected_before_failure="${test_directory}/expected-before-failure.log"
success_log="${test_directory}/success.log"
failure_log="${test_directory}/failure.log"

printf '%s\n' \
  'check:format' \
  'test:authoritative-gate' \
  'check:go' \
  'api:check' \
  'check:web' \
  'test:integration' \
  'test:production' \
  'test:e2e' \
  'test:production:database-lifecycle' \
  'test:production:artifact-fixtures' \
  'test:production:artifacts' >"${expected_all}"
printf '%s\n' 'check:format' 'test:authoritative-gate' 'check:go' 'api:check' >"${expected_before_failure}"

MONITRA_CHECK_TASKFILE="${fixture}" \
MONITRA_GATE_PROBE_LOG="${success_log}" \
  "${runner}" >/dev/null
cmp "${expected_all}" "${success_log}"

set +e
MONITRA_CHECK_TASKFILE="${fixture}" \
MONITRA_GATE_PROBE_LOG="${failure_log}" \
MONITRA_GATE_PROBE_FAIL_API=1 \
  "${runner}" >"${test_directory}/failure-output.log" 2>&1
failure_status=$?
set -e

if (( failure_status == 0 )); then
  printf '%s\n' 'authoritative gate swallowed a child failure' >&2
  exit 1
fi
cmp "${expected_before_failure}" "${failure_log}"
grep -Fq 'AUTHORITATIVE_GATE_FAILED step=api:check' "${test_directory}/failure-output.log"

printf 'AUTHORITATIVE_GATE_CONTRACT_OK steps=11 failure_status=%s\n' "${failure_status}"
