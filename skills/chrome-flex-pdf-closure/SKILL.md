---
name: chrome-flex-pdf-closure
description: Close the complete 40-case Chromium Flexbox interaction inventory in gowkhtmltopdf by converting static inputs, proving the right engine and output behavior for each case, and recording coverage evidence. Use when agents are assigned individual Chrome Flexbox cases, case groups, or the whole inventory.
---

# Close the Chrome Flexbox interaction inventory

Use this skill to move every case in the 40-case Chrome Flexbox inventory from
scaffold to evidence-backed completion. The proof target depends on the
manifest's `goTarget`. PDF is required for print cases, but it is not the only
valid proof boundary. A passing direct layout test does not prove that a PDF,
PNG, or Chromium-reference path preserves the result.

Read these files before editing:

- `knowledge-base/wiki/index.md`
- `plans/0.2.7/chrome-flex-interactions/00-canonical-chrome-flex-interaction-plan.md`
- `test/Chrome/README.md`
- `test/Chrome/manifest.json`
- the Chromium source file named by the case
- the existing test helper in the package you will change

## Work boundaries

The case assignment is the write boundary. Do not edit another agent's case,
shared test helper, production layout code, or manifest row unless the
assignment includes it. If a shared fix is needed, report the failing input,
the expected geometry, and the owning package instead of taking that file.

Agents may work in parallel only when their write sets are disjoint. Keep
fixture files, layout tests, PDF tests, and coverage metadata in separate
ownership groups.

Do not run Git commands. Do not build Chromium. Do not copy Blink C++ test
assertions into Go. Reuse the HTML and CSS behavior, then write an assertion
at a Go-owned boundary.

## The four completion layers

Apply all four layers to every assigned case. Do not mark one case complete
because a representative case in the same property family passes.

### 1. Static case input

Replace the generated scaffold with a reviewed static HTML file under
`test/Chrome/cases/`.

The file must have:

- a valid doctype;
- the exact CSS interaction under test;
- stable labels or geometry markers that make the expected result observable;
- a short expected-behavior comment;
- the Chromium source path comment;
- no browser-only JavaScript or generated assertion matrix.

Keep the case's `id`, `fixture`, source path, category, and `goTarget` aligned
with `test/Chrome/manifest.json`. The manifest status remains `scaffold` until
the required evidence exists.

Run the Chrome inventory validator after changing the input:

```sh
python3 scripts/generate_chrome_flex_cases.py
go test ./test/Chrome
```

If the generator rewrites the file, update the generator's case template or
case table instead of leaving a hand edit that regeneration will erase.

Completion condition: the file is a reviewed static input, its expected
behavior is named, and the validator passes.

### 2. Engine behavior

Add or update the focused test at the boundary named by the case's `goTarget`.
Use an independent expected result from the Chromium behavior, CSS example,
or worked specification case. Do not recompute the expected value from the
implementation.

Use these boundaries:

- `layout-unit`: focused `internal/layout` geometry assertions;
- `chrome-reference`: a named Chromium result compared with the Go result;
- `golden-fixture`: PDF page and semantic assertions in `internal/convert`;
- image regressions: decoded PNG pixels or a reviewed raster crop in
  `internal/imageout`.

The assertion should cover the layout decision named by the case:

- sizing: used widths or heights after basis, grow, shrink, min, and max;
- alignment: item positions relative to the container, including exact center
  or free-space distribution;
- flow: line assignment, axis positions, reverse order, wrapping, and
  `align-content` placement;
- print fragmentation: the page and line geometry that must survive a break.

A test that only checks that an item moved away from the origin is too weak.
For centering, compare the item center with the container center. For wrapping,
check both line membership and the cross-axis offset. For reverse flow, check
the positions of named items, not only their extracted text order.

Run the focused package test first:

```sh
go test ./internal/layout -run '<test name>' -count=1
```

Completion condition: the expected behavior fails against the old behavior or
would catch a regression, passes with the current implementation, and the test
names the case or interaction it proves.

### 3. Product output

Run the same static HTML through the product output path required by the case.
Use `internal/convert` for PDF, `internal/imageout` for PNG or JPEG, and the
Chromium runner for reference cases.

Every output case must prove the checks that apply:

- PDF: valid structure, embedded font, expected page envelope, ordered
  semantic text where relevant, and geometry or a reviewed raster crop;
- PNG or JPEG: decoded dimensions, visible geometry, and pixels that prove
  the case's expected placement or sizing;
- Chromium reference: the same static input, matching selected geometry or
  pixels, and an explicit tolerance or unsupported-feature decision.

Do not treat a successful conversion as proof of the interaction. Do not
treat `make golden` as proof of centering, item width, or line placement by
itself. Existing golden checks cover structure, page envelopes, selected text
needles, and selected features. Add a geometry assertion or crop check for the
behavior under test.

For the print-fragmentation case, use a golden fixture and pin page count,
ordered text, and the location of repeated content. Add a raster crop only
when structural and semantic checks cannot detect the regression.

Run the focused output test before the full gates:

```sh
go test ./internal/convert -run '<test name>' -count=1
```

Use the equivalent focused command for `internal/imageout` or the reference
runner when the case has another output target.

Completion condition: the selected product output proves the case's expected
behavior, not merely that conversion completed successfully.

### 4. Coverage record

Record the evidence for every assigned case in the canonical v0.2.7 plan and
update the matching knowledge-base page in the same work session.

Use the case allocation already defined by the plan:

- `layout-unit`: direct box geometry;
- `chrome-reference`: the same static input compared with Chromium geometry
  or pixels;
- `golden-fixture`: page bounds and semantic PDF checks.

Do not create a pairwise matrix of every CSS property. The inventory is the
40 named cases in `test/Chrome/manifest.json`. Close each case individually.
Add a named case when two declarations share a layout decision or exercise a
distinct branch. The coverage record must make missing families visible,
including direction plus alignment, automatic sizing plus alignment, wrapping
plus gaps, percentage sizes, intrinsic sizing, replaced elements, and print
fragmentation.

Only close a checklist row after its named test and validation command pass.
Do not change a row to `[x]` from intent.

Completion condition: every assigned case has an agreed status, the plan,
manifest, test name, and evidence agree about what it proves, and remaining
interaction families are explicit.

## Delegation modes

When several agents use this skill, partition the 40 manifest case IDs first.
Assign one mode and a disjoint file set to each agent. A family label such as
"sizing" is not an ownership boundary by itself because cases in one family
may share files.

| Mode | Owns | Returns |
| --- | --- | --- |
| Static input | one or more assigned HTML cases and their generator entries | case IDs, source paths, expected behavior, validator output |
| Layout unit | assigned `internal/layout` tests and only the required production fix | test names, expected geometry, red/green commands, changed files |
| PDF integration | assigned `internal/convert` test or golden fixture | page count, semantic text, geometry proof, focused test output |
| Reference comparison | assigned Chromium-reference cases and comparison notes | browser result, Go result, tolerance, unsupported-feature decision |
| Coverage record | assigned plan and knowledge-base rows | closed rows, evidence links, and remaining interaction families |

Each agent must return a short handoff:

```text
Cases: <ids>
Files: <paths>
Behavior: <one sentence per case>
Proof: <commands and pass results>
Remaining: <explicit gaps>
```

## Whole-inventory completion

The inventory is complete only when all 40 manifest cases have one of these
evidence-backed outcomes:

- completed, with all four layers closed;
- explicitly unsupported, with a source-backed reason, a safe skip or rewrite
  note, and coverage recorded;
- blocked by a named engine gap, with a failing test and an owner.

An agent may not mark an unworked case complete because another case uses the
same CSS property. Representative tests prove a shared branch. They do not
close every case that mentions that property.

## Final verification

After all assigned cases are integrated, run the repository gates in this
order:

```sh
make test
make golden
make claim-scan
make lint
```

The final report must distinguish:

- the 40 case IDs and their outcomes;
- static inputs completed;
- direct layout cases completed;
- PDF, image, or Chromium-reference cases completed;
- print-fragmentation coverage completed;
- interaction families still missing.

Do not call the whole inventory complete when only representative layout tests
have passed.
