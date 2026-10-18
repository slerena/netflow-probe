# GUI de Linux (prueba de concepto)

La misma aplicación Fyne que en macOS/Windows, compilada para Linux. Ventana,
**bandeja**, Iniciar/Detener, estado en tiempo real y flujo de permisos
(`CAP_NET_RAW`).

> Hay que compilarla **en Linux**: Fyne usa cgo y no se cross-compila bien desde
> macOS (necesitaría el toolchain C de Linux con las cabeceras de X11/Wayland/GL).

## Dependencias de compilación

| Distro | Paquetes |
|---|---|
| Debian / Ubuntu | `gcc libgl1-mesa-dev xorg-dev libxkbcommon-dev libwayland-dev` |
| Fedora / Rocky | `gcc mesa-libGL-devel libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel wayland-devel libxkbcommon-devel` |
| Arch | `base-devel mesa libx11 libxcursor libxrandr libxinerama libxi wayland libxkbcommon` |

Además, Go 1.27+ y un servidor gráfico (X11 o Wayland) con driver OpenGL.

## Compilar

```sh
scripts/build-linux.sh
```

Genera en `dist/`:

| Salida | Qué es |
|---|---|
| `netflow-probe-gui` | GUI Fyne (cgo). |
| `netflow-probe` | Agente y servicio systemd (sin cgo). |
| `netflow-probe.desktop`, `netflow-probe.png` | Recursos de escritorio. |

## Permisos (CAP_NET_RAW)

En Linux **no hay ningún diálogo del SO** para esto: la captura con `AF_PACKET`
requiere la capacidad **`CAP_NET_RAW`**. Dos formas:

1. **Al binario** (recomendado para uso de escritorio). Necesita `libcap`
   (`libcap2-bin` en Debian/Ubuntu, `libcap` en Fedora/Rocky):

   ```sh
   sudo setcap cap_net_raw+ep dist/netflow-probe-gui
   ```

   La GUI ofrece un botón **«Conceder acceso a la captura (setcap)»** que ejecuta
   esto con `pkexec` (o `sudo`) y reintenta sola.

2. **Ejecutar como root** (menos recomendable).

Nota: el `setcap` se pierde si mueves o recompilas el binario; hay que repetirlo.

## Bandeja

`fyne.io/systray` en Linux usa **D-Bus (StatusNotifierItem)**, sin cgo ni GTK.
Funciona en **KDE, XFCE, Cinnamon, MATE**… En **GNOME** necesita la **extensión
AppIndicator/KStatusNotifierItem** (GNOME quitó la bandeja por defecto). Si no
hay host de bandeja, la ventana y todo lo demás siguen funcionando.

## Servicio systemd (sin GUI, sin root)

El agente se instala como servicio con `CAP_NET_RAW` propio (el unit que genera
el proyecto incluye `AmbientCapabilities=CAP_NET_RAW`):

```sh
sudo ./dist/netflow-probe -install -config /etc/netflow-probe.yaml
sudo ./dist/netflow-probe -uninstall
```

Arranca solo, sin consola y **sin ejecutarse como root** (solo esa capacidad).
Alternativamente, si usas el unit generado a mano, añade:

```ini
[Service]
AmbientCapabilities=CAP_NET_RAW
CapabilityBoundingSet=CAP_NET_RAW
```

## Instalarlo como aplicación de escritorio

```sh
sudo install -Dm755 dist/netflow-probe-gui /usr/local/bin/netflow-probe-gui
sudo install -Dm644 dist/netflow-probe.png /usr/share/icons/hicolor/512x512/apps/netflow-probe.png
sudo install -Dm644 dist/netflow-probe.desktop /usr/share/applications/netflow-probe.desktop
sudo setcap cap_net_raw+ep /usr/local/bin/netflow-probe-gui
```

## Instalador `.deb` (Ubuntu/Debian)

La forma recomendada en Ubuntu es el paquete `.deb` (instala en `/usr/bin`, el
menú, el icono y aplica `CAP_NET_RAW` solo):

```sh
# En Linux, con las dependencias de Fyne instaladas:
make linux-deb          # -> dist/netflow-probe_<ver>_amd64.deb

sudo apt install ./dist/netflow-probe_<ver>_amd64.deb
# o: sudo dpkg -i dist/netflow-probe_<ver>_amd64.deb

netflow-probe-gui       # desde el menú "Pandora FMS Netflow Probe"
sudo apt remove netflow-probe
```

El `postinst` ejecuta `setcap cap_net_raw+ep` sobre `netflow-probe-gui` y
`netflow-probe`; por eso el paquete depende de `libcap2-bin`. El CLI puede
instalarse como servicio con `sudo netflow-probe -install -config …`.

## Instalador `.rpm` (Fedora)

En Fedora, con `rpm-build` instalado y los binarios ya compilados en
`dist/linux`:

```sh
sudo dnf install -y rpm-build
make linux-rpm          # -> dist/netflow-probe-<ver>-1.fcXX.x86_64.rpm

sudo dnf install ./dist/netflow-probe-<ver>-1.fcXX.x86_64.rpm
netflow-probe-gui       # desde el menú "Pandora FMS Netflow Probe"
sudo dnf remove netflow-probe
```

El RPM instala lo mismo que el `.deb` (GUI, CLI, menú, icono) y su `%post`
aplica `CAP_NET_RAW`; depende de `libcap`, `libglvnd-glx`, `libX11` y
`libwayland-client`. Se puede construir en Fedora a partir de los binarios
compilados en cualquier Linux (misma arquitectura).

## Instalación a pelo (`.tar.gz`)

Paquete portable con binarios, `.desktop`, icono, config de ejemplo e
`install.sh`:

```sh
make linux-tar          # -> dist/netflow-probe-<ver>-linux-amd64.tar.gz
tar -xzf dist/netflow-probe-<ver>-linux-amd64.tar.gz
cd netflow-probe-<ver>-linux-amd64

sudo ./install.sh              # sistema: /usr/local + CAP_NET_RAW
./install.sh --user            # usuario: ~/.local (sin root)
sudo ./install.sh --uninstall  # desinstalar (o --user --uninstall)
```

`install.sh` reescribe el `Exec` del lanzador con la ruta absoluta, actualiza
las cachés de menú/iconos y no toca `~/.netflow-probe.yaml`.

## Empaquetado (opcional)

`fyne package -os linux` genera un `.tar.xz`/AppImage con icono y `.desktop`; o
usa `nfpm` para `.deb`/`.rpm`. Queda como trabajo de empaquetado.

## Limitaciones de la PoC

- Hay que compilar y probar en Linux (no desde macOS).
- `filter` (BPF) no se expone aún en la GUI; el backend AF_PACKET captura todo.
- El `setcap` no persiste si cambias el binario de sitio.
