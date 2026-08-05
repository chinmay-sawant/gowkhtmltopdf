// Package layout turns the parsed HTML tree plus resolved styles into a
// display list: absolute-positioned drawing operations in a continuous
// canvas (y grows downward from the top of the page content area). Painting
// into a pdf.Document is done by Paint (paint.go).
//
// Report-engine scope: block and inline flow, margin collapsing between
// siblings, tables (separate borders, colspan), images, lists, text wrapping
// with the embedded Liberation Sans font, float lite (left/right + clear),
// real inline-block, box-sizing, position relative/absolute/fixed lite,
// print-scoped sticky (page content box = scrollport; overflow boxes at
// scroll offset 0 — see sticky.go), and a
// partial flex (row/column) subset, CSS grid lite (tracks, gap, column span),
// and CSS multi-column lite (column-count/width/gap/span/fill).
package layout

import (
	"encoding/binary"
	"errors"
	"strconv"
	"strings"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
	"gowkhtmltopdf/internal/svg"
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
	// PrintLinkUnderline is an opt-in operator policy (--print-link-underline):
	// after cascade, force text-decoration:underline on a[href]. Default off
	// so author CSS (including inherit → none) is honored.
	PrintLinkUnderline bool
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

	// StickyID links display-list ops to a position:sticky box after parent
	// prependChrome shifts op indices (0 = not sticky).
	StickyID int

	// ZIndex paints later (higher) above earlier ops when non-zero or set.
	ZIndex    int
	ZIndexSet bool

	// RotateDeg rotates the glyph around its baseline origin (PDF text matrix).
	// Used for writing-mode:vertical-* upright→sideways CJK (90°). Independent
	// of CSS transform CTM (which wraps the whole op via Xform).
	RotateDeg float64

	// Xform is a baked canvas-space CSS 2D transform (identity if unset).
	// Applied at paint via PDF cm (see pdfCTMFromCSS). Sibling flow unaffected.
	Xform    Matrix2D
	XformSet bool

	// PaintOpacity is element opacity (CSS opacity / filter:opacity), 0..1.
	// 0 or unset (≥1) means fully opaque. Nested opacities are multiplied.
	PaintOpacity float64
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
	stickySeq int // monotonically increasing sticky box IDs (for Op.StickyID)
	// transformCBDepth counts ancestors with transform≠none; fixed→absolute CB.
	transformCBDepth int
	// imgMaxW > 0 clamps replaced <img> boxes to this containing-block width
	// (table cell / float / inline formatting context).
	imgMaxW float64
	// bfcFloats is the floatState of the nearest enclosing BFC. Ordinary
	// blocks reuse it so floats affect later siblings; BFC roots push a
	// fresh state (see pushBFCFloats).
	bfcFloats *floatState
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

// faceForRune picks the first CSS font-family face (then defaults) that has a
// glyph for r — browser-like fallback so Hangul/Latin/CJK can come from
// different faces in one run.
func (e *engine) faceForRune(st ResolvedStyle, r rune) *pdf.Font {
	if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
		return e.faceFor(st)
	}
	if e.registry != nil {
		for _, fam := range st.FontFamily {
			f := e.registry.Lookup([]string{fam}, st.FontWeight, st.FontItalic)
			if f != nil && f.GlyphID(r) != 0 {
				return f
			}
		}
	}
	if e.faces != nil {
		if f := e.faces.Resolve(st.FontWeight, st.FontItalic); f != nil && f.GlyphID(r) != 0 {
			return f
		}
	}
	if e.font != nil && e.font.GlyphID(r) != 0 {
		return e.font
	}
	// Last resort: any opt-in registry face that covers this codepoint
	// (DejaVu/Noto when --font-path / --use-system-fonts scanned them).
	if e.registry != nil {
		if f := e.registry.FindWithGlyph(r, st.FontWeight, st.FontItalic); f != nil {
			return f
		}
	}
	return e.faceFor(st)
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
	} else if st.HasTransform || st.Opacity < 1 {
		// CSS: transform/opacity create a stacking context (like z-index:0).
		e.zIndex = 0
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

	// Pass 1: cascade without @container (used sizes unknown).
	styles := resolveStylesOpts(root, opts)
	// After definite inline sizes of size containers are known, re-cascade so
	// matching @container rules apply, then lay out once with final styles.
	if css.HasContainerRules(opts.Sheets) {
		cinfo := measureSizeContainers(root, styles, opts.Width)
		if len(cinfo) > 0 {
			styles = resolveStylesWithContainersOpts(root, opts, cinfo)
			// One nested remount: @container may change nested container-type.
			cinfo2 := measureSizeContainers(root, styles, opts.Width)
			if len(cinfo2) != len(cinfo) {
				styles = resolveStylesWithContainersOpts(root, opts, cinfo2)
			} else {
				for n, a := range cinfo {
					b, ok := cinfo2[n]
					if !ok || a.inlineSize != b.inlineSize || a.names != b.names {
						styles = resolveStylesWithContainersOpts(root, opts, cinfo2)
						break
					}
				}
			}
		}
	}

	e := &engine{
		opts:     opts,
		font:     font,
		faces:    faces,
		registry: opts.Registry,
		styles:   styles,
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
	// Paint-time CSS transforms/opacity: stamp composed CTMs after geometry
	// is final so transform-origin % resolves against the border box.
	stampBoxTransforms(b, IdentityMatrix(), res.Ops)
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
	rowSpan   int // vertical span (default 1) for <td rowspan>
	// rowBoxH is the height of the cell's starting row only. For rowspan>1,
	// h covers the full span (background/borders) while rowBoxH is what
	// rowsIntact uses so bottom-edge paint ops do not make the first row
	// look multi-page.
	rowBoxH    float64
	contentW   float64 // max-content border-box (preferred column contribution)
	contentMin float64 // min-content border-box (shrink floor)
	contentH   float64
	// rows[i] holds the cell boxes of table row i, in document order. The
	// row's op range is from rows[i][0].opStart to rows[i][len-1].opEnd.
	rows [][]*box
	// headerRows is the number of leading rows that came from <thead> /
	// table-header-group (for repeating headers across pages).
	headerRows int
	// sticky: print-scoped position:sticky (see sticky.go). Insets are scaled
	// points; cb* is filled at pagination time from the parent box.
	// stickyPort is the nearest overflow:auto|scroll|hidden|clip ancestor
	// (scrollport at offset 0); nil means page content box is the scrollport.
	sticky                         bool
	stickyID                       int
	stickyTop, stickyRight         float64
	stickyBottom, stickyLeft       float64
	stickyTopSet, stickyRightSet   bool
	stickyBottomSet, stickyLeftSet bool
	stickyPort                     *box
	cbX, cbY, cbW, cbH             float64
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
	// Ancestor transforms only (own transform does not change this box's CB).
	underXformCB := e.transformCBDepth > 0
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
	// Transformed ancestors establish a CB for fixed (treated as absolute).
	if b == nil && st.Position == "fixed" {
		if underXformCB {
			b = e.buildAbsolute(n, st, availW, x, y)
		} else {
			b = e.buildFixed(n, st, availW, x, y)
		}
	}
	if b == nil && st.Position == "absolute" {
		b = e.buildAbsolute(n, st, availW, x, y)
	}
	// Descendants of a transformed box see this as a containing block.
	if st.HasTransform {
		e.transformCBDepth++
		defer func() { e.transformCBDepth-- }()
	}
	if b == nil && (st.WritingMode == "vertical-rl" || st.WritingMode == "vertical-lr") {
		b = e.buildVerticalBlock(n, st, availW, x, y)
	}
	if b == nil && (st.Display == "flex" || st.Display == "inline-flex") {
		b = e.buildFlex(n, st, availW, x, y)
	}
	if b == nil && (st.Display == "grid" || st.Display == "inline-grid" || st.Display == "subgrid") {
		b = e.buildGrid(n, st, availW, x, y)
	}
	if b == nil && isMulticol(st) {
		b = e.buildMulticol(n, st, availW, x, y)
	}
	if b == nil && isTableDisplay(st.Display) {
		b = e.buildTable(n, st, availW, x, y)
	}
	if b == nil {
		b = e.buildBlock(n, st, availW, x, y)
	}
	if b != nil && st.Position == "relative" {
		e.applyRelativeOffset(b)
	}
	b.opStart, b.opEnd = start, len(e.ops)-1
	if b != nil && st.Position == "sticky" {
		e.tagSticky(b)
	}
	if b != nil && st.Position == "fixed" && !underXformCB {
		// Only viewport-fixed when not under a transformed ancestor CB.
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
		// Cyclic % honesty: indefinite containing block → treat as auto.
		if availW > 0 && availW < 1e12 {
			b.w = availW * st.WidthPercent / 100
		} else {
			definiteW = false
		}
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
	if st.MinWidthPercent >= 0 && availW > 0 && availW < 1e12 {
		mn := availW * st.MinWidthPercent / 100
		if b.w < mn {
			b.w = mn
		}
	} else if st.MinWidth > 0 && b.w < e.scalePt(st.MinWidth) {
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
	pop, enclose := e.pushBFCFloats(st, contentX, contentW)
	cy = e.flowChildren(b, n.Children, st, contentW, contentX, y, cy)
	if enclose && e.bfcFloats != nil {
		cy = e.bfcFloats.extentCy(y, cy)
	}
	pop()
	// padding-bottom is inside the border box (space above border-bottom /
	// letterhead rules — fixture-07/16).
	cy += e.scalePt(st.PaddingBottom)

	// list marker (outside the principal box content — in the marker area)
	if n.Name == "li" && b.firstBaseline > 0 {
		e.emitListMarker(n, st, contentX, b.firstBaseline)
	}

	if h, ok := resolveUsedHeight(st, -1, e); ok {
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

// resolveUsedHeight returns a definite border-box height when the style has a
// usable height. HeightPercent requires a definite containing-block height
// (cbH >= 0); otherwise the percentage is treated as auto (cyclic honesty).
func resolveUsedHeight(st ResolvedStyle, cbH float64, e *engine) (float64, bool) {
	if st.HeightPercent >= 0 {
		if cbH < 0 {
			return 0, false
		}
		h := cbH * st.HeightPercent / 100
		if st.BoxSizing != "border-box" {
			h += e.scalePt(st.PaddingTop) + e.scalePt(st.PaddingBottom) +
				e.scalePt(st.BorderTop.Width) + e.scalePt(st.BorderBottom.Width)
		}
		return h, true
	}
	if st.Height < 0 {
		return 0, false
	}
	h := e.scalePt(st.Height)
	if st.BoxSizing != "border-box" {
		h += e.scalePt(st.PaddingTop) + e.scalePt(st.PaddingBottom) +
			e.scalePt(st.BorderTop.Width) + e.scalePt(st.BorderBottom.Width)
	}
	return h, true
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

// buildInFlowDisplay builds flex/grid/multicol/table/block ignoring position.
func (e *engine) buildInFlowDisplay(n *html.Node, st ResolvedStyle, availW, x, y float64) *box {
	if st.Display == "flex" || st.Display == "inline-flex" {
		return e.buildFlex(n, st, availW, x, y)
	}
	if st.Display == "grid" || st.Display == "inline-grid" || st.Display == "subgrid" {
		return e.buildGrid(n, st, availW, x, y)
	}
	if isMulticol(st) {
		return e.buildMulticol(n, st, availW, x, y)
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
// Float enclosure (extentCy) is the caller's job when it owns a BFC.
func (e *engine) flowChildren(parent *box, children []*html.Node, st ResolvedStyle, contentW, contentX, y, cy float64) float64 {
	prevBottom := 0.0
	var local floatState
	floats := e.bfcFloats
	if floats == nil {
		// Defensive: callers should pushBFCFloats first. Keep a local state
		// so isolated measure passes (layoutCell) still work.
		local = newFloatState(contentX, contentW)
		floats = &local
	}
	var deferred []*html.Node
	// Absolute/fixed containing-block origin is the content edge at entry.
	// Do not use the post-flow cy or deferred boxes sit below in-flow siblings.
	absOriginY := y + cy
	absCBX, absCBW := contentX, contentW
	if st.HasTransform {
		// Transformed element: padding box is the CB for abs/fixed descendants.
		absCBX = contentX - e.scalePt(st.PaddingLeft)
		absOriginY = absOriginY - e.scalePt(st.PaddingTop)
		absCBW = contentW + e.scalePt(st.PaddingLeft) + e.scalePt(st.PaddingRight)
	}
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
			fb := e.placeFloat(n, cs, floats, contentW, contentX, y, cy)
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
				h := e.layoutInlineFloats(pb, run, contentW, contentX, y+cy, floats)
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
		// In-flow tables always clear below preceding floats (deterministic
		// report policy). Shrink-to-fit / squeeze-beside is unsupported.
		clear := cs.Clear
		if cs.Display == "table" {
			clear = "both"
		}
		cy = floats.clear(clear, y, cy)
		cy += collapseMargins(prevBottom, e.scalePt(cs.MarginTop))
		bx, bw := floats.exclusion(contentX, contentW, y, cy)
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
		ab := e.build(n, absCBW, absCBX, absOriginY)
		if ab != nil && parent != nil {
			parent.children = append(parent.children, ab)
		}
	}
	return cy
}

// pushBFCFloats installs a floatState for the current box. When the box
// establishes a BFC (or is the root), a fresh state is used and enclose is
// true so the caller should extend height with extentCy. Otherwise the
// parent BFC's state is reused and floats may protrude.
func (e *engine) pushBFCFloats(st ResolvedStyle, contentX, contentW float64) (pop func(), enclose bool) {
	prev := e.bfcFloats
	if prev == nil || establishesBFC(st) {
		fs := newFloatState(contentX, contentW)
		e.bfcFloats = &fs
		return func() { e.bfcFloats = prev }, true
	}
	return func() { e.bfcFloats = prev }, false
}

// emitListMarker paints the list marker in the marker area to the left of
// the content edge so it does not overlap the principal text.
func (e *engine) emitListMarker(n *html.Node, st ResolvedStyle, contentX, baseline float64) {
	typ := st.ListStyleType
	if typ == "" {
		typ = "disc"
	}
	if typ == "none" {
		return
	}
	size := e.scalePt(st.FontSize)
	face := e.faceFor(st)
	var text string
	switch typ {
	case "disc":
		text = "\u2022"
	case "circle":
		text = "\u25E6"
	case "square":
		text = "\u25AA"
	case "decimal", "decimal-leading-zero":
		text = strconv.Itoa(listItemIndex(n)) + "."
	case "lower-alpha", "lower-latin":
		text = alphaMarker(listItemIndex(n), false) + "."
	case "upper-alpha", "upper-latin":
		text = alphaMarker(listItemIndex(n), true) + "."
	case "lower-roman":
		text = romanMarker(listItemIndex(n), false) + "."
	case "upper-roman":
		text = romanMarker(listItemIndex(n), true) + "."
	default:
		text = "\u2022"
	}
	mw := 0.0
	if face != nil {
		for _, r := range text {
			mw += face.AdvanceInPoints(r, size)
		}
	}
	if mw <= 0 {
		mw = size * float64(len([]rune(text))) * 0.5
	}
	// Outside marker: sit in the padding/margin gutter left of contentX.
	gap := size * 0.35
	x := contentX - gap - mw
	if x < 0 {
		x = 0
	}
	e.add(Op{
		Kind: OpBullet, X: x, Y: baseline, Text: text, Font: face, Size: size,
		R: st.Color[0], G: st.Color[1], B: st.Color[2],
	})
}

// listItemIndex is the 1-based index among element siblings that are list items.
func listItemIndex(n *html.Node) int {
	if n == nil || n.Parent == nil {
		return 1
	}
	i := 0
	for _, c := range n.Parent.Children {
		if c.Type != html.ElementNode {
			continue
		}
		if !strings.EqualFold(c.Name, "li") {
			continue
		}
		i++
		if c == n {
			return i
		}
	}
	return 1
}

func alphaMarker(n int, upper bool) string {
	if n < 1 {
		n = 1
	}
	var chars []byte
	for n > 0 {
		n--
		ch := byte('a' + (n % 26))
		if upper {
			ch = byte('A' + (n % 26))
		}
		chars = append(chars, ch)
		n /= 26
	}
	for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
		chars[i], chars[j] = chars[j], chars[i]
	}
	return string(chars)
}

func romanMarker(n int, upper bool) string {
	if n < 1 {
		n = 1
	}
	vals := []int{1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1}
	syms := []string{"m", "cm", "d", "cd", "c", "xc", "l", "xl", "x", "ix", "v", "iv", "i"}
	var b strings.Builder
	for i, v := range vals {
		for n >= v {
			b.WriteString(syms[i])
			n -= v
		}
	}
	s := b.String()
	if upper {
		return strings.ToUpper(s)
	}
	return s
}

// placeFloat lays out n as a float:left|right box and records it in floats.
// Consecutive same-side floats pack horizontally when width remains;
// otherwise they stack below the previous float bottom.
func (e *engine) placeFloat(n *html.Node, cs ResolvedStyle, floats *floatState, contentW, contentX, y, cy float64) *box {
	avail := contentW
	if cs.Width < 0 && cs.WidthPercent < 0 {
		var intr float64
		if isSizeContainer(cs) {
			// Size containment: intrinsic inline size as-if-empty (padding+border
			// only) so used size does not depend on descendants.
			intr = e.scalePt(cs.PaddingLeft) + e.scalePt(cs.PaddingRight) +
				e.scalePt(cs.BorderLeft.Width) + e.scalePt(cs.BorderRight.Width) +
				e.scalePt(cs.MarginLeft) + e.scalePt(cs.MarginRight)
		} else {
			intr = e.measureCellContent(n, cs)
			intr += e.scalePt(cs.PaddingLeft) + e.scalePt(cs.PaddingRight) +
				e.scalePt(cs.BorderLeft.Width) + e.scalePt(cs.BorderRight.Width) +
				e.scalePt(cs.MarginLeft) + e.scalePt(cs.MarginRight)
		}
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

	oldMax := e.imgMaxW
	if cs.Width >= 0 {
		e.imgMaxW = e.scalePt(cs.Width)
	} else if cs.WidthPercent >= 0 && contentW > 0 {
		e.imgMaxW = contentW * cs.WidthPercent / 100
	} else if avail > 0 && avail < contentW {
		e.imgMaxW = avail
	}
	fb := e.build(n, avail, fx, fy)
	e.imgMaxW = oldMax
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
	ml := e.scalePt(cs.MarginLeft)
	mr := e.scalePt(cs.MarginRight)
	if cs.Float == "right" {
		wantX := contentX + contentW - fb.w - mr
		dx := wantX - fb.x
		fb.x = wantX
		e.shiftBoxOps(fb, dx, 0)
	}
	floats.place(cs.Float, fb, ml, mr)
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
			if png, pw, ph, err := svg.Rasterize(data, 1024); err == nil {
				b.imgSrc, b.imgData, b.imgJPEG, b.imgW, b.imgH = src, png, false, pw, ph
			} else if w, h, jpeg, ok := imageDims(data); ok {
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
	maxW := -1.0
	if st.MaxWidth >= 0 {
		maxW = e.scalePt(st.MaxWidth)
	}
	if st.MaxWidthPercent >= 0 {
		cb := e.imgMaxW
		if cb <= 0 {
			cb = e.opts.Width
		}
		if cb > 0 {
			pct := cb * st.MaxWidthPercent / 100
			if maxW < 0 || pct < maxW {
				maxW = pct
			}
		}
	}
	// imgMaxW caps auto-sized images inside floats/narrow BFCs. A definite
	// CSS width (wiki wordmark 8.75em) must win — otherwise header logos
	// collapse to a few points beside the globe icon.
	if st.Width < 0 && e.imgMaxW > 0 && (maxW < 0 || e.imgMaxW < maxW) {
		maxW = e.imgMaxW
	}
	if maxW >= 0 && b.w > maxW && b.w > 0 {
		b.h = b.h * maxW / b.w
		b.w = maxW
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

	// Occupancy grid for rowspan: occupied[r][c] counts remaining rows that
	// column c is covered by a prior rowspan (including the current row).
	nRows := len(rows)
	// Pass 1: assign each cell a column index honoring rowspan holes, and
	// discover nCols.
	type tcell struct {
		node         *html.Node
		row, col     int
		cSpan, rSpan int
	}
	var placed []tcell
	occupied := make([][]int, nRows) // per-row remaining coverage counts
	nCols := 0
	for ri, r := range rows {
		if occupied[ri] == nil {
			occupied[ri] = make([]int, nCols)
		}
		ci := 0
		for _, cellNode := range r {
			cs, rs := colSpan(cellNode), cellRowSpan(cellNode)
			if cs < 1 {
				cs = 1
			}
			if rs < 1 {
				rs = 1
			}
			for ci < len(occupied[ri]) && occupied[ri][ci] > 0 {
				ci++
			}
			for len(occupied[ri]) < ci+cs {
				occupied[ri] = append(occupied[ri], 0)
			}
			for k := 0; k < cs; k++ {
				occupied[ri][ci+k] = rs // covered for rs rows including this one
			}
			// Mark subsequent rows.
			for rr := 1; rr < rs && ri+rr < nRows; rr++ {
				for len(occupied[ri+rr]) < ci+cs {
					occupied[ri+rr] = append(occupied[ri+rr], 0)
				}
				for k := 0; k < cs; k++ {
					if occupied[ri+rr][ci+k] < rs-rr {
						occupied[ri+rr][ci+k] = rs - rr
					}
				}
			}
			placed = append(placed, tcell{node: cellNode, row: ri, col: ci, cSpan: cs, rSpan: rs})
			if end := ci + cs; end > nCols {
				nCols = end
			}
			ci += cs
		}
		if len(occupied[ri]) > nCols {
			nCols = len(occupied[ri])
		}
	}
	if nCols == 0 {
		return tb
	}
	// Normalize occupied rows to nCols.
	for ri := range occupied {
		for len(occupied[ri]) < nCols {
			occupied[ri] = append(occupied[ri], 0)
		}
	}

	// measure each cell's min/max-content width; colspan cells contribute their
	// content width evenly across the spanned columns (min floor per col).
	colW := make([]float64, nCols)   // preferred = max-content
	colMin := make([]float64, nCols) // shrink floor = min-content
	colPct := make([]float64, nCols) // >=0 means width:% of table; -1 = auto
	colAbs := make([]float64, nCols) // >=0 means absolute width pt; -1 = auto
	for i := range colPct {
		colPct[i] = -1
		colAbs[i] = -1
	}
	cellData := make([][]*box, nRows)
	for _, p := range placed {
		cell := e.buildCell(p.node, p.col, p.cSpan)
		cell.rowSpan = p.rSpan
		cellData[p.row] = append(cellData[p.row], cell)
		cs := e.styles[p.node]
		if p.cSpan == 1 {
			if cell.contentW > colW[p.col] {
				colW[p.col] = cell.contentW
			}
			if cell.contentMin > colMin[p.col] {
				colMin[p.col] = cell.contentMin
			}
			if cs.WidthPercent >= 0 && colPct[p.col] < 0 {
				colPct[p.col] = cs.WidthPercent
			}
			if cs.Width >= 0 && colAbs[p.col] < 0 {
				colAbs[p.col] = e.scalePt(cs.Width)
			}
		} else if p.cSpan > 1 {
			var sumMax, sumMin float64
			for k := 0; k < p.cSpan && p.col+k < nCols; k++ {
				sumMax += colW[p.col+k]
				sumMin += colMin[p.col+k]
			}
			if cell.contentW > sumMax {
				extra := (cell.contentW - sumMax) / float64(p.cSpan)
				for k := 0; k < p.cSpan && p.col+k < nCols; k++ {
					colW[p.col+k] += extra
				}
			}
			if cell.contentMin > sumMin {
				extra := (cell.contentMin - sumMin) / float64(p.cSpan)
				for k := 0; k < p.cSpan && p.col+k < nCols; k++ {
					colMin[p.col+k] += extra
				}
			}
		}
	}

	// table width
	// border-collapse: collapse suppresses the separate-border gap so colspan
	// header rows and body cells share edges instead of looking double-lined.
	spacing := e.scalePt(st.BorderSpacing)
	if st.BorderCollapse == "collapse" {
		spacing = 0
	}
	sumMax := 0.0
	sumMin := 0.0
	for i := range colW {
		sumMax += colW[i]
		sumMin += colMin[i]
	}
	chrome := spacing*float64(nCols+1) +
		e.scalePt(st.BorderLeft.Width) + e.scalePt(st.BorderRight.Width) +
		e.scalePt(st.PaddingLeft) + e.scalePt(st.PaddingRight)
	sumMax += chrome
	sumMin += chrome
	tableW := availW
	definiteTable := false
	if st.WidthPercent >= 0 {
		// width:% of the containing block (parent cell / block), not viewport
		tableW = availW * st.WidthPercent / 100
		definiteTable = true
	} else if st.Width >= 0 {
		tableW = e.scalePt(st.Width)
		if tableW > availW && availW > 0 {
			tableW = availW
		}
		definiteTable = true
	} else if sumMax < availW {
		// width:auto — shrink-wrap to max-content (not min-content).
		tableW = sumMax
	}

	// Apply column width:% / absolute widths when the table size is definite.
	hasColHint := false
	for i := range colPct {
		if colPct[i] >= 0 || colAbs[i] >= 0 {
			hasColHint = true
			break
		}
	}
	if definiteTable && hasColHint {
		inner := tableW - chrome
		if inner < 0 {
			inner = 0
		}
		used := 0.0
		autoMax := 0.0
		for i := range colW {
			switch {
			case colPct[i] >= 0:
				colW[i] = inner * colPct[i] / 100
				if colW[i] < colMin[i] {
					colW[i] = colMin[i]
				}
				used += colW[i]
			case colAbs[i] >= 0:
				colW[i] = colAbs[i]
				if colW[i] < colMin[i] {
					colW[i] = colMin[i]
				}
				used += colW[i]
			default:
				autoMax += colW[i]
			}
		}
		remain := inner - used
		if remain < 0 {
			remain = 0
		}
		if autoMax > 0 && remain > 0 {
			for i := range colW {
				if colPct[i] < 0 && colAbs[i] < 0 {
					colW[i] = remain * (colW[i] / autoMax)
					if colW[i] < colMin[i] {
						colW[i] = colMin[i]
					}
				}
			}
		} else if autoMax == 0 && remain > 0 {
			// All columns hinted — distribute leftover by % share, else evenly.
			pctTotal := 0.0
			for i := range colPct {
				if colPct[i] > 0 {
					pctTotal += colPct[i]
				}
			}
			for i := range colW {
				if pctTotal > 0 && colPct[i] > 0 {
					colW[i] += remain * (colPct[i] / pctTotal)
				} else {
					colW[i] += remain / float64(nCols)
				}
			}
		}
	} else if tableW > sumMax {
		extra := (tableW - sumMax) / float64(nCols)
		for i := range colW {
			colW[i] += extra
		}
	} else if tableW < sumMax {
		innerAvail := tableW - chrome
		if innerAvail < 0 {
			innerAvail = 0
		}
		innerMax := sumMax - chrome
		innerMin := sumMin - chrome
		if innerAvail >= innerMin && innerMax > innerMin {
			// Grow each column from min toward max proportional to free space.
			free := innerAvail - innerMin
			span := innerMax - innerMin
			for i := range colW {
				grow := colW[i] - colMin[i]
				if grow < 0 {
					grow = 0
				}
				colW[i] = colMin[i] + free*(grow/span)
			}
		} else if innerMax > 0 {
			scale := innerAvail / innerMax
			if scale < 0 {
				scale = 0
			}
			for i := range colW {
				colW[i] *= scale
			}
		}
	}
	tb.w = tableW
	cy := e.scalePt(st.PaddingTop) + e.scalePt(st.BorderTop.Width)
	rowHeights := make([]float64, nRows)
	rowTops := make([]float64, nRows)
	padL := e.scalePt(st.PaddingLeft) + e.scalePt(st.BorderLeft.Width)
	// Measure each cell at its final column width; row height from single-row
	// cells first. Rowspan cells enlarge the spanned rows afterward.
	for ri, cells := range cellData {
		rowTops[ri] = y + cy
		rowH := 0.0
		for _, cell := range cells {
			if cell.rowSpan < 1 {
				cell.rowSpan = 1
			}
			cellW := 0.0
			for k := 0; k < cell.span && cell.col+k < nCols; k++ {
				cellW += colW[cell.col+k]
			}
			cellW += spacing * float64(cell.span-1)
			cell.w = cellW
			cell.x = x + padL
			for c := 0; c < cell.col && c < nCols; c++ {
				cell.x += colW[c] + spacing
			}
			cell.y = rowTops[ri]
			e.measureCellHeight(cell, cellW)
			if cell.rowSpan == 1 && cell.contentH > rowH {
				rowH = cell.contentH
			}
			tb.children = append(tb.children, cell)
		}
		if rowH <= 0 {
			rowH = 1
		}
		rowHeights[ri] = rowH
		cy += rowH + spacing
	}
	// Grow rows so rowspan cells fit their content across the spanned band.
	for _, cell := range tb.children {
		if cell.rowSpan <= 1 {
			continue
		}
		start := -1
		for ri, cells := range cellData {
			for _, c := range cells {
				if c == cell {
					start = ri
					break
				}
			}
			if start >= 0 {
				break
			}
		}
		if start < 0 {
			continue
		}
		end := start + cell.rowSpan
		if end > nRows {
			end = nRows
		}
		sum := 0.0
		for ri := start; ri < end; ri++ {
			sum += rowHeights[ri]
			if ri+1 < end {
				sum += spacing
			}
		}
		if cell.contentH > sum {
			extra := (cell.contentH - sum) / float64(end-start)
			for ri := start; ri < end; ri++ {
				rowHeights[ri] += extra
			}
		}
	}
	// Recompute tops and assign final cell heights after rowspan growth.
	cy = e.scalePt(st.PaddingTop) + e.scalePt(st.BorderTop.Width)
	for ri := range rowHeights {
		rowTops[ri] = y + cy
		cy += rowHeights[ri] + spacing
	}
	for _, cell := range tb.children {
		start := -1
		for ri, cells := range cellData {
			for _, c := range cells {
				if c == cell {
					start = ri
					break
				}
			}
			if start >= 0 {
				break
			}
		}
		if start < 0 {
			continue
		}
		cell.y = rowTops[start]
		rs := cell.rowSpan
		if rs < 1 {
			rs = 1
		}
		end := start + rs
		if end > nRows {
			end = nRows
		}
		h := 0.0
		for ri := start; ri < end; ri++ {
			h += rowHeights[ri]
			if ri+1 < end {
				h += spacing
			}
		}
		cell.h = h
		cell.rowBoxH = rowHeights[start]
	}
	tb.rows = cellData
	tb.h = cy + e.scalePt(st.PaddingBottom) + e.scalePt(st.BorderBottom.Width)

	if st.BGColor[3] > 0 && e.opts.Background {
		e.add(Op{Kind: OpFillRect, X: x, Y: y, W: tb.w, H: tb.h,
			R: st.BGColor[0], G: st.BGColor[1], B: st.BGColor[2], Alpha: st.BGColor[3]})
	}
	e.emitBorders(st, x, y, tb.w, tb.h)

	collapse := st.BorderCollapse == "collapse"
	// emit cell content and boxes now that positions are final
	for _, cell := range tb.children {
		e.emitCell(cell, collapse)
	}
	if collapse {
		e.emitCollapsedGrid(tb, x, y, padL, spacing, colW, rowTops, rowHeights)
	}
	return tb
}

// emitCollapsedGrid strokes a single shared border grid for border-collapse
// tables. Per-cell borders leave gaps at rowspan holes and double strokes on
// shared edges; one grid from final geometry stays continuous.
func (e *engine) emitCollapsedGrid(tb *box, x, y, padL, spacing float64, colW, rowTops, rowHeights []float64) {
	if len(rowTops) == 0 || len(colW) == 0 {
		return
	}
	bw := 0.5
	var r, g, b float64
	for _, cell := range tb.children {
		st := cell.style
		sides := []struct {
			w     float64
			style string
			c     [3]float64
		}{
			{e.scalePt(st.BorderTop.Width), st.BorderTop.Style, st.BorderTop.Color},
			{e.scalePt(st.BorderLeft.Width), st.BorderLeft.Style, st.BorderLeft.Color},
		}
		for _, side := range sides {
			if side.w > 0 && side.style != "none" {
				bw = side.w
				r, g, b = side.c[0], side.c[1], side.c[2]
				break
			}
		}
		if bw != 0.5 || r+g+b > 0 {
			break
		}
	}
	nCols := len(colW)
	nRows := len(rowHeights)
	left := x + padL
	xs := make([]float64, nCols+1)
	xs[0] = left
	for i := 0; i < nCols; i++ {
		xs[i+1] = xs[i] + colW[i]
		if i+1 < nCols {
			xs[i+1] += spacing
		}
	}
	right := xs[nCols]
	ys := make([]float64, nRows+1)
	for i := 0; i < nRows; i++ {
		ys[i] = rowTops[i]
	}
	ys[nRows] = rowTops[nRows-1] + rowHeights[nRows-1]

	hline := func(x0, x1, yy float64) {
		e.add(Op{Kind: OpLine, X: x0, Y: yy, W: x1 - x0, H: 0, Width: bw, R: r, G: g, B: b})
	}
	vline := func(xx, y0, y1 float64) {
		e.add(Op{Kind: OpLine, X: xx, Y: y0, W: 0, H: y1 - y0, Width: bw, R: r, G: g, B: b})
	}

	// Horizontal rules — skip segments covered by a rowspan continuing through
	// this boundary.
	for ri := 0; ri <= nRows; ri++ {
		yy := ys[ri]
		for ci := 0; ci < nCols; ci++ {
			if ri > 0 && ri < nRows && rowspanCovers(tb, ri-1, ri, ci) {
				continue
			}
			hline(xs[ci], xs[ci+1], yy)
		}
	}
	// Vertical rules — skip segments covered by a colspan.
	for ci := 0; ci <= nCols; ci++ {
		xx := xs[ci]
		for ri := 0; ri < nRows; ri++ {
			if ci > 0 && ci < nCols && colspanCovers(tb, ri, ci-1, ci) {
				continue
			}
			vline(xx, ys[ri], ys[ri+1])
		}
	}
	_ = right
	_ = y
}

// rowspanCovers reports whether some cell occupies column ci across the
// boundary between row above and row below (so the horizontal rule is omitted).
func rowspanCovers(tb *box, above, below, ci int) bool {
	for _, cell := range tb.children {
		if cell.rowSpan <= 1 {
			continue
		}
		start := cellStartRow(tb, cell)
		if start < 0 {
			continue
		}
		if start <= above && start+cell.rowSpan > below &&
			cell.col <= ci && cell.col+cell.span > ci {
			return true
		}
	}
	return false
}

func colspanCovers(tb *box, ri, leftCol, rightCol int) bool {
	if ri < 0 || ri >= len(tb.rows) {
		return false
	}
	for _, cell := range tb.rows[ri] {
		if cell.span > 1 && cell.col <= leftCol && cell.col+cell.span > rightCol {
			return true
		}
	}
	// Rowspan continuation rows have no local cell — find covering cell.
	for _, cell := range tb.children {
		start := cellStartRow(tb, cell)
		if start < 0 {
			continue
		}
		rs := cell.rowSpan
		if rs < 1 {
			rs = 1
		}
		if start <= ri && start+rs > ri &&
			cell.span > 1 && cell.col <= leftCol && cell.col+cell.span > rightCol {
			return true
		}
	}
	return false
}

func cellStartRow(tb *box, cell *box) int {
	for ri, cells := range tb.rows {
		for _, c := range cells {
			if c == cell {
				return ri
			}
		}
	}
	return -1
}

// buildCell measures a table cell's min/max-content width (no ops emitted).
// Height is not final here: layoutCell must run again with the real column
// width after column sizing, or narrow max-content widths force false wraps
// and inflate row heights (empty bands under single-line cell text).
func (e *engine) buildCell(n *html.Node, col, span int) *box {
	st := e.styles[n]
	b := &box{node: n, style: st, kind: "cell", col: col, span: span}
	b.contentMin, b.contentW = e.measureCellMinMax(n, st)
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
// skipBorders is set for border-collapse tables whose grid is stroked once
// by the parent table (avoids doubled/gapped per-cell edges).
func (e *engine) emitCell(b *box, skipBorders bool) {
	st := b.style
	start := len(e.ops)
	if e.opts.Background {
		if r, g, bl, a, ok := e.cellBG(b); ok {
			e.add(Op{Kind: OpFillRect, X: b.x, Y: b.y, W: b.w, H: b.h,
				R: r, G: g, B: bl, Alpha: a})
		}
	}
	if !skipBorders {
		e.emitBorders(st, b.x, b.y, b.w, b.h)
	}
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
	// flowChildren advances cy; cell content is rooted at absolute canvas y
	// (pass y=0, contentX=cx, cy=content top) so floats pack inside the cell
	// BFC. Pass the cell as parent so float/block children attach for tests.
	oldMax := e.imgMaxW
	if contentW > 0 {
		e.imgMaxW = contentW
	}
	pop, enclose := e.pushBFCFloats(st, cx, contentW)
	_ = e.flowChildren(b, b.node.Children, st, contentW, cx, 0, cy)
	if enclose && e.bfcFloats != nil {
		// Cell border box already sized; floats are clipped to the cell BFC.
		_ = e.bfcFloats.extentCy(0, cy)
	}
	pop()
	e.imgMaxW = oldMax
	b.opStart, b.opEnd = start, len(e.ops)-1
}

// measureCellContent returns the max-content border-box width of the cell
// (longest unwrapped line, not longest word). Using min-content here made
// auto tables shrink-wrap to a rivulet of columns and inflate row heights
// via forced wraps (wiki filmography / any dense multi-column table).
func (e *engine) measureCellContent(n *html.Node, st ResolvedStyle) float64 {
	minW, maxW := e.measureCellMinMax(n, st)
	_ = minW
	return maxW
}

// measureCellMinMax returns min-content and max-content border-box widths.
// min-content ≈ longest unbreakable word; max-content ≈ widest line if soft
// wraps are not taken (hard breaks from <br>/blocks still split lines).
func (e *engine) measureCellMinMax(n *html.Node, st ResolvedStyle) (minW, maxW float64) {
	var lineW, longestWord float64
	flushLine := func() {
		if lineW > maxW {
			maxW = lineW
		}
		lineW = 0
	}
	var walk func(n *html.Node, fs float64, nowrap bool)
	walk = func(n *html.Node, fs float64, nowrap bool) {
		switch n.Type {
		case html.TextNode:
			text := n.Text
			if !nowrap {
				// Collapse runs of whitespace to a single space for measure,
				// matching normal white-space:normal inline layout.
				fields := strings.Fields(text)
				if len(fields) == 0 {
					return
				}
				// Leading space if original had leading WS and line already started.
				if lineW > 0 && len(text) > 0 && isHTMLSpace(text[0]) {
					lineW += e.measureText(" ", fs*e.scale)
				}
				for i, word := range fields {
					if i > 0 {
						lineW += e.measureText(" ", fs*e.scale)
					}
					w := e.measureText(word, fs*e.scale)
					if w > longestWord {
						longestWord = w
					}
					lineW += w
				}
				return
			}
			w := e.measureText(text, fs*e.scale)
			if w > longestWord {
				longestWord = w
			}
			lineW += w
		case html.ElementNode:
			cs := e.styles[n]
			if cs.Display == "none" {
				return
			}
			fs = cs.FontSize
			if n.Name == "br" {
				flushLine()
				return
			}
			// Block-level in-cell boxes start a new line (simplified).
			blockish := cs.Display == "block" || cs.Display == "table" ||
				cs.Display == "list-item" || cs.Display == "flex" || cs.Display == "grid"
			if blockish {
				flushLine()
			}
			childNowrap := nowrap || cs.WhiteSpace == "nowrap" || cs.WhiteSpace == "pre"
			for _, c := range n.Children {
				walk(c, fs, childNowrap)
			}
			if blockish {
				flushLine()
			}
		}
	}
	walk(n, st.FontSize, st.WhiteSpace == "nowrap" || st.WhiteSpace == "pre")
	flushLine()
	chrome := e.scalePt(st.PaddingLeft) + e.scalePt(st.PaddingRight) +
		e.scalePt(st.BorderLeft.Width) + e.scalePt(st.BorderRight.Width)
	minW = longestWord + chrome
	maxW += chrome
	if maxW < minW {
		maxW = minW
	}
	return minW, maxW
}

func isHTMLSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f'
}

// layoutCell measures the height of a cell's content (no ops emitted).
func (e *engine) layoutCell(n *html.Node, st ResolvedStyle, width float64) float64 {
	contentW := width - e.scalePt(st.PaddingLeft) - e.scalePt(st.PaddingRight) - e.scalePt(st.BorderLeft.Width) - e.scalePt(st.BorderRight.Width)
	if contentW < 0 {
		contentW = 0
	}
	cy := e.scalePt(st.PaddingTop) + e.scalePt(st.BorderTop.Width)
	pop, enclose := e.pushBFCFloats(st, 0, contentW)
	cy = e.flowChildren(nil, n.Children, st, contentW, 0, 0, cy)
	if enclose && e.bfcFloats != nil {
		cy = e.bfcFloats.extentCy(0, cy)
	}
	pop()
	return cy + e.scalePt(st.PaddingBottom) + e.scalePt(st.BorderBottom.Width)
}

func colSpan(n *html.Node) int {
	if v, err := strconv.Atoi(strings.TrimSpace(n.Attribute("colspan"))); err == nil && v > 1 {
		return v
	}
	return 1
}

func cellRowSpan(n *html.Node) int {
	if v, err := strconv.Atoi(strings.TrimSpace(n.Attribute("rowspan"))); err == nil && v > 1 {
		return v
	}
	return 1
}
