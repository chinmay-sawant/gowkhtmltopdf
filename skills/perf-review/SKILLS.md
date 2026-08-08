---
name: perf-review
description: Squeeze performance out of the codebase with a parallel sub-agent review-and-fix loop. Use when the user asks to review performance, squeeze more speed, cut allocations, or profile-guided optimization of the current working tree. Runs 5 detailed review agents (read-only, no build/test), delegates fixes to fix agents, and gates the whole wave on a single final `make lint` + `make test` run.
---

# Perf Review (Parallel Squeeze Wave)

One performance review wave: 5 detailed review agents in parallel, findings
synthesized into fix briefs, fixes delegated to fix agents, and a single
final verification gate run by the orchestrator (never by subagents).

## Workflow

1. **Baseline context.** Record current benchmark numbers from
   `testdata/golden/benchmarks/benchmark-results.txt` and note the machine
   (`uname -r`, `nproc`, CPU model) before anything else. The final report
   must state whether measured numbers moved after the wave.

2. **Review wave (5 agents in parallel).** Split the codebase into 5
   non-overlapping review areas (match the module's hot paths; example split
   for this repo):
   - A: `internal/css` + `internal/html` (tokenizer, selector matching, cascade)
   - B: `internal/layout/inline.go` + text measurement hot loops
   - C: `internal/layout/layout.go` + `style.go` + `transform.go`
   - D: `internal/layout/paint.go` + display-list/page-splitting paths
   - E: `internal/pdf` (fonts, subsetting, content streams) + `internal/convert`
     + `internal/load` orchestration

   Each agent must ONLY read source and search (rg/grep/read). **Subagents
   must never run `go test`, `go build`, `go run`, `go vet`, benchmarks, or
   any command that compiles.** Analysis is static.

3. **Findings format.** Each agent returns a prioritized list; every finding
   must include: file:line, the hot-path evidence (called per page/row/rune?
   loop nesting), the specific waste (re-alloc, re-scan, fmt cost, map churn,
   slice growth, avoidable copy), a concrete fix sketch, and an effort/risk
   estimate. Mark findings as HIGH (big win, low risk), MED (tactical), LOW
   (cleanup). No finding without evidence is actionable.

4. **Synthesis + fix briefs.** The orchestrator dedupes findings and groups
   them by file so fix agents do not collide. Refuse unsafe refactors of
   rendering math; prefer: buffer reuse, fewer allocs per rune, preallocation
   with known bounds, replacing fmt in hot loops with strconv/append, hoisting
   invariant work out of per-page/per-rune loops, avoiding O(n²) scans.

5. **Fix wave (parallel agents).** Delegate one fix brief per agent (file-scoped).
   Fix agents also must NOT run `go test`/`go build`/benchmarks — they write
   code only and must keep behavior byte-identical (verify by reading, not by
   compiling). Anything that could change output must be flagged back instead
   of applied.

6. **Final gate (orchestrator only).** After the fix wave converges, run:
   - `make lint` — must be clean
   - `make test` — full suite green
   - If either fails, delegate the specific failures to fix agents, then
     re-run the gate. Repeat until green; do not weaken the gate.

7. **Evidence close-out.** Report what changed, where the wins are, whether
   benchmark numbers were re-measured (optional; only with the same command
   and machine as the baseline), and any findings explicitly NOT applied with
   reasons. Update `skills/perf-review/SKILLS.md` only if the workflow itself
   needs refinement.

## Rules

- Never run build/test/benchmark commands inside subagents. The orchestrator
  is the only executor of `make lint` / `make test`.
- No behavior changes without a byte-identical-output argument; rendering
  regressions are worse than missing micro-optimizations.
- Evidence over vibes: every finding cites file:line and the loop/hot path.
- Keep the wave small: one review → one fix wave → one gate. Do not pile
  unrelated work into the same wave.
- Do not commit or push unless explicitly asked.

## Completion Handoff

Summarize per-area findings count, fixes applied (file:line), the final
`make lint` + `make test` outcomes, and remaining opportunities (deferred
findings). State measured impact only if benchmarks were re-run with the
identical command and machine as the baseline snapshot.
