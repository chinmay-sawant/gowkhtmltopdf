package layout

import (
	"math"
	"sort"
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

	drawLen, gap := width*three, width*two
	if dotted {
		drawLen, gap = width, width*dashGapMul
	}

	if drawLen < halfRatio {
		drawLen = 0.5
	}

	if gap < halfRatio {
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
	const borderSideCount = 4

	ops := make([]Op, 0, borderSideCount)

	ops = appendBorderLineOps(ops, posX, posY, wid, 0, e.scalePt(sty.BorderTop.Width), sty.BorderTop.Style,
		sty.BorderTop.Color[0], sty.BorderTop.Color[1], sty.BorderTop.Color[2])
	ops = appendBorderLineOps(ops, posX+wid, posY, 0, height, e.scalePt(sty.BorderRight.Width), sty.BorderRight.Style,
		sty.BorderRight.Color[0], sty.BorderRight.Color[1], sty.BorderRight.Color[2])
	ops = appendBorderLineOps(ops, posX, posY+height, wid, 0, e.scalePt(sty.BorderBottom.Width), sty.BorderBottom.Style,
		sty.BorderBottom.Color[0], sty.BorderBottom.Color[1], sty.BorderBottom.Color[2])
	ops = appendBorderLineOps(ops, posX, posY, 0, height, e.scalePt(sty.BorderLeft.Width), sty.BorderLeft.Style,
		sty.BorderLeft.Color[0], sty.BorderLeft.Color[1], sty.BorderLeft.Color[2])

	return ops
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
//nolint:cyclop,wsl // paint ordering and border geometry stay together
func (e *engine) prependChrome(insertAt int, boxNode *box, sty ResolvedStyle, posX, posY, width, height float64) {
	if e.noEmit {
		return
	}

	var chrome []Op
	radius := usedBorderRadius(sty, width, height)
	if sty.BGColor[3] > 0 && e.opts.Background {
		chrome = append(chrome, Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpFillRect, X: posX, Y: posY, W: width, H: height,
			R: sty.BGColor[0], G: sty.BGColor[1], B: sty.BGColor[2], Alpha: sty.BGColor[3], Radius: radius,
		})
	}

	if radius > 0 && uniformRoundedBorder(sty) {
		border := sty.BorderTop
		chrome = append(chrome, Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpStrokeRect, X: posX, Y: posY, W: width, H: height,
			R: border.Color[0], G: border.Color[1], B: border.Color[2], Width: e.scalePt(border.Width), Radius: radius,
		})
	} else {
		chrome = append(chrome, e.borderOps(sty, posX, posY, width, height)...)
	}
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

//nolint:wsl,mnd // border radius rules are direct CSS geometry
func usedBorderRadius(sty ResolvedStyle, width, height float64) float64 {
	radius := sty.BorderRadius
	if sty.BorderRadiusPercent >= 0 {
		short := width
		if height < short {
			short = height
		}
		radius = short * sty.BorderRadiusPercent / 100
	}

	if radius < 0 {
		return 0
	}

	if maxRadius := math.Min(width, height) / 2; radius > maxRadius {
		return maxRadius
	}

	return radius
}

//nolint:goconst // border style is a direct CSS keyword comparison
func uniformRoundedBorder(sty ResolvedStyle) bool {
	top := sty.BorderTop

	return top.Width > 0 && top.Style == "solid" &&
		top == sty.BorderRight && top == sty.BorderBottom && top == sty.BorderLeft
}

// finalizeChrome merges deferred background/border ops into e.ops in one
// linear pass and reindexes box op ranges. Paint order for multiple entries
// at the same index matches immediate-splice nesting: later (outer) entries
// paint first.
func (e *engine) finalizeChrome(root *box) {
	if len(e.deferredChrome) == 0 {
		return
	}

	entries := e.deferredChrome
	e.deferredChrome = nil

	out, oldToNew, ownerChrome := mergeDeferredChrome(e.ops, entries)

	e.ops = out

	// Remap content op ranges, expand owners with their chrome, then union
	// parent ranges over children so ancestor ranges still cover nested chrome.
	remapBoxRangesWithChrome(root, oldToNew, ownerChrome)
	unionChildOpRanges(root)

	// Deferred chrome under sticky ancestors never received StickyID at build
	// time; re-stamp from the box tree. Fixed content already marked Fixed —
	// expand Fixed onto chrome in the same range when any op is Fixed.
	restampStickyFixed(root, e.ops)
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
