package layout

import (
	"math"
	"strings"

	"gowkhtmltopdf/internal/html"
)

const columnSpanAll = "all"

// multicolItem is one in-flow child measured at column width.
type multicolItem struct {
	n *html.Node
	h float64
}

// multicolSeg groups consecutive in-flow multicol children, optionally a
// column-span:all spanner.
type multicolSeg struct {
	spanner bool
	nodes   []*html.Node
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
	margL := e.scalePt(st.MarginLeft)
	boxNode := &box{node: n, style: st, kind: "block", x: x, y: y} //nolint:exhaustruct // intentional zero fields
	boxNode.w = resolveUsedWidth(st, availW, e)

	if definiteW := st.Width >= 0 || st.WidthPercent >= 0; definiteW && (st.MarginLeftAuto || st.MarginRightAuto) {
		free := availW - boxNode.w
		if free < 0 {
			free = 0
		}

		margR := e.scalePt(st.MarginRight)

		switch {
		case st.MarginLeftAuto && st.MarginRightAuto:
			margL = free / two
		case st.MarginLeftAuto:
			margL = free - margR
			if margL < 0 {
				margL = 0
			}
		}
	}

	boxNode.x = x + margL

	contentX, contentW := e.contentBox(boxNode.x, boxNode.w, st)
	contentStart := len(e.ops)

	curY := e.scalePt(st.PaddingTop) + e.scalePt(st.BorderTop.Width)
	gap := e.multicolGap(st)
	nCols, colW := usedColumnCountWidth(contentW, gap, st.ColumnWidth, st.ColumnCount)
	// One column: use ordinary block flow. The multicol line/page snap path
	// otherwise reserves a tall empty band before a single wide child
	// (wiki reflist with column-width:30em on a narrow page).
	if nCols <= 1 {
		return e.flowMulticolSingleColumn(boxNode, n, st, contentX, contentW, y, curY, contentStart)
	}

	var segs []multicolSeg

	for _, kid := range multicolKids(n, e) {
		spanAll := kid.Type == html.ElementNode && e.styles[kid].ColumnSpan == columnSpanAll
		if spanAll {
			segs = append(segs, multicolSeg{spanner: true, nodes: []*html.Node{kid}})

			continue
		}

		if len(segs) == 0 || segs[len(segs)-1].spanner {
			segs = append(segs, multicolSeg{nodes: []*html.Node{kid}}) //nolint:exhaustruct // intentional zero fields
		} else {
			segs[len(segs)-1].nodes = append(segs[len(segs)-1].nodes, kid)
		}
	}

	for _, cssSheet := range segs {
		if cssSheet.spanner {
			curY = e.flowMulticolSpanner(boxNode, cssSheet.nodes, contentW, contentX, y, curY)

			continue
		}

		curY = e.flowMulticolSegment(boxNode, cssSheet.nodes, st, nCols, colW, gap, contentX, y, curY)
	}

	curY = clampMulticolHeight(curY, st, e)
	boxNode.height = curY
	e.prependChrome(contentStart, boxNode, st, boxNode.x, y, boxNode.w, boxNode.height)

	return boxNode
}

// multicolKids returns the non-hidden in-flow children of a multicol container.
func multicolKids(n *html.Node, e *engine) []*html.Node {
	var kids []*html.Node

	for _, child := range n.Children {
		if child.Type == html.ElementNode {
			if e.styles[child].Display == displayNone {
				continue
			}

			kids = append(kids, child)
		} else if child.Type == html.TextNode && strings.TrimSpace(child.Text) != "" {
			kids = append(kids, child)
		}
	}

	return kids
}

// flowMulticolSingleColumn lays out a one-column multicol container with the
// ordinary block path (no line/page snapping).
func (e *engine) flowMulticolSingleColumn(boxNode *box, n *html.Node, st ResolvedStyle, contentX, contentW, y, curY float64, contentStart int) *box {
	pop, enclose := e.pushBFCFloats(st, contentX, contentW)
	curY = e.flowChildren(boxNode, n.Children, st, contentW, contentX, y, curY)

	if enclose && e.bfcFloats != nil {
		curY = e.bfcFloats.extentCy(y, curY)
	}

	pop()

	boxNode.height = clampMulticolHeight(curY, st, e)
	e.prependChrome(contentStart, boxNode, st, boxNode.x, y, boxNode.w, boxNode.height)

	return boxNode
}

// flowMulticolSpanner lays out column-span:all children across the full
// content width. Returns the advanced content-relative cy.
func (e *engine) flowMulticolSpanner(boxNode *box, nodes []*html.Node, contentW, contentX, y, curY float64) float64 {
	for _, kid := range nodes {
		cs := e.styles[kid]
		curY += collapseMargins(0, e.scalePt(cs.MarginTop))

		cblock := e.build(kid, contentW, contentX, y+curY)
		if cblock == nil {
			continue
		}

		curY += cblock.height
		boxNode.children = append(boxNode.children, cblock)
	}

	return curY
}

// clampMulticolHeight applies padding-bottom and min/max height constraints to
// the accumulated content height of a multicol container.
func clampMulticolHeight(curY float64, st ResolvedStyle, e *engine) float64 {
	curY += e.scalePt(st.PaddingBottom)
	if h, ok := resolveUsedHeight(st, -1, e); ok {
		if curY < h {
			curY = h
		}
	}

	if st.MinHeight > 0 && curY < e.scalePt(st.MinHeight) {
		curY = e.scalePt(st.MinHeight)
	}

	if st.MaxHeight >= 0 && curY > e.scalePt(st.MaxHeight) {
		curY = e.scalePt(st.MaxHeight)
	}

	return curY
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
		return fitColumnCount(colCount, avail, gap)
	case colCount <= 0 && colWidth >= 0:
		return autoColumnCount(avail, gap, colWidth)
	default:
		return shrinkColumnCount(colCount, avail, gap, colWidth)
	}
}

// fitColumnCount clamps the column count to ≥1 and returns the widths that
// fill avail with gap between columns.
func fitColumnCount(n int, avail, gap float64) (int, float64) {
	if n < 1 {
		n = 1
	}

	return n, columnWidth(n, avail, gap)
}

// autoColumnCount derives the count from column-width (CSS Multicol §3.3).
func autoColumnCount(avail, gap, colWidth float64) (int, float64) {
	den := colWidth + gap
	if den <= 0 {
		return 1, avail
	}

	return fitColumnCount(int(math.Floor((avail+gap)/den)), avail, gap)
}

// shrinkColumnCount starts from the specified count and reduces it until all
// columns fit avail, then stretches the widths to fill.
func shrinkColumnCount(colCount int, avail, gap, colWidth float64) (int, float64) {
	n := colCount
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

	return fitColumnCount(n, avail, gap)
}

// columnWidth returns the used column width when n columns of the container's
// content width are separated by gap.
func columnWidth(n int, avail, gap float64) float64 {
	w := (avail - float64(n-1)*gap) / float64(n)
	if w < 0 {
		w = 0
	}

	return w
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

		items = append(items, multicolItem{n: kid, h: e.measureMulticolChildHeight(kid, colW)})
	}

	if len(items) == 0 {
		return cy
	}

	balance := st.ColumnFill != "auto"
	definiteH := resolveContentHeight(st, e)
	padTop := e.scalePt(st.PaddingTop) + e.scalePt(st.BorderTop.Width)

	idx := 0
	for idx < len(items) {
		absTop := y + cy

		pageIdx := int(absTop / pageH)
		if pageIdx < 0 {
			pageIdx = 0
		}

		pageTop := float64(pageIdx) * pageH
		boundary := pageTop + pageH
		remain := boundary - absTop
		minUseful := e.scalePt(st.FontSize) * defaultLineHeightRatio
		// Snap to the next page when little/no room remains. Guard against
		// float edges where absTop/pageH truncates just below an integer and
		// remain≈0 (would spin forever on cy += remain).
		if remain < minUseful {
			cy = snapMulticolToPage(cy, absTop, boundary, pageH)

			continue
		}

		maxColH := remain

		if !balance && definiteH >= 0 {
			maxColH = clampMulticolRemainder(maxColH, definiteH-(cy-padTop))
		}

		if items[idx].h > maxColH+1e-6 && absTop > pageTop+1e-6 {
			cy = snapMulticolToPage(cy, absTop, boundary, pageH)

			continue
		}

		batch, nextIdx, totalH := collectMulticolBatch(items, idx, maxColH, maxColH*float64(nCols))
		idx = nextIdx
		if len(batch) == 0 {
			break
		}

		cy += e.placeMulticolLine(parent, batch, nCols, colW, gap, contentX, y, cy, maxColH, balance, totalH)
	}

	return cy
}

// snapMulticolToPage advances cy to the next usable page boundary when the
// current position is too close to (or past) the page end.
func snapMulticolToPage(cy, absTop, boundary, pageH float64) float64 {
	target := boundary
	if target <= absTop+1e-6 {
		target += pageH
	}

	return cy + target - absTop
}

// clampMulticolRemainder limits the usable column height to the remaining
// definite container height (never negative).
func clampMulticolRemainder(maxColH, left float64) float64 {
	if left < maxColH {
		maxColH = left
	}

	if maxColH < 0 {
		maxColH = 0
	}

	return maxColH
}

// collectMulticolBatch greedily fills one multicol line's worth of items from
// items[idx:]. Returns the batch, the next index, and the total batch height.
func collectMulticolBatch(items []multicolItem, idx int, maxColH, capacity float64) (batch []multicolItem, next int, totalH float64) {
	for idx < len(items) {
		item := items[idx]
		if len(batch) > 0 && totalH+item.h > capacity+1e-6 {
			break
		}

		batch = append(batch, item)
		totalH += item.h
		idx++

		if item.h > maxColH+1e-6 && len(batch) == 1 {
			break
		}
	}

	return batch, idx, totalH
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

	for _, item := range items {
		if advanceMulticolColumn(col, colHeights, item, nCols, maxColH, target, balance) {
			col++
		}

		cblock := e.build(item.n, colW, colX(col), y+cy+colHeights[col])
		if cblock == nil {
			continue
		}

		colHeights[col] += cblock.height

		if parent != nil {
			parent.children = append(parent.children, cblock)
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

// advanceMulticolColumn reports whether the item must start a new column
// under balance or auto column-fill.
func advanceMulticolColumn(col int, colHeights []float64, item multicolItem, nCols int, maxColH, target float64, balance bool) bool {
	if col >= nCols-1 || colHeights[col] <= 0 {
		return false
	}

	if balance {
		return colHeights[col]+item.h/2 > target && colHeights[col] >= target*0.85
	}

	return colHeights[col]+item.h > maxColH+1e-6
}

// measureMulticolChildHeight lays out n at availW without emitting ops.
func (e *engine) measureMulticolChildHeight(n *html.Node, availW float64) float64 {
	prev := e.noEmit
	e.noEmit = true
	start := len(e.ops)
	boxNode := e.build(n, availW, 0, 0)

	if len(e.ops) > start {
		e.ops = e.ops[:start]
	}

	e.noEmit = prev

	if boxNode == nil {
		return 0
	}

	return boxNode.height
}
