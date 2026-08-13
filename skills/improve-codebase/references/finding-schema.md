# Finding record

One record per finding. This is the only finding shape the pack emits.
Ledger rows are **not** defined here — after findings exist, follow
`skills/phase-wise-checklist/SKILLS.md`.

## Block

```text
### <LENS>-<NN> · <P0|P1|P2|P3> · <defect|friction|risk|refuse>
**Title:** one line, present tense, names the module not the symptom
**Location:** path:line (current source; not a historical review quote)
**Evidence:** 3–12 lines of current code + why this is real (callers, tests, or import)
**Cost:** who pays today (N call sites, forked adapters, silent no-op, untestable seam)
**Change:** the deep module / table / sentinel that should absorb this
**Proof:** smallest command or test that would close the row
**Depends-on:** other IDs, or `none`
**Not:** the over-engineered alternative you considered and rejected
```

## Class

| Class | Means | May become a checklist row? |
|---|---|---|
| `defect` | Current source violates an invariant or contract | Yes |
| `friction` | Works, but the next honest change will fork or leak | Yes |
| `risk` | Hypothesis. Must include the validation step in **Proof** | Only as `[~]` until validated |
| `refuse` | Looks like a finding; it is a decided tradeoff | No. List under "Refused" so it is not re-filed |

## Severity

- **P0** — correctness, security, output integrity, or a broken public contract
- **P1** — a seam that every later change crosses, or a half-wired extension
- **P2** — locality / duplication that does not change behavior
- **P3** — docs, comments, or cleanup that does not unblock other rows

## ID prefixes (assigned after dedupe)

| Prefix | Lens |
|---|---|
| `ARC-` | architecture-deepening |
| `EXT-` | extension-seams |
| `PRAC-` | go-practices |

The orchestrator assigns numbers. A lens working alone uses a temporary
`T-NN` and the ledger pass renumbers.

## Hard filters

Drop the finding (or mark `refuse`) when any of these is true:

- Evidence is a historical review snippet, not current source.
- The same location is already `[x]` or `[~]` in a canonical ledger under
  `plans/reviews/` and source has not regressed.
- The change is deletion-only (that is ponytail, not this pack).
- The change is a micro-allocation or benchmark claim (that is perf-review).
- The change is a visual HTML/PDF template symptom (that is debug-html-template).
- The proposed fix invents a plugin framework, a one-adapter interface, or a
  second settings system.

## Dedup key

`path` + the module that should absorb the change. Two write-ups of the
same leak (e.g. "imageout imports convert" vs "prepare is reached through
a facade") collapse to one finding. Keep the stronger evidence block.
