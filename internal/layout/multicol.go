package layout

import (
	"math"
	"strings"

	"gowkhtmltopdf/internal/html"
)

// multicolItem is one in-flow child measured at column width.
type multicolItem struct {
	n *html.Node
	h float64
}

// buildMulticol lays out a CSS multi-column container (report-engine lite):
// column-count / column-width / columns, column-gap (normal → 1em),
// column-span: none|all, column-fill: balance|auto.
//
// Each multicol line is a row of column boxes that must not straddle a page
// boundary: when remaining page space is exhausted, a new multicol line starts
// on the next page. Nested floats/abspos inside columns use the ordinary BFC
// path (best-effort; not Chrome-balanced with floats).
func (e *engine) buildMulticol(n *html.Node, st ResolvedStyle, availW, x, y float64) *box {
	ml := e.scalePt(st.MarginLeft)
	b := &box{node: n, style: st, kind: "block", x: x, y: y}
	b.w = resolveUsedWidth(st, availW, e)
	if definiteW := st.Width >= 0 || st.WidthPercent >= 0; definiteW && (st.MarginLeftAuto || st.MarginRightAuto) {
		free := availW - b.w
		if free < 0 {
			free = 0
		}
		mr := e.scalePt(st.MarginRight)
		switch {
		case st.MarginLeftAuto && st.MarginRightAuto:
			ml = free / 2
		case st.MarginLeftAuto:
			ml = free - mr
			if ml < 0 {
				ml = 0
			}
		}
	}
	b.x = x + ml

	contentW := b.w - e.scalePt(st.PaddingLeft) - e.scalePt(st.PaddingRight) -
		e.scalePt(st.BorderLeft.Width) - e.scalePt(st.BorderRight.Width)
	if contentW < 0 {
		contentW = 0
	}
	contentX := b.x + e.scalePt(st.BorderLeft.Width) + e.scalePt(st.PaddingLeft)
	contentStart := len(e.ops)

	cy := e.scalePt(st.PaddingTop) + e.scalePt(st.BorderTop.Width)
	gap := e.multicolGap(st)
	nCols, colW := usedColumnCountWidth(contentW, gap, st.ColumnWidth, st.ColumnCount)

	var kids []*html.Node
	for _, c := range n.Children {
		if c.Type == html.ElementNode {
			if e.styles[c].Display == "none" {
				continue
			}
			kids = append(kids, c)
		} else if c.Type == html.TextNode && strings.TrimSpace(c.Text) != "" {
			kids = append(kids, c)
		}
	}

	type seg struct {
		spanner bool
		nodes   []*html.Node
	}
	var segs []seg
	for _, kid := range kids {
		spanAll := kid.Type == html.ElementNode && e.styles[kid].ColumnSpan == "all"
		if spanAll {
			segs = append(segs, seg{spanner: true, nodes: []*html.Node{kid}})
			continue
		}
		if len(segs) == 0 || segs[len(segs)-1].spanner {
			segs = append(segs, seg{nodes: []*html.Node{kid}})
		} else {
			segs[len(segs)-1].nodes = append(segs[len(segs)-1].nodes, kid)
		}
	}

	for _, s := range segs {
		if s.spanner {
			for _, kid := range s.nodes {
				cs := e.styles[kid]
				cy += collapseMargins(0, e.scalePt(cs.MarginTop))
				cb := e.build(kid, contentW, contentX, y+cy)
				if cb == nil {
					continue
				}
				cy += cb.h
				b.children = append(b.children, cb)
			}
			continue
		}
		cy = e.flowMulticolSegment(b, s.nodes, st, nCols, colW, gap, contentX, y, cy)
	}

	cy += e.scalePt(st.PaddingBottom)
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

// multicolGap returns the used column-gap for a multicol container.
// column-gap: normal (initial) computes to 1em; flex/grid treat normal as 0.
func (e *engine) multicolGap(st ResolvedStyle) float64 {
	if st.ColumnGapNormal {
		return e.scalePt(st.FontSize)
	}
	return e.scalePt(st.ColumnGap)
}

// usedColumnCountWidth resolves used column count and width from container
// content width, gap, and specified column-width / column-count (CSS Multicol §3.3).
func usedColumnCountWidth(avail, gap, colWidth float64, colCount int) (n int, w float64) {
	switch {
	case colCount <= 0 && colWidth < 0:
		return 1, avail
	case colCount > 0 && colWidth < 0:
		n = colCount
		if n < 1 {
			n = 1
		}
		w = (avail - float64(n-1)*gap) / float64(n)
		if w < 0 {
			w = 0
		}
		return n, w
	case colCount <= 0 && colWidth >= 0:
		den := colWidth + gap
		if den <= 0 {
			return 1, avail
		}
		n = int(math.Floor((avail + gap) / den))
		if n < 1 {
			n = 1
		}
		w = (avail - float64(n-1)*gap) / float64(n)
		if w < 0 {
			w = 0
		}
		return n, w
	default:
		// Both specified: shrink count until columns fit, then stretch widths.
		n = colCount
		if n < 1 {
			n = 1
		}
		for n > 1 {
			need := float64(n)*colWidth + float64(n-1)*gap
			if need <= avail+1e-6 {
				break
			}
			n--
		}
		w = (avail - float64(n-1)*gap) / float64(n)
		if w < 0 {
			w = 0
		}
		return n, w
	}
}

// flowMulticolSegment places in-flow children across column boxes, starting a
// new multicol line on the next page when the current line would cross a page
// boundary. Returns the advanced content-relative cy.
func (e *engine) flowMulticolSegment(parent *box, nodes []*html.Node, st ResolvedStyle, nCols int, colW, gap, contentX, y, cy float64) float64 {
	if nCols < 1 {
		nCols = 1
	}
	pageH := e.opts.Height
	if pageH <= 0 {
		pageH = 1e12
	}

	items := make([]multicolItem, 0, len(nodes))
	for _, kid := range nodes {
		if kid.Type != html.ElementNode {
			continue
		}
		h := e.measureMulticolChildHeight(kid, colW)
		items = append(items, multicolItem{n: kid, h: h})
	}
	if len(items) == 0 {
		return cy
	}

	balance := st.ColumnFill != "auto"
	definiteH := resolveContentHeight(st, e)
	padTop := e.scalePt(st.PaddingTop) + e.scalePt(st.BorderTop.Width)

	i := 0
	for i < len(items) {
		absTop := y + cy
		pageIdx := int(absTop / pageH)
		if pageIdx < 0 {
			pageIdx = 0
		}
		pageTop := float64(pageIdx) * pageH
		boundary := pageTop + pageH
		remain := boundary - absTop
		minUseful := e.scalePt(st.FontSize) * 1.2
		// Snap to the next page when little/no room remains. Guard against
		// float edges where absTop/pageH truncates just below an integer and
		// remain≈0 (would spin forever on cy += remain).
		if remain < minUseful {
			target := boundary
			if target <= absTop+1e-6 {
				target += pageH
			}
			cy += target - absTop
			continue
		}

		maxColH := remain
		if !balance && definiteH >= 0 {
			left := definiteH - (cy - padTop)
			if left < maxColH {
				maxColH = left
			}
			if maxColH < 0 {
				maxColH = 0
			}
		}

		if items[i].h > maxColH+1e-6 && absTop > pageTop+1e-6 {
			target := boundary
			if target <= absTop+1e-6 {
				target += pageH
			}
			cy += target - absTop
			continue
		}

		capacity := maxColH * float64(nCols)
		var batch []multicolItem
		totalH := 0.0
		for i < len(items) {
			it := items[i]
			if len(batch) > 0 && totalH+it.h > capacity+1e-6 {
				break
			}
			batch = append(batch, it)
			totalH += it.h
			i++
			if it.h > maxColH+1e-6 && len(batch) == 1 {
				break
			}
		}
		if len(batch) == 0 {
			break
		}

		lineH := e.placeMulticolLine(parent, batch, nCols, colW, gap, contentX, y, cy, maxColH, balance, totalH)
		cy += lineH
	}
	return cy
}

// placeMulticolLine assigns items into columns (balance or auto fill) and
// builds them. Returns the line's used height (max column stack).
func (e *engine) placeMulticolLine(parent *box, items []multicolItem, nCols int, colW, gap, contentX, y, cy, maxColH float64, balance bool, totalH float64) float64 {
	colX := func(c int) float64 {
		return contentX + float64(c)*(colW+gap)
	}
	colHeights := make([]float64, nCols)
	target := 0.0
	if balance && nCols > 0 {
		target = totalH / float64(nCols)
	}

	col := 0
	for _, it := range items {
		if balance {
			if col < nCols-1 && colHeights[col] > 0 &&
				colHeights[col]+it.h/2 > target &&
				colHeights[col] >= target*0.85 {
				col++
			}
		} else if col < nCols-1 && colHeights[col] > 0 &&
			colHeights[col]+it.h > maxColH+1e-6 {
			col++
		}
		cb := e.build(it.n, colW, colX(col), y+cy+colHeights[col])
		if cb == nil {
			continue
		}
		colHeights[col] += cb.h
		if parent != nil {
			parent.children = append(parent.children, cb)
		}
	}

	lineH := 0.0
	for _, h := range colHeights {
		if h > lineH {
			lineH = h
		}
	}
	return lineH
}

// measureMulticolChildHeight lays out n at availW without emitting ops.
func (e *engine) measureMulticolChildHeight(n *html.Node, availW float64) float64 {
	prev := e.noEmit
	e.noEmit = true
	start := len(e.ops)
	b := e.build(n, availW, 0, 0)
	if len(e.ops) > start {
		e.ops = e.ops[:start]
	}
	e.noEmit = prev
	if b == nil {
		return 0
	}
	return b.h
}
