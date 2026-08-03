// Package layout turns the parsed HTML tree plus resolved styles into a
// display list: absolute-positioned drawing operations in a continuous
// canvas (y grows downward from the top of the page content area). Painting
// into a pdf.Document is done by Paint (paint.go).
//
// Phase-04 scope: block and inline flow, margin collapsing between siblings,
// tables (separate borders, colspan), images, lists, text wrapping with the
// embedded Liberation Sans font. Floats, absolute positioning, flex and grid
// are out of scope; their properties degrade to in-flow block layout.
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
}

type engine struct {
	opts   Options
	font   *pdf.Font // default/regular face (metrics fallback)
	faces  *pdf.FaceSet
	styles map[*html.Node]ResolvedStyle
	ops    []Op
	noEmit bool // measurement mode: compute geometry without emitting ops
	height float64
	scale  float64 // zoom factor applied to style lengths (>= 1)
}

// faceFor selects the TrueType face for a resolved style (bold/italic).
func (e *engine) faceFor(st ResolvedStyle) *pdf.Font {
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
		e.ops = append(e.ops, op)
	}
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
	font := opts.Font
	if font == nil {
		font = faces.Regular
	}
	e := &engine{
		opts:   opts,
		font:   font,
		faces:  faces,
		styles: resolveStyles(root, opts.Sheets, opts.Media, opts.Width, opts.Height),
		scale:  zoomScale(opts.Zoom),
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
	start := len(e.ops)
	var b *box
	switch n.Name {
	case "img":
		b = e.buildImage(n, st, x, y)
	case "hr":
		b = e.buildHR(n, st, availW, x, y)
	}
	if b == nil && isTableDisplay(st.Display) {
		b = e.buildTable(n, st, availW, x, y)
	}
	if b == nil {
		b = e.buildBlock(n, st, availW, x, y)
	}
	b.opStart, b.opEnd = start, len(e.ops)-1
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
	if st.Width >= 0 {
		b.w = e.scalePt(st.Width)
	}
	if st.MinWidth > 0 && b.w < e.scalePt(st.MinWidth) {
		b.w = e.scalePt(st.MinWidth)
	}
	if st.MaxWidth >= 0 && b.w > e.scalePt(st.MaxWidth) {
		b.w = e.scalePt(st.MaxWidth)
	}
	// Horizontal margin: auto centers (or pushes) a definite-width box.
	if st.Width >= 0 && (st.MarginLeftAuto || st.MarginRightAuto) {
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

	var blocks, inlines []*html.Node
	e.partition(n.Children, &blocks, &inlines)

	// Content ops are recorded first so we know the box height; background
	// and borders are then inserted *before* those ops so paint order is
	// bg → borders → children (otherwise fills cover text).
	contentStart := len(e.ops)

	cy := e.scalePt(st.PaddingTop) + e.scalePt(st.BorderTop.Width)
	prevBottom := 0.0

	for _, cn := range blocks {
		cs := e.styles[cn]
		cy += collapseMargins(prevBottom, e.scalePt(cs.MarginTop))
		cb := e.build(cn, contentW, contentX, y+cy)
		if cb == nil {
			prevBottom = 0
			continue
		}
		cy += cb.h
		prevBottom = e.scalePt(cs.MarginBottom)
		b.children = append(b.children, cb)
		if e.opts.DebugBoxes {
			e.add(Op{Kind: OpStrokeRect, X: cb.x, Y: cb.y, W: cb.w, H: cb.h, R: 1, G: 0, B: 0})
		}
	}

	if len(inlines) > 0 {
		cy += e.layoutInline(b, inlines, contentW, contentX, y+cy)
	}

	// list marker
	if n.Name == "li" && b.firstBaseline > 0 {
		e.add(Op{Kind: OpBullet, X: contentX + 4, Y: b.firstBaseline, Text: "\u2022", Font: e.font, Size: e.scalePt(st.FontSize), R: st.Color[0], G: st.Color[1], B: st.Color[2]})
	}

	if st.Height >= 0 && cy < e.scalePt(st.Height) {
		cy = e.scalePt(st.Height)
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
		if cs.Display == "inline" || c.Name == "img" || cs.Float != "none" {
			*inlines = append(*inlines, c)
			continue
		}
		*blocks = append(*blocks, c)
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
	// flatten row groups into rows
	var rows [][]*html.Node
	var collect func(n *html.Node)
	collect = func(n *html.Node) {
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
			case strings.HasSuffix(cs.Display, "row-group"):
				collect(c)
			}
		}
	}
	collect(n)

	tb := &box{node: n, style: st, kind: "table", x: x, y: y}
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
	spacing := e.scalePt(st.BorderSpacing)
	sum := 0.0
	for _, w := range colW {
		sum += w
	}
	sum += spacing * float64(nCols+1)
	sum += e.scalePt(st.BorderLeft.Width) + e.scalePt(st.BorderRight.Width) + e.scalePt(st.PaddingLeft) + e.scalePt(st.PaddingRight)
	tableW := availW
	if st.Width >= 0 {
		tableW = e.scalePt(st.Width)
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

// buildCell measures a table cell (no ops emitted).
func (e *engine) buildCell(n *html.Node, col, span int) *box {
	st := e.styles[n]
	b := &box{node: n, style: st, kind: "cell", col: col, span: span}
	b.contentW = e.measureCellContent(n, st)
	e.noEmit = true
	b.contentH = e.layoutCell(n, st, b.contentW)
	e.noEmit = false
	return b
}

// emitCell paints a placed cell's background, borders and content.
func (e *engine) emitCell(b *box) {
	st := b.style
	start := len(e.ops)
	if st.BGColor[3] > 0 && e.opts.Background {
		e.add(Op{Kind: OpFillRect, X: b.x, Y: b.y, W: b.w, H: b.h,
			R: st.BGColor[0], G: st.BGColor[1], B: st.BGColor[2], Alpha: st.BGColor[3]})
	}
	e.emitBorders(st, b.x, b.y, b.w, b.h)
	contentW := b.w - e.scalePt(st.PaddingLeft) - e.scalePt(st.PaddingRight) - e.scalePt(st.BorderLeft.Width) - e.scalePt(st.BorderRight.Width)
	if contentW < 0 {
		contentW = 0
	}
	cx := b.x + e.scalePt(st.PaddingLeft) + e.scalePt(st.BorderLeft.Width)
	cy := b.y + e.scalePt(st.PaddingTop) + e.scalePt(st.BorderTop.Width)
	var blocks, inlines []*html.Node
	e.partition(b.node.Children, &blocks, &inlines)
	prevBottom := 0.0
	for _, cn := range blocks {
		cs := e.styles[cn]
		cy += collapseMargins(prevBottom, e.scalePt(cs.MarginTop))
		cb := e.build(cn, contentW, cx, cy)
		if cb == nil {
			prevBottom = 0
			continue
		}
		cy += cb.h + e.scalePt(cs.MarginBottom)
		prevBottom = e.scalePt(cs.MarginBottom)
	}
	if len(inlines) > 0 {
		pb := &box{style: st, firstBaseline: 0}
		e.layoutInline(pb, inlines, contentW, cx, cy)
	}
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
	var blocks, inlines []*html.Node
	e.partition(n.Children, &blocks, &inlines)
	prevBottom := 0.0
	for _, cn := range blocks {
		cs := e.styles[cn]
		cy += collapseMargins(prevBottom, e.scalePt(cs.MarginTop))
		cb := e.build(cn, contentW, 0, cy)
		if cb == nil {
			prevBottom = 0
			continue
		}
		cy += cb.h + e.scalePt(cs.MarginBottom)
		prevBottom = e.scalePt(cs.MarginBottom)
	}
	if len(inlines) > 0 {
		pb := &box{style: st}
		cy += e.layoutInline(pb, inlines, contentW, 0, cy)
	}
	return cy + e.scalePt(st.PaddingBottom) + e.scalePt(st.BorderBottom.Width)
}

func colSpan(n *html.Node) int {
	if v, err := strconv.Atoi(strings.TrimSpace(n.Attribute("colspan"))); err == nil && v > 1 {
		return v
	}
	return 1
}
