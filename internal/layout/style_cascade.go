package layout

import (
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

const (
	borderProperty       = "border"
	borderTopProperty    = "border-top"
	borderBottomProperty = "border-bottom"
	borderLeftProperty   = "border-left"
	borderRightProperty  = "border-right"
)

// mergeCustomProps inherits parent custom properties and overlays any --*
// declarations from raw, resolving var() chains via css.ResolveCustomProps.
func mergeCustomProps(parentProps map[string]string, raw map[string]string) map[string]string {
	var declared map[string]string

	for prop, value := range raw {
		if strings.HasPrefix(prop, "--") {
			if declared == nil {
				declared = make(map[string]string)
			}

			declared[prop] = value
		}
	}

	if len(declared) == 0 {
		return parentProps
	}

	return css.ResolveCustomProps(declared, parentProps)
}

// resolveRawVars expands var() in cascaded property values using customProps.
// Custom property keys (--*) are left unchanged (already resolved in the map).
// When no value contains var(), raw is returned as-is (no map copy).
//
//nolint:cyclop // hot path; flat scan then expand of var() refs stays readable
func resolveRawVars(raw map[string]string, customProps map[string]string) map[string]string {
	if len(raw) == 0 {
		return raw
	}

	needs := false

	for prop, val := range raw {
		if strings.HasPrefix(prop, "--") {
			continue
		}

		if containsVarFunc(val) {
			needs = true

			break
		}
	}

	if !needs {
		return raw
	}

	lookup := func(name string) (string, bool) {
		if customProps == nil {
			return "", false
		}

		v, ok := customProps[name]

		return v, ok && strings.TrimSpace(v) != ""
	}
	out := make(map[string]string, len(raw))

	for prop, val := range raw {
		if strings.HasPrefix(prop, "--") {
			out[prop] = val

			continue
		}

		if containsVarFunc(val) {
			out[prop] = css.ResolveVars(val, lookup)
		} else {
			out[prop] = val
		}
	}

	return out
}

// containsVarFunc reports whether s has a CSS var( function (ASCII, case-insensitive)
// without allocating a lowercased copy of s.
func containsVarFunc(s string) bool {
	for i := 0; i+4 <= len(s); i++ {
		if (s[i]|asciiFoldBit) == 'v' && (s[i+1]|asciiFoldBit) == 'a' && (s[i+2]|asciiFoldBit) == 'r' && s[i+3] == '(' {
			return true
		}
	}

	return false
}

// blockifyDisplayForFloat maps specified display to the used value when
// float is left|right (CSS2.1 §9.7). table stays table (floated table
// wrapper); table-cell/row/… and inlines become block.
func blockifyDisplayForFloat(decl string) string {
	switch decl {
	case "inline", "inline-block", "inline-table", "inline-flex", "inline-grid",
		"run-in", "table-row-group", "table-header-group", "table-footer-group",
		"table-row", "table-cell", "table-caption", "table-column",
		"table-column-group", "list-item":
		return displayBlock
	default:
		return decl
	}
}

// inheritCopy is one inheritable property group: the CSS property names
// (declaring any of them on the element suppresses the copy) and the closure
// that copies the parent's resolved value into the child.
type inheritCopy struct {
	names []string
	copy  func(dst, src *ResolvedStyle)
}

// inheritableProps is the immutable inherit table used by inheritProps.
// Package-level so inheritProps does not allocate a new slice, name slices,
// and closures on every styled node (was ~40% of alloc_objects on 500-page PDF).
var inheritableProps = []inheritCopy{ //nolint:gochecknoglobals // static inherit table
	{[]string{"color"}, func(dst, src *ResolvedStyle) { dst.Color = src.Color }},
	{[]string{"accent-color"}, func(dst, src *ResolvedStyle) {
		dst.AccentColor = src.AccentColor
		dst.AccentColorSet = src.AccentColorSet
	}},
	{[]string{"font-family"}, func(dst, src *ResolvedStyle) { dst.FontFamily = src.FontFamily }},
	{[]string{"font-size"}, func(dst, src *ResolvedStyle) { dst.FontSize = src.FontSize }},
	{[]string{"font-weight"}, func(dst, src *ResolvedStyle) { dst.FontWeight = src.FontWeight }},
	{[]string{"font-style"}, func(dst, src *ResolvedStyle) { dst.FontItalic = src.FontItalic }},
	{[]string{"line-height"}, func(dst, src *ResolvedStyle) {
		dst.LineHeight = src.LineHeight
		dst.LineHeightUnitless = src.LineHeightUnitless
	}},
	{[]string{"text-align"}, func(dst, src *ResolvedStyle) { dst.TextAlign = src.TextAlign }},
	{[]string{"text-transform"}, func(dst, src *ResolvedStyle) { dst.TextTransform = src.TextTransform }},
	{[]string{"white-space"}, func(dst, src *ResolvedStyle) { dst.WhiteSpace = src.WhiteSpace }},
	// overflow-wrap / word-wrap and word-break are inherited (CSS Text).
	{
		[]string{"overflow-wrap", "word-wrap"},
		func(dst, src *ResolvedStyle) { dst.OverflowWrap = src.OverflowWrap },
	},
	{[]string{"word-break"}, func(dst, src *ResolvedStyle) { dst.WordBreak = src.WordBreak }},
	{
		[]string{"vertical-align"},
		func(dst, src *ResolvedStyle) {
			dst.VerticalAlign = src.VerticalAlign
			dst.VerticalAlignShift = src.VerticalAlignShift
		},
	},
	{
		[]string{"text-decoration"},
		func(dst, src *ResolvedStyle) { dst.TextDecoration = src.TextDecoration },
	},
	{
		[]string{"letter-spacing"},
		func(dst, src *ResolvedStyle) { dst.LetterSpacing = src.LetterSpacing },
	},
	{
		[]string{"word-spacing"},
		func(dst, src *ResolvedStyle) { dst.WordSpacing = src.WordSpacing },
	},
	{[]string{"visibility"}, func(dst, src *ResolvedStyle) { dst.Visibility = src.Visibility }},
	{[]string{"caption-side"}, func(dst, src *ResolvedStyle) { dst.CaptionSide = src.CaptionSide }},
	{
		[]string{"list-style-type", "list-style"},
		func(dst, src *ResolvedStyle) { dst.ListStyleType = src.ListStyleType },
	},
	{
		[]string{"border-collapse"},
		func(dst, src *ResolvedStyle) { dst.BorderCollapse = src.BorderCollapse },
	},
	{
		[]string{"border-spacing"},
		func(dst, src *ResolvedStyle) { dst.BorderSpacing = src.BorderSpacing },
	},
	{[]string{"orphans"}, func(dst, src *ResolvedStyle) { dst.Orphans = src.Orphans }},
	{[]string{"widows"}, func(dst, src *ResolvedStyle) { dst.Widows = src.Widows }},
	{[]string{"writing-mode"}, func(dst, src *ResolvedStyle) { dst.WritingMode = src.WritingMode }},
	{[]string{"text-indent"}, func(dst, src *ResolvedStyle) { dst.TextIndent = src.TextIndent }},
}

// inheritProps copies inheritable properties from the parent, unless the
// element declares its own value (present in raw).
func inheritProps(dst *ResolvedStyle, parent *ResolvedStyle, raw map[string]string) {
	if parent == nil {
		return
	}

	for _, entry := range inheritableProps {
		declared := false

		if raw != nil {
			for _, name := range entry.names {
				if _, ok := raw[name]; ok {
					declared = true

					break
				}
			}
		}

		if !declared {
			entry.copy(dst, parent)
		}
	}
}

// ruleHit is one selector match from the shared cascade rule walk.
type ruleHit struct {
	r       css.Rule
	a, b, c int
}

// matchedRules walks sheets with the cascade's gates (media, @container,
// selector match, specificity). pseudoElem != "" matches ::before/::after
// shapes instead of the element (pseudo-content path).
//
//nolint:wsl // cascade gates are intentionally evaluated in source order.
func (ctx *styleContext) matchedRules(node *html.Node, pseudoElem string) []ruleHit {
	if ctx == nil {
		return nil
	}
	if ctx.pollContext() {
		return nil
	}

	// Reuse one growable buffer across sequential element lookups. The caller
	// consumes the returned slice before resolving the next element.
	hits := ctx.ruleHits[:0]
	for _, sheet := range ctx.sheets {
		if ctx.pollContext() {
			return nil
		}
		hits = ctx.appendSheetRuleHits(hits, sheet, node, pseudoElem)
	}

	ctx.ruleHits = hits

	return hits
}

// appendSheetRuleHits appends matches from one stylesheet into hits.
//
//nolint:wsl // cascade gates are intentionally evaluated in source order.
func (ctx *styleContext) appendSheetRuleHits(
	hits []ruleHit, sheet *css.Stylesheet, node *html.Node, pseudoElem string,
) []ruleHit {
	if sheet == nil {
		return hits
	}

	for _, rule := range sheet.Rules {
		if ctx.pollContext() {
			return hits
		}
		if !css.MediaMatches(rule.Media, ctx.media, ctx.viewportW, ctx.viewportH) {
			continue
		}

		if !ctx.containerGateMatches(node, rule) {
			continue
		}

		hits = ctx.appendRuleSelectorHits(hits, rule, node, pseudoElem)
	}

	return hits
}

// appendRuleSelectorHits appends matching selectors of one rule into hits.
//
//nolint:wsl // selector gates are intentionally evaluated in source order.
func (ctx *styleContext) appendRuleSelectorHits(
	hits []ruleHit, rule css.Rule, node *html.Node, pseudoElem string,
) []ruleHit {
	for _, sel := range rule.Selectors {
		if ctx.pollContext() {
			return hits
		}
		if !selectorMatches(sel, node, pseudoElem) {
			continue
		}

		a, b, c := css.Specificity(sel)
		hits = append(hits, ruleHit{r: rule, a: a, b: b, c: c})
	}

	return hits
}

// selectorMatches reports whether sel matches node, using the pseudo-shape
// matcher when pe is non-empty.
func selectorMatches(sel css.Selector, node *html.Node, pe string) bool {
	if pe != "" {
		return css.MatchPseudo(sel, node, pe)
	}

	return css.Match(sel, node)
}

// containerGateMatches checks the rule's @container query against the nearest
// eligible size container (skipped on passes without container sizes).
func (ctx *styleContext) containerGateMatches(node *html.Node, runic css.Rule) bool {
	if runic.Container == nil {
		return true
	}

	if ctx.containers == nil {
		return false // pass 1 / pseudo pass without sizes: skip
	}

	info, ok := findSizeContainer(node, runic.Container.Name, ctx.containers)

	return ok && runic.Container.Cond.Matches(info.inlineSize, info.fontSize)
}

// cascadeWinHint is the initial capacity for cascade winner maps. Most elements
// win a small handful of properties (UA + a few author rules).
const cascadeWinHint = 8

// cascadeWin is the winning cascaded declaration for one property: value plus
// the specificity/order bits needed to compare later candidates. important is
// a separate cascade layer (any !important beats any normal).
type cascadeWin struct {
	value               string
	ids, classes, types int
	order               int
	important           bool
}

// cascadeRaw returns the winning declaration per property for the element
// across UA sheet, author sheets and the inline style attribute.
// Uses one winner map (value+spec+order+important) instead of six maps.
//
//nolint:cyclop // hot path; three fixed cascade tiers read clearer than one loop
func cascadeRaw( //nolint:funlen // cascade tiers are deliberately visible in one hot-path function
	ctx *styleContext, node *html.Node,
) map[string]string {
	var wins map[string]cascadeWin
	if ctx == nil {
		wins = make(map[string]cascadeWin, cascadeWinHint)
	} else {
		wins = ctx.cascadeWins
		if wins == nil {
			wins = make(map[string]cascadeWin, cascadeWinHint)
			ctx.cascadeWins = wins
		} else {
			clear(wins)
		}
	}

	// UA sheet (lowest priority; specificity 0, order -1)
	for _, d := range uaRules(node.Name) {
		applyCascadeWin(wins, d.Prop, d.Value, 0, 0, 0, -1, false)
	}

	// author sheets in source order (shared matchedRules walk)
	if ctx != nil {
		for _, hit := range ctx.matchedRules(node, "") {
			rule := hit.r
			for _, d := range rule.Decls {
				if !supportedDeclaration(d.Value) {
					continue
				}

				applyCascadeDeclaration(wins, d.Prop, d.Value, hit.a, hit.b, hit.c, rule.Order, d.Important)
			}
		}
	}

	// inline style attribute: outranks all normal declarations and all sheet
	// important declarations (spec 1<<maxIntShift).
	for _, d := range css.ParseInline(node.Attribute("style")) {
		if !supportedDeclaration(d.Value) {
			continue
		}

		applyCascadeDeclaration(wins, d.Prop, d.Value, 1<<maxIntShift, 0, 0, 1<<maxIntShift, d.Important)
	}

	if len(wins) == 0 {
		if ctx != nil && ctx.cascadeProps != nil {
			clear(ctx.cascadeProps)
		}

		return nil
	}

	var out map[string]string
	if ctx == nil {
		out = make(map[string]string, len(wins))
	} else {
		out = ctx.cascadeProps
		if out == nil {
			out = make(map[string]string, len(wins))
			ctx.cascadeProps = out
		} else {
			clear(out)
		}
	}

	for prop, w := range wins {
		out[prop] = w.value
	}

	return out
}

// cascadePseudoRaw returns the winning declarations for a generated
// ::before/::after box. Pseudo-elements have no inline attribute or UA rule of
// their own; their declarations come from the author rules that matched the
// host and pseudo shape.
func cascadePseudoRaw(ctx *styleContext, node *html.Node, pseudoElem string) map[string]string {
	if ctx == nil || node == nil {
		return nil
	}

	wins := ctx.cascadeWins
	if wins == nil {
		wins = make(map[string]cascadeWin, cascadeWinHint)
		ctx.cascadeWins = wins
	} else {
		clear(wins)
	}

	for _, hit := range ctx.matchedRules(node, pseudoElem) {
		for _, d := range hit.r.Decls {
			if !supportedDeclaration(d.Value) {
				continue
			}

			applyCascadeDeclaration(wins, d.Prop, d.Value, hit.a, hit.b, hit.c, hit.r.Order, d.Important)
		}
	}

	if len(wins) == 0 {
		return nil
	}

	out := make(map[string]string, len(wins))
	for prop, win := range wins {
		out[prop] = win.value
	}

	return out
}

// applyCascadeDeclaration expands box shorthands before selecting winners.
// A shorthand and a longhand compete per physical property: the declaration
// that wins by specificity and source order must win that side, regardless of
// which form it used. Keeping the shorthand intact until after the cascade
// made an earlier margin-top declaration override a later margin shorthand.
func applyCascadeDeclaration(
	wins map[string]cascadeWin,
	prop, value string,
	ids, classes, types, order int,
	important bool,
) {
	values, ok := expandBoxShorthand(prop, value)
	if !ok {
		applyCascadeWin(wins, prop, value, ids, classes, types, order, important)

		return
	}

	for idx, side := range [...]string{"top", "right", "bottom", "left"} {
		applyCascadeWin(wins, prop+"-"+side, values[idx], ids, classes, types, order, important)
	}
}

func expandBoxShorthand(prop, value string) ([4]string, bool) {
	var values [4]string

	switch prop {
	case marginProperty, paddingProperty:
		// 1–4 space-separated sides (CSS box shorthand).
	case borderProperty:
		// border: <width> <style> <color> applies the same value to every side.
		// Expand so it competes with border-top/right/bottom/left longhands
		// (otherwise an earlier border-top can paint after a later border).
		values = [4]string{value, value, value, value}

		return values, true
	default:
		return values, false
	}

	var tokens [4]string
	count := splitSpaceTokens(value, tokens[:])

	if count < 1 || count > len(tokens) {
		return values, false
	}

	switch count {
	case 1:
		values = [4]string{tokens[0], tokens[0], tokens[0], tokens[0]}
	case two:
		values = [4]string{tokens[0], tokens[1], tokens[0], tokens[1]}
	case three:
		values = [4]string{tokens[0], tokens[1], tokens[2], tokens[1]}
	default:
		values = tokens
	}

	return values, true
}

// supportedDeclaration rejects modern color functions that this lite renderer
// cannot compute. Excluding them from the cascade preserves an earlier valid
// fallback declaration, matching the fixture's fallback-first contract.
// clamp() is computed by clampLength and is therefore allowed.
func supportedDeclaration(value string) bool {
	value = strings.ToLower(value)

	for _, unsupported := range []string{"color-mix(", "light-dark(", "oklch("} {
		if strings.Contains(value, unsupported) {
			return false
		}
	}

	return true
}

// applyCascadeWin folds one declaration into the winner map when its layer,
// specificity, or source order beats the current winner.
func applyCascadeWin(
	wins map[string]cascadeWin,
	prop, value string, ids, classes, types, order int, important bool,
) {
	// prop is already lowercase: sheet and inline declarations are folded by
	// css.parseDeclarations and the UA table is hard-coded lowercase.
	cur, ok := wins[prop]
	if !ok {
		wins[prop] = cascadeWin{
			value: value, ids: ids, classes: classes, types: types,
			order: order, important: important,
		}

		return
	}

	// !important is a higher cascade layer than normal (any origin here).
	if important != cur.important {
		if !important {
			return
		}

		wins[prop] = cascadeWin{
			value: value, ids: ids, classes: classes, types: types,
			order: order, important: true,
		}

		return
	}

	if specificityBeats([4]int{cur.ids, cur.classes, cur.types, 0}, ids, classes, types, order, cur.order) {
		wins[prop] = cascadeWin{
			value: value, ids: ids, classes: classes, types: types,
			order: order, important: important,
		}
	}
}

// specificityBeats reports whether (ids, classes, types) with the given source
// order wins over the current winning specificity/order.
func specificityBeats(cur [4]int, ids, classes, types, order, curOrder int) bool {
	if ids > cur[0] {
		return true
	}

	if ids < cur[0] {
		return false
	}

	if classes > cur[1] {
		return true
	}

	if classes < cur[1] {
		return false
	}

	if types > cur[2] {
		return true
	}

	if types < cur[2] {
		return false
	}

	return order >= curOrder
}

// applyFontProps resolves font-size/family/weight/style/font first, using the
// parent's size for percentages and em, and ctx.remBase for rem.
func applyFontProps(style *ResolvedStyle, raw map[string]string, parentSize float64, ctx *styleContext) {
	remBase := pxToPt(cssPxRoot)
	if ctx != nil && ctx.remBase > 0 {
		remBase = ctx.remBase
	}

	if v, ok := raw["font-size"]; ok {
		containing := 0.0
		if ctx != nil {
			containing = ctx.viewportW
		}

		if pt, ok := clampLength(v, parentSize, containing); ok {
			style.FontSize = pt
		} else {
			style.FontSize = fontSize(v, parentSize, remBase)
		}
	}

	if v, ok := raw["font-family"]; ok {
		if fam := css.ParseFontFamily(v); len(fam) > 0 {
			style.FontFamily = fam
		}
	}

	if val, ok := raw["font-weight"]; ok {
		style.FontWeight = resolveFontWeight(style.FontWeight, val)
	}

	if v, ok := raw["font-style"]; ok {
		style.FontItalic = v == "italic" || v == "oblique"
	}

	if v, ok := raw["font"]; ok {
		parseFontShorthand(style, v, remBase)
	}
}

// resolveFontWeight maps a font-weight keyword/number onto a weight value.
func resolveFontWeight(current int, val string) int {
	switch val {
	case contentNormal:
		return fontWeightNormal
	case "bold":
		return fontWeightBold
	case "bolder":
		return current + fontWeightStep
	case "lighter":
		return current - fontWeightStep
	default:
		if n, ok := css.ParseNumber(val); ok && n >= 100 && n <= 900 {
			return int(n)
		}
	}

	return current
}

// restShorthandProps are applied before other cascaded properties so a winning
// longhand (e.g. margin-bottom) always overrides its shorthand (margin).
// Package-level to avoid per-node slice/array rebuilds.
var restShorthandProps = [...]string{ //nolint:gochecknoglobals // static apply order
	"margin", "padding", borderProperty, borderTopProperty, borderRightProperty, borderBottomProperty, borderLeftProperty,
	borderWidthKeyword, borderStyleKeyword,
	borderColorKeyword, gapKeyword, flexKeyword, containerKeyword,
	"margin-inline", "margin-block", "padding-inline", "padding-block",
	"inset", "inset-block", "inset-inline",
}

// applyRestProps resolves every non-font property once the font size is known.
// Shorthands run first in a fixed order; remaining longhands run in any order
// (longhands do not clobber each other via shorthand expansion). This avoids
// sorting and intermediate prop slices on every element.
func applyRestProps(
	style *ResolvedStyle, raw map[string]string, ctx *styleContext,
	parent *ResolvedStyle,
) {
	if len(raw) == 0 {
		return
	}

	fsize := style.FontSize
	hasParent := parent != nil

	for i := range restShorthandProps {
		prop := restShorthandProps[i]

		value, ok := raw[prop]
		if !ok {
			continue
		}

		applyStyleProp(style, prop, value, fsize, ctx, parent, hasParent)
	}

	for prop, value := range raw {
		switch prop {
		case "margin", "padding", borderProperty, borderTopProperty,
			borderRightProperty, borderBottomProperty, borderLeftProperty,
			borderWidthKeyword, borderStyleKeyword,
			borderColorKeyword, gapKeyword, flexKeyword, containerKeyword,
			"margin-inline", "margin-block", "padding-inline", "padding-block",
			"inset", "inset-block", "inset-inline":
			continue
		}

		applyStyleProp(style, prop, value, fsize, ctx, parent, hasParent)
	}
}

// styleGroupFn is one property-group handler in the applyStyleProp dispatch.
// Groups return false when they do not own prop.
type styleGroupFn func(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext,
	parent *ResolvedStyle, hasParent bool,
) bool

// styleGroups is the immutable dispatch order for applyStyleProp.
// Package-level so applyStyleProp does not rebuild the 11-entry array on
// every cascaded property of every element.
var styleGroups = [...]styleGroupFn{ //nolint:gochecknoglobals // static dispatch table
	applyDisplayGroup,
	applyPositionGroup,
	applyFlexGroup,
	applyMulticolGroup,
	applyGridGroup,
	applyBoxGroup,
	applyBorderGroup,
	applyColorGroup,
	applyTextGroup,
	applyTableBreakGroup,
	applyTransformGroup,
}

// applyStyleProp routes one cascaded property to the group that owns it.
func applyStyleProp(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext,
	parent *ResolvedStyle, hasParent bool,
) {
	for _, group := range styleGroups {
		if group(style, prop, value, fsize, ctx, parent, hasParent) {
			return
		}
	}

	applyIgnoredGroup(style, prop, value)
}

// applyDisplayGroup handles display, position-adjacent flow and stacking props.
