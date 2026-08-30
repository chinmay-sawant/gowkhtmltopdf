//nolint:all
package layout

import (
	"math"
	"sort"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func (e *engine) markOpsFixed(start, end int) {
	if end < start {
		return
	}

	for i := start; i <= end && i < len(e.ops); i++ {
		e.ops[i].Fixed = true
	}
}

// appendBorderLineOps appends border segment ops into dst (may be nil).
func appendBorderLineOps(
	dst []Op, posX, posY, boxW, boxH, width float64, style string, red, green, blue float64,
) []Op {
	if width <= 0 || style == cssDisplayNone || (boxW <= 0 && boxH <= 0) {
		return dst
	}

	if style != borderStyleDashed && style != borderStyleDotted {
		return append(dst, Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpLine, X: posX, Y: posY, W: boxW, H: boxH, Width: width, R: red, G: green, B: blue,
		})
	}

	return appendDashedLineSegments(dst, posX, posY, boxW, boxH, width, style == borderStyleDotted, red, green, blue)
}

// appendDashedLineSegments expands a dashed/dotted border edge into segment ops.
func appendDashedLineSegments(
	dst []Op, posX, posY, boxW, boxH, width float64, dotted bool, red, green, blue float64,
) []Op {
	horizontal := boxW > 0
	length := boxW

	if !horizontal {
		length = boxH
	}

	drawLen, gap := width*3, width*2 // three=3, two=2 inlined; remaining two/three usages parked
	if dotted {
		drawLen, gap = width, width*dashGapMul
	}

	if drawLen < 0.5 { // halfRatio 0.5 inlined
		drawLen = 0.5
	}

	if gap < 0.5 { // halfRatio 0.5 inlined
		gap = 0.5
	}

	if n := int(length/(drawLen+gap)) + 1; cap(dst)-len(dst) < n {
		// Grow once for the expected segment count.
		grown := make([]Op, len(dst), len(dst)+n)
		copy(grown, dst)
		dst = grown
	}

	for pos := 0.0; pos < length-0.001; pos += drawLen + gap {
		seg := math.Min(drawLen, length-pos)
		if seg <= 0 {
			break
		}

		segX, segY, segW, segH := posX+pos, posY, seg, 0.0
		if !horizontal {
			segX, segY, segW, segH = posX, posY+pos, 0.0, seg
		}

		dst = append(dst, Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpLine, X: segX, Y: segY, W: segW, H: segH,
			Width: width, R: red, G: green, B: blue,
		})
	}

	return dst
}

// emitBorderLine appends one edge's border segments straight onto e.ops —
// no intermediate []Op (hot path for collapsed table grids).
func (e *engine) emitBorderLine(posX, posY, boxW, boxH, width float64, style string, red, green, blue float64) {
	if width <= 0 || style == cssDisplayNone || (boxW <= 0 && boxH <= 0) {
		return
	}

	if style != borderStyleDashed && style != borderStyleDotted {
		e.add(Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpLine, X: posX, Y: posY, W: boxW, H: boxH, Width: width, R: red, G: green, B: blue,
		})

		return
	}

	// Dashed/dotted: append into a tiny stack buffer then emit.
	var buf [8]Op

	segs := appendDashedLineSegments(buf[:0], posX, posY, boxW, boxH, width, style == borderStyleDotted, red, green, blue)
	for i := range segs {
		e.add(segs[i])
	}
}

// borderOps returns the four border line ops for the given border box.
func (e *engine) borderOps(sty ResolvedStyle, posX, posY, wid, height float64) []Op {
	return e.borderOpsSides(sty, posX, posY, wid, height, true, true, true, true)
}

// borderOpsSides emits the requested sides of a border box.
func (e *engine) borderOpsSides(
	sty ResolvedStyle, posX, posY, wid, height float64, top, right, bottom, left bool,
) []Op {
	const borderSideCount = 4

	ops := make([]Op, 0, borderSideCount)
	if top {
		ops = appendBorderLineOps(ops, posX, posY, wid, 0, e.scalePt(borderPaint(sty.BorderTop)), sty.BorderTop.Style,
			sty.BorderTop.Color[0], sty.BorderTop.Color[1], sty.BorderTop.Color[2])
	}

	if right {
		ops = appendBorderLineOps(ops, posX+wid, posY, 0, height,
			e.scalePt(borderPaint(sty.BorderRight)), sty.BorderRight.Style,
			sty.BorderRight.Color[0], sty.BorderRight.Color[1], sty.BorderRight.Color[2])
	}

	if bottom {
		ops = appendBorderLineOps(ops, posX, posY+height, wid, 0,
			e.scalePt(borderPaint(sty.BorderBottom)), sty.BorderBottom.Style,
			sty.BorderBottom.Color[0], sty.BorderBottom.Color[1], sty.BorderBottom.Color[2])
	}

	if left {
		ops = appendBorderLineOps(ops, posX, posY, 0, height, e.scalePt(borderPaint(sty.BorderLeft)), sty.BorderLeft.Style,
			sty.BorderLeft.Color[0], sty.BorderLeft.Color[1], sty.BorderLeft.Color[2])
	}

	return ops
}

func hasVerticalBorder(sty ResolvedStyle) bool {
	left := borderPaint(sty.BorderLeft) > 0 && sty.BorderLeft.Style != cssDisplayNone
	right := borderPaint(sty.BorderRight) > 0 && sty.BorderRight.Style != cssDisplayNone

	return left || right
}

const htmlFigure = "figure"

func isCaptionBox(caption *box) bool {
	if caption == nil || caption.node == nil || caption.style == nil {
		return false
	}

	return caption.node.Name == "figcaption" || caption.style.Display == displayTableCaption
}

func isCollapsedThumbPair(parent *html.Node, parentStyle, captionStyle *ResolvedStyle) bool {
	if parent == nil || parentStyle == nil || captionStyle == nil {
		return false
	}

	isFigure := parent.Name == htmlFigure || parentStyle.Display == displayTable
	if !isFigure || !hasVerticalBorder(*parentStyle) || !hasVerticalBorder(*captionStyle) {
		return false
	}

	if parentStyle.BorderCollapse == borderCollapseValue {
		return true
	}

	return borderPaint(parentStyle.BorderBottom) <= 0 && borderPaint(captionStyle.BorderTop) <= 0
}

// collapsedThumbCaption reports a MediaWiki-style figure/figcaption pair
// whose CSS is display:table + border-collapse (or matching open sides).
// Those sides must paint as one frame, not a second caption box.
func (e *engine) collapsedThumbCaption(caption *box) (*ResolvedStyle, bool) {
	if e == nil || !isCaptionBox(caption) {
		return nil, false
	}

	parent := caption.node.Parent
	if parent == nil {
		return nil, false
	}

	parentStyle := e.styles[parent]
	if !isCollapsedThumbPair(parent, parentStyle, caption.style) {
		return nil, false
	}

	return parentStyle, true
}

// thumbImageInsideFigure reports an <img> whose nearest figure uses the
// collapsed thumb frame. Its left/right/top sit on the figure rails; only
// the bottom separator should paint.
func (e *engine) thumbImageInsideFigure(node *html.Node) bool {
	if e == nil || node == nil || node.Name != cssTagImg {
		return false
	}

	for parent := node.Parent; parent != nil; parent = parent.Parent {
		if parent.Name != htmlFigure {
			continue
		}

		for _, child := range parent.Children {
			if child != nil && child.Name == "figcaption" {
				captionBox := &box{node: child, style: e.styles[child]} //nolint:exhaustruct // style probe only
				_, ok := e.collapsedThumbCaption(captionBox)

				return ok
			}
		}

		return false
	}

	return false
}

// chromeMustSpliceImmediately reports boxes whose chrome must land in e.ops
// during the build: sticky (StickyID stamp), fixed (Fixed stamp), and
// transform (op-range exclusive CTM stamp). Common static/relative boxes defer.
func chromeMustSpliceImmediately(st ResolvedStyle) bool {
	if st.HasTransform {
		return true
	}

	switch st.Position {
	case positionSticky, positionFixed:
		return true
	}

	return false
}

// prependChrome inserts background + border ops at insertAt so they paint
// under any content ops already appended for this box.
//
// Common path defers the splice until finalizeChrome (one linear merge).
// Sticky/fixed/transform keep an immediate splice so mid-build StickyID/Fixed
// stamps and transform exclusive ranges stay correct without re-derivation.
//
//nolint:cyclop,wsl,funlen // paint ordering and border geometry stay together
func (e *engine) prependChrome(insertAt int, boxNode *box, sty ResolvedStyle, posX, posY, width, height float64) {
	if e.noEmit {
		return
	}
	if isNeutralFrameSection(boxNode.node, sty) {
		sty.BorderTop = sty.BorderBottom
	}

	if overflowClipsPaint(sty.Overflow) && insertAt >= 0 && insertAt <= len(e.ops) {
		clipOpsSlice(e.ops[insertAt:], e.paddingBoxRect(posX, posY, width, height, sty))
	}

	var chrome []Op
	radii, radiiY := usedBorderRadiiXY(sty, width, height)
	radius := uniformRadius(radii)
	// Outset box-shadow paints behind the background so blur rings do not
	// cover the border box.
	chrome = e.appendBoxShadow(chrome, sty, posX, posY, width, height, radii, radiiY)
	if sty.BGColor[3] > 0 && e.opts.Background {
		chrome = append(chrome, Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpFillRect, X: posX, Y: posY, W: width, H: height,
			R: sty.BGColor[0], G: sty.BGColor[1], B: sty.BGColor[2], Alpha: sty.BGColor[3], Radius: radius,
			RadiusTopLeft: radii[0], RadiusTopRight: radii[1], RadiusBottomRight: radii[2], RadiusBottomLeft: radii[3],
		})
	}
	chrome = e.appendBackgroundImage(chrome, sty, posX, posY, width, height)

	if sty.BorderImageSource != "" {
		chrome = e.appendBorderImage(chrome, sty, posX, posY, width, height)
	} else {
		switch {
		case hasRoundedRadii(radii) && roundedSolidBorder(sty):
			chrome = append(chrome, e.roundedBorderOps(sty, posX, posY, width, height, radii)...)
		case hasRoundedRadii(radii) && roundedAccentBorder(sty):
			chrome = append(chrome, e.roundedAccentBorderOps(sty, posX, posY, width, height, radii)...)
		default:
			chrome = append(chrome, e.collapsedOrFullBorderOps(boxNode, sty, posX, posY, width, height)...)
		}
	}
	chrome = append(chrome, e.outlineOps(&sty, posX, posY, width, height)...)
	stampOpRadiiY(chrome, radiiY)
	if len(chrome) == 0 {
		return
	}

	for i := range chrome {
		chrome[i].ZIndex = e.zIndex
		chrome[i].ZIndexSet = e.zIndexSet
		chrome[i].Positioned = e.positioned
	}

	if !chromeMustSpliceImmediately(sty) {
		e.deferredChrome = append(e.deferredChrome, chromeEntry{at: insertAt, ops: chrome, b: boxNode})

		return
	}
	// Immediate splice (sticky/fixed/transform).
	tail := append([]Op(nil), e.ops[insertAt:]...)
	e.ops = e.ops[:insertAt]
	e.ops = append(e.ops, chrome...)
	e.ops = append(e.ops, tail...)
	// Keep deferred insert indices valid after this mid-build splice.
	n := len(chrome)

	for i := range e.deferredChrome {
		if e.deferredChrome[i].at >= insertAt {
			e.deferredChrome[i].at += n
		}
	}
}

// collapsedOrFullBorderOps paints a single joined frame for a collapsed
// figure/figcaption thumb: the figure already owns the vertical rails, so the
// caption emits only the closing bottom edge at the figure's border-box width.
func (e *engine) collapsedOrFullBorderOps(
	boxNode *box, sty ResolvedStyle, posX, posY, width, height float64,
) []Op {
	parentStyle, ok := e.collapsedThumbCaption(boxNode)
	if !ok {
		return e.borderOps(sty, posX, posY, width, height)
	}

	padL := e.scalePt(parentStyle.PaddingLeft + parentStyle.BorderLeft.Width)
	padR := e.scalePt(parentStyle.PaddingRight + parentStyle.BorderRight.Width)

	return e.borderOpsSides(sty, posX-padL, posY, width+padL+padR, height, false, false, true, false)
}

// isNeutralFrameSection recognizes a generic frame pattern rather than a
// fixture ID or literal text: a section with a solid, neutral outer border
// keeps its top edge consistent with its bottom edge across page fragments.
//
//nolint:wsl // generic frame checks remain adjacent to their border gates.
func isNeutralFrameSection(node *html.Node, sty ResolvedStyle) bool {
	if node == nil || node.Name != htmlSection {
		return false
	}
	if sty.BorderTop.Style != solidKeyword || sty.BorderBottom.Style != solidKeyword {
		return false
	}

	return nearlyEqual(sty.BorderTop.Width, sty.BorderBottom.Width)
}

func uniformRadius(radii [4]float64) float64 {
	if radii[0] == radii[1] && radii[1] == radii[2] && radii[2] == radii[3] {
		return radii[0]
	}

	return 0
}

func hasRoundedRadii(radii [4]float64) bool {
	for _, radius := range radii {
		if radius > 0 {
			return true
		}
	}

	return false
}

func uniformRoundedBorder(sty ResolvedStyle) bool {
	top := sty.BorderTop

	return top.Width > 0 && top.Style == solidKeyword &&
		top == sty.BorderRight && top == sty.BorderBottom && top == sty.BorderLeft
}

func roundedSolidBorder(sty ResolvedStyle) bool {
	return sty.BorderTop.Width > 0 && sty.BorderRight.Width > 0 &&
		sty.BorderBottom.Width > 0 && sty.BorderLeft.Width > 0 &&
		sty.BorderTop.Style == solidKeyword && sty.BorderRight.Style == solidKeyword &&
		sty.BorderBottom.Style == solidKeyword && sty.BorderLeft.Style == solidKeyword
}

func roundedAccentBorder(sty ResolvedStyle) bool {
	return (sty.BorderTop.Width > 0 && sty.BorderTop.Style == solidKeyword) ||
		(sty.BorderRight.Width > 0 && sty.BorderRight.Style == solidKeyword) ||
		(sty.BorderBottom.Width > 0 && sty.BorderBottom.Style == solidKeyword) ||
		(sty.BorderLeft.Width > 0 && sty.BorderLeft.Style == solidKeyword)
}

func squareSideBorderRadii(side border, radii [4]float64, left bool) [4]float64 {
	const radiusWidthSlack = 0.25
	if left {
		if radii[0] <= borderPaint(side)+radiusWidthSlack && radii[3] <= borderPaint(side)+radiusWidthSlack {
			return [4]float64{}
		}

		return radii
	}

	if radii[1] <= borderPaint(side)+radiusWidthSlack && radii[2] <= borderPaint(side)+radiusWidthSlack {
		return [4]float64{}
	}

	return radii
}

// roundedBorderOps keeps the rounded outer geometry for a solid border whose
// sides differ in width or color. The base stroke supplies the corner arcs;
// differing sides are overlaid as masked OpStrokeRect sides so accent rails
// keep corner arcs without a second imageout OpLine rewrite.
func (e *engine) roundedBorderOps(
	sty ResolvedStyle, posX, posY, width, height float64, radii [4]float64,
) []Op {
	// Use the bottom side as the base geometry. CSS frequently uses an
	// accented top rail (for example the architecture pipeline cards); using
	// BorderTop here would spread that accent across the rounded stroke on all
	// four sides before the per-side overlays are painted.
	base := sty.BorderBottom
	ops := []Op{{ //nolint:exhaustruct // intentional zero fields
		Kind: OpStrokeRect, X: posX, Y: posY, W: width, H: height,
		R: base.Color[0], G: base.Color[1], B: base.Color[2], Width: e.scalePt(borderPaint(base)),
		Radius: uniformRadius(radii), RadiusTopLeft: radii[0], RadiusTopRight: radii[1],
		RadiusBottomRight: radii[2], RadiusBottomLeft: radii[3],
	}}

	type mixedSide struct {
		border border
		mask   uint8
	}

	sides := []mixedSide{
		{border: sty.BorderTop, mask: StrokeMaskTop},
		{border: sty.BorderRight, mask: StrokeMaskRight},
		{border: sty.BorderBottom, mask: StrokeMaskBottom},
		{border: sty.BorderLeft, mask: StrokeMaskLeft},
	}

	for _, side := range sides {
		if side.border == base || side.border.Style != solidKeyword {
			continue
		}

		sideRadii := radii
		if side.mask == StrokeMaskLeft || side.mask == StrokeMaskRight {
			sideRadii = squareSideBorderRadii(side.border, radii, side.mask == StrokeMaskLeft)
		}

		ops = append(ops, Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpStrokeRect, X: posX, Y: posY, W: width, H: height,
			R: side.border.Color[0], G: side.border.Color[1], B: side.border.Color[2],
			Width: e.scalePt(borderPaint(side.border)), Radius: uniformRadius(sideRadii),
			RadiusTopLeft: sideRadii[0], RadiusTopRight: sideRadii[1],
			RadiusBottomRight: sideRadii[2], RadiusBottomLeft: sideRadii[3],
			StrokeMask: side.mask,
		})
	}

	return ops
}

// roundedAccentBorderOps keeps a solid accent rail curved when the remaining
// border sides are absent or use dotted/dashed styles.
// Solid remnant sides use StrokeMask* overlays; dashed/dotted sides stay as
// segmented OpLines that stop at the corner radii.
func (e *engine) roundedAccentBorderOps(
	sty ResolvedStyle,
	posX, posY, width, height float64,
	radii [4]float64,
) []Op {
	var ops []Op

	appendSolidMask := func(side border, mask uint8, sideRadii [4]float64) {
		if borderPaint(side) <= 0 || side.Style != solidKeyword {
			return
		}

		ops = append(ops, Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpStrokeRect, X: posX, Y: posY, W: width, H: height,
			R: side.Color[0], G: side.Color[1], B: side.Color[2],
			Width: e.scalePt(borderPaint(side)), Radius: uniformRadius(sideRadii),
			RadiusTopLeft: sideRadii[0], RadiusTopRight: sideRadii[1],
			RadiusBottomRight: sideRadii[2], RadiusBottomLeft: sideRadii[3],
			StrokeMask: mask,
		})
	}

	appendDashedSide := func(sideX, sideY, sideW, sideH float64, side border) {
		if borderPaint(side) <= 0 || side.Style == solidKeyword {
			return
		}

		ops = appendBorderLineOps(
			ops, sideX, sideY, sideW, sideH,
			e.scalePt(borderPaint(side)), side.Style,
			side.Color[0], side.Color[1], side.Color[2],
		)
	}

	rightRadii := squareSideBorderRadii(sty.BorderRight, radii, false)
	leftRadii := squareSideBorderRadii(sty.BorderLeft, radii, true)
	appendSolidMask(sty.BorderTop, StrokeMaskTop, radii)
	appendSolidMask(sty.BorderRight, StrokeMaskRight, rightRadii)
	appendSolidMask(sty.BorderBottom, StrokeMaskBottom, radii)
	appendSolidMask(sty.BorderLeft, StrokeMaskLeft, leftRadii)

	appendDashedSide(posX+radii[0], posY, math.Max(width-radii[0]-radii[1], 0), 0, sty.BorderTop)
	appendDashedSide(posX+width, posY+radii[1], 0, math.Max(height-radii[1]-radii[2], 0), sty.BorderRight)
	appendDashedSide(posX+radii[3], posY+height, math.Max(width-radii[3]-radii[2], 0), 0, sty.BorderBottom)
	appendDashedSide(posX, posY+radii[0], 0, math.Max(height-radii[0]-radii[3], 0), sty.BorderLeft)

	return ops
}

// finalizeChrome merges deferred background/border ops into e.ops in one
// linear pass and reindexes box op ranges. Paint order for multiple entries
// at the same index matches immediate-splice nesting: later (outer) entries
// paint first.
func (e *engine) finalizeChrome(root *box) {
	if len(e.deferredChrome) > 0 {
		entries := e.deferredChrome
		e.deferredChrome = nil

		out, oldToNew, ownerChrome := mergeDeferredChrome(e.ops, entries)

		e.ops = out

		// Remap content op ranges, expand owners with their chrome, then union
		// parent ranges over children so ancestor ranges still cover nested chrome.
		remapBoxRangesWithChrome(root, oldToNew, ownerChrome)
		unionChildOpRanges(root)

		// Deferred chrome under sticky ancestors never received StickyID at build
		// time; re-stamp from the box tree. Fixed content already marked Fixed -
		// expand Fixed onto chrome in the same range when any op is Fixed.
		restampStickyFixed(root, e.ops)
	}

	e.applyOverflowClips(root)
}

// chromeSpan is an inclusive op range owned by one box's chrome.
type chromeSpan struct{ start, end int }

// mergeDeferredChrome splices deferred background/border ops into oldOps in
// one linear pass. Paint order for multiple entries at the same index matches
// immediate-splice nesting: later (outer) entries paint first.
func mergeDeferredChrome(
	oldOps []Op, entries []chromeEntry,
) ([]Op, []int, map[*box]chromeSpan) {
	// Sort by insert index ascending; same index → reverse registration order
	// (parent registered after child, paints under content first).
	type indexed struct {
		ord int
		ent chromeEntry
	}

	order := make([]indexed, len(entries))
	for i, ent := range entries {
		order[i] = indexed{ord: i, ent: ent}
	}

	sort.SliceStable(order, func(i, j int) bool {
		if order[i].ent.at != order[j].ent.at {
			return order[i].ent.at < order[j].ent.at
		}
		// Higher ord (later register) first within the same at.
		return order[i].ord > order[j].ord
	})

	totalChrome := 0
	for _, it := range order {
		totalChrome += len(it.ent.ops)
	}

	out := make([]Op, 0, len(oldOps)+totalChrome)
	oldToNew := make([]int, len(oldOps))
	ownerChrome := map[*box]chromeSpan{}

	oidx := 0
	for idx, paintOp := range oldOps {
		for oidx < len(order) && order[oidx].ent.at == idx {
			ent := order[oidx].ent
			cs := len(out)
			out = append(out, ent.ops...)
			recordOwnerChrome(ownerChrome, ent.b, cs, len(out)-1)

			oidx++
		}

		oldToNew[idx] = len(out)
		out = append(out, paintOp)
	}
	// Trailing chrome (chrome-only boxes with insertAt == len(ops)).
	for oidx < len(order) {
		ent := order[oidx].ent
		cs := len(out)
		out = append(out, ent.ops...)
		recordOwnerChrome(ownerChrome, ent.b, cs, len(out)-1)

		oidx++
	}

	return out, oldToNew, ownerChrome
}

// recordOwnerChrome widens the chrome op span recorded for a box so a later
// (outer) entry keeps the owner's range covering all of its chrome.
func recordOwnerChrome(ownerChrome map[*box]chromeSpan, boxNode *box, start, endIdx int) {
	if boxNode == nil || endIdx < start {
		return
	}

	if prev, ok := ownerChrome[boxNode]; ok {
		if start < prev.start {
			prev.start = start
		}

		if endIdx > prev.end {
			prev.end = endIdx
		}

		ownerChrome[boxNode] = prev

		return
	}

	ownerChrome[boxNode] = chromeSpan{start: start, end: endIdx}
}

// remapBoxRangesWithChrome rewrites box op ranges through the old→new index
// map, then expands each owner with its recorded chrome span.
func remapBoxRangesWithChrome(root *box, oldToNew []int, ownerChrome map[*box]chromeSpan) {
	var remap func(b *box)
	remap = func(boxNode *box) {
		if boxNode == nil {
			return
		}

		if boxNode.opEnd >= boxNode.opStart && boxNode.opStart >= 0 && boxNode.opEnd < len(oldToNew) {
			boxNode.opStart = oldToNew[boxNode.opStart]
			boxNode.opEnd = oldToNew[boxNode.opEnd]
		}

		if span, ok := ownerChrome[boxNode]; ok {
			mergeOwnerChromeSpan(boxNode, span)
		}

		for _, child := range boxNode.children {
			remap(child)
		}
	}
	remap(root)
}

// mergeOwnerChromeSpan unions a box's content range with its chrome span.
func mergeOwnerChromeSpan(boxNode *box, span chromeSpan) {
	if boxNode.opEnd < boxNode.opStart {
		boxNode.opStart, boxNode.opEnd = span.start, span.end

		return
	}

	if span.start < boxNode.opStart {
		boxNode.opStart = span.start
	}

	if span.end > boxNode.opEnd {
		boxNode.opEnd = span.end
	}
}

// unionChildOpRanges widens every box range over its children's ranges so
// ancestor ranges still cover nested chrome.
func unionChildOpRanges(root *box) {
	var unionChildren func(b *box)
	unionChildren = func(boxNode *box) {
		if boxNode == nil {
			return
		}

		for _, child := range boxNode.children {
			unionChildren(child)

			if child.opEnd < child.opStart {
				continue
			}

			if boxNode.opEnd < boxNode.opStart {
				boxNode.opStart, boxNode.opEnd = child.opStart, child.opEnd

				continue
			}

			if child.opStart < boxNode.opStart {
				boxNode.opStart = child.opStart
			}

			if child.opEnd > boxNode.opEnd {
				boxNode.opEnd = child.opEnd
			}
		}
	}
	unionChildren(root)
}

// restampStickyFixed re-applies StickyID from sticky boxes and expands Fixed
// onto the full op range when the box was viewport-fixed (any op already Fixed).
func restampStickyFixed(boxNode *box, ops []Op) {
	if boxNode == nil {
		return
	}

	reapplyStickyID(boxNode, ops)
	expandFixedOps(boxNode, ops)

	for _, c := range boxNode.children {
		restampStickyFixed(c, ops)
	}
}

// reapplyStickyID stamps StickyID onto a sticky box's whole op range.
func reapplyStickyID(boxNode *box, ops []Op) {
	if !boxNode.sticky || boxNode.stickyID == 0 || boxNode.opEnd < boxNode.opStart || boxNode.opStart < 0 {
		return
	}

	for i := boxNode.opStart; i <= boxNode.opEnd && i < len(ops); i++ {
		ops[i].StickyID = boxNode.stickyID
	}
}

// expandFixedOps spreads the Fixed mark over a viewport-fixed box's op range
// when any op in it is already Fixed (chrome added after build is included).
func expandFixedOps(boxNode *box, ops []Op) {
	if boxNode.style.Position != positionFixed || boxNode.opEnd < boxNode.opStart || boxNode.opStart < 0 {
		return
	}

	hasFixed := false

	for i := boxNode.opStart; i <= boxNode.opEnd && i < len(ops); i++ {
		if ops[i].Fixed {
			hasFixed = true

			break
		}
	}

	if !hasFixed {
		return
	}

	for i := boxNode.opStart; i <= boxNode.opEnd && i < len(ops); i++ {
		ops[i].Fixed = true
	}
}

// contentBox returns the content-box origin and width for a border box.
// Single home for "content = border-box − scaled padding − scaled border".
