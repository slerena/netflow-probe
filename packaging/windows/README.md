# Packaging — Windows

Empaquetado para Windows. Ver también [../../docs/gui-windows.md](../../docs/gui-windows.md).

| Recurso | Para qué |
|---|---|
| `AppIcon.ico` | Icono del ejecutable (6 tamaños: 16–256). |
| `app.manifest` | Manifiesto con `requireAdministrator` (UAC al abrir) y SOs soportados. |
| `../../scripts/build-windows.ps1` | Genera el `.syso` (rsrc) y compila GUI + agente. |
| `../../scripts/fetch-windivert.ps1` | Descarga WinDivert y copia DLL + driver. |

## Pasos

```powershell
pwsh scripts\build-windows.ps1
pwsh scripts\fetch-windivert.ps1
```

Salida en `dist\`:

- `Pandora FMS Netflow Probe.exe` — GUI Fyne (cgo), sin consola, auto-elevada.
- `netflow-probe.exe` — agente/servicio (sin cgo).

## Servicio

```powershell
.\netflow-probe.exe -install -config "C:\ProgramData\Pandora\netflow-probe.yaml"
.\netflow-probe.exe -uninstall
```

El manifiesto de la GUI pide administrador; el servicio lo ejecuta el SCM con la
cuenta de sistema, sin UAC por sesión.
