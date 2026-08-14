# fix-log/fix-css.md — wave-1 remediation of `internal/css`

Agent: fix-css (owns `internal/css/*`). Date: 2026-08-07.
Source of truth: `phases/phase-03-css-engine.md` anchors p3-02, p3-03, p3-04, p3-06.

## Per-CID status

### P3-06 — One rule-body parser: merge Parse's top loop into parseRuleList — DONE
- Extracted `parseOneRule(selText, block, media string, cq *ContainerQuery, order *int) (Rule, bool)`
  (per Future snippet; owns the order counter and @container copy) and
  `skipAtRule(src string) (rest string, err error)` (braced block first, else
  `;`-statement, else discard-all).
- `Parse`'s top-level rule path and `parseRuleList`'s rule path now both loop on
  `parseOneRule`; `@page` / `@keyframes` / unknown at-rules in `Parse` and the
  unknown-at-rule arm in `parseRuleList` all use `skipAtRule`. `@media` /
  `@container` / `@font-face` keep their `takeBlock` (they need the block body).
- Empty-prelude policy kept as today: `Parse` retains its separate
  `selText == ""` check (css.go top-level loop).
- Behavior note (degenerate input only): an at-rule with **no** `{` that is
  followed by `;` and more content (`@page; p { color: red }`) previously
  discarded the whole rest of the sheet; `skipAtRule`'s `;`-scan now keeps the
  trailing rules. Real CSS (`@page` requires a block) is unaffected; parse
  outcomes for well-formed input are identical.

### P3-03 — css side: LengthToPt — DONE
- Promoted the existing internal `lengthToPt` (container.go) to exported
  `func LengthToPt(val float64, unit string, basePt float64) (float64, bool)`
  with the exact existing signature/body (px 0.75, pt, in 72, cm 72/2.54, mm
  72/25.4, pc 12, em via basePt, rem 16px-root, ex/ch basePt*0.5; %/vw/vh and
  unknown units → false). Updated the internal call site in media.go
  (`matchesAxis`) and the doc comment.
- Contract note: fix-contract #3 sketches `LengthToPt(v Length, basePt float64)
  float64`, but the phase file (source of truth) and the orchestrator brief both
  specify the existing (val, unit string, basePt) shape; kept that shape to
  avoid breaking layout callers. fix-layout must call
  `css.LengthToPt(val, unit, basePt) (float64, bool)`.
- Tests: `TestLengthToPt` covers all units + case-insensitivity + unsupported
  (%/vw/unknown/empty).

### P3-04 — css side: ResolveCustomProps — DONE
- Added `func ResolveCustomProps(declared, inherited map[string]string)
  map[string]string` per the Future snippet: merged overlay (inherited then
  declared wins), memoized var() chain expansion with cycle stack, using the
  existing `ResolveVar`.
- `ParseColor`'s own var()-fallback path (css.go) kept; added
  `// FIX-REVIEW: P3-04` marker stating it stays until layout's var handling
  via ResolveCustomProps genuinely replaces it.
- Tests: `TestResolveCustomProps` (chain + fallback), `TestResolveCustomPropsInheritedOverlay`
  (declared wins, inherited-only props survive, chain into overlaid prop),
  `TestResolveCustomPropsCycle` (cycle → empty), `TestResolveCustomPropsDeepChain`
  (21-hop chain resolves despite ResolveVar's depth-16 bound — stack walk is the
  stricter policy), `TestResolveCustomPropsSelfReferenceWithFallback`
  (cyclic ref with fallback → fallback).
- Semantic note: mid-string `var()` inside larger values (e.g. `calc(var(--x) * 2)`)
  passes through untouched — same as pre-existing `ResolveVar` whole-value policy.

### P3-02 — css side: ParseSelectors — DONE
- Added exported `func ParseSelectors(s string) ([]Selector, bool)` wrapping
  `parseSelectorListStrict(s, false)` (css.go, near parseSelectorList).
- Tests: `TestParseSelectors` — comma lists parse fully (all parts kept), single
  complex selector, invalid part fails the whole list, empty/whitespace/
  empty-list-item fail, unsupported pseudo-element (`::first-line`) fails.
- convert/outline.go consumption is fix-convert's row.

### P3-01 / P3-05 — N/A (fully internal/layout rows; nothing for fix-css)

### Supplementary note: ContainerNames — DEFERRED (marker only)
- `type ContainerNames []string` with `Matches(name string) bool` would ripple
  into layout (it re-splits the space-joined string at style.go:1098 and
  container.go:19-24), so per the brief it is deferred to avoid cross-package
  churn in wave 1. Added `// FIX-REVIEW: P3-03` marker on
  `ParseContainerNameValue` documenting the deferral.

## Files changed
- `internal/css/css.go` — parseOneRule + skipAtRule; Parse + parseRuleList
  refactor; ParseSelectors; ResolveCustomProps; FIX-REVIEW marker at ParseColor.
- `internal/css/container.go` — lengthToPt → exported LengthToPt; FIX-REVIEW
  marker (ContainerNames deferral).
- `internal/css/media.go` — internal caller updated to LengthToPt.
- `internal/css/css_test.go` — TestLengthToPt, TestResolveCustomProps*
  (4 tests), TestParseSelectors; import fmt added.
- `plans/reviews/improve-codebase/architecture-review-2026-08-07/fix-log/fix-css.md`
  (this file).

## Validation
- `go build ./internal/css/...` — PASS
- `go vet ./internal/css/...` — PASS
- `go test -count=1 ./internal/css/...` — PASS (ok)
- `gofmt -l internal/css/` — clean after formatting css_test.go
- `go build ./...` — only `internal/load` fails
  (`settings.LoadGlobal.Allow/EnableLocalFileAccess`, `LoadPage.InlineHTML/
  InlineBase` undefined), which is fix-html-load-outline's in-flight P2-04/P2-07
  work, not css. internal/layout and internal/convert still build against the
  current css API (they have not yet migrated to the new exports; fix-layout /
  fix-convert land those call-site changes in parallel).

## Remaining markers (deferred intentionally)
1. `css.go` ParseColor var()-fallback path — `// FIX-REVIEW: P3-04` (keep until
   layout var handling via ResolveCustomProps genuinely replaces it; re-evaluate
   in wave 2/3).
2. `container.go` ParseContainerNameValue — `// FIX-REVIEW: P3-03` (ContainerNames
   type deferred; needs layout-side co-change).
