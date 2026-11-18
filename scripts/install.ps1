function Install-Opus {
    Set-StrictMode -Version Latest
    $ErrorActionPreference = 'Stop'
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12

    $arch = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
    if ($arch -ne 'AMD64') {
        throw "Unsupported Windows architecture '$arch'. Only AMD64 is supported."
    }

    if ($env:OPUS_VERSION) {
        $version = $env:OPUS_VERSION.TrimStart('v')
        $tag = "v$version"
    } else {
        $headers = @{ 'User-Agent' = 'opus-installer' }
        if ($env:GITHUB_TOKEN) {
            $headers['Authorization'] = "Bearer $($env:GITHUB_TOKEN)"
        }
        $releases = Invoke-RestMethod -Uri 'https://api.github.com/repos/kilip/opus/releases?per_page=1' -Headers $headers
        if (-not $releases -or $releases.Count -eq 0 -or -not $releases[0].tag_name) {
            throw 'Failed to resolve latest Opus release tag. Set $env:OPUS_VERSION and retry.'
        }
        $tag = [string]$releases[0].tag_name
        $version = $tag.TrimStart('v')
    }

    # ponytail: OPUS_DOWNLOAD_BASE hook overrides release URL for CI fixture tests
    $baseUrl = if ($env:OPUS_DOWNLOAD_BASE) { $env:OPUS_DOWNLOAD_BASE.TrimEnd('/') } else { "https://github.com/kilip/opus/releases/download/$tag" }
    $archive = "opus_${version}_windows_amd64.zip"

    $downloadFile = {
        param([string]$Url, [string]$Dest)
        $uri = [Uri]$Url
        if ($uri.IsFile) {
            Copy-Item -LiteralPath $uri.LocalPath -Destination $Dest -Force
        } else {
            Invoke-WebRequest -Uri $Url -OutFile $Dest -UseBasicParsing
        }
    }

    $tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ("opus-install-" + [System.Guid]::NewGuid().ToString('N'))
    New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null

    $installDir = if ($env:OPUS_INSTALL_DIR) { $env:OPUS_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'opus\bin' }
    $target = Join-Path $installDir 'opus.exe'

    try {
        $archivePath = Join-Path $tmpDir $archive
        $checksumsPath = Join-Path $tmpDir 'checksums.txt'

        & $downloadFile "$baseUrl/$archive" $archivePath
        & $downloadFile "$baseUrl/checksums.txt" $checksumsPath

        $expected = $null
        foreach ($line in Get-Content -LiteralPath $checksumsPath) {
            if ($line -match '^([0-9a-fA-F]{64})\s+(.+)$') {
                if ($Matches[2].Trim() -eq $archive) {
                    $expected = $Matches[1].ToLowerInvariant()
                    break
                }
            }
        }
        if (-not $expected) {
            throw "Checksum entry for '$archive' not found in checksums.txt."
        }

        $actual = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($actual -ne $expected) {
            throw "Checksum mismatch for '$archive' (expected $expected, got $actual)."
        }

        $extractDir = Join-Path $tmpDir 'extract'
        Expand-Archive -LiteralPath $archivePath -DestinationPath $extractDir -Force

        $extractedBin = Join-Path $extractDir 'opus.exe'
        if (-not (Test-Path -LiteralPath $extractedBin)) {
            throw "Extracted archive does not contain 'opus.exe'."
        }

        New-Item -ItemType Directory -Path $installDir -Force | Out-Null
        $oldTarget = Join-Path $installDir 'opus.exe.old'
        if (Test-Path -LiteralPath $target) {
            if (Test-Path -LiteralPath $oldTarget) {
                Remove-Item -LiteralPath $oldTarget -Force
            }
            Move-Item -LiteralPath $target -Destination $oldTarget -Force
        }
        Copy-Item -LiteralPath $extractedBin -Destination $target -Force
    } finally {
        if (Test-Path -LiteralPath $tmpDir) {
            Remove-Item -LiteralPath $tmpDir -Recurse -Force
        }
    }

    $normalizedTarget = $installDir.TrimEnd('\')
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    $parts = if ($userPath) { @($userPath -split ';' | Where-Object { $_ -ne '' }) } else { @() }
    $hasEntry = $false
    foreach ($p in $parts) {
        if ($p.TrimEnd('\').Equals($normalizedTarget, [System.StringComparison]::OrdinalIgnoreCase)) {
            $hasEntry = $true
            break
        }
    }
    if (-not $hasEntry) {
        $newUserPath = if ($parts.Count -gt 0) { ($parts + $installDir) -join ';' } else { $installDir }
        [Environment]::SetEnvironmentVariable('Path', $newUserPath, 'User')
    }

    $envParts = @($env:Path -split ';' | Where-Object { $_ -ne '' })
    $inSession = $false
    foreach ($p in $envParts) {
        if ($p.TrimEnd('\').Equals($normalizedTarget, [System.StringComparison]::OrdinalIgnoreCase)) {
            $inSession = $true
            break
        }
    }
    if (-not $inSession) {
        $env:Path = "$installDir;$env:Path"
    }

    & $target --version

    foreach ($dep in @('git', 'gh', 'agy')) {
        if (-not (Get-Command $dep -ErrorAction SilentlyContinue)) {
            Write-Warning "'$dep' not found in PATH (required on worker machines)."
        }
    }
}

Install-Opus
