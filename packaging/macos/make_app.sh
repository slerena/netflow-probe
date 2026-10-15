#!/usr/bin/env bash
# Empaqueta "Pandora FMS Netflow Probe" como .app de macOS (y ZIP).
# Uso: packaging/macos/make_app.sh   (o `make macapp`)
#
# Firma ad-hoc por defecto (suficiente en el Mac que compila). Para distribuir
# sin avisos de Gatekeeper, exporta SIGN_IDENTITY con tu identidad
# "Developer ID Application: ...": se firma con hardened runtime + timestamp,
# requisito para notarizar. Usa SKIP_ZIP=1 para no generar el .zip.
set -euo pipefail

cd "$(dirname "$0")/../.."

APP_NAME="Pandora FMS Netflow Probe"
EXE="NetFlowProbe"
APP="$APP_NAME.app"
ICON="cmd/netflow-probe-gui/icon.png"
DIST="dist"

rm -rf "$DIST/$APP"
mkdir -p "$DIST/$APP/Contents/MacOS" "$DIST/$APP/Contents/Resources"

echo "==> Compilando la GUI (macOS, cgo)…"
CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" \
	-o "$DIST/$APP/Contents/MacOS/$EXE" ./cmd/netflow-probe-gui

echo "==> Generando el icono .icns…"
TMP="$(mktemp -d)"
ICONSET="$TMP/AppIcon.iconset"
mkdir -p "$ICONSET"
for s in 16 32 128 256 512; do
	sips -z "$s" "$s" "$ICON" --out "$ICONSET/icon_${s}x${s}.png" >/dev/null
	d=$((s * 2))
	sips -z "$d" "$d" "$ICON" --out "$ICONSET/icon_${s}x${s}@2x.png" >/dev/null
done
iconutil -c icns "$ICONSET" -o "$DIST/$APP/Contents/Resources/AppIcon.icns"
rm -rf "$TMP"

cp packaging/macos/Info.plist "$DIST/$APP/Contents/Info.plist"

echo "==> Firmando…"
SIGN_IDENTITY="${SIGN_IDENTITY:--}"
if command -v codesign >/dev/null 2>&1; then
	if [ "$SIGN_IDENTITY" = "-" ]; then
		codesign --force --deep --sign - "$DIST/$APP" >/dev/null 2>&1 &&
			echo "Firmado ad-hoc: OK" ||
			echo "AVISO: no se pudo firmar ad-hoc (¿codesign?)" >&2
	else
		codesign --force --options runtime --timestamp --sign "$SIGN_IDENTITY" "$DIST/$APP" ||
			{ echo "ERROR: falló la firma con '$SIGN_IDENTITY'." >&2; exit 1; }
		echo "Firmado con: $SIGN_IDENTITY"
	fi
fi

if command -v ditto >/dev/null 2>&1 && [ "${SKIP_ZIP:-0}" != "1" ]; then
	rm -f "$DIST/$APP_NAME.zip"
	ditto -c -k --sequesterRsrc --keepParent "$DIST/$APP" "$DIST/$APP_NAME.zip"
	echo "ZIP: $DIST/$APP_NAME.zip"
fi

echo "OK: $DIST/$APP"
du -sh "$DIST/$APP" 2>/dev/null || true
