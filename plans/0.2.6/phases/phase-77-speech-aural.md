# Phase 77: Speech and aural

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 77
> **Status:** not started
> **Estimated effort:** M
> **Owner:** `internal/layout` / `internal/css` / catalog
> **Depends on:** Phase 76
> **Unblocks:** Phase 78

---

## Overview

speak/voice/cue/pause/rest and related aural properties. Browser print rarely needs these; still owned by this program per scope.

Bar: **browser-level print** for the former Ignored set (247 names). Goldens stay structural unless amended. Flip mapping only with code + tests + matrix agreement.

**Count:** 19

## Goals

- Clear every owned name from the work list into Implemented, or mark `[~]` with an explicit reason
- Keep catalog counts honest after each promotion batch

## Checklist

### 77.1 scope lock

- [ ] 77.1.1 Own these 19 properties (from Phase 68 inventory): `cue`, `cue-after`, `cue-before`, `pause`, `pause-after`, `pause-before`, `rest`, `rest-after`, `rest-before`, `speak`, `speak-as`, `voice-balance`, `voice-duration`, `voice-family`, `voice-pitch`, `voice-range`, `voice-rate`, `voice-stress`, `voice-volume`. Proof: names still `unsupported` (or listed) in `mapping.json` at phase start.

### 77.2 implementation

- [ ] 77.2.1 Implement browser-level print behavior for each owned name (or alias to an existing Implemented longhand). Proof: tests cited per promotion.
- [ ] 77.2.2 Flip each finished name to `engine_status: implemented` with matrix notes. Proof: mapping + matrix.

### 77.3 gates

- [ ] 77.3.1 Targeted package tests exit 0.
- [ ] 77.3.2 `python3 scripts/css-catalog-map.py --check` exit 0.
- [ ] 77.3.3 `make test` and `make lint` exit 0 before phase complete; `make golden` if paint/layout/pagination changed.


## Out of scope

JavaScript execution. Pixel-diff Chrome goldens as the default gate. New direct Go modules without sign-off. Growing `paint_flow.go` / `paint_pagination.go` past the soft cap without extracting.

## Handoff

Next is Phase 78.
