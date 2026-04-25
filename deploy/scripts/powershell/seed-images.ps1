# seed-images.ps1
# 扫描 deploy/images/**/manifest.yaml，通过后端管理 API 同步到 images 表

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$BackendUrl = if ($env:BACKEND_URL) { $env:BACKEND_URL.TrimEnd('/') } else { "http://localhost:8080/api/v1" }
$AdminToken = if ($env:ADMIN_TOKEN) { $env:ADMIN_TOKEN } else { throw "ADMIN_TOKEN is required" }
$repoRoot = Split-Path -Parent (Split-Path -Parent (Split-Path -Parent $PSScriptRoot))
$deployDir = Join-Path $repoRoot "deploy"
$count = 0
$failed = 0

Write-Host "==> Scanning manifest files in $deployDir/images/"
$manifests = Get-ChildItem -Path (Join-Path $deployDir "images") -Filter "manifest.yaml" -Recurse -File

if ($manifests.Count -eq 0) {
    Write-Host "==> No manifest.yaml files found under $deployDir/images/"
    exit 0
}

foreach ($manifest in $manifests) {
    $name = Split-Path -Leaf $manifest.DirectoryName
    Write-Host "    Syncing: $name"

    try {
        $response = Invoke-WebRequest -Method Post -Uri "$BackendUrl/admin/images/sync" -Headers @{
            Authorization = "Bearer $AdminToken"
            "Content-Type" = "application/yaml"
        } -InFile $manifest.FullName

        if ($response.StatusCode -ge 200 -and $response.StatusCode -lt 300) {
            $count++
        }
        else {
            Write-Host "    FAILED ($($response.StatusCode)): $name"
            $failed++
        }
    }
    catch {
        $statusCode = $_.Exception.Response.StatusCode.value__
        if (-not $statusCode) {
            $statusCode = "ERR"
        }
        Write-Host "    FAILED ($statusCode): $name"
        $failed++
    }
}

Write-Host "==> Seed complete: $count synced, $failed failed"
if ($failed -gt 0) {
    exit 1
}
