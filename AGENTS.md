# AGENTS.md - gowkhtmltopdf

> This file is the conventions ledger for every coding agent working in this
> repo (opencode, grok, gemini, codex, antigravity/agy, claude). Read it at
> session start. It encodes lessons from a 490-session audit of our sibling
> projects plus everything learned shipping gowkhtmltopdf itself through
> v0.2.x: the mistakes, repeat work, and avoidable waste we already paid for,
> so we do not repeat them here.

## Project

Pure-Go HTML-to-PDF engine and wkhtmltopdf-style work-alike, plus an
HTML-to-image rasterizer. Two static binaries (`cmd/gowkhtmltopdf`,
`cmd/gowkhtmltoimage`) and a Go library (`Document` / `ImageDocument` at
the repo root). The product is print-oriented structured documents
(invoices, receipts, certificates), not Chrome visual parity. No JavaScript.
No CGO.

Module: `github.com/chinmay-sawant/gowkhtmltopdf`.
GitHub repo: `https://github.com/chinmay-sawant/gowkhtmltopdf`.
Default branch: `master`. Current version: `VERSION` (single line, injected
into both binaries via ldflags).

Pipeline order, owned by `internal/`: load -> html parse -> css cascade ->
layout -> paginate -> paint -> pdf write. `internal/convert/` orchestrates;
`internal/pdf/` is the version-aware writer; `internal/imageout/` shares the
pipeline for PNG/JPEG.

## Todo protocol - response-only (mandatory)

Every agent must manage todos live in the API response, not on disk. This is
the first action on any task, before any reads, edits, or other tool calls.

1. **Create todos in the response via API before any work.** On receiving any
   task (feature, fix, docs, question with multi-step work), immediately call
   the todo API (`todowrite` or equivalent) to publish the plan as todos. Do
   not start work until the todo list is visible in the API response.
2. **Show current todos in every response.** Each assistant turn must render
   the current todo list with status markers (`pending`, `in_progress`,
   `completed`, `cancelled`) and clearly highlight which item is
   `in_progress`. The todo list is the live progress bar for the user.
3. **Do not store todos on disk.** Do not create `TODO.md`, `todos.json`, or
   any other todo-tracking file. Todos live only in the API response state.
4. **Keep response todos updated as you go.** Mark items `in_progress` when
   started and `completed` when finished. If scope changes, update the list
   immediately.
5. **Completion requires todos to show done.** A task is done only when all
   todos show `completed` and you have sent a final summary stating what
   shipped.

If the todo API is unavailable, state that in the response and list todos
inline as a fallback - still do not write a file.

## Golden rules

1. **No git commands without explicit permission.** Never run `git add`,
   `git commit`, `git push`, `git restore`, `git clean`, `git reset`, or
   `git stash` unless the user asks. Subagent prompts carry this ban by
   default.
2. **No em dashes ("—") in any written output, docs, or commit messages.**
   Use plain hyphens or restructure. This includes this file.
3. **Commit at session end; never leave a dirty tree.** Uncommitted work is
   the top cause of cross-session rework. If interrupted, record what landed.
   Ask before committing if the user has not authorized it.
4. **Branch naming:** lowercase `feature/`, `fix/`, `chore/`, or
   `docs/<short-description>`. Verify the branch name before the first commit
   (a typo'd `chore/frontend-udpates` once lived across 3 commits and a PR).
5. **PRs** use `skills/PR/PR_TEMPLATE.md`; issues use
   `skills/PR/ISSUE_TEMPLATE.md`. Body file lives at
   `plans/PR/pr-<short-slug>.md` and must stay in sync with the GitHub PR.
   PRs require self-assignee and at least one label.
6. **Checklists are live ledgers:** phase files under `plans/<version>/`
    close rows `[x]` in the same change that implements them, and only when
    the gate actually passed. Never mark `[x]` from intent. When you create
    or finish a plan under `plans/`, update `knowledge-base/` in the same
    session so the local narrative matches what shipped.
7. **Answer from knowledge-base, prove with code.** Start in
   `knowledge-base/wiki/index.md`, then open the real source under
   `internal/`. A KB hit is a starting point, not proof. If KB and code
   disagree, trust the code, fix the KB, answer from the code. For prose
   claims (performance numbers, fidelity statements), the committed reference
   is `documentation/*.md`, and `make claim-scan` polices forbidden claims
   there.
8. **Unslop all writing.** Before writing plans, docs, knowledge-base pages,
   PR/issue text, commit messages, or user-facing replies, apply `/unslop`.
   Cut puffery, em dashes, chatbot filler, synonym cycling, title-case
   headings.
9. **Feynman every explanation and document.** Whenever you respond with an
   explanation or produce documentation (plans, `documentation/`,
   knowledge-base pages, README sections, PR bodies, design notes), apply
   `skills/feynman/SKILLS.md`: plain words a smart 12-year-old could retell,
   grounded in real source (`file:line` citations), self-audited for jargon,
   hand-waves, circularity, and name-dropping until zero findings remain.
   Report the pass count when the loop runs as `/feynman`.

## Unslop (`/unslop`)

Apply to every prose surface: chat replies, `plans/`, `documentation/`,
`knowledge-base/`, README, CHANGELOG entries, PR and issue bodies, commit
messages. Load the unslop skill, rewrite to plain human voice, then self-
audit: "What makes this obviously AI generated?" Prefer concrete facts and
file paths over vague summary language.

## Feynman (`/feynman`)

Apply to every response that explains something and to every documentation
surface. Load `skills/feynman/SKILLS.md` and run its loop: explain in plain
words, audit your own explanation for jargon / hand-waves / circularity /
name-dropping, fill each gap from real source with `file:line` citations,
repeat until zero findings. Never explain from memory alone; read the code
or doc first. Report the pass count when invoked as `/feynman`.

## Knowledge base (local only)

`knowledge-base/` is gitignored on purpose. Keep it current even though git
does not track it. It has three layers (`knowledge-base/README.md`,
`SCHEMA.md`):

- `raw/` - immutable captures (api-contract, core-specs, security-posture)
- `wiki/` - compiled articles. Entry point `wiki/index.md`; append-only op
  log `wiki/log.md`; concept pages under `wiki/concepts/`; doc summaries
  under `wiki/summaries/`; cross-cutting syntheses under `wiki/syntheses/`
- `outputs/` - one derived report per query

### When to update it

- Creating or editing a plan under `plans/`
- Shipping a behavior change: update the matching wiki pages in the same
  session as the code change
- Any time code and KB disagree while answering a question: fix the KB

`documentation/` remains the committed reference. Note the trap: `docs/` is
NOT the markdown reference - it is the built static website generated from
`frontend/` (Vite React app). Never hand-edit `docs/`.

## Verification gates

The real gates, in order of cost:

| Gate | Command | What it proves |
|------|---------|----------------|
| Unit + integration | `make test` | Full suite green (`-p 2 -parallel 2` by default; see Makefile) |
| Claims | `make claim-scan` | No forbidden claims (stdlib-only, Qt WebKit, byte-identical determinism, etc.) in doc.go, README, documentation/, frontend content, cli help |
| Lint | `make lint` | golangci-lint (pinned v1.64.8) clean; chains `lint-frontend` (npm) |
| Golden corpus | `make golden` | All 61 fixtures convert with correct structure, page-count envelopes, embedded fonts, ordered text needles |

Run targeted package tests during a session; run the full gate set once at
session end before claiming done. CI additionally runs `-race` on hot
packages (`convert`, `layout`, `pdf`, `imageout`, `load`), a CGO_ENABLED=0
static build with version-stamp assertion, and a frontend production build
that fails if `docs/` goes dirty.

## Things to AVOID (paid-for lessons)

1. **Lint whack-a-mole.** Run `make lint` before opening any PR; fix all
   linter categories in one pass. `//nolint` is a last resort with a written
   reason. Never auto-rename or bulk-fix; each mechanical rename is followed
   by `go build ./...` + targeted tests.
2. **Verifying against stale artifacts.** Rebuild before verifying CLI
   behavior: `make build` produces `bin/gowkhtmltopdf` and
   `bin/gowkhtmltoimage`. Committed `output/*.pdf` files are regenerated
   samples, not behavior baselines ("not golden byte baselines" per
   `output/README.md`). Regenerate with `make samples` when needed.
3. **Claiming completion without the gate output.** Read the final exit code.
   A task is done when the last validation exits 0, not when you expect it
   to.
4. **Fixes that regress other fixtures.** The golden corpus exists for this:
   after any layout/paint/pagination change, run `make golden`, not just the
   package tests.
5. **Guessing APIs and paths.** Grep the symbol, glob the file, read the
   package doc comment before writing against it. `go build` before writing
   tests.
6. **Silent coverage gaps.** Every new golden fixture needs a page-count
   envelope entry in `fixturePageBounds` (`internal/convert/golden_test.go`)
   plus feature flags where relevant; a missing key hard-fails by design.
   Fixture naming needs DOCTYPE + naming comment (`fixtureHeaderOK`).
7. **Scope creep.** Edit only the files in the task.
8. **Whole-file re-reading.** View targeted ranges/diffs. Line-by-line claims
   require actual coverage.
9. **Duplicate artifacts.** Update in place: one canonical ledger per version
   under `plans/<version>/`, one body per PR under `plans/PR/`.
10. **Repeated full-suite verification after every micro-fix.** Targeted
    tests after edits; full gate once at session end. Cached results count.
11. **Throwaway tooling.** Anything used twice belongs in `scripts/` with a
    note. Bench harnesses live in `scripts/bench-external.sh`, screenshot
    tooling in `scripts/screenshot_showcase.py`; do not re-inline them.
12. **User-in-the-loop aesthetic loops.** Verify rendering yourself first:
    `make samples` regenerates `output/` PDFs/PNGs you can inspect before
    asking the user to look.
13. **Dead subagents.** Confirm spawned agents produced turns; diagnose
    before re-spawning identically; check cancelled agents for landed work
    before redoing it.
14. **Parallel agents on one shared tree.** One agent owns one package
    (e.g. one on `internal/layout`, another on `internal/pdf`, never both on
    `internal/convert`). No two agents run lint on the same tree.

## Engine specifics

- **Golden tests are a structural contract, not pixel diffs**: `%PDF-`
  header, `/FontFile2` subset presence, xref/EOF integrity, per-fixture page
  envelope, feature flags (`images`, `uris`), and ordered text needles via
  `pdf.ParseSemantic`. See `testdata/golden/README.md`.
- **Regeneration is guarded.** `make golden-update GOLDEN_FIXTURE=<name>
  GOLDEN_APPROVE=1` writes only `testdata/golden/out/` and never touches
  committed fixtures. Treat an approved golden output like a reviewed
  artifact.
- **Compliance validators live outside the Makefile.**
  `compliance/verify_pdfs.sh` (veraPDF parse+flavour checks, structure-tree
  check, optional avalpdf) is invoked directly; some targets named in
  `compliance/README.md` do not exist yet. When two validators disagree,
  both are suspect until explained.
- **Dependency allowlist is mechanically enforced.** Direct third-party
  modules may only be `github.com/go-text/typesetting` (OpenType shaping) and
  `github.com/tdewolff/canvas` (SVG rasterization), checked by
  `TestDirectModuleAllowlist`. Everything else stays `// indirect`.
- **Version discipline.** `VERSION` is injected via ldflags; the release
  workflow hard-fails if `VERSION` does not match the pushed tag. Version
  bumps change VERSION + CHANGELOG together and pass `make test` first.
- **Frontend is part of the surface.** `frontend/` builds the product site
  into `docs/`; `make lint` chains into its ESLint; CI fails on a dirty
  `docs/`. Run frontend checks when touching `frontend/src/data/content/`
  because `claim-scan` reads it too.
- **CLIs are safe headless** (no TTY requirement). What bites instead is
  environment drift: pin expectations via the Makefile rather than ad-hoc
  invocations.

## Code structure

- **File size soft limit: ~2,000 lines.** Two legacy files exceed it today:
  `internal/layout/paint_flow.go` (~2.4k) and
  `internal/layout/paint_pagination.go` (~2.2k). Do not grow them further;
  extract a cohesive piece into a same-package file whenever you touch them.
  No new file crosses the limit without a written reason.
- **Split module-wise, not length-wise.** Divide by responsibility (each
  stage, view, store, or profile gets a focused file), never "cut here"
  chunks. Follow existing seams: `internal/layout` already splits paint /
  pagination / tables / images / measure.
- **Use abstractions at real seams** (settings stamps, prepare options,
  registry constructors), keep constructors small, prefer composition over
  fat structs.
- Verify with `go build ./...` + targeted tests after any split; do not
  reorder or rename code during a split.

## Plans and ledgers

`plans/` is version-partitioned (`plans/0.1.0/` ... `plans/0.2.4/`), indexed
by `plans/README.md`. Each version dir holds a numbered canonical ledger plus
per-phase checklists; audits land under `<version>/improve-codebase/<date>/`;
PR bodies live in `plans/PR/`. Phase checklist format comes from
`skills/phase-wise-checklist/`. Rows close on proof only (golden rule 6).

## Skills (this folder)

- `skills/PR/` - templates for PRs, issues, review comments
- `skills/feynman/` - plain-words explanation loop with self-audit;
  mandatory default for every explanatory reply and documentation surface
  (golden rule 9)
- `skills/phase-wise-checklist/` - evidence-backed plan ledgers under
  `plans/`
- `skills/improve-codebase/` - architecture/seams/go-practices audit pack
  producing a phase-wise ledger
- `skills/critical-go-review/`, `skills/perf-review/` - multi-agent review
  waves (read-only reviewers, then fix agents, then one lint+test gate)
- `skills/release-note/` - cut a release: VERSION + CHANGELOG + notes + stamp
- `skills/debug-html-template/` - diagnose a wrapping/misaligned template,
  propose fixes, wait for the pick
- `skills/diagnose-golden-fixture/` - golden corpus failure loop: tight red
  test, bisect, falsifiable probes, fix the interaction without undoing
  intentional prior work
- `skills/golang-anti-patterns/` - top 50 Go anti-patterns catalog, detection
  heuristics, and idiomatic Go pattern replacements
- `skills/ponytail*` - laziness protocol family (YAGNI reviews, debt ledger)

## Dependency policy

Exactly two direct dependencies are allowed (`go-text/typesetting` for text
shaping, `tdewolff/canvas` for SVG rasterization), and the allowlist is
enforced by a test. Any dependency addition is a project-policy change:
announce its purpose, amend the Makefile allowlist, update the affected
plan, and get explicit user sign-off. No silent additions.

---

## FAQ - does the root AGENTS.md get read automatically?

Yes. Creating `AGENTS.md` at the repo root is the standard, and it is read
automatically by every major tool: opencode, codex, gemini CLI,
antigravity/agy, grok, claude code, cursor. You do not need anything under
`.agents/`. Keep this one file at the root and keep it current.
