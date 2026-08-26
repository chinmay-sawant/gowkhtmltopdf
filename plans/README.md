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
| [0.2.5/](0.2.5/README.md) | **v0.2.5 Python cgo c-shared bindings and PyPI** — phases 40–47 (in-process, `CGO_ENABLED=0` pure-Go default kept); font track `font/` already complete | Complete (implementation validated 2026-08-26; `VERSION` still 0.2.4 until tag) |

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
- `font/` — Already complete font resolution track (phases 01–08, cited in syntheses/roadmap); `VERSION` still `0.2.4` until tag
