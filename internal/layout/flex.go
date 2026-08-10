package layout

import (
	"sort"

	"gowkhtmltopdf/internal/html"
)

const (
	fxAround    = "space-around"
	fxAuto      = "auto"
	fxBetween   = "space-between"
	fxCenter    = "center"
	fxCol       = "column"
	fxColRev    = "column-reverse"
	fxEnd       = "end"
	fxEvenly    = "space-evenly"
	fxFlexEnd   = "flex-end"
	fxFlexStart = "flex-start"
	fxRow       = "row"
	fxStart     = "start"
	fxStretch   = "stretch"
	fxWrapRev   = "wrap-reverse"
)

type flexMeas struct {
	n      *html.Node
	baseW  float64
	grow   float64
	shrink float64
	order  int
}

type flexColMeas struct {
	n      *html.Node
	baseH  float64
	grow   float64
	shrink float64
}

type flexLinePlace struct {
	startChild int
	endChild   int
	y0         float64
	h          float64
}

type flexPlacedItem struct {
	box *box
	h   float64
	n   *html.Node
}

// buildFlex lays out a flex container (row or column) with a report-friendly
// subset: justify-content, align-items/self, align-content, gap/row-gap/
// column-gap, flex-grow/shrink/basis, order, wrap, and reverse directions.
func (e *engine) buildFlex(node *html.Node, sty ResolvedStyle, availW, x, posY float64) *box {
	ml := e.scalePt(sty.MarginLeft)
	boxNode := &box{ //nolint:exhaustruct // intentional zero fields
		node: node, style: e.stylePtr(node), kind: displayBlock, x: x + ml, y: posY,
	}
	boxNode.w = resolveUsedWidth(sty, availW, e)
	contentX, contentW := e.contentBox(boxNode.x, boxNode.w, sty)
	contentStart := len(e.ops)
	curY := e.scalePt(sty.PaddingTop) + e.scalePt(sty.BorderTop.Width)

	kids := make([]*html.Node, 0, len(node.Children))

	for _, child := range node.Children {
		if child.Type != html.ElementNode {
			continue
		}

		cs := e.styles[child]
		if cs.Display == cssDisplayNone {
			continue
		}

		kids = append(kids, child)
	}

	rowGap, colGap := e.styleGaps(sty)

	dir := sty.FlexDirection
	if dir == "" {
		dir = fxRow
	}

	if dir == fxCol || dir == fxColRev {
		curY = e.flowFlexColumn(boxNode, kids, sty, contentW, contentX, posY, curY, rowGap)
	} else {
		curY = e.flowFlexRow(boxNode, kids, sty, contentW, contentX, posY, curY, colGap, rowGap)
	}

	curY += e.scalePt(sty.PaddingBottom)

	if sty.Height >= 0 {
		height := e.scalePt(sty.Height)
		if sty.BoxSizing != borderBox {
			height += e.scalePt(sty.PaddingTop) + e.scalePt(sty.PaddingBottom) +
				e.scalePt(sty.BorderTop.Width) + e.scalePt(sty.BorderBottom.Width)
		}

		if curY < height {
			curY = height
		}
	}

	boxNode.height = curY
	e.prependChrome(contentStart, boxNode, sty, boxNode.x, posY, boxNode.w, boxNode.height)

	return boxNode
}

func (e *engine) flowFlexRow(
	parent *box, kids []*html.Node, style ResolvedStyle,
	contentW, contentX, topY, curY, colGap, rowGap float64,
) float64 {
	if len(kids) == 0 {
		return curY
	}

	wrap := style.FlexWrap == "wrap" || style.FlexWrap == fxWrapRev
	reverse := style.FlexDirection == "row-reverse"
	items := e.flexRowItems(kids, contentW)
	lines := flexWrapLines(items, wrap, reverse, style.FlexWrap == fxWrapRev, colGap, contentW)

	lineCross := -1.0
	if !wrap {
		lineCross = resolveContentHeight(style, e)
	}

	placed := make([]flexLinePlace, 0, len(lines))

	for lidx, line := range lines {
		startChild := 0
		if parent != nil {
			startChild = len(parent.children)
		}

		yStart := curY
		curY = e.placeFlexLineMeasured(parent, style, line, contentW, contentX, topY, curY, colGap, lineCross)

		endChild := startChild
		if parent != nil {
			endChild = len(parent.children)
		}

		placed = append(placed, flexLinePlace{startChild: startChild, endChild: endChild, y0: yStart, h: curY - yStart})

		if lidx < len(lines)-1 {
			curY += rowGap
		}
	}

	return e.applyAlignContentRow(parent, style.AlignContent, placed, rowGap, resolveContentHeight(style, e), curY)
}

func (e *engine) flexRowItems(kids []*html.Node, contentW float64) []flexMeas {
	items := make([]flexMeas, 0, len(kids))

	for _, kid := range kids {
		cstate := e.styles[kid]

		grow := cstate.FlexGrow
		if grow < 0 {
			grow = 0
		}

		shrink := cstate.FlexShrink
		if shrink < 0 {
			shrink = 1
		}

		items = append(items, flexMeas{
			n: kid, baseW: e.flexItemBaseWidth(kid, *cstate, contentW),
			grow: grow, shrink: shrink, order: cstate.FlexOrder,
		})
	}

	sort.SliceStable(items, func(i, j int) bool { return items[i].order < items[j].order })

	return items
}

// flexWrapLines packs measured items into flex lines, honoring wrap mode and
// reverse direction (item order within each line, then line order).
func flexWrapLines(items []flexMeas, wrap, reverse, wrapReverse bool, colGap, contentW float64) [][]flexMeas {
	if !wrap {
		if reverse {
			reverseFlexMeas(items)
		}

		return [][]flexMeas{items}
	}

	var lines [][]flexMeas

	line := make([]flexMeas, 0, len(items))
	used := 0.0

	for _, item := range items {
		need := item.baseW
		if len(line) > 0 {
			need += colGap
		}

		if len(line) > 0 && used+need > contentW+1e-6 {
			lines = append(lines, finalizeFlexLine(line, reverse))
			line = nil
			used = 0
			need = item.baseW
		}

		line = append(line, item)
		used += need
	}

	if len(line) > 0 {
		lines = append(lines, finalizeFlexLine(line, reverse))
	}

	if wrapReverse {
		for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
			lines[i], lines[j] = lines[j], lines[i]
		}
	}

	return lines
}

func finalizeFlexLine(line []flexMeas, reverse bool) []flexMeas {
	if reverse {
		reverseFlexMeas(line)
	}

	return line
}

func reverseFlexMeas(line []flexMeas) {
	for i, j := 0, len(line)-1; i < j; i, j = i+1, j-1 {
		line[i], line[j] = line[j], line[i]
	}
}

// applyAlignContentRow distributes free cross space between wrapped flex
// lines when the container height is definite and wrapping produced multiple
// lines. Height:auto → pack at start (no-op).
func (e *engine) applyAlignContentRow(
	parent *box, alignContent string, placed []flexLinePlace,
	rowGap, contentH, curY float64,
) float64 {
	if contentH < 0 || len(placed) <= 1 {
		return curY
	}

	linesH := 0.0
	for _, line := range placed {
		linesH += line.h
	}

	free := contentH - linesH - rowGap*float64(len(placed)-1)
	if free <= layoutEpsilon {
		return curY
	}

	offsets := alignContentOffsets(alignContent, free, len(placed))

	if parent != nil {
		for i, line := range placed {
			deltaY := offsets[i]
			if deltaY == 0 {
				continue
			}

			e.shiftPlacedChildren(parent, line.startChild, line.endChild, deltaY)
		}
	}

	last := placed[len(placed)-1]

	return last.y0 + offsets[len(offsets)-1] + last.h
}

func (e *engine) shiftPlacedChildren(parent *box, startChild, endChild int, deltaY float64) {
	for ci := startChild; ci < endChild && ci < len(parent.children); ci++ {
		cb := parent.children[ci]
		e.shiftBoxOps(cb, 0, deltaY)
		cb.y += deltaY
	}
}

// alignContentOffsets maps align-content to per-line cross-axis offsets.
func alignContentOffsets(alignContent string, free float64, count int) []float64 {
	base, step := 0.0, 0.0

	switch alignContent {
	case fxFlexEnd, fxEnd:
		base = free
	case fxCenter:
		base = free / two
	case fxBetween:
		step = free / float64(count-1)
	case fxAround:
		unit := free / float64(two*count)
		base, step = unit, two*unit
	case fxEvenly:
		unit := free / float64(count+1)
		base, step = unit, unit
	}

	offsets := make([]float64, count)
	for i := range offsets {
		offsets[i] = base + step*float64(i)
	}

	return offsets
}

// flexItemBaseWidth resolves the flex base size on the row main axis.
// mainSize is the flex container content-box width, or <0 when indefinite
// (shrink-to-fit). Percentage flex-basis against an indefinite main size is
// treated as auto (content-based) — CSS Flexbox L1 §9.2 cyclic %-sizing subset.
func (e *engine) flexItemBaseWidth(node *html.Node, style ResolvedStyle, mainSize float64) float64 {
	pad := e.scalePt(style.PaddingLeft) + e.scalePt(style.PaddingRight) +
		e.scalePt(style.BorderLeft.Width) + e.scalePt(style.BorderRight.Width)

	if w, ok := e.flexSpecifiedBaseWidth(style, mainSize, pad); ok {
		return w
	}

	capW := mainSize
	if capW < 0 {
		capW = 1e9
	}

	intr := e.measureFlexItemMaxContent(node, style) + pad +
		e.scalePt(style.MarginLeft) + e.scalePt(style.MarginRight)
	if intr <= 0 {
		intr = pad + e.scalePt(style.FontSize)*two
	}

	if intr > capW {
		intr = capW
	}

	return intr
}

// measureFlexItemMaxContent includes block descendants in a flex item's
// intrinsic width. The generic cell measure intentionally stops at nested
// block formatting contexts, which is correct for table-cell line collection
// but makes a wrapper such as a section header measure only its eyebrow and
// collapse the heading into a narrow column.
//
//nolint:cyclop,wsl // intrinsic flex measurement keeps the CSS cases together
func (e *engine) measureFlexItemMaxContent(node *html.Node, style ResolvedStyle) float64 {
	_, maxW := e.measureCellMinMax(node, style)
	chrome := e.scalePt(style.PaddingLeft) + e.scalePt(style.PaddingRight) +
		e.scalePt(style.BorderLeft.Width) + e.scalePt(style.BorderRight.Width)
	contentW := maxW - chrome
	if contentW < 0 {
		contentW = 0
	}
	if style.Width >= 0 {
		specified := e.flexBoxSized(style, e.scalePt(style.Width), chrome)
		if specified > contentW+chrome {
			contentW = specified - chrome
		}
	}
	rowFlex := style.Display == displayFlex && style.FlexDirection != fxCol && style.FlexDirection != fxColRev
	rowContentW := 0.0
	rowChildCount := 0

	for _, child := range node.Children {
		if child.Type != html.ElementNode {
			continue
		}

		childStyle := e.styles[child]
		if childStyle == nil || childStyle.Display == cssDisplayNone {
			continue
		}

		childW := e.measureFlexItemMaxContent(child, *childStyle)
		if rowFlex {
			rowContentW += childW
			rowChildCount++
		} else if childW > contentW {
			contentW = childW
		}
	}

	if rowFlex {
		_, columnGap := e.styleGaps(style)
		if rowChildCount > 1 {
			rowContentW += columnGap * float64(rowChildCount-1)
		}

		if rowContentW > contentW {
			contentW = rowContentW
		}
	}

	return contentW + chrome
}

// flexSpecifiedBaseWidth resolves a definite flex base size (flex-basis then
// width). Cyclic percentage basis against an indefinite main size falls
// through to the next specified size, then content.
func (e *engine) flexSpecifiedBaseWidth(style ResolvedStyle, mainSize, pad float64) (float64, bool) {
	switch {
	case style.FlexBasisPercent >= 0 && mainSize >= 0:
		return e.flexBoxSized(style, mainSize*style.FlexBasisPercent/cssPercent, pad), true
	case style.FlexBasis >= 0:
		return e.flexBoxSized(style, e.scalePt(style.FlexBasis), pad), true
	case style.WidthPercent >= 0 && mainSize >= 0:
		return e.flexBoxSized(style, mainSize*style.WidthPercent/cssPercent, pad), true
	case style.Width >= 0:
		return e.flexBoxSized(style, e.scalePt(style.Width), pad), true
	}

	return 0, false
}

// flexBoxSized converts a content-box size to border-box when BoxSizing says so.
func (e *engine) flexBoxSized(style ResolvedStyle, size, pad float64) float64 {
	if style.BoxSizing != borderBox {
		return size + pad
	}

	return size
}

// flexMinMainSize is the content-based minimum main size (Flexbox §4.5 lite /
// css-sizing-3): max(specified min-width, min(min-content suggestion,
// specified size suggestion when definite)). Used as the shrink floor so text
// does not crush to 0 while ordinary wrapping text can still share a flex row.
// measureCellMinMax already returns border-box widths, so padding and borders
// must not be added a second time here. mainSize is the definite flex
// container content main size, or <0 when indefinite (then % min-width is
// ignored — cyclic honesty).
func (e *engine) flexMinMainSize(item flexMeas, mainSize float64) float64 {
	cstate := e.styles[item.n]
	floor := 0.0
	if cstate.MinWidthSet {
		if cstate.MinWidthPercent >= 0 && mainSize >= 0 {
			return mainSize * cstate.MinWidthPercent / cssPercent
		}

		return e.scalePt(cstate.MinWidth)
	}

	if cstate.MinWidthPercent >= 0 && mainSize >= 0 {
		floor = mainSize * cstate.MinWidthPercent / cssPercent
	} else if cstate.MinWidth > 0 {
		floor = e.scalePt(cstate.MinWidth)
	}
	// Automatic minimum (min-width:auto): min-content size suggestion. The
	// returned width already includes the item's horizontal chrome.
	contentSug, _ := e.measureCellMinMax(item.n, *cstate)
	pad := e.scalePt(cstate.PaddingLeft) + e.scalePt(cstate.PaddingRight) +
		e.scalePt(cstate.BorderLeft.Width) + e.scalePt(cstate.BorderRight.Width)
	// Specified size suggestion when width/% is definite against mainSize.
	specSug := e.flexSpecifiedWidthSuggestion(*cstate, item.baseW, mainSize, pad)

	autoMin := contentSug
	if specSug >= 0 && specSug < autoMin {
		autoMin = specSug
	}
	// Overflow non-visible → automatic min size is 0 (CSS Flexbox §4.5).
	if overflowCreatesStickyScrollport(cstate.Overflow) {
		autoMin = 0
	}

	if autoMin > floor {
		floor = autoMin
	}

	return floor
}

func (e *engine) flexSpecifiedWidthSuggestion(style ResolvedStyle, baseW, mainSize, pad float64) float64 {
	switch {
	case style.WidthPercent >= 0 && mainSize >= 0:
		return e.flexBoxSized(style, mainSize*style.WidthPercent/cssPercent, pad)
	case style.Width >= 0:
		return e.flexBoxSized(style, e.scalePt(style.Width), pad)
	case baseW > 0:
		return baseW
	}

	return -1
}

// flexClampMainWidths applies min/max after grow/shrink, then re-resolves so
// that percentage-driven floors and content mins that raised used sizes are
// honored without leaving the line sum inconsistent when space remains.
func (e *engine) flexClampMainWidths(items []flexMeas, widths []float64, contentW, mainSize float64) {
	for idx, it := range items {
		cstate := e.styles[it.n]

		floor := e.flexMinMainSize(it, mainSize)
		if widths[idx] < floor {
			widths[idx] = floor
		}

		if cstate.MaxWidth >= 0 {
			mx := e.scalePt(cstate.MaxWidth)
			if widths[idx] > mx {
				widths[idx] = mx
			}
		}
	}
	// If mins pushed the sum over contentW, freeze at floors and re-shrink
	// remaining flexible items (css-sizing / flex redistribution lite).
	sum := 0.0
	for _, w := range widths {
		sum += w
	}

	if sum <= contentW+1e-6 || contentW < 0 {
		return
	}

	e.reshrinkFlexWidths(items, widths, contentW, mainSize, sum)
}

// reshrinkFlexWidths iteratively cuts flexible items back toward contentW,
// freezing items at their min floors until the deficit is exhausted.
func (e *engine) reshrinkFlexWidths(items []flexMeas, widths []float64, contentW, mainSize, sum float64) {
	deficit := sum - contentW

	for deficit > 1e-6 {
		shrinkable := e.flexShrinkableRoom(items, widths, mainSize)
		if shrinkable <= layoutEpsilon {
			break
		}

		step := deficit
		if step > shrinkable {
			step = shrinkable
		}

		e.cutFlexWidths(items, widths, mainSize, step, shrinkable)

		sum = 0
		for _, w := range widths {
			sum += w
		}

		if sum >= contentW-1e-6 && sum <= contentW+1e-6 {
			break
		}

		next := sum - contentW
		if next >= deficit-1e-9 {
			break
		}

		deficit = next
	}
}

func (e *engine) flexShrinkableRoom(items []flexMeas, widths []float64, mainSize float64) float64 {
	var roomSum float64

	for i, it := range items {
		floor := e.flexMinMainSize(it, mainSize)

		room := widths[i] - floor
		if room > 1e-6 && it.shrink > 0 {
			roomSum += room
		}
	}

	return roomSum
}

func (e *engine) cutFlexWidths(items []flexMeas, widths []float64, mainSize, step, shrinkable float64) {
	for idx, it := range items {
		floor := e.flexMinMainSize(it, mainSize)

		room := widths[idx] - floor
		if room <= 1e-6 || it.shrink <= 0 {
			continue
		}

		cut := step * (room / shrinkable)

		widths[idx] -= cut
		if widths[idx] < floor {
			widths[idx] = floor
		}
	}
}

//nolint:wsl // flex placement keeps its measured state updates together
func (e *engine) placeFlexLineMeasured(
	parent *box, style ResolvedStyle, items []flexMeas,
	contentW, contentX, topY, curY, gap, lineCross float64,
) float64 {
	widths := e.flexLineWidths(items, contentW, gap)
	gaps := gap * float64(len(items)-1)
	if gaps < 0 {
		gaps = 0
	}

	sumW := 0.0
	for _, w := range widths {
		sumW += w
	}

	startX, justifyGap := justifyRowStart(style.JustifyContent, contentX, contentW, sumW, gaps, gap, len(items))

	targetCross := lineCross
	if targetCross < 0 {
		targetCross = e.measureFlexCrossMax(items, widths, startX, topY, curY, justifyGap)
	}

	built, rowH := e.buildRowItems(parent, style, items, widths, topY, curY, startX, justifyGap, targetCross)

	alignH := rowH
	if lineCross > alignH {
		alignH = lineCross
	}

	if targetCross > alignH {
		alignH = targetCross
	}

	e.alignRowItems(style, built, topY, curY, alignH)

	if lineCross > rowH {
		return curY + lineCross
	}

	if targetCross > rowH {
		return curY + targetCross
	}

	return curY + rowH
}

// flexLineWidths resolves used main sizes: grow/shrink redistribution over
// the line, then min/max clamping with floor-frozen re-shrink.
func (e *engine) flexLineWidths(items []flexMeas, contentW, gap float64) []float64 {
	widths := make([]float64, len(items))
	for i, it := range items {
		widths[i] = it.baseW
	}

	e.flexDistributeWidths(items, widths, contentW, gap)
	e.flexClampMainWidths(items, widths, contentW, contentW)

	return widths
}

func (e *engine) flexDistributeWidths(items []flexMeas, widths []float64, contentW, gap float64) {
	var fixed, growSum, shrinkSum float64

	for _, it := range items {
		fixed += it.baseW
		growSum += it.grow
		shrinkSum += it.shrink * it.baseW
	}

	gaps := gap * float64(len(items)-1)
	if gaps < 0 {
		gaps = 0
	}

	free := contentW - fixed - gaps
	if free <= 0 || growSum <= 0 {
		if free < 0 && shrinkSum > 0 {
			e.flexShrinkWidths(items, widths, -free, shrinkSum, contentW)
		}

		return
	}

	for i, it := range items {
		if it.grow > 0 {
			widths[i] += free * (it.grow / growSum)
		}
	}
}

func (e *engine) flexShrinkWidths(items []flexMeas, widths []float64, deficit, shrinkSum, contentW float64) {
	for idx, item := range items {
		if item.shrink <= 0 || item.baseW <= 0 {
			continue
		}

		share := (item.shrink * item.baseW) / shrinkSum
		widths[idx] -= deficit * share
		floor := e.flexMinMainSize(item, contentW)

		if widths[idx] < floor {
			widths[idx] = floor
		}
	}
}

// justifyRowStart resolves the main-axis start offset and gap for a row line
// from justify-content, returning (startX, justifyGap).
func justifyRowStart(justify string, contentX, contentW, sumW, gaps, gap float64, count int) (float64, float64) {
	switch justify {
	case fxFlexEnd, fxEnd:
		return contentX + contentW - sumW - gaps, gap
	case fxCenter:
		return contentX + (contentW-sumW-gaps)/two, gap
	case fxBetween, fxAround, fxEvenly:
		return justifyDistributed(justify, contentX, contentW, sumW, gap, count)
	}

	return contentX, gap
}

func justifyDistributed(justify string, contentX, contentW, sumW, gap float64, count int) (float64, float64) {
	rem := contentW - sumW
	if rem < 0 {
		rem = 0
	}

	switch justify {
	case fxBetween:
		if count > 1 && rem > 0 {
			return contentX, rem / float64(count-1)
		}

		return contentX, gap
	case fxAround:
		unit := rem / float64(two*count)

		return contentX + unit, two * unit
	case fxEvenly:
		unit := rem / float64(count+1)

		return contentX + unit, unit
	}

	return contentX, gap
}

// measureFlexCrossMax measures the tallest item to get the cross size for a
// line when the container cross size is indefinite (noEmit measure pass).
func (e *engine) measureFlexCrossMax(
	items []flexMeas, widths []float64, startX, topY, curY, justifyGap float64,
) float64 {
	was := e.noEmit
	e.noEmit = true
	maxH := 0.0
	maxX := startX

	for idx, it := range items {
		cb := e.build(it.n, widths[idx], maxX, topY+curY)
		if cb != nil && cb.height > maxH {
			maxH = cb.height
		}

		maxX += widths[idx]
		if idx < len(items)-1 {
			maxX += justifyGap
		}
	}

	e.noEmit = was

	return maxH
}

// forceFlexItemCrossSize derives a style override with a forced border-box
// cross size. The canonical resolved style stays immutable while build uses
// the flex-computed grow/shrink size.
func (e *engine) forceFlexItemCrossSize(style ResolvedStyle, forceH float64) ResolvedStyle {
	if e.scale <= 0 {
		return style
	}

	if style.BoxSizing == borderBox {
		style.Height = forceH / e.scale
	} else {
		inner := forceH - e.scalePt(style.PaddingTop) - e.scalePt(style.PaddingBottom) -
			e.scalePt(style.BorderTop.Width) - e.scalePt(style.BorderBottom.Width)
		if inner < 0 {
			inner = 0
		}

		style.Height = inner / e.scale
	}

	return style
}

func (e *engine) buildFlexRowItem(
	node *html.Node, style *ResolvedStyle, forceStretch bool,
	targetCross, availW, posX, posY float64,
) *box {
	if !forceStretch {
		return e.build(node, availW, posX, posY)
	}

	// Used cross size = line cross size (border box), matching column
	// main-size forcing so backgrounds fill the flex line (fixture-33).
	override := e.forceFlexItemCrossSize(*style, targetCross)

	return e.buildWithStyle(node, &override, availW, posX, posY)
}

func (e *engine) buildRowItems(
	parent *box, style ResolvedStyle, items []flexMeas, widths []float64,
	topY, curY, startX, justifyGap, targetCross float64,
) ([]flexPlacedItem, float64) {
	built := make([]flexPlacedItem, 0, len(items))
	rowH := 0.0
	leftX := startX

	for idx, item := range items {
		cstate := e.styles[item.n]

		forceStretch := flexItemCrossStretch(style, *cstate) && targetCross > 0
		cblock := e.buildFlexRowItem(item.n, cstate, forceStretch, targetCross, widths[idx], leftX, topY+curY)

		if cblock == nil {
			built = append(built, flexPlacedItem{n: item.n}) //nolint:exhaustruct // intentional zero fields

			leftX += widths[idx]
			if idx < len(items)-1 {
				leftX += justifyGap
			}

			continue
		}

		dx := leftX - cblock.x
		dy := (topY + curY) - cblock.y
		e.shiftBoxOps(cblock, dx, dy)
		cblock.x += dx
		cblock.y += dy

		if cblock.height > rowH {
			rowH = cblock.height
		}

		built = append(built, flexPlacedItem{box: cblock, h: cblock.height, n: item.n})

		if parent != nil {
			parent.children = append(parent.children, cblock)
		}

		leftX += widths[idx]
		if idx < len(items)-1 {
			leftX += justifyGap
		}
	}

	return built, rowH
}

// alignRowItems applies align-items/align-self offsets on the cross axis
// within a line (stretch sizing happened during build).
func (e *engine) alignRowItems(style ResolvedStyle, built []flexPlacedItem, topY, cyOffset, alignH float64) {
	for _, page := range built {
		if page.box == nil {
			continue
		}

		align := style.AlignItems
		if cs, ok := e.styles[page.n]; ok && cs.AlignSelf != "" && cs.AlignSelf != fxAuto {
			align = cs.AlignSelf
		}

		var deltaY float64

		switch align {
		case fxFlexEnd, fxEnd:
			deltaY = (topY + cyOffset + alignH) - (page.box.y + page.box.height)
		case fxCenter:
			deltaY = (topY + cyOffset + (alignH-page.box.height)/two) - page.box.y
		}

		if deltaY != 0 {
			e.shiftBoxOps(page.box, 0, deltaY)
			page.box.y += deltaY
		}
	}
}

// flexItemCrossStretch reports whether a flex item should stretch on the cross
// axis (align-items/self stretch, and no definite cross size).
func flexItemCrossStretch(cstate, cstate2 ResolvedStyle) bool {
	align := cstate.AlignItems
	if align == "" {
		align = fxStretch
	}

	if cstate2.AlignSelf != "" && cstate2.AlignSelf != fxAuto {
		align = cstate2.AlignSelf
	}

	switch align {
	case fxFlexStart, fxStart, fxFlexEnd, fxEnd, fxCenter:
		return false
	}
	// Definite height/% means the used cross size is already specified.
	if cstate2.Height >= 0 || cstate2.HeightPercent >= 0 {
		return false
	}

	return true
}

// flexItemBaseHeight resolves the flex base size on the column main axis.
// mainSize is the flex container content-box height from resolveContentHeight.
// (−1 when height is auto / indefinite). Percentage flex-basis against an
// indefinite main size is treated as auto (content-based) — CSS Flexbox L1
// §9.2 cyclic %-sizing subset; do not resolve % as 0 silently.
func (e *engine) flexItemBaseHeight(node *html.Node, style ResolvedStyle, contentW, mainSize float64) float64 {
	padV := e.scalePt(style.PaddingTop) + e.scalePt(style.PaddingBottom) +
		e.scalePt(style.BorderTop.Width) + e.scalePt(style.BorderBottom.Width)

	if h, ok := e.flexSpecifiedBaseHeight(style, mainSize, padV); ok {
		return h
	}

	start := len(e.ops)
	height := e.layoutCell(node, style, contentW)
	e.ops = e.ops[:start]

	if height <= 0 {
		height = padV + e.scalePt(style.FontSize)*defaultLineHeightRatio
	}

	return height
}

func (e *engine) flexSpecifiedBaseHeight(style ResolvedStyle, mainSize, padV float64) (float64, bool) {
	switch {
	case style.FlexBasisPercent >= 0 && mainSize >= 0:
		return e.flexBoxSized(style, mainSize*style.FlexBasisPercent/cssPercent, padV), true
	case style.FlexBasis >= 0:
		return e.flexBoxSized(style, e.scalePt(style.FlexBasis), padV), true
	case style.HeightPercent >= 0 && mainSize >= 0:
		return e.flexBoxSized(style, mainSize*style.HeightPercent/cssPercent, padV), true
	case style.Height >= 0:
		return e.flexBoxSized(style, e.scalePt(style.Height), padV), true
	}

	return 0, false
}

// flexMinCrossMainSize is the column-axis content-based min-height floor
// (Flexbox §4.5 lite). mainSize is the definite flex container content height.
func (e *engine) flexMinCrossMainSize(node *html.Node, baseH, mainSize float64) float64 {
	cstate := e.styles[node]
	floor := 0.0

	if cstate.MinHeightPercent >= 0 && mainSize >= 0 {
		floor = mainSize * cstate.MinHeightPercent / cssPercent
	} else if cstate.MinHeight > 0 {
		floor = e.scalePt(cstate.MinHeight)
	}

	if overflowCreatesStickyScrollport(cstate.Overflow) {
		return floor
	}

	padV := e.scalePt(cstate.PaddingTop) + e.scalePt(cstate.PaddingBottom) +
		e.scalePt(cstate.BorderTop.Width) + e.scalePt(cstate.BorderBottom.Width)
	start := len(e.ops)
	contentSug := e.layoutCell(node, *cstate, infiniteMeasure)
	e.ops = e.ops[:start]

	if contentSug < padV {
		contentSug = padV + e.scalePt(cstate.FontSize)*defaultLineHeightRatio
	}

	specSug := e.flexSpecifiedHeightSuggestion(*cstate, baseH, mainSize, padV)

	autoMin := contentSug
	if specSug >= 0 && specSug < autoMin {
		autoMin = specSug
	}

	if autoMin > floor {
		floor = autoMin
	}

	return floor
}

func (e *engine) flexSpecifiedHeightSuggestion(style ResolvedStyle, baseH, mainSize, padV float64) float64 {
	switch {
	case style.HeightPercent >= 0 && mainSize >= 0:
		return e.flexBoxSized(style, mainSize*style.HeightPercent/cssPercent, padV)
	case style.Height >= 0:
		return e.flexBoxSized(style, e.scalePt(style.Height), padV)
	case baseH > 0:
		return baseH
	}

	return -1
}

func (e *engine) flowFlexColumn(
	parent *box, kids []*html.Node, style ResolvedStyle,
	contentW, contentX, topY, curY, gap float64,
) float64 {
	contentH := resolveContentHeight(style, e)
	items := e.flexColumnItems(kids, contentW, contentH)

	if len(items) == 0 {
		return curY
	}

	if style.FlexDirection == fxColRev {
		for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
			items[i], items[j] = items[j], items[i]
		}
	}

	heights := e.flexColumnHeights(items, contentH, gap)

	gaps := gap * float64(len(items)-1)
	if gaps < 0 {
		gaps = 0
	}

	sumH := 0.0
	for _, h := range heights {
		sumH += h
	}

	startY, justifyGap := justifyColumnStart(style.JustifyContent, contentH, curY, sumH+gaps, sumH, gap, len(items))

	endY := e.buildColumnItems(parent, style, items, heights, contentW, contentX, topY, curY, startY, justifyGap)

	if contentH >= 0 && endY < curY+contentH {
		return curY + contentH
	}

	return endY
}

func (e *engine) flexColumnItems(kids []*html.Node, contentW, contentH float64) []flexColMeas {
	items := make([]flexColMeas, 0, len(kids))

	for _, kid := range kids {
		cstate := e.styles[kid]

		grow := cstate.FlexGrow
		if grow < 0 {
			grow = 0
		}

		shrink := cstate.FlexShrink
		if shrink < 0 {
			shrink = 1
		}

		items = append(items, flexColMeas{
			n: kid, baseH: e.flexItemBaseHeight(kid, *cstate, contentW, contentH),
			grow: grow, shrink: shrink,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		return e.styles[items[i].n].FlexOrder < e.styles[items[j].n].FlexOrder
	})

	return items
}

// flexColumnHeights resolves used main sizes: grow/shrink redistribution over
// the definite container height, then min/max-height clamping.
func (e *engine) flexColumnHeights(items []flexColMeas, contentH, gap float64) []float64 {
	heights := make([]float64, len(items))

	var fixed, growSum, shrinkSum float64

	for i, it := range items {
		heights[i] = it.baseH
		fixed += it.baseH
		growSum += it.grow
		shrinkSum += it.shrink * it.baseH
	}

	if contentH < 0 {
		return heights
	}

	gaps := gap * float64(len(items)-1)
	if gaps < 0 {
		gaps = 0
	}

	free := contentH - fixed - gaps
	if free > 0 && growSum > 0 {
		e.flexGrowHeights(items, heights, free, growSum)
	} else if free < 0 && shrinkSum > 0 {
		e.flexShrinkHeights(items, heights, -free, shrinkSum, contentH)
	}
	// Re-apply min/max-height after grow/shrink (percentage re-resolve).
	e.flexClampColumnHeights(items, heights, contentH)

	return heights
}

func (e *engine) flexGrowHeights(items []flexColMeas, heights []float64, free, growSum float64) {
	for i, it := range items {
		if it.grow > 0 {
			heights[i] += free * (it.grow / growSum)
		}
	}
}

func (e *engine) flexShrinkHeights(items []flexColMeas, heights []float64, deficit, shrinkSum, contentH float64) {
	for idx, item := range items {
		if item.shrink <= 0 || item.baseH <= 0 {
			continue
		}

		share := (item.shrink * item.baseH) / shrinkSum
		heights[idx] -= deficit * share
		floor := e.flexMinCrossMainSize(item.n, item.baseH, contentH)

		if heights[idx] < floor {
			heights[idx] = floor
		}
	}
}

func (e *engine) flexClampColumnHeights(items []flexColMeas, heights []float64, contentH float64) {
	for idx, it := range items {
		cstate := e.styles[it.n]

		floor := e.flexMinCrossMainSize(it.n, it.baseH, contentH)
		if heights[idx] < floor {
			heights[idx] = floor
		}

		if cstate.MaxHeight >= 0 {
			mx := e.scalePt(cstate.MaxHeight)
			if heights[idx] > mx {
				heights[idx] = mx
			}
		}
	}
}

// justifyColumnStart resolves the main-axis start offset and gap for a column
// from justify-content, returning (startY, justifyGap).
func justifyColumnStart(justify string, contentH, curY, totalH, sumH, gap float64, count int) (float64, float64) {
	if contentH >= 0 {
		switch justify {
		case fxFlexEnd, fxEnd:
			return curY + contentH - totalH, gap
		case fxCenter:
			return curY + (contentH-totalH)/two, gap
		case fxBetween, fxAround, fxEvenly:
			return justifyColumnDistributed(justify, curY, contentH, sumH, gap, count)
		}
	}

	return curY, gap
}

func justifyColumnDistributed(justify string, curY, contentH, sumH, gap float64, count int) (float64, float64) {
	rem := contentH - sumH
	if rem < 0 {
		rem = 0
	}

	switch justify {
	case fxBetween:
		if count > 1 && rem > 0 {
			return curY, rem / float64(count-1)
		}

		return curY, gap
	case fxAround:
		unit := rem / float64(two*count)

		return curY + unit, two * unit
	case fxEvenly:
		unit := rem / float64(count+1)

		return curY + unit, unit
	}

	return curY, gap
}

func (e *engine) buildColumnItems(
	parent *box, style ResolvedStyle, items []flexColMeas, heights []float64,
	contentW, contentX, topY, curY, startY, justifyGap float64,
) float64 {
	leftY := startY
	endY := curY

	for idx, item := range items {
		cstate := e.styles[item.n]
		// Force border-box height so grow/shrink targets stick through build.
		override := e.forceFlexItemCrossSize(*cstate, heights[idx])
		cblock := e.buildWithStyle(item.n, &override, contentW, contentX, topY+leftY)

		if cblock == nil {
			leftY += heights[idx]
			if idx < len(items)-1 {
				leftY += justifyGap
			}

			continue
		}

		dx := contentX - cblock.x
		dy := (topY + leftY) - cblock.y
		e.shiftBoxOps(cblock, dx, dy)
		cblock.x += dx
		cblock.y += dy

		e.alignColumnItem(cblock, style, *cstate, contentX, contentW)

		if parent != nil {
			parent.children = append(parent.children, cblock)
		}

		leftY += heights[idx]
		endY = leftY

		if idx < len(items)-1 {
			leftY += justifyGap
			endY = leftY
		}
	}

	return endY
}

func (e *engine) alignColumnItem(cblock *box, st ResolvedStyle, cs ResolvedStyle, contentX, contentW float64) {
	align := st.AlignItems
	if cs.AlignSelf != "" && cs.AlignSelf != fxAuto {
		align = cs.AlignSelf
	}

	switch align {
	case fxCenter:
		adx := contentX + (contentW-cblock.w)/two - cblock.x
		if adx != 0 {
			e.shiftBoxOps(cblock, adx, 0)
			cblock.x += adx
		}
	case fxFlexEnd, fxEnd:
		adx := contentX + contentW - cblock.w - cblock.x
		if adx != 0 {
			e.shiftBoxOps(cblock, adx, 0)
			cblock.x += adx
		}
	}
}

// applyRelativeOffset shifts a position:relative box and its ops by top/left
// (right/bottom when the corresponding auto flags are set). position:sticky
// uses tagSticky + applyStickyPrint instead (print scrollport clamp).
func (e *engine) applyRelativeOffset(boxNode *box) {
	if boxNode == nil || boxNode.style.Position != "relative" {
		return
	}

	sty := boxNode.style
	deltaX, deltaY := 0.0, 0.0

	if !sty.LeftAuto {
		deltaX = e.scalePt(sty.Left)
	} else if !sty.RightAuto {
		deltaX = -e.scalePt(sty.Right)
	}

	if !sty.TopAuto {
		deltaY = e.scalePt(sty.Top)
	} else if !sty.BottomAuto {
		deltaY = -e.scalePt(sty.Bottom)
	}

	if deltaX == 0 && deltaY == 0 {
		return
	}

	boxNode.x += deltaX
	boxNode.y += deltaY
	e.shiftBoxOps(boxNode, deltaX, deltaY)
}
