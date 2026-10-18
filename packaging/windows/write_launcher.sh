#!/usr/bin/env bash
# Escribe "Modo compatible.bat" (arranque con Mesa/llvmpipe).
# CRLF y sin acentos: cmd.exe usa la página de códigos ANSI.
# Uso: write_launcher.sh <destino>
set -euo pipefail

dest="${1:?uso: write_launcher.sh <destino>}"

printf '%s\r\n' \
	'@echo off' \
	'rem Pandora FMS Netflow Probe - modo compatible (renderizado por software).' \
	'rem Mesa lee estas variables al cargar la DLL: se fijan en el entorno del proceso.' \
	'set GALLIUM_DRIVER=llvmpipe' \
	'set MESA_GL_VERSION_OVERRIDE=3.3' \
	'set MESA_GLSL_VERSION_OVERRIDE=330' \
	'start "" "%~dp0swgl\PandoraFMSNetflowProbe.exe"' \
	> "$dest"
