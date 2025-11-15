if (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole] "Administrator")) {
    Write-Host "Requesting administrator privileges..." -ForegroundColor Yellow
    $arguments = "-NoProfile -ExecutionPolicy Bypass -File `"$PSCommandPath`""
    Start-Process powershell.exe -Verb RunAs -ArgumentList $arguments
    exit
}

Write-Host "MP4 Video Compressor" -ForegroundColor Cyan
Write-Host ""

$installDir = "$env:LOCALAPPDATA\mp4_compress"

Write-Host "Checking for ffmpeg..." -NoNewline
$ffmpegInstalled = $null -ne (Get-Command ffmpeg -ErrorAction SilentlyContinue)

if (-not $ffmpegInstalled) {
    Write-Host " NOT FOUND" -ForegroundColor Yellow
    Write-Host "Installing ffmpeg via winget..." -ForegroundColor Yellow
    
    try {
        winget install --id Gyan.FFmpeg --silent --accept-package-agreements --accept-source-agreements
        Write-Host "ffmpeg installed successfully!" -ForegroundColor Green
        
        $env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")
    }
    catch {
        Write-Host "FATAL: Failed to install ffmpeg automatically." -ForegroundColor Red
        Write-Host "Please install ffmpeg manually: winget install ffmpeg" -ForegroundColor Red
        Read-Host "Press Enter to exit"
        exit 1
    }
}
else {
    Write-Host " FOUND" -ForegroundColor Green
}

Write-Host "Setting PowerShell execution policy..." -NoNewline
try {
    Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser -Force
    Write-Host " DONE" -ForegroundColor Green
}
catch {
    Write-Host " FAILED" -ForegroundColor Yellow
}

Write-Host "Creating installation directory..." -NoNewline
New-Item -Force -ItemType Directory -Path $installDir | Out-Null
Write-Host " DONE" -ForegroundColor Green

Write-Host "Building video compressor..." -NoNewline
$scriptDir = Join-Path -Path (Split-Path -Parent $MyInvocation.MyCommand.Path) -ChildPath "internal"
# $scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path + "\internal\"

$goInstalled = $null -ne (Get-Command go -ErrorAction SilentlyContinue)
$goSourceExists = Test-Path "$scriptDir\mp4_compress.go"
$precompiledExists = Test-Path "$scriptDir\mp4_compress.exe"

if ($goInstalled -and $goSourceExists) {
    try {
        Push-Location $scriptDir
        go build -o "$installDir\mp4_compress.exe" .\mp4_compress.go 2>&1 | Out-Null
        Pop-Location
        Write-Host " BUILT FROM SOURCE" -ForegroundColor Green
    }
    catch {
        Pop-Location
        Write-Host " BUILD FAILED" -ForegroundColor Yellow
        
        if ($precompiledExists) {
            Copy-Item -Force "$scriptDir\mp4_compress.exe" "$installDir\mp4_compress.exe"
            Write-Host "Using pre-compiled binary instead" -ForegroundColor Green
        }
        else {
            Write-Host "FATAL: Go build failed and no pre-compiled binary found." -ForegroundColor Red
            Read-Host "Press Enter to exit"
            exit 1
        }
    }
}
elseif ($precompiledExists) {
    Copy-Item -Force "$scriptDir\mp4_compress.exe" "$installDir\mp4_compress.exe"
    Write-Host " COPIED PRE-COMPILED BINARY" -ForegroundColor Green
}
else {
    Write-Host " FAILED" -ForegroundColor Red
    Write-Host "FATAL: No pre-compiled binary found and Go is not installed." -ForegroundColor Red
    Write-Host "Please either:" -ForegroundColor Yellow
    Write-Host "  1. Install Go and ensure mp4_compress.go is present" -ForegroundColor Yellow
    Write-Host "  2. Or place a pre-compiled mp4_compress.exe in the internal folder" -ForegroundColor Yellow
    Read-Host "Press Enter to exit"
    exit 1
}

Write-Host "Copying support files..." -NoNewline
$filesToCopy = @(
    "mp4_compress.ps1",
    "remove_mp4_compress_context.reg"
)

foreach ($file in $filesToCopy) {
    $sourcePath = Join-Path $scriptDir $file
    if (Test-Path $sourcePath) {
        Copy-Item -Force $sourcePath "$installDir\"
    }
}
Write-Host " DONE" -ForegroundColor Green

Write-Host "Registering right-click context menu..." -NoNewline
$regPath = "HKCU:\Software\Classes\SystemFileAssociations\.mp4\shell\Compress Video"
$commandPath = "$regPath\command"

try {
    New-Item -Path $regPath -Force | Out-Null
    Set-ItemProperty -Path $regPath -Name "(Default)" -Value "Compress Video"
    
    New-Item -Path $commandPath -Force | Out-Null
    $commandValue = "powershell.exe -ExecutionPolicy Bypass -WindowStyle Hidden -File `"$installDir\mp4_compress.ps1`" `"%1`""
    Set-ItemProperty -Path $commandPath -Name "(Default)" -Value $commandValue
    Write-Host " DONE" -ForegroundColor Green
}
catch {
    Write-Host " FAILED" -ForegroundColor Red
    Write-Host "Error registering context menu: $($_.Exception.Message)" -ForegroundColor Red
	Write-Host "Change and run the manual script at '$scriptDir\add_mp4_compress_context.reg' instead" -ForegroundColor Red
}

Write-Host ""
Write-Host "Installation Complete!" -ForegroundColor Green
Write-Host ""
Write-Host "Installed to: $installDir" -ForegroundColor Cyan
Write-Host ""
Write-Host "Usage:" -ForegroundColor White
Write-Host "  Right-click any .mp4 file and select 'Compress Video'" -ForegroundColor White
Write-Host ""
Write-Host "To uninstall:" -ForegroundColor White
Write-Host "  Run: $installDir\uninstall.ps1" -ForegroundColor White
Write-Host ""

$uninstallScript = @"
Write-Host "Uninstalling MP4 Video Compressor..." -ForegroundColor Yellow

`$regPath = "HKCU:\Software\Classes\SystemFileAssociations\.mp4\shell\Compress Video"
if (Test-Path `$regPath) {
    Remove-Item -Path `$regPath -Recurse -Force
    Write-Host "Context menu removed" -ForegroundColor Green
}

`$installDir = "$installDir"
if (Test-Path `$installDir) {
    Remove-Item -Path `$installDir -Recurse -Force
    Write-Host "Installation directory removed" -ForegroundColor Green
}

Write-Host "Uninstall complete!" -ForegroundColor Green
Read-Host "Press Enter to close"
"@

Set-Content -Path "$installDir\uninstall.ps1" -Value $uninstallScript

Read-Host "Press Enter to close"
