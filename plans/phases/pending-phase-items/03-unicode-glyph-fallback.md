# Pending — Phase 3: Unicode / IPA glyph fallback + URL font flags

> **Parent:** [`README.md`](README.md)  
> **Status:** done (2026-08-05)  
> **Estimated effort:** 2–5 days  
> **Prior plan coverage:** **Yes** — Phase 19 `--font-path` / `--use-system-fonts`; remote WOFF2 **out of scope**. Gap: missing-glyph fallback + URL smoke not wiring fonts  

---

## Overview

Ana IPA (`ˈ` `ɾ` etc.) mangles because **Liberation Sans lacks glyphs** and
fallback does not pick DejaVu/Noto. Spanish orthography (`á`) is fine; IPA is
not “Spanish fonts missing” — it is **Unicode coverage**.

Prefer local/system fallback over remote webfonts (Phase 9).

### Smoke proof

```sh
./bin/gowkhtmltopdf --use-system-fonts \
  'https://en.wikipedia.org/wiki/Ana_de_Armas' output/wiki-ana-de-armas.pdf
# or explicit: --font-path /usr/share/fonts/truetype/dejavu
```

Expect pronunciation line closer to `[ˈana ˈselja ðe ˈaɾmas ˈkaso]` (not `È` / `~` junk).

---

## Phase 3 checklist

### 3.1 Glyph fallback

- [x] When measuring/painting, if primary face `GlyphID(r)==0`, try next faces in CSS `font-family` list then registry/system fallback face that has the glyph (`faceForRune` → `Registry.FindWithGlyph`)
- [x] Prefer a known Unicode-capable face (e.g. DejaVu Sans) when present on `--font-path` / system scan — document which (score boost for dejavu/noto/freesans)
- [x] Do not invent glyphs; tofu/box only if no face has the codepoint
- [x] Tests: string containing U+02C8 / U+027E with Liberation primary + DejaVu on path → correct advances / embedded subset includes fallback face (`TestIPAGlyphRegistryFallback`)

### 3.2 URL / samples wiring

- [x] Document recommended `--use-system-fonts` / `--font-path` for URL mode (`documentation/fonts.md`; Phase 5 may refine recipe)
- [x] `make samples` wiki smoke passes `--use-system-fonts` (product call; keep **no** `--simplify-dom`)
- [x] Update `documentation/fonts.md` IPA / phonetic note

### 3.3 Honesty

- [x] Matrix / fonts.md: remote WOFF2 still unsupported; local fallback is the supported path
- [x] Fidelity: IPA quality depends on configured fonts

### 3.4 Gates

- [x] `make lint` → pass
- [x] `make test` → pass
- [x] Smoke with system/DejaVu fonts; note outcome beside this section
- [x] Status → done

**Smoke note (2026-08-05):** unit proof via DejaVu path + `TestIPAGlyphRegistryFallback`. Live Ana regeneration uses `--use-system-fonts` in `make samples`. Pronunciation glyphs no longer force Liberation-only tofu when DejaVu/Noto is on the registry.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 19 registry | Face list |
| Phase 5 URL docs | Discoverability |

---

## Out of scope

- Auto-download Google Fonts / WOFF2 (→ Phase 9)
- Bundling full Noto in binary
