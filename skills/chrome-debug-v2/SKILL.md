---
name: chrome-debug-v2
description: Budgeted Chrome-versus-Go PDF debugging for one fixture. Use when one test/chrome case looks different in Chrome and gowkhtmltopdf and you want the shortest path to a verified fix. Runs one compare, one bounded diagnostic agent, one fixer, one criticizer, then one gate pass. Not for golden-corpus failures or multi-case sweeps.
---

# Chrome debug v2

Fix one Chrome-versus-Go PDF difference with as few loops as possible.

This skill supersedes `skills/chrome-debug/SKILL.md`; keep the old file as
long-form reference, do not edit it.

## Why this exists

The 2026-09-20 runs over-iterated: case 30 blocked 10m42s on a council that
started after both root causes were recorded; case 05 spent 25 subagents on 5
fixtures. The budgets below are the fix; treat them as caps.

## Rules

- One fixture per run; other differences are reported, not fixed. A second
  fixture starts a new run after the current run is closed. Never branch
  mid-run.
- One Go render per edit round; re-render Chromium only if the fixture HTML
  changed since Step 1. Never re-render, re-measure, or re-run a probe without
  an intervening edit.
- Probe cap: 2 runs per probe, and at most 2 distinct Go-engine probes (a test
  run or a compare rerun) before the first source edit. `chrome_rects.py` and
  the Step 1 compare do not count. The fixer's first red probe replaces the
  pre-edit allowance. After an edit, only the prediction's probe; a third means
  stop and ask.
- Agents: one diagnostic, one fixer, one criticizer. Nothing else. Budgets are
  guidance: take output when the contract returns and stop a drifting agent.
- One agent per package. Two sessions on one checkout may run only on disjoint
  packages and only with explicit user sign-off; the 2026-09-20 collision in
  `internal/layout` forced a 12x lint loop and a worktree verification pass.
- Gates once, at the end, on the frozen tree: `make test`, `make golden`,
  `make lint`. Add `make claim-scan` only when docs or frontend changed.
- Stop when the pictures match, when the user says it looks good, or after 3
  fixer edits without a match. Stop means stop: no extra render, no extra gate.
  If the user approves after at least one source edit, run the Step 6 gates once
  on the frozen tree if they have not already run; otherwise report and stop.
  If approval arrives before any edit, stop without gates.
- A criticizer defect gets one more fixer round and counts against the 3-edit
  cap. When the cap is reached, report the defect and ask. Never reopen a
  closed run.
- No git unless asked. No bare `go test ./...`. No PDF sha256 comparisons:
  the writer embeds `time.Now()`. If the todo API is missing, say so and go on.

## Step 1: artifacts and one compare

```bash
set -o pipefail  # tee must not mask the compare exit code
slug="case-30-wpt-auto-margins-column"
html="test/chrome/cases/${slug}.html"
run="/tmp/chrome-debug/${slug}"
make build
mkdir -p "$run"
node scripts/puppeteer_print.js "$html" "$run/chromium.pdf"
./bin/gowkhtmltopdf --allow-local-files -o "$run/gowkhtmltopdf.pdf" "$html"
python3 skills/chrome-debug-v2/scripts/compare_pdfs.py \
  "$run/chromium.pdf" "$run/gowkhtmltopdf.pdf" --outdir "$run/pages" \
  --max-rows 40 | tee "$run/compare-before.txt"
```

The script prints per page: page size, ink bbox with pixel count, pixel delta,
font lists, and matched/unmatched drawing, text, and image rows. Page images
land in `$run/pages` with `--outdir`. The RESULT line separates page count
delta, size mismatches (over 0.1pt), unmatched rows per kind, and pages over
the threshold; `pixel-only` means only pixels failed.

Concurrent-run hint, not a guarantee: `ls -lt /tmp/chrome-debug/ | head -6`.
If another fixture's directory changed in the last 30 minutes, stop and ask
the user to serialize.

For authored CSS, Chrome's own numbers:
`python3 scripts/chrome_rects.py "$html" --selector .some-class`.

Never normalize page scale or margins; case 26 hid a Chrome page shrink that
way and its first "match" failed the user check.

Refresh `test/chrome/pdf/<slug>.pdf` only when asked, and only after the last
edit: `go test ./test/chrome -run TestChromeCasePDFOutputs -count=1` rewrites
every case PDF, so keep only the slug's file.

## Step 2: read the table

Most mismatches classify themselves. Pick one row, then move.

| table shows | owner | next |
|-------------|-------|------|
| drawing row missing, moved, or resized | layout geometry | diagnostic agent |
| drawing present, wrong order or clipped | paint | diagnostic agent |
| drawing count much higher on one side | paint or pagination | diagnostic agent |
| page size or global placement differs | setup or scale | run `chrome_rects.py`, then diagnostic agent |
| pixel delta over threshold, signatures sample-equal | image handling, color, or PDF writer | open `$run/pages` crops, then diagnostic agent |
| text only, matching known font differences | font substitution | out of scope, report |
| nothing differs | done | report |

Rows assigned an owner run steps 3 to 5 in order. Rows marked out of scope or
done report and stop.

### Visual equivalence stop rule

If page sizes match within 0.1pt and every page's pixel delta is at or under
the threshold, but signature rows still differ, the difference may be
representation-only. Check `$run/pages` once and compare the ink bboxes: when
the ink bounding boxes coincide within 0.5pt, report visually equivalent and
stop. Do not edit the engine to match Chrome's path decomposition; this stops
the run before steps 3 to 5.

Call a text delta font substitution only when the strings and sizes match and
the bbox delta is under 0.5 pt; otherwise it is a layout text difference.

## Step 3: diagnostic agent

One read-only agent, 12 tool calls. Give it the compare output and paths.

```text
You are the diagnostic agent for one Chrome-versus-Go PDF mismatch.

Run dir: <path>
Fixture: <slug> (<path>)
Chromium PDF: <path>
Go PDF: <path>
Compare table: read <run-dir>/compare-before.txt (do not re-run it)
Chrome DOM numbers: <paste chrome_rects.py JSON, or "not run">

Rules: read-only. No edits, builds, renders, tests, git, gates, or probes; do
not re-measure the table. Budget 12 tool calls, one pass. If the todo API is
missing, say so.

Return exactly:
1. Classification: fixture/CSS authoring, placement or scale, layout geometry, paint order or clipping, PDF writer, or scope difference.
2. Owning package and function, or "not visible".
3. One hypothesis that explains every measured row.
4. One prediction naming the row or value that will change.
5. The cheapest probe that would falsify the hypothesis.

If two owners remain possible, say which single measurement separates them.
```

Continue only when the hypothesis has a prediction. If nothing is visible, do
not edit: run the one measurement the agent named, or stop and ask. After 3
edits without a match, run no gates: report the diff and failed predictions.

## Step 4: fixer agent

One agent, max 3 edits. It may edit the owning package only.

```text
You are the fixer. Make the smallest change that satisfies the prediction.

Run dir: <path>
Fixture: <slug>
Owning package: <internal/...>
Hypothesis: <...>
Prediction: <...>

Rules:
- Edit the owning package only. Do not touch other packages.
- Red probe first. Prefer a permanent test. Temporary probes: write the test
  to /tmp and use
  `go test -overlay=/tmp/<name>.overlay.json ./internal/<pkg> -run <Test> -count=1`
  with `{"Replace":{"/abs/path/zz_probe_test.go":"/tmp/zz_probe_test.go"}}`.
  Probe files under /tmp do not count toward the 3-edit cap; never leave zz_*
  files in the tree.
- Max 3 edits; on the fourth attempt, stop and report the diff and
  measurements instead of editing again.
- After the last edit, once: make build; go test ./internal/<pkg> -count=1
  -short; regenerate the Go PDF; re-render Chromium only if the fixture HTML
  changed since Step 1; run compare_pdfs.py once with `set -o pipefail` before
  `| tee "<run-dir>/compare-after.txt"` (pipefail is required when piping to
  tee).
- Do not run make test, make golden, or make lint. Do not touch
  knowledge-base, plans, or docs.

Return: files changed, one line per edit, probe and compare results, and whether the prediction held.
```

## Step 5: criticizer agent

One read-only agent. Its first job: prove the fix did not cause the remaining
differences.

```text
You are the criticizer. Decide whether the fix is real and whether it caused
any regression.

Run dir: <path>
Inputs: <run-dir>/compare-before.txt, <run-dir>/compare-after.txt,
<run-dir>/pages/, the fix diff, Chrome DOM numbers.

Rules: read-only. No edits, builds, renders, or gates. Budget 12 tool calls.
Answer "did the change cause each remaining difference" against the pre-change
control; differences present before are out of scope. Review only pages or rows
with nonzero deltas; ignore control font differences.

Return: per-page verdict (matched, expected difference, defect), the evidence
for each, and one sentence on regression risk outside the fixture.
```

Do not accept "looks good" without a crop, a number, or a source line. When
only expected differences remain, the loop is done.

## Step 6: gates once and report

Run these after the last edit, once:

```sh
go test ./internal/<owning-package> -count=1 -short  # fixer ran this for the owning package; repeat only if not
make test
make golden
make lint
```

Then report, stating whether the fixture matches and which differences remain:

```text
Fixture and revision:
Chromium PDF:            Go PDF:
Compare result:          (drawings X unmatched, text Y unmatched, images Z unmatched, pixel deltas, page-size delta)
Classification, owner, hypothesis, prediction:
Fix:                     (files, one line per edit)
Criticizer verdict:
Gates:                   (commands and exit codes)
Remaining differences:   (in scope or pre-existing)
```

## Do not do these

All of these happened on 2026-09-20: a council after the root cause, a picture
council for a mechanical fix, the 7-agent ritual per fixture, gates on an
unchanged tree, a cycled probe, a re-render per micro-edit, an approval that
restarted the loop.

Escalate instead: with the compare table, the current diff, and the failed
prediction, ask the user. A question costs less than another hour.

Related: `skills/diagnose-fixture-picture/` for deep picture review, `skills/diagnose-golden-fixture/` for golden failures.
