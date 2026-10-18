#!/usr/bin/env python3
"""Convierte un .txt UTF-8 a RTF para la página de licencia de NSIS.

NSIS lee los .txt de licencia con la página de códigos ANSI del instalador, así
que acentos y símbolos se rompen. En RTF los caracteres no ASCII van como \\uNNNN
y se renderizan bien en cualquier idioma.

Uso:  python3 make_license_rtf.py entrada.txt salida.rtf
"""
import sys


def escape(text):
    out = []
    for ch in text:
        if ch == "\\":
            out.append("\\\\")
        elif ch == "{":
            out.append("\\{")
        elif ch == "}":
            out.append("\\}")
        elif ch == "\n":
            out.append("\\par\n")
        elif ch == "\r":
            continue
        elif ch == "\t":
            out.append("\\tab ")
        elif ord(ch) < 128:
            out.append(ch)
        else:
            for i in range(0, len(ch.encode("utf-16-le")), 2):
                unit = int.from_bytes(ch.encode("utf-16-le")[i:i + 2], "little")
                if unit >= 0x8000:
                    unit -= 0x10000
                out.append("\\u%d?" % unit)
    return "".join(out)


def main(src, dst):
    text = open(src, encoding="utf-8").read()
    rtf = ("{\\rtf1\\ansi\\ansicpg1252\\deff0"
           "{\\fonttbl{\\f0\\fswiss Helvetica;}}\\fs20 " + escape(text) + "}")
    with open(dst, "w", encoding="ascii") as fh:
        fh.write(rtf)
    print("escrito:", dst)


if __name__ == "__main__":
    main(sys.argv[1], sys.argv[2])
