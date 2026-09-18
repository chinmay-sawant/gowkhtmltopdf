# Chromium Flexbox interaction plan

## Status

The Chromium source map and the Go-side inventory scaffold are complete. The
24 layout-unit cases, 15 Chrome-reference cases, and one print case remain
porting candidates.

## Definition of done

- The 40 cases remain tied to exact Chromium source paths.
- Every case has a static HTML input and one named expected behavior.
- Direct layout cases use Go box geometry assertions.
- Print cases use the repository's PDF page and semantic checks.
- Reference cases compare the same input in Chromium and the Go renderer.
- A later coverage check reports which interaction families have no case.

## Evidence boundary

The local `chromium/` directory is a shallow, blob-filtered source checkout.
It contains the Blink Flexbox implementation, the legacy Flexbox tests, and
the external WPT Flexbox tests. This run did not build Chromium.

Four read-only workers inspected separate slices. They found that Chromium
C++ tests cannot be copied into Go because they inspect Blink fragments,
constraint spaces, lifecycle state, and scroll state. Static HTML and CSS
inputs can be reused. JavaScript-generated matrices need fixed cases. Reftests
need an explicit reference comparison.

## Case allocation

| Go target | Count | First owner | Gate |
| --- | ---: | --- | --- |
| `layout-unit` | 24 | `internal/layout` | Direct box positions and sizes |
| `chrome-reference` | 15 | Chromium plus the Go CLI | Matching selected geometry or pixels |
| `golden-fixture` | 1 | `internal/convert` | Page bounds and semantic PDF checks |

## Phase 0. Source map and scope

- [x] Use a shallow Chromium checkout with only the Blink Flexbox and test
  paths needed for this study.
- [x] Select exactly 40 high-value cases.
- [x] Record the source path, interaction family, expected result, and Go
  target in `test/Chrome/manifest.json`.
- [x] Keep the checkout and the local decision trail out of the repository.

Proof uses `python3 scripts/generate_chrome_flex_cases.py` and
`go test ./test/Chrome`.

## Phase 1. Porting scaffold

- [x] Generate one static HTML scaffold for each mapped case.
- [x] Validate the count, unique IDs, safe fixture paths, doctype, and source
  paths when the checkout is present.
- [ ] Replace each scaffold with a reviewed case that preserves the Chromium
  behavior without browser-only JavaScript.

The scaffold is an inventory check. It is not an engine pass.

## Phase 2. Direct layout-unit cases

Port the 24 `layout-unit` cases in groups that share one layout decision.

1. Flex basis, grow, shrink, min, and max constraints.
2. Main-axis alignment, cross-axis alignment, and auto margins.
3. Direction, reverse flow, wrapping, and align-content.
4. Gaps, intrinsic sizing, aspect ratio, and replaced elements.
5. Definite percentages and border-box cross sizes.

Each case ends with a direct `internal/layout` geometry assertion. A case does
not move to the next group until its input, expected geometry, and targeted Go
test pass.

## Phase 3. Chrome-reference cases

Port the 15 cases that need a browser reference or a larger rewrite.

- Expand JavaScript-generated matrices into a small set of named static cases.
- Replace `offset*` assertions with the geometry that the Go layout result can
  expose.
- Use Puppeteer to capture Chromium reference output for the same HTML.
- Record unsupported features as explicit skips or rewrite notes. Do not turn
  an unsupported feature into a false pass.

The existing `scripts/puppeteer_print.js` path can provide a PDF reference.
The future reference runner should also collect browser box rectangles for
cases where PDF coordinates are too indirect.

## Phase 4. Print fragmentation case

Port the nested flex fragmentation case as a golden fixture. Pin page count,
ordered text, and the location of the repeated content. Add a raster crop
check only if the structural checks cannot detect a regression.

## Phase 5. Interaction coverage gate

Create a small interaction registry from the manifest categories. The registry
will report missing pairs such as direction plus alignment, alignment plus
automatic sizing, wrapping plus gaps, and aspect ratio plus percentage size.

The registry will not request every mathematical pair of CSS properties. It
will cover shared layout decisions and require a named case for each selected
branch.

## Validation commands

```sh
python3 scripts/generate_chrome_flex_cases.py
go test ./test/Chrome
go test ./internal/layout -run '^TestFlex'
make golden
```

Run `make test`, `make claim-scan`, and `make lint` after the first converted
case group lands. Do not use `go test ./...` without the repository's test
concurrency limits.

## Non-goals

- Building Chromium locally.
- Porting Blink's C++ test harness.
- Claiming full Chrome or WebKit parity.
- Generating tens of thousands of pairwise fixtures.
