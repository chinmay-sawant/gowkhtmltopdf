# Pending — Phase 3: Unicode / IPA glyph fallback + URL font flags

> **Parent:** [`README.md`](README.md)  
> **Status:** not started  
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

- [ ] When measuring/painting, if primary face `GlyphID(r)==0`, try next faces in CSS `font-family` list then registry/system fallback face that has the glyph
- [ ] Prefer a known Unicode-capable face (e.g. DejaVu Sans) when present on `--font-path` / system scan — document which
- [ ] Do not invent glyphs; tofu/box only if no face has the codepoint
- [ ] Tests: string containing U+02C8 / U+027E with Liberation primary + DejaVu on path → correct advances / embedded subset includes fallback face

### 3.2 URL / samples wiring

- [ ] Document recommended `--use-system-fonts` / `--font-path` for URL mode (Phase 5 may own final recipe)
- [ ] Decide whether `make samples` wiki smoke should pass `--use-system-fonts` (product call; keep **no** `--simplify-dom`)
- [ ] Update `documentation/fonts.md` IPA / phonetic note

### 3.3 Honesty

- [ ] Matrix / fonts.md: remote WOFF2 still unsupported; local fallback is the supported path
- [ ] Fidelity: IPA quality depends on configured fonts

### 3.4 Gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] Smoke with system/DejaVu fonts; note outcome beside this section
- [ ] Status → done

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
