---
name: generate-fixture-test
description: >
  Create a PDF-operation regression test for an existing gowkhtmltopdf
  fixture. Use when a user asks to test a fixture PDF, pin PDF locations,
  compare a committed sample with a fresh conversion, or generate fixture
  test cases from the output corpus. Measure every reference PDF directly;
  never copy page numbers or coordinates from another fixture. Not for
  diagnosing a visibly wrong PDF, which belongs to diagnose-fixture-picture.
---

# Generate a fixture test

Create one evidence-backed Go test for one existing fixture PDF. The test must
prove that the committed sample and a fresh conversion of its matching HTML
produce the same page operations within the repository tolerances. It must also
pin a small set of authored features measured from that exact PDF.

This skill is for test generation and the supporting reader coverage needed by
those tests. It is not a PDF redesign, a layout fix, or a visual parity
investigation.

## Hard boundaries

- Work on one fixture PDF at a time unless the user explicitly requests a
  batch.
- Do not run any Git command. This includes read-only commands such as status,
  diff, log, and ls-files. The user may inspect or publish the changes later.
- Do not overwrite the reference PDF while measuring it. Treat the existing
  `output/*.pdf` file as the reference artifact.
- Never invent a coordinate and never copy a coordinate, page number, font, or
  operation count from another fixture.
- Do not widen a tolerance to make a failing assertion pass.
- Do not mark a plan row complete until its exact proof command has passed.

## The contract

The reference pair is:

```text
output/fixture-NN-<slug>.pdf
testdata/golden/fixture-NN-<slug>.html
```

The committed PDF is the artifact a person opens. The fresh PDF is produced by
the current conversion pipeline through `requestForFixture`. Compare both with
`pdf.ParsePageOps` and then pin authored details in the committed record.

Coordinates use PDF user space:

- The origin is the bottom-left corner.
- X grows right and Y grows up.
- Distances are points, where 72 points equal one inch.
- Page numbers in the test helpers are 1-based. `pdf.PageOps.Page` is 0-based.
- PyMuPDF reports Y from the top, so the inspection script flips it before
  printing values for the Go test.

The shared comparison checks the ordered operation record:

- page count and MediaBox
- text page, string, origin, size, resolved BaseFont, and fill color
- stroked segment page, endpoints, width, and stroke color
- filled box page, bounds, and fill color
- image page and placement box

The fixture-specific assertions pin representative authored features. They do
not need to repeat every text string when the full operation record is already
compared. Choose anchors that make a layout change easy to understand.

## Phase 1: discover the current implementation

Read the local knowledge-base entry point first:

```text
knowledge-base/wiki/index.md
```

Then inspect the current source. Reuse existing helpers before adding another
one:

```text
internal/convert/golden_test.go
internal/convert/output_ops_test.go
internal/pdf/page_ops.go
internal/pdf/page_ops_test.go
scripts/inspect_pdf_ops.py
```

Confirm these facts from source, not memory:

1. `requestForFixture` selects the matching golden HTML and sets the fixture's
   normal A4, margin, background, and local-file options.
2. `readCommittedOps` reads the reference PDF under `output/`.
3. `freshOps` converts the named HTML and parses the fresh bytes.
4. `assertPageOpsMatch` compares both operation records.
5. The `assertOps*` helpers use the repository's existing tolerances.

If the shared reader or helpers do not exist, add the smallest package-local
infrastructure needed before writing the fixture test. Keep the reader in
`internal/pdf` and the fixture assertions in `internal/convert`. Do not add
fixture-specific parsing logic to the PDF reader.

## Phase 2: resolve the exact fixture pair

Start from the PDF the user names. Check that it exists, then derive the HTML
by matching the complete basename, not only the numeric ID. This matters for
duplicate IDs such as the two fixture-29 files.

For example:

```text
PDF:  output/fixture-01-simple-invoice.pdf
HTML: testdata/golden/fixture-01-simple-invoice.html
```

Read the HTML before choosing anchors. Record which visible features it
actually authors:

- text groups and their semantic role
- tables and repeated rows
- borders and rules
- filled backgrounds
- images and their source paths
- links, if link rectangles are in the requested scope
- header and footer companions, if the fixture has them

Do not call a styled text block an image. Do not add an image assertion when
the HTML has no image. Do not assume that the next fixture has the same page
count, margins, rows, or feature mix.

If the basename has no unique matching HTML, stop and report the ambiguous
pair. Resolve the pair before measuring anything.

## Phase 3: measure the reference PDF

Open the exact reference PDF with the independent inspector before writing Go
assertions:

```bash
python3 scripts/inspect_pdf_ops.py output/fixture-NN-<slug>.pdf
```

Use `--page N` after the all-page pass when a PDF is long:

```bash
python3 scripts/inspect_pdf_ops.py output/fixture-NN-<slug>.pdf --page N
```

The output is the source of truth for the test values. Record, per selected
operation:

```text
page, text or drawing, x, y, width/height, size, color, font
```

For every fixture, also record:

- page count
- each page's MediaBox
- total text, stroke, fill, and image counts
- whether the output contains authored images or fills

Select anchors by geometry and meaning:

- one upper-page title or heading
- one middle-page item or table value
- one lower-page or footer value
- a right-aligned total when present
- a rule or border when present
- an image placement when the HTML authors an image

For a multi-page fixture, use anchors from the first and last pages and at
least one meaningful middle page when the document has one. For a one-page
fixture, use different vertical regions. Prefer unique strings on the named
page.

Do not copy a location from the HTML's CSS, a screenshot, a different PDF, or
another fixture. CSS coordinates and screenshot coordinates use different
origins and units. The PDF inspector output is the value to pin.

If PyMuPDF is unavailable, do not guess. Report the missing measurement tool
and use an approved PDF inspection fallback that produces page-space values.
Ghostscript can provide text or raster evidence, but a raster screenshot alone
is not a precise coordinate source for a new assertion.

## Phase 4: write the fixture test

Use the naming pattern:

```text
internal/convert/output_fixture_NN_<slug>_test.go
TestOutputFixtureNN<PascalCaseSlug>
```

Use the shared shape from `references/test-template.md`:

1. Read the committed PDF.
2. Convert the matching HTML through `requestForFixture`.
3. Parse the fresh bytes.
4. Compare the complete operation records.
5. Pin measured authored text, rules, fills, images, page count, and counts.

Keep the test parallel-safe. Use `t.Parallel()` when the existing package
allows it. Do not write generated PDFs into `output/` from the test. Use the
existing temporary output path used by `runPDF`.

Use the fixture's own measured values. For example, a fixture-01 test may pin
the title at one location and a table row at another, but those numbers must
not appear in fixture-02's test. The page number and every coordinate belong to
the PDF that was inspected.

The fixture test should fail for both kinds of drift:

- The committed sample changes while the current conversion stays the same.
- The current conversion changes while the committed sample stays the same.

The independent pins prevent a stale but self-consistent reference PDF from
being the only oracle. Keep the pins tied to features actually authored by the
HTML. Existing semantic golden tests remain responsible for broad text needles,
URI presence, image presence, and PDF structure.

## Phase 5: cover the reader before scaling out

When this work adds or changes `pdf.ParsePageOps`, synthetic tests must cover
the operators that the reader claims to support. At minimum, cover:

- uncompressed and Flate-compressed streams
- `Td` and `Tj`
- `Tm` and `TJ`
- font resource resolution and text size
- RGB, grayscale, and CMYK color paths when the reader supports them
- stroked paths and line width
- filled rectangles
- image placement through `cm` and `Do`
- text or drawing inside a Form XObject

A fixture-01 PDF may use only `Td`, `Tj`, RGB colors, a stroked rule, and a
Flate stream. That proves the fixture path, not every branch of a reusable
reader. Add synthetic coverage before relying on an untested branch for later
fixtures.

If the reader intentionally supports only PDFs emitted by this repository,
state that in its comments and the plan. A general PDF reader would also need
to consider page-tree nesting, contents arrays, inherited resources, rotated
pages, inline images, clipping, line caps and joins, dash patterns, and
transparency. Do not silently imply that narrow support is general PDF
support.

## Phase 6: validate the exact final tree

Run formatting first. An empty `gofmt -d` result is required for touched Go
files.

For a single fixture, run the focused proofs with a writable temporary Go
cache:

```bash
review_cache=/tmp/gowkhtmltopdf-fixture-test-cache
GOCACHE="$review_cache" go test ./internal/pdf -run 'TestPageOps' -count=1
GOCACHE="$review_cache" go test ./internal/convert -run 'TestOutputFixtureNN<Slug>$' -count=1
```

Then run the affected packages together:

```bash
GOCACHE="$review_cache" go test ./internal/pdf ./internal/convert -count=1
```

For a batch or a shared reader change, finish with the repository gates in the
order required by `AGENTS.md`:

```bash
make test
make golden
make claim-scan
make lint
```

Read the exit status of every command. A cached earlier result does not prove a
later source tree. Re-run the focused fixture test after the last source or
test edit.

Do not run Git commands during this workflow. Report the files inspected and
the validation results without claiming a clean or dirty Git state.

## Phase 7: close the ledger

If the user asked for plan tracking, update the matching phase file only after
the exact fixture proof passes. Keep the row tied to the actual test name and
actual file name. Update the parent ledger and the local knowledge-base when
the plan or behavior is completed, following the repository instructions.

Do not mark later fixtures complete because the shared helper passed on one
fixture. Each PDF needs its own inspection, measured coordinates, test case,
and proof command.

## Completion checklist

The task is complete only when all applicable answers are yes:

- The PDF and matching HTML were resolved by complete basename.
- The exact reference PDF was opened and measured.
- Every page number and location in the test came from that PDF.
- Multi-page anchors cover the required pages.
- Authored images, fills, rules, and text features are represented correctly.
- The committed record and fresh conversion are compared.
- Synthetic reader branches used by the rollout have tests.
- Formatting and focused tests pass on the final tree.
- Full repository gates pass when the scope requires them.
- The plan row and knowledge-base are synchronized when requested by repo
  policy.

## Report shape

```text
Fixture: output/<pdf> ↔ testdata/golden/<html>
Test: internal/convert/<test-file>:<line>
Measured: pages, MediaBoxes, anchors, authored drawings, counts
Proof: commands and exit results
Remaining: unimplemented fixtures or reader branches
Git: not inspected or run
```

## Related skills

- `diagnose-golden-fixture` for structural golden failures.
- `diagnose-fixture-picture` for a visibly wrong fixture PDF or screenshot.
- `phase-wise-checklist` for evidence-backed plan ledgers.
