# image-gc.ps1
# 镜像版本清理脚本

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$RegistryUrl = if ($env:REGISTRY_URL) { $env:REGISTRY_URL.TrimEnd('/') } else { throw "REGISTRY_URL is required" }
$KeepVersions = if ($env:KEEP_VERSIONS) { [int]$env:KEEP_VERSIONS } else { 3 }
$credential = $null
if ($env:REGISTRY_USERNAME -and $env:REGISTRY_PASSWORD) {
    $securePassword = ConvertTo-SecureString $env:REGISTRY_PASSWORD -AsPlainText -Force
    $credential = [PSCredential]::new($env:REGISTRY_USERNAME, $securePassword)
}

function Invoke-RegistryRequest {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Method,
        [Parameter(Mandatory = $true)]
        [string]$Url,
        [hashtable]$Headers
    )

    $params = @{
        Method      = $Method
        Uri         = $Url
        Headers     = $Headers
        ErrorAction = "Stop"
    }
    if ($credential) {
        $params.Credential = $credential
    }

    return Invoke-WebRequest @params
}

$catalog = (Invoke-RegistryRequest -Method "GET" -Url "$RegistryUrl/v2/_catalog?n=1000" | Select-Object -ExpandProperty Content) | ConvertFrom-Json
foreach ($repo in $catalog.repositories) {
    if (-not $repo) {
        continue
    }

    $tagResponse = (Invoke-RegistryRequest -Method "GET" -Url "$RegistryUrl/v2/$repo/tags/list" | Select-Object -ExpandProperty Content) | ConvertFrom-Json
    $tags = @($tagResponse.tags) | Sort-Object
    if ($tags.Count -le $KeepVersions) {
        continue
    }

    $removeCount = $tags.Count - $KeepVersions
    for ($i = 0; $i -lt $removeCount; $i++) {
        $tag = $tags[$i]
        Write-Host "==> deleting $repo`:$tag"
        $head = Invoke-RegistryRequest -Method "GET" -Url "$RegistryUrl/v2/$repo/manifests/$tag" -Headers @{
            Accept = "application/vnd.docker.distribution.manifest.v2+json"
        }
        $digest = $head.Headers["Docker-Content-Digest"]
        if ($digest) {
            Invoke-RegistryRequest -Method "DELETE" -Url "$RegistryUrl/v2/$repo/manifests/$digest" | Out-Null
        }
    }
}

