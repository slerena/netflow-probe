# Downloads WinDivert and copies its DLL + driver next to the built executables.
#
# Source: the Native.WinDivert NuGet package (reqrypt.org is often unreachable
# from some networks/VMs). WinDivert is dual-licensed LGPL-3.0 / GPL-2.0, which
# is compatible with this project's GPL-2.0 license.
param(
    [string]$OutDir = "dist",
    [string]$Version = "2.2.2"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
Set-Location $root
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

$url = "https://www.nuget.org/api/v2/package/Native.WinDivert/$Version"
$nupkg = Join-Path $env:TEMP "native.windivert.$Version.nupkg"
$dir = Join-Path $env:TEMP "native.windivert.$Version"

Write-Host "==> Descargando $url"
Invoke-WebRequest -Uri $url -OutFile $nupkg
Expand-Archive -Path $nupkg -DestinationPath $dir -Force

Copy-Item (Join-Path $dir "runtimes\win-x64\native\WinDivert.dll") $OutDir -Force
Copy-Item (Join-Path $dir "runtimes\win-x64\native\WinDivert64.sys") $OutDir -Force

Write-Host "==> WinDivert copiado en $OutDir"
Write-Host "    Archivos: WinDivert.dll, WinDivert64.sys"
