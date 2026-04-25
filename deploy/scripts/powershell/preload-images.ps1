# preload-images.ps1
# 手动刷新镜像预拉取清单并触发 DaemonSet 重新拉取

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$Namespace = if ($env:NAMESPACE) { $env:NAMESPACE } else { "lenschain-system" }
$Timeout = if ($env:TIMEOUT) { $env:TIMEOUT } else { "30m" }
$Registry = if ($env:REGISTRY) { $env:REGISTRY } else { "registry.lianjing.com" }
$ConfigMapName = if ($env:CONFIGMAP_NAME) { $env:CONFIGMAP_NAME } else { "image-manifest" }
$PhaseFilter = if ($env:PHASE_FILTER) { $env:PHASE_FILTER } else { "" }

$repoRoot = Split-Path -Parent (Split-Path -Parent (Split-Path -Parent $PSScriptRoot))
$deployDir = Join-Path $repoRoot "deploy"
$tmpManifest = Join-Path ([System.IO.Path]::GetTempPath()) ("image-manifest-" + [System.Guid]::NewGuid().ToString("N") + ".yaml")
$count = 0

try {
    Set-Content -Path $tmpManifest -Value @("registry: $Registry", "images:")

    $manifests = Get-ChildItem -Path (Join-Path $deployDir "images") -Filter "manifest.yaml" -Recurse -File
    if ($manifests.Count -eq 0) {
        throw "no manifest.yaml files found under $deployDir/images/"
    }

    foreach ($manifest in $manifests) {
        $content = Get-Content $manifest.FullName
        $projectLine = $content | Where-Object { $_ -match '^registry_project:' } | Select-Object -First 1
        $nameLine = $content | Where-Object { $_ -match '^name:' } | Select-Object -First 1
        $phaseLine = $content | Where-Object { $_ -match '^prepull_phase:' } | Select-Object -First 1
        if (-not $projectLine -or -not $nameLine) {
            continue
        }

        $project = ($projectLine -split ':', 2)[1].Trim().Trim('"')
        $name = ($nameLine -split ':', 2)[1].Trim().Trim('"')
        $phase = ""
        if ($phaseLine) {
            $phase = ($phaseLine -split ':', 2)[1].Trim().Trim('"')
        }
        if ($PhaseFilter -and $phase -ne $PhaseFilter) {
            continue
        }

        $tagLines = $content | Where-Object { $_ -match '^\s{4}tag:' }
        foreach ($tagLine in $tagLines) {
            $tag = ($tagLine -split ':', 2)[1].Trim().Trim('"')
            if (-not $tag) {
                continue
            }

            Add-Content -Path $tmpManifest -Value @(
                "  - project: $project"
                "    name: $name"
                "    tag: $tag"
            )
            $count++
        }
    }

    Write-Host "==> Prepared $count image entries"
    if ($count -eq 0) {
        throw "preload manifest is empty"
    }

    Write-Host "==> Applying ConfigMap $ConfigMapName in namespace $Namespace"
    & kubectl create configmap $ConfigMapName --namespace $Namespace --from-file "manifest.yaml=$tmpManifest" --dry-run=client -o yaml | & kubectl apply -f -
    if ($LASTEXITCODE -ne 0) {
        throw "应用 ConfigMap 失败"
    }

    Write-Host "==> Restarting image-prepuller DaemonSet"
    & kubectl -n $Namespace rollout restart daemonset/image-prepuller
    if ($LASTEXITCODE -ne 0) {
        throw "重启 DaemonSet 失败"
    }

    Write-Host "==> Waiting for rollout to complete (timeout: $Timeout)"
    & kubectl -n $Namespace rollout status daemonset/image-prepuller --timeout=$Timeout
    if ($LASTEXITCODE -ne 0) {
        throw "等待 DaemonSet 滚动完成失败"
    }

    Write-Host "==> Image preload trigger complete"
    & kubectl -n $Namespace get pods -l app.kubernetes.io/name=image-prepuller -o wide
}
finally {
    if (Test-Path $tmpManifest) {
        Remove-Item -LiteralPath $tmpManifest -Force
    }
}

