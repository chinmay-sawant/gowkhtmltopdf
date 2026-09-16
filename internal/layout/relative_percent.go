package layout

// Deferred percentage insets for position:relative boxes.
//
// A relative box shifts from its static position by top/left (or
// bottom/right). Percentages resolve against the containing block: the height
// for top/bottom, the width for left/right (CSS 2.1 9.4.3). The height is only
// known after the containing block's subtree is built, so percentage terms are
// registered here during the build and applied in one post-measure pass, after
// the root box finalizes. Length terms keep the build-time path
// (applyRelativeOffset in flex.go).
//
// CSS 2.1 cyclic-percentage policy: when the containing block height is auto,
// the percentage is treated as 0 and the box keeps its static position. That
// keeps an auto-height wrapper from shifting by half its content height; the
// engine's earlier viewport-height probe moved the learncpp masthead 392.6pt
// (see style_calc_test.go TestPercentTopResolvesAgainstContainingBlock).

// registerRelativePct defers a relative box's percentage insets to
// resolveRelativePercents. Measurement passes (noEmit) build throwaway boxes,
// so they are skipped.
func (e *engine) registerRelativePct(boxNode *box) {
	if e.noEmit || boxNode == nil {
		return
	}

	e.pendingRelPct = append(e.pendingRelPct, boxNode)
}

// resolveRelativePercents applies the deferred percentage insets against each
// box's containing block (the parent box's final content box). Call once after
// the document build, before finalizeChrome and the transform stamps.
func (e *engine) resolveRelativePercents(root *box) {
	if root == nil || len(e.pendingRelPct) == 0 {
		return
	}

	pending := make(map[*box]struct{}, len(e.pendingRelPct))
	for _, b := range e.pendingRelPct {
		pending[b] = struct{}{}
	}

	e.walkRelativePercents(root, pending)
	e.pendingRelPct = e.pendingRelPct[:0]
}

func (e *engine) walkRelativePercents(parent *box, pending map[*box]struct{}) {
	if parent == nil || parent.style == nil {
		return
	}

	_, contentW := e.contentBox(parent.x, parent.w, parent.style)
	contentH := 0.0

	// Definite height only; resolveContentHeight returns -1 for auto and for
	// an unresolved percent height (cyclic honesty), matching the CSS rule.
	if resolveContentHeight(*parent.style, e) >= 0 {
		contentH = e.contentBoxHeight(parent)
	}

	for _, child := range parent.children {
		if _, ok := pending[child]; ok {
			e.applyRelativePercent(child, contentW, contentH)
			delete(pending, child)
		}

		e.walkRelativePercents(child, pending)
	}
}

// contentBoxHeight returns the content-box height of a border box.
func (e *engine) contentBoxHeight(boxNode *box) float64 {
	sty := boxNode.style
	if sty == nil {
		return 0
	}

	height := boxNode.height -
		e.scalePt(sty.PaddingTop) - e.scalePt(sty.PaddingBottom) -
		e.scalePt(borderLayoutWidth(sty, sty.BorderTop)) -
		e.scalePt(borderLayoutWidth(sty, sty.BorderBottom))

	if height < 0 {
		return 0
	}

	return height
}

// applyRelativePercent shifts one relative box by its deferred percentage
// insets and moves its ops with it.
func (e *engine) applyRelativePercent(boxNode *box, contentW, contentH float64) {
	sty := boxNode.style
	if sty == nil || sty.Position != positionRelative {
		return
	}

	var deltaX, deltaY float64

	switch {
	case sty.LeftPercent >= 0:
		deltaX = insetUsedPercent(sty.LeftPercent, contentW)
	case sty.RightPercent >= 0:
		deltaX = -insetUsedPercent(sty.RightPercent, contentW)
	}

	switch {
	case sty.TopPercent >= 0:
		deltaY = insetUsedPercent(sty.TopPercent, contentH)
	case sty.BottomPercent >= 0:
		deltaY = -insetUsedPercent(sty.BottomPercent, contentH)
	}

	if deltaX == 0 && deltaY == 0 {
		return
	}

	boxNode.x += deltaX
	boxNode.y += deltaY
	e.shiftBoxOps(boxNode, deltaX, deltaY)
}
