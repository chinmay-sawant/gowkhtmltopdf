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
	if res != nil && res.skipInitialBeforeAlways {
		return nil
	}

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

		if res.hasAvoidInside && avoidInside(res, contentH) {
			changed = true
		}

		if beforeAlways(res, contentH) {
			changed = true
		}

		if res.hasAfterBreak && afterBreaks(res, contentH) {
			changed = true
		}

		if rowsIntact(res, contentH) {
			changed = true
		}

		if flexWrapLinesIntact(res, contentH) {
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

// flexWrapLinesIntact moves a wrapped flex line that does not fit the
// remaining page space wholly to the next page. CSS Flexbox 10.1, multi-line
// row container: a class A break opportunity sits between sibling flex lines,
// and "if a line doesn't fit on the page, and the line is not at the top of
// the page, move the line to the next page". Items keep their one band: a
// fill must not be sliced at the boundary while its text stays behind.
func flexWrapLinesIntact(res *Result, contentH float64) bool {
	if res == nil || contentH <= 0 {
		return false
	}

	changed := false

	for _, container := range flowBoxList(res) {
		if !isWrappedFlexContainer(container) {
			continue
		}

		if moveStraddlingFlexLines(res, container, contentH) {
			changed = true
		}
	}

	return changed
}

// isWrappedFlexContainer reports a flex formatting context whose items may
// wrap into multiple lines. Single-line rows keep the snap path's behavior.
func isWrappedFlexContainer(container *box) bool {
	if container == nil || container.style == nil ||
		container.style.Display != displayFlex && container.style.Display != displayInlineFlex {
		return false
	}

	return container.style.FlexWrap == fxWrap || container.style.FlexWrap == fxWrapRev
}

// flexLine holds one wrapped line's vertical bounds and its op span.
type flexLine struct {
	top, bottom float64
	opStart     int
	opEnd       int
}

// moveStraddlingFlexLines shifts the first wrapped line of container that
// crosses a page boundary and fits one content page. The fixpoint calls it
// again until lines settle, so each call works from live box positions.
func moveStraddlingFlexLines(res *Result, container *box, contentH float64) bool {
	for _, line := range flexLineBands(container) {
		boundary, ok := straddledFlexLineBoundary(line, contentH)
		if !ok {
			continue
		}

		// A straddling first line whose top coincides with the container top
		// carries the container with it; otherwise the container box would
		// stay above the boundary and orphansWidows would shift the whole
		// flow a second time.
		carryContainer := withinTol(container.y, line.top) &&
			container.opStart >= 0 && container.opStart <= line.opStart

		if carryContainer {
			from := line.opStart
			if container.opStart < from {
				from = container.opStart
			}

			shiftFlowY(res, from, line.opEnd, container.y, boundary-line.top)
			shiftLineBoxY(res, container, boundary-line.top)
		} else {
			shiftFlowY(res, line.opStart, line.opEnd, line.top, boundary-line.top)
		}

		return true
	}

	return false
}

// straddledFlexLineBoundary returns the boundary a line must move to when it
// straddles one and fits a page, and whether the line qualifies.
func straddledFlexLineBoundary(line flexLine, contentH float64) (float64, bool) {
	if line.bottom-line.top > contentH+layoutCoordEpsilon {
		return 0, false
	}

	page, ok := flowPageOfY(line.top, contentH, layoutEpsilon)
	if !ok {
		return 0, false
	}

	boundary := float64(page+1) * contentH
	if line.top >= boundary-layoutCoordEpsilon || line.bottom <= boundary+layoutCoordEpsilon {
		return 0, false
	}

	// A line already at the top of its page cannot move further.
	if line.top-float64(page)*contentH <= layoutCoordEpsilon {
		return 0, false
	}

	return boundary, true
}

// flexLineBands groups a flex container's item boxes into wrapped lines by
// shared top Y and returns each line's bounds and op span.
func flexLineBands(container *box) []flexLine {
	lines := make([]flexLine, 0, len(container.children))

	for _, item := range container.children {
		if item == nil {
			continue
		}

		if lineIdx := findFlexLineBand(lines, item.y); lineIdx >= 0 {
			growFlexLineBand(&lines[lineIdx], item)

			continue
		}

		lines = append(lines, flexLine{
			top: item.y, bottom: item.y + item.height,
			opStart: item.opStart, opEnd: item.opEnd,
		})
	}

	return lines
}

// findFlexLineBand returns the line whose top matches topY, or -1.
func findFlexLineBand(lines []flexLine, topY float64) int {
	for lineIdx := range lines {
		if withinTol(lines[lineIdx].top, topY) {
			return lineIdx
		}
	}

	return -1
}

// growFlexLineBand extends one line band with an item box.
func growFlexLineBand(line *flexLine, item *box) {
	if item.y+item.height > line.bottom {
		line.bottom = item.y + item.height
	}

	if item.opStart >= 0 && (line.opStart < 0 || item.opStart < line.opStart) {
		line.opStart = item.opStart
	}

	if item.opEnd > line.opEnd {
		line.opEnd = item.opEnd
	}
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
			// A crossing op inside a page-break-inside:avoid block that the
			// fixpoint will move wholly to the next page is left for that
			// shift. Snapping first moves the row text and its chrome while
			// the block box and frame stay behind; the later whole-block
			// shift then preserves the Chrome Flex section geometry.
			if paintOp.Y+opH > boundary+1e-9 && !avoidBoxWillKeepTogether(res, idx, contentH) {
				snapOpToBoundary(res, idx, paintOp, boundary)
			}
		case OpFillRect, OpStrokeRect, OpLine, OpGridRun, OpUnknown, opKindNoop:
		}
	}

	return nil
}

// avoidBoxWillKeepTogether reports whether a crossing op sits inside a
// page-break-inside:avoid block that keepTogetherForAvoid can move wholly to
// the next page. Such a block already cannot stay on the current page, so
// snapping its inner text first would only split the block's frame from the
// row; the fixpoint's own shift keeps box, content and chrome together.
func avoidBoxWillKeepTogether(res *Result, opIndex int, contentH float64) bool {
	return deepestKeepTogetherAvoidBox(res, res.root, opIndex, contentH) != nil
}

// deepestKeepTogetherAvoidBox returns the deepest avoid-inside box holding
// opIndex that the keep-together gate accepts, or nil.
func deepestKeepTogetherAvoidBox(res *Result, boxNode *box, opIndex int, contentH float64) *box {
	if res == nil || boxNode == nil || opIndex < boxNode.opStart || opIndex > boxNode.opEnd {
		return nil
	}

	for _, child := range boxNode.children {
		if found := deepestKeepTogetherAvoidBox(res, child, opIndex, contentH); found != nil {
			return found
		}
	}

	if keepTogetherAvoidFits(res, boxNode, contentH) {
		return boxNode
	}

	return nil
}

// keepTogetherAvoidFits mirrors the keepTogetherForAvoid move gate: the box
// must straddle a page boundary, fit one content page, and not prefer a split
// over a blank band.
func keepTogetherAvoidFits(res *Result, boxNode *box, contentH float64) bool {
	if boxNode.height <= 0 || boxInsideTable(boxNode) || !isAvoidInsideBreak(boxNode.style) {
		return false
	}

	bottom := boxNode.y + boxNode.height
	if ink := boxInkExtent(res, boxNode); ink > bottom {
		bottom = ink
	}

	height := bottom - boxNode.y

	layoutOut := int(boxNode.y / contentH)
	hi := int(bottom / contentH)

	if hi <= layoutOut || height > contentH+layoutCoordEpsilon {
		return false
	}

	remaining := float64(layoutOut+1)*contentH - boxNode.y

	return !rejectKeepTogetherShift(boxNode, remaining, contentH)
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
	//
	// Collect the line's sibling chrome before any shift: flex line
	// placement gives every item in a row one top Y (flex.go buildRowItems),
	// but shiftNearestOwnedChrome only visits the snapped op's own box path,
	// so a 50pt sibling fill stayed behind and splitCrossingRects staircased
	// the row. snapLineChrome returns the owned fills and
	// the boxes that must move with the line.
	oldY := paintOp.Y
	lineChrome, lineBoxes := snapLineChrome(res, idx, oldY)
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

	sweepSnapLineChrome(res, lineChrome, lineBoxes, deltaY)
}

// lineChromeRef pins one chrome op of a snapped line to its pre-shift Y, so
// the sweep skips ops an earlier repair already moved.
type lineChromeRef struct {
	index int
	oldY  float64
}

// lineBox pins a flex-line box to its pre-shift Y, so a later sweep does not
// move a box that an earlier repair (shiftFlowY's flex-line pass) already
// moved.
type lineBox struct {
	box  *box
	oldY float64
}

// snapLineChrome collects chrome ops owned by the boxes on the snapped op's
// flex line, plus those boxes themselves. A box joins the line when it is a
// child of a flex container, the path box is a flex item, and both tops agree
// within layout epsilon: flex line placement gives every item in a row one Y
// (flex.go buildRowItems), so the shared top is the line band. Ownership via
// opOwnedBy gates each candidate. Ops at or below the snapped baseline are
// skipped because shiftFlowY already carries them.
func snapLineChrome(res *Result, opIndex int, oldY float64) ([]lineChromeRef, []lineBox) {
	boxes := snapLineBoxes(res, opIndex)

	return lineChromeRefs(res, boxes, oldY), boxes
}

// snapLineBoxes resolves the boxes that share the snapped op's flex line: the
// op's own box plus every flex sibling whose top sits in the same Y band.
func snapLineBoxes(res *Result, opIndex int) []lineBox {
	if res == nil || res.root == nil {
		return nil
	}

	path := make([]*box, 0, flexLineItemPathCap)
	if !snapOpBoxPath(res.root, opIndex, &path) {
		return nil
	}

	boxes := make([]lineBox, 0, rowChromeCap)
	seen := make(map[*box]bool, rowChromeCap)

	for pathIndex := len(path) - 1; pathIndex > 0; pathIndex-- {
		boxNode := path[pathIndex]
		container := path[pathIndex-1]

		if !isFlexLineContainer(container) {
			continue
		}

		for _, sibling := range container.children {
			if !withinTol(sibling.y, boxNode.y) || seen[sibling] {
				continue
			}

			seen[sibling] = true

			boxes = append(boxes, lineBox{box: sibling, oldY: sibling.y})
		}
	}

	return boxes
}

// lineChromeRefs gathers the owned chrome above the snapped baseline for each
// line box. Tall flex item fills are the reason this pass exists: the
// rowChromeAbove height cap rejects them, and shiftNearestOwnedChrome only
// walks the snapped op's own path.
func lineChromeRefs(res *Result, boxes []lineBox, oldY float64) []lineChromeRef {
	refs := make([]lineChromeRef, 0, rowChromeCap)

	for _, line := range boxes {
		boxNode := line.box
		if boxNode.opStart < 0 || boxNode.opEnd < boxNode.opStart {
			continue
		}

		end := boxNode.opEnd
		if end >= len(res.Ops) {
			end = len(res.Ops) - 1
		}

		for chromeIdx := boxNode.opStart; chromeIdx <= end; chromeIdx++ {
			chromeOp := &res.Ops[chromeIdx]
			if chromeOp.Fixed || chromeOp.Y >= oldY-layoutCoordEpsilon {
				continue
			}

			if !opOwnedBy(chromeOp, boxNode, opOwnerChrome) {
				continue
			}

			refs = append(refs, lineChromeRef{index: chromeIdx, oldY: chromeOp.Y})
		}
	}

	return refs
}

// isFlexLineContainer reports whether a box establishes a flex formatting
// context, so its same-Y children form one flex line.
func isFlexLineContainer(container *box) bool {
	if container == nil || container.style == nil {
		return false
	}

	return container.style.Display == displayFlex || container.style.Display == displayInlineFlex
}

// snapOpBoxPath appends the chain from boxNode to the deepest box whose op
// range contains opIndex, including that deepest box. findBoxPathForOp pops
// the matched leaf, which is right for chrome repair (it starts at the
// parent) but wrong for line collection: the snapped text's own item box is
// the leaf and owns the fill that must join the line.
func snapOpBoxPath(boxNode *box, opIndex int, path *[]*box) bool {
	if boxNode == nil || opIndex < boxNode.opStart || opIndex > boxNode.opEnd {
		return false
	}

	*path = append(*path, boxNode)

	for _, child := range boxNode.children {
		if snapOpBoxPath(child, opIndex, path) {
			return true
		}
	}

	return true
}

// sweepSnapLineChrome moves the snapped line's sibling chrome by deltaY and
// then the line boxes themselves. Moving the boxes keeps box.y and chrome in
// step, so orphansWidows does not read each item as a fresh straddler and
// staircase the row one snap at a time.
func sweepSnapLineChrome(res *Result, refs []lineChromeRef, boxes []lineBox, deltaY float64) {
	if res == nil || deltaY == 0 {
		return
	}

	for _, ref := range refs {
		chromeOp := &res.Ops[ref.index]
		if !withinTol(chromeOp.Y, ref.oldY) {
			continue
		}

		shiftOpY(chromeOp, deltaY)
	}

	for _, line := range boxes {
		if !withinTol(line.box.y, line.oldY) {
			continue
		}

		shiftLineBoxY(res, line.box, deltaY)
	}
}

// shiftLineBoxY moves one flex-line box with its line. It rides the live flow
// box index so page buckets stay exact and falls back to a bare Y update when
// pagination runs without one.
func shiftLineBoxY(res *Result, boxNode *box, deltaY float64) {
	if res == nil || boxNode == nil || deltaY == 0 {
		return
	}

	index := boxNode.flowIndex
	if index >= 0 && index < len(res.boxes) && res.boxes[index] == boxNode && index < len(res.flowBoxPage) {
		shiftIndexedBox(res, index, deltaY)

		return
	}

	boxNode.y += deltaY
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
