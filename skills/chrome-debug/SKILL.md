---
name: chrome-debug
description: Compare an installed Chromium PDF with a gowkhtmltopdf PDF for one HTML fixture, measure visual and geometry differences, coordinate bounded diagnosis and picture-review subagents, apply a minimal verified fix when authorized, and refresh artifacts. Use for Chrome-versus-Go PDF mismatches, not generic golden failures or unrelated browser automation.
---

# Chrome Debug

Use this skill for a browser-backed PDF mismatch in this repository. Keep the
comparison tied to one fixture until the cause is understood. This workflow
can diagnose only, or diagnose and fix when the user authorizes source edits.

## Rules

- Read the repository `AGENTS.md` before acting.
- Do not edit an existing skill. This skill is the repository-local workflow
  for Chromium-versus-Go PDF debugging.
- Do not use Git unless the user explicitly asks for Git work.
- Do not change an existing scale setting or compensate for a mismatch by
  changing renderer scale before proving that scale is the cause.
- Never call a Go PDF a Chrome PDF. Keep the artifacts separate:
  - Chromium reference: `/tmp/chrome-debug/<fixture>/chromium.pdf`
  - Go output: `/tmp/chrome-debug/<fixture>/gowkhtmltopdf.pdf`
  - Repository Go artifact, only when explicitly requested:
    `test/chrome/pdf/<fixture>.pdf`
- Use the same HTML input for both renderers. Do not compare different source
  revisions or stale PDFs.
- Do not run bare `go test ./...`; use the repository Makefile gates for the
  full suite.

## 1. Establish the case and fresh artifacts

Complete this phase when the exact fixture, renderer versions, and both fresh
PDFs are recorded.

1. Identify the HTML file, normally under `test/chrome/cases/`, and record its
   basename. Read the relevant CSS and the nearby Chrome test or manifest.
2. Confirm the browser with `command -v google-chrome` and
   `google-chrome --version`. The repository helper defaults to
   `/usr/bin/google-chrome` and uses a fixed device scale factor of 1.
3. Build the current product before generating behavior artifacts:

   ```sh
   make build
   ```

4. Generate a fresh Chromium reference with the repository helper:

   ```sh
   case_slug="case-14-legacy-flex-align-baseline"
   html="test/chrome/cases/${case_slug}.html"
   run_dir="/tmp/chrome-debug/${case_slug}"
   mkdir -p "$run_dir"
   node scripts/puppeteer_print.js "$html" "$run_dir/chromium.pdf"
   ```

   Set `PUPPETEER_EXECUTABLE_PATH` only when the installed Chromium binary is
   elsewhere. Do not change the helper's page format, background printing, or
   device scale for a normal comparison.
5. Generate the Go PDF from the same HTML and freshly built binary:

   ```sh
   ./bin/gowkhtmltopdf --allow-local-files \
     -o "$run_dir/gowkhtmltopdf.pdf" "$html"
   ```

   If the user asks to refresh the repository artifact, generate the Go PDF
   at `test/chrome/pdf/${case_slug}.pdf` as a separate, explicit action. The
   repository path is lowercase `test/chrome/pdf`; it is not the Chromium
   reference directory.

## 2. Render and measure before interpreting

Complete this phase when both PDFs have page images and a short evidence table.

Render every page at the same DPI:

```sh
python3 skills/diagnose-fixture-picture/scripts/render_fixture_pages.py \
  "$run_dir/chromium.pdf" "$run_dir/chromium-pages" 150
python3 skills/diagnose-fixture-picture/scripts/render_fixture_pages.py \
  "$run_dir/gowkhtmltopdf.pdf" "$run_dir/gowkhtmltopdf-pages" 150
```

Inspect the actual PNGs with the image viewer. Use PyMuPDF (`fitz`) for page
count, page size, text boxes, drawings, and coordinates. Use Pillow for exact
pixel colors, bounding boxes, edge continuity, and crops. Keep measurements
reproducible in a command or a small `/tmp` probe rather than estimating from
the screen.

Normalize only the comparison math for page size, DPI, and global placement.
Do not alter the renderer's scale to make the images overlap. Separate a
global scale or translation difference from a local layout, paint, clipping,
or stacking difference. A local edge that differs after normalization is
engine evidence, not proof that the fixture is authored incorrectly.

For each affected branch or row, record:

- Chromium coordinates and colors.
- Go coordinates and colors.
- Whether the difference is geometry, paint order, clipping, text shaping, or
  a global transform.
- The CSS rule and source function that own the observed behavior.

For an outline, border, or overflow mismatch, measure the whole edge and the
overflowing child separately. CSS `outline` is not the same thing as a border.
Check the display-list or paint phase before changing `z-index` or stacking
context code. The expected browser picture may be an outline painted above
descendant overflow even when no explicit `z-index` is present.

## 3. Classify the mismatch and make one falsifiable probe

Complete this phase when one hypothesis explains the measured difference and a
minimal probe can distinguish it from the nearest alternative.

Classify the finding as one of:

- fixture or authoring error;
- expected browser-versus-product scope difference;
- global scale or placement mismatch;
- local layout or geometry defect;
- paint order, clipping, or stacking defect;
- PDF writer or rasterization-only defect.

Use the owning layer as the tie-breaker. For layout, inspect direct geometry
results and add a focused geometry assertion. For paint, inspect operation
order and layer classification. For PDF output, compare the same rasterized
page before blaming the PDF writer.

Make one red probe before a production fix. Prefer a permanent focused
regression test in the owning package when the behavior is clear. If a probe
must be temporary, put it under `/tmp` and do not leave a repository file.
The probe must fail on the current behavior and pass only when the hypothesis
is addressed. Record its output.

When changing paint operations, preserve the packed hot operation contract.
The `layout.Op` size must remain within its existing limit. Rare metadata such
as an outline marker belongs in the optional `opExtra` payload with nil-safe
accessors, not in the hot struct fields. Re-run the size test after such a
change.

## 4. Run the diagnosis council

Complete this phase when all assigned agents have returned evidence or have
been explicitly marked unavailable.

Launch three read-only subagents in parallel, with the same fixture, PDFs,
page images, measurements, source revision, and probe result:

1. **Analyst**: trace the relevant load, layout, paint, pagination, and PDF
   paths; identify the first layer where Chromium and Go can diverge.
2. **Interpreter**: inspect the rendered evidence and CSS semantics; decide
   whether the picture is authored, expected, or an engine defect.
3. **Critic**: attack the leading hypothesis with alternative explanations,
   especially scale, clipping, stale artifacts, and PDF rasterization.

Require each agent to return concrete file paths, line references, commands,
and a falsifiable conclusion. Agents must not edit files or run Git. A stalled
worker is not evidence. Check that each spawned worker produced a turn before
using its conclusion.

Do not implement a fix from one untested opinion. Continue only when the
evidence agrees, or when the disagreement has been resolved by a new probe.

## 5. Apply the smallest authorized fix

Complete this phase when the focused probe and a permanent regression test pass
for the intended behavior.

Edit the owning package only. Preserve unrelated work in the checkout. For a
paint-order defect, change the layer classification or operation ordering at
the point where the engine owns that decision. Do not hide the symptom with a
global scale, arbitrary z-index, clipping, or a broad refactor.

After the source edit:

```sh
make build
go test ./internal/<owning-package> -count=1 -short
```

Regenerate both PDFs and page PNGs after the final source edit. Never compare
an old Go PDF against a new Chromium PDF.

## 6. Run the picture council

Complete this phase when the final picture has been checked across the full
fixture and no assigned row or page slice has an unexplained regression.

Launch four read-only picture-review subagents in parallel. Give each a
disjoint page, row, or branch slice. Each agent must answer:

- Does the Go picture match the Chromium reference for this slice?
- What exact pixels or geometry support the answer?
- Is any remaining difference global, local, or expected?
- What regression risk remains outside this slice?

Do not accept a general "looks good." Require a crop, measurement, or source
reference. Close completed agents, and investigate any worker that returns no
evidence before declaring the council complete.

## 7. Run final gates and report the result

Complete this phase only after the final source tree, binary, PDFs, and images
have all been regenerated and inspected.

Run the narrow tests first, then the repository gates:

```sh
go test ./internal/<owning-package> -count=1 -short
make test
make golden
make lint
```

If documentation or frontend content changed, also run `make claim-scan`.
Verify the final artifacts with `file`, PyMuPDF page counts, and the rendered
PNG inspection. The last validation must cover the exact final source tree,
not a tree from before the last edit.

Report the result in this order:

```text
Fixture and source revision:
Chromium PDF:
Go PDF:
Measured difference:
Classification:
Diagnosis council:
Probe and fix:
Picture council:
Validation:
Remaining differences:
```

State clearly whether the PDFs match locally, whether only a global scale or
placement difference remains, and which artifacts can be reproduced.
