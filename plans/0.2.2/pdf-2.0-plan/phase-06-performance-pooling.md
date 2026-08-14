# Phase 6 — Performance & Pooling

**Status:** ⏸️ NOT STARTED — deprioritized; Phase 4 + Phase 5 gates green  
**Depends on:** Phase 4 + Phase 5 green (do not optimize non-compliant output)  
**Base plan refs:** §5 Phase 1 buffer notes, §11 Phase F, HFT-scale lessons from gopdfsuit

---

## Goal

Make large compliant documents (dense tables, many MCIDs) fast and memory-efficient without breaking PDF/A-4 or PDF/UA-2 gates.

---

## Checklist — buffer & allocation

- [ ] Reuse `bytes.Buffer` pools for final PDF and compress scratch
- [ ] Capacity estimate before emit (avoid mid-write growth when possible)
- [ ] Scratch byte slices for number formatting / struct emit
- [ ] Xref offset slice pooling

## Checklist — structure hot path

- [ ] Arena / pool for StructElem on large tables
- [ ] Batch TD leaf StructElem serialization
- [ ] Fast path for TR with only element-ref kids
- [ ] Pre-reserve MCID / ParentTree capacity per page stripe
- [ ] Deferred ParentTree fill where safe

## Checklist — content & compression

- [ ] Parallel FlateEncode of page content streams (serialized write order preserved)
- [ ] Avoid per-cell heavy allocations when emitting BDC/EMC
- [ ] Lightweight cell marked-content write when StructElem already reserved

## Checklist — fonts & color

- [ ] ICC profiles built/compressed once at process init
- [ ] XMP template prefix reuse (fill dates/IDs only)
- [ ] Font subset cache optional (same face + same glyph set)

## Checklist — regression gates

- [ ] Re-run **veraPDF `-f 4`** on phase-5 fixtures after each major opt
- [ ] Re-run **veraPDF `-f ua2`** on same fixtures
- [ ] Re-run structure_tree_check
- [ ] Add micro-bench for: retail-like, table 2k rows, HFT-like scale
- [ ] Track: ns/op, B/op, allocs/op, peak heap under N workers

## Checklist — capacity debug (optional)

- [ ] Env flag to log buffer len/cap high-water
- [ ] Fail test if unexpected buffer grow during final emit (parity with gopdfsuit optional)

---

## Acceptance criteria

- [ ] No compliance regression on full fixture matrix
- [ ] Documented before/after numbers for at least one large-table fixture
- [ ] No new exports that couple layout to compliance modules

---

## Explicitly out of scope

- [ ] New PDF features (phase 7)
- [ ] HTTP server load tests
- [ ] Language binding overhead

---

## Done when

Compliant generation is stable under large-table load and benches are recorded; all phase-5 gates still green.
