#!/usr/bin/env bash
# Construye el paquete .deb de netflow-probe (amd64). Ejecutar EN Linux
# (Ubuntu/Debian), donde estén las dependencias de Fyne (cgo).
#
# Uso: packaging/linux/make_deb.sh   (o `make linux-deb`)
# Instalación: sudo apt install ./dist/netflow-probe_<ver>_amd64.deb
set -euo pipefail

cd "$(dirname "$0")/../.."

VERSION="$(cat VERSION)"
ARCH="${ARCH:-amd64}"
BIN_DIR="dist/linux"
DEB="dist/netflow-probe_${VERSION}_${ARCH}.deb"
mkdir -p "$BIN_DIR" dist

echo "==> Compilando GUI (cgo)"
CGO_ENABLED=1 go build -trimpath -ldflags "-s -w -X main.version=$VERSION" \
	-o "$BIN_DIR/netflow-probe-gui" ./cmd/netflow-probe-gui

echo "==> Compilando CLI (sin cgo)"
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$VERSION" \
	-o "$BIN_DIR/netflow-probe" ./cmd/netflow-probe

STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT

mkdir -p "$STAGE/DEBIAN" \
	"$STAGE/usr/bin" \
	"$STAGE/usr/share/applications" \
	"$STAGE/usr/share/icons/hicolor/512x512/apps" \
	"$STAGE/usr/share/doc/netflow-probe" \
	"$STAGE/usr/share/netflow-probe"

install -m755 "$BIN_DIR/netflow-probe-gui" "$STAGE/usr/bin/netflow-probe-gui"
install -m755 "$BIN_DIR/netflow-probe" "$STAGE/usr/bin/netflow-probe"
install -m644 packaging/linux/netflow-probe.desktop "$STAGE/usr/share/applications/netflow-probe.desktop"
install -m644 packaging/linux/netflow-probe.png "$STAGE/usr/share/icons/hicolor/512x512/apps/netflow-probe.png"
install -m644 LICENSE "$STAGE/usr/share/doc/netflow-probe/copyright"
install -m644 examples/netflow-probe.yaml "$STAGE/usr/share/netflow-probe/netflow-probe.yaml.example"

cat > "$STAGE/DEBIAN/control" <<EOF
Package: netflow-probe
Version: $VERSION
Section: net
Priority: optional
Architecture: $ARCH
Maintainer: Pandora FMS <info@pandorafms.com>
Depends: libc6, libgl1, libx11-6, libwayland-client0, libcap2-bin
Homepage: https://github.com/slerena/netflow-probe
Description: Pandora FMS NetFlow Probe
 Ligero agente que captura el trafico del propio equipo y lo exporta como
 NetFlow v9 al colector de Pandora FMS (nfcapd, UDP/9995).
EOF

cat > "$STAGE/DEBIAN/postinst" <<'EOF'
#!/bin/sh
set -e
# Captura AF_PACKET sin ejecutar como root.
if command -v setcap >/dev/null 2>&1; then
  setcap cap_net_raw+ep /usr/bin/netflow-probe-gui || true
  setcap cap_net_raw+ep /usr/bin/netflow-probe || true
fi
if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database -q /usr/share/applications || true
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
  gtk-update-icon-cache -q -t -f /usr/share/icons/hicolor || true
fi
exit 0
EOF

cat > "$STAGE/DEBIAN/postrm" <<'EOF'
#!/bin/sh
set -e
if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database -q /usr/share/applications || true
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
  gtk-update-icon-cache -q -t -f /usr/share/icons/hicolor || true
fi
exit 0
EOF
chmod 755 "$STAGE/DEBIAN/postinst" "$STAGE/DEBIAN/postrm"

rm -f "$DEB"
dpkg-deb --build --root-owner-group "$STAGE" "$DEB"

echo "DEB: $DEB"
dpkg-deb --info "$DEB" | head -20
