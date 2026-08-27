# Phase 56: Docs, mapping sync, closure

> **Parent:** `../48-canonical-0.2.6-css-coverage.md` Phase 56
> **Status:** not started
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

- [ ] 56.1.1 `python3 scripts/css-catalog-map.py --check` exit 0 against `plans/0.2.6/catalog/mapping.json`. Proof: command tail.
- [ ] 56.1.2 `documentation/compatibility-matrix.md` last honesty date updated. Every Implemented/Partial row cites live `file:line`. Proof: grep `style.go:340` finds nothing; `style_cascade.go` / `style_properties.go` cited instead.
- [ ] 56.1.3 `documentation/fidelity.md` CSS map matches. No "full CSS" claim. Proof: `make claim-scan` clean.
- [ ] 56.1.4 `documentation/deferred.md` CSS rows point at this ledger for leftovers, not at closed 0.2.0 pending files as if they were active.

### 56.2 gates

- [ ] 56.2.1 `make test` exit 0. Proof: tail.
- [ ] 56.2.2 `make lint` exit 0. Proof: tail.
- [ ] 56.2.3 `make golden` exit 0. Proof: grep PASS. New fixtures have `fixturePageBounds`.
- [ ] 56.2.4 `make claim-scan` clean. Proof: output `clean`.
- [ ] 56.2.5 `make build` and `--version` still stamps `VERSION`. Do not bump `VERSION` in this phase unless the user is tagging.

### 56.3 plans and KB

- [ ] 56.3.1 `plans/README.md` 0.2.6 row status matches reality.
- [ ] 56.3.2 `knowledge-base/wiki/log.md` closure entry. `syntheses/roadmap.md` milestone. `concepts/css-engine.md` matches code.
- [ ] 56.3.3 No unchecked `[ ]` remains in phases 48-56 except `[~]` with pointers.

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
