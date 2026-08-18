# Phase 38 - Closure

> **Parent:** `plans/0.2.4/31-canonical-0.2.4-roadmap.md`
> **Status:** not started
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

- [ ] `VERSION` → `0.2.4`
- [ ] `CHANGELOG.md`: Unreleased → `0.2.4` with **Breaking** section (library + CLI) and migration link
- [ ] README install / version badges or snippets cite `v0.2.4` when tagging
- [ ] `LibraryVersion` remains settings-surface id unless product decides otherwise (do not silently conflate with `VERSION`)

### 38.2 Cross-doc agreement

- [ ] `documentation/deferred.md` / overview do not advertise Converter as current API
- [ ] `documentation/MIGRATION-0.2.4.md` linked from CHANGELOG
- [ ] `plans/0.2.4/README.md` and parent `plans/README.md` status updated

### 38.3 Ledger closure

- [ ] Every phase file 31–37 and 39 has closure gates filled with commands/outcomes
- [ ] Parent status board 31–39 all `[x]` (38 last)
- [ ] Canonical ledger Overview status → complete (date)
- [ ] No duplicate active work left in older plans without `[~]` pointers
- [ ] Benchmark README + Makefile comments agree with Phase 39 path map

### 38.4 Final gates

- [ ] `make lint` → (record outcome)
- [ ] `make test` → (record outcome)
- [ ] Optional: `go run ./examples/pdf …` and `go run ./examples/image …` magic-byte check
- [ ] Optional: new CLI smoke on one golden fixture
- [ ] Optional: `make bench-cli-compare` / `./scripts/bench-external.sh --sizes=2 --runs=1` when host tools exist

### 38.5 Handoff

- [ ] List next work **not** in 0.2.4 (fidelity leftovers under `plans/0.2.0/`, deferred.md items, 500-page deferred)
- [ ] Explicitly state engine layout/CSS was not part of this release
- [ ] Parent Phase 38 row checked

### 38.6 Closure gates

- [ ] `make lint` → (record outcome)
- [ ] `make test` → (record outcome)
- [ ] Release-prep PR / notes drafted under `plans/0.2.4/PR/` when ready to tag
- [ ] Done

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phases 31–37 | Tagable tree |

---

## Out of scope

- Opening Phase 22/23
- Post-tag marketing site redesign beyond snippet accuracy
