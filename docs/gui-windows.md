# GUI de Windows (prueba de concepto)

La misma aplicación Fyne que en macOS, compilada para Windows. Ofrece ventana,
**bandeja (área de notificación)**, Iniciar/Detener, estado en tiempo real y el
flujo de permisos adaptado a Windows (administrador).

## Compilar

En un *Developer PowerShell* en Windows, con **Go 1.27+** y un compilador C para
cgo (mingw-w64 o TDM-GCC en el `PATH`):

```powershell
pwsh scripts\build-windows.ps1
```

Genera en `dist\`:

| Salida | Qué es |
|---|---|
| `Pandora FMS Netflow Probe.exe` | GUI (cgo), sin consola, **auto-elevada** por manifiesto |
| `netflow-probe.exe` | Agente sin interfaz y **servicio** de Windows (sin cgo) |

El script usa `rsrc` (se descarga solo) para incrustar el icono (`AppIcon.ico`) y
el manifiesto (`app.manifest`, con `requireAdministrator`).

## Instalador y ZIP

Además de los binarios sueltos, hay dos paquetes listos para distribuir (ver
`packaging/windows/README.md`):

- **ZIP portable**: `make windows-zip` →
  `dist/Pandora FMS Netflow Probe <ver> (Windows).zip` con la GUI, el agente,
  WinDivert y `swgl/` (Mesa) + `Modo compatible.bat`.
- **Instalador NSIS**: `make windows-installer` →
  `dist/Pandora FMS Netflow Probe Setup <ver>.exe` (licencia, componentes,
  accesos directos y desinstalador). Se compila desde macOS/Linux con
  `makensis` (`brew install makensis`), sin necesidad de Windows.

El payload (`.exe` compilado + WinDivert + Mesa) va en
`packaging/windows/payload/` (no versionado). Ver el README de packaging para
la lista exacta de ficheros.

## WinDivert (captura)

La captura la hace el driver **WinDivert**. El instalador y el ZIP **ya
incluyen** `WinDivert.dll` + `WinDivert64.sys` junto al `.exe`. Si compilas a
mano, cópialos con:

```powershell
pwsh scripts\fetch-windivert.ps1
```

WinDivert es dual **LGPL-3.0 / GPL-2.0** (compatible con la licencia del
proyecto). No confundir con Npcap, que exige licencia OEM de pago para
redistribuir.

## Permisos

Capturar necesita **administrador** (el driver WinDivert es de kernel). Se
resuelve de dos formas:

- El manifiesto del `.exe` pide **`requireAdministrator`**: al abrir la app,
  Windows muestra el UAC una vez y arranca elevada.
- Si por lo que sea no está elevada y la captura falla, el diálogo de
  **Permisos de captura** ofrece **«Reiniciar como administrador»**, que relanza
  el ejecutable con `Start-Process -Verb RunAs`.

No hay que elegir interfaz: WinDivert captura todo el equipo. La GUI lo indica y
oculta el selector.

## Uso

1. **Colector NetFlow (Pandora)**: IP/host del servidor; **Puerto**: `9995`.
2. **Iniciar** → barra verde + contadores (paquetes, MB, flujos activos,
   exportados).
3. **Detener** vuelca los flujos pendientes y para.
4. **Bandeja**: Mostrar/ocultar, Iniciar captura, Detener captura, Salir. Cerrar
   la ventana la oculta a la bandeja.
5. **Guardar configuración**: `%USERPROFILE%\.netflow-probe.yaml`.

## Ejecutarlo como servicio (sin sesión de usuario)

El agente `netflow-probe.exe` puede correr como servicio de Windows (arranca
solo, sin consola y sin depender del usuario):

```powershell
# Instalar (indica la ruta del YAML de configuración)
.\netflow-probe.exe -install -config "C:\ProgramData\Pandora\netflow-probe.yaml"

# Quitar
.\netflow-probe.exe -uninstall
```

El servicio lo gestiona el SCM de Windows (`PandoraFMSNetflowProbe`). Mientras
corre en segundo plano, la GUI puede seguir usándose para configurar/pruebas.

## Limitaciones de la PoC

- Firma **sin certificado**: Windows SmartScreen puede avisar la primera vez.
- La GUI y el servicio comparten el YAML pero son procesos independientes; evita
  capturar con ambos a la vez sobre la misma máquina.
- `filter` (WinDivert) todavía no se expone en la GUI; por defecto `ip or ipv6`.
