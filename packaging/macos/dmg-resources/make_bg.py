#!/usr/bin/env python3
"""Genera el fondo del DMG (1x y 2x): degradado + título + flecha.

Requiere Pillow:  pip3 install --user pillow
Uso:  python3 packaging/macos/dmg-resources/make_bg.py
"""
import os

from PIL import Image, ImageDraw, ImageFont

BASE = os.path.dirname(os.path.abspath(__file__))

W, H = 640, 420           # ventana del DMG (1x)
APP_X, DEST_X = 170, 470  # centros horizontales de la app y de Aplicaciones
ICON_Y = 210              # centro vertical de los iconos (1x)

TOP = (248, 248, 251, 255)
BOT = (226, 228, 236, 255)
ARROW = (120, 122, 135, 255)
TITLE = (52, 54, 64, 255)


def load_font(size):
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


def render(scale, out_path):
    w, h = W * scale, H * scale
    img = Image.new("RGBA", (w, h))
    draw = ImageDraw.Draw(img)

    # Degradado vertical.
    for y in range(h):
        t = y / (h - 1)
        col = tuple(int(TOP[i] + (BOT[i] - TOP[i]) * t) for i in range(4))
        draw.line([(0, y), (w, y)], fill=col)

    # Título.
    font = load_font(int(22 * scale))
    title = "Pandora FMS Netflow Probe"
    tw = draw.textlength(title, font=font)
    draw.text(((w - tw) / 2, 40 * scale), title, font=font, fill=TITLE)

    # Flecha de la app a Aplicaciones.
    y = ICON_Y * scale
    x0, x1 = (APP_X + 84) * scale, (DEST_X - 84) * scale
    draw.line([x0, y, x1, y], fill=ARROW, width=max(2, 4 * scale))
    head = 12 * scale
    draw.polygon([(x1, y - head), (x1 + int(16 * scale), y), (x1, y + head)], fill=ARROW)

    img.convert("RGB").save(out_path)
    print("escrito:", out_path, Image.open(out_path).size)


render(1, os.path.join(BASE, "dmg-bg.png"))
render(2, os.path.join(BASE, "dmg-bg@2x.png"))
