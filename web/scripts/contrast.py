"""WCAG 2.x contrast for the whence palette. oklch -> sRGB -> relative luminance.

    python3 scripts/contrast.py

Asserts the AA floor (4.5:1) for every foreground on every surface, and for
every rung of the opacity ladder. A palette edit that breaks accessibility
fails here rather than in review.

Values must stay in sync with src/index.css. DESIGN.md quotes this script's
output, so re-run it and update that table when the palette moves.
"""

import math

AA = 4.5


def oklch_to_linear_srgb(L, C, h_deg):
    h = math.radians(h_deg)
    a, b = C * math.cos(h), C * math.sin(h)
    l_ = L + 0.3963377774 * a + 0.2158037573 * b
    m_ = L - 0.1055613458 * a - 0.0638541728 * b
    s_ = L - 0.0894841775 * a - 1.2914855480 * b
    l, m, s = l_**3, m_**3, s_**3
    r = +4.0767416621 * l - 3.3077115913 * m + 0.2309699292 * s
    g = -1.2684380046 * l + 2.6097574011 * m - 0.3413193965 * s
    bl = -0.0041960863 * l - 0.7034186147 * m + 1.7076147010 * s
    return [max(0.0, min(1.0, v)) for v in (r, g, bl)]


def luminance_linear(rgb):
    """WCAG relative luminance wants LINEAR sRGB, so no gamma decode here."""
    return 0.2126 * rgb[0] + 0.7152 * rgb[1] + 0.0722 * rgb[2]


def encode(rgb):
    """Linear -> gamma-encoded sRGB. Browsers composite alpha in this space."""
    f = lambda v: 12.92 * v if v <= 0.0031308 else 1.055 * v ** (1 / 2.4) - 0.055
    return [f(v) for v in rgb]


def decode(rgb):
    f = lambda v: v / 12.92 if v <= 0.04045 else ((v + 0.055) / 1.055) ** 2.4
    return [f(v) for v in rgb]


def contrast(l1, l2):
    hi, lo = max(l1, l2), min(l1, l2)
    return (hi + 0.05) / (lo + 0.05)


def ratio(fg, bg):
    return contrast(
        luminance_linear(oklch_to_linear_srgb(*fg)),
        luminance_linear(oklch_to_linear_srgb(*bg)),
    )


def ratio_alpha(alpha, fg, bg):
    """Foreground at `alpha` over `bg`, composited the way a browser does."""
    f, b = encode(oklch_to_linear_srgb(*fg)), encode(oklch_to_linear_srgb(*bg))
    mixed = [alpha * f[i] + (1 - alpha) * b[i] for i in range(3)]
    return contrast(luminance_linear(decode(mixed)), luminance_linear(decode(b)))


# --- the palette, mirroring src/index.css -------------------------------
SURFACES = {
    "basin": (0.145, 0, 0),
    "sediment": (0.205, 0, 0),
    "terminal": (0.115, 0, 0),
}

# Monochrome: ochre, verdigris and cinnabar are all silt now. Listed
# separately anyway, so that giving one of them a hue again shows up here.
FOREGROUNDS = {
    "silt": (0.985, 0, 0),
    "ochre": (0.985, 0, 0),
    "verdigris": (0.985, 0, 0),
    "cinnabar": (0.985, 0, 0),
    "primary": (0.922, 0, 0),
    "dim / muted-foreground": (0.708, 0, 0),
}

# The opacity ladder that carries state now that no token carries a hue.
LADDER = [1.0, 0.80, 0.55]

failures = []


def table(title, rows, measure):
    print(f"\n{title}")
    print(f"{'':24}" + "".join(f"{n:>12}" for n in SURFACES))
    worst = (99.0, "", "")
    for label, key in rows:
        line = f"{label:24}"
        for sname, surface in SURFACES.items():
            r = measure(key, surface)
            line += f"{r:>12.2f}"
            if r < worst[0]:
                worst = (r, label, sname)
        print(line)
    print(f"  worst: {worst[1]} on {worst[2]} = {worst[0]:.2f}")
    if worst[0] < AA:
        failures.append(f"{worst[1]} on {worst[2]} = {worst[0]:.2f}")
    return worst


table(
    "tokens",
    [(n, v) for n, v in FOREGROUNDS.items()],
    lambda fg, bg: ratio(fg, bg),
)

table(
    "opacity ladder (silt at alpha)",
    [(f"silt/{int(a * 100)}", a) for a in LADDER],
    lambda a, bg: ratio_alpha(a, FOREGROUNDS["silt"], bg),
)

# The white button's own label — a pairing neither table covers, since both
# backgrounds are page surfaces and this one is the primary fill.
btn = ratio((0.205, 0, 0), (0.922, 0, 0))
print(f"\nprimary-foreground on primary (the button label): {btn:.2f}")
if btn < AA:
    failures.append(f"button label = {btn:.2f}")

print()
assert not failures, "WCAG AA failures:\n  " + "\n  ".join(failures)
print(f"all pairs and ladder rungs clear WCAG AA ({AA}:1)")
