# Phase 3 — CSS engine

> **Parent:** [`architecture-review-2026-08-07.md`](../architecture-review-2026-08-07.md) — canonical architecture-review ledger
> **Status:** pending (findings gathered 2026-08-07 by 7 explore agents; remediation not started)
> **Depends on:** see phase map in ledger
> **Evidence archive:** raw agent findings were archived off-repo on 2026-08-07; every row below carries its Before/After snippets inline

---

## Overview

…

## Checklist

- [x] **P3-01** — One cascade rule-walk — pseudo-content silently skips the @container gate — matchedRules unifies cascadeRaw + pseudo-content
- [x] **P3-02** — Export a strict selector-parse entry; stop brewing entire stylesheets — css.ParseSelectors + convert parseExcludeSelectors
- [x] **P3-03** — One authoritative LengthToPt; conversion is written six times with drift — css.LengthToPt + layout + convert/toc consumers
- [x] **P3-04** — var() resolution semantics live on one side of the css/layout seam — ResolveCustomProps + layout merge; ParseColor var path still FIX-REVIEW residual
- [x] **P3-05** — Collapse layout's four near-identical style surfaces + hand-rolled cascade — resolveStylesWith landed (fix-layout)
- [x] **P3-06** — One rule-body parser: merge Parse's top loop into parseRuleList — done by fix-css

---

<a id="p3-01"></a>
## P3-01 — One cascade rule-walk — pseudo-content silently skips the @container gate

- [ ] **P3-01** — pending (fix-layout never landed)

- **Locations:** `internal/layout/pseudo_content.go:38-71` vs `internal/layout/style.go:519-553` (cascadeRaw)
- **Evidence sources:** area-4 F2

---

### Evidence — F2

- **Severity:** high
- **Location:** `internal/layout/pseudo_content.go:38-71` vs `internal/layout/style.go:519-553` (cascadeRaw)
- **Current (verbatim):**
```go
	for _, sheet := range e.opts.Sheets {
		if sheet == nil {
			continue
		}
		for _, r := range sheet.Rules {
			if !css.MediaMatches(r.Media, media, e.opts.Width, e.opts.Height) {
				continue
			}
			for _, sel := range r.Selectors {
				if !css.MatchPseudo(sel, n, pe) {
					continue
				}
				a, b, c := css.Specificity(sel)
				for _, d := range r.Decls {
					if !strings.EqualFold(d.Prop, "content") {
						continue
					}
					h := hit{value: d.Value, a: a, b: b, c: c, order: r.Order, important: d.Important}
					if better(h) {
						hh := h
						best = &hh
					}
				}
			}
		}
	}
```
`style.go:519-545` (`cascadeRaw`) walks the same sheets/rules with the same `css.MediaMatches` → `css.Match` → `css.Specificity` → Order+Important machinery — plus the `r.Container != nil` gate (`findSizeContainer` + `Cond.Matches`) that pseudoContent lacks. So `::before { content: "…" }` inside a non-matching `@container` block is applied unconditionally by pseudoContent while the same rule's `color` property would be correctly gated by cascadeRaw. The winner comparison is also re-encoded in pseudoContent's `better` closure (compare a/b/c/order/important) instead of reusing the cascade's.
- **Future:** a shared rule walk in the layout package that owns the gates and hands both consumers their hits:
```go
// style.go — the cascade's shared rule walk. pe != "" matches ::before/::after
// shapes instead of the element.
func (ctx *styleContext) matchedRules(n *html.Node, pe string) []ruleHit {
	var hits []ruleHit
	for _, sheet := range ctx.sheets {
		if sheet == nil {
			continue
		}
		for _, r := range sheet.Rules {
			if !css.MediaMatches(r.Media, ctx.media, ctx.viewportW, ctx.viewportH) {
				continue
			}
			if r.Container != nil {
				if ctx.containers == nil {
					continue // pass 1 / pseudo pass: used sizes unknown
				}
				info, ok := findSizeContainer(n, r.Container.Name, ctx.containers)
				if !ok || !r.Container.Cond.Matches(info.inlineSize, info.fontSize) {
					continue
				}
			}
			for _, sel := range r.Selectors {
				if pe != "" && !css.MatchPseudo(sel, n, pe) {
					continue
				}
				if pe == "" && !css.Match(sel, n) {
					continue
				}
				a, b, c := css.Specificity(sel)
				hits = append(hits, ruleHit{r: r, a: a, b: b, c: c})
			}
		}
	}
	return hits
}
```
`cascadeRaw` iterates `ctx.matchedRules(n, "")` into its normal/important apply() loop; `pseudoContent` builds a one-shot `&styleContext{sheets: e.opts.Sheets, media: media, viewportW: e.opts.Width, viewportH: e.opts.Height}` and iterates `matchedRules(n, pe)` filtering Decls for "content", with one shared `better` helper.
- **Why:** the cascade's gating knowledge — media, container query, selector match, specificity, order — is core CSS semantics, yet lives in two independent rule-walks that have already diverged once (the @container gate). Every future gate (e.g. @supports, media edge cases) must be ported twice or divergence grows. One walk means one gate in one place; the divergence disappears. Behavior change: `content` under an unmatched `@container` stops applying (was applied unconditionally). hypothesis: no fixture depends on the leak — validate by `go test ./internal/layout/... ./internal/convert/...` (targets: pseudo_content.go path, container_test.go).
- **Cost paid today:** anyone changing cascadeRaw's gate rules (media or container) must remember pseudoContent re-implements it — the last person who forgot created a real output bug.

---

<a id="p3-02"></a>
## P3-02 — Export a strict selector-parse entry; stop brewing entire stylesheets

- [~] **P3-02** — css.ParseSelectors done (fix-css); convert parseExcludeSelectors consumer pending (fix-convert)

- **Locations:** `internal/convert/outline.go:100-114`; same pattern in `internal/outline/outline_test.go:24-27`
- **Evidence sources:** area-4 F3

---

### Evidence — F3

- **Severity:** medium
- **Location:** `internal/convert/outline.go:100-114`; same pattern in `internal/outline/outline_test.go:24-27`
- **Current (verbatim):**
```go
func parseExcludeSelectors(specs []string, log io.Writer) []css.Selector {
	var out []css.Selector
	for _, s := range specs {
		sheet, err := css.Parse(s + "{}")
		if err != nil || sheet == nil || len(sheet.Rules) == 0 || len(sheet.Rules[0].Selectors) == 0 {
			fmt.Fprintf(log, "warning: ignoring invalid --exclude-from-outline selector %q\n", s)
			continue
		}
		out = append(out, sheet.Rules[0].Selectors[0])
	}
	return out
}
```
To parse *one selector* the caller fabricates a stylesheet (`s + "{}"`) and indexes `Rules[0].Selectors[0]` — the full Stylesheet/Rule layout becomes the parsing interface. `Selectors[0]` also drops every selector after the first in a comma list (`h1, h2`) silently, without a warning.
- **Future:** export the strict list parser the package already has internally (`parseSelectorListStrict(s, false)`, used for `:not()`/`:has()` args):
```go
// css.go — new export. ParseSelectors parses a comma-separated selector list;
// every part must parse (strict). Callers needing a standalone selector no
// longer fabricate a whole stylesheet.
func ParseSelectors(s string) ([]Selector, bool) { return parseSelectorListStrict(s, false) }
```
and the caller becomes:
```go
sels, ok := css.ParseSelectors(s)
if !ok || len(sels) == 0 {
	fmt.Fprintf(log, "warning: ignoring invalid --exclude-from-outline selector %q\n", s)
	continue
}
out = append(out, sels...)
```
`outline_test.go`'s `parseSel` moves to `css.ParseSelectors` too.
- **Why:** the seam leaks: a caller must know Rules keep source order and that the first selector of the first rule is "the" selector. The comma-list silent drop is a shape-of-the-structs accident, not a CSS decision. A 4-line exported parse gives a strict, self-describing interface and fixes the behavior. Export change: additive (`ParseSelectors`); behavior change: comma lists in `--exclude-from-outline` now exclude all listed selectors (previously only the first). Only caller that moves: `internal/convert/outline.go` (+ the outline test helper).

---

<a id="p3-03"></a>
## P3-03 — One authoritative LengthToPt; conversion is written six times with drift

- [~] **P3-03** — css.LengthToPt done (fix-css); layout callers + convert/toc wrapper pending (fix-layout, fix-convert)

- **Locations:** `internal/css/container.go:78-98`; `internal/layout/style.go:1432-1456` (fontSize), `style.go:1496-1549` (lengthBox), `style.go:1541-1585` (marginLen), `style.go:1572` (pxToPt); `internal/layout/transform.go:389-430`; `internal/convert/toc.go:42-67`
- **Evidence sources:** area-4 F4

---

### Evidence — F4

- **Severity:** medium
- **Location:** `internal/css/container.go:78-98`; `internal/layout/style.go:1432-1456` (fontSize), `style.go:1496-1549` (lengthBox), `style.go:1541-1585` (marginLen), `style.go:1572` (pxToPt); `internal/layout/transform.go:389-430`; `internal/convert/toc.go:42-67`
- **Current (verbatim):**
```go
func lengthToPt(val float64, unit string, fontSizePt float64) (float64, bool) {
	switch strings.ToLower(unit) {
	case "px":
		return val * 0.75, true
	case "pt":
		return val, true
	case "in":
		return val * 72, true
	case "cm":
		return val * 72 / 2.54, true
	case "mm":
		return val * 72 / 25.4, true
	case "pc":
		return val * 12, true
	case "em", "rem":
		if unit == "rem" {
			return val * 16 * 0.75, true // 16px root
		}
		return val * fontSizePt, true
```
(css/container.go:80-96). The same switch is hand-rewritten in `fontSize` (style.go:1432-1452), `lineHeight` (style.go:1437-1456, whose `default` arm does `val * 72 / 25.4`, i.e. any unknown unit such as `vw` silently computes as millimetres!), `lengthBox` (style.go:1496-1549), `marginLen` (style.go:1541-1585), `parseTransformLength` (transform.go:389-430), and `convert/toc.go:45-67` re-encodes px as `* 72 / 96` and brings its own constants.
- **Future:** promote the copy already inside the css package and make the rest callers:
```go
// css/container.go — export the existing helper; basePt is the em/rem base.
// Same conversions, no new deps.
func LengthToPt(val float64, unit string, basePt float64) (float64, bool) { … }
```
Each layout converter keeps only its per-property policy:
```go
func lineHeight(value string, fs float64) float64 {
	if value == "normal" {
		return 0
	}
	if v, ok := css.ParseNumber(value); ok {
		return v * fs
	}
	if v, unit, ok := css.ParseLength(value); ok {
		if pt, ok := css.LengthToPt(v, unit, fs); ok {
			return pt
		}
		if unit == "%" {
			return fs * v / 100
		}
	}
	return 0
}
```
For `line-height` the unknown-unit `default:` arm disappears (unknown units now inherit instead of pretending to be mm). `convert/toc.go`'s `lengthToPt` becomes a thin wrapper around `css.LengthToPt` (it keeps its own `-1` fallback semantics for callers).
- **Why:** "CSS unit → points" is css-package domain knowledge (the parser decides px = 96 dpi, em vs rem bases, ex/ch support), yet today it is pasted into four callers, each partially out of sync (`ex`, `rem`-base, vw/vh, the mm-default). One authoritative `LengthToPt` gives a single place where a conversion change lands and by construction deletes the `lineHeight` mm-default bug. Behavior change is limited to unknown units in `line-height` (vw-style), which currently render as millimeters; hypothesis: no fixture passes such a value — validate with `go test ./internal/layout/... ./internal/convert/...` (a golden diff would surface any reliance).

---

<a id="p3-04"></a>
## P3-04 — var() resolution semantics live on one side of the css/layout seam

- [~] **P3-04** — css.ResolveCustomProps done (fix-css); layout mergeCustomProps side pending (fix-layout)

- **Locations:** `internal/css/css.go:1416-1439` (ResolveVar) vs `internal/layout/style.go:319-397` (mergeCustomProps)
- **Evidence sources:** area-4 F5

---

### Evidence — F5

- **Severity:** medium
- **Location:** `internal/css/css.go:1416-1439` (ResolveVar) vs `internal/layout/style.go:319-397` (mergeCustomProps)
- **Current (verbatim):**
```go
func ResolveVar(v string, lookup func(name string) (string, bool)) string {
	v = strings.TrimSpace(v)
	for depth := 0; depth < 16; depth++ {
		if !strings.HasPrefix(strings.ToLower(v), "var(") {
			return v
		}
		name, fallback, ok := parseVarFn(v)
		if !ok {
			return v
		}
		if lookup != nil {
			if val, found := lookup(name); found && strings.TrimSpace(val) != "" {
				v = strings.TrimSpace(val)
				continue
			}
		}
		if fallback != "" {
			v = fallback
			continue
		}
		return ""
	}
	return v
}
```
Meanwhile `style.go:319-397` hand-rolls a *second* var() evaluation policy — cycle detection via a `stack` map, memoization, inherited-vs-declared merge — calling back into `css.ResolveVar` one level at a time. CSS-variable semantics (cycle → invalid, fallback extraction, chained expansion) live in two places with two termination policies (bounded depth 16 vs stack-based cycle detection), so a deep chain can resolve differently depending on which implementation catches it, and any spec change has to be made twice.
- **Future:** move the whole custom-property graph walk into the css package; layout delegates:
```go
// css.go — the CSS Variables 1 walk: inherited overlay + declared, var()
// chains expanded once with memo + cycle stack + fallback. The single place
// custom-property policy lives.
func ResolveCustomProps(declared, inherited map[string]string) map[string]string {
	work := make(map[string]string, len(inherited)+len(declared))
	for k, v := range inherited {
		work[k] = v
	}
	for k, v := range declared {
		work[k] = v
	}
	memo := map[string]string{}
	var eval func(name string, stack map[string]bool) string
	eval = func(name string, stack map[string]bool) string {
		if v, ok := memo[name]; ok {
			return v
		}
		raw, ok := work[name]
		if !ok || !strings.Contains(strings.ToLower(raw), "var(") {
			memo[name] = raw
			return raw
		}
		if stack[name] {
			return ""
		}
		stack[name] = true
		val := ResolveVar(raw, func(n string) (string, bool) {
			s := eval(n, stack)
			_, exists := work[n]
			return s, exists && strings.TrimSpace(s) != ""
		})
		delete(stack, name)
		memo[name] = val
		return val
	}
	for name := range work {
		eval(name, map[string]bool{})
	}
	return memo
}
```
`style.go`'s `mergeCustomProps` shrinks to: gather `--*` keys from raw, return `parent.CustomProps` when none are declared, else `css.ResolveCustomProps(declared, inherited)`. `resolveRawVars` (style.go:381-410) stays as the plain property-value pass over the map css produced.
- **Why:** whether a custom property is a cycle or resolves to a usable value is css semantics; today a layout-side recursion (stack/memo) plus a depth-16 loop in the css package can disagree (a 20-hop chain resolves under the stack policy but truncates under ResolveVar's 16 bound if anyone reroutes the property pass). Consolidating makes the policy testable at the css seam (cycle maps in css_test.go) and removes the parallel implementation. No behavior change for acyclic sheets; deep-chain/cycle sheets produce the same result by construction because the stack walk was the stricter one.

---

<a id="p3-05"></a>
## P3-05 — Collapse layout's four near-identical style surfaces + hand-rolled cascade

- [ ] **P3-05** — pending (fix-layout never landed)

- **Locations:** `internal/layout/style.go:222-270` (resolveStyles/resolveStylesOpts/resolveStylesWithContainers/resolveStylesWithContainersOpts) and `internal/layout/layout.go:266-285` (measure→re-cascade→re-measure→compare in nested ifs)
- **Evidence sources:** area-4 F6

---

### Evidence — F6

- **Severity:** low
- **Location:** `internal/layout/style.go:222-270` (resolveStyles/resolveStylesOpts/resolveStylesWithContainers/resolveStylesWithContainersOpts) and `internal/layout/layout.go:266-285` (measure→re-cascade→re-measure→compare in nested ifs)
- **Current (verbatim):**
```go
func resolveStyles(root *html.Node, sheets []*css.Stylesheet, media string, viewportW, viewportH float64) map[*html.Node]ResolvedStyle {
	return resolveStylesCtx(root, &styleContext{
		sheets: sheets, media: media, viewportW: viewportW, viewportH: viewportH,
	})
}

// resolveStylesOpts is like resolveStyles but honors layout operator policies
// (e.g. PrintLinkUnderline) carried on Options.
func resolveStylesOpts(root *html.Node, opts Options) map[*html.Node]ResolvedStyle {
	return resolveStylesCtx(root, &styleContext{
		sheets:             opts.Sheets,
		media:              opts.Media,
		viewportW:          opts.Width,
		viewportH:          opts.Height,
		printLinkUnderline: opts.PrintLinkUnderline,
	})
}
```
…and `resolveStylesWithContainers` (style.go:242-254) + `resolveStylesWithContainersOpts` (style.go:256-270) repeat the same struct literal with the `containers` map. Only the two `…Opts` forms are used in production (`layout.go:27,33,37,42`); the other two exist for ~12 test call sites (`layout_test.go:846`, `grid_test.go:316`, `container_test.go:21,60,117…`, `has_test.go:16`, `logo_stack_test.go:48`, etc.). Then `layout.go:266-285` hand-rolls the refinement: measure → re-cascade → re-measure → compare maps with a len-then-per-entry compare inside nested ifs.
- **Future:** one entry point + one bounded refinement loop:
```go
// style.go — the single cascade entry; the four prior functions were this
// same struct literal in four argument shapes.
func resolveStylesWith(root *html.Node, opts Options, containers map[*html.Node]sizeContainer) map[*html.Node]ResolvedStyle {
	return resolveStylesCtx(root, &styleContext{
		sheets:             opts.Sheets,
		media:              opts.Media,
		viewportW:          opts.Width,
		viewportH:          opts.Height,
		printLinkUnderline: opts.PrintLinkUnderline,
		containers:         containers,
	})
}

// layout.go
styles := resolveStylesWith(root, opts, nil)
if css.HasContainerRules(opts.Sheets) {
	styles = cascadeContainers(root, opts, styles)
}

// cascadeContainers re-cascades with measured size containers until sizes
// stabilize. Mirrors layout.go:266-285 (pass1 + up to two remounts).
func cascadeContainers(root *html.Node, opts Options, start map[*html.Node]ResolvedStyle) map[*html.Node]ResolvedStyle {
	cur := start
	for pass := 0; pass < 2; pass++ {
		next := resolveStylesWith(root, opts, measureSizeContainers(root, cur, opts.Width))
		if sizeContainersEqual(measureSizeContainers(root, next, opts.Width),
			measureSizeContainers(root, cur, opts.Width)) {
			return cur // stable: keep the last applied styles
		}
		cur = next
	}
	return cur
}
```
Tests that need a raw pass call `resolveStylesWith(root, opts, nil)` (they already build `Options{Width:…, Height:…, Sheets:…, Media:…}`).
- **Why:** the four-entry surface means a cascade change (new ResolvedStyle field, new operator policy) must be threaded through four struct literals and understood across ~12 call sites; which entry takes `Options` vs tuples is a recurring reader tax. The re-cascade logic is sequence control a 12-line helper expresses once; outputs stay identical because the loop is the same three-render schedule (start + 2 remounts) and `sizeContainersEqual` is exactly today's len-then-field compare. Bonus: a non-convergence guard for pathological sheets would now have one home (cascadeContainers).

---

<a id="p3-06"></a>
## P3-06 — One rule-body parser: merge Parse's top loop into parseRuleList

- [x] **P3-06** — done (fix-css)

- **Locations:** `internal/css/css.go:97-232` (Parse) vs `internal/css/css.go:276-353` (parseRuleList)
- **Evidence sources:** area-4 F7

---

### Evidence — F7

- **Severity:** low
- **Location:** `internal/css/css.go:97-232` (Parse) vs `internal/css/css.go:276-353` (parseRuleList)
- **Current (verbatim):**
```go
		selText := strings.TrimSpace(src[:selEnd])
		block, rest, err := takeBlock(src, selEnd)
		if err != nil {
			return nil, err
		}
		src = rest
		if selText == "" {
			continue
		}
		sel, ok := parseSelectorList(selText)
		if !ok || len(sel) == 0 {
			continue
		}
		s.Rules = append(s.Rules, Rule{
			Selectors: sel,
			Decls:     parseDeclarations(block),
			Media:     "all",
			Order:     order,
		})
		order++
```
(Parse's top-level rule path, css.go:205-227). `parseRuleList` repeats the same findBlock → takeBlock → parseSelectorList → parseDeclarations → `Rule{…}` → `*orderPtr++` sequence a few lines later (css.go:329-353), differing only in `media`/`cq` handling — and the two have already drifted: Parse checks `selText == ""` separately, while parseRuleList folded empties into the same `ok` path. The skip-an-at-rule dance (`IndexByte('{')` → takeBlock → discard, else `;`-scan) is repeated 5× in Parse for `@page`/`@keyframes`/`@font-face`/unknown at-rules, and again inside parseRuleList's unknown-at-rule arm.
- **Future:** extract one rule-body function and one skip-statement helper:
```go
// parseOneRule builds a Rule from selector text + declaration block, owning
// the order counter. Shared by Parse and parseRuleList.
func parseOneRule(selText, block, media string, cq *ContainerQuery, order *int) (Rule, bool) {
	sel, ok := parseSelectorList(selText)
	if !ok || len(sel) == 0 {
		return Rule{}, false
	}
	r := Rule{
		Selectors: sel,
		Decls:     parseDeclarations(block),
		Media:     media,
		Order:     *order,
	}
	if cq != nil {
		cp := *cq
		r.Container = &cp
	}
	*order++
	return r, true
}
```
`Parse` and `parseRuleList` both loop `if r, ok := parseOneRule(…); ok { … }`; the at-rule skips become `skipAtRule(src) (rest string, err error)` used in both functions.
- **Why:** the css package's core invariants are source-order semantics and @container flattening; today rule-body construction is written twice, so an ordering change must be audited in both loops. parseOneRule makes ordering/Container assignment a single fact; skipAtRule removes ~20 lines of repeated brace accounting. Pure refactor: no public API change; parse outcomes identical if the empty-prelude policy is kept as today (a one-line decision when merging).

## Supplementary stray notes (not findings)
- `ContainerCond`'s `Kind string` + four-arm switch (container.go:31-59): fine for a fixed boolean grammar; a table would obscure.
- `parseContainerNameValue` returns a *space-joined string* that layout re-splits with `strings.Fields` in two places (style.go:1098, findSizeContainer at layout/container.go:19-24): the encoding handshake crosses the seam as an untyped string. A `type ContainerNames []string` with a `Matches(name string) bool` method would seal it; pairs with F6's container work.
- `splitMediaList` (media.go:20-43) vs `splitTopLevel` (css.go:284-307): two comma-splitters in the same package, only one quote-aware. If a future media feature embeds a string (e.g. `(format("woff2"))`), media.go will split inside it. A one-helper collapse.
- `ParseColor` resolves `var(...)` fallbacks itself (css.go:1120-1146) while layout generally pre-resolves vars via `css.ResolveVar`. Two var() paths to keep in mind; F5 consolidates the policy and this second path can then be deleted.

_Written by explore agent 4 (CSS engine), read-only; all "Current" blocks verbatim; "Future" blocks are illustrative with explicit behavior notes; hypotheses are labelled hypothesis: … validate by …._

---
