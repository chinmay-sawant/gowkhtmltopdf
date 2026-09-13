## Summary

Land the 0.2.6 review wave on top of v0.2.5: warm-path time and memory recovery, the architecture-deepening and ponytail cleanup ledgers, 24 previously demoted CSS properties re-implemented with real consumers, element transparency groups, and layout fixes, plus refreshed docs, site, samples, and WASM artifact. The 500-page warm path moves from 1,228.72 ms to 576.33 ms in the same-source phase-6 capture (2.13x), and Snapshot M records 733.48 ms / 163.02 MB B/op on the shared working tree, with B/op below the 0.2.4 baseline. The wave-2 50 percent time target for the public library path is not met and stays open in `plans/0.2.6/perf-improve/wave-2-50pct/phase-wise-checklist.md`. The tree is stamped 0.2.6 for the release cut: version files, CHANGELOG facts, docs/site content, the WASM artifact, and the samples/showcase assets are regenerated on that stamp.

---

## Motivation / context

- Plans:
  - `plans/0.2.6/perf-review/phase-wise-checklist.md` (21/21 rows closed)
  - `plans/0.2.6/perf-improve/phase-wise-checklist.md` (32/32 rows closed)
  - `plans/0.2.6/perf-time/phase-wise-checklist.md` (35/35 rows closed)
  - `plans/0.2.6/review/architecture-deepening-2026-09-10.md` (ARC-24..45)
  - `plans/0.2.6/review/memory-profiling-2026-09-10.md` (IMPROV-01..13)
  - `plans/0.2.6/ponytail/0.2.6-ponytail-audit.md` (40 rows plus 5 gates)
  - `plans/0.2.6/perf-improve/wave-2-50pct/phase-wise-checklist.md` (1/43 rows closed, still open)
- `VERSION` reads 0.2.6 on this branch (release bump); the tag and GitHub Release are cut separately.
- Issues: see **Related issues**.

---

## Changes

### Performance and memory (`internal/layout`, `internal/pdf`, `internal/imageout`)

- Style resolution memoized with a declared-property mask (`internal/layout/style_cascade.go`); `ResolvedStyle` records interned via generated fingerprint code (`internal/layout/style_intern_gen.go`, generator `scripts/gen-style-intern`).
- Grid border ops batched into `OpGridRun` (`internal/layout/grid_run.go`): 174,000 to 66,500 paint ops in the captured 500-page case; compact `Op` payload via embedded `*opExtra`; O(1) paint-range checks and census-skipped pagination walks.
- PDF writer: retained parallel flate workers (`internal/pdf/flate_parallel.go`), bounded LRU parsed-font cache keyed on path+size+mtime (`internal/pdf/font_file_cache.go`), lazy per-face parsing, page raw buffer released after stream materialization with a sticky finalize error.
- Imageout: direct final-resolution raster threshold, streaming filter-none PNG writer (`internal/imageout/pngfast.go`), strip-window raster reuse, bounded supersample cache, pooled encode buffers, glyph scratch pools.
- Memory profiling IMPROV-01..13: PDF B/op down 52 percent and allocs down 94.5 percent; image 500-tile B/op 94.97 MB to 27.2 MB; JPEG B/op down 43 percent (`plans/0.2.6/review/memory-profiling-2026-09-10.md`).
- Snapshot M (committed in `testdata/golden/benchmarks/benchmark-results.txt`) records warm 500-page medians of 695.42 ms internal / 698.79 ms public library / 0.70 s CLI with B/op at or below target. The 615 ms warm, 620 ms public, and 0.66 s CLI time lines were not reproduced in that capture; host drift is the recorded difference. That is a recorded miss, not a claim.
- Post-review strip fix (16c46fb): clipped draws of an oversized scaled image now scale only the visible window, so the 4096x1270 PNG posters stop rebuilding a 65.6 MiB canvas on each 1 MiB strip. fixture-49 1334.1 MB to 43.4 MB B/op and 1395 ms to 151 ms; fixture-53 1329.2 MB to 38.2 MB and 1486 ms to 151 ms; PNG corpus 3.65 GB to 0.94 GB B/op and 8023 ms to 4917 ms (62 rasterable of 66 templates, 3 passes, `-benchmem`).
- Post-review JPEG restore (84a5b68): `encodeInto` converts NRGBA canvases to full-resolution YCbCr before `jpeg.Encode`, replacing per-pixel color boxing. JPEG corpus allocs 101,272,253 to 526,832 (-99.48 percent), time 6935 to 5248 ms, B/op 1.549 to 1.437 GB, peak heap 215.0 to 171.9 MiB; PNG and PDF flat within noise. The parity suite now covers 13 shapes, including even-dimension partial-MCU edges that a 4:2:0 preconversion cannot reproduce exactly.

### Layout, CSS, and correctness

- 24 demoted CSS properties re-implemented with real consumers (new files under `internal/layout/`, including `style_text_support_props.go`, `style_containment_props.go`, `style_font_variant_props.go`, `style_image_adjust_props.go`, `style_color_adjust_props.go`, `image_exif.go`). Catalog moves from 328/2/488 to 354 implemented / 0 partial / 464 unsupported (`plans/0.2.6/catalog/coverage-summary.json`).
- Element transparency groups complete `mix-blend-mode` and `isolation` (`internal/layout/blend_group.go`, `internal/layout/paint_groups.go`, `internal/pdf/content.go`), shared with PNG compositing (`internal/imageout/groups.go`).
- Fixes: size-contained subtrees fold to one intrinsic placeholder in auto table cell measure (`internal/layout/layout_measure.go`); collapsed grid segments move with floated tables (`internal/layout/layout_flow.go`); transformed rails are skipped when collecting page-break seals (`internal/layout/paint_pagination_seal.go`); upright vertical text, CSS-wide keyword handling in containers, and sub/sup shift direction (`inline_vertical_writing.go`, `style_container_props.go`, `inline_vertical_align.go`).

### Architecture and cleanup

- Architecture deepening ARC-24..45: per-copy link destination remapping, LRU-capped sibling cache, imageout zoom/media validation gates, shared layout seams, one border-box height path. fixture-56 envelope moved from 20 to 21 pages, verified by bisect.
- Ponytail cleanup: removed dead APIs and paths including the deprecated `load.NewLoader`, `settings.ColorMode`/`ParseColorMode`, the benchmark islands path, and the process-global CSS sibling cache. Style memoization was restored after its removal measured a 1.89x CPU regression (`plans/0.2.6/ponytail/0.2.6-ponytail-audit.md`).
- Lint and tooling: file-size gate added (`scripts/check-file-size.sh`, `scripts/file-size-allowlist.txt`) and wired into `make lint`; `nolint`, wsl, and lll findings cleared; `skills/diff-verify/` added.

### Tests and benchmarks

- Root benchmarks now validate their own output before reporting (`document_bench_test.go`): PDF header, exact page count, ordered needles, PNG decode, tile counts, opacity.
- `document_bench_validate_test.go` proves the validators accept real output and reject specific corruptions.
- `document_perf3_pin_test.go` pins the 500-page public benchmark at 1,420,537 output bytes, 500 pages, and ordered needles.
- `TestWriteBenchmarkHTML` is opt-in for `scripts/bench-performance-recovery.sh`.

### Golden corpus, samples, and WASM

- fixture-63 page-level demos added (`testdata/golden/fixture-63-page-level-demos.html`, 7 pages); fixture-61/62 updated; envelopes updated in `internal/convert/golden_test.go`.
- `output/` PDF/PNG samples regenerated with the 0.2.6 stamp (`make samples`, including the live Wikipedia smoke); showcase assets re-rasterized to 223 PNGs plus WebP thumbs (`make screenshots`).
- WASM rebuilt from 29,299,445 to 30,027,708 bytes with `-X main.wasmVersion=0.2.6`; `make wasm-test` exit 0 (contract tests, frontend lint/build/tests, Chrome live-demo smoke for PDF, PNG, JPEG).
- Issue dossier re-verified against 0.2.6: 547 implemented / 339 partial / 443 not-implemented out of 1,329 verdicts.

### Docs and site

- `documentation/benchmarks.md` added; `performance.md` re-based to the 2026-09-12 capture; architecture pages, `cli.md`, `compatibility-matrix.md`, `fonts.md`, and `samples.md` refreshed.
- `frontend/`: content refreshed for 0.2.6, Getting Started moved to `/documentation/getting-started`, compatibility table sorted by support tier, landing hero reworked, benchmarks page consolidated. `docs/` is the generated site rebuild.
- README performance table re-based to 2026-09-12: 562 ms vs wkhtmltopdf 1.760 s at 500 pages (3.13x), lower RSS at every measured size.
- 0.2.6 release stamp: `VERSION`, `internal/cli/help.go`, `bindings/c/include/gowkhtmltopdf.h`, `bindings/python/pyproject.toml` + `src/gowkhtmltopdf/__init__.py` + the binding test, `README.md`, `doc.go`, `documentation/*`, and the frontend content/palette now read 0.2.6; `make check-versions` reports aligned. Dated benchmark captures (`page-performance.json`, `benchmarks.js`, `BenchmarksPage.jsx`) keep their measured 0.2.5 labels.
- `CHANGELOG.md` 0.2.6 facts corrected to the generated catalog: 354 implemented / 0 partial / 464 unsupported (`plans/0.2.6/catalog/coverage-summary.json`), 63-fixture corpus total, and the CSS `font-feature-settings`/`font-kerning`/`font-variant-caps` wiring bullet removed (the 2026-09-12 audit keeps those three names demoted; `internal/layout/style_font_variant_props_test.go`).
- `docs/` rebuilt after the WASM and showcase refresh (`npm --prefix frontend run build`; frontend smoke tests green).

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Warm 500p same-source 1,228.72 to 576.33 ms (2.13x, Snapshot M note). Snapshot M shared-tree medians: internal 695.42 ms, public library 698.79 ms, CLI 0.70 s. External 2026-09-12 capture: 14 ms vs 260 ms at 2 pages (18.50x), 562 ms vs 1.760 s at 500 pages (3.13x). Public 620 ms, CLI 0.66 s, and wave-2 50 percent targets recorded as not met. |
| **Memory** | Internal 500p B/op 321.10 MB (Snapshot K) to 234.92 MB (L) to 163.02 MB (M, warm matrix); image 500-tile B/op 94.97 MB (K) to 27.21 MB, and 26.41 MB to 10.59 MB in the wave-2 capture. PDF allocs 21.72M to 1.19M (down 94.5 percent). B/op is cumulative allocation traffic, not peak RSS. |
| **Behavior / correctness** | Table, float, page-break, vertical-text, and containment fixes listed above; `mix-blend-mode`/`isolation` now have real group semantics; fixture-63 added; fixture-56 page count moved 20 to 21. |
| **API / CLI** | No public Go API changes and no CLI flag changes (`cmd/` untouched); the stamped `--version` output now reports 0.2.6. Internal-only removals and signature changes (listed under Breaking changes). Library `Validate` now accepts negative top/bottom margins as the auto header/footer sentinel, matching CLI behavior. |
| **Dependencies** | None added or removed; still exactly the two allowlisted direct modules. |
| **Binary size / build time** | WASM artifact grows by 728,263 bytes (29,299,445 to 30,027,708). `make lint` now also runs the file-size gate. Go binaries unchanged in build shape. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None for the public Go API and CLI | - |
| `Document.Validate` accepts negative top/bottom margins (auto header/footer sentinel) | Callers that relied on `ErrInvalidMargin` for negative top/bottom must validate those fields themselves; negative left/right and non-finite values still error. |
| Internal-only removals: `load.NewLoader`, `settings.ColorMode`/`ParseColorMode`, `settings.StampEmptyHFOverride`, `settings.ApplyImageKeyNormalized`, `convert.NewBenchmarkPDFRequest`, `layout.Plan`/`BenchmarkPlan`, `cli.ParseMode`, benchmark islands path, prepare test seams | No external impact: `internal/` packages are not importable outside this module. |
| Internal signature changes: `pdf.Page.AddLinkDest` takes `*Page`; `layout.Op` payload moved behind `*opExtra`; `imageout.Request.Validate` rejects invalid canvas geometry earlier | Same as above. |

---

## Test plan

- [x] `make test` (all packages green; `internal/convert` fresh at 11.3s)
- [x] `make lint` (golangci-lint v1.64.8, size-check clean with 3 allowlisted files, frontend lint clean)
- [x] `make golden` (full corpus, fresh run, all fixtures pass)
- [x] `make build` (both binaries built and `--version` asserted against `VERSION`; stamp reads 0.2.6)
- [x] `make claim-scan` (clean)
- [x] `make test-race` rerun on the final tree 2026-09-13, exit 0: convert 78.256s, layout 44.174s, pdf 11.623s, imageout 11.344s, load 4.029s. Supersedes the phase-7 closure run, which predated b35dc18 and 0562c20 (log: `plans/0.2.6/perf-time/results/recheck-2026-09-13/gate-make-test-race.log`).
- [x] `make test-race` rerun again after the strip fix (`16c46fb`), exit 0: convert 69.892s, layout 39.890s, pdf 15.092s, imageout 10.305s, load 4.047s (log: `plans/0.2.6/perf-time/results/recheck-2026-09-13/gate-make-test-race-strip-fix.log`).
- [x] `make test-race` rerun after the JPEG restore (`84a5b68`), exit 0: convert 85.029s, layout 48.099s, pdf 18.128s, imageout 13.361s, load 4.022s (log: `plans/0.2.6/perf-time/results/recheck-2026-09-13/gate-make-test-race-ycbcr.log`).
- [ ] `make bench` external captures not rerun; the committed 2026-09-12 capture in `documentation/benchmarks.md` is cited above
- [x] 0.2.6-stamp rerun 2026-09-13, all exit 0: `make check-versions`, `make test`, `make lint`, `make golden`, `make claim-scan`, `make build` (stamp assert 0.2.6), `make python-binding-test` (c-shared `libVersion=0.2.6`, 44 tests OK), `make wasm-test` (browser smoke passed), `make samples`, `make screenshots`.

### Commands

```sh
make test
make lint
make golden
make build
make claim-scan
make test-race
make check-versions
make python-binding-test
make wasm-test
make samples
make screenshots
```

---

## Screenshots / sample output

```
# Snapshot M medians (testdata/golden/benchmarks/benchmark-results.txt)
internal generic PDF 2 pages:      6.08 ms    4,031,712 B/op    4,453 allocs/op
internal generic PDF 500 pages:  695.42 ms  167,865,712 B/op  755,091 allocs/op
public library PDF 2 pages:        6.11 ms    4,048,000 B/op    4,464 allocs/op
public library PDF 500 pages:    698.79 ms  169,650,976 B/op  755,100 allocs/op
public library image 250 tiles:   25.72 ms   14,296,096 B/op    7,204 allocs/op
public library image 500 tiles:   44.27 ms   26,414,016 B/op   13,635 allocs/op
CLI 500 pages: 0.70 s / 147,264 KiB RSS, 1,419,234 PDF bytes

# External CLI capture, 2026-09-12 (documentation/benchmarks.md)
2 pages:    14 ms vs wkhtmltopdf 260 ms     18.50x
500 pages: 562 ms vs wkhtmltopdf 1.760 s   3.13x
```

`make golden` at HEAD:

```
ok  github.com/chinmay-sawant/gowkhtmltopdf/internal/convert  5.379s
```

0.2.6 regeneration, 2026-09-13:

```
make samples:             63 fixture PDFs/PNGs + 1.7/2.0 profile smokes regenerated; live Wikipedia smoke OK
make screenshots:         223 PNG(s) + 223 WebP thumb(s) -> frontend/src/assets/showcase
make wasm-test:           live-demo browser smoke passed for PDF, PNG, JPEG, validation, and mobile layout
make python-binding-test: Ran 44 tests in 0.039s - OK (dist/libgowkhtmltopdf.so stamped 0.2.6)
make build:               bin/gowkhtmltopdf --version -> 0.2.6 (matches VERSION)
```

---

## Related issues

- Closes: none (no open issues or PRs on the repo as of 2026-09-13)
- Relates to #62 (prior 0.2.6 CSS and layout tranche)

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`performance`, `enhancement`, `documentation`)
- [x] Related issues filled with real ticket IDs
- [x] Filled body committed under `plans/PR/pr-0.2.6-review-cleanup-perf.md`

---

## Follow-ups (out of scope)

- `plans/0.2.6/perf-improve/wave-2-50pct/phase-wise-checklist.md` is 1/43 rows closed; the PDF 50 percent time target reads 624.49 ms / 121.02 MB vs 367 ms / 81.5 MB. Track it there.
- Document the `Margin` sentinel (`-1`) on the public struct, not only in error text and docs.
- fixture-63 has no row yet in the `testdata/golden/README.md` corpus table.
- PERF3 byte and dimension pins are toolchain-sensitive (measured on go1.26.4); read the pin failure before blaming layout.
- The wave-2 capture (`11752d3`) ran without `make lint`; the HEAD gates in this PR cover the current tree.

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
- [ ] Diff-stat-by-extension table pasted at the bottom

---

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.css` | 5 | 115 | 154 |
| `.csv` | 3 | 17 | 17 |
| `.go` | 244 | 27808 | 4540 |
| `.h` | 1 | 1 | 1 |
| `.html` | 7 | 1267 | 55 |
| `.js` | 15 | 293 | 198 |
| `.json` | 19 | 2588 | 2220 |
| `.jsx` | 8 | 337 | 393 |
| `.md` | 76 | 10423 | 652 |
| `.mjs` | 1 | 9 | 0 |
| `.pdf` | 74 | Binary | Binary |
| `.png` | 429 | Binary | Binary |
| `.py` | 2 | 2 | 2 |
| `.sh` | 2 | 490 | 0 |
| `.toml` | 1 | 1 | 1 |
| `.txt` | 3 | 407 | 0 |
| `.wasm` | 2 | Binary | Binary |
| `.webp` | 395 | Binary | Binary |
| No extension | 3 | 26 | 6 |
| **Total** | **1290** | **43784** | **8239** |
