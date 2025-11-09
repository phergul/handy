New-Item -Force -ItemType Directory -Path "$env:LOCALAPPDATA\mp4_compress" | Out-Null

try {
    go build -o "$env:LOCALAPPDATA\mp4_compress\mp4_compress.exe" .\mp4_compress.go 2>&1 | Out-Null
    Write-Host "Build complete."
}
catch {
    Write-Warning "Go build failed"
}

$filesToCopy = @(
    ".\mp4_compress.ps1",
    ".\add_mp4_compress_context.reg",
    ".\remove_mp4_compress_context.reg"
)

foreach ($file in $filesToCopy) {
    if (Test-Path $file) {
        Copy-Item -Force $file "$env:LOCALAPPDATA\mp4_compress\" | Out-Null
        Write-Host "Copied $(Split-Path $file -Leaf)"
    } else {
        Write-Warning "File not found: $file"
    }
}

Write-Host "`nInstalled to: $env:LOCALAPPDATA\mp4_compress"
Write-Host "To enable right-click compression: double-click 'add_mp4_compress_context.reg'"
Write-Host "To remove it later: double-click 'remove_mp4_compress_context.reg'"
