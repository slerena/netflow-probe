# Packaging — Windows

ZIP portable e **instalador NSIS** para Windows, basados en el empaquetado de
Pandora MINI. Ver también [../../docs/gui-windows.md](../../docs/gui-windows.md).

| Recurso | Para qué |
|---|---|
| `PandoraFMSNetflowProbe.nsi` | Instalador NSIS (es/en, licencia, componentes, desinstalador). |
| `make_installer.sh` | Genera `dist/Pandora FMS Netflow Probe Setup <ver>.exe` (makensis). |
| `make_zip.sh` | Genera `dist/Pandora FMS Netflow Probe <ver> (Windows).zip`. |
| `write_launcher.sh` | Escribe `Modo compatible.bat` (Mesa/llvmpipe). |
| `make_assets.py` | Genera `AppIcon.ico`, `welcome.bmp`, `header.bmp`. |
| `make_license_rtf.py` | Convierte la licencia UTF-8 a RTF para NSIS. |
| `LEEME.txt` | Léeme del paquete (modos, WinDivert, colector, servicio). |
| `fetch_mesa.sh` | Descarga Mesa 26.2.4 y verifica SHA-256. |
| `app.manifest` | Manifiesto `requireAdministrator` del `.exe`. |
| `payload/` | **No versionado**: `.exe` compilado + WinDivert + Mesa. |

## 1. Preparar el payload

El `.exe` de la GUI se compila **en Windows** (Fyne/cgo; ver
`docs/gui-windows.md`). Deja en `packaging/windows/payload/`:

```
PandoraFMSNetflowProbe.exe     GUI compilada (con manifiesto/icono)
netflow-probe.exe              agente/servicio (CGO_ENABLED=0)
WinDivert.dll                  WinDivert 2.2.2
WinDivert64.sys
mesa/opengl32.dll              Mesa llvmpipe (modo compatible)
mesa/libgallium_wgl.dll
```

## 2. Construir

Requiere `makensis` y Pillow:

```sh
brew install makensis
python3 -m pip install --user pillow

make windows-zip         # ZIP portable
make windows-installer   # instalador NSIS
```

Salida en `dist/`:

- `Pandora FMS Netflow Probe <ver> (Windows).zip` — modo normal + `swgl/` (Mesa)
  + `Modo compatible.bat` + LEEME.
- `Pandora FMS Netflow Probe Setup <ver>.exe` — instalador con licencia,
  componentes (app, escritorio, modo compatible), accesos directos y
  desinstalador registrado en "Aplicaciones instaladas".

## 3. Notas

- El instalador pide administrador (instala en `Program Files` y el `.exe`
  necesita admin para WinDivert).
- La configuración del usuario (`%USERPROFILE%\.netflow-probe.yaml`) **no** se
  toca al instalar/desinstalar.
- El modo compatible (Mesa) es necesario en VMs/RDP sin OpenGL 2.x; en hardware
  normal se usa el ejecutable de la raíz.
- El agente `netflow-probe.exe` puede instalarse como servicio:
  `netflow-probe.exe -install -config "C:\ProgramData\Pandora\netflow-probe.yaml"`.
