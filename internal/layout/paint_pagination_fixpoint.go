package layout

import (
	"cmp"
	"context"
	"fmt"
	"slices"
)

// paginateOps assigns every op a page. Crossing text/image/link ops snap to
// the next page boundary (taking following flow with them so row spacing is
// preserved); then page-break policies are applied as canvas-Y shifts; finally
// pages derive from the final Y positions. Rect-type ops crossing a boundary
// are split by Paint.
//
// It returns only an error: PaintContext rebuilds page buckets after rect
// splitting and sticky shifts (buildPagesAfterSplits), so a pre-split
// op-to-page slice here would be discarded. Tests that assert the settled
// pre-split assignment use paginateOpsForTest.
//
// ctx is polled once per fixpoint iteration and every 64 op slots; on
// cancellation it returns the error and the caller abandons the partially
// shifted display list.
func paginateOps(ctx context.Context, res *Result, contentH float64) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("layout: paginate ops: %w", err)
	}

	ensureFlowIndex(res, contentH)
	res.buildPaginationCensus()
	// Resolve forced section starts before snapping text to provisional page
	// boundaries. Otherwise a row near the boundary of the unbroken flow can
	// move its text alone; a later page-break-before shift then leaves the
	// collapsed-table chrome behind at the old row position.
	if err := settleBeforeAlways(ctx, res, contentH); err != nil {
		return err
	}

	// Lift aside callouts that do not fit the remaining Y on this page
	// before snapCrossingTextOps splits their last lines off to the next
	// page top (that snap-then-shift left an internal gap in the card).
	for range 10 {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("layout: paginate ops: %w", err)
		}

		if !keepImplicitAsides(res, contentH) {
			break
		}
	}

	if err := snapCrossingTextOps(ctx, res, contentH); err != nil {
		return fmt.Errorf("layout: paginate ops: %w", err)
	}

	if err := paginationFixpoint(ctx, res, contentH); err != nil {
		return err
	}

	// After flow has settled, clone <thead> onto continuation pages.
	// Blank avoid-list bands are controlled by preferSplitOverBlank during
	// the fixpoint above (former packAvoidGaps sibling packing was a no-op).
	repeatTableHeaders(res, contentH)
	// Page-break fixups can temporarily move a final row across a boundary and
	// later pull the table back, leaving that row's chrome at the old position.
	// Collapse those stale inter-row gaps before painting the table.
	normalizeTableRowGaps(res, contentH)
	// Header continuation shifts can reintroduce a small leading band above a
	// rounded security callout. Normalize after all flow shifts are complete.
	normalizeLeadingRoundedCallouts(res, contentH)
	// Forced breaks win over the callout pack: a same-page snap must not
	// leave page-break-before:always parked on the previous page.
	if err := settleBeforeAlways(ctx, res, contentH); err != nil {
		return err
	}
	// Sticky is applied in Paint after rect splitting (see splitCrossingRects).

	return nil
}

// settleBeforeAlways runs forced section-start resolution to a fixpoint:
// page-break-before shifts repeat until none move anything.
func settleBeforeAlways(ctx context.Context, res *Result, contentH float64) error {
	for range 10 {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("layout: paginate ops: %w", err)
		}

		if !beforeAlways(res, contentH) {
			break
		}
	}

	return nil
}

// paginationFixpoint runs the page-break policies until none of them move
// anything or the iteration cap is reached. ctx is checked once per
// iteration: each iteration is a full display-list pass, so a check keeps
// cancellation latency under one pass instead of ten.
//
//nolint:cyclop // policy gates add explicit branches to the fixpoint
func paginationFixpoint(ctx context.Context, res *Result, contentH float64) error {
	for range 10 {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("layout: pagination fixpoint: %w", err)
		}

		changed := false

		if res.hasAvoidInside {
			changed = avoidInside(res, contentH)
		}

		if beforeAlways(res, contentH) {
			changed = true
		}

		if res.hasAfterBreak {
			if afterBreaks(res, contentH) {
				changed = true
			}
		}

		if rowsIntact(res, contentH) {
			changed = true
		}

		if keepHeadingWithNext(res, contentH) {
			changed = true
		}

		if orphansWidows(res, contentH) {
			changed = true
		}

		if !changed {
			break
		}
	}

	return nil
}

// buildPaginationCensus records the style-only facts the fixpoint policies
// branch on, so a policy that cannot fire does not walk the whole box tree
// every iteration. The facts are static: they read styles and structure, not
// positions, so one walk per Paint is enough even when the fixpoint loops.
//
// hasAvoidInside mirrors avoidInside's guard exactly:
// !boxInsideTable(b) && b.height > 0 && isAvoidInsideBreak(b.style).
// hasAfterBreak is conservative: it is set whenever any box declares
// page-break-after: always|avoid, which is the only way afterBreaks can move
// anything.
func (res *Result) buildPaginationCensus() {
	if res == nil {
		return
	}

	res.hasAvoidInside = false
	res.hasAfterBreak = false

	if res.root == nil {
		return
	}

	for _, boxNode := range flowBoxList(res) {
		res.censusBox(boxNode)

		if res.hasAvoidInside && res.hasAfterBreak {
			return
		}
	}
}

// censusBox folds one box into the census flags.
func (res *Result) censusBox(boxNode *box) {
	if boxNode.style == nil {
		return
	}

	if !res.hasAvoidInside && boxNode.height > 0 &&
		!boxInsideTable(boxNode) && isAvoidInsideBreak(boxNode.style) {
		res.hasAvoidInside = true
	}

	switch boxNode.style.PageBreakAfter {
	case pageBreakAlways, pageBreakAvoid:
		res.hasAfterBreak = true
	}
}

// snapCrossingTextOps moves text/image/link ops that cross a page boundary to
// the next page (taking following flow with them so row spacing is preserved).
func snapCrossingTextOps(ctx context.Context, res *Result, contentH float64) error {
	poll := newCtxPoll(ctx)
	tableRanges := tablePaintRanges(res)

	for idx := range len(res.Ops) {
		if poll.poll() {
			return fmt.Errorf("snap crossing ops: %w", poll.err)
		}

		paintOp := &res.Ops[idx]
		if paintOp.Fixed || opInPaintRange(idx, tableRanges) {
			continue
		}

		switch paintOp.Kind {
		case OpText, OpBullet, OpImage, OpLinkURI:
			opH := opInkHeight(*paintOp)

			page, ok := checkedFlowPageOfY(paintOp.Y, contentH)
			if !ok {
				continue
			}

			boundary := float64(page+1) * contentH
			if paintOp.Y+opH > boundary+1e-9 {
				snapOpToBoundary(res, idx, paintOp, boundary)
			}
		case OpFillRect, OpStrokeRect, OpLine, OpGridRun, OpUnknown, opKindNoop:
		}
	}

	return nil
}

type paintRange struct{ first, last int }

// tablePaintRanges returns the table op spans prepared for stabbing queries:
// sorted by first index, with each last replaced by the prefix maximum. The
// caller's membership test is then a binary search plus one comparison, so
// 174,000 queries cost O(log tables) each instead of an 87M-check scan.
//
// The spans are nested or disjoint (a table's range contains its descendants),
// so the prefix maximum over first-sorted spans equals the union membership.
func tablePaintRanges(res *Result) []paintRange {
	if res == nil || res.root == nil {
		return nil
	}

	ranges := make([]paintRange, 0)

	for _, boxNode := range flowBoxList(res) {
		if boxNode.kind != boxKindTable || boxNode.opStart > boxNode.opEnd {
			continue
		}

		ranges = append(ranges, paintRange{first: boxNode.opStart, last: boxNode.opEnd})
	}

	slices.SortFunc(ranges, func(left, right paintRange) int {
		return cmp.Compare(left.first, right.first)
	})

	for idx := 1; idx < len(ranges); idx++ {
		if ranges[idx].last < ranges[idx-1].last {
			ranges[idx].last = ranges[idx-1].last
		}
	}

	return ranges
}

func opInPaintRange(index int, ranges []paintRange) bool {
	low, high := 0, len(ranges)-1
	candidate := -1

	for low <= high {
		mid := int(uint(low+high) >> 1) //nolint:gosec // bounded by the range slice length
		if ranges[mid].first <= index {
			candidate = mid
			low = mid + 1
		} else {
			high = mid - 1
		}
	}

	return candidate >= 0 && ranges[candidate].last >= index
}

// snapAscenderRatio reserves ascender room above snapped text so snapped
// lines do not paint into the top margin.
const snapAscenderRatio = 0.75

// minSnapLead is the smallest ascender lead when snapping text forward.
const minSnapLead = 8

// snapOpToBoundary shifts one crossing op (and the row chrome under it) onto
// the next page, leaving ascender room above the boundary.
func snapOpToBoundary(res *Result, idx int, paintOp *Op, boundary float64) {
	if boundary-paintOp.Y > layoutEpsilon {
		snapOpForward(res, idx, paintOp, boundary)

		return
	}

	paintOp.Y = boundary
}

// snapOpForward snaps the op forward to the next page with the row chrome
// that belongs to it, keeping the same-row fill tops above the text.
func snapOpForward(res *Result, idx int, paintOp *Op, boundary float64) {
	// Snap text (+ following flow). Same-row fills sit above the
	// baseline; include them in deltaY via minY so their tops clear
	// onto this page with the text (fixture-31 Row 28 white bg).
	// Keep chrome matching tight (one row) so table reports do
	// not inflate deltaY. Never clamp fill tops to `boundary` alone
	// - that collapses them onto the text Y and leaves section
	// gray showing through the ascent/padding band.
	oldY := paintOp.Y
	chrome, minY := rowChromeAbove(res, idx, oldY)
	// Leave room for ascenders above the baseline so snapped
	// lines do not paint into the top margin (page-4/5 bleed).
	lead := 0.0
	if paintOp.Kind == OpText || paintOp.Kind == OpBullet {
		lead = paintOp.Size * snapAscenderRatio
		if lead < minSnapLead {
			lead = minSnapLead
		}
	}

	deltaY := boundary + lead - minY
	shiftFlowY(res, idx, idx, oldY-layoutCoordEpsilon, deltaY)
	shiftNearestOwnedChrome(res, idx, oldY, deltaY)

	for _, j := range chrome {
		o := &res.Ops[j]
		if o.Y < oldY-layoutCoordEpsilon {
			shiftOpY(o, deltaY)
		}
	}
}

// shiftNearestOwnedChrome moves the nearest block's background/side rail with
// a text line that was snapped to the next page. Row-sized chrome is handled
// separately by rowChromeAbove; this covers callout rails and other block
// chrome that can span more than one row.
//
//nolint:cyclop,wsl,mnd // chrome ownership walk
func shiftNearestOwnedChrome(res *Result, opIndex int, oldY, deltaY float64) {
	if res == nil || res.root == nil || deltaY == 0 {
		return
	}

	path := make([]*box, 0, 8)
	if !findBoxPathForOp(res.root, opIndex, &path) {
		return
	}

	for pathIndex := len(path) - 1; pathIndex >= 0; pathIndex-- {
		boxNode := path[pathIndex]
		moved := false
		for idx := boxNode.opStart; idx <= boxNode.opEnd && idx < len(res.Ops); idx++ {
			chromeOp := &res.Ops[idx]
			if idx == opIndex || !opOwnedBy(chromeOp, boxNode, opOwnerChrome) ||
				chromeOp.Y >= oldY-layoutCoordEpsilon ||
				(oldY-chromeOp.Y > rowChromeBandTolerance && chromeOp.Kind != OpLine) {
				continue
			}

			shiftOpY(chromeOp, deltaY)
			moved = true
		}
		if moved {
			return
		}
	}
}

//nolint:wsl // recursive ownership walk keeps path mutation adjacent
func findBoxPathForOp(boxNode *box, opIndex int, path *[]*box) bool {
	if boxNode == nil || opIndex < boxNode.opStart || opIndex > boxNode.opEnd {
		return false
	}

	*path = append(*path, boxNode)
	for _, child := range boxNode.children {
		if findBoxPathForOp(child, opIndex, path) {
			return true
		}
	}
	*path = (*path)[:len(*path)-1]

	return true
}

// rowChromeAbove collects fill/stroke rects whose band touches oldY, with
// their minimum top Y. Chrome is row-tight (rows never split), so when the
// flow index is available the scan is limited to the op's page bucket; the
// page above is scanned too when the band reaches across the page top. A full
// display-list scan is the fallback.
func rowChromeAbove(res *Result, idx int, oldY float64) ([]int, float64) {
	chrome := make([]int, 0, rowChromeCap)

	minY := oldY

	//nolint:nestif // row chrome candidate resolution
	if res.flowPageSize > 0 {
		page, ok := checkedFlowPageOfY(oldY, res.flowPageSize)
		if !ok {
			return chrome, minY
		}

		if page < 0 {
			page = 0
		}

		if page < len(res.flowPages) {
			if page > 0 && oldY-float64(page)*res.flowPageSize < rowChromeBandTolerance {
				chrome, minY = appendRowChromeCandidates(chrome, res.Ops, res.flowPages[page-1], idx, oldY, minY)
			}

			chrome, minY = appendRowChromeCandidates(chrome, res.Ops, res.flowPages[page], idx, oldY, minY)

			return chrome, minY
		}
	}

	for jdx := range res.Ops {
		obj := &res.Ops[jdx]
		if !rowChromeBandCandidate(obj, jdx, idx, oldY) {
			continue
		}

		chrome = append(chrome, jdx)

		if obj.Y < minY {
			minY = obj.Y
		}
	}

	return chrome, minY
}

// appendRowChromeCandidates appends the band candidates of idxs to chrome,
// lowering minY for candidates whose top sits above oldY.
func appendRowChromeCandidates(chrome []int, ops []Op, idxs []int, idx int, oldY, minY float64) ([]int, float64) {
	for _, jdx := range idxs {
		obj := &ops[jdx]
		if !rowChromeBandCandidate(obj, jdx, idx, oldY) {
			continue
		}

		chrome = append(chrome, jdx)

		if obj.Y < minY {
			minY = obj.Y
		}
	}

	return chrome, minY
}

// rowChromeBandCandidate reports whether obj is a one-row fill/stroke rect
// sitting on the same band as the op at oldY (not the op itself).
func rowChromeBandCandidate(obj *Op, jdx, idx int, oldY float64) bool {
	if obj.Fixed || jdx == idx {
		return false
	}

	if obj.Kind != OpFillRect && obj.Kind != OpStrokeRect {
		return false
	}

	if obj.H <= 0.5 || obj.H > 40 {
		return false
	}

	if obj.Y > oldY+0.5 || obj.Y+obj.H < oldY-0.5 {
		return false
	}

	return oldY-obj.Y <= obj.H+2
}
