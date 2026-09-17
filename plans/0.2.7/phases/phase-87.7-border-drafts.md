# Phase 87.7: Border clip / limit / shape / boundary (draft honesty)

> **Parent:** `../87-canonical-0.2.7-next-72.md`
> **Status:** complete (defer all 14)
> **Estimated effort:** S (defer) or L (experimental paint)
> **Owner:** `internal/layout` (chrome paint) if anything ships
> **Depends on:** none
> **Unblocks:** catalog honesty for 14 draft names on fixture-64
> **Mid-batch gate:** package tests only. No `make lint`. No `make test`.

---

## Overview

Fourteen Borders-4 / Round Display names. Agent scan found **no** apply or paint
arms. Spec sections are largely "not ready for implementation". Chrome BCD: 13
no / 1 yes (`border-shape` since 147). Prior 0.2.6 phase-80/85 already deferred
similar drafts.

**Default recommendation:** keep **Unsupported**. Do not claim Implemented for
parse-only stubs. Fixture-64 Effect cells already warn `Chrome: no render expected`
where applicable.

## Properties (14)

`border-block-clip`, `border-block-end-clip`, `border-block-start-clip`,
`border-bottom-clip`, `border-boundary`, `border-clip`, `border-inline-clip`,
`border-inline-end-clip`, `border-inline-start-clip`, `border-left-clip`,
`border-limit`, `border-right-clip`, `border-shape`, `border-top-clip`

## Product decision (v0.2.7)

**Defer all 14.** No apply/paint stubs. No experimental Borders-4 paint. All
remain `engine_status: unsupported` in `plans/0.2.6/catalog/mapping.json`.
Matrix contract: `documentation/compatibility-matrix.md` §5.5 (Unsupported /
draft-not-ready). Proof: `_proof-87.7.md`.

Deferred names:

1. `border-block-clip`
2. `border-block-end-clip`
3. `border-block-start-clip`
4. `border-bottom-clip`
5. `border-boundary`
6. `border-clip`
7. `border-inline-clip`
8. `border-inline-end-clip`
9. `border-inline-start-clip`
10. `border-left-clip`
11. `border-limit`
12. `border-right-clip`
13. `border-shape`
14. `border-top-clip`

## Paint path (if ever)

`layout_chrome.go` `prependChrome` / `borderOpsSides` / `roundedBorderOps`; also
`emitBorders` and table border emitters. Three backends make partial geometry easy
to ship half-broken. Pagination chrome repair is sensitive to vertical rails.

If wiring anything: new `style_border_partial_props.go` only; **do not** grow
`style_properties.go`.

## Checklist

### 87.7.1 defer decision

- [x] 87.7.1.1 Record in this file: **Defer all 14** (recommended) or list a tiny experimental subset. Decision: **Defer all 14** (see Product decision above).
- [x] 87.7.1.2 Update matrix / deferred notes if the defer is newly explicit for v0.2.7. Added `documentation/compatibility-matrix.md` §5.5.
- [x] 87.7.1.3 Leave `mapping.json` `engine_status: unsupported` for all 14 unless a real paint consumer lands. Confirmed all 14 remain `unsupported`; no flips.

### 87.7.2 optional experimental subset (only if product overrides defer)

- [x] 87.7.2.1 Extract apply into `style_border_partial_props.go`. N/A (defer all 14 chosen).
- [x] 87.7.2.2 Consumer in `borderOpsSides` for a documented lite (e.g. `border-limit` truncate on straight OpLine only). N/A (defer all 14 chosen).
- [x] 87.7.2.3 `border-shape`: treat as **L**; invalid fixture value `bevel` must be fixed to legal syntax before claiming anything. N/A (defer all 14 chosen).
- [x] 87.7.2.4 Package tests prove visible geometry change. Flip mapping last with honesty packet. N/A (defer all 14 chosen).

### 87.7.R batch gate (package only)

- [x] 87.7.R.1 If deferred: no code change required; check this batch complete when 87.7.1 rows are `[x]`.
- [x] 87.7.R.2 If experimental code landed: `go test ./internal/layout -run 'TestBorderClip|TestBorderLimit|TestBorderShape' -count=1` exit 0. N/A (no experimental code).
- [x] 87.7.R.3 **No `make test` / `make lint`.**

## Out of scope

Mask/clip-path hard-defer families outside these 14. Chrome pixel parity.
