package layout

import (
	"math"
)

func calculateChromeInkBottom(res *Result, boxNode *box, oldBottom float64) (float64, bool) {
	inkBottom := boxNode.y
	hasInk := false

	for idx := boxNode.opStart; idx <= boxNode.opEnd; idx++ {
		operation := res.Ops[idx]
		if isOwnBoxChrome(operation, boxNode, oldBottom) ||
			isOwnBoxChromeFragment(operation, boxNode, oldBottom) ||
			operation.Positioned {
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

func isBoxChromeEligible(res *Result, boxNode *box) bool {
	if boxNode.opStart < 0 || boxNode.opStart > boxNode.opEnd || boxNode.opEnd >= len(res.Ops) || boxNode.height <= 0 {
		return false
	}

	if boxInsideTable(boxNode) || !hasOwnVerticalChrome(res.Ops, boxNode) {
		return false
	}

	return true
}

func calculateChromeContentBottom(boxNode *box, oldBottom, inkBottom float64, hasInk bool) float64 {
	if !hasInk {
		return oldBottom
	}

	padB := 0.0
	if boxNode.style != nil {
		padB = boxNode.style.PaddingBottom
	}

	desiredBottom := inkBottom + padB
	if desiredBottom > oldBottom {
		return desiredBottom
	}

	return oldBottom
}

func stretchBoxChrome(res *Result, boxNode *box) {
	if !isBoxChromeEligible(res, boxNode) {
		return
	}

	oldBottom := boxNode.y + boxNode.height
	normalizeOwnVerticalChrome(res.Ops, boxNode)

	inkBottom, hasInk := calculateChromeInkBottom(res, boxNode, oldBottom)
	contentBottom := calculateChromeContentBottom(boxNode, oldBottom, inkBottom, hasInk)

	if contentBottom > oldBottom+1e-6 {
		boxNode.height = contentBottom - boxNode.y
	}

	normalizeOwnVerticalChrome(res.Ops, boxNode)

	for idx := boxNode.opStart; idx <= boxNode.opEnd; idx++ {
		stretchOwnBoxChrome(&res.Ops[idx], boxNode, oldBottom, contentBottom)
	}
}

// stretchPaginatedChrome repairs block chrome after pagination has shifted a
// descendant past the block's original bottom. The layout box is built before
// page-break fixups, so its background and side rails otherwise stop at the
// stale natural height while the moved footer/text continues below it.
func stretchPaginatedChrome(res *Result) {
	if res == nil || res.root == nil {
		return
	}

	var walk func(*box)
	walk = func(boxNode *box) {
		for _, child := range boxNode.children {
			walk(child)
		}

		stretchBoxChrome(res, boxNode)
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
func hasOwnVerticalChrome(ops []Op, boxNode *box) bool {
	if boxNode == nil || boxNode.style == nil {
		return false
	}
	leftBorder := boxNode.style.BorderLeft.Width > 0 && boxNode.style.BorderLeft.Style != cssDisplayNone
	rightBorder := boxNode.style.BorderRight.Width > 0 && boxNode.style.BorderRight.Style != cssDisplayNone
	if !leftBorder && !rightBorder {
		return false
	}

	for idx := boxNode.opStart; idx <= boxNode.opEnd && idx < len(ops); idx++ {
		op := ops[idx]
		if isVerticalChromeForBox(op, boxNode, leftBorder, rightBorder) {
			return true
		}
	}

	return false
}

//nolint:cyclop,wsl // fragment collection deliberately mirrors paint ownership
func normalizeOwnVerticalChrome(ops []Op, boxNode *box) {
	if boxNode == nil || boxNode.style == nil {
		return
	}

	leftBorder := boxNode.style.BorderLeft.Width > 0 && boxNode.style.BorderLeft.Style != cssDisplayNone
	rightBorder := boxNode.style.BorderRight.Width > 0 && boxNode.style.BorderRight.Style != cssDisplayNone
	minY := math.Inf(1)
	verticalIndexes := make([]int, 0, 4)
	for idx := boxNode.opStart; idx <= boxNode.opEnd && idx < len(ops); idx++ {
		if isVerticalChromeForBox(ops[idx], boxNode, leftBorder, rightBorder) {
			verticalIndexes = append(verticalIndexes, idx)
			if ops[idx].Y < minY {
				minY = ops[idx].Y
			}
		}
	}
	if math.IsInf(minY, 1) {
		return
	}

	delta := boxNode.y - minY
	if math.Abs(delta) <= layoutEpsilon {
		return
	}
	for _, idx := range verticalIndexes {
		ops[idx].Y += delta
	}

	// Solid rails are one continuous OpLine: extend the last segment to the
	// box bottom after a Y realign. Dashed/dotted sides are many short
	// segments - growing the last one paints a solid stub past the dashes
	// (fixture-40 abs-host, fixture-48 tracking).
	last := verticalIndexes[len(verticalIndexes)-1]
	if isDashLikeVerticalRail(ops[last], boxNode) {
		return
	}

	lastBottom := ops[last].Y + ops[last].H
	if lastBottom < boxNode.y+boxNode.height {
		ops[last].H += boxNode.y + boxNode.height - lastBottom
	}
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
		operation.H <= boxNode.height+layoutEpsilon &&
		((leftBorder && nearLayout(operation.X, boxNode.x) && (line || maskedLeft)) ||
			(rightBorder && nearLayout(operation.X, boxNode.x+boxNode.w) && (line || maskedRight))) &&
		operation.Y >= boxNode.y-layoutEpsilon && operation.Y <= boxNode.y+boxNode.height+layoutEpsilon
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

	maxSeg := math.Max(strokeWidth*three, minDashPt) + halfRatio
	if strokeWidth <= 0 {
		maxSeg = three + halfRatio
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

	// Dashed/dotted fragments sit on the edge with dash-sized W.
	inside := operation.X >= boxNode.x-layoutEpsilon &&
		operation.X+operation.W <= boxNode.x+boxNode.w+layoutEpsilon
	if !inside {
		return false
	}

	if onTop && isDashedOrDottedStyle(boxNode.style.BorderTop.Style) {
		return true
	}

	if onBottom && isDashedOrDottedStyle(boxNode.style.BorderBottom.Style) {
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

//nolint:cyclop // classify box-owned paint
func isOwnBoxChrome(operation Op, boxNode *box, oldBottom float64) bool {
	if boxNode == nil {
		return false
	}

	if (operation.Kind == OpFillRect || operation.Kind == OpStrokeRect) &&
		nearLayout(operation.X, boxNode.x) && nearLayout(operation.Y, boxNode.y) &&
		nearLayout(operation.W, boxNode.w) && nearLayout(operation.H, boxNode.height) {
		return true
	}

	if operation.Kind != OpLine {
		return false
	}

	if boxNode.style == nil {
		return false
	}

	vertical := isVerticalChromeForBox(operation, boxNode,
		boxNode.style.BorderLeft.Width > 0 && boxNode.style.BorderLeft.Style != cssDisplayNone,
		boxNode.style.BorderRight.Width > 0 && boxNode.style.BorderRight.Style != cssDisplayNone)
	horizontal := isHorizontalChromeForBox(operation, boxNode, oldBottom)

	return vertical || horizontal
}

func isOwnBoxRectFragment(operation Op, boxNode *box, oldBottom float64) bool {
	isRectKind := operation.Kind == OpFillRect || operation.Kind == OpStrokeRect
	if !isRectKind || !nearLayout(operation.X, boxNode.x) || !nearLayout(operation.W, boxNode.w) {
		return false
	}

	return operation.Y >= boxNode.y-layoutEpsilon && operation.Y+operation.H <= oldBottom+1
}

// isOwnBoxChromeFragment reports page-split fill/stroke/rail pieces of the
// box's own frame. After openStrokeFragment these no longer match the full
// border-box height, so isOwnBoxChrome alone would treat them as content ink
// and re-add padding-bottom on every stretch pass.
func isOwnBoxChromeFragment(operation Op, boxNode *box, oldBottom float64) bool {
	if boxNode == nil || boxNode.style == nil {
		return false
	}

	if isOwnBoxRectFragment(operation, boxNode, oldBottom) || isHorizontalChromeForBox(operation, boxNode, oldBottom) {
		return true
	}

	hasLeft := boxNode.style.BorderLeft.Width > 0 && boxNode.style.BorderLeft.Style != cssDisplayNone
	hasRight := boxNode.style.BorderRight.Width > 0 && boxNode.style.BorderRight.Style != cssDisplayNone

	return isVerticalChromeForBox(operation, boxNode, hasLeft, hasRight)
}

const boxBottomMatchSlack = 1.5

//nolint:cyclop // mutate owned paint
func stretchOwnBoxChrome(operation *Op, boxNode *box, oldBottom, newBottom float64) {
	if operation == nil || boxNode == nil {
		return
	}

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

	if operation.Kind == OpLine && operation.W == 0 && operation.H > 0 &&
		((boxNode.style != nil && boxNode.style.BorderLeft.Width > 0 && nearLayout(operation.X, boxNode.x)) ||
			(boxNode.style != nil && boxNode.style.BorderRight.Width > 0 && nearLayout(operation.X, boxNode.x+boxNode.w))) &&
		operation.Y >= boxNode.y-layoutEpsilon && operation.Y <= boxNode.y+boxNode.height+layoutEpsilon &&
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
