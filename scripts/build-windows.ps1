# Builds the Windows deliverables:
#   dist\Pandora FMS Netflow Probe.exe  (Fyne GUI, cgo, auto-elevated via manifest)
#   dist\netflow-probe.exe              (headless agent / Windows service, no cgo)
#
# Requirements:
#   - Go 1.27+
#   - A C compiler for cgo: mingw-w64 (MSYS2) or TDM-GCC, in PATH.
#   - The rsrc tool is downloaded on the fly with `go run`.
#
# Usage (from a Developer PowerShell):  pwsh scripts\build-windows.ps1
param(
    [string]$Version = "1.0.33"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root
$dist = Join-Path $root "dist"
New-Item -ItemType Directory -Force -Path $dist | Out-Null

Write-Host "==> Generando recursos (icono + manifiesto) para la GUI"
go run github.com/akavel/rsrc@latest `
    -manifest packaging/windows/app.manifest `
    -ico packaging/windows/AppIcon.ico `
    -o cmd/netflow-probe-gui/rsrc_windows_amd64.syso

Write-Host "==> Compilando la GUI (cgo, ventana sin consola)"
$env:CGO_ENABLED = "1"
go build -trimpath `
    -ldflags "-H windowsgui -s -w -X main.version=$Version" `
    -o (Join-Path $dist "Pandora FMS Netflow Probe.exe") `
    ./cmd/netflow-probe-gui

Write-Host "==> Compilando el agente/servicio (sin cgo)"
$env:CGO_ENABLED = "0"
go build -trimpath `
    -ldflags "-s -w -X main.version=$Version" `
    -o (Join-Path $dist "netflow-probe.exe") `
    ./cmd/netflow-probe

Remove-Item cmd/netflow-probe-gui/rsrc_windows_amd64.syso -ErrorAction SilentlyContinue

Write-Host "==> Listo. Salida en $dist"
Write-Host "    Opcional: pwsh scripts\fetch-windivert.ps1  (copia WinDivert junto al .exe)"
