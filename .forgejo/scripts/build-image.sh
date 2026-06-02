#!/usr/bin/env bash
set -euo pipefail

test -n "${IMAGE:?IMAGE is required}"
test -n "${ARCH:?ARCH is required}"
test -n "${BUILDKIT_HOST:?BUILDKIT_HOST is required}"
test -n "${CONTEXT_DIR:?CONTEXT_DIR is required}"
test -n "${DOCKERFILE:?DOCKERFILE is required}"
test -n "${REGISTRY_USERNAME:?REGISTRY_USERNAME is required}"
test -n "${REGISTRY_PASSWORD:?REGISTRY_PASSWORD is required}"
test -n "${GITHUB_SHA:?GITHUB_SHA is required}"

install_buildkit_client() {
  local version="${BUILDKIT_VERSION:-v0.24.0}"
  local arch_name
  local tools_dir="$PWD/.forgejo/tools"
  local archive="${tools_dir}/buildkit.tar.gz"

  case "$(uname -m)" in
    x86_64|amd64) arch_name=amd64 ;;
    aarch64|arm64) arch_name=arm64 ;;
    *) echo "unsupported BuildKit architecture: $(uname -m)" >&2; exit 1 ;;
  esac

  mkdir -p "${tools_dir}"
  curl -fsSL "https://github.com/moby/buildkit/releases/download/${version}/buildkit-${version}.linux-${arch_name}.tar.gz" -o "${archive}"
  tar -xzf "${archive}" -C "${tools_dir}"
  export PATH="${tools_dir}/bin:${PATH}"
}

if ! command -v buildctl >/dev/null 2>&1; then
  install_buildkit_client
fi

registry_host="${IMAGE%%/*}"
mkdir -p .forgejo/buildkit-auth
auth="$(printf '%s:%s' "${REGISTRY_USERNAME}" "${REGISTRY_PASSWORD}" | base64 | tr -d '\n')"
printf '{"auths":{"%s":{"auth":"%s"}}}\n' "${registry_host}" "${auth}" > .forgejo/buildkit-auth/config.json
export DOCKER_CONFIG="$PWD/.forgejo/buildkit-auth"

buildctl --addr "${BUILDKIT_HOST}" build \
  --frontend dockerfile.v0 \
  --local "context=${CONTEXT_DIR}" \
  --local "dockerfile=${CONTEXT_DIR}" \
  --opt "filename=${DOCKERFILE}" \
  --opt "platform=linux/${ARCH}" \
  --import-cache "type=registry,ref=${IMAGE}-buildcache:buildkit-${ARCH}" \
  --export-cache "type=registry,ref=${IMAGE}-buildcache:buildkit-${ARCH},mode=max,oci-mediatypes=true" \
  --output "type=image,name=${IMAGE}:${GITHUB_SHA}-${ARCH},push=true,oci-mediatypes=true"
