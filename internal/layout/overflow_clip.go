//nolint:varnamelen // clip math uses compact x/y/w/h/op geometry names
package layout

import "math"

// clipRect is an axis-aligned padding-box clip in canvas points.
type clipRect struct {
	x, y, w, h float64
}

// overflowClipsPaint reports overflow values that clip descendant paint to
// the padding box. auto/scroll clip the same in PDF (no user scroll).
func overflowClipsPaint(overflow string) bool {
	switch overflow {
	case overflowHidden, overflowClip, overflowScroll, overflowAuto:
		return true
	}

	return false
}

func (r clipRect) empty() bool {
	return r.w <= 0 || r.h <= 0
}

func intersectClip(a, b clipRect) clipRect {
	x1 := math.Max(a.x, b.x)
	y1 := math.Max(a.y, b.y)
	x2 := math.Min(a.x+a.w, b.x+b.w)
	y2 := math.Min(a.y+a.h, b.y+b.h)

	if x2 <= x1 || y2 <= y1 {
		return clipRect{} //nolint:exhaustruct // empty intersection
	}

	return clipRect{x: x1, y: y1, w: x2 - x1, h: y2 - y1}
}

func (e *engine) paddingBoxRect(posX, posY, width, height float64, sty ResolvedStyle) clipRect {
	left := e.scalePt(sty.BorderLeft.Width)
	top := e.scalePt(sty.BorderTop.Width)
	right := e.scalePt(sty.BorderRight.Width)
	bottom := e.scalePt(sty.BorderBottom.Width)
	w := width - left - right
	h := height - top - bottom

	if w < 0 {
		w = 0
	}

	if h < 0 {
		h = 0
	}

	rect := clipRect{x: posX + left, y: posY + top, w: w, h: h}

	// Inflate for overflow-clip-margin (used by fixture-62 rows 27-37).
	// Values are in points already via parseAdvancedLength.
	if sty.OverflowClipMarginTop != 0 || sty.OverflowClipMarginRight != 0 ||
		sty.OverflowClipMarginBottom != 0 || sty.OverflowClipMarginLeft != 0 {
		rect.x -= sty.OverflowClipMarginLeft
		rect.y -= sty.OverflowClipMarginTop
		rect.w += sty.OverflowClipMarginLeft + sty.OverflowClipMarginRight
		rect.h += sty.OverflowClipMarginTop + sty.OverflowClipMarginBottom

		if rect.w < 0 {
			rect.w = 0
		}

		if rect.h < 0 {
			rect.h = 0
		}
	}

	return rect
}

func (e *engine) paddingBoxOf(boxNode *box) clipRect {
	if boxNode == nil || boxNode.style == nil {
		return clipRect{} //nolint:exhaustruct // missing box
	}

	return e.paddingBoxRect(boxNode.x, boxNode.y, boxNode.w, boxNode.height, *boxNode.style)
}

// applyOverflowClips clips descendant paint to padding boxes of overflow
// hidden|clip|auto|scroll ancestors. Own background/border/outline of the
// overflow box are kept. Call after finalizeChrome has merged deferred ops.
func (e *engine) applyOverflowClips(root *box) {
	if e == nil || root == nil {
		return
	}

	e.clipOverflowTree(root, nil)
}

const (
	unconstrainedClipOffset = -1e9
	unconstrainedClipSpan   = 2e9
	clipPointTolerance      = 0.01
	clipZeroLineEpsilon     = 1e-6
)

func (e *engine) computeBoxOverflowClip(boxNode *box, current *clipRect) *clipRect {
	if boxNode.style == nil {
		return current
	}

	clipX := overflowClipsPaint(boxNode.style.OverflowX) || overflowClipsPaint(boxNode.style.Overflow)
	clipY := overflowClipsPaint(boxNode.style.OverflowY) || overflowClipsPaint(boxNode.style.Overflow)

	if !clipX && !clipY {
		return current
	}

	pb := e.paddingBoxOf(boxNode)
	if !clipX {
		pb.x = unconstrainedClipOffset
		pb.w = unconstrainedClipSpan
	}

	if !clipY {
		pb.y = unconstrainedClipOffset
		pb.h = unconstrainedClipSpan
	}

	if current != nil {
		merged := intersectClip(*current, pb)

		return &merged
	}

	return &pb
}

func (e *engine) clipOverflowTree(boxNode *box, clip *clipRect) {
	if boxNode == nil {
		return
	}

	next := e.computeBoxOverflowClip(boxNode, clip)

	switch {
	case clip != nil:
		// Ancestor clip applies to this whole box, including its chrome.
		clipOpsRange(e.ops, boxNode.opStart, boxNode.opEnd, *clip)
	case next != nil:
		// This box established the clip: keep its chrome, clip descendants
		// and its own in-flow content.
		e.clipBoxContents(boxNode, *next)
	}

	for _, child := range boxNode.children {
		e.clipOverflowTree(child, next)
	}
}

func (e *engine) clipBoxContents(boxNode *box, clip clipRect) {
	if boxNode == nil || clip.empty() {
		return
	}

	for _, child := range boxNode.children {
		if child == nil {
			continue
		}

		clipOpsRange(e.ops, child.opStart, child.opEnd, clip)
	}

	e.clipOwnContentOps(e.ops, boxNode, clip)
}

func clipOpsRange(ops []Op, start, end int, clip clipRect) {
	if end < start || start < 0 || clip.empty() {
		return
	}

	for i := start; i <= end && i < len(ops); i++ {
		clipPaintOp(&ops[i], clip)
	}
}

func clipOpsSlice(ops []Op, clip clipRect) {
	if clip.empty() {
		return
	}

	for i := range ops {
		clipPaintOp(&ops[i], clip)
	}
}

func (e *engine) clipOwnContentOps(ops []Op, boxNode *box, clip clipRect) {
	if e == nil || boxNode == nil || boxNode.opEnd < boxNode.opStart || boxNode.opStart < 0 {
		return
	}

	for i := boxNode.opStart; i <= boxNode.opEnd && i < len(ops); i++ {
		if opInChildRange(boxNode, i) || e.isOwnChromeOp(&ops[i], boxNode) {
			continue
		}

		clipPaintOp(&ops[i], clip)
	}
}

func opInChildRange(boxNode *box, idx int) bool {
	if boxNode == nil {
		return false
	}

	for _, child := range boxNode.children {
		if child == nil || child.opEnd < child.opStart {
			continue
		}

		if idx >= child.opStart && idx <= child.opEnd {
			return true
		}
	}

	return false
}

func (e *engine) isOwnChromeOp(op *Op, boxNode *box) bool {
	if op == nil || boxNode == nil {
		return false
	}

	switch op.Kind {
	case OpFillRect, OpStrokeRect:
		return nearRectOp(op, boxNode.x, boxNode.y, boxNode.w, boxNode.height)
	case OpLine:
		if lineOnRectEdges(op, boxNode.x, boxNode.y, boxNode.w, boxNode.height) {
			return true
		}

		return e.isOwnOutlineLine(op, boxNode)
	case OpText, OpImage, OpLinkURI, OpBullet, opKindNoop:
		return false
	default:
		return false
	}
}

func (e *engine) isOwnOutlineLine(op *Op, boxNode *box) bool {
	if e == nil || op == nil || boxNode == nil || boxNode.style == nil || !outlinePaints(boxNode.style) {
		return false
	}

	ow := e.scalePt(boxNode.style.OutlineWidth)
	off := e.scalePt(boxNode.style.OutlineOffset)
	inflate := outlineInflate(ow, off)

	return lineOnRectEdges(
		op,
		boxNode.x-inflate, boxNode.y-inflate,
		boxNode.w+2*inflate, boxNode.height+2*inflate,
	)
}

func nearRectOp(op *Op, x, y, w, h float64) bool {
	if op == nil {
		return false
	}

	return math.Abs(op.X-x) <= clipPointTolerance &&
		math.Abs(op.Y-y) <= clipPointTolerance &&
		math.Abs(op.W-w) <= clipPointTolerance &&
		math.Abs(op.H-h) <= clipPointTolerance
}

func lineOnRectEdges(op *Op, x, y, w, h float64) bool {
	if op == nil || op.Kind != OpLine {
		return false
	}

	if math.Abs(op.H) <= clipPointTolerance && op.W > 0 {
		return horizontalOnRectEdges(op, x, y, w, h)
	}

	if math.Abs(op.W) <= clipPointTolerance && op.H > 0 {
		return verticalOnRectEdges(op, x, y, w, h)
	}

	return false
}

func horizontalOnRectEdges(op *Op, x, y, w, h float64) bool {
	if op == nil {
		return false
	}

	onTop := math.Abs(op.Y-y) <= clipPointTolerance
	onBot := math.Abs(op.Y-(y+h)) <= clipPointTolerance

	if !onTop && !onBot {
		return false
	}

	return op.X+op.W >= x-clipPointTolerance && op.X <= x+w+clipPointTolerance
}

func verticalOnRectEdges(op *Op, x, y, w, h float64) bool {
	if op == nil {
		return false
	}

	onLeft := math.Abs(op.X-x) <= clipPointTolerance
	onRight := math.Abs(op.X-(x+w)) <= clipPointTolerance

	if !onLeft && !onRight {
		return false
	}

	return op.Y+op.H >= y-clipPointTolerance && op.Y <= y+h+clipPointTolerance
}

func clipPaintOp(op *Op, clip clipRect) {
	if op == nil || clip.empty() || op.Kind == opKindNoop {
		return
	}

	switch op.Kind {
	case OpFillRect, OpStrokeRect, OpImage, OpLinkURI:
		clipRectOp(op, clip)
	case OpLine:
		clipLineOp(op, clip)
	case OpText, OpBullet:
		clipTextOp(op, clip)
	case opKindNoop:
	}
}

func clipRectOp(op *Op, clip clipRect) {
	x1 := math.Max(op.X, clip.x)
	y1 := math.Max(op.Y, clip.y)
	x2 := math.Min(op.X+op.W, clip.x+clip.w)
	y2 := math.Min(op.Y+op.H, clip.y+clip.h)

	if x2-x1 <= clipPointTolerance || y2-y1 <= clipPointTolerance {
		DeactivateOp(op)

		return
	}

	op.X, op.Y, op.W, op.H = x1, y1, x2-x1, y2-y1
}

func clipLineOp(op *Op, clip clipRect) {
	x0, y0 := op.X, op.Y
	x1, y1 := op.X+op.W, op.Y+op.H

	if math.Abs(op.H) <= clipZeroLineEpsilon {
		clipHorizontalLine(op, clip, x0, x1, y0)

		return
	}

	if math.Abs(op.W) <= clipZeroLineEpsilon {
		clipVerticalLine(op, clip, y0, y1, x0)

		return
	}

	if !rectIntersects(x0, y0, x1, y1, clip) {
		DeactivateOp(op)
	}
}

func clipHorizontalLine(op *Op, clip clipRect, x0, x1, y0 float64) {
	if y0 < clip.y-clipPointTolerance || y0 > clip.y+clip.h+clipPointTolerance {
		DeactivateOp(op)

		return
	}

	left := math.Min(x0, x1)
	right := math.Max(x0, x1)
	nl := math.Max(left, clip.x)
	nr := math.Min(right, clip.x+clip.w)

	if nr-nl <= 0 {
		DeactivateOp(op)

		return
	}

	op.X, op.Y, op.W, op.H = nl, y0, nr-nl, 0
}

func clipVerticalLine(op *Op, clip clipRect, y0, y1, x0 float64) {
	if x0 < clip.x-clipPointTolerance || x0 > clip.x+clip.w+clipPointTolerance {
		DeactivateOp(op)

		return
	}

	top := math.Min(y0, y1)
	bot := math.Max(y0, y1)
	nt := math.Max(top, clip.y)
	nb := math.Min(bot, clip.y+clip.h)

	if nb-nt <= 0 {
		DeactivateOp(op)

		return
	}

	op.X, op.Y, op.W, op.H = x0, nt, 0, nb-nt
}

func rectIntersects(x0, y0, x1, y1 float64, clip clipRect) bool {
	minX, maxX := math.Min(x0, x1), math.Max(x0, x1)
	minY, maxY := math.Min(y0, y1), math.Max(y0, y1)

	return maxX >= clip.x && minX <= clip.x+clip.w &&
		maxY >= clip.y && minY <= clip.y+clip.h
}

func clipTextOp(op *Op, clip clipRect) {
	left := op.X
	right := op.X

	if op.W > 0 {
		right = op.X + op.W
	}

	bottom := op.Y
	top := op.Y

	if op.H > 0 {
		top = op.Y - op.H
	} else if op.Size > 0 {
		top = op.Y - op.Size
	}

	if right < left {
		left, right = right, left
	}

	if bottom < top {
		top, bottom = bottom, top
	}

	outside := right <= clip.x || left >= clip.x+clip.w || bottom <= clip.y || top >= clip.y+clip.h
	if outside {
		DeactivateOp(op)
	}
}
