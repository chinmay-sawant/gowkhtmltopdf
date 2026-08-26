package layout

import (
	"math"
)

func isSplittable(op *Op) bool {
	return op.Kind == OpFillRect || op.Kind == OpStrokeRect || op.Kind == OpLine
}

// capTablePageBreaks draws a horizontal top edge on pages where a table
// continuation begins mid-grid (split vertical rules at the page top with no
// matching full-width horizontal). Without this, border-collapse rowspan
// tables leave open tops and orphan vertical stubs (wiki awards before
// "2024 Razzie").
func capTablePageBreaks(res *Result, contentH float64) {
	if res == nil || contentH <= 0 || len(res.Ops) == 0 {
		return
	}

	maxPage := capTableMaxPage(res, contentH)
	if maxPage == 0 {
		return
	}

	const eps = 2.0

	_, horiz, vertStarts, vertEnds, horizByY := collectTableBorderSegments(res)
	seal := func(gVal, minX, maxX, borderW, red, green, blue float64) {
		sealBorderGap(res, &horiz, horizByY, eps, gVal, minX, maxX, borderW, red, green, blue)
	}
	coverage := func(y, minX, maxX float64) bool {
		full, _, _, _ := hCoverage(horizByY, y, minX, maxX)

		return full
	}

	// (1) Classic page-top stubs.
	sealPageTopStubs(vertStarts, coverage, seal, maxPage, contentH, eps)

	// (2) Seal incomplete tops of multi-column vertical clusters that start a
	// continuation-page body band (under repeated thead or at page top).
	// Mid-table rowspan holes keep skipped tops so continuous year cells stay
	// unsplit; only the page-fragment open edge is closed.
	sealPageTopClusters(vertStarts, vertEnds, contentH, coverage, seal)

	// Row bottoms: seal when verticals end near a page bottom and no full
	// horizontal closes the strip (next row's top moved to the following page).
	sealPageBottomClusters(vertStarts, vertEnds, contentH, coverage, seal, eps)
}

// sealBorderGap appends a horizontal rule at gVal spanning [minX, maxX] unless
// a near-identical rule already exists.
func sealBorderGap(
	res *Result, horiz *[]hseg, horizByY map[int][]hseg, eps, gVal, minX, maxX, borderW, red, green, blue float64,
) {
	if maxX-minX < 20 || borderW < 0 {
		return
	}

	if borderW < minBorderWidthPt {
		borderW = 0.5
	}
	// Avoid exact duplicates.
	for _, h := range *horiz {
		if math.Abs(h.y-gVal) <= 0.5 && math.Abs(h.x0-minX) <= eps && math.Abs(h.x1-maxX) <= eps {
			return
		}
	}

	op := Op{ //nolint:exhaustruct // intentional zero fields
		Kind: OpLine, X: minX, Y: gVal, W: maxX - minX, H: 0,
		Width: borderW, R: red, G: green, B: blue,
	}
	res.Ops = append(res.Ops, op)
	// This generated op is appended after flow pagination may have indexed the
	// display list. Rebuild the index before any later movement consults it.
	invalidateFlowIndex(res)

	sealed := hseg{minX, maxX, gVal, borderW, red, green, blue}
	*horiz = append(*horiz, sealed)
	horizByY[roundY(gVal)] = append(horizByY[roundY(gVal)], sealed)
}

// sealPageTopClusters closes vertical clusters that start near the top of a
// continuation page's body band.
func sealPageTopClusters(
	vertStarts, vertEnds map[int][]vseg, contentH float64,
	coverage func(y, minX, maxX float64) bool,
	seal func(gVal, minX, maxX, borderW, red, green, blue float64),
) {
	for _, child := range clusterVerticals(vertStarts, vertEnds, true) {
		if child.n < three || child.maxX-child.minX < 20 {
			continue
		}

		if coverage(child.y, child.minX, child.maxX) {
			continue
		}

		page, ok := checkedFlowPageOfY(child.y, contentH)
		if !ok {
			continue
		}

		if page <= 0 {
			continue
		}

		pageTop := float64(page) * contentH
		// Body under thead typically starts within ~header+padding of page top.
		if child.y > pageTop+80 {
			continue
		}

		seal(child.y, child.minX, child.maxX, child.bw, child.r, child.g, child.b)
	}
}

// sealPageBottomClusters closes vertical clusters that end near a page bottom
// with no full horizontal rule (next row's top moved to the following page).
func sealPageBottomClusters(
	vertStarts, vertEnds map[int][]vseg, contentH float64,
	coverage func(y, minX, maxX float64) bool,
	seal func(gVal, minX, maxX, borderW, red, green, blue float64),
	eps float64,
) {
	for _, child := range clusterVerticals(vertStarts, vertEnds, false) {
		if child.n < three || child.maxX-child.minX < 20 {
			continue
		}

		page := int((child.y - layoutSlack) / contentH)
		pageBot := float64(page+1) * contentH
		// Only near the page boundary (row ended as last on page).
		if child.y < pageBot-40 || child.y > pageBot+eps {
			continue
		}

		if coverage(child.y, child.minX, child.maxX) {
			continue
		}

		if page >= 0 {
			seal(child.y, child.minX, child.maxX, child.bw, child.r, child.g, child.b)
		}
	}
}

// vseg is one vertical border segment.
type vseg struct{ x, y0, y1, w, r, g, b float64 }

// hseg is one horizontal border segment.
type hseg struct{ x0, x1, y, w, r, g, b float64 }

// borderCluster is a group of vertical segments sharing a start/end Y.
type borderCluster struct {
	y           float64
	minX, maxX  float64
	bw, r, g, b float64
	n           int
}

func capTableMaxPage(res *Result, contentH float64) int {
	maxPage := 0

	for i := range res.Ops {
		if res.Ops[i].Fixed {
			continue
		}

		pageVal, ok := checkedFlowPageOfY(res.Ops[i].Y, contentH)
		if !ok {
			continue
		}

		if pageVal > maxPage {
			maxPage = pageVal
		}
	}

	return maxPage
}

// collectTableBorderSegments gathers non-fixed vertical/horizontal line ops
// once and groups them by rounded Y.
func collectTableBorderSegments(res *Result) ([]vseg, []hseg, map[int][]vseg, map[int][]vseg, map[int][]hseg) {
	verts, horiz := collectBorderSegmentOps(res.Ops)

	// Count each rounded-Y bucket first so the segment slices grow once. A
	// table-heavy document has many repeated Y keys; appending directly to the
	// maps makes each bucket repeatedly reallocate as rows accumulate.
	startCounts := make(map[int]int)
	endCounts := make(map[int]int)
	horizCounts := make(map[int]int)

	for i := range verts {
		v := verts[i]
		k0, k1 := roundY(v.y0), roundY(v.y1)
		startCounts[k0]++
		endCounts[k1]++
	}

	for i := range horiz {
		h := horiz[i]
		ky := roundY(h.y)
		horizCounts[ky]++
	}

	vertStarts := make(map[int][]vseg, len(startCounts))
	vertEnds := make(map[int][]vseg, len(endCounts))
	horizByY := make(map[int][]hseg, len(horizCounts))

	for key, count := range startCounts {
		vertStarts[key] = make([]vseg, 0, count)
	}

	for key, count := range endCounts {
		vertEnds[key] = make([]vseg, 0, count)
	}

	for key, count := range horizCounts {
		horizByY[key] = make([]hseg, 0, count)
	}

	for i := range verts {
		v := verts[i]
		k0, k1 := roundY(v.y0), roundY(v.y1)
		vertStarts[k0] = append(vertStarts[k0], v)
		vertEnds[k1] = append(vertEnds[k1], v)
	}

	for i := range horiz {
		h := horiz[i]
		ky := roundY(h.y)
		horizByY[ky] = append(horizByY[ky], h)
	}

	return verts, horiz, vertStarts, vertEnds, horizByY
}

// collectBorderSegmentOps gathers non-fixed line ops as vertical or horizontal
// border segments.
func collectBorderSegmentOps(ops []Op) ([]vseg, []hseg) { //nolint:cyclop // two-pass classify-then-build
	// First pass: count so we allocate exact-capacity slices once.
	nVert, nHoriz := 0, 0

	for i := range ops {
		paintOp := &ops[i]
		if paintOp.Fixed || paintOp.Kind != OpLine {
			continue
		}

		if paintOp.H > 2 && (paintOp.W < 1 || paintOp.W < paintOp.H*0.05) {
			nVert++

			continue
		}

		if paintOp.W > 2 && paintOp.H < 1 {
			nHoriz++
		}
	}

	verts := make([]vseg, 0, nVert)
	horiz := make([]hseg, 0, nHoriz)

	for i := range ops {
		paintOp := &ops[i]
		if paintOp.Fixed || paintOp.Kind != OpLine {
			continue
		}

		if paintOp.H > 2 && (paintOp.W < 1 || paintOp.W < paintOp.H*0.05) {
			verts = append(verts, vseg{
				x: paintOp.X, y0: paintOp.Y, y1: paintOp.Y + paintOp.H,
				w: paintOp.Width, r: paintOp.R, g: paintOp.G, b: paintOp.B,
			})

			continue
		}

		if paintOp.W > 2 && paintOp.H < 1 {
			horiz = append(horiz, hseg{
				x0: paintOp.X, x1: paintOp.X + paintOp.W, y: paintOp.Y,
				w: paintOp.Width, r: paintOp.R, g: paintOp.G, b: paintOp.B,
			})
		}
	}

	return verts, horiz
}

// roundY bins a canvas Y into 0.5pt buckets.
func roundY(y float64) int { return int(math.Round(y * two)) }

// clusterVerticals merges vertical segments sharing a start (byStart) or end
// Y into clusters with the min/max x and dominant stroke.
func clusterVerticals(vertStarts, vertEnds map[int][]vseg, byStart bool) map[int]borderCluster {
	groups := vertStarts
	if !byStart {
		groups = vertEnds
	}

	out := make(map[int]borderCluster, len(groups))

	for _, group := range groups {
		for _, val := range group {
			keyY := val.y0
			if !byStart {
				keyY = val.y1
			}

			bucket := roundY(keyY)

			child, ok := out[bucket]
			if !ok {
				out[bucket] = borderCluster{y: keyY, minX: val.x, maxX: val.x, bw: val.w, r: val.r, g: val.g, b: val.b, n: 1}

				continue
			}

			child.n++
			if val.x < child.minX {
				child.minX = val.x
			}

			if val.x > child.maxX {
				child.maxX = val.x
			}
			// Prefer average y so we sit on the dominant edge.
			child.y = (child.y*float64(child.n-1) + keyY) / float64(child.n)
			out[bucket] = child
		}
	}

	return out
}

// hCoverage reports whether horizontal segments near posY span [minX, maxX].
func hCoverage(horizByY map[int][]hseg, posY, minX, maxX float64) (bool, float64, float64, bool) {
	const eps = 2.0

	var covMin, covMax float64

	has := false

	key := roundY(posY)
	for k := key - int(eps*two) - 1; k <= key+int(eps*two)+1; k++ {
		for _, height := range horizByY[k] {
			covMin, covMax, has = mergeCoverageSeg(height, posY, minX, maxX, eps, covMin, covMax, has)
		}
	}

	if !has {
		return false, 0, 0, false
	}

	full := covMin <= minX+eps && covMax >= maxX-eps

	return full, covMin, covMax, true
}

// mergeCoverageSeg extends the running coverage with one segment that lies at
// posY within eps and overlaps the vertical band.
func mergeCoverageSeg(
	height hseg, posY, minX, maxX, eps float64, covMin, covMax float64, has bool,
) (float64, float64, bool) {
	if math.Abs(height.y-posY) > eps {
		return covMin, covMax, has
	}
	// Only count segments that overlap the vertical band.
	if height.x1 < minX-eps || height.x0 > maxX+eps {
		return covMin, covMax, has
	}

	if !has {
		return height.x0, height.x1, true
	}

	if height.x0 < covMin {
		covMin = height.x0
	}

	if height.x1 > covMax {
		covMax = height.x1
	}

	return covMin, covMax, true
}

// sealPageTopStubs seals vertical stubs that start exactly at a page top with
// no closing horizontal rule.
func sealPageTopStubs(
	vertStarts map[int][]vseg,
	coverage func(y, minX, maxX float64) bool,
	seal func(gVal, minX, maxX, borderW, red, green, blue float64),
	maxPage int, contentH, eps float64,
) {
	for p := 1; p <= maxPage; p++ {
		pageTop := float64(p) * contentH

		minX, maxX, borderW, redN, green, blueN, node := pageTopStubBounds(vertStarts, pageTop, eps)

		if node < two {
			continue
		}

		if coverage(pageTop, minX, maxX) {
			continue
		}

		seal(pageTop, minX, maxX, borderW, redN, green, blueN)
	}
}

// pageTopStubBounds scans vertical segments at pageTop for the min/max x and
// dominant stroke of the stub cluster, returning how many stubs matched.
func pageTopStubBounds(
	vertStarts map[int][]vseg, pageTop, eps float64,
) (float64, float64, float64, float64, float64, float64, int) {
	var minX, maxX, borderW, red, green, blue float64

	node := 0

	key := roundY(pageTop)
	for k := key - int(eps*two) - 1; k <= key+int(eps*two)+1; k++ {
		for _, val := range vertStarts[k] {
			if val.y0 < pageTop-eps || val.y0 > pageTop+eps {
				continue
			}

			if node == 0 {
				minX, maxX, borderW, red, green, blue = val.x, val.x, val.w, val.r, val.g, val.b
			} else {
				if val.x < minX {
					minX = val.x
				}

				if val.x > maxX {
					maxX = val.x
				}
			}

			node++
		}
	}

	return minX, maxX, borderW, red, green, blue, node
}

// paginateOps assigns every op a page. Crossing text/image/link ops snap to
// the next page boundary (taking following flow with them so row spacing is
// preserved); then page-break policies are applied as canvas-Y shifts; finally
// pages derive from the final Y positions. Rect-type ops crossing a boundary
// are split by Paint.
func paginateOps(res *Result, contentH float64) []int {
	ensureFlowIndex(res, contentH)
	// Resolve forced section starts before snapping text to provisional page
	// boundaries. Otherwise a row near the boundary of the unbroken flow can
	// move its text alone; a later page-break-before shift then leaves the
	// collapsed-table chrome behind at the old row position.
	for range 10 {
		if !beforeAlways(res, contentH) {
			break
		}
	}

	// Lift aside callouts that do not fit the remaining Y on this page
	// before snapCrossingTextOps splits their last lines off to the next
	// page top (that snap-then-shift left an internal gap in the card).
	for range 10 {
		if !keepImplicitAsides(res, contentH) {
			break
		}
	}

	snapCrossingTextOps(res, contentH)

	paginationFixpoint(res, contentH)
	// After flow has settled, clone <thead> onto continuation pages.
	// Blank avoid-list bands are controlled by preferSplitOverBlank during
	// the fixpoint above (former packAvoidGaps sibling packing was a no-op).
	repeatTableHeaders(res, contentH)
	// Page-break fixups can temporarily move a final row across a boundary and
	// later pull the table back, leaving that row's chrome at the old position.
	// Collapse those stale inter-row gaps before painting the table.
	normalizeTableRowGaps(res, contentH)
	// Header continuation shifts can reintroduce a small leading band above a
	// rounded security callout. Normalize after all flow shifts are complete.
	normalizeLeadingRoundedCallouts(res, contentH)
	// Forced breaks win over the callout pack: a same-page snap must not
	// leave page-break-before:always parked on the previous page.
	for range 10 {
		if !beforeAlways(res, contentH) {
			break
		}
	}
	// Sticky is applied in Paint after rect splitting (see splitCrossingRects).
	opPage := make([]int, len(res.Ops))

	for opIdx := range res.Ops {
		page, ok := checkedFlowPageOfY(res.Ops[opIdx].Y, contentH)
		if !ok {
			opPage[opIdx] = -1
		} else {
			opPage[opIdx] = page
		}
	}

	return opPage
}

// paginationFixpoint runs the page-break policies until none of them move
// anything or the iteration cap is reached.
func paginationFixpoint(res *Result, contentH float64) {
	for range 10 {
		changed := avoidInside(res, contentH)
		if beforeAlways(res, contentH) {
			changed = true
		}

		if afterBreaks(res, contentH) {
			changed = true
		}

		if rowsIntact(res, contentH) {
			changed = true
		}

		if keepHeadingWithNext(res, contentH) {
			changed = true
		}

		if orphansWidows(res, contentH) {
			changed = true
		}

		if !changed {
			break
		}
	}
}

// snapCrossingTextOps moves text/image/link ops that cross a page boundary to
// the next page (taking following flow with them so row spacing is preserved).
func snapCrossingTextOps(res *Result, contentH float64) {
	tableRanges := tablePaintRanges(res)

	for idx := range len(res.Ops) {
		paintOp := &res.Ops[idx]
		if paintOp.Fixed || opInPaintRange(idx, tableRanges) {
			continue
		}

		switch paintOp.Kind {
		case OpText, OpBullet, OpImage, OpLinkURI:
			opH := opInkHeight(*paintOp)

			page, ok := checkedFlowPageOfY(paintOp.Y, contentH)
			if !ok {
				continue
			}

			boundary := float64(page+1) * contentH
			if paintOp.Y+opH > boundary+1e-9 {
				snapOpToBoundary(res, idx, paintOp, boundary)
			}
		case OpFillRect, OpStrokeRect, OpLine, opKindNoop:
		}
	}
}

type paintRange struct{ first, last int }

func tablePaintRanges(res *Result) []paintRange {
	if res == nil || res.root == nil {
		return nil
	}

	ranges := make([]paintRange, 0)

	for _, boxNode := range flowBoxList(res) {
		if boxNode.kind != displayTable || boxNode.opStart > boxNode.opEnd {
			continue
		}

		ranges = append(ranges, paintRange{first: boxNode.opStart, last: boxNode.opEnd})
	}

	return ranges
}

func opInPaintRange(index int, ranges []paintRange) bool {
	for _, span := range ranges {
		if index >= span.first && index <= span.last {
			return true
		}
	}

	return false
}

// snapOpToBoundary shifts one crossing op (and the row chrome under it) onto
// the next page, leaving ascender room above the boundary.
func snapOpToBoundary(res *Result, idx int, paintOp *Op, boundary float64) {
	if boundary-paintOp.Y > layoutEpsilon {
		snapOpForward(res, idx, paintOp, boundary)

		return
	}

	paintOp.Y = boundary
}

// snapOpForward snaps the op forward to the next page with the row chrome
// that belongs to it, keeping the same-row fill tops above the text.
func snapOpForward(res *Result, idx int, paintOp *Op, boundary float64) {
	// Snap text (+ following flow). Same-row fills sit above the
	// baseline; include them in deltaY via minY so their tops clear
	// onto this page with the text (fixture-31 Row 28 white bg).
	// Keep chrome matching tight (one row) so table reports do
	// not inflate deltaY. Never clamp fill tops to `boundary` alone
	// — that collapses them onto the text Y and leaves section
	// gray showing through the ascent/padding band.
	oldY := paintOp.Y
	chrome, minY := rowChromeAbove(res, idx, oldY)
	// Leave room for ascenders above the baseline so snapped
	// lines do not paint into the top margin (page-4/5 bleed).
	lead := 0.0
	if paintOp.Kind == OpText || paintOp.Kind == OpBullet {
		lead = paintOp.Size * pxToPtFactor
		if lead < maxGlueEm {
			lead = 8
		}
	}

	deltaY := boundary + lead - minY
	shiftFlowY(res, idx, idx, oldY-layoutSlack, deltaY)
	shiftNearestOwnedChrome(res, idx, oldY, deltaY)

	for _, j := range chrome {
		o := &res.Ops[j]
		if o.Y < oldY-0.01 {
			o.Y += deltaY
		}
	}
}

// shiftNearestOwnedChrome moves the nearest block's background/side rail with
// a text line that was snapped to the next page. Row-sized chrome is handled
// separately by rowChromeAbove; this covers callout rails and other block
// chrome that can span more than one row.
//
//nolint:cyclop,wsl,mnd // chrome ownership walk
func shiftNearestOwnedChrome(res *Result, opIndex int, oldY, deltaY float64) {
	if res == nil || res.root == nil || deltaY == 0 {
		return
	}

	path := make([]*box, 0, 8)
	if !findBoxPathForOp(res.root, opIndex, &path) {
		return
	}

	for pathIndex := len(path) - 1; pathIndex >= 0; pathIndex-- {
		boxNode := path[pathIndex]
		moved := false
		for idx := boxNode.opStart; idx <= boxNode.opEnd && idx < len(res.Ops); idx++ {
			chromeOp := &res.Ops[idx]
			if idx == opIndex || !isOwnBoxChrome(*chromeOp, boxNode, boxNode.y+boxNode.height) ||
				chromeOp.Y >= oldY-0.01 ||
				(oldY-chromeOp.Y > rowChromeBandTolerance && chromeOp.Kind != OpLine) {
				continue
			}

			chromeOp.Y += deltaY
			moved = true
		}
		if moved {
			return
		}
	}
}

//nolint:wsl // recursive ownership walk keeps path mutation adjacent
func findBoxPathForOp(boxNode *box, opIndex int, path *[]*box) bool {
	if boxNode == nil || opIndex < boxNode.opStart || opIndex > boxNode.opEnd {
		return false
	}

	*path = append(*path, boxNode)
	for _, child := range boxNode.children {
		if findBoxPathForOp(child, opIndex, path) {
			return true
		}
	}
	*path = (*path)[:len(*path)-1]

	return true
}

// rowChromeAbove collects fill/stroke rects whose band touches oldY, with
// their minimum top Y. Chrome is row-tight (rows never split), so when the
// flow index is available the scan is limited to the op's page bucket; the
// page above is scanned too when the band reaches across the page top. A full
// display-list scan is the fallback.
func rowChromeAbove(res *Result, idx int, oldY float64) ([]int, float64) {
	chrome := make([]int, 0, rowChromeCap)

	minY := oldY

	//nolint:nestif // row chrome candidate resolution
	if res.flowPageSize > 0 {
		page, ok := checkedFlowPageOfY(oldY, res.flowPageSize)
		if !ok {
			return chrome, minY
		}

		if page < 0 {
			page = 0
		}

		if page < len(res.flowPages) {
			if page > 0 && oldY-float64(page)*res.flowPageSize < rowChromeBandTolerance {
				chrome, minY = appendRowChromeCandidates(chrome, res.Ops, res.flowPages[page-1], idx, oldY, minY)
			}

			chrome, minY = appendRowChromeCandidates(chrome, res.Ops, res.flowPages[page], idx, oldY, minY)

			return chrome, minY
		}
	}

	for jdx := range res.Ops {
		obj := &res.Ops[jdx]
		if !rowChromeBandCandidate(obj, jdx, idx, oldY) {
			continue
		}

		chrome = append(chrome, jdx)

		if obj.Y < minY {
			minY = obj.Y
		}
	}

	return chrome, minY
}

// appendRowChromeCandidates appends the band candidates of idxs to chrome,
// lowering minY for candidates whose top sits above oldY.
func appendRowChromeCandidates(chrome []int, ops []Op, idxs []int, idx int, oldY, minY float64) ([]int, float64) {
	for _, jdx := range idxs {
		obj := &ops[jdx]
		if !rowChromeBandCandidate(obj, jdx, idx, oldY) {
			continue
		}

		chrome = append(chrome, jdx)

		if obj.Y < minY {
			minY = obj.Y
		}
	}

	return chrome, minY
}

// rowChromeBandCandidate reports whether obj is a one-row fill/stroke rect
// sitting on the same band as the op at oldY (not the op itself).
func rowChromeBandCandidate(obj *Op, jdx, idx int, oldY float64) bool {
	if obj.Fixed || jdx == idx {
		return false
	}

	if obj.Kind != OpFillRect && obj.Kind != OpStrokeRect {
		return false
	}

	if obj.H <= 0.5 || obj.H > 40 {
		return false
	}

	if obj.Y > oldY+0.5 || obj.Y+obj.H < oldY-0.5 {
		return false
	}

	return oldY-obj.Y <= obj.H+2
}

// splitCrossingRects truncates splittable ops at each page boundary and keeps
// remainders immediately after (document paint order). Built into a new slice
// rather than mid-slice insert+copy (O(n²) and float-edge infinite loops when
// Y sits on a page boundary — TestTenPageTableReportPerformance hang).
type opSpan struct{ start, end int }

//nolint:cyclop // per-op page math plus fragment clone/remap bookkeeping
func splitCrossingRects(res *Result, contentH float64, opPage []int) {
	_ = opPage

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

		page, ok := checkedFlowPageOfY(paintOp.Y+layoutEpsilon, contentH)
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
		if guard > paginationGuardMax {
			frags = append(frags, opFragment{y: rest.Y, h: rest.H})

			break
		}

		page, ok := checkedFlowPageOfY(rest.Y+layoutEpsilon, contentH)
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
		if firstH <= layoutEpsilon {
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

func openLeftStrokeFragment(paintOp Op, isFirst, isLast bool) Op {
	if !isFirst {
		paintOp.RadiusTopLeft = 0
	}

	if !isLast {
		paintOp.RadiusBottomLeft = 0
	}

	if paintOp.RadiusTopLeft == 0 && paintOp.RadiusBottomLeft == 0 {
		paintOp.Radius = 0
	}

	return paintOp
}

func openRightStrokeFragment(paintOp Op, isFirst, isLast bool) Op {
	if !isFirst {
		paintOp.RadiusTopRight = 0
	}

	if !isLast {
		paintOp.RadiusBottomRight = 0
	}

	if paintOp.RadiusTopRight == 0 && paintOp.RadiusBottomRight == 0 {
		paintOp.Radius = 0
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
		return openLeftStrokeFragment(paintOp, isFirst, isLast)
	case StrokeMaskRight:
		return openRightStrokeFragment(paintOp, isFirst, isLast)
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

func calculateChromeInkBottom(res *Result, boxNode *box, oldBottom float64) (float64, bool) {
	inkBottom := boxNode.y
	hasInk := false

	for idx := boxNode.opStart; idx <= boxNode.opEnd; idx++ {
		operation := res.Ops[idx]
		if isOwnBoxChrome(operation, boxNode, oldBottom) ||
			isOwnBoxChromeFragment(operation, boxNode, oldBottom) ||
			operation.Positioned {
			continue
		}

		bottom := opInkBottom(operation)
		if bottom > inkBottom {
			inkBottom = bottom
			hasInk = true
		}
	}

	if boxIsFloatOrFigure(boxNode) {
		if childBottom := lastInFlowChildBottom(boxNode); childBottom > oldBottom {
			return childBottom, true
		}

		return inkBottom, false
	}

	if childBottom := lastInFlowChildBottom(boxNode); childBottom > inkBottom {
		return childBottom, true
	}

	return inkBottom, hasInk
}

func isBoxChromeEligible(res *Result, boxNode *box) bool {
	if boxNode.opStart < 0 || boxNode.opStart > boxNode.opEnd || boxNode.opEnd >= len(res.Ops) || boxNode.height <= 0 {
		return false
	}

	if boxInsideTable(boxNode) || !hasOwnVerticalChrome(res.Ops, boxNode) {
		return false
	}

	return true
}

func calculateChromeContentBottom(boxNode *box, oldBottom, inkBottom float64, hasInk bool) float64 {
	if !hasInk {
		return oldBottom
	}

	padB := 0.0
	if boxNode.style != nil {
		padB = boxNode.style.PaddingBottom
	}

	desiredBottom := inkBottom + padB
	if desiredBottom > oldBottom {
		return desiredBottom
	}

	return oldBottom
}

func stretchBoxChrome(res *Result, boxNode *box) {
	if !isBoxChromeEligible(res, boxNode) {
		return
	}

	oldBottom := boxNode.y + boxNode.height
	normalizeOwnVerticalChrome(res.Ops, boxNode)

	inkBottom, hasInk := calculateChromeInkBottom(res, boxNode, oldBottom)
	contentBottom := calculateChromeContentBottom(boxNode, oldBottom, inkBottom, hasInk)

	if contentBottom > oldBottom+1e-6 {
		boxNode.height = contentBottom - boxNode.y
	}

	normalizeOwnVerticalChrome(res.Ops, boxNode)

	for idx := boxNode.opStart; idx <= boxNode.opEnd; idx++ {
		stretchOwnBoxChrome(&res.Ops[idx], boxNode, oldBottom, contentBottom)
	}
}

// stretchPaginatedChrome repairs block chrome after pagination has shifted a
// descendant past the block's original bottom. The layout box is built before
// page-break fixups, so its background and side rails otherwise stop at the
// stale natural height while the moved footer/text continues below it.
func stretchPaginatedChrome(res *Result) {
	if res == nil || res.root == nil {
		return
	}

	var walk func(*box)
	walk = func(boxNode *box) {
		for _, child := range boxNode.children {
			walk(child)
		}

		stretchBoxChrome(res, boxNode)
	}

	walk(res.root)
}

func boxIsFloatOrFigure(boxNode *box) bool {
	if boxNode == nil {
		return false
	}

	if boxNode.node != nil && boxNode.node.Name == "figure" {
		return true
	}

	if boxNode.style == nil {
		return false
	}

	return boxNode.style.Float == floatLeft || boxNode.style.Float == floatRight
}

func lastInFlowChildBottom(boxNode *box) float64 {
	if boxNode == nil {
		return 0
	}

	bottom := 0.0

	for _, child := range boxNode.children {
		if child == nil {
			continue
		}

		if child.style != nil && (child.style.Position == positionAbsolute || child.style.Position == positionFixed) {
			continue
		}

		if childBottom := child.y + child.height; childBottom > bottom {
			bottom = childBottom
		}
	}

	return bottom
}

//nolint:wsl // border ownership checks are intentionally explicit
func hasOwnVerticalChrome(ops []Op, boxNode *box) bool {
	if boxNode == nil || boxNode.style == nil {
		return false
	}
	leftBorder := boxNode.style.BorderLeft.Width > 0 && boxNode.style.BorderLeft.Style != cssDisplayNone
	rightBorder := boxNode.style.BorderRight.Width > 0 && boxNode.style.BorderRight.Style != cssDisplayNone
	if !leftBorder && !rightBorder {
		return false
	}

	for idx := boxNode.opStart; idx <= boxNode.opEnd && idx < len(ops); idx++ {
		op := ops[idx]
		if isVerticalChromeForBox(op, boxNode, leftBorder, rightBorder) {
			return true
		}
	}

	return false
}

//nolint:cyclop,wsl // fragment collection deliberately mirrors paint ownership
func normalizeOwnVerticalChrome(ops []Op, boxNode *box) {
	if boxNode == nil || boxNode.style == nil {
		return
	}

	leftBorder := boxNode.style.BorderLeft.Width > 0 && boxNode.style.BorderLeft.Style != cssDisplayNone
	rightBorder := boxNode.style.BorderRight.Width > 0 && boxNode.style.BorderRight.Style != cssDisplayNone
	minY := math.Inf(1)
	verticalIndexes := make([]int, 0)
	for idx := boxNode.opStart; idx <= boxNode.opEnd && idx < len(ops); idx++ {
		if isVerticalChromeForBox(ops[idx], boxNode, leftBorder, rightBorder) {
			verticalIndexes = append(verticalIndexes, idx)
			if ops[idx].Y < minY {
				minY = ops[idx].Y
			}
		}
	}
	if math.IsInf(minY, 1) {
		return
	}

	delta := boxNode.y - minY
	if math.Abs(delta) <= layoutEpsilon {
		return
	}
	for _, idx := range verticalIndexes {
		ops[idx].Y += delta
	}

	// Solid rails are one continuous OpLine: extend the last segment to the
	// box bottom after a Y realign. Dashed/dotted sides are many short
	// segments — growing the last one paints a solid stub past the dashes
	// (fixture-40 abs-host, fixture-48 tracking).
	last := verticalIndexes[len(verticalIndexes)-1]
	if isDashLikeVerticalRail(ops[last], boxNode) {
		return
	}

	lastBottom := ops[last].Y + ops[last].H
	if lastBottom < boxNode.y+boxNode.height {
		ops[last].H += boxNode.y + boxNode.height - lastBottom
	}
}

//nolint:cyclop // chrome classification keeps line and masked-side geometry explicit
func isVerticalChromeForBox(operation Op, boxNode *box, leftBorder, rightBorder bool) bool {
	line := operation.Kind == OpLine && operation.W == 0
	maskedLeft := operation.Kind == OpStrokeRect && operation.StrokeMask == StrokeMaskLeft
	maskedRight := operation.Kind == OpStrokeRect && operation.StrokeMask == StrokeMaskRight

	if !line && !maskedLeft && !maskedRight {
		return false
	}

	return operation.H > 0 &&
		operation.H <= boxNode.height+layoutEpsilon &&
		((leftBorder && nearLayout(operation.X, boxNode.x) && (line || maskedLeft)) ||
			(rightBorder && nearLayout(operation.X, boxNode.x+boxNode.w) && (line || maskedRight))) &&
		operation.Y >= boxNode.y-layoutEpsilon && operation.Y <= boxNode.y+boxNode.height+layoutEpsilon
}

// isDashedOrDottedStyle reports border styles expanded into multi-segment OpLines.
func isDashedOrDottedStyle(style string) bool {
	return style == borderStyleDashed || style == borderStyleDotted
}

// looksLikeDashSegmentLength is true for edge pieces sized like appendDashedLineSegments
// (drawLen = width*3 dashed, width dotted), not a continuous solid rail.
func looksLikeDashSegmentLength(segLen, strokeWidth float64) bool {
	if segLen <= 0 {
		return false
	}

	maxSeg := math.Max(strokeWidth*three, minDashPt) + halfRatio
	if strokeWidth <= 0 {
		maxSeg = three + halfRatio
	}

	return segLen <= maxSeg
}

func isVerticalLineOp(operation Op) bool {
	return operation.Kind == OpLine && operation.W == 0 && operation.H > 0
}

// isDashLikeVerticalRail is true when a vertical side stroke must not be H-stretched:
// dashed/dotted CSS on that side, or a short segment that is already a dash piece.
func isDashLikeVerticalRail(operation Op, boxNode *box) bool {
	if !isVerticalLineOp(operation) || boxNode == nil || boxNode.style == nil {
		return false
	}

	if looksLikeDashSegmentLength(operation.H, operation.Width) {
		return true
	}

	onLeft := nearLayout(operation.X, boxNode.x) && isDashedOrDottedStyle(boxNode.style.BorderLeft.Style)
	onRight := nearLayout(operation.X, boxNode.x+boxNode.w) && isDashedOrDottedStyle(boxNode.style.BorderRight.Style)

	return onLeft || onRight
}

// isHorizontalChromeForBox reports top/bottom edge strokes owned by the box,
// including short dashed/dotted segments (full-width nearLayout alone misses those).
//
//nolint:cyclop // edge membership mirrors vertical chrome checks
func isHorizontalChromeForBox(operation Op, boxNode *box, oldBottom float64) bool {
	if operation.Kind != OpLine || operation.H != 0 || operation.W <= 0 || boxNode == nil {
		return false
	}

	onTop := nearLayout(operation.Y, boxNode.y)
	onBottom := nearLayout(operation.Y, oldBottom)

	if !onTop && !onBottom {
		return false
	}

	// Solid (or single) full-width edge.
	if nearLayout(operation.X, boxNode.x) && nearLayout(operation.W, boxNode.w) {
		return true
	}

	if boxNode.style == nil {
		return false
	}

	// Dashed/dotted fragments sit on the edge with dash-sized W.
	inside := operation.X >= boxNode.x-layoutEpsilon &&
		operation.X+operation.W <= boxNode.x+boxNode.w+layoutEpsilon
	if !inside {
		return false
	}

	if onTop && isDashedOrDottedStyle(boxNode.style.BorderTop.Style) {
		return true
	}

	if onBottom && isDashedOrDottedStyle(boxNode.style.BorderBottom.Style) {
		return true
	}

	return looksLikeDashSegmentLength(operation.W, operation.Width)
}

func opInkBottom(operation Op) float64 {
	if operation.Kind == OpText || operation.Kind == OpBullet {
		return operation.Y + opVisibleInkHeight(operation)
	}

	if operation.H > 0 {
		return operation.Y + operation.H
	}

	if operation.Kind == OpLine && operation.W > 0 {
		return operation.Y + math.Max(operation.Width, 1)
	}

	return operation.Y
}

//nolint:cyclop // classify box-owned paint
func isOwnBoxChrome(operation Op, boxNode *box, oldBottom float64) bool {
	if boxNode == nil {
		return false
	}

	if (operation.Kind == OpFillRect || operation.Kind == OpStrokeRect) &&
		nearLayout(operation.X, boxNode.x) && nearLayout(operation.Y, boxNode.y) &&
		nearLayout(operation.W, boxNode.w) && nearLayout(operation.H, boxNode.height) {
		return true
	}

	if operation.Kind != OpLine {
		return false
	}

	if boxNode.style == nil {
		return false
	}

	vertical := isVerticalChromeForBox(operation, boxNode,
		boxNode.style.BorderLeft.Width > 0 && boxNode.style.BorderLeft.Style != cssDisplayNone,
		boxNode.style.BorderRight.Width > 0 && boxNode.style.BorderRight.Style != cssDisplayNone)
	horizontal := isHorizontalChromeForBox(operation, boxNode, oldBottom)

	return vertical || horizontal
}

func isOwnBoxRectFragment(operation Op, boxNode *box, oldBottom float64) bool {
	isRectKind := operation.Kind == OpFillRect || operation.Kind == OpStrokeRect
	if !isRectKind || !nearLayout(operation.X, boxNode.x) || !nearLayout(operation.W, boxNode.w) {
		return false
	}

	return operation.Y >= boxNode.y-layoutEpsilon && operation.Y+operation.H <= oldBottom+1
}

// isOwnBoxChromeFragment reports page-split fill/stroke/rail pieces of the
// box's own frame. After openStrokeFragment these no longer match the full
// border-box height, so isOwnBoxChrome alone would treat them as content ink
// and re-add padding-bottom on every stretch pass.
func isOwnBoxChromeFragment(operation Op, boxNode *box, oldBottom float64) bool {
	if boxNode == nil || boxNode.style == nil {
		return false
	}

	if isOwnBoxRectFragment(operation, boxNode, oldBottom) || isHorizontalChromeForBox(operation, boxNode, oldBottom) {
		return true
	}

	hasLeft := boxNode.style.BorderLeft.Width > 0 && boxNode.style.BorderLeft.Style != cssDisplayNone
	hasRight := boxNode.style.BorderRight.Width > 0 && boxNode.style.BorderRight.Style != cssDisplayNone

	return isVerticalChromeForBox(operation, boxNode, hasLeft, hasRight)
}

const boxBottomMatchSlack = 1.5

//nolint:cyclop // mutate owned paint
func stretchOwnBoxChrome(operation *Op, boxNode *box, oldBottom, newBottom float64) {
	if operation == nil || boxNode == nil {
		return
	}

	if (operation.Kind == OpFillRect || operation.Kind == OpStrokeRect) &&
		nearLayout(operation.X, boxNode.x) && nearLayout(operation.W, boxNode.w) {
		// Only stretch chrome that currently owns the box bottom. Earlier
		// multi-page fragments end at a page boundary and must stay open;
		// stretching them to newBottom refilled whole pages (fixture-31).
		fullMatch := nearLayout(operation.Y, boxNode.y) && nearLayout(operation.H, oldBottom-boxNode.y)
		ownsBottom := math.Abs(operation.Y+operation.H-oldBottom) < boxBottomMatchSlack

		if fullMatch || ownsBottom {
			operation.H = newBottom - operation.Y
			if operation.H < 0 {
				operation.H = 0
			}

			return
		}
	}

	if operation.Kind == OpLine && operation.W == 0 && operation.H > 0 &&
		((boxNode.style != nil && boxNode.style.BorderLeft.Width > 0 && nearLayout(operation.X, boxNode.x)) ||
			(boxNode.style != nil && boxNode.style.BorderRight.Width > 0 && nearLayout(operation.X, boxNode.x+boxNode.w))) &&
		operation.Y >= boxNode.y-layoutEpsilon && operation.Y <= boxNode.y+boxNode.height+layoutEpsilon &&
		nearLayout(operation.Y+operation.H, oldBottom) {
		// Never elongate a dash/dot segment into a solid stub.
		if isDashLikeVerticalRail(*operation, boxNode) {
			return
		}

		operation.H = newBottom - operation.Y

		return
	}

	// Bottom edge: solid full-width line, or every dashed/dotted fragment on
	// that edge (short W would miss nearLayout(W, box.w)).
	if operation.Kind == OpLine && operation.H == 0 && operation.W > 0 &&
		isHorizontalChromeForBox(*operation, boxNode, oldBottom) &&
		nearLayout(operation.Y, oldBottom) {
		operation.Y = newBottom
	}
}

func nearLayout(a, b float64) bool { return math.Abs(a-b) < 0.01 } //nolint:mnd // layout epsilon

// stripOrphanRowChrome removes row-sized fills and horizontal rules that sit
// on a page with no overlapping text/bullet/image ink. Page-break snaps move
// the text but leave the previous row's trailing fill / the snapped row's
// background behind, which reads as empty rows (fixture-31 after Row 27).
func stripOrphanRowChrome(res *Result, contentH float64) {
	if res == nil || contentH <= 0 || len(res.Ops) == 0 {
		return
	}

	pageOps := pageIndexedOps(res, contentH)

	stickyTargets := stickySectionChromeTargets(res.root)

	for page := range pageOps {
		pageTop := float64(page) * contentH
		pageBot := pageTop + contentH

		lastInkBot, hasInk := lastInkBottom(res, pageOps[page], pageTop, pageBot)
		if !hasInk {
			continue
		}

		if stripOrphanRows(res, pageOps[page], pageTop, pageBot, lastInkBot) {
			tightenLastRowChrome(res, pageOps[page], pageTop, pageBot, lastInkBot)
		}
		// Pull section washes / borders up to the last row chrome / ink so grey
		// does not pad an empty band to the page bottom (fixture-31 page 1).
		// Only section-colored chrome is clipped — arbitrary tall fills
		// (TestBoundaryFillSplit) are left to the normal page-split remnant.
		contentBot := sectionContentBottom(res, pageOps[page], pageTop, pageBot, lastInkBot)

		if pageBot-contentBot < maxGlueEm {
			continue
		}

		clipSectionTrailingBand(res, pageOps[page], pageTop, pageBot, contentBot)
		// A sticky containing block may begin on this page and continue onto
		// the next one. Its page fragment must still end at the last real row;
		// otherwise the unsplit section wash fills the unused page tail.
		clipStickySectionChrome(res, pageOps[page], pageTop, pageBot, contentBot, stickyTargets)
	}

	closePageLeadingSectionChromeWithTargets(res, contentH, stickyTargets)
}

// pageIndexedOps buckets non-fixed ops by their canvas page.
func pageIndexedOps(res *Result, contentH float64) [][]int {
	for idx := range res.Ops {
		if res.Ops[idx].Fixed {
			continue
		}

		if _, ok := checkedFlowPageOfY(res.Ops[idx].Y, contentH); !ok {
			return nil
		}
	}

	maxPage := maxNonFixedOpPage(res.Ops, contentH)

	counts := pageOpCounts(res.Ops, contentH, maxPage)

	pageOps := make([][]int, len(counts))

	for p := range counts {
		pageOps[p] = make([]int, 0, counts[p])
	}

	fillPageOpBuckets(pageOps, res.Ops, contentH)

	return pageOps
}

func maxNonFixedOpPage(ops []Op, contentH float64) int {
	maxPage := 0

	for idx := range ops {
		if ops[idx].Fixed {
			continue
		}

		page, ok := checkedFlowPageOfY(ops[idx].Y, contentH)
		if !ok {
			continue
		}

		if page > maxPage {
			maxPage = page
		}
	}

	return maxPage
}

func pageOpCounts(ops []Op, contentH float64, maxPage int) []int {
	counts := make([]int, maxPage+1)

	for idx := range ops {
		if ops[idx].Fixed {
			continue
		}

		page, ok := checkedFlowPageOfY(ops[idx].Y, contentH)
		if !ok {
			continue
		}

		if page <= maxPage {
			counts[page]++
		}
	}

	return counts
}

func fillPageOpBuckets(pageOps [][]int, ops []Op, contentH float64) {
	for idx := range ops {
		if ops[idx].Fixed {
			continue
		}

		page, ok := checkedFlowPageOfY(ops[idx].Y, contentH)
		if !ok || page >= len(pageOps) {
			continue
		}

		pageOps[page] = append(pageOps[page], idx)
	}
}

// opInPageBand reports whether the op's top edge sits within [pageTop, pageBot).
func opInPageBand(op *Op, pageTop, pageBot float64) bool {
	return op.Y >= pageTop-1e-9 && op.Y < pageBot-1e-9
}

// lastInkBottom scans a page for the lowest text/bullet/image ink bottom.
func lastInkBottom(res *Result, idxs []int, pageTop, pageBot float64) (float64, bool) {
	lastInkBot := pageTop
	hasInk := false

	for _, i := range idxs {
		paintOp := &res.Ops[i]
		if !opInPageBand(paintOp, pageTop, pageBot) {
			continue
		}

		var bot float64

		switch paintOp.Kind {
		case OpText, OpBullet:
			height := opInkHeight(*paintOp)

			if height < minBoxPt {
				height = 4
			}

			bot = paintOp.Y + height
		case OpImage:
			bot = paintOp.Y + paintOp.H
		case OpFillRect, OpStrokeRect, OpLine, OpLinkURI, opKindNoop:
			continue
		}

		hasInk = true

		if bot > lastInkBot {
			lastInkBot = bot
		}
	}

	return lastInkBot, hasInk
}

// stripOrphanRows zeros row-sized fills / rules that sit below the last ink.
// Returns whether anything was stripped.
func stripOrphanRows(res *Result, idxs []int, pageTop, pageBot, lastInkBot float64) bool {
	stripped := false

	for _, i := range idxs {
		paintOp := &res.Ops[i]
		if paintOp.StickyID != 0 || !opInPageBand(paintOp, pageTop, pageBot) {
			continue
		}

		if stripOrphanRowOp(paintOp, lastInkBot) {
			stripped = true
		}
	}

	return stripped
}

// stripOrphanRowOp zeros one row-sized fill or horizontal rule that sits
// below the last ink. Returns whether it was stripped.
func stripOrphanRowOp(paintOp *Op, lastInkBot float64) bool {
	switch paintOp.Kind {
	case OpFillRect, OpStrokeRect:
		// Multi-page frame fragments keep a StrokeMask (open top/bottom).
		// Zeroing their height still leaves a masked top stroke that paints as
		// a full-width hairline on the previous page (fixture-56 page 14).
		if paintOp.StrokeMask != 0 {
			return false
		}

		// Row-sized shells whose center sits below the last ink are
		// empty trailing row backgrounds (not the cell that holds the
		// last text, whose center is at/above the baseline band).
		if paintOp.H <= 0.5 || paintOp.H > 40 {
			return false
		}

		if paintOp.Y+paintOp.H/2 > lastInkBot+0.5 {
			paintOp.H = 0

			return true
		}
	case OpLine:
		if paintOp.H >= 1 {
			return false
		}
		// Horizontal rule below the last ink (empty row separator).
		if paintOp.Y > lastInkBot+0.5 {
			paintOp.Width = 0

			return true
		}
	case OpText, OpImage, OpLinkURI, OpBullet, opKindNoop:
	}

	return false
}

// tightenLastRowChrome shortens the last row's fill so padding under the
// final baseline does not read as another empty row (fixture-31 Row 27 cell).
func tightenLastRowChrome(res *Result, idxs []int, pageTop, pageBot, lastInkBot float64) {
	const underPad = 8.0

	for _, i := range idxs {
		paintOp := &res.Ops[i]
		if paintOp.StickyID != 0 || !opInPageBand(paintOp, pageTop, pageBot) {
			continue
		}

		tightenLastRowOp(paintOp, lastInkBot, underPad)
	}
}

// tightenLastRowOp shortens the last row's fill and pulls the trailing rule
// up under the final baseline.
func tightenLastRowOp(paintOp *Op, lastInkBot, underPad float64) {
	if isLastRowFill(paintOp, lastInkBot, underPad) {
		paintOp.H = lastInkBot + underPad - paintOp.Y
		if paintOp.H < 1 {
			paintOp.H = 1
		}
	}

	if isTrailingRowRule(paintOp, lastInkBot, underPad) {
		paintOp.Y = lastInkBot + underPad
	}
}

// isLastRowFill reports whether paintOp is the row-sized fill that runs under
// the last ink plus more than a small padding.
func isLastRowFill(paintOp *Op, lastInkBot, underPad float64) bool {
	if paintOp.Kind != OpFillRect && paintOp.Kind != OpStrokeRect {
		return false
	}

	return paintOp.H > 0.5 && paintOp.H <= 40 &&
		paintOp.Y < lastInkBot && paintOp.Y+paintOp.H > lastInkBot+underPad+2
}

// isTrailingRowRule reports whether paintOp is the horizontal rule that sits
// just below the last ink's padding band.
func isTrailingRowRule(paintOp *Op, lastInkBot, underPad float64) bool {
	return paintOp.Kind == OpLine && paintOp.H < 1 && paintOp.Width > 0 &&
		paintOp.Y > lastInkBot+underPad+1 && paintOp.Y < lastInkBot+40
}

// sectionContentBottom extends the content bottom over the last row chrome.
func sectionContentBottom(res *Result, idxs []int, pageTop, pageBot, contentBot float64) float64 {
	for _, i := range idxs {
		paintOp := &res.Ops[i]
		if paintOp.StickyID != 0 || !opInPageBand(paintOp, pageTop, pageBot) {
			continue
		}

		contentBot = extendSectionContentBot(paintOp, contentBot)
	}

	return contentBot
}

// extendSectionContentBot extends the content bottom over one row-chrome op.
func extendSectionContentBot(paintOp *Op, contentBot float64) float64 {
	if (paintOp.Kind == OpFillRect || paintOp.Kind == OpStrokeRect) && paintOp.H > 0.5 && paintOp.H <= 40 {
		if bot := paintOp.Y + paintOp.H; bot > contentBot {
			contentBot = bot
		}
	}

	if paintOp.Kind == OpLine && paintOp.H < 1 && paintOp.Width > 0 && paintOp.Y > contentBot {
		contentBot = paintOp.Y
	}

	return contentBot
}

// clipSectionTrailingBand trims section-colored washes/borders that extend
// into the empty tail of the page.
func clipSectionTrailingBand(res *Result, idxs []int, pageTop, pageBot, contentBot float64) {
	for _, i := range idxs {
		paintOp := &res.Ops[i]
		if paintOp.StickyID != 0 || !opInPageBand(paintOp, pageTop, pageBot) {
			continue
		}

		clipTrailingBandOp(res, paintOp, pageTop, pageBot, contentBot)
	}
}

// clipTrailingBandOp trims one trailing continuation-fragment wash or border
// to contentBot. Identity is the op ID plus page-top / overflow geometry, not
// a fixture-named RGB range.
func clipTrailingBandOp(res *Result, paintOp *Op, pageTop, pageBot, contentBot float64) {
	switch paintOp.Kind {
	case OpFillRect:
		if isTrailingContinuationWash(res, paintOp, pageTop, pageBot, contentBot) {
			paintOp.H = contentBot - paintOp.Y
		}
	case OpLine:
		if isTrailingContinuationBorder(res, paintOp, pageTop, pageBot, contentBot) {
			paintOp.H = contentBot - paintOp.Y
		} else if isTrailingContinuationRule(res, paintOp, pageBot, contentBot) {
			paintOp.Y = contentBot
		}
	case OpStrokeRect, OpText, OpImage, OpLinkURI, OpBullet, opKindNoop:
	}
}

// isTrailingContinuationWash reports a continuation-fragment wash that begins
// at the page top and runs past the content bottom. A complete one-page
// block fill keeps its authored height even when the fill color is a cool
// grey and the page has unused space below.
//
// Page paper (html/body background) is not clipped: it has no side rail and
// must fill the unused page tail (fixture-56 page 2 after a short section).
// Section frames with co-located side rails still clip (fixture-31).
func isTrailingContinuationWash(
	res *Result, paintOp *Op, pageTop, pageBot, contentBot float64,
) bool {
	if paintOp.Y > pageTop+1 || paintOp.H <= 40 ||
		paintOp.Y+paintOp.H <= contentBot+1 || paintOp.Y >= contentBot ||
		!isContinuingBlockFragment(res, paintOp, pageBot) {
		return false
	}

	// Paper wash: full-page fill without a matching vertical rail.
	if !hasCoLocatedSideRail(res, paintOp, pageTop, pageBot) {
		return false
	}

	return true
}

const minSideRailHeight = 40.0

func isSideRailOnPage(rail *Op, left, right, pageTop, pageBot float64) bool {
	if rail.Kind != OpLine || rail.W > 1 || rail.H <= minSideRailHeight {
		return false
	}

	if rail.Y+rail.H <= pageTop+1 || rail.Y >= pageBot-1 {
		return false
	}

	return nearLayout(rail.X, left) || nearLayout(rail.X, right)
}

// hasCoLocatedSideRail reports a vertical border near paintOp's left or right
// edge on the same page band — the signature of a framed section card, not
// plain page paper.
func hasCoLocatedSideRail(res *Result, paintOp *Op, pageTop, pageBot float64) bool {
	if res == nil || paintOp == nil {
		return false
	}

	left := paintOp.X
	right := paintOp.X + paintOp.W

	for i := range res.Ops {
		if isSideRailOnPage(&res.Ops[i], left, right, pageTop, pageBot) {
			return true
		}
	}

	return false
}

// isTrailingContinuationBorder reports a page-top side border that belongs
// to a block continuing onto a later page and runs past the content bottom.
func isTrailingContinuationBorder(
	res *Result, paintOp *Op, pageTop, pageBot, contentBot float64,
) bool {
	return paintOp.Y <= pageTop+1 && paintOp.H > 40 &&
		paintOp.Y+paintOp.H > contentBot+1 && paintOp.Y < contentBot &&
		isContinuingBlockFragment(res, paintOp, pageBot)
}

// isTrailingContinuationRule reports a thin closing rule stranded near the
// page bottom, below the last content, that belongs to a continuing block.
func isTrailingContinuationRule(res *Result, paintOp *Op, pageBot, contentBot float64) bool {
	return paintOp.H < 1 && paintOp.Width > 0 &&
		paintOp.Y > contentBot+1 && paintOp.Y > pageBot-30 &&
		isContinuingBlockFragment(res, paintOp, pageBot)
}

// isContinuingBlockFragment reports that paintOp is a page fragment of a
// block that continues after pageBot. Split fragments share Op.ID; a remnant
// that still sits on the page edge is treated as continuing when no ID is
// assigned (legacy/test-constructed ops).
func isContinuingBlockFragment(res *Result, paintOp *Op, pageBot float64) bool {
	if paintOp == nil {
		return false
	}

	if paintOp.ID != 0 && res != nil {
		for i := range res.Ops {
			other := &res.Ops[i]
			if other.ID != paintOp.ID || other == paintOp {
				continue
			}

			if other.Y >= pageBot-1 {
				return true
			}
		}

		return false
	}

	return paintOp.Y+paintOp.H > pageBot-1
}

// clipStickySectionChrome ends sticky-section chrome at the last real row and
// seals the section bottom border when the section continues on the next page.
func clipStickySectionChrome(
	res *Result, idxs []int, pageTop, pageBot, contentBot float64, targets []stickySectionChromeTarget,
) {
	for _, target := range targets {
		for _, i := range idxs {
			paintOp := &res.Ops[i]
			if paintOp.StickyID != 0 || !opInPageBand(paintOp, pageTop, pageBot) || paintOp.H <= 40 ||
				paintOp.Y+paintOp.H <= contentBot+1 {
				continue
			}

			clipStickySectionChromeOp(paintOp, target, contentBot)
		}

		sealStickySectionBottom(res, target, pageBot, contentBot)
	}
}

// clipStickySectionChromeOp trims one sticky-section wash or side border to
// the content bottom.
func clipStickySectionChromeOp(paintOp *Op, target stickySectionChromeTarget, contentBot float64) {
	switch paintOp.Kind {
	case OpFillRect:
		if target.hasBackground && sameRectFrame(paintOp, target) && sameRGB(paintOp, target.background) {
			paintOp.H = contentBot - paintOp.Y
		}
	case OpLine:
		if target.sideMatches(paintOp) {
			paintOp.H = contentBot - paintOp.Y
		}
	case OpStrokeRect, OpText, OpImage, OpLinkURI, OpBullet, opKindNoop:
	}
}

// sealStickySectionBottom appends the section's bottom border when the
// section continues on the next page and no closing border exists yet.
func sealStickySectionBottom(res *Result, target stickySectionChromeTarget, pageBot, contentBot float64) {
	if target.hasBottom && stickySectionContinuesAfterPage(res, target, pageBot) &&
		!hasStickySectionBottomBorder(res, target, contentBot) {
		res.Ops = append(res.Ops, Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpLine, X: target.x, Y: contentBot, W: target.w,
			Width: target.borderBottomWidth,
			R:     target.borderBottom[0], G: target.borderBottom[1], B: target.borderBottom[2],
		})
		// The generated closing rule changes the indexed display-list length.
		invalidateFlowIndex(res)
	}
}

// closePageLeadingSectionChrome keeps continuation-page section washes and
// side borders aligned with their horizontal closing border. Pagination can
// move the last child row without updating the original block chrome. Only
// sections that contain a sticky box are considered, and their own box
// geometry/colors identify the chrome; unrelated wide rules cannot match.
func closePageLeadingSectionChrome(res *Result, contentH float64) {
	if res == nil {
		return
	}

	closePageLeadingSectionChromeWithTargets(res, contentH, stickySectionChromeTargets(res.root))
}

func closePageLeadingSectionChromeWithTargets(res *Result, contentH float64, targets []stickySectionChromeTarget) {
	if res == nil || res.root == nil || contentH <= 0 {
		return
	}

	if len(targets) == 0 {
		return
	}

	maxPage := pageMaxOfOps(res, contentH)

	for page := 1; page <= maxPage; page++ {
		pageTop := float64(page) * contentH
		pageBottom := pageTop + contentH

		for _, target := range targets {
			closeY := closingBorderY(res, target, pageTop, pageBottom)
			if closeY < 0 {
				continue
			}

			clipSectionChromeToCloseY(res, target, pageTop, pageBottom, closeY)
		}
	}
}

// pageMaxOfOps returns the highest non-fixed page index in the display list.
func pageMaxOfOps(res *Result, contentH float64) int {
	maxPage := 0

	for _, op := range res.Ops {
		if op.Fixed {
			continue
		}

		page, ok := checkedFlowPageOfY(op.Y, contentH)
		if !ok {
			continue
		}

		if page > maxPage {
			maxPage = page
		}
	}

	return maxPage
}

// closingBorderY finds the lowest horizontal border matching target on the
// page (or -1 when none).
func closingBorderY(res *Result, target stickySectionChromeTarget, pageTop, pageBottom float64) float64 {
	closeY := -1.0

	for i := range res.Ops {
		paintOp := &res.Ops[i]
		if paintOp.Fixed || paintOp.Kind != OpLine || paintOp.H >= 1 || paintOp.Y <= pageTop+1 || paintOp.Y >= pageBottom {
			continue
		}

		if !sameHorizontalFrame(paintOp, target) || !sameRGB(paintOp, target.borderBottom) {
			continue
		}

		if paintOp.Y > closeY {
			closeY = paintOp.Y
		}
	}

	return closeY
}

// clipSectionChromeToCloseY trims the target's page-leading washes and side
// borders to the closing border so no chrome runs past the last row.
func clipSectionChromeToCloseY(res *Result, target stickySectionChromeTarget, pageTop, pageBottom, closeY float64) {
	for i := range res.Ops {
		paintOp := &res.Ops[i]
		if paintOp.Fixed || paintOp.Y < pageTop-1 || paintOp.Y > pageTop+1 || paintOp.Y >= pageBottom {
			continue
		}

		clipSectionChromeOp(paintOp, target, closeY)
	}
}

// clipSectionChromeOp trims one page-leading wash or side border to closeY.
func clipSectionChromeOp(paintOp *Op, target stickySectionChromeTarget, closeY float64) {
	switch paintOp.Kind {
	case OpFillRect:
		if isSectionChromeWash(paintOp, target, closeY) {
			paintOp.H = closeY - paintOp.Y
		}
	case OpLine:
		if isSectionChromeSideBorder(paintOp, target, closeY) {
			paintOp.H = closeY - paintOp.Y
		}
	case OpStrokeRect, OpText, OpImage, OpLinkURI, OpBullet, opKindNoop:
	}
}

// isSectionChromeWash reports a page-leading section background wash whose
// bottom does not align with closeY.
func isSectionChromeWash(paintOp *Op, target stickySectionChromeTarget, closeY float64) bool {
	return target.hasBackground && paintOp.H > 40 && sameRectFrame(paintOp, target) &&
		sameRGB(paintOp, target.background) && !nearLayout(paintOp.Y+paintOp.H, closeY)
}

// isSectionChromeSideBorder reports a page-leading section side border whose
// bottom does not align with closeY.
func isSectionChromeSideBorder(paintOp *Op, target stickySectionChromeTarget, closeY float64) bool {
	return paintOp.H > 40 && target.sideMatches(paintOp) && !nearLayout(paintOp.Y+paintOp.H, closeY)
}

type stickySectionChromeTarget struct {
	x, y, w           float64
	background        [3]float64
	hasBackground     bool
	borderLeft        [3]float64
	borderRight       [3]float64
	borderBottom      [3]float64
	borderBottomWidth float64
	hasBottom         bool
	hasLeft, hasRight bool
}

func stickySectionChromeTargets(root *box) []stickySectionChromeTarget {
	var targets []stickySectionChromeTarget

	var walk func(b, parent *box)
	walk = func(boxNode, parent *box) {
		if boxNode == nil {
			return
		}

		if boxNode.sticky && parent != nil {
			target := stickyTargetFor(parent)
			if !hasDuplicateTarget(targets, target) {
				targets = append(targets, target)
			}
		}

		for _, child := range boxNode.children {
			walk(child, boxNode)
		}
	}
	walk(root, nil)

	return targets
}

// stickyTargetFor builds the section-chrome match target for a sticky box's
// parent: geometry, border colors and the background wash.
func stickyTargetFor(parent *box) stickySectionChromeTarget {
	sty := parent.style
	target := stickySectionChromeTarget{ //nolint:exhaustruct // intentional zero fields
		x:                 parent.x,
		y:                 parent.y,
		w:                 parent.w,
		borderLeft:        sty.BorderLeft.Color,
		borderRight:       sty.BorderRight.Color,
		borderBottom:      sty.BorderBottom.Color,
		borderBottomWidth: sty.BorderBottom.Width,
		hasBottom:         sty.BorderBottom.Width > 0 && sty.BorderBottom.Style != cssDisplayNone,
		hasLeft:           sty.BorderLeft.Width > 0 && sty.BorderLeft.Style != cssDisplayNone,
		hasRight:          sty.BorderRight.Width > 0 && sty.BorderRight.Style != cssDisplayNone,
	}

	if sty.BGColor[3] > 0 {
		target.background = [3]float64{sty.BGColor[0], sty.BGColor[1], sty.BGColor[2]}
		target.hasBackground = true
	}

	return target
}

// hasDuplicateTarget reports whether targets already holds a target with the
// same frame geometry.
func hasDuplicateTarget(targets []stickySectionChromeTarget, target stickySectionChromeTarget) bool {
	for _, prior := range targets {
		if math.Abs(prior.x-target.x) < 0.01 && math.Abs(prior.y-target.y) < 0.01 && math.Abs(prior.w-target.w) < 0.01 {
			return true
		}
	}

	return false
}

func sameHorizontalFrame(op *Op, target stickySectionChromeTarget) bool {
	return target.hasBottom && math.Abs(op.X-target.x) < 1 && math.Abs(op.W-target.w) < 1
}

func stickySectionContinuesAfterPage(res *Result, target stickySectionChromeTarget, pageBottom float64) bool {
	if res == nil {
		return false
	}

	for _, op := range res.Ops {
		if op.Fixed || (op.Kind != OpText && op.Kind != OpBullet) || op.Y < pageBottom {
			continue
		}

		if op.X >= target.x-1 && op.X <= target.x+target.w+1 {
			return true
		}
	}

	return false
}

func hasStickySectionBottomBorder(res *Result, target stickySectionChromeTarget, posY float64) bool {
	if res == nil {
		return false
	}

	for _, op := range res.Ops {
		if op.Fixed || op.Kind != OpLine || op.H >= 1 || math.Abs(op.Y-posY) >= 1 {
			continue
		}

		if sameHorizontalFrame(&op, target) && sameRGB(&op, target.borderBottom) {
			return true
		}
	}

	return false
}

func sameRectFrame(op *Op, target stickySectionChromeTarget) bool {
	return sameHorizontalFrame(op, target)
}

func (target stickySectionChromeTarget) sideMatches(paintOp *Op) bool {
	if paintOp.W > 1 || math.Abs(paintOp.X-target.x) >= 1 && math.Abs(paintOp.X-(target.x+target.w)) >= 1 {
		return false
	}

	if math.Abs(paintOp.X-target.x) < 1 {
		return target.hasLeft && sameRGB(paintOp, target.borderLeft)
	}

	return target.hasRight && sameRGB(paintOp, target.borderRight)
}

func sameRGB(op *Op, rgb [3]float64) bool {
	return math.Abs(op.R-rgb[0]) < 0.01 && math.Abs(op.G-rgb[1]) < 0.01 && math.Abs(op.B-rgb[2]) < 0.01
}
