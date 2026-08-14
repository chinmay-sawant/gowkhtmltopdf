---
name: improve-codebase-architecture
description: >
  Audit a Go codebase for deep-module architecture: package DAG, seams,
  ownership, locality, and shallow or leaky interfaces. Emits structured
  findings for a phase-wise checklist. Use when the user wants to improve
  architecture, deepen modules, review seams, find package leaks, or runs
  /improve-codebase-architecture. Do not use for deletion (ponytail),
  performance squeeze (perf-review), or one visual HTML/PDF bug
  (debug-html-template).
---

# Improve-codebase — architecture deepening

Find places where **behavior should sit behind a smaller interface**.
Do not implement. Do not invent plugins.

Read first: `codebase-design` (vocabulary), then
`../references/finding-schema.md`. If the module path is `gowkhtmltopdf`,
also read `../references/gowkhtmltopdf.md`.

## 1. Freeze scope

- Record HEAD and whether the tree is dirty. Evidence is **current
  source**. Exclude unrelated dirty files from quotes.
- Scan `plans/reviews/` for open or closed rows on the same paths.
  Closed + unregressed → `refuse`, not a new defect.
- Product ceiling (this repo, and any report renderer like it):
  authored HTML templates, not a browser. A finding that demands JS, CGO,
  or Chrome parity is out of scope.

## 2. Discover the DAG and the intended seam

From source, not from docs:

1. Public / cmd entrypoints → the **one** job type the engine accepts.
2. Import direction. Flag any production import that points *up* toward
   a hub, CLI, or settings package a sink should not know.
3. Where untrusted input crosses a trust boundary.
4. Where PDF vs image (or any two adapters) are supposed to share work
   and where they are supposed to fork.

Docs that disagree with source are a **P3 friction** (stale claim),
not a redesign.

## 3. Score each candidate module

Use these tests. A finding needs at least one failed test plus a
location.

| Test | Fail means |
|---|---|
| **Deletion** | Removing the module does not reappear as complexity in callers — it was a pass-through |
| **One vs two adapters** | An interface has one production implementation and no test double that is actually used |
| **Interface is the test surface** | Tests reach *past* the seam into private state to prove the contract |
| **Locality** | The next honest change to this behavior already has two homes |
| **Ownership** | Caller-owned buffers/maps/slices alias into a long-lived job |
| **Sentinel identity** | The same condition is `errors.New`'d in more than one package |

Depth is leverage at the interface, not line count. A 3k-line layout
engine with a small `Layout`/`Result`/`Op` surface can be deep. A
50-line facade that re-exports another package is shallow.

## 4. What deepening looks like here

Prefer, in this order:

1. Make the existing job type the only entry (delete the extra union /
   alias once tests move).
2. Point a sibling at the real package (`prepare`) instead of a hub
   facade (`convert`).
3. Collapse a forked helper (`mediaFor`, font-registry construction)
   into the package that already owns the rule.
4. Split a named helper out of a god-function **without** a new package.
5. Document an ownership rule that is already true (CLI exclusive,
   single-goroutine document) so the next patch does not "fix" it with
   a mutex.

Refuse:

- A plugin / visitor / hook framework.
- Splitting a deep engine into a package-per-file DAG.
- A paint-sink interface "for purity" when the two adapters do not
  share a paint loop (they share ops + paint policy).
- A second typed settings hierarchy beside a compatibility `Set`.
- Extra lifecycle stages on a 3-stage pipeline that already has a
  cancellation check between stages.

## 5. Hunt list (classes, not tickets)

Walk these. File a finding only when current source fails a test in §3.

- Dual public stories that both construct the same job (tax is fine;
  three *internal* request types for one job is not).
- Settings dispatch with no compile-time link to fields — product
  surface; only file if a key is half-wired (CLI xor `Set` xor engine).
- Mode forks after the shared front half: prepare aliases, first-object
  vs validate, leftover ignore-extras warnings.
- Mutable post-handoff state (`Result`, `pdf.Document`, deprecated
  snapshots on a context object).
- Validation repeated with **different** predicates, not defense-in-depth
  with aliased sentinels.
- `containedctx` / process-global maps / `init()` outside the documented
  exceptions.

## 6. Emit findings

Write records in the finding-schema format. Cap a solo run at **15**
findings; merge the rest into the strongest related ID. Sort
P0 → P3. Every `risk` includes the experiment that would promote it.

If invoked **by** `/improve-codebase`, return the finding list only.

If invoked **alone**, continue to
`skills/phase-wise-checklist/SKILLS.md` and write:

`plans/reviews/improve-codebase/architecture-<YYYY-MM-DD>/phase-wise-checklist.md`

Phase order for this lens: invariants / DAG → job seam → ownership →
fork collapse → docs. Closure gates stay in the checklist skill.
