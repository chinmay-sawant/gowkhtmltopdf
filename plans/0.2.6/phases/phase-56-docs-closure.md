# Phase 56: Docs, mapping sync, closure

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 56
> **Status:** complete - gates 2026-08-27
> **Estimated effort:** 2-3 days
> **Owner:** ledger
> **Depends on:** Phases 48-55 all `[x]` or `[~]` with pointers
> **Unblocks:** v0.2.6 tag when the rest of the release is ready

---

## Overview

Prove the mapping matches code, the matrix matches code, and the gates are green. This is the only phase that can mark the canonical ledger complete.

Unshipped rows from 49-55 must be `[~]` with reason. Do not `[x]` this phase while open `[ ]` remain in earlier phases.

## Goals

- Mapping `--check` green
- Matrix and fidelity honest
- `make test` / `make lint` / `make golden` / `make claim-scan` exit 0
- KB and `plans/README.md` in sync

## Checklist

### 56.1 mapping and contract

- [x] 56.1.1 `python3 scripts/css-catalog-map.py --check` exit 0. Proof: check ok 157 apply arms.
- [x] 56.1.2 Matrix honesty. Proof: no `style.go:340`; `make claim-scan` clean.
- [x] 56.1.3 fidelity.md CSS map. Proof: `make claim-scan` clean.
- [x] 56.1.4 deferred.md points at 0.2.6 ledger.

### 56.2 gates

- [x] 56.2.1 `make test` exit 0 (2026-08-27).
- [x] 56.2.2 `make lint` exit 0 (golangci-lint v1.64.8 + frontend eslint).
- [x] 56.2.3 `make golden` exit 0. TestGoldenCorpusAllFixtures PASS including fixture-56.
- [x] 56.2.4 `make claim-scan` clean.
- [x] 56.2.5 `make build`; `./bin/gowkhtmltopdf --version` is 0.2.5 matching VERSION. No tag this phase.

### 56.3 plans and KB

- [x] 56.3.1 plans/README.md 0.2.6 row.
- [x] 56.3.2 KB log/roadmap/css-engine updated this session.
- [x] 56.3.3 Remaining open work is `[~]` with pointers (fixture-gated flex leftovers). Phase 54 `page: ident` and margin boxes shipped lite.

## Dependencies

All prior phases closed on evidence.

## Evidence

Gate logs. Mapping check. Matrix diff.

## Out of scope

Tagging v0.2.6 without user ask. Reopening WOFF2. Phase 22/23.

## Handoff

Declare the canonical ledger complete only when every row above has proof. Next work is whatever `[~]` leftovers this ledger still lists, or a new version folder.

## Required checks

- Docs-only: skip lint/test.
- Otherwise: `make lint` and `make test` before marking complete. Leave unchecked if either fails.
