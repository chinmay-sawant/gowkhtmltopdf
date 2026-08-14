# Tier 2 Subplan - Phase 19 Pending (@font-face audit + i18n honesty)

> **Parent:** [`plans/phases/phase-19-fonts-i18n.md`](../phase-19-fonts-i18n.md) — Pending (after #17)  
> **Amendment:** [`plans/amendments/2026-08-04-shaping-stdlib.md`](../../amendments/2026-08-04-shaping-stdlib.md) (interim; partly superseded)  
> **Status:** **done** — `@font-face` PDF+image audit; OT shaping via `go-text/typesetting`; matrix honesty landed  
> **Estimated effort:** 0.5–1 day audit/tests + docs (shared pass)  
> **Depends on:** [00-shared-doc-honesty.md](00-shared-doc-honesty.md) for matrix i18n rows  
> **Constraint:** stdlib TTF + allowlisted `go-text/typesetting`; **no CGO HarfBuzz**; no WOFF unless amended

---

## Overview

Phase 19 **core is shipped**: `--font-path` / `--use-system-fonts`, Type0/CID,
per-rune fallback, Arabic OT (`ShapeTextFont`) + presentation-form fallback,
vertical-rl lite, glyf subset fidelity (#17), local `@font-face` on PDF **and**
image paths. Remaining items are intentional non-goals (WOFF, Noto bundle,
`halt`/`palt`, CGO HarfBuzz).

## Executive Summary

| Pending item (parent) | Disposition | Primary work |
|-----------------------|-------------|--------------|
| `@font-face` local wiring vs matrix | **[x] done** | PDF + image Partial; matrix Partial |
| Compatibility-matrix i18n / CJK rows | **[x] done** | Shared Pass 0 |
| OpenType GSUB via `go-text/typesetting` | **[x] done** | [`shaping-gotext-typesetting.md`](shaping-gotext-typesetting.md) |
| OpenType `halt`/`palt` | **[~]** not planned | Checkbox only |
| Full Indic / CGO HarfBuzz | **[~]** rejected | Checkbox only |
| Bundle full Noto CJK | **[~]** no — prefer `--font-path` | Checkbox only |
| WOFF/WOFF2 | **[~]** deferred | Checkbox only |
| Image-mode `@font-face` | **[x] done** | [`image-mode-fontface.md`](image-mode-fontface.md) |

---

## Phase 1: Evidence baseline (scanned 2026-08-05; refreshed same day)

### 1.1 `@font-face` pipeline

| Stage | Status | Path |
|-------|--------|------|
| CSS parse | Works | `internal/css/css.go` `parseFontFace` → `Stylesheet.FontFaces` |
| URL extract | Works | `FontFaceURLs` — `url(...)` only (not `local(...)`) |
| Load under ACL | Works (PDF + image) | `convert.MergeFontFaces` → `loader.FetchSub` → ACL |
| Register | Works | `ParseTTF` → `Registry.AddFont` + `AddFamilyAlias` |
| Layout | Works | `faceFor` / `faceForRune` → `Lookup` |
| PDF embed | Works | Same registry; Type0 when needed |
| Image mode | **Wired** | `imageout` calls `convert.MergeFontFaces` |
| E2E convert test | **Shipped** | `internal/convert/fontface_test.go` + `internal/imageout/fontface_test.go` |

### 1.2 Partial gaps (document or harden)

- [x] WOFF / `https://` src skipped in `mergeFontFaces` (correct)
- [x] `ff.Weight` / `ff.Style` parsed but **unused** at register time
- [x] Network webfonts rejected before HTTP fetch
- [x] `data:` URLs rejected in `mergeFontFaces` (would bypass `://` gate)
- [x] Image-mode `@font-face` wired via `convert.MergeFontFaces`
- [x] Convert golden asserting Custom face embedding (`TestFontFaceLocalEmbed`)

### 1.3 Shipped i18n (honesty targets)

| Capability | Path | Tests |
|------------|------|-------|
| Type0 / CID Identity-H | `internal/pdf/fonttype0.go` | `fonttype0_test.go`; fixture-27 |
| Arabic OT + fallback | `ShapeTextFont` / `ShapeText` | `shape_test.go`; `shape_gotext.go` |
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

- [x] Convert integration test: HTML with `@font-face { font-family: Custom; src: url(<testdata ttf>) }` + `font-family: Custom`
  - Path: `internal/convert/fontface_test.go`
  - Enable local ACL appropriately (`EnableLocalFileAccess` / `--allow`)
  - Assert PDF embeds the face (`/BaseFont /Custom` + `/FontFile2`)
- [x] ACL deny case: page under `--allow`; sibling font outside → warning; Liberation fallback; no panic
- [x] WOFF src → skipped warning; no panic
- [x] `https://` src → skipped; no fetch
- [x] Proof: `go test ./internal/convert -run FontFace -count=1`

### 2.2 Document Partial boundaries

- [x] Record: image mode calls `MergeFontFaces` (PDF + image local TTF/OTF Partial)
- [x] Record: `font-weight` / `font-style` on `@font-face` ignored at register
- [x] Spot-check `data:` `@font-face`; **reject** (skip warning)
- [x] Tiny harden: reject `data:` in `mergeFontFaces`

### 2.3 Image-mode parity

- [x] Port / share `MergeFontFaces` into `internal/imageout` — [`image-mode-fontface.md`](image-mode-fontface.md)
- [x] fonts.md + matrix: PDF + image local TTF; WOFF/remote still skipped

---

## Phase 3: Documentation honesty

### 3.1 Shared matrix / fidelity (must)

Owned by [00-shared-doc-honesty.md](00-shared-doc-honesty.md) §2.4 / §3:

- [x] §4 split `@page` vs `@font-face` Partial; fix stale cites
- [x] §5 `@font-face` Ignored → Partial wording (PDF + image local TTF)
- [x] §5 RTL/CJK → Type0 + Arabic OT / presentation-form fallback + vertical lite
- [x] §2.3 font-family registry shipped
- [x] Add `--font-path` / `--use-system-fonts` flag rows
- [x] `fidelity.md` feature map: Phase 19 Partial
- [x] Overview / README CJK rows aligned with deferred table

### 3.2 Fonts.md + threat model (should)

- [x] `documentation/fonts.md`: local `@font-face` PDF + image; weight/style descriptors ignored; WOFF/remote deferred
- [x] `documentation/THREAT-MODEL.md`: TTF via `@font-face` is untrusted parse under ACL; `--font-path` is operator-controlled
- [x] Keep shaping honesty: OT via `go-text/typesetting` when GSUB present; presentation-form fallback; no CGO HarfBuzz

### 3.3 Recommended matrix status labels

| Topic | Status label |
|-------|--------------|
| `@font-face` local TTF/OTF (PDF + image) | **Partial** |
| `@font-face` remote / WOFF | Not implemented (deferred) |
| Unicode / CJK Type0 | Partial / Implemented subset |
| Arabic | Partial (OT GSUB when face has it + presentation-form fallback) |
| Hangul | Partial (needs face; CI subset) |
| `writing-mode` vertical | Partial (lite) |
| `halt`/`palt` | Not planned |
| CGO HarfBuzz / full Indic production | Rejected / not claimed |
| Full Noto CJK bundle | Not planned |
| WOFF/WOFF2 | Deferred |

---

## Phase 4: Deferred confirmations (checkboxes only)

- [~] OpenType `halt`/`palt`: **not planned** (optional feature tags not wired)
- [~] Full Indic production claim / CGO HarfBuzz: **rejected** unless product changes amendment
- [~] Bundle full Noto CJK: **no** — prefer `--font-path`; CI keeps tiny Hangul subset
- [~] WOFF/WOFF2: **deferred** — keep skip in `mergeFontFaces`

---

## Phase 5: Closure gates

### 5.1 Required

- [x] Audit tests green (`go test ./internal/convert -run FontFace`) — see outcomes below
- [x] Shared doc-honesty i18n/@font-face rows landed
- [x] Parent Phase 19 Pending / 19.3 audit rows updated
- [x] Non-doc changes: `make lint` → ; FontFace tests → ; record outcomes beside gates

**Outcomes (2026-08-05):**
- `go test ./internal/convert -run FontFace -count=1` → PASS (LocalEmbed, ACLDeny, WOFF, HTTPS, Data)
- `make lint` → PASS (`go vet ./...`)

### 5.2 Next

- [x] Phase 20 HF fragment GoTo closed; product next is **Phase 21**

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 12 registry | Alias registration |
| Shared doc-honesty | Closable matrix lies |
| Amendment gotext-typesetting | OT shaping allowlist |
| Audit proof | Safe Partial label |

---

## Out of scope

- WOFF/WOFF2 decode
- CGO HarfBuzz
- Auto-download Google Fonts
- Shipping multi-megabyte CJK faces in the default binary
- Full OpenType vertical metrics / `halt`/`palt`
