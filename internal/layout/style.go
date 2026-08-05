package layout

import (
	"sort"
	"strconv"
	"strings"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

// ResolvedStyle is the used style of one element: values the layout engine
// consumes, in points (or unitless where noted). Only the phase-04 subset is
// modeled; everything else keeps its initial value.
type ResolvedStyle struct {
	Display             string
	Position            string  // "static" | "relative" | "absolute" | "fixed" | "sticky"
	Float               string  // "none" | "left" | "right"
	Clear               string  // "none" | "left" | "right" | "both"
	BoxSizing           string  // "content-box" | "border-box"
	Top                 float64 // position offsets (pt); 0 = unset for absolute uses Auto flags
	Right               float64
	Bottom              float64
	Left                float64
	TopAuto             bool
	RightAuto           bool
	BottomAuto          bool
	LeftAuto            bool
	FlexDirection       string  // "row" | "column" | "row-reverse" | "column-reverse"
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
	ColumnSpan          string  // "none" | "all" (multicol spanner)
	ColumnFill          string  // "balance" | "auto"
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
	GridAutoFlow        string  // "row" | "column" | "dense" | "row dense" | "column dense"
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
	MinHeight           float64
	MinHeightPercent    float64 // >=0 means % of CB height; indefinite → ignore
	MaxHeight           float64
	Overflow            string // "visible" | "hidden" | "scroll" | "auto" | "clip" — sticky scrollport when non-visible
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
	TextAlign           string  // "left" | "right" | "center" | "justify"
	VerticalAlign       string  // "baseline" | "top" | "middle" | "bottom"
	WhiteSpace          string  // "normal" | "nowrap" | "pre"
	TextDecoration      string  // "none" | "underline" | "line-through"
	LetterSpacing       float64
	TextIndent          float64
	BorderCollapse      string // "separate" | "collapse"
	BorderSpacing       float64
	TableLayout         string // "auto" | "fixed"
	IsReplaced          bool   // img, hr
	PageBreakBefore     string // "" | "always" | "avoid"
	PageBreakAfter      string // "" | "always" | "avoid"
	PageBreakInside     string // "" | "always" | "avoid"
	Orphans             int    // CSS orphans; inherited; initial 2; integer ≥ 1
	Widows              int    // CSS widows; inherited; initial 2; integer ≥ 1
	ContainerType       string // "" | "normal" | "inline-size" | "size"
	ContainerName       string // space-separated lower-case names; empty = none
	// Static 2D CSS transforms (paint-time CTM; sibling flow unchanged).
	Transform       Matrix2D
	HasTransform    bool
	TransformOrigin transformOriginSpec
	Opacity         float64 // 0..1; initial 1; also from filter:opacity()
}

type border struct {
	Width float64
	Style string // "none" | "solid" | "dashed" | "dotted"
	Color [3]float64
}

// initialStyle returns the CSS initial values.
func initialStyle() ResolvedStyle {
	return ResolvedStyle{
		Display:          "inline",
		Position:         "static",
		Float:            "none",
		FlexGrow:         0,
		FlexShrink:       1,
		FlexBasis:        -1,
		FlexBasisPercent: -1,
		Clear:            "none",
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
		AlignSelf:        "auto",
		JustifyItems:     "stretch",
		JustifySelf:      "auto",
		ColumnGapNormal:  true,
		ColumnWidth:      -1,
		ColumnSpan:       "none",
		ColumnFill:       "balance",
		Width:            -1,
		WidthPercent:     -1,
		Height:           -1,
		HeightPercent:    -1,
		MinWidth:         0,
		MinWidthPercent:  -1,
		MaxWidth:         -1,
		MinHeight:        0,
		MinHeightPercent: -1,
		MaxHeight:        -1,
		Overflow:         "visible",
		Color:            [3]float64{0, 0, 0},
		BGColor:          [4]float64{0, 0, 0, 0},
		FontSize:         12, // 16px at 96dpi
		FontWeight:       400,
		VerticalAlign:    "baseline",
		WhiteSpace:       "normal",
		TextDecoration:   "none",
		BorderCollapse:   "separate",
		BorderSpacing:    0,
		TableLayout:      "auto",
		GridColumnSpan:   1,
		GridRowSpan:      1,
		WritingMode:      "horizontal-tb",
		Orphans:          2,
		Widows:           2,
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

// resolveStyles walks the tree top-down, computing the used style of every
// element from the cascade (UA → sheets in order → inline) plus inheritance.
// @container rules are ignored on this first pass (no used sizes yet).
func resolveStyles(root *html.Node, sheets []*css.Stylesheet, media string, viewportW, viewportH float64) map[*html.Node]ResolvedStyle {
	return resolveStylesCtx(root, &styleContext{
		sheets: sheets, media: media, viewportW: viewportW, viewportH: viewportH,
	})
}

// resolveStylesWithContainers is the second style pass: @container rules are
// applied when their query matches the nearest eligible ancestor in containers.
func resolveStylesWithContainers(
	root *html.Node,
	sheets []*css.Stylesheet,
	media string,
	viewportW, viewportH float64,
	containers map[*html.Node]sizeContainer,
) map[*html.Node]ResolvedStyle {
	return resolveStylesCtx(root, &styleContext{
		sheets: sheets, media: media, viewportW: viewportW, viewportH: viewportH,
		containers: containers,
	})
}

func resolveStylesCtx(root *html.Node, ctx *styleContext) map[*html.Node]ResolvedStyle {
	out := map[*html.Node]ResolvedStyle{}
	var walk func(n *html.Node, parent *ResolvedStyle)
	walk = func(n *html.Node, parent *ResolvedStyle) {
		var st ResolvedStyle
		if n.Type == html.ElementNode {
			raw := cascadeRaw(ctx, n)
			st = initialStyle()
			if parent != nil {
				inheritProps(&st, *parent, raw)
			}
			applyFontProps(&st, raw, parent)
			applyRestProps(&st, raw, ctx, parent)
			// Chrome print keeps body links underlined even when wiki print CSS
			// sets text-decoration:inherit (!important) → none. Explicit none
			// (e.g. .IPA a) stays none. color:inherit → body black makes links
			// invisible as links; use Vector progressive blue for discoverability.
			if n.Name == "a" && strings.TrimSpace(n.Attribute("href")) != "" {
				if v, ok := raw["text-decoration"]; ok && strings.EqualFold(strings.TrimSpace(v), "inherit") {
					st.TextDecoration = "underline"
				}
				if v, ok := raw["color"]; ok && strings.EqualFold(strings.TrimSpace(v), "inherit") {
					st.Color = [3]float64{0x06 / 255.0, 0x45 / 255.0, 0xad / 255.0}
				}
			}
			// CSS2.1 §9.7: float ≠ none blockifies table-internal / inline
			// displays before layout (table/flex/grid stay). Floated <table>
			// keeps display:table so fixture-29 wrapper packing still works.
			if st.Float != "none" {
				st.Display = blockifyDisplayForFloat(st.Display)
			}
		} else if n.Type == html.TextNode {
			st = initialStyle()
			if parent != nil {
				inheritProps(&st, *parent, nil)
			}
		}
		out[n] = st
		for _, c := range n.Children {
			walk(c, &st)
		}
	}
	walk(root, nil)
	return out
}

// blockifyDisplayForFloat maps specified display to the used value when
// float is left|right (CSS2.1 §9.7). table stays table (floated table
// wrapper); table-cell/row/… and inlines become block.
func blockifyDisplayForFloat(d string) string {
	switch d {
	case "inline", "inline-block", "inline-table", "inline-flex", "inline-grid",
		"run-in", "table-row-group", "table-header-group", "table-footer-group",
		"table-row", "table-cell", "table-caption", "table-column",
		"table-column-group", "list-item":
		return "block"
	default:
		return d
	}
}

// inheritProps copies inheritable properties from the parent, unless the
// element declares its own value (present in raw).
func inheritProps(st *ResolvedStyle, parent ResolvedStyle, raw map[string]string) {
	set := func(prop string) bool {
		if raw == nil {
			return false
		}
		_, ok := raw[prop]
		return ok
	}
	if !set("color") {
		st.Color = parent.Color
	}
	if !set("font-family") {
		st.FontFamily = parent.FontFamily
	}
	if !set("font-size") {
		st.FontSize = parent.FontSize
	}
	if !set("font-weight") {
		st.FontWeight = parent.FontWeight
	}
	if !set("font-style") {
		st.FontItalic = parent.FontItalic
	}
	if !set("line-height") {
		st.LineHeight = parent.LineHeight
	}
	if !set("text-align") {
		st.TextAlign = parent.TextAlign
	}
	if !set("white-space") {
		st.WhiteSpace = parent.WhiteSpace
	}
	if !set("vertical-align") {
		st.VerticalAlign = parent.VerticalAlign
	}
	if !set("text-decoration") {
		st.TextDecoration = parent.TextDecoration
	}
	if !set("letter-spacing") {
		st.LetterSpacing = parent.LetterSpacing
	}
	if !set("border-collapse") {
		st.BorderCollapse = parent.BorderCollapse
	}
	if !set("border-spacing") {
		st.BorderSpacing = parent.BorderSpacing
	}
	if !set("orphans") {
		st.Orphans = parent.Orphans
	}
	if !set("widows") {
		st.Widows = parent.Widows
	}
}

// cascadeRaw returns the winning declaration per property for the element
// across UA sheet, author sheets and the inline style attribute.
func cascadeRaw(ctx *styleContext, n *html.Node) map[string]string {
	normal := map[string]string{}
	important := map[string]string{}
	var nSpec, nOrder = map[string][4]int{}, map[string]int{}
	var iSpec, iOrder = map[string][4]int{}, map[string]int{}

	apply := func(m map[string]string, spec map[string][4]int, ord map[string]int, prop, value string, a, b, c, order int) {
		prop = strings.ToLower(prop)
		if _, ok := m[prop]; !ok {
			m[prop] = value
			spec[prop] = [4]int{a, b, c, 0}
			ord[prop] = order
			return
		}
		cur := spec[prop]
		if a > cur[0] || (a == cur[0] && b > cur[1]) || (a == cur[0] && b == cur[1] && c > cur[2]) ||
			(a == cur[0] && b == cur[1] && c == cur[2] && order >= ord[prop]) {
			m[prop] = value
			spec[prop] = [4]int{a, b, c, 0}
			ord[prop] = order
		}
	}

	// UA sheet (lowest priority; specificity 0, order -1)
	for _, d := range uaRules(n.Name) {
		apply(normal, nSpec, nOrder, d.Prop, d.Value, 0, 0, 0, -1)
	}

	// author sheets in source order
	for _, sheet := range ctx.sheets {
		for _, r := range sheet.Rules {
			if !css.MediaMatches(r.Media, ctx.media, ctx.viewportW, ctx.viewportH) {
				continue
			}
			if r.Container != nil {
				if ctx.containers == nil {
					continue // first pass: used sizes unknown
				}
				info, ok := findSizeContainer(n, r.Container.Name, ctx.containers)
				if !ok || !r.Container.Cond.Matches(info.inlineSize, info.fontSize) {
					continue
				}
			}
			for _, sel := range r.Selectors {
				if !css.Match(sel, n) {
					continue
				}
				a, b, c := css.Specificity(sel)
				for _, d := range r.Decls {
					if d.Important {
						apply(important, iSpec, iOrder, d.Prop, d.Value, a, b, c, r.Order)
					} else {
						apply(normal, nSpec, nOrder, d.Prop, d.Value, a, b, c, r.Order)
					}
				}
			}
		}
	}

	// inline style attribute: outranks all normal declarations and all sheet
	// important declarations (spec 1<<30).
	for _, d := range css.ParseInline(n.Attribute("style")) {
		if d.Important {
			apply(important, iSpec, iOrder, d.Prop, d.Value, 1<<30, 0, 0, 1<<30)
		} else {
			apply(normal, nSpec, nOrder, d.Prop, d.Value, 1<<30, 0, 0, 1<<30)
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

// applyFontProps resolves font-size/family/weight/style/font first, using the
// parent's size for percentages and em.
func applyFontProps(st *ResolvedStyle, raw map[string]string, parent *ResolvedStyle) {
	parentSize := st.FontSize
	if parent != nil {
		parentSize = parent.FontSize
	}
	if v, ok := raw["font-size"]; ok {
		st.FontSize = fontSize(v, parentSize)
	}
	if v, ok := raw["font-family"]; ok {
		if fam := css.ParseFontFamily(v); len(fam) > 0 {
			st.FontFamily = fam
		}
	}
	if v, ok := raw["font-weight"]; ok {
		switch v {
		case "normal":
			st.FontWeight = 400
		case "bold":
			st.FontWeight = 700
		case "bolder":
			st.FontWeight += 100
		case "lighter":
			st.FontWeight -= 100
		default:
			if n, ok := css.ParseNumber(v); ok && n >= 100 && n <= 900 {
				st.FontWeight = int(n)
			}
		}
	}
	if v, ok := raw["font-style"]; ok {
		st.FontItalic = v == "italic" || v == "oblique"
	}
	if v, ok := raw["font"]; ok {
		parseFontShorthand(st, v)
	}
}

// applyRestProps resolves every non-font property once the font size is known.
// Properties are applied in a fixed order: the shorthand groups first, then
// every other property alphabetically. raw is a map, so iterating it directly
// would be nondeterministic and could let a shorthand (e.g. UA "margin")
// clobber a winning longhand (e.g. author "margin-bottom") depending on map
// iteration order.
func applyRestProps(st *ResolvedStyle, raw map[string]string, ctx *styleContext, parent *ResolvedStyle) {
	fs := st.FontSize
	// gap/flex/container applied before longhands so row-gap/column-gap,
	// flex-*, and container-type/name win over shorthands.
	shorthands := []string{"margin", "padding", "border", "border-width", "border-style", "border-color", "gap", "flex", "container"}
	isShorthand := map[string]bool{}
	for _, p := range shorthands {
		isShorthand[p] = true
	}
	var rest []string
	for prop := range raw {
		if !isShorthand[prop] {
			rest = append(rest, prop)
		}
	}
	sort.Strings(rest)
	props := append(append([]string{}, shorthands...), rest...)
	for _, prop := range props {
		value, ok := raw[prop]
		if !ok {
			continue
		}
		switch prop {
		case "display":
			switch value {
			case "block", "inline", "none", "list-item", "table", "table-row", "table-cell",
				"table-row-group", "table-header-group", "table-footer-group",
				"inline-block", "table-caption", "table-column", "table-column-group",
				"flex", "inline-flex", "grid", "inline-grid", "subgrid":
				st.Display = value
			}
		case "position":
			switch value {
			case "static", "relative", "absolute", "fixed", "sticky":
				st.Position = value
			}
		case "top":
			st.Top, st.TopAuto = marginLenAuto(value, fs, ctx.viewportH)
		case "right":
			st.Right, st.RightAuto = marginLenAuto(value, fs, ctx.viewportW)
		case "bottom":
			st.Bottom, st.BottomAuto = marginLenAuto(value, fs, ctx.viewportH)
		case "left":
			st.Left, st.LeftAuto = marginLenAuto(value, fs, ctx.viewportW)
		case "flex-direction":
			switch value {
			case "row", "column", "row-reverse", "column-reverse":
				st.FlexDirection = value
			}
		case "flex-wrap":
			if value == "nowrap" || value == "wrap" || value == "wrap-reverse" {
				st.FlexWrap = value
			}
		case "justify-content":
			switch value {
			case "flex-start", "flex-end", "center", "space-between", "space-around", "space-evenly", "start", "end":
				st.JustifyContent = value
			}
		case "align-items":
			switch value {
			case "stretch", "flex-start", "flex-end", "center", "start", "end":
				st.AlignItems = value
			}
		case "align-content":
			switch value {
			case "flex-start", "flex-end", "center", "space-between", "space-around", "space-evenly", "stretch", "start", "end":
				st.AlignContent = value
			}
		case "align-self":
			switch value {
			case "auto", "stretch", "flex-start", "flex-end", "center", "start", "end":
				st.AlignSelf = value
			}
		case "justify-items":
			switch value {
			case "stretch", "start", "end", "center", "flex-start", "flex-end":
				st.JustifyItems = value
			}
		case "justify-self":
			switch value {
			case "auto", "stretch", "start", "end", "center", "flex-start", "flex-end":
				st.JustifySelf = value
			}
		case "gap":
			if value == "normal" {
				st.Gap = 0
				st.RowGap = 0
				st.ColumnGap = 0
				st.ColumnGapNormal = true
			} else if v, ok := lengthBox(value, fs, ctx.viewportW, "none"); ok && v >= 0 {
				st.Gap = v
				st.RowGap = v
				st.ColumnGap = v
				st.ColumnGapNormal = false
			}
		case "row-gap":
			if v, ok := lengthBox(value, fs, ctx.viewportW, "none"); ok && v >= 0 {
				st.RowGap = v
				st.Gap = v
			}
		case "column-gap":
			if value == "normal" {
				st.ColumnGap = 0
				st.ColumnGapNormal = true
			} else if v, ok := lengthBox(value, fs, ctx.viewportW, "none"); ok && v >= 0 {
				st.ColumnGap = v
				st.ColumnGapNormal = false
			}
		case "column-count":
			if value == "auto" {
				st.ColumnCount = 0
			} else if n, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && n >= 1 {
				st.ColumnCount = n
			}
		case "column-width":
			if value == "auto" {
				st.ColumnWidth = -1
			} else if v, ok := lengthBox(value, fs, ctx.viewportW, "auto"); ok && v >= 0 {
				st.ColumnWidth = v
			}
		case "columns":
			parseColumnsShorthand(st, value, fs, ctx.viewportW)
		case "column-span":
			switch value {
			case "none", "all":
				st.ColumnSpan = value
			}
		case "column-fill":
			switch value {
			case "balance", "auto":
				st.ColumnFill = value
			}
		case "flex":
			parseFlexShorthand(st, value, fs, ctx.viewportW)
		case "flex-grow":
			if v, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil && v >= 0 {
				st.FlexGrow = v
			}
		case "flex-shrink":
			if v, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil && v >= 0 {
				st.FlexShrink = v
			}
		case "flex-basis":
			if value == "auto" {
				st.FlexBasis = -1
				st.FlexBasisPercent = -1
			} else if v, unit, ok := css.ParseLength(value); ok && unit == "%" {
				st.FlexBasisPercent = v
				st.FlexBasis = -1
			} else if v, ok := lengthBox(value, fs, ctx.viewportW, "auto"); ok {
				st.FlexBasis = v
				st.FlexBasisPercent = -1
			}
		case "order":
			if v, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
				st.FlexOrder = v
			}
		case "z-index":
			if value == "auto" {
				st.ZIndexSet = false
				st.ZIndex = 0
			} else if v, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
				st.ZIndex = v
				st.ZIndexSet = true
			}
		case "writing-mode":
			switch value {
			case "horizontal-tb", "vertical-rl", "vertical-lr":
				st.WritingMode = value
			}
		case "grid-template-columns":
			st.GridTemplateColumns = value
		case "grid-template-rows":
			st.GridTemplateRows = value
		case "grid-template-areas":
			st.GridTemplateAreas = value
		case "grid-area":
			parseGridArea(st, value)
		case "grid-auto-flow":
			st.GridAutoFlow = parseGridAutoFlowValue(value)
		case "grid-column", "grid-column-end":
			parseGridColumn(st, value)
		case "grid-column-start":
			if v, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && v > 0 {
				st.GridColumnStart = v
			}
		case "grid-row", "grid-row-end":
			parseGridRow(st, value)
		case "grid-row-start":
			if v, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && v > 0 {
				st.GridRowStart = v
			}
		case "float":
			switch value {
			case "left", "right", "none":
				st.Float = value
			}
		case "clear":
			switch value {
			case "left", "right", "both", "none":
				st.Clear = value
			}
		case "box-sizing":
			switch value {
			case "content-box", "border-box":
				st.BoxSizing = value
			}
		case "width":
			if value == "auto" {
				st.Width = -1
				st.WidthPercent = -1
			} else if v, unit, ok := css.ParseLength(value); ok && unit == "%" {
				// Resolve % against the layout containing block (availW), not
				// the viewport — nested width:100% must fill the parent cell.
				st.WidthPercent = v
				st.Width = -1
			} else if v, ok := lengthBox(value, fs, ctx.viewportW, "auto"); ok {
				st.Width = v
				st.WidthPercent = -1
			}
		case "height":
			if value == "auto" {
				st.Height = -1
				st.HeightPercent = -1
			} else if v, unit, ok := css.ParseLength(value); ok && unit == "%" {
				// Defer % height to layout; indefinite containing block → auto
				// (cyclic percentage honesty for flex/grid children).
				st.HeightPercent = v
				st.Height = -1
			} else if v, ok := lengthBox(value, fs, ctx.viewportH, "auto"); ok {
				st.Height = v
				st.HeightPercent = -1
			}
		case "min-width":
			if value == "auto" || value == "none" {
				st.MinWidth = 0
				st.MinWidthPercent = -1
			} else if v, unit, ok := css.ParseLength(value); ok && unit == "%" {
				st.MinWidthPercent = v
				st.MinWidth = 0
			} else if v, ok := lengthBox(value, fs, ctx.viewportW, "none"); ok {
				st.MinWidth = v
				st.MinWidthPercent = -1
			}
		case "max-width":
			if v, ok := lengthBox(value, fs, ctx.viewportW, "none"); ok {
				st.MaxWidth = v
			}
		case "min-height":
			if value == "auto" || value == "none" {
				st.MinHeight = 0
				st.MinHeightPercent = -1
			} else if v, unit, ok := css.ParseLength(value); ok && unit == "%" {
				st.MinHeightPercent = v
				st.MinHeight = 0
			} else if v, ok := lengthBox(value, fs, ctx.viewportH, "none"); ok {
				st.MinHeight = v
				st.MinHeightPercent = -1
			}
		case "max-height":
			if v, ok := lengthBox(value, fs, ctx.viewportH, "none"); ok {
				st.MaxHeight = v
			}
		case "overflow":
			if ov, ok := parseOverflowKeyword(value); ok {
				st.Overflow = ov
			}
		case "overflow-x", "overflow-y":
			// Either axis non-visible creates a sticky scrollport (CSS Position 3).
			if ov, ok := parseOverflowKeyword(value); ok && ov != "visible" {
				st.Overflow = ov
			}
		case "margin":
			setFourMargin(st, value, fs, ctx.viewportW)
		case "margin-top":
			st.MarginTop = marginLen(value, fs, ctx.viewportW)
		case "margin-right":
			st.MarginRight, st.MarginRightAuto = marginLenAuto(value, fs, ctx.viewportW)
		case "margin-bottom":
			st.MarginBottom = marginLen(value, fs, ctx.viewportW)
		case "margin-left":
			st.MarginLeft, st.MarginLeftAuto = marginLenAuto(value, fs, ctx.viewportW)
		case "padding":
			setFour(st, value, &st.PaddingTop, &st.PaddingRight, &st.PaddingBottom, &st.PaddingLeft, fs, ctx.viewportW)
		case "padding-top":
			st.PaddingTop = marginLen(value, fs, ctx.viewportW)
		case "padding-right":
			st.PaddingRight = marginLen(value, fs, ctx.viewportW)
		case "padding-bottom":
			st.PaddingBottom = marginLen(value, fs, ctx.viewportW)
		case "padding-left":
			st.PaddingLeft = marginLen(value, fs, ctx.viewportW)
		case "border":
			if b, ok := parseBorder(value, fs); ok {
				st.BorderTop, st.BorderRight, st.BorderBottom, st.BorderLeft = b, b, b, b
			}
		case "border-top":
			if b, ok := parseBorder(value, fs); ok {
				st.BorderTop = b
			}
		case "border-right":
			if b, ok := parseBorder(value, fs); ok {
				st.BorderRight = b
			}
		case "border-bottom":
			if b, ok := parseBorder(value, fs); ok {
				st.BorderBottom = b
			}
		case "border-left":
			if b, ok := parseBorder(value, fs); ok {
				st.BorderLeft = b
			}
		case "border-width":
			w := borderWidth(value, fs)
			st.BorderTop.Width, st.BorderRight.Width, st.BorderBottom.Width, st.BorderLeft.Width = w, w, w, w
		case "border-top-width":
			st.BorderTop.Width = borderWidth(value, fs)
		case "border-right-width":
			st.BorderRight.Width = borderWidth(value, fs)
		case "border-bottom-width":
			st.BorderBottom.Width = borderWidth(value, fs)
		case "border-left-width":
			st.BorderLeft.Width = borderWidth(value, fs)
		case "border-style":
			s := value
			if s != "solid" && s != "dashed" && s != "dotted" {
				s = "none"
			}
			st.BorderTop.Style, st.BorderRight.Style, st.BorderBottom.Style, st.BorderLeft.Style = s, s, s, s
		case "border-color":
			if r, g, b, _, ok := css.ParseColor(value); ok {
				c := [3]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255}
				st.BorderTop.Color, st.BorderRight.Color, st.BorderBottom.Color, st.BorderLeft.Color = c, c, c, c
			}
		case "color":
			if value == "inherit" {
				if parent != nil {
					st.Color = parent.Color
				}
			} else if r, g, b, _, ok := css.ParseColor(value); ok {
				st.Color = [3]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255}
			}
		case "background-color":
			if r, g, b, a, ok := css.ParseColor(value); ok {
				st.BGColor = [4]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255, a}
			}
		case "background":
			// Shorthand: take the first parseable color token (ignore images/repeat).
			for _, tok := range strings.Fields(value) {
				if r, g, b, a, ok := css.ParseColor(tok); ok {
					st.BGColor = [4]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255, a}
					break
				}
			}
		case "line-height":
			st.LineHeight = lineHeight(value, st.FontSize)
		case "text-align":
			switch value {
			case "left", "center", "justify":
				st.TextAlign = value
			case "right", "end":
				st.TextAlign = "right"
			case "start":
				st.TextAlign = "left"
			}
		case "vertical-align":
			switch value {
			case "baseline", "top", "middle", "bottom":
				st.VerticalAlign = value
			}
		case "white-space":
			switch value {
			case "normal", "nowrap":
				st.WhiteSpace = value
			case "pre", "pre-wrap", "pre-line":
				st.WhiteSpace = "pre"
			}
		case "text-decoration":
			switch value {
			case "underline":
				st.TextDecoration = "underline"
			case "line-through":
				st.TextDecoration = "line-through"
			case "none":
				st.TextDecoration = "none"
			case "inherit":
				if parent != nil {
					st.TextDecoration = parent.TextDecoration
				}
			}
		case "letter-spacing":
			st.LetterSpacing = marginLen(value, fs, ctx.viewportW)
		case "text-indent":
			st.TextIndent = marginLen(value, fs, ctx.viewportW)
		case "border-collapse":
			if value == "collapse" || value == "separate" {
				st.BorderCollapse = value
			}
		case "border-spacing":
			st.BorderSpacing = marginLen(value, fs, ctx.viewportW)
		case "table-layout":
			if value == "fixed" || value == "auto" {
				st.TableLayout = value
			}
		case "page-break-before", "break-before":
			// Lite: column | avoid-column alias to always | avoid (new multicol
			// line via page break). Spec column breaks beyond that are deferred.
			switch value {
			case "always", "column":
				st.PageBreakBefore = "always"
			case "avoid", "avoid-column":
				st.PageBreakBefore = "avoid"
			}
		case "page-break-after", "break-after":
			switch value {
			case "always", "column":
				st.PageBreakAfter = "always"
			case "avoid", "avoid-column":
				st.PageBreakAfter = "avoid"
			}
		case "page-break-inside", "break-inside":
			switch value {
			case "always", "column":
				st.PageBreakInside = "always"
			case "avoid", "avoid-column":
				st.PageBreakInside = "avoid"
			}
		case "orphans":
			if n, ok := parseOrphansWidowsInt(value); ok {
				st.Orphans = n
			}
		case "widows":
			if n, ok := parseOrphansWidowsInt(value); ok {
				st.Widows = n
			}
		case "container-type":
			switch strings.ToLower(value) {
			case "normal", "size", "inline-size":
				st.ContainerType = strings.ToLower(value)
			}
		case "container-name":
			st.ContainerName = css.ParseContainerNameValue(value)
		case "container":
			name, ctype := css.ParseContainerShorthand(value)
			st.ContainerName = name
			if ctype != "" {
				st.ContainerType = ctype
			}
		case "transform":
			// Animations/transitions ignored: cascaded static value only.
			if m, has, ok := parseTransformList(value, fs); ok {
				st.Transform = m
				st.HasTransform = has
			}
		case "transform-origin":
			if spec, ok := parseTransformOrigin(value, fs); ok {
				st.TransformOrigin = spec
			}
		case "opacity":
			if v, ok := parseOpacityValue(value); ok {
				st.Opacity = v
			}
		case "filter":
			// opacity() via ExtGState; blur/drop-shadow permanent non-goals.
			if v, ok := parseFilterOpacity(value); ok {
				st.Opacity *= v
			}
		case "animation", "animation-name", "animation-duration",
			"animation-timing-function", "animation-delay", "animation-iteration-count",
			"animation-direction", "animation-fill-mode", "animation-play-state",
			"transition", "transition-property", "transition-duration",
			"transition-timing-function", "transition-delay":
			// Parse-ignore: no timeline; static cascaded transform/opacity only.
		}
	}
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
func parseColumnsShorthand(st *ResolvedStyle, value string, fs, viewportW float64) {
	st.ColumnCount = 0
	st.ColumnWidth = -1
	for _, tok := range strings.Fields(value) {
		tok = strings.TrimSpace(tok)
		if tok == "" || tok == "auto" {
			continue
		}
		if n, err := strconv.Atoi(tok); err == nil && n >= 1 {
			st.ColumnCount = n
			continue
		}
		if v, ok := lengthBox(tok, fs, viewportW, "auto"); ok && v >= 0 {
			st.ColumnWidth = v
		}
	}
}

// isMulticol reports whether st establishes a multi-column container
// (column-count or column-width is not auto).
func isMulticol(st ResolvedStyle) bool {
	return st.ColumnCount > 0 || st.ColumnWidth >= 0
}

// parseFontShorthand handles "font: italic bold 12px/1.4 Arial, sans-serif".
func parseFontShorthand(st *ResolvedStyle, value string) {
	parts := strings.Fields(value)
	for i, p := range parts {
		if p == "italic" || p == "oblique" {
			st.FontItalic = true
			continue
		}
		if p == "bold" {
			st.FontWeight = 700
			continue
		}
		if n, ok := css.ParseNumber(p); ok && n >= 100 && n <= 900 {
			st.FontWeight = int(n)
			continue
		}
		// first size token
		rest := p
		if j := strings.IndexByte(rest, '/'); j >= 0 {
			st.LineHeight = lineHeight(rest[j+1:], st.FontSize)
			rest = rest[:j]
		}
		st.FontSize = fontSize(rest, st.FontSize)
		if i+1 < len(parts) {
			if fam := css.ParseFontFamily(strings.Join(parts[i+1:], " ")); len(fam) > 0 {
				st.FontFamily = fam
			}
		}
		return
	}
}

// parseFlexShorthand handles flex: none | auto | <grow> | <grow> <shrink> | <grow> <shrink> <basis>.
func parseFlexShorthand(st *ResolvedStyle, value string, fs, pctBase float64) {
	value = strings.TrimSpace(value)
	switch value {
	case "none":
		st.FlexGrow, st.FlexShrink = 0, 0
		st.FlexBasis, st.FlexBasisPercent = -1, -1
		return
	case "auto":
		st.FlexGrow, st.FlexShrink = 1, 1
		st.FlexBasis, st.FlexBasisPercent = -1, -1
		return
	}
	parts := strings.Fields(value)
	if len(parts) == 0 {
		return
	}
	isBasis := func(tok string) bool {
		if tok == "auto" || tok == "content" {
			return true
		}
		if _, _, ok := css.ParseLength(tok); ok {
			return true
		}
		return false
	}
	setBasis := func(tok string) {
		if tok == "auto" || tok == "content" {
			st.FlexBasis = -1
			st.FlexBasisPercent = -1
			return
		}
		if v, unit, ok := css.ParseLength(tok); ok && unit == "%" {
			st.FlexBasisPercent = v
			st.FlexBasis = -1
			return
		}
		if v, ok := lengthBox(tok, fs, pctBase, "auto"); ok {
			st.FlexBasis = v
			st.FlexBasisPercent = -1
		}
	}
	switch len(parts) {
	case 1:
		if g, err := strconv.ParseFloat(parts[0], 64); err == nil {
			// flex: <number> → grow <number>, shrink 1, basis 0%
			st.FlexGrow = g
			st.FlexShrink = 1
			st.FlexBasis = -1
			st.FlexBasisPercent = 0
			return
		}
		if isBasis(parts[0]) {
			st.FlexGrow, st.FlexShrink = 1, 1
			setBasis(parts[0])
		}
	case 2:
		g, errG := strconv.ParseFloat(parts[0], 64)
		if errG != nil {
			return
		}
		st.FlexGrow = g
		if sh, err := strconv.ParseFloat(parts[1], 64); err == nil {
			st.FlexShrink = sh
			st.FlexBasis = -1
			st.FlexBasisPercent = 0
			return
		}
		st.FlexShrink = 1
		if isBasis(parts[1]) {
			setBasis(parts[1])
		}
	default:
		g, errG := strconv.ParseFloat(parts[0], 64)
		sh, errS := strconv.ParseFloat(parts[1], 64)
		if errG != nil || errS != nil {
			return
		}
		st.FlexGrow, st.FlexShrink = g, sh
		setBasis(parts[2])
	}
}

// setFourMargin applies a margin shorthand and tracks horizontal auto.
func setFourMargin(st *ResolvedStyle, value string, fs, ctxW float64) {
	v := strings.Fields(value)
	st.MarginLeftAuto, st.MarginRightAuto = false, false
	switch len(v) {
	case 0:
		return
	case 1:
		st.MarginTop = marginLen(v[0], fs, ctxW)
		st.MarginRight, st.MarginRightAuto = marginLenAuto(v[0], fs, ctxW)
		st.MarginBottom = marginLen(v[0], fs, ctxW)
		st.MarginLeft, st.MarginLeftAuto = marginLenAuto(v[0], fs, ctxW)
	case 2:
		st.MarginTop = marginLen(v[0], fs, ctxW)
		st.MarginRight, st.MarginRightAuto = marginLenAuto(v[1], fs, ctxW)
		st.MarginBottom = marginLen(v[0], fs, ctxW)
		st.MarginLeft, st.MarginLeftAuto = marginLenAuto(v[1], fs, ctxW)
	case 3:
		st.MarginTop = marginLen(v[0], fs, ctxW)
		st.MarginRight, st.MarginRightAuto = marginLenAuto(v[1], fs, ctxW)
		st.MarginBottom = marginLen(v[2], fs, ctxW)
		st.MarginLeft, st.MarginLeftAuto = marginLenAuto(v[1], fs, ctxW)
	default:
		st.MarginTop = marginLen(v[0], fs, ctxW)
		st.MarginRight, st.MarginRightAuto = marginLenAuto(v[1], fs, ctxW)
		st.MarginBottom = marginLen(v[2], fs, ctxW)
		st.MarginLeft, st.MarginLeftAuto = marginLenAuto(v[3], fs, ctxW)
	}
}

func setFour(st *ResolvedStyle, value string, top, right, bottom, left *float64, fs, ctxW float64) {
	v := strings.Fields(value)
	if len(v) == 0 {
		return
	}
	if len(v) == 1 {
		*top = marginLen(v[0], fs, ctxW)
		*right, *bottom, *left = *top, *top, *top
		return
	}
	if len(v) == 2 {
		*top = marginLen(v[0], fs, ctxW)
		*right = marginLen(v[1], fs, ctxW)
		*bottom, *left = *top, *right
		return
	}
	if len(v) == 3 {
		*top = marginLen(v[0], fs, ctxW)
		*right = marginLen(v[1], fs, ctxW)
		*bottom = marginLen(v[2], fs, ctxW)
		*left = *right
		return
	}
	*top = marginLen(v[0], fs, ctxW)
	*right = marginLen(v[1], fs, ctxW)
	*bottom = marginLen(v[2], fs, ctxW)
	*left = marginLen(v[3], fs, ctxW)
}

func parseBorder(value string, fs float64) (border, bool) {
	var b border
	for _, f := range strings.Fields(value) {
		switch f {
		case "solid", "dashed", "dotted":
			b.Style = f
		case "none", "hidden":
			b.Style = "none"
		default:
			if r, g, bb, _, ok := css.ParseColor(f); ok {
				b.Color = [3]float64{float64(r) / 255, float64(g) / 255, float64(bb) / 255}
			} else if v, _, ok := css.ParseLength(f); ok {
				b.Width = v
			}
		}
	}
	if b.Style == "" {
		b.Style = "solid"
	}
	if b.Width == 0 {
		b.Width = 1
	}
	return b, b.Style != "none"
}

func borderWidth(value string, fs float64) float64 {
	switch value {
	case "thin":
		return pxToPt(1)
	case "medium":
		return pxToPt(3)
	case "thick":
		return pxToPt(5)
	}
	if v, _, ok := css.ParseLength(value); ok {
		return v
	}
	return 0
}

func fontSize(value string, parent float64) float64 {
	switch value {
	case "xx-small":
		return pxToPt(9)
	case "x-small":
		return pxToPt(10)
	case "small":
		return pxToPt(13)
	case "medium":
		return pxToPt(16)
	case "large":
		return pxToPt(18)
	case "x-large":
		return pxToPt(24)
	case "xx-large":
		return pxToPt(32)
	case "smaller":
		return parent * 0.833
	case "larger":
		return parent * 1.2
	}
	if v, unit, ok := css.ParseLength(value); ok {
		switch unit {
		case "%":
			return parent * v / 100
		case "em":
			return parent * v
		case "pt":
			return v
		case "px":
			return pxToPt(v)
		case "rem":
			return pxToPt(16) * v
		case "in":
			return v * 72
		case "cm":
			return v * 72 / 2.54
		case "mm":
			return v * 72 / 25.4
		case "pc":
			return v * 12
		}
	}
	return parent
}

func lineHeight(value string, fs float64) float64 {
	if value == "normal" {
		return 0
	}
	if v, ok := css.ParseNumber(value); ok {
		return v * fs
	}
	if v, unit, ok := css.ParseLength(value); ok {
		switch unit {
		case "%":
			return fs * v / 100
		case "em":
			return fs * v
		case "pt":
			return v
		case "px":
			return pxToPt(v)
		default:
			return v * 72 / 25.4
		}
	}
	return 0
}

// parseOverflowKeyword accepts CSS overflow keywords used for sticky scrollport
// detection. clip is treated like hidden (scroll container, no user scroll).
func parseOverflowKeyword(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "visible", "hidden", "scroll", "auto", "clip":
		return strings.ToLower(strings.TrimSpace(value)), true
	}
	return "", false
}

// overflowCreatesStickyScrollport reports whether overflow establishes a sticky
// scrollport (CSS Position 3 / Overflow 3). PDF has no user scroll, so sticky
// inside these boxes clamps at scroll offset 0 against the box edges.
func overflowCreatesStickyScrollport(overflow string) bool {
	switch overflow {
	case "auto", "scroll", "hidden", "clip":
		return true
	}
	return false
}

// lengthBox parses a length for box-sizing properties. "auto" maps to -1,
// "none" to -1 as well (max-*). Percentages resolve against the containing
// block dimension (approximated by viewport at this phase).
func lengthBox(value string, fs, containing float64, autoValue string) (float64, bool) {
	if value == autoValue || value == "inherit" || value == "initial" {
		return -1, true
	}
	v, unit, ok := css.ParseLength(value)
	if !ok {
		return 0, false
	}
	switch unit {
	case "px":
		return pxToPt(v), true
	case "pt":
		return v, true
	case "in":
		return v * 72, true
	case "cm":
		return v * 72 / 2.54, true
	case "mm":
		return v * 72 / 25.4, true
	case "pc":
		return v * 12, true
	case "em":
		return fs * v, true
	case "rem":
		return pxToPt(16) * v, true
	case "%":
		return containing * v / 100, true
	case "vw":
		return containing * v / 100, true
	case "vh":
		return containing * v / 100, true
	}
	return 0, false
}

// marginLenAuto parses a horizontal margin; auto yields (0, true).
func marginLenAuto(value string, fs, ctxW float64) (float64, bool) {
	if value == "auto" {
		return 0, true
	}
	return marginLen(value, fs, ctxW), false
}

// marginLen parses a margin/padding/letter-spacing length in points; 0 when
// unparseable. Percentages resolve against the containing width.
func marginLen(value string, fs, ctxW float64) float64 {
	if value == "auto" || value == "inherit" || value == "initial" {
		return 0
	}
	v, unit, ok := css.ParseLength(value)
	if !ok {
		return 0
	}
	switch unit {
	case "px":
		return pxToPt(v)
	case "pt":
		return v
	case "in":
		return v * 72
	case "cm":
		return v * 72 / 2.54
	case "mm":
		return v * 72 / 25.4
	case "pc":
		return v * 12
	case "em":
		return fs * v
	case "rem":
		return pxToPt(16) * v
	case "%":
		return ctxW * v / 100
	}
	return 0
}

func pxToPt(px float64) float64 { return px * 0.75 }

// parseGridAutoFlowValue normalizes grid-auto-flow to one of:
// "row", "column", "dense", "row dense", "column dense".
func parseGridAutoFlowValue(value string) string {
	toks := strings.Fields(strings.ToLower(strings.TrimSpace(value)))
	row, col, dense := false, false, false
	for _, t := range toks {
		switch t {
		case "row":
			row = true
		case "column":
			col = true
		case "dense":
			dense = true
		}
	}
	switch {
	case col && dense:
		return "column dense"
	case row && dense:
		return "row dense"
	case dense:
		return "dense"
	case col:
		return "column"
	default:
		return "row"
	}
}

// parseGridArea handles grid-area: <custom-ident> or the lite line form
// row-start / column-start / row-end / column-end (and 1–2 slash forms).
func parseGridArea(st *ResolvedStyle, value string) {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "auto") {
		st.GridArea = ""
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
			parseGridRow(st, tok)
			st.GridArea = ""
			return
		}
		if strings.HasPrefix(tok, "span ") {
			parseGridRow(st, tok)
			st.GridArea = ""
			return
		}
		// Named area.
		st.GridArea = tok
		return
	}
	st.GridArea = ""
	switch len(parts) {
	case 2:
		// CSS: row-start / column-start (omitted ends copy starts → span 1).
		parseGridRow(st, parts[0])
		parseGridColumn(st, parts[1])
	case 3:
		// row-start / column-start / row-end
		parseGridRow(st, parts[0])
		parseGridColumn(st, parts[1])
		applyGridLineEnd(st, true, parts[2])
	default:
		// row-start / column-start / row-end / column-end
		parseGridRow(st, parts[0])
		parseGridColumn(st, parts[1])
		applyGridLineEnd(st, true, parts[2])
		applyGridLineEnd(st, false, parts[3])
	}
}

// applyGridLineEnd sets span from an end line or "span N" on row (isRow) or column.
func applyGridLineEnd(st *ResolvedStyle, isRow bool, end string) {
	end = strings.TrimSpace(end)
	if strings.HasPrefix(end, "span ") {
		n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(end, "span ")))
		if err != nil || n < 1 {
			return
		}
		if isRow {
			st.GridRowSpan = n
		} else {
			st.GridColumnSpan = n
		}
		return
	}
	v, err := strconv.Atoi(end)
	if err != nil {
		return
	}
	if isRow {
		if st.GridRowStart > 0 {
			sp := v - st.GridRowStart
			if sp < 1 {
				sp = 1
			}
			st.GridRowSpan = sp
		}
	} else if st.GridColumnStart > 0 {
		sp := v - st.GridColumnStart
		if sp < 1 {
			sp = 1
		}
		st.GridColumnSpan = sp
	}
}

func parseGridColumn(st *ResolvedStyle, value string) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "span ") {
		n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(value, "span ")))
		if err == nil && n > 0 {
			st.GridColumnSpan = n
		}
		return
	}
	// "1 / 3" or "1 / span 2"
	parts := strings.Split(value, "/")
	if len(parts) == 1 {
		if v, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil && v > 0 {
			st.GridColumnStart = v
			st.GridColumnSpan = 1
		}
		return
	}
	if v, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil && v > 0 {
		st.GridColumnStart = v
	}
	end := strings.TrimSpace(parts[1])
	if strings.HasPrefix(end, "span ") {
		n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(end, "span ")))
		if err == nil && n > 0 {
			st.GridColumnSpan = n
		}
		return
	}
	if v, err := strconv.Atoi(end); err == nil && st.GridColumnStart > 0 {
		sp := v - st.GridColumnStart
		if sp < 1 {
			sp = 1
		}
		st.GridColumnSpan = sp
	}
}

func parseGridRow(st *ResolvedStyle, value string) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "span ") {
		n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(value, "span ")))
		if err == nil && n > 0 {
			st.GridRowSpan = n
		}
		return
	}
	// "1 / 3" or "1 / span 2"
	parts := strings.Split(value, "/")
	if len(parts) == 1 {
		if v, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil && v > 0 {
			st.GridRowStart = v
			st.GridRowSpan = 1
		}
		return
	}
	if v, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil && v > 0 {
		st.GridRowStart = v
	}
	end := strings.TrimSpace(parts[1])
	if strings.HasPrefix(end, "span ") {
		n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(end, "span ")))
		if err == nil && n > 0 {
			st.GridRowSpan = n
		}
		return
	}
	if v, err := strconv.Atoi(end); err == nil && st.GridRowStart > 0 {
		sp := v - st.GridRowStart
		if sp < 1 {
			sp = 1
		}
		st.GridRowSpan = sp
	}
}

// uaRules returns the user-agent declarations for an element name.
func uaRules(name string) []css.Declaration {
	switch name {
	case "html":
		return []css.Declaration{{Prop: "display", Value: "block"}}
	case "body":
		return []css.Declaration{
			{Prop: "display", Value: "block"},
			{Prop: "margin", Value: "8px"},
		}
	case "div", "section", "article", "header", "footer", "main", "aside",
		"nav", "form", "fieldset", "figure", "figcaption", "blockquote",
		"address", "dl", "dd", "menu", "details", "summary":
		return []css.Declaration{{Prop: "display", Value: "block"}}
	case "p":
		return []css.Declaration{
			{Prop: "display", Value: "block"},
			{Prop: "margin", Value: "1em 0"},
		}
	case "pre":
		// Match browser UA: preserve newlines/spaces; monospace is a
		// soft preference (we fall back to Liberation Sans metrics).
		return []css.Declaration{
			{Prop: "display", Value: "block"},
			{Prop: "margin", Value: "1em 0"},
			{Prop: "white-space", Value: "pre"},
			{Prop: "font-family", Value: "monospace"},
		}
	case "code", "kbd", "samp":
		return []css.Declaration{{Prop: "font-family", Value: "monospace"}}
	case "h1":
		return []css.Declaration{{Prop: "display", Value: "block"}, {Prop: "font-size", Value: "2em"}, {Prop: "margin", Value: "0.67em 0"}, {Prop: "font-weight", Value: "bold"}}
	case "h2":
		return []css.Declaration{{Prop: "display", Value: "block"}, {Prop: "font-size", Value: "1.5em"}, {Prop: "margin", Value: "0.83em 0"}, {Prop: "font-weight", Value: "bold"}}
	case "h3":
		return []css.Declaration{{Prop: "display", Value: "block"}, {Prop: "font-size", Value: "1.17em"}, {Prop: "margin", Value: "1em 0"}, {Prop: "font-weight", Value: "bold"}}
	case "h4", "h5", "h6":
		return []css.Declaration{{Prop: "display", Value: "block"}, {Prop: "font-weight", Value: "bold"}, {Prop: "font-size", Value: "1em"}, {Prop: "margin", Value: "1.33em 0"}}
	case "ul", "ol":
		return []css.Declaration{{Prop: "display", Value: "block"}, {Prop: "margin", Value: "1em 0"}, {Prop: "padding-left", Value: "40px"}}
	case "li":
		return []css.Declaration{{Prop: "display", Value: "list-item"}}
	case "table":
		return []css.Declaration{{Prop: "display", Value: "table"}, {Prop: "border-spacing", Value: "2px"}}
	case "thead":
		return []css.Declaration{{Prop: "display", Value: "table-header-group"}}
	case "tfoot":
		return []css.Declaration{{Prop: "display", Value: "table-footer-group"}}
	case "tbody":
		return []css.Declaration{{Prop: "display", Value: "table-row-group"}}
	case "tr":
		return []css.Declaration{{Prop: "display", Value: "table-row"}}
	case "td":
		return []css.Declaration{{Prop: "display", Value: "table-cell"}, {Prop: "padding", Value: "1px"}}
	case "th":
		return []css.Declaration{{Prop: "display", Value: "table-cell"}, {Prop: "padding", Value: "1px"}, {Prop: "text-align", Value: "center"}, {Prop: "font-weight", Value: "bold"}}
	case "img":
		return []css.Declaration{{Prop: "display", Value: "inline-block"}}
	case "hr":
		return []css.Declaration{{Prop: "display", Value: "block"}, {Prop: "border", Value: "1px inset"}, {Prop: "margin", Value: "0.5em auto"}}
	case "a":
		return []css.Declaration{{Prop: "color", Value: "#0000ee"}, {Prop: "text-decoration", Value: "underline"}}
	case "b", "strong":
		return []css.Declaration{{Prop: "font-weight", Value: "bold"}}
	case "i", "em", "cite", "dfn", "var":
		return []css.Declaration{{Prop: "font-style", Value: "italic"}}
	case "u":
		return []css.Declaration{{Prop: "text-decoration", Value: "underline"}}
	case "s", "strike", "del":
		return []css.Declaration{{Prop: "text-decoration", Value: "line-through"}}
	case "small":
		return []css.Declaration{{Prop: "font-size", Value: "smaller"}}
	case "big":
		return []css.Declaration{{Prop: "font-size", Value: "larger"}}
	case "center":
		return []css.Declaration{{Prop: "text-align", Value: "center"}}
	case "title", "style", "script", "meta", "link", "head":
		return []css.Declaration{{Prop: "display", Value: "none"}}
	case "textarea":
		return []css.Declaration{{Prop: "white-space", Value: "pre"}, {Prop: "font-family", Value: "monospace"}}
	case "br":
		return []css.Declaration{{Prop: "display", Value: "block"}}
	}
	return nil
}
