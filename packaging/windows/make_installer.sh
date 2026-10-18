#!/usr/bin/env bash
# Construye el instalador NSIS de Windows (Pandora FMS Netflow Probe Setup <ver>.exe).
# Se ejecuta desde macOS/Linux: makensis no necesita Windows.
# Uso: packaging/windows/make_installer.sh   (o `make windows-installer`)
#
# Requisitos:
#   - makensis en el PATH           ->  brew install makensis
#   - Pillow (recursos y licencias) ->  pip3 install --user pillow
#   - payload ya preparado en packaging/windows/payload/:
#       PandoraFMSNetflowProbe.exe, netflow-probe.exe,
#       WinDivert.dll, WinDivert64.sys, mesa/{opengl32,libgallium_wgl}.dll
set -euo pipefail

cd "$(dirname "$0")/../.."

WIN="packaging/windows"
PAYLOAD="$WIN/payload"
EXE="$PAYLOAD/PandoraFMSNetflowProbe.exe"
CLI="$PAYLOAD/netflow-probe.exe"
WD_DLL="$PAYLOAD/WinDivert.dll"
WD_SYS="$PAYLOAD/WinDivert64.sys"
MESA="$PAYLOAD/mesa"
ICO="$WIN/AppIcon.ico"
NSI="$WIN/PandoraFMSNetflowProbe.nsi"
LEGAL="packaging/macos/dmg-resources/legal"

VERSION="$(python3 - <<'PY'
import plistlib
with open("packaging/macos/Info.plist", "rb") as fh:
    print(plistlib.load(fh)["CFBundleShortVersionString"])
PY
)"
OUT="$PWD/dist/Pandora FMS Netflow Probe Setup ${VERSION}.exe"
mkdir -p dist

for f in "$EXE" "$CLI" "$WD_DLL" "$WD_SYS"; do
	[ -f "$f" ] || { echo "Falta $f (prepara el payload)." >&2; exit 1; }
done
command -v makensis >/dev/null 2>&1 || { echo "Falta makensis: brew install makensis" >&2; exit 1; }

if [ ! -f "$ICO" ] || [ ! -f "$WIN/welcome.bmp" ] || [ ! -f "$WIN/header.bmp" ]; then
	python3 "$WIN/make_assets.py"
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
python3 "$WIN/make_license_rtf.py" "$LEGAL/es.txt" "$TMP/es.rtf"
python3 "$WIN/make_license_rtf.py" "$LEGAL/en.txt" "$TMP/en.rtf"
{ printf '\xef\xbb\xbf'; cat "$WIN/LEEME.txt"; } > "$TMP/LEEME.txt"

# Mesa probada (26.2.4 release-msvc). Si no coincide, se aborta: otra build
# puede tomar el camino D3D12 y hacer que Fyne no arranque.
check_mesa() {
	local want_ogl="75b295fc80df0eaa8f1264c28663f6932c8b72d36513eb092556608858b7f5e0"
	local want_gal="53e49adf87590d5b0cab8553b1e98084812040cc1fca8c487695b17b7af3702e"
	local sha_cmd got
	if command -v sha256sum >/dev/null 2>&1; then
		sha_cmd() { sha256sum "$1" | awk '{print $1}'; }
	else
		sha_cmd() { shasum -a 256 "$1" | awk '{print $1}'; }
	fi
	got="$(sha_cmd "$MESA/opengl32.dll")"
	[ "$got" = "$want_ogl" ] || { echo "Mesa opengl32.dll inesperado ($got)" >&2; exit 1; }
	got="$(sha_cmd "$MESA/libgallium_wgl.dll")"
	[ "$got" = "$want_gal" ] || { echo "Mesa libgallium_wgl.dll inesperado ($got)" >&2; exit 1; }
}

MESA_ARGS=()
if [ -f "$MESA/opengl32.dll" ] && [ -f "$MESA/libgallium_wgl.dll" ]; then
	check_mesa
	"$WIN/write_launcher.sh" "$TMP/Modo compatible.bat"
	MESA_ARGS=(-DWITH_MESA=1 -DMESA_DIR="$PWD/$MESA" -DCOMPAT_BAT="$TMP/Modo compatible.bat")
	echo "Modo compatible: incluido (Mesa)"
else
	echo "AVISO: sin Mesa en $MESA; el instalador saldrá sólo en modo hardware." >&2
fi

rm -f "$OUT"
makensis -V2 \
	-DVERSION="$VERSION" \
	-DEXE_SRC="$PWD/$EXE" \
	-DCLI_SRC="$PWD/$CLI" \
	-DWINDIVERT_DLL="$PWD/$WD_DLL" \
	-DWINDIVERT_SYS="$PWD/$WD_SYS" \
	-DICO="$PWD/$ICO" \
	-DWELCOME_BMP="$PWD/$WIN/welcome.bmp" \
	-DHEADER_BMP="$PWD/$WIN/header.bmp" \
	-DLICENSE_ES="$TMP/es.rtf" \
	-DLICENSE_EN="$TMP/en.rtf" \
	-DLEEME="$TMP/LEEME.txt" \
	-DOUT_FILE="$OUT" \
	${MESA_ARGS[@]+"${MESA_ARGS[@]}"} \
	"$PWD/$NSI"

echo "Instalador: ${OUT#"$PWD"/}"
ls -lh "$OUT"
