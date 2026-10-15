# Configuración de dmgbuild para "Pandora FMS Netflow Probe.dmg".
# Lo usa packaging/macos/make_dmg.sh, que se ejecuta desde la RAÍZ del repo: por
# eso las rutas son relativas (dmgbuild hace exec() de este fichero y no define
# __file__). Ver https://dmgbuild.readthedocs.io/
import os

APP = "dist/Pandora FMS Netflow Probe.app"
DEST = "Aplicaciones"

# Contenido del volumen: la app y un enlace a /Applications para arrastrarla.
files = [APP]
symlinks = {DEST: "/Applications"}

# Icono del volumen (el propio icono de la app).
badge_icon = os.path.join(APP, "Contents", "Resources", "AppIcon.icns")

# Fondo con degradado + título + flecha (dmg-bg.png / dmg-bg@2x.png = Retina).
background = "packaging/macos/dmg-resources/dmg-bg.png"

# Posiciones (en puntos, dentro de la ventana de 640x420).
icon_locations = {
    "Pandora FMS Netflow Probe.app": (170, 210),
    DEST: (470, 210),
}

format = "UDZO"                 # comprimido
window_rect = ((220, 140), (640, 420))
default_view = "icon-view"
show_status_bar = False
show_tab_view = False
show_toolbar = False
show_pathbar = False
show_sidebar = False
icon_size = 128
text_size = 13
