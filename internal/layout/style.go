package layout

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

// CSS keyword constants shared by the cascade (goconst).
const (
	inheritKeyword     = "inherit"
	solidKeyword       = "solid"
	clearKeyword       = "clear"
	visibleKeyword     = "visible"
	pageKeyword        = "page"
	avoidKeyword       = "avoid"
	avoidPageValue     = "avoid-page"
	remUnit            = "rem"
	divElementName     = "div"
	styleElement       = "style"
	borderWidthKeyword = "border-width"
	borderStyleKeyword = "border-style"
	borderColorKeyword = "border-color"
	gapKeyword         = "gap"
	containerKeyword   = "container"
	flexKeyword        = "flex"
	flexStartKeyword   = "flex-start"
)

// fontWeightStep is the bolder/lighter adjustment applied to the current
// weight (CSS Fonts 3 §3.3; clamped by the 100..900 numeric range).
const fontWeightStep = 100

// ResolvedStyle is the used style of one element: values the layout engine
// consumes, in points (or unitless where noted). Only the phase-04 subset is
// modeled; everything else keeps its initial value.
type ResolvedStyle struct {
	Display             string
	Position            string  // "static" | "relative" | "absolute" | "fixed" | "sticky"
	Float               string  // cssDisplayNone | floatLeft | floatRight
	Clear               string  // cssDisplayNone | floatLeft | floatRight | "both"
	BoxSizing           string  // "content-box" | "border-box"
	Top                 float64 // position offsets (pt); 0 = unset for absolute uses Auto flags
	Right               float64
	Bottom              float64
	Left                float64
	TopAuto             bool
	RightAuto           bool
	BottomAuto          bool
	LeftAuto            bool
	FlexDirection       string  // "row" | fxCol | "row-reverse" | "column-reverse"
	FlexWrap            string  // "nowrap" | "wrap" | "wrap-reverse"
	JustifyContent      string  // flex-start | flex-end | center | space-between | space-around | space-evenly
	AlignItems          string  // stretch | flex-start | center | flex-end
	AlignContent        string  // flex-start | flex-end | center | space-between | space-around | space-evenly | stretch
	AlignSelf           string  // auto | stretch | flex-start | flex-end | center | start | end
	JustifyItems        string  // grid: stretch | start | end | center
	JustifySelf         string  // grid item: auto | stretch | start | end | center
	Gap                 float64 // flex/grid gap shorthand (pt); kept for backward compat
	RowGap              float64 // pt; 0 with ColumnGap 0 → layout falls back to Gap
	ColumnGap           float64
	ColumnGapNormal     bool    // true when column-gap is normal/initial (multicol → 1em; flex/grid → 0)
	ColumnCount         int     // 0 = auto; ≥1 = used count hint
	ColumnWidth         float64 // -1 = auto; else length in pt
	ColumnSpan          string  // cssDisplayNone | "all" (multicol spanner)
	ColumnFill          string  // "balance" | overflowAuto
	FlexGrow            float64
	FlexShrink          float64 // default 1; 0 disables shrink
	FlexBasis           float64 // -1 = auto
	FlexBasisPercent    float64 // >=0 means % of flex container content main size (width/height)
	FlexOrder           int
	ZIndex              int
	ZIndexSet           bool
	WritingMode         string // "" | "horizontal-tb" | "vertical-rl" | "vertical-lr"
	GridTemplateColumns string // raw grid-template-columns value
	GridTemplateRows    string
	GridTemplateAreas   string  // raw grid-template-areas value
	GridArea            string  // named area (custom-ident); empty = line-based placement
	GridAutoFlow        string  // "row" | fxCol | "dense" | "row dense" | "column dense"
	GridColumnSpan      int     // from grid-column: span N (default 1)
	GridColumnStart     int     // 1-based; 0 = auto
	GridRowSpan         int     // from grid-row: span N (default 1)
	GridRowStart        int     // 1-based; 0 = auto
	Width               float64 // -1 = auto; absolute length in pt when WidthPercent < 0
	WidthPercent        float64 // >=0 means width is that % of the containing block at layout time
	Height              float64 // -1 = auto; absolute length in pt when HeightPercent < 0
	HeightPercent       float64 // >=0 means height is that % of the CB; indefinite CB → auto (cyclic honesty)
	MinWidth            float64 // absolute pt when MinWidthPercent < 0; 0 = auto (content min for flex)
	MinWidthPercent     float64 // >=0 means % of containing block (deferred like WidthPercent)
	MaxWidth            float64
	MaxWidthPercent     float64 // >=0 means % of containing block / img clamp context
	MinHeight           float64
	MinHeightPercent    float64 // >=0 means % of CB height; indefinite → ignore
	MaxHeight           float64
	Overflow            string // "visible" | "hidden" | "scroll" | "auto" | "clip" (non-visible = sticky scrollport)
	MarginTop           float64
	MarginRight         float64
	MarginBottom        float64
	MarginLeft          float64
	MarginLeftAuto      bool // margin-left: auto (horizontal centering with right auto)
	MarginRightAuto     bool // margin-right: auto
	PaddingTop          float64
	PaddingRight        float64
	PaddingBottom       float64
	PaddingLeft         float64
	BorderTop           border
	BorderRight         border
	BorderBottom        border
	BorderLeft          border
	Color               [3]float64
	BGColor             [4]float64 // rgba, 0..1
	FontFamily          []string
	FontSize            float64 // pts
	FontWeight          int
	FontItalic          bool
	LineHeight          float64 // pts; 0 = "normal"
	TextAlign           string  // floatLeft | floatRight | "center" | "justify"
	VerticalAlign       string  // "baseline" | "top" | "middle" | cssVerticalAlignBottom
	WhiteSpace          string  // "normal" | "nowrap" | "pre"
	// OverflowWrap is CSS overflow-wrap / word-wrap: "normal" | "break-word" | "anywhere".
	OverflowWrap string
	// WordBreak is CSS word-break: "normal" | "break-all" | "keep-all".
	WordBreak       string
	TextDecoration  string // cssDisplayNone | "underline" | "line-through"
	LetterSpacing   float64
	TextIndent      float64
	ListStyleType   string // "disc" | "circle" | "square" | "decimal" | cssDisplayNone | …
	BorderCollapse  string // "separate" | "collapse"
	BorderSpacing   float64
	TableLayout     string // overflowAuto | "fixed"
	IsReplaced      bool   // img, hr
	PageBreakBefore string // "" | "always" | "avoid"
	PageBreakAfter  string // "" | "always" | "avoid"
	PageBreakInside string // "" | "always" | "avoid"
	Orphans         int    // CSS orphans; inherited; initial 2; integer ≥ 1
	Widows          int    // CSS widows; inherited; initial 2; integer ≥ 1
	ContainerType   string // "" | "normal" | "inline-size" | "size"
	ContainerName   string // space-separated lower-case names; empty = none
	// Static 2D CSS transforms (paint-time CTM; sibling flow unchanged).
	Transform       Matrix2D
	HasTransform    bool
	TransformOrigin transformOriginSpec
	Opacity         float64 // 0..1; initial 1; also from filter:opacity()
	// CustomProps holds resolved CSS custom properties (--*) for this element
	// (inherited). Shared with the parent map when the element declares none.
	CustomProps map[string]string
}

type border struct {
	Width float64
	Style string // cssDisplayNone | "solid" | "dashed" | "dotted"
	Color [3]float64
}

// initialStyle returns the CSS initial values.
func initialStyle() ResolvedStyle {
	return ResolvedStyle{ //nolint:exhaustruct // intentional zero fields
		Display:          "inline",
		Position:         "static",
		Float:            cssDisplayNone,
		FlexGrow:         0,
		FlexShrink:       1,
		FlexBasis:        -1,
		FlexBasisPercent: -1,
		Clear:            cssDisplayNone,
		BoxSizing:        "content-box",
		TopAuto:          true,
		RightAuto:        true,
		BottomAuto:       true,
		LeftAuto:         true,
		FlexDirection:    "row",
		FlexWrap:         "nowrap",
		JustifyContent:   "flex-start",
		AlignItems:       "stretch",
		AlignContent:     "stretch",
		AlignSelf:        overflowAuto,
		JustifyItems:     "stretch",
		JustifySelf:      overflowAuto,
		ColumnGapNormal:  true,
		ColumnWidth:      -1,
		ColumnSpan:       cssDisplayNone,
		ColumnFill:       "balance",
		Width:            -1,
		WidthPercent:     -1,
		Height:           -1,
		HeightPercent:    -1,
		MinWidth:         0,
		MinWidthPercent:  -1,
		MaxWidth:         -1,
		MaxWidthPercent:  -1,
		MinHeight:        0,
		MinHeightPercent: -1,
		MaxHeight:        -1,
		Overflow:         "visible",
		Color:            [3]float64{0, 0, 0},
		BGColor:          [4]float64{0, 0, 0, 0},
		FontSize:         defaultFontSizePt, // 16px at 96dpi
		FontWeight:       fontWeightNormal,
		VerticalAlign:    "baseline",
		WhiteSpace:       "normal",
		OverflowWrap:     "normal",
		WordBreak:        "normal",
		TextDecoration:   cssDisplayNone,
		ListStyleType:    "disc",
		BorderCollapse:   "separate",
		BorderSpacing:    0,
		TableLayout:      overflowAuto,
		GridColumnSpan:   1,
		GridRowSpan:      1,
		WritingMode:      "horizontal-tb",
		Orphans:          two,
		Widows:           two,
		Transform:        IdentityMatrix(),
		TransformOrigin:  defaultTransformOrigin(),
		Opacity:          1,
	}
}

// styleContext carries per-element resolution inputs.
type styleContext struct {
	sheets    []*css.Stylesheet
	media     string
	viewportW float64 // containing-block width for % of margins/padding/width
	viewportH float64 // for % of height
	// remBase is the used font-size of the root element for rem units (pt).
	// 0 means the CSS initial medium size (16px → 12pt).
	remBase float64
	// printLinkUnderline is the opt-in --print-link-underline operator policy.
	printLinkUnderline bool
	// containers maps size-query containers (inline-size|size) to their used
	// content-box inline size. nil means first pass: skip @container rules.
	containers map[*html.Node]sizeContainer
}

// sizeContainer is one element that establishes a size query container.
type sizeContainer struct {
	inlineSize float64 // content-box inline size in pt
	fontSize   float64 // used font-size (em base for query lengths)
	names      string  // space-separated container-name values
}

// sameSizeContainerState reports whether a second container measurement is
// equivalent to the previous one. Container queries can use em lengths, so a
// changed used font-size is just as significant as a changed inline size or
// name. Keeping this comparison here gives the convergence loop one policy
// for deciding whether a second style pass is required.
func sameSizeContainerState(a, b sizeContainer) bool {
	return nearlyEqual(a.inlineSize, b.inlineSize) &&
		nearlyEqual(a.fontSize, b.fontSize) &&
		a.names == b.names
}

func nearlyEqual(a, b float64) bool {
	return math.Abs(a-b) <= styleEpsilon
}

// resolveStylesWith is the single cascade entry: Options + optional size
// containers for @container rules (nil = first pass, skip container queries).
func resolveStylesWith(
	root *html.Node, opts Options, containers map[*html.Node]sizeContainer,
) map[*html.Node]ResolvedStyle {
	return resolveStylesCtx(root, &styleContext{ //nolint:exhaustruct // intentional zero fields
		sheets:             opts.Sheets,
		media:              opts.Media,
		viewportW:          opts.Width,
		viewportH:          opts.Height,
		printLinkUnderline: opts.PrintLinkUnderline,
		containers:         containers,
	})
}

// resolveStyles walks the tree top-down (test helper; no operator policies).
// @container rules are ignored on this first pass (no used sizes yet).
func resolveStyles(
	root *html.Node, sheets []*css.Stylesheet, media string, viewportW, viewportH float64,
) map[*html.Node]ResolvedStyle {
	return resolveStylesWith(root, Options{ //nolint:exhaustruct // intentional zero fields
		Sheets: sheets, Media: media, Width: viewportW, Height: viewportH,
	}, nil)
}

// resolveStylesWithContainers is the second style pass: @container rules are
// applied when their query matches the nearest eligible ancestor in containers.
// Test helper; media/viewport always come from the caller's fixture.
func resolveStylesWithContainers(
	root *html.Node,
	sheets []*css.Stylesheet,
	media string, //nolint:unparam // test helper: media fixed per call site
	viewportW, viewportH float64, //nolint:unparam // test helper: viewport fixed per call site
	containers map[*html.Node]sizeContainer,
) map[*html.Node]ResolvedStyle {
	return resolveStylesWith(root, Options{ //nolint:exhaustruct // intentional zero fields
		Sheets: sheets, Media: media, Width: viewportW, Height: viewportH,
	}, containers)
}

func resolveStylesCtx(root *html.Node, ctx *styleContext) map[*html.Node]ResolvedStyle {
	out := map[*html.Node]ResolvedStyle{}

	var walk func(n *html.Node, parent ResolvedStyle, hasParent bool)
	walk = func(node *html.Node, parent ResolvedStyle, hasParent bool) {
		var sty ResolvedStyle

		switch node.Type {
		case html.ElementNode:
			sty = resolveElementStyle(node, ctx, parent, hasParent)
		case html.TextNode:
			sty = initialStyle()
			if hasParent {
				inheritProps(&sty, parent, nil)
				sty.CustomProps = parent.CustomProps
			}
		case html.CommentNode, html.DoctypeNode: // no style resolution; keep the zero value
		}

		out[node] = sty

		for _, c := range node.Children {
			walk(c, sty, true)
		}
	}
	walk(root, ResolvedStyle{}, false) //nolint:exhaustruct // intentional zero fields

	return out
}

// resolveElementStyle cascades one element: inheritance, custom properties,
// fonts, the remaining properties, and the operator/blockify policies.
func resolveElementStyle(node *html.Node, ctx *styleContext, parent ResolvedStyle, hasParent bool) ResolvedStyle {
	raw := cascadeRaw(ctx, node)
	sty := initialStyle()

	var parentProps map[string]string

	if hasParent {
		inheritProps(&sty, parent, raw)
		parentProps = parent.CustomProps
	}

	sty.CustomProps = mergeCustomProps(parentProps, raw)
	raw = resolveRawVars(raw, sty.CustomProps)

	parentSize := sty.FontSize
	if hasParent {
		parentSize = parent.FontSize
	}

	applyFontProps(&sty, raw, parentSize, ctx)

	if node.Name == "html" && sty.FontSize > 0 {
		ctx.remBase = sty.FontSize
	}

	applyRestProps(&sty, raw, ctx, parent, hasParent)
	// Opt-in operator policy (--print-link-underline): underline
	// anchors with href after the cascade. Default off — author CSS
	// (including text-decoration: inherit → parent) wins otherwise.
	if ctx != nil && ctx.printLinkUnderline && node.Name == "a" && strings.TrimSpace(node.Attribute("href")) != "" {
		sty.TextDecoration = cssTextDecorationUnderline
	}
	// CSS2.1 §9.7: float ≠ none blockifies table-internal / inline
	// displays before layout (table/flex/grid stay). Floated <table>
	// keeps display:table so fixture-29 wrapper packing still works.
	if sty.Float != cssDisplayNone {
		sty.Display = blockifyDisplayForFloat(sty.Display)
	}

	return sty
}

// mergeCustomProps inherits parent custom properties and overlays any --*
// declarations from raw, resolving var() chains via css.ResolveCustomProps.
func mergeCustomProps(parentProps map[string]string, raw map[string]string) map[string]string {
	declared := map[string]string{}

	for prop, v := range raw {
		if strings.HasPrefix(prop, "--") {
			declared[prop] = v
		}
	}

	if len(declared) == 0 {
		return parentProps
	}

	return css.ResolveCustomProps(declared, parentProps)
}

// resolveRawVars expands var() in cascaded property values using customProps.
// Custom property keys (--*) are left unchanged (already resolved in the map).
func resolveRawVars(raw map[string]string, customProps map[string]string) map[string]string {
	if len(raw) == 0 {
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

		if strings.Contains(strings.ToLower(val), "var(") {
			out[prop] = css.ResolveVar(val, lookup)
		} else {
			out[prop] = val
		}
	}

	return out
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
	copy  func(dst *ResolvedStyle, src ResolvedStyle)
}

// inheritableProps is the immutable inherit table used by inheritProps.
// Package-level so inheritProps does not allocate a new slice, name slices,
// and closures on every styled node (was ~40% of alloc_objects on 500-page PDF).
var inheritableProps = []inheritCopy{ //nolint:gochecknoglobals // static inherit table
	{[]string{"color"}, func(dst *ResolvedStyle, src ResolvedStyle) { dst.Color = src.Color }},
	{[]string{"font-family"}, func(dst *ResolvedStyle, src ResolvedStyle) { dst.FontFamily = src.FontFamily }},
	{[]string{"font-size"}, func(dst *ResolvedStyle, src ResolvedStyle) { dst.FontSize = src.FontSize }},
	{[]string{"font-weight"}, func(dst *ResolvedStyle, src ResolvedStyle) { dst.FontWeight = src.FontWeight }},
	{[]string{"font-style"}, func(dst *ResolvedStyle, src ResolvedStyle) { dst.FontItalic = src.FontItalic }},
	{[]string{"line-height"}, func(dst *ResolvedStyle, src ResolvedStyle) { dst.LineHeight = src.LineHeight }},
	{[]string{"text-align"}, func(dst *ResolvedStyle, src ResolvedStyle) { dst.TextAlign = src.TextAlign }},
	{[]string{"white-space"}, func(dst *ResolvedStyle, src ResolvedStyle) { dst.WhiteSpace = src.WhiteSpace }},
	// overflow-wrap / word-wrap and word-break are inherited (CSS Text).
	{
		[]string{"overflow-wrap", "word-wrap"},
		func(dst *ResolvedStyle, src ResolvedStyle) { dst.OverflowWrap = src.OverflowWrap },
	},
	{[]string{"word-break"}, func(dst *ResolvedStyle, src ResolvedStyle) { dst.WordBreak = src.WordBreak }},
	{
		[]string{"vertical-align"},
		func(dst *ResolvedStyle, src ResolvedStyle) { dst.VerticalAlign = src.VerticalAlign },
	},
	{
		[]string{"text-decoration"},
		func(dst *ResolvedStyle, src ResolvedStyle) { dst.TextDecoration = src.TextDecoration },
	},
	{
		[]string{"letter-spacing"},
		func(dst *ResolvedStyle, src ResolvedStyle) { dst.LetterSpacing = src.LetterSpacing },
	},
	{
		[]string{"list-style-type", "list-style"},
		func(dst *ResolvedStyle, src ResolvedStyle) { dst.ListStyleType = src.ListStyleType },
	},
	{
		[]string{"border-collapse"},
		func(dst *ResolvedStyle, src ResolvedStyle) { dst.BorderCollapse = src.BorderCollapse },
	},
	{
		[]string{"border-spacing"},
		func(dst *ResolvedStyle, src ResolvedStyle) { dst.BorderSpacing = src.BorderSpacing },
	},
	{[]string{"orphans"}, func(dst *ResolvedStyle, src ResolvedStyle) { dst.Orphans = src.Orphans }},
	{[]string{"widows"}, func(dst *ResolvedStyle, src ResolvedStyle) { dst.Widows = src.Widows }},
}

// inheritProps copies inheritable properties from the parent, unless the
// element declares its own value (present in raw).
func inheritProps(dst *ResolvedStyle, parent ResolvedStyle, raw map[string]string) {
	set := func(prop string) bool {
		if raw == nil {
			return false
		}

		_, ok := raw[prop]

		return ok
	}

	for _, entry := range inheritableProps {
		declared := false

		for _, name := range entry.names {
			if set(name) {
				declared = true

				break
			}
		}

		if !declared {
			entry.copy(dst, parent)
		}
	}
}

// cascadeRaw returns the winning declaration per property for the element
// across UA sheet, author sheets and the inline style attribute.
// ruleHit is one selector match from the shared cascade rule walk.
type ruleHit struct {
	r       css.Rule
	a, b, c int
}

// matchedRules walks sheets with the cascade's gates (media, @container,
// selector match, specificity). pseudoElem != "" matches ::before/::after
// shapes instead of the element (pseudo-content path).
func (ctx *styleContext) matchedRules(node *html.Node, pseudoElem string) []ruleHit {
	if ctx == nil {
		return nil
	}

	hits := make([]ruleHit, 0, len(ctx.sheets))

	for _, sheet := range ctx.sheets {
		hits = append(hits, ctx.sheetRuleHits(sheet, node, pseudoElem)...)
	}

	return hits
}

// sheetRuleHits walks one stylesheet's rules, gating on media and @container
// before descending into selector matching.
func (ctx *styleContext) sheetRuleHits(sheet *css.Stylesheet, node *html.Node, pseudoElem string) []ruleHit {
	if sheet == nil {
		return nil
	}

	hits := make([]ruleHit, 0, len(sheet.Rules))

	for _, rule := range sheet.Rules {
		if !css.MediaMatches(rule.Media, ctx.media, ctx.viewportW, ctx.viewportH) {
			continue
		}

		if !ctx.containerGateMatches(node, rule) {
			continue
		}

		hits = append(hits, ctx.ruleSelectorHits(rule, node, pseudoElem)...)
	}

	return hits
}

// ruleSelectorHits scores every selector of one rule that matches the node.
func (ctx *styleContext) ruleSelectorHits(rule css.Rule, node *html.Node, pseudoElem string) []ruleHit {
	hits := make([]ruleHit, 0, len(rule.Selectors))

	for _, sel := range rule.Selectors {
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

func cascadeRaw(ctx *styleContext, node *html.Node) map[string]string {
	normal := map[string]string{}
	important := map[string]string{}

	nSpec, nOrder := map[string][4]int{}, map[string]int{}

	iSpec, iOrder := map[string][4]int{}, map[string]int{}

	// UA sheet (lowest priority; specificity 0, order -1)
	for _, d := range uaRules(node.Name) {
		applyDeclaration(normal, nSpec, nOrder, d.Prop, d.Value, 0, 0, 0, -1)
	}

	// author sheets in source order (shared matchedRules walk)
	for _, hit := range ctx.matchedRules(node, "") {
		r := hit.r
		for _, d := range r.Decls {
			if d.Important {
				applyDeclaration(important, iSpec, iOrder, d.Prop, d.Value, hit.a, hit.b, hit.c, r.Order)
			} else {
				applyDeclaration(normal, nSpec, nOrder, d.Prop, d.Value, hit.a, hit.b, hit.c, r.Order)
			}
		}
	}

	// inline style attribute: outranks all normal declarations and all sheet
	// important declarations (spec 1<<maxIntShift).
	for _, d := range css.ParseInline(node.Attribute("style")) {
		if d.Important {
			applyDeclaration(important, iSpec, iOrder, d.Prop, d.Value, 1<<maxIntShift, 0, 0, 1<<maxIntShift)
		} else {
			applyDeclaration(normal, nSpec, nOrder, d.Prop, d.Value, 1<<maxIntShift, 0, 0, 1<<maxIntShift)
		}
	}

	out := map[string]string{}
	for prop, v := range normal {
		out[prop] = v
	}

	for prop, v := range important {
		out[prop] = v // important beats normal
	}

	return out
}

// applyDeclaration folds one declaration into the winning map when its
// specificity/order beats the current winner.
func applyDeclaration(
	winning map[string]string, spec map[string][4]int, ord map[string]int,
	prop, value string, ids, classes, types, order int,
) {
	prop = strings.ToLower(prop)
	if _, ok := winning[prop]; !ok {
		winning[prop] = value
		spec[prop] = [4]int{ids, classes, types, 0}
		ord[prop] = order

		return
	}

	if specificityBeats(spec[prop], ids, classes, types, order, ord[prop]) {
		winning[prop] = value
		spec[prop] = [4]int{ids, classes, types, 0}
		ord[prop] = order
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
		style.FontSize = fontSize(v, parentSize, remBase)
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

// applyRestProps resolves every non-font property once the font size is known.
// Properties are applied in a fixed order: the shorthand groups first, then
// every other property alphabetically. raw is a map, so iterating it directly
// would be nondeterministic and could let a shorthand (e.g. UA "margin")
// clobber a winning longhand (e.g. author "margin-bottom") depending on map
// iteration order.
func applyRestProps(
	style *ResolvedStyle, raw map[string]string, ctx *styleContext,
	parent ResolvedStyle, hasParent bool,
) {
	fsize := style.FontSize
	// gap/flex/container applied before longhands so row-gap/column-gap,
	// flex-*, and container-type/name win over shorthands.
	shorthands := [...]string{
		"margin", "padding", "border", borderWidthKeyword, borderStyleKeyword,
		borderColorKeyword, gapKeyword, flexKeyword, containerKeyword,
	}
	rest := make([]string, 0, len(raw))

	for prop := range raw {
		switch prop {
		case "margin", "padding", "border", borderWidthKeyword, borderStyleKeyword,
			borderColorKeyword, gapKeyword, flexKeyword, containerKeyword:
			continue
		}

		rest = append(rest, prop)
	}

	sort.Strings(rest)
	props := make([]string, 0, len(shorthands)+len(rest))
	props = append(props, shorthands[:]...)
	props = append(props, rest...)

	for _, prop := range props {
		value, ok := raw[prop]
		if !ok {
			continue
		}

		applyStyleProp(style, prop, value, fsize, ctx, parent, hasParent)
	}
}

// styleGroupFn is one property-group handler in the applyStyleProp dispatch.
// Groups return false when they do not own prop.
type styleGroupFn func(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext,
	parent ResolvedStyle, hasParent bool,
) bool

// applyStyleProp routes one cascaded property to the group that owns it.
func applyStyleProp(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext,
	parent ResolvedStyle, hasParent bool,
) {
	groups := [...]styleGroupFn{
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
	for _, group := range groups {
		if group(style, prop, value, fsize, ctx, parent, hasParent) {
			return
		}
	}

	applyIgnoredGroup(style, prop, value)
}

// applyDisplayGroup handles display, position-adjacent flow and stacking props.
func applyDisplayGroup(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext, _ ResolvedStyle, _ bool,
) bool {
	if applyDisplayFlowProps(style, prop, value) {
		return true
	}

	return applyDisplayEffectProps(style, prop, value)
}

// applyDisplayFlowProps owns the display/position/float/clear/box-sizing/
// writing-mode/overflow keyword properties.
func applyDisplayFlowProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "display":
		setDisplayKeyword(style, value)
	case "position":
		setPositionKeyword(style, value)
	case "float":
		setFloatKeyword(style, value)
	case clearKeyword:
		setClearKeyword(style, value)
	case "box-sizing":
		setBoxSizingKeyword(style, value)
	case "writing-mode":
		setWritingModeKeyword(style, value)
	case "overflow", "overflow-x", "overflow-y":
		setOverflowKeyword(style, prop, value)
	default:
		return false
	}

	return true
}

func setDisplayKeyword(style *ResolvedStyle, value string) {
	switch value {
	case displayBlock, "inline", cssDisplayNone, displayListItem, displayTable, displayTableRow, displayTableCell,
		displayRowGroup, displayHeaderGroup, displayFooterGroup,
		cssDisplayInlineBlock, displayTableCaption, "table-column", "table-column-group",
		displayFlex, displayInlineFlex, displayGrid, displayInlineGrid, displaySubgrid, displayFlowRoot:
		style.Display = value
	}
}

func setPositionKeyword(style *ResolvedStyle, value string) {
	switch value {
	case "static", "relative", "absolute", "fixed", "sticky":
		style.Position = value
	}
}

func setFloatKeyword(style *ResolvedStyle, value string) {
	switch value {
	case floatLeft, floatRight, cssDisplayNone:
		style.Float = value
	}
}

func setClearKeyword(style *ResolvedStyle, value string) {
	switch value {
	case floatLeft, floatRight, clearBoth, cssDisplayNone:
		style.Clear = value
	}
}

func setBoxSizingKeyword(style *ResolvedStyle, value string) {
	switch value {
	case "content-box", borderBox:
		style.BoxSizing = value
	}
}

// setOverflowKeyword applies overflow on either axis; a non-visible overflow
// on an axis creates a sticky scrollport (CSS Position 3).
func setOverflowKeyword(style *ResolvedStyle, prop, value string) {
	overflow, ok := parseOverflowKeyword(value)
	if !ok {
		return
	}

	if prop == "overflow" || overflow != visibleKeyword {
		style.Overflow = overflow
	}
}

func setWritingModeKeyword(style *ResolvedStyle, value string) {
	switch value {
	case "horizontal-tb", "vertical-rl", "vertical-lr":
		style.WritingMode = value
	}
}

// applyDisplayEffectProps owns z-index, opacity and filter:opacity().
func applyDisplayEffectProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "z-index":
		setZIndexValue(style, value)
	case "opacity":
		setOpacityValue(style, value)
	case "filter":
		setFilterValue(style, value)
	default:
		return false
	}

	return true
}

func setZIndexValue(style *ResolvedStyle, value string) {
	if value == overflowAuto {
		style.ZIndexSet = false
		style.ZIndex = 0
	} else if v, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
		style.ZIndex = v
		style.ZIndexSet = true
	}
}

func setOpacityValue(style *ResolvedStyle, value string) {
	if v, ok := parseOpacityValue(value); ok {
		style.Opacity = v
	}
}

func setFilterValue(style *ResolvedStyle, value string) {
	// opacity() via ExtGState; blur/drop-shadow permanent non-goals.
	if v, ok := parseFilterOpacity(value); ok {
		style.Opacity *= v
	}
}

// applyPositionGroup handles the top/right/bottom/left offsets.
func applyPositionGroup(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext, _ ResolvedStyle, _ bool,
) bool {
	switch prop {
	case cssVerticalAlignTop:
		style.Top, style.TopAuto = marginLenAuto(value, fsize, ctx.viewportH)
	case floatRight:
		style.Right, style.RightAuto = marginLenAuto(value, fsize, ctx.viewportW)
	case cssVerticalAlignBottom:
		style.Bottom, style.BottomAuto = marginLenAuto(value, fsize, ctx.viewportH)
	case floatLeft:
		style.Left, style.LeftAuto = marginLenAuto(value, fsize, ctx.viewportW)
	default:
		return false
	}

	return true
}

// applyFlexGroup handles flex layout props and the gap family.
func applyFlexGroup(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext, _ ResolvedStyle, _ bool,
) bool {
	switch prop {
	case gapKeyword, "row-gap", "column-gap":
		return applyGapProps(style, prop, value, fsize, ctx)
	case "flex-direction", "flex-wrap", "justify-content", "align-items",
		"align-content", "align-self", "justify-items", "justify-self":
		return applyFlexAlignmentProps(style, prop, value)
	case flexKeyword, "flex-grow", "flex-shrink", "flex-basis", "order":
		return applyFlexBasisProps(style, prop, value, fsize, ctx)
	default:
		return false
	}
}

// applyGapProps owns the gap family, dispatching each property to its parser.
func applyGapProps(style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext) bool {
	switch prop {
	case gapKeyword:
		return applyGapShorthand(style, value, fsize, ctx.viewportW)
	case "row-gap":
		return applyRowGap(style, value, fsize, ctx.viewportW)
	case "column-gap":
		return applyColumnGap(style, value, fsize, ctx.viewportW)
	default:
		return false
	}
}

func applyGapShorthand(style *ResolvedStyle, value string, fsize, viewportW float64) bool {
	if value == contentNormal {
		style.Gap = 0
		style.RowGap = 0
		style.ColumnGap = 0
		style.ColumnGapNormal = true
	} else if v, ok := lengthBox(value, fsize, viewportW, cssDisplayNone); ok && v >= 0 {
		style.Gap = v
		style.RowGap = v
		style.ColumnGap = v
		style.ColumnGapNormal = false
	}

	return true
}

func applyRowGap(style *ResolvedStyle, value string, fsize, viewportW float64) bool {
	if v, ok := lengthBox(value, fsize, viewportW, cssDisplayNone); ok && v >= 0 {
		style.RowGap = v
		style.Gap = v
	}

	return true
}

func applyColumnGap(style *ResolvedStyle, value string, fsize, viewportW float64) bool {
	if value == contentNormal {
		style.ColumnGap = 0
		style.ColumnGapNormal = true
	} else if v, ok := lengthBox(value, fsize, viewportW, cssDisplayNone); ok && v >= 0 {
		style.ColumnGap = v
		style.ColumnGapNormal = false
	}

	return true
}

// applyFlexAlignmentProps owns the flex/grid alignment keywords.
func applyFlexAlignmentProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "flex-direction":
		setFlexDirectionValue(style, value)
	case "flex-wrap":
		setFlexWrapValue(style, value)
	case "justify-content":
		setJustifyContentValue(style, value)
	case "align-items":
		setAlignItemsValue(style, value)
	case "align-content":
		setAlignContentValue(style, value)
	case "align-self":
		setAlignSelfValue(style, value)
	case "justify-items":
		setJustifyItemsValue(style, value)
	case "justify-self":
		setJustifySelfValue(style, value)
	default:
		return false
	}

	return true
}

func setFlexDirectionValue(style *ResolvedStyle, value string) {
	switch value {
	case fxRow, fxCol, "row-reverse", fxColRev:
		style.FlexDirection = value
	}
}

func setFlexWrapValue(style *ResolvedStyle, value string) {
	if value == "nowrap" || value == "wrap" || value == "wrap-reverse" {
		style.FlexWrap = value
	}
}

func setJustifyContentValue(style *ResolvedStyle, value string) {
	switch value {
	case flexStartKeyword, fxFlexEnd, fxCenter, fxBetween, fxAround, fxEvenly, fxStart, fxEnd:
		style.JustifyContent = value
	}
}

func setAlignItemsValue(style *ResolvedStyle, value string) {
	switch value {
	case fxStretch, flexStartKeyword, fxFlexEnd, fxCenter, fxStart, fxEnd:
		style.AlignItems = value
	}
}

func setAlignContentValue(style *ResolvedStyle, value string) {
	switch value {
	case flexStartKeyword, fxFlexEnd, fxCenter, fxBetween, fxAround,
		fxEvenly, fxStretch, fxStart, fxEnd:
		style.AlignContent = value
	}
}

func setAlignSelfValue(style *ResolvedStyle, value string) {
	switch value {
	case overflowAuto, fxStretch, flexStartKeyword, fxFlexEnd, fxCenter, fxStart, fxEnd:
		style.AlignSelf = value
	}
}

func setJustifyItemsValue(style *ResolvedStyle, value string) {
	switch value {
	case fxStretch, fxStart, fxEnd, fxCenter, flexStartKeyword, fxFlexEnd:
		style.JustifyItems = value
	}
}

func setJustifySelfValue(style *ResolvedStyle, value string) {
	switch value {
	case overflowAuto, fxStretch, fxStart, fxEnd, fxCenter, flexStartKeyword, fxFlexEnd:
		style.JustifySelf = value
	}
}

// applyFlexBasisProps owns the flex shorthand, grow/shrink/basis and order.
func applyFlexBasisProps(style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext) bool {
	switch prop {
	case flexKeyword:
		parseFlexShorthand(style, value, fsize, ctx.viewportW)
	case "flex-grow":
		setFlexGrowValue(style, value)
	case "flex-shrink":
		setFlexShrinkValue(style, value)
	case "flex-basis":
		setFlexBasisValue(style, value, fsize, ctx.viewportW)
	case "order":
		setFlexOrderValue(style, value)
	default:
		return false
	}

	return true
}

func setFlexGrowValue(style *ResolvedStyle, value string) {
	if v, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil && v >= 0 {
		style.FlexGrow = v
	}
}

func setFlexShrinkValue(style *ResolvedStyle, value string) {
	if v, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil && v >= 0 {
		style.FlexShrink = v
	}
}

func setFlexBasisValue(style *ResolvedStyle, value string, fsize, viewportW float64) {
	if value == overflowAuto {
		style.FlexBasis = -1
		style.FlexBasisPercent = -1
	} else if v, unit, ok := css.ParseLength(value); ok && unit == "%" {
		style.FlexBasisPercent = v
		style.FlexBasis = -1
	} else if v, ok := lengthBox(value, fsize, viewportW, overflowAuto); ok {
		style.FlexBasis = v
		style.FlexBasisPercent = -1
	}
}

func setFlexOrderValue(style *ResolvedStyle, value string) {
	if v, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
		style.FlexOrder = v
	}
}

// applyMulticolGroup handles column-* props.
func applyMulticolGroup(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext, _ ResolvedStyle, _ bool,
) bool {
	if applyColumnCountWidthProps(style, prop, value, fsize, ctx.viewportW) {
		return true
	}

	return applyColumnFillSpanProps(style, prop, value)
}

func applyColumnCountWidthProps(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	switch prop {
	case "column-count":
		return setColumnCountValue(style, value)
	case "column-width":
		return setColumnWidthValue(style, value, fsize, viewportW)
	case "columns":
		parseColumnsShorthand(style, value, fsize, viewportW)
	default:
		return false
	}

	return true
}

func setColumnCountValue(style *ResolvedStyle, value string) bool {
	if value == overflowAuto {
		style.ColumnCount = 0
	} else if n, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && n >= 1 {
		style.ColumnCount = n
	}

	return true
}

func setColumnWidthValue(style *ResolvedStyle, value string, fsize, viewportW float64) bool {
	if value == overflowAuto {
		style.ColumnWidth = -1
	} else if v, ok := lengthBox(value, fsize, viewportW, overflowAuto); ok && v >= 0 {
		style.ColumnWidth = v
	}

	return true
}

func applyColumnFillSpanProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "column-span":
		switch value {
		case cssDisplayNone, "all":
			style.ColumnSpan = value
		}
	case "column-fill":
		switch value {
		case "balance", overflowAuto:
			style.ColumnFill = value
		}
	default:
		return false
	}

	return true
}

// applyGridGroup handles grid template/placement props.
func applyGridGroup(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext, _ ResolvedStyle, _ bool,
) bool {
	if applyGridTemplateProps(style, prop, value) {
		return true
	}

	return applyGridPlacementProps(style, prop, value)
}

func applyGridTemplateProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "grid-template-columns":
		style.GridTemplateColumns = value
	case "grid-template-rows":
		style.GridTemplateRows = value
	case "grid-template-areas":
		style.GridTemplateAreas = value
	case "grid-area":
		parseGridArea(style, value)
	case "grid-auto-flow":
		style.GridAutoFlow = parseGridAutoFlowValue(value)
	default:
		return false
	}

	return true
}

func applyGridPlacementProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "grid-column", "grid-column-end":
		parseGridColumn(style, value)
	case "grid-column-start":
		setGridStartIndex(style, "grid-column-start", value)
	case "grid-row", "grid-row-end":
		parseGridRow(style, value)
	case "grid-row-start":
		setGridStartIndex(style, "grid-row-start", value)
	default:
		return false
	}

	return true
}

// setGridStartIndex parses a positive grid line index for one axis.
func setGridStartIndex(style *ResolvedStyle, prop, value string) {
	line, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || line <= 0 {
		return
	}

	if prop == "grid-row-start" {
		style.GridRowStart = line
	} else {
		style.GridColumnStart = line
	}
}

// applyBoxGroup handles the sizing and box props (width/height/margins/padding).
func applyBoxGroup(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext, _ ResolvedStyle, _ bool,
) bool {
	if applyBoxSizingProps(style, prop, value, fsize, ctx) {
		return true
	}

	return applyBoxSpacingProps(style, prop, value, fsize, ctx.viewportW)
}

func applyBoxSizingProps(style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext) bool {
	switch prop {
	case "width", "height":
		return applyBoxMainSizeProps(style, prop, value, fsize, ctx)
	case "min-width", "min-height":
		return applyBoxMinExtentProps(style, prop, value, fsize, ctx)
	case "max-width", "max-height":
		return applyBoxMaxExtentProps(style, prop, value, fsize, ctx)
	default:
		return false
	}
}

func applyBoxSpacingProps(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	switch prop {
	case "margin":
		return applyMarginShorthandProps(style, value, fsize, viewportW)
	case "margin-top", "margin-bottom":
		return applyMarginVerticalProps(style, prop, value, fsize, viewportW)
	case "margin-right", "margin-left":
		return applyMarginHorizontalProps(style, prop, value, fsize, viewportW)
	case "padding":
		return applyPaddingShorthandProps(style, value, fsize, viewportW)
	case "padding-top", "padding-right", "padding-bottom", "padding-left":
		return applyPaddingSideProps(style, prop, value, fsize, viewportW)
	default:
		return false
	}
}

func applyBoxMainSizeProps(style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext) bool {
	switch prop {
	case "width":
		return setWidthValue(style, value, fsize, ctx.viewportW)
	case "height":
		return setHeightValue(style, value, fsize, ctx.viewportH)
	default:
		return false
	}
}

func setWidthValue(style *ResolvedStyle, value string, fsize, viewportW float64) bool {
	if value == overflowAuto {
		style.Width = -1
		style.WidthPercent = -1
	} else if v, unit, ok := css.ParseLength(value); ok && unit == "%" {
		// Resolve % against the layout containing block (availW), not
		// the viewport — nested width:100% must fill the parent cell.
		style.WidthPercent = v
		style.Width = -1
	} else if v, ok := lengthBox(value, fsize, viewportW, overflowAuto); ok {
		style.Width = v
		style.WidthPercent = -1
	}

	return true
}

func setHeightValue(style *ResolvedStyle, value string, fsize, viewportH float64) bool {
	if value == overflowAuto {
		style.Height = -1
		style.HeightPercent = -1
	} else if v, unit, ok := css.ParseLength(value); ok && unit == "%" {
		// Defer % height to layout; indefinite containing block → auto
		// (cyclic percentage honesty for flex/grid children).
		style.HeightPercent = v
		style.Height = -1
	} else if v, ok := lengthBox(value, fsize, viewportH, overflowAuto); ok {
		style.Height = v
		style.HeightPercent = -1
	}

	return true
}

func applyBoxMinExtentProps(style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext) bool {
	switch prop {
	case "min-width":
		return setMinWidthValue(style, value, fsize, ctx.viewportW)
	case "min-height":
		return setMinHeightValue(style, value, fsize, ctx.viewportH)
	default:
		return false
	}
}

func setMinWidthValue(style *ResolvedStyle, value string, fsize, viewportW float64) bool {
	if value == overflowAuto || value == cssDisplayNone {
		style.MinWidth = 0
		style.MinWidthPercent = -1
	} else if v, unit, ok := css.ParseLength(value); ok && unit == "%" {
		style.MinWidthPercent = v
		style.MinWidth = 0
	} else if v, ok := lengthBox(value, fsize, viewportW, cssDisplayNone); ok {
		style.MinWidth = v
		style.MinWidthPercent = -1
	}

	return true
}

func setMinHeightValue(style *ResolvedStyle, value string, fsize, viewportH float64) bool {
	if value == overflowAuto || value == cssDisplayNone {
		style.MinHeight = 0
		style.MinHeightPercent = -1
	} else if v, unit, ok := css.ParseLength(value); ok && unit == "%" {
		style.MinHeightPercent = v
		style.MinHeight = 0
	} else if v, ok := lengthBox(value, fsize, viewportH, cssDisplayNone); ok {
		style.MinHeight = v
		style.MinHeightPercent = -1
	}

	return true
}

func applyBoxMaxExtentProps(style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext) bool {
	switch prop {
	case "max-width":
		return setMaxWidthValue(style, value, fsize, ctx.viewportW)
	case "max-height":
		return setMaxHeightValue(style, value, fsize, ctx.viewportH)
	default:
		return false
	}
}

func setMaxWidthValue(style *ResolvedStyle, value string, fsize, viewportW float64) bool {
	if value == cssDisplayNone {
		style.MaxWidth = -1
		style.MaxWidthPercent = -1
	} else if v, unit, ok := css.ParseLength(value); ok && unit == "%" {
		style.MaxWidthPercent = v
		style.MaxWidth = -1
	} else if v, ok := lengthBox(value, fsize, viewportW, cssDisplayNone); ok {
		style.MaxWidth = v
		style.MaxWidthPercent = -1
	}

	return true
}

func setMaxHeightValue(style *ResolvedStyle, value string, fsize, viewportH float64) bool {
	if v, ok := lengthBox(value, fsize, viewportH, cssDisplayNone); ok {
		style.MaxHeight = v
	}

	return true
}

func applyMarginShorthandProps(style *ResolvedStyle, value string, fsize, viewportW float64) bool {
	setFourMargin(style, value, fsize, viewportW)

	return true
}

func applyMarginVerticalProps(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	switch prop {
	case "margin-top":
		style.MarginTop = marginLen(value, fsize, viewportW)
	case "margin-bottom":
		style.MarginBottom = marginLen(value, fsize, viewportW)
	default:
		return false
	}

	return true
}

func applyMarginHorizontalProps(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	switch prop {
	case "margin-right":
		style.MarginRight, style.MarginRightAuto = marginLenAuto(value, fsize, viewportW)
	case "margin-left":
		style.MarginLeft, style.MarginLeftAuto = marginLenAuto(value, fsize, viewportW)
	default:
		return false
	}

	return true
}

func applyPaddingShorthandProps(style *ResolvedStyle, value string, fsize, viewportW float64) bool {
	setFour(style, value,
		&style.PaddingTop, &style.PaddingRight, &style.PaddingBottom, &style.PaddingLeft,
		fsize, viewportW)

	return true
}

func applyPaddingSideProps(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	switch prop {
	case "padding-top":
		style.PaddingTop = marginLen(value, fsize, viewportW)
	case "padding-right":
		style.PaddingRight = marginLen(value, fsize, viewportW)
	case "padding-bottom":
		style.PaddingBottom = marginLen(value, fsize, viewportW)
	case "padding-left":
		style.PaddingLeft = marginLen(value, fsize, viewportW)
	default:
		return false
	}

	return true
}

// applyBorderGroup handles the border shorthand and per-side props.
func applyBorderGroup(
	style *ResolvedStyle, prop, value string, fsize float64, _ *styleContext, _ ResolvedStyle, _ bool,
) bool {
	switch prop {
	case "border":
		return applyBorderAllSides(style, value, fsize)
	case "border-top", "border-right", "border-bottom", "border-left":
		return applyBorderOneSide(style, prop, value, fsize)
	case borderWidthKeyword, "border-top-width", "border-right-width", "border-bottom-width", "border-left-width":
		return applyBorderWidthProps(style, prop, value, fsize)
	case borderStyleKeyword, borderColorKeyword:
		return applyBorderStyleColorProps(style, prop, value)
	default:
		return false
	}
}

func applyBorderAllSides(style *ResolvedStyle, value string, fsize float64) bool {
	if b, ok := parseBorder(value, fsize); ok {
		style.BorderTop, style.BorderRight, style.BorderBottom, style.BorderLeft = b, b, b, b
	}

	return true
}

func applyBorderOneSide(style *ResolvedStyle, prop, value string, fsize float64) bool {
	switch prop {
	case "border-top":
		setBorderSide(style, &style.BorderTop, value, fsize)
	case "border-right":
		setBorderSide(style, &style.BorderRight, value, fsize)
	case "border-bottom":
		setBorderSide(style, &style.BorderBottom, value, fsize)
	case "border-left":
		setBorderSide(style, &style.BorderLeft, value, fsize)
	default:
		return false
	}

	return true
}

func setBorderSide(_ *ResolvedStyle, side *border, value string, fsize float64) {
	if b, ok := parseBorder(value, fsize); ok {
		*side = b
	}
}

func applyBorderWidthProps(style *ResolvedStyle, prop, value string, fsize float64) bool {
	switch prop {
	case borderWidthKeyword:
		w := borderWidth(value, fsize)
		style.BorderTop.Width, style.BorderRight.Width, style.BorderBottom.Width, style.BorderLeft.Width = w, w, w, w
	case "border-top-width":
		style.BorderTop.Width = borderWidth(value, fsize)
	case "border-right-width":
		style.BorderRight.Width = borderWidth(value, fsize)
	case "border-bottom-width":
		style.BorderBottom.Width = borderWidth(value, fsize)
	case "border-left-width":
		style.BorderLeft.Width = borderWidth(value, fsize)
	default:
		return false
	}

	return true
}

func applyBorderStyleColorProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case borderStyleKeyword:
		s := value
		if s != solidKeyword && s != "dashed" && s != "dotted" {
			s = cssDisplayNone
		}

		style.BorderTop.Style, style.BorderRight.Style, style.BorderBottom.Style, style.BorderLeft.Style = s, s, s, s
	case borderColorKeyword:
		if r, g, b, _, ok := css.ParseColor(value); ok {
			c := [3]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255}
			style.BorderTop.Color, style.BorderRight.Color, style.BorderBottom.Color, style.BorderLeft.Color = c, c, c, c
		}
	default:
		return false
	}

	return true
}

// applyColorGroup handles foreground and background colors.
func applyColorGroup(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext, parent ResolvedStyle, hasParent bool,
) bool {
	if applyColorForegroundProps(style, prop, value, parent, hasParent) {
		return true
	}

	return applyColorBackgroundProps(style, prop, value)
}

func applyColorForegroundProps(style *ResolvedStyle, prop, value string, parent ResolvedStyle, hasParent bool) bool {
	switch prop {
	case "color":
		if value == inheritKeyword {
			if hasParent {
				style.Color = parent.Color
			}
		} else if r, g, b, _, ok := css.ParseColor(value); ok {
			style.Color = [3]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255}
		}
	default:
		return false
	}

	return true
}

func applyColorBackgroundProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "background-color":
		if r, g, b, a, ok := css.ParseColor(value); ok {
			style.BGColor = [4]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255, a}
		}
	case "background":
		// Shorthand: take the first parseable color token (ignore images/repeat).
		for _, tok := range strings.Fields(value) {
			if r, g, b, a, ok := css.ParseColor(tok); ok {
				style.BGColor = [4]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255, a}

				break
			}
		}
	default:
		return false
	}

	return true
}

// applyTextGroup handles typography and list props.
func applyTextGroup(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext,
	parent ResolvedStyle, hasParent bool,
) bool {
	if applyTextLayoutProps(style, prop, value) {
		return true
	}

	if applyTextWrapProps(style, prop, value) {
		return true
	}

	if applyTextDecorationProps(style, prop, value, parent, hasParent) {
		return true
	}

	if applyListProps(style, prop, value) {
		return true
	}

	return applyTextSpacingProps(style, prop, value, fsize, ctx)
}

func applyTextLayoutProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "line-height":
		style.LineHeight = lineHeight(value, style.FontSize)
	case "text-align":
		setTextAlignValue(style, value)
	case "vertical-align":
		setVerticalAlignValue(style, value)
	case "white-space":
		setWhiteSpaceValue(style, value)
	default:
		return false
	}

	return true
}

func setTextAlignValue(style *ResolvedStyle, value string) {
	switch value {
	case floatLeft, fxCenter, cssTextAlignJustify:
		style.TextAlign = value
	case floatRight, fxEnd:
		style.TextAlign = floatRight
	case fxStart:
		style.TextAlign = floatLeft
	}
}

func setVerticalAlignValue(style *ResolvedStyle, value string) {
	switch value {
	case "baseline", cssVerticalAlignTop, "middle", cssVerticalAlignBottom:
		style.VerticalAlign = value
	}
}

func setWhiteSpaceValue(style *ResolvedStyle, value string) {
	switch value {
	case contentNormal, "nowrap":
		style.WhiteSpace = value
	case "pre", "pre-wrap", "pre-line":
		style.WhiteSpace = "pre"
	}
}

func applyTextWrapProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "overflow-wrap", "word-wrap":
		setOverflowWrapValue(style, value)
	case "word-break":
		setWordBreakValue(style, value)
	default:
		return false
	}

	return true
}

func setOverflowWrapValue(style *ResolvedStyle, value string) {
	// word-wrap is the legacy alias of overflow-wrap.
	switch value {
	case contentNormal, "break-word", overflowWrapAnywhere:
		style.OverflowWrap = value
	case "break-spaces":
		// Treat like anywhere for line breaking (extra space preservation omitted).
		style.OverflowWrap = overflowWrapAnywhere
	}
}

func setWordBreakValue(style *ResolvedStyle, value string) {
	switch value {
	case contentNormal, "break-all", "keep-all":
		style.WordBreak = value
	case "break-word":
		// Legacy alias ≈ overflow-wrap:anywhere + word-break:normal.
		style.OverflowWrap = overflowWrapAnywhere
	}
}

func applyTextDecorationProps(style *ResolvedStyle, prop, value string, parent ResolvedStyle, hasParent bool) bool {
	switch prop {
	case "text-decoration":
		switch value {
		case cssTextDecorationUnderline:
			style.TextDecoration = cssTextDecorationUnderline
		case "line-through":
			style.TextDecoration = "line-through"
		case cssDisplayNone:
			style.TextDecoration = cssDisplayNone
		case inheritKeyword:
			if hasParent {
				style.TextDecoration = parent.TextDecoration
			}
		}
	default:
		return false
	}

	return true
}

func applyListProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "list-style-type":
		if t := parseListStyleType(value); t != "" {
			style.ListStyleType = t
		}
	case "list-style":
		// Shorthand: accept type keywords; ignore position/image for now.
		for _, tok := range strings.Fields(value) {
			if t := parseListStyleType(tok); t != "" {
				style.ListStyleType = t
			}
		}
	default:
		return false
	}

	return true
}

func applyTextSpacingProps(style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext) bool {
	switch prop {
	case "letter-spacing":
		style.LetterSpacing = marginLen(value, fsize, ctx.viewportW)
	case "text-indent":
		style.TextIndent = marginLen(value, fsize, ctx.viewportW)
	default:
		return false
	}

	return true
}

// applyTableBreakGroup handles table borders/spacing and page-break props.
func applyTableBreakGroup(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext, _ ResolvedStyle, _ bool,
) bool {
	switch prop {
	case "border-collapse", "border-spacing", "table-layout":
		return applyTableProps(style, prop, value, fsize, ctx.viewportW)
	case "page-break-before", "break-before", "page-break-after", "break-after",
		"page-break-inside", "break-inside":
		return applyPageBreakProps(style, prop, value)
	case "orphans", "widows":
		return applyOrphansWidowsProps(style, prop, value)
	case "container-type", "container-name", containerKeyword:
		return applyContainerProps(style, prop, value)
	default:
		return false
	}
}

func applyTableProps(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	switch prop {
	case "border-collapse":
		if value == "collapse" || value == "separate" {
			style.BorderCollapse = value
		}
	case "border-spacing":
		style.BorderSpacing = marginLen(value, fsize, viewportW)
	case "table-layout":
		if value == "fixed" || value == overflowAuto {
			style.TableLayout = value
		}
	default:
		return false
	}

	return true
}

func applyPageBreakProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "page-break-before", "break-before":
		return applyBreakBeforeProps(style, value)
	case "page-break-after", "break-after":
		return applyBreakAfterProps(style, value)
	case "page-break-inside", "break-inside":
		return applyBreakInsideProps(style, value)
	default:
		return false
	}
}

func applyBreakBeforeProps(style *ResolvedStyle, value string) bool {
	// column → page always is a multicol approximation.
	// avoid-column is column-only (CSS Break) — do NOT map to page avoid
	// (wiki .mw-references-columns li{break-inside:avoid-column} was
	// leaving huge gaps between reference list items).
	switch value {
	case pageBreakAlways, fxCol, pageKeyword, floatLeft, floatRight:
		style.PageBreakBefore = pageBreakAlways
	case avoidKeyword, avoidPageValue:
		style.PageBreakBefore = avoidKeyword
	default:
		return false
	}

	return true
}

func applyBreakAfterProps(style *ResolvedStyle, value string) bool {
	switch value {
	case pageBreakAlways, fxCol, pageKeyword, floatLeft, floatRight:
		style.PageBreakAfter = pageBreakAlways
	case avoidKeyword, avoidPageValue:
		style.PageBreakAfter = avoidKeyword
	default:
		return false
	}

	return true
}

func applyBreakInsideProps(style *ResolvedStyle, value string) bool {
	// avoid-column: ignored for page pagination.
	switch value {
	case pageBreakAlways, pageKeyword:
		style.PageBreakInside = pageBreakAlways
	case avoidKeyword, avoidPageValue:
		style.PageBreakInside = avoidKeyword
	default:
		return false
	}

	return true
}

func applyOrphansWidowsProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "orphans":
		if n, ok := parseOrphansWidowsInt(value); ok {
			style.Orphans = n
		}
	case "widows":
		if n, ok := parseOrphansWidowsInt(value); ok {
			style.Widows = n
		}
	default:
		return false
	}

	return true
}

func applyContainerProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "container-type":
		switch strings.ToLower(value) {
		case contentNormal, "size", "inline-size":
			style.ContainerType = strings.ToLower(value)
		}
	case "container-name":
		style.ContainerName = css.ParseContainerNameValue(value)
	case containerKeyword:
		name, ctype := css.ParseContainerShorthand(value)
		style.ContainerName = name

		if ctype != "" {
			style.ContainerType = ctype
		}
	default:
		return false
	}

	return true
}

// applyTransformGroup handles transform and transform-origin.
func applyTransformGroup(
	style *ResolvedStyle, prop, value string, fsize float64, _ *styleContext, _ ResolvedStyle, _ bool,
) bool {
	switch prop {
	case "transform":
		// Animations/transitions ignored: cascaded static value only.
		if m, has, ok := parseTransformList(value, fsize); ok {
			style.Transform = m
			style.HasTransform = has
		}
	case "transform-origin":
		if spec, ok := parseTransformOrigin(value, fsize); ok {
			style.TransformOrigin = spec
		}
	default:
		return false
	}

	return true
}

// applyIgnoredGroup parses but ignores the animation/transition family.
func applyIgnoredGroup(st *ResolvedStyle, prop, value string) {
	_ = st
	_ = prop
	_ = value
}

// parseOrphansWidowsInt accepts a CSS <integer ≥ 1>. Invalid values are
// ignored by the caller (declaration dropped).
func parseOrphansWidowsInt(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	n, err := strconv.Atoi(value)
	if err != nil || n < 1 {
		return 0, false
	}

	return n, true
}

// parseColumnsShorthand parses CSS columns: [ <column-width> || <column-count> ].
// Both longhands reset to auto before applying tokens (CSS Cascading).
func parseColumnsShorthand(sty *ResolvedStyle, value string, fsize, viewportW float64) {
	sty.ColumnCount = 0
	sty.ColumnWidth = -1

	for _, tok := range strings.Fields(value) {
		tok = strings.TrimSpace(tok)
		if tok == "" || tok == overflowAuto {
			continue
		}

		if n, err := strconv.Atoi(tok); err == nil && n >= 1 {
			sty.ColumnCount = n

			continue
		}

		if v, ok := lengthBox(tok, fsize, viewportW, overflowAuto); ok && v >= 0 {
			sty.ColumnWidth = v
		}
	}
}

// isMulticol reports whether st establishes a multi-column container
// (column-count or column-width is not auto).
func isMulticol(st ResolvedStyle) bool {
	return st.ColumnCount > 0 || st.ColumnWidth >= 0
}

// parseFontShorthand handles "font: italic bold 12px/1.4 Arial, sans-serif".
func parseFontShorthand(style *ResolvedStyle, value string, remBase float64) {
	parts := strings.Fields(value)
	for idx, page := range parts {
		if applyFontStyleKeyword(style, page) {
			continue
		}

		if n, ok := css.ParseNumber(page); ok && n >= 100 && n <= 900 {
			style.FontWeight = int(n)

			continue
		}
		// first size token
		rest, lineH := fontSizeToken(page, style.FontSize)
		if lineH >= 0 {
			style.LineHeight = lineH
		}

		style.FontSize = fontSize(rest, style.FontSize, remBase)

		if idx+1 < len(parts) {
			if fam := css.ParseFontFamily(strings.Join(parts[idx+1:], " ")); len(fam) > 0 {
				style.FontFamily = fam
			}
		}

		return
	}
}

// applyFontStyleKeyword handles the italic/oblique/bold style keywords; false
// when the token is not a font style keyword.
func applyFontStyleKeyword(style *ResolvedStyle, page string) bool {
	switch page {
	case "italic", "oblique":
		style.FontItalic = true
	case "bold":
		style.FontWeight = fontWeightBold
	default:
		return false
	}

	return true
}

// fontSizeToken splits "12px/1.4" into the size part and line-height (or -1).
func fontSizeToken(page string, fsize float64) (string, float64) {
	if j := strings.IndexByte(page, '/'); j >= 0 {
		return page[:j], lineHeight(page[j+1:], fsize)
	}

	return page, -1
}

// parseFlexShorthand handles flex: none | auto | <grow> | <grow> <shrink> | <grow> <shrink> <basis>.
func parseFlexShorthand(style *ResolvedStyle, value string, fontSize, pctBase float64) {
	value = strings.TrimSpace(value)
	switch value {
	case cssDisplayNone:
		style.FlexGrow, style.FlexShrink = 0, 0
		style.FlexBasis, style.FlexBasisPercent = -1, -1

		return
	case overflowAuto:
		style.FlexGrow, style.FlexShrink = 1, 1
		style.FlexBasis, style.FlexBasisPercent = -1, -1

		return
	}

	parts := strings.Fields(value)
	switch len(parts) {
	case 0:
		return
	case 1:
		parseFlexOne(style, parts[0], fontSize, pctBase)
	case two:
		parseFlexTwo(style, parts, fontSize, pctBase)
	default:
		parseFlexThree(style, parts, fontSize, pctBase)
	}
}

// flexIsBasis reports whether a token can be a flex-basis value.
func flexIsBasis(tok string) bool {
	if tok == overflowAuto || tok == "content" {
		return true
	}

	_, _, ok := css.ParseLength(tok)

	return ok
}

// flexSetBasis writes the basis longhands from a token.
func flexSetBasis(style *ResolvedStyle, tok string, fontSize, pctBase float64) {
	if tok == overflowAuto || tok == "content" {
		style.FlexBasis = -1
		style.FlexBasisPercent = -1

		return
	}

	if v, unit, ok := css.ParseLength(tok); ok && unit == "%" {
		style.FlexBasisPercent = v
		style.FlexBasis = -1

		return
	}

	if v, ok := lengthBox(tok, fontSize, pctBase, overflowAuto); ok {
		style.FlexBasis = v
		style.FlexBasisPercent = -1
	}
}

func parseFlexOne(style *ResolvedStyle, part string, fontSize, pctBase float64) {
	if g, err := strconv.ParseFloat(part, 64); err == nil {
		// flex: <number> → grow <number>, shrink 1, basis 0%
		style.FlexGrow = g
		style.FlexShrink = 1
		style.FlexBasis = -1
		style.FlexBasisPercent = 0

		return
	}

	if flexIsBasis(part) {
		style.FlexGrow, style.FlexShrink = 1, 1

		flexSetBasis(style, part, fontSize, pctBase)
	}
}

func parseFlexTwo(style *ResolvedStyle, parts []string, fontSize, pctBase float64) {
	g, errG := strconv.ParseFloat(parts[0], 64)
	if errG != nil {
		return
	}

	style.FlexGrow = g
	if sh, err := strconv.ParseFloat(parts[1], 64); err == nil {
		style.FlexShrink = sh
		style.FlexBasis = -1
		style.FlexBasisPercent = 0

		return
	}

	style.FlexShrink = 1

	if flexIsBasis(parts[1]) {
		flexSetBasis(style, parts[1], fontSize, pctBase)
	}
}

func parseFlexThree(style *ResolvedStyle, parts []string, fontSize, pctBase float64) {
	gap, errG := strconv.ParseFloat(parts[0], 64)
	shval, errS := strconv.ParseFloat(parts[1], 64)

	if errG != nil || errS != nil {
		return
	}

	style.FlexGrow, style.FlexShrink = gap, shval

	flexSetBasis(style, parts[2], fontSize, pctBase)
}

// setFourMargin applies a margin shorthand and tracks horizontal auto.
func setFourMargin(sty *ResolvedStyle, value string, fsize, ctxW float64) {
	val := strings.Fields(value)
	sty.MarginLeftAuto, sty.MarginRightAuto = false, false

	switch len(val) {
	case 0:
		return
	case 1:
		sty.MarginTop = marginLen(val[0], fsize, ctxW)
		sty.MarginRight, sty.MarginRightAuto = marginLenAuto(val[0], fsize, ctxW)
		sty.MarginBottom = marginLen(val[0], fsize, ctxW)
		sty.MarginLeft, sty.MarginLeftAuto = marginLenAuto(val[0], fsize, ctxW)
	case two:
		sty.MarginTop = marginLen(val[0], fsize, ctxW)
		sty.MarginRight, sty.MarginRightAuto = marginLenAuto(val[1], fsize, ctxW)
		sty.MarginBottom = marginLen(val[0], fsize, ctxW)
		sty.MarginLeft, sty.MarginLeftAuto = marginLenAuto(val[1], fsize, ctxW)
	case three:
		sty.MarginTop = marginLen(val[0], fsize, ctxW)
		sty.MarginRight, sty.MarginRightAuto = marginLenAuto(val[1], fsize, ctxW)
		sty.MarginBottom = marginLen(val[2], fsize, ctxW)
		sty.MarginLeft, sty.MarginLeftAuto = marginLenAuto(val[1], fsize, ctxW)
	default:
		sty.MarginTop = marginLen(val[0], fsize, ctxW)
		sty.MarginRight, sty.MarginRightAuto = marginLenAuto(val[1], fsize, ctxW)
		sty.MarginBottom = marginLen(val[2], fsize, ctxW)
		sty.MarginLeft, sty.MarginLeftAuto = marginLenAuto(val[3], fsize, ctxW)
	}
}

func setFour(_ *ResolvedStyle, value string, top, right, bottom, left *float64, fsVal, ctxW float64) {
	val := strings.Fields(value)
	if len(val) == 0 {
		return
	}

	if len(val) == 1 {
		*top = marginLen(val[0], fsVal, ctxW)
		*right, *bottom, *left = *top, *top, *top

		return
	}

	if len(val) == two {
		*top = marginLen(val[0], fsVal, ctxW)
		*right = marginLen(val[1], fsVal, ctxW)
		*bottom, *left = *top, *right

		return
	}

	if len(val) == three {
		*top = marginLen(val[0], fsVal, ctxW)
		*right = marginLen(val[1], fsVal, ctxW)
		*bottom = marginLen(val[2], fsVal, ctxW)
		*left = *right

		return
	}

	*top = marginLen(val[0], fsVal, ctxW)
	*right = marginLen(val[1], fsVal, ctxW)
	*bottom = marginLen(val[2], fsVal, ctxW)
	*left = marginLen(val[3], fsVal, ctxW)
}

func parseBorder(value string, _ float64) (border, bool) {
	var boxNode border

	for _, face := range strings.Fields(value) {
		switch face {
		case "solid", "dashed", "dotted":
			boxNode.Style = face
		case cssDisplayNone, overflowHidden:
			boxNode.Style = cssDisplayNone
		default:
			if r, g, bb, _, ok := css.ParseColor(face); ok {
				boxNode.Color = [3]float64{float64(r) / 255, float64(g) / 255, float64(bb) / 255}
			} else if v, _, ok := css.ParseLength(face); ok {
				boxNode.Width = v
			}
		}
	}

	if boxNode.Style == "" {
		boxNode.Style = "solid"
	}

	if boxNode.Width == 0 {
		boxNode.Width = 1
	}

	return boxNode, boxNode.Style != cssDisplayNone
}

func borderWidth(value string, _ float64) float64 {
	switch value {
	case "thin":
		return pxToPt(1)
	case "medium":
		return pxToPt(three)
	case "thick":
		return pxToPt(borderWidthMediumPx)
	}

	if v, _, ok := css.ParseLength(value); ok {
		return v
	}

	return 0
}

func fontSize(value string, parent, remBase float64) float64 {
	if remBase <= 0 {
		remBase = pxToPt(cssPxRoot)
	}

	if pt, ok := fontSizeKeyword(value, parent); ok {
		return pt
	}

	if val, unit, ok := css.ParseLength(value); ok {
		switch unit {
		case "%":
			return parent * val / cssPercent
		case remUnit:
			return remBase * val
		default:
			if pt, ok := css.LengthToPt(val, unit, parent); ok {
				return pt
			}
		}
	}

	return parent
}

// fontSizeKeyword resolves the named font-size keywords relative to parent.
func fontSizeKeyword(value string, parent float64) (float64, bool) {
	switch value {
	case "xx-small":
		return pxToPt(fontSizeXSmallPx), true
	case "x-small":
		return pxToPt(fontSizeSmallPx), true
	case "small":
		return pxToPt(fontSizeMediumPx), true
	case "medium":
		return pxToPt(cssPxRoot), true
	case "large":
		return pxToPt(fontSizeLargePx), true
	case "x-large":
		return pxToPt(twoLineRoomPt), true
	case "xx-large":
		return pxToPt(fontSizeXXXLargePx), true
	case "smaller":
		return parent * smallerFontRatio, true
	case "larger":
		return parent * defaultLineHeightRatio, true
	}

	return 0, false
}

func lineHeight(value string, fsize float64) float64 {
	if value == contentNormal {
		return 0
	}

	if v, ok := css.ParseNumber(value); ok {
		return v * fsize
	}

	if v, unit, ok := css.ParseLength(value); ok {
		if unit == "%" {
			return fsize * v / cssPercent
		}

		if pt, ok := css.LengthToPt(v, unit, fsize); ok {
			return pt
		}
	}

	return 0
}

// parseOverflowKeyword accepts CSS overflow keywords used for sticky scrollport
// detection. clip is treated like hidden (scroll container, no user scroll).
func parseOverflowKeyword(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "visible", overflowHidden, "scroll", overflowAuto, "clip":
		return strings.ToLower(strings.TrimSpace(value)), true
	}

	return "", false
}

// parseListStyleType returns a known list-style-type keyword, or "" if unknown.
func parseListStyleType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "disc", "circle", "square", "decimal", "decimal-leading-zero",
		"lower-roman", "upper-roman", "lower-alpha", "lower-latin",
		"upper-alpha", "upper-latin", cssDisplayNone:
		return strings.ToLower(strings.TrimSpace(value))
	}

	return ""
}

// overflowCreatesStickyScrollport reports whether overflow establishes a sticky
// scrollport (CSS Position 3 / Overflow 3). PDF has no user scroll, so sticky
// inside these boxes clamps at scroll offset 0 against the box edges.
func overflowCreatesStickyScrollport(overflow string) bool {
	switch overflow {
	case overflowAuto, "scroll", overflowHidden, "clip":
		return true
	}

	return false
}

// lengthBox parses a length for box-sizing properties. "auto" maps to -1,
// "none" to -1 as well (max-*). Percentages resolve against the containing
// block dimension (approximated by viewport at this phase).
func lengthBox(value string, fsize, containing float64, autoValue string) (float64, bool) {
	if value == autoValue || value == "inherit" || value == "initial" {
		return -1, true
	}

	val, unit, ok := css.ParseLength(value)
	if !ok {
		return 0, false
	}

	switch unit {
	case "%", "vw", "vh":
		return containing * val / cssPercent, true
	default:
		if point, ok := css.LengthToPt(val, unit, fsize); ok {
			// rem uses LengthToPt's 16px root; keep remBase-independent path
			// matching prior lengthBox (rem → 12pt * v via pxToPt(16)).
			if unit == remUnit {
				return pxToPt(cssPxRoot) * val, true
			}

			return point, true
		}
	}

	return 0, false
}

// marginLenAuto parses a horizontal margin; auto yields (0, true).
func marginLenAuto(value string, fs, ctxW float64) (float64, bool) {
	if value == overflowAuto {
		return 0, true
	}

	return marginLen(value, fs, ctxW), false
}

// marginLen parses a margin/padding/letter-spacing length in points; 0 when
// unparseable. Percentages resolve against the containing width.
func marginLen(value string, fsize, ctxW float64) float64 {
	if value == overflowAuto || value == "inherit" || value == "initial" {
		return 0
	}

	val, unit, ok := css.ParseLength(value)
	if !ok {
		return 0
	}

	if unit == "%" {
		return ctxW * val / cssPercent
	}

	if unit == "rem" {
		return pxToPt(cssPxRoot) * val
	}

	if pt, ok := css.LengthToPt(val, unit, fsize); ok {
		return pt
	}

	return 0
}

func pxToPt(px float64) float64 { return px * pxToPtFactor }

// parseGridAutoFlowValue normalizes grid-auto-flow to one of:
// "row", "column", "dense", "row dense", "column dense".
func parseGridAutoFlowValue(value string) string {
	toks := strings.Fields(strings.ToLower(strings.TrimSpace(value)))
	row, col, dense := false, false, false

	for _, t := range toks {
		switch t {
		case fxRow:
			row = true
		case fxCol:
			col = true
		case gridFlowDense:
			dense = true
		}
	}

	return gridAutoFlowName(row, col, dense)
}

// gridAutoFlowName maps the parsed tokens onto the canonical keyword.
func gridAutoFlowName(row, col, dense bool) string {
	switch {
	case col && dense:
		return gridFlowColumnDense
	case row && dense:
		return gridFlowRowDense
	case dense:
		return gridFlowDense
	case col:
		return fxCol
	default:
		return fxRow
	}
}

// parseGridArea handles grid-area: <custom-ident> or the lite line form
// row-start / column-start / row-end / column-end (and 1–2 slash forms).
func parseGridArea(sty *ResolvedStyle, value string) {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, overflowAuto) {
		sty.GridArea = ""

		return
	}

	parts := strings.Split(value, "/")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	if len(parts) == 1 {
		tok := parts[0]
		if _, err := strconv.Atoi(tok); err == nil {
			// Single line index → row-start (CSS shorthand lite).
			parseGridRow(sty, tok)
			sty.GridArea = ""

			return
		}

		if strings.HasPrefix(tok, "span ") {
			parseGridRow(sty, tok)
			sty.GridArea = ""

			return
		}
		// Named area.
		sty.GridArea = tok

		return
	}

	sty.GridArea = ""

	switch len(parts) {
	case two:
		// CSS: row-start / column-start (omitted ends copy starts → span 1).
		parseGridRow(sty, parts[0])
		parseGridColumn(sty, parts[1])
	case three:
		// row-start / column-start / row-end
		parseGridRow(sty, parts[0])
		parseGridColumn(sty, parts[1])
		applyGridLineEnd(sty, true, parts[2])
	default:
		// row-start / column-start / row-end / column-end
		parseGridRow(sty, parts[0])
		parseGridColumn(sty, parts[1])
		applyGridLineEnd(sty, true, parts[2])
		applyGridLineEnd(sty, false, parts[3])
	}
}

// applyGridLineEnd sets span from an end line or "span N" on row (isRow) or column.
func applyGridLineEnd(style *ResolvedStyle, isRow bool, end string) {
	target := gridTarget(style, isRow)

	end = strings.TrimSpace(end)
	if strings.HasPrefix(end, "span ") {
		node, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(end, "span ")))
		if err != nil || node < 1 {
			return
		}

		*target.span = node

		return
	}

	val, err := strconv.Atoi(end)
	if err != nil {
		return
	}

	if *target.start > 0 {
		sp := val - *target.start
		if sp < 1 {
			sp = 1
		}

		*target.span = sp
	}
}

func parseGridColumn(st *ResolvedStyle, value string) { parseGridLineAt(colGridTarget(st), value) }

func parseGridRow(st *ResolvedStyle, value string) { parseGridLineAt(rowGridTarget(st), value) }

// gridLineTarget points at the start/span fields of one grid axis.
type gridLineTarget struct {
	start *int
	span  *int
}

func rowGridTarget(st *ResolvedStyle) gridLineTarget {
	return gridLineTarget{start: &st.GridRowStart, span: &st.GridRowSpan}
}

func colGridTarget(st *ResolvedStyle) gridLineTarget {
	return gridLineTarget{start: &st.GridColumnStart, span: &st.GridColumnSpan}
}

func gridTarget(st *ResolvedStyle, isRow bool) gridLineTarget {
	if isRow {
		return rowGridTarget(st)
	}

	return colGridTarget(st)
}

// parseGridLineAt handles "N", "span N", "N / M" and "N / span M" for one
// grid axis.
func parseGridLineAt(target gridLineTarget, value string) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "span ") {
		applyGridSpanToken(target, strings.TrimSpace(strings.TrimPrefix(value, "span ")))

		return
	}

	// "1 / 3" or "1 / span 2"
	parts := strings.Split(value, "/")
	if len(parts) == 1 {
		if v, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil && v > 0 {
			*target.start = v
			*target.span = 1
		}

		return
	}

	setGridStartToken(target, parts[0])
	applyGridEndToken(target, parts[1])
}

// setGridStartToken applies a positive start line index.
func setGridStartToken(target gridLineTarget, token string) {
	if v, err := strconv.Atoi(strings.TrimSpace(token)); err == nil && v > 0 {
		*target.start = v
	}
}

// applyGridEndToken applies a "span N" or absolute end line; absolute ends
// become spans relative to the start line.
func applyGridEndToken(target gridLineTarget, end string) {
	end = strings.TrimSpace(end)
	if strings.HasPrefix(end, "span ") {
		applyGridSpanToken(target, strings.TrimSpace(strings.TrimPrefix(end, "span ")))

		return
	}

	if v, err := strconv.Atoi(end); err == nil && *target.start > 0 {
		sp := v - *target.start
		if sp < 1 {
			sp = 1
		}

		*target.span = sp
	}
}

// applyGridSpanToken sets the span when token is a positive integer.
func applyGridSpanToken(target gridLineTarget, token string) {
	if n, err := strconv.Atoi(token); err == nil && n > 0 {
		*target.span = n
	}
}

// uaDecls is the user-agent declaration table for element names. Lookup is
// per element; unknown names get the initial values.
var uaDecls = map[string][]css.Declaration{ //nolint:gochecknoglobals // static UA table
	"html": {{Prop: "display", Value: "block"}}, //nolint:exhaustruct // intentional zero fields
	"body": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "8px"},    //nolint:exhaustruct // intentional zero fields
	},
	divElementName: {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"section": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"article": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"header": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"footer": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"main": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"aside": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"nav": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"form": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"fieldset": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"figure": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"figcaption": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"blockquote": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"address": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"dl": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"dd": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"details": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"summary": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
	"p": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "1em 0"},  //nolint:exhaustruct // intentional zero fields
	},
	"pre": {
		// Match browser UA: preserve newlines/spaces; monospace is a
		// soft preference (we fall back to Liberation Sans metrics).
		{Prop: "display", Value: "block"},         //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "1em 0"},          //nolint:exhaustruct // intentional zero fields
		{Prop: "white-space", Value: "pre"},       //nolint:exhaustruct // intentional zero fields
		{Prop: "font-family", Value: "monospace"}, //nolint:exhaustruct // intentional zero fields
	},
	"code": {
		{Prop: "font-family", Value: "monospace"}, //nolint:exhaustruct // intentional zero fields
	},
	"kbd": {
		{Prop: "font-family", Value: "monospace"}, //nolint:exhaustruct // intentional zero fields
	},
	"samp": {
		{Prop: "font-family", Value: "monospace"}, //nolint:exhaustruct // intentional zero fields
	},
	"h1": {
		{Prop: "display", Value: "block"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "font-size", Value: "2em"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "0.67em 0"},  //nolint:exhaustruct // intentional zero fields
		{Prop: "font-weight", Value: "bold"}, //nolint:exhaustruct // intentional zero fields
	},
	"h2": {
		{Prop: "display", Value: "block"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "font-size", Value: "1.5em"},  //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "0.83em 0"},  //nolint:exhaustruct // intentional zero fields
		{Prop: "font-weight", Value: "bold"}, //nolint:exhaustruct // intentional zero fields
	},
	"h3": {
		{Prop: "display", Value: "block"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "font-size", Value: "1.17em"}, //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "1em 0"},     //nolint:exhaustruct // intentional zero fields
		{Prop: "font-weight", Value: "bold"}, //nolint:exhaustruct // intentional zero fields
	},
	"h4": {
		{Prop: "display", Value: "block"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "font-weight", Value: "bold"}, //nolint:exhaustruct // intentional zero fields
		{Prop: "font-size", Value: "1em"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "1.33em 0"},  //nolint:exhaustruct // intentional zero fields
	},
	"h5": {
		{Prop: "display", Value: "block"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "font-weight", Value: "bold"}, //nolint:exhaustruct // intentional zero fields
		{Prop: "font-size", Value: "1em"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "1.33em 0"},  //nolint:exhaustruct // intentional zero fields
	},
	"h6": {
		{Prop: "display", Value: "block"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "font-weight", Value: "bold"}, //nolint:exhaustruct // intentional zero fields
		{Prop: "font-size", Value: "1em"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "1.33em 0"},  //nolint:exhaustruct // intentional zero fields
	},
	"ul": {
		{Prop: "display", Value: "block"},        //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "1em 0"},         //nolint:exhaustruct // intentional zero fields
		{Prop: "padding-left", Value: "40px"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "list-style-type", Value: "disc"}, //nolint:exhaustruct // intentional zero fields
	},
	"menu": {
		{Prop: "display", Value: "block"},        //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "1em 0"},         //nolint:exhaustruct // intentional zero fields
		{Prop: "padding-left", Value: "40px"},    //nolint:exhaustruct // intentional zero fields
		{Prop: "list-style-type", Value: "disc"}, //nolint:exhaustruct // intentional zero fields
	},
	"ol": {
		{Prop: "display", Value: "block"},           //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "1em 0"},            //nolint:exhaustruct // intentional zero fields
		{Prop: "padding-left", Value: "40px"},       //nolint:exhaustruct // intentional zero fields
		{Prop: "list-style-type", Value: "decimal"}, //nolint:exhaustruct // intentional zero fields
	},
	"li": {
		{Prop: "display", Value: "list-item"}, //nolint:exhaustruct // intentional zero fields
	},
	"table": {
		{Prop: "display", Value: "table"},      //nolint:exhaustruct // intentional zero fields
		{Prop: "border-spacing", Value: "2px"}, //nolint:exhaustruct // intentional zero fields
	},
	"thead": {
		{Prop: "display", Value: "table-header-group"}, //nolint:exhaustruct // intentional zero fields
	},
	"tfoot": {
		{Prop: "display", Value: "table-footer-group"}, //nolint:exhaustruct // intentional zero fields
	},
	"tbody": {
		{Prop: "display", Value: "table-row-group"}, //nolint:exhaustruct // intentional zero fields
	},
	"tr": {
		{Prop: "display", Value: "table-row"}, //nolint:exhaustruct // intentional zero fields
	},
	"td": {
		{Prop: "display", Value: "table-cell"}, //nolint:exhaustruct // intentional zero fields
		{Prop: "padding", Value: "1px"},        //nolint:exhaustruct // intentional zero fields
	},
	"th": {
		{Prop: "display", Value: "table-cell"}, //nolint:exhaustruct // intentional zero fields
		{Prop: "padding", Value: "1px"},        //nolint:exhaustruct // intentional zero fields
		{Prop: "text-align", Value: "center"},  //nolint:exhaustruct // intentional zero fields
		{Prop: "font-weight", Value: "bold"},   //nolint:exhaustruct // intentional zero fields
	},
	"img": {
		{Prop: "display", Value: "inline-block"}, //nolint:exhaustruct // intentional zero fields
	},
	"hr": {
		{Prop: "display", Value: "block"},     //nolint:exhaustruct // intentional zero fields
		{Prop: "border", Value: "1px inset"},  //nolint:exhaustruct // intentional zero fields
		{Prop: "margin", Value: "0.5em auto"}, //nolint:exhaustruct // intentional zero fields
	},
	"a": {
		{Prop: "color", Value: "#0000ee"},             //nolint:exhaustruct // intentional zero fields
		{Prop: "text-decoration", Value: "underline"}, //nolint:exhaustruct // intentional zero fields
	},
	"b": {
		{Prop: "font-weight", Value: "bold"}, //nolint:exhaustruct // intentional zero fields
	},
	"strong": {
		{Prop: "font-weight", Value: "bold"}, //nolint:exhaustruct // intentional zero fields
	},
	"i": {
		{Prop: "font-style", Value: "italic"}, //nolint:exhaustruct // intentional zero fields
	},
	"em": {
		{Prop: "font-style", Value: "italic"}, //nolint:exhaustruct // intentional zero fields
	},
	"cite": {
		{Prop: "font-style", Value: "italic"}, //nolint:exhaustruct // intentional zero fields
	},
	"dfn": {
		{Prop: "font-style", Value: "italic"}, //nolint:exhaustruct // intentional zero fields
	},
	"var": {
		{Prop: "font-style", Value: "italic"}, //nolint:exhaustruct // intentional zero fields
	},
	"u": {
		{Prop: "text-decoration", Value: "underline"}, //nolint:exhaustruct // intentional zero fields
	},
	"s": {
		{Prop: "text-decoration", Value: "line-through"}, //nolint:exhaustruct // intentional zero fields
	},
	"strike": {
		{Prop: "text-decoration", Value: "line-through"}, //nolint:exhaustruct // intentional zero fields
	},
	"del": {
		{Prop: "text-decoration", Value: "line-through"}, //nolint:exhaustruct // intentional zero fields
	},
	"small": {
		{Prop: "font-size", Value: "smaller"}, //nolint:exhaustruct // intentional zero fields
	},
	"big": {
		{Prop: "font-size", Value: "larger"}, //nolint:exhaustruct // intentional zero fields
	},
	"center": {
		{Prop: "text-align", Value: "center"}, //nolint:exhaustruct // intentional zero fields
	},
	"title": {
		{Prop: "display", Value: cssDisplayNone}, //nolint:exhaustruct // intentional zero fields
	},
	styleElement: {
		{Prop: "display", Value: cssDisplayNone}, //nolint:exhaustruct // intentional zero fields
	},
	"script": {
		{Prop: "display", Value: cssDisplayNone}, //nolint:exhaustruct // intentional zero fields
	},
	"meta": {
		{Prop: "display", Value: cssDisplayNone}, //nolint:exhaustruct // intentional zero fields
	},
	"link": {
		{Prop: "display", Value: cssDisplayNone}, //nolint:exhaustruct // intentional zero fields
	},
	"head": {
		{Prop: "display", Value: cssDisplayNone}, //nolint:exhaustruct // intentional zero fields
	},
	"textarea": {
		{Prop: "white-space", Value: "pre"},       //nolint:exhaustruct // intentional zero fields
		{Prop: "font-family", Value: "monospace"}, //nolint:exhaustruct // intentional zero fields
	},
	"br": {
		{Prop: "display", Value: "block"}, //nolint:exhaustruct // intentional zero fields
	},
}

// uaRules returns the user-agent declarations for an element name.
func uaRules(name string) []css.Declaration {
	return uaDecls[name]
}
