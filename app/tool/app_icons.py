"""Makes every icon of SABIFOX from its symbol, which it draws itself: a
low-poly fox head in a 24 grid (FOX, the colored mark) and its silhouette
(GLYPH, white on transparent, for the status bar and the tray).

Outputs: the Android launcher icons (adaptive and legacy), the Android TV
banner, the macOS icon set, the Windows .ico files of the app and the core,
the web dashboard's icon, the app's logo, the Linux icon, the tray icons, and
the documentation's logo and banner (docs/images).

Run from app/ after changing the symbol or the colors (needs Pillow):

    python tool/app_icons.py
"""
import os

from PIL import Image, ImageDraw, ImageEnhance, ImageFilter, ImageFont

HERE = os.path.dirname(os.path.abspath(__file__))
APP = os.path.join(HERE, '..')
ROOT = os.path.join(APP, '..')
CORE = os.path.join(ROOT, 'core')
RES = os.path.join(APP, 'android', 'app', 'src', 'main', 'res')
FONT = os.path.join(APP, 'assets', 'fonts', 'Vazirmatn-Bold.ttf')

DENSITY = {'mdpi': 1, 'hdpi': 1.5, 'xhdpi': 2, 'xxhdpi': 3, 'xxxhdpi': 4}
WINDOWS_SIZES = [16, 20, 24, 32, 40, 48, 64, 128, 256]
MACOS_SIZES = [16, 32, 64, 128, 256, 512, 1024]
TRAY_SIZES = [16, 20, 24, 32, 48, 64]

# Brand colors: a midnight indigo tile and a fox in warm oranges.
NIGHT_TOP, NIGHT_BOTTOM = (34, 30, 88), (11, 9, 28)
ORANGE_L, ORANGE, ORANGE_D = (255, 159, 67), (249, 115, 22), (214, 84, 10)
FOREHEAD = (255, 182, 102)
CREAM, CREAM_D = (255, 247, 237), (236, 224, 212)
EAR_IN, EAR_IN_D = (124, 45, 18), (100, 35, 14)
DARK = (28, 25, 23)
WHITE = (255, 255, 255, 255)

# The fox in a 24 grid: the left half; the right half is its mirror image,
# a shade darker, which gives the head its depth.
EAR = [(3.2, 1.8), (9.6, 7.6), (4.4, 11.2)]
EAR_INNER = [(4.4, 4.4), (8.2, 7.9), (5.2, 9.6)]
TOP = [(9.6, 7.6), (14.4, 7.6), (12, 12.2)]
HEAD = [(4.4, 11.2), (9.6, 7.6), (12, 12.2), (8.3, 14.2), (2.6, 12.4)]
CHEEK = [(2.6, 12.4), (8.3, 14.2), (12, 16.2), (12, 21.8)]
MUZZLE = [(8.3, 14.2), (12, 12.2), (15.7, 14.2), (12, 16.2)]
NOSE = [(10.6, 19.6), (13.4, 19.6), (12, 21.8)]
EYE = [(6.9, 12.2), (10.2, 13.0), (8.5, 14.0)]
SILHOUETTE = [(3.2, 1.8), (9.6, 7.6), (14.4, 7.6), (20.8, 1.8), (19.6, 11.2),
              (21.4, 12.4), (12, 21.8), (2.6, 12.4), (4.4, 11.2)]
GLYPH_MASK = [(12, 12.9), (14.6, 14.4), (12, 15.6), (9.4, 14.4)]


def mirror(points):
    return [(24 - x, y) for x, y in points]


def fox(size):
    """The colored fox, on transparent, size px square. Drawn at 4x."""
    k = 4
    s = size * k / 24
    img = Image.new('RGBA', (size * k, size * k), (0, 0, 0, 0))
    d = ImageDraw.Draw(img)

    def poly(points, color):
        d.polygon([(x * s, y * s) for x, y in points], fill=color)

    poly(EAR, ORANGE); poly(mirror(EAR), ORANGE_D)
    poly(EAR_INNER, EAR_IN); poly(mirror(EAR_INNER), EAR_IN_D)
    poly(HEAD, ORANGE_L); poly(mirror(HEAD), ORANGE)
    poly(TOP, FOREHEAD)
    poly(CHEEK, CREAM); poly(mirror(CHEEK), CREAM_D)
    poly(MUZZLE, ORANGE_D)
    poly(EYE, DARK); poly(mirror(EYE), DARK)
    poly(NOSE, DARK)
    return img.resize((size, size), Image.LANCZOS)


def glyph(size):
    """The fox's silhouette, white on transparent, with its eyes and mask
    cut out."""
    k = 4
    s = size * k / 24
    img = Image.new('RGBA', (size * k, size * k), (0, 0, 0, 0))
    ImageDraw.Draw(img).polygon([(x * s, y * s) for x, y in SILHOUETTE], fill=WHITE)
    hole = Image.new('L', img.size, 0)
    h = ImageDraw.Draw(hole)
    for points in (EYE, mirror(EYE), GLYPH_MASK):
        h.polygon([(x * s, y * s) for x, y in points], fill=255)
    img.paste((0, 0, 0, 0), (0, 0), hole)
    return img.resize((size, size), Image.LANCZOS)


def night(w, h):
    """The tile's midnight gradient, top left to bottom right."""
    out = Image.new('RGBA', (w, h))
    px = out.load()
    for y in range(h):
        for x in range(w):
            t = (x / max(1, w - 1) + y / max(1, h - 1)) / 2
            px[x, y] = tuple(round(a + (b - a) * t) for a, b in zip(NIGHT_TOP, NIGHT_BOTTOM)) + (255,)
    return out


def face(body, fox_share=0.68, ring=True):
    """The square face of the tile: the gradient, a warm glow, a thin ring,
    and the fox."""
    out = night(body, body)
    glow = Image.new('RGBA', (body, body), (0, 0, 0, 0))
    ImageDraw.Draw(glow).ellipse([body * 0.2, body * 0.2, body * 0.8, body * 0.82], fill=ORANGE + (120,))
    out.alpha_composite(glow.filter(ImageFilter.GaussianBlur(body * 0.09)))
    if ring:
        r = Image.new('RGBA', (body, body), (0, 0, 0, 0))
        ImageDraw.Draw(r).ellipse([body * 0.09, body * 0.09, body * 0.91, body * 0.91],
                                  outline=(255, 170, 90, 80), width=max(1, round(body * 0.011)))
        out.alpha_composite(r)
    f = fox(round(body * fox_share))
    out.alpha_composite(f, ((body - f.width) // 2, round((body - f.height) / 2 + body * 0.01)))
    return out


def rounded(image, radius):
    mask = Image.new('L', image.size, 0)
    ImageDraw.Draw(mask).rounded_rectangle([0, 0, image.width - 1, image.height - 1], radius=radius, fill=255)
    out = Image.new('RGBA', image.size, (0, 0, 0, 0))
    out.paste(image, (0, 0), mask)
    return out


def tile(size, share=0.805):
    """The rounded tile, share of size, centered on transparent."""
    body = round(size * share)
    out = Image.new('RGBA', (size, size), (0, 0, 0, 0))
    out.alpha_composite(rounded(face(body), round(body * 0.225)), ((size - body) // 2, (size - body) // 2))
    return out


def save(image, *parts):
    path = os.path.join(*parts)
    os.makedirs(os.path.dirname(path), exist_ok=True)
    image.save(path, optimize=True)


def crisp(art, size):
    """A small size, sharpened, which a few pixels otherwise blur."""
    image = art.resize((size, size), Image.LANCZOS)
    alpha = image.getchannel('A')
    rgb = image.convert('RGB').filter(ImageFilter.UnsharpMask(radius=0.6 if size <= 24 else 0.8, percent=120, threshold=0))
    rgb = ImageEnhance.Color(ImageEnhance.Contrast(rgb).enhance(1.06)).enhance(1.10)
    out = rgb.convert('RGBA')
    out.putalpha(alpha)
    return out


def write_ico(path, art, sizes=WINDOWS_SIZES):
    images = [crisp(art, n) if n <= 48 else art.resize((n, n), Image.LANCZOS) for n in sizes]
    images[-1].save(path, format='ICO', sizes=[(n, n) for n in sizes], append_images=images[:-1])


ADAPTIVE = """<?xml version="1.0" encoding="utf-8"?>
<!-- The launcher icon on Android 8+: the fox over the midnight background;
     the launcher crops both layers to its own shape. Made by
     tool/app_icons.py. -->
<adaptive-icon xmlns:android="http://schemas.android.com/apk/res/android">
    <background android:drawable="@mipmap/ic_launcher_background" />
    <foreground android:drawable="@mipmap/ic_launcher_foreground" />
</adaptive-icon>
"""


def android():
    for dpi, k in DENSITY.items():
        folder = os.path.join(RES, 'mipmap-' + dpi)
        px = round(108 * k)
        # Launchers show about the middle 72 of the 108 dp: the fox fills
        # about 60% of what shows.
        fg = Image.new('RGBA', (px, px), (0, 0, 0, 0))
        f = fox(round(px * 0.42))
        fg.alpha_composite(f, ((px - f.width) // 2, (px - f.height) // 2))
        save(fg, folder, 'ic_launcher_foreground.png')
        bg = face(px, fox_share=0.0001, ring=False)
        save(bg, folder, 'ic_launcher_background.png')
        save(rounded(face(round(48 * k * 4)).resize((round(48 * k), round(48 * k)), Image.LANCZOS), round(48 * k / 5)),
             folder, 'ic_launcher.png')
    with open(os.path.join(RES, 'mipmap-anydpi-v26', 'ic_launcher.xml'), 'w', newline='\n') as f:
        f.write(ADAPTIVE)


def wordmark(draw, xy, size, fill_sabi=(255, 255, 255, 255), fill_fox=ORANGE_L + (255,)):
    font = ImageFont.truetype(FONT, size)
    x, y = xy
    l, t, r, b = draw.textbbox((0, 0), 'SABI', font=font)
    draw.text((x - l, y - t), 'SABI', font=font, fill=fill_sabi)
    l2, t2, r2, b2 = draw.textbbox((0, 0), 'FOX', font=font)
    draw.text((x + (r - l) - l2 + round(size * 0.04), y - t), 'FOX', font=font, fill=fill_fox)
    return (r - l) + (r2 - l2) + round(size * 0.04), b - t


def text_size(text, size):
    font = ImageFont.truetype(FONT, size)
    l, t, r, b = ImageDraw.Draw(Image.new('RGBA', (1, 1))).textbbox((0, 0), text, font=font)
    return r - l, b - t


def tv_banner():
    """Android TV's banner, 320 x 180 px: the fox and the name on the
    midnight gradient."""
    w, h = 320, 180
    out = night(w, h)
    mark = fox(92)
    tw, th = text_size('SABIFOX', 40)
    gap = 12
    x = (w - (mark.width + gap + tw)) // 2
    out.alpha_composite(mark, (x, (h - mark.height) // 2))
    wordmark(ImageDraw.Draw(out), (x + mark.width + gap, (h - th) // 2), 40)
    save(out, RES, 'drawable-xhdpi', 'banner.png')


def macos():
    size, body = 1024, 824
    off = (size - body) // 2
    mask = Image.new('L', (body, body), 0)
    ImageDraw.Draw(mask).rounded_rectangle([0, 0, body - 1, body - 1], radius=round(body * 0.225), fill=255)
    shadow = Image.new('RGBA', (size, size), (0, 0, 0, 0))
    shadow.paste((0, 0, 0, 90), (off, off + 12), mask)
    out = shadow.filter(ImageFilter.GaussianBlur(14))
    out.paste(face(body), (off, off), mask)
    folder = os.path.join(APP, 'macos', 'Runner', 'Assets.xcassets', 'AppIcon.appiconset')
    for n in MACOS_SIZES:
        save(out.resize((n, n), Image.LANCZOS), folder, 'app_icon_%d.png' % n)


DOT_GOOD, DOT_BAD, DOT_SET = (63, 225, 176), (226, 40, 80), (96, 165, 250)
TRAY_TILE = (30, 27, 60, 255)


def asleep(color):
    return tuple(round(c * 0.45 + g * 0.55) for c, g in zip(color, (150, 160, 158)))


def tray_icon(dot=None, ring=None):
    """A tray icon (80 px): the white fox on the dark tile; a status dot at
    the bottom right; ring, a share of a turning arc while connecting."""
    n = 320
    img = Image.new('RGBA', (n, n), (0, 0, 0, 0))
    d = ImageDraw.Draw(img)
    d.rounded_rectangle([0, 0, n - 1, n - 1], radius=round(n * 0.22), fill=TRAY_TILE)
    art = glyph(round(n * 0.86))
    img.alpha_composite(art, ((n - art.width) // 2, (n - art.height) // 2 + round(n * 0.02)))
    if ring is not None:
        w = round(n * 0.07)
        d.arc([w // 2, w // 2, n - 1 - w // 2, n - 1 - w // 2], start=ring * 90 - 90, end=ring * 90, fill=ORANGE_L + (255,), width=w)
    if dot:
        r, gap = n * 0.17, n * 0.04
        x = y = n - r - n * 0.02
        d.ellipse([x - r - gap, y - r - gap, x + r + gap, y + r + gap], fill=TRAY_TILE)
        d.ellipse([x - r, y - r, x + r, y + r], fill=dot + (255,))
    return img.resize((80, 80), Image.LANCZOS)


def save_tray(path, big):
    images = [big.resize((k, k), Image.LANCZOS) for k in TRAY_SIZES]
    images[-1].save(path, format='ICO', sizes=[(k, k) for k in TRAY_SIZES], append_images=images[:-1])


def tray():
    app = os.path.join(APP, 'assets', 'tray')
    for old in os.listdir(app):
        os.remove(os.path.join(app, old))
    icons = {'off': tray_icon(), 'on': tray_icon(DOT_GOOD), 'idle': tray_icon(DOT_GOOD), 'error': tray_icon(DOT_BAD)}
    for frame in range(1, 5):
        icons['busy%d' % frame] = tray_icon(ring=frame - 1)
    for name, big in icons.items():
        save_tray(os.path.join(app, name + '.ico'), big)
        big.resize((64, 64), Image.LANCZOS).save(os.path.join(app, name + '.png'), optimize=True)
    folder = os.path.join(CORE, 'internal', 'desktop', 'tray')
    states = {
        'running': tray_icon(DOT_GOOD), 'set': tray_icon(DOT_SET),
        'running-asleep': tray_icon(asleep(DOT_GOOD)), 'set-asleep': tray_icon(asleep(DOT_SET)),
        'waiting': tray_icon(ring=0), 'error': tray_icon(DOT_BAD), 'off': tray_icon(),
    }
    for name, big in states.items():
        save_tray(os.path.join(folder, name + '.ico'), big)
        big.resize((32, 32), Image.LANCZOS).save(os.path.join(folder, name + '.png'), optimize=True)


def docs():
    images = os.path.join(ROOT, 'docs', 'images')
    save(tile(1024, 0.92).resize((512, 512), Image.LANCZOS), images, 'logo.png')
    # The README's banner: the tile, the name, and the tagline.
    w, h = 1600, 560
    out = night(w, h)
    glow = Image.new('RGBA', (w, h), (0, 0, 0, 0))
    ImageDraw.Draw(glow).ellipse([60, 40, 620, 520], fill=ORANGE + (90,))
    out.alpha_composite(glow.filter(ImageFilter.GaussianBlur(90)))
    dots = ImageDraw.Draw(out)
    for gx in range(0, w, 40):
        for gy in range(0, h, 40):
            dots.ellipse([gx - 1, gy - 1, gx + 1, gy + 1], fill=(255, 255, 255, 18))
    mark = tile(1024, 0.92).resize((380, 380), Image.LANCZOS)
    out.alpha_composite(mark, (130, (h - mark.height) // 2))
    d = ImageDraw.Draw(out)
    wordmark(d, (580, 150), 150)
    sub = ImageFont.truetype(FONT, 44)
    d.text((586, 345), 'Fast  ·  Free  ·  Open source VPN', font=sub, fill=(226, 232, 240, 255))
    small = ImageFont.truetype(FONT, 30)
    d.text((588, 420), 'Android  ·  Android TV  ·  Windows  ·  macOS  ·  Linux', font=small, fill=(255, 182, 102, 255))
    save(rounded(out, 36), images, 'banner.png')


def main():
    square = tile(1024)
    android()
    tv_banner()
    macos()
    write_ico(os.path.join(APP, 'windows', 'runner', 'resources', 'app_icon.ico'), square)
    write_ico(os.path.join(CORE, 'internal', 'desktop', 'appicon.ico'), square)
    save(tile(1024, 0.94).resize((192, 192), Image.LANCZOS), CORE, 'internal', 'service', 'web', 'icon.png')
    save(tile(1024, 0.94).resize((256, 256), Image.LANCZOS), APP, 'assets', 'logo.png')
    save(tile(1024, 0.94).resize((512, 512), Image.LANCZOS), APP, 'linux', 'runner', 'resources', 'app_icon.png')
    tray()
    docs()


if __name__ == '__main__':
    main()
