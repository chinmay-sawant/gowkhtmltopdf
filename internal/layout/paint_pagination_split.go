//nolint:all
package layout

func isSplittable(op *Op) bool {
	return op.Kind == OpFillRect || op.Kind == OpStrokeRect || op.Kind == OpLine
}

// splitCrossingRects truncates splittable ops at each page boundary and keeps
// remainders immediately after (document paint order). Built into a new slice
// rather than mid-slice insert+copy (O(n2) and float-edge infinite loops when
// Y sits on a page boundary - TestTenPageTableReportPerformance hang).
type opSpan struct{ start, end int }

//nolint:cyclop // per-op page math plus fragment clone/remap bookkeeping
func splitCrossingRects(res *Result, contentH float64) {
	if res == nil || contentH <= 0 {
		return
	}
	// Give legacy/test-constructed operations an identity before any rewrite.
	// Layout-generated operations already receive IDs from engine.add. A split
	// fragment keeps its source ID; the box-range remap below is what makes all
	// fragments remain owned by the same element.
	assignOpIDs(res)

	// Count crossing ops first: when none split, the display list, box-range
	// remap (identity spans) and any growth are skipped entirely.
	crossings := 0

	for idx := range res.Ops {
		paintOp := &res.Ops[idx]
		if paintOp.Fixed || !isSplittable(paintOp) || paintOp.H <= 0 {
			continue
		}

		page, ok := checkedFlowPageOfY(paintOp.Y+1e-6, contentH)
		if !ok {
			continue
		}

		if paintOp.Y+paintOp.H > float64(page+1)*contentH+1e-9 {
			crossings++
		}
	}

	if crossings == 0 {
		return
	}

	spans := make([]opSpan, len(res.Ops))
	out := make([]Op, 0, len(res.Ops)+crossings*splitSlackPerCrossing)

	for idx := range res.Ops {
		paintOp := res.Ops[idx]
		start := len(out)

		if paintOp.Fixed || !isSplittable(&paintOp) || paintOp.H <= 0 {
			out = append(out, paintOp)
			spans[idx] = opSpan{start: start, end: len(out) - 1}

			continue
		}

		out = appendOpFragments(out, paintOp, contentH)
		spans[idx] = opSpan{start: start, end: len(out) - 1}
	}

	res.Ops = out
	remapBoxOpRanges(res.root, spans)
}

// assignOpIDs gives legacy/test-constructed operations an identity, keeping
// every ID unique across the display list.
func assignOpIDs(res *Result) {
	var nextID uint64

	for i := range res.Ops {
		if res.Ops[i].ID > nextID {
			nextID = res.Ops[i].ID
		}
	}

	for i := range res.Ops {
		if res.Ops[i].ID == 0 {
			nextID++
			res.Ops[i].ID = nextID
		}
	}
}

// appendOpFragments truncates one splittable rect at each page boundary and
// appends the fragments in document paint order. Keeping the destination
// owned by splitCrossingRects avoids one temporary slice allocation per op.
type opFragment struct{ y, h float64 }

func collectOpFragments(paintOp Op, contentH float64) []opFragment {
	frags := make([]opFragment, 0, splitSlackPerCrossing+1)
	rest := paintOp
	guard := 0

	for rest.H > 1e-9 {
		guard++
		if guard > 10000 {
			frags = append(frags, opFragment{y: rest.Y, h: rest.H})

			break
		}

		page, ok := checkedFlowPageOfY(rest.Y+1e-6, contentH)
		if !ok {
			frags = append(frags, opFragment{y: rest.Y, h: rest.H})

			break
		}

		boundary := float64(page+1) * contentH
		if rest.Y+rest.H <= boundary+1e-9 {
			frags = append(frags, opFragment{y: rest.Y, h: rest.H})

			break
		}

		firstH := boundary - rest.Y
		if firstH <= 1e-6 {
			rest.Y = float64(page+1) * contentH

			continue
		}

		frags = append(frags, opFragment{y: rest.Y, h: firstH})
		rest.Y = boundary
		rest.H -= firstH
	}

	return frags
}

// Multi-page stroke frames open at intermediate page edges: only the first
// fragment keeps a top border and only the last keeps a bottom border, so a
// domain card is not falsely closed at a page break and reopened below.
func appendOpFragments(dst []Op, paintOp Op, contentH float64) []Op {
	frags := collectOpFragments(paintOp, contentH)

	if len(frags) == 0 {
		return dst
	}

	if len(frags) == 1 {
		return append(dst, paintOp)
	}

	for i, piece := range frags {
		isFirst := i == 0
		isLast := i == len(frags)-1

		if paintOp.Kind == OpStrokeRect && paintOp.StrokeMask == StrokeMaskTop && !isFirst {
			// Top accent belongs only on the first page of the section.
			dst = append(dst, Op{ //nolint:exhaustruct // intentional no-op fragment
				ID: paintOp.ID, Kind: opKindNoop, Y: piece.y, H: piece.h,
			})

			continue
		}

		fragOp := paintOp
		fragOp.Y = piece.y
		fragOp.H = piece.h

		if paintOp.Kind == OpStrokeRect {
			fragOp = openStrokeFragment(fragOp, isFirst, isLast)
		}

		dst = append(dst, fragOp)
	}

	return dst
}

func openSideStrokeFragment(paintOp Op, isFirst, isLast, isLeft bool) Op {
	if isLeft { //nolint:nestif
		if !isFirst {
			paintOp.RadiusTopLeft = 0
		}

		if !isLast {
			paintOp.RadiusBottomLeft = 0
		}

		if paintOp.RadiusTopLeft == 0 && paintOp.RadiusBottomLeft == 0 {
			paintOp.Radius = 0
		}
	} else {
		if !isFirst {
			paintOp.RadiusTopRight = 0
		}

		if !isLast {
			paintOp.RadiusBottomRight = 0
		}

		if paintOp.RadiusTopRight == 0 && paintOp.RadiusBottomRight == 0 {
			paintOp.Radius = 0
		}
	}

	return paintOp
}

func openFullFrameStrokeFragment(paintOp Op, isFirst, isLast bool) Op {
	// Full frame: open the edges that continue across the page break.
	mask := StrokeMaskLeft | StrokeMaskRight
	if isFirst {
		mask |= StrokeMaskTop
	} else {
		paintOp.RadiusTopLeft, paintOp.RadiusTopRight = 0, 0
	}

	if isLast {
		mask |= StrokeMaskBottom
	} else {
		paintOp.RadiusBottomLeft, paintOp.RadiusBottomRight = 0, 0
	}

	paintOp.StrokeMask = mask
	paintOp.Radius = 0

	return paintOp
}

// openStrokeFragment clears borders that would falsely close a multi-page
// frame at an intermediate page edge.
func openStrokeFragment(paintOp Op, isFirst, isLast bool) Op {
	switch paintOp.StrokeMask {
	case StrokeMaskTop:
		return paintOp
	case StrokeMaskLeft:
		return openSideStrokeFragment(paintOp, isFirst, isLast, true)
	case StrokeMaskRight:
		return openSideStrokeFragment(paintOp, isFirst, isLast, false)
	case 0:
		return openFullFrameStrokeFragment(paintOp, isFirst, isLast)
	default:
		return paintOp
	}
}

// remapBoxOpRanges updates the layout-owned operation ranges after a display
// list rewrite. In particular, a source rectangle can become two or more
// page fragments; mapping the box end to the final fragment keeps pagination,
// sticky/fixed stamping, and ElementLocation ownership aligned.
func remapBoxOpRanges(boxNode *box, spans []opSpan) {
	if boxNode == nil {
		return
	}

	if boxNode.opStart >= 0 && boxNode.opEnd >= boxNode.opStart &&
		boxNode.opStart < len(spans) && boxNode.opEnd < len(spans) {
		boxNode.opStart = spans[boxNode.opStart].start
		boxNode.opEnd = spans[boxNode.opEnd].end
	}

	for _, child := range boxNode.children {
		remapBoxOpRanges(child, spans)
	}
}
