# Downloads WinDivert and copies its DLL + driver next to the built executables,
# so the agent can capture without a separate installation step.
#
# WinDivert is dual-licensed LGPL-3.0 / GPL-2.0, which is compatible with this
# project's GPL-2.0 license.
param(
    [string]$OutDir = "dist",
    [string]$Version = "2.2.2"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

$url = "https://reqrypt.org/download/WinDivert-$Version-A.zip"
$zip = Join-Path $env:TEMP "windivert-$Version.zip"
$dir = Join-Path $env:TEMP "windivert-$Version"

Write-Host "==> Descargando $url"
Invoke-WebRequest -Uri $url -OutFile $zip
Expand-Archive -Path $zip -DestinationPath $dir -Force

Copy-Item (Join-Path $dir "x64\WinDivert.dll")   $OutDir -Force
Copy-Item (Join-Path $dir "x64\WinDivert64.sys") $OutDir -Force

Write-Host "==> WinDivert copiado en $OutDir"
Write-Host "    Archivos: WinDivert.dll, WinDivert64.sys"
