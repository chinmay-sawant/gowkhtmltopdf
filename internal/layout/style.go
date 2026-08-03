package layout

import (
	"sort"
	"strings"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

// ResolvedStyle is the used style of one element: values the layout engine
// consumes, in points (or unitless where noted). Only the phase-04 subset is
// modeled; everything else keeps its initial value.
type ResolvedStyle struct {
	Display         string
	Position        string  // "static" | "relative"
	Float           string  // "none" | "left" | "right"
	Clear           string  // "none" | "left" | "right" | "both"
	Width           float64 // -1 = auto; absolute length in pt when WidthPercent < 0
	WidthPercent    float64 // >=0 means width is that % of the containing block at layout time
	Height          float64
	MinWidth        float64
	MaxWidth        float64
	MinHeight       float64
	MaxHeight       float64
	MarginTop       float64
	MarginRight     float64
	MarginBottom    float64
	MarginLeft      float64
	MarginLeftAuto  bool // margin-left: auto (horizontal centering with right auto)
	MarginRightAuto bool // margin-right: auto
	PaddingTop      float64
	PaddingRight    float64
	PaddingBottom   float64
	PaddingLeft     float64
	BorderTop       border
	BorderRight     border
	BorderBottom    border
	BorderLeft      border
	Color           [3]float64
	BGColor         [4]float64 // rgba, 0..1
	FontFamily      []string
	FontSize        float64 // pts
	FontWeight      int
	FontItalic      bool
	LineHeight      float64 // pts; 0 = "normal"
	TextAlign       string  // "left" | "right" | "center" | "justify"
	VerticalAlign   string  // "baseline" | "top" | "middle" | "bottom"
	WhiteSpace      string  // "normal" | "nowrap" | "pre"
	TextDecoration  string  // "none" | "underline" | "line-through"
	LetterSpacing   float64
	TextIndent      float64
	BorderCollapse  string // "separate" | "collapse"
	BorderSpacing   float64
	TableLayout     string // "auto" | "fixed"
	IsReplaced      bool   // img, hr
	PageBreakBefore string // "" | "always" | "avoid"
	PageBreakAfter  string // "" | "always" | "avoid"
	PageBreakInside string // "" | "always" | "avoid"
}

type border struct {
	Width float64
	Style string // "none" | "solid" | "dashed" | "dotted"
	Color [3]float64
}

// initialStyle returns the CSS initial values.
func initialStyle() ResolvedStyle {
	return ResolvedStyle{
		Display:        "inline",
		Position:       "static",
		Float:          "none",
		Clear:          "none",
		Width:          -1,
		WidthPercent:   -1,
		Height:         -1,
		MinWidth:       0,
		MaxWidth:       -1,
		MinHeight:      0,
		MaxHeight:      -1,
		Color:          [3]float64{0, 0, 0},
		BGColor:        [4]float64{0, 0, 0, 0},
		FontSize:       12, // 16px at 96dpi
		FontWeight:     400,
		VerticalAlign:  "baseline",
		WhiteSpace:     "normal",
		TextDecoration: "none",
		BorderCollapse: "separate",
		BorderSpacing:  0,
		TableLayout:    "auto",
	}
}

// styleContext carries per-element resolution inputs.
type styleContext struct {
	sheets    []*css.Stylesheet
	media     string
	viewportW float64 // containing-block width for % of margins/padding/width
	viewportH float64 // for % of height
}

// resolveStyles walks the tree top-down, computing the used style of every
// element from the cascade (UA → sheets in order → inline) plus inheritance.
func resolveStyles(root *html.Node, sheets []*css.Stylesheet, media string, viewportW, viewportH float64) map[*html.Node]ResolvedStyle {
	ctx := &styleContext{sheets: sheets, media: media, viewportW: viewportW, viewportH: viewportH}
	out := map[*html.Node]ResolvedStyle{}
	raws := map[*html.Node]map[string]string{}
	var walk func(n *html.Node, parent *ResolvedStyle)
	walk = func(n *html.Node, parent *ResolvedStyle) {
		var st ResolvedStyle
		if n.Type == html.ElementNode {
			raw := cascadeRaw(ctx, n)
			raws[n] = raw
			st = initialStyle()
			if parent != nil {
				inheritProps(&st, *parent, raw)
			}
			applyFontProps(&st, raw, parent)
			applyRestProps(&st, raw, ctx)
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
			if ctx.media != "" && r.Media != "all" && r.Media != ctx.media {
				continue
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
func applyRestProps(st *ResolvedStyle, raw map[string]string, ctx *styleContext) {
	fs := st.FontSize
	shorthands := []string{"margin", "padding", "border", "border-width", "border-style", "border-color"}
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
				"inline-block", "table-caption", "table-column", "table-column-group":
				st.Display = value
			}
		case "position":
			if value == "static" || value == "relative" {
				st.Position = value
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
			if v, ok := lengthBox(value, fs, ctx.viewportH, "auto"); ok {
				st.Height = v
			}
		case "min-width":
			if v, ok := lengthBox(value, fs, ctx.viewportW, "none"); ok {
				st.MinWidth = v
			}
		case "max-width":
			if v, ok := lengthBox(value, fs, ctx.viewportW, "none"); ok {
				st.MaxWidth = v
			}
		case "min-height":
			if v, ok := lengthBox(value, fs, ctx.viewportH, "none"); ok {
				st.MinHeight = v
			}
		case "max-height":
			if v, ok := lengthBox(value, fs, ctx.viewportH, "none"); ok {
				st.MaxHeight = v
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
			if r, g, b, _, ok := css.ParseColor(value); ok {
				st.Color = [3]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255}
			}
		case "background-color":
			if r, g, b, a, ok := css.ParseColor(value); ok {
				st.BGColor = [4]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255, a}
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
			if value == "always" || value == "avoid" {
				st.PageBreakBefore = value
			}
		case "page-break-after", "break-after":
			if value == "always" || value == "avoid" {
				st.PageBreakAfter = value
			}
		case "page-break-inside", "break-inside":
			if value == "always" || value == "avoid" {
				st.PageBreakInside = value
			}
		}
	}
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
	case "thead", "tbody", "tfoot":
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
