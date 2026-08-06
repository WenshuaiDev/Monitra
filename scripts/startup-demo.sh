#!/usr/bin/env bash

set -euo pipefail

demo_parent="$(cd "${TMPDIR:-/tmp}" && pwd -P)"
demo_directory="$(mktemp -d "${demo_parent}/monitra-startup-demo.XXXXXX")"
postgres_container=""
core_pid=""

cleanup() {
  if [[ -n "${core_pid}" ]] && kill -0 "${core_pid}" 2>/dev/null; then
    kill -TERM "${core_pid}" 2>/dev/null || true
    wait "${core_pid}" 2>/dev/null || true
  fi
  if [[ "${postgres_container}" =~ ^[0-9a-f]{64}$ ]]; then
    docker stop --time 2 "${postgres_container}" >/dev/null 2>&1 || true
  fi
  if [[ -d "${demo_directory}" && "${demo_directory}" == "${demo_parent}"/monitra-startup-demo.* ]]; then
    find "${demo_directory}" -type f -delete
    rmdir "${demo_directory}"
  fi
}
trap cleanup EXIT

wait_for_log_field() {
  local log_file="$1"
  local field="$2"
  local deadline=$((SECONDS + 5))
  local value=""
  while (( SECONDS < deadline )); do
    value="$(sed -n "s/.*\"${field}\":\"\([^\"]*\)\".*/\1/p" "${log_file}" | head -n 1)"
    if [[ -n "${value}" ]]; then
      printf '%s\n' "${value}"
      return 0
    fi
    sleep 0.05
  done
  return 1
}

wait_for_ready() {
  local address="$1"
  local deadline=$((SECONDS + 12))
  while (( SECONDS < deadline )); do
    if curl --fail --silent "http://${address}/readyz"; then
      return 0
    fi
    sleep 0.1
  done
  return 1
}

printf '%s\n' 'demo-secret-never-log' >"${demo_directory}/postgres-password"
chmod 600 "${demo_directory}/postgres-password"
go build -o "${demo_directory}/monitra" ./cmd/monitra

postgres_container="$(docker run --rm --detach \
  --publish 127.0.0.1::5432 \
  --env POSTGRES_PASSWORD=demo-secret-never-log \
  --env POSTGRES_USER=monitra \
  --env POSTGRES_DB=monitra \
  postgres:18.4-alpine)"
if [[ ! "${postgres_container}" =~ ^[0-9a-f]{64}$ ]]; then
  printf 'Docker returned an unexpected container ID: %s\n' "${postgres_container}" >&2
  exit 1
fi
postgres_address="$(docker port "${postgres_container}" 5432/tcp)"
postgres_host="${postgres_address%:*}"
postgres_port="${postgres_address##*:}"

printf '%s\n' 'SUCCESS PATH'
MONITRA_RELEASE_IDENTITY=demo \
MONITRA_MANAGEMENT_ADDRESS=127.0.0.1:0 \
MONITRA_POSTGRES_HOST="${postgres_host}" \
MONITRA_POSTGRES_PORT="${postgres_port}" \
MONITRA_POSTGRES_DATABASE=monitra \
MONITRA_POSTGRES_USER=monitra \
MONITRA_POSTGRES_PASSWORD_FILE="${demo_directory}/postgres-password" \
MONITRA_POSTGRES_SSL_MODE=disable \
MONITRA_POSTGRES_MAX_CONNECTIONS=2 \
MONITRA_POSTGRES_STARTUP_TIMEOUT=10s \
  "${demo_directory}/monitra" >"${demo_directory}/success.log" 2>&1 &
core_pid=$!
management_address="$(wait_for_log_field "${demo_directory}/success.log" address)"
printf 'liveness: '
curl --fail --silent --show-error "http://${management_address}/livez"
printf 'readiness while PostgreSQL initializes: '
curl --silent --show-error "http://${management_address}/readyz"
printf 'readiness after PostgreSQL connects: '
wait_for_ready "${management_address}"
kill -TERM "${core_pid}"
wait "${core_pid}"
core_pid=""
printf '%s\n' 'success exit code: 0'

docker stop --time 2 "${postgres_container}" >/dev/null
postgres_container=""

printf '%s\n' 'FAILURE PATH'
MONITRA_RELEASE_IDENTITY=demo \
MONITRA_MANAGEMENT_ADDRESS=127.0.0.1:0 \
MONITRA_POSTGRES_HOST="${postgres_host}" \
MONITRA_POSTGRES_PORT="${postgres_port}" \
MONITRA_POSTGRES_DATABASE=monitra \
MONITRA_POSTGRES_USER=monitra \
MONITRA_POSTGRES_PASSWORD_FILE="${demo_directory}/postgres-password" \
MONITRA_POSTGRES_SSL_MODE=disable \
MONITRA_POSTGRES_MAX_CONNECTIONS=2 \
MONITRA_POSTGRES_STARTUP_TIMEOUT=900ms \
  "${demo_directory}/monitra" >"${demo_directory}/failure.log" 2>&1 &
core_pid=$!
management_address="$(wait_for_log_field "${demo_directory}/failure.log" address)"
printf 'liveness: '
curl --fail --silent --show-error "http://${management_address}/livez"
printf 'readiness: '
curl --silent --show-error "http://${management_address}/readyz"
set +e
wait "${core_pid}"
failure_exit=$?
set -e
core_pid=""
if (( failure_exit == 0 )); then
  printf '%s\n' 'failure path unexpectedly exited zero' >&2
  exit 1
fi
printf 'bounded failure exit code: %d\n' "${failure_exit}"
grep -F '"dependency":"postgresql"' "${demo_directory}/failure.log"
if grep -F 'demo-secret-never-log' "${demo_directory}/success.log" "${demo_directory}/failure.log"; then
  printf '%s\n' 'Secret leaked into structured logs' >&2
  exit 1
fi
