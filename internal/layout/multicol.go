//nolint:all
package layout

import (
	"math"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
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
func (e *engine) buildMulticol(node *html.Node, style ResolvedStyle, availW, x, yPos float64) *box {
	boxNode := &box{ //nolint:exhaustruct // intentional zero fields
		node: node, style: e.stylePtr(node), kind: displayBlock, x: x, y: yPos,
	}
	boxNode.w = resolveUsedWidth(style, availW, e)
	boxNode.x = x + e.multicolAutoMargin(style, availW, boxNode.w)

	contentX, contentW := e.contentBox(boxNode.x, boxNode.w, style)
	contentStart := len(e.ops)

	curY := e.scalePt(style.PaddingTop) + e.scalePt(style.BorderTop.Width)
	gap := e.multicolGap(style)
	nCols, colW := usedColumnCountWidth(contentW, gap, style.ColumnWidth, style.ColumnCount)
	// One column: use ordinary block flow. The multicol line/page snap path
	// otherwise reserves a tall empty band before a single wide child
	// (wiki reflist with column-width:30em on a narrow page).
	if nCols <= 1 {
		return e.flowMulticolSingleColumn(boxNode, node, style, contentX, contentW, yPos, curY, contentStart)
	}

	for _, seg := range e.collectMulticolSegs(node) {
		if seg.spanner {
			curY = e.flowMulticolSpanner(boxNode, seg.nodes, contentW, contentX, yPos, curY)

			continue
		}

		curY = e.flowMulticolSegment(boxNode, seg.nodes, style, nCols, colW, gap, contentX, yPos, curY)
	}

	curY = clampMulticolHeight(curY, style, e)
	boxNode.height = curY
	e.prependChrome(contentStart, boxNode, style, boxNode.x, yPos, boxNode.w, boxNode.height)

	return boxNode
}

// multicolAutoMargin resolves the used left margin of a multicol container
// when auto margins are present and the width is definite.
func (e *engine) multicolAutoMargin(style ResolvedStyle, availW, boxW float64) float64 {
	margL := e.scalePt(style.MarginLeft)

	if definiteW := style.Width >= 0 || style.WidthPercent >= 0; definiteW &&
		(style.MarginLeftAuto || style.MarginRightAuto) {
		free := availW - boxW
		if free < 0 {
			free = 0
		}

		margR := e.scalePt(style.MarginRight)

		switch {
		case style.MarginLeftAuto && style.MarginRightAuto:
			margL = free / 2
		case style.MarginLeftAuto:
			margL = free - margR
			if margL < 0 {
				margL = 0
			}
		}
	}

	return margL
}

// collectMulticolSegs groups consecutive in-flow multicol children into
// segments, with column-span:all children as standalone spanner segments.
func (e *engine) collectMulticolSegs(n *html.Node) []multicolSeg {
	var segs []multicolSeg

	for _, kid := range multicolKids(n, e) {
		spanAll := kid.Type == html.ElementNode && e.stylePtr(kid).ColumnSpan == columnSpanAll
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

	return segs
}

// multicolKids returns the non-hidden in-flow children of a multicol container.
func multicolKids(n *html.Node, e *engine) []*html.Node {
	var kids []*html.Node

	for _, child := range n.Children {
		if child.Type == html.ElementNode {
			if e.stylePtr(child).Display == displayNone {
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
func (e *engine) flowMulticolSingleColumn(
	boxNode *box, node *html.Node, style ResolvedStyle, contentX, contentW, yPos, curY float64, contentStart int,
) *box {
	enclose := e.pushBFCFloats(style, contentX, contentW)
	curY = e.flowChildren(boxNode, node.Children, style, contentW, contentX, yPos, curY)

	if enclose && e.bfcFloats != nil {
		curY = e.bfcFloats.extentCy(yPos, curY)
	}

	e.popBFCFloats(enclose)

	boxNode.height = clampMulticolHeight(curY, style, e)
	e.prependChrome(contentStart, boxNode, style, boxNode.x, yPos, boxNode.w, boxNode.height)

	return boxNode
}

// flowMulticolSpanner lays out column-span:all children across the full
// content width. Returns the advanced content-relative cy.
func (e *engine) flowMulticolSpanner(boxNode *box, nodes []*html.Node, contentW, contentX, yPos, curY float64) float64 {
	for _, kid := range nodes {
		cs := e.stylePtr(kid)
		curY += collapseMargins(0, e.scalePt(cs.MarginTop))

		cblock := e.build(kid, contentW, contentX, yPos+curY)
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
func clampMulticolHeight(curY float64, style ResolvedStyle, eng *engine) float64 {
	curY += eng.scalePt(style.PaddingBottom)
	if h, ok := resolveUsedHeight(style, -1, eng); ok {
		if curY < h {
			curY = h
		}
	}

	if style.MinHeight > 0 && curY < eng.scalePt(style.MinHeight) {
		curY = eng.scalePt(style.MinHeight)
	}

	if style.MaxHeight >= 0 && curY > eng.scalePt(style.MaxHeight) {
		curY = eng.scalePt(style.MaxHeight)
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
func usedColumnCountWidth(avail, gap, colWidth float64, colCount int) (int, float64) {
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
	count := colCount
	if count < 1 {
		count = 1
	}

	for count > 1 {
		need := float64(count)*colWidth + float64(count-1)*gap
		if need <= avail+1e-6 {
			break
		}

		count--
	}

	return fitColumnCount(count, avail, gap)
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
func (e *engine) flowMulticolSegment(
	parent *box, nodes []*html.Node, style ResolvedStyle, nCols int, colW, gap, contentX, yPos, curY float64,
) float64 {
	if nCols < 1 {
		nCols = 1
	}

	pageH := e.opts.Height
	if pageH <= 0 {
		pageH = 1e12
	}

	items := e.measureMulticolItems(nodes, colW)
	if len(items) == 0 {
		return curY
	}

	balance := style.ColumnFill != overflowAuto
	definiteH := resolveContentHeight(style, e)
	padTop := e.scalePt(style.PaddingTop) + e.scalePt(style.BorderTop.Width)

	idx := 0
	for idx < len(items) {
		maxColH, snappedCy, snap := e.multicolColumnHeight(items, idx, style, balance, definiteH, padTop, yPos, curY, pageH)
		if snap {
			curY = snappedCy

			continue
		}

		batch, nextIdx, totalH := collectMulticolBatch(items, idx, maxColH, maxColH*float64(nCols))
		idx = nextIdx

		if len(batch) == 0 {
			break
		}

		curY += e.placeMulticolLine(
			parent, batch, style, nCols, colW, gap, contentX, yPos, curY, maxColH, balance, totalH,
		)
	}

	return curY
}

// measureMulticolItems measures each in-flow element child at the column width,
// skipping text nodes that never produce column content.
func (e *engine) measureMulticolItems(nodes []*html.Node, colW float64) []multicolItem {
	items := make([]multicolItem, 0, len(nodes))

	for _, kid := range nodes {
		if kid.Type != html.ElementNode {
			continue
		}

		items = append(items, multicolItem{n: kid, h: e.measureMulticolChildHeight(kid, colW)})
	}

	return items
}

// multicolColumnHeight computes the usable column height at the current page
// position. When the remaining page space is too small for the next item, it
// snaps to the next page and reports that via the second return value.
func (e *engine) multicolColumnHeight(
	items []multicolItem, idx int, style ResolvedStyle, balance bool, definiteH, padTop, yPos, curY, pageH float64,
) (float64, float64, bool) {
	absTop := yPos + curY

	pageIdx := int(absTop / pageH)
	if pageIdx < 0 {
		pageIdx = 0
	}

	pageTop := float64(pageIdx) * pageH
	boundary := pageTop + pageH
	remain := boundary - absTop
	minUseful := e.scalePt(style.FontSize) * 1.2
	// Snap to the next page when little/no room remains. Guard against
	// float edges where absTop/pageH truncates just below an integer and
	// remain≈0 (would spin forever on cy += remain).
	if remain < minUseful {
		return 0, snapMulticolToPage(curY, absTop, boundary, pageH), true
	}

	maxColH := remain

	if !balance && definiteH >= 0 {
		maxColH = clampMulticolRemainder(maxColH, definiteH-(curY-padTop))
	}

	if items[idx].h > maxColH+1e-6 && absTop > pageTop+1e-6 {
		return 0, snapMulticolToPage(curY, absTop, boundary, pageH), true
	}

	return maxColH, 0, false
}

// snapMulticolToPage advances cy to the next usable page boundary when the
// current position is too close to (or past) the page end.
func snapMulticolToPage(curY, absTop, boundary, pageH float64) float64 {
	target := boundary
	if target <= absTop+1e-6 {
		target += pageH
	}

	return curY + target - absTop
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
func collectMulticolBatch(items []multicolItem, idx int, maxColH, capacity float64) ([]multicolItem, int, float64) {
	var batch []multicolItem

	totalH := 0.0

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
func (e *engine) placeMulticolLine(
	parent *box, items []multicolItem, style ResolvedStyle, nCols int, colW, gap, contentX, yPos, curY, maxColH float64,
	balance bool, totalH float64,
) float64 {
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

		cblock := e.build(item.n, colW, colX(col), yPos+curY+colHeights[col])
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
func advanceMulticolColumn(
	col int, colHeights []float64, item multicolItem, nCols int, maxColH, target float64, balance bool,
) bool {
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
