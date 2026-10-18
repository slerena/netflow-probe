#!/usr/bin/env python3
"""Cambia la versión interna del proyecto en todos los sitios.

Uso: python3 scripts/bump_version.py 1.0.33   (o `make bump V=1.0.33`)

Actualiza:
  - VERSION (fuente de verdad)
  - packaging/macos/Info.plist (CFBundleShortVersionString / CFBundleVersion)
  - scripts/build-windows.ps1 (versión por defecto)
  - scripts/build-linux.sh (versión por defecto)
  - cmd/netflow-probe/main.go y cmd/netflow-probe-gui/main_gui.go (default de
    la variable `version`, que ldflags puede sobrescribir)
"""
import plistlib
import re
import sys
from pathlib import Path


def main():
    if len(sys.argv) != 2:
        sys.exit("uso: bump_version.py <x.y.z>")
    v = sys.argv[1].strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", v):
        sys.exit("versión inválida: usa x.y.z (p. ej. 1.0.33)")

    root = Path(__file__).resolve().parent.parent
    (root / "VERSION").write_text(v + "\n", encoding="utf-8")

    info = root / "packaging/macos/Info.plist"
    with info.open("rb") as fh:
        data = plistlib.load(fh)
    data["CFBundleShortVersionString"] = v
    data["CFBundleVersion"] = v
    with info.open("wb") as fh:
        plistlib.dump(data, fh)

    ps1 = root / "scripts/build-windows.ps1"
    s = ps1.read_text(encoding="utf-8")
    s = re.sub(r'\[string\]\$Version = "[^"]+"', f'[string]$Version = "{v}"', s)
    ps1.write_text(s, encoding="utf-8")

    sh = root / "scripts/build-linux.sh"
    s = sh.read_text(encoding="utf-8")
    s = re.sub(r'VERSION="\$\{VERSION:-[^}]+\}"', f'VERSION="${{VERSION:-{v}}}"', s)
    sh.write_text(s, encoding="utf-8")

    for rel in ("cmd/netflow-probe/main.go", "cmd/netflow-probe-gui/main_gui.go"):
        p = root / rel
        s = p.read_text(encoding="utf-8")
        s = re.sub(r'var version = "[^"]*"', f'var version = "{v}"', s)
        p.write_text(s, encoding="utf-8")

    print(f"Versión -> {v} (VERSION, Info.plist, scripts y binarios)")


if __name__ == "__main__":
    main()
