// Package layout turns the parsed HTML tree plus resolved styles into a
// display list: absolute-positioned drawing operations in a continuous
// canvas (y grows downward from the top of the page content area). Painting
// into a pdf.Document is done by Paint (paint.go).
//
// Report-engine scope: block and inline flow, margin collapsing between
// siblings, tables (separate borders, colspan), images, lists, text wrapping
// with the embedded Liberation Sans font, float lite (left/right + clear),
// real inline-block, box-sizing, position relative/absolute lite, and a
// partial flex (row/column) subset and CSS grid lite (tracks, gap, column span).
package layout

import (
	"encoding/binary"
	"errors"
	"strconv"
	"strings"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

// Options controls a Layout run.
type Options struct {
	Width      float64 // viewport/content width in points
	Height     float64 // viewport height in points (for % heights)
	Font       *pdf.Font
	Faces      *pdf.FaceSet  // optional Liberation family; defaults loaded when nil
	Registry   *pdf.Registry // optional discovered fonts (--font-path)
	Sheets     []*css.Stylesheet
	Media      string // "print" or "screen"; "" = apply "all" rules only
	Images     func(src string) ([]byte, error)
	Background bool    // paint background colors
	DebugBoxes bool    // outline every box for test/golden output
	Zoom       float64 // zoom factor; style lengths are scaled by it (any positive value, < 1 shrinks)
}

// Result is a display list plus the canvas bounds.
type Result struct {
	Ops    []Op
	Width  float64
	Height float64

	root *box // element box tree, kept for Paint (Locations)

	// Pages maps page index → indices into Ops of the ops painted on that
	// page. Filled by Paint using its pagination semantics (an op goes to
	// the page containing its top edge; ops crossing a page boundary move
	// wholly to the next page).
	Pages [][]int
	// Locations lists every element box in document order with the page its
	// first op landed on and its canvas rect. Filled by Paint; boxes without
	// ops use the page of their y position.
	Locations []ElementLocation
}

// ElementLocation describes where one element box landed after pagination.
// X/Y/W/H are in canvas coordinates (y down from the top of the page content
// area); Page is the page index the box's first op was painted on.
type ElementLocation struct {
	Node *html.Node
	Page int
	X, Y float64
	W, H float64
}

// OpKind discriminates display-list operations.
type OpKind int

const (
	OpFillRect OpKind = iota
	OpStrokeRect
	OpLine
	OpText
	OpImage
	OpLinkURI
	OpBullet
)

// Op is one display-list operation. Coordinates are in canvas points; for
// OpText and OpBullet, Y is the baseline.
type Op struct {
	Kind    OpKind
	X, Y    float64
	W, H    float64
	R, G, B float64 // 0..1
	Alpha   float64
	Width   float64 // stroke width for OpLine

	Text string
	Font *pdf.Font
	Size float64
	Bold bool

	URI string

	Image  []byte // PNG or JPEG bytes
	ImgW   int
	ImgH   int
	IsJPEG bool

	// Fixed marks ops from position:fixed boxes; Paint stamps them on every
	// page at viewport-relative coordinates.
	Fixed bool

	// ZIndex paints later (higher) above earlier ops when non-zero or set.
	ZIndex    int
	ZIndexSet bool

	// RotateDeg rotates the glyph around its baseline origin (PDF text matrix).
	// Used for writing-mode:vertical-* upright→sideways CJK (90°).
	RotateDeg float64
}

type engine struct {
	opts      Options
	font      *pdf.Font // default/regular face (metrics fallback)
	faces     *pdf.FaceSet
	registry  *pdf.Registry
	styles    map[*html.Node]ResolvedStyle
	ops       []Op
	noEmit    bool // measurement mode: compute geometry without emitting ops
	height    float64
	scale     float64 // zoom factor applied to style lengths (>= 1)
	zIndex    int
	zIndexSet bool
}

// faceFor selects the TrueType face for a resolved style (bold/italic),
// preferring CSS font-family matches from the opt-in registry, then the
// bundled Liberation FaceSet.
func (e *engine) faceFor(st ResolvedStyle) *pdf.Font {
	if e.registry != nil {
		if f := e.registry.Lookup(st.FontFamily, st.FontWeight, st.FontItalic); f != nil {
			return f
		}
	}
	if e.faces != nil {
		if f := e.faces.Resolve(st.FontWeight, st.FontItalic); f != nil {
			return f
		}
	}
	return e.font
}

// scalePt applies the engine zoom factor to a style length in points.
func (e *engine) scalePt(v float64) float64 { return v * e.scale }

// zoomScale returns the effective zoom factor: any positive opts.Zoom
// (values below 1 shrink; 0 means no zoom).
func zoomScale(z float64) float64 {
	if z > 0 {
		return z
	}
	return 1
}

func (e *engine) add(op Op) {
	if !e.noEmit {
		op.ZIndex = e.zIndex
		op.ZIndexSet = e.zIndexSet
		e.ops = append(e.ops, op)
	}
}

func (e *engine) pushZ(st ResolvedStyle) (prevZ int, prevSet bool) {
	prevZ, prevSet = e.zIndex, e.zIndexSet
	if st.ZIndexSet {
		e.zIndex = st.ZIndex
		e.zIndexSet = true
	}
	return prevZ, prevSet
}

func (e *engine) popZ(prevZ int, prevSet bool) {
	e.zIndex, e.zIndexSet = prevZ, prevSet
}

// Layout renders the document into a display list.
func Layout(root *html.Node, opts Options) (*Result, error) {
	if root == nil {
		return nil, errors.New("layout: nil root")
	}
	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		return nil, err
	}
	if opts.Faces != nil {
		faces = opts.Faces
	}
	font := opts.Font
	if font == nil {
		font = faces.Regular
	}
	e := &engine{
		opts:     opts,
		font:     font,
		faces:    faces,
		registry: opts.Registry,
		styles:   resolveStyles(root, opts.Sheets, opts.Media, opts.Width, opts.Height),
		scale:    zoomScale(opts.Zoom),
	}

	b := e.build(root, opts.Width, 0, 0)
	res := &Result{Ops: e.ops, Width: opts.Width, Height: opts.Height, root: b}
	if b != nil {
		res.Height = b.y + b.h
	}
	if res.Height < e.height {
		res.Height = e.height
	}
	return res, nil
}

// box is one laid-out box.
type box struct {
	node  *html.Node
	style ResolvedStyle
	x, y  float64 // border-box top-left
	w, h  float64 // border-box size
	kind  string  // "block" | "table" | "cell" | "replaced"
	// opStart/opEnd bound the inclusive range of e.ops indices that this
	// box's subtree emitted. opEnd < opStart means the box emitted nothing
	// (e.g. boxes built during a noEmit measure pass).
	opStart, opEnd int
	children       []*box
	firstBaseline  float64
	// table cells
	col, span int
	contentW  float64
	contentH  float64
	// rows[i] holds the cell boxes of table row i, in document order. The
	// row's op range is from rows[i][0].opStart to rows[i][len-1].opEnd.
	rows [][]*box
	// headerRows is the number of leading rows that came from <thead> /
	// table-header-group (for repeating headers across pages).
	headerRows int
	// replaced
	imgSrc  string
	imgData []byte
	imgJPEG bool
	imgW    int
	imgH    int
}

func (e *engine) build(n *html.Node, availW, x, y float64) *box {
	st := e.styles[n]
	if n.Type == html.TextNode {
		return nil
	}
	if st.Display == "none" {
		return nil
	}
	prevZ, prevSet := e.pushZ(st)
	defer e.popZ(prevZ, prevSet)
	start := len(e.ops)
	var b *box
	switch n.Name {
	case "img":
		b = e.buildImage(n, st, x, y)
	case "hr":
		b = e.buildHR(n, st, availW, x, y)
	}
	// Out-of-flow positioning wraps the display type so fixed/absolute flex
	// and grid containers still get the right formatting context.
	if b == nil && st.Position == "fixed" {
		b = e.buildFixed(n, st, availW, x, y)
	}
	if b == nil && st.Position == "absolute" {
		b = e.buildAbsolute(n, st, availW, x, y)
	}
	if b == nil && (st.WritingMode == "vertical-rl" || st.WritingMode == "vertical-lr") {
		b = e.buildVerticalBlock(n, st, availW, x, y)
	}
	if b == nil && (st.Display == "flex" || st.Display == "inline-flex") {
		b = e.buildFlex(n, st, availW, x, y)
	}
	if b == nil && (st.Display == "grid" || st.Display == "inline-grid") {
		b = e.buildGrid(n, st, availW, x, y)
	}
	if b == nil && isTableDisplay(st.Display) {
		b = e.buildTable(n, st, availW, x, y)
	}
	if b == nil {
		b = e.buildBlock(n, st, availW, x, y)
	}
	if b != nil && (st.Position == "relative" || st.Position == "sticky") {
		e.applyRelativeOffset(b)
	}
	b.opStart, b.opEnd = start, len(e.ops)-1
	if b != nil && st.Position == "fixed" {
		e.markOpsFixed(b.opStart, b.opEnd)
	}
	return b
}

func isTableDisplay(d string) bool {
	switch d {
	case "table", "table-row", "table-row-group", "table-header-group",
		"table-footer-group", "table-cell", "table-caption":
		return true
	}
	return false
}

// buildBlock lays out a block-level box.
func (e *engine) buildBlock(n *html.Node, st ResolvedStyle, availW, x, y float64) *box {
	ml, mr := e.scalePt(st.MarginLeft), e.scalePt(st.MarginRight)
	b := &box{node: n, style: st, kind: "block", x: x, y: y}
	// Default: fill remaining width after horizontal margins.
	b.w = availW - ml - mr
	if b.w < 0 {
		b.w = 0
	}
	definiteW := st.Width >= 0 || st.WidthPercent >= 0
	if st.WidthPercent >= 0 {
		b.w = availW * st.WidthPercent / 100
	} else if st.Width >= 0 {
		b.w = e.scalePt(st.Width)
	}
	// content-box (default): specified width is the content width, so the
	// border box grows by horizontal padding + border. border-box: specified
	// width already is the border-box size.
	if definiteW && st.BoxSizing != "border-box" {
		b.w += e.scalePt(st.PaddingLeft) + e.scalePt(st.PaddingRight) +
			e.scalePt(st.BorderLeft.Width) + e.scalePt(st.BorderRight.Width)
	}
	if st.MinWidth > 0 && b.w < e.scalePt(st.MinWidth) {
		b.w = e.scalePt(st.MinWidth)
	}
	if st.MaxWidth >= 0 && b.w > e.scalePt(st.MaxWidth) {
		b.w = e.scalePt(st.MaxWidth)
	}
	// Horizontal margin: auto centers (or pushes) a definite-width box.
	if definiteW && (st.MarginLeftAuto || st.MarginRightAuto) {
		free := availW - b.w
		if free < 0 {
			free = 0
		}
		switch {
		case st.MarginLeftAuto && st.MarginRightAuto:
			ml = free / 2
			mr = free - ml
		case st.MarginLeftAuto:
			ml = free - mr
			if ml < 0 {
				ml = 0
			}
		case st.MarginRightAuto:
			mr = free - ml
			if mr < 0 {
				mr = 0
			}
		}
	}
	b.x = x + ml
	contentW := b.w - e.scalePt(st.PaddingLeft) - e.scalePt(st.PaddingRight) - e.scalePt(st.BorderLeft.Width) - e.scalePt(st.BorderRight.Width)
	if contentW < 0 {
		contentW = 0
	}
	contentX := b.x + e.scalePt(st.BorderLeft.Width) + e.scalePt(st.PaddingLeft)

	// Content ops are recorded first so we know the box height; background
	// and borders are then inserted *before* those ops so paint order is
	// bg → borders → children (otherwise fills cover text).
	contentStart := len(e.ops)

	cy := e.scalePt(st.PaddingTop) + e.scalePt(st.BorderTop.Width)
	cy = e.flowChildren(b, n.Children, st, contentW, contentX, y, cy)
	// padding-bottom is inside the border box (space above border-bottom /
	// letterhead rules — fixture-07/16).
	cy += e.scalePt(st.PaddingBottom)

	// list marker
	if n.Name == "li" && b.firstBaseline > 0 {
		e.add(Op{Kind: OpBullet, X: contentX + 4, Y: b.firstBaseline, Text: "\u2022", Font: e.font, Size: e.scalePt(st.FontSize), R: st.Color[0], G: st.Color[1], B: st.Color[2]})
	}

	if st.Height >= 0 {
		h := e.scalePt(st.Height)
		if st.BoxSizing != "border-box" {
			h += e.scalePt(st.PaddingTop) + e.scalePt(st.PaddingBottom) +
				e.scalePt(st.BorderTop.Width) + e.scalePt(st.BorderBottom.Width)
		}
		if cy < h {
			cy = h
		}
	}
	if st.MinHeight > 0 && cy < e.scalePt(st.MinHeight) {
		cy = e.scalePt(st.MinHeight)
	}
	if st.MaxHeight >= 0 && cy > e.scalePt(st.MaxHeight) {
		cy = e.scalePt(st.MaxHeight)
	}
	b.h = cy

	e.prependChrome(contentStart, st, b.x, y, b.w, b.h)
	return b
}

// buildAbsolute places an out-of-flow box using left/top (and optional width/
// height). Containing block is the parent's content edge approximation
// (availW/x/y passed from the caller).
func (e *engine) buildAbsolute(n *html.Node, st ResolvedStyle, availW, x, y float64) *box {
	start := len(e.ops)
	b := e.buildInFlowDisplay(n, st, availW, x, y)
	if b == nil {
		return nil
	}
	b.opStart, b.opEnd = start, len(e.ops)-1
	ax, ay := x, y
	if !st.LeftAuto {
		ax = x + e.scalePt(st.Left)
	} else if !st.RightAuto {
		ax = x + availW - b.w - e.scalePt(st.Right)
	}
	if !st.TopAuto {
		ay = y + e.scalePt(st.Top)
	} else if !st.BottomAuto {
		ay = y + e.scalePt(st.Bottom)
	}
	dx, dy := ax-b.x, ay-b.y
	b.x, b.y = ax, ay
	e.shiftBoxOps(b, dx, dy)
	return b
}

// buildFixed places an out-of-flow box against the initial containing block
// (viewport origin). Ops are marked Fixed so Paint stamps them on every page.
func (e *engine) buildFixed(n *html.Node, st ResolvedStyle, availW, x, y float64) *box {
	_ = availW
	_ = x
	_ = y
	cbW := e.opts.Width
	if cbW <= 0 {
		cbW = availW
	}
	start := len(e.ops)
	b := e.buildInFlowDisplay(n, st, cbW, 0, 0)
	if b == nil {
		return nil
	}
	b.opStart, b.opEnd = start, len(e.ops)-1
	ax, ay := 0.0, 0.0
	if !st.LeftAuto {
		ax = e.scalePt(st.Left)
	} else if !st.RightAuto {
		ax = cbW - b.w - e.scalePt(st.Right)
	}
	if !st.TopAuto {
		ay = e.scalePt(st.Top)
	} else if !st.BottomAuto {
		ay = e.opts.Height - b.h - e.scalePt(st.Bottom)
		if ay < 0 {
			ay = e.scalePt(st.Bottom)
		}
	}
	dx, dy := ax-b.x, ay-b.y
	b.x, b.y = ax, ay
	e.shiftBoxOps(b, dx, dy)
	return b
}

// buildInFlowDisplay builds flex/grid/table/block ignoring position.
func (e *engine) buildInFlowDisplay(n *html.Node, st ResolvedStyle, availW, x, y float64) *box {
	if st.Display == "flex" || st.Display == "inline-flex" {
		return e.buildFlex(n, st, availW, x, y)
	}
	if st.Display == "grid" || st.Display == "inline-grid" {
		return e.buildGrid(n, st, availW, x, y)
	}
	if isTableDisplay(st.Display) {
		return e.buildTable(n, st, availW, x, y)
	}
	return e.buildBlock(n, st, availW, x, y)
}

func (e *engine) markOpsFixed(start, end int) {
	if end < start {
		return
	}
	for i := start; i <= end && i < len(e.ops); i++ {
		e.ops[i].Fixed = true
	}
}

// prependChrome inserts background + border ops at insertAt so they paint
// under any content ops already appended for this box.
func (e *engine) prependChrome(insertAt int, st ResolvedStyle, x, y, w, h float64) {
	if e.noEmit {
		return
	}
	var chrome []Op
	if st.BGColor[3] > 0 && e.opts.Background {
		chrome = append(chrome, Op{Kind: OpFillRect, X: x, Y: y, W: w, H: h,
			R: st.BGColor[0], G: st.BGColor[1], B: st.BGColor[2], Alpha: st.BGColor[3]})
	}
	// borders (same geometry as emitBorders)
	wt, wr, wb, wl := e.scalePt(st.BorderTop.Width), e.scalePt(st.BorderRight.Width), e.scalePt(st.BorderBottom.Width), e.scalePt(st.BorderLeft.Width)
	if wt > 0 && st.BorderTop.Style != "none" {
		chrome = append(chrome, Op{Kind: OpLine, X: x, Y: y, W: w, H: 0, Width: wt, R: st.BorderTop.Color[0], G: st.BorderTop.Color[1], B: st.BorderTop.Color[2]})
	}
	if wr > 0 && st.BorderRight.Style != "none" {
		chrome = append(chrome, Op{Kind: OpLine, X: x + w, Y: y, W: 0, H: h, Width: wr, R: st.BorderRight.Color[0], G: st.BorderRight.Color[1], B: st.BorderRight.Color[2]})
	}
	if wb > 0 && st.BorderBottom.Style != "none" {
		chrome = append(chrome, Op{Kind: OpLine, X: x, Y: y + h, W: w, H: 0, Width: wb, R: st.BorderBottom.Color[0], G: st.BorderBottom.Color[1], B: st.BorderBottom.Color[2]})
	}
	if wl > 0 && st.BorderLeft.Style != "none" {
		chrome = append(chrome, Op{Kind: OpLine, X: x, Y: y, W: 0, H: h, Width: wl, R: st.BorderLeft.Color[0], G: st.BorderLeft.Color[1], B: st.BorderLeft.Color[2]})
	}
	if len(chrome) == 0 {
		return
	}
	for i := range chrome {
		chrome[i].ZIndex = e.zIndex
		chrome[i].ZIndexSet = e.zIndexSet
	}
	// insert chrome before content ops
	tail := append([]Op(nil), e.ops[insertAt:]...)
	e.ops = e.ops[:insertAt]
	e.ops = append(e.ops, chrome...)
	e.ops = append(e.ops, tail...)
}

// partition splits children into block-level and inline nodes.
func (e *engine) partition(children []*html.Node, blocks, inlines *[]*html.Node) {
	for _, c := range children {
		if c.Type != html.ElementNode {
			if c.Type == html.TextNode {
				*inlines = append(*inlines, c)
			}
			continue
		}
		cs := e.styles[c]
		if cs.Display == "none" {
			continue
		}
		if cs.Float != "none" || cs.Position == "absolute" || cs.Position == "fixed" {
			*blocks = append(*blocks, c)
			continue
		}
		if cs.Display == "inline" || cs.Display == "inline-block" || cs.Display == "inline-flex" || c.Name == "img" {
			*inlines = append(*inlines, c)
			continue
		}
		*blocks = append(*blocks, c)
	}
}

// isInlineChild reports whether n participates in an inline formatting context.
func (e *engine) isInlineChild(n *html.Node) bool {
	if n.Type == html.TextNode {
		return true
	}
	if n.Type != html.ElementNode {
		return false
	}
	cs := e.styles[n]
	if cs.Display == "none" || cs.Float != "none" || cs.Position == "absolute" || cs.Position == "fixed" {
		return false
	}
	return cs.Display == "inline" || cs.Display == "inline-block" || cs.Display == "inline-flex" || n.Name == "img"
}

// onlyCollapsibleWS reports whether every node is a text node of only
// whitespace (collapses between blocks and must not kill margin collapse).
func onlyCollapsibleWS(nodes []*html.Node) bool {
	if len(nodes) == 0 {
		return true
	}
	for _, n := range nodes {
		if n.Type != html.TextNode || strings.TrimSpace(n.Text) != "" {
			return false
		}
	}
	return true
}

// flowChildren lays out children in document order: runs of inlines, then
// block boxes, alternating as they appear. Floated children are positioned
// out of flow with a lite exclusion model; clear advances past them.
// Returns the advanced content height (cy end − cy start contribution is
// encoded as the final cy relative to start; callers pass starting cy).
func (e *engine) flowChildren(parent *box, children []*html.Node, st ResolvedStyle, contentW, contentX, y, cy float64) float64 {
	prevBottom := 0.0
	floats := newFloatState(contentX, contentW)
	var deferred []*html.Node
	// Absolute/fixed containing-block origin is the content edge at entry.
	// Do not use the post-flow cy or deferred boxes sit below in-flow siblings.
	absOriginY := y + cy
	i := 0
	for i < len(children) {
		n := children[i]
		if n.Type == html.ElementNode && e.styles[n].Display == "none" {
			i++
			continue
		}
		// Skip pure whitespace text so it does not interrupt margin collapse
		// between block siblings (fixture-19 margin-bottom between divs).
		if n.Type == html.TextNode && strings.TrimSpace(n.Text) == "" {
			i++
			continue
		}
		if n.Type == html.ElementNode && (e.styles[n].Position == "absolute" || e.styles[n].Position == "fixed") {
			// Defer out-of-flow boxes so they paint above in-flow content
			// (absolute overlays sit on top of later siblings' text).
			deferred = append(deferred, n)
			i++
			continue
		}
		if n.Type == html.ElementNode && e.styles[n].Float != "none" {
			cs := e.styles[n]
			cy = floats.clear(cs.Clear, y, cy)
			fb := e.placeFloat(n, cs, &floats, contentW, contentX, y, cy)
			if fb != nil && parent != nil {
				parent.children = append(parent.children, fb)
				if e.opts.DebugBoxes {
					e.add(Op{Kind: OpStrokeRect, X: fb.x, Y: fb.y, W: fb.w, H: fb.h, R: 1, G: 0, B: 0})
				}
			}
			prevBottom = 0
			i++
			continue
		}
		if e.isInlineChild(n) {
			var run []*html.Node
			for i < len(children) {
				c := children[i]
				if c.Type == html.ElementNode && e.styles[c].Display == "none" {
					i++
					continue
				}
				if c.Type == html.ElementNode && e.styles[c].Float != "none" {
					break
				}
				if c.Type == html.TextNode && strings.TrimSpace(c.Text) == "" {
					// keep interior whitespace inside an inline run, but a
					// run that is only WS is dropped below.
					run = append(run, c)
					i++
					continue
				}
				if !e.isInlineChild(c) {
					break
				}
				run = append(run, c)
				i++
			}
			if onlyCollapsibleWS(run) {
				continue
			}
			if len(run) > 0 {
				pb := parent
				if pb == nil {
					pb = &box{style: st}
				}
				ix, iw := floats.exclusion(y, cy)
				h := e.layoutInline(pb, run, iw, ix, y+cy)
				cy += h
				if h > 0 {
					prevBottom = 0
				}
			}
			continue
		}
		// block-level
		if n.Type != html.ElementNode {
			i++
			continue
		}
		cs := e.styles[n]
		cy = floats.clear(cs.Clear, y, cy)
		cy += collapseMargins(prevBottom, e.scalePt(cs.MarginTop))
		bx, bw := floats.exclusion(y, cy)
		cb := e.build(n, bw, bx, y+cy)
		if cb == nil {
			prevBottom = 0
			i++
			continue
		}
		cy += cb.h
		prevBottom = e.scalePt(cs.MarginBottom)
		if parent != nil {
			parent.children = append(parent.children, cb)
			if e.opts.DebugBoxes {
				e.add(Op{Kind: OpStrokeRect, X: cb.x, Y: cb.y, W: cb.w, H: cb.h, R: 1, G: 0, B: 0})
			}
		}
		i++
	}
	for _, n := range deferred {
		ab := e.build(n, contentW, contentX, absOriginY)
		if ab != nil && parent != nil {
			parent.children = append(parent.children, ab)
		}
	}
	return floats.extentCy(y, cy)
}

// placeFloat lays out n as a float:left|right box and records it in floats.
// Consecutive same-side floats pack horizontally when width remains;
// otherwise they stack below the previous float bottom.
func (e *engine) placeFloat(n *html.Node, cs ResolvedStyle, floats *floatState, contentW, contentX, y, cy float64) *box {
	avail := contentW
	if cs.Width < 0 && cs.WidthPercent < 0 {
		intr := e.measureCellContent(n, cs)
		intr += e.scalePt(cs.PaddingLeft) + e.scalePt(cs.PaddingRight) +
			e.scalePt(cs.BorderLeft.Width) + e.scalePt(cs.BorderRight.Width) +
			e.scalePt(cs.MarginLeft) + e.scalePt(cs.MarginRight)
		if intr > 0 && intr < avail {
			avail = intr
		}
	}
	flowY := y + cy
	fx, fy := contentX, flowY

	switch cs.Float {
	case "left":
		if floats.hasLeft {
			room := contentX + contentW - floats.leftEdge
			if floats.hasRight && floats.rightEdge-floats.leftEdge < room {
				room = floats.rightEdge - floats.leftEdge
			}
			if room >= avail*0.5 { // enough room to attempt side-by-side
				fx = floats.leftEdge
				fy = floats.leftTop
				if fy < flowY {
					fy = flowY
				}
				if avail > room {
					avail = room
				}
			} else if floats.leftBottom > fy {
				fy = floats.leftBottom
			}
		}
	case "right":
		if floats.hasRight {
			room := floats.rightEdge - contentX
			if floats.hasLeft && floats.rightEdge-floats.leftEdge < room {
				room = floats.rightEdge - floats.leftEdge
			}
			if room >= avail*0.5 {
				fy = floats.rightTop
				if fy < flowY {
					fy = flowY
				}
				if avail > room {
					avail = room
				}
			} else if floats.rightBottom > fy {
				fy = floats.rightBottom
			}
		}
	}

	fb := e.build(n, avail, fx, fy)
	if fb == nil {
		return nil
	}
	if cs.Float == "left" && floats.hasLeft && fb.x+fb.w > contentX+contentW {
		// Overflowed the pack attempt — stack below.
		fy = floats.leftBottom
		if fy < flowY {
			fy = flowY
		}
		dx, dy := contentX-fb.x, fy-fb.y
		fb.x, fb.y = contentX, fy
		e.shiftBoxOps(fb, dx, dy)
	}
	if cs.Float == "right" {
		mr := e.scalePt(cs.MarginRight)
		wantX := contentX + contentW - fb.w - mr
		dx := wantX - fb.x
		fb.x = wantX
		e.shiftBoxOps(fb, dx, 0)
	}
	floats.place(cs.Float, fb)
	return fb
}

// shiftBoxOps translates every op in b's op range by (dx, dy).
func (e *engine) shiftBoxOps(b *box, dx, dy float64) {
	if dx == 0 && dy == 0 {
		return
	}
	if b.opEnd < b.opStart {
		return
	}
	for k := b.opStart; k <= b.opEnd && k < len(e.ops); k++ {
		e.ops[k].X += dx
		e.ops[k].Y += dy
	}
}

func collapseMargins(a, b float64) float64 {
	if a <= 0 && b <= 0 {
		return a + b
	}
	if a > b {
		return a
	}
	return b
}

func (e *engine) emitBorders(st ResolvedStyle, x, y, w, h float64) {
	wt, wr, wb, wl := e.scalePt(st.BorderTop.Width), e.scalePt(st.BorderRight.Width), e.scalePt(st.BorderBottom.Width), e.scalePt(st.BorderLeft.Width)
	if wt > 0 && st.BorderTop.Style != "none" {
		e.add(Op{Kind: OpLine, X: x, Y: y, W: w, H: 0, Width: wt, R: st.BorderTop.Color[0], G: st.BorderTop.Color[1], B: st.BorderTop.Color[2]})
	}
	if wr > 0 && st.BorderRight.Style != "none" {
		e.add(Op{Kind: OpLine, X: x + w, Y: y, W: 0, H: h, Width: wr, R: st.BorderRight.Color[0], G: st.BorderRight.Color[1], B: st.BorderRight.Color[2]})
	}
	if wb > 0 && st.BorderBottom.Style != "none" {
		e.add(Op{Kind: OpLine, X: x, Y: y + h, W: w, H: 0, Width: wb, R: st.BorderBottom.Color[0], G: st.BorderBottom.Color[1], B: st.BorderBottom.Color[2]})
	}
	if wl > 0 && st.BorderLeft.Style != "none" {
		e.add(Op{Kind: OpLine, X: x, Y: y, W: 0, H: h, Width: wl, R: st.BorderLeft.Color[0], G: st.BorderLeft.Color[1], B: st.BorderLeft.Color[2]})
	}
}

// --- replaced elements ---

func (e *engine) buildImage(n *html.Node, st ResolvedStyle, x, y float64) *box {
	b := &box{node: n, style: st, kind: "replaced", x: x, y: y}
	src := n.Attribute("src")
	if src != "" && e.opts.Images != nil {
		if data, err := e.opts.Images(src); err == nil {
			if w, h, jpeg, ok := imageDims(data); ok {
				b.imgSrc, b.imgData, b.imgJPEG, b.imgW, b.imgH = src, data, jpeg, w, h
			}
		}
	}
	b.w = e.scalePt(pxToPt(float64(b.imgW)))
	b.h = e.scalePt(pxToPt(float64(b.imgH)))
	// width/height HTML attributes are pixel values at 96dpi
	if v, err := strconv.Atoi(n.Attribute("width")); err == nil && v > 0 {
		if n.Attribute("height") == "" && st.Height < 0 && b.w > 0 {
			b.h = b.h * e.scalePt(pxToPt(float64(v))) / b.w
		}
		b.w = e.scalePt(pxToPt(float64(v)))
	}
	if v, err := strconv.Atoi(n.Attribute("height")); err == nil && v > 0 {
		if n.Attribute("width") == "" && st.Width < 0 && b.h > 0 {
			b.w = b.w * e.scalePt(pxToPt(float64(v))) / b.h
		}
		b.h = e.scalePt(pxToPt(float64(v)))
	}
	if st.Width >= 0 {
		b.w = e.scalePt(st.Width)
	}
	if st.Height >= 0 {
		b.h = e.scalePt(st.Height)
	}
	if st.MaxWidth >= 0 && b.w > e.scalePt(st.MaxWidth) {
		b.h = b.h * e.scalePt(st.MaxWidth) / b.w
		b.w = e.scalePt(st.MaxWidth)
	}
	if st.MaxHeight >= 0 && b.h > e.scalePt(st.MaxHeight) {
		b.w = b.w * e.scalePt(st.MaxHeight) / b.h
		b.h = e.scalePt(st.MaxHeight)
	}
	// Float (and other out-of-line) images paint here; in-flow <img> is
	// collected into the inline formatting context and painted on the line.
	if st.Float != "none" && b.imgData != nil {
		e.add(Op{Kind: OpImage, X: x, Y: y, W: b.w, H: b.h,
			Image: b.imgData, ImgW: b.imgW, ImgH: b.imgH, IsJPEG: b.imgJPEG})
	}
	return b
}

func (e *engine) buildHR(n *html.Node, st ResolvedStyle, availW, x, y float64) *box {
	b := &box{node: n, style: st, kind: "replaced", x: x, y: y, w: availW}
	if st.Width >= 0 {
		b.w = e.scalePt(st.Width)
	}
	b.h = e.scalePt(st.BorderTop.Width) + e.scalePt(st.BorderBottom.Width)
	if b.h <= 0 {
		b.h = 1
	}
	c := [3]float64{0, 0, 0}
	if st.BorderTop.Style != "none" {
		c = st.BorderTop.Color
	}
	if b.h > 0 {
		e.add(Op{Kind: OpFillRect, X: x, Y: y, W: b.w, H: b.h, R: c[0], G: c[1], B: c[2]})
	}
	return b
}

// imageDims extracts pixel dimensions from PNG or JPEG bytes.
func imageDims(data []byte) (w, h int, isJPEG bool, ok bool) {
	if len(data) >= 24 && string(data[:8]) == "\x89PNG\r\n\x1a\n" {
		return int(binary.BigEndian.Uint32(data[16:20])), int(binary.BigEndian.Uint32(data[20:24])), false, true
	}
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xD8 {
		pos := 2
		for pos+4 <= len(data) {
			if data[pos] != 0xFF {
				pos++
				continue
			}
			marker := data[pos+1]
			if marker == 0xD9 || marker == 0xDA {
				return 0, 0, false, false
			}
			if marker >= 0xC0 && marker <= 0xCF && marker != 0xC4 && marker != 0xC8 && marker != 0xCC {
				return int(binary.BigEndian.Uint16(data[pos+5 : pos+7])), int(binary.BigEndian.Uint16(data[pos+3 : pos+5])), true, true
			}
			segLen := int(data[pos+2])<<8 | int(data[pos+3])
			if segLen < 2 {
				return 0, 0, false, false
			}
			pos += 2 + segLen
		}
	}
	return 0, 0, false, false
}

// --- tables ---

func (e *engine) buildTable(n *html.Node, st ResolvedStyle, availW, x, y float64) *box {
	// flatten row groups into rows; count leading header-group rows
	var rows [][]*html.Node
	headerRows := 0
	var collect func(n *html.Node, inHeader bool)
	collect = func(n *html.Node, inHeader bool) {
		for _, c := range n.Children {
			if c.Type != html.ElementNode {
				continue
			}
			cs := e.styles[c]
			if cs.Display == "none" {
				continue
			}
			switch {
			case cs.Display == "table-row":
				var cells []*html.Node
				for _, cell := range c.Children {
					if cell.Type == html.ElementNode && e.styles[cell].Display == "table-cell" {
						cells = append(cells, cell)
					}
				}
				rows = append(rows, cells)
				if inHeader {
					headerRows++
				}
			case cs.Display == "table-header-group":
				collect(c, true)
			case strings.HasSuffix(cs.Display, "row-group"):
				collect(c, false)
			}
		}
	}
	collect(n, false)

	tb := &box{node: n, style: st, kind: "table", x: x, y: y, headerRows: headerRows}
	if len(rows) == 0 {
		return tb
	}

	// determine columns, honoring colspan
	nCols := 0
	for _, r := range rows {
		col := 0
		for _, cell := range r {
			col += colSpan(cell)
		}
		if col > nCols {
			nCols = col
		}
	}
	if nCols == 0 {
		return tb
	}

	// measure each cell's max-content width; colspan cells contribute their
	// content width evenly across the spanned columns (min floor per col).
	colW := make([]float64, nCols)
	var cellData [][]*box
	for _, r := range rows {
		col := 0
		var cells []*box
		for _, cellNode := range r {
			span := colSpan(cellNode)
			if span < 1 {
				span = 1
			}
			if col+span > nCols {
				span = nCols - col
				if span < 1 {
					span = 1
				}
			}
			cell := e.buildCell(cellNode, col, span)
			cells = append(cells, cell)
			if span == 1 {
				if cell.contentW > colW[col] {
					colW[col] = cell.contentW
				}
			} else if span > 1 {
				// Prefer existing single-col measurements; only grow if the
				// spanned content needs more total space than current sum.
				var sum float64
				for k := 0; k < span && col+k < nCols; k++ {
					sum += colW[col+k]
				}
				need := cell.contentW
				if need > sum {
					extra := (need - sum) / float64(span)
					for k := 0; k < span && col+k < nCols; k++ {
						colW[col+k] += extra
					}
				}
			}
			col += span
		}
		cellData = append(cellData, cells)
	}

	// table width
	// border-collapse: collapse suppresses the separate-border gap so colspan
	// header rows and body cells share edges instead of looking double-lined.
	spacing := e.scalePt(st.BorderSpacing)
	if st.BorderCollapse == "collapse" {
		spacing = 0
	}
	sum := 0.0
	for _, w := range colW {
		sum += w
	}
	sum += spacing * float64(nCols+1)
	sum += e.scalePt(st.BorderLeft.Width) + e.scalePt(st.BorderRight.Width) + e.scalePt(st.PaddingLeft) + e.scalePt(st.PaddingRight)
	tableW := availW
	if st.WidthPercent >= 0 {
		// width:% of the containing block (parent cell / block), not viewport
		tableW = availW * st.WidthPercent / 100
	} else if st.Width >= 0 {
		tableW = e.scalePt(st.Width)
		if tableW > availW && availW > 0 {
			tableW = availW
		}
	} else if sum < availW {
		tableW = sum
	}
	if tableW > sum {
		extra := (tableW - sum) / float64(nCols)
		for i := range colW {
			colW[i] += extra
		}
	} else if tableW < sum {
		// shrink proportionally
		inner := sum - spacing*float64(nCols+1)
		if inner > 0 {
			scale := (tableW - spacing*float64(nCols+1)) / inner
			if scale > 0 {
				for i := range colW {
					colW[i] *= scale
				}
			}
		}
	}
	tb.w = tableW
	cy := e.scalePt(st.PaddingTop) + e.scalePt(st.BorderTop.Width)
	for _, cells := range cellData {
		rowH := 0.0
		rowX := e.scalePt(st.PaddingLeft) + e.scalePt(st.BorderLeft.Width)
		for _, cell := range cells {
			cell.x = x + rowX
			cell.y = y + cy
			cellW := 0.0
			for k := 0; k < cell.span; k++ {
				cellW += colW[cell.col+k]
			}
			cellW += spacing * float64(cell.span-1)
			cell.w = cellW
			// Height against final column width so wrap matches paint.
			e.measureCellHeight(cell, cellW)
			if cell.contentH > rowH {
				rowH = cell.contentH
			}
			rowX += cellW + spacing
			tb.children = append(tb.children, cell)
		}
		// Row height must land on every cell so backgrounds/borders paint
		// with non-zero height (emitCell uses b.h, not contentH).
		if rowH <= 0 {
			rowH = 1
		}
		for _, cell := range cells {
			cell.h = rowH
		}
		cy += rowH + spacing
	}
	tb.rows = cellData
	tb.h = cy + e.scalePt(st.PaddingBottom) + e.scalePt(st.BorderBottom.Width)

	if st.BGColor[3] > 0 && e.opts.Background {
		e.add(Op{Kind: OpFillRect, X: x, Y: y, W: tb.w, H: tb.h,
			R: st.BGColor[0], G: st.BGColor[1], B: st.BGColor[2], Alpha: st.BGColor[3]})
	}
	e.emitBorders(st, x, y, tb.w, tb.h)

	// emit cell content and boxes now that positions are final
	for _, cell := range tb.children {
		e.emitCell(cell)
	}
	return tb
}

// buildCell measures a table cell's max-content width (no ops emitted).
// Height is not final here: layoutCell must run again with the real column
// width after column sizing, or narrow max-content widths force false wraps
// and inflate row heights (empty bands under single-line cell text).
func (e *engine) buildCell(n *html.Node, col, span int) *box {
	st := e.styles[n]
	b := &box{node: n, style: st, kind: "cell", col: col, span: span}
	b.contentW = e.measureCellContent(n, st)
	return b
}

// measureCellHeight lays out the cell at width (border-box) without emitting
// paint ops, and stores the result on b.contentH.
func (e *engine) measureCellHeight(b *box, width float64) {
	was := e.noEmit
	e.noEmit = true
	// Preserve the caller's noEmit flag. Nested tables call this during an
	// outer measure pass; restoring false mid-measure leaked ops at wrong
	// positions (fixture-10 nested table borders/text).
	b.contentH = e.layoutCell(b.node, b.style, width)
	e.noEmit = was
}

// cellBG returns the background to paint for a cell: the cell's own color,
// or the parent table-row's background when the cell is transparent (CSS
// does not inherit background, but row backgrounds show through empty
// cells in browsers — required for tr.good / tr.warn / tr.bad).
func (e *engine) cellBG(b *box) (r, g, bl, a float64, ok bool) {
	st := b.style
	if st.BGColor[3] > 0 {
		return st.BGColor[0], st.BGColor[1], st.BGColor[2], st.BGColor[3], true
	}
	if b.node != nil && b.node.Parent != nil {
		if ps, has := e.styles[b.node.Parent]; has && ps.Display == "table-row" && ps.BGColor[3] > 0 {
			return ps.BGColor[0], ps.BGColor[1], ps.BGColor[2], ps.BGColor[3], true
		}
	}
	return 0, 0, 0, 0, false
}

// emitCell paints a placed cell's background, borders and content.
func (e *engine) emitCell(b *box) {
	st := b.style
	start := len(e.ops)
	if e.opts.Background {
		if r, g, bl, a, ok := e.cellBG(b); ok {
			e.add(Op{Kind: OpFillRect, X: b.x, Y: b.y, W: b.w, H: b.h,
				R: r, G: g, B: bl, Alpha: a})
		}
	}
	e.emitBorders(st, b.x, b.y, b.w, b.h)
	contentW := b.w - e.scalePt(st.PaddingLeft) - e.scalePt(st.PaddingRight) - e.scalePt(st.BorderLeft.Width) - e.scalePt(st.BorderRight.Width)
	if contentW < 0 {
		contentW = 0
	}
	cx := b.x + e.scalePt(st.PaddingLeft) + e.scalePt(st.BorderLeft.Width)
	cy := b.y + e.scalePt(st.PaddingTop) + e.scalePt(st.BorderTop.Width)
	// vertical-align on table cells: shift content within the row box.
	if extra := b.h - b.contentH; extra > 0 {
		switch st.VerticalAlign {
		case "middle":
			cy += extra / 2
		case "bottom":
			cy += extra
		}
	}
	// flowChildren advances cy; cell content is rooted at absolute y=b.y so
	// pass y=0 and absolute positions via contentX / cy as canvas coords.
	_ = e.flowChildren(&box{style: st}, b.node.Children, st, contentW, cx, 0, cy)
	// Note: flowChildren uses y+cy for block placement; with y=0 that is cy.
	b.opStart, b.opEnd = start, len(e.ops)-1
}

// measureCellContent returns the max-content width of the cell.
func (e *engine) measureCellContent(n *html.Node, st ResolvedStyle) float64 {
	maxW := 0.0
	var measure func(n *html.Node, fs float64)
	measure = func(n *html.Node, fs float64) {
		switch n.Type {
		case html.TextNode:
			for _, word := range strings.Fields(n.Text) {
				w := e.measureText(word, fs*e.scale)
				if w > maxW {
					maxW = w
				}
			}
		case html.ElementNode:
			if e.styles[n].Display == "none" {
				return
			}
			fs = e.styles[n].FontSize
			for _, c := range n.Children {
				measure(c, fs)
			}
		}
	}
	measure(n, st.FontSize)
	maxW += e.scalePt(st.PaddingLeft) + e.scalePt(st.PaddingRight) + e.scalePt(st.BorderLeft.Width) + e.scalePt(st.BorderRight.Width)
	return maxW
}

// layoutCell measures the height of a cell's content (no ops emitted).
func (e *engine) layoutCell(n *html.Node, st ResolvedStyle, width float64) float64 {
	contentW := width - e.scalePt(st.PaddingLeft) - e.scalePt(st.PaddingRight) - e.scalePt(st.BorderLeft.Width) - e.scalePt(st.BorderRight.Width)
	if contentW < 0 {
		contentW = 0
	}
	cy := e.scalePt(st.PaddingTop) + e.scalePt(st.BorderTop.Width)
	cy = e.flowChildren(nil, n.Children, st, contentW, 0, 0, cy)
	return cy + e.scalePt(st.PaddingBottom) + e.scalePt(st.BorderBottom.Width)
}

func colSpan(n *html.Node) int {
	if v, err := strconv.Atoi(strings.TrimSpace(n.Attribute("colspan"))); err == nil && v > 1 {
		return v
	}
	return 1
}
