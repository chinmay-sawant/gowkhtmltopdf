# Phase 69: Vendor-prefix aliases

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 69
> **Status:** complete (22 webkit aliases implemented/partial with tests; unsupported unprefixed remain unsupported)
> **Estimated effort:** M
> **Owner:** `internal/layout` cascade/apply
> **Depends on:** Phase 68
> **Unblocks:** Phase 70
> **Honesty:** `../HONESTY-GATES.md`

---

## Overview

Implemented vendor-prefix aliasing for `-webkit-*` properties mapped onto existing unprefixed engines via `normalizeVendorPrefix` in `style_cascade.go`.

## Checklist

- [x] 69.1.1 Ownership list locked (above).
- [x] 69.2.1 Implement alias helper + wire into cascade/apply. Proof: `normalizeVendorPrefix` in `internal/layout/style_cascade.go:785`.
- [x] 69.2.2 Prefixed-name tests for each alias claimed. Proof: `TestWebkitPrefixAliases` in `internal/layout/style_cascade_test.go`.
- [x] 69.2.3 Flip mapping only for aliases with flip packets (20 implemented, 2 partial); leave others unsupported. Proof: `mapping.json` and `coverage-summary.json`.
- [x] 69.3.1 `go test ./internal/layout -run "TestWebkit|TestFlex|TestRadius|TestTransform|TestBoxShadow"`; `--check`; `make test` / `make lint`. Proof: all exit 0.

## Forbidden proofs

- Citing unprefixed tests only
- Mass-flipping all 70 because “webkit is like the standard property”
