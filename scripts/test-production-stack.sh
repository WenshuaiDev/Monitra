#!/bin/sh
set -eu

repository_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
test_directory=$(mktemp -d "${TMPDIR:-/tmp}/monitra-production-test.XXXXXX")
project_name="monitra-ticket05-$$"
secret_file="$test_directory/postgresql-password"
headers_file="$test_directory/headers"
root_document="$test_directory/root.html"
deep_link_document="$test_directory/deep-link.html"

cleanup() {
	MONITRA_RELEASE_IDENTITY=ticket05-test \
		MONITRA_POSTGRES_PASSWORD_FILE="$secret_file" \
		MONITRA_WEB_PORT=0 \
		docker compose --project-name "$project_name" --file "$repository_root/compose.production.yaml" \
		down --volumes --remove-orphans >/dev/null 2>&1 || true
	rm -rf "$test_directory"
}
trap cleanup EXIT INT TERM

umask 077
printf '%s\n' 'ticket05-disposable-password' >"$secret_file"

compose() {
	MONITRA_RELEASE_IDENTITY=ticket05-test \
		MONITRA_POSTGRES_PASSWORD_FILE="$secret_file" \
		MONITRA_WEB_PORT=0 \
		docker compose --project-name "$project_name" --file "$repository_root/compose.production.yaml" "$@"
}

fail() {
	printf 'production stack test failed: %s\n' "$1" >&2
	exit 1
}

assert_container_security() {
	service_name=$1
	container_id=$(compose ps --all --quiet "$service_name")
	test -n "$container_id" || fail "$service_name container is missing"

	container_user=$(docker inspect "$container_id" --format '{{.Config.User}}')
	case "$container_user" in
		"" | 0 | 0:*) fail "$service_name runs as root" ;;
	esac
	test "$(docker inspect "$container_id" --format '{{.HostConfig.ReadonlyRootfs}}')" = true ||
		fail "$service_name root filesystem is writable"
	test "$(docker inspect "$container_id" --format '{{json .HostConfig.CapDrop}}')" = '["ALL"]' ||
		fail "$service_name does not drop every Linux capability"
	case "$(docker inspect "$container_id" --format '{{json .HostConfig.SecurityOpt}}')" in
		*no-new-privileges:true*) ;;
		*) fail "$service_name does not enable no-new-privileges" ;;
	esac
	test "$(docker inspect "$container_id" --format '{{.HostConfig.Privileged}}')" = false ||
		fail "$service_name is privileged"
	test "$(docker inspect "$container_id" --format '{{.HostConfig.NetworkMode}}')" != host ||
		fail "$service_name uses the host network"
	test "$(docker inspect "$container_id" --format '{{.HostConfig.PidMode}}')" != host ||
		fail "$service_name uses the host PID namespace"
	case "$(docker inspect "$container_id" --format '{{range .Mounts}}{{.Destination}} {{end}}')" in
		*/var/run/docker.sock*) fail "$service_name mounts the Docker socket" ;;
	esac
}

cd "$repository_root"
compose config --quiet
compose build
compose up --detach --wait --wait-timeout 90

web_address=$(compose port caddy 8080)
web_url="http://$web_address"

curl --fail --silent --show-error --dump-header "$headers_file" "$web_url/" --output "$root_document"
case "$(tr '[:upper:]' '[:lower:]' <"$headers_file")" in
	*"cache-control: no-cache"*) ;;
	*) fail "HTML is not revalidated" ;;
esac
curl --fail --silent --show-error "$web_url/app/deep-link" --output "$deep_link_document"
cmp "$root_document" "$deep_link_document" >/dev/null || fail "SPA deep link did not return the entry document"

runtime_config=$(curl --fail --silent --show-error --dump-header "$headers_file" "$web_url/runtime-config.json")
test "$runtime_config" = '{"expected_api_major":1,"expected_release_identity":"ticket05-test"}' ||
	fail "runtime config is not the strict public projection"
case "$(tr '[:upper:]' '[:lower:]' <"$headers_file")" in
	*"cache-control: no-store"*) ;;
	*) fail "runtime config is cacheable" ;;
esac

handshake=$(curl --fail --silent --show-error "$web_url/api/v1/startup-handshake")
HANDSHAKE_DOCUMENT="$handshake" node -e '
const response = JSON.parse(process.env.HANDSHAKE_DOCUMENT);
if (
  response.code !== "FOUNDATION_STARTUP_READY" ||
  response.data?.release_identity !== "ticket05-test" ||
  response.data?.api_major !== 1 ||
  typeof response.request_id !== "string" ||
  response.request_id.length === 0
) process.exit(1);
' || fail "same-origin startup handshake is incomplete"

management_status=$(curl --silent --output /dev/null --write-out '%{http_code}' "$web_url/readyz")
test "$management_status" = 404 || fail "Caddy exposes the Management Listener"

for service_name in postgresql runtime-config core caddy; do
	assert_container_security "$service_name"
done

core_ports=$(docker inspect "$(compose ps --quiet core)" --format '{{json .NetworkSettings.Ports}}')
postgresql_ports=$(docker inspect "$(compose ps --quiet postgresql)" --format '{{json .NetworkSettings.Ports}}')
test "$core_ports" = '{}' || fail "core publishes a host port"
case "$postgresql_ports" in
	*'"HostPort"'*) fail "PostgreSQL publishes a host port" ;;
esac

compose stop core >/dev/null
curl --fail --silent --show-error "$web_url/" >/dev/null || fail "SPA became unavailable with the core stopped"
backend_status=$(curl --silent --output /dev/null --write-out '%{http_code}' "$web_url/api/v1/startup-handshake")
test "$backend_status" = 502 || fail "Caddy did not report the unavailable core"

core_digest=$(docker image inspect monitra-core:ticket05-test --format '{{.Id}}')
web_digest=$(docker image inspect monitra-web:ticket05-test --format '{{.Id}}')
printf 'PRODUCTION_STACK_OK core=%s web=%s\n' "$core_digest" "$web_digest"
