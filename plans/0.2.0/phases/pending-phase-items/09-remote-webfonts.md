# Pending — Phase 9: Remote webfonts (WOFF2 / HTTPS `@font-face`)

> **Parent:** [`README.md`](README.md)  
> **Status:** `[~]` superseded for remaining WOFF2 decode — execution moved to  
> [`plans/0.2.6/woff2-metric-aliases/`](../../../0.2.6/woff2-metric-aliases/README.md)  
> **Estimated effort:** days (needs allowlist amendment for Brotli)  
> **Prior plan coverage:** Phase 19 / fonts-remaining permanent non-goal —  
> **superseded** first by this ledger, then by the 0.2.6 track for unfinished work

---

## Overview

Chrome loads wiki webfonts. Implement remote `https://` `@font-face` under the
existing loader ACL/timeouts, plus WOFF2 decode (Brotli). Phase 3 local fallback
remains the offline path.

**Disposition (2026-08-20):** HTTPS TTF/OTF/WOFF1 already shipped. Remaining
WOFF2/Brotli decode, docs, and gates are executed under
`plans/0.2.6/woff2-metric-aliases/` (with
[`2026-08-20-woff2-brotli-allowlist.md`](../../../0.2.6/woff2-metric-aliases/amendments/2026-08-20-woff2-brotli-allowlist.md)).
Metric aliases are **not** part of this Phase 9 file; they live only in the
0.2.6 combined track.

---

## Phase 9 checklist

### 9.1 Implement

- [~] Amendment: WOFF2/Brotli — **moved** to `plans/0.2.6/woff2-metric-aliases/`
- [x] Fetch `https://` `@font-face` src via `FetchSub` (same ACL as images/CSS) — `MergeFontFaces`
- [~] Decode WOFF2 → SFNT; register face like WOFF1 — **moved** to 0.2.6 Phase 02–03
- [x] Tests: `TestFontFaceHTTPSFetchAttempted` (fetch attempted, not policy-skipped)
- [~] fonts.md / matrix updated for remaining WOFF2 gap — **moved** to 0.2.6 Phase 07

### 9.2 Gates

- [~] `make lint` / `make test` / status → done — **moved** to 0.2.6 Phase 08

---

## Out of scope

- Auto-installing system font packages
- CGO HarfBuzz
- Opt-in Fontconfig-style metric aliases (see `plans/0.2.6/woff2-metric-aliases/`)
