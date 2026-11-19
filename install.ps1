# Installs the POEditor CLI on Windows.
#
#   irm https://raw.githubusercontent.com/POEditor/poeditor-cli/master/install.ps1 | iex
#
# Environment:
#   POEDITOR_VERSION       release to install, e.g. v0.1.0 (default: the latest release)
#   POEDITOR_INSTALL_DIR   where to put poeditor.exe (default: %LOCALAPPDATA%\Programs\poeditor)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue' # Invoke-WebRequest is very slow with the progress bar
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

$repo = 'POEditor/poeditor-cli'
$base = if ($env:POEDITOR_RELEASES_URL) { $env:POEDITOR_RELEASES_URL } else { "https://github.com/$repo/releases" } # overridden only by tests

# throw, not exit: under `irm | iex` the script runs in the user's own session, and exit would
# close their PowerShell window.
function Fail($message) {
    throw "poeditor install: $message"
}

# A 32-bit PowerShell on 64-bit Windows reports x86; the real architecture is in PROCESSOR_ARCHITEW6432.
$cpu = if ($env:PROCESSOR_ARCHITEW6432) { $env:PROCESSOR_ARCHITEW6432 } else { $env:PROCESSOR_ARCHITECTURE }
switch ($cpu) {
    'AMD64' { $arch = 'amd64' }
    'ARM64' { $arch = 'arm64' }
    default { Fail "unsupported architecture $cpu" }
}

$tag = $env:POEDITOR_VERSION
if (-not $tag) {
    try {
        $tag = (Invoke-RestMethod -UseBasicParsing "https://api.github.com/repos/$repo/releases/latest").tag_name
    } catch {
        Fail "could not find the latest release: $_"
    }
}
if (-not $tag.StartsWith('v')) { $tag = "v$tag" }
$archive = "poeditor_$($tag.Substring(1))_windows_$arch.zip"

$tmp = Join-Path ([IO.Path]::GetTempPath()) ([IO.Path]::GetRandomFileName())
New-Item -ItemType Directory -Path $tmp | Out-Null
try {
    Write-Host "Downloading poeditor $tag for windows/$arch..."
    try {
        Invoke-WebRequest -UseBasicParsing "$base/download/$tag/$archive" -OutFile "$tmp\$archive"
        Invoke-WebRequest -UseBasicParsing "$base/download/$tag/checksums.txt" -OutFile "$tmp\checksums.txt"
    } catch {
        Fail "could not download $archive`: $_"
    }

    $line = Get-Content "$tmp\checksums.txt" | Where-Object { ($_ -split '\s+')[1] -eq $archive } | Select-Object -First 1
    if (-not $line) { Fail "$archive is not listed in checksums.txt" }
    $want = ($line -split '\s+')[0]
    $got = (Get-FileHash -Algorithm SHA256 "$tmp\$archive").Hash
    if ($got -ne $want) { Fail "checksum mismatch for $archive" } # -ne ignores case

    Expand-Archive -Path "$tmp\$archive" -DestinationPath "$tmp\out"

    $dir = if ($env:POEDITOR_INSTALL_DIR) { $env:POEDITOR_INSTALL_DIR } else { Join-Path $env:LOCALAPPDATA 'Programs\poeditor' }
    New-Item -ItemType Directory -Force -Path $dir | Out-Null
    Copy-Item -Force "$tmp\out\poeditor.exe" "$dir\poeditor.exe"
} finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if (-not (($userPath -split ';') -contains $dir)) {
    $newPath = if ($userPath) { "$userPath;$dir" } else { $dir }
    [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
    $env:Path = "$env:Path;$dir"
    Write-Host "Added $dir to your user PATH; open a new terminal to use it there."
}

Write-Host "Installed $(& "$dir\poeditor.exe" --version) to $dir\poeditor.exe"
