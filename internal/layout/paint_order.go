package layout

import "sort"

// paintCtxFrame is one entered stacking context on the element walk. Frames
// form an immutable parent chain and are created once per context-creating
// element (explicit z-index, transform, opacity < 1, blend, isolation); every
// op inside that element holds a pointer to its innermost frame. Paint order
// compares chains so ancestor chrome always paints below descendant content,
// which a flat z-index sort cannot express. A non-none transform creates a
// frame even when it resolves to identity: CSS does the same.
type paintCtxFrame struct {
	parent *paintCtxFrame
	// z is this context's z-index within its parent context.
	z int
	// positioned mirrors Op.Positioned at entry: the subtree came from an
	// absolute/fixed box.
	positioned bool
	// seq is the context's creation order (document pre-order), the tiebreak
	// for equal-z sibling contexts.
	seq int
	// depth counts this frame and its ancestors (1 for a root-level context).
	depth int
}

// PaintOrder returns operation indices in the canonical display-list paint
// order. PDF and raster adapters consume this same policy; backend drawing
// remains responsible for interpreting each operation.
func PaintOrder(ops []Op) []int {
	idx := make([]int, len(ops))
	for i := range ops {
		idx[i] = i
	}

	sortPaintIndices(ops, idx)

	return idx
}

// paintOrderSubset returns an ordered copy of an operation subset. All
// adapters use this policy so a band cannot accidentally fall back to source
// order while the paginated body uses paint order.
func paintOrderSubset(ops []Op, idxs []int) []int {
	ordered := append([]int(nil), idxs...)
	sortPaintIndices(ops, ordered)

	return ordered
}

// sortPaintIndices sorts an existing operation subset without changing which
// operations it contains. Pagination tests use this compatibility helper for
// fixed and per-page lists; the comparison itself lives in paintOrderBefore.
func sortPaintIndices(ops []Op, idxs []int) {
	sort.SliceStable(idxs, func(i, j int) bool {
		return paintOrderBefore(ops, idxs[i], idxs[j])
	})
}

func paintOrderBefore(ops []Op, left, right int) bool {
	leftOp, rightOp := &ops[left], &ops[right]

	if before, decided := paintStackingBefore(leftOp, rightOp); decided {
		return before
	}

	// Same stacking context (or legacy ops without a chain): flat order.
	leftZ, rightZ := 0, 0
	if leftOp.ZIndexSet {
		leftZ = leftOp.ZIndex
	}

	if rightOp.ZIndexSet {
		rightZ = rightOp.ZIndex
	}

	if leftZ != rightZ {
		return leftZ < rightZ
	}

	if leftOp.Positioned != rightOp.Positioned {
		return !leftOp.Positioned
	}

	leftLayer, rightLayer := paintLayer(leftOp), paintLayer(rightOp)
	if leftLayer != rightLayer {
		return leftLayer < rightLayer
	}

	return left < right
}

// paintStackingBefore orders two ops by stacking-context nesting and reports
// whether the chains decided the order. Ops in the same context are left to
// the flat comparison.
func paintStackingBefore(left, right *Op) (bool, bool) {
	leftChain, rightChain := left.zChain(), right.zChain()
	if leftChain == rightChain {
		return false, false
	}

	if chainIsAncestor(leftChain, rightChain) {
		divergent := chainFrameAtDepth(rightChain, chainDepth(leftChain)+1)

		return paintAncestorContextFirst(left, divergent), true
	}

	if chainIsAncestor(rightChain, leftChain) {
		divergent := chainFrameAtDepth(leftChain, chainDepth(rightChain)+1)

		return !paintAncestorContextFirst(right, divergent), true
	}

	// Sibling contexts: the deepest divergent frames decide (z, then
	// positioned, then document order).
	leftDiv, rightDiv := divergentFrames(leftChain, rightChain)

	switch {
	case leftDiv.z != rightDiv.z:
		return leftDiv.z < rightDiv.z, true
	case leftDiv.positioned != rightDiv.positioned:
		return !leftDiv.positioned, true
	case leftDiv.seq != rightDiv.seq:
		return leftDiv.seq < rightDiv.seq, true
	}

	return false, false
}

// paintAncestorContextFirst reports whether an op sitting at an ancestor
// context level paints before an op inside the deeper context divergent. CSS
// paints an element's own content before child contexts with z-index >= 0 and
// after negative-z child contexts; chrome (background, border, shadow)
// always paints before descendant content.
func paintAncestorContextFirst(ancestorOp *Op, divergent *paintCtxFrame) bool {
	if divergent.z < 0 && !ancestorOp.isChrome() {
		return false
	}

	return true
}

// chainDepth returns the number of contexts on a chain; the root context is 0.
func chainDepth(frame *paintCtxFrame) int {
	if frame == nil {
		return 0
	}

	return frame.depth
}

// chainIsAncestor reports whether anc is a strict ancestor of desc; the nil
// root chain is an ancestor of every context chain.
func chainIsAncestor(anc, desc *paintCtxFrame) bool {
	ancDepth, descDepth := chainDepth(anc), chainDepth(desc)
	if ancDepth >= descDepth {
		return false
	}

	return chainFrameAtDepth(desc, ancDepth) == anc
}

// chainFrameAtDepth walks a chain up to the frame at depth, or nil when the
// chain is shorter.
func chainFrameAtDepth(frame *paintCtxFrame, depth int) *paintCtxFrame {
	for frame != nil && frame.depth > depth {
		frame = frame.parent
	}

	return frame
}

// divergentFrames returns the deepest frames where two chains split. Both
// chains must be non-nil, different, and neither an ancestor of the other.
func divergentFrames(left, right *paintCtxFrame) (*paintCtxFrame, *paintCtxFrame) {
	leftFrame, rightFrame := left, right

	for chainDepth(leftFrame) > chainDepth(rightFrame) {
		leftFrame = leftFrame.parent
	}

	for chainDepth(rightFrame) > chainDepth(leftFrame) {
		rightFrame = rightFrame.parent
	}

	leftDiv, rightDiv := leftFrame, rightFrame

	for leftFrame != nil && rightFrame != nil && leftFrame != rightFrame {
		leftDiv, rightDiv = leftFrame, rightFrame
		leftFrame, rightFrame = leftFrame.parent, rightFrame.parent
	}

	return leftDiv, rightDiv
}

// paintLayer orders ops within a z-index band: chrome under content.
func paintLayer(op *Op) int {
	if op.IsBackground {
		return 0
	}

	switch op.Kind {
	case OpFillRect, OpStrokeRect, OpLine, OpGridRun:
		return 0
	case OpText, OpImage, OpLinkURI, OpBullet, OpUnknown, opKindNoop:
		return 1
	}

	return 1
}
