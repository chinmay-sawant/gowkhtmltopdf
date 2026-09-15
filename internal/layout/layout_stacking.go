package layout

// This file owns the engine's stacking and compositing scope: entering an
// element's scope (pushZ), leaving it (popZ), and stamping the active scope
// onto emitted ops. The stacking-context frame chain built here is what
// paint_order.go consumes to keep ancestor chrome below descendant content.
// Extracted from layout.go to keep that file under the size ledger.

// blendScope captures the state pushZ overrides for one element so popZ can
// restore it and close the element's group.
type blendScope struct {
	prevZ          int
	prevZSet       bool
	prevPositioned bool
	prevFrame      *paintCtxFrame
	prevBlendMode  string
	prevGroup      *BlendGroup
	prevGroupOwner *ResolvedStyle
	// beginMark is the op index of this element's begin marker, -1 when the
	// element creates no group.
	beginMark int
}

// pushZ enters one element's stacking and compositing scope, emitting the
// group begin marker when mix-blend-mode or isolation creates a group. owner
// is the resolved style pointer that identifies the element.
func (e *engine) pushZ(style ResolvedStyle, owner *ResolvedStyle) blendScope {
	prev := blendScope{
		prevZ:          e.zIndex,
		prevZSet:       e.zIndexSet,
		prevPositioned: e.positioned,
		prevFrame:      e.zFrame,
		prevBlendMode:  e.blendMode,
		prevGroup:      e.blendGroup,
		prevGroupOwner: e.blendGroupOwner,
		beginMark:      -1,
	}
	createsBlendContext := style.MixBlendMode != blendNormal || style.Isolation == "isolate"

	if style.Position == positionAbsolute || style.Position == positionFixed {
		e.positioned = true
	}

	e.enterStackingContext(style, createsBlendContext)
	e.enterBlendIsolation(style, owner)

	if e.blendGroup != prev.prevGroup {
		prev.beginMark = e.addGroupMark(e.blendGroup, groupMarkBegin)
	}

	return prev
}

// enterStackingContext applies CSS stacking-context creation: an explicit
// z-index wins, otherwise transform, opacity, blend, or isolation reset the
// level to zero. Entering a context also pushes a frame so paint order can
// tell an ancestor's chrome from its own content.
func (e *engine) enterStackingContext(style ResolvedStyle, createsBlendContext bool) {
	if style.ZIndexSet {
		e.zIndex = style.ZIndex
		e.zIndexSet = true
		e.zFrame = e.pushFrame(style.ZIndex)
	} else if style.HasTransform || style.Opacity < 1 || createsBlendContext {
		// CSS: transform, opacity, blend, and isolation create a stacking context.
		e.zIndex = 0
		e.zIndexSet = true
		e.zFrame = e.pushFrame(0)
	}
}

// enterBlendIsolation applies the transform stamp and enters the element's
// blend group. An HTML element that blends, or that isolates, is an isolated
// group per CSS Compositing 3.2, so the group starts on a transparent
// backdrop; descendants inherit the group membership instead of the mode.
func (e *engine) enterBlendIsolation(style ResolvedStyle, owner *ResolvedStyle) {
	if style.HasTransform || style.Opacity < 1 {
		e.needsXformStamp = true
	}

	// normalizeBlendMode accepts the CSS vocabulary; the style apply arm
	// already normalized MixBlendMode, so only "normal"/empty skip.
	mode := ""
	if style.MixBlendMode != "" && style.MixBlendMode != blendNormal {
		mode = style.MixBlendMode
	}

	if mode == "" && style.Isolation != "isolate" {
		return
	}

	e.nextGroupID++
	e.blendGroup = &BlendGroup{
		ID:      e.nextGroupID,
		Mode:    mode,
		Isolate: true,
		Parent:  e.blendGroup,
	}
	e.blendGroupOwner = owner

	if style.Isolation == "isolate" {
		e.blendMode = ""
	}

	if mode != "" {
		e.blendMode = mode
	}
}

// addGroupMark appends a begin/end boundary op for group and returns its op
// index, or -1 when emission is suppressed.
func (e *engine) addGroupMark(group *BlendGroup, mark uint8) int {
	if group == nil || e.checkContext() || e.noEmit {
		return -1
	}

	e.nextOpID++

	markOp := Op{ //nolint:exhaustruct // marker ops carry only identity and scope
		ID:   e.nextOpID,
		Kind: OpUnknown,
	}
	e.stampPaintState(&markOp)
	markOp.setGroupMark(group, mark)
	e.ops = append(e.ops, markOp)

	return len(e.ops) - 1
}

// patchGroupMark stamps the owning element's geometry onto a boundary marker.
// Flow scans (table sliver repair, row tops, page buckets) read op Y, so a
// zero-height marker at the canvas origin would move unrelated boxes.
func (e *engine) patchGroupMark(idx int, posX, posY, width, height float64) {
	if idx < 0 || idx >= len(e.ops) {
		return
	}

	e.ops[idx].X = posX
	e.ops[idx].Y = posY
	e.ops[idx].W = width
	e.ops[idx].H = height
}

// popZ restores the scope pushZ saved. When the element created a group, the
// end marker is emitted here so the markers stay balanced even when the
// subtree build emitted no paint operations.
func (e *engine) popZ(scope blendScope, boxNode *box) {
	if scope.beginMark >= 0 {
		if boxNode != nil {
			e.patchGroupMark(scope.beginMark, boxNode.x, boxNode.y, boxNode.w, boxNode.height)
		}

		endMark := e.addGroupMark(e.blendGroup, groupMarkEnd)

		if boxNode != nil {
			e.patchGroupMark(endMark, boxNode.x, boxNode.y, boxNode.w, boxNode.height)
		}
	}

	e.zIndex = scope.prevZ
	e.zIndexSet = scope.prevZSet
	e.positioned = scope.prevPositioned
	e.zFrame = scope.prevFrame
	e.blendMode = scope.prevBlendMode
	e.blendGroup = scope.prevGroup
	e.blendGroupOwner = scope.prevGroupOwner
}

// pushFrame enters a stacking context and returns its frame. A non-none
// transform creates a context even when it resolves to identity, so frames
// are pushed for every parsed transform, not only visually active ones.
func (e *engine) pushFrame(z int) *paintCtxFrame {
	e.nextCtxSeq++
	e.zFrame = &paintCtxFrame{
		parent:     e.zFrame,
		z:          z,
		positioned: e.positioned,
		seq:        e.nextCtxSeq,
		depth:      chainDepth(e.zFrame) + 1,
	}

	return e.zFrame
}

// stampPaintState stamps the active stacking scope (flat level plus context
// chain) onto an emitted op.
func (e *engine) stampPaintState(op *Op) {
	op.bindEmptyExtra()
	op.ZIndex = e.zIndex
	op.ZIndexSet = e.zIndexSet
	op.Positioned = e.positioned
	op.setZChain(e.zFrame)
}
