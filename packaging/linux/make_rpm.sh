#!/usr/bin/env bash
# Construye el paquete .rpm de netflow-probe (x86_64). Ejecutar EN Fedora, con
# rpm-build instalado. Usa los binarios Linux ya compilados en dist/linux
# (compílalos en cualquier Linux con cgo: scripts/build-linux.sh o make linux-gui).
#
# Uso: packaging/linux/make_rpm.sh   (o `make linux-rpm`)
set -euo pipefail

cd "$(dirname "$0")/../.."

VERSION="$(cat VERSION)"
BIN_DIR="${BIN_DIR:-dist/linux}"
ARCH="${ARCH:-x86_64}"

command -v rpmbuild >/dev/null 2>&1 || {
	echo "Falta rpmbuild. En Fedora: sudo dnf install -y rpm-build" >&2
	exit 1
}
if [ ! -x "$BIN_DIR/netflow-probe-gui" ] || [ ! -x "$BIN_DIR/netflow-probe" ]; then
	echo "Faltan los binarios en $BIN_DIR (compílalos en Linux con make linux-gui)." >&2
	exit 1
fi

TOP="$PWD/dist/rpm-top"
rm -rf "$TOP"
mkdir -p "$TOP/SOURCES" "$TOP/SPECS" "$TOP/BUILD" "$TOP/RPMS" "$TOP/SRPMS" dist

cp "$BIN_DIR/netflow-probe-gui" "$BIN_DIR/netflow-probe" "$TOP/SOURCES/"
cp packaging/linux/netflow-probe.desktop packaging/linux/netflow-probe.png "$TOP/SOURCES/"
cp LICENSE "$TOP/SOURCES/"
cp examples/netflow-probe.yaml "$TOP/SOURCES/netflow-probe.yaml"
sed "s/__VERSION__/$VERSION/g" packaging/linux/netflow-probe.spec > "$TOP/SPECS/netflow-probe.spec"

rpmbuild -bb "$TOP/SPECS/netflow-probe.spec" \
	--define "_topdir $TOP" \
	--define "debug_package %{nil}" \
	--define "__os_install_post %{nil}"

find "$TOP/RPMS" -name '*.rpm' -exec cp {} dist/ \;
echo "RPM:"
ls -lh dist/netflow-probe-"$VERSION"-*.rpm
