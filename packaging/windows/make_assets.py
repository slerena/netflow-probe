#!/usr/bin/env python3
"""Genera los recursos gráficos del instalador de Windows (requiere Pillow).

  AppIcon.ico   icono del instalador, accesos directos y .exe
  welcome.bmp   panel lateral de Bienvenida/Final (164x314)
  header.bmp    cabecera de las páginas interiores (150x57)

Uso:  python3 packaging/windows/make_assets.py
"""
import os

from PIL import Image, ImageDraw, ImageFont

BASE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.abspath(os.path.join(BASE, "..", ".."))
ICON = os.path.join(REPO, "cmd/netflow-probe-gui/icon.png")

TOP = (247, 248, 251)
BOT = (218, 224, 238)
TEXT = (48, 52, 66)
WHITE = (255, 255, 255)


def font(size):
    for path in (
        "/System/Library/Fonts/SFNS.ttf",
        "/System/Library/Fonts/Supplemental/Arial.ttf",
        "/System/Library/Fonts/Helvetica.ttc",
    ):
        try:
            return ImageFont.truetype(path, size)
        except OSError:
            continue
    return ImageFont.load_default()


def make_ico():
    im = Image.open(ICON).convert("RGBA")
    im.save(os.path.join(BASE, "AppIcon.ico"), format="ICO",
            sizes=[(16, 16), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)])


def gradient(w, h):
    img = Image.new("RGB", (w, h))
    d = ImageDraw.Draw(img)
    for y in range(h):
        t = y / (h - 1)
        col = tuple(int(TOP[i] + (BOT[i] - TOP[i]) * t) for i in range(3))
        d.line([(0, y), (w, y)], fill=col)
    return img


def make_welcome():
    w, h = 164, 314
    canvas = gradient(w, h)

    icon = Image.open(ICON).convert("RGBA").resize((76, 76), Image.LANCZOS)
    canvas.paste(icon, ((w - 76) // 2, 78), icon)

    d = ImageDraw.Draw(canvas)
    f = font(15)
    for i, line in enumerate(("Pandora FMS", "Netflow Probe")):
        tw = d.textlength(line, font=f)
        d.text(((w - tw) / 2, 176 + i * 20), line, font=f, fill=TEXT)

    canvas.save(os.path.join(BASE, "welcome.bmp"))


def make_header():
    w, h = 150, 57
    canvas = Image.new("RGB", (w, h), WHITE)
    icon = Image.open(ICON).convert("RGBA").resize((38, 38), Image.LANCZOS)
    canvas.paste(icon, (10, (h - 38) // 2), icon)

    d = ImageDraw.Draw(canvas)
    f = font(11)
    d.text((56, 12), "Pandora FMS", font=f, fill=TEXT)
    d.text((56, 27), "Netflow Probe", font=f, fill=TEXT)

    canvas.save(os.path.join(BASE, "header.bmp"))


if __name__ == "__main__":
    make_ico()
    make_welcome()
    make_header()
    for name in ("AppIcon.ico", "welcome.bmp", "header.bmp"):
        p = os.path.join(BASE, name)
        print("escrito:", name, Image.open(p).size)
