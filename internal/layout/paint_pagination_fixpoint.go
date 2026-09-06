package layout

// paginateOps assigns every op a page. Crossing text/image/link ops snap to
// the next page boundary (taking following flow with them so row spacing is
// preserved); then page-break policies are applied as canvas-Y shifts; finally
// pages derive from the final Y positions. Rect-type ops crossing a boundary
// are split by Paint.
func paginateOps(res *Result, contentH float64) []int {
	ensureFlowIndex(res, contentH)
	// Resolve forced section starts before snapping text to provisional page
	// boundaries. Otherwise a row near the boundary of the unbroken flow can
	// move its text alone; a later page-break-before shift then leaves the
	// collapsed-table chrome behind at the old row position.
	for range 10 {
		if !beforeAlways(res, contentH) {
			break
		}
	}

	// Lift aside callouts that do not fit the remaining Y on this page
	// before snapCrossingTextOps splits their last lines off to the next
	// page top (that snap-then-shift left an internal gap in the card).
	for range 10 {
		if !keepImplicitAsides(res, contentH) {
			break
		}
	}

	snapCrossingTextOps(res, contentH)

	paginationFixpoint(res, contentH)
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
	for range 10 {
		if !beforeAlways(res, contentH) {
			break
		}
	}
	// Sticky is applied in Paint after rect splitting (see splitCrossingRects).
	opPage := make([]int, len(res.Ops))

	for opIdx := range res.Ops {
		page, ok := checkedFlowPageOfY(res.Ops[opIdx].Y, contentH)
		if !ok {
			opPage[opIdx] = -1
		} else {
			opPage[opIdx] = page
		}
	}

	return opPage
}

// paginationFixpoint runs the page-break policies until none of them move
// anything or the iteration cap is reached.
func paginationFixpoint(res *Result, contentH float64) {
	for range 10 {
		changed := avoidInside(res, contentH)
		if beforeAlways(res, contentH) {
			changed = true
		}

		if afterBreaks(res, contentH) {
			changed = true
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
}

// snapCrossingTextOps moves text/image/link ops that cross a page boundary to
// the next page (taking following flow with them so row spacing is preserved).
func snapCrossingTextOps(res *Result, contentH float64) {
	tableRanges := tablePaintRanges(res)

	for idx := range len(res.Ops) {
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
		case OpFillRect, OpStrokeRect, OpLine, opKindNoop:
		}
	}
}

type paintRange struct{ first, last int }

func tablePaintRanges(res *Result) []paintRange {
	if res == nil || res.root == nil {
		return nil
	}

	ranges := make([]paintRange, 0)

	for _, boxNode := range flowBoxList(res) {
		if boxNode.kind != displayTable || boxNode.opStart > boxNode.opEnd {
			continue
		}

		ranges = append(ranges, paintRange{first: boxNode.opStart, last: boxNode.opEnd})
	}

	return ranges
}

func opInPaintRange(index int, ranges []paintRange) bool {
	for _, span := range ranges {
		if index >= span.first && index <= span.last {
			return true
		}
	}

	return false
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
			o.Y += deltaY
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
			if idx == opIndex || !isOwnBoxChrome(*chromeOp, boxNode, boxNode.y+boxNode.height) ||
				chromeOp.Y >= oldY-layoutCoordEpsilon ||
				(oldY-chromeOp.Y > rowChromeBandTolerance && chromeOp.Kind != OpLine) {
				continue
			}

			chromeOp.Y += deltaY
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
