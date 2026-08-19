# Phase 36 - CLI Redesign

> **Parent:** `plans/0.2.4/31-canonical-0.2.4-roadmap.md`
> **Status:** complete — validated 2026-08-18
> **Estimated effort:** 1–2 weeks
> **Depends on:** Phase 34 adapters; preferably Phase 35 library delete complete for examples/docs alignment
> **Unblocks:** Phase 37 CLI docs

---

## Overview

Replace the wkhtmltopdf multi-object argv grammar (`page` / `cover` / `toc`
tokens and dotted flag names as the user-facing contract) with a CLI that
builds a `Document` / `ImageDocument`. No legacy compat mode in 0.2.4.

## Executive Summary

| Today | Target |
|-------|--------|
| `gowkhtmltopdf [opts] page in.html out.pdf` object grammar | `gowkhtmltopdf [opts] -o out.pdf in.html` |
| Dotted / wkhtml long options as primary UX | Go-friendly flags (`--page-size`, `--margin-top`, `--allow-local-files`, …) |
| Cover/TOC as object tokens | `--cover`, `--toc` flags + positional pages |
| Image: wkhtmltoimage-shaped flags | `gowkhtmltoimage -o out.png …` aligned with ImageDocument |

Target examples:

```text
gowkhtmltopdf [global] -o out.pdf page.html
gowkhtmltopdf [global] -o out.pdf --html '<html>…</html>'
gowkhtmltopdf [global] -o out.pdf --url https://example.com/print
gowkhtmltopdf [global] -o book.pdf --cover cover.html --toc page1.html page2.html

gowkhtmltoimage [global] -o out.png page.html
```

---

## Phase 36 checklist

### 36.1 Grammar and flags

- [x] Require `-o` / `--output` (or positional output policy documented and tested — prefer explicit `-o`)
- [x] Positional args are page files (Document.Pages)
- [x] `--html` inline document (mutually exclusive with page files / `--url` per Validate rules)
- [x] `--url` remote page
- [x] `--cover PATH` optional
- [x] `--toc` optional flag inserting TOC
- [x] Global flags allowlist: page size, orientation, margins, title, pdf-version, pdf-profile, outline, allow-local-files, font-path, header/footer text fields as needed
- [x] **No** generic `--set key=value` escape hatch on the new CLI
- [x] **No** Policy A ignored flags on the new primary help

### 36.2 Implementation seam

- [x] Parse → a private Document-shaped `Command` → Phase 34-equivalent internal adapters → engine; the CLI intentionally does not import the root package
- [x] `internal/app` / `internal/cli` refactored so user-facing parse does not require public dotted Set
- [x] Exit codes: preserve load/HTTP exit behavior where still applicable; document changes in cli.md (Phase 37)
- [x] Progress / quiet behavior mapped to Document hooks or internal loggers

### 36.3 Image CLI

- [x] `gowkhtmltoimage` same source flags (`--html`, `--url`, files) + image options (width, height, format, quality, …)
- [x] Output via `-o`

### 36.4 Help and tests

- [x] `--help` rewritten for the new grammar
- [x] Remove primary documentation of wkhtml object tokens
- [x] `cmd/gowkhtmltopdf` + `cmd/gowkhtmltoimage` tests updated
- [x] Smoke: convert a golden HTML fixture via new CLI to `%PDF-`
- [x] Smoke: image CLI to PNG/JPEG magic

### 36.5 Closure gates

- [x] `make lint` → changed-surface lint passes; repository-wide result recorded in Phase 38
- [x] `make test` → passed with GOCACHE=/tmp/gowkhtmltopdf-go-cache
- [x] Parent Phase 36 row checked
- [x] Next: Phase 37

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 34 Document adapters | CLI without old public API |
| Phase 35 (preferred) | Single story for docs |

---

## Out of scope

- Restoring `toc` / `cover` / `page` wkhtml object tokens as the default parser
- `--read-args-from-stdin` batch loop
- Stdin HTML `-` as a hidden GuessURL path (use `--html` or explicit stdin flag if added — document honestly)
- Refreshing external compare result tables (Phase 39 owns `bench-external.sh` / `bench-cli-compare` argv follow-up after this phase)

## Validation record (2026-08-18)

- `internal/cli` and both command binaries now require `-o/--output`, accept positional pages or `--html` / `--url`, support `--cover` / `--toc`, and reject legacy object tokens. Help text and command tests match the grammar.
- CLI PDF and image smoke conversions produced `%PDF-` and PNG magic; `make test` and `make build` passed.
