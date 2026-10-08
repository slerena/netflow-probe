# Packaging — Windows

Aquí vivirá el empaquetado para Windows (pendiente). Plan:

- `installer/` — definición del MSI (WiX Toolset) o NSIS.
- `driver/` — WinDivert `WinDivert.dll` + `WinDivert64.sys` (o embebidos en el
  binario con `go:embed` / `-tags divert_rsrc`).
- `service/` — instalación del agente como servicio de Windows (`LocalSystem`)
  para evitar UAC en cada arranque (p. ej. con `kardianos/service`).

Pasos previstos:

```powershell
# Compilar el ejecutable (desde cualquier SO con Go):
make windows      # -> netflow-probe.exe

# Embeber driver (futuro):
#   go build -tags divert_rsrc ...
```

Ver [../../docs/instalacion-windows.md](../../docs/instalacion-windows.md).
