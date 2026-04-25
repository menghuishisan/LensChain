#!/bin/bash
# preload-images.sh
# 手动触发镜像预拉取 DaemonSet 重新拉取（运维调试用）
# 用途：镜像清单更新后，触发所有 K8s 节点重新拉取

set -euo pipefail

NAMESPACE=${NAMESPACE:-lenschain-system}
TIMEOUT=${TIMEOUT:-30m}

echo "==> Restarting image-prepuller DaemonSet"
kubectl -n "$NAMESPACE" rollout restart daemonset/image-prepuller

echo "==> Waiting for rollout to complete (timeout: $TIMEOUT)"
kubectl -n "$NAMESPACE" rollout status daemonset/image-prepuller --timeout="$TIMEOUT"

echo "==> Image preload trigger complete"
kubectl -n "$NAMESPACE" get pods -l app.kubernetes.io/name=image-prepuller -o wide
