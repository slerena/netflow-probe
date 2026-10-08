#!/usr/bin/env bash
# Builds "Pandora FMS Netflow Probe.app" (macOS bundle) with the Pandora icon.
# Requires cgo + the Apple toolchain (Fyne uses Cocoa/OpenGL).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

APP_DISPLAY="Pandora FMS Netflow Probe"
EXE="NetFlowProbe"
DIST="$ROOT/dist"
APP="$DIST/$APP_DISPLAY.app"

echo "==> Compilando GUI (macOS, cgo)…"
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o "$APP/Contents/MacOS/$EXE" ./cmd/netflow-probe-gui

echo "==> Copiando icono…"
cp "$ROOT/packaging/macos/AppIcon.icns" "$APP/Contents/Resources/AppIcon.icns"

echo "==> Escribiendo Info.plist…"
cat > "$APP/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key><string>Pandora FMS Netflow Probe</string>
	<key>CFBundleDisplayName</key><string>Pandora FMS Netflow Probe</string>
	<key>CFBundleIdentifier</key><string>es.artica.netflowprobe</string>
	<key>CFBundleVersion</key><string>0.1.0</string>
	<key>CFBundleShortVersionString</key><string>0.1.0</string>
	<key>CFBundlePackageType</key><string>APPL</string>
	<key>CFBundleExecutable</key><string>NetFlowProbe</string>
	<key>CFBundleIconFile</key><string>AppIcon</string>
	<key>LSMinimumSystemVersion</key><string>11.0</string>
	<key>NSHighResolutionCapable</key><true/>
	<key>NSPrincipalClass</key><string>NSApplication</string>
</dict>
</plist>
PLIST

echo "==> Firma ad-hoc (para pruebas locales)…"
codesign --force --deep --sign - "$APP" >/dev/null 2>&1 || echo "  (codesign no disponible; se ejecutará igual)"

echo "==> Listo: $APP"
echo "    Ábrelo con: open \"$APP\""
