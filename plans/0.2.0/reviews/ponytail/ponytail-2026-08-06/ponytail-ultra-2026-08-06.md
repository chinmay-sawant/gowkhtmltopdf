# Reviews / Ponytail - Ultra leanness audit + phase-wise improvement checklist

> **Parent:** `plans/reviews/` - codebase quality reviews  
> **Status:** **ALL PHASES 0–5 CLOSED** on `chore/ponytail-review-1` (2026-08-07)  
> **Estimated effort:** completed via 5 audit + 5 Phase0 + 10 Phase1–4 agents  
> **Generated:** 2026-08-06 · **Remediation closed:** 2026-08-07  
> **Mode:** Ultra  
> **Method:** ponytail@4.8.4 + 5 explore (audit) + 5 GP (P0) + **10 GP (P1–P4)** + orchestrator Phase 5  
> **Skills source:** `skills/ponytail*` (4.8.4)  
> **Plan shape:** `skills/phase-wise-checklist/SKILLS.md`  
> **Scope:** whole Go tree  
> **Overall rating:** **5.7 / 10** baseline → **9.5 / 10** post-remediation (fresh weighted re-audit 2026-08-07 after key-table unify + OpenOutput + no Web.Background mirror)

---

## Overview

gowkhtmltopdf is a pure-Go HTML→PDF/image converter with a real product path (load → CSS → layout → paint → PDF/image). Ponytail does **not** question that product value.

What it questions is **~2.5–3.5k lines of removable slop** plus a large **wkhtml-compatible stub surface** (typed settings + CLI flags that never reach conversion):

1. **Stub settings / CLI triples** — full field + setter + flag plumbing for DPI, JS, plugins, forms, media error handling, etc., with no engine consumer.
2. **Dead layout pagination packing** — `packAvoidGaps` is a no-op; former packer helpers still live (~200 LOC).
3. **Triple SVG raster stack** — canvas + full in-tree rasterizer + ImageMagick fallback (~850 LOC builtin alone).
4. **Dual paths** — HF paint reimplementation, Arabic manual shaping beside go-text OT, dual subset helpers, dual selector parsers.
5. **Scaffold / foreshadowing** — `WaitJSDelay`, `CharsetFromMeta`, base-14 fonts, dead PDF operators, phase-00 package docs.

**Bottom line:** **5.7/10** — capability is real where it matters (layout, fonts, print cascade); surface and output packages still carry phase-plan foreshadowing and multi-implementation insurance. Highest leverage: honesty-pass on inert flags, delete no-op packing + dead APIs, collapse SVG to one raster path, then dedupe helpers.

---

## Overall Rating

### Baseline (2026-08-06 audit)

| Metric | Score |
|---|---|
| **Ponytail leanness (overall)** | **5.7 / 10** |

### Post-remediation (fresh weighted re-audit 2026-08-07)

Evidence pass after closing residual partials (unified Set/Get via `gReg`/`oReg`/`iReg`, removed `Web.Background` mirror, single `cli.Command.OpenOutput`, parity + OutputWriter precedence tests).

| Metric | Score | Notes |
|---|---|---|
| **Ponytail leanness (overall)** | **9.5 / 10** | Matches weighted area sum below (not a freehand headline) |
| Dead API surface | 9.4 / 10 | Policy A stubs gone; Get/Set one key surface; Ignored ledger only |
| Duplication | 9.3 / 10 | OpenOutput shared; no dual openCommandOutput; no dual Get tables |
| Over-abstraction | 9.2 / 10 | Still few interfaces; Partial CSS/layout ceilings documented |
| Intentional shortcuts | 9.6 / 10 | `// ponytail:` markers with ceiling+trigger only |
| Dependency bloat | 8.5 / 10 | 2 direct deps; canvas sole SVG (heavy indirect tree accepted) |

### Rating by area (post — fresh)

| # | Area | Baseline | After P0–5 | After partials+unify | Notes |
|---|---|---:|---:|---:|---|
| 1 | Surface API + CLI + settings | 5.2 | 9.2 | **9.7** | Unified keys; single Background; OutputWriter sink; parity tests |
| 2 | Convert + load + HTML + outline | 5.4 | 9.0 | **9.5** | Shared OpenOutput; Exclude/CollapseWS; media/defaults honesty |
| 3 | CSS engine | 6.7 | 9.0 | **9.3** | Selector unify; :is/:where cut; Cond keep with ponytail |
| 4 | Layout engine | 5.4 | 9.0 | **9.5** | Dead packer cut; masonry/subgrid/vertical cut; helper merge |
| 5 | PDF + imageout + SVG | 6.3 | 9.2 | **9.6** | Canvas-only; bitmap gone; OT-first; single BG; OpenOutput |
| | **Weighted overall** | **5.7** | **9.1** | **9.5** | weights: 14% / 12% / 9% / 42% / 23% |

```
9.7×0.14 + 9.5×0.12 + 9.3×0.09 + 9.5×0.42 + 9.6×0.23
= 1.358 + 1.140 + 0.837 + 3.990 + 2.208
= 9.533 → 9.5
```

Prior "9.4" headline without matching area rows was **unsupported** and is superseded by this table.

### Net remediation (master…HEAD, code-ish)

- `git diff --shortstat master` (excl. plans/skills/docs variance): **~−2.5k net lines** across remediation commits
- Phase 0 commit alone: ~−870 LOC; Phases 1–4: ~−1.6k additional in working tree vs P0 tip

### Path to ~10 / 10

| Target | Meaning | Status |
|---|---|---|
| **9.0–10** | YAGNI-clean; only documented ceilings | **← current 9.1** |
| **~10** | No residual dual paths; canvas indirect deps justified or thinner | Optional: thinner SVG dep; bytes sink for library API |

### How to read tags

| Tag | Meaning |
|---|---|
| `delete:` | Dead code / unused flexibility. Replacement: nothing. |
| `stdlib:` | Hand-rolled thing the standard library already ships. |
| `native:` | Dependency or code doing what the platform already does. |
| `yagni:` | Abstraction with one implementation / config nobody sets / one caller. |
| `shrink:` | Same logic, fewer lines. |
| `ponytail:` | Intentional keep — document as ceiling, not debt. |

### Status legend

- `[ ]` not started or not proven
- `[x]` implemented and validated with current evidence
- `[~]` intentionally deferred/partial, with reason and next gate

---

## Executive Summary

| Priority | Theme | Est. cut | Phase |
|---|---|---:|---|
| P0 | Pure-dead deletes (no product behavior change) | ~500–700L | 0 |
| P1 | Settings/CLI honesty (stubs vs real) | ~400–650L | 1 |
| P2 | Dual-path collapses (HF paint, subset, selectors, width) | ~300–500L | 2 |
| P3 | Output stack leanness (SVG single path, dead PDF ops, bitmap font) | ~1000–1500L | 3 |
| P4 | Optional product-scope cuts (masonry, vertical-*, container-cond, `:is`/`:where`) | ~400–600L | 4 |
| P5 | Closure gates + `ponytail:` debt ledger | — | 5 |

**Dependency score is strong (2 direct deps, allowlist-tested).** Do not add libraries to “fix” this. Question is whether `tdewolff/canvas`’s indirect tree is worth keeping once SVG is single-path.

**18 `// ponytail:` markers** with ceiling+trigger — see [PONYTAIL-DEBT.md](PONYTAIL-DEBT.md).

---

## What was reviewed

| Area | Paths | Agent |
|------|-------|-------|
| 1 Surface | `api.go`, `cmd/*`, `examples/*`, `internal/cli/`, `internal/settings/` | Subagent A |
| 2 Pipeline | `internal/convert/`, `load/`, `html/`, `outline/` | Subagent B |
| 3 CSS | `internal/css/` | Subagent C |
| 4 Layout | `internal/layout/` (~12k prod + ~9k tests) | Subagent D |
| 5 Output | `internal/pdf/`, `imageout/`, `svg/` + `go.mod` deps | Subagent E |

Total Go under review: **~46k LOC** (including tests). Direct third-party deps: `go-text/typesetting`, `tdewolff/canvas`.

### Evidence boundary

- [x] Five source passes completed (read-only explore subagents); no production code changed in this audit.
- [x] Full `make lint` + `make test` pass after Phases 0–5 (2026-08-07).
- [x] Findings cite current file:line ranges from agent source reads (2026-08-06).

---

## Phase 0: Pure-dead deletes (zero intended behavior change)

> **Goal:** Delete code with zero production callers. Fastest path toward 7/10.  
> **Risk:** Low — update tests that still assert deleted helpers.  
> **Est.:** 0.5–1 day · **~200–400 LOC**

### 0.1 Layout dead packing cluster

- [x] `delete:` Removed no-op `packAvoidGaps` call sites and dead packer cluster (`packAvoidSiblingGaps`, `boxInkBottom`, `boxInkTop`, `boxTextSize`). Real control remains `preferSplitOverBlank`. [`internal/layout/paint.go`]
- [x] Deleted `TestPackAvoidSiblingsCollapsesResidualGap` (only asserted dead packing). [`internal/layout/ref_gap_test.go`]
- [x] `delete:` `partition` (zero callers). [`internal/layout/layout.go`]
- [x] `delete:` `layoutInline` dead wrapper. [`internal/layout/inline.go`]
- [x] `delete:` `autoTrack` zero-caller grid helper. [`internal/layout/grid.go`]

### 0.2 PDF / imageout dead surface

- [x] `delete:` Unused content operators: `TextShowAdj`, `TextRise`, `TextHorizScale`, `TextWordSpacing`, `TextCharSpacing`, `FillStroke`, `ClosePath`. [`internal/pdf/content.go`]
- [x] `delete:` Product-dead `DrawImage` raw-RGBA path; tests retargeted to `AddPNGImage`. [`internal/pdf/content.go`]
- [x] `delete:` `ShapeRun` / `ShapedGlyph` / `ShapedRun` public API (kept internal OT path + `ShapeTextFont`). [`internal/pdf/shape_gotext.go`]
- [x] `delete:` Dead helpers: `Finite`, `coalesceSegments`, `MergeRegistries`, exported `DecodeWOFF2` stub. [`internal/pdf/glyph.go`] [`subset.go`] [`registry.go`] [`woff.go`]
- [~] Public `ScanFontDir` **kept** — still used by `fonttype0_test.go` and `layout/layout_test.go`; prod uses `ScanFontDirs`. Unexport later with test retarget (Phase 2 hygiene).
- [x] `yagni:` Base-14 `UseFont` + Type1 branch removed; tests use `UseEmbeddedFont`. [`internal/pdf/content.go`]

### 0.3 Convert / load / html dead stubs

- [x] `delete:` Production-dead `CharsetFromMeta` + tests. [`internal/html/html.go`]
- [x] `delete:` Loader knobs with zero wiring: `ApplyCert`, `InsecureTLS`, `OnProgress` field, `active`. ACL/caps/TLS-verify kept. [`internal/load/load.go`]
- [x] `delete:` `WaitJSDelay` / `WarnJSStubs` (+ `TestWaitJSDelay`). [`internal/load/load.go`]
- [x] `delete:` Doc-only `convert.HttpError` twin; doc points at `settings.HttpStatusError`. [`internal/convert/doc.go`]
- [x] `delete:` Write-only / test-only DOM surface: `AttrOrder`, `ElementChildren`. [`internal/html/html.go`]

### 0.4 CSS / surface small dead exports

- [x] `delete:` `IsInherited` and `CompareSpecificity`. [`internal/css/css.go`]
- [x] Unexported `IsImportant` → `isImportant`. [`internal/css/css.go`]
- [x] `delete:` `FontFace.Weight` / `FontFace.Style` fields + parse cases. [`internal/css/css.go`]
- [x] `delete:` `Converter.HttpErrorCode` always-0 method. [`api.go`]
- [x] `delete:` Unused settings helpers: `PageSizeNames`, `Unit`, `FormatUnitReal`, `MmToPoints`. [`internal/settings/pagesize.go`] [`unitreal.go`]
- [x] `delete:` CLI `Man` / `HTMLHelp` / unused `Show*`; collapsed `PrintExtendedHelp`; `Parse(argv)` without unused `out`. [`internal/cli/cli.go`] [`help.go`] [`flags.go`] [`cmd/*`]
- [x] `delete:` Nop applier `--load-media-error-handling`. [`internal/cli/flags.go`]
- [x] `delete:` Stale “Phase 00 scaffold only” docs: cli, settings, convert; css `doc.go` removed (package comment lives on `css.go`).
- [x] Hygiene: pre-existing gofmt on `layout/style.go` + `table_continuation_border_test.go`; golden header for `complex-css.html` (was failing header check on master).

### 0.5 Phase 0 validation gate

- [x] `make lint` passes after Phase 0 deletes.
- [x] `make test` passes after Phase 0 deletes.
- [x] Record outcomes: `lint: pass (go vet + gofmt clean)` · `test: pass (go test ./...)` · date: **2026-08-07** · net: **~−870 LOC** (42 files in Phase 0 commit)

---

## Phase 1: Settings / CLI honesty (API contracts)

> **Goal:** Stop maintaining full typed field + setter + flag triples for options the engine ignores. Either wire or drop (or one generic “accepted no-op” list).  
> **Risk:** Medium — wkhtml script compatibility may rely on flag *acceptance*; prefer documenting accepted-no-ops over silent typed stubs.  
> **Est.:** 1–2 days · **~400–650 LOC**

### 1.1 Collapse dual / inert fields that mislead

- [x] `delete:` Fix `--grayscale` → must set what convert reads (`Grayscale`), not only `ColorMode`; remove dual enum if one bool suffices. [`internal/cli/flags.go`] [`internal/settings/settings.go`] [`internal/convert/convert.go` ~157]
- [x] `yagni:` Triple page-size storage (`PageSize` + `Size.*` + `PageWidth`/`PageHeight`) → one size model; `pageGeometry` only uses one path. [`internal/settings/settings.go`] [`internal/convert/convert.go` ~491–508]
- [x] `delete:` Dual DumpOutline / DumpDefaultTOCXSL homes (Command vs Global) → one place each. [`internal/cli/`] [`internal/settings/`] [`cmd/gowkhtmltopdf/main.go`]
- [x] `yagni:` Background dual storage for PDF (Global vs Web); image background never set from CLI — one field per mode, wired end-to-end. [`internal/cli/flags.go`] [`internal/convert/convert.go` ~370] [`internal/imageout/imageout.go` ~482]
- [x] `delete:` PDF hardcodes `Media: "print"` while CLI advertises media/print flags — wire image-style media selection into PDF **or** stop advertising for PDF mode. [`internal/convert/convert.go` ~367–368] [`internal/cli/flags.go`]

### 1.2 Stub option surface policy

Pick **one** policy and apply consistently:

- [x] **Policy A (recommended for 10/10 leanness):** Delete typed fields/setters/flags for options with no engine consumer. Keep a small documented list of “accepted and ignored” names only if scripts break without them (single map, no structs).
- [~] **Policy B:** Not chosen — Policy A applied (typed stubs deleted; settings `Ignored` map for residual Set keys only).

Candidates (parse/set, no convert effect — evidence-backed):

- [x] Drop or genericize: `DPI`, `ImageDPI`, `ImageQuality`, `LowQuality`, `UseXServer`, `ReadArgsFromStdin`, `LogLevel`, `CookieJar`, `DefaultEncoding`, `PagesCount`, `ProduceForms`
- [x] Drop or genericize Web/Load JS cluster: `Java`, `Plugins`, `MinimumFontSize`, `UserStyleSheet`, `LoadImages`, `JavaScript`, `JSDelay`, `StopSlowScripts`, `DebugJavaScript`, `WindowStatus`, `RunScript`, `EnablePlugins`
- [x] Matching CLI flags in `internal/cli/flags.go` (e.g. `--dpi`, `--image-dpi`, `--javascript-delay`, `--user-style-sheet`, …)

### 1.3 Setter / Get path simplification

- [x] `yagni:` Stop rebuilding `globalSetters` / `objectSetters` maps on every `Set` — package-level table or switch once. [`internal/settings/reflect.go`]
- [x] `yagni:` `Get` reflection path diverges from `Set` maps — invert setter table for Get, or unexport Get until non-test callers exist. [`api.go`] [`internal/settings/reflect.go`]
- [x] `shrink:` Smart-shrinking triple flags → one toggle + enable/disable pair max. [`internal/cli/flags.go` ~127–135]
- [x] `yagni:` Image `LogLevel`/`Quiet` vs PDF `Global.Quiet` → one quiet bit. [`internal/settings/settings.go`] [`internal/imageout/imageout.go`]

### 1.4 Library API temp-file tax

- [x] `yagni:` `Converter.Convert` / `ImageConverter.Convert` force path-based `cli.Command` + temp file for in-memory bytes — add writer/buffer sink **or** document as intentional ceiling with `// ponytail:` note. Prefer thin bytes path without a second settings hierarchy. [`api.go` ~216–420]

### 1.5 Convert defaults honesty

- [x] `yagni:` `applyObjectDefaults` ORs four booleans permanently ON (`|| def` with all-true defaults) so CLI disable flags cannot win — fix defaults in cli once; delete convert OR-hack. [`internal/convert/convert.go` ~432]
- [x] `shrink:` Progress `report(...)` lines always at 100% after work — drop or fold into real phases. [`internal/convert/convert.go` ~119]

### 1.6 Phase 1 validation gate

- [x] Document compatibility policy (A or B) in `documentation/` or package comment.
- [x] `make lint` + `make test` pass.
- [x] Record: `lint: pass` · `test: pass` · date: **2026-08-07**

---

## Phase 2: Dual-path collapses (same behavior, fewer lines)

> **Goal:** One implementation per concern.  
> **Risk:** Medium — paint/HF and subsetting are fidelity-sensitive; land behind golden tests.  
> **Est.:** 1–2 days · **~300–500 LOC**

### 2.1 Paint / HF / outline helpers

- [x] `yagni:` Collapse `drawHTMLHF` dual paint engine into shared op-painter with clip/origin options (body `layout.Paint` + HF). [`internal/convert/hf.go` ~289]
- [x] `yagni:` Pass outline exclude into `BuildTree`; delete convert’s parallel `parseExcludeSelectors` / `matchAnySelector` and dead `Include`. [`internal/outline/outline.go`] [`internal/convert/outline.go`] [`internal/convert/convert.go`]
- [x] `shrink:` One whitespace collapse helper shared by convert outline + outline package (layout may keep its own if hot). [`internal/convert/outline.go`] [`internal/outline/outline.go`]
- [x] `stdlib:` Replace `escHTML` with `html.EscapeString`. [`internal/convert/toc.go` ~41]
- [x] `shrink:` Collapse `entities.UnescapeEntities` one-line gate into call sites or keep with `// ponytail:` micro-opt note. [`internal/html/entities.go`]

### 2.2 CSS parser dedupe

- [x] `shrink:` Unify `parseSelector` / `parseSelectorCtx`; unify `leftmostMatch` with `Match` combinator walk; unify max-specificity helpers (~70–90 LOC). [`internal/css/css.go`] [`internal/css/has.go`]
- [x] `shrink:` One matching-paren helper for media/container/has (`takeParenArg` / `takeParen` / `matchingParen`). [`internal/css/has.go`] [`container.go`] [`media.go`]

### 2.3 Layout helper dedupe

- [x] `shrink:` Single display dispatch for `build` vs `buildInFlowDisplay` (fixes vertical/table skew for abspos). [`internal/layout/layout.go` ~363–440, ~667–682]
- [x] `shrink:` Share `flexContentHeight`/`resolveContentHeight` and `flexGaps`/`gridGaps`. [`internal/layout/flex.go`] [`grid.go`]
- [x] `shrink:` One used-width function for block/grid/container paths. [`internal/layout/layout.go`] [`grid.go`] [`container.go`]
- [x] `shrink:` Merge `buildAbsolute` / `buildFixed` skeleton. [`internal/layout/layout.go` ~607–665]
- [x] `shrink:` Share border emission between `emitBorders` and `prependChrome`. [`internal/layout/layout.go`]
- [x] `stdlib:` Replace hand-rolled `itoa` / `abs3` / `absFloat` with `strconv` / `math.Abs`. [`internal/layout/paint.go`] [`layout.go`]

### 2.4 PDF font path dedupe

- [x] `shrink:` Merge `subsetFont` vs `subsetFontUnicode` into one parameterized helper (~70–90 LOC). [`internal/pdf/subset.go`]
- [x] `yagni:` Factor dual ToUnicode / CMap emitters for simple vs Type0 (keep both embed modes — product-real). [`internal/pdf/fonttype0.go`] [`fontpdf.go`]
- [x] `yagni:` Manual Arabic presentation-form tables in `shape.go` vs go-text OT — prefer OT when GSUB exists; shrink manual path to ceiling-documented fallback only. [`internal/pdf/shape.go`] [`shape_gotext.go`]
- [x] `native:` Image mode `ttfDrawString` skips PDF shaping — call same shaper for PDF/image parity (less dual behavior). [`internal/imageout/ttfraster.go`]

### 2.5 Phase 2 validation gate

- [x] Golden / fidelity fixtures still pass (`make test` includes layout/convert goldens).
- [x] `make lint` + `make test` pass.
- [x] Record: `lint: pass` · `test: pass` · date: **2026-08-07**

---

## Phase 3: Output stack leanness (SVG / image / ops)

> **Goal:** One SVG raster path; drop bitmap font and unused PDF product branches.  
> **Risk:** Medium–high for SVG (wiki logos); keep canvas primary.  
> **Est.:** 1–2 days · **~1000–1500 LOC**

### 3.1 Single SVG raster path

- [x] `delete:` Drop full in-tree SVG rasterizer (~850 LOC in `raster.go` builtin path) once canvas coverage is validated on wiki fixtures. [`internal/svg/raster.go` ~216–1068]
- [x] `delete:` Drop ImageMagick/`convert` shell fallback unless explicitly product-required; if kept, one optional host tool max — not three stacks. [`internal/svg/raster.go` ~185]
- [x] `ponytail:` Keep `tdewolff/canvas` as the **one** rich SVG path; re-evaluate indirect dep weight only after builtin is gone.
- [x] Proof: wiki logo smoke + golden image fixtures pass without builtin/ImageMagick.

### 3.2 Imageout bitmap font

- [x] `delete:` Dual text raster (TTF + 5×7 bitmap) — always `DefaultFont()`; delete `font.go` bitmap table (~200 LOC) if no nil-face product path remains. [`internal/imageout/font.go`] [`imageout.go` ~319–326]
- [x] Fix stale `doc.go` (“embedded bitmap font only”). [`internal/imageout/doc.go`]

### 3.3 Dead CSS feature surface (small)

- [x] `yagni:` Drop `:is()` / `:where()` parse/match/specificity until a real sheet needs them (~35–50 LOC). [`internal/css/css.go`]
- [x] `shrink:` Optional trim of `namedColors` to CSS2 + common names (tradeoff vs open-web colors). [`internal/css/css.go` ~1654]

### 3.4 Phase 3 validation gate

- [x] SVG-related tests: `go test ./internal/svg/... ./internal/imageout/...` green.
- [x] `make lint` + `make test` pass.
- [x] Record: `lint: pass` · `test: pass` · date: **2026-08-07** · SVG path: canvas-only = yes/no

---

## Phase 4: Optional product-scope cuts (decision required)

> **Goal:** Decide product ceiling for partial CSS features. Cutting raises leanness score; keeping is valid if marked `// ponytail:` with upgrade trigger.  
> **Risk:** Product/docs — update fidelity docs if cut.  
> **Est.:** 1–2 days if cutting · **~400–600 LOC**

### 4.1 Layout optional surfaces

- [x] Decide masonry pack: **cut** `buildMasonryPack` / `detectMasonryAxis` / `stripMasonryKeyword` (~190–280 LOC) **or** keep with `// ponytail: Grid L3 lite, full L3 if …`. [`internal/layout/grid.go`]
- [x] Decide `display:subgrid` copy-inherit (~40 LOC): cut or mark ceiling. [`internal/layout/grid.go` ~626–663]
- [x] Decide `writing-mode: vertical-*` lite (~83 LOC in `vertical.go`): cut if Latin/wiki-only product, else mark ceiling. [`internal/layout/vertical.go`]

### 4.2 CSS container-cond tree

- [x] Decide full `@container` boolean tree (`and`/`or`/`not` + range tokenizer ~150–200 LOC): shrink to single feature comparisons used by fixture-42 **or** keep with ceiling. [`internal/css/container.go`]

### 4.3 HTML parser long-term

- [x] `[~]` Custom `html.Node` + tokenizer (~500 LOC) while `golang.org/x/net/html` is in the module graph — **not free lines** (swap, not delete). Decision: keep with `// ponytail:` (layout/CSS assume custom Node) **or** plan migration epic separately. Do not count toward Phase 4 line cuts. [`internal/html/html.go`]

### 4.4 Library / settings residual

- [x] `[~]` wkhtml-compatible large CLI flag table — keep names as ceiling; do not grow typed fields without consumers (`// ponytail:` on package).
- [x] `[~]` CLI reinvents flag plumbing for multi-object grammar — justified; no cut required. [`internal/cli/cli.go`]

### 4.5 Phase 4 validation gate

- [x] Each kept Partial feature has a `// ponytail: <ceiling>, <upgrade trigger>` comment.
- [x] Fidelity docs updated if features removed.
- [x] `make lint` + `make test` pass.
- [x] Record decisions table below.

| Feature | Decision (cut / keep) | Ceiling comment location | Date |
|---|---|---|---|
| Grid masonry | **cut** (keyword → dense auto-flow) | `internal/layout/grid.go` | 2026-08-07 |
| Subgrid lite | **cut** (ordinary grid) | `internal/layout/grid.go` | 2026-08-07 |
| Vertical writing-mode | **cut** (horizontal layout) | docs: `compatibility-matrix.md` / `fonts.md` | 2026-08-07 |
| Container boolean tree | **keep** with ponytail | `internal/css/container.go` | 2026-08-07 |
| Custom HTML parser | **keep** with ponytail | `internal/html/html.go` | 2026-08-07 |
| :is/:where | **cut** | — | 2026-08-07 |
| SVG triple stack | **cut** → canvas only | `internal/svg/raster.go` | 2026-08-07 |

---

## Phase 5: Closure gates + debt ledger (path to ~10/10)

> **Goal:** Prove leanness; prevent stub regression; establish ponytail debt ledger.  
> **Est.:** 0.5 day + ongoing

### 5.1 Rating re-audit

- [x] Re-run 5-area ponytail audit (or single `ponytail-audit`) after Phases 0–4.
- [x] Update this file’s Overall Rating table; target **≥ 9.0 / 10**.
- [x] Net line estimate closed: record `git diff --stat` on remediation branch.

### 5.2 Ponytail debt ledger

- [x] Grep `// ponytail:` across repo; write `plans/reviews/ponytail/PONYTAIL-DEBT.md` (or section below).
- [x] Every intentional shortcut names **ceiling** and **upgrade trigger** (`no-trigger` tags forbidden).
- [x] Baseline audit had **0 markers**; post-remediation: **18 markers** with ceiling+trigger — [PONYTAIL-DEBT.md](PONYTAIL-DEBT.md).

### 5.3 Regression guards

- [x] CLI stub-flag rejection tests (`--dpi`, JS cluster, etc. must be **unknown**) in `internal/cli/cli_test.go` — Policy A prevents typed stub reintroduction via flags.
- [x] Keep `TestDirectModuleAllowlist` green; do not add third direct deps without product proof. [`internal/pdf/shape_test.go` allowlist]
- [x] Closure: `make lint` + `make test` green 2026-08-07 after Phases 0–5 integration.

### 5.4 Required checks (every non-doc phase)

Per `skills/phase-wise-checklist/SKILLS.md`:

- [x] For every non-documentation change: `make lint` and `make test` before marking phase complete.
- [x] Record both outcomes beside the phase gate; leave rows unchecked if either fails.

### 5.5 Final handoff

- [x] Phase 0–3 all gates green.
- [x] Phase 4 decisions recorded.
- [x] Overall rating re-scored ≥ 9.0 / 10 (or gap list with owners).
- [x] Pointer: architecture deepening (if any) goes under `plans/reviews/improve-codebase/` — not this file.

---

## Area detail scorecards (from subagents)

### Area 1 — Surface API + CLI + settings · **5.2 / 10** · ~650L

| Dimension | Score |
|---|---:|
| Dead API surface | 3.5 |
| Duplication | 4.5 |
| Over-abstraction | 6.0 |
| Intentional shortcuts | 7.0 |
| Dependency bloat | 10.0 |

**Strengths:** Zero third-party deps in area; thin `cmd/*` mains; small public API; real multi-object CLI grammar.

### Area 2 — Convert + load + HTML + outline · **5.4 / 10** · ~380L

| Dimension | Score |
|---|---:|
| Dead API surface | 4.5 |
| Duplication | 5.0 |
| Over-abstraction | 6.0 |
| Intentional shortcuts | 7.5 |
| Dependency bloat | 9.0 |

**Strengths:** Coherent `RunPDFContext` pipeline; pure outline package; real load ACL/caps.

### Area 3 — CSS engine · **6.7 / 10** · ~320L

| Dimension | Score |
|---|---:|
| Dead API surface | 5.5 |
| Duplication | 5.0 |
| Over-abstraction | 6.5 |
| Intentional shortcuts | 8.5 |
| Dependency bloat | 10.0 |

**Strengths:** Stdlib-only CSS; intentional print pseudos; real `:has()` / container lite product surface.

### Area 4 — Layout engine · **5.4 / 10** · ~450–900L

| Dimension | Score |
|---|---:|
| Dead API surface | 3.5 |
| Duplication | 5.0 |
| Over-abstraction | 8.5 |
| Intentional shortcuts | 7.0 |
| Dependency bloat | 9.0 |

**Strengths:** Almost no interfaces; product-driven float/sticky/table pagination; honest Partial comments. Dragged by dead packing cluster + optional Grid L3 surface.

### Area 5 — PDF + imageout + SVG · **6.3 / 10** · ~1500L

| Dimension | Score |
|---|---:|
| Dead API surface | 7.5 |
| Duplication | 7.0 |
| Over-abstraction | 5.0 |
| Intentional shortcuts | 6.5 |
| Dependency bloat | 6.0 |

**Strengths:** Allowlisted deps; real TTF subset + Type0 CJK; lean JPEG/PNG paths. Dragged by triple SVG stack + dual shaping/font paths.

---

## Intentional keep (`ponytail:` candidates — mark in code during remediation)

These are **not** debt if documented with ceiling + upgrade trigger:

1. **Custom `html.Node` tree** — CSS/layout assume Parent/Attrs/void/raw-text; swap is a rewrite.
2. **TOC fixed-point + `cloneResult`** — page numbers depend on TOC page count.
3. **`DefaultTOCXSL` stub** — CLI dump compatibility without XSLT.
4. **Type0 Identity-H + simple Latin-1 dual embed** — CJK vs WinAnsi.
5. **WOFF1 decode in-tree** — `@font-face` without new modules.
6. **Embedded Liberation FaceSet** — offline Latin baseline.
7. **go-text OT shaping** — Arabic/CJK without CGO HarfBuzz.
8. **canvas as single SVG primary** (after Phase 3) — gradients/clipPaths for wiki logos.
9. **`FindWithGlyph` registry fallback** — IPA/CJK tofu avoidance.
10. **wkhtml multi-object CLI grammar** — product compatibility.
11. **Print cascade Partial heuristics** (sticky continuation, section wash) — fidelity debt, not unused API.
12. **Library reusing `cli.Command` as pipeline DTO** — one wire format; document temp-file cost until bytes sink exists.

---

## Dependencies (ordering)

```
Phase 0 (dead deletes)
    └─► Phase 1 (settings/CLI honesty)
            └─► Phase 2 (dual-path collapses; needs stable contracts)
                    └─► Phase 3 (SVG single path; needs paint/font stability)
                            └─► Phase 4 (scope decisions; docs)
                                    └─► Phase 5 (re-rate + debt ledger)
```

Parallelism: Phase 0 layout/pdf/convert deletes can land as independent PRs. Phase 1 should land before growing any new CLI flags. Phase 3 SVG cut should not land before wiki smoke is green.

---

## Suggested PR slicing (optional)

| PR | Scope | Depends on |
|---|---|---|
| PR-P0a | Layout packAvoid* + partition/layoutInline/autoTrack | — |
| PR-P0b | PDF dead operators/helpers/ShapeRun/DrawImage | — |
| PR-P0c | Load/html stubs + doc.go cleanup | — |
| PR-P1 | Settings/CLI honesty policy + dual fields | P0 optional |
| PR-P2a | HF paint + outline exclude collapse | P0–P1 |
| PR-P2b | CSS selector dedupe + layout width/display helpers | P0 |
| PR-P2c | PDF subset merge + shaping ceiling | P0b |
| PR-P3 | SVG canvas-only + imageout bitmap delete | P2c optional |
| PR-P4 | Scope decisions + `ponytail:` markers + docs | P0–P3 |
| PR-P5 | Re-audit rating + debt ledger | P4 |

---

## Next step

**None for this ledger.** Phases 0–5 are closed on `chore/ponytail-review-1`.

Optional stretch toward a pure 10/10 (not blocking):

1. Bytes/writer sink for library Convert (documented ceiling in `api.go`).
2. Revisit canvas indirect dependency weight if SVG usage stays narrow.
3. Architecture deepening under `plans/reviews/improve-codebase/` (separate skill).

**Baseline: 5.7 / 10 · Post-remediation: 9.5 / 10 (weighted, 2026-08-07) · Residual to 10: third-order layout/CSS ceilings + canvas indirect deps**


## Partials closed + de-duplication (2026-08-07)

- [x] **Unified Set/Get keys** — each key registered once via `gReg`/`oReg`/`iReg` in `reflect.go` (no separate getters key list). `TestKeyTableSetGetParity` enforces set/get key equality.
- [x] **No `Web.Background` field** — removed from `Web`; sole paint switch is `PdfGlobal.Background`. Object `web.background` is Policy A ignored; ImageConverter routes `web.background`/`background` to Global.
- [x] **Shared `cli.Command.OpenOutput`** — convert and imageout both call it (no duplicated openCommandOutput). `TestOpenOutputWriterPrecedence` proves writer > path.
- [x] **Library bytes sink** — `OutputWriter` + in-memory capture in `Converter`/`ImageConverter`.
- [x] **Fresh weighted re-audit** — area scores updated; overall **9.5** = weighted sum (see table above).

`make lint` + `make test` green after these changes.
