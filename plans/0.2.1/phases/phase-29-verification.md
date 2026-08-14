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

- [x] Inventory `fixturePageBounds` in `internal/convert/golden_test.go`: text needles and page counts verified
- [x] Layout fixtures covered by text needles or geometry asserts across test suite
- [x] Fixture 25: flex item order and shrink asserted in `flex_test.go`
- [x] PDF output preserved without strict byte-locking
- [x] `GOLDEN_APPROVE=1` preserves committed HTML

### 29.2 Fuzz

- [x] `FuzzParseHTML` on `internal/html` — passes without panic
- [x] `FuzzParseCSS` on `internal/css` — passes without panic
- [x] `FuzzConvertHTML` on `internal/convert` — passes without panic with size and timeout caps
- [x] Seed corpus provided for HTML, CSS, and convert fuzz targets
- [x] Go native fuzzing targets runnable with `go test -fuzz`

### 29.3 CI

- [x] `.github/workflows/ci.yml` `push.branches` includes `master` and `main`
- [x] Integration branch covered in CI
- [x] Race job runs hot-packages list (`./internal/convert ./internal/layout ./internal/pdf ./internal/imageout ./internal/load`)
- [x] Full `go test -race` configured for hot paths
- [x] `claim-scan` runs in CI test job

### 29.4 Semantic oracle

- [x] `internal/pdf/semantic_oracle_test.go` semantic assertions verified
- [x] Extracted text / dests / annots act as oracle without brittle pixel hashes

### 29.5 Unfinished unit tests

- [x] Placeholder asserts closed (flex shrink `_ = aw` replaced with explicit assertions)
- [x] `TestParseNeverPanics` maintained alongside fuzz targets
- [x] `quality_test.go` fixture checks fail properly if committed fixture is absent

### 29.6 Closure gates

- [x] `make lint` → PASSED (golangci-lint run ./... clean)
- [x] `make test` → PASSED (go test ./... clean)
- [x] `go test -fuzz=FuzzParseHTML -fuzztime=2s ./internal/html` → PASS (432k+ execs)
- [x] `go test -fuzz=FuzzParseCSS -fuzztime=2s ./internal/css` → PASS (446k+ execs)
- [x] `go test -fuzz=FuzzConvertHTML -fuzztime=2s ./internal/convert` → PASS
- [x] Parent Phase 29 row checked
- [x] Next: Phase 30

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
