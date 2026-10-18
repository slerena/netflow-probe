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

cp "$EXE" "$STAGE/PandoraFMSNetflowProbe.exe"
cp "$CLI" "$STAGE/netflow-probe.exe"
cp "$WD_DLL" "$WD_SYS" "$STAGE/"
{ printf '\xef\xbb\xbf'; cat "$WIN/LEEME.txt"; } > "$STAGE/LEEME.txt"

if [ -f "$MESA/opengl32.dll" ] && [ -f "$MESA/libgallium_wgl.dll" ]; then
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
