# pv-cleanup.ps1
# 实验卷延迟清理脚本

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RetentionHours = if ($env:RETENTION_HOURS) { [int]$env:RETENTION_HOURS } else { 24 }
$now = Get-Date

$pvs = (& kubectl get pv -o json | Out-String | ConvertFrom-Json).items
foreach ($pv in $pvs) {
    $phase = $pv.status.phase
    $claimNamespace = $pv.spec.claimRef.namespace
    $createdAt = $pv.metadata.creationTimestamp
    $pvName = $pv.metadata.name

    if (-not $claimNamespace) {
        continue
    }
    if ($claimNamespace -notlike "lenschain-exp*" -and $claimNamespace -notlike "exp-*" -and $claimNamespace -notlike "ctf-*") {
        continue
    }
    if ($phase -ne "Released" -and $phase -ne "Failed") {
        continue
    }

    $created = Get-Date $createdAt
    $ageHours = [int](($now - $created).TotalHours)
    if ($ageHours -lt $RetentionHours) {
        continue
    }

    Write-Host "==> deleting PV $pvName (namespace=$claimNamespace, age=${ageHours}h, phase=$phase)"
    & kubectl delete pv $pvName
    if ($LASTEXITCODE -ne 0) {
        throw "删除 PV 失败: $pvName"
    }
}

