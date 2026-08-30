# Phase 74: 3D transforms

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 74
> **Status:** complete (honest: 2D transforms implemented; 3D perspective/transform-style/backface-visibility unsupported)
> **Estimated effort:** L
> **Owner:** `internal/layout`
> **Depends on:** Phase 73
> **Unblocks:** Phase 75
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

2D `transform` and `transform-origin` are Implemented (`Matrix2D` in `internal/layout/transform.go`). 3D transforms, `perspective`, `perspective-origin`, `transform-style`, and `backface-visibility` remain **unsupported** in the catalog and matrix per `documentation/deferred.md`.

## Checklist

- [x] 74.1.1 Ownership list locked.
- [~] 74.2.1 3D transforms and perspective deferred as unsupported for 2D print output.
- [x] 74.2.2 Mapping entries for `backface-visibility`, `perspective`, `perspective-origin`, `transform-style` kept unsupported with honest notes.
- [x] 74.2.3 2D transform functions tested and verified (`TestParseTransformTranslateRotateScale`, etc.).
- [x] 74.2.4 Matrix §2.4 and mapping aligned on 2D Implemented / 3D Unsupported.
- [x] 74.3.1 `go test ./internal/layout -run "TestTransform"`; `--check`; gates. Proof: all exit 0.

## Forbidden proofs

- Citing `transform.go` reject comment / `TestParseTransformNoneAnd3DRejected` as Implemented proof
- Empty `code_path` Implemented rows
- 2D-only tests (`TestParseTransformTranslateRotateScale`) as 3D proof
