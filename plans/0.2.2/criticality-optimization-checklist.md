# 00 — Post-#45/#46 Criticality and Optimization

> **Parent:** `plans/0.2.2/README.md` — follow-up to shipped #31 / #32 / #33
> **Status:** not started (ledger only; no code landed from this review)
> **Estimated effort:** 1.5–2.5 weeks across phases 1–6
> **Baseline commit:** `2a18608794e904884e6fa97bcf23c914ac2ba92c` (merge of PR #44)
> **Reviewed PRs:** [#45](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/45) (PDF 1.7 + PDF/A-3a + PDF/UA-1), [#46](https://github.com/chinmay-sawant/gowkhtmltopdf/pull/46) (PDF 2.0 + PDF/A-4 + PDF/UA-2)
> **Workflow:** [`skills/phase-wise-checklist/SKILLS.md`](../../skills/phase-wise-checklist/SKILLS.md)
> **Constraint:** compliance is **done**. Do not add PDF/A or PDF/UA features. Do not reopen completed 0.2.2 compliance ledgers. Do not rewrite the layout engine.
> **Review wave:** 5 read-only sub-agents (API, PDF writer, convert+tagging, performance, critic). Static analysis only; no `go test` / benches were re-run in this wave.

---

## Overview

After `2a186087` the tree absorbed **14 commits / 2 merged PRs** and **~10.6k Go lines** (50 files). The product split is correct: empty version + empty profile is still unclaimed PDF 1.4; `--pdf-version` is not a claim; `--pdf-profile` is. veraPDF 1.30.2 already passes A-3a+UA-1 and A-4+UA-2 on the committed fixtures.

This ledger is **not** more compliance. It is the criticality + optimization follow-up: confirmed UA wiring bugs, a public settings contract that stores two spellings for one key, leftover default-path tax, and an unmeasured profile path.

Default PDF 1.4 serialization is gated (no XMP, ICC, StructTree, ParentTree, named dests, trailer `/ID`). The remaining tax is extra `Op` width, per-op `Policy().IsPDFUA*()` calls, and always-on early-out stubs.

---

## Executive Summary

| Target | Score | Notes |
|--------|-------|-------|
| **0.2.2 slice (PRs #45 + #46)** | **7.3 / 10** | Honest version≠claim, `WriterPolicy`, veraPDF-backed tagging. Dragged down by confirmed UA wiring bugs, two alias tables, two conflict sentinels, a zombie public error, and no profile benches. |
| **Repo after this work** | **8.3 / 10** | Was **8.6 / 10** (2026-08-12, `reports/critical-golang-architecture-review.md`). Engine locks/ownership unchanged. Capability went up; API honesty and unmeasured UA cost pulled the needle down **0.3**. |

### Weighted slice matrix

| Dimension | Weight | Score | Why |
|-----------|--------|-------|-----|
| Concurrency & correctness | 20% | **7.0** | Writer stays single-goroutine. Default 1.4 header is tested. Confirmed UA defects: `DuplicatePage` drops MCIDs; link fallback assumes the wrong op order; outline `/SD` is sequential index; `SetLinkDestStruct` is never called; each `PaintContext` adds another `/Document`. |
| Test quality | 20% | **8.5** | Strong needles + skippable veraPDF. Weak on identity (same `/Link` owns text MCID **and** OBJR), `--copies`+UA, outline exclusion, and public `ErrConformanceRequiresPDF20`. |
| Performance & allocation | 20% | **7.0** | Heavy work is gated. Leftover default tax is real and **unmeasured**. Profile path is `fmt`/`Join` per `StructElem` and re-flates ICC every document. |
| Maintainability | 15% | **6.8** | `WriterPolicy` is the right type. Then: two alias tables, A-3/A-4 copy-paste in `finalize`, 8-wide `nolint` on `tagging.go:36` and `:208`. |
| API design | 20% | **6.8** | CLI `Set` path is sound. Builder stores raw aliases; conflict check is an exact `"1.4"` compare; public `ErrProfilePDF20Unsupported` cannot fire. |
| Documentation | 5% | **8.2** | Version≠claim is written down. `doc.go` still undersells 2.0; library-api Set-key table omits `pdfprofile`. |

**Slice total:** `0.2×7.0 + 0.2×8.5 + 0.2×7.0 + 0.15×6.8 + 0.2×6.8 + 0.05×8.2 = 7.29` → **7.3 / 10**.

Honest ceiling after this ledger (dotted wkhtmltopdf keys remain the product): **slice ~9.2**, **repo ~8.7**. 10/10 is the wrong goal while `Set("pdfprofile", …)` is the public contract.

### Sub-agent scores (independent slices)

| Agent | Area | /10 |
|-------|------|-----|
| 1 | API / settings / CLI | 5.6 |
| 2 | PDF writer engine / memory | 6.5 |
| 3 | Convert + layout tagging | 6.0 |
| 4 | Performance / allocations | 6.0 |
| 5 | Idioms / architecture / critic | 7.8 slice · 8.4 repo |

The headline 7.3/8.3 is the critic matrix **after** folding Agents 2–3 confirmed HIGH bugs that Agent 5 treated as hypotheses.

### Changed surface (code only, vs `2a186087`)

| Area | Files | What landed |
|------|-------|-------------|
| Public API | `api.go`, `api_test.go`, `doc.go` | `WithPDFVersion` / `WithPDFProfile`, new sentinels |
| Settings / CLI | `internal/settings/*`, `internal/cli/flags.go` | `pdfversion` / `pdfprofile` keys, alias tables |
| Convert | `convert.go`, `hf.go`, `links.go`, `outline.go`, `toc.go`, `pdf_pipeline.go` | `PolicyForGlobal`, HF artifacts, dest/outline wiring |
| Layout | `tagging.go` (new), `paint.go`, `layout.go`, image/inline | HTML → structure tree |
| PDF writer | `pdf.go` (+688), `structure.go`, `policy.go`, `metadata.go`, `icc.go` | finalize, ParentTree, XMP, ICC, dual dests |

---

## What is GOOD (do not undo)

1. **Version ≠ conformance.** Empty version + empty profile is unclaimed PDF 1.4. A profile implies 1.7 or 2.0. Catalog `/Version` is omitted so the file header is the single authority (`internal/pdf/pdf.go:840-843`).
2. **Default serialization is gated.** XMP only when `Version >= PDF17`. ICC only when `IsPDFA3` / `IsPDFA4`. `finalizeStructure` and `buildStructureTree` return immediately unless UA.
3. **`WriterPolicy` + `Validate()`** is a stdlib-shaped config value. `Document.WriteTo` is `io.WriterTo`.
4. **`convert.PolicyForGlobal` is the right adapter.** `settings` must not import `pdf`; `pdf` stays a leaf. Do not move this function out of convert.
5. **Structure implementation is spec-literate.** MCR for multi-page elems, ParentTree for MCIDs **and** annots, UA-2 `/NS` + `/ListNumbering` + dual `/D`+`/SD`, HF `/Artifact /Pagination`.
6. **veraPDF evidence exists** (1.30.2, fixtures 21/56, `-f 3a`/`ua1`/`4`/`ua2`). Fonts already have the Arlington `/FontName` == `/BaseFont` fix.

---

## What is BAD (this ledger)

Confirmed, not folklore:

| ID | Sev | Defect |
|----|-----|--------|
| C1 | HIGH | `DuplicatePage` clones BDC/MCID bytes but drops `page.mcids` and never appends `contentRef`s. `--copies` + UA is an untagged clone. |
| C2 | HIGH | Link fallback assumes `[OpLinkURI, OpText]`; layout emits `[OpText, OpLinkURI]`. Text MCID and annot OBJR land on two `/Link`s. |
| C3 | HIGH | Outline `/SD` binds `HeadingStructElems()[i++]`. `--exclude-from-outline` desyncs bookmarks from headings. |
| C4 | HIGH | `Page.SetLinkDestStruct` is never called. UA-2 `#id` dests fall back to the first MCID on the dest page. |
| C5 | HIGH | Every `PaintContext` does `root.NewChild(Document)`. Multi-object / TOC grow extra `/Document` children. |
| C6 | HIGH | `WithPDFVersion(" 1.4 ").WithPDFProfile("a3a-ua1")` bypasses the exact-string conflict check and emits PDF 1.7. |
| A1 | HIGH | Builder `Get("pdfprofile")` returns `"a3a-ua1"`; `Set`/`WithSetting` returns `"PDF/A-3a+PDF/UA-1"`. Tests lock the split in. |
| A2 | HIGH | Alias tables + `Profile*` constants are duplicated in `settings` and `pdf`; convert re-classifies the same tokens. |
| A3 | HIGH | `convert.ErrProfileRequiresPDF17` and `pdf.ErrConformanceRequiresPDF17` are different `errors.New` values. |
| A4 | HIGH | Public `ErrProfilePDF20Unsupported` is documented as current and is never returned. |
| P1 | HIGH | Per-op `Policy().IsPDFUA1() \|\| IsPDFUA2()` on the paint hot path (default + UA). |
| P2 | HIGH | `CanonicalProfile` re-parses on every `IsPDF*` call. |
| P3 | HIGH | ICC profile bytes are rebuilt and Flate-compressed every A-3/A-4 document. |
| P4 | HIGH | UA serialize is `fmt.Sprintf` + `strings.Join` per `StructElem` / ParentTree row. |
| P5 | HIGH | No bench covers `a3a-ua1` or `a4-ua2`. Default-path delta vs Snapshot A is unknown. |

---

## Out of scope

- New PDF/A or PDF/UA flavours, extra structure roles, extra XMP claims.
- Reopening `plans/0.2.2/pdf-1.7-plan/`, `pdf-1.7-compliance-plan/`, `pdf-2.0-plan/` completed rows.
- Layout-engine rewrite, pagination math, glyph advances, subset cmap bytes.
- Making the dotted wkhtmltopdf settings table “fully typed.” Strings stay the product; this ledger only makes `Get`/`Set`/builder agree.
- A new `internal/tagging` package. Layout already imported `pdf` for `Font`. Extract helpers **in** `layout`.
- Moving `PolicyForGlobal` out of convert.

---

## Phase 1 — Correctness of the shipped contract

> **Status:** not started
> **Estimated effort:** 3–4 days
> **Depends on:** nothing (compliance already shipped)
> **Unblocks:** phases 2–6
> **Gate:** `make lint` + `make test` after the phase; skippable veraPDF if C1–C5 land

Fix silent UA wiring and the builder hole that can emit the wrong version. No new claims.

### 1.1 UA copies (`DuplicatePage`)

- [ ] `internal/pdf/pdf.go` `DuplicatePage` (~367): copy `src.mcids` onto the clone and append `contentRef{page: clone, mcid: i}` on each owning `StructElem`. Leave cloned `annotRef = 0` so finalize allocates a new annot + OBJR.
- [ ] Test: UA document + `DuplicatePage` asserts clone `/StructParents`, ParentTree length, and MCR `/Pg` for the clone. Default 1.4 copy bytes must not change (no `/StructParents` on unclaimed copies).
- [ ] Proof: `go test ./internal/pdf -run 'DuplicatePage|Copies' -count=1` and a convert test with `--copies 2 --pdf-profile ua1` (or `a3a-ua1`).

### 1.2 Link MCID + OBJR identity

- [ ] `internal/layout/tagging.go:64-78`: if `opMap[i]` already has a `/Link` from the `<a>` walk, reuse it. Else look at `i-1` when it is `OpText`. Do not `NewChild` a second Link for `[OpText, OpLinkURI]` (`inline_paint.go:311-327`).
- [ ] Map `OpLinkURI` in the walker `<a>` box range (same `StructLink` as the text run).
- [ ] Test: one `StructElem` owns the text MCID **and** the annot OBJR. Phase-6 “tokens exist” is not enough (`internal/convert/phase6_test.go:396-433`).
- [ ] Proof: `go test ./internal/layout ./internal/convert ./internal/pdf -run 'Link|OBJR|Tagging' -count=1`.

### 1.3 Outline `/SD` identity

- [ ] `internal/convert/outline.go:223-247`: stop binding `headingElems[headingIdx++]`. Match by `(DocPage, Y, X)` or the layout location / synthetic anchor already on `outline.Heading`. Skip assignment when there is no match.
- [ ] Test: `--exclude-from-outline` (or `IncludeInOutline=false`) drops one heading; remaining outline `/SE` / `/SD` stay on the surviving headings.
- [ ] Proof: new convert test fails on index drift, then passes. Re-run skippable veraPDF `-f ua2` on fixture-21 if this row lands.

### 1.4 UA-2 dest structure

- [ ] After `AddLinkDest` (`internal/convert/links.go`), resolve dest id → heading/paragraph `StructElem` and call `Page.SetLinkDestStruct` (`internal/pdf/pdf.go:437`). Today the identifier has **zero** call sites.
- [ ] Test: multi-page `#id` / TOC dest: `/SD` targets the dest heading, not `firstPageStructElem` (`pdf.go:1086-1118`).
- [ ] Proof: `go test ./internal/convert ./internal/pdf -run 'Dest|SD|Link' -count=1`.

### 1.5 Single `/Document` child

- [ ] `internal/layout/tagging.go:42-48`: reuse `root.Children[0]` when it is already `StructDocument`. Do not `NewChild(Document)` on every `PaintContext` (second body object, TOC paint at `convert.go:637`, `toc.go:168`).
- [ ] Test: two body objects + TOC → exactly one `/Document` under `StructTreeRoot`.
- [ ] Proof: `go test ./internal/layout ./internal/convert -run 'Document|Structure|TOC' -count=1`.

### 1.6 Builder version/profile conflict

- [ ] `internal/convert/convert.go:229-258` `compliancePolicy`: `ParsePDFVersion` the stored string, then compare the **parsed** token to the implied version. `" 1.4 "` + `a3a-ua1` must return `ErrConformanceRequiresPDF17`, not emit PDF 1.7.
- [ ] Distinguish omitted version (imply) from explicit 1.4. Empty stays “imply”; explicit 1.4 + 1.7 profile stays a hard error.
- [ ] Test: `WithPDFVersion(" 1.4 ").WithPDFProfile("a3a-ua1")` and `" 1.7 "` + `a4`. Public `api_test.go` must assert `ErrConformanceRequiresPDF20` (`1.7`+`a4`, `1.4`+`ua2`).
- [ ] Proof: `go test ./internal/convert . -run 'PDFProfile|PolicyForGlobal|Conformance' -count=1`.

### 1.7 HF links must not join the body tree

- [ ] `internal/convert/hf.go` + `links.go:109`: do not `attachLinkStructElem` for header/footer bands (already wrapped in `/Artifact << /Type /Pagination >>`). Keep the visual artifact wrap (`hf.go:169-216`, `paint.go:658-662`).
- [ ] Test: `--header-html` with an `<a>` under UA → no extra Document `/Link` / OBJR for the chrome link.
- [ ] Proof: convert test + existing dual-profile HF artifact needle still green.

---

## Phase 2 — API and data contracts (one source of truth)

> **Status:** not started
> **Estimated effort:** 2–3 days
> **Depends on:** 1.6 (conflict checker uses parsed tokens)
> **Unblocks:** phase 6 docs rows

The string CLI stays. The bug is **two write paths, two tables, two sentinels**.

### 2.1 Canonical builder

- [ ] `internal/settings/options.go:71-80`: `WithPDFVersion` / `WithPDFProfile` store `ParsePDFVersion` / `ParsePDFProfile` canonical tokens (or fail). Invalid raw is a `Build`/`Set` error, not a second spelling.
- [ ] Flip `api_test.go:1810-1824` and `options_test.go:15-29`: builder `Get("pdfprofile")` after `"a3a-ua1"` equals `PDF/A-3a+PDF/UA-1` (same as `Set` / `WithSetting`).
- [ ] Proof: `go test . ./internal/settings -run 'PDFProfile|PDFVersion|Options' -count=1`.

### 2.2 One alias table

- [ ] Keep the alias switch in **one** function. Prefer `settings.ParsePDFProfile` **or** a tiny `internal/pdfprofile` with no `pdf`/`layout` imports. `WriterPolicy.CanonicalProfile` (`internal/pdf/policy.go:100-154`) calls it.
- [ ] Delete the duplicate `Profile*` constant block from one package (`settings.go:15-34` vs `policy.go:65-84`).
- [ ] `convert.go:198-208`: delete `isPDF17ComplianceProfile` / `isPDF20ComplianceProfile`; classify via the shared tokens / `WriterPolicy.Validate`.
- [ ] Add `TestProfileAliasParity`: every alias both tables used to accept maps to the same canonical string.
- [ ] Proof: `rg 'ProfilePDFA3a =' --type go` has one definition; `rg 'a3a-ua1' internal/settings internal/pdf` is one switch.

### 2.3 One conflict sentinel

- [ ] Delete `convert.ErrProfileRequiresPDF17` / `ErrProfileRequiresPDF20` as independent `errors.New` (`convert.go:190-196`). Return `pdf.ErrConformanceRequiresPDF17` / `pdf.ErrConformanceRequiresPDF20`.
- [ ] Public `api.go:85-92` aliases the **pdf** values once. Drop public `ErrProfileRequiresPDF*` (or keep as aliases of the same var, not a second name in docs).
- [ ] Proof: `errors.Is(PolicyForGlobal(1.4+a3a), pdf.ErrConformanceRequiresPDF17)` and public `ErrConformanceRequiresPDF17` match. `errors.Is` across `NewDocumentWithPolicy` and `ValidatePDF` is true.

### 2.4 Kill the zombie

- [ ] `api.go:81-82` + `settings.go:46-49`: stop advertising `ErrProfilePDF20Unsupported` as current behavior. Unexport, or comment “never returned; retained for source compatibility” **and** add a test that `a4` / `ua2` / `a4-ua2` do **not** match it.
- [ ] Proof: `rg 'return .*ErrProfilePDF20Unsupported' --type go` is empty in production code.

### 2.5 Public write failures

- [ ] Alias `pdf.ErrTitleRequired` and `pdf.ErrPDFUAMissingAlt` on the public API, **or** fail them in `ValidatePDF` for UA profiles so embedders do not string-match `"pdf: %w"`.
- [ ] Keep one public name for version conflict (`ErrConformanceRequiresPDF17` / `20`).
- [ ] Proof: `go doc` lists the UA sentinels; `ValidatePDF` / `RunPDF` tests use `errors.Is`.

### 2.6 Shrink greedy aliases

- [ ] Documented + ISO spellings stay: `a3a-ua1`, `a3a`, `ua1`, `a4-ua2`, `a4`, `ua2`, `PDF/A-3a`, `PDF/UA-1`, `PDF/A-4+PDF/UA-2`, …
- [ ] Reject `"a3"`, `"ua"`, `"ua2+a"` (and the `"a4+ua"` → A-4+UA-2 surprise). `"a4"` as a profile next to `--page-size A4` must stay documented if kept.
- [ ] Delete the dead second `switch` on original spelling in `ParsePDFProfile` / `CanonicalProfile` (`settings.go:116-129`, `policy.go:138-151`).
- [ ] Proof: table test of rejected tokens; CLI combo-negative `--pdf-version 1.7 --pdf-profile a4`.

### 2.7 Annot setter soup (optional, same phase if cheap)

- [ ] `internal/pdf/structure.go:205-234`: one real implementation (`SetObjRef` or `AddAnnot`); the rest wrappers or deleted.
- [ ] Proof: `rg 'func \(e \*StructElem\) SetAnnot' ` shows one body.

---

## Phase 3 — Measurement and default-path isolation

> **Status:** not started
> **Estimated effort:** 1–2 days
> **Depends on:** nothing (can overlap phase 1)
> **Unblocks:** phase 4 (no optimization without numbers)

Snapshot A (2026-08-09, **before** #45/#46, default unclaimed 1.4, i7-13700HX WSL2):

```
BenchmarkPDFPages/500Pages      873 ms   336 MB   518 K allocs
BenchmarkTemplatePages/500Pages 942 ms   340 MB   569 K allocs
```

Source: `testdata/golden/benchmarks/benchmark-results.txt`. Those benches still do **not** set `PdfProfile`. Current names may be `BenchmarkPDFPages/generic/500Pages`.

### 3.1 Default-path isolation test

- [ ] Convert golden: empty version + empty profile → assert **absence** of `/StructTreeRoot`, `/MarkInfo`, `/MCID`, `/Artifact`, `/Tabs /S` (unless 3.3 keeps `/Tabs` as accepted). Existing 1.7 isolation (`compliance_golden_test.go:349-371`) does not cover default 1.4.
- [ ] Proof: `go test ./internal/convert -run 'Golden|Unclaimed|PDF14' -count=1`.

### 3.2 Add profile benches **before** claiming wins

- [ ] Extend `internal/convert` bench matrix: same HTML as Snapshot A at 10 / 50 / 500 pages for (a) default, (b) `PdfProfile=PDF/A-3a+PDF/UA-1` (non-empty title + img alts), (c) `PdfProfile=PDF/A-4+PDF/UA-2`.
- [ ] Add `internal/pdf` `BenchmarkWrite50Pages/{pdf14,pdf17,a3a-ua1,a4-ua2}` to isolate writer/ICC/XMP/structure from HTML/layout.
- [ ] Record `ns/op`, `B/op`, `allocs/op`, and output bytes in this ledger (release measurement, not a one-off laptop claim without command + machine).
- [ ] Commands:

```sh
go test ./internal/convert -run '^$' \
  -bench '^BenchmarkPDFPages/(generic|a3a-ua1|a4-ua2)/(10|50|500)Pages$' \
  -benchmem -benchtime=1x -count=3

go test ./internal/pdf -run '^$' \
  -bench '^BenchmarkWrite50Pages' -benchmem -count=10
```

- [ ] Proof: numbers pasted under “Evidence” below. No optimization in this row.

### 3.3 Default-path `/Tabs /S` leak

- [ ] `internal/pdf/pdf.go:985-994`: emit `/Tabs /S` only when `IsPDFUA1() || IsPDFUA2()`. Today every linked 1.4 page gets a UA tab-order key without a structure tree.
- [ ] If a default-1.4 golden pins `/Tabs /S`, update the golden **or** mark this row `[~]` with the pin and a reason. Semantic no-op for 1.4 viewers.
- [ ] Proof: 1.4 link fixture has no `/Tabs`; UA fixture still has it.

### 3.4 Evidence (fill after 3.2)

| Workload | Machine | Command | ns/op | B/op | allocs/op | bytes |
|----------|---------|---------|-------|------|-----------|-------|
| default 500 | | | | | | |
| a3a-ua1 500 | | | | | | |
| a4-ua2 500 | | | | | | |
| Write50 pdf14 | | | | | | |
| Write50 a3a-ua1 | | | | | | |
| Write50 a4-ua2 | | | | | | |

Do not mark phase 4 rows `[x]` against folklore. Compare default 500 to Snapshot A on the **same** machine or say the machines differ.

---

## Phase 4 — Performance (only after Phase 3 numbers)

> **Status:** not started
> **Estimated effort:** 2–3 days
> **Depends on:** phase 3.2 numbers
> **Unblocks:** phase 5 (split tagging after hoist)

Refuse: glyph advances, subset cmap bytes, paint coordinates, compliance dictionary **key sets**. Those change output.

### 4.1 Hoist UA / A flags off the paint loop

- [ ] Cache `isUA` / `isUA2` / `isPDFA3` / `isPDFA4` on `Document` (or `WriterPolicy` after `Validate` / `NewDocumentWithPolicy`). `IsPDF*` become field reads (`policy.go:156-187`).
- [ ] `pagePainter.isUA` set once in `paintPages`. Same bool in `paintBandOp` (`paint.go:430, 658`), HF (`hf.go:169, 491`), `attachLinkStructElem` (`links.go:109`).
- [ ] Skip `buildStructureTree` / `HeadingStructElems` / `attachLinkStructElem` with one `if !doc.IsUA()`.
- [ ] Gate (default must not regress Snapshot A **shape**):

```sh
go test ./internal/convert -run '^$' \
  -bench '^Benchmark(PDFPages|TemplatePages)/generic/500Pages$' \
  -benchmem -benchtime=1x -count=3
```

- [ ] If the delta is lost in noise, mark `[~]` with the number. Do not leave the per-op `Policy()` call “because it is cheap” without a measurement.

### 4.2 Precompute Flate ICC

- [ ] `sync.OnceValue` (or precomputed `[]byte`) for flated sRGB + gray (`icc.go`, used at `pdf.go:698-724`). `finalize` copies the cached slice.
- [ ] One `embedICC(doc, n, alt, raw)` helper for A-3 and A-4. A-4 additionally embeds gray. Output-identical if the cached slice is the exact `flateBytes` result.
- [ ] Gate: `go test ./internal/pdf -run '^$' -bench '^BenchmarkWrite50Pages/(a3a-ua1|a4-ua2)$' -benchmem -count=10`.

### 4.3 Structure / ParentTree builders

- [ ] `formatStructKids` / `serializeStructElem` / `buildParentTree` (`structure.go:431-723`): `strings.Builder` / `[]byte` + `strconv.AppendInt`. No `fmt.Sprintf` per MCR/OBJR/page row. Same token spacing (veraPDF + goldens).
- [ ] `pruneEmptyStructElems` (`structure.go:353`): in-place filter; reuse `elem.Kids` when nothing is dropped.
- [ ] `computeTrailerID` (`pdf.go:538`): `hasher.Write` of the same tokens; **do not** add stream bytes (that changes `/ID` goldens).
- [ ] `encodeUTF8Hex` / `encodeUTF16BEHex` (`pdf.go:1403-1440`): nibble table / existing `appendHex4`. Uppercase zero-padded hex must stay.
- [ ] Gate: a3a-ua1 / a4-ua2 50- and 500-page benches from 3.2; `make test` for `/ID` and structure needles.

### 4.4 Tagging storage

- [ ] Delete `opTagInfo` / `opMap` heap objects (`tagging.go:48, 72-77`). Stamp `Op.StructElem` in the walker; paint reads the field. Heading BDC tag is `elem.Tag`.
- [ ] Do **not** remove `Op.StructElem` unless a side table is proven smaller on the UA path. Default-path width stays an accepted tax if 4.1 is hoisted.
- [ ] Gate: default 500-page bench + UA benches.

### 4.5 TOC scratch + font finalize

- [ ] `toc.go:162-172`: if UA benches show `paintCount` is material, paint the scratch doc with the same `Version` and an **empty** profile so estimation does not build a structure tree. Page counts must not change. Else `[~]` with the number.
- [ ] `unionFontRunes` (`pdf.go:773-824`): record `name → *Font` in `recordFontRune` / `UseEmbeddedFont`. Drop the O(fonts × pages) scan. Keep Type0 off the ASCII `TextShow` path (`content.go:383-397`).
- [ ] Gate: default 500-page bench (font row); UA 50-page (TOC row).

### 4.6 UA-2 dest object count (optional)

- [ ] Inline dest dicts in the name tree (`pdf.go:882-935`) if veraPDF still accepts `/Names [ (D1) << /D … /SD … >> ]`. One fewer object per outline/link dest. Medium golden risk — do not take if 3.2 shows dest objects are noise vs F4.
- [ ] Proof: `a4-ua2` Write50 bench + veraPDF `-f ua2` + `-f 4`.

---

## Phase 5 — Complexity cleanup (same package, no behavior change)

> **Status:** not started
> **Estimated effort:** 2 days
> **Depends on:** phase 1 (correct tagging) + 4.1 (hoisted flags)
> **Unblocks:** phase 6

A PDF `finalize()` may stay branchy. An 8-linter HTML walker may not.

### 5.1 Split `tagging.go`

- [ ] Extract `tagTable`, `tagListItem`, `tagHeading`, `mapSemanticOps`, `associateUnmappedOps` **in `internal/layout`**. Do not invent a tagging package.
- [ ] Map leftover ops **after** recurse (not parent-first full `opStart..opEnd`). Nested `<h*>` / `<a>` / `<ul>` keep ownership. Thead clones (`paint_flow.go:2183-2206`) → Pagination artifacts, not fallback `P`.
- [ ] `case "table"`: set `currentParent = tableElem` even when `b.rows` is empty; do not walk caption twice.
- [ ] Drop the 8-name `nolint` on `tagging.go:36` and `:208` to at most `funlen`/`cyclop` on leftover glue.
- [ ] Proof: `tagging_test.go` + convert structure goldens; `rg 'nolint:.*nilnil' internal/layout/tagging.go` is empty or documented.

### 5.2 Split `finalize` compliance objects

- [ ] Extract `embedMetadata` + `embedOutputIntents` from `Document.finalize` (`pdf.go:658`). Preserve **structure before outlines/annots** (comment at ~738) so `/SD` refs stay valid.
- [ ] Shrink `//nolint:cyclop,funlen` on `finalize`. Do not invent a pipeline type.
- [ ] Proof: 1.4 / 1.7 / 2.0 / dual goldens unchanged.

### 5.3 XMP / comments / nil walks

- [ ] `metadata.go`: `buf.Grow`; skip `xmlEscape` when `!strings.ContainsAny(s, "&<>\"'")`; use cached profile flags. Keep XMP whitespace (goldens compare exact XML).
- [ ] Fix stale package comments that still say A-4 / UA-2 are deferred to #33 (`internal/pdf/doc.go`, `pdf.go` header).
- [ ] Skip nil `Kids` / `Children` in `pruneEmptyStructElems`, `assignStructElemRefs`, `HeadingStructElems`.
- [ ] Proof: `go test ./internal/pdf -count=1`.

### 5.4 Validate once

- [ ] Delete extra `ParsePDF*` + `PolicyForGlobal` preflight in `ConvertTo` / `ValidatePDF` (`api.go:676-690`, `1172-1186`). Call `PolicyForGlobal` once inside `convert.Run`. Delete the one-line `policyForGlobal` wrapper (`convert.go:292-295`).
- [ ] Drop new `nolint:wrapcheck` on this path; wrap `fmt.Errorf("pdfprofile: %w", err)` / `"pdf policy: %w"`. `errors.Is` stays valid.
- [ ] `HasConformanceProfile` uses `CanonicalProfile() != ""` (or delete it if still unused).
- [ ] Proof: public API tests; `rg 'nolint:wrapcheck' api.go internal/convert/convert.go` only pre-existing, documented exceptions.

---

## Phase 6 — Closure gates

> **Status:** not started
> **Estimated effort:** 0.5–1 day after 1–5
> **Depends on:** phases 1–5 (unchecked rows stay unchecked)

### 6.1 Required checks

- [ ] `make lint` → record version + outcome here. Leave the phase unchecked if it fails.
- [ ] `make test` → record outcome. Leave unchecked if it fails.
- [ ] `go test -race ./internal/pdf ./internal/convert ./internal/layout` if C1 / C2 / 4.2 (`sync.Once`) landed.
- [ ] Skippable veraPDF only if phase 1 structure rows landed:

```sh
./compliance/run_verapdf.sh --both \
  output/pdf-1.7-compliance/fixture-21-detailed-report.pdf \
  output/pdf-1.7-compliance/fixture-56-architecture-diagram.pdf \
  output/pdf-2.0-compliance/fixture-21-detailed-report.pdf \
  output/pdf-2.0-compliance/fixture-56-architecture-diagram.pdf
```

  If `verapdf` is absent, record **skip**. Do not invent a PASS.

### 6.2 Docs (contract, not marketing)

- [ ] `documentation/library-api.md` Set-key table: add `pdfprofile` (canonical values, implied version, conflict sentinels). Today `pdfversion` is listed and `pdfprofile` is not.
- [ ] `doc.go`: name 1.4 / 1.7 / 2.0 and the six profiles. Drop “opt-in 1.7 + A-3a/UA-1 only” wording.
- [ ] CLI help closed sets: `--pdf-version 1.4|1.7|2.0`, `--pdf-profile a3a-ua1|a3a|ua1|a4-ua2|a4|ua2` (`internal/cli/help.go`).
- [ ] `api.go` comment on `ErrProfilePDF20Unsupported` matches 2.4 (never returned, or gone).

### 6.3 Ledger hygiene

- [ ] Every in-scope row in phases 1–5 is `[x]` with evidence, or `[~]` with reason, owner boundary, and next gate.
- [ ] No new compliance flavour. No layout-engine rewrite. Completed `plans/0.2.2/pdf-*-plan/` rows stay completed.
- [ ] `plans/0.2.2/README.md` lists this ledger’s status.
- [ ] Re-rate this slice with the same matrix. Honest ceiling: **slice ~9.2**, **repo ~8.7**.

### 6.4 Evidence (fill at close)

| Check | Command | Result | Date |
|-------|---------|--------|------|
| lint | `make lint` | | |
| test | `make test` | | |
| race | `go test -race ./internal/pdf ./internal/convert ./internal/layout` | | |
| veraPDF | `./compliance/run_verapdf.sh --both …` | | |
| default 500 bench | see 3.2 | | |
| a3a-ua1 500 bench | see 3.2 | | |
| a4-ua2 500 bench | see 3.2 | | |

---

## Dependencies

| Phase | Depends on | Provides to |
|-------|------------|-------------|
| 1 Correctness | shipped #45/#46 | safe UA copies/links/outline/dests; builder conflict |
| 2 API contracts | 1.6 | one alias table, one sentinel, canonical `Get` |
| 3 Measurement | none (overlap 1) | default isolation + bench numbers |
| 4 Performance | 3.2 numbers | hoist, ICC once, structure builders |
| 5 Complexity | 1 + 4.1 | split tagging / finalize; wrapcheck cleanup |
| 6 Closure | 1–5 | lint/test/veraPDF + docs + re-rate |

```text
Phase 1 (correctness) ──┐
                        ├──► Phase 5 (split tagging) ──► Phase 6 (gates)
Phase 3 (benches) ─► 4 ─┘
Phase 2 (API) ─────────────────────────────────────────► Phase 6 docs
```

---

## Suggested PR split

Each PR independently reviewable. No compliance reopen.

| PR | Title | Phases | Notes |
|----|-------|--------|-------|
| 1 | fix(pdf/ua): copies, link identity, outline /SD, dest struct, single Document | 1.1–1.5, 1.7 | veraPDF on existing fixtures |
| 2 | fix(api): canonical builder + parsed version conflict | 1.6, 2.1, 2.3, 2.4 | public test matrix |
| 3 | refactor(pdf): one profile alias table | 2.2, 2.6, 2.7 | parity test; no byte change |
| 4 | test(perf): default isolation + a3a-ua1/a4-ua2 benches | 3 | numbers only |
| 5 | perf(pdf): hoist flags, ICC once, structure builders | 4 | after PR 4 numbers |
| 6 | refactor(layout): split tagging walker | 5 | after PR 1 |
| 7 | docs+closure | 6 | lint/test recorded |

---

## Devil's advocate (what not to do)

| Temptation | Verdict |
|------------|---------|
| Typed public `PDFVersion` / `PDFProfile` enums | **Nit.** Product is a string CLI. Canonical `Get` (2.1) matters more. |
| New `internal/tagging` package | **Reject.** Layout already imported `pdf`. Cost > benefit. |
| Move `PolicyForGlobal` out of convert | **Reject.** settings ↛ pdf. convert **is** the adapter. |
| Split `finalize()` into a pipeline framework | **Reject.** Extract ICC/XMP helpers (5.2). Do not invent types. |
| Treat stacked `nolint` as a ship blocker | **Reject.** It is a 6.8 maintainability score. veraPDF-passing ugly functions beat a pretty empty tree. |
| Cache `CanonicalProfile` before benches | **Nit unless 3.2 shows it.** Pair with 4.1. |
| Optimize `TextShow` / subset math “while we are here” | **Reject.** Default ASCII path is already simple. Byte-change risk. |
| Reopen #31/#32/#33 ledgers | **Reject.** This file is the follow-up. |

---

## Review method

Five parallel read-only sub-agents, 2026-08-15, base `2a186087`:

1. API / settings / CLI — `/tmp/grok-1000/review-82a480d7-agent1-api.md`
2. PDF writer engine / memory — `/tmp/grok-1000/review-82a480d7-agent2-pdf.md`
3. Convert + layout tagging — `/tmp/grok-1000/review-82a480d7-agent3-convert-tagging.md`
4. Performance / allocations — `/tmp/grok-1000/review-82a480d7-agent4-perf.md`
5. Idioms / architecture / critic — `/tmp/grok-1000/review-82a480d7-agent5-critic.md`

Orchestrator verified C1 (`DuplicatePage` does not copy `mcids`), C2 (`[OpText, OpLinkURI]` vs `i+1` text peek), C3 (index increment), C4 (`SetLinkDestStruct` has no call sites), C6 (exact `glob.PdfVersion == forbidden`), A1 (builder stores raw), A4 (public zombie), P1 (per-op `isUA` in `paint.go:430`), and `/Tabs /S` (ungated at `pdf.go:994`) against current source before writing this ledger.

Rows stay `[ ]` until the matching source/test/bench check succeeds. Successful execution is not benchmark proof.
