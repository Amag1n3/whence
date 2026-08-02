---
name: whence
description: Git remembers what changed. whence remembers why.
colors:
  basin: "oklch(0.152 0.014 62)"
  sediment: "oklch(0.204 0.016 62)"
  terminal: "oklch(0.118 0.012 62)"
  silt: "oklch(0.858 0.018 78)"
  ochre: "oklch(0.762 0.118 68)"
  verdigris: "oklch(0.688 0.052 154)"
  cinnabar: "oklch(0.622 0.138 36)"
  dim: "oklch(0.598 0.018 70)"
  border: "oklch(1 0 0 / 8%)"
typography:
  display-xl:
    fontFamily: "Archivo Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "clamp(2.4rem, 6vw, 4.4rem)"
    fontWeight: 600
    lineHeight: 1.04
    letterSpacing: "-0.012em"
  display-lg:
    fontFamily: "Archivo Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "clamp(2.1rem, 4.6vw, 3.4rem)"
    fontWeight: 600
    lineHeight: 1.08
    letterSpacing: "-0.012em"
  display:
    fontFamily: "Archivo Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "clamp(1.65rem, 2.7vw, 2.3rem)"
    fontWeight: 600
    lineHeight: 1.15
    letterSpacing: "-0.012em"
  display-sm:
    fontFamily: "Archivo Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "clamp(1.3rem, 2vw, 1.6rem)"
    fontWeight: 600
    lineHeight: 1.2
    letterSpacing: "-0.012em"
  lede:
    fontFamily: "Instrument Sans Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "21px"
    fontWeight: 400
    lineHeight: 1.55
    letterSpacing: "normal"
  lede-sm:
    fontFamily: "Instrument Sans Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "19px"
    fontWeight: 400
    lineHeight: 1.55
    letterSpacing: "normal"
  body:
    fontFamily: "Instrument Sans Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "15.5px"
    fontWeight: 400
    lineHeight: 1.7
    letterSpacing: "normal"
  body-sm:
    fontFamily: "Instrument Sans Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "14.5px"
    fontWeight: 400
    lineHeight: 1.68
    letterSpacing: "normal"
  ui:
    fontFamily: "JetBrains Mono Variable, ui-monospace, SFMono-Regular, Menlo, monospace"
    fontSize: "13.5px"
    fontWeight: 500
    lineHeight: 1.5
    letterSpacing: "-0.01em"
  code:
    fontFamily: "JetBrains Mono Variable, ui-monospace, SFMono-Regular, Menlo, monospace"
    fontSize: "12.5px"
    fontWeight: 400
    lineHeight: 1.65
    letterSpacing: "normal"
  eyebrow:
    fontFamily: "JetBrains Mono Variable, ui-monospace, SFMono-Regular, Menlo, monospace"
    fontSize: "11px"
    fontWeight: 400
    lineHeight: 1.9
    letterSpacing: "0.2em"
  pull-quote:
    fontFamily: "Instrument Sans Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "23px"
    fontWeight: 500
    lineHeight: 1.5
    letterSpacing: "normal"
  card-title:
    fontFamily: "Archivo Variable, ui-sans-serif, system-ui, sans-serif"
    fontSize: "18px"
    fontWeight: 600
    lineHeight: 1.25
    letterSpacing: "-0.012em"
  badge:
    fontFamily: "JetBrains Mono Variable, ui-monospace, SFMono-Regular, Menlo, monospace"
    fontSize: "10px"
    fontWeight: 400
    lineHeight: 1.4
    letterSpacing: "0.02em"
rounded:
  xs: "2px"
  sm: "6px"
  md: "8px"
  lg: "10px"
  xl: "14px"
  full: "999px"
spacing:
  xs: "6px"
  sm: "12px"
  md: "24px"
  lg: "44px"
  xl: "96px"
components:
  button-primary:
    backgroundColor: "{colors.ochre}"
    textColor: "oklch(0.178 0.030 68)"
    rounded: "{rounded.md}"
    padding: "10px 20px"
  layer-eyebrow:
    textColor: "{colors.dim}"
    typography: "{typography.eyebrow}"
  terminal-surface:
    backgroundColor: "{colors.terminal}"
    textColor: "{colors.silt}"
    rounded: "{rounded.xl}"
    padding: "20px"
---

# whence — design system

Derived from the shipped implementation (`web/src/index.css`, `App.tsx`,
`components/`), not from intentions. Where code and this file disagree, the code
is right and this file is stale.

> [!warning] **This file must stay in `web/`. Do not move it to the repo root.**
> impeccable's two scripts disagree about where it lives: `context.mjs` looks at
> the repo root (where `PRODUCT.md` correctly sits), while `detect.mjs` resolves
> it from the target file's package root, which is `web/`. Moving it up satisfies
> the first and **silently disables every design-system check in the second** —
> it reports zero findings rather than reporting that it cannot find the file.
> That was done once, on 2026-08-03, and caught only by injecting a deliberately
> illegal value to see whether the linter still complained. A silently disabled
> check is worse than a loudly missing one, so the split stands.

> [!note] The type-ramp check is weak, and knowing why matters
> Documenting four fluid `clamp()` display steps spanning roughly 21px→70px, plus
> discrete steps from 10px to 23px, means the ramp now covers nearly every value
> a person would plausibly type. Verified: a stray `37px` passes because it falls
> inside a clamp range; only something like `7px` still trips it.
> **So "detector clean" is close to meaningless for font sizes here.** It catches
> outliers, not drift. Read the ramp yourself rather than trusting a green run —
> the same failure as a metric that returns PASS for every input.

## Overview

The page is **a core sample**. Depth is time: you scroll down through strata,
each layer numbered like a core log, and the further down you go the more the
argument has settled. That conceit is not decoration — it is the same claim the
product makes, that the reason a thing is the way it is lies underneath it and
has to be dug up.

Everything sits in the **30–80° hue band** — iron, clay, bone. The page should
read as mineral rather than as a screen. The single most load-bearing
consequence: **body text is `silt`, never white.** White-on-near-black is the
default this design deliberately leaves behind, and reaching for `#fff` undoes
the whole palette.

Surface mode is **Persuade**: a visitor decides whether this tool is real and
whether its author is honest. The tone that earns that is not enthusiasm — it is
a project willing to print the number that would kill it.

## Colors

| Token | Role | Rule |
|---|---|---|
| `basin` | the page, deepest layer | Page background. Never a card. |
| `sediment` | raised surfaces | Cards, wells, inset panels. |
| `terminal` | the cut face of the core | Code and terminal blocks only. |
| `silt` | body text | The default foreground. Not white. |
| `ochre` | identity, primary | Accents, markers, links, the logo dot, focus rings. |
| `verdigris` | **"this works" and nothing else** | Never identity, never decoration. |
| `cinnabar` | violation, orphaned, lost | Never decoration. |
| `dim` | secondary text | Eyebrows, metadata, footers. |

**The two semantic colors are the sharpest rule in this system.** `verdigris`
means a thing functions; `cinnabar` means a record was damaged or a decision was
lost. Using either because a section needed visual variety makes the page lie
about its own subject. If a third accent is genuinely needed, it comes from the
ochre family, not from the semantic pair.

Colors are authored in **oklch** and that is the normative source. Do not convert
to hex; the palette was picked for perceptual evenness across the hue band and
round-tripping through sRGB loses it.

### Contrast — measured, not assumed

`PRODUCT.md` commits to **WCAG 2.2 AA**, so these are obligations. Measured
2026-08-03 against all three background tokens:

| Foreground | on `basin` | on `sediment` | on `terminal` |
|---|---|---|---|
| `silt` | 12.74 | 11.66 | 13.19 |
| `ochre` | 9.02 | 8.26 | 9.34 |
| `verdigris` | 7.19 | 6.59 | 7.45 |
| `muted-foreground` | 6.01 | 5.51 | 6.23 |
| `cinnabar` | 5.13 | 4.69 | 5.31 |
| `dim` | 4.92 | **4.51** | 5.10 |

**`dim` was the only failure and it failed everywhere.** At `L 0.562` it measured
4.24 / 3.89 / 4.40 — under the 4.5 required for normal text, and every one of its
uses (eyebrows at 11px, depth markers, the footer at 12.5px) is normal text, so
the 3.0 large-text allowance never applied. Raised to `L 0.598`, the minimum that
clears 4.5 against `sediment`, the worst case. **Do not lower it back**; it now
sits with almost no margin, so any new background lighter than `sediment` needs
re-measuring rather than assuming.

Everything else has real headroom. The mineral palette's avoidance of pure white
costs nothing here — `silt` at 12.74 is far above the bar.

## Typography

Three families, each with one job:

- **Archivo Variable** — headings only, and always via the width axis. Set at
  `font-stretch: 112%`, weight 600, letter-spacing `-0.012em`. Expanded width
  against tight tracking is the type's entire personality: bands that are wide
  but not loose. The variable font is loaded from `wdth.css`, not the default
  stylesheet, because the width axis is the point.
- **Instrument Sans Variable** — body. 15.5px/1.7 for lede text, 14.5px/1.68 for
  supporting prose.
- **JetBrains Mono Variable** — code, terminal output, eyebrows, depth markers,
  and every number. Monospace here is not "code font", it is *measurement*.

**The ramp is thirteen steps, and that is two or three too many.** Four fluid
display sizes, two lede sizes, two body sizes, two monospace sizes, and three
role-specific one-offs (`pull-quote`, `card-title`, `badge`) that each appear
exactly once on the landing page. The half-pixel values (`15.5px`, `14.5px`,
`13.5px`, `12.5px`) are deliberate and worth keeping — they sit between the sizes
a default scale would give, and they are part of why the page does not read as
Tailwind defaults.

The one-offs are the debt. `17.5px→19px`, `16.5px`, `20px→23px` and `18px` all sit
inside a 6px band doing similar work, and a consolidation pass could plausibly
fold them onto `lede`/`lede-sm`/`body` without anyone noticing. That has not been
done, because it means editing four landing sections to fix an advisory-only
finding. Recorded here so it is a known cost rather than an accident.

Reaching for `text-sm` instead of a documented step is the actual error; adding a
fourteenth step needs a reason written down next to it.

Prose lines cap between `46ch` and `56ch`. Headings cap around `19ch`, which is
what keeps them reading as bands rather than sentences. `text-wrap: balance` on
headings, `pretty` on paragraphs.

## Layout

- Shell: `max-width: 1280px`, padding `24px` rising to `40px` at `sm`.
- Sections: `py-16` rising to `py-24` at `sm`, separated by a single
  `border-white/[0.07]` hairline. The hairline *is* the stratum boundary — it is
  the only divider in the system and it should never be thickened for emphasis.
- **The strata rail** is fixed on the left at `lg` and up, offsetting content by
  `pl-14`. It lists every layer in page order. Rail order and page order are the
  same list in the source (`STRATA`), and that is deliberate: depth is the one
  thing both have to agree on.
- **Strictly left-aligned.** There is exactly one centred line on the entire site
  and its code comment says it earns it. A second centred element is a bug.
- Navigation scrolls without writing a hash into the address bar. The `href`
  stays so it works without JS and reads correctly to a screen reader.

## Elevation & Depth

Depth is carried by three things and never by a border-radius bump:

- **`.grain`** — a fixed fractal-noise overlay at `0.038` opacity across the
  whole viewport. Sediment texture. On-concept, not merely anti-flatness.
- **`.lit`** — the only shadow recipe: a `6%` white inset top edge plus two
  stacked drops. Reads as a lit face rather than as a floating card.
- **Hairlines at `white/[0.07]`** — every boundary in the system.

No blurs, no glows, no gradient fills behind text.

## Shapes

Radius base is `0.5rem`, stepped `sm/md/lg/xl` by ±2 and +6px. Pills (`999px`)
are reserved for status dots and scrollbar thumbs. Terminal and code surfaces
take the largest radius (`xl`), which is what separates "cut face of the core"
from an ordinary card.

## Components

- **`Layer`** — the section primitive. Depth marker, eyebrow, title, optional
  lede, then children. Every top-level section on the site is a `Layer`; new
  sections must use it rather than hand-rolling a heading.
- **`Reveal`** — scroll-in, `opacity 0 → 1` with an 18px rise, `once: true`.
  A page that re-animates on every scroll-by is a page that fights the reader.
  It already handles `prefers-reduced-motion` internally, so callers must not add
  their own guard.
- **`Terminal`** — mirrors the CLI's real output verbatim, by design. When the
  CLI's output changes this component becomes a lie until updated.
- **`DriftDemo`**, **`Gate`** — narrative components whose state labels and
  numbers are taken from the Go constants. Same rule: they are wrong the moment
  the code changes without them.
- **`Accordion`** — Radix, `type="multiple"`. The trigger marker rotates a plus
  into a minus rather than swapping glyphs, so the control reads as one object in
  two states.

## Do's and Don'ts

**Do**

- Reuse `Layer`, `Reveal`, and the shell constant. New sections that hand-roll
  their own spacing drift immediately.
- Keep the rail list and the page order identical.
- Let the ugly parts stay ugly and honest. The falsification panel — the one that
  says the project gets archived if a number stays zero — is the most persuasive
  element on the page precisely because it is not softened.
- Cap prose width. Full-bleed paragraphs are the fastest way to make this page
  look like a template.

**Don't**

- Don't use white for text. Use `silt`.
- Don't use `verdigris` or `cinnabar` decoratively. They are load-bearing.
- Don't add a second centred element.
- Don't thicken the hairline, add a second shadow recipe, or introduce a gradient.
- Don't write hashes into the URL for in-page navigation.
- Don't add a font. Three families, three jobs, and the third one is measurement.
