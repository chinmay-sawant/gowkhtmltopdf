# Reviews / Improve codebase architecture — gowkhtmltopdf (2026-08-07)

Architecture **deepening** review (deep modules, seams, locality, idiomatic Go),
run **after** the ponytail leanness audit (closed 9.5/10 on 2026-08-07).

## Canonical ledger

| File | Date | Status | Findings | Rows |
|------|------|--------|---------:|-----:|
| [architecture-review-2026-08-07.md](./architecture-review-2026-08-07.md) | 2026-08-07 | remediation complete (all rows + closure gates) | 49 | 46 |

## Phase-wise checklist (smaller files)

| Phase | File | Rows | Theme |
|-------|------|-----:|-------|
| 1 — Engine seam & surface | [phases/phase-01-engine-seam-and-surface.md](./phases/phase-01-engine-seam-and-surface.md) | 9 | CLI-independent engine `Request`, value contract, settings registry |
| 2 — Document prep & pipeline | [phases/phase-02-document-prep-and-pipeline.md](./phases/phase-02-document-prep-and-pipeline.md) | 14 | stylesheet gatherer, Heading contract, page-index/copies, objectState |
| 3 — CSS engine | [phases/phase-03-css-engine.md](./phases/phase-03-css-engine.md) | 6 | cascade rule-walk, selectors, LengthToPt, var() |
| 4 — Layout engine | [phases/phase-04-layout-engine.md](./phases/phase-04-layout-engine.md) | 7 | img decode-once, box geometry, table sizing, text-wrap |
| 5 — Output, fonts, raster | [phases/phase-05-output-fonts-raster.md](./phases/phase-05-output-fonts-raster.md) | 7 | paint semantics, PDF page-assembly, font embed, grayscale seam |
| 6 — Cross-cutting & closure | [phases/phase-06-cross-cutting-and-closure.md](./phases/phase-06-cross-cutting-and-closure.md) | 3+closure | log protocol, examples, exit-code dispatch, validation gates |

## Skills / method used

- Explore agents: 7 (one per package area, byte-weighted; `codebase-design` vocabulary:
  module depth, seam, locality, leverage).
- Plan shape: `skills/phase-wise-checklist/SKILLS.md` conventions (canonical ledger,
  `[ ]`/`[x]`/`[~]` statuses) — the skill file itself was left untouched.
- Relation to ponytail: this folder = deepening; [`../ponytail/`](../ponytail/) =
  leanness (delete).
