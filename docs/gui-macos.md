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

macOS exige privilegios para capturar paquetes: hay que poder abrir los
dispositivos **`/dev/bpf*`** (propiedad de `root`, modo `0600`). Sin eso, el
arranque falla.

Si la app no consigue capturar, abre el diálogo **"Permisos de captura"** con dos
acciones:

- **Abrir Ajustes del Sistema** → abre *Ajustes del Sistema → Privacidad y
  seguridad* (pane de **Acceso total al disco**). Se prueban varios esquemas de
  URL porque Apple los cambió en versiones recientes.
- **Conceder acceso a /dev/bpf (admin)** → pide tu contraseña de administrador
  (aviso nativo de macOS) y ejecuta `chmod o+rw /dev/bpf*`. Es un atajo de PoC
  para que la captura funcione ya. Tras concederlo, pulsa **Iniciar** otra vez.

### Ruta recomendada en producción

`chmod o+rw` es amplio (cualquier usuario podría capturar). La forma correcta es
la de Wireshark: un **LaunchDaemon `ChmodBPF`** que crea un grupo dedicado
(`access_bpf`), añade al usuario y ajusta `/dev/bpf*` al grupo. Queda como
trabajo de endurecimiento.

## Limitaciones de la PoC

- Solo macOS; el resto de plataformas usa el CLI.
- Firma **ad-hoc**, no notarizada: la primera vez macOS puede bloquearla
  (clic derecho → **Abrir**, o *Ajustes → Privacidad y seguridad → Abrir de todos
  modos*).
- Sin filtro BPF en la GUI todavía (captura todo y agrega en user-space).
- El icono es el del agente Pandora RC (`integria_enterprise`), reutilizado aquí.

## Recursos

| Recurso | Ruta |
|---|---|
| Código GUI (macOS) | `cmd/netflow-probe-gui/main_gui.go` |
| Stub otras plataformas | `cmd/netflow-probe-gui/main_stub.go` |
| Icono en runtime | `cmd/netflow-probe-gui/icon.png` |
| Icono del bundle | `packaging/macos/AppIcon.icns` |
| Script de empaquetado | `scripts/build-macos-app.sh` |
