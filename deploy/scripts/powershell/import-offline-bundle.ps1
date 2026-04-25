# import-offline-bundle.ps1
# 导入离线镜像包

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

param(
    [Parameter(Mandatory = $true, Position = 0)]
    [string]$BundleFile,
    [Parameter(Position = 1)]
    [string]$TargetRegistry = "registry.local"
)

if (-not (Test-Path $BundleFile)) {
    throw "Bundle file not found: $BundleFile"
}

$shaFile = "$BundleFile.sha256"
if (Test-Path $shaFile) {
    Write-Host "==> Verifying SHA256 checksum"
    $expected = ((Get-Content $shaFile | Select-Object -First 1) -split '\s+')[0].ToLower()
    $actual = (Get-FileHash -Algorithm SHA256 -Path $BundleFile).Hash.ToLower()
    if ($expected -ne $actual) {
        throw "SHA256 校验失败"
    }
}
else {
    Write-Host "WARNING: No SHA256 file found, skipping verification"
}

Write-Host "==> Loading images from bundle"
$loadOutput = & docker load -i $BundleFile 2>&1
if ($LASTEXITCODE -ne 0) {
    throw "导入镜像包失败"
}

$images = @()
foreach ($line in $loadOutput) {
    if ($line -match 'Loaded image:\s+(.+)$') {
        $images += $Matches[1].Trim()
    }
}

Write-Host "==> Retagging and pushing to $TargetRegistry"
foreach ($image in $images) {
    $newTag = [regex]::Replace($image, '^[^/]+/', "$TargetRegistry/")
    Write-Host "    $image -> $newTag"
    & docker tag $image $newTag
    if ($LASTEXITCODE -ne 0) {
        throw "重打标签失败: $image"
    }
    & docker push $newTag
    if ($LASTEXITCODE -ne 0) {
        throw "推送镜像失败: $newTag"
    }
}

Write-Host "==> Import complete"

