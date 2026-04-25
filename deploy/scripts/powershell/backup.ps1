# backup.ps1
# 手动触发数据库备份

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

param(
    [Parameter(Mandatory = $true, Position = 0)]
    [string]$BackupId
)

$Namespace = if ($env:NAMESPACE) { $env:NAMESPACE } else { "lenschain-system" }
$jobName = "db-backup-manual-$BackupId"

Write-Host "==> Creating backup job: $jobName"
& kubectl -n $Namespace create job --from=cronjob/db-backup $jobName
if ($LASTEXITCODE -ne 0) {
    throw "创建备份任务失败"
}

Write-Host "==> Waiting for backup job to complete"
& kubectl -n $Namespace wait --for=condition=complete "job/$jobName" --timeout=600s
if ($LASTEXITCODE -ne 0) {
    throw "等待备份任务完成失败"
}

Write-Host "==> Backup complete"
& kubectl -n $Namespace logs "job/$jobName"

