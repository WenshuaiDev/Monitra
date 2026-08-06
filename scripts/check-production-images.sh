#!/usr/bin/env bash

set -euo pipefail

if (( $# == 0 )); then
  printf 'usage: %s <production-image>...\n' "$0" >&2
  exit 2
fi

repository_root="$(cd "$(dirname "$0")/.." && pwd -P)"
checker="${repository_root}/scripts/check-production-artifact.sh"
temporary_parent="$(cd "${TMPDIR:-/tmp}" && pwd -P)"
inspection_directory="$(mktemp -d "${temporary_parent}/monitra-image-inspection.XXXXXX")"
created_containers=()

cleanup() {
  local container_id
  for container_id in "${created_containers[@]}"; do
    if [[ "${container_id}" =~ ^[0-9a-f]{64}$ ]]; then
      docker rm --force "${container_id}" >/dev/null 2>&1 || true
    fi
  done
  if [[ -d "${inspection_directory}" && "${inspection_directory}" == "${temporary_parent}"/monitra-image-inspection.* ]]; then
    find "${inspection_directory}" -xdev -type f -delete
    rmdir "${inspection_directory}"
  fi
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

image_index=0
for image in "$@"; do
  image_index=$((image_index + 1))
  label="production-image-${image_index}"
  archive="${inspection_directory}/${label}.tar"
  manifest="${inspection_directory}/${label}.manifest"
  packages="${inspection_directory}/${label}.packages"

  image_id="$(docker image inspect "${image}" --format '{{.Id}}')"
  image_command="$(docker image inspect "${image}" --format '{{json .Config.Entrypoint}} {{json .Config.Cmd}}')"
  case "${image_command}" in
    *playwright* | *chromium* | *google-chrome* | *firefox* | *webkit* | *vite* | *webpack* | *vite-node*)
      printf 'PRODUCTION_IMAGE_REJECTED image=%s reason=test-or-development-command command=%s\n' \
        "${image}" "${image_command}" >&2
      exit 1
      ;;
  esac

  container_id="$(docker create "${image}")"
  if [[ ! "${container_id}" =~ ^[0-9a-f]{64}$ ]]; then
    printf 'Docker returned an unexpected container ID for %s: %s\n' "${image}" "${container_id}" >&2
    exit 1
  fi
  created_containers+=("${container_id}")
  docker export --output "${archive}" "${container_id}"
  docker rm "${container_id}" >/dev/null

  tar -tf "${archive}" | sed 's#^\./##' | LC_ALL=C sort -u >"${manifest}"
  : >"${packages}"
  apk_database="$(sed -n '\#^lib/apk/db/installed$#p' "${manifest}" | head -n 1)"
  if [[ -n "${apk_database}" ]]; then
    tar -xOf "${archive}" "${apk_database}" |
      awk -F: '$1 == "P" { print $2 }' |
      LC_ALL=C sort -u >"${packages}"
    sed 's#^#dependencies/apk/#' "${packages}" >>"${manifest}"
  fi

  "${checker}" "${manifest}" "${label}"
  file_count="$(awk '!/^dependencies\// { count++ } END { print count + 0 }' "${manifest}")"
  package_count="$(awk 'END { print NR + 0 }' "${packages}")"
  package_list="$(paste -sd, "${packages}")"
  printf 'PRODUCTION_IMAGE_INVENTORY image=%s id=%s files=%s apk_packages=%s packages=%s\n' \
    "${image}" "${image_id}" "${file_count}" "${package_count}" "${package_list:-none}"
done

printf 'PRODUCTION_IMAGES_CLEAN images=%s\n' "${image_index}"
