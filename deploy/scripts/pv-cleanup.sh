#!/bin/bash
# pv-cleanup.sh
# 实验卷延迟清理脚本
# 用途：删除超过保留窗口的 Released/Failed 状态实验相关 PV

set -euo pipefail

RETENTION_HOURS=${RETENTION_HOURS:-24}
NOW_TS=$(date +%s)

log() {
  echo "==> $*"
}

is_target_pv() {
  local claim_namespace=$1
  [[ "$claim_namespace" == lenschain-exp* || "$claim_namespace" == exp-* || "$claim_namespace" == ctf-* ]]
}

kubectl get pv -o json | jq -c '.items[]' | while read -r item; do
  phase=$(jq -r '.status.phase // ""' <<<"$item")
  claim_namespace=$(jq -r '.spec.claimRef.namespace // ""' <<<"$item")
  created_at=$(jq -r '.metadata.creationTimestamp // ""' <<<"$item")
  pv_name=$(jq -r '.metadata.name' <<<"$item")

  [[ -z "$claim_namespace" ]] && continue
  is_target_pv "$claim_namespace" || continue
  [[ "$phase" != "Released" && "$phase" != "Failed" ]] && continue

  created_ts=$(date -d "$created_at" +%s 2>/dev/null || echo 0)
  age_hours=$(( (NOW_TS - created_ts) / 3600 ))
  if (( age_hours < RETENTION_HOURS )); then
    continue
  fi

  log "deleting PV ${pv_name} (namespace=${claim_namespace}, age=${age_hours}h, phase=${phase})"
  kubectl delete pv "$pv_name"
done
