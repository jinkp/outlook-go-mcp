# outlook-mcp installer for Windows
# Usage: irm https://raw.githubusercontent.com/jinkp/outlook-go-mcp/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$releasesUrl = "https://github.com/jinkp/outlook-go-mcp/releases"

function Fail-Install {
    param([string]$Message)

    Write-Host ""
    Write-Host "outlook-mcp install failed: $Message" -ForegroundColor Red
    Write-Host "Download a release manually from: $releasesUrl" -ForegroundColor Yellow
    exit 1
}

try {
    if ($env:OS -ne "Windows_NT") {
        Fail-Install "This installer only supports Windows. outlook-mcp requires Outlook Desktop (COM automation)."
    }

    $arch = $null
    if ([System.Environment]::Is64BitOperatingSystem) {
        $cpuArch = $env:PROCESSOR_ARCHITECTURE
        if ($cpuArch -eq "ARM64") {
            $arch = "arm64"
        } else {
            $arch = "amd64"
        }
    }
    if (-not $arch) {
        Fail-Install "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"
    }

    $assetName  = "outlook-mcp-windows-$arch.exe"
    $downloadUrl = "https://github.com/jinkp/outlook-go-mcp/releases/latest/download/$assetName"
    $installDir  = Join-Path $env:LOCALAPPDATA "outlook-mcp"
    $target      = Join-Path $installDir "outlook-mcp.exe"

    Write-Host ""
    Write-Host "outlook-mcp installer" -ForegroundColor Cyan
    Write-Host "Downloading $assetName..." -ForegroundColor Cyan

    New-Item -ItemType Directory -Force -Path $installDir | Out-Null
    Invoke-WebRequest -Uri $downloadUrl -OutFile $target

    # Add install dir to user PATH if not already present
    $userPath   = [Environment]::GetEnvironmentVariable("Path", "User")
    $pathEntries = @($userPath -split ";" | Where-Object { $_ })
    if ($pathEntries -notcontains $installDir) {
        $newUserPath = if ([string]::IsNullOrWhiteSpace($userPath)) { $installDir } else { "$userPath;$installDir" }
        [Environment]::SetEnvironmentVariable("Path", $newUserPath, "User")
        Write-Host "Added $installDir to your user PATH." -ForegroundColor Green
    }

    # Also update the current session PATH so we can call it immediately
    if (-not (($env:Path -split ";") -contains $installDir)) {
        $env:Path = "$installDir;$env:Path"
    }

    # Smoke test
    $versionOutput = & outlook-mcp --version 2>&1
    if ($LASTEXITCODE -ne 0) {
        Fail-Install "Installed binary did not pass 'outlook-mcp --version'."
    }

    Write-Host ""
    Write-Host "outlook-mcp installed successfully to $target" -ForegroundColor Green
    Write-Host $versionOutput
    Write-Host ""
    Write-Host "Next steps:" -ForegroundColor Cyan
    Write-Host "  1. Copy configs\config.example.yaml to config.yaml and edit it"
    Write-Host "  2. Register in your AI client:"
    Write-Host "       outlook-mcp setup opencode"
    Write-Host "       outlook-mcp setup claude"
    Write-Host "  3. Start a new terminal — the PATH change takes effect in fresh shells"
}
catch {
    Fail-Install $_.Exception.Message
}
