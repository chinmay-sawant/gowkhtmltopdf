package layout

import (
	"math"
)

// maxFlowPageIndex is the exclusive upper bound for the outer pagination-index
// slices. Coordinates beyond it are rejected instead of being aliased into a
// final catch-all bucket.
const maxFlowPageIndex = 16384

// shiftFlowY moves the ops of the target range [from,to] - plus every op
// strictly below fromY - down by deltaY canvas points. Ops of earlier boxes
// that touch fromY exactly (collapsed margins) are left alone so the
// page-break fixpoint converges instead of dragging boundary ops along each
// iteration. Box.y is kept in sync for boxes whose top moved.
func shiftFlowY(res *Result, from, toIdx int, fromY, deltaY float64) {
	shiftFlowBounded(res, from, toIdx, fromY, math.Inf(1), deltaY)
}

// shiftFlowBounded is shiftFlowY that leaves ops and boxes at or below
// beforeY unmoved (except the target range). Implicit keep-together uses
// the next page-break-before:always as beforeY so later sections do not
// cascade extra pages.
func shiftFlowBounded(res *Result, from, toIdx int, fromY, beforeY, deltaY float64) {
	if res == nil || len(res.Ops) == 0 || deltaY == 0 {
		return
	}

	ensureFlowIndex(res, flowIndexPageSize(res))

	startPage, ok := checkedFlowPageOfY(fromY, res.flowPageSize)
	if !ok || len(res.flowPageOf) != len(res.Ops) {
		invalidateFlowIndex(res)

		return
	}

	shiftOpsRange(res, from, toIdx, deltaY)

	shiftFlowOps(res, from, toIdx, fromY, beforeY, deltaY, startPage)

	if res.root == nil {
		return
	}

	if len(res.flowBoxes) == 0 {
		ensureFlowBoxIndex(res, flowBoxList(res))
	}

	shiftFlowBoxes(res, from, toIdx, fromY, beforeY, startPage, deltaY)
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
func shiftFlowOps(res *Result, from, toIdx int, fromY, beforeY, deltaY float64, startPage int) {
	if deltaY > 0 {
		for p := len(res.flowPages) - 1; p >= startPage; p-- {
			shiftOpsBucket(res, p, from, toIdx, fromY, beforeY, deltaY)
		}

		return
	}

	for p := startPage; p < len(res.flowPages); p++ {
		shiftOpsBucket(res, p, from, toIdx, fromY, beforeY, deltaY)
	}
}

// shiftOpsBucket shifts the ops of one page bucket that sit strictly below
// fromY. Removing the current item in shiftIndexedOp swaps the bucket's last
// item into its place; re-read the live bucket each step so a stale slice
// header cannot re-process an op that already left the page (that bug caused
// double negative shifts and infinite positive-shift loops).
//
//nolint:cyclop // page-bucket shift algorithm
func shiftOpsBucket(res *Result, page, from, toIdx int, fromY, beforeY, deltaY float64) {
	if page < 0 || page >= len(res.flowPages) {
		return
	}

	for jdx := 0; ; {
		bucket := res.flowPages[page]
		if jdx >= len(bucket) {
			return
		}

		idx := bucket[jdx]
		if idx < 0 || idx >= len(res.Ops) || idx >= len(res.flowPageOf) {
			jdx++

			continue
		}

		if (idx >= from && idx <= toIdx) || res.Ops[idx].Y <= fromY || res.Ops[idx].Y >= beforeY {
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
func shiftFlowBoxes(res *Result, from, toIdx int, fromY, beforeY float64, startPage int, deltaY float64) {
	if deltaY > 0 {
		for p := len(res.flowBoxes) - 1; p >= startPage; p-- {
			shiftBoxesBucket(res, p, from, toIdx, fromY, beforeY, startPage, deltaY)
		}

		return
	}

	for p := startPage; p < len(res.flowBoxes); p++ {
		shiftBoxesBucket(res, p, from, toIdx, fromY, beforeY, startPage, deltaY)
	}
}

// shiftBoxesBucket shifts the boxes of one page bucket whose top moved.
// Re-reads res.flowBoxes[page] each step (same swap-remove hazard as ops).
func shiftBoxesBucket(res *Result, page, from, toIdx int, fromY, beforeY float64, startPage int, deltaY float64) {
	if page < 0 || page >= len(res.flowBoxes) {
		return
	}

	for jdx := 0; ; {
		bucket := res.flowBoxes[page]
		if jdx >= len(bucket) {
			return
		}

		boxIndex := bucket[jdx]
		if boxIndex < 0 || boxIndex >= len(res.boxes) || boxIndex >= len(res.flowBoxPage) {
			jdx++

			continue
		}

		if skipBoxShift(res, boxIndex, from, toIdx, fromY, beforeY, startPage) {
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
func skipBoxShift(res *Result, boxIndex, from, toIdx int, fromY, beforeY float64, startPage int) bool {
	if startPage != res.flowBoxPage[boxIndex] {
		return res.boxes[boxIndex].y >= beforeY
	}

	targetBox := res.boxes[boxIndex]
	if targetBox.y >= beforeY {
		return true
	}

	if targetBox.y > fromY {
		return false
	}

	if targetBox.y == fromY && targetBox.opStart >= from && targetBox.opEnd <= toIdx {
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

	var ok bool

	res.flowPages, res.flowPageOf, res.flowPos, ok = buildFlowOpIndex(res.Ops, pageSize)
	if !ok {
		invalidateFlowIndex(res)
		res.flowPageSize = pageSize

		return
	}

	boxes := res.boxes
	if len(boxes) == 0 && res.root != nil {
		boxes = make([]*box, 0)
		flattenBoxes(res.root, &boxes)
		res.boxes = boxes
	}

	ensureFlowBoxIndex(res, boxes)
}

// buildFlowOpIndex buckets non-fixed ops by their canvas page.
func buildFlowOpIndex(ops []Op, pageSize float64) ([][]int, []int, []int, bool) {
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

		page, ok := checkedFlowPageOfY(ops[idx].Y, pageSize)
		if !ok {
			return nil, nil, nil, false
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

	return pages, pageOf, pos, true
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

		page, ok := checkedFlowPageOfY(b.y, res.flowPageSize)
		if !ok {
			res.flowBoxes = nil
			res.flowBoxPage = nil
			res.flowBoxPos = nil

			return
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
	if index < 0 || index >= len(res.Ops) || index >= len(res.flowPageOf) || res.Ops[index].Fixed {
		return
	}

	oldPage := res.flowPageOf[index]
	res.Ops[index].Y += deltaY

	newPage, ok := checkedFlowPageOfY(res.Ops[index].Y, res.flowPageSize)
	if !ok {
		invalidateFlowIndex(res)

		return
	}

	if oldPage == newPage {
		return
	}

	removeFromFlowBucket(&res.flowPages, res.flowPos, oldPage, index)
	appendToFlowBucket(&res.flowPages, &res.flowPageOf, &res.flowPos, index, newPage)
}

func shiftIndexedBox(res *Result, index int, deltaY float64) {
	if index < 0 || index >= len(res.boxes) || index >= len(res.flowBoxPage) {
		return
	}

	b := res.boxes[index]
	oldPage := res.flowBoxPage[index]
	b.y += deltaY

	newPage, ok := checkedFlowPageOfY(b.y, res.flowPageSize)
	if !ok {
		invalidateFlowIndex(res)

		return
	}

	if oldPage == newPage {
		return
	}

	removeFromFlowBucket(&res.flowBoxes, res.flowBoxPos, oldPage, index)
	appendToFlowBucket(&res.flowBoxes, &res.flowBoxPage, &res.flowBoxPos, index, newPage)
}

// checkedFlowPageOfY maps a canvas Y to its page index and reports whether the
// index is safe for the bounded flow slices. Non-positive Y remains page zero
// for compatibility; invalid or oversized values are rejected explicitly.
func checkedFlowPageOfY(yCoord, pageSize float64) (int, bool) {
	if pageSize <= 0 || math.IsNaN(pageSize) || math.IsInf(pageSize, 0) || math.IsNaN(yCoord) || math.IsInf(yCoord, 0) {
		return 0, false
	}

	if yCoord <= 0 {
		return 0, true
	}

	page := yCoord / pageSize
	if page < 0 || page >= float64(maxFlowPageIndex) {
		return 0, false
	}

	return int(page), true
}

// removeFromFlowBucket swaps the entry out of its bucket (keeping cursor
// positions valid for shiftIndexedOp's in-place iteration).
func removeFromFlowBucket(buckets *[][]int, pos []int, page, index int) {
	if buckets == nil || page < 0 || page >= len(*buckets) || index < 0 || index >= len(pos) {
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
	if buckets == nil || pageOf == nil || pos == nil || index < 0 || index >= len(*pageOf) || index >= len(*pos) {
		return
	}

	if page < 0 || page >= maxFlowPageIndex {
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
