#!/usr/bin/env bash

set -euo pipefail

repository_root="$(cd "$(dirname "$0")/.." && pwd -P)"
test_parent="$(cd "${TMPDIR:-/tmp}" && pwd -P)"
test_directory="$(mktemp -d "${test_parent}/monitra-production-database-lifecycle.XXXXXX")"
project_prefix="monitra-ticket07-$$"
failure_project="${project_prefix}-failure"
recovery_project="${project_prefix}-recovery"
secret_file="${test_directory}/postgresql-password"
production_compose="${repository_root}/compose.production.yaml"
lifecycle_compose="${repository_root}/tests/production/database-lifecycle.compose.yaml"
release_identity="ticket07-test"

timestamp() {
  date -u '+%Y-%m-%dT%H:%M:%SZ'
}

compose() {
  local project_name="$1"
  shift
  MONITRA_RELEASE_IDENTITY="${release_identity}" \
  MONITRA_POSTGRES_PASSWORD_FILE="${secret_file}" \
  MONITRA_WEB_PORT=0 \
    docker compose \
      --project-name "${project_name}" \
      --file "${production_compose}" \
      --file "${lifecycle_compose}" \
      "$@"
}

valid_project_name() {
  [[ "$1" =~ ^monitra-ticket07-[0-9]+-(failure|recovery)$ ]]
}

cleanup_project() {
  local project_name="$1"
  valid_project_name "${project_name}" || return 0
  compose "${project_name}" down --volumes --remove-orphans >/dev/null 2>&1 || true
}

cleanup() {
  cleanup_project "${failure_project}"
  cleanup_project "${recovery_project}"
  if [[ -d "${test_directory}" && "${test_directory}" == "${test_parent}"/monitra-production-database-lifecycle.* ]]; then
    find "${test_directory}" -xdev -type f -delete
    find "${test_directory}" -xdev -depth -type d -empty -delete
  fi
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

fail() {
  printf 'production database lifecycle test failed: %s\n' "$1" >&2
  for project_name in "${failure_project}" "${recovery_project}"; do
    if valid_project_name "${project_name}"; then
      compose "${project_name}" ps --all >&2 || true
      compose "${project_name}" logs --no-color --tail 80 core postgresql >&2 || true
    fi
  done
  exit 1
}

container_id() {
  local project_name="$1"
  local service_name="$2"
  compose "${project_name}" ps --all --quiet "${service_name}"
}

container_running() {
  docker inspect "$1" --format '{{.State.Running}}' 2>/dev/null | grep -qx true
}

management_status() {
  local core_container="$1"
  local path="$2"
  local response
  response="$(docker exec "${core_container}" wget --server-response --spider "http://127.0.0.1:9090${path}" 2>&1 || true)"
  sed -n 's/.*HTTP\/1\.[01] \([0-9][0-9][0-9]\).*/\1/p' <<<"${response}" | tail -n 1
}

wait_for_management_status() {
  local core_container="$1"
  local path="$2"
  local expected_status="$3"
  local timeout_seconds="$4"
  local deadline=$((SECONDS + timeout_seconds))
  local observed_status=""

  while (( SECONDS < deadline )); do
    if ! container_running "${core_container}"; then
      return 1
    fi
    observed_status="$(management_status "${core_container}" "${path}")"
    if [[ "${observed_status}" == "${expected_status}" ]]; then
      return 0
    fi
    sleep 0.1
  done
  return 1
}

wait_for_container_exit() {
  local core_container="$1"
  local timeout_seconds="$2"
  local deadline=$((SECONDS + timeout_seconds))

  while (( SECONDS < deadline )); do
    if ! container_running "${core_container}"; then
      return 0
    fi
    sleep 0.1
  done
  return 1
}

wait_for_container_health() {
  local target_container="$1"
  local expected_health="$2"
  local timeout_seconds="$3"
  local deadline=$((SECONDS + timeout_seconds))
  local observed_health=""

  while (( SECONDS < deadline )); do
    observed_health="$(docker inspect "${target_container}" --format '{{if .State.Health}}{{.State.Health.Status}}{{end}}' 2>/dev/null || true)"
    if [[ "${observed_health}" == "${expected_health}" ]]; then
      return 0
    fi
    sleep 0.2
  done
  return 1
}

assert_same_core_instance() {
  local core_container="$1"
  local expected_pid="$2"
  local expected_restarts="$3"
  local observed_pid
  local observed_restarts

  container_running "${core_container}" || fail "core container exited during the runtime outage"
  observed_pid="$(docker inspect "${core_container}" --format '{{.State.Pid}}')"
  observed_restarts="$(docker inspect "${core_container}" --format '{{.RestartCount}}')"
  [[ "${observed_pid}" == "${expected_pid}" ]] ||
    fail "core process PID changed: before=${expected_pid} after=${observed_pid}"
  [[ "${observed_restarts}" == "${expected_restarts}" ]] ||
    fail "core container restarted: before=${expected_restarts} after=${observed_restarts}"
}

umask 077
printf '%s\n' 'ticket07-disposable-password' >"${secret_file}"

cd "${repository_root}"
compose "${failure_project}" config --quiet

printf '%s FIRST-START FAILURE: build production core image\n' "$(timestamp)"
compose "${failure_project}" build core
printf '%s FIRST-START FAILURE: start core while PostgreSQL remains stopped\n' "$(timestamp)"
compose "${failure_project}" up --detach --no-deps core
failure_core="$(container_id "${failure_project}" core)"
[[ "${failure_core}" =~ ^[0-9a-f]{64}$ ]] || fail "initial-failure core container is missing"
failure_started_at="$(timestamp)"
saw_live=false
saw_not_ready=false
saw_ready=false
failure_deadline=$((SECONDS + 10))

while (( SECONDS < failure_deadline )) && container_running "${failure_core}"; do
  live_status="$(management_status "${failure_core}" /livez)"
  ready_status="$(management_status "${failure_core}" /readyz)"
  [[ "${live_status}" == 200 ]] && saw_live=true
  [[ "${ready_status}" == 503 ]] && saw_not_ready=true
  [[ "${ready_status}" == 200 ]] && saw_ready=true
  sleep 0.1
done

wait_for_container_exit "${failure_core}" 5 || fail "core did not exit after the initial PostgreSQL deadline"
failure_exit="$(docker inspect "${failure_core}" --format '{{.State.ExitCode}}')"
compose "${failure_project}" logs --no-color core >"${test_directory}/initial-failure.log"
[[ "${failure_exit}" != 0 ]] || fail "core exited zero while PostgreSQL was unavailable"
[[ "${saw_live}" == true ]] || fail "liveness never succeeded during bounded startup"
[[ "${saw_not_ready}" == true ]] || fail "readiness did not report 503 during bounded startup"
[[ "${saw_ready}" == false ]] || fail "readiness succeeded while PostgreSQL was unavailable"
grep -Fq '"reason":"startup_deadline_exceeded"' "${test_directory}/initial-failure.log" ||
  fail "bounded dependency failure is absent from core logs"
if grep -Fq '"msg":"core process ready"' "${test_directory}/initial-failure.log"; then
  fail "core logged ready while PostgreSQL was unavailable"
fi
printf '%s FIRST-START FAILURE: container=%s exit=%s live=200 readiness=503 never_ready=true started=%s\n' \
  "$(timestamp)" "${failure_core}" "${failure_exit}" "${failure_started_at}"
grep -F '"msg":"required dependency startup failed"' "${test_directory}/initial-failure.log"

printf '%s RUNTIME RECOVERY: start production PostgreSQL and core services\n' "$(timestamp)"
compose "${recovery_project}" up --detach --wait --wait-timeout 90 postgresql core
recovery_core="$(container_id "${recovery_project}" core)"
recovery_postgresql="$(container_id "${recovery_project}" postgresql)"
[[ "${recovery_core}" =~ ^[0-9a-f]{64}$ ]] || fail "runtime-recovery core container is missing"
[[ "${recovery_postgresql}" =~ ^[0-9a-f]{64}$ ]] || fail "runtime-recovery PostgreSQL container is missing"
wait_for_management_status "${recovery_core}" /livez 200 5 || fail "ready core is not live"
wait_for_management_status "${recovery_core}" /readyz 200 5 || fail "core did not become ready"
initial_core_pid="$(docker inspect "${recovery_core}" --format '{{.State.Pid}}')"
initial_restart_count="$(docker inspect "${recovery_core}" --format '{{.RestartCount}}')"
printf '%s RUNTIME RECOVERY: ready container=%s pid=%s restarts=%s readiness=200\n' \
  "$(timestamp)" "${recovery_core}" "${initial_core_pid}" "${initial_restart_count}"

printf '%s RUNTIME RECOVERY: stop PostgreSQL container=%s\n' "$(timestamp)" "${recovery_postgresql}"
compose "${recovery_project}" stop --timeout 2 postgresql >/dev/null
wait_for_management_status "${recovery_core}" /readyz 503 10 ||
  fail "readiness did not become 503 after PostgreSQL stopped"
wait_for_management_status "${recovery_core}" /livez 200 3 ||
  fail "liveness failed while PostgreSQL was unavailable"
assert_same_core_instance "${recovery_core}" "${initial_core_pid}" "${initial_restart_count}"
sleep 1
wait_for_management_status "${recovery_core}" /readyz 503 2 ||
  fail "readiness did not remain 503 during the PostgreSQL outage"
assert_same_core_instance "${recovery_core}" "${initial_core_pid}" "${initial_restart_count}"
printf '%s RUNTIME RECOVERY: live=200 readiness=503 container=%s pid=%s restarts=%s\n' \
  "$(timestamp)" "${recovery_core}" "${initial_core_pid}" "${initial_restart_count}"

printf '%s RUNTIME RECOVERY: restart PostgreSQL\n' "$(timestamp)"
compose "${recovery_project}" start postgresql >/dev/null
wait_for_container_health "${recovery_postgresql}" healthy 30 || fail "PostgreSQL did not become healthy after restart"
wait_for_management_status "${recovery_core}" /readyz 200 20 ||
  fail "the existing core process did not recover readiness"
wait_for_management_status "${recovery_core}" /livez 200 3 || fail "recovered core is not live"
assert_same_core_instance "${recovery_core}" "${initial_core_pid}" "${initial_restart_count}"
compose "${recovery_project}" logs --no-color core >"${test_directory}/runtime-recovery.log"
[[ "$(grep -Fc '"msg":"postgresql connection pool created"' "${test_directory}/runtime-recovery.log")" == 1 ]] ||
  fail "runtime recovery did not retain exactly one PostgreSQL connection pool"
grep -Fq '"msg":"required dependency unavailable"' "${test_directory}/runtime-recovery.log" ||
  fail "PostgreSQL outage is absent from core logs"
grep -Fq '"msg":"required dependency restored"' "${test_directory}/runtime-recovery.log" ||
  fail "PostgreSQL recovery is absent from core logs"
grep -Fq '"connection_pool":"existing"' "${test_directory}/runtime-recovery.log" ||
  fail "logs do not identify recovery through the existing connection pool"
printf '%s RUNTIME RECOVERY: restored live=200 readiness=200 container=%s pid=%s restarts=%s pool=existing\n' \
  "$(timestamp)" "${recovery_core}" "${initial_core_pid}" "${initial_restart_count}"
grep -F '"msg":"required dependency unavailable"' "${test_directory}/runtime-recovery.log"
grep -F '"msg":"required dependency restored"' "${test_directory}/runtime-recovery.log"

printf 'PRODUCTION_DATABASE_LIFECYCLE_OK initial_exit=%s core_container=%s core_pid=%s restart_count=%s\n' \
  "${failure_exit}" "${recovery_core}" "${initial_core_pid}" "${initial_restart_count}"
