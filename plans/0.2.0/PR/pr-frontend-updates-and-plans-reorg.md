# chore(frontend): documentation site UI/UX overhaul and plans versioning

## Summary

Comprehensive documentation site UI/UX overhaul across all views (Overview, Getting Started, Docs, Showcase, Benchmarks, Dossier), along with reorganization of implementation ledgers under versioned `plans/0.1.0/` and `plans/0.2.0/` directories and linter cleanups in `api_test.go`.

---

## Motivation / context

- **Plans:** [`plans/0.2.0/frontend-improves/phase-wise-checklist.md`](plans/0.2.0/frontend-improves/phase-wise-checklist.md), [`plans/README.md`](plans/README.md)
- **Goals:**
  1. Complete phases 1–9 of the frontend design and UX improvement plan for the documentation site.
  2. Structure the `plans/` hierarchy cleanly by release versions (`0.1.0/` for MVP release and `0.2.0/` for active post-MVP work).
  3. Ensure zero lint warnings while retaining all test assertions and defensive nil-checks.

---

## Changes

### 1. Frontend & Documentation Site UI/UX Overhaul
- **Command Palette (`⌘K` / `Ctrl+K`):** Added responsive keyboard shortcut labels (Mac vs Linux/Windows), category icons, fast multi-term search matching CLI flags, documentation topics, and showcase samples.
- **Header & Navigation:** Added live GitHub star counter with `localStorage` caching and API refresh, improved theme switcher (Dark/Light mode), and polished mobile navigation menu.
- **Showcase:** Added high-resolution rendered screenshot thumbnails and generator scripts (`scripts/generate_showcase_thumbs.py`, `scripts/screenshot_showcase.py`).
- **Typography & Layout:** Optimized typography hierarchy, table cell alignment, code snippet blocks, copy buttons, and callout containers across all pages.
- **Built-with Credits:** Updated footer engineering credits in UI and documentation.

### 2. Plans Reorganization & Version Partitioning
- **`plans/0.1.0/`:** Partitioned all 23 historical files present at tag `v0.1.0` (phases 00–09 checklists, pure-Go rewrite ledger, exploration notes, and initial PR/issue bodies).
- **`plans/0.2.0/`:** Organized all post-MVP artifacts (phases 10–23, performance profiles, architecture reviews, frontend ledgers, amendments, deferred items, and post-MVP PR bodies).
- **Index & Relative Links:** Created top-level [`plans/README.md`](plans/README.md), dedicated version READMEs, and resolved all relative markdown links across [`documentation/`](documentation/) and [`CONTRIBUTIONS.md`](CONTRIBUTIONS.md).

### 3. API & Test Maintenance
- **`api_test.go`:** Converted `nil` context literals in `TestConverterValidationErrorsReachOnError` to typed `var nilCtx context.Context` variables, eliminating staticcheck SA1012 linter warnings while preserving nil-context preflight validation tests.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Zero regression; frontend assets bundled and minified. |
| **Memory** | No impact on Go runtime or memory allocations. |
| **Behavior / correctness** | Unchanged conversion logic; improved documentation and UI experience. |
| **API / CLI** | Pure-Go library API and CLI arguments unchanged and backward-compatible. |
| **Dependencies** | No new Go dependencies added. |
| **Binary size / build time** | Unchanged (`CGO_ENABLED=0` builds verified). |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | Fully backward-compatible |

---

## Test plan

- [x] `go test ./...` passed across all packages
- [x] `golangci-lint run ./...` passed with zero errors
- [x] All relative markdown links verified across `documentation/` and `plans/`
- [x] Frontend build and preview tested

### Commands

```sh
go test ./...
golangci-lint run ./...
```

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`enhancement`, `documentation`)
- [x] Filled body committed under `plans/0.2.0/PR/pr-frontend-updates-and-plans-reorg.md`
