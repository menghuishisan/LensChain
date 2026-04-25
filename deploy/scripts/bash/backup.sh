#!/bin/bash
# backup.sh
# 手动触发数据库备份（被模块08 数据备份管理 API 调用）
# 用途：创建一个一次性 K8s Job 执行 pg_dump 并上传到 MinIO

set -euo pipefail

BACKUP_ID=${1:?"Usage: backup.sh <backup_id>"}
NAMESPACE=${NAMESPACE:-lenschain-system}

echo "==> Creating backup job: db-backup-manual-$BACKUP_ID"
kubectl -n "$NAMESPACE" create job \
  --from=cronjob/db-backup \
  "db-backup-manual-$BACKUP_ID"

echo "==> Waiting for backup job to complete"
kubectl -n "$NAMESPACE" wait --for=condition=complete \
  "job/db-backup-manual-$BACKUP_ID" --timeout=600s

echo "==> Backup complete"
kubectl -n "$NAMESPACE" logs "job/db-backup-manual-$BACKUP_ID"
