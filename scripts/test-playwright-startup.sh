#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
temporary_parent=$(CDPATH= cd -- "${TMPDIR:-/tmp}" && pwd -P)
test_directory=$(mktemp -d "$temporary_parent/monitra-playwright.XXXXXX")
project_name="monitra-ticket06-$$"
release_identity="ticket06-test"
secret_file="$test_directory/postgresql-password"
runtime_config_directory="$test_directory/runtime-config"
runtime_config_file="$runtime_config_directory/runtime-config.json"

export COMPOSE_PROJECT_NAME="$project_name"
export COMPOSE_FILE="$repository_root/compose.production.yaml:$repository_root/e2e/compose.playwright.yaml"
export MONITRA_RELEASE_IDENTITY="$release_identity"
export MONITRA_POSTGRES_PASSWORD_FILE="$secret_file"
export MONITRA_WEB_PORT=0
export MONITRA_E2E_RUNTIME_CONFIG_DIR="$runtime_config_directory"
export MONITRA_E2E_RELEASE_IDENTITY="$release_identity"
export MONITRA_E2E_RUNTIME_CONFIG_FILE="$runtime_config_file"

compose() {
	docker compose "$@"
}

cleanup() {
	run_status=$?
	trap - EXIT INT TERM
	cleanup_status=0

	if ! compose down --volumes --remove-orphans >/dev/null 2>&1; then
		printf '%s\n' 'Playwright production stack cleanup failed' >&2
		cleanup_status=1
	fi
	if remaining=$(compose ps --all --quiet 2>/dev/null) && [ -n "$remaining" ]; then
		printf 'Playwright production stack left containers: %s\n' "$remaining" >&2
		cleanup_status=1
	fi

	case "$test_directory" in
		"$temporary_parent"/monitra-playwright.*)
			rm -rf -- "$test_directory"
			;;
		*)
			printf 'refusing to remove unexpected test directory: %s\n' "$test_directory" >&2
			cleanup_status=1
			;;
	esac

	if [ "$cleanup_status" -eq 0 ]; then
		printf 'PLAYWRIGHT_STACK_CLEAN project=%s\n' "$project_name"
	fi
	if [ "$run_status" -ne 0 ]; then
		exit "$run_status"
	fi
	exit "$cleanup_status"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

umask 077
mkdir "$runtime_config_directory"
printf '%s\n' 'ticket06-disposable-password' >"$secret_file"
printf '%s\n' \
	'{"expected_api_major":1,"expected_release_identity":"ticket06-test"}' \
	>"$runtime_config_file"

cd "$repository_root"
compose config --quiet
compose build
compose up --detach --wait --wait-timeout 90

web_address=$(compose port caddy 8080)
web_url="http://$web_address"
export MONITRA_E2E_WEB_URL="$web_url"

pnpm exec playwright test "$@" || test_status=$?

compose ps --all
if [ "${test_status:-0}" -ne 0 ]; then
	compose logs --timestamps --tail 100 runtime-config core caddy >&2
	exit "$test_status"
fi
