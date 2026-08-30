# Phase 77: Speech and aural

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 77
> **Status:** complete (honest: aural and speech properties unsupported for visual print PDF output)
> **Estimated effort:** M
> **Owner:** catalog policy / optional `internal/layout`
> **Depends on:** Phase 76
> **Unblocks:** Phase 78
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

PDF is a visual print format without aural speech synthesis. All 19 speech and aural properties (`cue*`, `pause*`, `rest*`, `speak*`, `voice-*`) remain **unsupported** in the catalog with honest notes.

## Checklist

- [x] 77.1.1 Ownership list locked.
- [x] 77.2.1 Policy A (Permanent ignore for aural/speech CSS in visual PDF engine).
- [x] 77.2.2 Mapping and matrix notes kept honest.
- [x] 77.2.3 All 19 names verified as unsupported in `mapping.json`.
- [x] 77.3.1 `python3 scripts/css-catalog-map.py --check`; `make test`; `make lint`. Proof: all exit 0.

## Forbidden proofs

- PDF structure / tagging tests as `speak` / `voice-*` Implemented proof
