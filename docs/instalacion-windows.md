# Instalación en Windows

El agente usa **WinDivert** para capturar. WinDivert es un driver de kernel: hay
que instalarlo y el proceso que captura debe ser **administrador**.

> Estado actual: hay **GUI**, **servicio** y scripts de empaquetado. Ver
> [gui-windows.md](gui-windows.md) para construir la app y el servicio.

## Construir

En Windows, con Go 1.27+ y un compilador C (mingw-w64/TDM-GCC):

```powershell
pwsh scripts\build-windows.ps1      # GUI (.exe sin consola, auto-elevada) + agente/servicio
pwsh scripts\fetch-windivert.ps1    # copia WinDivert.dll + WinDivert64.sys junto al .exe
```

El manifiesto pide `requireAdministrator`, así que Windows muestra el UAC al
abrir la GUI y la captura funciona sin pasos extra.

## Requisitos

- Windows 7 o superior (WinDivert usa WFP).
- **WinDivert** 2.2: `WinDivert.dll` + `WinDivert64.sys` (ya vienen firmados).
  - Para pruebas, se pueden copiar junto al `.exe` o a una ruta de búsqueda.
  - En la versión final, el binario los **embebe** y los extrae/instala en
    tiempo de ejecución (o el instalador los coloca).
- Privilegios de **administrador** para cargar el driver.

## Prueba manual rápida

1. Descarga WinDivert 2.2 y coloca `WinDivert.dll` y `WinDivert64.sys` junto a
   `netflow-probe.exe`.
2. Abre **PowerShell/CMD como administrador**.
3. Ejecuta:

   ```powershell
   .\netflow-probe.exe -collector 192.168.1.10 -port 9995
   ```

   - `-collector` es el host con el colector NetFlow de Pandora (`nfcapd`).
   - El puerto por defecto del colector de Pandora es `9995/UDP`.

4. Comprueba en Pandora FMS (Operation → Views → Network) que aparecen los
   flujos del equipo.

## Filtro

En Windows, `filter` (config o por código) usa el lenguaje de WinDivert:

- `ip or ipv6` — todo el tráfico IP (por defecto).
- `tcp or udp` — solo TCP/UDP.
- `not loopback` — excluye loopback.
- `ip and not net 224.0.0.0/4 and not net 239.0.0.0/8` — excluye multicast.

## Plan de empaquetado (pendiente)

1. **Embeber el driver** con `go:embed` (o usar `divert-go` con
   `-tags divert_rsrc`) para no depender de ficheros externos.
2. **Instalador MSI** que:
   - copie el `.exe` (y firme),
   - registre el agente como **servicio de Windows** (`LocalSystem`), de modo
     que no haya UAC en cada arranque,
   - configure `collector`/`port` y el arranque automático.
3. **Firma de código** del binario (recomendado para evitar SmartScreen).
4. Nota legal: si algún día se empaqueta Npcap en lugar de WinDivert, hará falta
   **licencia OEM** de Nmap; con WinDivert no (GPL-2.0/LGPL-3.0).

## Servicio de Windows

Para capturar sin sesión de usuario ni consola:

```powershell
.\netflow-probe.exe -install -config "C:\ProgramData\Pandora\netflow-probe.yaml"
.\netflow-probe.exe -uninstall
```

## Solución de problemas

- `windivert open ... ¿proceso elevado y driver WinDivert presente?`: falta el
  driver o no se ejecuta como administrador.
- No llegan flujos a Pandora: verifica que el puerto `9995/UDP` está abierto en
  el firewall de salida y que `collector` apunta al host correcto. Para aislar,
  usa `-dry-run` (no envía) y confirma que los flujos se decodifican.
