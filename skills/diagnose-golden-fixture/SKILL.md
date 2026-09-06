---
name: diagnose-golden-fixture
description: >
  Diagnose a failing gowkhtmltopdf golden corpus fixture (wrong page count,
  missing needles, structural envelope) by building a tight red loop, bisecting
  to the first bad commit, ranking falsifiable hypotheses, instrumenting one
  variable at a time, and fixing the interaction without undoing intentional
  prior work. Use when TestGoldenCorpusAllFixtures fails, pages = N want [A,B],
  a golden fixture regresses after a layout change, or the user runs
  /diagnose-golden-fixture. Not for visual PDF side-by-side compare (that is
  fixture-pdf-regression) and not for template look-and-feel pick lists
  (debug-html-template).
---

# Diagnose a golden fixture

Structural golden failure: page envelope, needles, images/URI flags, PDF
integrity. Prove the mechanism before editing. Prefer a fix that keeps the
intent of the blamed commit and only breaks the bad interaction.

## Phase 1 - Tight red loop

Completion: one command you have already run that goes red on this fixture
and green when fixed.

```bash
go test ./internal/convert/ -run 'TestGoldenCorpusAllFixtures/fixture-NN' -count=1
```

Record the exact symptom (`pages = 37, want [9, 9]`). Sibling fixtures that
share the envelope are a cheap contrast set (run them too).

Do not open a theory before this loop exists.

## Phase 2 - First bad commit

Bisect `master..HEAD` (or the merge-base of the branch) with that same test
as the oracle. Skip commits that lack the fixture or fail to compile
(`exit 125`).

```bash
git bisect start HEAD master
git bisect run <script that runs the Phase 1 command>
```

Read the blamed commit message and diff. Name what it was trying to fix
(fixture-61 blank pages, Chrome parity, etc.). That intent is load-bearing
for Phase 5.

Worktrees are fine for read-only compare against `master`. Prefer bisect over
guessing from `git log -S` when the symptom is a number (page count).

## Phase 3 - Ranked hypotheses

Write 3-5 falsifiable hypotheses before testing any. Each needs a prediction:

> If X is the cause, then changing Y makes the page count return to the
> envelope / makes it worse.

Typical buckets for page-count blow-ups:

1. Forced breaks (`page-break-*` / `break-*`) firing on more boxes than intended
2. Definite height / min-height / flex stretch writing `Height` into rebuild
3. Multicol / float / table measure reserving tall empty bands or page-snapping
4. Content width collapse causing massive wrap
5. Paint ops at tall Y while box height is later crushed (hidden blow-up)

Show the ranked list. Then test top-down.

## Phase 4 - Instrument one variable

Change one gate at a time. Tag every temporary log `[DEBUG-<short>]` and
remove them before the final commit.

Useful probes:

- Log at the suspected clamp / snap / measure site: before value, after
  value, `Height`, `definiteH`, node `class` / `data-prop`
- Temporarily disable one path (early `return false` on a predicate) and
  re-run Phase 1
- Prefer evidence over reverting the whole blamed commit

When box height looks fine but pages are huge, assert on **paint extent**
(`maxY` of `OpText` / paginated ops), not `box.height`. A later clamp can
crush the box while ops still span many pages.

## Phase 5 - Fix the interaction, keep the intent

Do not auto-revert the blamed commit. Split what it needed from what it
broke:

1. Keep the behavior the blamed commit proved (unit test or fixture that
   still must pass).
2. Narrow the new gate so the accidental caller stops hitting it (example:
   flex stretch writes `Height`, but `column-fill:balance` must not treat
   that as a column fill cap).
3. Add a package regression test that fails under the old gate and passes
   under the new one. Prove it by temporarily re-breaking the gate once.
4. Re-run Phase 1 on the original fixture, the blamed commit's fixture(s),
   and the new unit test.

## Phase 6 - Cleanup

- All `[DEBUG-...]` gone
- No throwaway harness files left tracked
- Touched package tests green; original golden fixture green
- Report: first bad commit, causal chain in plain words, what stayed, what
  narrowed, proof commands

## Output shape

```
Symptom: pages = N, want [A, B] (fixture-NN)
First bad: <sha> <subject>
Intent kept: <one line>
Causal chain: <3-6 short steps with numbers>
Fix: <file:line behavior change>
Proof: <test commands and pass/fail>
```

## Related skills

- `fixture-pdf-regression` - visual good-vs-current PDF region compare
- `debug-html-template` - template symptom table; wait for user pick
- diagnosing-bugs (user skill) - general red-loop discipline this skill specializes
