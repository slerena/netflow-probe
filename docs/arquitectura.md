# Arquitectura

## Vista general

```
            ┌──────────────────────────── netflow-probe (un binario) ───────────────────────────┐
            │                                                                                    │
 paquetes   │   ┌───────────────┐   ┌──────────────┐   ┌───────────────┐   ┌────────────────┐  │
 ─────────▶ │   │  capture      │──▶│  flow        │──▶│  flow.Table   │──▶│  export        │  │  UDP/9995
 (NIC)      │   │  (por SO)     │   │  decodifica  │   │  agrega y     │   │  NetFlow v9    │ ─┼──────▶ Pandora
            │   └───────────────┘   │  5-tupla     │   │  caduca       │   └────────────────┘  │  (nfcapd)
            │                       └──────────────┘   └───────────────┘                       │
            └────────────────────────────────────────────────────────────────────────────────────┘
```

Flujo de datos:
1. **capture** entrega frames crudos (Ethernet en Linux/macOS, IP desnudo en
   Windows) con su timestamp.
2. **flow.Observe** decodifica IP + transporte y produce una `flow.Key`
   (5-tupla + ToS), el tamaño L3 y los flags TCP.
3. **flow.Table.Observe** acumula bytes/paquetes por clave y aplica timeouts.
4. El ticker llama a **Table.Expire** (caducados) o, al parar, **Table.Flush**
   (todos). Los `flow.Record` van al exportador.
5. **export.Exporter.Send** serializa NetFlow v9 (plantillas + datos, con
   refresco de plantillas) y lo envía por UDP.

## Paquetes

| Paquete | Responsabilidad |
|---|---|
| `internal/config` | Configuración YAML con `Duration` legible (`30m`, `15s`). |
| `internal/capture` | Interfaz `Source` + backends por SO (build tags). |
| `internal/flow` | Decodificación a `Key` y tabla de agregación (`Table`). |
| `internal/export` | Serialización NetFlow v9 y transporte UDP. |
| `cmd/netflow-probe` | CLI, cableado y bucle principal. |

## Backends de captura (build tags)

| Fichero | Build tag | Tecnología | LinkType |
|---|---|---|---|
| `capture_linux.go` | `linux` | AF_PACKET (`gopacket/afpacket`) | Ethernet |
| `capture_darwin.go` | `darwin` | `/dev/bpf` (`gopacket/bsdbpf`) | Ethernet |
| `capture_windows.go` | `windows` | WinDivert (`imgk/divert-go`) | Raw IP |
| `capture_other.go` | resto | no soportado | — |
| `capture.go` (`OpenFile`) | todas | pcap/pcapng (`gopacket/pcapgo`) | la del fichero |

Todos los backends son **puros Go**: `CGO_ENABLED=0` en Linux, macOS y Windows.
Esto permite cross-compilar el `.exe` de Windows desde macOS/Linux.

## Decodificación

`flow.Observe` usa `gopacket.NewPacket` con `Lazy:true, NoCopy:true` (sin copias
para ir rápido). Para `layers.LinkTypeRaw` (Windows) decodifica directamente
IPv4/IPv6 desnudos; para Ethernet, tras el frame. La longitud contabilizada es
la **L3** (campo total length de IPv4, 40 + payload length en IPv6), que es lo
que espera el campo `IN_BYTES` de NetFlow.

## NetFlow v9

- Cabecera de 20 bytes: versión, count (plantillas + registros), sysUptime,
  unixSecs, sequence, sourceID.
- Dos plantillas: IPv4 (ID 256) e IPv6 (ID 257). El campo `count` incluye las
  plantillas enviadas en ese datagrama.
- Refresco de plantillas por familia cada `template_refresh` paquetes.
- Los timestamps `FIRST_SWITCHED`/`LAST_SWITCHED` son milisegundos relativos al
  arranque del exportador.

## Concurrencia

Una goroutine de captura llama a `Table.Observe`; el bucle principal llama a
`Table.Expire`/`Flush` en cada tick. `Table` y `Exporter` están protegidos con
mutex. El apagado (SIGINT/SIGTERM, o EOF en modo `-read`) vuelca todos los flujos.

## Rendimiento (notas)

- Sin copias de paquetes en decodificación (`NoCopy`).
- Acotado por `max_flows`.
- Para enlaces muy cargados, el siguiente paso es mover el filtrado a kernel
  (BPF en Linux/macOS; WinDivert ya filtra en kernel con el `filter` dado).
