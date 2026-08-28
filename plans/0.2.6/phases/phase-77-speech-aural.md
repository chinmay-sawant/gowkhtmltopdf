# Phase 77: Speech and aural

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 77
> **Status:** not started (19 names unsupported)
> **Estimated effort:** M
> **Owner:** catalog policy / optional `internal/layout`
> **Depends on:** Phase 76
> **Unblocks:** Phase 78
> **Honesty:** `../HONESTY-GATES.md`

---

## Owned names (19)

`cue`, `cue-after`, `cue-before`, `pause`, `pause-after`, `pause-before`, `rest`, `rest-after`, `rest-before`, `speak`, `speak-as`, `voice-balance`, `voice-duration`, `voice-family`, `voice-pitch`, `voice-range`, `voice-rate`, `voice-stress`, `voice-volume`

## Work order

Same fork as Phase 76:

**A. Permanent ignore** for aural CSS in a PDF engine (recommended unless product demands otherwise): `goal: ignore`, `engine_status: ignored`, matrix note.

**B. Real aural mapping into PDF structure / alt text** if product requires it: concrete consumer in `internal/pdf` tagging + tests. PDF/UA structure-tree tests alone are **not** `speak` support.

## Checklist

- [x] 77.1.1 Ownership list locked.
- [ ] 77.2.1 Choose A or B in writing.
- [ ] 77.2.2 Apply catalog/code accordingly.
- [ ] 77.2.3 No Implemented without consumer.
- [ ] 77.3.1 Gates.

## Forbidden proofs

- PDF structure / tagging tests as `speak` / `voice-*` Implemented proof
