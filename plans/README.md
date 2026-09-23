# Plans

Implementation plans and roadmaps for `gowkhtmltopdf`, partitioned by release version:

| Version Directory | Scope | Status |
|-------------------|-------|--------|
| [0.1.0/](0.1.0/README.md) | **v0.1.0 MVP Release** — pure-Go rewrite foundation, phases 00–09, exploration studies, and MVP PRs | Complete (Released 2026-08-03) |
| [0.2.0/](0.2.0/README.md) | **v0.2.0 Post-MVP, Performance & Reviews** — phases 10–23, performance optimizations, architecture audits, and frontend improvements | Shipped core; leftovers stay in 0.2.0 or move with a `[~]` pointer |
| [0.2.1/](0.2.1/README.md) | **v0.2.1 Contracts, print layout, and verification** — phases 24–30 | Complete (Released 2026-08-14) |
| [0.2.2/](0.2.2/README.md) | **v0.2.2 Newer PDF versions** — PDF 1.7 (#31), PDF 2.0 (#32), 1.7/2.0 compliance (#33); criticality/optimization follow-up | Complete; [release notes](0.2.2/PR/release-v0.2.2.md) |
| [0.2.3/](0.2.3/README.md) | **v0.2.3** — same engine as 0.2.2; GitHub module path / `go install` | [release notes](0.2.3/PR/release-v0.2.3.md) |
| [0.2.4/](0.2.4/README.md) | **v0.2.4** — idiomatic Document API + CLI rethink + external benches (phases 31–39) | Complete; [release notes](0.2.4/PR/release-v0.2.4.md) |
| [0.2.5/](0.2.5/README.md) | **v0.2.5 Python cgo c-shared bindings and PyPI** — phases 40–47 (in-process, `CGO_ENABLED=0` pure-Go default kept); font track `font/` already complete | Complete (released 2026-08-26; `VERSION` 0.2.5) |
| [0.2.6/](0.2.6/README.md) | **v0.2.6 CSS coverage and browser WASM** - catalog-driven print CSS (354 implemented / 0 partial / 464 unsupported), browser WASM output, warm-path recovery | Complete (released 2026-09-13; `VERSION` 0.2.6). [Release notes](0.2.6/PR/release-v0.2.6.md) |
| [0.2.7/](0.2.7/README.md) | **v0.2.7 next-72 + next-100 CSS** - 72-property wave (fixture-64) closed. Next-100 is planned. The phase 95 visual pilot is in progress. | Next-72 is complete. Next-100 is planned. The fixture 01 pixel pilot passes (`0.2.7/validator-test/95-canonical-pixel-regression-tests.md:57-82`, `internal/convert/fixturetests/pixel_regression_cases_test.go:103-128`). Phase 94 stays active until full migration. |

---

## 0.1.0 (MVP Release)

- [0.1.0 README](0.1.0/README.md)
- [00-canonical-pure-go-rewrite.md](0.1.0/00-canonical-pure-go-rewrite.md) — Canonical execution ledger for MVP phases 00–09
- [phases/](0.1.0/phases) — Atomic checklists for MVP phases 00–09
- [exploration/](0.1.0/exploration) — Pipeline, loader, and pure-Go feasibility research
- [PR/](0.1.0/PR) — MVP PR and initial issue definitions

## 0.2.0 (Post-MVP, Performance & Reviews)

- [0.2.0 README](0.2.0/README.md)
- [10-canonical-post-mvp-roadmap.md](0.2.0/10-canonical-post-mvp-roadmap.md) — Canonical execution ledger for post-MVP phases 10–23
- [phases/](0.2.0/phases) — Phase checklists 10–23, Tier-2 subplans, and pending items
- [performance/](0.2.0/performance) — Allocation profiles, 500-page target, RSS reduction, and architecture benchmarks
- [reviews/](0.2.0/reviews) — Ponytail leanness audits and architectural improvement reviews
- [frontend-improves/](0.2.0/frontend-improves) — Docs-site UI/UX ledgers
- [amendments/](0.2.0/amendments) — Shaping and library amendments
- [deferred/](0.2.0/deferred) — Deferred roadmap targets
- [PR/](0.2.0/PR) — Post-MVP PR archives and issue dossiers

## 0.2.1 (Contracts, print layout, and verification)

- [0.2.1 README](0.2.1/README.md)
- [24-canonical-0.2.1-roadmap.md](0.2.1/24-canonical-0.2.1-roadmap.md) — Canonical execution ledger for phases 24–30
- [phases/](0.2.1/phases) — Atomic checklists for phases 24–30

## 0.2.2 (Newer PDF versions)

- [0.2.2 README](0.2.2/README.md)
- [pdf-1.7-plan/](0.2.2/pdf-1.7-plan/) — Issue #31 PDF 1.7 version support
- [pdf-1.7-compliance-plan/](0.2.2/pdf-1.7-compliance-plan/) — Highest 1.7 conformance: PDF/A-3a + PDF/UA-1
- [pdf-2.0-plan/](0.2.2/pdf-2.0-plan/) — Issue #32 PDF 2.0 version support
- [criticality-optimization-checklist.md](0.2.2/criticality-optimization-checklist.md) — Post-#45/#46 criticality and optimization follow-up

## 0.2.3

- [0.2.3 README](0.2.3/README.md)
- [PR/release-v0.2.3.md](0.2.3/PR/release-v0.2.3.md) — GitHub release body

## 0.2.4 (Idiomatic Document API + CLI rethink + external benches)

- [0.2.4 README](0.2.4/README.md)
- [31-canonical-0.2.4-roadmap.md](0.2.4/31-canonical-0.2.4-roadmap.md) — Canonical execution ledger for phases 31–39
- [phases/](0.2.4/phases) — Atomic checklists for phases 31–39 (Phase 39: wk / WeasyPrint / Puppeteer compare paths)
- [PR/release-v0.2.4.md](0.2.4/PR/release-v0.2.4.md) — GitHub release body

## 0.2.5 (Python cgo c-shared bindings and PyPI + Font tracks)

- [0.2.5 README](0.2.5/README.md)
- [40-canonical-0.2.5-python-bindings.md](0.2.5/40-canonical-0.2.5-python-bindings.md) — Canonical execution ledger for python cgo bindings and PyPI (phases 40–47), issue #35
- [phases/](0.2.5/phases) — Per-phase atomic checklists 40–47
- `font/` — Already complete font resolution track (phases 01–08, cited in syntheses/roadmap); `VERSION` `0.2.5`

## 0.2.6 (CSS coverage)

- [0.2.6 README](0.2.6/README.md)
- [PR/release-v0.2.6.md](0.2.6/PR/release-v0.2.6.md) - GitHub Release body for v0.2.6
- [48-canonical-0.2.6-css-coverage.md](0.2.6/48-canonical-0.2.6-css-coverage.md) - Canonical execution ledger (phases 48-78)
- [phases/](0.2.6/phases) - Per-phase atomic checklists 48-78
- [catalog/](0.2.6/catalog) - Frozen webref / W3C / MDN JSON plus `mapping.json`
- [property-counts.md](0.2.6/property-counts.md) - Implemented / Partial / Unsupported / Ignored counts
- [ignored-inventory.json](0.2.6/ignored-inventory.json) - Ownership map for the 247 Ignored names (phases 68-78)
- [AGENTS.md](0.2.6/AGENTS.md) - Agent rules for this ledger

## 0.2.7 (Next 72 CSS, then next 100)

- [0.2.7 README](0.2.7/README.md)
- [87-canonical-0.2.7-next-72.md](0.2.7/87-canonical-0.2.7-next-72.md) - Canonical execution ledger (batches 87.1-87.8)
- [88-canonical-0.2.7-next-100.md](0.2.7/88-canonical-0.2.7-next-100.md) - Canonical execution ledger (batches 88.1-88.9)
- [phases/](0.2.7/phases) - Per-batch checklists; **87.8 and 88.9 own `make test` / `make golden`**
- [next-72-properties.json](0.2.7/next-72-properties.json) - 72-property inventory + Chrome BCD
- [next-100-properties.json](0.2.7/next-100-properties.json) - 100-property inventory + Chrome BCD, zero overlap with the 72
- [next-72-fixture-authoring.md](0.2.7/next-72-fixture-authoring.md) - fixture-64 authoring contract
- [next-100-fixture-authoring.md](0.2.7/next-100-fixture-authoring.md) - fixture-66 authoring contract
- [chrome-flex-interactions/](0.2.7/chrome-flex-interactions) - Chromium-backed Flexbox interaction inventory and Go conversion plan
- [validator-test/](0.2.7/validator-test/95-canonical-pixel-regression-tests.md) - Pixel comparison pilot passes for fixture 01 (`internal/convert/fixturetests/pixel_regression_cases_test.go:103-128`). Approval of the other 64 references and CI remain open. Phase 94 stays active until full migration.
- [86-canonical-0.2.6-wasm.md](0.2.6/86-canonical-0.2.6-wasm.md) - Browser WASM conversion, PDF preview, and image output phases 86-93
- [perf-improve/wave-2-50pct/](0.2.6/perf-improve/wave-2-50pct/) - Live ledger to cut remaining Snapshot M ns/op and B/op in half (library and CLI). Wave 1 Snapshot L checklist stays closed.

## Performance (cross-version)

- [performance/2026-09-21/](performance/2026-09-21/00-canonical-chrome-case-alloc-reduction.md) - Chrome 40-case allocation reductions: retained serial flate state (713.50 -> 485.98 MB total) and one shared cascade (485.98 -> 327.88 MB total); chunk-tail phase deferred by measurement (about 1% waste vs a 5% gate)

Phases 57-67 closed Partials (174 Implemented / 0 Partial). Phases 68-78 reopen all 247 Ignored for browser-level print. Leftover CSS rows in `0.2.0/phases/pending-phase-items/` move here with `[~]` pointers. WOFF2 sidecar cited in KB is not in this worktree unless amended.
