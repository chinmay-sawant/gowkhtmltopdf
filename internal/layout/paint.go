package layout

import (
	"gowkhtmltopdf/internal/pdf"
)

// PaintOptions describes the destination page geometry, in points.
type PaintOptions struct {
	PageWidth    float64
	PageHeight   float64
	MarginTop    float64
	MarginBottom float64
	MarginLeft   float64
	MarginRight  float64
}

// Paint paginates the display list across pages and paints it into doc.
//
// Pagination is box-aware (phase 05): ops are placed on the page containing
// their top edge; rect-type ops that cross a page boundary are split at the
// boundary; text, images and links move wholly (text ops are already
// line-level). Page-break policies are honored: page-break-before/after:
// always, page-break-inside: avoid (best-effort, box must fit one page), and
// table rows never split.
//
// After pagination Paint fills res.Pages (page → op indices) and res.Locations
// (element boxes in document order with their page and canvas rect).
func Paint(doc *pdf.Document, res *Result, opts PaintOptions) error {
	if doc == nil || res == nil {
		return nil
	}
	contentH := opts.PageHeight - opts.MarginTop - opts.MarginBottom
	if contentH <= 0 {
		contentH = opts.PageHeight
	}
	if len(res.Ops) == 0 {
		doc.AddPage(opts.PageWidth, opts.PageHeight)
		populateLocations(res, contentH, nil)
		return nil
	}

	opPage := paginateOps(res, contentH)

	// assemble final page lists, splitting rect ops at page boundaries
	res.Pages = nil
	perPage := map[int][]int{}
	for i := range res.Ops {
		op := &res.Ops[i]
		p := opPage[i]
		if isSplittable(op) {
			boundary := float64(p+1) * contentH
			if op.Y+op.H > boundary+1e-9 {
				rest := len(res.Ops)
				op2 := *op
				op2.Y = boundary
				op2.H = op.Y + op.H - boundary
				op.H = boundary - op.Y
				res.Ops = append(res.Ops, op2)
				perPage[p] = append(perPage[p], i)
				perPage[p+1] = append(perPage[p+1], rest)
				continue
			}
		}
		perPage[p] = append(perPage[p], i)
	}
	maxP := 0
	for p := range perPage {
		if p > maxP {
			maxP = p
		}
	}
	res.Pages = make([][]int, maxP+1)
	for p := 0; p <= maxP; p++ {
		res.Pages[p] = perPage[p]
	}
	populateLocations(res, contentH, opPage)

	for pageIdx, idxs := range res.Pages {
		p := doc.AddPage(opts.PageWidth, opts.PageHeight)
		c := p.Content()
		fontNames := map[*pdf.Font]string{}
		nextFont := 0
		resName := func(f *pdf.Font) string {
			if f == nil {
				return "F0"
			}
			if n, ok := fontNames[f]; ok {
				return n
			}
			n := "F" + itoa(nextFont)
			nextFont++
			fontNames[f] = n
			c.UseEmbeddedFont(n, f)
			return n
		}
		// Unique image resource names per page: reusing "I0" for every
		// image made later images overwrite earlier ones (fixture-20).
		nextImg := 0
		for _, idx := range idxs {
			op := &res.Ops[idx]
			switch op.Kind {
			case OpFillRect:
				drawFill(c, op, pageIdx, contentH, opts, p.Height())
			case OpStrokeRect:
				drawStroke(c, op, pageIdx, contentH, opts, p.Height())
			case OpLine:
				drawLine(c, op, pageIdx, contentH, opts, p.Height())
			case OpText, OpBullet:
				drawText(c, op, pageIdx, contentH, opts, p.Height(), resName(op.Font))
			case OpImage:
				name := "I" + itoa(nextImg)
				nextImg++
				drawImage(p, c, op, pageIdx, contentH, opts, name)
			case OpLinkURI:
				drawLink(p, op, pageIdx, contentH, opts)
			}
		}
	}
	return nil
}

func isSplittable(op *Op) bool {
	return op.Kind == OpFillRect || op.Kind == OpStrokeRect || op.Kind == OpLine
}

// paginateOps assigns every op a page. Crossing text/image/link ops snap to
// the next page boundary; then page-break policies are applied as canvas-Y
// shifts (moving content down with its flow); finally pages derive from the
// final Y positions. Rect-type ops crossing a boundary are split by Paint.
func paginateOps(res *Result, contentH float64) []int {
	ops := res.Ops
	for i := range ops {
		op := &ops[i]
		switch op.Kind {
		case OpText, OpBullet, OpImage, OpLinkURI:
			opH := op.H
			if op.Kind == OpText || op.Kind == OpBullet {
				opH = op.Size * 1.2
			}
			page := int(op.Y / contentH)
			if op.Y+opH > float64(page+1)*contentH {
				op.Y = float64(page+1) * contentH
			}
		}
	}
	for iter := 0; iter < 10; iter++ {
		changed := avoidInside(res, contentH)
		if beforeAlways(res, contentH) {
			changed = true
		}
		if afterBreaks(res, contentH) {
			changed = true
		}
		if rowsIntact(res, contentH) {
			changed = true
		}
		if keepHeadingWithNext(res, contentH) {
			changed = true
		}
		if orphansWidows(res, contentH) {
			changed = true
		}
		if !changed {
			break
		}
	}
	// After flow has settled, clone <thead> onto continuation pages.
	repeatTableHeaders(res, contentH)
	opPage := make([]int, len(res.Ops))
	for i := range res.Ops {
		opPage[i] = int(res.Ops[i].Y / contentH)
	}
	return opPage
}

// shiftFlowY moves the ops of the target range [from,to] - plus every op
// strictly below fromY - down by dy canvas points. Ops of earlier boxes that
// touch fromY exactly (collapsed margins) are left alone so the page-break
// fixpoint converges instead of dragging boundary ops along each iteration.
// Box.y is kept in sync for boxes whose top moved.
func shiftFlowY(res *Result, from, to int, fromY, dy float64) {
	for i := from; i <= to; i++ {
		res.Ops[i].Y += dy
	}
	for i := range res.Ops {
		if i < from || i > to {
			if res.Ops[i].Y > fromY {
				res.Ops[i].Y += dy
			}
		}
	}
	if res.root == nil {
		return
	}
	var walk func(b *box)
	walk = func(b *box) {
		if b.y > fromY || (b.y == fromY && b.opStart >= from && b.opEnd <= to) {
			b.y += dy
		}
		for _, c := range b.children {
			walk(c)
		}
	}
	walk(res.root)
}

// avoidInside walks post-order and moves page-break-inside: avoid boxes wholly
// to the next page when they span multiple pages but fit one content height.
func avoidInside(res *Result, contentH float64) bool {
	var walk func(b *box) bool
	walk = func(b *box) bool {
		changed := false
		for _, c := range b.children {
			if walk(c) {
				changed = true
			}
		}
		if b.style.PageBreakInside == "avoid" && b.h > 0 {
			lo := int(b.y / contentH)
			hi := int((b.y + b.h) / contentH)
			if hi > lo && b.h <= contentH+0.01 {
				shiftFlowY(res, b.opStart, b.opEnd, b.y, float64(hi)*contentH-b.y)
				changed = true
			}
		}
		return changed
	}
	return walk(res.root)
}

// beforeAlways walks pre-order and moves page-break-before: always boxes to a
// fresh page after everything preceding them.
func beforeAlways(res *Result, contentH float64) bool {
	var walk func(b *box) bool
	walk = func(b *box) bool {
		if b.style.PageBreakBefore == "always" {
			lastBefore := 0.0
			for i := 0; i < b.opStart; i++ {
				if res.Ops[i].Y > lastBefore {
					lastBefore = res.Ops[i].Y
				}
			}
			loPage := int(b.y / contentH)
			lastPage := int(lastBefore / contentH)
			if loPage <= lastPage {
				dy := float64(lastPage+1)*contentH - b.y
				if dy > 0 {
					shiftFlowY(res, b.opStart, b.opEnd, b.y, dy)
					return true
				}
			}
		}
		changed := false
		for _, c := range b.children {
			if walk(c) {
				changed = true
			}
		}
		return changed
	}
	return walk(res.root)
}

// afterBreaks walks in document order and applies page-break-after:
// always|avoid to the box that follows each box in flow.
func afterBreaks(res *Result, contentH float64) bool {
	var boxes []*box
	var flatten func(b *box)
	flatten = func(b *box) {
		boxes = append(boxes, b)
		for _, c := range b.children {
			flatten(c)
		}
	}
	if res.root != nil {
		flatten(res.root)
	}
	changed := false
	for i, b := range boxes {
		var next *box
		for j := i + 1; j < len(boxes); j++ {
			if boxes[j].opStart <= boxes[j].opEnd {
				next = boxes[j]
				break
			}
		}
		if next == nil || b.opStart > b.opEnd {
			continue
		}
		lastY := res.Ops[b.opStart].Y
		for k := b.opStart + 1; k <= b.opEnd; k++ {
			if res.Ops[k].Y > lastY {
				lastY = res.Ops[k].Y
			}
		}
		lastPage := int(lastY / contentH)
		switch {
		case b.style.PageBreakAfter == "always":
			dy := float64(lastPage+1)*contentH - next.y
			if dy > 0 {
				shiftFlowY(res, next.opStart, next.opEnd, next.y, dy)
				changed = true
			}
		case b.style.PageBreakAfter == "avoid":
			remaining := float64(lastPage+1)*contentH - lastY
			if next.h <= remaining {
				dy := lastY - next.y
				if dy < -0.001 {
					shiftFlowY(res, next.opStart, next.opEnd, next.y, dy)
					changed = true
				}
			}
		}
	}
	return changed
}

// rowsIntact keeps each table row on a single page: a row spanning multiple
// pages moves wholly to the next.
func rowsIntact(res *Result, contentH float64) bool {
	ops := res.Ops
	var walk func(b *box) bool
	walk = func(b *box) bool {
		changed := false
		for _, c := range b.children {
			if walk(c) {
				changed = true
			}
		}
		for _, row := range b.rows {
			if len(row) == 0 {
				continue
			}
			first, last := -1, -1
			for _, cell := range row {
				if cell.opStart <= cell.opEnd {
					if first < 0 {
						first = cell.opStart
					}
					if cell.opEnd > last {
						last = cell.opEnd
					}
				}
			}
			if first < 0 {
				continue
			}
			rowTop, rowBottom := ops[first].Y, ops[first].Y
			for k := first + 1; k <= last; k++ {
				if ops[k].Y < rowTop {
					rowTop = ops[k].Y
				}
				if ops[k].Y > rowBottom {
					rowBottom = ops[k].Y
				}
			}
			lo, hi := int(rowTop/contentH), int(rowBottom/contentH)
			if hi > lo {
				shiftFlowY(res, first, last, rowTop, float64(hi)*contentH-rowTop)
				changed = true
			}
		}
		return changed
	}
	return walk(res.root)
}

// keepHeadingWithNext moves a heading to the next page when it would sit alone
// near the bottom (less than ~2 line-heights of room for following content).
func keepHeadingWithNext(res *Result, contentH float64) bool {
	if res.root == nil {
		return false
	}
	var boxes []*box
	var flatten func(b *box)
	flatten = func(b *box) {
		boxes = append(boxes, b)
		for _, c := range b.children {
			flatten(c)
		}
	}
	flatten(res.root)
	changed := false
	for i, b := range boxes {
		if b.node == nil || !isHeadingName(b.node.Name) || b.opStart > b.opEnd {
			continue
		}
		page := int(b.y / contentH)
		room := float64(page+1)*contentH - (b.y + b.h)
		if room >= 24 { // ~2 lines at 12pt
			continue
		}
		// Find next flow sibling with ops.
		var next *box
		for j := i + 1; j < len(boxes); j++ {
			if boxes[j].opStart <= boxes[j].opEnd && boxes[j].y >= b.y {
				next = boxes[j]
				break
			}
		}
		if next == nil {
			continue
		}
		nextPage := int(next.y / contentH)
		if nextPage > page {
			continue // already separated by a break
		}
		dy := float64(page+1)*contentH - b.y
		if dy > 0 {
			shiftFlowY(res, b.opStart, b.opEnd, b.y, dy)
			changed = true
		}
	}
	return changed
}

func isHeadingName(name string) bool {
	switch name {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return true
	}
	return false
}

// orphansWidows keeps a short block from leaving a single line stranded at the
// bottom of a page when the whole block fits on the next page.
func orphansWidows(res *Result, contentH float64) bool {
	if res.root == nil {
		return false
	}
	changed := false
	var walk func(b *box)
	walk = func(b *box) {
		for _, c := range b.children {
			walk(c)
		}
		if b.kind != "block" || b.h <= 0 || b.opStart > b.opEnd {
			return
		}
		// Only short paragraphs (roughly 2–4 line boxes).
		if b.h < 14 || b.h > 60 {
			return
		}
		lo := int(b.y / contentH)
		hi := int((b.y + b.h) / contentH)
		if hi <= lo {
			return
		}
		// Straddles a boundary; if it fits entirely on the next page, move it.
		if b.h <= contentH {
			dy := float64(hi)*contentH - b.y
			if dy > 0 {
				shiftFlowY(res, b.opStart, b.opEnd, b.y, dy)
				changed = true
			}
		}
	}
	walk(res.root)
	return changed
}

// repeatTableHeaders clones thead row ops onto every page that continues a
// multi-page table body, shifting body content down by the header height.
// Nested tables: each table repeats only its own thead.
func repeatTableHeaders(res *Result, contentH float64) {
	if res.root == nil || contentH <= 0 {
		return
	}
	var tables []*box
	var walk func(b *box)
	walk = func(b *box) {
		if b.kind == "table" && b.headerRows > 0 && b.headerRows < len(b.rows) {
			tables = append(tables, b)
		}
		for _, c := range b.children {
			walk(c)
		}
	}
	walk(res.root)

	for _, tb := range tables {
		nHdr := tb.headerRows
		if nHdr > len(tb.rows) {
			nHdr = len(tb.rows)
		}
		hdrFirst, hdrLast, hdrTop, hdrH := rowSpan(tb.rows[:nHdr], res)
		if hdrFirst < 0 || hdrH <= 0 {
			continue
		}
		firstPage := int(tb.y / contentH)
		pages := map[int]bool{}
		for _, row := range tb.rows[nHdr:] {
			top, _ := rowYBounds(row, res)
			if top < 0 {
				continue
			}
			pages[int(top/contentH)] = true
		}
		for page := range pages {
			if page <= firstPage {
				continue
			}
			pageTop := float64(page) * contentH
			bodyTop := -1.0
			shiftFrom, shiftTo := -1, -1
			for _, row := range tb.rows[nHdr:] {
				f, l := rowOpRange(row)
				if f < 0 {
					continue
				}
				top, _ := rowYBounds(row, res)
				if int(top/contentH) < page {
					continue
				}
				if bodyTop < 0 || top < bodyTop {
					bodyTop = top
				}
				if shiftFrom < 0 || f < shiftFrom {
					shiftFrom = f
				}
				if l > shiftTo {
					shiftTo = l
				}
			}
			if shiftFrom >= 0 && bodyTop >= 0 && bodyTop < pageTop+hdrH-0.5 {
				dy := pageTop + hdrH - bodyTop
				if dy > 0 {
					shiftFlowY(res, shiftFrom, shiftTo, bodyTop-0.01, dy)
				}
			}
			baseY := hdrTop
			for k := hdrFirst; k <= hdrLast && k < len(res.Ops); k++ {
				op := res.Ops[k]
				op.Y = pageTop + (op.Y - baseY)
				res.Ops = append(res.Ops, op)
			}
		}
	}
}

func rowOpRange(row []*box) (first, last int) {
	first, last = -1, -1
	for _, cell := range row {
		if cell.opStart <= cell.opEnd {
			if first < 0 || cell.opStart < first {
				first = cell.opStart
			}
			if cell.opEnd > last {
				last = cell.opEnd
			}
		}
	}
	return first, last
}

func rowYBounds(row []*box, res *Result) (top, bottom float64) {
	first, last := rowOpRange(row)
	if first < 0 || first >= len(res.Ops) {
		return -1, -1
	}
	top, bottom = res.Ops[first].Y, res.Ops[first].Y
	for k := first; k <= last && k < len(res.Ops); k++ {
		y := res.Ops[k].Y
		h := res.Ops[k].H
		if res.Ops[k].Kind == OpText || res.Ops[k].Kind == OpBullet {
			h = res.Ops[k].Size * 1.2
		}
		if y < top {
			top = y
		}
		if y+h > bottom {
			bottom = y + h
		}
	}
	for _, cell := range row {
		if cell.h > 0 && cell.y+cell.h > bottom {
			bottom = cell.y + cell.h
		}
		if cell.y < top && cell.y > 0 {
			top = cell.y
		}
	}
	return top, bottom
}

func rowSpan(rows [][]*box, res *Result) (first, last int, top, height float64) {
	first, last = -1, -1
	top, bottom := 0.0, 0.0
	set := false
	for _, row := range rows {
		f, l := rowOpRange(row)
		if f < 0 {
			continue
		}
		if first < 0 || f < first {
			first = f
		}
		if l > last {
			last = l
		}
		rt, rb := rowYBounds(row, res)
		if rt < 0 {
			continue
		}
		if !set || rt < top {
			top = rt
		}
		if !set || rb > bottom {
			bottom = rb
		}
		set = true
	}
	if first < 0 || !set {
		return -1, -1, 0, 0
	}
	height = bottom - top
	if height < 1 {
		height = 0
		for _, row := range rows {
			rowH := 0.0
			for _, cell := range row {
				if cell.h > rowH {
					rowH = cell.h
				}
			}
			height += rowH
		}
	}
	return first, last, top, height
}

// populateLocations records the canvas rect and page of every element box in
// document order. A box's page is the page of its first op; boxes without ops
// (or before an op→page map exists) fall back to the page of their y position.
func populateLocations(res *Result, contentH float64, opPage []int) {
	res.Locations = nil
	if res.root == nil {
		return
	}
	var walk func(b *box)
	walk = func(b *box) {
		if b.node != nil {
			page := -1
			if b.opStart <= b.opEnd && b.opStart < len(opPage) {
				page = opPage[b.opStart]
			}
			if page < 0 {
				page = int(b.y / contentH)
				if page < 0 {
					page = 0
				}
			}
			res.Locations = append(res.Locations, ElementLocation{
				Node: b.node, Page: page, X: b.x, Y: b.y, W: b.w, H: b.h,
			})
		}
		for _, c := range b.children {
			walk(c)
		}
	}
	walk(res.root)
}

// canvasToPDF converts a canvas point (y down, origin at content top-left of
// page 0) to PDF coordinates on the given page.
func canvasToPDF(opX, opY float64, pageIdx int, contentH float64, opts PaintOptions, pageH float64) (x, y float64) {
	x = opts.MarginLeft + opX
	y = pageH - opts.MarginTop - opY + float64(pageIdx)*contentH
	return x, y
}

func drawFill(c *pdf.Content, op *Op, pageIdx int, contentH float64, opts PaintOptions, pageH float64) {
	x, y := canvasToPDF(op.X, op.Y+op.H, pageIdx, contentH, opts, pageH)
	r, g, b := op.R, op.G, op.B
	// Pre-composite translucent fills against white paper. Relying on PDF
	// ExtGState alone left rgba(…) bands looking like solid dark blue when
	// the graphics state was missing or not reset (fixture-14 .alpha).
	if op.Alpha > 0 && op.Alpha < 1 {
		a := op.Alpha
		r = r*a + (1 - a) // white backdrop
		g = g*a + (1 - a)
		b = b*a + (1 - a)
	}
	c.SetFillColor(r, g, b)
	c.Rect(x, y, op.W, op.H)
	c.Fill()
}

func drawStroke(c *pdf.Content, op *Op, pageIdx int, contentH float64, opts PaintOptions, pageH float64) {
	x, y := canvasToPDF(op.X, op.Y+op.H, pageIdx, contentH, opts, pageH)
	c.SetStrokeColor(op.R, op.G, op.B)
	c.SetLineWidth(1)
	c.Rect(x, y, op.W, op.H)
	c.Stroke()
}

func drawLine(c *pdf.Content, op *Op, pageIdx int, contentH float64, opts PaintOptions, pageH float64) {
	x1, y1 := canvasToPDF(op.X, op.Y, pageIdx, contentH, opts, pageH)
	x2, y2 := canvasToPDF(op.X+op.W, op.Y+op.H, pageIdx, contentH, opts, pageH)
	w := op.Width
	if w <= 0 {
		w = 1
	}
	c.SetStrokeColor(op.R, op.G, op.B)
	c.SetLineWidth(w)
	c.MoveTo(x1, y1)
	c.LineTo(x2, y2)
	c.Stroke()
}

func drawText(c *pdf.Content, op *Op, pageIdx int, contentH float64, opts PaintOptions, pageH float64, fontName string) {
	x, y := canvasToPDF(op.X, op.Y, pageIdx, contentH, opts, pageH)
	c.SetFillColor(op.R, op.G, op.B)
	if fontName == "" {
		fontName = "F0"
	}
	c.SetFont(fontName, op.Size)
	c.BeginText()
	c.TextAt(x, y)
	// Fake bold only when CSS wants bold but the face is not a real bold TTF.
	fakeBold := op.Bold && (op.Font == nil || !op.Font.Bold())
	if fakeBold {
		c.SetLineWidth(op.Size * 0.06)
		c.TextRenderMode(2) // fill + stroke
	}
	c.TextShow(op.Text)
	if fakeBold {
		c.TextRenderMode(0)
	}
	c.EndText()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func drawImage(p *pdf.Page, c *pdf.Content, op *Op, pageIdx int, contentH float64, opts PaintOptions, name string) {
	x, y := canvasToPDF(op.X, op.Y+op.H, pageIdx, contentH, opts, p.Height())
	if name == "" {
		name = "I0"
	}
	if op.IsJPEG {
		_ = c.AddJPEGImage(name, x, y, op.W, op.H, op.Image)
	} else {
		_ = c.AddPNGImage(name, x, y, op.W, op.H, op.Image)
	}
}

func drawLink(p *pdf.Page, op *Op, pageIdx int, contentH float64, opts PaintOptions) {
	// Same-document fragments are resolved to GoTo annotations in convert
	// (applyInternalLinks) after all pages exist.
	if len(op.URI) > 0 && op.URI[0] == '#' {
		return
	}
	x, y := canvasToPDF(op.X, op.Y+op.H, pageIdx, contentH, opts, p.Height())
	p.AddLinkURI([4]float64{x, y, x + op.W, y + op.H}, op.URI)
}
