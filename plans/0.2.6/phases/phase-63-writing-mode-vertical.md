# Phase 63: writing-mode vertical

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 63
> **Status:** complete (honest: writing-mode kept Partial with documented glyph-rotate behavior, matrix §2.3 updated, full vertical line flow deferred)
> **Estimated effort:** XL
> **Owner:** `internal/layout`
> **Depends on:** Phase 62
> **Unblocks:** logical-box vertical mapping leftovers
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

`writing-mode` is honestly kept **Partial**. Vertical values set `RotateDeg == -90` and adjust box height from rotated run width (`inline_paint.go`, `layout.go`). Full vertical line progression and orthogonal block flow are explicitly deferred per `documentation/deferred.md`.

## Checklist

- [x] 63.1.1 Owned property: `writing-mode` (in mapping as partial with honest notes).
- [~] 63.2.1 Full vertical line progression deferred to post-0.2.x per `documentation/deferred.md`.
- [x] 63.2.2 Existing tests: `TestWritingModeInherits` in `style_cascade_test.go` and rotate ops in `inline_paint.go`.
- [x] 63.2.3 Update matrix §2.3 / §3 and `documentation/deferred.md` so they align on Partial status. Proof: Matrix §2.3 updated to Partial.
- [x] 63.2.4 Mapping confirmed as Partial with code_path `internal/layout/style_properties.go` and honest notes.
- [x] 63.3.1 `go test ./internal/layout -run "TestWritingMode"`; `--check`; `make test` / `make lint`. Proof: exit 0.

## Forbidden proofs

- `RotateDeg == -90` as sole success criterion
- Inherit-only tests

## Handoff

Phase 67 closure stays reopen until this and other honesty demotes are resolved or explicitly `[~]`.
