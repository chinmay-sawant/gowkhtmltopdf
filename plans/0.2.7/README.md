# Plans 0.2.7 - Next 72 and next 100 CSS properties

> **Branch:** `feature/027-next-72` (next-100 authored on `docs/027-next-100` then merged)
> **Status:** next-72 complete (87.1-87.8 closed 2026-09-17); next-100 planned (88.1-88.9)
> **Baseline:** 354 Implemented / 0 Partial / 464 Unsupported (v0.2.6 catalog)
> **After next-72 honest flips:** 407 Implemented / 0 Partial / 411 Unsupported of 818 (53 of 72 Implemented; 0 Partial; 19 Unsupported by choice)

## What this folder is

Execution ledger for two CSS waves:

1. The next 72 print-relevant properties in `next-72-properties.json`, proven by
   `testdata/golden/fixture-64-next-72-props.html`.
2. The next 100 after that, in `next-100-properties.json`. Zero overlap with the
   72. Fixture contract: `next-100-fixture-authoring.md` (fixture-65).

## Start here

| File | Role |
|------|------|
| [87-canonical-0.2.7-next-72.md](87-canonical-0.2.7-next-72.md) | Next-72 parent: batches 87.1-87.8 |
| [88-canonical-0.2.7-next-100.md](88-canonical-0.2.7-next-100.md) | Next-100 parent: batches 88.1-88.9 |
| [phases/](phases/) | Per-batch atomic checklists (87.1-87.8 and 88.1-88.9) |
| [next-72-properties.json](next-72-properties.json) | 72-property inventory + Chrome BCD labels |
| [next-100-properties.json](next-100-properties.json) | 100-property inventory + Chrome BCD labels |
| [next-72-fixture-authoring.md](next-72-fixture-authoring.md) | Fixture-64 authoring contract |
| [next-100-fixture-authoring.md](next-100-fixture-authoring.md) | Fixture-65 authoring contract |

## Gate policy (this version only)

| When | Allowed | Forbidden |
|------|---------|-----------|
| Mid-batch | `go test ./internal/<pkg> -run '…'` (and sibling package tests) | `make lint`, `make test`, bare `go test ./...` |
| Final batch only (`phase-87.8-closure-integration.md` or `phase-88.9-closure-integration.md`) | `make test`, `make golden`, catalog `--check`, claim-scan if docs touched | Still no `make lint` (owner runs lint manually after) |

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
