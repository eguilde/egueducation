#!/usr/bin/env bash
set -euo pipefail

test -n "${IMAGE:?IMAGE is required}"
test -n "${GITHUB_SHA:?GITHUB_SHA is required}"
test -n "${REGISTRY_USERNAME:?REGISTRY_USERNAME is required}"
test -n "${REGISTRY_PASSWORD:?REGISTRY_PASSWORD is required}"
test -n "${GITOPS_USERNAME:?GITOPS_USERNAME is required}"
test -n "${GITOPS_TOKEN:?GITOPS_TOKEN is required}"
test -n "${GITOPS_FILE:?GITOPS_FILE is required}"
test -n "${SERVICE_NAME:?SERVICE_NAME is required}"

install_crane() {
  local version="${CRANE_VERSION:-v0.20.6}"
  local arch_name
  local tools_dir="$PWD/.forgejo/tools"
  local archive="${tools_dir}/crane.tar.gz"

  case "$(uname -m)" in
    x86_64|amd64) arch_name=x86_64 ;;
    aarch64|arm64) arch_name=arm64 ;;
    *) echo "unsupported crane architecture: $(uname -m)" >&2; exit 1 ;;
  esac

  mkdir -p "${tools_dir}/bin"
  curl -fsSL "https://github.com/google/go-containerregistry/releases/download/${version}/go-containerregistry_Linux_${arch_name}.tar.gz" -o "${archive}"
  tar -xzf "${archive}" -C "${tools_dir}/bin" crane
  export PATH="${tools_dir}/bin:${PATH}"
}

retry_cmd() {
  local attempt=1
  while true; do
    "$@" && return 0
    if [ "$attempt" -ge 5 ]; then
      return 1
    fi
    sleep "$((attempt * 5))"
    attempt="$((attempt + 1))"
  done
}

if ! command -v crane >/dev/null 2>&1; then
  install_crane
fi

registry_host="${IMAGE%%/*}"
crane auth login "${registry_host}" --username "${REGISTRY_USERNAME}" --password "${REGISTRY_PASSWORD}"
retry_cmd crane index append \
  --tag "${IMAGE}:${GITHUB_SHA}" \
  --manifest "${IMAGE}:${GITHUB_SHA}-amd64" \
  --manifest "${IMAGE}:${GITHUB_SHA}-arm64"
retry_cmd crane index append \
  --tag "${IMAGE}:latest" \
  --manifest "${IMAGE}:${GITHUB_SHA}-amd64" \
  --manifest "${IMAGE}:${GITHUB_SHA}-arm64"

auth_header="$(printf '%s:%s' "${GITOPS_USERNAME}" "${GITOPS_TOKEN}" | base64 | tr -d '\n')"
rm -rf gitops
retry_cmd git -c http.extraHeader="Authorization: Basic ${auth_header}" \
  clone --depth 1 --single-branch --branch main "https://forge.eguilde.cloud/eguilde/gitops.git" gitops
cd gitops
git config user.name "forgejo-actions"
git config user.email "forgejo-actions@eguilde.cloud"

legacy_image="${IMAGE/registry.eguilde.cloud/forge.eguilde.cloud}"
awk -v legacy_image="${legacy_image}" -v target_image="${IMAGE}" -v tag="${GITHUB_SHA}" '
  $0 ~ "^- name: " legacy_image "$" || $0 ~ "^[[:space:]]+- name: " legacy_image "$" || $0 ~ "^- name: " target_image "$" || $0 ~ "^[[:space:]]+- name: " target_image "$" {
    print "  - name: " target_image
    print "    newName: " target_image
    print "    newTag: " tag
    in_image=1
    next
  }
  in_image && $0 ~ "^[[:space:]]+newName:" { next }
  in_image && $0 ~ "^[[:space:]]+newTag:" { next }
  in_image && $0 ~ "^[[:space:]]*- name:" { in_image=0; print; next }
  { print }
' "${GITOPS_FILE}" > "${GITOPS_FILE}.tmp"
mv "${GITOPS_FILE}.tmp" "${GITOPS_FILE}"

git add "${GITOPS_FILE}"
git diff --cached --quiet && exit 0
git commit -m "chore(egueducation-${SERVICE_NAME}): deploy ${GITHUB_SHA}"
retry_cmd git -c http.extraHeader="Authorization: Basic ${auth_header}" pull --rebase origin main
retry_cmd git -c http.extraHeader="Authorization: Basic ${auth_header}" push origin HEAD:main
