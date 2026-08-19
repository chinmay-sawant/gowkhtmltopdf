# Phase 38 - Closure

> **Parent:** `plans/0.2.4/31-canonical-0.2.4-roadmap.md`
> **Status:** complete — validated 2026-08-18
> **Estimated effort:** 2–3 days
> **Depends on:** Phases 31–37 and 39
> **Unblocks:** v0.2.4 tag / release

---

## Overview

Close the v0.2.4 ledger: version metadata, changelog, full gate evidence
(including the three-engine benchmark path freeze), and an explicit handoff
for work that remains outside this release.

## Executive Summary

| Gate | Requirement |
|------|-------------|
| VERSION | `0.2.4` |
| CHANGELOG | Breaking Document API + CLI redesign notes |
| Lint / test | Green on the final tree |
| Ledger | Status board all `[x]` with recorded outcomes |

---

## Phase 38 checklist

### 38.1 Release metadata

- [x] `VERSION` → `0.2.4`
- [x] `CHANGELOG.md`: Unreleased → `0.2.4` with **Breaking** section (library + CLI) and migration link
- [x] README install / version badges or snippets cite `v0.2.4` when tagging
- [x] `LibraryVersion` remains settings-surface id unless product decides otherwise (do not silently conflate with `VERSION`)

### 38.2 Cross-doc agreement

- [x] `documentation/deferred.md` / overview do not advertise Converter as current API
- [x] `documentation/MIGRATION-0.2.4.md` linked from CHANGELOG
- [x] `plans/0.2.4/README.md` and parent `plans/README.md` status updated

### 38.3 Ledger closure

- [x] Every phase file 31–37 and 39 has closure gates filled with commands/outcomes
- [x] Parent status board 31–39 all `[x]` (38 last)
- [x] Canonical ledger Overview status → complete (date)
- [x] No duplicate active work remains in older plans
- [x] Benchmark README + Makefile comments agree with Phase 39 path map

### 38.4 Final gates

- [x] `make lint` → passed; Go lint and frontend lint/content validation are clean
- [x] `make test` → passed with GOCACHE=/tmp/gowkhtmltopdf-go-cache
- [x] Optional: `go run ./examples/pdf …` and `go run ./examples/image …` magic-byte check
- [x] Optional: new CLI smoke on one golden fixture
- [x] Optional: `make bench-cli-compare` / `./scripts/bench-external.sh --sizes=2 --runs=1` when host tools exist

### 38.5 Handoff

- [x] List next work **not** in 0.2.4 (fidelity leftovers under `plans/0.2.0/`, deferred.md items, 500-page deferred)
- [x] Explicitly state engine layout/CSS was not part of this release
- [x] Parent Phase 38 row checked

### 38.6 Closure gates

- [x] `make lint` → passed; Go lint and frontend lint/content validation are clean
- [x] `make test` → passed with GOCACHE=/tmp/gowkhtmltopdf-go-cache
- [x] Release-prep PR / notes drafted under `plans/0.2.4/PR/` when ready to tag
- [x] Done

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phases 31–37 | Tagable tree |

---

## Out of scope

- Opening Phase 22/23
- Post-tag marketing site redesign beyond snippet accuracy

## Validation record (2026-08-18)

- `VERSION` is `0.2.4`; `CHANGELOG.md`, README, migration links, plan indexes, and the canonical roadmap agree on the release and hard break.
- `GOCACHE=/tmp/gowkhtmltopdf-go-cache make test`, `go test -race ./...`, `make build`, frontend lint/build, CLI/example smokes, and benchmark smokes passed. No Git commands were run by the coordinator.
- Full `make lint` was run after the implementation and frontend build; Go lint and frontend lint/content validation both pass. No linter policy was changed to hide findings.
