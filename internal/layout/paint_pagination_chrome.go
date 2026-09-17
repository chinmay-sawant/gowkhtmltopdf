//nolint:all
package layout

import (
	"math"
)

func calculateChromeInkBottom(res *Result, boxNode *box, oldBottom float64) (float64, bool) {
	inkBottom := boxNode.y
	hasInk := false

	for idx := boxNode.opStart; idx <= boxNode.opEnd; idx++ {
		operation := res.Ops[idx]
		if opOwnedBy(&operation, boxNode, opOwnerChrome) || operation.Positioned {
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

	if boxInsideTable(boxNode) {
		return false
	}

	return hasOwnVerticalChrome(res.Ops, boxNode) || hasOwnCardFace(res.Ops, boxNode)
}

// hasOwnCardFace reports a box that paints its own background or box-shadow.
// Split cards whose face is a background color or a shadow layer need the same
// post-pagination stretch as callouts with side rails, or the card fill stops
// at the stale layout height while the rows shifted below its bottom keep
// painting (learncpp lesson cards).
func hasOwnCardFace(ops []Op, boxNode *box) bool {
	if boxNode.style == nil || (boxNode.style.BGColor[3] <= 0 && !boxNode.style.BoxShadowSet) {
		return false
	}

	for idx := boxNode.opStart; idx <= boxNode.opEnd && idx < len(ops); idx++ {
		if opOwnedBy(&ops[idx], boxNode, opOwnerChrome) {
			return true
		}
	}

	return false
}

func calculateChromeContentBottom(
	boxNode *box, oldBottom, inkBottom float64, hasInk bool, contentH float64,
) float64 {
	if !hasInk {
		return oldBottom
	}

	if boxNode == nil || boxNode.style == nil {
		return oldBottom
	}

	desiredBottom := inkBottom + boxNode.style.PaddingBottom
	// html/body paper wash fills unused page tail on the last page that has
	// ink (gobyexample continuation pages; fixture-56 short page-2 wash).
	if contentH > 0 && isPaperWashRoot(boxNode) {
		pageBot := math.Ceil((inkBottom-layoutCoordEpsilon)/contentH) * contentH
		if pageBot < contentH {
			pageBot = contentH
		}

		if pageBot > desiredBottom {
			desiredBottom = pageBot
		}
	}

	if desiredBottom > oldBottom {
		return desiredBottom
	}

	return oldBottom
}

// isPaperWashRoot reports the viewport paper elements whose background must
// cover each page's content box, including empty tails after the last ink.
func isPaperWashRoot(boxNode *box) bool {
	if boxNode == nil || boxNode.node == nil || boxNode.style == nil {
		return false
	}

	if boxNode.style.BGColor[3] <= 0 {
		return false
	}

	switch boxNode.node.Name {
	case htmlRootName, htmlBodyName:
		return true
	default:
		return false
	}
}

func stretchBoxChrome(res *Result, boxNode *box, growShadowLayers bool, contentH float64) {
	if !isBoxChromeEligible(res, boxNode) {
		return
	}

	oldBottom := boxNode.y + boxNode.height
	normalizeOwnVerticalChrome(res.Ops, boxNode)

	inkBottom, hasInk := calculateChromeInkBottom(res, boxNode, oldBottom)
	contentBottom := calculateChromeContentBottom(boxNode, oldBottom, inkBottom, hasInk, contentH)

	if contentBottom > oldBottom+1e-6 {
		boxNode.height = contentBottom - boxNode.y
	}

	normalizeOwnVerticalChrome(res.Ops, boxNode)

	for idx := boxNode.opStart; idx <= boxNode.opEnd; idx++ {
		stretchOwnBoxChrome(&res.Ops[idx], boxNode, oldBottom, contentBottom, growShadowLayers)
	}
}

// stretchPaginatedChrome repairs block chrome after pagination has shifted a
// descendant past the block's original bottom. The layout box is built before
// page-break fixups, so its background and side rails otherwise stop at the
// stale natural height while the moved footer/text continues below it.
//
// growShadowLayers is set on the pre-split call only, while a box-shadow layer
// still covers the whole box. Page-split fragments end at a page boundary and
// must stay open, so the post-split calls never grow shadow layers.
// contentH is the page content-box height used to extend html/body paper wash
// to the last page bottom.
func stretchPaginatedChrome(res *Result, growShadowLayers bool, contentH float64) {
	if res == nil || res.root == nil {
		return
	}

	var walk func(*box)
	walk = func(boxNode *box) {
		for _, child := range boxNode.children {
			walk(child)
		}

		stretchBoxChrome(res, boxNode, growShadowLayers, contentH)
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

		if isVerticalChromeForBox(lineViewAt(ops, opIdx, segIdx), boxNode, leftBorder, rightBorder) {
			found = true
		}
	})

	return found
}

//nolint:cyclop,wsl // fragment collection deliberately mirrors paint ownership
func normalizeOwnVerticalChrome(ops []Op, boxNode *box) {
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
func stretchOwnBoxChrome(operation *Op, boxNode *box, oldBottom, newBottom float64, growShadowLayers bool) {
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

	// Box-shadow layers (the core and its blur steps) never match the frame's
	// top or width, so the branch above leaves them at the stale layout
	// height. Grow every owned layer by the bottom delta instead: a split
	// card whose rows moved below the layout bottom keeps its face covering
	// them (learncpp lesson cards are shadow-only, no background color).
	// Classify against the pre-stretch height: box height already grew above,
	// which makes every layer look shorter than the box and unmatched.
	if growShadowLayers && newBottom > oldBottom+1e-6 &&
		(operation.Kind == OpFillRect || operation.Kind == OpStrokeRect) &&
		style != nil && style.BoxShadowSet {
		staleBox := *boxNode
		staleBox.height = oldBottom - boxNode.y

		if opOwnedBy(operation, &staleBox, opOwnerChrome) {
			operation.H += newBottom - oldBottom

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
