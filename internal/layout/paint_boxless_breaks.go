package layout

import (
	"slices"
)

// applyBoxlessBreaks enforces blinkless page-break metadata on a boxless
// result (root nil, the blink path). Native results keep the box tree and
// return untouched. Forced breaks run first in ascending Y, each shifting
// every flow op at or below the break so the break lands on the next page
// top; keep ranges run after, each moving wholly when it fits one page but
// straddles a boundary. Both passes reuse shiftFlowY so page buckets stay
// exact.
func applyBoxlessBreaks(res *Result, contentH float64) {
	if res == nil || res.root != nil || contentH <= 0 {
		return
	}

	if len(res.ForcedBreaks) == 0 && len(res.KeepTogether) == 0 {
		return
	}

	if len(res.Ops) == 0 {
		return
	}

	ensureFlowIndex(res, contentH)

	breaks := sortedBoxlessBreaks(res.ForcedBreaks, res.Height, res.Ops)
	keeps := sortedBoxlessKeeps(res.KeepTogether)

	applyBoxlessForced(res, breaks, keeps, contentH)
	applyBoxlessKeeps(res, keeps, contentH)
}

// sortedBoxlessBreaks copies src, sorts ascending, merges near duplicates,
// and drops values outside content (at or past the content end, at or before
// zero). Height is the canvas end; hand-built results with no height fall
// back to the largest non-fixed op Y.
func sortedBoxlessBreaks(src []float64, height float64, ops []Op) []float64 {
	breaks := append([]float64(nil), src...)
	slices.Sort(breaks)

	deduped := breaks[:0]

	for _, breakY := range breaks {
		if len(deduped) > 0 && breakY-deduped[len(deduped)-1] <= layoutCoordEpsilon {
			continue
		}

		deduped = append(deduped, breakY)
	}

	end := height
	if end <= 0 {
		end = maxFlowOpY(ops)
	}

	kept := deduped[:0]

	for _, breakY := range deduped {
		if breakY <= layoutCoordEpsilon || breakY >= end-layoutCoordEpsilon {
			continue
		}

		kept = append(kept, breakY)
	}

	return kept
}

// sortedBoxlessKeeps copies src and sorts by ascending start.
func sortedBoxlessKeeps(src [][2]float64) [][2]float64 {
	keeps := append([][2]float64(nil), src...)
	slices.SortFunc(keeps, func(a, b [2]float64) int {
		switch {
		case a[0] < b[0]:
			return -1
		case a[0] > b[0]:
			return 1
		default:
			return 0
		}
	})

	return keeps
}

// maxFlowOpY returns the largest Y among non-fixed ops, or zero when none.
func maxFlowOpY(ops []Op) float64 {
	maxY := 0.0

	for opIdx := range ops {
		if ops[opIdx].Fixed {
			continue
		}

		if ops[opIdx].Y > maxY {
			maxY = ops[opIdx].Y
		}
	}

	return maxY
}

// applyBoxlessForced shifts flow below each break to the next page top.
// Later breaks and keep ranges at or below the break move with the shift so
// each break is evaluated in live coordinates (cascade).
func applyBoxlessForced(res *Result, breaks []float64, keeps [][2]float64, contentH float64) {
	for breakIdx := range breaks {
		curY := breaks[breakIdx]

		page, pageValid := checkedFlowPageOfY(curY, contentH)
		if !pageValid {
			continue
		}

		target := float64(page+1) * contentH
		delta := target - curY

		if delta <= layoutCoordEpsilon {
			continue
		}

		// Empty index range: only the bucket walk moves ops with Y above
		// curY-epsilon (the boundary itself counts as moving).
		shiftFlowY(res, 0, -1, curY-layoutCoordEpsilon, delta)
		cascadeBoxlessBreak(breaks, keeps, breakIdx, curY, delta)
	}
}

// cascadeBoxlessBreak moves later breaks and keep ranges with a forced shift.
func cascadeBoxlessBreak(breaks []float64, keeps [][2]float64, breakIdx int, curY, delta float64) {
	for nextIdx := breakIdx + 1; nextIdx < len(breaks); nextIdx++ {
		if breaks[nextIdx] >= curY-layoutCoordEpsilon {
			breaks[nextIdx] += delta
		}
	}

	for rangeIdx := range keeps {
		if keeps[rangeIdx][0] >= curY-layoutCoordEpsilon {
			keeps[rangeIdx][0] += delta
		}

		if keeps[rangeIdx][1] >= curY-layoutCoordEpsilon {
			keeps[rangeIdx][1] += delta
		}
	}
}

// applyBoxlessKeeps moves each keep range that fits one page but straddles a
// boundary so its start lands on the next page top. Later ranges move with
// each shift so they are evaluated in live coordinates.
func applyBoxlessKeeps(res *Result, keeps [][2]float64, contentH float64) {
	for keepIdx := range keeps {
		delta, start, shift := boxlessKeepShift(keeps[keepIdx], contentH)
		if !shift {
			continue
		}

		// Empty index range: only the bucket walk moves ops at or below start.
		shiftFlowY(res, 0, -1, start-layoutCoordEpsilon, delta)
		cascadeBoxlessKeeps(keeps, keepIdx, start, delta)
	}
}

// boxlessKeepShift returns the shift moving keep wholly to the next page top
// when it fits one page but straddles a boundary.
func boxlessKeepShift(keep [2]float64, contentH float64) (float64, float64, bool) {
	start, end := keep[0], keep[1]

	if end <= start || end-start > contentH+layoutCoordEpsilon {
		return 0, 0, false
	}

	startPage, startValid := checkedFlowPageOfY(start, contentH)
	if !startValid {
		return 0, 0, false
	}

	endPage, endValid := checkedFlowPageOfY(end-layoutCoordEpsilon, contentH)
	if !endValid || endPage <= startPage {
		return 0, 0, false
	}

	delta := float64(startPage+1)*contentH - start
	if delta <= layoutCoordEpsilon {
		return 0, 0, false
	}

	return delta, start, true
}

// cascadeBoxlessKeeps moves later keep ranges with a keep shift.
func cascadeBoxlessKeeps(keeps [][2]float64, keepIdx int, start, delta float64) {
	for nextIdx := keepIdx + 1; nextIdx < len(keeps); nextIdx++ {
		if keeps[nextIdx][0] >= start-layoutCoordEpsilon {
			keeps[nextIdx][0] += delta
		}

		if keeps[nextIdx][1] >= start-layoutCoordEpsilon {
			keeps[nextIdx][1] += delta
		}
	}
}
