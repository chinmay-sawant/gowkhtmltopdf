# Phase 29 - Verification

> **Parent:** `plans/0.2.1/24-canonical-0.2.1-roadmap.md`
> **Status:** not started
> **Estimated effort:** 1–2 weeks
> **Depends on:** Phase 25 needles for new table fixtures; other needle work can start immediately
> **Unblocks:** Phase 30 release gates

---

## Overview

v0.2.0 goldens are structural (header, xref, page envelope, optional
`/FontFile2` / `/Image`). That will not catch a scrambled flex order.
CI `push` listens on `main` while contributors branch from `master`.
There are no `Fuzz*` targets on the three parsers that accept untrusted
bytes. This phase adds cheap, named checks without turning the corpus
into pixel snapshots.

## Executive Summary

| Gate | Today | Target |
|------|-------|--------|
| Golden fixtures | Page bounds; ~8 text needles | Every layout-claiming fixture has at least one needle or geometry lock |
| CSS/HTML/PDF parsers | Panic smokes only | `Fuzz` on `html.Parse`, `css.Parse`, and PDF write of fuzz HTML |
| Race | `convert` `layout` `pdf` `imageout` `load` | Same list plus `./...` weekly or on PR if time-bounded |
| CI branch | `push: branches: [main]` | Matches `CONTRIBUTING.md` (`master`) |
| Claim scan | Makefile target | Still required; add phrases if 30’s ledger pass needs them |

---

## Phase 29 checklist

### 29.1 Fixture needles

- [ ] Inventory `fixturePageBounds` in `internal/convert/golden_test.go`: which entries have text needles vs page count only
- [ ] For each fixture that claims a layout feature (flex 25/28/32, grid 33–35, float 22/29/38, thead 23, orphans 30/37, CJK 27, sticky 31, multicol 39, `:has` 41, container 42), add a needle or a display-list geometry assert
- [ ] Fixture 25: flex item order is asserted (A/B/C or equivalent)
- [ ] Do not byte-lock PDF output
- [ ] `GOLDEN_APPROVE=1` still does not rewrite committed HTML
- [ ] Path: `internal/convert/golden_test.go` and/or existing `internal/layout/fixture_*_test.go`

### 29.2 Fuzz

- [ ] `FuzzParseHTML` on `internal/html` — must not panic; may return error
- [ ] `FuzzParseCSS` on `internal/css` — must not panic
- [ ] `FuzzConvertHTML` (or PDF write of `html.Parse` output) with a byte/page cap so the fuzzer cannot allocate unbounded documents
- [ ] Seed corpus: a handful of `testdata/golden` snippets, not all 56 PDFs
- [ ] CI: `go test -fuzz=Fuzz -fuzztime=20s` on those packages in the test job **or** a dedicated job with a time cap
- [ ] No third-party fuzz harness required

### 29.3 CI

- [ ] `.github/workflows/ci.yml` `push.branches` includes the integration branch named in `CONTRIBUTING.md` (`master`)
- [ ] If `main` is unused, drop it or document a mirror; do not listen only on a branch nobody pushes
- [ ] Race job: keep the hot-package list; add a note in the workflow why `./...` is or is not included
- [ ] `[~]` `go test -race ./...` on every PR — only if wall time stays acceptable; otherwise nightly / weekly
- [ ] `claim-scan` stays in the test job

### 29.4 Semantic oracle

- [ ] `internal/pdf/semantic_oracle_test.go` covers more than the current converted fixtures **or** the five-file set is listed as the accepted minimum in `testdata/golden/README.md`
- [ ] Extracted text / dests / annots stay the oracle; do not add pixel hashes here

### 29.5 Unfinished unit tests

- [ ] Close or delete placeholder asserts (flex shrink `_ = aw` is Phase 26; grep again in 29)
- [ ] `TestParseNeverPanics` may remain as a tiny smoke next to the new fuzz target
- [ ] `quality_test.go` fixture-read failures **fail** the test; do not `Skip` a missing committed fixture

### 29.6 Closure gates

- [ ] `make lint` →
- [ ] `make test` →
- [ ] `go test -fuzz=FuzzParseHTML -fuzztime=10s ./internal/html` → (record)
- [ ] `go test -fuzz=FuzzParseCSS -fuzztime=10s ./internal/css` → (record)
- [ ] Parent Phase 29 row checked
- [ ] Next: Phase 30

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 25/26 feature tests | Phase 30 release |
| Existing `claim-scan` | Honesty bar unchanged |

---

## Out of scope

- Image-diff / screenshot CI as the default golden
- Full-graph race on every push if it exceeds the current runner budget
- Coverage percentages as a gate
