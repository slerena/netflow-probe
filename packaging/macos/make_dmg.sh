#!/usr/bin/env bash
# Empaqueta "Pandora FMS Netflow Probe" como DMG de macOS (arrastrar a
# Aplicaciones) con fondo y aviso de licencia (GPL-2.0).
# Uso: packaging/macos/make_dmg.sh   (o `make dmg`)
#
# Requiere dmgbuild (+ Pillow sólo para regenerar el fondo):
#   python3 -m venv .venv && .venv/bin/pip install dmgbuild pillow
#   PYTHON=.venv/bin/python packaging/macos/make_dmg.sh
set -euo pipefail

cd "$(dirname "$0")/../.."

APP_NAME="Pandora FMS Netflow Probe"
APP="$APP_NAME.app"
DMG="$APP_NAME.dmg"
RES="packaging/macos/dmg-resources"
PYTHON="${PYTHON:-python3}"

# 1) .app (firma ad-hoc o con SIGN_IDENTITY si está definida), sin ZIP.
SKIP_ZIP=1 bash packaging/macos/make_app.sh
[ -d "dist/$APP" ] || { echo "No existe dist/$APP tras el empaquetado." >&2; exit 1; }

if ! "$PYTHON" -c "import dmgbuild" 2>/dev/null; then
	echo "Falta dmgbuild en '$PYTHON'. Instálalo, p. ej.:" >&2
	echo "  python3 -m venv .venv && .venv/bin/pip install dmgbuild pillow" >&2
	exit 1
fi
if [ ! -f "$RES/dmg-bg.png" ]; then
	echo "Falta $RES/dmg-bg.png (genéralo con: python3 $RES/make_bg.py)" >&2
	exit 1
fi

# 2) DMG con dmgbuild (fondo, iconos y enlace a Aplicaciones).
rm -f "dist/$DMG"
"$PYTHON" -m dmgbuild -s "$RES/settings.py" "$APP_NAME" "dist/$DMG"

# 3) Aviso de licencia (Aceptar/Rechazar al montar).
"$PYTHON" "$RES/add_license.py" "dist/$DMG"

echo "DMG: dist/$DMG"
du -sh "dist/$DMG"
