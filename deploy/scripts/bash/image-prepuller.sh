#!/bin/bash
# image-prepuller.sh
# 节点镜像预拉取执行脚本
# 用途：运行在 DaemonSet 中，定期读取 image-manifest 清单并调用 crictl 拉取镜像

set -euo pipefail
shopt -s lastpipe

MANIFEST_FILE=${MANIFEST_FILE:-/etc/image-manifest/manifest.yaml}
CRICTL_SOCKET=${CRICTL_SOCKET:-unix:///var/run/containerd/containerd.sock}
RECONCILE_INTERVAL=${RECONCILE_INTERVAL:-1800}
BACKEND_URL=${BACKEND_URL:-}

log() {
  echo "==> $*"
}

build_auth_args() {
  local args=()
  if [[ -n "${REGISTRY_USERNAME:-}" && -n "${REGISTRY_PASSWORD:-}" ]]; then
    args+=(--creds "${REGISTRY_USERNAME}:${REGISTRY_PASSWORD}")
  fi
  printf '%s\n' "${args[@]}"
}

parse_manifest_images() {
  awk '
    BEGIN {
      registry = ""
      project = ""
      name = ""
      tag = ""
    }
    /^registry:/ {
      registry = $2
      gsub(/"/, "", registry)
      next
    }
    /^[[:space:]]*-[[:space:]]project:/ {
      project = $3
      gsub(/"/, "", project)
      next
    }
    /^[[:space:]]*name:/ {
      name = $2
      gsub(/"/, "", name)
      next
    }
    /^[[:space:]]*tag:/ {
      tag = $2
      gsub(/"/, "", tag)
      if (registry != "" && project != "" && name != "" && tag != "") {
        printf "%s/%s/%s:%s\n", registry, project, name, tag
      }
      next
    }
  ' "$MANIFEST_FILE"
}

report_status() {
  if [[ -z "$BACKEND_URL" ]]; then
    return 0
  fi
  if [[ -z "${ADMIN_TOKEN:-}" ]]; then
    return 0
  fi
  curl -fsS -X POST "${BACKEND_URL%/}/internal/image-prepull/report" \
    -H "Authorization: Bearer ${ADMIN_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{\"node_name\":\"${NODE_NAME:-unknown}\",\"image\":\"$1\",\"status\":\"$2\",\"message\":\"$3\"}" >/dev/null || true
}

pull_once() {
  if [[ ! -f "$MANIFEST_FILE" ]]; then
    log "manifest file not found: $MANIFEST_FILE"
    return 1
  fi

  local auth_args=()
  while IFS= read -r line; do
    [[ -n "$line" ]] && auth_args+=("$line")
  done < <(build_auth_args)

  local images=()
  while IFS= read -r image; do
    [[ -n "$image" ]] && images+=("$image")
  done < <(parse_manifest_images)

  if [[ ${#images[@]} -eq 0 ]]; then
    log "no image entries found in manifest"
    return 0
  fi

  log "start pre-pull for ${#images[@]} images on node ${NODE_NAME:-unknown}"
  for image in "${images[@]}"; do
    log "pulling ${image}"
    if crictl --runtime-endpoint "$CRICTL_SOCKET" pull "${auth_args[@]}" "$image"; then
      report_status "$image" "success" ""
    else
      report_status "$image" "failed" "crictl pull failed"
    fi
  done
}

while true; do
  pull_once || true
  log "sleep ${RECONCILE_INTERVAL}s before next reconcile"
  sleep "$RECONCILE_INTERVAL"
done
