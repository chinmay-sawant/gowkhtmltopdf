# Tier 2 Subplan - Phase 18 Pending (Pagination docs + optional orphans)

> **Parent:** [`plans/phases/phase-18-pagination-polish.md`](../phase-18-pagination-polish.md) — Pending (after #17)  
> **Status:** not started  
> **Estimated effort:** 0.5 day docs + 0–1 day optional fixture/tests  
> **Depends on:** [00-shared-doc-honesty.md](00-shared-doc-honesty.md) for matrix/fidelity  
> **Constraint:** stdlib-only; CSS `orphans`/`widows` property parsing remains out of scope unless amended

---

## Overview

Phase 18 **core is shipped**: `<thead>` repeat, orphan/widow **heuristics**, `--zoom`
forwarding, smart-shrinking re-layout. Remaining work is honesty docs (matrix still
describes pre-phase-18 state), optional CLI/library notes, and an optional orphans
fixture. Do **not** implement CSS Fragmentation `orphans`/`widows` properties
([css-break-3](https://www.w3.org/TR/css-break-3/)) unless product amends scope.

## Executive Summary

| Pending item (parent) | Disposition | Primary work |
|-----------------------|-------------|--------------|
| Matrix §2.6 orphans/widows | **Must** | Partial (heuristics); CSS props absent |
| Fidelity + matrix pagination blurbs | **Must** | Shared Pass 0 — thead/zoom/orphan prose |
| CLI docs thead repeat | Should | `cli.md` + `library-api.md` notes |
| Optional orphans fixture | Nice-to-have → **included** | New `fixture-30-orphans-heuristic.html` |

---

## Phase 1: Evidence baseline (scanned 2026-08-05)

### 1.1 Shipped behavior

| Feature | Path | Proof |
|---------|------|-------|
| thead repeat | `internal/layout/paint.go` `repeatTableHeaders` | `pagination_thead_test.go`; `fixture-23-thead-repeat.html` |
| Header row detect | `layout.go` `buildTable` → `box.headerRows` | multi-header-row + nested own thead |
| Orphan/widow heuristic | `paint.go` `orphansWidows` (~L479–514) | short blocks straddling page moved via `shiftFlowY` |
| Keep-with-next heading | `paint.go` `keepHeadingWithNext` | h1–h6 with &lt;~24pt room moved |
| Zoom | `layout.Options.Zoom` + `convert.go` forward `obj.Load.ZoomFactor` | `TestZoom`; CLI `--zoom` |
| Smart-shrinking | `convert.go` re-`Layout` with `effZoom` | `TestRunPDFSmartShrinking` |

### 1.2 CSS property gap (intentional)

- [x] Confirmed: zero matches for `orphans`/`widows` under `internal/css/`
- [x] Confirmed: `applyRestProps` (`style.go`) handles `page-break-*` / `break-*` only
- [x] Spec note: CSS orphans = min lines left **before** break; widows = min lines **after** break (default 2). Our heuristic is geometric block-band, **not** author-valued CSS.

---

## Phase 2: Documentation honesty (shared + phase-owned)

### 2.1 Shared matrix / fidelity (must)

Owned by [00-shared-doc-honesty.md](00-shared-doc-honesty.md) §2.3 / §3.1:

- [ ] Matrix Pagination prose: remove “zoom not forwarded / smart-shrink warn-only / thead not implemented / orphan none”
- [ ] Matrix §2.6: `orphans`/`widows` → **Partial (heuristics)**; note CSS properties not parsed
- [ ] `fidelity.md` feature map: thead repeat shipped (not “thead repeat no”)
- [ ] Re-cite current code lines (`convert.go` ~355–400; `paint.go` `repeatTableHeaders` / `orphansWidows`) — old `convert.go:218-229` cites are obsolete
- [ ] Proof: `rg -n 'thead repeat no|does not forward|orphan/widow control not implemented' documentation/` → empty for false claims

### 2.2 CLI / library notes (should)

- [ ] `documentation/cli.md`: subsection **Pagination & tables**
  - thead / `table-header-group` repeats on continuation pages
  - `--zoom` scales layout; `--smart-shrinking` may re-layout to fit width
  - orphans/widows: automatic heuristics only; CSS properties ignored
- [ ] `documentation/library-api.md`: same behavioral note or pointer to matrix §2.6
- [ ] Proof: `rg -n 'thead|smart-shrink|orphan' documentation/cli.md documentation/library-api.md`

### 2.3 Samples inventory (optional hygiene)

- [ ] `documentation/samples.md`: fixture range includes 23+ (not “fixture-01 … fixture-21” only)

### 2.4 Already honest (do not regress)

- [x] `README.md` deferred table marks thead repeat **Shipped**
- [x] Matrix §7.3 / `--zoom` Supported rows

---

## Phase 3: Orphans fixture (new file only; do not edit fixture-11/23)

### 3.1 Design constraints

Heuristics depend on content height vs `contentH` and font metrics — brittle as
pixel goldens. Prefer:

1. New golden with **loose** page-count envelope (`minPages: 2`, no tight max), **or**
2. Unit test on layout ops (`orphansWidows` / `keepHeadingWithNext`)

Do **not** modify existing fixtures (especially fixture-11 / fixture-23).

### 3.2 New fixture checklist

- [x] Add `testdata/golden/fixture-30-orphans-heuristic.html` with:
  - enough filler / page-break to guarantee ≥2 pages
  - short straddler sentinel `ORPHAN-SHORT-STRADDLE`
  - heading + body sentinels for keep-with-next
  - explicit honesty: CSS `orphans`/`widows` **not** parsed
- [x] Envelope in `fixturePageBounds`
- [x] Document in `testdata/golden/README.md`
- [x] Proof: `go test ./internal/convert -run 'TestGoldenCorpusAllFixtures/fixture-30' -count=1` → pass
- [ ] Optional follow-up: `TestOrphansWidowsHeuristic` unit test under `internal/layout/`

**Shipped on `feature/tier-2-pending-2`:** fixture-30 added (existing fixtures untouched).

### 3.3 Explicit non-goal

- [~] Parsing CSS `orphans` / `widows` integers — **out of scope** (parent Out of scope + css-break-3 full semantics)

---

## Phase 4: Closure gates

### 4.1 Required

- [ ] Shared doc-honesty pagination + §2.6 + fidelity thead row done
- [ ] Parent Phase 18 Pending matrix/fidelity/CLI items updated
- [ ] If code/fixtures: `make lint` → ; `make test` → ; record outcomes

### 4.2 Docs-only path

- [ ] Documentation-only changes: skill says skip lint/test; still visually review matrix

### 4.3 Next

- [ ] Phase 19 `@font-face` audit or Phase 20 HF fragment GoTo

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 5 fragmenter + table layout | Already shipped thead geometry |
| Shared doc-honesty | Closable stale matrix prose |
| Optional unit tests | Confidence without brittle goldens |

---

## Out of scope

- Full CSS Paged Media Level 3 / Fragmentation Level 3 property fidelity
- Named pages / running elements beyond existing HF
- Footnote regions
- CSS `orphans` / `widows` property parsing
