#!/usr/bin/env python3
"""Añade al DMG un aviso de licencia (Aceptar/Rechazar al montarlo).

Usa dmgbuild para construir el plist de recursos de licencia y lo inserta con
`hdiutil udifrez`. Los textos están en packaging/macos/dmg-resources/legal/.

Uso: python3 add_license.py "dist/Pandora FMS Netflow Probe.dmg"
"""
import os
import plistlib
import subprocess
import sys
import tempfile

from dmgbuild.licensing import build_license

BASE = os.path.dirname(os.path.abspath(__file__))
LEGAL = os.path.join(BASE, "legal")


def main():
    dmg = sys.argv[1] if len(sys.argv) > 1 else "dist/Pandora FMS Netflow Probe.dmg"

    info = {
        "licenses": {
            "es_ES": os.path.join(LEGAL, "es.txt"),
            "en_US": os.path.join(LEGAL, "en.txt"),
        }
    }
    xml = build_license(info)

    with tempfile.NamedTemporaryFile(suffix=".plist", delete=False) as fh:
        plistlib.dump(xml, fh)
        plist_path = fh.name
    try:
        subprocess.check_call(["/usr/bin/hdiutil", "udifrez", "-xml", plist_path, "", dmg])
    finally:
        os.remove(plist_path)

    print("Aviso de licencia añadido a", dmg)


if __name__ == "__main__":
    main()
