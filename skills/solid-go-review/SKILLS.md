---
name: solid-go-review
description: Parallel SOLID plus Go design pattern review with package-scoped subagents, evidence-backed rating out of 10, and concrete fixes. Read-only, no git commands.
---

# Solid go review

Use this skill when asked if the app follows SOLID and Go design patterns, how to improve it, and what score it gets out of 10.

## Constraints

- No git commands in this workflow. Do not run git add, commit, push, restore, clean, reset, or stash.
- Read-only scan. Do not edit source files while reviewing.
- One subagent owns one package area. Never assign two agents to the same package in one wave.
- Do not run lint from two agents on the same tree at the same time.
- Every claim needs a file:line cite. Start from knowledge-base/wiki/index.md, then prove against code under internal/ and the repo root. If the knowledge base and code disagree, trust the code.
- Findings must use `skills/phase-wise-checklist/SKILLS.md` for structure and `/unslop` for prose. No findings reply without both.
- Keep prose plain. No em dashes. Use hyphens or short sentences.

## Wave layout

Launch four subagents in parallel, each read-only and scoped:

1. Root plus convert
   - Root *.go public API: Document, ImageDocument, validation, sentinels.
   - internal/convert plus render plus prepare plus settings stamps.
   - cmd/gowkhtmltopdf plus cmd/gowkhtmltoimage thin shells.
   - go.mod direct dependency check.
2. Layout
   - internal/layout only: layout, paint, pagination, tables, images, measure, flex, grid, inline, style.
   - Watch the soft file limit near 2,000 lines and fat engine or box structs.
3. Pdf plus image output
   - internal/pdf version-aware writer.
   - internal/imageout shared pipeline rasterizer.
4. Remaining internal
   - internal/css, internal/html, internal/load, internal/settings, internal/cli, internal/app, internal/outline, internal/svg, internal/pdfprofile, plus other small packages.
   - Do not enter convert, layout, pdf, or imageout except for one seam reference such as a golden test helper.

Each subagent prompt must repeat: no git commands, no edits, no lint, stay in scope, return file:line cites.

## Per area checks

For each area, report:

- Files in scope with one-line responsibility each.
- SOLID verdicts with evidence:
  - SRP: one reason to change per file or struct, flag god structs and god functions.
  - OCP: new behavior without editing core, flag switches that need edits per new case.
  - LSP: substitution holds, note where there is no hierarchy so the verdict is vacuous.
  - ISP: small interfaces, flag fat concretes passed across packages.
  - DIP: depend on narrow seams, flag concrete news inside high-level functions.
- Go patterns:
  - Small constructors with 4 or fewer params, functional options where present, composition over fat structs, error sentinels with %w wrap, boundary validation, resource bounds and pools, registry constructors, settings stamps.
  - Note direct dependency count from go.mod.
- Top 3 violations with file:line plus fix direction only, no code change.
- Sub-score out of 10 for that area.

## Findings ledger

Generate all findings as a phase-wise checklist using `skills/phase-wise-checklist/SKILLS.md`. One atomic row per violation or fix, each with affected path, SOLID letter, expected behavior, and required proof such as file:line, test, or grep output. Use `[ ]` for open, `[x]` only after current evidence passes, `[~]` for deferred with reason and owner. Order by risk: correctness and coupling first, then API seams, then cleanup. Never mark `[x]` from intent.

Apply `/unslop` to every prose surface in the findings and final reply: cut puffery, filler, and chatbot phrases, keep plain human voice, keep hyphens and commas only for breaks. Self-audit with the question "What makes this obviously AI generated" and fix remaining tells before sending.

## Synthesis

After the wave returns:

- Spot-check load-bearing claims with read or grep before scoring.
- Merge into one SOLID table with PASS, FAIL, or mixed per letter.
- List what is good with file:line proof.
- List what is bad ordered by coupling payoff.
- Give one overall score out of 10 as a mean of sub-scores adjusted for cross-package coupling.
- Give 3 to 5 fixes ordered by value, each naming target files and the seam to add or split.
- Keep the final reply short and concrete.
