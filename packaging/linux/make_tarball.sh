#!/usr/bin/env bash
# Construye el .tar.gz portable de Linux (binarios + install.sh). Usa los
# binarios ya compilados en dist/linux (compílalos en Linux con cgo:
# scripts/build-linux.sh o make linux-gui).
#
# Uso: packaging/linux/make_tarball.sh   (o `make linux-tar`)
set -euo pipefail

cd "$(dirname "$0")/../.."

VERSION="$(cat VERSION)"
ARCH="${ARCH:-amd64}"
BIN_DIR="dist/linux"
NAME="netflow-probe-${VERSION}-linux-${ARCH}"
OUT="dist/${NAME}.tar.gz"

if [ ! -x "$BIN_DIR/netflow-probe-gui" ] || [ ! -x "$BIN_DIR/netflow-probe" ]; then
	echo "Faltan los binarios en $BIN_DIR (compílalos en Linux con make linux-gui)." >&2
	exit 1
fi

STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT

DEST="$STAGE/$NAME"
mkdir -p "$DEST"
cp "$BIN_DIR/netflow-probe-gui" "$BIN_DIR/netflow-probe" "$DEST/"
cp packaging/linux/netflow-probe.desktop packaging/linux/netflow-probe.png "$DEST/"
cp examples/netflow-probe.yaml "$DEST/netflow-probe.yaml.example"
cp LICENSE "$DEST/"
cp packaging/linux/install.sh "$DEST/install.sh"
cp packaging/linux/LEEME.txt "$DEST/LEEME.txt"
chmod +x "$DEST/install.sh" "$DEST/netflow-probe-gui" "$DEST/netflow-probe"

mkdir -p dist
rm -f "$OUT"
tar -C "$STAGE" -czf "$OUT" "$NAME"

echo "TAR: $OUT"
ls -lh "$OUT"
tar -tzf "$OUT" | head -20
