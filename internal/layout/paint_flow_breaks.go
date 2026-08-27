package layout

import (
	"math"
	"sort"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

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

func shouldKeepImplicitAside(boxNode *box, inTable bool) bool {
	return !inTable && !boxInsideTable(boxNode) && boxNode.height > 0 && implicitAtomicBox(boxNode)
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

		if shouldKeepImplicitAside(boxNode, inTable) && keepTogetherForAvoid(res, boxNode, contentH) {
			changed = true
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
	case "table", htmlCaption, htmlColgroup, htmlThead, htmlTbody, htmlTfoot, "tr", "td", "th":
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

	deltaY := float64(layoutOut+1)*contentH - boxNode.y
	if deltaY <= layoutSlack {
		return false
	}

	if implicitAtomicBox(boxNode) {
		beforeY := nextForcedBreakY(res, boxNode.y)
		if boxNode.y+height+deltaY > beforeY-layoutSlack {
			return false
		}

		shiftImplicitUnit(res, boxNode, deltaY, beforeY)
	} else {
		shiftFlowY(res, boxNode.opStart, boxNode.opEnd, boxNode.y, deltaY)
	}

	return true
}

const initialShiftedRootsCap = 8

func isCandidateDescendant(candidate, boxNode *box) bool {
	return boxNode.node != nil && candidate.node != nil && isDescendantNode(candidate.node, boxNode.node)
}

func isCandidateOutOfImplicitScope(candidate *box, fromY, beforeY float64) bool {
	if math.IsInf(beforeY, 1) || candidate.y <= fromY || candidate.y >= beforeY {
		return true
	}

	return candidate.style != nil && candidate.style.PageBreakBefore == pageBreakAlways
}

func shiftImplicitCandidate(res *Result, candidate *box, deltaY float64, shiftedRoots []*box) []*box {
	if ancestor := shiftedAncestor(candidate, shiftedRoots); ancestor != nil {
		// Ops already moved with the ancestor's opStart-opEnd range.
		candidate.y += deltaY

		return shiftedRoots
	}

	if candidate.opStart <= candidate.opEnd {
		shiftOpsOnly(res, candidate.opStart, candidate.opEnd, deltaY)
	}

	candidate.y += deltaY

	return append(shiftedRoots, candidate)
}

// shiftImplicitUnit moves one aside and the in-section siblings after it
// (footer, trailing copy), stopping before the next page-break-before:always.
// Nested boxes under an already-shifted sibling only update box.y - their ops
// were shifted with the ancestor's op range (avoids double-shifting a progress
// bar inside a following paragraph, which collapsed its height to 0).
func shiftImplicitUnit(res *Result, boxNode *box, deltaY, beforeY float64) {
	if res == nil || boxNode == nil || deltaY == 0 {
		return
	}

	shiftOpsOnly(res, boxNode.opStart, boxNode.opEnd, deltaY)

	fromY := boxNode.y
	boxNode.y += deltaY

	shiftedRoots := make([]*box, 0, initialShiftedRootsCap)

	for _, candidate := range flowBoxList(res) {
		if candidate == nil || candidate == boxNode {
			continue
		}

		if isCandidateDescendant(candidate, boxNode) {
			candidate.y += deltaY

			continue
		}

		if isCandidateOutOfImplicitScope(candidate, fromY, beforeY) {
			continue
		}

		shiftedRoots = shiftImplicitCandidate(res, candidate, deltaY, shiftedRoots)
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

type beforeAlwaysTarget struct {
	box   *box
	start int
}

type breakEvent struct {
	start int
	dy    float64
}

// breakScanState encapsulates the scan state and break event accumulator
// used by beforeAlways to resolve forced page breaks.
type breakScanState struct {
	opIdx    int
	eventAt  int
	curOff   float64
	maxEff   float64
	events   []breakEvent
	suffixDy []float64
}

func newBreakScanState(opCount, targetCount int) *breakScanState {
	return &breakScanState{
		opIdx:    0,
		eventAt:  0,
		curOff:   0,
		maxEff:   0,
		events:   make([]breakEvent, 0, targetCount),
		suffixDy: make([]float64, opCount+1),
	}
}

func (s *breakScanState) advance(ops []Op, limit int) {
	for s.opIdx < limit {
		for s.eventAt < len(s.events) && s.events[s.eventAt].start <= s.opIdx {
			s.curOff += s.events[s.eventAt].dy
			s.eventAt++
		}

		eff := ops[s.opIdx].Y + s.curOff
		if eff > s.maxEff {
			s.maxEff = eff
		}

		s.opIdx++
	}
}

func (s *breakScanState) applyBreak(start int, deltaY float64, opCount int) {
	if start < opCount {
		s.suffixDy[start] += deltaY
	}

	s.events = append(s.events, breakEvent{start: start, dy: deltaY})
}

func shiftBoxesForForcedBreak(boxes []*box, targetBox *box, fromY, deltaY float64, start int) {
	for _, b := range boxes {
		if b == targetBox || b.y > fromY || (b.y == fromY && b.opStart >= start) {
			b.y += deltaY
		}
	}
}

func applySuffixDifferences(ops []Op, suffixDy []float64) {
	cum := 0.0

	for idx := range ops {
		cum += suffixDy[idx]
		// Pinned: thead clones sit at the end of the display list but belong
		// mid-document; suffix shifts must not drag them across page edges.
		if cum == 0 || ops[idx].Fixed || ops[idx].Pinned {
			continue
		}

		ops[idx].Y += cum
	}
}

func collectBeforeAlwaysTargets(root *box, boxes []*box, opCount int) []beforeAlwaysTarget {
	targetBoxes := collectBeforeAlwaysBoxes(root)
	targets := make([]beforeAlwaysTarget, 0, len(targetBoxes))

	for _, boxNode := range targetBoxes {
		targets = append(targets, beforeAlwaysTarget{
			box:   boxNode,
			start: beforeAlwaysOpStart(boxNode, boxes, opCount),
		})
	}

	if len(targets) > 1 {
		sort.SliceStable(targets, func(i, j int) bool {
			return targets[i].start < targets[j].start
		})
	}

	return targets
}

// beforeAlways moves page-break-before:always boxes onto a fresh page after
// everything that precedes them. Targets are collected in document order and
// processed by ascending opStart. Forced-break dys are recorded on a difference
// array and applied to ops in one O(n) pass (plus O(boxes) live box updates per
// break). Flow indexes are rebuilt once at the end.
func beforeAlways(res *Result, contentH float64) bool {
	if res == nil || res.root == nil || contentH <= 0 {
		return false
	}

	boxes := flowBoxList(res)
	opCount := len(res.Ops)
	targets := collectBeforeAlwaysTargets(res.root, boxes, opCount)

	if len(targets) == 0 {
		return false
	}

	state := newBreakScanState(opCount, len(targets))
	changed := false

	for _, target := range targets {
		if processBeforeAlwaysTarget(target, boxes, state, res.Ops, opCount, contentH) {
			changed = true
		}
	}

	if !changed {
		return false
	}

	applySuffixDifferences(res.Ops, state.suffixDy)
	invalidateFlowIndex(res)
	ensureFlowIndex(res, contentH)

	return true
}

func processBeforeAlwaysTarget(
	target beforeAlwaysTarget,
	boxes []*box,
	state *breakScanState,
	ops []Op,
	opCount int,
	contentH float64,
) bool {
	start := target.start
	if start < 0 {
		start = 0
	}

	if start > opCount {
		start = opCount
	}

	state.advance(ops, start)

	boxY := target.box.y
	targetY, alreadyFresh := forcedBreakTargetY(boxY, state.maxEff, contentH)

	if alreadyFresh {
		return false
	}

	deltaY := targetY - boxY
	if math.Abs(deltaY) <= layoutSlack {
		return false
	}

	state.applyBreak(start, deltaY, opCount)
	shiftBoxesForForcedBreak(boxes, target.box, boxY, deltaY, start)

	return true
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
	if onLaterPage && pageOff <= layoutSlack {
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
	// page (that pulled .keep boxes up onto paragraph baselines -
	// fixture-08 Forms index overlap).
	nextPage := int(next.y / contentH)
	if nextPage <= lastPage {
		return false
	}
	// Place the heading on next's page without a full-page shiftFlowY
	// (that blanked pages after avoid-inside tables). Clear the
	// page-top band first: paginateOps may already have snapped a
	// prior paragraph's continuation to pageStart - that text is
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

func shiftSamePageOps(res *Result, fromY float64, page int, contentH, deltaY float64) {
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
}

func shiftSamePageBoxes(res *Result, fromY float64, page int, contentH, deltaY float64) {
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

	shiftSamePageOps(res, fromY, page, contentH, deltaY)
	shiftSamePageBoxes(res, fromY, page, contentH, deltaY)
	invalidateFlowIndex(res)
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
	// Short/medium boxes (list items, citations, cards ~1-4 lines): only
	// keep-together when nearly at the page end. Each keep-together does
	// shiftFlowY on following siblings; sequences of short avoid items
	// otherwise expand inter-item gaps by remaining on every fixpoint
	// iteration (wiki references left 26-38pt bands).
	if height > 0 && height < contentH*0.35 {
		// Allow at most ~1.2 line-heights of trailing blank (or half the
		// box), whichever is larger - true end-of-page overflow only.
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
