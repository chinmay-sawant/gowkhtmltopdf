# Reviews / Improve codebase architecture — gowkhtmltopdf (2026-08-07)

> **Parent:** `plans/reviews/improve-codebase/README.md` - architecture deepening reviews
> **Status:** **remediation complete** (2026-08-07) — 46 checklist rows closed; see `phases/` and `fix-log/`
> **Scope:** whole Go tree (~714k non-test bytes), 7 non-overlapping areas
> **Method:** 7 explore subagents (surface/API, settings+CLI, HTML+load+outline, CSS, layout, convert, PDF/fonts/raster) — `improve-codebase-architecture` deepening, `codebase-design` vocabulary (module depth, seams, locality, leverage); ran after ponytail leanness closed (9.5/10)
> **Plan shape:** phase-wise checklist; report split into per-phase files

---

## Overview

This ledger is the canonical entry point for the **architecture deepening** review of
gowkhtmltopdf, kept separate from the ponytail leanness audit (delete/YAGNI). The
ponytail pass already closed at **9.5/10** (2026-08-07) and its `// ponytail:` ceilings
were respected: every row below deepens, none proposes deleting dead flexibility unless
it blocks architecture work.

The `plans/reviews/improve-codebase/` README reserved this review in advance; execution
was delegated to 7 explore subagents, one per package area (byte-weighted), each
instructed to quote verbatim current Go and propose idiomatic future Go, with no style
nits and no re-litigating ponytail rows.

## Executive Summary

- **49 findings** → **46 checklist rows** after fan-in dedupe (3 merges: engine seam
  pair, stylesheet-gatherer pair, paint-semantics pair).
- **Severity:** 10 high · 28 medium · 8 low rows (see counts below).
- **Top themes, in dependency order:**
  1. **Engine seam is absent** — `internal/convert`, `internal/imageout`, `api.go` and
     tests all depend on the CLI parser struct (`cli.Command`); a neutral `Request`
     type decouples argv parsing from the engine (P1-1).
  2. **Stylesheet gathering is duplicated** — DOM→sheet knowledge lives in two
     near-copies that have already drifted (plus a document-prep prologue copy);
     one gatherer module (P2-01).
  3. **Paint semantics are re-implemented per adapter** — body, header/footer and the
     image rasterizer each hand-roll the `layout.Op` table and have already diverged
     (fake-bold CJK gate, alpha, stroke) (P5-01).
  4. **The deepest module (layout, 318 KB) amortizes geometry math ~10×** and
     fetches/decodes each `<img>` 2–4× per run (P4-01/P4-02).
  5. **PDF and image mode maintain two page-assembly forks** instead of one shared
     pipeline (P5-02, depends on P2-01).
- **Not architecture findings, intentionally excluded:** dead-code leanness (ponytail
   handled), naming/formatting, single-line style changes.
- **Hypotheses labelled in-row** — e.g. area-3 aF7 (bytes→runes), area-4 F4
   (length→pt), area-3 F4 (loader policy) — each says what to validate and how.

## Phase map (report split into smaller files)

| Phase | File | Rows | Theme | Depends on |
|-------|------|-----:|-------|------------|
| 1 | [`phases/phase-01-engine-seam-and-surface.md`](phases/phase-01-engine-seam-and-surface.md) | 9 | CLI-independent engine `Request`, value contract, settings registry, root surface | — |
| 2 | [`phases/phase-02-document-prep-and-pipeline.md`](phases/phase-02-document-prep-and-pipeline.md) | 14 | stylesheet gatherer, Heading contract, page-index/copies model, loader policy, objectState | P1 |
| 3 | [`phases/phase-03-css-engine.md`](phases/phase-03-css-engine.md) | 6 | cascade rule-walk, selector parse seam, LengthToPt, var() | P1, P2 |
| 4 | [`phases/phase-04-layout-engine.md`](phases/phase-04-layout-engine.md) | 7 | img decode-once, box geometry, table sizing, text-wrap | P2, P3 |
| 5 | [`phases/phase-05-output-fonts-raster.md`](phases/phase-05-output-fonts-raster.md) | 7 | shared paint semantics, PDF object refs, font embed pipelines | P2, P4 |
| 6 | [`phases/phase-06-cross-cutting-and-closure.md`](phases/phase-06-cross-cutting-and-closure.md) | 3+closure | log protocol, examples, exit-code dispatch, validation gates | all |

## Rows by phase and severity

| Phase | High | Medium | Low | Total |
|-------|-----:|-------:|----:|------:|
| P1 | 3 | 5 | 1 | 9 |
| P2 | 3 | 9 | 2 | 14 |
| P3 | 1 | 3 | 2 | 6 |
| P4 | 1 | 4 | 2 | 7 |
| P5 | 2 | 5 | 0 | 7 |
| P6 | 0 | 2 | 1 | 3 |

## Execution order (why phases are ordered this way)

1. **P1 first** — every other phase crosses `cli.Command` / the flag-value contract /
   the settings registry; the engine `Request` seam shrinks all later diffs.
2. **P2 next** — document prep (stylesheets, Heading, page-index) feeds layout (P4)
   and the page-assembly fork (P5); several P2 rows unblock P4/P5 rows.
3. **P3 CSS before P4 layout** — layout consumes the cascade/style surface in P3.
4. **P4 layout before P5** — the layout op display list is what P5 paints.
5. **P6 last** — logs/examples/exit codes touch everything; closure gates validate the
   whole tree after all rows ship.

## Closure gates

- `make lint` and `make test` pass with no new failures after each phase.
- Golden fixture renders unchanged **where the change is behaviour-neutral**; where a
  snippet deliberately changes behaviour (e.g. P5-01 fake-bold CJK, P5-07 invisible
  runes), the golden diff must be reviewed and documented in the row.
- `go test -run 'Golden|Fixture' ./internal/...` for the affected package.
- `go vet ./...` clean.
- Benchmarks: for performance-labelled rows (P4-01 img decode, P4-07 op splice), record
  the command + dataset + before/after metric in the checklist row itself.
- Debt ledger: any new deliberate shortcut gets a `// ponytail:` marker naming a ceiling
  and upgrade trigger (repo convention).

## Evidence

Raw agent findings (verbatim) were archived **off-repo** on 2026-08-07 (safe external
location); the per-finding Before/After snippets live **inline** in each phase file, so
this report is self-contained:

| Area | Findings |
|------|---------:|
| Surface API / root / examples | 7 |
| Settings + CLI | 7 |
| HTML + Load + Outline | 7 |
| CSS engine | 7 |
| Layout engine | 7 |
| Convert pipeline | 7 |
| PDF + fonts + raster | 7 |
