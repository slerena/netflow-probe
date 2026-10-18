#!/usr/bin/env bash
# Builds the Linux deliverables (GUI with cgo + CLI/service). Run this ON Linux.
#
# Build dependencies (Fyne uses cgo + OpenGL/X11/Wayland):
#   Debian/Ubuntu: sudo apt-get install -y gcc libgl1-mesa-dev xorg-dev libxkbcommon-dev libwayland-dev
#   Fedora/Rocky:  sudo dnf install -y gcc mesa-libGL-devel libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel wayland-devel libxkbcommon-devel
#   Arch:          sudo pacman -S --needed base-devel mesa libx11 libxcursor libxrandr libxinerama libxi wayland libxkbcommon
#
# Runtime permission for capture: CAP_NET_RAW (see docs/gui-linux.md), e.g.
#   sudo setcap cap_net_raw+ep dist/netflow-probe-gui
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

VERSION="${VERSION:-1.0.34}"
DIST="$ROOT/dist"
mkdir -p "$DIST"

echo "==> Compilando la GUI (cgo)"
CGO_ENABLED=1 go build -trimpath -ldflags "-s -w -X main.version=$VERSION" \
    -o "$DIST/netflow-probe-gui" ./cmd/netflow-probe-gui

echo "==> Compilando el agente/servicio (sin cgo)"
CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$VERSION" \
    -o "$DIST/netflow-probe" ./cmd/netflow-probe

echo "==> Copiando recursos de escritorio"
cp packaging/linux/netflow-probe.desktop "$DIST/" || true
cp packaging/linux/netflow-probe.png "$DIST/" || true

echo "==> Listo: $DIST"
echo "    Permiso de captura: sudo setcap cap_net_raw+ep \"$DIST/netflow-probe-gui\""
