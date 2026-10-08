# Notas de investigación (estado a oct. 2026)

Referencias y estado de las herramientas y librerías evaluadas. Se conserva
aquí para no repetir el análisis y para justificar las decisiones.

## Herramientas de referencia (descartadas como base)

### softflowd
- **Qué es**: sonda pcap → NetFlow v1/v5/v9 (+PSAMP), IPv4/IPv6.
- **Versión**: v1.1.1 (feb. 2025). Mantenedor: Hitoshi Irino. ~6k LOC.
- **Licencia**: BSD 2-cláusulas (permisiva; se podría cerrar el producto).
- **Plataformas**: Linux, OpenBSD, macOS (POSIX); Solaris experimental. **Sin
  Windows.**
- **Conclusión**: base atractiva por tamaño/licencia, pero descartada por la
  ausencia de Windows + necesidad de GUI.

### pmacct / pmacctd
- **Qué es**: suite de contabilidad de tráfico. `pmacctd` captura con libpcap y
  exporta NetFlow/IPFIX/sFlow (`nfprobe`/`sfprobe`); `nfacctd`/`sfacctd` son
  colectores; además BGP/BMP/telemetría, nDPI, backends SQL/Kafka.
- **Versión**: 1.7.9 (ago. 2024); 1.8.0 en imágenes Docker recientes.
- **Licencia**: GPLv2 (hay una iniciativa de relicencia a BSD en marcha).
- **Plataformas**: Linux, BSDs, Solaris; macOS de facto (guías con Homebrew).
  **Sin Windows.**
- **Conclusión**: descartada como base (sin Windows, enorme para forkear). Útil
  como referencia funcional y para validar contra ella.

### GoFlow2 (netsampler/goflow2)
- **Qué es**: colector NetFlow/IPFIX/sFlow en Go (no una sonda pcap).
- **Uso aquí**: excelente **colector de pruebas** para validar nuestro
  exportador sin depender de Pandora.

## Captura en Windows

### Npcap
- Driver NDIS + API libpcap. **No es open source.**
- **Redistribución**: requiere licencia **OEM** (perpetua desde ~39.980 € para
  small/startup; ~59.980 €/79.980 € otras). La edición **gratis limita a 5
  instalaciones y no permite redistribuir**.
- **Conclusión**: descartado por coste de empaquetado.

### WinDivert
- Driver WFP/WDF, API user-mode. Soporta interceptar/descartar/modificar,
  loopback, IPv6; capa **L3 (IP)**.
- **Licencia**: dual **LGPL-3.0 / GPL-2.0**; licencia comercial 500 €/año
  (básica) o 1.000 €/año (avanzada).
- Drivers pre-firmados; embebibles (`go:embed`).
- **Requiere administrador** para cargar el driver.
- **Conclusión**: elegido. Encaja con GPL-2.0 y evita el coste de Npcap.

### ETW / IP Helper (`GetPerTcpConnectionEStats`)
- Sin driver y sin privilegios, pero solo estadísticas por conexión y solo
  Windows. Plan B, no base.

## Librerías Go evaluadas

### gopacket
- **`github.com/gopacket/gopacket`**: fork mantenido del original de Google.
  Versión actual **v1.7.4** (jul. 2026), Go 1.25+. Licencia BSD-3. Incluye
  `pcap` (cgo en Unix), `afpacket` (Linux), `bsdbpf` (BSD/macOS), `pcapgo`
  (puro Go). **Elegido.**
- `github.com/google/gopacket`: **sin mantenimiento** (última release 2020).
- `github.com/malivvan/gopacket`: fork con `purego` (sin cgo, `dlopen` de
  libpcap). Alternativa si algún día se quiere libpcap sin cgo.

### Bindings de WinDivert en Go
- **`github.com/imgk/divert-go`** (v0.1.0): bindings por syscall, soporta DLL o
  driver embebido (`-tags divert_rsrc`). **Elegido.**
- `github.com/one-api/godivert`: implementación pura Go que embebe los `.sys`.
  Alternativa.
- `oioio-space/pkt`: AF_PACKET (Linux) + WinDivert (Windows), API tipo gopacket.
  Útil como referencia del patrón unificado.

### Otras
- `packetcap/go-pcap`, `huatuo-ai/go-pcap`: puro Go, Linux+macOS, **sin
  Windows**.
- `wails` / `fyne` / `lxn/walk`: candidatos para la GUI de bandeja pendiente.

## Colector de Pandora FMS

- El colector es **`nfcapd` (de nfdump)**, escuchando en **9995/UDP**, que
  acepta **NetFlow v5/v9 e IPFIX**. Pandora arranca/para `nfcapd` con el server.
- La documentación de Pandora propone como sonda `pmacctd` con
  `nfprobe_version: 9` y `nfprobe_receiver: <ip>:9995`.

## Cambios que invalidarían decisiones

- Si **Windows dejara de ser requisito**: reconsiderar fork de softflowd (BSD,
  pequeño) o incluso desplegar pmacctd.
- Si hiciera falta **IPFIX/sFlow** o enriquecido BGP: pmacct (GPLv2) encajaría
  mejor que ampliar este agente.
- Si se quisiera **cerrar el producto**: pasar WinDivert a LGPL-3.0 (enlace
  dinámico) o comprar su licencia comercial; Npcap seguiría siendo caro.
