#!/bin/sh
# Instalador manual de Pandora FMS Netflow Probe (Linux).
#
# Uso:
#   sudo ./install.sh              -> /usr/local (sistema; aplica CAP_NET_RAW)
#   ./install.sh --user            -> ~/.local (sin root)
#   sudo ./install.sh --uninstall  -> desinstala de /usr/local
#   ./install.sh --user --uninstall
#
# La configuración del usuario (~/.netflow-probe.yaml) nunca se toca.
set -e

cd "$(dirname "$0")"

PREFIX="/usr/local"
MODE="install"
for arg in "$@"; do
	case "$arg" in
		--user)
			PREFIX="$HOME/.local"
			;;
		--uninstall)
			MODE="uninstall"
			;;
		-h|--help)
			echo "Uso: sudo ./install.sh | ./install.sh --user | ./install.sh [--user] --uninstall"
			exit 0
			;;
		*)
			echo "Opción desconocida: $arg" >&2
			exit 1
			;;
	esac
done

BIN="$PREFIX/bin"
APPS="$PREFIX/share/applications"
ICONS="$PREFIX/share/icons/hicolor/512x512/apps"
DOC="$PREFIX/share/doc/netflow-probe"
SHARE="$PREFIX/share/netflow-probe"

refresh_caches() {
	if command -v update-desktop-database >/dev/null 2>&1; then
		update-desktop-database -q "$APPS" 2>/dev/null || true
	fi
	if command -v gtk-update-icon-cache >/dev/null 2>&1; then
		gtk-update-icon-cache -q -t -f "$PREFIX/share/icons/hicolor" 2>/dev/null || true
	fi
}

if [ "$MODE" = "uninstall" ]; then
	rm -f "$BIN/netflow-probe-gui" "$BIN/netflow-probe" \
		"$APPS/netflow-probe.desktop" \
		"$ICONS/netflow-probe.png" \
		"$DOC/copyright" \
		"$SHARE/netflow-probe.yaml.example"
	rmdir "$DOC" "$SHARE" 2>/dev/null || true
	refresh_caches
	echo "Desinstalado de $PREFIX (config ~/.netflow-probe.yaml conservada)."
	exit 0
fi

mkdir -p "$BIN" "$APPS" "$ICONS" "$DOC" "$SHARE"

install -m755 netflow-probe-gui "$BIN/netflow-probe-gui"
install -m755 netflow-probe "$BIN/netflow-probe"
install -m644 netflow-probe.png "$ICONS/netflow-probe.png"
install -m644 netflow-probe.yaml.example "$SHARE/netflow-probe.yaml.example"
install -m644 LICENSE "$DOC/copyright"

# Lanzador con ruta absoluta (funciona con /usr/local y con ~/.local).
sed "s|^Exec=.*|Exec=$BIN/netflow-probe-gui|" netflow-probe.desktop > "$APPS/netflow-probe.desktop"
chmod 644 "$APPS/netflow-probe.desktop"

if [ "$(id -u)" = "0" ]; then
	if command -v setcap >/dev/null 2>&1; then
		setcap cap_net_raw+ep "$BIN/netflow-probe-gui" 2>/dev/null || true
		setcap cap_net_raw+ep "$BIN/netflow-probe" 2>/dev/null || true
		echo "CAP_NET_RAW aplicado a los binarios."
	else
		echo "AVISO: no hay setcap (instala libcap) para capturar sin root." >&2
	fi
else
	echo "AVISO: instalación de usuario; para capturar sin root ejecuta:" >&2
	echo "  sudo setcap cap_net_raw+ep $BIN/netflow-probe-gui" >&2
fi

refresh_caches
echo "Instalado en $PREFIX. Abre 'Pandora FMS Netflow Probe' desde el menú."
