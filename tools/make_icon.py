#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""Lastcast icon — the first design: deep-indigo squircle, warm-orange vortex,
white microphone in the middle. (The hand-drawn red-paper variant lives in
tools/make_icon_sketch.py.)

Usage: python tools/make_icon.py   (needs Pillow)
"""
import math
import os

from PIL import Image, ImageDraw, ImageFilter

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
OUT = os.path.join(ROOT, "web", "icons")
os.makedirs(OUT, exist_ok=True)

S = 1024
SS = 2
W = S * SS

BG_TOP = (26, 15, 51)      # #1a0f33
BG_BOTTOM = (52, 27, 94)   # #341b5e
SWIRL_A = (255, 107, 53)   # #ff6b35
SWIRL_B = (255, 188, 120)  # #ffbc78
MIC = (248, 246, 255)      # #f8f6ff
MIC_SOFT = (214, 205, 245)


def lerp(a, b, t):
    return tuple(int(round(a[i] + (b[i] - a[i]) * t)) for i in range(3))


def draw_arm(img, cx, cy, r0, r1, turns, w0, w1, phase):
    """One tapered spiral arm, filled per segment so it carries a gradient."""
    th0 = phase
    steps = 220
    for i in range(steps):
        t0, t1 = i / steps, (i + 1) / steps
        pts = []
        for t in (t0, t1):
            th = th0 + turns * 2 * math.pi * t
            r = r0 + (r1 - r0) * t
            w = w0 + (w1 - w0) * t
            dx = -r * math.sin(th) + (r1 - r0) / (turns * 2 * math.pi) * math.cos(th)
            dy = r * math.cos(th) + (r1 - r0) / (turns * 2 * math.pi) * math.sin(th)
            ln = math.hypot(dx, dy) or 1.0
            nx, ny = -dy / ln, dx / ln
            px, py = cx + r * math.cos(th), cy + r * math.sin(th)
            pts.append(((px + nx * w / 2, py + ny * w / 2), (px - nx * w / 2, py - ny * w / 2)))
        quad = [pts[0][0], pts[1][0], pts[1][1], pts[0][1]]
        col = lerp(SWIRL_A, SWIRL_B, (t0 + t1) / 2)
        d = ImageDraw.Draw(img)
        d.polygon(quad, fill=col)
        if i == 0:
            x = (pts[0][0][0] + pts[0][1][0]) / 2
            y = (pts[0][0][1] + pts[0][1][1]) / 2
            d.ellipse((x - w0 / 2, y - w0 / 2, x + w0 / 2, y + w0 / 2), fill=col)
        if i == steps - 1:
            x = (pts[1][0][0] + pts[1][1][0]) / 2
            y = (pts[1][0][1] + pts[1][1][1]) / 2
            d.ellipse((x - w1 / 2, y - w1 / 2, x + w1 / 2, y + w1 / 2), fill=col)


def rounded_bg(size, radius_ratio=0.225):
    img = Image.new("RGB", (size, size), BG_TOP)
    draw = ImageDraw.Draw(img)
    for y in range(size):
        draw.line([(0, y), (size, y)], fill=lerp(BG_TOP, BG_BOTTOM, y / (size - 1)))
    # subtle radial glow behind the swirl
    glow = Image.new("L", (size, size), 0)
    gd = ImageDraw.Draw(glow)
    c = size * 0.46
    for i in range(24):
        k = (24 - i) / 24
        r = size * (0.50 - 0.014 * i)
        gd.ellipse((c - r, c - r, c + r, c + r), fill=int(18 * k))
    glow = glow.filter(ImageFilter.GaussianBlur(size * 0.06))
    light = Image.new("RGB", (size, size), (120, 80, 200))
    img = Image.composite(light, img, glow.point(lambda v: min(255, int(v * 1.4))))

    mask = Image.new("L", (size, size), 0)
    ImageDraw.Draw(mask).rounded_rectangle((0, 0, size - 1, size - 1), radius=int(size * radius_ratio), fill=255)
    out = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    out.paste(img, (0, 0), mask)
    ImageDraw.Draw(out).rounded_rectangle(
        (int(size * 0.012), int(size * 0.012), size - int(size * 0.012), size - int(size * 0.012)),
        radius=int(size * radius_ratio * 0.97), outline=(255, 255, 255, 26), width=max(2, size // 340))
    return out


def draw_mic(img, cx, cy):
    d = ImageDraw.Draw(img)
    u = SS
    mx, my = 512, 512

    def X(x):
        return cx + (x - mx) * u

    def Y(y):
        return cy + (y - my) * u

    # capsule + grill
    d.rounded_rectangle([X(454), Y(344), X(570), Y(552)], radius=int(58 * u), fill=MIC)
    for i in range(5):
        y = 374 + i * 38
        d.rounded_rectangle([X(478), Y(y), X(546), Y(y + 9)], radius=int(4 * u), fill=MIC_SOFT)
    # U-shaped holder
    d.arc([X(404), Y(520), X(620), Y(650)], start=0, end=180, fill=MIC, width=int(20 * u))
    # stand + base
    d.rounded_rectangle([X(502), Y(644), X(522), Y(736)], radius=int(10 * u), fill=MIC)
    d.rounded_rectangle([X(434), Y(732), X(590), Y(754)], radius=int(11 * u), fill=MIC)


def build():
    img = rounded_bg(W)
    cx = cy = W / 2
    # soft dark well behind the mic
    well = Image.new("RGBA", (W, W), (0, 0, 0, 0))
    ImageDraw.Draw(well).ellipse((cx - W * 0.185, cy - W * 0.185, cx + W * 0.185, cy + W * 0.185), fill=(24, 13, 46, 235))
    well = well.filter(ImageFilter.GaussianBlur(W * 0.012))
    img.alpha_composite(well)
    # two mirrored arms
    draw_arm(img, cx, cy, r0=W * 0.205, r1=W * 0.440, turns=0.86, w0=W * 0.018, w1=W * 0.078, phase=math.radians(-58))
    draw_arm(img, cx, cy, r0=W * 0.205, r1=W * 0.440, turns=0.86, w0=W * 0.018, w1=W * 0.078, phase=math.radians(122))
    draw_mic(img, cx, cy)
    master = img.resize((S, S), Image.LANCZOS)
    master.save(os.path.join(OUT, "icon-1024.png"))
    for name, size in (("icon-512.png", 512), ("icon-192.png", 192),
                       ("apple-touch-icon.png", 180), ("favicon-32.png", 32)):
        master.resize((size, size), Image.LANCZOS).save(os.path.join(OUT, name))
    maskable = Image.new("RGBA", (S, S), BG_TOP + (255,))
    inner = master.resize((int(S * 0.8), int(S * 0.8)), Image.LANCZOS)
    maskable.paste(inner, (int(S * 0.1), int(S * 0.1)), inner)
    maskable.save(os.path.join(OUT, "icon-512-maskable.png"))
    return master


def write_svg():
    def arm_path(cx, cy, r0, r1, turns, w0, w1, phase, samples=200):
        th0 = phase
        th1 = phase + turns * 2 * math.pi
        left, right = [], []
        for i in range(samples + 1):
            t = i / samples
            th = th0 + (th1 - th0) * t
            r = r0 + (r1 - r0) * t
            w = w0 + (w1 - w0) * t
            dx = -r * math.sin(th) + (r1 - r0) / (th1 - th0) * math.cos(th)
            dy = r * math.cos(th) + (r1 - r0) / (th1 - th0) * math.sin(th)
            ln = math.hypot(dx, dy) or 1.0
            nx, ny = -dy / ln, dx / ln
            px, py = cx + r * math.cos(th), cy + r * math.sin(th)
            left.append((px + nx * w / 2, py + ny * w / 2))
            right.append((px - nx * w / 2, py - ny * w / 2))
        pts = left + list(reversed(right))
        return "M" + " L".join(f"{x:.1f},{y:.1f}" for x, y in pts) + " Z"

    cx = cy = 512.0
    a1 = arm_path(cx, cy, 205, 440, 0.86, 18, 78, math.radians(-58))
    a2 = arm_path(cx, cy, 205, 440, 0.86, 18, 78, math.radians(122))
    svg = f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024" role="img" aria-label="Lastcast">
  <defs>
    <linearGradient id="bg" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="#1a0f33"/><stop offset="1" stop-color="#341b5e"/>
    </linearGradient>
    <linearGradient id="sw" x1="0" y1="0" x2="1" y2="1">
      <stop offset="0" stop-color="#ff6b35"/><stop offset="1" stop-color="#ffbc78"/>
    </linearGradient>
    <clipPath id="m"><rect width="1024" height="1024" rx="230"/></clipPath>
  </defs>
  <g clip-path="url(#m)">
    <rect width="1024" height="1024" fill="url(#bg)"/>
    <circle cx="512" cy="512" r="430" fill="#5a3aa0" opacity=".18"/>
    <circle cx="512" cy="512" r="190" fill="#180d2e" opacity=".85"/>
    <path d="{a1}" fill="url(#sw)"/>
    <path d="{a2}" fill="url(#sw)" opacity=".92"/>
    <g fill="#f8f6ff">
      <rect x="454" y="344" width="116" height="208" rx="58"/>
      <path d="M404 520 v10 a108 130 0 0 0 216 0 v-10 h-20 v10 a88 110 0 0 1 -176 0 v-10 z"/>
      <rect x="502" y="644" width="20" height="92" rx="10"/>
      <rect x="434" y="732" width="156" height="22" rx="11"/>
    </g>
    <g fill="#d6cdf5" opacity=".65">
      <rect x="478" y="374" width="68" height="9" rx="4.5"/>
      <rect x="478" y="412" width="68" height="9" rx="4.5"/>
      <rect x="478" y="450" width="68" height="9" rx="4.5"/>
      <rect x="478" y="488" width="68" height="9" rx="4.5"/>
      <rect x="478" y="526" width="68" height="9" rx="4.5"/>
    </g>
  </g>
</svg>
'''
    with open(os.path.join(ROOT, "web", "favicon.svg"), "w", encoding="utf-8") as f:
        f.write(svg)


if __name__ == "__main__":
    build()
    write_svg()
    print("first-design icons written to", OUT)
