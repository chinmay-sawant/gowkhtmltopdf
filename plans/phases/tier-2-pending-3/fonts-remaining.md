# Tier 2 Pending-3 — Fonts / i18n remaining (policy + WOFF1)

> **Parent:** [`plans/phases/phase-19-fonts-i18n.md`](../phase-19-fonts-i18n.md)  
> **Related:** [`../subplans-tier-2/shaping-gotext-typesetting.md`](../subplans-tier-2/shaping-gotext-typesetting.md)  
> **Status:** done  
> **Estimated effort:** 1 day  
> **Constraint:** stdlib + `go-text/typesetting` only; **no CGO HarfBuzz**; no new direct modules  
> **Spec:** [WOFF](https://www.w3.org/TR/WOFF/) · [WOFF2](https://www.w3.org/TR/WOFF2/) · OT `halt`/`palt`

---

## Overview

Phase 19 core + OT shaping + image `@font-face` were already shipped. This ledger
closes remaining honesty rows and ships **WOFF1** decode (stdlib zlib → SFNT →
`ParseTTF`), optional **`halt`/`palt`** FontFeatures, and documents intentional
non-goals (WOFF2/Brotli, remote HTTPS `@font-face`, bundled Noto CJK, CGO HarfBuzz).

---

## Executive Summary

| Item | Disposition |
|------|-------------|
| Local TTF/OTF `@font-face` PDF+image | **Done** — keep |
| OT GSUB via typesetting | **Done** — keep |
| WOFF1 decode | **Done** — `internal/pdf/woff.go` → `ParseFontBytes` |
| WOFF2 + Brotli | **Confirmed unsupported** — typesetting has WOFF1 only; no new modules; `TestDecodeWOFF2Gap` |
| Remote `https://` `@font-face` | **Not supported by design** — ACL/network policy; skip kept |
| `halt` / `palt` | **Done** — CJK punct auto + `ParseFontFeatureSettings` |
| Indic production / CGO HarfBuzz | **Rejected / Partial honesty** — allowlist test |
| Bundle full Noto CJK | **Policy confirmed** — `--font-path` only |

---

## Phase 1: Honesty confirmations

### 1.1 Matrix / fonts.md

- [x] `@font-face`: local TTF/OTF/WOFF1; WOFF2 + remote HTTPS skipped (policy documented)
- [x] Complex-script: Arabic OT Partial + fallback; Indic **not production-claimed**
- [x] `halt`/`palt` → wired via `ShapeTextFont` FontFeatures (CJK punct + CSS parser helper)
- [x] Full Noto CJK bundle → Not planned; `--font-path` policy confirmed
- [x] Clarify “HarfBuzz” wording = pure-Go port inside `go-text/typesetting`, **not** CGO
- [x] Proof: `rg` stale “needs HarfBuzz” / “image mode N/A” claims gone from `documentation/`

### 1.2 Allowlist gate

- [x] Keep `TestDirectModuleAllowlist` green (CGO HarfBuzz rejected)
- [x] No new direct modules without amendment

### 1.3 Parent checklist

- [x] Phase-19 pending rows point here with dispositions above (no `[~]`)

---

## Phase 2: WOFF1 (+ WOFF2 honesty)

### 2.1 WOFF1

- [x] Decode WOFF1 → SFNT via `compress/zlib` (`internal/pdf/woff.go`)
- [x] Feed bytes into existing `ParseTTF` path (TrueType only; reject CFF/OTTO)
- [x] Security: reject bad offsets, decompress bombs (size caps), overlapping tables
- [x] Soften `.woff` skip in `MergeFontFaces`; keep `.woff2` / `.eot` skipped
- [x] Tests: `TestDecodeWOFFRoundTripParseTTF`, `TestFontFaceWOFFEmbed`; still skip WOFF2/HTTPS/`data:`
- [x] Threat model note update (`documentation/THREAT-MODEL.md`)

### 2.2 WOFF2 / remote (intentional product decisions)

- [x] WOFF2 + Brotli module — **not supported by design** without amendment; `DecodeWOFF2` / `TestDecodeWOFF2Gap` document gap (typesetting has no WOFF2)
- [x] Remote `https://` `@font-face` fetch — **not supported by design: ACL/network policy** (skip kept; docs + `TestFontFaceHTTPSSkipped`)

---

## Phase 3: `halt` / `palt`

### 3.1 Shipped

- [x] Pass `FontFeatures` (`halt`/`palt`) for CJK / East-Asian punctuation in `ShapeTextFont`
- [x] `ParseFontFeatureSettings` + `ShapeTextFontWithFeatures` for CSS `font-feature-settings`
- [x] Tests: `TestCJKPunctFontFeatures`, `TestParseFontFeatureSettings`, `TestShapeTextFontWithFeaturesCJKStillSafe`

---

## Phase 4: Closure

- [x] Required honesty pass landed
- [x] WOFF1 + halt/palt shipped; WOFF2/remote/Noto/CGO marked `[x]` as intentional policy
- [x] Phase 19 pending leftovers closed for Tier 2 fonts
- [x] `make lint` + `go test ./internal/pdf ./internal/convert ./internal/imageout -count=1` — see outcomes below
- [x] Next: Phase 21 or remaining CSS Must items

### Validation outcomes

```
$ make lint
go vet ./...
(exit 0)

$ go test ./internal/pdf ./internal/convert ./internal/imageout -count=1
ok  gowkhtmltopdf/internal/pdf       0.200s
ok  gowkhtmltopdf/internal/convert   1.068s
ok  gowkhtmltopdf/internal/imageout  0.567s
```

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| shaping-gotext done | OT baseline + FontFeatures |
| MergeFontFaces | WOFF1 hook |

---

## Out of scope

- Bundling multi-MB Noto CJK (policy: `--font-path`)
- CGO HarfBuzz
- Production Indic certification
- WOFF2 without a Brotli-module amendment
- Remote webfont CDN fetch
- `text-spacing-trim` full implementation
