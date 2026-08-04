package layout

import (
	"sort"

	"gowkhtmltopdf/internal/html"
)

type flexMeas struct {
	n      *html.Node
	baseW  float64
	grow   float64
	shrink float64
	order  int
}

// buildFlex lays out a flex container (row or column) with a report-friendly
// subset: justify-content, align-items, gap, flex-grow/shrink/basis, order,
// and flex-wrap.
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
		cy = e.flowFlexColumn(b, kids, contentW, contentX, y, cy, gap)
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
	wrap := st.FlexWrap == "wrap" || st.FlexWrap == "wrap-reverse"

	items := make([]flexMeas, 0, len(kids))
	for _, kid := range kids {
		cs := e.styles[kid]
		g := cs.FlexGrow
		if g < 0 {
			g = 0
		}
		sh := cs.FlexShrink
		if sh < 0 {
			sh = 1
		}
		items = append(items, flexMeas{
			n: kid, baseW: e.flexItemBaseWidth(kid, cs, contentW),
			grow: g, shrink: sh, order: cs.FlexOrder,
		})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].order < items[j].order })

	var lines [][]flexMeas
	if !wrap {
		lines = [][]flexMeas{items}
	} else {
		var line []flexMeas
		used := 0.0
		for _, it := range items {
			need := it.baseW
			if len(line) > 0 {
				need += gap
			}
			if len(line) > 0 && used+need > contentW+1e-6 {
				lines = append(lines, line)
				line = nil
				used = 0
				need = it.baseW
			}
			line = append(line, it)
			used += need
		}
		if len(line) > 0 {
			lines = append(lines, line)
		}
		if st.FlexWrap == "wrap-reverse" {
			for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
				lines[i], lines[j] = lines[j], lines[i]
			}
		}
	}

	for li, line := range lines {
		cy = e.placeFlexLineMeasured(parent, st, line, contentW, contentX, y, cy, gap)
		if li < len(lines)-1 {
			cy += gap
		}
	}
	return cy
}

func (e *engine) flexItemBaseWidth(n *html.Node, cs ResolvedStyle, contentW float64) float64 {
	pad := e.scalePt(cs.PaddingLeft) + e.scalePt(cs.PaddingRight) +
		e.scalePt(cs.BorderLeft.Width) + e.scalePt(cs.BorderRight.Width)
	if cs.FlexBasisPercent >= 0 {
		w := contentW * cs.FlexBasisPercent / 100
		if cs.BoxSizing != "border-box" {
			w += pad
		}
		return w
	}
	if cs.FlexBasis >= 0 {
		w := e.scalePt(cs.FlexBasis)
		if cs.BoxSizing != "border-box" {
			w += pad
		}
		return w
	}
	if cs.WidthPercent >= 0 {
		w := contentW * cs.WidthPercent / 100
		if cs.BoxSizing != "border-box" {
			w += pad
		}
		return w
	}
	if cs.Width >= 0 {
		w := e.scalePt(cs.Width)
		if cs.BoxSizing != "border-box" {
			w += pad
		}
		return w
	}
	intr := e.measureCellContent(n, cs) + pad +
		e.scalePt(cs.MarginLeft) + e.scalePt(cs.MarginRight)
	if intr <= 0 {
		intr = pad + e.scalePt(cs.FontSize)*2
	}
	if intr > contentW {
		intr = contentW
	}
	return intr
}

func (e *engine) placeFlexLineMeasured(parent *box, st ResolvedStyle, items []flexMeas, contentW, contentX, y, cy, gap float64) float64 {
	var fixed, growSum, shrinkSum float64
	for _, it := range items {
		fixed += it.baseW
		growSum += it.grow
		shrinkSum += it.shrink * it.baseW
	}
	gaps := gap * float64(len(items)-1)
	if gaps < 0 {
		gaps = 0
	}
	free := contentW - fixed - gaps
	widths := make([]float64, len(items))
	for i, it := range items {
		widths[i] = it.baseW
	}
	if free > 0 && growSum > 0 {
		for i, it := range items {
			if it.grow > 0 {
				widths[i] += free * (it.grow / growSum)
			}
		}
	} else if free < 0 && shrinkSum > 0 {
		deficit := -free
		for i, it := range items {
			if it.shrink <= 0 || it.baseW <= 0 {
				continue
			}
			share := (it.shrink * it.baseW) / shrinkSum
			widths[i] -= deficit * share
			if widths[i] < 0 {
				widths[i] = 0
			}
		}
	}
	// Clamp to min/max-width after grow/shrink (simplified flex algorithm).
	for i, it := range items {
		cs := e.styles[it.n]
		if cs.MinWidth > 0 {
			mn := e.scalePt(cs.MinWidth)
			if widths[i] < mn {
				widths[i] = mn
			}
		}
		if cs.MaxWidth >= 0 {
			mx := e.scalePt(cs.MaxWidth)
			if widths[i] > mx {
				widths[i] = mx
			}
		}
	}
	totalW := 0.0
	for _, w := range widths {
		totalW += w
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
			rem := contentW - (totalW - gaps)
			if rem > 0 {
				justifyGap = rem / float64(len(items)-1)
			}
			startX = contentX
		}
	}

	type placed struct {
		box *box
		h   float64
	}
	built := make([]placed, 0, len(items))
	rowH := 0.0
	lx := startX
	for i, it := range items {
		cb := e.build(it.n, widths[i], lx, y+cy)
		if cb == nil {
			built = append(built, placed{})
			lx += widths[i]
			if i < len(items)-1 {
				lx += justifyGap
			}
			continue
		}
		dx := lx - cb.x
		dy := (y + cy) - cb.y
		e.shiftBoxOps(cb, dx, dy)
		cb.x += dx
		cb.y += dy
		if cb.h > rowH {
			rowH = cb.h
		}
		built = append(built, placed{box: cb, h: cb.h})
		if parent != nil {
			parent.children = append(parent.children, cb)
		}
		lx += widths[i]
		if i < len(items)-1 {
			lx += justifyGap
		}
	}
	for _, p := range built {
		if p.box == nil {
			continue
		}
		dy := 0.0
		switch st.AlignItems {
		case "flex-end", "end":
			dy = (y + cy + rowH) - (p.box.y + p.box.h)
		case "center":
			dy = (y + cy + (rowH-p.box.h)/2) - p.box.y
		default:
			dy = 0
		}
		if dy != 0 {
			e.shiftBoxOps(p.box, 0, dy)
			p.box.y += dy
		}
	}
	return cy + rowH
}

func (e *engine) flowFlexColumn(parent *box, kids []*html.Node, contentW, contentX, y, cy, gap float64) float64 {
	type ord struct {
		n     *html.Node
		order int
	}
	items := make([]ord, 0, len(kids))
	for _, kid := range kids {
		items = append(items, ord{n: kid, order: e.styles[kid].FlexOrder})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].order < items[j].order })
	for i, it := range items {
		cb := e.build(it.n, contentW, contentX, y+cy)
		if cb == nil {
			continue
		}
		if parent != nil {
			parent.children = append(parent.children, cb)
		}
		cy += cb.h
		if i < len(items)-1 {
			cy += gap
		}
	}
	return cy
}

// applyRelativeOffset shifts a positioned:relative (or sticky-lite) box and
// its ops by top/left (right/bottom when the corresponding auto flags are set).
func (e *engine) applyRelativeOffset(b *box) {
	if b == nil || (b.style.Position != "relative" && b.style.Position != "sticky") {
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
