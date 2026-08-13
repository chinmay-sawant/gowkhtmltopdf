---
name: improve-codebase
description: >
  Run the improve-codebase pack (architecture deepening, extension seams,
  Go practices) and write one phase-wise checklist ledger. Use when the
  user wants an architecture-and-practices review, a phase-wise
  improvement report, a 10/10 plan, or runs /improve-codebase. Do not
  implement unless asked. Do not use for ponytail deletion, perf-review,
  critical-go-review, or a single visual HTML/PDF bug.
---

# Improve-codebase

One review wave. Three lenses. One ledger. No code changes.

Read `references/finding-schema.md` now. If the module path is
`gowkhtmltopdf`, also read `references/gowkhtmltopdf.md`. Ledger
rules come from `skills/phase-wise-checklist/SKILLS.md` — do not
restate them.

## 0. Defaults

- Review only. Implement only after an explicit ask on named IDs.
- Do not commit, push, or rewrite history.
- Preserve unrelated dirty files. Quote only the frozen snapshot.
- Do not launch `ponytail`, `critical-go-review`, or `perf-review`.
- Do not create a second status document beside the ledger.

## 1. Freeze

Record: branch, `HEAD`, dirty paths, today's date, product ceiling
(controlled-report renderer unless the user named another).

Existing canonical ledgers under `plans/reviews/improve-codebase/`
are claims. Re-open a closed ID only with current-source proof of
regression.

Output directory (create, do not reuse a closed folder):

`plans/reviews/improve-codebase/<slug>-<YYYY-MM-DD>/`

`<slug>` is `codebase` unless the user named a narrower slice
(`api`, `layout`, …).

## 2. Review wave — three agents, parallel

Spawn **three** read-only explore agents. Each agent reads its own
`SKILL.md` and returns finding-schema blocks only. They must not
run `go test`, `go build`, `make`, or edit files.

| Agent | Skill | Prefix after dedupe |
|---|---|---|
| A | `architecture-deepening/SKILL.md` | `ARC-` |
| B | `extension-seams/SKILL.md` | `EXT-` |
| C | `go-practices/SKILL.md` | `PRAC-` |

Give each agent: the frozen `HEAD`, the package slice (or "whole
tree"), and "do not re-file closed CR/ARC/P-rows without regression."

If the user named a single lens, skip the other two.

If the tree is tiny (one package) you may run the three lenses
yourself instead of spawning. Same finding shape.

## 3. Synthesize

1. Drop anything that fails the finding-schema hard filters.
2. Dedup on `path` + absorbing module. Keep the stronger evidence.
3. Assign `ARC-NN` / `EXT-NN` / `PRAC-NN` in priority order.
4. Mark leftover lookalikes `refuse` with the tradeoff name.
5. Cap the ledger at **25** active rows. Extra `P3`s stay in a
   "parked" list in the overview, not as fake phases.

## 4. Write the ledger

Follow `skills/phase-wise-checklist/SKILLS.md` exactly.

File: `plans/reviews/improve-codebase/<slug>-<YYYY-MM-DD>/phase-wise-checklist.md`

Parent: `plans/reviews/improve-codebase/README.md`.

Phase order (skip an empty phase):

1. **P0 — Integrity** — security, output, broken public contracts
2. **P1 — Seams** — job type, DAG leaks, half-wired flags/API
3. **P2 — Shared forks** — prepare/paint/settings helpers that
   already drifted
4. **P3 — Locality** — god-functions, stale docs, test oracles
   that lie
5. **P4 — Closure** — `make lint`, `make test`, and any
   layout/golden extras the checklist skill requires

Every row is one change or one proof. Status `[ ]` until current
evidence exists. `risk` rows start as `[~]` with the validation
step in the row text.

Do not check lint/test rows in a documentation-only wave.

After writing the ledger, add one bullet to
`plans/reviews/improve-codebase/README.md` pointing at the new
folder. Do not edit older ledgers except to `[~]`-pointer a row
you moved.

## 5. Handoff to the user

Return, in this order:

1. Path to the ledger (clickable).
2. Counts: findings in, rows out, refused.
3. The P0/P1 titles only.
4. What you did **not** do (no implementation, no perf, no
   ponytail).
5. The next command if they want work: implement named IDs, or
   run one lens again on a narrower slice.

Stop.
