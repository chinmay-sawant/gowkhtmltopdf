//nolint:all
package layout

import (
	"math"
)

func calculateChromeInkBottom(res *Result, boxNode *box, oldBottom float64, owners map[int]*box) (float64, bool) {
	inkBottom := boxNode.y
	hasInk := false

	for idx := boxNode.opStart; idx <= boxNode.opEnd; idx++ {
		// An op that paints a later sibling's chrome is not this box's ink:
		// the swallowed op range would otherwise count a sibling rail as
		// content at the document bottom and inflate this box.
		if owner := owners[idx]; owner != nil && owner != boxNode {
			continue
		}

		operation := res.Ops[idx]
		// A displaced own border rule is frame chrome, not content ink: a
		// shift can leave the rule away from the box edge, opOwnedBy then
		// fails, and the rule would inflate the box height below its content.
		if opOwnedBy(&operation, boxNode, opOwnerChrome) || operation.Positioned ||
			ownFrameRuleShape(&operation, boxNode) || foreignFrameRail(&operation, boxNode) {
			continue
		}

		bottom := opInkBottom(operation)
		if bottom > inkBottom {
			inkBottom = bottom
			hasInk = true
		}
	}

	if boxIsFloatOrFigure(boxNode) {
		if childBottom := lastInFlowChildBottom(boxNode); childBottom > oldBottom {
			return childBottom, true
		}

		return inkBottom, false
	}

	if childBottom := lastInFlowChildBottom(boxNode); childBottom > inkBottom {
		return childBottom, true
	}

	return inkBottom, hasInk
}

// ownFrameRuleShape reports a full-width horizontal border rule of the box.
// The rule is frame chrome wherever pagination moved it: the box's own top
// and bottom border rules span its whole width, so no descendant rule shape
// can match unless the descendant is exactly as wide.
func ownFrameRuleShape(operation *Op, boxNode *box) bool {
	if operation == nil || boxNode == nil || boxNode.style == nil {
		return false
	}

	if operation.Kind != OpLine || operation.H != 0 || operation.W <= 0 {
		return false
	}

	if !nearLayout(operation.X, boxNode.x) || !nearLayout(operation.W, boxNode.w) {
		return false
	}

	style := boxNode.style

	topBorder := style.BorderTop.Width > 0 && style.BorderTop.Style != cssDisplayNone
	bottomBorder := style.BorderBottom.Width > 0 && style.BorderBottom.Style != cssDisplayNone

	return topBorder || bottomBorder
}

// foreignFrameRail reports a vertical rail that runs on this box's own left or
// right edge but lies wholly outside the box's vertical span. The rail cannot
// be this box's content ink; it is another box's frame chrome that a widened
// op range swept in (fixture-33: transformed rows splice their chrome early,
// so unionChildOpRanges covers the later siblings' rails).
func foreignFrameRail(operation *Op, boxNode *box) bool {
	if operation == nil || boxNode == nil || !opIsVerticalSide(operation) || operation.H <= 0 {
		return false
	}

	onLeft := nearLayout(operation.X, boxNode.x)
	onRight := nearLayout(operation.X, boxNode.x+boxNode.w)
	if !onLeft && !onRight {
		return false
	}

	tol := opOwnerTolerance(opOwnerChrome)

	return operation.Y > boxNode.y+boxNode.height+tol || operation.Y+operation.H < boxNode.y-tol
}

// frameRuleIndices returns the op indices of the box's full-width top and
// bottom rules, if present. The highest rule is the top border, the lowest
// the bottom border. A rule outside the box's vertical span by more than the
// bottom-match slack is another box's frame chrome in a widened op range,
// never this box's own edge (fixture-33).
func frameRuleIndices(ops []Op, boxNode *box, owners map[int]*box) (int, int) {
	topIdx, bottomIdx := -1, -1
	topY, bottomY := math.Inf(1), math.Inf(-1)

	for idx := boxNode.opStart; idx <= boxNode.opEnd && idx < len(ops); idx++ {
		// A sibling stacked on this box's edge paints a coincident rule that
		// ownFrameRuleShape cannot tell apart; the owners map can.
		if owner := owners[idx]; owner != nil && owner != boxNode {
			continue
		}

		if !ownFrameRuleShape(&ops[idx], boxNode) ||
			ops[idx].Y < boxNode.y-boxBottomMatchSlack ||
			ops[idx].Y > boxNode.y+boxNode.height+boxBottomMatchSlack {
			continue
		}

		if ops[idx].Y < topY {
			topY, topIdx = ops[idx].Y, idx
		}

		if ops[idx].Y > bottomY {
			bottomY, bottomIdx = ops[idx].Y, idx
		}
	}

	return topIdx, bottomIdx
}

// realignOwnFrameRules snaps a box's own top and bottom border rules to the
// box rect when a pagination shift left them displaced. The rules and the
// rails then read as one frame around the box instead of a frame floating
// over the content.
func realignOwnFrameRules(ops []Op, boxNode *box, owners map[int]*box) {
	if boxNode == nil || boxNode.style == nil || boxNode.opStart < 0 ||
		boxNode.opStart > boxNode.opEnd || boxNode.opEnd >= len(ops) {
		return
	}

	style := boxNode.style
	hasTop := style.BorderTop.Width > 0 && style.BorderTop.Style != cssDisplayNone
	hasBottom := style.BorderBottom.Width > 0 && style.BorderBottom.Style != cssDisplayNone

	if !hasTop && !hasBottom {
		return
	}

	topIdx, bottomIdx := frameRuleIndices(ops, boxNode, owners)

	moved := false

	switch {
	case topIdx >= 0 && topIdx != bottomIdx:
		if hasTop && !nearLayout(ops[topIdx].Y, boxNode.y) {
			ops[topIdx].Y = boxNode.y
			moved = true
		}

		if hasBottom && !nearLayout(ops[bottomIdx].Y, boxNode.y+boxNode.height) {
			ops[bottomIdx].Y = boxNode.y + boxNode.height
			moved = true
		}
	case topIdx >= 0 && hasTop != hasBottom:
		target := boxNode.y
		if hasBottom {
			target = boxNode.y + boxNode.height
		}

		if !nearLayout(ops[topIdx].Y, target) {
			ops[topIdx].Y = target
			moved = true
		}
	}

	if moved {
		normalizeOwnVerticalChrome(ops, boxNode, owners)
	}
}

func isBoxChromeEligible(res *Result, boxNode *box, owners map[int]*box) bool {
	if boxNode.opStart < 0 || boxNode.opStart > boxNode.opEnd || boxNode.opEnd >= len(res.Ops) || boxNode.height <= 0 {
		return false
	}

	if boxInsideTable(boxNode) || !hasOwnVerticalChrome(res.Ops, boxNode, owners) {
		return false
	}

	return true
}

func calculateChromeContentBottom(boxNode *box, oldBottom, inkBottom float64, hasInk bool) float64 {
	if !hasInk {
		return oldBottom
	}

	if boxNode == nil || boxNode.style == nil {
		return oldBottom
	}

	desiredBottom := inkBottom + boxNode.style.PaddingBottom
	if desiredBottom > oldBottom {
		return desiredBottom
	}

	return oldBottom
}

// hasDefiniteHeight reports a box whose CSS height is authored rather than
// content-derived. Chromium keeps such a border box at the specified height
// and lets the children overflow it.
func hasDefiniteHeight(boxNode *box) bool {
	if boxNode == nil || boxNode.style == nil {
		return false
	}

	return boxNode.style.Height >= 0 && boxNode.style.HeightPercent < 0
}

// chromeContentDisplaced reports ink below oldBottom that starts at or under
// the box bottom (pagination displaced it) rather than natural overflow that
// starts inside the box. A definite-height box may grow only for the former.
func chromeContentDisplaced(res *Result, boxNode *box, oldBottom float64, owners map[int]*box) bool {
	for idx := boxNode.opStart; idx <= boxNode.opEnd && idx < len(res.Ops); idx++ {
		if owner := owners[idx]; owner != nil && owner != boxNode {
			continue
		}

		operation := res.Ops[idx]
		if opOwnedBy(&operation, boxNode, opOwnerChrome) || operation.Positioned ||
			ownFrameRuleShape(&operation, boxNode) {
			continue
		}

		if operation.Y >= oldBottom-layoutCoordEpsilon && opInkBottom(operation) > oldBottom {
			return true
		}
	}

	for _, child := range boxNode.children {
		if child == nil {
			continue
		}

		if child.style != nil && (child.style.Position == positionAbsolute || child.style.Position == positionFixed) {
			continue
		}

		if child.y >= oldBottom-layoutCoordEpsilon && child.y+child.height > oldBottom {
			return true
		}
	}

	return false
}

func stretchBoxChrome(res *Result, boxNode *box, owners map[int]*box) {
	if !isBoxChromeEligible(res, boxNode, owners) {
		return
	}

	oldBottom := boxNode.y + boxNode.height
	normalizeOwnVerticalChrome(res.Ops, boxNode, owners)
	restoreStrippedHorizontalChrome(res.Ops, boxNode, owners)

	inkBottom, hasInk := calculateChromeInkBottom(res, boxNode, oldBottom, owners)
	contentBottom := calculateChromeContentBottom(boxNode, oldBottom, inkBottom, hasInk)

	// A definite-height box keeps its border box unless pagination actually
	// displaced content below it. The op stretch below receives the real
	// bottom: oldBottom when growth is skipped, so chrome is never rewritten
	// past its layout rect for natural child overflow.
	newBottom := oldBottom
	mayGrow := !hasDefiniteHeight(boxNode) || chromeContentDisplaced(res, boxNode, oldBottom, owners)

	if mayGrow && contentBottom > oldBottom+1e-6 {
		boxNode.height = contentBottom - boxNode.y
		newBottom = contentBottom
	}

	normalizeOwnVerticalChrome(res.Ops, boxNode, owners)

	for idx := boxNode.opStart; idx <= boxNode.opEnd; idx++ {
		stretchOwnBoxChrome(&res.Ops[idx], boxNode, oldBottom, newBottom)
	}
}

// chromeOpOwners maps each op index to the box whose chrome that op paints.
// Later boxes in document order win a coincident-edge tie, so a sibling
// stacked on this box's bottom edge claims its own border op even though the
// earlier box's op range contains it.
func chromeOpOwners(root *box, ops []Op) map[int]*box {
	owners := make(map[int]*box)

	var walk func(*box)
	walk = func(boxNode *box) {
		if boxNode == nil {
			return
		}

		if boxNode.opStart >= 0 && boxNode.opStart <= boxNode.opEnd && boxNode.opEnd < len(ops) {
			for idx := boxNode.opStart; idx <= boxNode.opEnd; idx++ {
				if opOwnedBy(&ops[idx], boxNode, opOwnerChrome) {
					owners[idx] = boxNode
				}
			}
		}

		for _, child := range boxNode.children {
			walk(child)
		}
	}

	walk(root)

	return owners
}

// stretchPaginatedChrome repairs block chrome after pagination has shifted a
// descendant past the block's original bottom. The layout box is built before
// page-break fixups, so its background and side rails otherwise stop at the
// stale natural height while the moved footer/text continues below it.
func stretchPaginatedChrome(res *Result) {
	if res == nil || res.root == nil {
		return
	}

	owners := chromeOpOwners(res.root, res.Ops)

	var walk func(*box)
	walk = func(boxNode *box) {
		for _, child := range boxNode.children {
			walk(child)
		}

		stretchBoxChrome(res, boxNode, owners)
		realignOwnFrameRules(res.Ops, boxNode, owners)
	}

	walk(res.root)
}

func boxIsFloatOrFigure(boxNode *box) bool {
	if boxNode == nil {
		return false
	}

	if boxNode.node != nil && boxNode.node.Name == "figure" {
		return true
	}

	if boxNode.style == nil {
		return false
	}

	return boxNode.style.Float == floatLeft || boxNode.style.Float == floatRight
}

func lastInFlowChildBottom(boxNode *box) float64 {
	if boxNode == nil {
		return 0
	}

	bottom := 0.0

	for _, child := range boxNode.children {
		if child == nil {
			continue
		}

		if child.style != nil && (child.style.Position == positionAbsolute || child.style.Position == positionFixed) {
			continue
		}

		if childBottom := child.y + child.height; childBottom > bottom {
			bottom = childBottom
		}
	}

	return bottom
}

//nolint:wsl // border ownership checks are intentionally explicit
func hasOwnVerticalChrome(ops []Op, boxNode *box, owners map[int]*box) bool {
	if boxNode == nil || boxNode.style == nil {
		return false
	}

	style := boxNode.style

	leftBorder := style.BorderLeft.Width > 0 && style.BorderLeft.Style != cssDisplayNone
	rightBorder := style.BorderRight.Width > 0 && style.BorderRight.Style != cssDisplayNone
	if !leftBorder && !rightBorder {
		return false
	}

	found := false

	forEachLineIndex(ops, boxNode.opStart, boxNode.opEnd, func(opIdx, segIdx int) {
		if found {
			return
		}

		if owner := owners[opIdx]; owner != nil && owner != boxNode {
			return
		}

		if isVerticalChromeForBox(lineViewAt(ops, opIdx, segIdx), boxNode, leftBorder, rightBorder) {
			found = true
		}
	})

	return found
}

//nolint:cyclop,wsl // fragment collection deliberately mirrors paint ownership
func normalizeOwnVerticalChrome(ops []Op, boxNode *box, owners map[int]*box) {
	if boxNode == nil || boxNode.style == nil {
		return
	}

	style := boxNode.style

	leftBorder := style.BorderLeft.Width > 0 && style.BorderLeft.Style != cssDisplayNone
	rightBorder := style.BorderRight.Width > 0 && style.BorderRight.Style != cssDisplayNone

	type lineRef struct{ opIdx, segIdx int }

	minY := math.Inf(1)

	refs := make([]lineRef, 0, 4) //nolint:mnd

	forEachLineIndex(ops, boxNode.opStart, boxNode.opEnd, func(opIdx, segIdx int) {
		if owner := owners[opIdx]; owner != nil && owner != boxNode {
			return
		}

		line := lineViewAt(ops, opIdx, segIdx)
		if !isVerticalChromeForBox(line, boxNode, leftBorder, rightBorder) {
			return
		}

		refs = append(refs, lineRef{opIdx: opIdx, segIdx: segIdx})

		if line.Y < minY {
			minY = line.Y
		}
	})

	if math.IsInf(minY, 1) {
		return
	}

	delta := boxNode.y - minY
	if math.Abs(delta) <= 1e-6 {
		return
	}

	for _, ref := range refs {
		addLineAtY(ops, ref.opIdx, ref.segIdx, delta)
	}

	// Solid rails are one continuous OpLine: extend the last segment to the
	// box bottom after a Y realign. Dashed/dotted sides are many short
	// segments - growing the last one paints a solid stub past the dashes
	// (fixture-40 abs-host, fixture-48 tracking).
	last := refs[len(refs)-1]
	if isDashLikeVerticalRail(lineViewAt(ops, last.opIdx, last.segIdx), boxNode) {
		return
	}

	extendLineBottom(ops, last.opIdx, last.segIdx, boxNode.y+boxNode.height)
}

//nolint:cyclop // chrome classification keeps line and masked-side geometry explicit
func isVerticalChromeForBox(operation Op, boxNode *box, leftBorder, rightBorder bool) bool {
	line := operation.Kind == OpLine && operation.W == 0
	maskedLeft := operation.Kind == OpStrokeRect && operation.StrokeMask == StrokeMaskLeft
	maskedRight := operation.Kind == OpStrokeRect && operation.StrokeMask == StrokeMaskRight

	if !line && !maskedLeft && !maskedRight {
		return false
	}

	return operation.H > 0 &&
		operation.H <= boxNode.height+1e-6 &&
		((leftBorder && nearLayout(operation.X, boxNode.x) && (line || maskedLeft)) ||
			(rightBorder && nearLayout(operation.X, boxNode.x+boxNode.w) && (line || maskedRight))) &&
		operation.Y >= boxNode.y-1e-6 && operation.Y <= boxNode.y+boxNode.height+1e-6
}

// isDashedOrDottedStyle reports border styles expanded into multi-segment OpLines.
func isDashedOrDottedStyle(style string) bool {
	return style == borderStyleDashed || style == borderStyleDotted
}

// looksLikeDashSegmentLength is true for edge pieces sized like appendDashedLineSegments
// (drawLen = width*3 dashed, width dotted), not a continuous solid rail.
func looksLikeDashSegmentLength(segLen, strokeWidth float64) bool {
	if segLen <= 0 {
		return false
	}

	maxSeg := math.Max(strokeWidth*3, 0.5) + 0.5 //nolint:mnd // three=3, halfRatio=0.5
	if strokeWidth <= 0 {
		maxSeg = 3 + 0.5 //nolint:mnd // three=3, halfRatio=0.5
	}

	return segLen <= maxSeg
}

func isVerticalLineOp(operation Op) bool {
	return operation.Kind == OpLine && operation.W == 0 && operation.H > 0
}

// isDashLikeVerticalRail is true when a vertical side stroke must not be H-stretched:
// dashed/dotted CSS on that side, or a short segment that is already a dash piece.
func isDashLikeVerticalRail(operation Op, boxNode *box) bool {
	if !isVerticalLineOp(operation) || boxNode == nil || boxNode.style == nil {
		return false
	}

	if looksLikeDashSegmentLength(operation.H, operation.Width) {
		return true
	}

	onLeft := nearLayout(operation.X, boxNode.x) && isDashedOrDottedStyle(boxNode.style.BorderLeft.Style)
	onRight := nearLayout(operation.X, boxNode.x+boxNode.w) && isDashedOrDottedStyle(boxNode.style.BorderRight.Style)

	return onLeft || onRight
}

// isHorizontalChromeForBox reports top/bottom edge strokes owned by the box,
// including short dashed/dotted segments (full-width nearLayout alone misses those).
//
//nolint:cyclop // edge membership mirrors vertical chrome checks
func isHorizontalChromeForBox(operation Op, boxNode *box, oldBottom float64) bool {
	if operation.Kind != OpLine || operation.H != 0 || operation.W <= 0 || boxNode == nil {
		return false
	}

	onTop := nearLayout(operation.Y, boxNode.y)
	onBottom := nearLayout(operation.Y, oldBottom)

	if !onTop && !onBottom {
		return false
	}

	// Solid (or single) full-width edge.
	if nearLayout(operation.X, boxNode.x) && nearLayout(operation.W, boxNode.w) {
		return true
	}

	if boxNode.style == nil {
		return false
	}

	style := boxNode.style

	// Dashed/dotted fragments sit on the edge with dash-sized W.
	inside := operation.X >= boxNode.x-1e-6 &&
		operation.X+operation.W <= boxNode.x+boxNode.w+1e-6
	if !inside {
		return false
	}

	if onTop && isDashedOrDottedStyle(style.BorderTop.Style) {
		return true
	}

	if onBottom && isDashedOrDottedStyle(style.BorderBottom.Style) {
		return true
	}

	return looksLikeDashSegmentLength(operation.W, operation.Width)
}

func opInkBottom(operation Op) float64 {
	if operation.Kind == OpText || operation.Kind == OpBullet {
		return operation.Y + opVisibleInkHeight(operation)
	}

	if operation.H > 0 {
		return operation.Y + operation.H
	}

	if operation.Kind == OpLine && operation.W > 0 {
		return operation.Y + math.Max(operation.Width, 1)
	}

	return operation.Y
}

const boxBottomMatchSlack = 1.5

//nolint:cyclop // mutate owned paint
func stretchOwnBoxChrome(operation *Op, boxNode *box, oldBottom, newBottom float64) {
	if operation == nil || boxNode == nil {
		return
	}
	style := boxNode.style

	if (operation.Kind == OpFillRect || operation.Kind == OpStrokeRect) &&
		nearLayout(operation.X, boxNode.x) && nearLayout(operation.W, boxNode.w) {
		// Only stretch chrome that currently owns the box bottom. Earlier
		// multi-page fragments end at a page boundary and must stay open;
		// stretching them to newBottom refilled whole pages (fixture-31).
		fullMatch := nearLayout(operation.Y, boxNode.y) && nearLayout(operation.H, oldBottom-boxNode.y)
		ownsBottom := math.Abs(operation.Y+operation.H-oldBottom) < boxBottomMatchSlack

		if fullMatch || ownsBottom {
			operation.H = newBottom - operation.Y
			if operation.H < 0 {
				operation.H = 0
			}

			return
		}
	}

	if operation.Kind == OpLine && operation.W == 0 && operation.H > 0 && style != nil &&
		((style.BorderLeft.Width > 0 && nearLayout(operation.X, boxNode.x)) ||
			(style.BorderRight.Width > 0 && nearLayout(operation.X, boxNode.x+boxNode.w))) &&
		operation.Y >= boxNode.y-1e-6 && operation.Y <= boxNode.y+boxNode.height+1e-6 &&
		nearLayout(operation.Y+operation.H, oldBottom) {
		// Never elongate a dash/dot segment into a solid stub.
		if isDashLikeVerticalRail(*operation, boxNode) {
			return
		}

		operation.H = newBottom - operation.Y

		return
	}

	// Bottom edge: solid full-width line, or every dashed/dotted fragment on
	// that edge (short W would miss nearLayout(W, box.w)).
	if operation.Kind == OpLine && operation.H == 0 && operation.W > 0 &&
		isHorizontalChromeForBox(*operation, boxNode, oldBottom) &&
		nearLayout(operation.Y, oldBottom) {
		operation.Y = newBottom
	}
}

func nearLayout(a, b float64) bool { return math.Abs(a-b) < 0.01 } //nolint:mnd // layout epsilon

// strippedHorizontalRowHeight bounds the row-sized chrome the orphan-row
// strip removes. A frame taller than one row is a block, not an orphan row,
// so its own top/bottom border rule keeps its authored stroke.
const strippedHorizontalRowHeight = 40.0

// restoreStrippedHorizontalChrome re-strokes a block frame's own top/bottom
// border rules after the orphan-row strip zeroed their stroke width
// (paint_pagination_seal.go stripOrphanRowOp). Without the stroke the edge
// line paints nothing, so the box reads as borderless.
func restoreStrippedHorizontalChrome(ops []Op, boxNode *box, owners map[int]*box) {
	if boxNode == nil || boxNode.style == nil || boxNode.height <= strippedHorizontalRowHeight {
		return
	}

	for idx := boxNode.opStart; idx <= boxNode.opEnd && idx < len(ops); idx++ {
		if owner := owners[idx]; owner != nil && owner != boxNode {
			continue
		}

		paintOp := &ops[idx]
		if paintOp.Fixed || paintOp.Kind != OpLine || paintOp.H != 0 || paintOp.W <= 0 || paintOp.Width > 0 {
			continue
		}

		side := ownStrippedHorizontalSide(boxNode, paintOp)
		if side == nil {
			continue
		}

		paintOp.Width = borderPaint(*side) * boxBorderScale(ops, boxNode, owners)
	}
}

// ownStrippedHorizontalSide returns the border side a zero-stroke horizontal
// rule still draws, or nil when the rule is not the box's own frame edge.
func ownStrippedHorizontalSide(boxNode *box, paintOp *Op) *border {
	style := boxNode.style
	if !nearLayout(paintOp.X, boxNode.x) || !nearLayout(paintOp.W, boxNode.w) {
		return nil
	}

	if nearLayout(paintOp.Y, boxNode.y) && style.BorderTop.Width > 0 && style.BorderTop.Style != cssDisplayNone {
		return &style.BorderTop
	}

	if nearLayout(paintOp.Y, boxNode.y+boxNode.height) &&
		style.BorderBottom.Width > 0 && style.BorderBottom.Style != cssDisplayNone {
		return &style.BorderBottom
	}

	return nil
}

// boxBorderScale recovers the engine's CSS-to-device scale from any surviving
// border op on the box, so a restored stroke matches its authored side width.
func boxBorderScale(ops []Op, boxNode *box, owners map[int]*box) float64 {
	for idx := boxNode.opStart; idx <= boxNode.opEnd && idx < len(ops); idx++ {
		if owner := owners[idx]; owner != nil && owner != boxNode {
			continue
		}

		paintOp := ops[idx]
		if paintOp.Kind != OpLine || paintOp.Width <= 0 {
			continue
		}

		side := survivingBorderSide(boxNode, &paintOp)
		if side == nil {
			continue
		}

		if raw := borderPaint(*side); raw > 0 {
			return paintOp.Width / raw
		}
	}

	return 1
}

// survivingBorderSide maps an intact border line back to its CSS side by the
// edge it sits on: a zero-width line is a rail, a zero-height one a rule.
func survivingBorderSide(boxNode *box, paintOp *Op) *border {
	style := boxNode.style
	rail := paintOp.W == 0
	rule := paintOp.H == 0 && paintOp.W > 0

	switch {
	case rail && nearLayout(paintOp.X, boxNode.x):
		return &style.BorderLeft
	case rail && nearLayout(paintOp.X, boxNode.x+boxNode.w):
		return &style.BorderRight
	case rule && nearLayout(paintOp.X, boxNode.x) && nearLayout(paintOp.W, boxNode.w) &&
		nearLayout(paintOp.Y, boxNode.y):
		return &style.BorderTop
	case rule && nearLayout(paintOp.X, boxNode.x) && nearLayout(paintOp.W, boxNode.w) &&
		nearLayout(paintOp.Y, boxNode.y+boxNode.height):
		return &style.BorderBottom
	default:
		return nil
	}
}
