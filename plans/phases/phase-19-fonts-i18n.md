# Phase 19 - Fonts / i18n: Discovery, Folder Fonts, CJK / Unicode

> **Parent:** `plans/10-canonical-post-mvp-roadmap.md`  
> **Status:** done (core + shaping amendment) on `master` via #16 / #17  
> **Estimated effort:** 1–3 months  
> **Depends on:** Phase 12 font registry  
> **Unblocks:** localization, CJK reports, better Wikipedia language lists  
> **Tier:** 2 #7 · **Constraint:** stdlib-only TTF parse/embed; **no HarfBuzz**  
> **Amendment:** [`plans/amendments/2026-08-04-shaping-stdlib.md`](../amendments/2026-08-04-shaping-stdlib.md)

---

## Overview

Beyond bundled Liberation faces: **discover fonts** from a user-supplied folder
and optional system locations, map CSS `font-family` (including **per-rune**
fallback), embed **Unicode / CJK** via Type0/CID, and ship stdlib-safe
**Arabic presentation-form joining** + Hangul-capable fixture fonts. Honesty:
not OpenType GSUB/GPOS / HarfBuzz.

## Executive Summary

| Today (pre-19) | Target | Status (2026-08-05) |
|----------------|--------|---------------------|
| One Regular (+ bold from 12) | Multiple families from registry + disk | **Shipped** |
| WinAnsi / Latin-1 fold → `?` | Broader Unicode embedding | **Shipped** (Type0/CID BMP) |
| No folder scan | Opt-in `--font-path` / `--use-system-fonts` | **Shipped** |
| No complex script | Best-effort Arabic joining; Hangul via face | **Shipped** (stdlib) |
| CJK PDF garbled composites | Aligned glyf + strip hints | **Shipped** (#17) |
| `@font-face` | Local `src` under ACL | **Partial** (CSS parse; wiring/docs uneven) |

---

## Phase 19 checklist

### 19.1 Font discovery

- [x] Settings/CLI: `--font-path` / dotted key for one or more directories (`.ttf` + TrueType-flavored `.otf`)
- [x] Scan is **opt-in** (not all system fonts by default)
- [x] Optional `--use-system-fonts` with documented OS defaults
- [x] Index: family name from name table → file; weight/style heuristics
- [x] Tests / CI fonts under `testdata/fonts/` (OFL Noto Sans KR Hangul subset)
- [x] Path: `internal/pdf/registry.go` (+ convert/imageout wiring)
- [x] Security: font path subject to local ACL when loading files
- [x] Clear error for CFF / `OTTO` (CFF not supported)

### 19.2 CSS mapping

- [x] `font-family` list walks registry + discovered faces
- [x] Generics: `sans-serif`, `serif`, `monospace` → configured defaults
- [x] Fallback chain ends at bundled Liberation Sans
- [x] **Per-rune** face selection so mixed Han + Hangul lines work (Droid then Noto KR on fixture-27)
- [x] Tests: fixture-27 + `cjk_fallback_test.go`

### 19.3 `@font-face` local subset

- [x] Parse `@font-face` { font-family; src: url(...) } in `internal/css`
- [~] Support `url(file)` / relative path under allowlist end-to-end (docs claim; verify / harden as pending)
- [~] Register face for document lifetime consistently with `--font-path`
- [ ] Matrix §4 `@font-face` still “Not implemented” in places — **pending honesty update**
- [~] Remote / WOFF download — **out of scope** unless policy amended

### 19.4 Unicode / CJK embedding

- [x] PDF Type0 / CIDFontType2 Identity-H for BMP Unicode
- [x] Embed subset of glyphs actually used
- [x] Layout width using face metrics for CJK code points
- [x] `writing-mode: vertical-rl|lr` lite: stacked Latin upright + **90° rotated** ideographic/Hangul/kana
- [x] Test: Japanese/Chinese/Hangul sample renders (fixture-27) when faces on path
- [x] Subset fidelity (#17): **4-byte glyf/loca align**, preserve LSB, **strip hint bytecode** (no `fpgm`/`prep`/`cvt` in subset)
- [x] Keep full-em CJK punctuation metrics (match HTML+Droid; no fake `halt`)

### 19.5 Shaping (stdlib amendment)

- [x] Arabic presentation-form joining + Lam-Alef in `ShapeText` (not OpenType)
- [x] Existing RTL run reverse retained
- [x] Indic: NFC + honesty “not claimed” (no matra reordering)
- [x] Hangul: capable face via font-path; CI subset vendored
- [x] Docs: `documentation/fonts.md` shaping honesty language
- [~] OpenType `halt`/`palt` / GSUB/GPOS — **deferred** (would need HarfBuzz or a large pure-Go OT stack)

### 19.6 Localization product notes

- [x] Document regional fonts via `--font-path` (fonts.md + fixture README)
- [x] CLI help: font-path / use-system-fonts flags
- [x] Library API: settings keys for font paths
- [x] Explicit non-claim: full Arabic/Indic OpenType shaping not supported

### 19.7 Image mode

- [x] Image-mode wires `--font-path` registry (`internal/imageout`)
- [x] Missing glyph: tofu/`?` policy consistent with PDF path

### 19.8 Docs

- [x] fonts.md / README CJK + Hangul rows updated for #17 era
- [ ] Compatibility-matrix Unicode / flex / @font-face sections still stale in places
- [x] Threat model note: font parse is untrusted input (size/ACL)

### 19.9 Closure gates

- [x] `make lint` / `make test`
- [x] CJK fixture-27 + folder-font path green
- [x] Parent Phase 19 core checked
- [ ] Remaining: matrix honesty + @font-face end-to-end audit (see Pending)
- [x] Next: **Phase 21** or Phase 20 leftover HF HTML-link polish

---

## Pending (after #17)

| Item | Notes |
|------|--------|
| `@font-face` local wiring vs matrix | CSS parses; matrix still says ignored — audit and mark Partial correctly |
| Compatibility-matrix i18n / CJK rows | Update for Type0, Arabic joining, vertical lite, subset fixes |
| OpenType `halt`/`palt` | Only if a face exposes features **and** we add a stdlib-safe consumer (not planned) |
| Full Indic / HarfBuzz | Rejected by amendment unless product changes constraints |
| Bundle full Noto CJK | Prefer user `--font-path`; CI keeps tiny Hangul subset only |
| WOFF/WOFF2 | Deferred |

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 12 registry | Phase 21 non-Latin sites |
| Phase 15 raster | Consistent image CJK |

---

## Out of scope

- WOFF/WOFF2 (unless pure-Go decode added later)
- HarfBuzz / CGO / third-party shaping modules
- Auto-download Google Fonts
- Shipping multi-megabyte CJK faces in the default binary
