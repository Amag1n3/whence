---
name: whence
description: Git remembers what changed. whence remembers why.
colors:
  basin: "oklch(0.145 0 0)"
  sediment: "oklch(0.205 0 0)"
  terminal: "oklch(0.115 0 0)"
  silt: "oklch(0.985 0 0)"
  ochre: "oklch(0.985 0 0)"
  ochre-selection: "oklch(0.985 0 0 / 22%)"
  ochre-ring: "oklch(0.985 0 0 / 70%)"
  verdigris: "oklch(0.985 0 0)"
  cinnabar: "oklch(0.985 0 0)"
  dim: "oklch(0.708 0 0)"
  primary: "oklch(0.922 0 0)"
  primary-foreground: "oklch(0.205 0 0)"
  secondary: "oklch(0.269 0 0)"
  border: "oklch(1 0 0 / 10%)"
  input: "oklch(1 0 0 / 15%)"
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
  none: "0px"
  sm: "0px"
  md: "0px"
  lg: "0px"
  xl: "0px"
  full: "999px"
spacing:
  xs: "6px"
  sm: "12px"
  md: "24px"
  lg: "44px"
  xl: "96px"
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.primary-foreground}"
    typography: "{typography.ui}"
    rounded: "{rounded.none}"
    padding: "10px 24px"
  layer-eyebrow:
    textColor: "{colors.dim}"
    typography: "{typography.eyebrow}"
  terminal-surface:
    backgroundColor: "{colors.terminal}"
    textColor: "{colors.silt}"
    rounded: "{rounded.none}"
    padding: "20px"
  contents-index-item-active:
    borderLeft: "2px solid {colors.ochre}"
    textColor: "{colors.silt}"
  contents-index-item:
    borderLeft: "2px solid transparent"
    textColor: "{colors.dim}"
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

The landing page is **an introduction and a set of doors**, and nothing else:
the pitch, the terminal demo beside it, then cards to `/install`, `/docs`,
`/why`, `/faq` and the repo.

It used to be **a core sample** — eleven numbered strata scrolled top to bottom
with a depth-gauge rail down the left edge, the conceit being that the reason a
thing is the way it is lies underneath it and has to be dug up. That was a good
idea about the *product* and a bad idea about the *page*: it buried the install
command at depth 07, so every reader had to sit through the whole argument to
reach the thing they came for. The argument moved to `/why` on 2026-08-09 and
the rail was deleted with it.

Numbered sections survive on `/why` and the reference pages, where a reader has
already agreed to read in order. They are a table of contents now, not a depth
gauge.

**Every surface and text token is chroma 0.** Repainted 2026-08-09 onto the
shadcn `neutral` dark scale — which `components.json` had named as this
project's `baseColor` from the first commit, while `index.css` overrode all of
it with a 30–80° mineral band.

That band is what the repaint removed, and it is worth being precise about why,
because the old rationale was not stupid. Tinting the *surfaces* warm was
defensible. Tinting the surfaces **and** the body text warm at the same time was
not: two warm layers with nothing neutral between them left no reference point,
so the eye read the result as a sepia wash rather than as a deliberate hue. The
former rule "body text is `silt`, never white" was the load-bearing half of that
mistake — `silt` is now `oklch(0.985 0 0)`, near-white, and the page got *more*
legible for it, not less (12.74 → 18.96 against the page).

**As of 2026-08-09 there is no colour at all.** The single amber accent that
survived the first repaint went too. Every token is chroma 0.

`ochre`, `verdigris` and `cinnabar` still exist as names — 114 call sites use
them, and the names still say what the thing *means*, which is the part worth
keeping. None of them is a colour any more; all three resolve to the same
near-white. **Do not read them as hues.**

That means state cannot be carried by colour, so it is carried three ways that
were always already present:

1. **An opacity ladder.** Severity is brightness: intact `silt/55`, eroded
   `silt/80`, orphaned or failing full-strength `silt`. A finding gets brighter
   the worse it is. Used in `Gate` and `DriftDemo`.
2. **The icon and the label.** `✓` / `✗`, `works` / `not built`, the integrity
   percentage. These were doing the real work all along; the colour was
   reinforcement.
3. **Inversion, once.** The `exit 1` badge flips to a light fill with dark
   text. It is the loudest move available without hue, so it is spent on the
   single element that has earned it — the exit code, which the page calls the
   product. Nothing else on the site may invert.

The risk this accepts: scanning is slower. You can no longer find the failing
row by looking for red. That is the cost of the direction, taken deliberately —
not an oversight to be patched later by sneaking one hue back in for "just this
one status".

Depth is carried by **luminance, not hue**: cards sit above the page
(`0.205` on `0.145`), terminals sit below it (`0.115`). That reads as physical
without a single border, and it is the one part of the core-sample conceit that
survived the repaint intact.

## Corners: square, and why that is not just a trend

**Every radius in the ramp is `0`.** Set 2026-08-09.

Zero-radius hairline layouts are currently one of the three looks that
generated design falls into by default, so it is worth being explicit that this
one is argued rather than borrowed: everything this tool is *about* is
rectangular. A terminal, a diff, a line range, an exit code, a content hash.
The page had rounded cards describing square things. Squaring the corners costs
nothing and makes the chrome agree with its own subject.

`rounded-full` survives in exactly two situations, and adding a third needs a
reason written next to it:

1. **A thing that is genuinely a circle** — status dots, the logo mark.
2. **A capsule that encodes a continuous quantity** — the decay meter in
   `DriftDemo`, where the shape is doing the same job as a progress bar.

Pill-shaped *labels* were squared. A label on this site is a stamp — `works`,
`not built`, `exit 1` — and a stamp has corners. A pill reads as a chip you can
dismiss, which is the wrong affordance for a status that is not yours to change.

Surface mode is **Persuade**: a visitor decides whether this tool is real and
whether its author is honest. The tone that earns that is not enthusiasm — it is
a project willing to print the number that would kill it.

## Colors

| Token | Role | Rule |
|---|---|---|
| `basin` | the page, deepest layer | Page background. Never a card. |
| `sediment` | raised surfaces | Cards, wells, inset panels. |
| `terminal` | the cut face of the core | Code and terminal blocks only. |
| `silt` | body text | The default foreground. Near-white, chroma 0. |
| `ochre` | identity, status | Accents, markers, links, the logo dot, focus rings. Never a button fill. |
| `primary` | button fills | Near-white with dark text. The registry default. |
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

`PRODUCT.md` commits to **WCAG 2.2 AA**, so these are obligations. Re-measured
2026-08-09 after the repaint, against all three background tokens:

| Foreground | on `basin` | on `sediment` | on `terminal` |
|---|---|---|---|
| `silt` / `ochre` / `verdigris` / `cinnabar` | 18.96 | 17.16 | 19.52 |
| `primary` | 15.72 | 14.22 | 16.18 |
| `dim` / `muted-foreground` | 7.63 | **6.91** | 7.86 |

Plus `primary-foreground` on `primary` — the white button's own label — at 14.22.

The table collapsed when the palette went monochrome: the four foreground
tokens are now the same value, so they measure the same. The floor moved *up*
as a result — the worst pair in the system used to be `cinnabar` on `sediment`
at 6.19, and is now `dim` on `sediment` at 6.91.

The opacity ladder is the thing to watch instead, because a rung is a
composite and not a token — nothing in the palette table describes it.
Measured, compositing in gamma-encoded sRGB the way a browser does:

| Rung | on `basin` | on `sediment` | on `terminal` |
|---|---|---|---|
| `silt` (full) | 18.96 | 17.16 | 19.52 |
| `silt/80` | 12.07 | 11.22 | 12.30 |
| `silt/55` | 6.04 | **5.90** | 6.04 |

`silt/55` on `sediment` is the floor of the whole system at 5.90. It clears AA
with room, but **a quieter fourth rung needs re-measuring before it ships** —
the ladder is the one place where adding a step silently costs contrast, since
it looks like a styling choice rather than a colour change.

**The repaint fixed the one thing that was failing.** `dim` used to be the
palette's sole AA violation and it failed everywhere it was used; it survived at
`L 0.598` with a margin of 0.01 against `sediment`, which meant any new surface
lighter than `sediment` would have silently broken it. Dropping the hue and
moving to the neutral scale's `0.708` took it to 6.91 — the constraint is now
comfortable rather than knife-edge. **Do not lower it back.**

The worst pair in the system is now `cinnabar` on `sediment` at 6.19, against a
floor of 4.5. Everything has real headroom, and the near-white foreground is a
large part of why.

The measurement is reproducible, not asserted. `scripts/contrast.py` does the
oklch→sRGB→luminance conversion and **asserts** the 4.5 floor for every token
pairing and every ladder rung, so a palette edit that breaks AA exits non-zero
instead of reaching review:

```
python3 scripts/contrast.py
```

Its values mirror `src/index.css` by hand — there is no import between them, so
moving a token means moving it in both places. That is the one seam in this
check, and it is why the script lists `ochre`, `verdigris` and `cinnabar`
separately even though all three are the same white today: if one of them is
given a hue again, it shows up as its own row rather than hiding behind `silt`.

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

- **`Layer`** — the section primitive **of the landing page**. Depth marker,
  eyebrow, title, optional lede, then children. Every top-level section of the
  core sample is a `Layer`; new landing sections must use it rather than
  hand-rolling a heading.
- **`DocPage` / `DocSection`** — the shell for the three reference pages
  (`/install`, `/docs`, `/faq`). Hero, a sticky contents index, numbered
  sections. **Deliberately not `Layer` and deliberately no strata rail:** depth-
  is-time belongs to the core-sample conceit, and reference material is not a
  layer of anything. The contents index is the rail's analogue — same position,
  same monospace, different claim. A fourth reference page uses this; it does not
  hand-roll the layout again, which is the mistake `Chrome` and this component
  both exist to prevent.
- **`Code`** — code and terminal blocks, shared by all four pages. Note it ships
  at `rounded-md`, not the `xl` this file names for terminal surfaces. The code
  is right and this line is the record of the discrepancy.
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

- Reuse `Layer` (landing), `DocPage` (reference pages), `Reveal`, and the shell
  constant. New sections that hand-roll their own spacing drift immediately.
- Add a page to the `NAV` list in `Chrome.tsx` and to `rollupOptions.input`.
  Both, or the page is unreachable from one of header/footer, or not built at
  all. The single `NAV` array is why the header and footer cannot disagree.
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
