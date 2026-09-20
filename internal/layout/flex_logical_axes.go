package layout

import (
	"slices"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// flowFlexVerticalRow maps a row flex container's logical inline axis onto
// physical Y. The ordinary row path is intentionally horizontal-only; this
// path keeps vertical writing modes from silently laying their main axis on X.
//
//nolint:cyclop,wsl // logical-axis setup stays adjacent to its placement phases
func (e *engine) flowFlexVerticalRow(
	parent *box, kids []*html.Node, style ResolvedStyle,
	contentW, contentX, topY, curY, gap float64,
) float64 {
	contentH := resolveFlexColumnContentHeight(style, e)
	items := e.flexColumnItems(kids, contentW, contentH)
	if len(items) == 0 {
		return curY
	}
	if style.FlexWrap == fxWrap || style.FlexWrap == fxWrapRev {
		mapped := style
		if flexRowPhysicalReverse(style) {
			mapped.FlexDirection = fxColRev
		} else {
			mapped.FlexDirection = fxCol
		}
		mapped.Direction = cssDirectionLTR
		if style.WritingMode == writingModeVerticalRL {
			mapped.Direction = cssDirectionRTL
		}

		return e.flowFlexColumnWrapped(parent, mapped, items, contentW, contentX, topY, curY, gap, 0, contentH)
	}

	if flexRowPhysicalReverse(style) {
		slices.Reverse(items)
	}

	heights := e.flexColumnHeights(items, contentH, gap)
	sumH := 0.0
	for idx, height := range heights {
		sumH += height + flexColumnMainMargins(e, items[idx])
	}

	gaps := gap * float64(len(items)-1)
	if gaps < 0 {
		gaps = 0
	}

	startY, justifyGap := justifyColumnStart(
		flexMainJustify(style), contentH, curY, sumH+gaps, sumH, gap, len(items),
	)
	if flexRowPhysicalReverse(style) && flexStartJustify(flexMainJustify(style)) &&
		contentH >= 0 && !flexColumnHasAutoMargins(items, e) {
		startY = curY + contentH - sumH - gaps
	}
	endY := e.buildVerticalRowItems(
		parent, style, items, heights, contentW, contentX, topY, curY, startY, justifyGap, contentH,
	)

	if contentH >= 0 && endY < curY+contentH {
		return curY + contentH
	}

	return endY
}

// flowFlexVerticalColumn maps a column flex container's logical block axis
// onto physical X. Its cross axis is the logical inline axis, which is
// physical Y in vertical writing modes.
//
//nolint:cyclop,funlen,nestif,wsl // the physical-axis mapping mirrors the row path
func (e *engine) flowFlexVerticalColumn(
	parent *box, kids []*html.Node, style ResolvedStyle,
	contentW, contentX, topY, curY, gap float64,
) float64 {
	contentH := resolveContentHeight(style, e)
	items := e.flexRowItems(kids, contentW, contentH)
	if len(items) == 0 {
		return curY
	}
	if style.FlexWrap == fxWrap || style.FlexWrap == fxWrapRev {
		mapped := style
		if style.FlexDirection == fxColRev {
			mapped.FlexDirection = fxRowRev
		} else {
			mapped.FlexDirection = fxRow
		}
		mapped.Direction = cssDirectionLTR
		if style.Direction == cssDirectionRTL {
			if style.FlexWrap == fxWrap {
				mapped.FlexWrap = fxWrapRev
			} else {
				mapped.FlexWrap = fxWrap
			}
		}

		nodes := make([]*html.Node, len(items))
		for idx, item := range items {
			nodes[idx] = item.n
		}

		return e.flowFlexRow(parent, nodes, mapped, contentW, contentX, topY, curY, gap, 0)
	}

	mainReverse := style.FlexDirection == fxColRev
	if style.WritingMode == writingModeVerticalRL {
		mainReverse = !mainReverse
	}
	if mainReverse {
		slices.Reverse(items)
	}

	widths := e.flexLineWidths(items, contentW, gap)
	sumW := 0.0
	for idx, width := range widths {
		sumW += width + flexRowNormalMargins(e, items[idx])
	}

	gaps := gap * float64(len(items)-1)
	if gaps < 0 {
		gaps = 0
	}
	startX, justifyGap := justifyRowStart(
		flexMainJustify(style), contentX, contentW, sumW, gaps, gap, len(items),
	)
	if mainReverse && flexStartJustify(flexMainJustify(style)) {
		startX = contentX + contentW - sumW
	}
	if flexRowAutoMarginCount(items, e) > 0 {
		startX = contentX
		justifyGap = gap
	}

	lineCross := contentH
	if lineCross < 0 {
		lineCross = e.measureVerticalColumnCrossMax(items, widths, startX, topY)
	}

	e.buildVerticalColumnItems(
		parent, style, items, widths, contentW, contentX, topY, curY, startX, justifyGap, lineCross,
		contentH >= 0,
	)

	if contentH >= 0 && curY+contentH > curY {
		return curY + contentH
	}

	return curY + lineCross
}

//nolint:lll,wsl // intrinsic cross-size measurement mirrors the placement pass
func (e *engine) measureVerticalColumnCrossMax(
	items []flexMeas, widths []float64, startX, topY float64,
) float64 {
	was := e.noEmit
	e.noEmit = true
	maxH := 0.0
	leftX := startX
	for idx, item := range items {
		style := e.stylePtr(item.n)
		if measured := e.buildFlexRowItem(item.n, style, false, -1, widths[idx], leftX, topY); measured != nil && measured.height > maxH {
			maxH = measured.height
		}
		leftX += widths[idx] + flexRowNormalMargins(e, item)
	}
	e.noEmit = was

	return maxH
}

// Cross-axis alignment is explicit for this path and keeps the physical mapping auditable.
//
//nolint:cyclop,funlen,gocognit,nlreturn,wsl
func (e *engine) buildVerticalColumnItems(
	parent *box, style ResolvedStyle, items []flexMeas, widths []float64,
	contentW, _ /* contentX */, topY, curY, startX, justifyGap, lineCross float64, definiteCross bool,
) {
	autoMainMargins := flexRowAutoMarginCount(items, e)
	autoMainUnit := 0.0
	if autoMainMargins > 0 {
		gapCount := len(widths) - 1
		if gapCount < 0 {
			gapCount = 0
		}
		used := justifyGap * float64(gapCount)
		for idx, item := range items {
			used += widths[idx] + flexRowNormalMargins(e, item)
		}
		if free := contentW - used; free > 0 {
			autoMainUnit = free / float64(autoMainMargins)
		}
	}

	leftX := startX
	crossReverse := style.Direction == cssDirectionRTL
	baselineTarget := verticalColumnBaselineTarget(style, items, topY+curY, e)
	poll := newCtxPoll(e.ctx)
	for idx, item := range items {
		if poll.poll() {
			e.err = poll.err
			return
		}

		itemStyle := e.stylePtr(item.n)
		if itemStyle.MarginLeftAuto {
			leftX += autoMainUnit
		} else {
			leftX += e.scalePt(itemStyle.MarginLeft)
		}

		forceStretch := definiteCross && flexItemCrossStretchNode(item.n, style, *itemStyle) &&
			!flexVerticalColumnCrossAutoMargin(*itemStyle)
		crossSize := lineCross
		if forceStretch {
			crossSize -= e.scalePt(itemStyle.MarginTop) + e.scalePt(itemStyle.MarginBottom)
			if crossSize < 0 {
				crossSize = 0
			}
		}
		itemCross := crossSize
		if !forceStretch {
			itemCross = e.measureVerticalColumnItemCross(item.n, itemStyle, widths[idx], leftX, topY+curY)
		}

		override := e.forceFlexItemMainSize(*itemStyle, widths[idx])
		if forceStretch {
			override = e.forceFlexItemCrossSize(override, crossSize)
		}
		baselineY := -1.0
		if baselineTarget != nil {
			baselineY = baselineTarget[item.n]
		}
		crossY := verticalColumnCrossY(style, *itemStyle, topY+curY, lineCross, itemCross, crossReverse, baselineY, e)
		cblock := e.buildWithStyle(item.n, &override, widths[idx], leftX, crossY)
		if cblock != nil {
			dx := leftX - cblock.x
			dy := crossY - cblock.y
			e.shiftBoxOps(cblock, dx, dy)
			cblock.x += dx
			cblock.y += dy
			if parent != nil {
				parent.children = append(parent.children, cblock)
			}
		}

		leftX += widths[idx]
		if itemStyle.MarginRightAuto {
			leftX += autoMainUnit
		} else {
			leftX += e.scalePt(itemStyle.MarginRight)
		}
		if idx < len(items)-1 {
			leftX += justifyGap
		}
	}
}

// flexVerticalColumnCrossAutoMargin reports auto margins on the physical
// vertical cross axis of a column in vertical writing mode.
func flexVerticalColumnCrossAutoMargin(style ResolvedStyle) bool {
	return style.MarginTopAuto || style.MarginBottomAuto
}

//nolint:wsl // measurement toggles emission around one recursive build
func (e *engine) measureVerticalColumnItemCross(
	node *html.Node, style *ResolvedStyle, width, x, y float64,
) float64 {
	was := e.noEmit
	e.noEmit = true
	measured := e.buildFlexRowItem(node, style, false, -1, width, x, y)
	e.noEmit = was
	if measured == nil {
		return 0
	}

	return measured.height
}

//nolint:wsl // baseline target collection keeps each item decision adjacent
func verticalColumnBaselineTarget(
	container ResolvedStyle, items []flexMeas, lineTop float64, eng *engine,
) map[*html.Node]float64 {
	maxOffset := 0.0
	baselineItems := make(map[*html.Node]float64)
	for _, item := range items {
		style := eng.stylePtr(item.n)
		align := container.AlignItems
		if style.AlignSelf != "" && style.AlignSelf != fxAuto {
			align = style.AlignSelf
		}
		if align != "baseline" {
			continue
		}
		cross := eng.measureVerticalColumnItemCross(item.n, style, item.baseW, 0, lineTop)
		offset := eng.scalePt(style.MarginTop) + cross
		baselineItems[item.n] = cross
		if offset > maxOffset {
			maxOffset = offset
		}
	}
	if len(baselineItems) == 0 {
		return nil
	}
	target := lineTop + maxOffset
	for node := range baselineItems {
		baselineItems[node] = target
	}

	return baselineItems
}

//nolint:cyclop,wsl,nlreturn // vertical cross-axis alignment keeps CSS branches together
func verticalColumnCrossY(
	containerStyle, itemStyle ResolvedStyle, lineTop, lineCross, itemCross float64,
	crossReverse bool, baselineTarget float64, e *engine,
) float64 {
	top := e.scalePt(itemStyle.MarginTop)
	bottom := e.scalePt(itemStyle.MarginBottom)
	if itemStyle.MarginTopAuto || itemStyle.MarginBottomAuto {
		free := lineCross - itemCross
		if !itemStyle.MarginTopAuto {
			free -= top
		}
		if !itemStyle.MarginBottomAuto {
			free -= bottom
		}
		if free < 0 {
			free = 0
		}
		switch {
		case itemStyle.MarginTopAuto && itemStyle.MarginBottomAuto:
			return lineTop + top + free/2
		case itemStyle.MarginTopAuto:
			return lineTop + top + free
		default:
			return lineTop + top
		}
	}

	align := containerStyle.AlignItems
	if itemStyle.AlignSelf != "" && itemStyle.AlignSelf != fxAuto {
		align = itemStyle.AlignSelf
	}
	switch align {
	case "baseline":
		if baselineTarget >= 0 {
			return baselineTarget - itemCross
		}
	case fxCenter:
		return lineTop + (lineCross-itemCross)/2
	case fxFlexEnd, fxEnd:
		if crossReverse {
			return lineTop + top
		}
		return lineTop + lineCross - itemCross - bottom
	case fxFlexStart, fxStart:
		if crossReverse {
			return lineTop + lineCross - itemCross - bottom
		}
	}

	if crossReverse {
		return lineTop + lineCross - itemCross - bottom
	}

	return lineTop + top
}

//nolint:cyclop,funlen,wsl // vertical row placement mirrors the column placement phases
func (e *engine) buildVerticalRowItems(
	parent *box, style ResolvedStyle, items []flexColMeas, heights []float64,
	contentW, contentX, topY, curY, startY, justifyGap float64,
	contentH float64,
) float64 {
	leftY := startY
	endY := curY
	poll := newCtxPoll(e.ctx)
	autoMargins := 0
	used := 0.0
	for idx, item := range items {
		style := e.stylePtr(item.n)
		if style.MarginTopAuto {
			autoMargins++
		} else {
			used += e.scalePt(style.MarginTop)
		}
		if style.MarginBottomAuto {
			autoMargins++
		} else {
			used += e.scalePt(style.MarginBottom)
		}
		used += heights[idx]
		if idx < len(items)-1 {
			used += justifyGap
		}
	}
	autoUnit := 0.0
	if contentH >= 0 && autoMargins > 0 {
		if free := contentH - used; free > 0 {
			autoUnit = free / float64(autoMargins)
		}
	}

	for idx, item := range items {
		if poll.poll() {
			e.err = poll.err

			return endY
		}

		cstate := e.stylePtr(item.n)
		if cstate.MarginTopAuto {
			leftY += autoUnit
		} else {
			leftY += e.scalePt(cstate.MarginTop)
		}

		override := e.flexColumnItemOverride(item.n, *cstate, style, heights[idx], contentW)
		cblock := e.buildWithStyle(item.n, &override, contentW, contentX, topY+leftY)
		if cblock != nil {
			targetX := verticalRowCrossX(style, contentX, contentW, *cstate, cblock.w, e)
			dx := targetX - cblock.x
			dy := (topY + leftY) - cblock.y
			e.shiftBoxOps(cblock, dx, dy)
			cblock.x += dx
			cblock.y += dy

			if parent != nil {
				parent.children = append(parent.children, cblock)
			}
		}

		leftY += heights[idx]
		if cstate.MarginBottomAuto {
			leftY += autoUnit
		} else {
			leftY += e.scalePt(cstate.MarginBottom)
		}
		endY = leftY
		if idx < len(items)-1 {
			leftY += justifyGap
			endY = leftY
		}
	}

	return endY
}

//nolint:cyclop,nlreturn,varnamelen,wsl // vertical writing maps cross-axis alignment explicitly
func verticalRowCrossX(
	containerStyle ResolvedStyle, contentX, contentW float64,
	itemStyle ResolvedStyle, itemW float64,
	e *engine,
) float64 {
	align := containerStyle.AlignItems
	if itemStyle.AlignSelf != "" && itemStyle.AlignSelf != fxAuto {
		align = itemStyle.AlignSelf
	}

	left := e.scalePt(itemStyle.MarginLeft)
	right := e.scalePt(itemStyle.MarginRight)
	if itemStyle.MarginLeftAuto || itemStyle.MarginRightAuto {
		free := contentW - itemW
		if !itemStyle.MarginLeftAuto {
			free -= left
		}
		if !itemStyle.MarginRightAuto {
			free -= right
		}
		if free < 0 {
			free = 0
		}
		switch {
		case itemStyle.MarginLeftAuto && itemStyle.MarginRightAuto:
			return contentX + left + free/2
		case itemStyle.MarginLeftAuto:
			return contentX + left + free
		default:
			return contentX + left
		}
	}

	if align == fxCenter {
		return contentX + (contentW-itemW)/2
	}

	reverseCross := containerStyle.WritingMode == writingModeVerticalRL
	switch align {
	case fxFlexEnd, fxEnd:
		if reverseCross {
			return contentX + left
		}
		return contentX + contentW - itemW - right
	case fxFlexStart, fxStart:
		if reverseCross {
			return contentX + contentW - itemW - right
		}
	}
	if reverseCross {
		return contentX + contentW - itemW - right
	}

	return contentX + left
}

// flexRowPhysicalReverse reports whether the row main axis progresses toward
// the physical left or top edge when the logical direction is applied.
// direction reverses row and row-reverse in horizontal and vertical writing.
func flexRowPhysicalReverse(style ResolvedStyle) bool {
	reverse := style.FlexDirection == fxRowRev
	if style.Direction == cssDirectionRTL {
		return !reverse
	}

	return reverse
}

// flexRowNormalMargins returns the non-auto outer margins that consume row
// main-axis space. Auto margins absorb only the positive space left after
// flex sizing.
//
//nolint:varnamelen,wsl // e is the package's established engine receiver name
func flexRowNormalMargins(e *engine, item flexMeas) float64 {
	style := e.stylePtr(item.n)
	margins := 0.0
	if !style.MarginLeftAuto {
		margins += e.scalePt(style.MarginLeft)
	}
	if !style.MarginRightAuto {
		margins += e.scalePt(style.MarginRight)
	}

	return margins
}

// flexColumnMainMargins returns the normal, non-auto margins that consume
// space on a column flex container's main axis. Flex items do not collapse
// their vertical margins with one another.
//
//nolint:varnamelen,wsl // e is the package's established engine receiver name
func flexColumnMainMargins(e *engine, item flexColMeas) float64 {
	style := e.stylePtr(item.n)
	margins := 0.0
	if !style.MarginTopAuto {
		margins += e.scalePt(style.MarginTop)
	}
	if !style.MarginBottomAuto {
		margins += e.scalePt(style.MarginBottom)
	}

	return margins
}
