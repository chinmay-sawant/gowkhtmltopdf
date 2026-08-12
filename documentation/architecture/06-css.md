# CSS subsystem: parse, selectors, cascade

## 1. Responsibility & position in the pipeline

`internal/css` implements the **CSS subset** that gowkhtmltopdf accepts, and nothing
more. Its package doc (internal/css/css.go, lines 1–18) states the contract
precisely:

> Scope: `*`, type, `.class`, `#id`, attribute selectors, `:first-child` /
> `:last-child` / `:nth-child` / `:has()` / `:not()`, descendant/child/sibling
> combinators, `@media` type + size-feature matching, `@container` size queries,
> `!important`, inline style attributes. Unsupported constructs degrade without
> panicking.

The package sits **between the HTML tree and the layout engine** in the pipeline:

```text
internal/load    → internal/html   → internal/css   → internal/layout   → internal/pdf
   (fetch)           (tree)            (this pkg)       (style+cascade     (paint)
                                          │              +box layout)
                                          ├──> internal/convert/prepare   (stylesheet collection, @font-face)
                                          ├──> internal/outline           (exclude-selector matching)
                                          └──> internal/imageout          (image-mode style)
```

It is a **pure parsing/matching/value library**: it never fetches resources,
never resolves page size, and never decides layout. It produces three kinds of
artifacts:

1. **Parsed sheets** (`*css.Stylesheet`) containing ordered rules, `@font-face`
   metadata and `@page` margin/size declarations.
2. **Boolean matching results** (`css.Match`, `css.MatchPseudo`,
   `css.MediaMatches`, `css.ContainerCond.Matches`) evaluated against the
   `internal/html` tree and viewport numbers supplied by the caller.
3. **Value helpers** (`css.ParseLength`, `css.ParseColor`, `css.ParseFontFamily`,
   `css.LengthToPt`, `css.ResolveCustomProps`, ...) that downstream style
   resolution uses to turn raw declaration strings into typed geometry.

Layout owns everything after this: computed styles, inheritance, cascade
*merging*, unit-to-point interpretation inside box contexts, and painting. The
boundary rule is that **css never imports layout** (see §5), so the package can
be reasoned about and fuzzed in isolation.

## 2. Package / file map

All files live under `internal/css/` and belong to `package css`. Approximate
line counts from `wc -l` (2026-08):

| File | Lines | Responsibility |
|------|------:|----------------|
| `css.go` | 1784 | Stylesheet/rule/selector/declaration model; top-level parser; at-rule handling (`@media`, `@container`, `@page`, `@font-face`, skip-others); selector parsing; right-to-left matching; specificity; `:nth-child` arithmetic; class/attr helpers |
| `values.go` | 621 | Declaration-block splitting (`ParseInline`); `!important`; length/number/color parsing; `var()` fallback + custom-property resolution; font-family splitting; named-color table |
| `container.go` | 723 | `@container` prelude parsing (`ContainerQuery`), boolean condition tree (`ContainerCond`), size-feature evaluation, length→pt conversion (`LengthToPt`), container-name/shorthand sidecars, `HasContainerRules` |
| `has.go` | 406 | Paren/quote scanning shared by `:has()`, `:not()`, media & container features; strict selector-list parsing; relative selector matching; leftmost-match combinator walk; specificity max-of-arguments helpers |
| `media.go` | 230 | `MediaMatches` evaluation of raw `@media` preludes: types (`print`/`screen`), size features, `orientation`, `not`/`only`, comma OR-lists |
| `css_test.go` | 973 | Parse/matching/specificity/value/custom-prop tests |
| `has_test.go` | 161 | `:has()`/`:not()` parse+match+specificity tests |
| `container_test.go` | 189 | Container query parse/eval tests |
| `media_test.go` | 86 | Media-query evaluation tests |
| `pseudo_element_drop_test.go` | 74 | Regression: `::before/::after` must not apply to the host |
| `target_pseudo_test.go` | 55 | Regression: `:target` must not match the bare host |
| `wiki_print_hide_test.go` | 52 | Real-world parser smoke test (Wikipedia print-hide sheets) |

The two complementary roles are worth distinguishing early:

- **Parse-time**: `css.go` + `container.go` + `has.go` + `values.go` build
  `Stylesheet`/`Selector`/`ContainerQuery`/`Declaration` values.
- **Match-time**: `css.go` (`Match`, `MatchPseudo`, `matchPart`),
  `has.go` (combinator walk), `media.go` (`MediaMatches`), `container.go`
  (`ContainerCond.Matches`).

## 3. Key types, functions & entry points

### 3.1 Type model (all in internal/css/css.go)

| Type | Location | Purpose |
|------|----------|---------|
| `Stylesheet` | css.go:49 | Parsed sheet: `Rules []Rule`, `FontFaces []FontFace`, `Page *PageStyle`. Rules keep source order |
| `Rule` | css.go:72 | Selector list + declaration block + raw `Media` prelude + `Order` (source-order tiebreak, rebased by callers across sheets) + optional `Container *ContainerQuery` |
| `Selector` | css.go:83 | Chain of `SelectorPart` linked by combinators; carries a cached specificity triple (`spec`/`specValid`, computed at parse time) |
| `SelectorPart` | css.go:95 | One compound: `Tag`, `Classes`, `ID`, `Attrs []AttrSelector`, `Pseudos []PseudoClass`, `PseudoElement` ("before"/"after"/""), `Combinator` ("" first part, ">" "+" "~" " ") |
| `AttrSelector` | css.go:108 | `[name]`, `[name=value]`, `~=`, `*=`, `^=`, `$=`, `|=` forms |
| `RelativeSelector` | css.go:116 | Selectors-4 relative selector (`>`, `+`, `~`, descendant) used inside `:has()` |
| `PseudoClass` | css.go:123 | Named pseudo with optional `Arg` (`:nth-child`), `Has []RelativeSelector`, `Not []Selector`, and a pre-parsed integer `nth nthForm` |
| `Declaration` | css.go:134 | `Prop`, `Value`, `Important` — the raw wire form of a `prop: value[!important]` pair |
| `PageStyle` | css.go:59 | `@page` margin/size declarations kept as raw strings, resolved at the PDF boundary |
| `FontFace` | css.go:66 | `@font-face` local subset: `Family` + raw `Src` (consumed by `convert.MergeFontFaces`) |

### 3.2 Public entry points (exported functions)

| Function | Location | Purpose |
|----------|----------|---------|
| `Parse(src string) (*Stylesheet, error)` | css.go:143 | Parse a whole stylesheet. Only unbalanced-brace input errors; garbage degrades silently |
| `ParseSelectors(s string) ([]Selector, bool)` | css.go:644 | Strict parse of a comma-separated selector list (used by `--exclude-from-outline`, internal/convert/outline.go:123) |
| `ParseInline(style string) []Declaration` | values.go:9 | Parse a `style=""` attribute value |
| `Match(s Selector, n *html.Node) bool` | css.go:1216 | Does the selector match the element? (right-to-left) |
| `MatchPseudo(sel Selector, n *html.Node, pseudo string) bool` | css.go:1222 | `::before`/`::after` shape matching (host must match the pseudo's compound) |
| `Specificity(s Selector) (a, b, c int)` | css.go:1738 | ID / class·attr·pseudo / type counts; `:has()/:not()` contribute max-of-arguments |
| `MediaMatches(query, mediaType string, widthPt, heightPt float64) bool` | media.go:24 | Evaluate a raw `@media` prelude against conversion media + viewport |
| `ParseLength(val string) (float64, string, bool)` | values.go:108 | Number+unit split; bare numbers default to px; accepts px/pt/pc/in/cm/mm/em/rem/ex/ch/%/vw/vh |
| `LengthToPt(val float64, unit string, basePt float64) (float64, bool)` | container.go:113 | Unit→pt conversion incl. em/rem/ex/ch; `%` and viewport units unsupported (false) |
| `ParseNumber(val string) (float64, bool)` | values.go:163 | Bare number (line-height, font-weight) |
| `ParseColor(val string) (r,g,b int, alpha float64, ok bool)` | values.go:182 | `#rgb`, `#rrggbb`, `#rrggbbaa`, `rgb()/rgba()` int/float/percent, `transparent`, named-color subset, `var()` fallback |
| `ParseFontFamily(value string) []string` | values.go:562 | Comma split + quote trim |
| `ResolveVar(val string, lookup func(string)(string,bool)) string` | values.go:412 | Expand one `var(--name, fallback)` chain (bounded depth) |
| `ResolveCustomProps(declared, inherited map[string]string) map[string]string` | values.go:448 | The single place custom-property policy lives: overlay + memoized var expansion + cycle guard |
| `ParseContainerNameValue(value string) string` | container.go:165 | `container-name: none | <custom-ident>+` |
| `ParseContainerShorthand(value string) (name, ctype string)` | container.go:189 | `container: <name> [ / <type> ]?` |
| `HasContainerRules(sheets []*Stylesheet) bool` | container.go:709 | Fast gate: does any sheet contain `@container` rules? (layout skips its second style pass when false, internal/layout/layout.go:831) |
| `FontFaceURLs(src string) []string` | css.go:392 | Extract `url(...)` from an `@font-face src` (case-preserving) |

### 3.3 Interesting non-exported machinery

| Symbol | Location | Why it exists |
|--------|----------|---------------|
| `parseAtRule` | css.go:172 | Dispatch: `@media`/`@container`/`@page`/`@font-face` parsed; `@keyframes` and anything unknown **parse-ignored** (static cascade only; there is no animation) |
| `stripComments` | css.go:537 | Removes `/* */`, preserving `\n` so line numbers stay stable |
| `findBlock`/`takeBlock` | css.go:572/614 | Brace finding with quote/paren tracking; the only source of real parse errors (`errUnbalanced`, `errNoBlock` at css.go:526) |
| `splitTopLevel` | css.go:668 | Splits selector lists and declaration blocks on top-level `,`/`;` outside parens/brackets/quotes |
| `splitSelectorChain` | css.go:712 | Breaks a selector into compounds + separators, including the `addDescendantCombinator` whitespace rule |
| `writePseudoLiteral` | css.go:826 | Keeps `:pseudo(...)` and `::before/::after` inside the compound so unsupported pseudos reject the selector instead of degrading to the host |
| `parseCompoundCtx` | css.go:862 | One compound → `SelectorPart`; `insideHas` rejects nested `:has`/pseudo-elements |
| `appendSimplePseudo` | css.go:1086 | Classifies pseudos: matchable (`first-child`, `nth-child`, `link/visited`…), never-match-but-keep (`hover/active/focus/target`, unknown), rejected (`first-line/first-letter`) |
| `parseNthArg`/`matchNth` | css.go:1576/1608 | `:nth-child()` pre-parsed to integer form at parse time; matching is pure arithmetic |
| `leftmostMatch`/`leftmostStep` | has.go:256/284 | Shared right-to-left combinator walk; `Match`, `MatchPseudo` and `:has` relative matching all ride it |
| `computeSpecificity` | css.go:1747 | Selectors-4 specificity incl. `:has()/:not()` max-of-argument contribution |
| `parseDeclarations` | values.go:15 | Block → `[]Declaration`; `validPropName` allowlist (lowercase `-` alnum only); `isImportant` accepts `! important` spacing |
| `namedColorTable` | values.go:594 | Curated CSS2 + common web colors (read-only cache; not full CSS Color 4) |

## 4. Data & control flow

### 4.1 Stylesheet collection (the entry into css for a document)

1. `internal/convert/prepare/styles.go` `CollectSheets` walks the HTML tree with
   `root.Walk`, visiting every `style` and `link` element (styles.go:35–48).
2. Inline `<style>` text → `css.Parse` (styles.go:90); `<link rel=stylesheet>`
   is first gated by media/viewport via `linkStylesheet` (styles.go:164, itself
   using `css.MediaMatches`), then fetched through `load.ResourceContext.Fetch`
   under the document's ACL policy, and parsed with `css.Parse` (styles.go:110).
3. A hard rule limit of `maxStylesheetRules = 1_000_000` (styles.go:25) with a
   soft warning at 25 000 rules bounds hostile input.
4. The resulting `[]*css.Stylesheet` plus the internal UA/helper sheets
   (`prepare.SimplifyChromeCSS`, `prepare.SimplifyMediaWikiCSS` parsed at
   simplify.go:49/55) are handed to layout as `opts.Sheets`.

### 4.2 Cascade in the layout consumer (internal/layout/style_cascade.go)

The css package reports *matches and specificity*; layout performs the cascade:

1. `matchedRules(node, pseudoElem)` (style_cascade.go:207) iterates every sheet
   in order and calls `css.MediaMatches(rule.Media, ctx.media, ctx.viewportW,
   ctx.viewportH)` (style_cascade.go:231) and `containerGateMatches`
   (style_cascade.go:273, using `runic.Container.Cond.Matches(info.inlineSize,
   info.fontSize)`) before selector matching — so media/container filtering is
   layered *on top of* the css package.
2. `appendRuleSelectorHits` (style_cascade.go:247) loops selectors, calls
   `css.Match`/`css.MatchPseudo` and records `css.Specificity` as the
   `ruleHit` triple.
3. `cascadeRaw` (style_cascade.go:297) merges three tiers into one winner map:
   - UA sheet rules (hard-coded `uaRules`, priority `(0,0,0)` order `-1`);
   - author-sheet hits with `(a,b,c)` `r.Order` `d.Important`;
   - inline style via `css.ParseInline(node.Attribute("style"))` with a
     sentinel specificity `1<<maxIntShift` described as "outranks all normal
     declarations and all sheet important declarations" (style_cascade.go:337).
4. `applyCascadeWin` compares `(important ⇒ ids/classes/types ⇒ order)` — the
   canonical CSS cascade ordering. Any `!important` beats any normal value; the
   important layer is a separate tier.
5. Custom properties: `mergeCustomProps` (style.go:8) extracts `--*`
   declarations and calls `css.ResolveCustomProps(declared, parentProps)`;
   `resolveRawVars` (style.go:18) then expands `var()` in ordinary values via
   `css.ResolveVar` with a lookup into the resolved custom-prop map.
6. Typed conversion happens later in `internal/layout/style_properties.go`
   using `css.ParseLength`/`css.ParseColor`/`css.ParseFontFamily`/
   `css.ParseContainerNameValue`/`css.ParseContainerShorthand` (e.g.
   style_properties.go:350, 809, 1192).

### 4.3 Matching walk (inside css, selector → bool)

`Match(sel, n)` (css.go:1216) → `leftmostMatch` (has.go:256):

- The **last part** must match `n` via `matchPart` (css.go:1266): tag
  case-insensitive, `id` exact, every class a whitespace token
  (`hasClassToken`, css.go:1669; non-ASCII whitespace falls back to
  `hasUnicodeClassToken`, css.go:1698), every attribute selector
  (`matchAttrs`/`attrValueMatches`, css.go:1322/1350), every pseudo
  (`matchPseudos`/`matchPseudo`, css.go:1296/1410).
- `matchPseudo` dispatches: `first-child`/`last-child` via element-sibling
  scans (css.go:1489/1509), `nth-child` via `matchNth` (css.go:1608),
  `:has()` via `matchAnyRelative` (css.go:1439), `:not()` via `matchNone`
  (css.go:1450), `link`/`visited` → both mean "a[href]"
  (`isLinkAnchor`, css.go:1479), `root` via `isRootElement` (css.go:1463),
  interactive pseudos → `false` (never match in print).
- Earlier parts step left through the combinator chain (`leftmostStep`,
  has.go:284): `>` child, `+` adjacent, `~` sibling, descendant ancestor.

`:has()` (Selectors 4 relative selectors): `parseRelativeSelectorList`
(has.go:125) parses `>`, `+`, `~`, or descendant-relative argument lists;
`matchRelative` (has.go:180) anchors the argument at the subject and uses
`elementDescendants` (has.go:341) for scoped walks. Nested `:has()` and
pseudo-elements inside `:has()` are rejected at parse time (has.go's
`insideHas` flag, css.go:1033).

### 4.4 Media & container evaluation

- `@media` preludes are stored **raw** on `Rule.Media` (css.go:72) at parse
  time; the viewport is unknown to the parser. Conversion later supplies
  `(mediaType, viewportW, viewportH)` to `MediaMatches` (media.go:24): comma
  lists OR, `not`/`only` supported, size features (`width`, `height`,
  `inline-size`, `block-size` with min-/max- prefix) compare against the
  viewport in points with an em base of 12pt (`defaultMediaFontPt`, media.go:9),
  `orientation` supported, unknown features → `false`.
- `@container` preludes are parsed eagerly into `ContainerQuery{Cond}`
  (container.go:20). `Cond` is a boolean tree of `SizeFeature` comparisons
  (`and`/`or`/`not`, container.go:30). `ContainerCond.Matches` (container.go:48)
  evaluates against a container's inline size and font size, supplied by layout
  after box layout (`findSizeContainer` in layout). Nested `@container` inside
  `@media` is flattened (parseNestedAtRule, css.go:489) and the nested query
  *replaces* (not combines) the outer one.

### 4.5 @font-face and @page side channels

- `@font-face` rules are collected into `Stylesheet.FontFaces`
  (parseFontFaceRule, css.go:275; `parseFontFace`, css.go:371). Conversion
  (`convert/prepare/styles.go:191`) iterates these and calls
  `css.FontFaceURLs` (css.go:392) to fetch each `src` through the document's
  resource policy. Font weight/style are intentionally ignored (css.go:66).
- Unnamed `@page` declarations keep raw `margin`/`size` strings in
  `PageStyle` (css.go:59, parsePageRule css.go:192) so physical units resolve
  at the PDF boundary.

## 5. Cross-package dependencies

### 5.1 What css imports

| Import | Why |
|--------|-----|
| `gowkhtmltopdf/internal/html` | `Match`, `MatchPseudo` and all matching walkers operate on `*html.Node` (custom allowlisted tree, internal/html/html.go:36). This is the only intra-repo dependency |
| stdlib: `errors`, `strconv`, `strings`, `unicode`, `unicode/utf8` | Parsing, identifier/class scanning, number/color conversion |

The import graph is a **strict lower layer**: `internal/css → internal/html →
stdlib only`. This guarantees no import cycle can ever form with layout,
convert, or pdf, and it makes the package independently testable.

### 5.2 Who depends on css

| Consumer | What it uses |
|----------|--------------|
| `internal/layout` (style_cascade.go, style.go, style_properties.go, transform.go, pseudo_content.go, layout.go) | `Parse`, `Match`, `MatchPseudo`, `Specificity`, `MediaMatches`, `ParseInline`, `ParseLength`, `LengthToPt`, `ParseColor`, `ParseFontFamily`, `ParseNumber`, `ResolveVar`, `ResolveCustomProps`, `ParseContainerNameValue`, `ParseContainerShorthand`, `HasContainerRules` |
| `internal/convert/prepare` (styles.go, simplify.go, prepare.go) | `Parse` for `<style>`/`<link>`/helper sheets; `MediaMatches` for link media gating; `FontFaces`/`FontFaceURLs` for font merging |
| `internal/convert` (outline.go, toc.go, page_islands.go, simplify.go) | `ParseSelectors` for `--exclude-from-outline`; `ParseLength` in TOC geometry; `Parse` of `islandBreakOverrideCSS` |
| `internal/outline` (outline.go) | `css.Selector` matching for outline exclusion |
| `internal/imageout` (imageout.go) | Same pipeline in image mode (load → parse → css → layout → raster) |
| `api.go` / `internal/cli` / `internal/settings` | *None directly* — settings inject sheets/media via layout options |

**Import-direction rule**: nothing below css (html) knows css exists; css never
imports layout, convert, pdf, load, or settings. All viewport/media/container
*context* is passed as arguments, never pulled in.

### 5.3 Latent coupling worth knowing

- `Stylesheet.Order` is owned by the parser (single sheet), but the cascade
  needs a document-global order across many sheets. Layout treats the order as
  per-sheet and relies on sheet iteration order for cross-sheet tiebreaks
  (style_cascade.go:225). Callers who build compound sheet lists must keep
  document order intact (CollectSheets does).
- `Selector.spec` cache means **mutating a parsed Selector's parts after
  parsing yields stale specificity**; `Specificity` only falls back to a walk
  when `specValid` is false (css.go:1738). Today no caller mutates parsed
  selectors, but the invariant is documented in the field comment (css.go:86).

## 6. Design decisions & trade-offs

1. **Zero third-party CSS dependencies.** The project rule is pure Go, no cgo,
   no third-party HTML/CSS/PDF APIs. Unlike the layout package's one narrow
   exception (OpenType shaping via `github.com/go-text/typesetting`), the CSS
   layer is 100% handwritten — parser, selector engine, cascade helpers, unit
   math. This keeps `CGO_ENABLED=0` trivially satisfiable and the security
   surface small.

2. **Allowlist over completeness.** The package deliberately implements a
   *report* subset: no full Selectors 4, no grid-of-everything, no animations.
   The compatibility matrix
   (documentation/compatibility-matrix.md §2) is the normative contract, and
   `documentation/fidelity.md` frames it: this is Tier 1–2 "leave wkhtmltopdf
   for most jobs", explicitly **not** Chrome-quality arbitrary-web print
   (`fidelity.md` Tier 3 stays banned for a pure-Go report engine).

3. **Degrade, never panic, never misapply.** Two distinct softenings:
   - *Garbage is dropped*: recoverable parse debris is skipped; only
     unbalanced braces error (`errUnbalanced`, `errNoBlock`, css.go:526) and
     even then `ParseNeverPanics` (css_test.go:178) pins the "no panic on
     adversarial input" property.
   - *Unsupported selectors never degrade to the host*. This is the
     single most subtle design rule in the package. `writePseudoLiteral`
     (css.go:826) and `appendSimplePseudo` (css.go:1086) deliberately **keep
     unknown/unmatchable pseudos on the compound** so that, e.g., `li:target`
     cannot silently become `li` and apply `:target`'s declarations to every
     list item (the code comment at css.go:826 documents a real regression:
     `p::before{width:120pt}` used to crush wiki body columns, and `li:target`
     would otherwise paint every reflist item blue). Unmatchable pseudos are
     stored, matched as `false`, and thus **suppress** the whole rule.
     `:first-line`/`:first-letter` are rejected outright; `:is()/:where()` are
     unknown (never match, css.go:123).

4. **Parse early, evaluate late.** Media preludes stay raw strings; container
   queries and `:nth-child` arguments are eagerly compiled (integer `nthForm`,
   css.go:1566). The former exists because the viewport is only known at
   conversion; the latter exists because matching is then pure integer math —
   a measurable hot path in the per-element cascade.

5. **Selectors-4 specificity for functional pseudos.** `:has()`/`:not()`
   contribute the specificity of their most specific argument (css.go:1738),
   not a flat class-level count. This matches current CSS behavior and avoids
   the classic `:not()` over/under-matching bugs.

6. **One owner for custom-property policy.** `css.ResolveCustomProps`
   (values.go:448) is documented as "the single place custom-property policy
   lives": inherited overlay + declared values, memoized expansion, cycle stack
   (cycles → empty → invalid). Layout drives it; css defines it. `ParseColor`
   handles `var()` only at the fallback level (values.go:182) — a deliberate
   simplification flagged with a `ponytail:` note.

7. **Caller-supplied context keeps css pure.** Matching needs the viewport
   (`MediaMatches`), a container's used size (layout), and a base font size
   (em/rem in `LengthToPt`, container.go:113). All are function arguments.
   This purity is what lets layout do container-query gating in two passes
   (css only *reports* `HasContainerRules`, layout.go:831) and what keeps css
   trivially fuzzable.

8. **Micro-allocations matter in the hot path.** `matchPart`, `hasClasses`
   and `matchAttrs` are called once per (element, candidate-selector) pair;
   the class/word token scanning avoids `strings.Fields` allocations
   (`containsWord`/`hasClassToken`, css.go:1381/1669, with a Unicode fallback
   only on non-ASCII whitespace), the color parser scans channels in place
   (`parseRGBColor`, values.go:267), and `namedColorTable` is a cached global
   (values.go:594), all with explicit `//nolint:cyclop` notes explaining why
   the linear dispatch stays.

9. **Unit math tuned for IEEE float cancellation.** `LengthToPt`
   (container.go:113) deliberately multiplies-then-divides physical units
   (`val * 72 / 25.4`) so `25.4mm` cancels cleanly to `72pt` instead of
   `71.999…`; `px` maps at `0.75` (96 CSS px/in → 72 pt/in). `%` and viewport
   units return `false`, leaving the decision to caller policy (e.g.
   line-height inherits, percentages resolve against the containing block in
   layout).

## 7. Notable patterns & invariants

- **Parse-order stability**: `Rule.Order` is a per-sheet monotonic counter
  owned by `Parse` (parseOneRule, css.go:328); across sheets, callers preserve
  document order and use sheet order as the final tiebreak — the CSS "later
  wins" rule is reproduced exactly by `applyCascadeWin`'s order comparison.
- **At-rules taxonomy**: `@media`/`@container` (flattened into rules),
  `@page`/`@font-face` (side-channel structs), `@keyframes`/unknown (skipped).
  There is **no `@import`, `@supports`, `@layer`, `@charset`, or nesting**;
  any of these is silently skipped by `skipAtRule` (css.go:354).
- **Inline is strongest**: layout gives `style=""` the sentinel specificity
  `1<<maxIntShift` — stronger than any sheet rule including `!important`
  (style_cascade.go:337). `!important` in inline style is parsed by
  `isImportant` but the sentinel already dominates.
- **Pseudo-element shapes**: `::before/::after` never match the host
  element (css.go:1266 checks `part.PseudoElement != ""`), only through
  `MatchPseudo` used by layout's pseudo-content path (`pseudo_content.go`,
  style_cascade.go:265).
- **Never-match sets**: interactive pseudos (`hover/active/focus/target`) are
  *kept on the compound* but `matchPseudo` returns `false` for them (css.go:1410)
  — a static PDF has no pointer/focus state.
- **Link semantics**: `:link` and `:visited` both mean "an `<a>` with a
  non-empty `href`" (any scheme incl. `#fragments`), since print has no history.
- **Root definition**: `:root` matches the document element while excluding
  the synthetic `#document` wrapper (`isRootElement`, css.go:1463 — the HTML
  tree wraps everything under an element named `#document`).
- **Container condition grammar**: `or < and < not < paren/feature` precedence
  with top-level keyword splitting (`splitCondKeyword`, container.go:390),
  range forms `(`name `>` 20em / `20em < name`)` supported as single
  comparisons (rangeFeatureFromTokens, container.go:668), and only
  `width`/`inline-size` features (`matches`, container.go:85).
- **String/paren scanning shared**: `matchingParen`/`skipQuoted`/`takeParen`
  (has.go:12, container.go:442, has.go:49) are the single implementation of
  balanced-paren/quoted scanning reused by `:has()`, `:not()`, media features,
  and container conditions, honoring backslash escapes.
- **Identifier allowlist**: compound tags, ids, classes, attribute names and
  property names all route through `validIdent`/`validPropName`
  (css.go:1190, values.go:56) — lowercase `[a-z0-9-]`, no leading digit.
  Property names that fail are dropped with their declaration.

## 8. Security considerations

The css package itself executes no code and fetches nothing, but it sits on the
attack path for hostile HTML, so its posture matters:

- **Parsing is memory-safe by construction**: bounded index scanning,
  quote/paren tracking with sentinel errors, and the `ParseNeverPanics`
  regression test (css_test.go:178). There is no regex engine, no eval, no
  network access anywhere in the package.
- **Resource amplification is capped outside css**: linked-stylesheet and
  `@font-face` fetches are governed by `internal/load`'s ACL and by
  `prepare.CollectSheets`'s rule limits (soft warn 25k, hard cap 1M,
  styles.go:21/33/63) — see `documentation/THREAT-MODEL.md`.
- **No CSS-triggered exfiltration**: `url()` is only honored in
  `@font-face src` via `FontFaceURLs` (and only through the loader's policy);
  `background: url(...)` etc. are inert raw strings. Media queries cannot
  probe anything beyond the numbers the caller supplies.
- **The `--enable-local-file-access` / `--allow` story applies at load, not
  css**; css just consumes whatever sheets the loader delivers.
- Fuzz-relevant surface: `Parse`, `ParseSelectors`, `ParseInline`,
  `ParseLength`, `ParseColor`, `LengthToPt`, `MediaMatches` — all pure string
  → value functions ideal for fuzzing harnesses (see §9).

## 9. Testing & verification

Unit tests live in the package (973 + 161 + 189 + 86 + 74 + 55 + 52 lines of
`*_test.go`):

| Test file | Coverage |
|-----------|----------|
| `css_test.go` | Parse basics; selector lists; comments/garbage; `!important`; media parse; at-rule skipping; `@page`; order & nested-media order; **never-panics**; unbalanced-brace errors; inline parse; compound parsing; `Match`; link/visited; root; attr operators; sibling combinators; `Specificity`; `ParseLength`/`ParseNumber`/`ParseColor`/`ParseFontFamily`; newline-preserving comment strip; `LengthToPt`; custom-property resolution incl. inheritance overlay, cycles, deep chains, self-reference fallback; strict `ParseSelectors` |
| `has_test.go` | `:has()` parse+match, invalid forms, `:has` specificity, `:not()` matching |
| `container_test.go` | `container` shorthand/name parsing, `@container` rule parsing, `Cond.Matches` truth tables, non-size container rejection at eval, `HasContainerRules`, invalid preludes skipped |
| `media_test.go` | type matching, size features (min-/max-/equal), orientation, empty media-type legacy behavior |
| `pseudo_element_drop_test.go` | Regression: `::before/::after` selectors never apply declarations to the host element |
| `target_pseudo_test.go` | Regression: `li:target` does not match a bare `li` (wiki reflist highlight bug class) |
| `wiki_print_hide_test.go` | Real-world Wikipedia print-hide sheet parses and the `.noprint` class semantics survive |

Cross-package validation:

- `internal/layout/*_test.go` and golden `fixtures` (testdata/golden/) exercise
  css end-to-end: `css.Parse` of embedded `<style>`, cascade output, container
  gating (`layout.go:831` + `containerGateMatches`), and pseudo-content via
  `MatchPseudo`. Fixtures 22/29/38 (float), 25/28/32–35 (flex/grid),
  30/37 (orphans/widows) are cited by the compatibility matrix as evidence.
- `internal/convert/outline.go` `--exclude-from-outline` selectors parse via
  `css.ParseSelectors` and are exercised by outline tests.
- Run via `make test` / `go test ./internal/css/...` (standard repo flow;
  verified by CI workflow .github/workflows/ci.yml).

## 10. Known limitations, deferred items & open questions

Ground truth: documentation/compatibility-matrix.md (the normative allowlist)
and documentation/deferred.md. Confirmed gaps in css itself:

1. **Selectors**: `:is()`/`:where()` not implemented (unknown → never match,
   css.go:123). `:first-line`/`:first-letter` rejected (css.go:1086).
   `:has()` forbids nested `:has()` and pseudo-elements inside arguments
   (css.go:1033). No `[attr operator value i]` case-insensitivity flag; no
   escaped-ident unescaping beyond the naive `\` copy (css.go:882 area).
2. **At-rules**: `@import` unsupported (linked sheets arrive only via
   `<link>`/`<style>`); `@supports`, `@layer`, nesting, `@keyframes` and all
   unknown at-rules are skip-parsed. Animations/transitions are explicitly
   out of scope (static cascaded values only, css.go:195, deferred.md §4).
3. **Values**: `%` and viewport units (`vw`/`vh`) parse but `LengthToPt`
   returns `false` for them (container.go:113) — layout decides policy; a
   curated named-color subset, not CSS Color 4 (values.go:594, ponytail note);
   `var()` inside `ParseColor` resolves fallback only (no prop map — layout's
   `ResolveCustomProps` fills that gap, values.go:182 comment).
4. **Media queries**: unknown features → `false` (media.go:158); only
   width/height/inline-size/block-size + orientation; no `resolution`,
   `pointer`, `prefers-*`, no `@media print and (min-width:...)` distinction
   by page box (viewport = page content box).
5. **Container queries**: size-only (`width`/`inline-size`), no style queries,
   no `container-type: style`, range comparisons limited to a single
   three-token form (`rangeFeatureFromTokens`, container.go:668); nested
   `@container` replaces rather than combines the outer query
   (parseNestedAtRule, css.go:489).
6. **Specificity cache staleness risk**: hand-built or post-parse-mutated
   selectors recompute specificity, but mutating a *parsed* selector's parts
   after `setSpec` leaves the stale cache (css.go:86 comment; no current
   caller does this).
7. **No cascade layers / important-layer scoping**: `!important` is a single
   global tier over all normal declarations (style_cascade.go:297); UA sheet
   is a fixed lowest tier. `@layer` sheets would be flattened today.
8. **Performance ceilings**: selector matching is O(selector parts × related
   nodes) per element with no index (no class/id dispatch table); acceptable
   for report documents, warned at 25k rules in preparation
   (styles.go:63). The `ponytail:` notes (e.g. ContainerCond tree
   simplification, container-name wire form) flag future internal cleanups.

Open questions an architect should keep an eye on:

- Whether `:is()/:where()` should join before Tier 3 work (they appear
  frequently in real-world sheets and would replace the "unknown → never
  match" default).
- Whether `@import` should be honored under the same ACL as `<link>` (currently
  it silently disables the sheet entirely — a silent *feature loss*, not a
  security hole).
- Cross-sheet `Order` rebasing: if layout ever needs a single global order,
  `Stylesheet` will need a rebase helper rather than per-sheet counters.

## 11. Related documents

- `../architecture.md` — package map overview (`internal/css` row).
- `../compatibility-matrix.md` — normative allowlist: supported properties
  (§2), selectors, media, flex/grid stages, pagination evidence fixtures.
- `../fidelity.md` — tiers (Tier 1/2 shipped; Tier 3 arbitrary-web banned),
  "report subset" framing, `:nth-child`/attr/siblings shipped note (16.1).
- `../THREAT-MODEL.md` — ACL and network-request policy around stylesheet and
  font-face fetching.
- `../deferred.md` — deferred CSS/JS/SPA items to keep the matrix honest.
- `../fonts.md` — `@font-face`/`font-family` interplay with `FontFaces` and
  `ParseFontFamily`.
- Sibling architecture docs in this directory:
  - `05-html-parser.md` — the `internal/html` tree and tokenizer css matches
    against.
  - `07-layout.md` — style resolution/cascade consumption of this package
    (style_cascade.go, style.go, style_properties.go, transform.go,
    pseudo_content.go).
  - `04-load.md` — the loader whose ACL governs stylesheet/font fetches that
    `prepare.CollectSheets` triggers.
  - `08-convert-pipeline.md` — prepare/simplify/outline/islands consumers
    (`.Sheets`, `FontFaces`, `ParseSelectors`, `islandBreakOverrideCSS`).
  - `10-imageout-svg.md` — image-mode reuse of the same css→layout path.