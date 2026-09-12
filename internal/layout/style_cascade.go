package layout

import (
	"maps"
	"strconv"
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

	marginProperty  = "margin"
	paddingProperty = "padding"

	textEmphasisProperty         = "text-emphasis"
	textEmphasisStyleProperty    = "text-emphasis-style"
	textEmphasisColorProperty    = "text-emphasis-color"
	textEmphasisPositionProperty = "text-emphasis-position"
	textEmphasisSkipProperty     = "text-emphasis-skip"
	textShadowProperty           = "text-shadow"
	tabSizeProperty              = "tab-size"

	inlineStylePriority = 1 << 30

	defaultRootFontPx = 16

	fontWeightNormalValue = 400
	fontWeightBoldValue   = 700

	cssFontStyleItalic  = "italic"
	cssFontStyleOblique = "oblique"
	cssFontWeightBold   = "bold"

	boxShorthandTwoSides   = 2
	boxShorthandThreeSides = 3
)

// internalCustomPropWriters reports whether raw declares a property whose
// applier stores engine bookkeeping in CustomProps. An element that declares
// no --* property inherits the parent's map; when one of these appliers runs,
// mergeCustomProps must hand it a copy or the parent's stored style would be
// mutated after insertion (styleStore interning shares records).
func internalCustomPropWriters(raw map[string]string) bool {
	for prop := range raw {
		switch prop {
		case textEmphasisProperty, textEmphasisStyleProperty, textEmphasisColorProperty,
			textEmphasisPositionProperty, textEmphasisSkipProperty,
			textShadowProperty, tabSizeProperty:
			return true
		}
	}

	return false
}

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
		if len(parentProps) > 0 && internalCustomPropWriters(raw) {
			return maps.Clone(parentProps)
		}

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
	{[]string{"text-align-last"}, func(dst, src *ResolvedStyle) { dst.TextAlignLast = src.TextAlignLast }},
	{[]string{"text-transform"}, func(dst, src *ResolvedStyle) { dst.TextTransform = src.TextTransform }},
	{[]string{"white-space"}, func(dst, src *ResolvedStyle) { dst.WhiteSpace = src.WhiteSpace }},
	{[]string{"white-space-collapse"}, func(dst, src *ResolvedStyle) { dst.WhiteSpaceCollapse = src.WhiteSpaceCollapse }},
	{[]string{"white-space-trim"}, func(dst, src *ResolvedStyle) { dst.WhiteSpaceTrim = src.WhiteSpaceTrim }},
	{[]string{"text-wrap"}, func(dst, src *ResolvedStyle) { dst.TextWrap = src.TextWrap }},
	{[]string{"text-wrap-mode"}, func(dst, src *ResolvedStyle) { dst.TextWrapMode = src.TextWrapMode }},
	{[]string{"text-wrap-style"}, func(dst, src *ResolvedStyle) { dst.TextWrapStyle = src.TextWrapStyle }},
	{[]string{tabSizeProperty}, func(dst, src *ResolvedStyle) { dst.TabSize = src.TabSize }},
	{[]string{"hyphens"}, func(dst, src *ResolvedStyle) { dst.Hyphens = src.Hyphens }},
	{[]string{"hyphenate-character"}, func(dst, src *ResolvedStyle) { dst.HyphenateCharacter = src.HyphenateCharacter }},
	{[]string{"text-justify"}, func(dst, src *ResolvedStyle) { dst.TextJustify = src.TextJustify }},
	{[]string{"line-break"}, func(dst, src *ResolvedStyle) { dst.LineBreak = src.LineBreak }},
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
		[]string{"list-style-position", "list-style"},
		func(dst, src *ResolvedStyle) { dst.ListStylePosition = src.ListStylePosition },
	},
	{
		[]string{"quotes"},
		func(dst, src *ResolvedStyle) {
			dst.QuotesRaw = src.QuotesRaw
			dst.QuotesOpen = src.QuotesOpen
			dst.QuotesClose = src.QuotesClose
		},
	},
	{
		[]string{"border-collapse"},
		func(dst, src *ResolvedStyle) { dst.BorderCollapse = src.BorderCollapse },
	},
	{
		[]string{"border-spacing"},
		func(dst, src *ResolvedStyle) {
			dst.BorderSpacing = src.BorderSpacing
			dst.BorderSpacingV = src.BorderSpacingV
		},
	},
	{[]string{"orphans"}, func(dst, src *ResolvedStyle) { dst.Orphans = src.Orphans }},
	{[]string{"widows"}, func(dst, src *ResolvedStyle) { dst.Widows = src.Widows }},
	// CSS page is not inherited at computed-value time, but auto uses the
	// parent used value. Copying when the property is unspecified matches that.
	{[]string{pageKeyword}, func(dst, src *ResolvedStyle) { dst.PageName = src.PageName }},
	{[]string{"writing-mode"}, func(dst, src *ResolvedStyle) { dst.WritingMode = src.WritingMode }},
	{[]string{"direction"}, func(dst, src *ResolvedStyle) { dst.Direction = src.Direction }},
	{[]string{"text-indent"}, func(dst, src *ResolvedStyle) { dst.TextIndent = src.TextIndent }},
	{[]string{"fill"}, func(dst, src *ResolvedStyle) { dst.Fill = src.Fill; dst.FillSet = src.FillSet }},
	{[]string{"stroke"}, func(dst, src *ResolvedStyle) { dst.Stroke = src.Stroke; dst.StrokeSet = src.StrokeSet }},
	{[]string{"stroke-width"}, func(dst, src *ResolvedStyle) {
		dst.StrokeWidth = src.StrokeWidth
		dst.StrokeWidthSet = src.StrokeWidthSet
	}},
	{[]string{"stroke-linecap"}, func(dst, src *ResolvedStyle) { dst.StrokeLineCap = src.StrokeLineCap }},
	{[]string{"stroke-linejoin"}, func(dst, src *ResolvedStyle) { dst.StrokeLineJoin = src.StrokeLineJoin }},
	{[]string{"stroke-dasharray"}, func(dst, src *ResolvedStyle) { dst.StrokeDashArray = src.StrokeDashArray }},
	{[]string{"stroke-dashoffset"}, func(dst, src *ResolvedStyle) { dst.StrokeDashOffset = src.StrokeDashOffset }},
	{[]string{"stroke-miterlimit"}, func(dst, src *ResolvedStyle) { dst.StrokeMiterLimit = src.StrokeMiterLimit }},
	{[]string{"text-decoration-skip-ink"}, func(dst, src *ResolvedStyle) {
		dst.TextDecorationSkipInk = src.TextDecorationSkipInk
	}},
	{[]string{"empty-cells"}, func(dst, src *ResolvedStyle) { dst.EmptyCells = src.EmptyCells }},
}

// inheritablePropBits maps an inheritable property name to the bit set of its
// inheritableProps entries. Names can be shared between entries (list-style
// appears in the type and position entries), so bits are ORed. Built once from
// the table so inheritProps can fold the element's declarations into one word
// instead of testing every entry against the raw map (about 65 lookups per
// element before this).
//
//nolint:gochecknoglobals // derived from the static inherit table
var inheritablePropBits = func() map[string]uint64 {
	bits := make(map[string]uint64, len(inheritableProps))

	for i, entry := range inheritableProps {
		for _, name := range entry.names {
			bits[name] |= uint64(1) << i
		}
	}

	return bits
}()

// declaredInheritableMask folds the raw declarations into one bit per
// inheritableProps entry. Iterating the raw keys (about 10 per element)
// replaces the per-entry raw map lookups.
func declaredInheritableMask(raw map[string]string) uint64 {
	var mask uint64

	for prop := range raw {
		if bit, ok := inheritablePropBits[prop]; ok {
			mask |= bit
		}
	}

	return mask
}

// inheritProps copies inheritable properties from the parent, unless the
// element declares its own value (present in raw). The declared-set mask is a
// local word; it deliberately does not grow ResolvedStyle, whose byte size is
// pinned by the interning and storage tests.
func inheritProps(dst *ResolvedStyle, parent *ResolvedStyle, raw map[string]string) {
	if parent == nil {
		return
	}

	declared := declaredInheritableMask(raw)

	for i := range inheritableProps {
		if declared&(uint64(1)<<i) != 0 {
			continue
		}

		inheritableProps[i].copy(dst, parent)
	}
}

// ruleHit is one selector match from the shared cascade rule walk. rule
// points into its stylesheet, so pointer identity is the exact rule identity
// the style memo keys on; ruleHit therefore stays comparable and comparable
// field-for-field.
type ruleHit struct {
	rule    *css.Rule
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
func (ctx *styleContext) appendSheetRuleHits(
	hits []ruleHit, sheet *css.Stylesheet, node *html.Node, pseudoElem string,
) []ruleHit {
	if sheet == nil {
		return hits
	}

	for idx := range sheet.Rules {
		if ctx.pollContext() {
			return hits
		}

		rule := &sheet.Rules[idx]
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
	hits []ruleHit, rule *css.Rule, node *html.Node, pseudoElem string,
) []ruleHit {
	for _, sel := range rule.Selectors {
		if ctx.pollContext() {
			return hits
		}
		if !selectorMatches(sel, node, pseudoElem) {
			continue
		}

		a, b, c := css.Specificity(sel)
		hits = append(hits, ruleHit{rule: rule, a: a, b: b, c: c})
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
func (ctx *styleContext) containerGateMatches(node *html.Node, runic *css.Rule) bool {
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
// across UA sheet, the already matched author rules and the inline style
// attribute. Uses one winner map (value+spec+order+important) instead of six
// maps. hits is the matched-rule list, passed in so the caller can reuse it as
// the style memo key.
//
//nolint:cyclop // hot path; three fixed cascade tiers read clearer than one loop
func cascadeRaw( //nolint:funlen // cascade tiers are deliberately visible in one hot-path function
	ctx *styleContext, node *html.Node, hits []ruleHit,
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
	for _, hit := range hits {
		for _, d := range hit.rule.Decls {
			if !supportedDeclaration(d.Value) {
				continue
			}

			applyCascadeDeclaration(wins, d.Prop, d.Value, hit.a, hit.b, hit.c, hit.rule.Order, d.Important)
		}
	}

	// inline style attribute: outranks all normal declarations and all sheet
	// important declarations (spec 1<<maxIntShift).
	for _, d := range css.ParseInline(node.Attribute("style")) {
		if !supportedDeclaration(d.Value) {
			continue
		}

		applyCascadeDeclaration(wins, d.Prop, d.Value, inlineStylePriority, 0, 0, inlineStylePriority, d.Important)
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
		for _, d := range hit.rule.Decls {
			if !supportedDeclaration(d.Value) {
				continue
			}

			applyCascadeDeclaration(wins, d.Prop, d.Value, hit.a, hit.b, hit.c, hit.rule.Order, d.Important)
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
	if expanded, ok := expandFontDeclaration(prop, value); ok {
		for _, item := range expanded {
			applyCascadeWin(wins, item.prop, item.val, ids, classes, types, order, important)
		}

		return
	}

	if expanded, ok := expandListStyleDeclaration(prop, value); ok {
		for _, item := range expanded {
			applyCascadeWin(wins, item.prop, item.val, ids, classes, types, order, important)
		}

		return
	}

	if expanded, ok := expandLogicalBoxDeclaration(prop, value); ok {
		for _, item := range expanded {
			applyCascadeDeclaration(wins, item.prop, item.val, ids, classes, types, order, important)
		}

		return
	}

	values, ok := expandBoxShorthand(prop, value)
	if !ok {
		applyCascadeWin(wins, prop, value, ids, classes, types, order, important)

		return
	}

	// The longhand names are static per shorthand; concatenating them here
	// used to allocate a fresh property string for every side of every
	// shorthand declaration (6.4 MB per 500-page conversion in the profile).
	for idx, longhand := range boxShorthandLonghands(prop) {
		applyCascadeWin(wins, longhand, values[idx], ids, classes, types, order, important)
	}
}

// boxShorthandLonghands returns the four physical longhand names for a box
// shorthand, in top/right/bottom/left order, or the zero array for any other
// property. Callers reach it only after expandBoxShorthand reported success
// for margin, padding, or border. Literal returns keep the names static
// without package globals.
func boxShorthandLonghands(prop string) [4]string {
	switch prop {
	case marginProperty:
		return [4]string{"margin-top", "margin-right", "margin-bottom", "margin-left"}
	case paddingProperty:
		return [4]string{"padding-top", "padding-right", "padding-bottom", "padding-left"}
	case borderProperty:
		return [4]string{"border-top", "border-right", "border-bottom", "border-left"}
	default:
		return [4]string{}
	}
}

type logicalPropDecl struct {
	prop string
	val  string
}

// expandListStyleDeclaration expands the list-style shorthand into its
// type, position, and image longhands so each part joins the cascade with
// the shorthand's own origin and specificity. Without this, the shorthand
// and a weaker-origin longhand (such as the UA disc on ul) coexist as
// separate raw keys and the later-applied longhand wins regardless of
// origin. Only components present in the value expand; the rest keep
// whatever the cascade decided.
func expandListStyleDeclaration(prop, value string) ([]logicalPropDecl, bool) {
	if prop != "list-style" {
		return nil, false
	}

	var out []logicalPropDecl

	for _, tok := range strings.Fields(value) {
		if typ := parseListStyleType(tok); typ != "" {
			out = append(out, logicalPropDecl{prop: "list-style-type", val: typ})

			break
		}
	}

	for _, tok := range strings.Fields(value) {
		if pos := parseListStylePosition(tok); pos != "" {
			out = append(out, logicalPropDecl{prop: "list-style-position", val: pos})

			break
		}
	}

	if url, ok := firstCSSUrl(value); ok {
		out = append(out, logicalPropDecl{prop: "list-style-image", val: `url("` + url + `")`})
	} else if listStyleImageNone(value) {
		out = append(out, logicalPropDecl{prop: "list-style-image", val: "none"})
	}

	if len(out) == 0 {
		return nil, false
	}

	return out, true
}

// expandFontDeclaration expands the font shorthand into its size,
// line-height, style, weight, and family longhands so each component joins
// the cascade with the shorthand's own origin, specificity, and source
// order. The shorthand used to be applied after every longhand, so an
// earlier `font` beat a later `font-size` regardless of order. Values the
// expansion cannot read (var() references, system fonts) keep the raw key
// and fall back to parseFontShorthand after resolveRawVars.
func expandFontDeclaration(prop, value string) ([]logicalPropDecl, bool) {
	if prop != "font" {
		return nil, false
	}

	parts := strings.Fields(value)
	out := make([]logicalPropDecl, 0, len(parts))

	for idx := range parts {
		tok := parts[idx]

		// font-size with an attached line-height, e.g. 12pt/1.4.
		if strings.Contains(tok, "/") {
			return expandFontSizeToken(out, parts, idx), true
		}

		lower := strings.ToLower(tok)

		if decl, handled := fontPrefixDecl(lower); handled {
			// Style, variant, weight, or stretch keyword positions. normal is
			// the initial value; variant and stretch have no readers.
			if decl.prop != "" {
				out = append(out, decl)
			}

			continue
		}

		if isFontWeightNumber(tok) {
			out = append(out, logicalPropDecl{prop: "font-weight", val: tok})

			continue
		}

		// First token that is not a prefix component starts the required size.
		out = append(out, logicalPropDecl{prop: "font-size", val: tok})

		return appendFontFamilyTail(out, parts, idx), true
	}

	// A valid font shorthand requires a size. Missing or unreadable values
	// stay intact for the post-cascade fallback.
	return nil, false
}

// expandFontSizeToken expands one size token that carries a line-height, e.g.
// 12pt/1.4, plus the family tokens that follow it, onto out.
func expandFontSizeToken(out []logicalPropDecl, parts []string, idx int) []logicalPropDecl {
	size, line, _ := strings.Cut(parts[idx], "/")
	out = append(out, logicalPropDecl{prop: "font-size", val: size})

	if line != "" {
		out = append(out, logicalPropDecl{prop: "line-height", val: line})
	}

	return appendFontFamilyTail(out, parts, idx)
}

// fontPrefixDecl returns the longhand a font shorthand prefix keyword sets.
// handled reports whether lower is a prefix keyword at all; keywords whose
// declaration is empty (normal, variant, and stretch positions) are handled
// but contribute no longhand.
func fontPrefixDecl(lower string) (logicalPropDecl, bool) {
	switch lower {
	case cssFontStyleItalic, cssFontStyleOblique:
		return logicalPropDecl{prop: "font-style", val: lower}, true
	case cssFontWeightBold, "bolder", "lighter":
		return logicalPropDecl{prop: "font-weight", val: lower}, true
	case contentNormal, "small-caps", "condensed", "expanded",
		"semi-condensed", "semi-expanded", "ultra-condensed", "ultra-expanded":
		return logicalPropDecl{}, true //nolint:exhaustruct // intentional empty keyword position
	}

	return logicalPropDecl{}, false //nolint:exhaustruct // intentional empty keyword position
}

// isFontWeightNumber reports whether tok is a numeric font weight 100..900.
func isFontWeightNumber(tok string) bool {
	n, ok := css.ParseNumber(tok)

	return ok && n >= 100 && n <= 900
}

// appendFontFamilyTail appends the tokens after idx as font-family, if any.
func appendFontFamilyTail(out []logicalPropDecl, parts []string, idx int) []logicalPropDecl {
	if idx+1 < len(parts) {
		out = append(out, logicalPropDecl{prop: "font-family", val: strings.Join(parts[idx+1:], " ")})
	}

	return out
}

// expandLogicalBoxDeclaration expands logical margin/padding/inset/border
// declarations to their physical counterpart properties for horizontal-tb.
func expandLogicalBoxDeclaration(prop, value string) ([]logicalPropDecl, bool) {
	if decls, ok := expandLogicalMarginPadding(prop, value); ok {
		return decls, true
	}

	if decls, ok := expandLogicalInset(prop, value); ok {
		return decls, true
	}

	return expandLogicalBorder(prop, value)
}

//nolint:cyclop // logical margin/padding mapping table
func expandLogicalMarginPadding(prop, value string) ([]logicalPropDecl, bool) {
	switch prop {
	case cssPropMarginBlock:
		start, end, ok := logicalPair(value)
		if !ok {
			return nil, false
		}

		return []logicalPropDecl{{"margin-top", start}, {"margin-bottom", end}}, true
	case cssPropMarginInline:
		start, end, ok := logicalPair(value)
		if !ok {
			return nil, false
		}

		return []logicalPropDecl{{"margin-left", start}, {"margin-right", end}}, true
	case "margin-block-start":
		return []logicalPropDecl{{"margin-top", value}}, true
	case "margin-block-end":
		return []logicalPropDecl{{"margin-bottom", value}}, true
	case "margin-inline-start":
		return []logicalPropDecl{{"margin-left", value}}, true
	case "margin-inline-end":
		return []logicalPropDecl{{"margin-right", value}}, true
	case cssPropPaddingBlock:
		start, end, ok := logicalPair(value)
		if !ok {
			return nil, false
		}

		return []logicalPropDecl{{"padding-top", start}, {"padding-bottom", end}}, true
	case cssPropPaddingInline:
		start, end, ok := logicalPair(value)
		if !ok {
			return nil, false
		}

		return []logicalPropDecl{{"padding-left", start}, {"padding-right", end}}, true
	case "padding-block-start":
		return []logicalPropDecl{{"padding-top", value}}, true
	case "padding-block-end":
		return []logicalPropDecl{{"padding-bottom", value}}, true
	case "padding-inline-start":
		return []logicalPropDecl{{"padding-left", value}}, true
	case "padding-inline-end":
		return []logicalPropDecl{{"padding-right", value}}, true
	default:
		return nil, false
	}
}

func expandLogicalInset(prop, value string) ([]logicalPropDecl, bool) {
	switch prop {
	case cssPropInsetBlock:
		start, end, ok := logicalPair(value)
		if !ok {
			return nil, false
		}

		return []logicalPropDecl{{"top", start}, {"bottom", end}}, true
	case cssPropInsetInline:
		start, end, ok := logicalPair(value)
		if !ok {
			return nil, false
		}

		return []logicalPropDecl{{"left", start}, {"right", end}}, true
	case cssPropInsetBlockStart:
		return []logicalPropDecl{{"top", value}}, true
	case cssPropInsetBlockEnd:
		return []logicalPropDecl{{"bottom", value}}, true
	case cssPropInsetInlineStart:
		return []logicalPropDecl{{"left", value}}, true
	case cssPropInsetInlineEnd:
		return []logicalPropDecl{{"right", value}}, true
	default:
		return nil, false
	}
}

//nolint:cyclop,funlen // logical border mapping table
func expandLogicalBorder(prop, value string) ([]logicalPropDecl, bool) {
	switch prop {
	case cssPropBorderBlock:
		return []logicalPropDecl{{"border-top", value}, {"border-bottom", value}}, true
	case cssPropBorderInline:
		return []logicalPropDecl{{"border-left", value}, {"border-right", value}}, true
	case cssPropBorderBlockStart:
		return []logicalPropDecl{{"border-top", value}}, true
	case cssPropBorderBlockEnd:
		return []logicalPropDecl{{"border-bottom", value}}, true
	case cssPropBorderInlineStart:
		return []logicalPropDecl{{"border-left", value}}, true
	case cssPropBorderInlineEnd:
		return []logicalPropDecl{{"border-right", value}}, true
	case cssPropBorderBlockColor:
		return []logicalPropDecl{{"border-top-color", value}, {"border-bottom-color", value}}, true
	case cssPropBorderInlineColor:
		return []logicalPropDecl{{"border-left-color", value}, {"border-right-color", value}}, true
	case cssPropBorderBlockStartColor:
		return []logicalPropDecl{{"border-top-color", value}}, true
	case cssPropBorderBlockEndColor:
		return []logicalPropDecl{{"border-bottom-color", value}}, true
	case cssPropBorderInlineStartColor:
		return []logicalPropDecl{{"border-left-color", value}}, true
	case cssPropBorderInlineEndColor:
		return []logicalPropDecl{{"border-right-color", value}}, true
	case cssPropBorderBlockStyle:
		return []logicalPropDecl{{"border-top-style", value}, {"border-bottom-style", value}}, true
	case cssPropBorderInlineStyle:
		return []logicalPropDecl{{"border-left-style", value}, {"border-right-style", value}}, true
	case cssPropBorderBlockStartStyle:
		return []logicalPropDecl{{"border-top-style", value}}, true
	case cssPropBorderBlockEndStyle:
		return []logicalPropDecl{{"border-bottom-style", value}}, true
	case cssPropBorderInlineStartStyle:
		return []logicalPropDecl{{"border-left-style", value}}, true
	case cssPropBorderInlineEndStyle:
		return []logicalPropDecl{{"border-right-style", value}}, true
	case cssPropBorderBlockWidth:
		return []logicalPropDecl{{"border-top-width", value}, {"border-bottom-width", value}}, true
	case cssPropBorderInlineWidth:
		return []logicalPropDecl{{"border-left-width", value}, {"border-right-width", value}}, true
	case cssPropBorderBlockStartWidth:
		return []logicalPropDecl{{"border-top-width", value}}, true
	case cssPropBorderBlockEndWidth:
		return []logicalPropDecl{{"border-bottom-width", value}}, true
	case cssPropBorderInlineStartWidth:
		return []logicalPropDecl{{"border-left-width", value}}, true
	case cssPropBorderInlineEndWidth:
		return []logicalPropDecl{{"border-right-width", value}}, true
	default:
		return nil, false
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
	case boxShorthandTwoSides:
		values = [4]string{tokens[0], tokens[1], tokens[0], tokens[1]}
	case boxShorthandThreeSides:
		values = [4]string{tokens[0], tokens[1], tokens[2], tokens[1]}
	default:
		values = tokens
	}

	return values, true
}

// supportedDeclaration rejects modern value functions that this lite renderer
// cannot compute. Excluding them from the cascade preserves an earlier valid
// fallback declaration, matching the fixture's fallback-first contract
// (e.g. width:100%; width:clamp(...) must keep 100% while clampLength is gated).
func supportedDeclaration(value string) bool {
	value = strings.ToLower(value)

	for _, unsupported := range []string{"clamp(", "color-mix(", "light-dark(", "oklch("} {
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
	remBase := pxToPt(defaultRootFontPx)
	if ctx != nil && ctx.remBase > 0 {
		remBase = ctx.remBase
	}

	applyFontSizeValue(style, raw, parentSize, ctx, remBase)
	applyFontFamilyValue(style, raw)
	applyFontWeightValue(style, raw)
	applyFontStyleValue(style, raw)
	applyFontPropsWave4(style, raw)

	if val, found := raw["font"]; found {
		parseFontShorthand(style, val, remBase)
	}
}

func applyFontSizeValue(
	style *ResolvedStyle, raw map[string]string, parentSize float64, ctx *styleContext, remBase float64,
) {
	val, found := raw["font-size"]
	if !found {
		return
	}

	containing := 0.0
	if ctx != nil {
		containing = ctx.viewportW
	}

	if pt, parsed := clampLength(val, parentSize, containing); parsed {
		style.FontSize = pt

		return
	}

	style.FontSize = fontSize(val, parentSize, remBase)
}

func applyFontFamilyValue(style *ResolvedStyle, raw map[string]string) {
	val, found := raw["font-family"]
	if !found {
		return
	}

	if fam := css.ParseFontFamily(val); len(fam) > 0 {
		style.FontFamily = fam
	}
}

func applyFontWeightValue(style *ResolvedStyle, raw map[string]string) {
	val, found := raw["font-weight"]
	if found {
		style.FontWeight = resolveFontWeight(style.FontWeight, val)
	}
}

func applyFontStyleValue(style *ResolvedStyle, raw map[string]string) {
	val, found := raw["font-style"]
	if found {
		style.FontItalic = val == cssFontStyleItalic || val == cssFontStyleOblique
	}
}

// resolveFontWeight maps a font-weight keyword/number onto a weight value.
func resolveFontWeight(current int, val string) int {
	switch val {
	case contentNormal:
		return fontWeightNormalValue
	case cssFontWeightBold:
		return fontWeightBoldValue
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
	"display", marginProperty, paddingProperty, borderProperty, borderTopProperty,
	borderRightProperty, borderBottomProperty, borderLeftProperty,
	borderWidthKeyword, borderStyleKeyword,
	borderColorKeyword, gapKeyword, flexKeyword, containerKeyword,
	cssPropMarginInline, cssPropMarginBlock, cssPropPaddingInline, cssPropPaddingBlock,
	insetKeyword, cssPropInsetBlock, cssPropInsetInline, "column-rule",
	cssPropBorderBlock, cssPropBorderInline, cssPropBorderBlockStart, cssPropBorderBlockEnd,
	cssPropBorderInlineStart, cssPropBorderInlineEnd, cssPropBorderBlockWidth, cssPropBorderBlockStyle,
	cssPropBorderBlockColor, cssPropBorderInlineWidth, cssPropBorderInlineStyle, cssPropBorderInlineColor,
}

// restShorthandSet is derived from restShorthandProps so the longhand pass
// exclusion cannot drift from the ordered shorthand pass (display used to be
// missing from the hand-written switch and was applied twice).
var restShorthandSet = func() map[string]struct{} { //nolint:gochecknoglobals // derived from the apply table
	set := make(map[string]struct{}, len(restShorthandProps))

	for _, prop := range restShorthandProps {
		set[prop] = struct{}{}
	}

	return set
}()

// restLonghandStack is the stack scratch for the remaining longhands. Most
// elements declare far fewer than this many; larger declaration sets grow
// the slice on the heap the way any append would.
const restLonghandStack = 32

// applyRestProps resolves every non-font property once the font size is known.
// Shorthands run first in a fixed order. Remaining longhands still run in
// alphabetical order because longhands can overlap: overflow and overflow-x
// both write OverflowX, so the alphabetically later name must apply last for
// output to stay byte-identical. The order is reproduced with an insertion
// sort over a stack buffer, so the pass allocates no per-element key slice
// and does not call sort.Strings.
func applyRestProps(
	style *ResolvedStyle, raw map[string]string, ctx *styleContext,
	parent *ResolvedStyle,
) {
	if len(raw) == 0 {
		return
	}

	fsize := style.FontSize
	hasParent := parent != nil

	for _, prop := range restShorthandProps {
		value, ok := raw[prop]
		if !ok {
			continue
		}

		applyStyleProp(style, prop, value, fsize, ctx, parent, hasParent)
	}

	var keys [restLonghandStack]string

	rest := keys[:0]

	for key := range raw {
		if _, shorthand := restShorthandSet[key]; shorthand {
			continue
		}

		rest = append(rest, key)
	}

	rest = sortRestLonghandProps(rest)

	for _, prop := range rest {
		applyStyleProp(style, prop, raw[prop], fsize, ctx, parent, hasParent)
	}
}

// sortRestLonghandProps insertion-sorts property names into the byte order
// sort.Strings produced. A typical element has about ten remaining longhands,
// so an in-place insertion sort is cheaper than the allocation the old
// per-element slice plus stdlib sort needed.
func sortRestLonghandProps(props []string) []string {
	for i := 1; i < len(props); i++ {
		prop := props[i]
		prev := i - 1

		for prev >= 0 && props[prev] > prop {
			props[prev+1] = props[prev]
			prev--
		}

		props[prev+1] = prop
	}

	return props
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

//nolint:cyclop,goconst,funlen // vendor prefix lookup map
func normalizeVendorPrefix(prop string) string {
	if !strings.HasPrefix(prop, "-webkit-") {
		return prop
	}

	switch prop {
	case "-webkit-box-sizing":
		return "box-sizing"
	case "-webkit-border-radius":
		return "border-radius"
	case "-webkit-border-top-left-radius":
		return "border-top-left-radius"
	case "-webkit-border-top-right-radius":
		return "border-top-right-radius"
	case "-webkit-border-bottom-left-radius":
		return "border-bottom-left-radius"
	case "-webkit-border-bottom-right-radius":
		return "border-bottom-right-radius"
	case "-webkit-transform":
		return "transform"
	case "-webkit-transform-origin":
		return "transform-origin"
	case "-webkit-flex":
		return "flex"
	case "-webkit-flex-basis":
		return "flex-basis"
	case "-webkit-flex-direction":
		return "flex-direction"
	case "-webkit-flex-flow":
		return "flex-flow"
	case "-webkit-flex-grow":
		return "flex-grow"
	case "-webkit-flex-shrink":
		return "flex-shrink"
	case "-webkit-flex-wrap":
		return "flex-wrap"
	case "-webkit-justify-content":
		return "justify-content"
	case "-webkit-align-content":
		return "align-content"
	case "-webkit-align-items":
		return "align-items"
	case "-webkit-align-self":
		return "align-self"
	case "-webkit-order":
		return "order"
	case "-webkit-box-shadow":
		return "box-shadow"
	case "-webkit-filter":
		return "filter"
	case "-webkit-box-align":
		return "align-items"
	case "-webkit-box-flex":
		return "flex-grow"
	case "-webkit-box-ordinal-group":
		return "order"
	case "-webkit-box-orient":
		return "flex-direction"
	case "-webkit-box-pack":
		return "justify-content"
	case "-webkit-text-fill-color":
		return "color"
	default:
		return prop
	}
}

// applyStyleProp routes one cascaded property to the group that owns it.
//
//nolint:wsl // branching on effectiveProp
func applyStyleProp(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext,
	parent *ResolvedStyle, hasParent bool,
) {
	effectiveProp := normalizeVendorPrefix(prop)
	effectiveValue := value
	if effectiveProp != prop {
		effectiveValue = remapWebkitValue(prop, value)
	}

	for _, group := range styleGroups {
		if group(style, effectiveProp, effectiveValue, fsize, ctx, parent, hasParent) {
			return
		}
	}

	applyIgnoredGroup(style, prop, value)
}

//nolint:cyclop,goconst,wsl,nlreturn,funlen // 2009 box value remaps
func remapWebkitValue(prop, value string) string {
	trimmed := strings.TrimSpace(value)
	low := strings.ToLower(trimmed)
	switch prop {
	case "-webkit-box-align":
		switch low {
		case "start":
			return flexStartKeyword
		case "end":
			return fxFlexEnd
		case "center":
			return fxCenter
		case "stretch":
			return fxStretch
		case "baseline":
			return "baseline"
		default:
			return low
		}
	case "-webkit-box-pack":
		switch low {
		case "start":
			return flexStartKeyword
		case "end":
			return fxFlexEnd
		case "center":
			return fxCenter
		case "justify":
			return fxBetween
		default:
			return low
		}
	case "-webkit-box-orient":
		switch low {
		case "horizontal":
			return fxRow
		case "vertical":
			return fxCol
		case "inline-axis":
			return fxRow
		case "block-axis":
			return fxCol
		default:
			return low
		}
	case "-webkit-box-ordinal-group":
		if n, err := strconv.Atoi(trimmed); err == nil {
			if n > 0 {
				n--
			}
			return strconv.Itoa(n)
		}
		return trimmed
	default:
		return value
	}
}

// applyDisplayGroup handles display, position-adjacent flow and stacking props.
