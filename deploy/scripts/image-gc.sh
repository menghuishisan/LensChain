#!/bin/bash
# image-gc.sh
# 镜像版本清理脚本
# 用途：调用 OCI Registry API 清理超出保留数量的旧标签

set -euo pipefail

REGISTRY_URL=${REGISTRY_URL:?"REGISTRY_URL is required"}
KEEP_VERSIONS=${KEEP_VERSIONS:-3}

log() {
  echo "==> $*"
}

registry_curl() {
  local method=${1:-GET}
  local url=${2:?url is required}
  shift 2 || true
  local args=(-fsS -X "$method")
  if [[ -n "${REGISTRY_USERNAME:-}" && -n "${REGISTRY_PASSWORD:-}" ]]; then
    args+=(-u "${REGISTRY_USERNAME}:${REGISTRY_PASSWORD}")
  fi
  curl "${args[@]}" "$url" "$@"
}

list_repositories() {
  registry_curl GET "${REGISTRY_URL%/}/v2/_catalog?n=1000" | jq -r '.repositories[]?'
}

list_tags() {
  local repo=$1
  registry_curl GET "${REGISTRY_URL%/}/v2/${repo}/tags/list" | jq -r '.tags[]?' | sort -V
}

manifest_digest() {
  local repo=$1
  local tag=$2
  local headers
  headers=$(mktemp)
  registry_curl GET "${REGISTRY_URL%/}/v2/${repo}/manifests/${tag}" \
    -H "Accept: application/vnd.docker.distribution.manifest.v2+json" \
    -D "$headers" -o /dev/null
  awk -F': ' 'tolower($1)=="docker-content-digest" { gsub("\r","",$2); print $2 }' "$headers"
  rm -f "$headers"
}

delete_manifest() {
  local repo=$1
  local digest=$2
  registry_curl DELETE "${REGISTRY_URL%/}/v2/${repo}/manifests/${digest}" >/dev/null
}

while IFS= read -r repo; do
  [[ -z "$repo" ]] && continue
  mapfile -t tags < <(list_tags "$repo")
  if (( ${#tags[@]} <= KEEP_VERSIONS )); then
    continue
  fi

  remove_count=$(( ${#tags[@]} - KEEP_VERSIONS ))
  for ((i=0; i<remove_count; i++)); do
    tag=${tags[$i]}
    log "deleting ${repo}:${tag}"
    digest=$(manifest_digest "$repo" "$tag")
    [[ -n "$digest" ]] && delete_manifest "$repo" "$digest"
  done
done < <(list_repositories)
