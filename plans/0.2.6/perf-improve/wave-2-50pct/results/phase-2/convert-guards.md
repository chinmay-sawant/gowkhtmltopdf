# PERF3-07 / PERF3-08 / PERF3-09 convert guards

Date: 2026-09-11
Owner: internal/convert (layout.Result census fields only)
Pre-change pin: 500-page generic `output-bytes` 1419234, 500 pages;
standalone 671143702 ns/op, 169955488 B/op.

## What changed

### PERF3-07 empty header/footer

`drawHeadersFootersResult` (`internal/convert/hf.go:762`) returns a zero
`hfDrawResult` when `headersFootersHaveContent` is false (`hf.go:173`).
That helper checks global Header/Footer and every plan object's effective
header/footer (object overrides and `@page` margin boxes). On the report
fixture both are empty, so the function does not build date/clock/`idIndex`
and does not walk 500 pages. A canceled context still sets `fatal`, which
keeps `TestDrawHeadersFootersStopsWhenCanceled` passing. If either band
has content, the old per-page path runs unchanged.

### PERF3-08 empty navigation

`layout.Result` gained `HasIDs` and `HasFragmentLinks`. Layout sets
`HasFragmentLinks` in `censusOps` (`internal/layout/layout.go:1114`).
Paint sets `HasIDs` in `populateLocations` (`internal/layout/paint.go:861`).
`collectBodyNavigation` (`internal/convert/links.go:54`) returns a zero
nav without allocating `ids`/`idElems` when `MaxContentX > 0` and both
flags are false. Hand-built results keep `MaxContentX == 0` and still
scan; maps are created only on the first id. Existing
`TestCollectBodyNavigationCopiesOnlyPostPaintLinkData` still passes.

### PERF3-09 measured width

`measuredWidth` (`internal/convert/convert_helpers.go:253`) uses
`measuredWidthFast` when `MaxContentX > 0`. Layout sets `MaxContentX` to
at least `Options.Width`, then raises it for fill/stroke/image ops.
Hand-built results (`MaxContentX == 0`) still walk ops, so over-wide
smart-shrink in `TestLayoutBodyKeepsSmartShrinkAtAReplaceableSeam` is
unchanged. Content that fits within 0.1 pt layouts once
(`TestSmartShrinkNoRelayoutWhenWithinTenthPoint`).

## Commands and outcomes

All commands used `-count=1`. No git commands. No `make lint` / `make test`
/ `make golden` (out of scope for this owner).

```
go test ./internal/convert -count=1 -run 'TestAssemble|TestSmartShrink|TestMeasuredWidth|TestPerf3|TestCollectBodyNavigation' -v
```

PASS. Includes `TestAssembleSkipsEmptyChrome` (empty HF does not call
`Now`), `TestAssembleEmptyChromeProducesPDF`,
`TestAssembleObjectHeaderStillDraws`,
`TestSmartShrinkNoRelayoutWhenWithinTenthPoint` (layoutFn == 1),
`TestPerf3NavWithID`, `TestPerf3NavWithoutIDsConverts`,
`TestCollectBodyNavigationEmptyDoesNotAllocateMaps` (0 allocs).

```
go test ./internal/convert -count=1 -run 'TestTextHeader|TestHTMLHeader|TestInternalLinkDest|TestDrawHeaders|TestLayoutBody|TestCoverNoHeader|TestExternalLinks|TestFromPage' -v
```

PASS. Existing HF, link, cancellation, and smart-shrink tests.

```
go test ./internal/convert -count=1
```

PASS in 11.781s.

```
go test ./internal/convert -run '^$' -bench '^BenchmarkPDFPages$/^generic$/^2Pages$' -benchtime=1x -count=1
```

`34209 output-bytes`, 2.000 pages. Matches the historical 2-page size.

```
go test ./internal/convert -run '^$' -bench '^BenchmarkPDFPages$/^generic$/^500Pages$' -benchtime=1x -count=2 -benchmem
```

Both samples: `1419234 output-bytes`, 500.0 pages.
Cold ~807 ms / 169974600 B/op. Warm ~660 ms / 162597904 B/op.
Pre-change cold pin was 671 ms / 169955488 B/op. Bytes and page count
did not move. Time sits in host noise; this is not the 50% target.

## Not closed

Checklist rows PERF3-07/08/09 stay `[ ]` until `make golden` and the
phase-2 lint/test gate (PERF3-12) run. Glyph cache is PERF3-10, not this
change.
