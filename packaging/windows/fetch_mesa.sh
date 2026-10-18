#!/usr/bin/env bash
# Descarga Mesa (llvmpipe) para el "modo compatible" de Windows y lo deja en
# packaging/windows/payload/mesa/ (no versionado). Verifica SHA-256.
#
# Uso: packaging/windows/fetch_mesa.sh
set -euo pipefail

cd "$(dirname "$0")/../.."

DEST="packaging/windows/payload/mesa"
URL="https://github.com/pal1000/mesa-dist-win/releases/download/26.2.4/mesa3d-26.2.4-release-msvc.7z"
WANT_OGL="75b295fc80df0eaa8f1264c28663f6932c8b72d36513eb092556608858b7f5e0"
WANT_GAL="53e49adf87590d5b0cab8553b1e98084812040cc1fca8c487695b17b7af3702e"

sha() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | awk '{print $1}'
	else
		shasum -a 256 "$1" | awk '{print $1}'
	fi
}

if [ -f "$DEST/opengl32.dll" ] && [ "$(sha "$DEST/opengl32.dll")" = "$WANT_OGL" ] &&
	[ -f "$DEST/libgallium_wgl.dll" ] && [ "$(sha "$DEST/libgallium_wgl.dll")" = "$WANT_GAL" ]; then
	echo "Mesa ya está en $DEST con el SHA-256 correcto."
	exit 0
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "Descargando Mesa 26.2.4…"
curl -fsSL -o "$TMP/mesa.7z" "$URL"

echo "Extrayendo…"
if ! tar -xf "$TMP/mesa.7z" -C "$TMP" 2>/dev/null; then
	if command -v 7zz >/dev/null 2>&1; then
		7zz x -y -o"$TMP" "$TMP/mesa.7z" >/dev/null
	else
		echo "No sé extraer .7z: instala 7-Zip (brew install sevenzip) o usa bsdtar." >&2
		exit 1
	fi
fi

mkdir -p "$DEST"
cp "$TMP/x64/opengl32.dll" "$TMP/x64/libgallium_wgl.dll" "$DEST/"

got_ogl="$(sha "$DEST/opengl32.dll")"
got_gal="$(sha "$DEST/libgallium_wgl.dll")"
if [ "$got_ogl" != "$WANT_OGL" ] || [ "$got_gal" != "$WANT_GAL" ]; then
	echo "SHA-256 inesperado en Mesa; no se usa." >&2
	echo "  opengl32: $got_ogl" >&2
	echo "  gallium : $got_gal" >&2
	exit 1
fi
echo "Mesa listo en $DEST (SHA-256 verificado)."
