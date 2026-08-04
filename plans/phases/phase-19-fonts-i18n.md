# Phase 19 - Fonts / i18n: Discovery, Folder Fonts, CJK / Unicode

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** done (core) on `feature/tier-2`  
> **Estimated effort:** 1–3 months  
> **Depends on:** Phase 12 font registry  
> **Unblocks:** localization, CJK reports, better Wikipedia language lists  
> **Tier:** 2 #7 · **Constraint:** stdlib-only TTF parse/embed; no HarfBuzz

---

## Overview

Beyond bundled Liberation faces: **discover fonts** from a user-supplied folder and optional system locations, map CSS `font-family`, and add a **Unicode / CJK path** (CID/Type0 or multi-face) with honest shaping limits. Localization here means **using the right fonts for the document language**, not shipping translation strings for the CLI.

## Executive Summary

| Today | Target |
|-------|--------|
| One Regular (+ bold from 12) embedded | Multiple families from registry + disk |
| WinAnsi / Latin-1 fold → `?` | Broader Unicode embedding |
| No `@font-face` | Local `src` subset with ACL |
| No system scan | Opt-in paths; documented |

---

## Phase 19 checklist

### 19.1 Font discovery

- [ ] Settings/CLI: `--font-path` / dotted key for **one or more directories** to scan for `.ttf`/`.otf` (TTF subset first; OTF only if parser supports)
- [ ] Scan is **opt-in** (do not read all system fonts by default - startup + privacy)
- [ ] Optional documented defaults per OS (e.g. `/usr/share/fonts`) behind explicit flag `--use-system-fonts`
- [ ] Index: family name from name table → file path; weight/style heuristics from subfamily
- [ ] Tests with fonts under `testdata/fonts/` (commit small OFL samples only)
- [ ] Path: new `internal/fonts` or extend `internal/pdf/fonts.go`
- [ ] Security: font path subject to local ACL rules when loading files

### 19.2 CSS mapping

- [ ] `font-family` list walks registry + discovered faces
- [ ] Generics: `sans-serif`, `serif`, `monospace` map to configured defaults
- [ ] Fallback chain ends at bundled Liberation Sans
- [ ] Tests: request “Noto Sans” from folder → that face used

### 19.3 `@font-face` local subset

- [ ] Parse `@font-face` { font-family; src: url(...) format }
- [ ] Support `url(file)` / relative path under allowlist; **no** remote download unless load policy explicitly allows and is reviewed
- [ ] Register face for document lifetime
- [ ] Matrix §4 `@font-face` status update
- [ ] Path: `internal/css` + font registry

### 19.4 Unicode / CJK embedding

- [ ] Design: PDF Type0/CID-keyed font **or** multiple simple fonts with ToUnicode - pick one and document
- [ ] Embed subset of glyphs actually used (CJK subsets can be large - budget pages)
- [ ] Layout width using face metrics for CJK code points (advance per glyph; no vertical text yet)
- [ ] `[~]` Vertical writing mode - deferred
- [ ] Test: Japanese/Chinese sample string renders (not all `?`) in PDF
- [ ] Wikipedia language list smoke improves when faces present

### 19.5 Localization product notes

- [ ] Document how to point at regional fonts for invoices (folder layout example)
- [ ] CLI help: font-path flags
- [ ] Library API: settings keys for font paths
- [ ] Explicit non-claim: full Arabic/Indic **reordering/shaping** not supported (no HarfBuzz)

### 19.6 Image mode

- [ ] Image-mode raster uses same resolved face as PDF when phase 15 done
- [ ] Missing glyph: tofu/`?` consistent policy

### 19.7 Docs

- [ ] Matrix Unicode / font-family / @font-face sections
- [ ] Fidelity: CJK “best-effort horizontal” language
- [ ] README CJK deferred row updated when shipped
- [ ] Threat model: font file parse is untrusted input - fuzz or size caps note

### 19.8 Closure gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] CJK fixture + folder-font fixture green
- [ ] Parent Phase 19 checked
- [ ] Next: **Phase 20** or **21**

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 12 registry | Phase 21 non-Latin sites |
| Phase 15 raster | Consistent image CJK |

---

## Out of scope

- WOFF/WOFF2 (unless pure-Go decode added later)
- Complex-script shaping
- Auto-download Google Fonts
- Shipping multi-megabyte CJK faces in default binary (prefer user folder)
