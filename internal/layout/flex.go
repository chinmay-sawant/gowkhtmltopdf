package layout

import (
	"slices"
	"sort"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
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
	fxRowRev    = "row-reverse"
	fxStart     = "start"
	fxStretch   = "stretch"
	fxWrap      = "wrap"
	fxWrapRev   = "wrap-reverse"
)

type flexMeas struct {
	n             *html.Node
	baseW         float64
	hypotheticalW float64
	grow          float64
	shrink        float64
	order         int
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
//
//nolint:cyclop,gocritic,wsl // dispatch also installs the containing-block scope
func (e *engine) buildFlex(node *html.Node, sty ResolvedStyle, availW, x, posY float64) *box {
	ml := e.scalePt(sty.MarginLeft)
	boxNode := &box{ //nolint:exhaustruct // intentional zero fields
		node: node, style: e.stylePtr(node), kind: boxKindBlock, x: x + ml, y: posY,
	}
	boxNode.w = resolveUsedWidth(sty, availW, e)
	if isIntrinsicWidth(sty.Width) {
		boxNode.w = e.flexIntrinsicWidth(node, sty, sty.Width == widthMinContent)
	}

	if sty.IsWebkitBox && sty.Width < 0 && sty.WidthPercent < 0 {
		if intr := e.measureFlexItemMaxContent(node, sty); intr > 0 && intr < boxNode.w {
			boxNode.w = intr
		}
	}

	contentX, contentW := e.contentBox(boxNode.x, boxNode.w, &sty)
	sty = withAspectRatioHeight(sty, contentW, e)

	contentStart := len(e.ops)
	curY := e.scalePt(sty.PaddingTop) + e.scalePt(sty.BorderTop.Width)

	kids := e.flexChildren(node, sty)

	rowGap, colGap := e.styleGaps(sty)
	previousCB, parentCB := e.setFlowCB(sty)
	defer func() { e.flowCBHeight = previousCB }()

	dir := sty.FlexDirection
	if dir == "" {
		dir = fxRow
	}

	if isVerticalWritingMode(sty.WritingMode) && (dir == fxCol || dir == fxColRev) {
		curY = e.flowFlexVerticalColumn(boxNode, kids, sty, contentW, contentX, posY, curY, rowGap)
	} else if dir == fxCol || dir == fxColRev {
		curY = e.flowFlexColumn(boxNode, kids, sty, contentW, contentX, posY, curY, rowGap, colGap)
	} else if isVerticalWritingMode(sty.WritingMode) {
		curY = e.flowFlexVerticalRow(boxNode, kids, sty, contentW, contentX, posY, curY, colGap)
	} else {
		curY = e.flowFlexRow(boxNode, kids, sty, contentW, contentX, posY, curY, colGap, rowGap)
	}

	// The shared resolver owns bottom padding, bottom border, and the
	// height/min-height/max-height constraints for auto-height boxes.
	resolvedHeightStyle := withAspectRatioHeight(sty, contentW, e)
	contentBottom := e.borderBoxBottom(resolvedHeightStyle, curY)
	if usedHeight, definite := resolveUsedHeight(&resolvedHeightStyle, parentCB, e); definite {
		// A definite flex-container height is a used size, not a minimum. Its
		// flex items may overflow the container when shrink is disabled.
		vChrome := resolvedHeightStyle.verticalChrome(e)
		curY = usedHeight
		curY = e.clampBlockMaxHeight(&resolvedHeightStyle, curY, parentCB, vChrome)
		curY = e.clampBlockMinHeight(&resolvedHeightStyle, curY, parentCB, vChrome)
	} else {
		curY = e.applyHeightConstraintsWithCB(&resolvedHeightStyle, contentBottom, parentCB)
	}

	boxNode.height = curY
	e.prependChrome(contentStart, boxNode, sty, boxNode.x, posY, boxNode.w, boxNode.height)

	return boxNode
}

// flexChildren returns the element flex items plus anonymous flex items for
// direct text runs. CSS turns non-whitespace text directly under a flex
// container into anonymous block-level flex items; dropping those nodes makes
// prose after an inline marker disappear from the display list.
func (e *engine) flexChildren(node *html.Node, parentStyle ResolvedStyle) []*html.Node {
	kids := make([]*html.Node, 0, len(node.Children))

	for idx := 0; idx < len(node.Children); idx++ {
		child := node.Children[idx]
		if child.Type == html.ElementNode {
			if cs := e.stylePtr(child); cs.Display != cssDisplayNone {
				kids = append(kids, child)
			}

			continue
		}

		if child.Type != html.TextNode || strings.TrimSpace(child.Text) == "" {
			continue
		}

		textNodes := []*html.Node{child}

		for idx+1 < len(node.Children) && node.Children[idx+1].Type == html.TextNode {
			idx++
			if strings.TrimSpace(node.Children[idx].Text) != "" {
				textNodes = append(textNodes, node.Children[idx])
			}
		}

		anonymous := &html.Node{ //nolint:exhaustruct // synthetic anonymous flex item
			Type: html.ElementNode, Name: "span", Parent: node, Children: textNodes,
		}
		anonymousStyle := anonymousFlexItemStyle(parentStyle)
		e.setSyntheticStyle(anonymous, &anonymousStyle)

		kids = append(kids, anonymous)
	}

	return kids
}

// anonymousFlexItemStyle preserves inherited text properties while removing
// the flex container's own box and layout properties from the anonymous item.
//
//nolint:funlen // resets the complete anonymous box model
func anonymousFlexItemStyle(parent ResolvedStyle) ResolvedStyle {
	style := parent
	style.Display = displayBlock
	style.Position = positionStatic
	style.Float = cssDisplayNone
	style.Clear = cssDisplayNone
	style.FlexDirection = fxRow
	style.FlexWrap = cssWhiteSpaceNowrap
	style.FlexGrow = 0
	style.FlexShrink = 1
	style.FlexBasis = -1
	style.FlexBasisPercent = -1
	style.FlexOrder = 0
	style.Width = -1
	style.WidthPercent = -1
	style.Height = -1
	style.HeightPercent = -1
	style.MinWidth = 0
	style.MinWidthPercent = -1
	style.MinWidthSet = false
	style.MaxWidth = -1
	style.MaxWidthPercent = -1
	style.MinHeight = 0
	style.MinHeightPercent = -1
	style.MaxHeight = -1
	style.MarginTop = 0
	style.MarginRight = 0
	style.MarginBottom = 0
	style.MarginLeft = 0
	style.MarginTopAuto = false
	style.MarginRightAuto = false
	style.MarginBottomAuto = false
	style.MarginLeftAuto = false
	style.PaddingTop = 0
	style.PaddingRight = 0
	style.PaddingBottom = 0
	style.PaddingLeft = 0
	style.BorderTop = border{}    //nolint:exhaustruct // zero border
	style.BorderRight = border{}  //nolint:exhaustruct // zero border
	style.BorderBottom = border{} //nolint:exhaustruct // zero border
	style.BorderLeft = border{}   //nolint:exhaustruct // zero border
	style.BorderRadius = 0
	style.BorderRadiusPercent = 0
	style.BorderRadiusTopLeft = 0
	style.BorderRadiusTopRight = 0
	style.BorderRadiusBottomRight = 0
	style.BorderRadiusBottomLeft = 0
	style.BGColor = [4]float64{}
	style.ListStyleType = cssDisplayNone
	style.PageBreakBefore = ""
	style.PageBreakAfter = ""
	style.PageBreakInside = ""
	style.HasTransform = false

	return style
}

//nolint:funlen // row flow combines line formation, sizing, and placement
func (e *engine) flowFlexRow(
	parent *box, kids []*html.Node, style ResolvedStyle,
	contentW, contentX, topY, curY, colGap, rowGap float64,
) float64 {
	if len(kids) == 0 {
		return curY
	}

	wrap := style.FlexWrap == fxWrap || style.FlexWrap == fxWrapRev
	reverse := flexRowPhysicalReverse(style)
	contentH := resolveContentHeight(style, e)
	items := e.flexRowItems(kids, contentW, contentH)
	lines := e.flexWrapLines(items, wrap, reverse, style.FlexWrap == fxWrapRev, colGap, contentW)

	lineCross := -1.0
	if !wrap {
		lineCross = contentH
	}

	stretchCross := e.alignContentStretchLineCross(
		style, lines, contentW, colGap, rowGap, contentX, topY, curY,
	)

	placed := make([]flexLinePlace, 0, len(lines))

	poll := newCtxPoll(e.ctx)
	for lidx, line := range lines {
		if poll.poll() {
			e.err = poll.err

			return curY
		}

		startChild := 0
		if parent != nil {
			startChild = len(parent.children)
		}

		yStart := curY
		cross := lineCross

		if stretchCross != nil {
			cross = stretchCross[lidx]
		}

		curY = e.placeFlexLineMeasured(
			parent,
			style,
			line,
			contentW,
			contentX,
			topY,
			curY,
			colGap,
			cross,
			style.FlexWrap == fxWrapRev,
			reverse,
		)

		endChild := startChild
		if parent != nil {
			endChild = len(parent.children)
		}

		placed = append(placed, flexLinePlace{startChild: startChild, endChild: endChild, y0: yStart, h: curY - yStart})

		if lidx < len(lines)-1 {
			curY += rowGap
		}
	}

	return e.applyAlignContentRow(parent, style.AlignContent, placed, rowGap, contentH, curY)
}

func (e *engine) flexRowItems(kids []*html.Node, contentW float64, crossSize ...float64) []flexMeas {
	items := make([]flexMeas, 0, len(kids))

	for _, kid := range kids {
		cstate := e.stylePtr(kid)

		grow := cstate.FlexGrow
		if grow < 0 {
			grow = 0
		}

		shrink := cstate.FlexShrink
		if shrink < 0 {
			shrink = 1
		}

		baseW := e.flexItemBaseWidth(kid, *cstate, contentW)
		if len(crossSize) > 0 {
			if aspectW := e.flexAspectContentWidth(kid, *cstate, crossSize[0]); aspectW > baseW {
				baseW = aspectW
			}
		}

		item := flexMeas{
			n: kid, baseW: baseW,
			hypotheticalW: 0, grow: grow, shrink: shrink, order: cstate.FlexOrder,
		}
		item.hypotheticalW = e.flexHypotheticalMainSize(item, contentW)
		items = append(items, item)
	}

	sort.SliceStable(items, func(i, j int) bool { return items[i].order < items[j].order })

	return items
}

//nolint:cyclop,goconst,wsl // recursive aspect-ratio contribution walks the small flex tree
func (e *engine) flexAspectContentWidth(node *html.Node, style ResolvedStyle, crossSize float64) float64 {
	if node != nil {
		switch node.Name {
		case cssTagImg, cssTagSVG, "video", "canvas":
			return 0
		}
	}

	width := 0.0
	if style.AspectRatio > 0 {
		height := -1.0
		if style.Height >= 0 {
			height = e.scalePt(style.Height)
		} else if style.HeightPercent >= 0 && crossSize >= 0 {
			height = crossSize * style.HeightPercent / oneHundred
		}
		if height > 0 {
			width = height * style.AspectRatio
		}
	}

	for _, child := range node.Children {
		if child.Type != html.ElementNode {
			continue
		}
		childStyle := e.stylePtr(child)
		if childStyle.Display == cssDisplayNone {
			continue
		}
		if childWidth := e.flexAspectContentWidth(child, *childStyle, crossSize); childWidth > width {
			width = childWidth
		}
	}
	if width <= 0 {
		return 0
	}

	if style.BoxSizing != borderBox {
		width += e.scalePt(style.PaddingLeft) + e.scalePt(style.PaddingRight) +
			e.scalePt(style.BorderLeft.Width) + e.scalePt(style.BorderRight.Width)
	}

	return width
}

// flexWrapLines packs measured items into flex lines, honoring wrap mode and
// reverse direction (item order within each line, then line order). The trial
// line is checked after flex grow/shrink and min/max clamping, because a
// content-based minimum can make a line overflow even when raw flex bases fit.
func (e *engine) flexWrapLines(
	items []flexMeas, wrap, reverse, wrapReverse bool, colGap, contentW float64,
) [][]flexMeas {
	if !wrap {
		if reverse {
			reverseFlexMeas(items)
		}

		return [][]flexMeas{items}
	}

	var lines [][]flexMeas

	line := make([]flexMeas, 0, len(items))
	used := 0.0

	poll := newCtxPoll(e.ctx)
	for _, item := range items {
		if poll.poll() {
			e.err = poll.err

			return nil
		}

		need := item.hypotheticalW
		if len(line) > 0 {
			need += colGap
		}

		trial := append(append([]flexMeas(nil), line...), item)
		if len(line) > 0 && !e.flexLineFits(trial, contentW, colGap) {
			lines = append(lines, finalizeFlexLine(line, reverse))
			line = nil
			used = 0
			need = item.hypotheticalW
		}

		line = append(line, item)
		used += need
	}

	if len(line) > 0 {
		lines = append(lines, finalizeFlexLine(line, reverse))
	}

	if wrapReverse {
		slices.Reverse(lines)
	}

	return lines
}

func (e *engine) flexLineFits(items []flexMeas, contentW, gap float64) bool {
	widths := e.flexLineWidths(items, contentW, gap)
	total := gap * float64(len(widths)-1)

	for idx, width := range widths {
		total += width + flexRowNormalMargins(e, items[idx])
	}

	if total > contentW+layoutEpsilon {
		return false
	}

	// A wrapped line may be a few points over its raw hypothetical sizes when
	// flex-shrink resolves font-metric rounding. Do not use a large shrink to
	// defeat wrapping: that collapses later items into unreadable slivers and
	// also pulls full-width captions into the row. A candidate is therefore
	// accepted only when every positive-base item remains within ten percent of
	// its hypothetical size.
	const maxLineShrink = 0.10

	for idx, item := range items {
		if item.baseW <= layoutEpsilon || item.shrink <= 0 {
			continue
		}

		if widths[idx] < item.baseW*(1-maxLineShrink) {
			return false
		}
	}

	return true
}

// flexHypotheticalMainSize is the size used when deciding whether an item
// belongs on the current flex line. Wrapping is based on the hypothetical
// main size, not the raw flex base size: min-width:auto contributes the
// content-based minimum before line packing (CSS Flexbox §9.2). Without this
// clamp, cards with long inline tokens incorrectly remain on a single line;
// fixture-56's D02 source cards are the concrete regression.
func (e *engine) flexHypotheticalMainSize(item flexMeas, mainSize float64) float64 {
	size := item.baseW
	minSize := e.flexMinMainSize(item, mainSize)

	if size < minSize {
		size = minSize
	}

	if style := e.stylePtr(item.n); style.MaxWidth >= 0 {
		maxSize := e.scalePt(style.MaxWidth)
		if size > maxSize {
			size = maxSize
		}
	}

	return size
}

func finalizeFlexLine(line []flexMeas, reverse bool) []flexMeas {
	if reverse {
		reverseFlexMeas(line)
	}

	return line
}

func reverseFlexMeas(line []flexMeas) {
	slices.Reverse(line)
}

// applyAlignContentRow distributes free cross space between wrapped flex
// lines when the container height is definite and wrapping produced multiple
// lines. Height:auto → pack at start (no-op).
// Single-line containers with a definite height and align-content:center/end
// are also centered/end-aligned to match the audit fixture's expectation:
// -webkit-align-content:center with height:56px and a single wrapped line
// should appear vertically centered, not pinned to the top.
func (e *engine) applyAlignContentRow(
	parent *box, alignContent string, placed []flexLinePlace,
	rowGap, contentH, curY float64,
) float64 {
	if contentH < 0 || len(placed) == 0 {
		return curY
	}

	if len(placed) == 1 && !alignContentSingleLineOK(alignContent) {
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

	e.shiftAlignContentLines(parent, placed, offsets)

	last := placed[len(placed)-1]

	return last.y0 + offsets[len(offsets)-1] + last.h
}

// alignContentSingleLineOK reports whether a single-line container with a
// definite height participates in align-content distribution.
func alignContentSingleLineOK(alignContent string) bool {
	switch alignContent {
	case fxCenter, fxFlexEnd, fxEnd, fxStart, flexStartKeyword:
		return true
	default:
		return false
	}
}

// shiftAlignContentLines shifts each placed line's children by its
// align-content offset.
func (e *engine) shiftAlignContentLines(parent *box, placed []flexLinePlace, offsets []float64) {
	if parent == nil {
		return
	}

	for i, line := range placed {
		deltaY := offsets[i]
		if deltaY == 0 {
			continue
		}

		e.shiftPlacedChildren(parent, line.startChild, line.endChild, deltaY)
	}
}

// alignContentStretchLineCross returns per-line cross sizes when
// align-content:stretch (the initial value) should grow wrapped lines into
// leftover definite cross space. nil means pack at start (height:auto, one
// line, or a non-stretch alignment).
func (e *engine) alignContentStretchLineCross(
	style ResolvedStyle, lines [][]flexMeas, contentW, colGap, rowGap, contentX, topY, curY float64,
) []float64 {
	contentH := resolveContentHeight(style, e)
	if contentH < 0 || len(lines) <= 1 {
		return nil
	}

	align := style.AlignContent
	if align != "" && align != fxStretch {
		return nil
	}

	heights := make([]float64, len(lines))
	sum := 0.0

	for i, line := range lines {
		heights[i] = e.flexLineNaturalCross(style, line, contentW, colGap, contentX, topY, curY)
		sum += heights[i]
	}

	free := contentH - sum - rowGap*float64(len(lines)-1)
	if free <= layoutEpsilon {
		return nil
	}

	extra := free / float64(len(lines))
	for i := range heights {
		heights[i] += extra
	}

	return heights
}

func (e *engine) flexLineNaturalCross(
	style ResolvedStyle, items []flexMeas, contentW, gap, contentX, topY, curY float64,
) float64 {
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

	return e.measureFlexCrossMax(items, widths, startX, topY, curY, justifyGap)
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

	if isInputCheckbox(node) {
		return defaultCheckboxSize(e, style) + e.scalePt(style.MarginLeft) + e.scalePt(style.MarginRight)
	}

	capW := mainSize
	if capW < 0 {
		capW = indefiniteContentCap
	}

	// measureFlexItemMaxContent already returns the border-box contribution,
	// including the item's horizontal padding and borders. Adding pad here a
	// second time widens every auto-sized flex item and can force a final pill
	// onto a new row (fixture-56 d01-flow).
	intr := e.measureFlexItemMaxContent(node, style) +
		e.scalePt(style.MarginLeft) + e.scalePt(style.MarginRight)
	if intr <= 0 {
		intr = pad
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
	if style.Display == displayFlex || style.Display == displayInlineFlex {
		return e.flexIntrinsicWidth(node, style, false)
	}

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

		childStyle := e.stylePtr(child)
		if childStyle.Display == cssDisplayNone {
			continue
		}

		childW := e.measureFlexItemMaxContent(child, *childStyle)
		childW += e.scalePt(childStyle.MarginLeft) + e.scalePt(childStyle.MarginRight)
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

// flexIntrinsicWidth returns the border-box width contribution of a flex
// container. Row containers sum their item contributions; column containers
// use the largest contribution because their items share the cross axis.
//
//nolint:cyclop,wsl // intrinsic flex contributions keep the row and column rules together
func (e *engine) flexIntrinsicWidth(node *html.Node, style ResolvedStyle, _ bool) float64 {
	rowFlex := style.FlexDirection != fxCol && style.FlexDirection != fxColRev
	chrome := e.scalePt(style.PaddingLeft) + e.scalePt(style.PaddingRight) +
		e.scalePt(style.BorderLeft.Width) + e.scalePt(style.BorderRight.Width)
	contentW := 0.0
	if !rowFlex {
		_, maxW := e.measureCellMinMax(node, style)
		contentW = maxW - chrome
		if contentW < 0 {
			contentW = 0
		}
	}

	childW := 0.0
	childCount := 0
	for _, child := range node.Children {
		if child.Type != html.ElementNode {
			continue
		}

		childStyle := e.stylePtr(child)
		if childStyle.Display == cssDisplayNone {
			continue
		}

		var contribution float64
		if childStyle.Width >= 0 && childStyle.WidthPercent < 0 {
			pad := e.scalePt(childStyle.PaddingLeft) + e.scalePt(childStyle.PaddingRight) +
				e.scalePt(childStyle.BorderLeft.Width) + e.scalePt(childStyle.BorderRight.Width)
			contribution = e.flexBoxSized(*childStyle, e.scalePt(childStyle.Width), pad)
		} else {
			contribution = e.measureFlexItemMaxContent(child, *childStyle)
		}
		contribution += e.scalePt(childStyle.MarginLeft) + e.scalePt(childStyle.MarginRight)
		if rowFlex {
			childW += contribution
			childCount++
		} else if contribution > childW {
			childW = contribution
		}
	}

	if rowFlex {
		_, columnGap := e.styleGaps(style)
		if childCount > 1 {
			childW += columnGap * float64(childCount-1)
		}
	}
	if childW > contentW {
		contentW = childW
	}

	return contentW
}

// flexSpecifiedBaseWidth resolves a definite flex base size (flex-basis then
// width). Cyclic percentage basis against an indefinite main size falls
// through to the next specified size, then content.
func (e *engine) flexSpecifiedBaseWidth(style ResolvedStyle, mainSize, pad float64) (float64, bool) {
	switch {
	case style.FlexBasisPercent >= 0 && mainSize >= 0:
		return e.flexBoxSized(style, mainSize*style.FlexBasisPercent/oneHundred, pad), true
	case style.FlexBasis >= 0:
		return e.flexBoxSized(style, e.scalePt(style.FlexBasis), pad), true
	case style.WidthPercent >= 0 && mainSize >= 0:
		return e.flexBoxSized(style, mainSize*style.WidthPercent/oneHundred, pad), true
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
//
//nolint:cyclop // CSS min-size decision tree
func (e *engine) flexMinMainSize(item flexMeas, mainSize float64) float64 {
	cstate := e.stylePtr(item.n)
	floor := 0.0

	if cstate.MinWidthSet {
		if cstate.MinWidthPercent >= 0 && mainSize >= 0 {
			return mainSize * cstate.MinWidthPercent / oneHundred
		}

		return e.scalePt(cstate.MinWidth)
	}

	if cstate.MinWidthPercent >= 0 && mainSize >= 0 {
		floor = mainSize * cstate.MinWidthPercent / oneHundred
	} else if cstate.MinWidth > 0 {
		floor = e.scalePt(cstate.MinWidth)
	}
	// Automatic minimum (min-width:auto): min-content size suggestion. The
	// returned width already includes the item's horizontal chrome.
	contentSug, _ := e.measureCellMinMax(item.n, *cstate)
	pad := e.scalePt(cstate.PaddingLeft) + e.scalePt(cstate.PaddingRight) +
		e.scalePt(cstate.BorderLeft.Width) + e.scalePt(cstate.BorderRight.Width)
	// Specified size suggestion when width/% is definite against mainSize.
	specSug := e.flexSpecifiedWidthSuggestion(*cstate, mainSize, pad)

	autoMin := contentSug
	if specSug >= 0 && specSug < autoMin {
		autoMin = specSug
	}
	// Overflow non-visible → automatic min size is 0 (CSS Flexbox §4.5).
	if flexAutoMinSizeIsZero(cstate.Overflow) {
		autoMin = 0
	}

	if autoMin > floor {
		floor = autoMin
	}

	return floor
}

func (e *engine) flexSpecifiedWidthSuggestion(style ResolvedStyle, mainSize, pad float64) float64 {
	switch {
	case style.WidthPercent >= 0 && mainSize >= 0:
		return e.flexBoxSized(style, mainSize*style.WidthPercent/oneHundred, pad)
	case style.Width >= 0:
		return e.flexBoxSized(style, e.scalePt(style.Width), pad)
	}

	return -1
}

// flexClampMainWidths applies min/max after grow/shrink, then re-resolves so
// that percentage-driven floors and content mins that raised used sizes are
// honored without leaving the line sum inconsistent when space remains.
//
//nolint:cyclop,wsl // clamp and redistribution are one ordered flex phase
func (e *engine) flexClampMainWidths(items []flexMeas, widths []float64, contentW, mainSize, gap float64) {
	maxFrozen := make([]bool, len(items))
	available := contentW - gap*float64(len(items)-1)
	for _, item := range items {
		available -= flexRowNormalMargins(e, item)
	}
	if available < 0 {
		available = 0
	}

	for idx, it := range items {
		cstate := e.stylePtr(it.n)

		floor := e.flexMinMainSize(it, mainSize)
		if widths[idx] < floor {
			widths[idx] = floor
		}

		if cstate.MaxWidth >= 0 {
			mx := e.scalePt(cstate.MaxWidth)
			if widths[idx] > mx {
				widths[idx] = mx
				maxFrozen[idx] = true
			}
		}
	}

	sum := flexWidthSum(widths)
	if sum < available-layoutEpsilon && available >= 0 && hasFrozenMax(maxFrozen) {
		e.regrowFlexWidths(items, widths, available, maxFrozen)

		return
	}

	// If mins pushed the sum over contentW, freeze at floors and re-shrink
	// remaining flexible items (css-sizing / flex redistribution lite).
	if sum <= available+layoutEpsilon || available < 0 {
		return
	}
	if flexShrinkFactorSum(items) < 1 {
		return
	}

	e.reshrinkFlexWidths(items, widths, available, mainSize, sum)
}

//nolint:wsl // the sum is a small companion to the shrink redistribution
func flexShrinkFactorSum(items []flexMeas) float64 {
	sum := 0.0
	for _, item := range items {
		if item.shrink > 0 {
			sum += item.shrink
		}
	}

	return sum
}

func hasFrozenMax(maxFrozen []bool) bool {
	for _, frozen := range maxFrozen {
		if frozen {
			return true
		}
	}

	return false
}

func flexWidthSum(widths []float64) float64 {
	sum := 0.0
	for _, width := range widths {
		sum += width
	}

	return sum
}

// regrowFlexWidths gives the remaining positive space to items that were not
// frozen by max-width. A max clamp can turn an initial shrink pass into a
// second positive-free-space pass, as in the flex base size max-width case.
func (e *engine) regrowFlexWidths(items []flexMeas, widths []float64, contentW float64, maxFrozen []bool) {
	for range items {
		remaining := contentW - flexWidthSum(widths)
		if remaining <= layoutEpsilon {
			return
		}

		growSum := flexGrowSum(items, maxFrozen)

		if growSum <= layoutEpsilon {
			return
		}

		if !e.applyFlexRegrow(items, widths, remaining, growSum, maxFrozen) {
			return
		}
	}
}

//nolint:wsl // grow-factor summation keeps the frozen-item filter beside the add
func flexGrowSum(items []flexMeas, maxFrozen []bool) float64 {
	sum := 0.0
	for idx, item := range items {
		if maxFrozen[idx] {
			continue
		}

		factor := item.grow
		if factor <= 0 {
			factor = 1
		}
		sum += factor
	}

	return sum
}

//nolint:wsl // redistribution keeps candidate clamping beside width updates
func (e *engine) applyFlexRegrow(
	items []flexMeas,
	widths []float64,
	remaining, growSum float64,
	maxFrozen []bool,
) bool {
	frozenAnother := false
	for idx, item := range items {
		if maxFrozen[idx] {
			continue
		}

		factor := item.grow
		if factor <= 0 {
			factor = 1
		}
		share := remaining * factor / growSum
		candidate := widths[idx] + share
		style := e.stylePtr(item.n)
		if style.MaxWidth >= 0 && candidate > e.scalePt(style.MaxWidth) {
			widths[idx] = e.scalePt(style.MaxWidth)
			maxFrozen[idx] = true
			frozenAnother = true

			continue
		}

		widths[idx] = candidate
	}

	return frozenAnother
}

// reshrinkFlexWidths iteratively cuts flexible items back toward contentW,
// freezing items at their min floors until the deficit is exhausted.
func (e *engine) reshrinkFlexWidths(items []flexMeas, widths []float64, contentW, mainSize, sum float64) {
	deficit := sum - contentW

	for deficit > layoutEpsilon {
		shrinkable := e.flexShrinkableRoom(items, widths, mainSize)
		if shrinkable <= layoutEpsilon {
			break
		}

		step := deficit
		if step > shrinkable {
			step = shrinkable
		}

		e.cutFlexWidths(items, widths, mainSize, step, shrinkable)

		sum = flexWidthSum(widths)

		if sum >= contentW-layoutEpsilon && sum <= contentW+layoutEpsilon {
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
		if room > layoutEpsilon && it.shrink > 0 {
			roomSum += room
		}
	}

	return roomSum
}

func (e *engine) cutFlexWidths(items []flexMeas, widths []float64, mainSize, step, shrinkable float64) {
	for idx, it := range items {
		floor := e.flexMinMainSize(it, mainSize)

		room := widths[idx] - floor
		if room <= layoutEpsilon || it.shrink <= 0 {
			continue
		}

		cut := step * (room / shrinkable)

		widths[idx] -= cut
		if widths[idx] < floor {
			widths[idx] = floor
		}
	}
}

//nolint:cyclop,funlen,wsl // flex placement keeps its measured state updates together
func (e *engine) placeFlexLineMeasured(
	parent *box, style ResolvedStyle, items []flexMeas,
	contentW, contentX, topY, curY, gap, lineCross float64,
	crossReverse, mainReverse bool,
) float64 {
	widths := e.flexLineWidths(items, contentW, gap)
	gaps := gap * float64(len(items)-1)
	if gaps < 0 {
		gaps = 0
	}

	sumW := 0.0
	for idx, w := range widths {
		sumW += w + flexRowNormalMargins(e, items[idx])
	}

	justify := flexMainJustify(style)
	startX, justifyGap := justifyRowStart(justify, contentX, contentW, sumW, gaps, gap, len(items))
	if mainReverse && flexStartJustify(justify) {
		startX = contentX + contentW - sumW
	}
	if flexRowAutoMarginCount(items, e) > 0 {
		startX = contentX
		justifyGap = gap
	}

	targetCross := lineCross
	if targetCross < 0 {
		targetCross = e.measureFlexCrossMax(items, widths, startX, topY, curY, justifyGap)
	}

	built, rowH := e.buildRowItems(
		parent,
		style,
		items,
		widths,
		contentW,
		topY,
		curY,
		startX,
		justifyGap,
		targetCross,
	)

	alignH := rowH
	if lineCross > alignH {
		alignH = lineCross
	}

	if targetCross > alignH {
		alignH = targetCross
	}

	e.alignRowItems(style, built, topY, curY, alignH, crossReverse)

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
	e.flexClampMainWidths(items, widths, contentW, contentW, gap)

	return widths
}

func (e *engine) flexDistributeWidths(items []flexMeas, widths []float64, contentW, gap float64) {
	var fixed, growSum, shrinkSum float64

	for _, it := range items {
		fixed += it.baseW + flexRowNormalMargins(e, it)
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

	distributionSum := growSum
	if distributionSum < 1 {
		distributionSum = 1
	}

	for i, it := range items {
		if it.grow > 0 {
			widths[i] += free * (it.grow / distributionSum)
		}
	}
}

//nolint:wsl // factor normalization belongs beside the shrink distribution
func (e *engine) flexShrinkWidths(items []flexMeas, widths []float64, deficit, shrinkSum, contentW float64) {
	factorSum := 0.0
	for _, item := range items {
		if item.shrink > 0 {
			factorSum += item.shrink
		}
	}
	if factorSum < 1 {
		deficit *= factorSum
	}

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

// flexMainJustify picks the main-axis packing keyword for a flex container.
// Only justify-content applies here. Non-stretch justify-self on the container
// itself shrinks and aligns the flex box in layoutBlockChild (CSS Align), it
// does not remap to justify-content.
func flexMainJustify(style ResolvedStyle) string {
	jc := style.JustifyContent
	if jc == "" {
		return fxFlexStart
	}

	return jc
}

func flexStartJustify(justify string) bool {
	return justify == "" || justify == flexStartKeyword || justify == fxStart
}

// justifyRowStart resolves the main-axis start offset and gap for a row line
// from justify-content, returning (startX, justifyGap).
func justifyRowStart(justify string, contentX, contentW, sumW, gaps, gap float64, count int) (float64, float64) {
	switch justify {
	case fxFlexEnd, fxEnd:
		return contentX + contentW - sumW - gaps, gap
	case fxCenter:
		return contentX + (contentW-sumW-gaps)/2, gap
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
//
//nolint:wsl // measurement keeps item build and cross-margin accounting together
func (e *engine) measureFlexCrossMax(
	items []flexMeas, widths []float64, startX, topY, curY, justifyGap float64,
) float64 {
	was := e.noEmit
	e.noEmit = true
	maxH := 0.0
	maxX := startX

	for idx, it := range items {
		cb := e.build(it.n, widths[idx], maxX, topY+curY)
		if cb != nil {
			itemStyle := e.stylePtr(it.n)
			itemH := cb.height + e.scalePt(itemStyle.MarginTop) + e.scalePt(itemStyle.MarginBottom)
			if itemH > maxH {
				maxH = itemH
			}
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

	// Flex stretch assigns a used border-box size even when the item's content
	// is taller than that size. The ordinary block resolver deliberately keeps
	// overflowing content from shrinking an explicit height, so carry the
	// flex assignment through as a matching max-height override. This keeps the
	// flex-specific constraint local and lets the content paint/overflow while
	// the item box retains its assigned cross size.
	if style.BoxSizing == borderBox {
		style.Height = forceH / e.scale
		style.MaxHeight = forceH / e.scale
	} else {
		inner := forceH - e.scalePt(style.PaddingTop) - e.scalePt(style.PaddingBottom) -
			e.scalePt(style.BorderTop.Width) - e.scalePt(style.BorderBottom.Width)
		if inner < 0 {
			inner = 0
		}

		style.Height = inner / e.scale
		style.MaxHeight = inner / e.scale
	}

	style.HeightPercent = -1
	style.MaxHeightPercent = -1

	return style
}

// forceFlexItemMainSize preserves the used row-axis size selected by flexbox
// through the child build. Without the override, a percentage-sized item is
// resolved a second time against its already-assigned flex width (46% of 46%
// in fixture-56's gauge row).
func (e *engine) forceFlexItemMainSize(style ResolvedStyle, forceW float64) ResolvedStyle {
	return e.forceFlexItemWidth(style, forceW, true)
}

// forceFlexItemCrossWidth preserves a column flex item's used cross size
// through the child build without changing its main-axis flex basis.
func (e *engine) forceFlexItemCrossWidth(style ResolvedStyle, forceW float64) ResolvedStyle {
	return e.forceFlexItemWidth(style, forceW, false)
}

func (e *engine) forceFlexItemWidth(style ResolvedStyle, forceW float64, clearFlexBasis bool) ResolvedStyle {
	if e.scale <= 0 {
		return style
	}

	if style.BoxSizing == borderBox {
		style.Width = forceW / e.scale
	} else {
		inner := forceW - e.scalePt(style.PaddingLeft) - e.scalePt(style.PaddingRight) -
			e.scalePt(style.BorderLeft.Width) - e.scalePt(style.BorderRight.Width)
		if inner < 0 {
			inner = 0
		}

		style.Width = inner / e.scale
	}

	style.WidthPercent = -1
	if clearFlexBasis {
		style.FlexBasis = -1
		style.FlexBasisPercent = -1
	}

	return style
}

func (e *engine) buildFlexRowItem(
	node *html.Node, style *ResolvedStyle, forceStretch bool,
	targetCross, availW, posX, posY float64,
) *box {
	override := e.forceFlexItemMainSize(*style, availW)
	if forceStretch {
		override = e.forceFlexItemCrossSize(override, targetCross)
	}

	return e.buildWithStyle(node, &override, availW, posX, posY)
}

//nolint:cyclop,funlen,gocognit,wsl // row placement keeps main- and cross-axis phases together
func (e *engine) buildRowItems(
	parent *box, style ResolvedStyle, items []flexMeas, widths []float64,
	contentW, topY, curY, startX, justifyGap, targetCross float64,
) ([]flexPlacedItem, float64) {
	built := make([]flexPlacedItem, 0, len(items))
	rowH := 0.0
	leftX := startX
	autoMainMargins := flexRowAutoMarginCount(items, e)
	autoMainUnit := 0.0
	if autoMainMargins > 0 {
		fixedMargins := 0.0
		for _, item := range items {
			fixedMargins += flexRowNormalMargins(e, item)
		}
		free := contentW - flexWidthSum(widths) - fixedMargins - justifyGap*float64(len(items)-1)
		if free > 0 {
			autoMainUnit = free / float64(autoMainMargins)
		}
	}

	poll := newCtxPoll(e.ctx)
	for idx, item := range items {
		if poll.poll() {
			e.err = poll.err

			return nil, rowH
		}

		cstate := e.stylePtr(item.n)
		if cstate.MarginLeftAuto {
			leftX += autoMainUnit
		} else {
			leftX += e.scalePt(cstate.MarginLeft)
		}

		itemY := topY + curY
		if cstate.MarginTopAuto {
			itemY += flexRowCrossAutoMarginUnit(e, cstate, targetCross)
		} else {
			itemY += e.scalePt(cstate.MarginTop)
		}

		// A stretched item's used cross size is the line's cross size and
		// becomes definite for its contents (CSS Flexbox 9.4 step 5), so a
		// nested flex container re-resolves its children's percentages against
		// it. Intrinsic containers are not exempt: the measured hypothetical
		// cross size is already the line's cross size for a single-line row.
		forceStretch := flexItemCrossStretchNode(item.n, style, *cstate) && targetCross > 0
		stretchCross := targetCross
		if forceStretch {
			stretchCross -= e.scalePt(cstate.MarginTop) + e.scalePt(cstate.MarginBottom)
			if stretchCross < 0 {
				stretchCross = 0
			}
		}
		cblock := e.buildFlexRowItem(item.n, cstate, forceStretch, stretchCross, widths[idx], leftX, itemY)

		if cblock == nil {
			built = append(built, flexPlacedItem{n: item.n}) //nolint:exhaustruct // intentional zero fields

			leftX += widths[idx]
			if cstate.MarginRightAuto {
				leftX += autoMainUnit
			} else {
				leftX += e.scalePt(cstate.MarginRight)
			}
			if idx < len(items)-1 {
				leftX += justifyGap
			}

			continue
		}

		dx := leftX - cblock.x
		dy := itemY - cblock.y
		e.shiftBoxOps(cblock, dx, dy)
		cblock.x += dx
		cblock.y += dy

		itemH := cblock.height + e.scalePt(cstate.MarginTop) + e.scalePt(cstate.MarginBottom)
		if itemH > rowH {
			rowH = itemH
		}

		built = append(built, flexPlacedItem{box: cblock, h: cblock.height, n: item.n})

		if parent != nil {
			parent.children = append(parent.children, cblock)
		}

		leftX += widths[idx]
		if cstate.MarginRightAuto {
			leftX += autoMainUnit
		} else {
			leftX += e.scalePt(cstate.MarginRight)
		}
		if idx < len(items)-1 {
			leftX += justifyGap
		}
	}

	return built, rowH
}

// alignRowItems applies align-items/align-self offsets on the cross axis
// within a line (stretch sizing happened during build).
//
//nolint:cyclop,goconst // alignment keeps baseline and cross-axis branches together
func (e *engine) alignRowItems(
	style ResolvedStyle,
	built []flexPlacedItem,
	topY, cyOffset, alignH float64,
	crossReverse bool,
) {
	baselineY := e.rowBaseline(style, built, topY+cyOffset)

	for _, page := range built {
		if page.box == nil {
			continue
		}

		pageStyle := e.stylePtr(page.n)
		if pageStyle.MarginTopAuto || pageStyle.MarginBottomAuto {
			continue
		}

		align := style.AlignItems
		if pageStyle.AlignSelf != "" && pageStyle.AlignSelf != fxAuto {
			align = pageStyle.AlignSelf
		}

		var deltaY float64

		switch align {
		case fxFlexEnd, fxEnd:
			if !crossReverse {
				deltaY = (topY + cyOffset + alignH) - (page.box.y + page.box.height)
			}
		case fxFlexStart, fxStart:
			if crossReverse {
				deltaY = (topY + cyOffset + alignH) - (page.box.y + page.box.height)
			}
		case fxCenter:
			deltaY = (topY + cyOffset + (alignH-page.box.height)/2) - page.box.y
		case "baseline":
			if target, ok := baselineY[page.box]; ok {
				deltaY = target - page.box.y
			}
		default:
			if crossReverse {
				deltaY = (topY + cyOffset + alignH) - (page.box.y + page.box.height)
			}
		}

		if deltaY != 0 {
			e.shiftBoxOps(page.box, 0, deltaY)
			page.box.y += deltaY
		}
	}
}

//nolint:wsl // baseline collection keeps each item decision adjacent
func (e *engine) rowBaseline(
	style ResolvedStyle, built []flexPlacedItem, lineTop float64,
) map[*box]float64 {
	type baselineItem struct {
		box    *box
		margin float64
		offset float64
	}

	items := make([]baselineItem, 0, len(built))
	maxOffset := 0.0
	for _, page := range built {
		if page.box == nil {
			continue
		}

		itemStyle := e.stylePtr(page.n)
		align := style.AlignItems
		if itemStyle.AlignSelf != "" && itemStyle.AlignSelf != fxAuto {
			align = itemStyle.AlignSelf
		}
		if align != "baseline" {
			continue
		}

		offset := page.box.height
		if page.box.firstBaseline > page.box.y {
			offset = page.box.firstBaseline - page.box.y
		}
		margin := e.scalePt(itemStyle.MarginTop)
		items = append(items, baselineItem{box: page.box, margin: margin, offset: offset})
		if margin+offset > maxOffset {
			maxOffset = margin + offset
		}
	}

	result := make(map[*box]float64, len(items))
	for _, item := range items {
		// The flex line's baseline includes the largest cross-axis margin, but
		// the item margin is not subtracted again when the border box is placed.
		// This matches empty-item baselines and keeps margin-different items on
		// the same physical edge.
		result[item.box] = lineTop + maxOffset - item.offset
	}

	return result
}

//nolint:wsl // main-axis margin counting keeps both auto sides in one pass
func flexRowAutoMarginCount(items []flexMeas, e *engine) int {
	count := 0
	for _, item := range items {
		style := e.stylePtr(item.n)
		if style.MarginLeftAuto {
			count++
		}
		if style.MarginRightAuto {
			count++
		}
	}

	return count
}

//nolint:wsl // cross-axis auto-margin calculation follows the CSS decision order
func flexRowCrossAutoMarginUnit(eng *engine, style *ResolvedStyle, targetCross float64) float64 {
	if targetCross < 0 {
		return 0
	}

	count := 0
	if style.MarginTopAuto {
		count++
	}
	if style.MarginBottomAuto {
		count++
	}
	if count == 0 {
		return 0
	}

	height := targetCross
	switch {
	case style.Height >= 0:
		height = eng.scalePt(style.Height)
	case style.HeightPercent >= 0:
		height = targetCross * style.HeightPercent / oneHundred
	}

	fixed := 0.0
	if !style.MarginTopAuto {
		fixed += eng.scalePt(style.MarginTop)
	}
	if !style.MarginBottomAuto {
		fixed += eng.scalePt(style.MarginBottom)
	}
	free := targetCross - height - fixed
	if free <= 0 {
		return 0
	}

	return free / float64(count)
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
	case fxFlexStart, fxStart, fxFlexEnd, fxEnd, fxCenter, "baseline":
		return false
	}
	// Definite height/% means the used cross size is already specified.
	if cstate2.Height >= 0 || cstate2.HeightPercent >= 0 {
		return false
	}

	return true
}

func flexItemCrossStretchNode(node *html.Node, cstate, cstate2 ResolvedStyle) bool {
	if node != nil {
		switch node.Name {
		case cssTagImg:
			if node.Attribute("width") == "1" && node.Attribute("height") == "1" {
				return false
			}
		case "video", "canvas":
			return false
		}
	}

	return flexItemCrossStretch(cstate, cstate2)
}

// flexItemBaseHeight resolves the flex base size on the column main axis.
// mainSize is the flex container content-box height from resolveContentHeight.
// (−1 when height is auto / indefinite). Percentage flex-basis against an
// indefinite main size is treated as auto (content-based) — CSS Flexbox L1
// §9.2 cyclic %-sizing subset; do not resolve % as 0 silently.
//
//nolint:wsl // no-emit measurement must restore the engine flag before branching
func (e *engine) flexItemBaseHeight(node *html.Node, style ResolvedStyle, contentW, mainSize float64) float64 {
	padV := e.scalePt(style.PaddingTop) + e.scalePt(style.PaddingBottom) +
		e.scalePt(style.BorderTop.Width) + e.scalePt(style.BorderBottom.Width)

	if h, ok := e.flexSpecifiedBaseHeight(style, mainSize, padV); ok {
		return h
	}

	was := e.noEmit
	e.noEmit = true
	measured := e.build(node, contentW, 0, 0)
	e.noEmit = was
	height := 0.0
	if measured != nil {
		height = measured.height
	}

	if height <= 0 {
		height = padV
	}

	return height
}

func (e *engine) flexSpecifiedBaseHeight(style ResolvedStyle, mainSize, padV float64) (float64, bool) {
	switch {
	case style.FlexBasisPercent >= 0 && mainSize >= 0:
		return e.flexBoxSized(style, mainSize*style.FlexBasisPercent/oneHundred, padV), true
	case style.FlexBasis >= 0:
		return e.flexBoxSized(style, e.scalePt(style.FlexBasis), padV), true
	case style.HeightPercent >= 0 && mainSize >= 0:
		return e.flexBoxSized(style, mainSize*style.HeightPercent/oneHundred, padV), true
	case style.Height >= 0:
		return e.flexBoxSized(style, e.scalePt(style.Height), padV), true
	}

	return 0, false
}

// flexMinCrossMainSize is the column-axis content-based min-height floor
// (Flexbox §4.5 lite). mainSize is the definite flex container content height.
func (e *engine) flexMinCrossMainSize(node *html.Node, baseH, mainSize float64) float64 {
	cstate := e.stylePtr(node)
	floor := 0.0

	if cstate.MinHeightPercent >= 0 && mainSize >= 0 {
		floor = mainSize * cstate.MinHeightPercent / oneHundred
	} else if cstate.MinHeight > 0 {
		floor = e.scalePt(cstate.MinHeight)
	}

	if flexAutoMinSizeIsZero(cstate.Overflow) {
		return floor
	}

	padV := e.scalePt(cstate.PaddingTop) + e.scalePt(cstate.PaddingBottom) +
		e.scalePt(cstate.BorderTop.Width) + e.scalePt(cstate.BorderBottom.Width)
	// layoutCell flows the item's children while the container height is the
	// active percentage basis. Measure without emitting, or prependChrome
	// defers the child's background/border into e.deferredChrome and the ops
	// truncation below cannot drop it: finalizeChrome splices the stale entry
	// back at its old index (case-05 painted its percentage child a second
	// time at the container origin). Same contract as measureCellHeight.
	was := e.noEmit
	e.noEmit = true
	start := len(e.ops)
	contentSug := e.layoutCell(node, *cstate, indefiniteContentCap)
	e.ops = e.ops[:start]
	e.noEmit = was

	if contentSug < padV {
		contentSug = padV + e.scalePt(cstate.FontSize)*textLineHeightFactor
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

// flexAutoMinSizeIsZero separates flex sizing from sticky scrollport
// detection. overflow: clip clips paint but does not create a scroll
// container, so it keeps the content-based automatic minimum.
func flexAutoMinSizeIsZero(overflow string) bool {
	switch overflow {
	case overflowAuto, "scroll", overflowHidden:
		return true
	default:
		return false
	}
}

func (e *engine) flexSpecifiedHeightSuggestion(style ResolvedStyle, baseH, mainSize, padV float64) float64 {
	switch {
	case style.HeightPercent >= 0 && mainSize >= 0:
		return e.flexBoxSized(style, mainSize*style.HeightPercent/oneHundred, padV)
	case style.Height >= 0:
		return e.flexBoxSized(style, e.scalePt(style.Height), padV)
	case baseH > 0:
		return baseH
	}

	return -1
}

//nolint:cyclop,wsl // column flow combines wrapping, sizing, and placement
func (e *engine) flowFlexColumn(
	parent *box, kids []*html.Node, style ResolvedStyle,
	contentW, contentX, topY, curY, gap, crossGap float64,
) float64 {
	contentH := resolveFlexColumnContentHeight(style, e)
	items := e.flexColumnItems(kids, contentW, contentH)

	if len(items) == 0 {
		return curY
	}
	if style.FlexWrap == fxWrap || style.FlexWrap == fxWrapRev {
		return e.flowFlexColumnWrapped(parent, style, items, contentW, contentX, topY, curY, gap, crossGap, contentH)
	}

	if style.FlexDirection == fxColRev {
		slices.Reverse(items)
	}

	heights := e.flexColumnHeights(items, contentH, gap)

	gaps := gap * float64(len(items)-1)
	if gaps < 0 {
		gaps = 0
	}

	sumH := 0.0
	for idx, h := range heights {
		sumH += h + flexColumnMainMargins(e, items[idx])
	}

	startY, justifyGap := justifyColumnStart(flexMainJustify(style), contentH, curY, sumH+gaps, sumH, gap, len(items))
	if style.FlexDirection == fxColRev && flexStartJustify(flexMainJustify(style)) &&
		contentH >= 0 && !flexColumnHasAutoMargins(items, e) {
		startY = curY + contentH - sumH - gaps
	}

	endY := e.buildColumnItems(parent, style, items, heights, contentW, contentX, topY, curY, startY, justifyGap, contentH)

	if contentH >= 0 && endY < curY+contentH {
		return curY + contentH
	}

	return endY
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
