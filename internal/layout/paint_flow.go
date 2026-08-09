package layout

import (
	"math"
	"sort"
)

// shiftFlowY moves the ops of the target range [from,to] - plus every op
// strictly below fromY - down by deltaY canvas points. Ops of earlier boxes
// that touch fromY exactly (collapsed margins) are left alone so the
// page-break fixpoint converges instead of dragging boundary ops along each
// iteration. Box.y is kept in sync for boxes whose top moved.

func shiftFlowY(res *Result, from, toIdx int, fromY, deltaY float64) {
	if res == nil || len(res.Ops) == 0 || deltaY == 0 {
		return
	}

	ensureFlowIndex(res, flowIndexPageSize(res))

	shiftOpsRange(res, from, toIdx, deltaY)

	startPage := int(fromY / res.flowPageSize)
	if startPage < 0 {
		startPage = 0
	}

	shiftFlowOps(res, from, toIdx, fromY, deltaY, startPage)

	if res.root == nil {
		return
	}

	if len(res.flowBoxes) == 0 {
		ensureFlowBoxIndex(res, flowBoxList(res))
	}

	shiftFlowBoxes(res, from, toIdx, fromY, startPage, deltaY)
}

// invalidateFlowIndex drops cached page buckets so the next ensureFlowIndex
// rebuilds from current coordinates.
func invalidateFlowIndex(res *Result) {
	if res == nil {
		return
	}

	res.flowPageOf = nil
	res.flowPages = nil
	res.flowPos = nil
	res.flowBoxes = nil
	res.flowBoxPage = nil
	res.flowBoxPos = nil
}

// shiftOpsRange shifts the non-fixed ops of [from,to] by deltaY.
func shiftOpsRange(res *Result, from, to int, deltaY float64) {
	for i := from; i <= to; i++ {
		if i < 0 || i >= len(res.Ops) || res.Ops[i].Fixed {
			continue
		}

		shiftIndexedOp(res, i, deltaY)
	}
}

// flowBoxList returns the flattened box list, caching it on res.
func flowBoxList(res *Result) []*box {
	boxes := res.boxes
	if len(boxes) == 0 {
		boxes = make([]*box, 0)
		flattenBoxes(res.root, &boxes)
		res.boxes = boxes
	}

	return boxes
}

// shiftFlowOps moves the ops of every page bucket in the direction of the
// shift: positive shifts process buckets from the end so an operation moved
// to another bucket is never visited twice; negative shifts go ascending so
// an operation moved backward is not revisited.
func shiftFlowOps(res *Result, from, toIdx int, fromY, deltaY float64, startPage int) {
	if deltaY > 0 {
		for p := len(res.flowPages) - 1; p >= startPage; p-- {
			shiftOpsBucket(res, p, from, toIdx, fromY, deltaY)
		}

		return
	}

	for p := startPage; p < len(res.flowPages); p++ {
		shiftOpsBucket(res, p, from, toIdx, fromY, deltaY)
	}
}

// shiftOpsBucket shifts the ops of one page bucket that sit strictly below
// fromY. Removing the current item in shiftIndexedOp swaps the bucket's last
// item into its place; re-read the live bucket each step so a stale slice
// header cannot re-process an op that already left the page (that bug caused
// double negative shifts and infinite positive-shift loops).
func shiftOpsBucket(res *Result, page, from, toIdx int, fromY, deltaY float64) {
	if page < 0 || page >= len(res.flowPages) {
		return
	}

	for jdx := 0; ; {
		bucket := res.flowPages[page]
		if jdx >= len(bucket) {
			return
		}

		idx := bucket[jdx]
		if (idx >= from && idx <= toIdx) || res.Ops[idx].Y <= fromY {
			jdx++

			continue
		}

		oldPage := res.flowPageOf[idx]
		shiftIndexedOp(res, idx, deltaY)

		if res.flowPageOf[idx] == oldPage {
			jdx++
		}
	}
}

// shiftFlowBoxes moves the box buckets in the direction of the shift.
func shiftFlowBoxes(res *Result, from, toIdx int, fromY float64, startPage int, deltaY float64) {
	if deltaY > 0 {
		for p := len(res.flowBoxes) - 1; p >= startPage; p-- {
			shiftBoxesBucket(res, p, from, toIdx, fromY, startPage, deltaY)
		}

		return
	}

	for p := startPage; p < len(res.flowBoxes); p++ {
		shiftBoxesBucket(res, p, from, toIdx, fromY, startPage, deltaY)
	}
}

// shiftBoxesBucket shifts the boxes of one page bucket whose top moved.
// Re-reads res.flowBoxes[page] each step (same swap-remove hazard as ops).
func shiftBoxesBucket(res *Result, page, from, toIdx int, fromY float64, startPage int, deltaY float64) {
	if page < 0 || page >= len(res.flowBoxes) {
		return
	}

	for jdx := 0; ; {
		bucket := res.flowBoxes[page]
		if jdx >= len(bucket) {
			return
		}

		boxIndex := bucket[jdx]
		if skipBoxShift(res, boxIndex, from, toIdx, fromY, startPage) {
			jdx++

			continue
		}

		oldPage := res.flowBoxPage[boxIndex]
		shiftIndexedBox(res, boxIndex, deltaY)

		if res.flowBoxPage[boxIndex] == oldPage {
			jdx++
		}
	}
}

// skipBoxShift reports whether a box on startPage should stay put during a
// flow shift (top at/above fromY, except the target op range sitting on fromY).
func skipBoxShift(res *Result, boxIndex, from, toIdx int, fromY float64, startPage int) bool {
	if startPage != res.flowBoxPage[boxIndex] {
		return false
	}

	b := res.boxes[boxIndex]
	if b.y > fromY {
		return false
	}

	if b.y == fromY && b.opStart >= from && b.opEnd <= toIdx {
		return false
	}

	return true
}

func flowIndexPageSize(res *Result) float64 {
	if res.flowPageSize > 0 {
		return res.flowPageSize
	}

	return 1
}

func ensureFlowIndex(res *Result, pageSize float64) {
	if res == nil || len(res.Ops) == 0 || pageSize <= 0 {
		return
	}

	if res.flowPageSize == pageSize && len(res.flowPageOf) == len(res.Ops) {
		return
	}

	res.flowPageSize = pageSize
	res.flowPages, res.flowPageOf, res.flowPos = buildFlowOpIndex(res.Ops, pageSize)

	boxes := res.boxes
	if len(boxes) == 0 && res.root != nil {
		boxes = make([]*box, 0)
		flattenBoxes(res.root, &boxes)
		res.boxes = boxes
	}

	ensureFlowBoxIndex(res, boxes)
}

// buildFlowOpIndex buckets non-fixed ops by their canvas page.
func buildFlowOpIndex(ops []Op, pageSize float64) ([][]int, []int, []int) {
	// Page numbers are dense from 0..maxP, so counts index directly instead
	// of a per-page map (page buckets below are exact-capacity, no growth).
	// Fixed ops leave pageOf/pos at their zero values; every reader guards
	// Fixed before use, so no explicit fill is needed.
	maxPage := 0
	pageOf := make([]int, len(ops))

	for idx := range ops {
		if ops[idx].Fixed {
			continue
		}

		page := int(ops[idx].Y / pageSize)
		if page < 0 {
			page = 0
		}

		pageOf[idx] = page

		if page > maxPage {
			maxPage = page
		}
	}

	// Count-then-fill: page buckets get exact capacity instead of growing
	// geometrically from zero.
	counts := make([]int, maxPage+1)

	for idx := range ops {
		if ops[idx].Fixed {
			continue
		}

		counts[pageOf[idx]]++
	}

	pages := make([][]int, maxPage+1)
	for p := range counts {
		pages[p] = make([]int, 0, counts[p])
	}

	pos := make([]int, len(ops))

	for idx := range ops {
		if ops[idx].Fixed {
			continue
		}

		page := pageOf[idx]
		pos[idx] = len(pages[page])
		pages[page] = append(pages[page], idx)
	}

	return pages, pageOf, pos
}

func ensureFlowBoxIndex(res *Result, boxes []*box) {
	if res == nil {
		return
	}

	if len(res.flowBoxPage) == len(boxes) && len(res.flowBoxes) > 0 {
		return
	}

	res.flowBoxes = make([][]int, len(res.flowPages))
	res.flowBoxPage = make([]int, len(boxes))
	res.flowBoxPos = make([]int, len(boxes))

	for idx, b := range boxes {
		b.flowIndex = idx

		page := int(b.y / res.flowPageSize)
		if page < 0 {
			page = 0
		}

		for len(res.flowBoxes) <= page {
			res.flowBoxes = append(res.flowBoxes, nil)
		}

		res.flowBoxPage[idx] = page
		res.flowBoxPos[idx] = len(res.flowBoxes[page])
		res.flowBoxes[page] = append(res.flowBoxes[page], idx)
	}
}

func shiftIndexedOp(res *Result, index int, deltaY float64) {
	if index < 0 || index >= len(res.Ops) || res.Ops[index].Fixed {
		return
	}

	oldPage := res.flowPageOf[index]
	res.Ops[index].Y += deltaY

	newPage := flowPageOfY(res.Ops[index].Y, res.flowPageSize)
	if oldPage == newPage {
		return
	}

	removeFromFlowBucket(&res.flowPages, res.flowPos, oldPage, index)
	appendToFlowBucket(&res.flowPages, &res.flowPageOf, &res.flowPos, index, newPage)
}

func shiftIndexedBox(res *Result, index int, deltaY float64) {
	if index < 0 || index >= len(res.boxes) {
		return
	}

	b := res.boxes[index]
	oldPage := res.flowBoxPage[index]
	b.y += deltaY

	newPage := flowPageOfY(b.y, res.flowPageSize)
	if oldPage == newPage {
		return
	}

	removeFromFlowBucket(&res.flowBoxes, res.flowBoxPos, oldPage, index)
	appendToFlowBucket(&res.flowBoxes, &res.flowBoxPage, &res.flowBoxPos, index, newPage)
}

// flowPageOfY maps a canvas Y to its page index.
func flowPageOfY(y, pageSize float64) int {
	page := int(y / pageSize)
	if page < 0 {
		return 0
	}

	return page
}

// removeFromFlowBucket swaps the entry out of its bucket (keeping cursor
// positions valid for shiftIndexedOp's in-place iteration).
func removeFromFlowBucket(buckets *[][]int, pos []int, page, index int) {
	if buckets == nil || page < 0 || page >= len(*buckets) {
		return
	}

	bucket := (*buckets)[page]
	slot := pos[index]

	if slot < 0 || slot >= len(bucket) {
		return
	}

	last := bucket[len(bucket)-1]
	bucket[slot] = last
	pos[last] = slot
	(*buckets)[page] = bucket[:len(bucket)-1]
}

// appendToFlowBucket registers the entry in its new page bucket.
// buckets is a pointer so growing the outer page slice is visible to the caller.
func appendToFlowBucket(buckets *[][]int, pageOf *[]int, pos *[]int, index, page int) {
	if buckets == nil {
		return
	}

	for len(*buckets) <= page {
		*buckets = append(*buckets, nil)
	}

	(*pageOf)[index] = page
	(*pos)[index] = len((*buckets)[page])
	(*buckets)[page] = append((*buckets)[page], index)
}

// shiftOpsOnly moves ops in [from,to] by dy without dragging later flow.
// Used when rejoining a page-break-after:avoid box to a following box that
// already sits on the next page.
func shiftOpsOnly(res *Result, from, tOrigin int, deltaY float64) {
	if res == nil || len(res.Ops) == 0 || deltaY == 0 {
		return
	}

	ensureFlowIndex(res, flowIndexPageSize(res))

	for i := from; i <= tOrigin; i++ {
		if i < 0 || i >= len(res.Ops) || res.Ops[i].Fixed {
			continue
		}

		shiftIndexedOp(res, i, deltaY)
	}
}

// avoidInside walks post-order and moves page-break-inside: avoid boxes wholly
// to the next page when they span multiple pages but fit one content height.
func avoidInside(res *Result, contentH float64) bool {
	var walk func(b *box) bool
	walk = func(boxNode *box) bool {
		changed := false

		for _, c := range boxNode.children {
			if walk(c) {
				changed = true
			}
		}

		if boxNode.style.PageBreakInside == pageBreakAvoid && boxNode.height > 0 {
			if keepTogetherForAvoid(res, boxNode, contentH) {
				changed = true
			}
		}

		return changed
	}

	return walk(res.root)
}

// keepTogetherForAvoid shifts one page-break-inside:avoid box wholly to the
// next page when it straddles the boundary but fits a full page. Returns
// whether a shift happened.
func keepTogetherForAvoid(res *Result, boxNode *box, contentH float64) bool {
	height := boxNode.height
	// Prefer ink extent when taller than the border box (rowspan /
	// deferred paint can make ops protrude past b.h — wiki awards).
	if ink := boxInkExtent(res, boxNode); ink > height {
		height = ink
	}

	layoutOut := int(boxNode.y / contentH)
	hi := int((boxNode.y + height) / contentH)

	if hi <= layoutOut || height > contentH+0.01 {
		return false
	}

	remaining := float64(layoutOut+1)*contentH - boxNode.y
	// Prefer splitting over large empty bands. Use border-box
	// height (b.h), not ink: after line-snap, ink can span a
	// page gap while the box is still a short list item —
	// classifying by ink disabled the short-box guard and
	// cascaded 100–150pt gaps (wiki references).
	if preferSplitOverBlank(remaining, boxNode.height, contentH) {
		return false
	}
	// Large boxes: also prefer split when less than half the box
	// fits (rowspan tables / tall avoid blocks).
	if remaining < boxNode.height*0.5 && boxNode.height > contentH*0.35 {
		return false
	}

	dy := float64(layoutOut+1)*contentH - boxNode.y
	if dy <= layoutSlack {
		return false
	}

	shiftFlowY(res, boxNode.opStart, boxNode.opEnd, boxNode.y, dy)

	return true
}

// boxInkExtent returns the bottom edge of the box's ink ops (boxNode.y when
// the op range is invalid or holds no ink).
func boxInkExtent(res *Result, boxNode *box) float64 {
	if boxNode.opStart > boxNode.opEnd || boxNode.opStart < 0 || boxNode.opEnd >= len(res.Ops) {
		return boxNode.y
	}

	bot := boxNode.y

	for k := boxNode.opStart; k <= boxNode.opEnd; k++ {
		paintOp := res.Ops[k]
		outBox := paintOp.Y

		switch paintOp.Kind {
		case OpText, OpBullet:
			outBox += paintOp.Size * defaultLineHeightRatio
		case OpFillRect, OpStrokeRect, OpLine, OpImage, OpLinkURI, opKindNoop:
			if paintOp.H > 0 {
				outBox += paintOp.H
			}
		}

		if outBox > bot {
			bot = outBox
		}
	}

	return bot
}

// beforeAlways moves page-break-before:always boxes onto a fresh page after
// everything that precedes them. Targets are collected in document order and
// processed by ascending opStart. Forced-break dys are recorded on a difference
// array and applied to ops in one O(n) pass (plus O(boxes) live box updates per
// break). Flow indexes are rebuilt once at the end.
func beforeAlways(res *Result, contentH float64) bool { //nolint:gocognit,cyclop,funlen // break-difference bookkeeping
	if res == nil || res.root == nil || contentH <= 0 {
		return false
	}

	targets := collectBeforeAlwaysBoxes(res.root)
	if len(targets) == 0 {
		return false
	}

	if len(targets) > 1 {
		sort.SliceStable(targets, func(i, j int) bool {
			return targets[i].opStart < targets[j].opStart
		})
	}

	ops := res.Ops
	opCount := len(ops)
	// suffixDy[i] is the additional Y applied to all ops at indices >= i.
	suffixDy := make([]float64, opCount+1)
	boxes := flowBoxList(res)

	changed := false
	opIdx := 0
	// maxEff tracks max effective Y of ops already scanned (original Y +
	// cumulative suffix dys from earlier breaks that cover that op).
	maxEff := 0.0
	// curOff is sum of break dys with start <= the next op index to be read.
	curOff := 0.0
	// events with start <= opIdx have been folded into curOff via eventAt.
	eventAt := 0

	type breakEvent struct {
		start int
		dy    float64
	}

	events := make([]breakEvent, 0, len(targets))

	for _, boxNode := range targets {
		start := boxNode.opStart
		if start < 0 {
			start = 0
		}

		if start > opCount {
			start = opCount
		}

		for opIdx < start {
			for eventAt < len(events) && events[eventAt].start <= opIdx {
				curOff += events[eventAt].dy
				eventAt++
			}

			eff := ops[opIdx].Y + curOff
			if eff > maxEff {
				maxEff = eff
			}

			opIdx++
		}

		// Box Y has been kept live (updated when earlier breaks applied).
		boxY := boxNode.y
		loPage := int(boxY / contentH)
		lastPage := int(maxEff / contentH)

		if loPage > lastPage {
			continue
		}

		deltaY := float64(lastPage+1)*contentH - boxY
		if deltaY <= 0 {
			continue
		}

		// Record op suffix shift; update boxes immediately (few relative to ops).
		if start < opCount {
			suffixDy[start] += deltaY
		}

		fromY := boxY
		for _, b := range boxes {
			if b.y > fromY ||
				(b.y == fromY && b.opStart >= boxNode.opStart && b.opEnd <= boxNode.opEnd) {
				b.y += deltaY
			}
		}

		events = append(events, breakEvent{start: start, dy: deltaY})
		changed = true
	}

	if !changed {
		return false
	}

	// One linear apply of the difference array onto ops.
	cum := 0.0

	for idx := range opCount {
		cum += suffixDy[idx]
		if cum == 0 || ops[idx].Fixed {
			continue
		}

		ops[idx].Y += cum
	}

	invalidateFlowIndex(res)
	ensureFlowIndex(res, contentH)

	return true
}

// collectBeforeAlwaysBoxes returns boxes with page-break-before:always in
// preorder (document order).
func collectBeforeAlwaysBoxes(root *box) []*box {
	var out []*box

	var walk func(*box)
	walk = func(boxNode *box) {
		if boxNode.style.PageBreakBefore == pageBreakAlways {
			out = append(out, boxNode)
		}

		for _, child := range boxNode.children {
			walk(child)
		}
	}
	walk(root)

	return out
}

// afterBreaks walks in document order and applies page-break-after:
// always|avoid to the box that follows each box in flow.
func afterBreaks(res *Result, contentH float64) bool {
	boxes := flowBoxList(res)

	changed := false

	for i, boxNode := range boxes {
		next := nextFlowSibling(boxes, i)
		if next == nil || boxNode.opStart > boxNode.opEnd {
			continue
		}

		lastPage := int(boxMaxOpY(res, boxNode) / contentH)

		switch boxNode.style.PageBreakAfter {
		case pageBreakAlways:
			if shiftAfterAlways(res, next, lastPage, contentH) {
				changed = true
			}
		case pageBreakAvoid:
			if keepAfterAvoid(res, boxNode, next, lastPage, contentH) {
				changed = true
			}
		}
	}

	return changed
}

// nextFlowSibling returns the first box after i that emitted ops.
func nextFlowSibling(boxes []*box, i int) *box {
	for j := i + 1; j < len(boxes); j++ {
		if boxes[j].opStart <= boxes[j].opEnd {
			return boxes[j]
		}
	}

	return nil
}

// boxMaxOpY returns the lowest Y of any op in the box's range.
func boxMaxOpY(res *Result, boxNode *box) float64 {
	lastY := res.Ops[boxNode.opStart].Y
	for k := boxNode.opStart + 1; k <= boxNode.opEnd; k++ {
		if res.Ops[k].Y > lastY {
			lastY = res.Ops[k].Y
		}
	}

	return lastY
}

// shiftAfterAlways pushes the next box to the page after this box's last op.
func shiftAfterAlways(res *Result, next *box, lastPage int, contentH float64) bool {
	dy := float64(lastPage+1)*contentH - next.y
	if dy <= 0 {
		return false
	}

	shiftFlowY(res, next.opStart, next.opEnd, next.y, dy)

	return true
}

// keepAfterAvoid keeps a page-break-after:avoid box with the following box
// across page boundaries: it clears the landing band, then sits the box just
// above the (possibly pushed) sibling. Returns whether it moved.
func keepAfterAvoid(res *Result, boxNode, next *box, lastPage int, contentH float64) bool {
	// Do NOT collapse natural flow spacing when they already share a
	// page (that pulled .keep boxes up onto paragraph baselines —
	// fixture-08 Forms index overlap).
	nextPage := int(next.y / contentH)
	if nextPage <= lastPage {
		return false
	}
	// Place the heading on next's page without a full-page shiftFlowY
	// (that blanked pages after avoid-inside tables). Clear the
	// page-top band first: paginateOps may already have snapped a
	// prior paragraph's continuation to pageStart — that text is
	// NOT `next` (next is the following sibling), so we must push
	// every op in the landing band, not only next.
	pageStart := float64(nextPage) * contentH

	need := avoidKeepBand(res, boxNode)

	const gap = 10.0
	need += gap
	bandTop := pageStart + need
	minY, minIdx := minOpYOnPage(res, nextPage, contentH, bandTop)

	if minIdx >= 0 && minY < bandTop-0.01 {
		push := bandTop - minY
		shiftFlowY(res, minIdx, minIdx, minY-layoutSlack, push)
	}

	target := bandTop - need // == pageStart when band was cleared
	if target < pageStart {
		target = pageStart
	}
	// Prefer sitting just above the (possibly pushed) next sibling.
	if next.y-need > target {
		target = next.y - need
	}

	if target < pageStart {
		target = pageStart
	}

	deltaY := target - boxNode.y
	if deltaY <= layoutSlackFine {
		return false
	}

	shiftOpsOnly(res, boxNode.opStart, boxNode.opEnd, deltaY)
	boxNode.y += deltaY

	return true
}

// minOpYOnPage finds the lowest non-fixed op on the page, starting from
// bandTop (returns -1 when none sit above it).
func minOpYOnPage(res *Result, nextPage int, contentH, bandTop float64) (float64, int) {
	minY := bandTop
	minIdx := -1

	for idx := range res.Ops {
		paintOp := &res.Ops[idx]
		if paintOp.Fixed {
			continue
		}

		if int(paintOp.Y/contentH) != nextPage {
			continue
		}

		if paintOp.Y < minY {
			minY = paintOp.Y
			minIdx = idx
		}
	}

	return minY, minIdx
}

// avoidKeepBand measures how much room the box needs on next's page (ink
// extent, else border-box height).
func avoidKeepBand(res *Result, boxNode *box) float64 {
	need := boxNode.height
	if need < 1 {
		need = 12
	}

	if ink := avoidKeepInkExtent(res, boxNode); ink > need {
		need = ink
	}

	return need
}

// avoidKeepInkExtent returns the ink band height of the box's ops (0 when the
// op range is invalid).
func avoidKeepInkExtent(res *Result, boxNode *box) float64 {
	if boxNode.opStart > boxNode.opEnd || boxNode.opStart < 0 || boxNode.opEnd >= len(res.Ops) {
		return 0
	}

	top, bot := boxNode.y, boxNode.y

	for k := boxNode.opStart; k <= boxNode.opEnd; k++ {
		yStart, yEnd := opInkEdges(res.Ops[k])

		if yStart < top {
			top = yStart
		}

		if yEnd > bot {
			bot = yEnd
		}
	}

	return bot - top
}

// opInkEdges returns the top and bottom ink edges of one op.
func opInkEdges(paintOp Op) (float64, float64) {
	yStart, yEnd := paintOp.Y, paintOp.Y

	switch paintOp.Kind {
	case OpText, OpBullet:
		yStart = paintOp.Y - paintOp.Size*ascentRatio
		yEnd = paintOp.Y + paintOp.Size*bulletGapRatio
	case OpLine:
		if paintOp.H == 0 {
			yEnd = paintOp.Y + math.Max(paintOp.Width, 1)
		} else {
			yEnd = paintOp.Y + paintOp.H
		}
	case OpFillRect, OpStrokeRect, OpImage, OpLinkURI, opKindNoop:
		if paintOp.H > 0 {
			yEnd = paintOp.Y + paintOp.H
		}
	}

	return yStart, yEnd
}

// rowsIntact keeps each table row on a single page: a row spanning multiple
// pages moves wholly to the next.
func rowsIntact(res *Result, contentH float64) bool {
	var walk func(b *box) bool
	walk = func(boxNode *box) bool {
		changed := false

		for _, c := range boxNode.children {
			if walk(c) {
				changed = true
			}
		}

		for _, row := range boxNode.rows {
			if shiftRowToPage(res, row, contentH) {
				changed = true
			}
		}

		return changed
	}

	return walk(res.root)
}

// shiftRowToPage moves a table row wholly to the next page when it spans
// multiple pages. Returns whether it moved.
func shiftRowToPage(res *Result, row []*box, contentH float64) bool {
	if len(row) == 0 {
		return false
	}

	first, last, rowTop, rowBottom, ok := rowOpGeometry(row)
	if !ok {
		return false
	}

	layoutOut, hi := int(rowTop/contentH), int(rowBottom/contentH)
	if hi <= layoutOut {
		return false
	}
	// Move only to the next page start. Using hi*contentH when the
	// row's measured bottom spans multiple pages (e.g. rowspan
	// paint height leaking into rowBoxH) skipped blank pages
	// between filmography and awards on long wiki tables.
	deltaY := float64(layoutOut+1)*contentH - rowTop
	if deltaY <= layoutSlack {
		return false
	}
	// fromY slightly above rowTop so border-collapse grid
	// lines that sit exactly on the row edge (and later
	// rows / chrome below) shift with the cells — otherwise
	// content moves and the grid stays behind (gapped /
	// misaligned music-video tables across page breaks).
	shiftFlowY(res, first, last, rowTop-layoutSlack, deltaY)

	return true
}

// rowOpGeometry returns the op range and starting-row geometry of a row:
// opStart..opEnd across its cells and the row band top/bottom (rowspan cells
// use their starting-row height, not the full span paint extent).
func rowOpGeometry(row []*box) (int, int, float64, float64, bool) {
	first, last := -1, -1
	haveGeom := false

	var rowTop, rowBottom float64

	for _, cell := range row {
		first, last = extendRowOpRange(first, last, cell)
		// Use starting-row geometry, not full rowspan paint extent.
		// Rowspan cells emit bottom borders at y+h (full span); scanning
		// those ops made the first row look multi-page and cascaded
		// blank pages (wiki awards tables with rowspan=10+).
		top, bot := rowCellGeometry(cell)

		if !haveGeom {
			rowTop, rowBottom, haveGeom = top, bot, true

			continue
		}

		if top < rowTop {
			rowTop = top
		}

		if bot > rowBottom {
			rowBottom = bot
		}
	}

	return first, last, rowTop, rowBottom, first >= 0 && haveGeom
}

// extendRowOpRange widens the row's op range with one cell's ops.
func extendRowOpRange(first, last int, cell *box) (int, int) {
	if cell.opStart > cell.opEnd {
		return first, last
	}

	if first < 0 {
		first = cell.opStart
	}

	if cell.opEnd > last {
		last = cell.opEnd
	}

	return first, last
}

// rowCellGeometry returns the starting-row band of one cell: rowspan cells
// use their starting-row height, not the full span paint extent.
func rowCellGeometry(cell *box) (float64, float64) {
	top := cell.y

	h := cell.height
	if cell.rowSpan > 1 && cell.rowBoxH > 0 {
		h = cell.rowBoxH
	}

	return top, top + h
}

// keepHeadingWithNext moves a heading to the next page when it would sit alone
// near the bottom (less than ~2 line-heights of room for following content).
func keepHeadingWithNext(res *Result, contentH float64) bool {
	if res.root == nil {
		return false
	}

	boxes := flowBoxList(res)

	changed := false

	for idx, boxNode := range boxes {
		if boxNode.node == nil || !isHeadingName(boxNode.node.Name) || boxNode.opStart > boxNode.opEnd {
			continue
		}

		page := int(boxNode.y / contentH)

		room := float64(page+1)*contentH - (boxNode.y + boxNode.height)
		if room >= twoLineRoomPt { // ~2 lines at 12pt
			continue
		}
		// Find next flow sibling with ops.
		next := nextBelowSibling(boxes, idx, boxNode.y)
		if next == nil {
			continue
		}

		nextPage := int(next.y / contentH)
		if nextPage > page {
			continue // already separated by a break
		}
		// Move heading + following content together so the heading does not
		// land on top of a line that already snapped to the next page.
		dy := float64(page+1)*contentH - boxNode.y
		if dy > 0 {
			shiftFlowY(res, boxNode.opStart, boxNode.opEnd, boxNode.y, dy)

			changed = true
		}
	}

	return changed
}

// nextBelowSibling returns the first flow sibling with ops at or below y.
func nextBelowSibling(boxes []*box, from int, y float64) *box {
	for j := from + 1; j < len(boxes); j++ {
		if boxes[j].opStart <= boxes[j].opEnd && boxes[j].y >= y {
			return boxes[j]
		}
	}

	return nil
}

func isHeadingName(name string) bool {
	switch name {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		return true
	}

	return false
}

// orphansWidows enforces CSS Fragmentation Level 3 Rule 3 (widows/orphans)
// when a leaf block has countable line boxes, and falls back to a geometric
// short-block heuristic when line counts are unavailable.
//
// Class B breaks are rejected when lines-before < orphans or lines-after <
// widows (or the block has fewer lines than orphans+widows can satisfy). If
// the whole block fits on the next page it is shifted; otherwise progress
// escape leaves the break (content taller than one page). Forced breaks
// (page-break-before/after: always) run earlier and are not undone here.
// break-inside: avoid remains higher priority via avoidInside.
func orphansWidows(res *Result, contentH float64) bool {
	if res.root == nil || contentH <= 0 {
		return false
	}

	changed := false

	var walk func(b *box)
	walk = func(boxNode *box) {
		for _, c := range boxNode.children {
			walk(c)
		}

		if orphansWidowsBox(res, boxNode, contentH) {
			changed = true
		}
	}
	walk(res.root)

	return changed
}

// orphansWidowsBox applies Rule 3 (or the geometric fallback) to one block
// box. Returns whether anything moved.
func orphansWidowsBox(res *Result, boxNode *box, contentH float64) bool {
	if boxNode.kind != displayBlock || boxNode.height <= 0 || boxNode.opStart > boxNode.opEnd {
		return false
	}
	// Nested block containers: children apply Rule 3; only heuristic on
	// short straddlers here.
	if hasNestedFlowChild(boxNode) {
		return orphansWidowsHeuristic(res, boxNode, contentH)
	}

	lines := countBlockLineYs(res, boxNode)
	if len(lines) == 0 {
		return orphansWidowsHeuristic(res, boxNode, contentH)
	}

	return enforceOrphansWidows(res, boxNode, lines, contentH)
}

// enforceOrphansWidows applies CSS Fragmentation Rule 3 (orphans/widows) to a
// leaf block that straddles a page boundary. Returns whether it moved.
func enforceOrphansWidows(res *Result, boxNode *box, lines []float64, contentH float64) bool {
	orphans, widows := resolveOrphansWidows(boxNode)

	layoutOut := int(boxNode.y / contentH)
	hIdx := int((boxNode.y + boxNode.height) / contentH)

	if hIdx <= layoutOut {
		return false
	}

	boundary := float64(layoutOut+1) * contentH
	before, after := countLinesAroundBoundary(lines, boundary)
	// Rule 3 applies to Class B breaks *between line boxes*. If all text
	// lines sit on one side of the boundary (only padding/bg straddles),
	// do not keep-together tall boxes — fall back to the short heuristic.
	if before == 0 || after == 0 {
		return orphansWidowsHeuristic(res, boxNode, contentH)
	}
	// Rule 3: legal Class B break only if both sides meet the minima.
	if before >= orphans && after >= widows {
		return false
	}
	// Keep the block together when it fits one page; else progress escape.
	// Same blank-band guard as avoidInside: do not open a large empty
	// region on the current page for a short keep-together.
	if boxNode.height > contentH+0.01 {
		return false
	}

	remaining := float64(layoutOut+1)*contentH - boxNode.y
	if preferSplitOverBlank(remaining, boxNode.height, contentH) {
		return false
	}

	dy := float64(hIdx)*contentH - boxNode.y
	if dy <= layoutEpsilon {
		return false
	}

	shiftFlowY(res, boxNode.opStart, boxNode.opEnd, boxNode.y, dy)

	return true
}

// resolveOrphansWidows returns the effective orphans/widows minima.
func resolveOrphansWidows(boxNode *box) (int, int) {
	orphans := boxNode.style.Orphans
	if orphans < 1 {
		orphans = defaultOrphansWidows
	}

	widows := boxNode.style.Widows
	if widows < 1 {
		widows = defaultOrphansWidows
	}

	return orphans, widows
}

// countLinesAroundBoundary splits line baselines into before/after counts.
func countLinesAroundBoundary(lines []float64, boundary float64) (int, int) {
	before, after := 0, 0

	for _, y := range lines {
		if y < boundary-1e-6 {
			before++
		} else {
			after++
		}
	}

	return before, after
}

// orphansWidowsHeuristic is the phase-18 geometric fallback: short blocks
// (~2–4 lines) that straddle a page boundary move wholly when they fit.
func orphansWidowsHeuristic(res *Result, boxNode *box, contentH float64) bool {
	if boxNode.height < 14 || boxNode.height > 60 {
		return false
	}

	layoutOut := int(boxNode.y / contentH)
	hIdx := int((boxNode.y + boxNode.height) / contentH)

	if hIdx <= layoutOut || boxNode.height > contentH {
		return false
	}

	remaining := float64(layoutOut+1)*contentH - boxNode.y
	if preferSplitOverBlank(remaining, boxNode.height, contentH) {
		return false
	}

	dy := float64(hIdx)*contentH - boxNode.y
	if dy <= layoutEpsilon {
		return false
	}

	shiftFlowY(res, boxNode.opStart, boxNode.opEnd, boxNode.y, dy)

	return true
}

// preferSplitOverBlank reports whether a keep-together shift would leave an
// unacceptably large empty band on the current page. Shared by avoidInside
// and orphans/widows so dense page-break-inside:avoid lists do not cascade
// expanding gaps between consecutive short blocks.
//
// h should be the border-box height (not ink): line-snap can inflate ink
// across a page boundary without making the box "tall".
func preferSplitOverBlank(remaining, height, contentH float64) bool {
	if contentH <= 0 {
		return false
	}
	// Never blank more than half a page to keep a box together.
	if remaining > contentH*0.5 {
		return true
	}
	// Short/medium boxes (list items, citations, cards ~1–4 lines): only
	// keep-together when nearly at the page end. Each keep-together does
	// shiftFlowY on following siblings; sequences of short avoid items
	// otherwise expand inter-item gaps by remaining on every fixpoint
	// iteration (wiki references left 26–38pt bands).
	if height > 0 && height < contentH*0.35 {
		// Allow at most ~1.2 line-heights of trailing blank (or half the
		// box), whichever is larger — true end-of-page overflow only.
		// Tighter than the prior 24pt/0.75h guard so modest remainders
		// never keep short avoid siblings apart.
		maxBlank := 14.0
		if height*0.5 > maxBlank {
			maxBlank = height * halfRatio
		}

		if remaining > maxBlank {
			return true
		}
	}

	return false
}

func hasNestedFlowChild(boxNode *box) bool {
	for _, c := range boxNode.children {
		if c.kind == displayBlock || c.kind == displayTable {
			return true
		}
	}

	return false
}

// countBlockLineYs returns distinct text/bullet baseline Y positions in the
// box's op range (approximate line boxes for an IFC leaf block).
func countBlockLineYs(res *Result, boxNode *box) []float64 {
	if res == nil || boxNode.opStart > boxNode.opEnd || boxNode.opStart < 0 {
		return nil
	}

	const eps = 0.5

	yCoords := make([]float64, 0, maxGlueEm)

	end := boxNode.opEnd
	if end >= len(res.Ops) {
		end = len(res.Ops) - 1
	}

	for i := boxNode.opStart; i <= end; i++ {
		paintOp := &res.Ops[i]
		if paintOp.Kind != OpText && paintOp.Kind != OpBullet {
			continue
		}

		if !hasLineY(yCoords, paintOp.Y, eps) {
			yCoords = append(yCoords, paintOp.Y)
		}
	}

	return yCoords
}

// hasLineY reports whether a baseline Y is already recorded within eps.
func hasLineY(yCoords []float64, y, eps float64) bool {
	for _, ey := range yCoords {
		if math.Abs(ey-y) <= eps {
			return true
		}
	}

	return false
}

// repeatTableHeaders clones thead row ops onto every page that continues a
// multi-page table body, shifting body content down by the header height.
// Nested tables: each table repeats only its own thead.
func repeatTableHeaders(res *Result, contentH float64) {
	if res.root == nil || contentH <= 0 {
		return
	}

	for _, tblBox := range tableBoxes(res.root) {
		repeatTableHeaderOnPages(res, tblBox, contentH)
	}
}

// tableBoxes collects every multi-page table with a header band.
func tableBoxes(root *box) []*box {
	var tables []*box

	var walk func(b *box)
	walk = func(b *box) {
		if b.kind == displayTable && b.headerRows > 0 && b.headerRows < len(b.rows) {
			tables = append(tables, b)
		}

		for _, c := range b.children {
			walk(c)
		}
	}
	walk(root)

	return tables
}

// repeatTableHeaderOnPages clones one table's thead onto its continuation
// pages, shifting the body rows down by the header height.
func repeatTableHeaderOnPages(res *Result, tblBox *box, contentH float64) {
	nHdr := tblBox.headerRows
	if nHdr > len(tblBox.rows) {
		nHdr = len(tblBox.rows)
	}

	hdrFirst, hdrLast, hdrTop, hdrH := rowSpan(tblBox.rows[:nHdr], res)
	if hdrFirst < 0 || hdrH <= 0 {
		return
	}

	firstPage := int(tblBox.y / contentH)
	pages := headerContinuationPages(tblBox, firstPage, res, contentH)

	for page := range pages {
		if page <= firstPage {
			continue
		}

		pageTop := float64(page) * contentH
		shiftFrom, shiftTo, bodyTop := tableBodyRange(tblBox, page, res, contentH)

		if shiftFrom >= 0 && bodyTop >= 0 && bodyTop < pageTop+hdrH-0.5 {
			dy := pageTop + hdrH - bodyTop
			if dy > 0 {
				shiftFlowY(res, shiftFrom, shiftTo, bodyTop-layoutSlack, dy)
			}
		}

		cloneHeaderOps(res, hdrFirst, hdrLast, hdrTop, pageTop)
	}
}

// headerContinuationPages is the set of pages holding table body rows.
func headerContinuationPages(tblBox *box, _ int, res *Result, contentH float64) map[int]bool {
	pages := map[int]bool{}

	for _, row := range tblBox.rows[tblBox.headerRows:] {
		top, _ := rowYBounds(row, res)
		if top < 0 {
			continue
		}

		pages[int(top/contentH)] = true
	}

	return pages
}

// tableBodyRange returns the op range and top Y of the body rows that start
// on the given page.
func tableBodyRange(tblBox *box, page int, res *Result, contentH float64) (int, int, float64) {
	shiftFrom, shiftTo := -1, -1
	bodyTop := -1.0

	for _, row := range tblBox.rows[tblBox.headerRows:] {
		face, lst := rowOpRange(row)
		if face < 0 {
			continue
		}

		top, _ := rowYBounds(row, res)
		if int(top/contentH) < page {
			continue
		}

		if bodyTop < 0 || top < bodyTop {
			bodyTop = top
		}

		if shiftFrom < 0 || face < shiftFrom {
			shiftFrom = face
		}

		if lst > shiftTo {
			shiftTo = lst
		}
	}

	return shiftFrom, shiftTo, bodyTop
}

// cloneHeaderOps copies the thead op range to the page top. The clone count
// is known up front, so the display list grows once instead of per op.
func cloneHeaderOps(res *Result, hdrFirst, hdrLast int, hdrTop, pageTop float64) {
	if hdrFirst < 0 || hdrFirst > hdrLast || hdrFirst >= len(res.Ops) {
		return
	}

	if hdrLast >= len(res.Ops) {
		hdrLast = len(res.Ops) - 1
	}

	start := len(res.Ops)
	res.Ops = append(res.Ops, make([]Op, hdrLast-hdrFirst+1)...)

	for k := hdrFirst; k <= hdrLast; k++ {
		op := res.Ops[k]
		op.Y = pageTop + (op.Y - hdrTop)
		res.Ops[start+k-hdrFirst] = op
	}
}

func rowOpRange(row []*box) (int, int) {
	first, last := -1, -1

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

func rowYBounds(row []*box, res *Result) (float64, float64) {
	first, last := rowOpRange(row)
	if first < 0 || first >= len(res.Ops) {
		return -1, -1
	}

	top, bottom := opBottomEdge(res.Ops[first])

	for k := first + 1; k <= last && k < len(res.Ops); k++ {
		posY, bot := opBottomEdge(res.Ops[k])
		if posY < top {
			top = posY
		}

		if bot > bottom {
			bottom = bot
		}
	}

	return rowCellBounds(row, top, bottom)
}

// opBottomEdge returns an op's top Y and approximate bottom Y.
func opBottomEdge(paintOp Op) (float64, float64) {
	posY := paintOp.Y

	height := paintOp.H
	if paintOp.Kind == OpText || paintOp.Kind == OpBullet {
		height = paintOp.Size * defaultLineHeightRatio
	}

	return posY, posY + height
}

// rowCellBounds widens the row band with the cells' own geometry.
func rowCellBounds(row []*box, top, bottom float64) (float64, float64) {
	for _, cell := range row {
		if cell.height > 0 && cell.y+cell.height > bottom {
			bottom = cell.y + cell.height
		}

		if cell.y < top && cell.y > 0 {
			top = cell.y
		}
	}

	return top, bottom
}

func rowSpan(rows [][]*box, res *Result) (int, int, float64, float64) {
	first, last, top, bottom, ok := rowsBandGeometry(rows, res)
	if !ok {
		return -1, -1, 0, 0
	}

	height := bottom - top
	if height >= 1 {
		return first, last, top, height
	}

	return first, last, top, sumRowHeights(rows)
}

// rowsBandGeometry returns the op range and Y band of the rows.
func rowsBandGeometry(rows [][]*box, res *Result) (int, int, float64, float64, bool) {
	first, last := -1, -1
	top, bottom := 0.0, 0.0
	set := false

	for _, row := range rows {
		rowFirst, rowLast, rowTop, rowBottom, rowOK := rowBandExtent(row, res)
		if !rowOK {
			continue
		}

		first, last = extendBandRange(first, last, rowFirst, rowLast)
		top, bottom, set = extendBandBounds(top, bottom, rowTop, rowBottom, set)
	}

	return first, last, top, bottom, first >= 0 && set
}

// rowBandExtent returns the op range and Y band of one row.
func rowBandExtent(row []*box, res *Result) (int, int, float64, float64, bool) {
	face, lst := rowOpRange(row)
	if face < 0 {
		return -1, -1, 0, 0, false
	}

	right, rowB := rowYBounds(row, res)
	if right < 0 {
		return -1, -1, 0, 0, false
	}

	return face, lst, right, rowB, true
}

// extendBandRange widens the band's op range with one row's range.
func extendBandRange(first, last, rowFirst, rowLast int) (int, int) {
	if first < 0 || rowFirst < first {
		first = rowFirst
	}

	if rowLast > last {
		last = rowLast
	}

	return first, last
}

// extendBandBounds widens the band's Y bounds with one row's band.
func extendBandBounds(top, bottom, rowTop, rowBottom float64, set bool) (float64, float64, bool) {
	if !set || rowTop < top {
		top = rowTop
	}

	if !set || rowBottom > bottom {
		bottom = rowBottom
	}

	return top, bottom, true
}

// sumRowHeights falls back to the sum of the cells' heights when the op band
// is too thin to measure (rows without op Y spread).
func sumRowHeights(rows [][]*box) float64 {
	height := 0.0

	for _, row := range rows {
		rowH := 0.0
		for _, cell := range row {
			if cell.height > rowH {
				rowH = cell.height
			}
		}

		height += rowH
	}

	return height
}

// populateLocations records the canvas rect and page of every element box in
// document order. A box's page is the page of its first op; boxes without ops
// (or before an op→page map exists) fall back to the page of their y position.
