package layout

import (
	"slices"
	"sort"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// flexColumnLine is one wrapped column. Its cross size is the width reserved
// for the line, including the widest item's cross-axis margins.
type flexColumnLine struct {
	items []flexColMeas
	width float64
}

// flowFlexColumnWrapped forms column lines against the definite main-axis
// size, sizes each line independently, and then distributes the lines on the
// physical cross axis. The ordinary column path remains separate because it
// has no line packing to perform.
//
// Wrapped columns keep line formation and physical placement together.
//
//nolint:cyclop,funlen,nlreturn,wsl
func (e *engine) flowFlexColumnWrapped(
	parent *box,
	style ResolvedStyle,
	items []flexColMeas,
	contentW, contentX, topY, curY, gap, crossGap, contentH float64,
) float64 {
	if contentH < 0 {
		nowrap := style
		nowrap.FlexWrap = ""
		return e.flowFlexColumn(
			parent, itemsToNodes(items), nowrap, contentW, contentX, topY, curY, gap, crossGap,
		)
	}

	lines := e.flexColumnLines(items, contentH, gap)
	widths := make([]float64, len(lines))
	for idx, line := range lines {
		widths[idx] = e.flexColumnLineCrossWidth(line.items)
	}

	widths = stretchFlexColumnLineWidths(style.AlignContent, widths, contentW, crossGap)
	offsets := flexColumnLineOffsets(style.AlignContent, widths, contentW, crossGap)
	reverseCross := style.Direction == cssDirectionRTL
	if style.FlexWrap == fxWrapRev {
		reverseCross = !reverseCross
	}

	for idx, line := range lines {
		lineWidth := widths[idx]
		lineX := contentX + offsets[idx]
		if reverseCross {
			lineX = contentX + contentW - lineWidth - offsets[idx]
		}

		heights := e.flexColumnHeights(line.items, contentH, gap)
		sumH := 0.0
		for itemIdx, height := range heights {
			sumH += height + flexColumnMainMargins(e, line.items[itemIdx])
		}

		gaps := gap * float64(len(heights)-1)
		if gaps < 0 {
			gaps = 0
		}
		startY, justifyGap := justifyColumnStart(
			flexMainJustify(style), contentH, curY, sumH+gaps, sumH, gap, len(heights),
		)
		if style.FlexDirection == fxColRev && flexStartJustify(flexMainJustify(style)) &&
			!flexColumnHasAutoMargins(line.items, e) {
			startY = curY + contentH - sumH - gaps
		}
		if style.FlexDirection == fxColRev {
			slices.Reverse(line.items)
			slices.Reverse(heights)
		}

		e.buildColumnItems(
			parent,
			style,
			line.items,
			heights,
			lineWidth,
			lineX,
			topY,
			curY,
			startY,
			justifyGap,
			contentH,
		)
	}

	return curY + contentH
}

func itemsToNodes(items []flexColMeas) []*html.Node {
	nodes := make([]*html.Node, len(items))
	for idx, item := range items {
		nodes[idx] = item.n
	}

	return nodes
}

func (e *engine) flexColumnLines(items []flexColMeas, contentH, gap float64) []flexColumnLine {
	lines := make([]flexColumnLine, 0, len(items))
	line := flexColumnLine{items: nil, width: 0}
	used := 0.0

	for _, item := range items {
		need := item.baseH + flexColumnMainMargins(e, item)
		if len(line.items) > 0 {
			need += gap
		}

		if len(line.items) > 0 && used+need > contentH+layoutEpsilon {
			lines = append(lines, line)
			line = flexColumnLine{items: nil, width: 0}
			used = 0
			need = item.baseH + flexColumnMainMargins(e, item)
		}

		line.items = append(line.items, item)
		used += need
	}

	if len(line.items) > 0 {
		lines = append(lines, line)
	}

	return lines
}

//nolint:wsl // width selection mirrors CSS fallback order
func (e *engine) flexColumnLineCrossWidth(items []flexColMeas) float64 {
	width := 0.0
	for _, item := range items {
		style := e.stylePtr(item.n)
		var itemWidth float64
		switch {
		case style.Width >= 0:
			chrome := e.scalePt(style.PaddingLeft + style.PaddingRight +
				style.BorderLeft.Width + style.BorderRight.Width)
			itemWidth = e.flexBoxSized(*style, e.scalePt(style.Width), chrome)
		case style.MinWidth > 0:
			itemWidth = e.scalePt(style.MinWidth)
		default:
			itemWidth = e.measureFlexItemMaxContent(item.n, *style)
		}

		itemWidth += e.scalePt(style.MarginLeft + style.MarginRight)
		if itemWidth > width {
			width = itemWidth
		}
	}

	return width
}

func flexColumnHasAutoMargins(items []flexColMeas, e *engine) bool {
	for _, item := range items {
		style := e.stylePtr(item.n)
		if style.MarginTopAuto || style.MarginBottomAuto {
			return true
		}
	}

	return false
}

func stretchFlexColumnLineWidths(align string, widths []float64, contentW, gap float64) []float64 {
	if align != "" && align != fxStretch || len(widths) == 0 {
		return widths
	}

	free := contentW - flexWidthSum(widths) - gap*float64(len(widths)-1)
	if free <= layoutEpsilon {
		return widths
	}

	extra := free / float64(len(widths))
	for idx := range widths {
		widths[idx] += extra
	}

	return widths
}

//nolint:wsl // offset distribution keeps keyword cases in one switch
func flexColumnLineOffsets(align string, widths []float64, contentW, gap float64) []float64 {
	free := contentW - flexWidthSum(widths) - gap*float64(len(widths)-1)
	if free < 0 {
		free = 0
	}

	base, step := 0.0, gap
	switch align {
	case fxFlexEnd, fxEnd:
		base = free
	case fxCenter:
		base = free / two
	case fxBetween:
		if len(widths) > 1 {
			step += free / float64(len(widths)-1)
		}
	case fxAround:
		unit := free / float64(two*len(widths))
		base, step = unit, gap+two*unit
	case fxEvenly:
		unit := free / float64(len(widths)+1)
		base, step = unit, gap+unit
	}

	offsets := make([]float64, len(widths))
	for idx := range offsets {
		offsets[idx] = base
		base += widths[idx] + step
	}

	return offsets
}

//nolint:wsl // min-height normalization is a short, ordered fallback
func resolveFlexColumnContentHeight(style ResolvedStyle, eng *engine) float64 {
	contentH := resolveContentHeight(style, eng)
	if contentH >= 0 || style.MinHeight <= 0 {
		return contentH
	}

	contentH = eng.scalePt(style.MinHeight)
	if style.BoxSizing == borderBox {
		contentH -= eng.scalePt(style.PaddingTop) + eng.scalePt(style.PaddingBottom) +
			eng.scalePt(style.BorderTop.Width) + eng.scalePt(style.BorderBottom.Width)
	}
	if contentH < 0 {
		contentH = 0
	}

	return contentH
}

func (e *engine) flexColumnItems(kids []*html.Node, contentW, contentH float64) []flexColMeas {
	items := make([]flexColMeas, 0, len(kids))

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

		items = append(items, flexColMeas{
			n: kid, baseH: e.flexItemBaseHeight(kid, *cstate, contentW, contentH),
			grow: grow, shrink: shrink,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		return e.stylePtr(items[i].n).FlexOrder < e.stylePtr(items[j].n).FlexOrder
	})

	return items
}

// flexColumnHeights resolves used main sizes: grow/shrink redistribution over
// the definite container height, then min/max-height clamping.
func (e *engine) flexColumnHeights(items []flexColMeas, contentH, gap float64) []float64 {
	heights := make([]float64, len(items))

	var fixed, growSum, shrinkSum float64

	for i, item := range items {
		heights[i] = item.baseH
		// Main-axis margins consume flex container space but do not participate
		// in the shrink factor. Placement adds the same margins after sizing.
		fixed += item.baseH + flexColumnMainMargins(e, item)
		growSum += item.grow
		shrinkSum += item.shrink * item.baseH
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
		cstate := e.stylePtr(it.n)

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
			return curY + (contentH-totalH)/2, gap
		case fxBetween, fxAround, fxEvenly:
			return justifyDistributed(justify, curY, contentH, sumH, gap, count)
		}
	}

	return curY, gap
}

//nolint:cyclop,funlen,gocognit,wsl // column placement keeps CSS auto-margin and alignment phases together
func (e *engine) buildColumnItems(
	parent *box, style ResolvedStyle, items []flexColMeas, heights []float64,
	contentW, contentX, topY, curY, startY, justifyGap, contentH float64,
) float64 {
	leftY := startY
	endY := curY
	autoMargins := 0
	poll := newCtxPoll(e.ctx)
	for _, item := range items {
		if poll.poll() {
			e.err = poll.err

			return endY
		}

		itemStyle := e.stylePtr(item.n)
		if itemStyle.MarginTopAuto {
			autoMargins++
		}
		if itemStyle.MarginBottomAuto {
			autoMargins++
		}
	}
	autoUnit := 0.0
	if contentH >= 0 && autoMargins > 0 {
		used := 0.0
		for _, height := range heights {
			used += height
		}
		for _, item := range items {
			used += flexColumnMainMargins(e, item)
		}
		used += justifyGap * float64(len(items)-1)
		if free := contentH - used; free > 0 {
			autoUnit = free / float64(autoMargins)
		}
	}

	for idx, item := range items {
		cstate := e.stylePtr(item.n)
		if cstate.MarginTopAuto {
			leftY += autoUnit
		} else {
			leftY += e.scalePt(cstate.MarginTop)
		}
		// Force border-box height so grow/shrink targets stick through build.
		override := e.flexColumnItemOverride(item.n, *cstate, style, heights[idx], contentW)
		cblock := e.buildWithStyle(item.n, &override, contentW, contentX, topY+leftY)

		if cblock == nil {
			leftY += heights[idx]
			if cstate.MarginBottomAuto {
				leftY += autoUnit
			} else {
				leftY += e.scalePt(cstate.MarginBottom)
			}
			if idx < len(items)-1 {
				leftY += justifyGap
			}

			continue
		}

		desiredX := contentX
		if !cstate.MarginLeftAuto {
			desiredX += e.scalePt(cstate.MarginLeft)
		}
		dx := desiredX - cblock.x
		dy := (topY + leftY) - cblock.y
		e.shiftBoxOps(cblock, dx, dy)
		cblock.x += dx
		cblock.y += dy

		e.alignColumnItem(cblock, style, *cstate, contentX, contentW)

		if parent != nil {
			parent.children = append(parent.children, cblock)
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

func (e *engine) flexColumnItemOverride(
	node *html.Node, itemStyle, containerStyle ResolvedStyle, height, contentW float64,
) ResolvedStyle {
	override := e.forceFlexItemCrossSize(itemStyle, height)
	if flexItemColumnCrossStretch(containerStyle, itemStyle) || itemStyle.Width >= 0 || itemStyle.WidthPercent >= 0 {
		return override
	}

	crossW := e.measureFlexItemMaxContent(node, itemStyle)
	if crossW > contentW {
		crossW = contentW
	}

	return e.forceFlexItemCrossWidth(override, crossW)
}

// alignColumnItem places a column item on the cross axis. Auto cross-axis
// margins take the line's positive free space before alignment (CSS Flexbox
// L1 8.1), and when they do, the alignment keywords no longer move the item.
//
//nolint:wsl // column alignment keeps auto-margin and keyword branches together
func (e *engine) alignColumnItem(
	cblock *box, containerStyle, itemStyle ResolvedStyle, contentX, contentW float64,
) {
	if flexColumnCrossAutoMargin(itemStyle) {
		if offset, absorbed := columnCrossAutoMarginOffset(itemStyle, contentW-cblock.w); absorbed {
			if offset != 0 {
				e.shiftBoxOps(cblock, offset, 0)
				cblock.x += offset
			}

			return
		}
	}

	align := containerStyle.AlignItems
	if align == "" {
		align = fxStretch
	}
	if itemStyle.AlignSelf != "" && itemStyle.AlignSelf != fxAuto {
		align = itemStyle.AlignSelf
	}

	reverseCross := containerStyle.Direction == cssDirectionRTL
	if containerStyle.FlexWrap == fxWrapRev {
		reverseCross = !reverseCross
	}
	if adx := columnAlignOffset(align, cblock, itemStyle, contentX, contentW, reverseCross, e); adx != 0 {
		e.shiftBoxOps(cblock, adx, 0)
		cblock.x += adx
	}
}

// columnCrossAutoMarginOffset resolves the cross-axis offset a column item
// takes from the line's positive free space. absorbed is false when free space
// is not positive, so the caller still applies align-items / align-self.
func columnCrossAutoMarginOffset(itemStyle ResolvedStyle, free float64) (float64, bool) {
	if free <= 0 {
		return 0, false
	}

	switch {
	case itemStyle.MarginLeftAuto && itemStyle.MarginRightAuto:
		return free / two, true
	case itemStyle.MarginLeftAuto:
		return free, true
	}

	// A right-only auto margin takes all the free space, so the border box
	// stays at the line start.
	return 0, true
}

// columnAlignOffset is the cross-axis shift for the alignment keyword.
//
//nolint:wsl // alignment keyword mapping is easiest to audit as one switch
func columnAlignOffset(
	align string, cblock *box, itemStyle ResolvedStyle, contentX, contentW float64,
	reverse bool, e *engine,
) float64 {
	left := e.scalePt(itemStyle.MarginLeft)
	right := e.scalePt(itemStyle.MarginRight)
	switch align {
	case "baseline":
		// A column's cross axis is the inline axis. For empty items, the
		// baseline is their start-side border edge, so direction controls
		// whether that edge is the physical left or right side.
		if reverse {
			return contentX + contentW - right - cblock.w - cblock.x
		}

		return contentX + left - cblock.x
	case fxCenter:
		return contentX + (contentW-cblock.w)/2 - cblock.x
	case fxFlexEnd, fxEnd:
		if reverse {
			return contentX + left - cblock.x
		}

		return contentX + contentW - right - cblock.w - cblock.x
	case fxFlexStart, fxStart:
		if reverse {
			return contentX + contentW - right - cblock.w - cblock.x
		}

		return contentX + left - cblock.x
	case fxStretch:
		if reverse {
			return contentX + contentW - right - cblock.w - cblock.x
		}

		return contentX + left - cblock.x
	}

	return 0
}

// flexColumnCrossAutoMargin reports whether a column item has an auto
// cross-axis (horizontal) margin.
func flexColumnCrossAutoMargin(style ResolvedStyle) bool {
	return style.MarginLeftAuto || style.MarginRightAuto
}

// flexItemColumnCrossStretch reports whether an auto-width column item uses
// the container's cross size. Non-stretch alignment uses the item's intrinsic
// width instead, so alignColumnItem can place it at start, center, or end.
// Auto cross-axis margins also suppress stretch (CSS Flexbox L1 8.5): the
// item keeps its hypothetical cross size and the margins absorb the rest.
func flexItemColumnCrossStretch(cstate, cstate2 ResolvedStyle) bool {
	if flexColumnCrossAutoMargin(cstate2) {
		return false
	}

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

	return cstate2.Width < 0 && cstate2.WidthPercent < 0
}
