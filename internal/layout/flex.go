package layout

import (
	"sort"

	"gowkhtmltopdf/internal/html"
)

type flexMeas struct {
	n      *html.Node
	baseW  float64
	grow   float64
	shrink float64
	order  int
}

// buildFlex lays out a flex container (row or column) with a report-friendly
// subset: justify-content, align-items/self, align-content, gap/row-gap/
// column-gap, flex-grow/shrink/basis, order, wrap, and reverse directions.
func (e *engine) buildFlex(node *html.Node, sty ResolvedStyle, availW, x, posY float64) *box {
	ml := e.scalePt(sty.MarginLeft)
	boxNode := &box{node: node, style: sty, kind: "block", x: x + ml, y: posY} //nolint:exhaustruct // intentional zero fields
	boxNode.w = resolveUsedWidth(sty, availW, e)
	contentX, contentW := e.contentBox(boxNode.x, boxNode.w, sty)
	contentStart := len(e.ops)
	curY := e.scalePt(sty.PaddingTop) + e.scalePt(sty.BorderTop.Width)

	var kids []*html.Node

	for _, child := range node.Children {
		if child.Type != html.ElementNode {
			continue
		}

		cs := e.styles[child]
		if cs.Display == "none" {
			continue
		}

		kids = append(kids, child)
	}

	rowGap, colGap := e.styleGaps(sty)

	dir := sty.FlexDirection
	if dir == "" {
		dir = "row"
	}

	if dir == "column" || dir == "column-reverse" {
		curY = e.flowFlexColumn(boxNode, kids, sty, contentW, contentX, posY, curY, rowGap)
	} else {
		curY = e.flowFlexRow(boxNode, kids, sty, contentW, contentX, posY, curY, colGap, rowGap)
	}

	curY += e.scalePt(sty.PaddingBottom)

	if sty.Height >= 0 {
		height := e.scalePt(sty.Height)
		if sty.BoxSizing != "border-box" {
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

func (e *engine) flowFlexRow(parent *box, kids []*html.Node, st ResolvedStyle, contentW, contentX, y, cy, colGap, rowGap float64) float64 {
	if len(kids) == 0 {
		return cy
	}

	wrap := st.FlexWrap == "wrap" || st.FlexWrap == "wrap-reverse"
	reverse := st.FlexDirection == "row-reverse"

	items := make([]flexMeas, 0, len(kids))

	for _, kid := range kids {
		cstate := e.styles[kid]

		gap := cstate.FlexGrow
		if gap < 0 {
			gap = 0
		}

		shval := cstate.FlexShrink
		if shval < 0 {
			shval = 1
		}

		items = append(items, flexMeas{
			n: kid, baseW: e.flexItemBaseWidth(kid, cstate, contentW),
			grow: gap, shrink: shval, order: cstate.FlexOrder,
		})
	}

	sort.SliceStable(items, func(i, j int) bool { return items[i].order < items[j].order })

	var lines [][]flexMeas

	if !wrap {
		line := items
		if reverse {
			reverseFlexMeas(line)
		}

		lines = [][]flexMeas{line}
	} else {
		var line []flexMeas

		used := 0.0

		for _, item := range items {
			need := item.baseW
			if len(line) > 0 {
				need += colGap
			}

			if len(line) > 0 && used+need > contentW+1e-6 {
				if reverse {
					reverseFlexMeas(line)
				}

				lines = append(lines, line)
				line = nil
				used = 0
				need = item.baseW
			}

			line = append(line, item)
			used += need
		}

		if len(line) > 0 {
			if reverse {
				reverseFlexMeas(line)
			}

			lines = append(lines, line)
		}

		if st.FlexWrap == "wrap-reverse" {
			for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
				lines[i], lines[j] = lines[j], lines[i]
			}
		}
	}

	type linePlace struct {
		startChild int
		endChild   int
		y0         float64
		h          float64
	}
	// Single-line flex with definite height: line cross size is the container
	// content height (CSS flexbox §9.6), so align-items/self have room.
	lineCross := -1.0
	if !wrap {
		lineCross = resolveContentHeight(st, e)
	}

	placed := make([]linePlace, 0, len(lines))

	for lidx, line := range lines {
		startChild := 0
		if parent != nil {
			startChild = len(parent.children)
		}

		yStart := cy
		cy = e.placeFlexLineMeasured(parent, st, line, contentW, contentX, y, cy, colGap, lineCross)

		endChild := startChild
		if parent != nil {
			endChild = len(parent.children)
		}

		placed = append(placed, linePlace{startChild: startChild, endChild: endChild, y0: yStart, h: cy - yStart})

		if lidx < len(lines)-1 {
			cy += rowGap
		}
	}

	// align-content: distribute free cross space when height is definite and
	// wrapping produced multiple lines. Height:auto → pack at start (no-op).
	contentH := resolveContentHeight(st, e)
	if contentH >= 0 && len(placed) > 1 {
		linesH := 0.0
		for _, lp := range placed {
			linesH += lp.h
		}

		gapsH := rowGap * float64(len(placed)-1)

		free := contentH - linesH - gapsH
		if free > layoutEpsilon {
			offsets := make([]float64, len(placed))

			switch st.AlignContent {
			case "flex-end", "end":
				for i := range offsets {
					offsets[i] = free
				}
			case "center":
				for i := range offsets {
					offsets[i] = free / two
				}
			case "space-between":
				step := free / float64(len(placed)-1)
				for i := range offsets {
					offsets[i] = step * float64(i)
				}
			case "space-around":
				unit := free / float64(two*len(placed))
				for i := range offsets {
					offsets[i] = unit + 2*unit*float64(i)
				}
			case "space-evenly":
				unit := free / float64(len(placed)+1)
				for i := range offsets {
					offsets[i] = unit * float64(i+1)
				}
			default:
				// stretch / flex-start: pack at start
			}

			if parent != nil {
				for lpad, lpad2 := range placed {
					deltaY := offsets[lpad]
					if deltaY == 0 {
						continue
					}

					for ci := lpad2.startChild; ci < lpad2.endChild && ci < len(parent.children); ci++ {
						cb := parent.children[ci]
						e.shiftBoxOps(cb, 0, deltaY)
						cb.y += deltaY
					}
				}
			}

			last := placed[len(placed)-1]
			cy = last.y0 + offsets[len(offsets)-1] + last.h
		}
	}

	return cy
}

func reverseFlexMeas(line []flexMeas) {
	for i, j := 0, len(line)-1; i < j; i, j = i+1, j-1 {
		line[i], line[j] = line[j], line[i]
	}
}

// flexItemBaseWidth resolves the flex base size on the row main axis.
// mainSize is the flex container content-box width, or <0 when indefinite
// (shrink-to-fit). Percentage flex-basis against an indefinite main size is
// treated as auto (content-based) — CSS Flexbox L1 §9.2 cyclic %-sizing subset.
func (e *engine) flexItemBaseWidth(n *html.Node, cs ResolvedStyle, mainSize float64) float64 {
	pad := e.scalePt(cs.PaddingLeft) + e.scalePt(cs.PaddingRight) +
		e.scalePt(cs.BorderLeft.Width) + e.scalePt(cs.BorderRight.Width)

	capW := mainSize
	if capW < 0 {
		capW = 1e9
	}

	if cs.FlexBasisPercent >= 0 {
		if mainSize < 0 {
			// Cyclic % basis → auto; fall through to width / content.
		} else {
			w := mainSize * cs.FlexBasisPercent / cssPercent
			if cs.BoxSizing != "border-box" {
				w += pad
			}

			return w
		}
	}

	if cs.FlexBasis >= 0 {
		w := e.scalePt(cs.FlexBasis)
		if cs.BoxSizing != "border-box" {
			w += pad
		}

		return w
	}

	if cs.WidthPercent >= 0 && mainSize >= 0 {
		w := mainSize * cs.WidthPercent / cssPercent
		if cs.BoxSizing != "border-box" {
			w += pad
		}

		return w
	}

	if cs.Width >= 0 {
		w := e.scalePt(cs.Width)
		if cs.BoxSizing != "border-box" {
			w += pad
		}

		return w
	}

	intr := e.measureCellContent(n, cs) + pad +
		e.scalePt(cs.MarginLeft) + e.scalePt(cs.MarginRight)
	if intr <= 0 {
		intr = pad + e.scalePt(cs.FontSize)*two
	}

	if intr > capW {
		intr = capW
	}

	return intr
}

// flexMinMainSize is the content-based minimum main size (Flexbox §4.5 lite /
// css-sizing-3): max(specified min-width, min(content suggestion, specified
// size suggestion when definite)). Used as the shrink floor so text does not
// crush to 0. mainSize is the definite flex container content main size, or
// <0 when indefinite (then % min-width is ignored — cyclic honesty).
func (e *engine) flexMinMainSize(it flexMeas, mainSize float64) float64 {
	cstate := e.styles[it.n]
	floor := 0.0

	if cstate.MinWidthPercent >= 0 && mainSize >= 0 {
		floor = mainSize * cstate.MinWidthPercent / cssPercent
	} else if cstate.MinWidth > 0 {
		floor = e.scalePt(cstate.MinWidth)
	}
	// Automatic minimum (min-width:auto): content size suggestion.
	intr := e.measureCellContent(it.n, cstate)
	pad := e.scalePt(cstate.PaddingLeft) + e.scalePt(cstate.PaddingRight) +
		e.scalePt(cstate.BorderLeft.Width) + e.scalePt(cstate.BorderRight.Width)
	contentSug := intr + pad
	// Specified size suggestion when width/% is definite against mainSize.
	specSug := -1.0
	if cstate.WidthPercent >= 0 && mainSize >= 0 {
		specSug = mainSize * cstate.WidthPercent / cssPercent
		if cstate.BoxSizing != "border-box" {
			specSug += pad
		}
	} else if cstate.Width >= 0 {
		specSug = e.scalePt(cstate.Width)
		if cstate.BoxSizing != "border-box" {
			specSug += pad
		}
	} else if it.baseW > 0 {
		specSug = it.baseW
	}

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

	gapExtra := 0.0 // gaps already excluded from contentW by caller
	_ = gapExtra

	if sum <= contentW+1e-6 || contentW < 0 {
		return
	}

	deficit := sum - contentW
	for deficit > 1e-6 {
		var shrinkable float64

		for i, it := range items {
			floor := e.flexMinMainSize(it, mainSize)

			room := widths[i] - floor
			if room > 1e-6 && it.shrink > 0 {
				shrinkable += room
			}
		}

		if shrinkable <= layoutEpsilon {
			break
		}

		step := deficit
		if step > shrinkable {
			step = shrinkable
		}

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

func (e *engine) placeFlexLineMeasured(parent *box, st ResolvedStyle, items []flexMeas, contentW, contentX, y, cy, gap, lineCross float64) float64 {
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

	widths := make([]float64, len(items))
	for i, it := range items {
		widths[i] = it.baseW
	}

	if free > 0 && growSum > 0 {
		for i, it := range items {
			if it.grow > 0 {
				widths[i] += free * (it.grow / growSum)
			}
		}
	} else if free < 0 && shrinkSum > 0 {
		deficit := -free

		for item, item2 := range items {
			if item2.shrink <= 0 || item2.baseW <= 0 {
				continue
			}

			share := (item2.shrink * item2.baseW) / shrinkSum
			widths[item] -= deficit * share
			floor := e.flexMinMainSize(item2, contentW)

			if widths[item] < floor {
				widths[item] = floor
			}
		}
	}
	// Clamp to min/max-width after grow/shrink; re-resolve when mins overflow.
	e.flexClampMainWidths(items, widths, contentW, contentW)

	sumW := 0.0
	for _, w := range widths {
		sumW += w
	}

	totalW := sumW + gaps
	startX := contentX
	justifyGap := gap

	switch st.JustifyContent {
	case "flex-end", "end":
		startX = contentX + (contentW - totalW)
	case "center":
		startX = contentX + (contentW-totalW)/two
	case "space-between":
		if len(items) > 1 {
			rem := contentW - sumW
			if rem > 0 {
				justifyGap = rem / float64(len(items)-1)
			}

			startX = contentX
		}
	case "space-around":
		if len(items) > 0 {
			rem := contentW - sumW
			if rem < 0 {
				rem = 0
			}

			unit := rem / float64(two*len(items))
			startX = contentX + unit
			justifyGap = two * unit
		}
	case "space-evenly":
		if len(items) > 0 {
			rem := contentW - sumW
			if rem < 0 {
				rem = 0
			}

			unit := rem / float64(len(items)+1)
			startX = contentX + unit
			justifyGap = unit
		}
	}

	type placed struct {
		box *box
		h   float64
		n   *html.Node
	}

	built := make([]placed, 0, len(items))
	rowH := 0.0
	leftX := startX

	// Cross-size target for align-items:stretch. Definite container height
	// (lineCross) is the flex line cross size; otherwise measure content max.
	targetCross := lineCross
	if targetCross < 0 {
		was := e.noEmit
		e.noEmit = true
		maxH := 0.0

		maxX := startX
		for idx, it := range items {
			cb := e.build(it.n, widths[idx], maxX, y+cy)
			if cb != nil && cb.height > maxH {
				maxH = cb.height
			}

			maxX += widths[idx]
			if idx < len(items)-1 {
				maxX += justifyGap
			}
		}

		e.noEmit = was
		targetCross = maxH
	}

	for item, item2 := range items {
		cstate := e.styles[item2.n]
		origH := cstate.Height

		forceStretch := flexItemCrossStretch(st, cstate) && targetCross > 0
		if forceStretch {
			// Used cross size = line cross size (border box), matching column
			// main-size forcing so backgrounds fill the flex line (fixture-33).
			padV := e.scalePt(cstate.PaddingTop) + e.scalePt(cstate.PaddingBottom) +
				e.scalePt(cstate.BorderTop.Width) + e.scalePt(cstate.BorderBottom.Width)
			forceH := targetCross

			if e.scale > 0 {
				if cstate.BoxSizing == "border-box" {
					cstate.Height = forceH / e.scale
				} else {
					inner := forceH - padV
					if inner < 0 {
						inner = 0
					}

					cstate.Height = inner / e.scale
				}

				e.styles[item2.n] = cstate
			}
		}

		cblock := e.build(item2.n, widths[item], leftX, y+cy)

		if forceStretch {
			cstate.Height = origH
			e.styles[item2.n] = cstate
		}

		if cblock == nil {
			built = append(built, placed{n: item2.n}) //nolint:exhaustruct // intentional zero fields

			leftX += widths[item]
			if item < len(items)-1 {
				leftX += justifyGap
			}

			continue
		}

		dx := leftX - cblock.x
		dy := (y + cy) - cblock.y
		e.shiftBoxOps(cblock, dx, dy)
		cblock.x += dx
		cblock.y += dy

		if cblock.height > rowH {
			rowH = cblock.height
		}

		built = append(built, placed{box: cblock, h: cblock.height, n: item2.n})

		if parent != nil {
			parent.children = append(parent.children, cblock)
		}

		leftX += widths[item]
		if item < len(items)-1 {
			leftX += justifyGap
		}
	}

	alignH := rowH
	if lineCross >= 0 && lineCross > alignH {
		alignH = lineCross
	}

	if targetCross > alignH {
		alignH = targetCross
	}

	for _, page := range built {
		if page.box == nil {
			continue
		}

		align := st.AlignItems
		if cs, ok := e.styles[page.n]; ok && cs.AlignSelf != "" && cs.AlignSelf != "auto" {
			align = cs.AlignSelf
		}

		deltaY := 0.0

		switch align {
		case "flex-end", "end":
			deltaY = (y + cy + alignH) - (page.box.y + page.box.height)
		case "center":
			deltaY = (y + cy + (alignH-page.box.height)/2) - page.box.y
		default:
			// stretch / flex-start / start: pack at cross-start (stretch already sized)
			deltaY = 0
		}

		if deltaY != 0 {
			e.shiftBoxOps(page.box, 0, deltaY)
			page.box.y += deltaY
		}
	}

	if lineCross >= 0 && lineCross > rowH {
		return cy + lineCross
	}

	if targetCross > rowH {
		return cy + targetCross
	}

	return cy + rowH
}

// flexItemCrossStretch reports whether a flex item should stretch on the cross
// axis (align-items/self stretch, and no definite cross size).
func flexItemCrossStretch(cstate, cstate2 ResolvedStyle) bool {
	align := cstate.AlignItems
	if align == "" {
		align = "stretch"
	}

	if cstate2.AlignSelf != "" && cstate2.AlignSelf != "auto" {
		align = cstate2.AlignSelf
	}

	switch align {
	case "flex-start", "start", "flex-end", "end", "center":
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
func (e *engine) flexItemBaseHeight(n *html.Node, cs ResolvedStyle, contentW, mainSize float64) float64 {
	padV := e.scalePt(cs.PaddingTop) + e.scalePt(cs.PaddingBottom) +
		e.scalePt(cs.BorderTop.Width) + e.scalePt(cs.BorderBottom.Width)

	if cs.FlexBasisPercent >= 0 {
		if mainSize < 0 {
			// Cyclic % basis → auto; fall through to height / content.
		} else {
			h := mainSize * cs.FlexBasisPercent / cssPercent
			if cs.BoxSizing != "border-box" {
				h += padV
			}

			return h
		}
	}

	if cs.FlexBasis >= 0 {
		h := e.scalePt(cs.FlexBasis)
		if cs.BoxSizing != "border-box" {
			h += padV
		}

		return h
	}

	if cs.HeightPercent >= 0 && mainSize >= 0 {
		h := mainSize * cs.HeightPercent / cssPercent
		if cs.BoxSizing != "border-box" {
			h += padV
		}

		return h
	}

	if cs.Height >= 0 {
		h := e.scalePt(cs.Height)
		if cs.BoxSizing != "border-box" {
			h += padV
		}

		return h
	}

	start := len(e.ops)
	height := e.layoutCell(n, cs, contentW)
	e.ops = e.ops[:start]

	if height <= 0 {
		height = padV + e.scalePt(cs.FontSize)*defaultLineHeightRatio
	}

	return height
}

// flexMinCrossMainSize is the column-axis content-based min-height floor
// (Flexbox §4.5 lite). mainSize is the definite flex container content height.
func (e *engine) flexMinCrossMainSize(n *html.Node, baseH, mainSize float64) float64 {
	cstate := e.styles[n]
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
	contentSug := e.layoutCell(n, cstate, infiniteMeasure)
	e.ops = e.ops[:start]

	if contentSug < padV {
		contentSug = padV + e.scalePt(cstate.FontSize)*defaultLineHeightRatio
	}

	specSug := -1.0
	if cstate.HeightPercent >= 0 && mainSize >= 0 {
		specSug = mainSize * cstate.HeightPercent / cssPercent
		if cstate.BoxSizing != "border-box" {
			specSug += padV
		}
	} else if cstate.Height >= 0 {
		specSug = e.scalePt(cstate.Height)
		if cstate.BoxSizing != "border-box" {
			specSug += padV
		}
	} else if baseH > 0 {
		specSug = baseH
	}

	autoMin := contentSug
	if specSug >= 0 && specSug < autoMin {
		autoMin = specSug
	}

	if autoMin > floor {
		floor = autoMin
	}

	return floor
}

func (e *engine) flowFlexColumn(parent *box, kids []*html.Node, st ResolvedStyle, contentW, contentX, y, cy, gap float64) float64 {
	type colMeas struct {
		n      *html.Node
		baseH  float64
		grow   float64
		shrink float64
	}

	contentH := resolveContentHeight(st, e)
	items := make([]colMeas, 0, len(kids))

	for _, kid := range kids {
		cstate := e.styles[kid]

		gap := cstate.FlexGrow
		if gap < 0 {
			gap = 0
		}

		shval := cstate.FlexShrink
		if shval < 0 {
			shval = 1
		}

		items = append(items, colMeas{
			n: kid, baseH: e.flexItemBaseHeight(kid, cstate, contentW, contentH),
			grow: gap, shrink: shval,
		})
	}

	sort.SliceStable(items, func(i, j int) bool {
		return e.styles[items[i].n].FlexOrder < e.styles[items[j].n].FlexOrder
	})

	if st.FlexDirection == "column-reverse" {
		for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
			items[i], items[j] = items[j], items[i]
		}
	}

	if len(items) == 0 {
		return cy
	}

	heights := make([]float64, len(items))

	var fixed, growSum, shrinkSum float64

	for i, it := range items {
		heights[i] = it.baseH
		fixed += it.baseH
		growSum += it.grow
		shrinkSum += it.shrink * it.baseH
	}

	gaps := gap * float64(len(items)-1)
	if gaps < 0 {
		gaps = 0
	}

	if contentH >= 0 {
		free := contentH - fixed - gaps
		if free > 0 && growSum > 0 {
			for i, it := range items {
				if it.grow > 0 {
					heights[i] += free * (it.grow / growSum)
				}
			}
		} else if free < 0 && shrinkSum > 0 {
			deficit := -free

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
		// Re-apply min/max-height after grow/shrink (percentage re-resolve).
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

	sumH := 0.0
	for _, h := range heights {
		sumH += h
	}

	totalH := sumH + gaps
	startY := cy
	justifyGap := gap

	switch st.JustifyContent {
	case "flex-end", "end":
		if contentH >= 0 {
			startY = cy + (contentH - totalH)
		}
	case "center":
		if contentH >= 0 {
			startY = cy + (contentH-totalH)/two
		}
	case "space-between":
		if len(items) > 1 && contentH >= 0 {
			rem := contentH - sumH
			if rem > 0 {
				justifyGap = rem / float64(len(items)-1)
			}

			startY = cy
		}
	case "space-around":
		if len(items) > 0 && contentH >= 0 {
			rem := contentH - sumH
			if rem < 0 {
				rem = 0
			}

			unit := rem / float64(two*len(items))
			startY = cy + unit
			justifyGap = two * unit
		}
	case "space-evenly":
		if len(items) > 0 && contentH >= 0 {
			rem := contentH - sumH
			if rem < 0 {
				rem = 0
			}

			unit := rem / float64(len(items)+1)
			startY = cy + unit
			justifyGap = unit
		}
	}

	leftY := startY
	endY := cy

	for idx, item := range items {
		cstate := e.styles[item.n]
		origH := cstate.Height
		// Force border-box height so grow/shrink targets stick through build.
		padV := e.scalePt(cstate.PaddingTop) + e.scalePt(cstate.PaddingBottom) +
			e.scalePt(cstate.BorderTop.Width) + e.scalePt(cstate.BorderBottom.Width)
		forceH := heights[idx]

		if e.scale > 0 {
			if cstate.BoxSizing == "border-box" {
				cstate.Height = forceH / e.scale
			} else {
				inner := forceH - padV
				if inner < 0 {
					inner = 0
				}

				cstate.Height = inner / e.scale
			}

			e.styles[item.n] = cstate
		}

		cblock := e.build(item.n, contentW, contentX, y+leftY)
		cstate.Height = origH
		e.styles[item.n] = cstate

		if cblock == nil {
			leftY += heights[idx]
			if idx < len(items)-1 {
				leftY += justifyGap
			}

			continue
		}

		dx := contentX - cblock.x
		dy := (y + leftY) - cblock.y
		e.shiftBoxOps(cblock, dx, dy)
		cblock.x += dx
		cblock.y += dy

		align := st.AlignItems
		if cstate.AlignSelf != "" && cstate.AlignSelf != "auto" {
			align = cstate.AlignSelf
		}

		switch align {
		case "center":
			adx := contentX + (contentW-cblock.w)/2 - cblock.x
			if adx != 0 {
				e.shiftBoxOps(cblock, adx, 0)
				cblock.x += adx
			}
		case "flex-end", "end":
			adx := contentX + contentW - cblock.w - cblock.x
			if adx != 0 {
				e.shiftBoxOps(cblock, adx, 0)
				cblock.x += adx
			}
		}

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

	if contentH >= 0 && endY < cy+contentH {
		return cy + contentH
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
