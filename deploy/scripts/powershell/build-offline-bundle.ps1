# build-offline-bundle.ps1
# 构建离线镜像包

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$Registry = if ($env:REGISTRY) { $env:REGISTRY } else { "registry.lianjing.com" }
$Version = if ($env:VERSION) { $env:VERSION } else { throw "VERSION is required" }
$OutputDir = if ($env:OUTPUT_DIR) { $env:OUTPUT_DIR } else { "." }
$BundleName = "lenschain-phase1-$Version"
$repoRoot = Split-Path -Parent (Split-Path -Parent (Split-Path -Parent $PSScriptRoot))
$deployDir = Join-Path $repoRoot "deploy"
$images = New-Object System.Collections.Generic.List[string]

Write-Host "==> Collecting Phase 1 image list from manifests"
$manifests = Get-ChildItem -Path (Join-Path $deployDir "images") -Filter "manifest.yaml" -Recurse -File
foreach ($manifest in $manifests) {
    $content = Get-Content $manifest.FullName
    $phaseLine = $content | Where-Object { $_ -match '^prepull_phase:' } | Select-Object -First 1
    if (-not $phaseLine) {
        continue
    }

    $phase = ($phaseLine -split ':', 2)[1].Trim().Trim('"')
    if ($phase -ne "1") {
        continue
    }

    $projectLine = $content | Where-Object { $_ -match '^registry_project:' } | Select-Object -First 1
    $nameLine = $content | Where-Object { $_ -match '^name:' } | Select-Object -First 1
    if (-not $projectLine -or -not $nameLine) {
        continue
    }

    $project = ($projectLine -split ':', 2)[1].Trim().Trim('"')
    $name = ($nameLine -split ':', 2)[1].Trim().Trim('"')
    $tagLines = $content | Where-Object { $_ -match '^\s{4}tag:' }

    foreach ($tagLine in $tagLines) {
        $tag = ($tagLine -split ':', 2)[1].Trim().Trim('"')
        if ($tag) {
            $images.Add("$Registry/$project/$name`:$tag")
        }
    }
}

Write-Host "==> Found $($images.Count) images to bundle"
foreach ($image in $images) {
    Write-Host "    Pulling: $image"
    & docker pull $image
    if ($LASTEXITCODE -ne 0) {
        throw "拉取镜像失败: $image"
    }
}

$bundleFile = Join-Path $OutputDir "$BundleName.tar"
$hashFile = "$bundleFile.sha256"

Write-Host "==> Saving to $bundleFile"
& docker save -o $bundleFile @($images.ToArray())
if ($LASTEXITCODE -ne 0) {
    throw "导出镜像包失败"
}

Write-Host "==> Generating SHA256 checksum"
$hash = (Get-FileHash -Algorithm SHA256 -Path $bundleFile).Hash.ToLower()
Set-Content -Path $hashFile -Value "$hash  $(Split-Path $bundleFile -Leaf)"

Write-Host "==> Bundle complete"
Get-Item $bundleFile, $hashFile | Select-Object FullName, Length

