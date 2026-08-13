package layout

import (
	"math"
	"sort"

	"gowkhtmltopdf/internal/html"
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

	b := res.boxes[boxIndex]
	if b.y >= beforeY {
		return true
	}

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

// avoidInside walks post-order and moves page-break-inside:avoid boxes wholly
// to the next page when they span multiple pages but fit one content height.
// Aside callouts are lifted earlier by keepImplicitAsides (before text snap).
func avoidInside(res *Result, contentH float64) bool {
	var walk func(b *box, inTable bool) bool
	walk = func(boxNode *box, inTable bool) bool {
		changed := false
		childInTable := inTable || boxNode.kind == displayTable

		for _, c := range boxNode.children {
			if walk(c, childInTable) {
				changed = true
			}
		}

		// Table cells inherit the avoid policy from table-row pagination, but
		// moving cells independently splits the collapsed grid. rowsIntact
		// owns the row-level move and keeps the cell borders/text together.
		if !inTable && !boxInsideTable(boxNode) && boxNode.height > 0 &&
			boxNode.style != nil && boxNode.style.PageBreakInside == pageBreakAvoid {
			if keepTogetherForAvoid(res, boxNode, contentH) {
				changed = true
			}
		}

		return changed
	}

	return walk(res.root, false)
}

// keepImplicitAsides is the pre-snap pass: move aside callouts that overflow
// the remaining page height while their ops are still a rigid unit.
func keepImplicitAsides(res *Result, contentH float64) bool {
	if res == nil || res.root == nil || contentH <= 0 {
		return false
	}

	var walk func(b *box, inTable bool) bool
	walk = func(boxNode *box, inTable bool) bool {
		changed := false
		childInTable := inTable || boxNode.kind == displayTable

		for _, child := range boxNode.children {
			if walk(child, childInTable) {
				changed = true
			}
		}

		if !inTable && !boxInsideTable(boxNode) && boxNode.height > 0 && implicitAtomicBox(boxNode) {
			if keepTogetherForAvoid(res, boxNode, contentH) {
				changed = true
			}
		}

		return changed
	}

	return walk(res.root, false)
}

// implicitAtomicBox is an aside callout that should not start on one page
// and finish on the next when it fits a single content height. Sections,
// lists, and tables stay fragmentable so dense avoid-lists do not cascade.
func implicitAtomicBox(boxNode *box) bool {
	if boxNode == nil || boxNode.node == nil {
		return false
	}

	switch boxNode.node.Name {
	case "aside":
		return true
	default:
		return false
	}
}

// boxInsideTable reports whether a box belongs to a table subtree. Some table
// descendants are kept in row metadata instead of the normal child tree, so
// the DOM ancestry is the reliable guard for avoiding independent cell moves.
func boxInsideTable(boxNode *box) bool {
	if boxNode == nil || boxNode.node == nil {
		return false
	}

	for node := boxNode.node.Parent; node != nil; node = node.Parent {
		if isTableElement(node.Name) {
			return true
		}
	}

	return false
}

func isTableElement(name string) bool {
	switch name {
	case "table", "caption", "colgroup", "thead", "tbody", "tfoot", "tr", "td", "th":
		return true
	default:
		return false
	}
}

// keepTogetherForAvoid shifts one unbreakable box wholly to the next page
// when it straddles the boundary but fits a full page. Remaining-Y is the
// space left on the current page; if height > remaining the box moves.
// Returns whether a shift happened.
func keepTogetherForAvoid(res *Result, boxNode *box, contentH float64) bool {
	// Ink bottom is a canvas Y. Compare it to the border-box bottom so a
	// card low on the page is not treated as taller than the page.
	bottom := boxNode.y + boxNode.height
	if ink := boxInkExtent(res, boxNode); ink > bottom {
		bottom = ink
	}

	height := bottom - boxNode.y
	layoutOut := int(boxNode.y / contentH)
	hi := int(bottom / contentH)

	if hi <= layoutOut || height > contentH+0.01 {
		return false
	}

	remaining := float64(layoutOut+1)*contentH - boxNode.y
	if rejectKeepTogetherShift(boxNode, remaining, contentH) {
		return false
	}

	dy := float64(layoutOut+1)*contentH - boxNode.y
	if dy <= layoutSlack {
		return false
	}

	if implicitAtomicBox(boxNode) {
		beforeY := nextForcedBreakY(res, boxNode.y)
		if boxNode.y+height+dy > beforeY-layoutSlack {
			return false
		}

		shiftImplicitUnit(res, boxNode, dy, beforeY)
	} else {
		shiftFlowY(res, boxNode.opStart, boxNode.opEnd, boxNode.y, dy)
	}

	return true
}

// shiftImplicitUnit moves one aside and the in-section siblings after it
// (footer, trailing copy), stopping before the next page-break-before:always.
// Nested boxes under an already-shifted sibling only update box.y — their ops
// were shifted with the ancestor's op range (avoids double-shifting a progress
// bar inside a following paragraph, which collapsed its height to 0).
func shiftImplicitUnit(res *Result, boxNode *box, dy, beforeY float64) {
	if res == nil || boxNode == nil || dy == 0 {
		return
	}

	shiftOpsOnly(res, boxNode.opStart, boxNode.opEnd, dy)

	fromY := boxNode.y
	boxNode.y += dy

	shiftedRoots := make([]*box, 0, 8)

	for _, candidate := range flowBoxList(res) {
		if candidate == nil || candidate == boxNode {
			continue
		}

		if boxNode.node != nil && candidate.node != nil && isDescendantNode(candidate.node, boxNode.node) {
			candidate.y += dy

			continue
		}

		if math.IsInf(beforeY, 1) || candidate.y <= fromY || candidate.y >= beforeY {
			continue
		}

		if candidate.style != nil && candidate.style.PageBreakBefore == pageBreakAlways {
			continue
		}

		if ancestor := shiftedAncestor(candidate, shiftedRoots); ancestor != nil {
			// Ops already moved with the ancestor's opStart–opEnd range.
			candidate.y += dy

			continue
		}

		if candidate.opStart <= candidate.opEnd {
			shiftOpsOnly(res, candidate.opStart, candidate.opEnd, dy)
		}

		candidate.y += dy
		shiftedRoots = append(shiftedRoots, candidate)
	}
}

// shiftedAncestor returns a previously shifted root that contains candidate.
func shiftedAncestor(candidate *box, roots []*box) *box {
	if candidate == nil || candidate.node == nil {
		return nil
	}

	for _, root := range roots {
		if root == nil || root.node == nil {
			continue
		}

		if isDescendantNode(candidate.node, root.node) {
			return root
		}
	}

	return nil
}

// nextForcedBreakY is the canvas Y of the next page-break-before:always box
// after afterY, or +Inf if none follows.
func nextForcedBreakY(res *Result, afterY float64) float64 {
	next := math.Inf(1)
	if res == nil {
		return next
	}

	for _, candidate := range flowBoxList(res) {
		if candidate == nil || candidate.style == nil || candidate.style.PageBreakBefore != pageBreakAlways {
			continue
		}

		if candidate.y > afterY && candidate.y < next {
			next = candidate.y
		}
	}

	return next
}

// rejectKeepTogetherShift reports that moving the box would leave too much
// empty space on the current page. Atomic cards only lift when they start
// in the last keepTogetherMaxBlankRatio of the page; short avoid list items
// keep preferSplitOverBlank so wiki-style reference lists do not cascade.
func rejectKeepTogetherShift(boxNode *box, remaining, contentH float64) bool {
	if implicitAtomicBox(boxNode) {
		// Only lift a card that starts in the last band of the page.
		// Mid-page cards that barely overflow would blank too much and
		// shove following in-section content onto an extra page.
		return remaining > contentH*keepTogetherMaxBlankRatio
	}

	if preferSplitOverBlank(remaining, boxNode.height, contentH) {
		return true
	}

	// Large explicit-avoid boxes: prefer split when less than half the box
	// fits (rowspan tables / tall avoid blocks).
	return remaining < boxNode.height*halfRatio && boxNode.height > contentH*0.35
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
			outBox += opVisibleInkHeight(paintOp)
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

// opInkHeight keeps pagination geometry tied to the line box emitted during
// layout. Text operations carry the resolved CSS line-height in H; the
// default ratio is only for legacy/generated operations that did not record it.
func opInkHeight(paintOp Op) float64 {
	if paintOp.H > 0 {
		return paintOp.H
	}

	if paintOp.Kind == OpText || paintOp.Kind == OpBullet {
		return paintOp.Size * defaultLineHeightRatio
	}

	return 0
}

func opVisibleInkHeight(paintOp Op) float64 {
	if (paintOp.Kind == OpText || paintOp.Kind == OpBullet) && paintOp.InkDescent > 0 {
		return paintOp.InkDescent
	}

	return opInkHeight(paintOp)
}

// beforeAlways moves page-break-before:always boxes onto a fresh page after
// everything that precedes them. Targets are collected in document order and
// processed by ascending opStart. Forced-break dys are recorded on a difference
// array and applied to ops in one O(n) pass (plus O(boxes) live box updates per
// break). Flow indexes are rebuilt once at the end.
//
//nolint:wsl // break-difference bookkeeping
func beforeAlways( //nolint:gocognit,gocyclo,cyclop,funlen // break-difference bookkeeping
	res *Result, contentH float64,
) bool {
	if res == nil || res.root == nil || contentH <= 0 {
		return false
	}

	boxes := flowBoxList(res)
	targetBoxes := collectBeforeAlwaysBoxes(res.root)
	targets := make([]beforeAlwaysTarget, 0, len(targetBoxes))
	for _, boxNode := range targetBoxes {
		targets = append(targets, beforeAlwaysTarget{
			box:   boxNode,
			start: beforeAlwaysOpStart(boxNode, boxes, len(res.Ops)),
		})
	}
	if len(targets) == 0 {
		return false
	}

	if len(targets) > 1 {
		sort.SliceStable(targets, func(i, j int) bool {
			return targets[i].start < targets[j].start
		})
	}

	ops := res.Ops
	opCount := len(ops)
	// suffixDy[i] is the additional Y applied to all ops at indices >= i.
	suffixDy := make([]float64, opCount+1)
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

	for _, target := range targets {
		boxNode := target.box
		start := target.start
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
		// lastPage comes from every preceding op so the previous section's
		// tail stays put; only this box and later flow move.
		boxY := boxNode.y
		targetY, alreadyFresh := forcedBreakTargetY(boxY, maxEff, contentH)
		if alreadyFresh {
			continue
		}

		deltaY := targetY - boxY
		if math.Abs(deltaY) <= layoutSlack {
			continue
		}

		// Record op suffix shift; update boxes immediately (few relative to ops).
		if start < opCount {
			suffixDy[start] += deltaY
		}

		fromY := boxY
		for _, b := range boxes {
			if b == boxNode || b.y > fromY ||
				(b.y == fromY && b.opStart >= start) {
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
		// Pinned: thead clones sit at the end of the display list but belong
		// mid-document; suffix shifts must not drag them across page edges.
		if cum == 0 || ops[idx].Fixed || ops[idx].Pinned {
			continue
		}

		ops[idx].Y += cum
	}

	invalidateFlowIndex(res)
	ensureFlowIndex(res, contentH)

	return true
}

type beforeAlwaysTarget struct {
	box   *box
	start int
}

// forcedBreakTargetY is the canvas Y of the first page top after preceding
// ink. A coordinate epsilon below a page multiple is that next page's top;
// a real sliver still on the previous page is not.
func forcedBreakTargetY(boxY, maxEff, contentH float64) (float64, bool) {
	if contentH <= 0 {
		return boxY, true
	}

	lastPage := int(maxEff / contentH)
	if lastPage < 0 {
		lastPage = 0
	}

	pageOff := math.Mod(boxY, contentH)
	if pageOff < 0 {
		pageOff += contentH
	}

	loPage := int(boxY / contentH)
	if contentH-pageOff <= layoutSlack {
		loPage++
		pageOff = 0
	}

	onLaterPage := loPage > lastPage
	atPageTop := pageOff <= layoutSlack
	if onLaterPage && atPageTop {
		return boxY, true
	}

	if onLaterPage {
		return float64(loPage) * contentH, false
	}

	return float64(lastPage+1) * contentH, false
}

// beforeAlwaysOpStart returns the first operation after a forced-break box.
// Empty break markers have no own operation range; using their stale build
// start would shift a suffix from an earlier sibling (fixture-21/28).
func beforeAlwaysOpStart(boxNode *box, boxes []*box, opCount int) int {
	if boxNode.opStart >= 0 && boxNode.opStart <= boxNode.opEnd {
		return boxNode.opStart
	}

	for idx, candidate := range boxes {
		if candidate != boxNode {
			continue
		}

		for _, following := range boxes[idx+1:] {
			if following.opStart >= 0 && following.opStart <= following.opEnd {
				return following.opStart
			}
		}

		return opCount
	}

	return opCount
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
//
//nolint:varnamelen,wsl // sibling scan uses conventional index names
func nextFlowSibling(boxes []*box, i int) *box {
	if i < 0 || i >= len(boxes) {
		return nil
	}

	current := boxes[i]
	for j := i + 1; j < len(boxes); j++ {
		candidate := boxes[j]
		if current.node != nil && candidate.node != nil && isDescendantNode(candidate.node, current.node) {
			continue
		}

		if candidate.opStart <= candidate.opEnd {
			return candidate
		}
	}

	return nil
}

// isDescendantNode reports whether candidate is below ancestor in the DOM.
// The flattened box list is preorder, so page-break-after must skip every
// descendant before it can reach the following flow sibling.
func isDescendantNode(candidate, ancestor *html.Node) bool {
	for node := candidate.Parent; node != nil; node = node.Parent {
		if node == ancestor {
			return true
		}
	}

	return false
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
	// A following forced break is stronger than page-break-after:avoid on the
	// preceding box. Keeping the two boxes together would move the preceding
	// content onto the forced-break page, and the next fixpoint would then move
	// that page again, producing one extra page per iteration (fixture-03).
	if next.style.PageBreakBefore == pageBreakAlways {
		return false
	}

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

// normalizeTableRowGaps removes stale vertical gaps left when pagination first
// moves a row to the next page and a later fixpoint pulls the table back.
// Collapsed table rows should remain adjacent when both rows fit on one page.
//
//nolint:cyclop,wsl // table-row geometry checks intentionally stay together
func normalizeTableRowGaps(res *Result, contentH float64) {
	if res == nil || res.root == nil || contentH <= 0 {
		return
	}

	for _, table := range flowBoxList(res) {
		if table.kind != displayTable || len(table.rows) < 2 {
			continue
		}

		for rowIndex := 1; rowIndex < len(table.rows); rowIndex++ {
			if !rowWasPaginationShifted(table.rows[rowIndex]) {
				continue
			}
			_, _, previousTop, previousBottom, previousOK := rowOpGeometry(table.rows[rowIndex-1])

			first, last, currentTop, currentBottom, currentOK := rowOpGeometry(table.rows[rowIndex])

			if !previousOK || !currentOK || first < 0 || last < first {
				continue
			}

			previousPage := int(previousTop / contentH)
			currentPage := int(currentTop / contentH)
			if previousPage != currentPage || int(currentBottom/contentH) != currentPage {
				continue
			}

			gap := currentTop - previousBottom
			if gap <= layoutEpsilon {
				continue
			}

			shiftOpsOnly(res, first, last, -gap)
			shiftTableRowBoxes(table.rows[rowIndex], -gap)
		}
	}
}

func shiftTableRowBoxes(row []*box, deltaY float64) {
	for _, cell := range row {
		shiftTableBox(cell, deltaY)
	}
}

func rowWasPaginationShifted(row []*box) bool {
	for _, cell := range row {
		if cell != nil && cell.paginationShifted {
			return true
		}
	}

	return false
}

func shiftTableBox(boxNode *box, deltaY float64) {
	if boxNode == nil {
		return
	}

	boxNode.y += deltaY
	for _, child := range boxNode.children {
		shiftTableBox(child, deltaY)
	}
}

// shiftRowToPage moves a table row wholly to the next page when it spans
// multiple pages. Returns whether it moved.
//
//nolint:wsl // row shifting and marker updates must stay adjacent.
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
	for _, cell := range row {
		if cell != nil {
			cell.paginationShifted = true
		}
	}

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
	if !hasRoundedOwnChrome(res, boxNode) && preferSplitOverBlank(remaining, boxNode.height, contentH) {
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

// hasRoundedOwnChrome identifies a short block whose background and side rail
// must fragment together. Splitting these callouts leaves the vertical rail
// on the continuation page while the rounded fill starts below it.
//
//nolint:cyclop,wsl // ownership predicate intentionally mirrors paint chrome
func hasRoundedOwnChrome(res *Result, boxNode *box) bool {
	if res == nil || boxNode == nil || boxNode.style == nil || boxNode.opStart < 0 || boxNode.opEnd >= len(res.Ops) {
		return false
	}
	if boxNode.style.BorderRadius <= 0 && boxNode.style.BorderRadiusPercent <= 0 && boxNode.style.BGColor[3] <= 0 {
		return false
	}
	if boxNode.style.BorderLeft.Style != solidKeyword || boxNode.style.BorderLeft.Width <= 2 {
		return false
	}

	hasRail := false
	for idx := boxNode.opStart; idx <= boxNode.opEnd; idx++ {
		op := res.Ops[idx]
		if op.Kind == OpLine && op.W == 0 && op.H > 0 &&
			nearLayout(op.X, boxNode.x) && nearLayout(op.Y, boxNode.y) {
			hasRail = true
		}
	}

	return hasRail
}

// normalizeLeadingRoundedCallouts removes a leading continuation gap from a
// short rounded block when no text or image ink precedes it on that page.
//
//nolint:wsl // the 32pt band is the renderer's continuation-gap threshold
func normalizeLeadingRoundedCallouts(res *Result, contentH float64) {
	if res == nil || contentH <= 0 {
		return
	}

	for _, boxNode := range flowBoxList(res) {
		if boxNode.height <= 0 || boxNode.height > contentH*0.35 || !hasRoundedOwnChrome(res, boxNode) {
			continue
		}

		// Match pageBuckets / shiftSamePageFromY: boundary-aligned Y must not
		// fall into the previous page via float truncation.
		page, ok := checkedFlowPageOfY(boxNode.y+layoutEpsilon, contentH)
		if !ok {
			continue
		}

		pageTop := float64(page) * contentH
		leadingOffset := boxNode.y - pageTop
		if leadingOffset <= layoutEpsilon || leadingOffset > 32 {
			continue
		}

		// Same-page only. shiftFlowY would also pull later pages backward
		// and can undo page-break-before:always (fixture-56 domain-05).
		shiftSamePageFromY(res, boxNode.y-layoutSlack, page, contentH, pageTop-boxNode.y)
	}
}

// shiftSamePageFromY applies deltaY to ops and boxes at or below fromY that
// still sit on page. Later pages are left alone.
//
// Page membership uses Y+layoutEpsilon so a coordinate that is exactly
// k*contentH (a forced page-break target) is not classified as the previous
// page via float truncation. Without the bump, normalizeLeadingRoundedCallouts
// can pull the next section's ops backward while leaving its box.y on the
// intended page (fixture-56 domain-09 / page-14 underline).
func shiftSamePageFromY(res *Result, fromY float64, page int, contentH, deltaY float64) {
	if res == nil || deltaY == 0 || contentH <= 0 {
		return
	}

	for idx := range res.Ops {
		if res.Ops[idx].Fixed || res.Ops[idx].Y < fromY {
			continue
		}

		opPage, ok := checkedFlowPageOfY(res.Ops[idx].Y+layoutEpsilon, contentH)
		if !ok || opPage != page {
			continue
		}

		res.Ops[idx].Y += deltaY
	}

	for _, boxNode := range flowBoxList(res) {
		if boxNode.y < fromY {
			continue
		}

		boxPage, ok := checkedFlowPageOfY(boxNode.y+layoutEpsilon, contentH)
		if !ok || boxPage != page {
			continue
		}

		boxNode.y += deltaY
	}

	invalidateFlowIndex(res)
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
//
//nolint:cyclop // table header repetition algorithm
func repeatTableHeaderOnPages(res *Result, tblBox *box, contentH float64) {
	nHdr := tblBox.headerRows
	if nHdr > len(tblBox.rows) {
		nHdr = len(tblBox.rows)
	}

	hdrFirst, hdrLast, hdrTop, hdrH := rowSpan(tblBox.rows[:nHdr], res)
	if hdrFirst < 0 || hdrH <= 0 {
		return
	}

	firstPage, ok := checkedFlowPageOfY(tblBox.y, contentH)
	if !ok {
		return
	}

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

		placeSliverBodyBelowHeader(res, tblBox, pageTop, hdrH)
		cloneHeaderOps(res, hdrFirst, hdrLast, hdrTop, pageTop)
	}
}

// placeSliverBodyBelowHeader pulls a body row that sits in the sliver just
// above a continuation page (or under the repeated thead) down below the
// header without moving rows that are already clear of the band.
func placeSliverBodyBelowHeader(res *Result, tblBox *box, pageTop, hdrH float64) {
	from, to, top, bottom := sliverBodyOps(tblBox, pageTop, hdrH, res)
	if from < 0 || top < 0 {
		return
	}

	target := pageTop + hdrH + 6
	sliverH := bottom - top
	if sliverH < 1 {
		sliverH = 12
	}

	offFrom, officialTop := nextBodyAfterSliver(tblBox, pageTop, from, to, res)
	if offFrom >= 0 && officialTop >= 0 && officialTop < target+sliverH-0.5 {
		push := target + sliverH - officialTop
		if push > 0 {
			shiftFlowY(res, offFrom, offFrom, officialTop-layoutSlack, push)
		}
	}

	dy := target - top
	if math.Abs(dy) > layoutSlack {
		shiftOpsOnly(res, from, to, dy)
		shiftTableBoxesInOpRange(tblBox, from, to, dy)
	}
}

// sliverBodyOps is the body-row paint sitting in [pageTop-8, pageTop+hdrH).
func sliverBodyOps(tblBox *box, pageTop, hdrH float64, res *Result) (int, int, float64, float64) {
	from, to := -1, -1
	top, bottom := -1.0, -1.0
	lo := pageTop - 8
	hi := pageTop + hdrH

	for _, row := range tblBox.rows[tblBox.headerRows:] {
		first, last := rowOpRange(row)
		if first < 0 {
			continue
		}

		for idx := first; idx <= last && idx < len(res.Ops); idx++ {
			posY := res.Ops[idx].Y
			if posY < lo || posY >= hi {
				continue
			}

			if from < 0 || idx < from {
				from = idx
			}

			if idx > to {
				to = idx
			}

			if top < 0 || posY < top {
				top = posY
			}

			if bot := posY + opInkHeight(res.Ops[idx]); bot > bottom {
				bottom = bot
			}
		}
	}

	return from, to, top, bottom
}

// nextBodyAfterSliver is the first body op on this page that is not in the sliver.
func nextBodyAfterSliver(tblBox *box, pageTop float64, sliverFrom, sliverTo int, res *Result) (int, float64) {
	from := -1
	top := -1.0

	for _, row := range tblBox.rows[tblBox.headerRows:] {
		first, last := rowOpRange(row)
		if first < 0 {
			continue
		}

		for idx := first; idx <= last && idx < len(res.Ops); idx++ {
			if idx >= sliverFrom && idx <= sliverTo {
				continue
			}

			posY := res.Ops[idx].Y
			if posY < pageTop-0.5 {
				continue
			}

			if from < 0 || idx < from {
				from = idx
			}

			if top < 0 || posY < top {
				top = posY
			}
		}
	}

	return from, top
}

func shiftTableBoxesInOpRange(tblBox *box, from, to int, deltaY float64) {
	if tblBox == nil || deltaY == 0 {
		return
	}

	var walk func(*box)
	walk = func(boxNode *box) {
		if boxNode == nil {
			return
		}

		if boxNode.opStart <= boxNode.opEnd && boxNode.opStart >= from && boxNode.opEnd <= to {
			boxNode.y += deltaY
		}

		for _, child := range boxNode.children {
			walk(child)
		}
	}

	for _, row := range tblBox.rows {
		for _, cell := range row {
			walk(cell)
		}
	}
}

// headerContinuationPages is the set of pages holding table body rows.
func headerContinuationPages(tblBox *box, _ int, res *Result, contentH float64) map[int]bool {
	pages := map[int]bool{}

	for _, row := range tblBox.rows[tblBox.headerRows:] {
		top := rowYBounds(row, res)
		if top < 0 {
			continue
		}

		page, ok := checkedFlowPageOfY(top, contentH)
		if ok {
			pages[page] = true
			off := math.Mod(top, contentH)
			if off < 0 {
				off += contentH
			}

			if off > contentH-8 {
				nextTop := float64(page+1) * contentH
				if sliverFrom, _, sliverTop, _ := sliverBodyOps(tblBox, nextTop, 20, res); sliverFrom >= 0 && sliverTop >= 0 {
					pages[page+1] = true
				}
			}
		}
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

		top := rowYBounds(row, res)

		topPage, ok := checkedFlowPageOfY(top, contentH)
		if !ok || topPage < page {
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
// Clones are Pinned so later page-break-before suffix shifts (by op index)
// cannot pull them back onto the previous page.
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
		op.Pinned = true
		// Clones are in-flow page furniture, not position:fixed stamps.
		op.Fixed = false
		res.Ops[start+k-hdrFirst] = op
	}

	// Header clones extend the display list after the page index was built.
	// Drop the cached buckets so later pagination shifts see the new ops.
	invalidateFlowIndex(res)
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

func rowYBounds(row []*box, res *Result) float64 {
	first, last := rowOpRange(row)
	if first < 0 || first >= len(res.Ops) {
		return -1
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

	height := opInkHeight(paintOp)

	return posY, posY + height
}

// rowCellBounds widens the row band with the cells' own geometry.
func rowCellBounds(row []*box, top, bottom float64) float64 {
	for _, cell := range row {
		if cell.height > 0 && cell.y+cell.height > bottom {
			bottom = cell.y + cell.height
		}

		if cell.y < top && cell.y > 0 {
			top = cell.y
		}
	}

	return top
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
func rowBandExtent(row []*box, _ *Result) (int, int, float64, float64, bool) {
	first, last, top, bottom, ok := rowOpGeometry(row)
	if !ok {
		return -1, -1, 0, 0, false
	}

	return first, last, top, bottom, true
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
