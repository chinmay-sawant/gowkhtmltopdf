# Tier 2 Pending-3 — Fonts / i18n remaining (policy + optional WOFF1)

> **Parent:** [`plans/phases/phase-19-fonts-i18n.md`](../phase-19-fonts-i18n.md)  
> **Related:** [`../subplans-tier-2/shaping-gotext-typesetting.md`](../subplans-tier-2/shaping-gotext-typesetting.md)  
> **Status:** not started  
> **Estimated effort:** 0.5–1 day honesty; **optional** 3–7 days WOFF1 if amended  
> **Constraint:** stdlib + `go-text/typesetting` only; **no CGO HarfBuzz**; WOFF2+Brotli needs **new amendment**  
> **Spec:** [WOFF](https://www.w3.org/TR/WOFF/) · [WOFF2](https://www.w3.org/TR/WOFF2/) · OT `halt`/`palt`

---

## Overview

Phase 19 core + OT shaping + image `@font-face` are shipped. Remaining rows are
mostly **policy confirmations**. Optional WOFF1 (zlib) may be added without new
modules; WOFF2, remote webfonts, `halt`/`palt`, Indic production claims, and
bundled Noto CJK stay deferred / rejected unless product amends.

---

## Executive Summary

| Item | Disposition |
|------|-------------|
| Local TTF/OTF `@font-face` PDF+image | **Done** — keep |
| OT GSUB via typesetting | **Done** — keep |
| WOFF1 decode | **Optional** (stdlib zlib) — only if product wants |
| WOFF2 + Brotli | **`[~]`** needs amendment |
| Remote `https://` `@font-face` | **`[~]`** keep skip |
| `halt` / `palt` | **`[~]`** not planned |
| Indic production / CGO HarfBuzz | **`[~]`** rejected / Partial honesty |
| Bundle full Noto CJK | **`[~]`** no — `--font-path` |

---

## Phase 1: Honesty confirmations (required)

### 1.1 Matrix / fonts.md

- [ ] `@font-face` remote / WOFF → Not implemented (deferred)
- [ ] Complex-script: Arabic OT Partial + fallback; Indic **not production-claimed**
- [ ] `halt`/`palt` → Not planned
- [ ] Full Noto CJK bundle → Not planned; `--font-path` policy
- [ ] Clarify “HarfBuzz” wording = pure-Go port inside `go-text/typesetting`, **not** CGO
- [ ] Proof: `rg` stale “needs HarfBuzz” / “image mode N/A” claims gone

### 1.2 Allowlist gate

- [ ] Keep `TestDirectModulesAllowlist` green
- [ ] No new direct modules without amendment

### 1.3 Parent checklist

- [ ] Phase-19 `[~]` rows point here with dispositions above
- [ ] Docs-only: skip lint/test per skill unless code changes

---

## Phase 2: Optional WOFF1 (product gate)

### 2.1 Only if amended / prioritized

- [~] Decode WOFF1 → SFNT via `compress/zlib`
- [~] Feed bytes into existing `ParseTTF` path (TrueType only; reject CFF)
- [~] Security: reject bad offsets, decompress bombs (size caps), overlapping tables
- [~] Soften `.woff` skip in `MergeFontFaces`; keep `.woff2` skipped
- [~] Tests: positive WOFF1 embed; still skip WOFF2/HTTPS/`data:`
- [~] Threat model note update

### 2.2 Explicit non-start without amendment

- [~] WOFF2 + Brotli module
- [~] Auto-download Google Fonts / remote `@font-face` fetch

---

## Phase 3: Optional `halt` / `palt` (low priority)

### 3.1 Default

- [~] Leave `ShapeTextFont` without feature tags
- [~] Keep full-em CJK punctuation metrics (no fake halt)

### 3.2 Only if product wants JLREQ polish

- [~] Pass `FontFeatures` (`halt`/`palt`) when `font-feature-settings` parsed
- [~] Requires face with tables + tests; else skip

---

## Phase 4: Closure

- [ ] Required honesty pass landed → mark subplan **done** for policy items
- [ ] Optional WOFF1/halt remain `[~]` with clear next gate
- [ ] Phase 19 pending leftovers closed for Tier 2
- [ ] Next: Phase 21 or remaining CSS Must items

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| shaping-gotext done | OT baseline |
| MergeFontFaces | WOFF1 hook |
| Amendment process | WOFF2 only |

---

## Out of scope

- Bundling multi-MB Noto CJK
- CGO HarfBuzz
- Production Indic certification
- `text-spacing-trim` full implementation
