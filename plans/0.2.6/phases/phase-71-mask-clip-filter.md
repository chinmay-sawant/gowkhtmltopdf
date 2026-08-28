# Phase 71: Mask, clip, and filter

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 71
> **Status:** complete (honest: filter is Partial opacity-only; clip-path/masks deferred as unsupported)
> **Estimated effort:** L
> **Owner:** `internal/layout` paint
> **Depends on:** Phase 70
> **Unblocks:** Phase 72
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

`filter` is honestly kept **Partial** (parses `opacity()` into `ResolvedStyle.Opacity` in `style_properties.go` and `transform.go`). `clip-path`, `clip`, `backdrop-filter`, and `mask*` properties remain `unsupported`.

## Checklist

- [x] 71.1.1 Ownership list locked.
- [~] 71.2.1 `clip-path` / `clip` deferred as unsupported per print engine scope.
- [x] 71.2.2 `mask*` properties left unsupported in mapping.
- [x] 71.2.3 `filter` kept Partial (opacity-only) with notes in `mapping.json` and matrix §2.4.
- [x] 71.2.4 Matrix and mapping aligned on Partial `filter` and unsupported `mask*`/`clip*`.
- [x] 71.3.1 `go test ./internal/layout -run "TestFilter|TestTransform|TestOpacity"`; `--check`; gates. Proof: all exit 0.

## Forbidden proofs

- `TestOverflowClip` as proof of `clip-path`
- Marking full `filter` Implemented while only `opacity()` works
