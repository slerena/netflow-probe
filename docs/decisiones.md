# Decisiones de diseño

Registro de las decisiones tomadas al arrancar `netflow-probe`, con el contexto
y las alternativas descartadas. Orden cronológico del análisis.

## Contexto y requisitos

- **Ligero**: solo capturar el tráfico del propio host y exportarlo.
- **Colector destino**: NetFlow de **Pandora FMS**.
- **Usuarios no técnicos**: debe ser fácil de instalar y operar; hará falta GUI.
- **Plataformas**: macOS, Windows y Linux; **target principal Windows (~80 %)**.
- Se acepta compilar por plataforma y ejecutar con privilegios de root/admin.

## D1 — Lenguaje y runtime: **Go**

**Decisión**: implementar el agente en Go, un único binario puro.

**Motivos**:
- Windows es el 80 % del target y las herramientas C de referencia
  (**softflowd**, **pmacct**) **no corren en Windows** (POSIX + autotools +
  `fork`/`daemon`/señales). Portarlas sería un proyecto por sí mismo.
- Go permite **un solo código fuente** con backends por SO, cross-compilación
  trivial con `CGO_ENABLED=0`, y un binario único fácil de empaquetar.
- La GUI para usuarios no técnicos es mucho más sencilla en Go (Wails/Fyne/tray).
- El alcance real es pequeño (capturar → agregar → exportar), no una suite.

**Alternativas descartadas**:
- **Fork de softflowd**: licencia BSD y código muy pequeño y legible, pero sin
  Windows y habría que escribirle una GUI encima (más trabajo que Go).
- **pmacct** (`pmacctd`): enormemente capaz (IPFIX, sFlow, BGP, nDPI, backends
  SQL/Kafka), GPLv2 (compatible con liberar como GPL-2.0), pero **sin Windows**,
  y su tamaño lo hace poco apropiado para forkear. WSL2 no sirve para capturar
  el NIC físico del host (VM con NAT).

## D2 — Captura por SO: **backends puros Go, sin cgo**

**Decisión**: `github.com/gopacket/gopacket` para decodificar y:
- **Linux**: `gopacket/afpacket` (AF_PACKET, puro Go).
- **macOS**: `gopacket/bsdbpf` (`/dev/bpf`, puro Go).
- **Windows**: **WinDivert** vía `github.com/imgk/divert-go` (puro Go, syscall).
- **Replay**: `gopacket/pcapgo` (pcap/pcapng, todas las plataformas).

**Motivos**: mantiene `CGO_ENABLED=0` en las tres plataformas (sin libpcap-dev
ni SDKs C), permite cross-compilar a Windows desde el Mac y hace el build
reproducible. WinDivert entrega paquetes IP desnudos a nivel de red, que
`gopacket` decodifica con `layers.LinkTypeRaw`.

**Nota**: se verificó que `layers.LinkTypeRaw` decodifica IPv4 e IPv6, y que
`layers.TCP` expone los flags como campos booleanos (se empaquetan a byte para
el campo `TCP_FLAGS` de NetFlow).

## D3 — Captura en Windows: **WinDivert** (no Npcap)

**Decisión**: WinDivert.

| Opción | Redistribución | Nivel | Veredicto |
|---|---|---|---|
| Npcap | licencia OEM desde ~39.980 € (la gratis no permite redistribuir) | L2 | descartado por coste |
| **WinDivert** | gratis bajo **GPL-2.0** (o LGPL-3.0, o 500 €/año comercial) | L3 (IP) | **elegido** |
| ETW / IP Helper | 0 € | estadísticas por conexión | plan B, más frágil |

Para NetFlow solo se necesita la 5-tupla IP, así que L3 es suficiente. WinDivert
tiene driver **pre-firmado y embebible**, soporta IPv6 y loopback, y encaja con
la licencia GPL-2.0 del proyecto.

**Pendiente**: embeber el driver en el binario de release, instalarlo como
**servicio** (`LocalSystem`) para que el usuario no vea UAC tras la instalación.

## D4 — Formato de export: **NetFlow v9** a `nfcapd :9995`

**Decisión**: NetFlow v9 por UDP al colector de Pandora.

**Motivos**:
- Pandora FMS usa **`nfcapd` (nfdump)** escuchando en **9995/UDP**, que acepta
  NetFlow v5/v9 e IPFIX. La doc de Pandora recomienda como sonda `pmacctd` con
  `nfprobe_version: 9`.
- v9 soporta **IPv6** mediante templates (v5 es solo IPv4).
- IPFIX no aporta nada para este destino y añade complejidad; queda fuera.

**Implementación**: dos plantillas (IPv4 e ID 256, IPv6 e ID 257) con los campos
`IN_BYTES`, `IN_PKTS`, `PROTOCOL`, `SRC_TOS`, `TCP_FLAGS`, `L4_SRC_PORT`,
`IP*_SRC_ADDR`, `L4_DST_PORT`, `IP*_DST_ADDR`, `LAST_SWITCHED`, `FIRST_SWITCHED`.
Las plantillas se refrescan cada `template_refresh` paquetes de export.

## D5 — Semántica de agregación

- Flujos **unidireccionales** por 5-tupla + ToS (como softflowd/most probes).
- **Inactivo** por defecto `15s`, **activo** `30m`.
- **`max_flows`** con desalojo **LRU** para acotar memoria.
- Los flujos se exportan al caducar (ticker) o al parar (volcado completo).

## D6 — Licencia: **GPL-2.0**

**Decisión**: GPL-2.0. Encaja con la dual GPL-2.0 de WinDivert y con el plan
declarado de publicar el proyecto bajo GPL-2.0. (Alternativa: si en el futuro se
quisiera cerrar el producto, WinDivert permite LGPL-3.0 enlazado dinámicamente o
licencia comercial.)

## Huecos abiertos (hoja de ruta)

1. **GUI de bandeja** (Wails o Fyne): elegir interfaz, IP/puerto del colector,
   Start/Stop, autostart. Con `kardianos/service` para el servicio.
2. **Empaquetado Windows**: instalador MSI, elevación, driver WinDivert embebido
   y firma.
3. **Filtros BPF en kernel** en Linux/macOS (hoy se captura todo y se agrega en
   espacio de usuario; en Windows el `filter` se pasa a WinDivert).
4. **Validación end-to-end** contra un nfcapd real (y/o `goflow2` como colector
   de pruebas).
