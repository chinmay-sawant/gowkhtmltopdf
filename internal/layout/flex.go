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

// flexGaps returns scaled row/column gaps. When both longhands are unset (0),
// fall back to the Gap shorthand for both axes.
func (e *engine) flexGaps(st ResolvedStyle) (rowGap, colGap float64) {
	if st.RowGap == 0 && st.ColumnGap == 0 {
		g := e.scalePt(st.Gap)
		return g, g
	}
	return e.scalePt(st.RowGap), e.scalePt(st.ColumnGap)
}

// flexContentHeight returns the definite content-box height of a flex
// container, or -1 when height is auto. HeightPercent against an indefinite
// containing block is treated as auto (cyclic % honesty).
func (e *engine) flexContentHeight(st ResolvedStyle) float64 {
	if st.HeightPercent >= 0 && st.Height < 0 {
		return -1
	}
	if st.Height < 0 {
		return -1
	}
	h := e.scalePt(st.Height)
	if st.BoxSizing == "border-box" {
		h -= e.scalePt(st.PaddingTop) + e.scalePt(st.PaddingBottom) +
			e.scalePt(st.BorderTop.Width) + e.scalePt(st.BorderBottom.Width)
	}
	if h < 0 {
		h = 0
	}
	return h
}

// buildFlex lays out a flex container (row or column) with a report-friendly
// subset: justify-content, align-items/self, align-content, gap/row-gap/
// column-gap, flex-grow/shrink/basis, order, wrap, and reverse directions.
func (e *engine) buildFlex(n *html.Node, st ResolvedStyle, availW, x, y float64) *box {
	ml, mr := e.scalePt(st.MarginLeft), e.scalePt(st.MarginRight)
	b := &box{node: n, style: st, kind: "block", x: x + ml, y: y}
	b.w = availW - ml - mr
	if b.w < 0 {
		b.w = 0
	}
	if st.WidthPercent >= 0 {
		// Cyclic % honesty: indefinite availW → keep fill-remaining (auto).
		if availW > 0 && availW < 1e12 {
			b.w = availW * st.WidthPercent / 100
		}
	} else if st.Width >= 0 {
		b.w = e.scalePt(st.Width)
		if st.BoxSizing != "border-box" {
			b.w += e.scalePt(st.PaddingLeft) + e.scalePt(st.PaddingRight) +
				e.scalePt(st.BorderLeft.Width) + e.scalePt(st.BorderRight.Width)
		}
	}
	contentW := b.w - e.scalePt(st.PaddingLeft) - e.scalePt(st.PaddingRight) -
		e.scalePt(st.BorderLeft.Width) - e.scalePt(st.BorderRight.Width)
	if contentW < 0 {
		contentW = 0
	}
	contentX := b.x + e.scalePt(st.BorderLeft.Width) + e.scalePt(st.PaddingLeft)
	contentStart := len(e.ops)
	cy := e.scalePt(st.PaddingTop) + e.scalePt(st.BorderTop.Width)

	var kids []*html.Node
	for _, c := range n.Children {
		if c.Type != html.ElementNode {
			continue
		}
		cs := e.styles[c]
		if cs.Display == "none" {
			continue
		}
		kids = append(kids, c)
	}
	rowGap, colGap := e.flexGaps(st)
	dir := st.FlexDirection
	if dir == "" {
		dir = "row"
	}

	if dir == "column" || dir == "column-reverse" {
		cy = e.flowFlexColumn(b, kids, st, contentW, contentX, y, cy, rowGap)
	} else {
		cy = e.flowFlexRow(b, kids, st, contentW, contentX, y, cy, colGap, rowGap)
	}
	cy += e.scalePt(st.PaddingBottom)
	if st.Height >= 0 {
		h := e.scalePt(st.Height)
		if st.BoxSizing != "border-box" {
			h += e.scalePt(st.PaddingTop) + e.scalePt(st.PaddingBottom) +
				e.scalePt(st.BorderTop.Width) + e.scalePt(st.BorderBottom.Width)
		}
		if cy < h {
			cy = h
		}
	}
	b.h = cy
	e.prependChrome(contentStart, st, b.x, y, b.w, b.h)
	return b
}

func (e *engine) flowFlexRow(parent *box, kids []*html.Node, st ResolvedStyle, contentW, contentX, y, cy, colGap, rowGap float64) float64 {
	if len(kids) == 0 {
		return cy
	}
	wrap := st.FlexWrap == "wrap" || st.FlexWrap == "wrap-reverse"
	reverse := st.FlexDirection == "row-reverse"

	items := make([]flexMeas, 0, len(kids))
	for _, kid := range kids {
		cs := e.styles[kid]
		g := cs.FlexGrow
		if g < 0 {
			g = 0
		}
		sh := cs.FlexShrink
		if sh < 0 {
			sh = 1
		}
		items = append(items, flexMeas{
			n: kid, baseW: e.flexItemBaseWidth(kid, cs, contentW),
			grow: g, shrink: sh, order: cs.FlexOrder,
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
		for _, it := range items {
			need := it.baseW
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
				need = it.baseW
			}
			line = append(line, it)
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
		lineCross = e.flexContentHeight(st)
	}
	placed := make([]linePlace, 0, len(lines))
	for li, line := range lines {
		startChild := 0
		if parent != nil {
			startChild = len(parent.children)
		}
		y0 := cy
		cy = e.placeFlexLineMeasured(parent, st, line, contentW, contentX, y, cy, colGap, lineCross)
		endChild := startChild
		if parent != nil {
			endChild = len(parent.children)
		}
		placed = append(placed, linePlace{startChild: startChild, endChild: endChild, y0: y0, h: cy - y0})
		if li < len(lines)-1 {
			cy += rowGap
		}
	}

	// align-content: distribute free cross space when height is definite and
	// wrapping produced multiple lines. Height:auto → pack at start (no-op).
	contentH := e.flexContentHeight(st)
	if contentH >= 0 && len(placed) > 1 {
		linesH := 0.0
		for _, lp := range placed {
			linesH += lp.h
		}
		gapsH := rowGap * float64(len(placed)-1)
		free := contentH - linesH - gapsH
		if free > 1e-6 {
			offsets := make([]float64, len(placed))
			switch st.AlignContent {
			case "flex-end", "end":
				for i := range offsets {
					offsets[i] = free
				}
			case "center":
				for i := range offsets {
					offsets[i] = free / 2
				}
			case "space-between":
				step := free / float64(len(placed)-1)
				for i := range offsets {
					offsets[i] = step * float64(i)
				}
			case "space-around":
				unit := free / float64(2*len(placed))
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
				for i, lp := range placed {
					dy := offsets[i]
					if dy == 0 {
						continue
					}
					for ci := lp.startChild; ci < lp.endChild && ci < len(parent.children); ci++ {
						cb := parent.children[ci]
						e.shiftBoxOps(cb, 0, dy)
						cb.y += dy
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
			w := mainSize * cs.FlexBasisPercent / 100
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
		w := mainSize * cs.WidthPercent / 100
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
		intr = pad + e.scalePt(cs.FontSize)*2
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
	cs := e.styles[it.n]
	floor := 0.0
	if cs.MinWidthPercent >= 0 && mainSize >= 0 {
		floor = mainSize * cs.MinWidthPercent / 100
	} else if cs.MinWidth > 0 {
		floor = e.scalePt(cs.MinWidth)
	}
	// Automatic minimum (min-width:auto): content size suggestion.
	intr := e.measureCellContent(it.n, cs)
	pad := e.scalePt(cs.PaddingLeft) + e.scalePt(cs.PaddingRight) +
		e.scalePt(cs.BorderLeft.Width) + e.scalePt(cs.BorderRight.Width)
	contentSug := intr + pad
	// Specified size suggestion when width/% is definite against mainSize.
	specSug := -1.0
	if cs.WidthPercent >= 0 && mainSize >= 0 {
		specSug = mainSize * cs.WidthPercent / 100
		if cs.BoxSizing != "border-box" {
			specSug += pad
		}
	} else if cs.Width >= 0 {
		specSug = e.scalePt(cs.Width)
		if cs.BoxSizing != "border-box" {
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
	if overflowCreatesStickyScrollport(cs.Overflow) {
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
	for i, it := range items {
		cs := e.styles[it.n]
		floor := e.flexMinMainSize(it, mainSize)
		if widths[i] < floor {
			widths[i] = floor
		}
		if cs.MaxWidth >= 0 {
			mx := e.scalePt(cs.MaxWidth)
			if widths[i] > mx {
				widths[i] = mx
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
		if shrinkable <= 1e-6 {
			break
		}
		step := deficit
		if step > shrinkable {
			step = shrinkable
		}
		for i, it := range items {
			floor := e.flexMinMainSize(it, mainSize)
			room := widths[i] - floor
			if room <= 1e-6 || it.shrink <= 0 {
				continue
			}
			cut := step * (room / shrinkable)
			widths[i] -= cut
			if widths[i] < floor {
				widths[i] = floor
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
		for i, it := range items {
			if it.shrink <= 0 || it.baseW <= 0 {
				continue
			}
			share := (it.shrink * it.baseW) / shrinkSum
			widths[i] -= deficit * share
			floor := e.flexMinMainSize(it, contentW)
			if widths[i] < floor {
				widths[i] = floor
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
		startX = contentX + (contentW-totalW)/2
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
			unit := rem / float64(2*len(items))
			startX = contentX + unit
			justifyGap = 2 * unit
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
	lx := startX

	// Cross-size target for align-items:stretch. Definite container height
	// (lineCross) is the flex line cross size; otherwise measure content max.
	targetCross := lineCross
	if targetCross < 0 {
		was := e.noEmit
		e.noEmit = true
		maxH := 0.0
		mx := startX
		for i, it := range items {
			cb := e.build(it.n, widths[i], mx, y+cy)
			if cb != nil && cb.h > maxH {
				maxH = cb.h
			}
			mx += widths[i]
			if i < len(items)-1 {
				mx += justifyGap
			}
		}
		e.noEmit = was
		targetCross = maxH
	}

	for i, it := range items {
		cs := e.styles[it.n]
		origH := cs.Height
		forceStretch := flexItemCrossStretch(st, cs) && targetCross > 0
		if forceStretch {
			// Used cross size = line cross size (border box), matching column
			// main-size forcing so backgrounds fill the flex line (fixture-33).
			padV := e.scalePt(cs.PaddingTop) + e.scalePt(cs.PaddingBottom) +
				e.scalePt(cs.BorderTop.Width) + e.scalePt(cs.BorderBottom.Width)
			forceH := targetCross
			if e.scale > 0 {
				if cs.BoxSizing == "border-box" {
					cs.Height = forceH / e.scale
				} else {
					inner := forceH - padV
					if inner < 0 {
						inner = 0
					}
					cs.Height = inner / e.scale
				}
				e.styles[it.n] = cs
			}
		}
		cb := e.build(it.n, widths[i], lx, y+cy)
		if forceStretch {
			cs.Height = origH
			e.styles[it.n] = cs
		}
		if cb == nil {
			built = append(built, placed{n: it.n})
			lx += widths[i]
			if i < len(items)-1 {
				lx += justifyGap
			}
			continue
		}
		dx := lx - cb.x
		dy := (y + cy) - cb.y
		e.shiftBoxOps(cb, dx, dy)
		cb.x += dx
		cb.y += dy
		if cb.h > rowH {
			rowH = cb.h
		}
		built = append(built, placed{box: cb, h: cb.h, n: it.n})
		if parent != nil {
			parent.children = append(parent.children, cb)
		}
		lx += widths[i]
		if i < len(items)-1 {
			lx += justifyGap
		}
	}
	alignH := rowH
	if lineCross >= 0 && lineCross > alignH {
		alignH = lineCross
	}
	if targetCross > alignH {
		alignH = targetCross
	}
	for _, p := range built {
		if p.box == nil {
			continue
		}
		align := st.AlignItems
		if cs, ok := e.styles[p.n]; ok && cs.AlignSelf != "" && cs.AlignSelf != "auto" {
			align = cs.AlignSelf
		}
		dy := 0.0
		switch align {
		case "flex-end", "end":
			dy = (y + cy + alignH) - (p.box.y + p.box.h)
		case "center":
			dy = (y + cy + (alignH-p.box.h)/2) - p.box.y
		default:
			// stretch / flex-start / start: pack at cross-start (stretch already sized)
			dy = 0
		}
		if dy != 0 {
			e.shiftBoxOps(p.box, 0, dy)
			p.box.y += dy
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
func flexItemCrossStretch(st, cs ResolvedStyle) bool {
	align := st.AlignItems
	if align == "" {
		align = "stretch"
	}
	if cs.AlignSelf != "" && cs.AlignSelf != "auto" {
		align = cs.AlignSelf
	}
	switch align {
	case "flex-start", "start", "flex-end", "end", "center":
		return false
	}
	// Definite height/% means the used cross size is already specified.
	if cs.Height >= 0 || cs.HeightPercent >= 0 {
		return false
	}
	return true
}

// flexItemBaseHeight resolves the flex base size on the column main axis.
// mainSize is the flex container content-box height from flexContentHeight
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
			h := mainSize * cs.FlexBasisPercent / 100
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
		h := mainSize * cs.HeightPercent / 100
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
	h := e.layoutCell(n, cs, contentW)
	e.ops = e.ops[:start]
	if h <= 0 {
		h = padV + e.scalePt(cs.FontSize)*1.2
	}
	return h
}

// flexMinCrossMainSize is the column-axis content-based min-height floor
// (Flexbox §4.5 lite). mainSize is the definite flex container content height.
func (e *engine) flexMinCrossMainSize(n *html.Node, baseH, mainSize float64) float64 {
	cs := e.styles[n]
	floor := 0.0
	if cs.MinHeightPercent >= 0 && mainSize >= 0 {
		floor = mainSize * cs.MinHeightPercent / 100
	} else if cs.MinHeight > 0 {
		floor = e.scalePt(cs.MinHeight)
	}
	if overflowCreatesStickyScrollport(cs.Overflow) {
		return floor
	}
	padV := e.scalePt(cs.PaddingTop) + e.scalePt(cs.PaddingBottom) +
		e.scalePt(cs.BorderTop.Width) + e.scalePt(cs.BorderBottom.Width)
	start := len(e.ops)
	contentSug := e.layoutCell(n, cs, 1e9)
	e.ops = e.ops[:start]
	if contentSug < padV {
		contentSug = padV + e.scalePt(cs.FontSize)*1.2
	}
	specSug := -1.0
	if cs.HeightPercent >= 0 && mainSize >= 0 {
		specSug = mainSize * cs.HeightPercent / 100
		if cs.BoxSizing != "border-box" {
			specSug += padV
		}
	} else if cs.Height >= 0 {
		specSug = e.scalePt(cs.Height)
		if cs.BoxSizing != "border-box" {
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
	contentH := e.flexContentHeight(st)
	items := make([]colMeas, 0, len(kids))
	for _, kid := range kids {
		cs := e.styles[kid]
		g := cs.FlexGrow
		if g < 0 {
			g = 0
		}
		sh := cs.FlexShrink
		if sh < 0 {
			sh = 1
		}
		items = append(items, colMeas{
			n: kid, baseH: e.flexItemBaseHeight(kid, cs, contentW, contentH),
			grow: g, shrink: sh,
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
			for i, it := range items {
				if it.shrink <= 0 || it.baseH <= 0 {
					continue
				}
				share := (it.shrink * it.baseH) / shrinkSum
				heights[i] -= deficit * share
				floor := e.flexMinCrossMainSize(it.n, it.baseH, contentH)
				if heights[i] < floor {
					heights[i] = floor
				}
			}
		}
		// Re-apply min/max-height after grow/shrink (percentage re-resolve).
		for i, it := range items {
			cs := e.styles[it.n]
			floor := e.flexMinCrossMainSize(it.n, it.baseH, contentH)
			if heights[i] < floor {
				heights[i] = floor
			}
			if cs.MaxHeight >= 0 {
				mx := e.scalePt(cs.MaxHeight)
				if heights[i] > mx {
					heights[i] = mx
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
			startY = cy + (contentH-totalH)/2
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
			unit := rem / float64(2*len(items))
			startY = cy + unit
			justifyGap = 2 * unit
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

	ly := startY
	endY := cy
	for i, it := range items {
		cs := e.styles[it.n]
		origH := cs.Height
		// Force border-box height so grow/shrink targets stick through build.
		padV := e.scalePt(cs.PaddingTop) + e.scalePt(cs.PaddingBottom) +
			e.scalePt(cs.BorderTop.Width) + e.scalePt(cs.BorderBottom.Width)
		forceH := heights[i]
		if e.scale > 0 {
			if cs.BoxSizing == "border-box" {
				cs.Height = forceH / e.scale
			} else {
				inner := forceH - padV
				if inner < 0 {
					inner = 0
				}
				cs.Height = inner / e.scale
			}
			e.styles[it.n] = cs
		}
		cb := e.build(it.n, contentW, contentX, y+ly)
		cs.Height = origH
		e.styles[it.n] = cs
		if cb == nil {
			ly += heights[i]
			if i < len(items)-1 {
				ly += justifyGap
			}
			continue
		}
		dx := contentX - cb.x
		dy := (y + ly) - cb.y
		e.shiftBoxOps(cb, dx, dy)
		cb.x += dx
		cb.y += dy

		align := st.AlignItems
		if cs.AlignSelf != "" && cs.AlignSelf != "auto" {
			align = cs.AlignSelf
		}
		switch align {
		case "center":
			adx := contentX + (contentW-cb.w)/2 - cb.x
			if adx != 0 {
				e.shiftBoxOps(cb, adx, 0)
				cb.x += adx
			}
		case "flex-end", "end":
			adx := contentX + contentW - cb.w - cb.x
			if adx != 0 {
				e.shiftBoxOps(cb, adx, 0)
				cb.x += adx
			}
		}

		if parent != nil {
			parent.children = append(parent.children, cb)
		}
		ly += heights[i]
		endY = ly
		if i < len(items)-1 {
			ly += justifyGap
			endY = ly
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
func (e *engine) applyRelativeOffset(b *box) {
	if b == nil || b.style.Position != "relative" {
		return
	}
	st := b.style
	dx, dy := 0.0, 0.0
	if !st.LeftAuto {
		dx = e.scalePt(st.Left)
	} else if !st.RightAuto {
		dx = -e.scalePt(st.Right)
	}
	if !st.TopAuto {
		dy = e.scalePt(st.Top)
	} else if !st.BottomAuto {
		dy = -e.scalePt(st.Bottom)
	}
	if dx == 0 && dy == 0 {
		return
	}
	b.x += dx
	b.y += dy
	e.shiftBoxOps(b, dx, dy)
}
