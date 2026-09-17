# Plans 0.2.7 - Next 72 CSS properties

> **Branch:** `feature/027-next-72`
> **Status:** complete (87.1–87.8 closed 2026-09-17)
> **Baseline:** 354 Implemented / 0 Partial / 464 Unsupported (v0.2.6 catalog)
> **After honest flips:** 397 Implemented / 10 Partial / 411 Unsupported of 818 (43 of 72 Implemented; 10 Partial; 19 Unsupported by choice)

## What this folder is

Execution ledger for implementing the next 72 print-relevant CSS properties listed in
`next-72-properties.json`, proven by `testdata/golden/fixture-64-next-72-props.html`.

## Start here

| File | Role |
|------|------|
| [87-canonical-0.2.7-next-72.md](87-canonical-0.2.7-next-72.md) | Canonical parent: batches, effort scale, architecture rules |
| [phases/](phases/) | Per-batch atomic checklists (87.1 … 87.8) |
| [next-72-properties.json](next-72-properties.json) | Property inventory + Chrome BCD labels |
| [next-72-fixture-authoring.md](next-72-fixture-authoring.md) | Fixture-64 authoring contract |

## Gate policy (this version only)

| When | Allowed | Forbidden |
|------|---------|-----------|
| Mid-batch | `go test ./internal/<pkg> -run '…'` (and sibling package tests) | `make lint`, `make test`, bare `go test ./...` |
| Final batch only (`phase-87.8-closure-integration.md`) | `make test`, `make golden`, catalog `--check`, claim-scan if docs touched | Still no `make lint` (owner runs lint manually after) |

## Soft file-size rule

Do not grow allowlisted files further:

- `internal/layout/layout.go` (2497)
- `internal/layout/style_properties.go` (2015 after gap+multicol extracts)
- `internal/imageout/imageout.go` (2010)

New apply arms go in focused `style_*_props.go` files registered on `styleGroups`.
Split consumers by responsibility (`shape_exclusion.go`, `object_fit.go`, …).

## Honesty

Every Implemented flip needs apply arm + consumer + package test + matrix note, then
mapping last. See `plans/0.2.6/HONESTY-GATES.md`.
