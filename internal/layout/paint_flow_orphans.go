//nolint:all
package layout

// orphansWidows enforces CSS Fragmentation Level 3 Rule 3 (widows/orphans)
// when a leaf block has countable line boxes, and falls back to a geometric
// short-block heuristic when line counts are unavailable.
//
// Class B breaks are rejected when lines-before < orphans or lines-after <
// widows (or the block has fewer lines than orphans+widows can satisfy). If
// the whole block fits on the next page it is shifted; otherwise progress
// escape leaves the break (content taller than one page). Forced breaks
// (page-break-before/after: always) run earlier and are not undone here.
// break-inside: avoid remains higher priority via avoidInside.
func orphansWidows(res *Result, contentH float64) bool {
	if res.root == nil || contentH <= 0 {
		return false
	}

	changed := false

	var walk func(b *box)
	walk = func(boxNode *box) {
		for _, c := range boxNode.children {
			walk(c)
		}

		if orphansWidowsBox(res, boxNode, contentH) {
			changed = true
		}
	}
	walk(res.root)

	return changed
}

// orphansWidowsBox applies Rule 3 (or the geometric fallback) to one block
// box. Returns whether anything moved.
func orphansWidowsBox(res *Result, boxNode *box, contentH float64) bool {
	if boxNode.kind != displayBlock || boxNode.height <= 0 || boxNode.opStart > boxNode.opEnd {
		return false
	}
	// Nested block containers: children apply Rule 3; only heuristic on
	// short straddlers here.
	if hasNestedFlowChild(boxNode) {
		return orphansWidowsHeuristic(res, boxNode, contentH)
	}

	lines := countBlockLineYs(res, boxNode)
	if len(lines) == 0 {
		return orphansWidowsHeuristic(res, boxNode, contentH)
	}

	return enforceOrphansWidows(res, boxNode, lines, contentH)
}

// enforceOrphansWidows applies CSS Fragmentation Rule 3 (orphans/widows) to a
// leaf block that straddles a page boundary. Returns whether it moved.
func enforceOrphansWidows(res *Result, boxNode *box, lines []float64, contentH float64) bool {
	orphans, widows := resolveOrphansWidows(boxNode)

	layoutOut := int(boxNode.y / contentH)
	hIdx := int((boxNode.y + boxNode.height) / contentH)

	if hIdx <= layoutOut {
		return false
	}

	boundary := float64(layoutOut+1) * contentH
	before, after := countLinesAroundBoundary(lines, boundary)
	// Rule 3 applies to Class B breaks *between line boxes*. If all text
	// lines sit on one side of the boundary (only padding/bg straddles),
	// do not keep-together tall boxes - fall back to the short heuristic.
	if before == 0 || after == 0 {
		return orphansWidowsHeuristic(res, boxNode, contentH)
	}
	// Rule 3: legal Class B break only if both sides meet the minima.
	if before >= orphans && after >= widows {
		return false
	}
	// Keep the block together when it fits one page; else progress escape.
	// Same blank-band guard as avoidInside: do not open a large empty
	// region on the current page for a short keep-together.
	if boxNode.height > contentH+0.01 {
		return false
	}

	remaining := float64(layoutOut+1)*contentH - boxNode.y
	if !hasRoundedOwnChrome(res, boxNode) && preferSplitOverBlank(remaining, boxNode.height, contentH) {
		return false
	}

	dy := float64(hIdx)*contentH - boxNode.y
	if dy <= 1e-6 {
		return false
	}

	shiftFlowY(res, boxNode.opStart, boxNode.opEnd, boxNode.y, dy)

	return true
}

// resolveOrphansWidows returns the effective orphans/widows minima.
func resolveOrphansWidows(boxNode *box) (int, int) {
	orphans := boxNode.style.Orphans
	if orphans < 1 {
		orphans = 2
	}

	widows := boxNode.style.Widows
	if widows < 1 {
		widows = 2
	}

	return orphans, widows
}

// countLinesAroundBoundary splits line baselines into before/after counts.
func countLinesAroundBoundary(lines []float64, boundary float64) (int, int) {
	before, after := 0, 0

	for _, y := range lines {
		if y < boundary-1e-6 {
			before++
		} else {
			after++
		}
	}

	return before, after
}

// orphansWidowsHeuristic is the phase-18 geometric fallback: short blocks
// (~2-4 lines) that straddle a page boundary move wholly when they fit.
func orphansWidowsHeuristic(res *Result, boxNode *box, contentH float64) bool {
	if boxNode.height < 14 || boxNode.height > 60 {
		return false
	}

	layoutOut := int(boxNode.y / contentH)
	hIdx := int((boxNode.y + boxNode.height) / contentH)

	if hIdx <= layoutOut || boxNode.height > contentH {
		return false
	}

	remaining := float64(layoutOut+1)*contentH - boxNode.y
	if preferSplitOverBlank(remaining, boxNode.height, contentH) {
		return false
	}

	dy := float64(hIdx)*contentH - boxNode.y
	if dy <= 1e-6 {
		return false
	}

	shiftFlowY(res, boxNode.opStart, boxNode.opEnd, boxNode.y, dy)

	return true
}

func hasNestedFlowChild(boxNode *box) bool {
	for _, c := range boxNode.children {
		if c.kind == displayBlock || c.kind == displayTable {
			return true
		}
	}

	return false
}

// countBlockLineYs returns distinct text/bullet baseline Y positions in the
// box's op range (approximate line boxes for an IFC leaf block).
func countBlockLineYs(res *Result, boxNode *box) []float64 {
	if res == nil || boxNode.opStart > boxNode.opEnd || boxNode.opStart < 0 {
		return nil
	}

	yCoords := make([]float64, 0, 8)
	seen := make(map[float64]bool)

	end := boxNode.opEnd
	if end >= len(res.Ops) {
		end = len(res.Ops) - 1
	}

	for i := boxNode.opStart; i <= end; i++ {
		paintOp := &res.Ops[i]
		if paintOp.Kind != OpText && paintOp.Kind != OpBullet {
			continue
		}

		if !seen[paintOp.Y] {
			seen[paintOp.Y] = true
			yCoords = append(yCoords, paintOp.Y)
		}
	}

	return yCoords
}

// hasLineY reports whether a baseline Y is already recorded within eps.
func hasLineY(yCoords []float64, y, _ float64) bool {
	seen := make(map[float64]bool, len(yCoords))
	for _, ey := range yCoords {
		seen[ey] = true
	}

	_, ok := seen[y]

	return ok
}
