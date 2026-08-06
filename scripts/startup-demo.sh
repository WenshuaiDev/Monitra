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
    docker rm --force "${postgres_container}" >/dev/null 2>&1 || true
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

wait_for_not_ready() {
  local address="$1"
  local deadline=$((SECONDS + 5))
  local status=""
  while (( SECONDS < deadline )); do
    status="$(curl --silent --output /dev/null --write-out '%{http_code}' "http://${address}/readyz" || true)"
    if [[ "${status}" == "503" ]]; then
      return 0
    fi
    sleep 0.1
  done
  return 1
}

printf '%s\n' 'demo-secret-never-log' >"${demo_directory}/postgres-password"
chmod 600 "${demo_directory}/postgres-password"
go build -o "${demo_directory}/monitra" ./cmd/monitra

postgres_port="$(python3 -c 'import socket; listener = socket.socket(); listener.bind(("127.0.0.1", 0)); print(listener.getsockname()[1]); listener.close()')"
postgres_container="$(docker create \
  --publish "127.0.0.1:${postgres_port}:5432" \
  --env POSTGRES_PASSWORD=demo-secret-never-log \
  --env POSTGRES_USER=monitra \
  --env POSTGRES_DB=monitra \
  postgres:18.4-alpine)"
if [[ ! "${postgres_container}" =~ ^[0-9a-f]{64}$ ]]; then
  printf 'Docker returned an unexpected container ID: %s\n' "${postgres_container}" >&2
  exit 1
fi
docker start "${postgres_container}" >/dev/null
postgres_address="127.0.0.1:${postgres_port}"
postgres_host="${postgres_address%:*}"

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
original_core_pid="${core_pid}"

printf '%s\n' 'RUNTIME RECOVERY PATH'
docker stop --time 2 "${postgres_container}" >/dev/null
printf 'liveness while PostgreSQL is unavailable: '
curl --fail --silent --show-error "http://${management_address}/livez"
printf 'readiness after PostgreSQL stops: '
wait_for_not_ready "${management_address}"
curl --silent --show-error "http://${management_address}/readyz"
sleep 1
if ! kill -0 "${original_core_pid}" 2>/dev/null; then
  printf '%s\n' 'core process exited during the PostgreSQL outage' >&2
  exit 1
fi
if [[ "$(curl --silent --output /dev/null --write-out '%{http_code}' "http://${management_address}/readyz")" != "503" ]]; then
  printf '%s\n' 'readiness did not remain unavailable while PostgreSQL was stopped' >&2
  exit 1
fi

docker start "${postgres_container}" >/dev/null
printf 'readiness after PostgreSQL returns: '
if ! wait_for_ready "${management_address}"; then
  printf '%s\n' 'core process did not recover readiness within the demonstration deadline' >&2
  tail -n 20 "${demo_directory}/success.log" >&2
  docker inspect --format 'postgresql status={{.State.Status}} exit={{.State.ExitCode}}' "${postgres_container}" >&2
  docker logs --tail 20 "${postgres_container}" >&2
  exit 1
fi
if [[ "${core_pid}" != "${original_core_pid}" ]] || ! kill -0 "${original_core_pid}" 2>/dev/null; then
  printf '%s\n' 'core process identity changed across PostgreSQL recovery' >&2
  exit 1
fi
printf 'core process PID remained: %s\n' "${original_core_pid}"
if [[ "$(grep -cF '"msg":"postgresql connection pool created"' "${demo_directory}/success.log")" != "1" ]]; then
  printf '%s\n' 'expected exactly one PostgreSQL connection pool creation' >&2
  exit 1
fi
grep -F '"msg":"required dependency unavailable"' "${demo_directory}/success.log"
grep -F '"msg":"required dependency restored"' "${demo_directory}/success.log"

kill -TERM "${core_pid}"
wait "${core_pid}"
core_pid=""
printf '%s\n' 'success exit code: 0'

docker stop --time 2 "${postgres_container}" >/dev/null

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
