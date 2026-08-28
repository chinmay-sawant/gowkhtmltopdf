# Phase 76: Pointer and form UI

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 76
> **Status:** not started (7 names unsupported)
> **Estimated effort:** M
> **Owner:** `internal/layout` / catalog policy
> **Depends on:** Phase 75
> **Unblocks:** Phase 77
> **Honesty:** `../HONESTY-GATES.md`

---

## Owned names (7)

`appearance`, `caret-color`, `cursor`, `pointer-events`, `resize`, `touch-action`, `user-select`

## Work order (code)

PDF has no mouse. Options (pick explicitly in this file before coding):

**A. Permanent print ignore:** set `goal: ignore`, `engine_status: ignored`, restore `PRINT_NOOP` enforcement in `scripts/css-catalog-map.py --check`, update matrix.

**B. Print fallbacks:** e.g. `cursor` stored but no-op with test that it parses into style; still usually stays ignored for Implemented bar. Implemented requires a visible/behavioral consumer (rare for these).

Do not mark Implemented without a consumer. Parsing alone is unsupported or ignored, not Implemented.

## Checklist

- [x] 76.1.1 Ownership list locked.
- [ ] 76.2.1 Choose A or B in writing here.
- [ ] 76.2.2 Code + script/matrix updates matching that choice.
- [ ] 76.2.3 No Implemented flips unless a real consumer exists.
- [ ] 76.3.1 `--check`; gates.

## Forbidden proofs

- Layout variable named `cursor` / link hit-testing comments as CSS `cursor` support
- Invented test names
