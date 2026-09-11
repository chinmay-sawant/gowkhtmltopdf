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

// flowIndexStorage is one retained page-index backing set: the per-page op
// buckets, the op -> page / page-position maps, and a per-page count scratch.
// A Result keeps one set for the live flow index, one for the box index, and
// one scratch set for the forced fresh builds, so the four per-conversion
// bucket computations reuse storage instead of reallocating. Every build
// resets and refills the arrays; a bucket is reallocated only when it outgrows
// its high-water mark.
type flowIndexStorage struct {
	pages  [][]int
	pageOf []int
	pos    []int
	counts []int
}

// reset drops the retained backing arrays.
func (s *flowIndexStorage) reset() {
	*s = flowIndexStorage{} //nolint:exhaustruct // zero value clears the retained arrays
}

// resetIntBuffer reslices dst to length, allocating only when capacity is
// short, and clears the retained prefix.
func resetIntBuffer(dst []int, length int) []int {
	if cap(dst) < length {
		return make([]int, length)
	}

	dst = dst[:length]
	clear(dst)

	return dst
}

// resetPageBuckets reslices dst to length, releasing bucket references past
// the end and truncating every retained bucket to zero length.
func resetPageBuckets(dst [][]int, length int) [][]int {
	if cap(dst) < length {
		return make([][]int, length)
	}

	for page := length; page < len(dst); page++ {
		dst[page] = nil
	}

	dst = dst[:length]

	for page := range dst {
		dst[page] = dst[page][:0]
	}

	return dst
}

// sizePageBuckets gives every bucket at least the counted capacity, reusing
// the retained bucket when it is already large enough.
func sizePageBuckets(pages [][]int, counts []int) {
	for page := range pages {
		if cap(pages[page]) < counts[page] {
			pages[page] = make([]int, 0, counts[page])
		} else {
			pages[page] = pages[page][:0]
		}
	}
}

// invalidateFlowIndex drops the live page buckets so the next
// ensureFlowIndex rebuilds from current coordinates. The backing arrays stay
// in the Result stores for reuse; only the live slices are cleared, and
// callers keep detecting a missing index by length.
func invalidateFlowIndex(res *Result) {
	if res == nil {
		return
	}

	res.flowPages = nil
	res.flowPageOf = nil
	res.flowPos = nil
	res.flowBoxes = nil
	res.flowBoxPage = nil
	res.flowBoxPos = nil
}

// releaseFlowIndex is the PDF-05 release point: it clears the live index and
// drops the retained backing arrays so PDF finalization cannot retain them.
// A later paint or pagination pass rebuilds from scratch.
func releaseFlowIndex(res *Result) {
	if res == nil {
		return
	}

	invalidateFlowIndex(res)

	res.flowStore.reset()
	res.flowBoxStore.reset()
	res.flowScratch.reset()
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

	if !buildPageIndex(res.Ops, pageSize, layoutEpsilon, &res.flowStore) {
		invalidateFlowIndex(res)
		res.flowPageSize = pageSize

		return
	}

	res.flowPages = res.flowStore.pages
	res.flowPageOf = res.flowStore.pageOf
	res.flowPos = res.flowStore.pos

	boxes := res.boxes
	if len(boxes) == 0 && res.root != nil {
		boxes = make([]*box, 0)
		flattenBoxes(res.root, &boxes)
		res.boxes = boxes
	}

	ensureFlowBoxIndex(res, boxes)
}

// buildPageIndex maps every non-fixed op to its canvas page and fills the
// exact-capacity page buckets into store. It is the single page-ownership
// mapping shared by the flow index and the painted page buckets. pageOf, pos,
// pages and the per-page buckets are reset and reused from store; only a
// first build (or one that outgrows the retained capacity) allocates.
//
// edgeBias selects the boundary policy: page ownership uses layoutEpsilon so
// a rect fragment that starts exactly at a page top (Y = k*contentH, whose
// float division can round just below k, e.g. (21*785.197)/785.197 =
// 20.9999...) stays on the page it starts. Fixed ops leave pageOf/pos at
// zero; every reader guards Fixed before use.
func buildPageIndex(ops []Op, pageSize, edgeBias float64, store *flowIndexStorage) bool {
	if store == nil {
		return false
	}

	pageOf := resetIntBuffer(store.pageOf, len(ops))

	maxPage := 0

	for idx := range ops {
		if ops[idx].Fixed {
			continue
		}

		page, ok := flowPageOfY(ops[idx].Y, pageSize, edgeBias)
		if !ok {
			return false
		}

		pageOf[idx] = page

		if page > maxPage {
			maxPage = page
		}
	}

	// Exact per-page capacity from a counting pass: appending into nil buckets
	// reallocated the whole per-page payload on every rebuild. Each bucket
	// allocates at most once, at its high-water mark, and is reused after that.
	pages := resetPageBuckets(store.pages, maxPage+1)
	counts := resetIntBuffer(store.counts, maxPage+1)

	for idx := range ops {
		if ops[idx].Fixed {
			continue
		}

		counts[pageOf[idx]]++
	}

	sizePageBuckets(pages, counts)

	pos := resetIntBuffer(store.pos, len(ops))

	for idx := range ops {
		if ops[idx].Fixed {
			continue
		}

		page := pageOf[idx]
		pos[idx] = len(pages[page])
		pages[page] = append(pages[page], idx)
	}

	store.pageOf = pageOf
	store.pages = pages
	store.pos = pos
	store.counts = counts

	return true
}

func ensureFlowBoxIndex(res *Result, boxes []*box) {
	if res == nil {
		return
	}

	if len(res.flowBoxPage) == len(boxes) && len(res.flowBoxes) > 0 {
		return
	}

	store := &res.flowBoxStore

	pageOf := resetIntBuffer(store.pageOf, len(boxes))

	maxPage := 0

	for idx, b := range boxes {
		b.flowIndex = idx

		page, ok := checkedFlowPageOfY(b.y, res.flowPageSize)
		if !ok {
			res.flowBoxes = nil
			res.flowBoxPage = nil
			res.flowBoxPos = nil

			return
		}

		pageOf[idx] = page

		if page > maxPage {
			maxPage = page
		}
	}

	counts := resetIntBuffer(store.counts, maxPage+1)

	for idx := range boxes {
		counts[pageOf[idx]]++
	}

	pages := resetPageBuckets(store.pages, maxPage+1)
	sizePageBuckets(pages, counts)

	pos := resetIntBuffer(store.pos, len(boxes))

	for idx := range boxes {
		page := pageOf[idx]
		pos[idx] = len(pages[page])
		pages[page] = append(pages[page], idx)
	}

	store.pages = pages
	store.pageOf = pageOf
	store.pos = pos
	store.counts = counts

	res.flowBoxes = pages
	res.flowBoxPage = pageOf
	res.flowBoxPos = pos
}

func shiftIndexedOp(res *Result, index int, deltaY float64) {
	if index < 0 || index >= len(res.Ops) || index >= len(res.flowPageOf) || res.Ops[index].Fixed {
		return
	}

	oldPage := res.flowPageOf[index]
	res.Ops[index].Y += deltaY

	newPage, ok := flowPageOfY(res.Ops[index].Y, res.flowPageSize, layoutEpsilon)
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

// flowPageOfY is the one page-ownership mapping. edgeBias nudges a y that
// sits a hair below a page top onto the page it starts (rect fragments split
// at the boundary); pass zero for raw truncation.
func flowPageOfY(yCoord, pageSize, edgeBias float64) (int, bool) {
	return checkedFlowPageOfY(yCoord+edgeBias, pageSize)
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
