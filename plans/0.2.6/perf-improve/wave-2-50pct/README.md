# 0.2.6 performance improve wave 2

Live ledger for cutting remaining warm-path `ns/op` and `B/op` in half
on the generic library path, the public `Document` / `ImageDocument` API,
and the CLI. This is a new run. It does not reopen Snapshot L rows.

| File | Role |
|------|------|
| [phase-wise-checklist.md](phase-wise-checklist.md) | Canonical execution ledger. Close rows only on measured proof. |
| [designs/page-at-a-time.md](designs/page-at-a-time.md) | Production page-at-a-time layout, paint, and Workspace reuse. Not certified page islands. |
| [designs/section-chrome-clone.md](designs/section-chrome-clone.md) | Repeated-section table chrome clone with unique-text reflow. |

Parent waves, all complete:

- Recovery: `plans/0.2.6/perf-review/phase-wise-checklist.md` (Snapshot K)
- Improve wave 1: `plans/0.2.6/perf-improve/phase-wise-checklist.md` (Snapshot L)
- Time: `plans/0.2.6/perf-time/phase-wise-checklist.md` (Snapshot M)

Workflow: `skills/phase-wise-checklist/SKILLS.md`.

Evidence lands under `results/` and `profiles/` in this folder when phases run.
Those directories are created by the capture, not in advance.
