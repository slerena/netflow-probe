#!/usr/bin/env bash
# Empaqueta el ZIP portable de Windows: GUI + CLI + WinDivert + swgl/ (Mesa).
# Uso: packaging/windows/make_zip.sh   (o `make windows-zip`)
set -euo pipefail

cd "$(dirname "$0")/../.."

WIN="packaging/windows"
PAYLOAD="$WIN/payload"
EXE="$PAYLOAD/PandoraFMSNetflowProbe.exe"
CLI="$PAYLOAD/netflow-probe.exe"
WD_DLL="$PAYLOAD/WinDivert.dll"
WD_SYS="$PAYLOAD/WinDivert64.sys"
MESA="$PAYLOAD/mesa"

VERSION="$(python3 - <<'PY'
import plistlib
with open("packaging/macos/Info.plist", "rb") as fh:
    print(plistlib.load(fh)["CFBundleShortVersionString"])
PY
)"
OUT="$PWD/dist/Pandora FMS Netflow Probe ${VERSION} (Windows).zip"
mkdir -p dist

for f in "$EXE" "$CLI" "$WD_DLL" "$WD_SYS"; do
	[ -f "$f" ] || { echo "Falta $f (prepara el payload)." >&2; exit 1; }
done

STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT

# Mesa probada (26.2.4 release-msvc): abortar si no coincide.
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

cp "$EXE" "$STAGE/PandoraFMSNetflowProbe.exe"
cp "$CLI" "$STAGE/netflow-probe.exe"
cp "$WD_DLL" "$WD_SYS" "$STAGE/"
{ printf '\xef\xbb\xbf'; cat "$WIN/LEEME.txt"; } > "$STAGE/LEEME.txt"

if [ -f "$MESA/opengl32.dll" ] && [ -f "$MESA/libgallium_wgl.dll" ]; then
	check_mesa
	mkdir -p "$STAGE/swgl"
	cp "$EXE" "$STAGE/swgl/PandoraFMSNetflowProbe.exe"
	cp "$WD_DLL" "$WD_SYS" "$STAGE/swgl/"
	cp "$MESA/opengl32.dll" "$MESA/libgallium_wgl.dll" "$STAGE/swgl/"
	"$WIN/write_launcher.sh" "$STAGE/Modo compatible.bat"
	echo "Modo compatible: incluido (Mesa)"
else
	echo "AVISO: sin Mesa; el ZIP saldrá sólo en modo hardware." >&2
fi

rm -f "$OUT"
( cd "$STAGE" && zip -q -r -9 "$OUT" . )
echo "ZIP: ${OUT#"$PWD"/}"
unzip -l "$OUT"
