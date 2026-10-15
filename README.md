# netflow-probe

Agente ligero y multiplataforma que **captura el tráfico del propio host y lo
exporta como NetFlow v9** a un colector externo. Su objetivo principal es el
colector NetFlow de **Pandora FMS** (`nfcapd`, UDP/9995) y su plataforma
prioritaria es **Windows**.

- Binario único, **puro Go, `CGO_ENABLED=0`** (cross-compila a Windows/Linux/macOS).
- Captura en Windows vía **WinDivert** (driver embebible, sin Npcap).
- Captura en Linux vía **AF_PACKET** y en macOS vía **`/dev/bpf`**.
- Exporta **NetFlow v9** (IPv4 e IPv6) por UDP.
- Pensado para acabar teniendo una **interfaz de bandeja** para usuarios no técnicos.

> Estado: **esqueleto funcional**. El pipeline (decodificación → agregación →
> export NetFlow v9) está implementado y probado; la GUI de bandeja, el
> servicio/instalador de Windows y los filtros BPF en kernel son hoja de ruta.
> Ver [CHANGELOG.md](CHANGELOG.md).

## Por qué Go (y no softflowd / pmacct)

El requisito 80 % Windows con interfaz gráfica descarta las herramientas C
clásicas (ni softflowd ni pmacct corren en Windows, y habría que envolverlas en
una GUI igualmente). El análisis completo, con las alternativas y los motivos,
está en [docs/decisiones.md](docs/decisiones.md) y
[docs/investigacion.md](docs/investigacion.md).

## Compilar

```sh
make build          # binario para tu plataforma
make test           # tests unitarios
make dist           # matriz completa Windows/Linux/macOS (amd64/arm64)
make windows        # solo el .exe de Windows
```

Requiere Go 1.27+ y **no** requiere cgo ni libpcap.

## Usar

```sh
# Replay de un fichero pcap (sin privilegios), solo para ver los flujos:
./netflow-probe -read captura.pcap -dry-run

# Captura en vivo en macOS/Linux (requiere root) y export a Pandora:
sudo ./netflow-probe -iface en0 -collector 192.168.1.10 -port 9995

# Con fichero de configuración:
sudo ./netflow-probe -config examples/netflow-probe.yaml
```

En **Windows** hay que ejecutar como administrador (el driver de WinDivert lo
exige). Ver [docs/instalacion-windows.md](docs/instalacion-windows.md).

### Flags

| Flag | Descripción |
|---|---|
| `-config` | Ruta al YAML de configuración. |
| `-iface` | Interfaz de captura (ignorada en Windows). |
| `-collector` | Host del colector NetFlow. |
| `-port` | Puerto UDP del colector (Pandora: `9995`). |
| `-read` | Leer de un pcap/pcapng en vez de capturar en vivo. |
| `-dry-run` | Decodificar y agregar sin enviar NetFlow. |
| `-flush-interval` | Cada cuánto caducar y exportar flujos (def. `5s`). |
| `-version` | Muestra la versión. |

## Arquitectura

```
captura (AF_PACKET / BPF / WinDivert) ─▶ decodificación 5-tupla ─▶ tabla de
flujos (timeouts) ─▶ exportador NetFlow v9 (UDP) ─▶ Pandora FMS :9995
```

Detalle de paquetes y flujo de datos en [docs/arquitectura.md](docs/arquitectura.md).

## GUI de macOS (prueba de concepto)

Interfaz de escritorio (Fyne) para usuarios no técnicos: elegir interfaz y
colector, Iniciar/Detener y registro. Incluye el flujo de **permisos de macOS**
(abre Ajustes del Sistema y permite conceder acceso a `/dev/bpf`).

```sh
make macapp
open "dist/Pandora FMS Netflow Probe.app"

# Instalador DMG (arrastrar a Aplicaciones), con dmgbuild:
make dmg
open "dist/Pandora FMS Netflow Probe.dmg"
```

Ver [docs/gui-macos.md](docs/gui-macos.md). Requiere cgo y Xcode CLT; el resto
del proyecto sigue siendo puro Go.

La **misma app compila para Windows y Linux**:
- Windows: GUI + bandeja + auto-elevación; ver [docs/gui-windows.md](docs/gui-windows.md).
  El agente también funciona como **servicio de Windows** (`netflow-probe.exe -install -config …`).
- Linux: GUI + bandeja + `CAP_NET_RAW`; ver [docs/gui-linux.md](docs/gui-linux.md).
  Servicio **systemd** con `AmbientCapabilities=CAP_NET_RAW` (`netflow-probe -install -config …`).

## Licencia

GPL-2.0. Ver [LICENSE](LICENSE). Nota: el backend de Windows usa WinDivert, que
es dual LGPL-3.0 / GPL-2.0 (por eso esta elección de licencia encaja).
