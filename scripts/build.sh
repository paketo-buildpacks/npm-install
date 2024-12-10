#!/usr/bin/env bash

set -eu
set -o pipefail

readonly PROGDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly BUILDPACKDIR="$(cd "${PROGDIR}/.." && pwd)"
readonly AVAILABLE_TARGETS=("linux/amd64" "linux/arm64")

function main() {
  while [[ "${#}" != 0 ]]; do
    case "${1}" in
      --help|-h)
        shift 1
        usage
        exit 0
        ;;

      "")
        # skip if the argument is empty
        shift 1
        ;;

      *)
        util::print::error "unknown argument \"${1}\""
    esac
  done

  mkdir -p "${BUILDPACKDIR}/bin"

  run::build
  cmd::build
}

function usage() {
  cat <<-USAGE
build.sh [OPTIONS]

Builds the buildpack executables.

OPTIONS
  --help  -h  prints the command usage
USAGE
}

function run::build() {
  if [[ -f "${BUILDPACKDIR}/run/main.go" ]]; then
    pushd "${BUILDPACKDIR}" > /dev/null || return
      for target in "${AVAILABLE_TARGETS[@]}"; do
        platform=$(echo "${target}" | cut -d '/' -f1)
        arch=$(echo "${target}" | cut -d'/' -f2)

        echo "Building run... for platform: ${platform} and arch: ${arch}"

        GOOS=$platform \
        GOARCH=$arch \
        CGO_ENABLED=0 \
          go build \
            -ldflags="-s -w" \
            -o "${platform}/${arch}/bin/run" \
              "${BUILDPACKDIR}/run"

          echo "Success!"

          names=("detect")

          if [ -f "${BUILDPACKDIR}/extension.toml" ]; then
            names+=("generate")
          else
            names+=("build")
          fi

        for name in "${names[@]}"; do
          printf "%s" "Linking ${name}... "

          ln -fs "run" "${platform}/${arch}/bin/${name}"

          echo "Success!"
        done
      done

    popd > /dev/null || return
  fi
}

function cmd::build() {
  if [[ -d "${BUILDPACKDIR}/cmd" ]]; then
    local name
    for src in "${BUILDPACKDIR}"/cmd/*; do
      name="$(basename "${src}")"
     for target in "${AVAILABLE_TARGETS[@]}"; do
        platform=$(echo "${target}" | cut -d '/' -f1)
        arch=$(echo "${target}" | cut -d'/' -f2)

        if [[ -f "${src}/main.go" ]]; then
          echo "Building ${name}... for platform: ${platform} and arch: ${arch}"

          GOOS=$platform \
          GOARCH=$arch \
          CGO_ENABLED=0 \
            go build \
              -ldflags="-s -w" \
              -o "${BUILDPACKDIR}/${platform}/${arch}/bin/${name}" \
                "${src}/main.go"

          echo "Success!"
        else
          printf "%s" "Skipping ${name}... "
        fi
      done
    done
  fi
}

main "${@:-}"
