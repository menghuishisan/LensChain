# image-prepuller.ps1
# 节点镜像预拉取执行脚本

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$ManifestFile = if ($env:MANIFEST_FILE) { $env:MANIFEST_FILE } else { "/etc/image-manifest/manifest.yaml" }
$CrictlSocket = if ($env:CRICTL_SOCKET) { $env:CRICTL_SOCKET } else { "unix:///var/run/containerd/containerd.sock" }
$ReconcileInterval = if ($env:RECONCILE_INTERVAL) { [int]$env:RECONCILE_INTERVAL } else { 1800 }
$BackendUrl = if ($env:BACKEND_URL) { $env:BACKEND_URL.TrimEnd('/') } else { "" }
$NodeName = if ($env:NODE_NAME) { $env:NODE_NAME } else { "unknown" }

function Get-ManifestImages {
    $registry = ""
    $project = ""
    $name = ""

    foreach ($line in Get-Content $ManifestFile) {
        if ($line -match '^registry:\s*(.+)$') {
            $registry = $Matches[1].Trim().Trim('"')
            continue
        }
        if ($line -match '^\s*-\s*project:\s*(.+)$') {
            $project = $Matches[1].Trim().Trim('"')
            continue
        }
        if ($line -match '^\s*name:\s*(.+)$') {
            $name = $Matches[1].Trim().Trim('"')
            continue
        }
        if ($line -match '^\s*tag:\s*(.+)$') {
            $tag = $Matches[1].Trim().Trim('"')
            if ($registry -and $project -and $name -and $tag) {
                "$registry/$project/$name`:$tag"
            }
        }
    }
}

function Report-Status {
    param(
        [string]$Image,
        [string]$Status,
        [string]$Message
    )

    if (-not $BackendUrl -or -not $env:ADMIN_TOKEN) {
        return
    }

    $body = @{
        node_name = $NodeName
        image     = $Image
        status    = $Status
        message   = $Message
    } | ConvertTo-Json -Compress

    try {
        Invoke-WebRequest -Method Post -Uri "$BackendUrl/internal/image-prepull/report" -Headers @{
            Authorization = "Bearer $($env:ADMIN_TOKEN)"
            "Content-Type" = "application/json"
        } -Body $body | Out-Null
    }
    catch {
    }
}

while ($true) {
    if (-not (Test-Path $ManifestFile)) {
        Write-Host "==> manifest file not found: $ManifestFile"
        Start-Sleep -Seconds $ReconcileInterval
        continue
    }

    $images = @(Get-ManifestImages)
    if ($images.Count -eq 0) {
        Write-Host "==> no image entries found in manifest"
        Start-Sleep -Seconds $ReconcileInterval
        continue
    }

    Write-Host "==> start pre-pull for $($images.Count) images on node $NodeName"
    foreach ($image in $images) {
        Write-Host "==> pulling $image"
        $args = @("--runtime-endpoint", $CrictlSocket, "pull")
        if ($env:REGISTRY_USERNAME -and $env:REGISTRY_PASSWORD) {
            $args += @("--creds", "$($env:REGISTRY_USERNAME):$($env:REGISTRY_PASSWORD)")
        }
        $args += $image

        & crictl @args
        if ($LASTEXITCODE -eq 0) {
            Report-Status -Image $image -Status "success" -Message ""
        }
        else {
            Report-Status -Image $image -Status "failed" -Message "crictl pull failed"
        }
    }

    Write-Host "==> sleep ${ReconcileInterval}s before next reconcile"
    Start-Sleep -Seconds $ReconcileInterval
}

