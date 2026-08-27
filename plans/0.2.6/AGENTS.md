# AGENTS.md - plans/0.2.6 CSS coverage

Scope: this directory. Root `AGENTS.md` still applies. This file only adds rules for executing this ledger.

## This plan

- Canonical ledger: `48-canonical-0.2.6-css-coverage.md`
- Phases: `phases/phase-48-*.md` through `phase-56-*.md`. Do not restart at 01. Sidecar tracks used 01-08 numbering.
- Product: print-oriented structured documents. Not Chrome visual parity. No JavaScript. No CGO on the default path.

Read `48-canonical-0.2.6-css-coverage.md` and `catalog/README.md` before editing CSS.

## Git

No `git add`, `commit`, `push`, `restore`, `clean`, `reset`, or `stash` unless the user asks. Branch names: lowercase `feature/`, `fix/`, `chore/`, or `docs/<short>`.

## Checklists

- `[x]` only after the named proof command exits 0 in the same change. Never from intent.
- `[~]` needs a reason, an owner, a next gate, and a pointer if work moved.
- Do not keep the same active row in `plans/0.2.0/` and here. Move leftovers with `[~]` plus a pointer to this ledger.

## Source of truth for CSS

1. Code in `internal/css` and `internal/layout` is what the engine does.
2. `catalog/mapping.json` is the inventory of names. Engine status in that file must match code.
3. `documentation/compatibility-matrix.md` is the committed contract. It follows code, then the mapping.
4. Knowledge-base follows those three. If they disagree, trust code, fix mapping and matrix, then KB.

Catalog JSON is not loaded at runtime. Do not import it from Go.

When catalog, matrix, and code disagree: trust code.

## How to add a CSS property

Usual landing order. Skip a step only if it already exists.

1. Parser: only if you need a new unit, function, at-rule, or selector. Unknown property *names* already parse (`internal/css/values.go` `validPropName`).
2. `ResolvedStyle` field plus `initialStyle` and intern table (`internal/layout/style.go`).
3. Inheritance row in `inheritableProps` if CSS inherits it (`internal/layout/style_cascade.go`).
4. Apply arm in the right `apply*Group` (`internal/layout/style_properties.go`). Shorthands that must lose to longhands go in `restShorthandProps`.
5. A layout or paint consumer. A field nobody reads is still unsupported.
6. Package test, then golden fixture if paint or pagination changes, then matrix row, then `mapping.json` `engine_status`.

## Claims

No browser-parity, Qt WebKit, stdlib-only, full CSS, or byte-identical PDF claims. `make claim-scan` is the claims gate. Wikipedia and Chrome PDFs are canaries, not pass/fail.

## Gates

- Targeted package tests while working: `go test ./internal/css` and `go test ./internal/layout`.
- Before marking a non-doc phase complete: `make lint` and `make test`. Record both outcomes on the phase file.
- After layout, paint, pagination, or CSS consume changes: `make golden`. New fixtures need a `fixturePageBounds` entry in `internal/convert/golden_test.go`.
- Direct modules stay `go-text/typesetting` and `tdewolff/canvas` unless the user signs off an amendment.

## Prose

No em dashes. Unslop plans and KB. Feynman: plain words, `file:line` citations. Do not write `TODO.md` or `todos.json`.

## Knowledge base

Creating or editing this plan updates KB in the same session: `wiki/log.md`, `wiki/syntheses/roadmap.md`, `wiki/concepts/css-engine.md`, `wiki/compatibility.md`. KB is gitignored. Never hand-edit `docs/`.
