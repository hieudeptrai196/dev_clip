#!/usr/bin/env python3
"""Generate docs/banner.gif — README banner for DevClip.

900x520. Left = static branding panel. Right = the REAL app screenshots
(docs/history.png + docs/settings.png) so the UI is pixel-accurate. Animation:
screenshot slides in + left text fades in, hold History, crossfade to Settings,
hold, crossfade back, loop.
"""
import os
from PIL import Image, ImageDraw, ImageFont, ImageFilter

W, H = 900, 520

# ---- palette ---------------------------------------------------------------
BG       = (13, 15, 18)     # #0d0f12
BORDER   = (48, 53, 62)
TXT      = (236, 238, 241)
TXT_DIM  = (140, 146, 156)
TXT_MUTE = (96, 102, 112)
CYAN     = (0, 212, 170)    # #00d4aa

# ---- fonts -----------------------------------------------------------------
FD = "C:/Windows/Fonts/"
def font(names, size):
    for n in names:
        p = FD + n
        if os.path.exists(p):
            try:
                return ImageFont.truetype(p, size)
            except Exception:
                pass
    return ImageFont.load_default()

SANS, SANSB, MONO = ["segoeui.ttf", "arial.ttf"], ["segoeuib.ttf", "arialbd.ttf"], ["consola.ttf"]
f_logo  = font(SANSB, 46)
f_tag   = font(SANSB, 21)
f_body  = font(SANS, 14)
f_small = font(SANS, 12)
f_badge = font(SANSB, 10)
f_btn   = font(SANSB, 14)
f_pill  = font(SANS, 12)
f_url   = font(MONO, 12)

def tw(d, s, f):
    b = d.textbbox((0, 0), s, font=f); return b[2] - b[0]

# ---- left branding panel (rendered once to RGBA) ---------------------------
def build_left():
    L = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    d = ImageDraw.Draw(L)
    x = 56
    d.rounded_rectangle([x, 84, x + 128, 105], radius=10, outline=tuple(int(CYAN[i]*0.6) for i in range(3)))
    d.text((x + 12, 88), "DEVELOPER TOOL", font=f_badge, fill=CYAN)
    d.text((x, 120), "Dev", font=f_logo, fill=TXT)
    d.text((x + tw(d, "Dev", f_logo), 120), "Clip", font=f_logo, fill=CYAN)
    d.text((x, 196), "Copy anything.", font=f_tag, fill=TXT)
    d.text((x, 224), "Paste it smarter.", font=f_tag, fill=TXT)
    for i, ln in enumerate([
        "Auto-detects JSON, SQL, JWT & timestamps.",
        "Format, decode and transform in one keystroke.",
        "Lives in RAM. Nothing ever touches your disk.",
    ]):
        d.text((x, 276 + i * 25), ln, font=f_body, fill=TXT_DIM)
    px = x
    for t in ["Go", "Wails", "React", "Windows"]:
        pw = tw(d, t, f_pill) + 22
        d.rounded_rectangle([px, 376, px + pw, 402], radius=13, outline=BORDER, width=1)
        d.text((px + 11, 381), t, font=f_pill, fill=TXT_DIM)
        px += pw + 9
    d.text((x, 436), "github.com/hieudeptrai196/dev_clip", font=f_url, fill=TXT_MUTE)
    d.text((x, 458), "Open source · MIT License", font=f_small, fill=TXT_MUTE)
    return L

LEFT = build_left()

# ---- real screenshots ------------------------------------------------------
SH = 462                          # target height for the app window
TWID = 356                        # uniform width (both shots ~0.77 ratio)
def load(name):
    return Image.open(f"docs/{name}.png").convert("RGBA").resize((TWID, SH), Image.LANCZOS)
HIST, SETT = load("history"), load("settings")

IMG_X = 488
IMG_Y = (H - SH) // 2

def make_shadow():
    s = Image.new("RGBA", (TWID + 80, SH + 80), (0, 0, 0, 0))
    ImageDraw.Draw(s).rounded_rectangle([40, 46, 40 + TWID, 46 + SH], radius=18, fill=(0, 0, 0, 160))
    return s.filter(ImageFilter.GaussianBlur(16))
SHADOW = make_shadow()

def compose(ox, left_alpha, shot):
    img = Image.new("RGB", (W, H), BG)
    if left_alpha > 0:
        layer = LEFT.copy()
        if left_alpha < 1:
            layer.putalpha(layer.split()[3].point(lambda p: int(p * left_alpha)))
        img.paste(layer, (0, 0), layer)
    x = IMG_X + ox
    img.paste(SHADOW, (x - 40, IMG_Y - 46), SHADOW)
    img.paste(shot, (x, IMG_Y), shot)
    return img

# ---- single static banner image --------------------------------------------
out = "docs/banner.png"
compose(0, 1, HIST).save(out)
print("saved", out, "size KB:", round(os.path.getsize(out) / 1024, 1))
