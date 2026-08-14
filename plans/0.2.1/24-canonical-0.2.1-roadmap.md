# 24 - v0.2.1 Contracts, Print Layout, and Verification (Canonical Execution Ledger)

> **Parent:** `plans/0.2.0/10-canonical-post-mvp-roadmap.md` (phases 10–23)
> **Status:** active — not started (2026-08-14)
> **Estimated effort:** 6–10 weeks across phases 24–30
> **Constraint:** pure Go, no CGO, no browser embed. Direct modules stay on the existing allowlist (`go-text/typesetting`, `tdewolff/canvas`) unless a new amendment is filed.
> **Ordering principle:** public-contract correctness first, then print-layout correctness, then internal seams, then verification, then docs/release.
> **Workflow:** [`skills/phase-wise-checklist/SKILLS.md`](../../skills/phase-wise-checklist/SKILLS.md)

---

## Overview

v0.2.0 shipped the HTML template pipeline (load → parse → style → layout → paginate → write) and the wkhtmltopdf-shaped CLI/library surface. v0.2.1 does not add JavaScript, Chrome parity, or new output formats. It tightens contracts callers already rely on, replaces a few print-layout heuristics with documented models, and makes regressions cheaper to catch.

**In scope**

1. One preferred library path, with a single nil / error / panic policy.
2. Print pagination and tables that fail on content, not only on page count.
3. Targeted flow-layout fixes (floats, margin collapse) that stay inside the existing print CSS subset.
4. Clearer internal seams so PDF and image sinks do not both interpret a kitchen-sink `Op`.
5. Fuzz, fixture needles, and CI that match the documented `master` branch.
6. Ledgers and package comments that match `go.mod` and the current tree.

**Hard non-goals (unless this ledger is amended)**

- JavaScript / Phase 22
- Open-web / Chrome print parity / Phase 23
- CGO, C ABI, PDF/A, encryption, SOCKS5, SVG image output
- Replacing the HTML tokenizer with a full HTML5 tree constructor
- Pixel-diff goldens as the default gate

---

## Phase map

```text
24 Library contract
  → 25 Pagination and tables
  → 26 Flow layout (floats, margins)
      → 27 Layout / paint seam
      → 28 Settings and request types
          → 29 Verification
              → 30 Docs, ledger hygiene, release
```

Phases 27 and 28 may run in parallel after 26. Phase 29 may start fixture-needle work as soon as 25 lands; fuzz and the CI branch fix can start any time and must be green before 30 closes.

| Phase | File | Goal |
|------:|------|------|
| 24 | [phases/phase-24-library-contract.md](phases/phase-24-library-contract.md) | Preferred API, nil/error/panic policy, embedder helpers |
| 25 | [phases/phase-25-pagination-tables.md](phases/phase-25-pagination-tables.md) | Table continuation and page-break model; fixture needles |
| 26 | [phases/phase-26-flow-layout.md](phases/phase-26-flow-layout.md) | Float packing, margin collapse, remaining flex/grid honesty |
| 27 | [phases/phase-27-layout-paint-seam.md](phases/phase-27-layout-paint-seam.md) | Font/paint types leave `layout`; drop unused alias façades |
| 28 | [phases/phase-28-settings-requests.md](phases/phase-28-settings-requests.md) | One network policy, one PDF request, leftover union fields |
| 29 | [phases/phase-29-verification.md](phases/phase-29-verification.md) | Fuzz, golden needles, race/CI alignment |
| 30 | [phases/phase-30-docs-hygiene-closure.md](phases/phase-30-docs-hygiene-closure.md) | Stale comments, ledger claims, VERSION/CHANGELOG, gates |

---

## Executive Summary

| Fact (current evidence) | Location |
|-------------------------|----------|
| Three public configuration styles (dotted `Set`, fluent `PdfGlobalOptions`, typed `PDFRequest`) | `api.go`, `doc.go`, `examples/` |
| Fluent `WithPageSize` / `WithCopies` panic on user input | `api.go` |
| `AddHTML` has no nil-receiver guard; `AddObject` does | `api.go` |
| Duplicate `NetworkPolicy` types | `api.go`, `internal/load/load.go` |
| `convert.Request` still carries image leftovers | `internal/convert/convert.go` |
| `layout.Options` holds `*pdf.Font` / `*pdf.FaceSet` / `*pdf.Registry` | `internal/layout/layout.go` |
| Pagination is a display-list fixpoint (10 iterations) | `internal/layout/paint_pagination.go` |
| Table collapse borders sealed after paint | `internal/layout/paint_pagination.go` `capTablePageBreaks` |
| In-flow tables forced `clear: both` | `internal/layout/layout_flow.go` |
| Certified page-islands path is benchmark-only | `internal/convert/convert.go` `benchmarkPageIslands` |
| Golden corpus is mostly page-envelope + structure | `internal/convert/golden_test.go`, `testdata/golden/README.md` |
| No `Fuzz*` functions | repo-wide |
| CI `push` listens on `main`; CONTRIBUTING uses `master` | `.github/workflows/ci.yml`, `CONTRIBUTING.md` |
| `internal/pdf/doc.go` still describes Phase 00 scaffold | `internal/pdf/doc.go` |
| Direct modules are `go-text/typesetting` + `tdewolff/canvas` | `go.mod` |

---

## Status board

| Phase | Status |
|------:|--------|
| 24 Library contract | [ ] |
| 25 Pagination and tables | [ ] |
| 26 Flow layout | [ ] |
| 27 Layout / paint seam | [ ] |
| 28 Settings and request types | [ ] |
| 29 Verification | [ ] |
| 30 Docs, hygiene, closure | [ ] |

Update a row only after the phase file’s closure gates record `make lint` and `make test`.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| v0.2.0 phases 10–23 (shipped core) | Starting tree for 24–30 |
| Phase 24 | Stable embedder surface for 30 docs |
| Phase 25 | Needles and table invariants for 29 |
| Phase 26 | Flow invariants before 27 moves types |
| Phases 27–28 | Smaller packages for 29 race/`./...` |
| Phase 29 | Evidence for 30 release |

---

## Out of scope

See `documentation/deferred.md`. v0.2.1 does not reopen Phase 22 (JavaScript) or Phase 23 (open-web / Chrome). Stdin HTML (`-`), AcroForm, WOFF2, and PDF encryption stay deferred.

---

## Closure (ledger)

- [ ] Every phase file 24–30 has its closure section filled with `make lint` and `make test` outcomes
- [ ] No duplicate active work: any 0.2.0 row superseded here is `[~]` with a pointer to this ledger
- [ ] `VERSION` / `CHANGELOG.md` / `documentation/deferred.md` agree
- [ ] Handoff: next unchecked work after v0.2.1 (if any) listed in phase 30
