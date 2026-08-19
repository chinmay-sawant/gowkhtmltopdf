# AGENTS.md — gowkhtmltopdf

Guidance for AI coding agents working in this repository.

## Project

**gowkhtmltopdf** is a no-cgo, pure-Go HTML→PDF and HTML→image engine — a
clean-room work-alike of the wkhtmltopdf CLI surface. No browser, no cgo, no
native converter process. Pipeline: load → parse → style → layout → paginate →
paint → write.

- **Status:** v0.2.4 (see `VERSION`). License: MIT.
- **Docs:** `documentation/` (index: `documentation/README.md`).
- **Knowledge base:** `knowledge-base/` — Karpathy-style wiki (raw sources,
  compiled wiki pages, derived outputs). Schema: `knowledge-base/SCHEMA.md`.
- **Plans:** `plans/` — versioned implementation ledgers; `CHANGELOG.md` for
  release history.

## Skills (globally installed)

This repo uses the **pstack** skill set (installed globally in
`~/.agents/skills/`), sourced from
`github.com/cursor/plugins/tree/main/pstack/skills`.

**Start here:** `/poteto-mode` — rigorous engineering mode with playbooks
(bug fix, feature, refactoring, perf, investigation, and more). It routes to
the other skills as needed.

Direct skills:

| Skill | Use when |
|-------|----------|
| `/how` | Walkthrough of how a subsystem works |
| `/why` | Why something was built this way (queries evidence sources) |
| `/architect` | Settle caller usage, types, and module shape before writing code |
| `/arena` | N parallel attempts at the same thing, then merge best parts |
| `/swarm` | N parallel workers across slices, one aggregated report |
| `/interrogate` | Several models try to break a diff |
| `/tdd` | Fixing a bug with a cheap local test path — failing test first |
| `/blast-radius` | Small-looking change; what else could it break |
| `/reflect` | Capture a long task's recipe as a skill edit |
| `/teach` | Actually understand a change, built up diagram by diagram |
| `/no-comments` | Strip comments before review |
| `/unslop` | Remove AI tells from writing |
| `/technical-writing` | Docs, RFCs, readmes, PR descriptions, commit messages |
| `/setup-pstack` | Configure pstack's per-role model choices |
| `/recall` | Rebuild recent context on a topic into a current-state brief |
| `/show-me-your-work` | Keep a reviewable decision trail |
| `/create-verification-skill` / `/maintain-verification-skill` | Project-local verify skills |
| `/figure-it-out` | Design a rigorous playbook when none fits |
| `/automate-me` | Draft your own `-mode` skill from how you've worked |

Plus 21 `principle-*` skills (laziness, foundational-thinking, prove-it-works,
fix-root-causes, model-the-domain, boundary-discipline, type-system-discipline,
guard-the-context-window, never-block-on-the-human, etc.) — `poteto-mode`
indexes them at task start.

## Ground rules

- **No cgo.** Builds target `CGO_ENABLED=0`. Allowlisted modules only:
  `go-text/typesetting` (OpenType shaping) and `tdewolff/canvas` (SVG raster).
  Nothing else may be added to `go.mod`.
- **No JS.** The engine never executes JavaScript. JS-related wkhtmltopdf
  flags are unknown options.
- **No wrappers.** This is an in-process engine, not a binding to the
  wkhtmltopdf binary.
- **Honest degrade.** Unknown CSS is ignored; missing images are skipped; the
  process does not crash.
- **Fidelity language.** `--pdf-version` is a version choice, never a PDF/A or
  PDF/UA claim. Profiles are opt-in only (`--pdf-profile`, `WithPDFProfile`).
  Do not claim conformance beyond the canonical tokens.

## Commands

```sh
make build        # static binaries in bin/
make test         # full test suite
make lint         # golangci-lint (enable-all, see .golangci.yml)
make fmt          # gofmt
make golden       # golden fixture structure checks (GOLDEN_APPROVE=1 to update)
make samples      # regenerate sample PDFs/PNG in output/
make bench        # engine benchmarks (also bench-engine, bench-lib, bench-cli-compare)
make claim-scan   # scans docs for over-claims vs compatibility matrix
make weasyprint   # external-engine comparison (needs weasyprint installed)
```

## Code layout

- Root package (`api.go`, `document.go`, `document_validate.go`) — public
  `Document` / `ImageDocument` API. Never imports `internal/cli`.
- `cmd/` — `gowkhtmltopdf` (PDF) and `gowkhtmltoimage` (image) binaries. Never
  import the root package.
- `internal/convert` — orchestration hub: `RenderObjects` → `Assemble` →
  `Finalize`. `prepare/`, `render/`, `islands/` must never import `convert`.
- `internal/cli` — parses argv into dotted settings; the engine never parses
  argv itself.
- `internal/settings` — wkhtmltopdf-style dotted settings; Policy-A ignored
  keys.
- `internal/load` — trust boundary: ACL, timeouts, body caps. Local files
  denied by default.
- `internal/html`, `internal/css`, `internal/layout` — parse → style → layout;
  `layout.PaintContext` paginates.
- `internal/pdf`, `internal/pdfprofile`, `internal/imageout`, `internal/svg` —
  sinks. PDF 1.4 default; opt-in 1.7 / 2.0; opt-in PDF/A + PDF/UA profiles.

Rules: the import graph is a DAG (nothing points back up); cycles are
forbidden. One job seam: library and CLI adapters build
`convert.Request` / `imageout.Request`.

## Conventions

- Go 1.26+, gofmt clean, golangci-lint enable-all (tenv and gofumpt disabled —
  see `.golangci.yml`).
- Comment style: package docs in `doc.go`; exported identifiers documented.
- Tests: golden fixtures in `testdata/golden/` (structure checks, not
  byte-identical PDFs); layout changes require visual QA (`make samples`) per
  `CONTRIBUTING.md` §Visual QA.
- Resource budgets live in `internal/convert` (objects ≤ 10k, copies ≤ 1k,
  pages ≤ 100k, stylesheet rules ≤ 1M), not in callers.
- Deterministic output: same HTML + settings + fonts produce the same layout;
  PDF bytes are hash-stable only when `Now` is injected.

## Workflow

- Branch + PR flow per `CONTRIBUTING.md`. Update `CHANGELOG.md` and
  `documentation/` with user-facing changes.
- Update `knowledge-base/` when architecture or features change materially
  (follow `knowledge-base/SCHEMA.md`; keep `wiki/index.md` and `wiki/log.md`
  current).
- Verify before declaring done: run `make test && make lint`, and regenerate
  samples for layout changes.