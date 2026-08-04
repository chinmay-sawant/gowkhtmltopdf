# Tier 2 Subplan - Phase 19 Pending (@font-face audit + i18n honesty)

> **Parent:** [`plans/phases/phase-19-fonts-i18n.md`](../phase-19-fonts-i18n.md) — Pending (after #17)  
> **Amendment:** [`plans/amendments/2026-08-04-shaping-stdlib.md`](../../amendments/2026-08-04-shaping-stdlib.md)  
> **Status:** not started  
> **Estimated effort:** 0.5–1 day audit/tests + docs (shared pass)  
> **Depends on:** [00-shared-doc-honesty.md](00-shared-doc-honesty.md) for matrix i18n rows  
> **Constraint:** stdlib-only TTF; **no HarfBuzz**; no WOFF unless amended

---

## Overview

Phase 19 **core is shipped**: `--font-path` / `--use-system-fonts`, Type0/CID,
per-rune fallback, Arabic presentation-form joining, vertical-rl lite, glyf subset
fidelity (#17). Remaining work is (1) audit `@font-face` end-to-end and mark
**Partial** correctly, (2) sync matrix/fidelity/overview/README i18n rows, and
(3) confirm deferred items stay deferred.

## Executive Summary

| Pending item (parent) | Disposition | Primary work |
|-----------------------|-------------|--------------|
| `@font-face` local wiring vs matrix | **Must audit** | PDF path wired; matrix says ignored |
| Compatibility-matrix i18n / CJK rows | **Must** | Shared Pass 0 |
| OpenType `halt`/`palt` | Confirm not planned | Checkbox only |
| Full Indic / HarfBuzz | Rejected by amendment | Checkbox only |
| Bundle full Noto CJK | Prefer `--font-path` | Checkbox only |
| WOFF/WOFF2 | Deferred | Checkbox only |

---

## Phase 1: Evidence baseline (scanned 2026-08-05)

### 1.1 `@font-face` pipeline

| Stage | Status | Path |
|-------|--------|------|
| CSS parse | Works | `internal/css/css.go` `parseFontFace` → `Stylesheet.FontFaces` |
| URL extract | Works | `FontFaceURLs` — `url(...)` only (not `local(...)`) |
| Load under ACL | Works (PDF) | `convert.mergeFontFaces` → `loader.FetchSub` → ACL |
| Register | Works (PDF) | `ParseTTF` → `Registry.AddFont` + `AddFamilyAlias` |
| Layout | Works | `faceFor` / `faceForRune` → `Lookup` |
| PDF embed | Works | Same registry; Type0 when needed |
| Image mode | **Unwired** | `imageout.Run` scans font-path only |
| E2E convert test | **Missing** | CSS unit parse only (`TestParseAtRulesSkipped`) |

### 1.2 Partial gaps (document or harden)

- [x] WOFF / `https://` src skipped in `mergeFontFaces` (correct)
- [x] `ff.Weight` / `ff.Style` parsed but **unused** at register time
- [x] Network webfonts rejected before HTTP fetch
- [~] `data:` URLs may pass network check (no `://`) — document or reject
- [~] Image-mode `@font-face` unwired
- [~] No convert golden asserting Custom face embedding

### 1.3 Shipped i18n (honesty targets)

| Capability | Path | Tests |
|------------|------|-------|
| Type0 / CID Identity-H | `internal/pdf/fonttype0.go` | `fonttype0_test.go`; fixture-27 |
| Arabic joining | `internal/pdf/shape.go` `ShapeText` | `shape_test.go` |
| vertical-rl lite | `internal/layout/vertical.go` + `RotateDeg` | flex_test vertical asserts |
| Glyf subset #17 | `internal/pdf/subset.go` align + `stripGlyphHints` | `subset_align_test.go` |
| Hangul / per-rune | `faceForRune` + Noto KR CI subset | `cjk_fallback_test.go`; fixture-27 |

### 1.4 ACL vs `--font-path`

- `@font-face` local `src`: subject to loader ACL (`fileAccessAllowed` / `--allow` / enable-local)
- `--font-path` / `--use-system-fonts`: operator opt-in via `ScanFontDirs` / `os.ReadFile` — **not** HTML ACL
- Docs must keep that distinction clear

---

## Phase 2: `@font-face` audit (code + tests)

### 2.1 Prove PDF Partial

- [ ] Convert integration test: HTML with `@font-face { font-family: Custom; src: url(<testdata ttf>) }` + `font-family: Custom`
  - Path suggestion: `internal/convert/fontface_test.go` (new) or extend existing convert tests
  - Enable local ACL appropriately (`EnableLocalFileAccess` / `--allow`)
  - Assert PDF embeds the face (font name / Type0 / subset presence — pick one stable signal)
- [ ] ACL deny case: same fixture without local enable → warning; Liberation/`?` fallback; no panic
- [ ] WOFF src → skipped warning; no panic
- [ ] `https://` src → skipped; no fetch
- [ ] Proof: `go test ./internal/convert -run FontFace -count=1`

### 2.2 Document Partial boundaries

- [ ] Record: image mode does **not** call `mergeFontFaces` (honest Partial, or tiny follow-up to port)
- [ ] Record: `font-weight` / `font-style` on `@font-face` ignored at register
- [ ] Spot-check `data:` `@font-face`; document reject-or-allow decision
- [ ] Optional tiny harden: reject `data:` unless explicitly allowed

### 2.3 Optional image-mode parity (separate slice)

- [~] Port `mergeFontFaces` into `internal/imageout` **only if** product wants image/PDF parity
- [~] Otherwise leave `[~]` with pointer from fonts.md

---

## Phase 3: Documentation honesty

### 3.1 Shared matrix / fidelity (must)

Owned by [00-shared-doc-honesty.md](00-shared-doc-honesty.md) §2.4 / §3:

- [ ] §4 split `@page` vs `@font-face` Partial; fix stale cites
- [ ] §5 `@font-face` Ignored → Partial wording
- [ ] §5 RTL/CJK Phase-3 Latin-first → Type0 + Arabic + vertical lite
- [ ] §2.3 font-family “until phase 19” → registry shipped
- [ ] Add `--font-path` / `--use-system-fonts` flag rows
- [ ] `fidelity.md` feature map L96: Phase 19 Partial (not “19 next”)
- [ ] `overview.md` / README overview CJK rows aligned with deferred table

### 3.2 Fonts.md + threat model (should)

- [ ] `documentation/fonts.md`: tighten `@font-face` to “PDF Partial; image mode N/A; weight/style descriptors ignored”
- [ ] `documentation/THREAT-MODEL.md`: one note — TTF via `@font-face` is untrusted parse under ACL; `--font-path` is operator-controlled
- [ ] Keep shaping honesty: presentation-form Arabic; no GSUB/GPOS / HarfBuzz

### 3.3 Recommended matrix status labels

| Topic | Status label |
|-------|--------------|
| `@font-face` local TTF/OTF (PDF) | **Partial** |
| `@font-face` remote / WOFF | Not implemented (deferred) |
| Unicode / CJK Type0 | Partial / Implemented subset |
| Arabic | Partial (joining, not OT) |
| Hangul | Partial (needs face; CI subset) |
| `writing-mode` vertical | Partial (lite) |
| `halt`/`palt` | Not planned |
| HarfBuzz / full Indic | Rejected |
| Full Noto CJK bundle | Not planned |
| WOFF/WOFF2 | Deferred |

---

## Phase 4: Deferred confirmations (checkboxes only)

- [ ] OpenType `halt`/`palt`: **not planned** (needs OT feature consumer)
- [ ] Full Indic / HarfBuzz: **rejected** unless product changes amendment
- [ ] Bundle full Noto CJK: **no** — prefer `--font-path`; CI keeps tiny Hangul subset
- [ ] WOFF/WOFF2: **deferred** — keep skip in `mergeFontFaces`

---

## Phase 5: Closure gates

### 5.1 Required

- [ ] Audit tests green (`go test ./internal/convert -run FontFace`)
- [ ] Shared doc-honesty i18n/@font-face rows landed
- [ ] Parent Phase 19 Pending / 19.3 / 19.8 / 19.9 remaining boxes updated
- [ ] Non-doc changes: `make lint` → ; `make test` → ; record outcomes beside gates

### 5.2 Next

- [ ] Phase 20 HF fragment GoTo, or Phase 21

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 12 registry | Alias registration |
| Shared doc-honesty | Closable matrix lies |
| Amendment shaping-stdlib | Non-HarfBuzz boundary |
| Audit proof | Safe Partial label |

---

## Out of scope

- WOFF/WOFF2 decode
- HarfBuzz / CGO / third-party shaping
- Auto-download Google Fonts
- Shipping multi-megabyte CJK faces in the default binary
- Full OpenType vertical metrics / `halt`/`palt`
