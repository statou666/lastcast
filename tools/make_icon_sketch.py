#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""Generate the Lastcast icon: a hand-drawn red marker sketch on paper.

A wobbly spiral (Dreamcast-like swirl) with a little microphone doodle in the
middle, drawn with intentionally uneven strokes so it feels handmade.

Usage: python tools/make_icon.py   (needs Pillow + numpy)
"""
import math
import os
import random

import numpy as np
from PIL import Image, ImageDraw, ImageFilter

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
OUT = os.path.join(ROOT, "web", "icons")
os.makedirs(OUT, exist_ok=True)

S = 1024
SS = 2
W = S * SS

PAPER = (246, 241, 231)
PAPER_DARK = (236, 229, 214)
INK = (222, 44, 32)
INK_DARK = (196, 34, 24)


def unit(v):
    n = math.hypot(*v) or 1.0
    return (v[0] / n, v[1] / n)


def wobble(pts, amp, seed, freq=3.0):
    """Push each point along its normal with a smooth pseudo-random wave."""
    rnd = random.Random(seed)
    ph1, ph2 = rnd.uniform(0, 6.28), rnd.uniform(0, 6.28)
    out = []
    n = len(pts)
    for i, (x, y) in enumerate(pts):
        t = i / max(1, n - 1)
        prev = pts[max(0, i - 1)]
        nxt = pts[min(n - 1, i + 1)]
        nx, ny = unit((-(nxt[1] - prev[1]), nxt[0] - prev[0]))
        d = amp * (math.sin(t * freq * 6.28 + ph1) * 0.6 + math.sin(t * freq * 2.7 * 6.28 + ph2) * 0.4)
        d += rnd.uniform(-amp * 0.18, amp * 0.18)
        out.append((x + nx * d, y + ny * d))
    return out


def brush(draw, pts, width, color, seed=1, junction=True):
    """Marker-like stroke: slightly varying width, round joints, a lighter second pass."""
    rnd = random.Random(seed)
    for i in range(len(pts) - 1):
        t = i / max(1, len(pts) - 2)
        w = width * (0.86 + 0.22 * math.sin(t * 11 + seed) + rnd.uniform(-0.06, 0.06))
        (x0, y0), (x1, y1) = pts[i], pts[i + 1]
        dx, dy = x1 - x0, y1 - y0
        ln = math.hypot(dx, dy) or 1.0
        nx, ny = -dy / ln, dx / ln
        draw.polygon([(x0 + nx * w / 2, y0 + ny * w / 2),
                      (x1 + nx * w / 2, y1 + ny * w / 2),
                      (x1 - nx * w / 2, y1 - ny * w / 2),
                      (x0 - nx * w / 2, y0 - ny * w / 2)], fill=color)
        if junction:
            draw.ellipse((x1 - w / 2, y1 - w / 2, x1 + w / 2, y1 + w / 2), fill=color)


def paper(size):
    """Warm paper with grain and a soft edge shadow."""
    base = Image.new("RGB", (size, size), PAPER)
    arr = np.asarray(base).astype(np.int16)
    rng = np.random.default_rng(7)
    grain = rng.normal(0, 3.0, (size, size, 1))
    arr = np.clip(arr + grain, 0, 255).astype(np.uint8)
    img = Image.fromarray(arr, "RGB")
    # gentle vignette
    vig = Image.new("L", (size, size), 0)
    vd = ImageDraw.Draw(vig)
    vd.ellipse((-size * 0.15, -size * 0.15, size * 1.15, size * 1.15), fill=255)
    vig = vig.filter(ImageFilter.GaussianBlur(size * 0.12))
    dark = Image.new("RGB", (size, size), PAPER_DARK)
    img = Image.composite(img, dark, vig)
    mask = Image.new("L", (size, size), 0)
    ImageDraw.Draw(mask).rounded_rectangle((0, 0, size - 1, size - 1), radius=int(size * 0.235), fill=255)
    out = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    out.paste(img, (0, 0), mask)
    return out


def rounded_rect_pts(x0, y0, x1, y1, r, n=9):
    """Clockwise rounded-rectangle outline points."""
    pts = []
    corners = ((x1 - r, y0 + r, -90, 0), (x1 - r, y1 - r, 0, 90),
               (x0 + r, y1 - r, 90, 180), (x0 + r, y0 + r, 180, 270))
    for cx, cy, a0, a1 in corners:
        for i in range(n + 1):
            a = math.radians(a0 + (a1 - a0) * i / n)
            pts.append((cx + r * math.cos(a), cy + r * math.sin(a)))
    return pts


def mic_shapes():
    """Apple-Podcasts-style line-art microphone in the same marker style.

    Capsule + grill lines + hugging U cradle + stem + base. Coordinates are 1024-space.
    """
    shapes = []
    # capsule: 200 x 300, centered at (512, 380)
    shapes.append(("stroke", rounded_rect_pts(412, 230, 612, 530, 62), 20))
    # grill lines
    for y in (330, 385, 440):
        shapes.append(("stroke", [(452, y), (572, y)], 13))
    # U cradle hugging the capsule: uprights + lower half circle
    arc = []
    for i in range(29):
        a = math.radians(180 + i * 180 / 28)
        arc.append((512 + 140 * math.cos(a), 580 + 140 * math.sin(a)))
    cradle = [(372, 470), (372, 580)] + arc[1:] + [(652, 580), (652, 470)]
    shapes.append(("stroke", cradle, 18))
    # stem + base
    shapes.append(("stroke", [(512, 720), (512, 844)], 18))
    shapes.append(("stroke", [(402, 870), (622, 870)], 20))
    return shapes


def spiral_points(cx, cy):
    """Marker spiral: tight near the mic, looser outside, tail at bottom right."""
    pts = []
    th0 = -0.5 * math.pi
    turns = 1.52
    th1 = th0 + turns * 2 * math.pi
    n = 300
    rot = math.radians(-100)
    sq = 0.94
    for i in range(n):
        t = i / (n - 1)
        th = th0 + (th1 - th0) * t
        r = 420 + (760 - 420) * t
        r += 12 * math.sin(t * 8.7) + 5 * math.sin(t * 21.0)  # hand sway
        x, y = r * math.cos(th) * 1.0, r * math.sin(th) * sq
        x, y = x * math.cos(rot) - y * math.sin(rot), x * math.sin(rot) + y * math.cos(rot)
        pts.append((cx + x, cy + y))
    tail = []
    for k in range(1, 11):
        t = k / 10
        th = th1 + 0.18 * t
        r = 760 + 150 * t * t
        x, y = r * math.cos(th), r * math.sin(th) * sq
        x, y = x * math.cos(rot) - y * math.sin(rot), x * math.sin(rot) + y * math.cos(rot)
        tail.append((cx + x, cy + y))
    return pts + tail


def build_sketch():
    img = paper(W)
    paper_img = img.copy()
    draw = ImageDraw.Draw(img)
    cx = cy = W / 2

    # spiral, drawn twice with a tiny offset for marker texture
    sp = wobble(spiral_points(cx, cy), 9, 21)
    brush(draw, [(x, y) for x, y in sp], 46, INK_DARK, seed=4)
    brush(draw, [(x + 3 * SS, y + 2 * SS) for x, y in sp], 40, INK, seed=5)

    # clear only the microphone footprint, restoring the real paper texture
    mic = mic_shapes()
    clear = Image.new("L", (W, W), 0)
    cd = ImageDraw.Draw(clear)
    cap_pts = [(cx + (x - 512) * SS, cy + (y - 512) * SS) for x, y in mic[0][1]]
    cd.polygon(cap_pts, fill=255)
    for shape in mic[1:]:
        pts = [(cx + (x - 512) * SS, cy + (y - 512) * SS) for x, y in shape[1]]
        brush(cd, pts, (shape[2] + 18) * SS, 255, seed=3, junction=False)
    clear = clear.filter(ImageFilter.GaussianBlur(SS * 1.2))
    img.paste(paper_img, (0, 0), clear)

    # red line-art microphone on top
    for shape in mic:
        pts = [(cx + (x - 512) * SS, cy + (y - 512) * SS) for x, y in shape[1]]
        brush(draw, pts, shape[2] * SS, INK, seed=int(shape[2]))

    # faded second pass over everything for that inked-over look
    ghost = img.copy()
    draw2 = ImageDraw.Draw(ghost)
    gsp = [(x + 4, y - 3) for x, y in sp]
    brush(draw2, gsp, 10 * SS / 2, INK, seed=31)
    ghost = ghost.filter(ImageFilter.GaussianBlur(SS * 0.6))
    img = Image.blend(img, ghost, 0.18)

    master = img.resize((S, S), Image.LANCZOS)
    master.save(os.path.join(OUT, "icon-1024.png"))
    for name, size in (("icon-512.png", 512), ("icon-192.png", 192),
                       ("apple-touch-icon.png", 180), ("favicon-32.png", 32)):
        master.resize((size, size), Image.LANCZOS).save(os.path.join(OUT, name))
    maskable = Image.new("RGBA", (S, S), PAPER + (255,))
    inner = master.resize((int(S * 0.8), int(S * 0.8)), Image.LANCZOS)
    maskable.paste(inner, (int(S * 0.1), int(S * 0.1)), inner)
    maskable.save(os.path.join(OUT, "icon-512-maskable.png"))
    return master


def write_svg():
    """Favicon SVG with the same geometry (1024 viewBox)."""
    def path(pts, close=False):
        d = "M" + " L".join(f"{x / SS:.1f},{y / SS:.1f}" for x, y in pts)
        return d + (" Z" if close else "")

    random.seed(21)
    sp = wobble(spiral_points(512 * SS, 512 * SS), 18, 21)
    shapes = mic_shapes()
    mic = [f'<path d="{path(shapes[0][1], True)}" fill="#f6f1e7"/>']
    for shape in shapes[1:]:
        mic.append(f'<path d="{path(shape[1])}" fill="none" stroke="#f6f1e7" '
                   f'stroke-width="{shape[2] + 18}" stroke-linecap="round" stroke-linejoin="round"/>')
    for shape in shapes:
        mic.append(f'<path d="{path(shape[1])}" fill="none" stroke="#de2c20" '
                   f'stroke-width="{shape[2]}" stroke-linecap="round" stroke-linejoin="round"/>')
    svg = f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024" role="img" aria-label="Lastcast">
  <defs><clipPath id="r"><rect width="1024" height="1024" rx="240"/></clipPath></defs>
  <g clip-path="url(#r)">
    <rect width="1024" height="1024" fill="#f6f1e7"/>
    <path d="{path(sp)}" fill="none" stroke="#c42218" stroke-width="23" stroke-linecap="round" stroke-linejoin="round"/>
    <path d="{path(sp)}" fill="none" stroke="#de2c20" stroke-width="20" stroke-linecap="round" stroke-linejoin="round" opacity=".85"/>
    {''.join(mic)}
  </g>
</svg>
'''
    with open(os.path.join(ROOT, "web", "favicon.svg"), "w", encoding="utf-8") as f:
        f.write(svg)


if __name__ == "__main__":
    build_sketch()
    write_svg()
    print("sketch icons written to", OUT)
