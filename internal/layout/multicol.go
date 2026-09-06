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
// Direct non-whitespace text is wrapped in anonymous block boxes (same idea as
// flexChildren) because build() ignores TextNode and measureMulticolItems only
// measures elements.
func multicolKids(n *html.Node, e *engine) []*html.Node {
	parentStyle := ResolvedStyle{}
	if n != nil {
		parentStyle = e.styleVal(n)
	}

	kids := make([]*html.Node, 0, len(n.Children))

	for idx := 0; idx < len(n.Children); idx++ {
		child := n.Children[idx]
		if child.Type == html.ElementNode {
			if e.stylePtr(child).Display == displayNone {
				continue
			}

			kids = append(kids, child)

			continue
		}

		if child.Type != html.TextNode || strings.TrimSpace(child.Text) == "" {
			continue
		}

		textNodes := []*html.Node{child}
		for idx+1 < len(n.Children) && n.Children[idx+1].Type == html.TextNode {
			idx++
			if strings.TrimSpace(n.Children[idx].Text) != "" {
				textNodes = append(textNodes, n.Children[idx])
			}
		}

		anonymous := &html.Node{ //nolint:exhaustruct // synthetic anonymous multicol item
			Type: html.ElementNode, Name: "span", Parent: n, Children: textNodes,
			Attrs: map[string]string{"data-gowk-anon": "multicol"},
		}
		anonStyle := anonymousMulticolItemStyle(parentStyle)
		e.styles[anonymous] = &anonStyle
		kids = append(kids, anonymous)
	}

	return kids
}

// anonymousMulticolItemStyle inherits text props from the multicol container
// and resets the box/layout props so the anonymous item is a plain block.
func anonymousMulticolItemStyle(parent ResolvedStyle) ResolvedStyle {
	style := anonymousFlexItemStyle(parent)
	style.ColumnCount = 0
	style.ColumnWidth = -1
	style.ColumnSpan = ""
	style.ColumnFill = ""
	style.ColumnGap = 0
	style.ColumnGapNormal = false

	return style
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
// the accumulated content height of a multicol container. A definite height
// both floors and caps the used height so oversized column strips cannot blow
// up table-row pagination into blank pages.
func clampMulticolHeight(curY float64, style ResolvedStyle, eng *engine) float64 {
	curY += eng.scalePt(style.PaddingBottom)
	if h, ok := resolveUsedHeight(style, -1, eng); ok {
		curY = h
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
// Explicit ColumnGap wins over the generic Gap shorthand so the result is
// deterministic even when map iteration order is stable-random (style_cascade
// nondeterminism lives outside this package).
func (e *engine) multicolGap(st ResolvedStyle) float64 {
	if !st.ColumnGapNormal {
		return e.scalePt(st.ColumnGap)
	}

	if st.Gap != 0 {
		return e.scalePt(st.Gap)
	}

	return e.scalePt(st.FontSize)
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

	// Single anonymous text item: fragment its line strip across columns.
	if nCols > 1 && len(items) == 1 && isAnonymousMulticolItem(items[0].n) {
		maxColH, snappedCy, snap := e.multicolColumnHeight(items, 0, style, balance, definiteH, padTop, yPos, curY, pageH)
		if snap {
			curY = snappedCy
			maxColH, _, _ = e.multicolColumnHeight(items, 0, style, balance, definiteH, padTop, yPos, curY, pageH)
		}
		lineTop := yPos + curY
		lineH := e.placeMulticolAnonColumns(
			parent, items[0], style, nCols, colW, gap, contentX, yPos, curY, maxColH, balance,
		)
		e.emitColumnRules(style, contentX, colW, gap, nCols, lineTop, lineH)

		return curY + lineH
	}

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

		lineTop := yPos + curY
		lineH := e.placeMulticolLine(
			parent, batch, style, nCols, colW, gap, contentX, yPos, curY, maxColH, balance, totalH,
		)
		e.emitColumnRules(style, contentX, colW, gap, nCols, lineTop, lineH)
		curY += lineH
	}

	return curY
}

func isAnonymousMulticolItem(n *html.Node) bool {
	return n != nil && n.Type == html.ElementNode && n.Attribute("data-gowk-anon") == "multicol"
}

// placeMulticolAnonColumns lays out one anonymous text item as a single-column
// strip at colW, then shifts line bands into subsequent columns (balance or
// auto fill against maxColH). Returns the used multicol-line height.
func (e *engine) placeMulticolAnonColumns(
	parent *box, item multicolItem, style ResolvedStyle,
	nCols int, colW, gap, contentX, yPos, curY, maxColH float64, balance bool,
) float64 {
	if nCols < 1 {
		nCols = 1
	}
	cblock := e.build(item.n, colW, contentX, yPos+curY)
	if cblock == nil {
		return 0
	}
	if parent != nil {
		parent.children = append(parent.children, cblock)
	}

	top := yPos + curY
	totalH := cblock.height
	if totalH <= 0 {
		return 0
	}

	// Balanced auto-height: even split of the strip. When maxColH is finite
	// (definite height or remaining page), never let the band exceed it —
	// otherwise totalH/nCols from a tall single-column strip creates multi-
	// page blank table rows.
	bandH := totalH / float64(nCols)
	if !balance && maxColH > 0 {
		bandH = maxColH
	}
	if maxColH > 0 && bandH > maxColH {
		bandH = maxColH
	}
	if bandH <= 0 {
		bandH = totalH
	}

	// Use the box op range: prependChrome may insert before len-at-build-start.
	opStart, opEnd := cblock.opStart, cblock.opEnd
	type colAssign struct {
		idx int
		col int
	}
	assigns := make([]colAssign, 0, opEnd-opStart+1)
	// Track ink (text/bullet) tops only. Border/background ops sit at the
	// box top and must not become the alignment anchor, or column 2+ gets
	// pulled through the padding into the border.
	colMinY := make([]float64, nCols)
	for i := range colMinY {
		colMinY[i] = math.Inf(1)
	}
	for k := opStart; k <= opEnd && k < len(e.ops); k++ {
		relY := e.ops[k].Y - top
		if relY < 0 {
			relY = 0
		}
		col := int(relY / bandH)
		if col < 0 {
			col = 0
		}
		if col >= nCols {
			// Past the last column band: mark for clip (no shift).
			assigns = append(assigns, colAssign{idx: k, col: -1})
			continue
		}
		if col > 0 {
			e.ops[k].X += float64(col) * (colW + gap)
			e.ops[k].Y -= float64(col) * bandH
		}
		if (e.ops[k].Kind == OpText || e.ops[k].Kind == OpBullet) && e.ops[k].Y < colMinY[col] {
			colMinY[col] = e.ops[k].Y
		}
		assigns = append(assigns, colAssign{idx: k, col: col})
	}
	// Top-align column 2+ to column 0's first ink line. Band cuts often leave a
	// mid-line orphan at the top of column 2+; anchoring that remnant to the
	// box top (or keeping it) pulls real lines through the padding/border
	// (fixture-61 #23/#24/#32). Drop orphans above column 0's first line, then
	// shift the remaining column ink up to that anchor.
	anchor := colMinY[0]
	if math.IsInf(anchor, 1) {
		anchor = top
	}
	for _, a := range assigns {
		if a.col < 1 {
			continue
		}
		op := &e.ops[a.idx]
		if (op.Kind == OpText || op.Kind == OpBullet) && op.Y < anchor-0.01 {
			DeactivateOp(op) // drop mid-band orphan
		}
	}
	for i := 1; i < nCols; i++ {
		colMinY[i] = math.Inf(1)
	}
	for _, a := range assigns {
		if a.col < 1 {
			continue
		}
		op := e.ops[a.idx]
		if (op.Kind == OpText || op.Kind == OpBullet) && op.Y < colMinY[a.col] {
			colMinY[a.col] = op.Y
		}
	}
	for _, a := range assigns {
		if a.col < 1 || math.IsInf(colMinY[a.col], 1) {
			continue
		}
		e.ops[a.idx].Y -= colMinY[a.col] - anchor
	}

	used := bandH
	if balance && maxColH <= 0 {
		// Auto-height balance: column height is the even split (or leftover).
		if rem := totalH - bandH*float64(nCols-1); rem > used {
			used = rem
		}
	}
	if maxColH > 0 && used > maxColH {
		used = maxColH
	}
	// Drop ink that still sits below the used column height so definite-height
	// demos (fixture-61 #23/#32) do not spill into the next table row.
	// Never blank OpLine/OpStrokeRect here: zeroing with Op{} would turn them
	// into OpFillRect (iota 0), and truncating vertical lines can eat chrome
	// that shares the child op range.
	limit := top + used
	for _, a := range assigns {
		op := &e.ops[a.idx]
		if op.Kind == OpLine || op.Kind == OpStrokeRect {
			continue
		}
		if a.col < 0 || op.Y >= limit-0.01 {
			DeactivateOp(op)
			continue
		}
		if (op.Kind == OpFillRect || op.Kind == OpImage) && op.Y+op.H > limit {
			op.H = limit - op.Y
			if op.H < 0 {
				op.H = 0
			}
		}
	}
	// Child box must not keep the unfragmented strip height; table/pagination
	// walk children and would otherwise reserve blank pages.
	cblock.height = used

	return used
}

// measureMulticolItems measures each in-flow element child at the column width.
// Anonymous text wrappers from multicolKids are elements, so they are included.
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

	if definiteH >= 0 {
		// Definite height caps column capacity for both balance and auto-fill.
		maxColH = clampMulticolRemainder(maxColH, definiteH-(curY-padTop))
	}

	// Oversized atomic items snap to the next page when mid-page. Anonymous
	// multicol text fragments across columns, so a tall measured strip must
	// not force a page snap.
	if items[idx].h > maxColH+1e-6 && absTop > pageTop+1e-6 &&
		!isAnonymousMulticolItem(items[idx].n) {
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

// emitColumnRules paints the column rules between columns for one multicol
// line. The rule is centered in the gap and spans lineH.
func (e *engine) emitColumnRules(style ResolvedStyle, contentX, colW, gap float64, nCols int, yTop, lineH float64) {
	if e == nil || e.noEmit || nCols <= 1 || lineH <= 0 {
		return
	}

	if style.ColumnRuleStyle == "" || style.ColumnRuleStyle == cssDisplayNone {
		return
	}

	ruleW := e.scalePt(style.ColumnRuleWidth)
	if ruleW <= 0 {
		return
	}

	var col [3]float64
	if style.ColumnRuleColorSet {
		col = style.ColumnRuleColor
	} else {
		col = style.Color
	}

	for i := 0; i < nCols-1; i++ {
		ruleX := contentX + float64(i+1)*(colW+gap) - gap/2
		e.emitBorderLine(ruleX, yTop, 0, lineH, ruleW, style.ColumnRuleStyle, col[0], col[1], col[2])
	}
}
