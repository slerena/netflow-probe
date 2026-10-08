# GUI de macOS (prueba de concepto)

App de escritorio (Fyne) que envuelve el motor de captura para que un usuario no
técnico pueda elegir la interfaz y el colector, arrancar/parar y ver el registro.
Primera versión **solo macOS** (es lo que hay para la PoC). Requiere **cgo** y el
toolchain de Apple (Fyne usa Cocoa/OpenGL).

## Compilar y abrir

```sh
make macapp
open "dist/Pandora FMS Netflow Probe.app"
```

`make macapp` ejecuta `scripts/build-macos-app.sh`, que:
1. compila `cmd/netflow-probe-gui` con cgo,
2. monta `dist/Pandora FMS Netflow Probe.app` con `Info.plist`,
3. copia el icono `packaging/macos/AppIcon.icns` (icono del agente Pandora),
4. firma **ad-hoc** para pruebas locales.

> En el resto de plataformas la ruta GUI compila a un *stub* que avisa de que la
> interfaz solo está en macOS. Así el proyecto sigue cross-compilando sin cgo.

## Uso

1. **Interfaz**: la lista muestra solo interfaces **conectadas** (con una IP
   utilizable, no link-local), con descripción y dirección, p. ej.
   `Wi-Fi (en0) · 172.20.10.9` o `VPN (utun13) · 100.99.221.4`. Se ocultan las
   internas de macOS (`anpi`, `awdl`, `llw`, `bridge`, `utun` sin IP…).
2. **Colector NetFlow**: IP/host del servidor Pandora. **Puerto**: `9995`.
3. Pulsa **Iniciar**. Mientras captura verás la **barra verde** (spinner) y el
   estado en tiempo real: `Capturando · N paquetes · X MB · M flujos activos ·
   K exportados`, refrescado cada segundo.
4. **Detener** vuelca los flujos pendientes y para limpiamente.
5. **Guardar configuración** escribe `~/.netflow-probe.yaml`.

## Permisos (lo importante)

Capturar tráfico en macOS exige poder abrir los dispositivos **`/dev/bpf*`**
(propiedad de `root`, modo `0600`). **No hay ninguna casilla en Ajustes del
Sistema para esto**: Acceso total al disco y demás permisos TCC no aplican a la
captura, por eso la app **no aparece en ninguna lista de Privacidad y seguridad**.

Si la app no consigue capturar, abre el diálogo **"Permisos de captura"** con una
única acción:

- **Conceder acceso a la captura** → pide tu contraseña de administrador (aviso
  nativo de macOS) e instala un pequeño **LaunchDaemon**
  (`/Library/LaunchDaemons/es.artica.netflowprobe.chmodbpf.plist`) que mantiene el
  acceso a `/dev/bpf*` también tras reiniciar; además aplica el permiso de
  inmediato y **reintenta la captura solo**.

### Endurecimiento pendiente

El acceso se concede con `chmod o+rw /dev/bpf*` (PoC: habilita "other"). Lo
estricto sería un grupo dedicado (`access_bpf`) como hace Wireshark. Queda como
trabajo de endurecimiento.

## Bandeja (barra de menús)

La app pone un **icono en la barra de menús** (glifo Pandora en gris) con este
menú: **Mostrar / ocultar**, **Iniciar captura**, **Detener captura** y **Salir**.
Cerrar la ventana **no cierra la app**: la oculta a la bandeja. Para terminar,
usa **Salir** en el menú de la bandeja (o Cmd+Q).

## Limitaciones de la PoC

- Solo macOS; el resto de plataformas usa el CLI.
- Firma **ad-hoc**, no notarizada: la primera vez macOS puede bloquearla
  (clic derecho → **Abrir**, o *Ajustes → Privacidad y seguridad → Abrir de todos
  modos*).
- Sin filtro BPF en la GUI todavía (captura todo y agrega en user-space).
- El icono es el del agente Pandora RC (`integria_enterprise`), reutilizado aquí.

## Diagnóstico

Prueba la captura sin abrir la interfaz (mismo binario del bundle):

```sh
"dist/Pandora FMS Netflow Probe.app/Contents/MacOS/NetFlowProbe" -selftest en0
```

Debe imprimir `OK: captura operativa en en0 — N paquetes …`. Si falla, muestra
el error y el remedio (`sudo chmod o+rw /dev/bpf*`).

## Recursos

| Recurso | Ruta |
|---|---|
| Código GUI (macOS) | `cmd/netflow-probe-gui/main_gui.go` |
| Stub otras plataformas | `cmd/netflow-probe-gui/main_stub.go` |
| Icono en runtime | `cmd/netflow-probe-gui/icon.png` |
| Icono del bundle | `packaging/macos/AppIcon.icns` |
| Script de empaquetado | `scripts/build-macos-app.sh` |
