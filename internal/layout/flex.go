package layout

import (
	"gowkhtmltopdf/internal/html"
)

// buildFlex lays out a flex container (row or column) with a report-friendly
// subset: justify-content, align-items, gap, and flex-grow 0/1.
func (e *engine) buildFlex(n *html.Node, st ResolvedStyle, availW, x, y float64) *box {
	ml, mr := e.scalePt(st.MarginLeft), e.scalePt(st.MarginRight)
	b := &box{node: n, style: st, kind: "block", x: x + ml, y: y}
	b.w = availW - ml - mr
	if b.w < 0 {
		b.w = 0
	}
	if st.WidthPercent >= 0 {
		b.w = availW * st.WidthPercent / 100
	} else if st.Width >= 0 {
		b.w = e.scalePt(st.Width)
		if st.BoxSizing != "border-box" {
			b.w += e.scalePt(st.PaddingLeft) + e.scalePt(st.PaddingRight) +
				e.scalePt(st.BorderLeft.Width) + e.scalePt(st.BorderRight.Width)
		}
	}
	contentW := b.w - e.scalePt(st.PaddingLeft) - e.scalePt(st.PaddingRight) -
		e.scalePt(st.BorderLeft.Width) - e.scalePt(st.BorderRight.Width)
	if contentW < 0 {
		contentW = 0
	}
	contentX := b.x + e.scalePt(st.BorderLeft.Width) + e.scalePt(st.PaddingLeft)
	contentStart := len(e.ops)
	cy := e.scalePt(st.PaddingTop) + e.scalePt(st.BorderTop.Width)

	var kids []*html.Node
	for _, c := range n.Children {
		if c.Type != html.ElementNode {
			continue
		}
		cs := e.styles[c]
		if cs.Display == "none" {
			continue
		}
		kids = append(kids, c)
	}
	gap := e.scalePt(st.Gap)
	dir := st.FlexDirection
	if dir == "" {
		dir = "row"
	}

	if dir == "column" {
		cy = e.flowFlexColumn(b, kids, st, contentW, contentX, y, cy, gap)
	} else {
		cy = e.flowFlexRow(b, kids, st, contentW, contentX, y, cy, gap)
	}
	cy += e.scalePt(st.PaddingBottom)
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
	b.h = cy
	e.prependChrome(contentStart, st, b.x, y, b.w, b.h)
	return b
}

func (e *engine) flowFlexRow(parent *box, kids []*html.Node, st ResolvedStyle, contentW, contentX, y, cy, gap float64) float64 {
	if len(kids) == 0 {
		return cy
	}
	type item struct {
		n    *html.Node
		box  *box
		grow float64
	}
	items := make([]item, 0, len(kids))
	var fixed float64
	var growSum float64
	for _, kid := range kids {
		cs := e.styles[kid]
		// Measure with a generous width; definite widths win inside build.
		cb := e.build(kid, contentW, contentX, y+cy)
		if cb == nil {
			continue
		}
		g := cs.FlexGrow
		if g < 0 {
			g = 0
		}
		items = append(items, item{n: kid, box: cb, grow: g})
		fixed += cb.w
		growSum += g
	}
	if len(items) == 0 {
		return cy
	}
	gaps := gap * float64(len(items)-1)
	free := contentW - fixed - gaps
	if free < 0 {
		free = 0
	}
	// Distribute free space via flex-grow, then justify-content for remainder.
	usedGrow := 0.0
	if growSum > 0 && free > 0 {
		for i := range items {
			if items[i].grow > 0 {
				extra := free * (items[i].grow / growSum)
				items[i].box.w += extra
				usedGrow += extra
			}
		}
		free -= usedGrow
		if free < 0 {
			free = 0
		}
	}
	totalW := 0.0
	for _, it := range items {
		totalW += it.box.w
	}
	totalW += gaps
	startX := contentX
	justifyGap := gap
	switch st.JustifyContent {
	case "flex-end", "end":
		startX = contentX + (contentW - totalW)
	case "center":
		startX = contentX + (contentW-totalW)/2
	case "space-between":
		if len(items) > 1 {
			justifyGap = gap
			rem := contentW - (totalW - gaps)
			if rem > 0 {
				justifyGap = rem / float64(len(items)-1)
			}
			startX = contentX
		}
	default: // flex-start
		startX = contentX
	}

	rowH := 0.0
	for _, it := range items {
		if it.box.h > rowH {
			rowH = it.box.h
		}
	}
	lx := startX
	for i, it := range items {
		cb := it.box
		dx := lx - cb.x
		dy := 0.0
		switch st.AlignItems {
		case "flex-end", "end":
			dy = (y + cy + rowH) - (cb.y + cb.h)
		case "center":
			dy = (y + cy + (rowH-cb.h)/2) - cb.y
		default: // stretch / flex-start: top-align
			dy = (y + cy) - cb.y
		}
		e.shiftBoxOps(cb, dx, dy)
		cb.x += dx
		cb.y += dy
		if parent != nil {
			parent.children = append(parent.children, cb)
		}
		lx += cb.w
		if i < len(items)-1 {
			lx += justifyGap
		}
	}
	return cy + rowH
}

func (e *engine) flowFlexColumn(parent *box, kids []*html.Node, st ResolvedStyle, contentW, contentX, y, cy, gap float64) float64 {
	for i, kid := range kids {
		cb := e.build(kid, contentW, contentX, y+cy)
		if cb == nil {
			continue
		}
		if parent != nil {
			parent.children = append(parent.children, cb)
		}
		cy += cb.h
		if i < len(kids)-1 {
			cy += gap
		}
	}
	return cy
}

// applyRelativeOffset shifts a positioned:relative box and its ops by top/left
// (right/bottom when the corresponding auto flags are set).
func (e *engine) applyRelativeOffset(b *box) {
	if b == nil || b.style.Position != "relative" {
		return
	}
	st := b.style
	dx, dy := 0.0, 0.0
	if !st.LeftAuto {
		dx = e.scalePt(st.Left)
	} else if !st.RightAuto {
		dx = -e.scalePt(st.Right)
	}
	if !st.TopAuto {
		dy = e.scalePt(st.Top)
	} else if !st.BottomAuto {
		dy = -e.scalePt(st.Bottom)
	}
	if dx == 0 && dy == 0 {
		return
	}
	b.x += dx
	b.y += dy
	e.shiftBoxOps(b, dx, dy)
}
