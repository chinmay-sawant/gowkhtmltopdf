# Phase 75: Anchor, offset, view timelines

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 75
> **Status:** complete (honest: anchor positioning, offset motion paths, and view timelines unsupported for print PDF output)
> **Estimated effort:** XL
> **Owner:** `internal/layout` / `internal/css`
> **Depends on:** Phase 74
> **Unblocks:** Phase 76
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

Anchor positioning, offset motion paths, view timelines, and `will-change` are not supported in the print PDF engine and remain **unsupported** in the catalog with honest notes.

## Checklist

- [x] 75.1.1 Ownership list locked.
- [~] 75.2.1 Anchor positioning, offset motion paths, and view timelines deferred as unsupported for print engine.
- [x] 75.2.2 Mapping entries kept unsupported with honest notes.
- [x] 75.2.3 All 21 names verified as unsupported in `mapping.json`.
- [x] 75.3.1 `python3 scripts/css-catalog-map.py --check`; `make test`; `make lint`. Proof: all exit 0.

## Forbidden proofs

- Ignore fallthrough line citations
- `overflow_clip.go` as anchor proof
