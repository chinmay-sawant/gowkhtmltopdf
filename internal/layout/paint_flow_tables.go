package layout

import (
	"math"
)

// rowsIntact keeps each table row on a single page: a row spanning multiple
// pages moves wholly to the next.
func rowsIntact(res *Result, contentH float64) bool {
	var walk func(b *box) bool
	walk = func(boxNode *box) bool {
		changed := false

		for _, c := range boxNode.children {
			if walk(c) {
				changed = true
			}
		}

		for _, row := range boxNode.rows {
			if shiftRowToPage(res, row, contentH) {
				changed = true
			}
		}

		return changed
	}

	return walk(res.root)
}

// normalizeTableRowGaps removes stale vertical gaps left when pagination first
// moves a row to the next page and a later fixpoint pulls the table back.
// Collapsed table rows should remain adjacent when both rows fit on one page.
//
//nolint:cyclop,wsl // table-row geometry checks intentionally stay together
func normalizeTableRowGaps(res *Result, contentH float64) {
	if res == nil || res.root == nil || contentH <= 0 {
		return
	}

	for _, table := range flowBoxList(res) {
		if table.kind != displayTable || len(table.rows) < 2 {
			continue
		}

		for rowIndex := 1; rowIndex < len(table.rows); rowIndex++ {
			if !rowWasPaginationShifted(table.rows[rowIndex]) {
				continue
			}
			_, _, previousTop, previousBottom, previousOK := rowOpGeometry(table.rows[rowIndex-1])

			first, last, currentTop, currentBottom, currentOK := rowOpGeometry(table.rows[rowIndex])

			if !previousOK || !currentOK || first < 0 || last < first {
				continue
			}

			previousPage := int(previousTop / contentH)
			currentPage := int(currentTop / contentH)
			if previousPage != currentPage || int(currentBottom/contentH) != currentPage {
				continue
			}

			gap := currentTop - previousBottom
			if gap <= layoutEpsilon {
				continue
			}

			// Do not collapse the band reserved for a repeated thead on
			// continuation pages (fixture-60 page-2 header overlapping body).
			if table.headerRows > 0 && currentPage > 0 {
				pageTop := float64(currentPage) * contentH
				_, _, _, hdrH := rowSpan(table.rows[:table.headerRows], res)
				if hdrH < 1 {
					hdrH = previousBottom - previousTop
				}
				minTop := pageTop + hdrH
				if currentTop-gap < minTop {
					gap = currentTop - minTop
					if gap <= layoutEpsilon {
						continue
					}
				}
			}

			shiftOpsOnly(res, first, last, -gap)
			shiftTableRowBoxes(table.rows[rowIndex], -gap)
		}
	}
}

func shiftTableRowBoxes(row []*box, deltaY float64) {
	for _, cell := range row {
		shiftTableBox(cell, deltaY)
	}
}

func rowWasPaginationShifted(row []*box) bool {
	for _, cell := range row {
		if cell != nil && cell.paginationShifted {
			return true
		}
	}

	return false
}

func shiftTableBox(boxNode *box, deltaY float64) {
	if boxNode == nil {
		return
	}

	boxNode.y += deltaY
	for _, child := range boxNode.children {
		shiftTableBox(child, deltaY)
	}
}

// shiftRowToPage moves a table row wholly to the next page when it spans
// multiple pages. Returns whether it moved.
//
//nolint:wsl // row shifting and marker updates must stay adjacent.
func shiftRowToPage(res *Result, row []*box, contentH float64) bool {
	if len(row) == 0 {
		return false
	}

	first, last, rowTop, rowBottom, ok := rowOpGeometry(row)
	if !ok {
		return false
	}

	layoutOut, hi := int(rowTop/contentH), int(rowBottom/contentH)
	if hi <= layoutOut {
		return false
	}
	// Move only to the next page start. Using hi*contentH when the
	// row's measured bottom spans multiple pages (e.g. rowspan
	// paint height leaking into rowBoxH) skipped blank pages
	// between filmography and awards on long wiki tables.
	deltaY := float64(layoutOut+1)*contentH - rowTop
	if deltaY <= layoutSlack {
		return false
	}
	// fromY slightly above rowTop so border-collapse grid
	// lines that sit exactly on the row edge (and later
	// rows / chrome below) shift with the cells - otherwise
	// content moves and the grid stays behind (gapped /
	// misaligned music-video tables across page breaks).
	shiftFlowY(res, first, last, rowTop-layoutSlack, deltaY)
	// Keep cell.y in sync with ops so later header-repeat / gap
	// logic (rowYBounds) does not read stale pre-pagination tops.
	shiftTableRowBoxes(row, deltaY)
	for _, cell := range row {
		if cell != nil {
			cell.paginationShifted = true
		}
	}

	return true
}

// rowOpGeometry returns the op range and starting-row geometry of a row:
// opStart..opEnd across its cells and the row band top/bottom (rowspan cells
// use their starting-row height, not the full span paint extent).
func rowOpGeometry(row []*box) (int, int, float64, float64, bool) {
	first, last := -1, -1
	haveGeom := false

	var rowTop, rowBottom float64

	for _, cell := range row {
		first, last = extendRowOpRange(first, last, cell)
		// Use starting-row geometry, not full rowspan paint extent.
		// Rowspan cells emit bottom borders at y+h (full span); scanning
		// those ops made the first row look multi-page and cascaded
		// blank pages (wiki awards tables with rowspan=10+).
		top, bot := rowCellGeometry(cell)

		if !haveGeom {
			rowTop, rowBottom, haveGeom = top, bot, true

			continue
		}

		if top < rowTop {
			rowTop = top
		}

		if bot > rowBottom {
			rowBottom = bot
		}
	}

	return first, last, rowTop, rowBottom, first >= 0 && haveGeom
}

// extendRowOpRange widens the row's op range with one cell's ops.
func extendRowOpRange(first, last int, cell *box) (int, int) {
	if cell.opStart > cell.opEnd {
		return first, last
	}

	if first < 0 {
		first = cell.opStart
	}

	if cell.opEnd > last {
		last = cell.opEnd
	}

	return first, last
}

// rowCellGeometry returns the starting-row band of one cell: rowspan cells
// use their starting-row height, not the full span paint extent.
func rowCellGeometry(cell *box) (float64, float64) {
	top := cell.y

	h := cell.height
	if cell.rowSpan > 1 && cell.rowBoxH > 0 {
		h = cell.rowBoxH
	}

	return top, top + h
}

// repeatTableHeaders clones thead row ops onto every page that continues a
// multi-page table body, shifting body content down by the header height.
// Nested tables: each table repeats only its own thead.
func repeatTableHeaders(res *Result, contentH float64) {
	if res.root == nil || contentH <= 0 {
		return
	}

	for _, tblBox := range tableBoxes(res.root) {
		repeatTableHeaderOnPages(res, tblBox, contentH)
	}
}

// tableBoxes collects every multi-page table with a header band.
func tableBoxes(root *box) []*box {
	var tables []*box

	var walk func(b *box)
	walk = func(b *box) {
		if b.kind == displayTable && b.headerRows > 0 && b.headerRows < len(b.rows) {
			tables = append(tables, b)
		}

		for _, c := range b.children {
			walk(c)
		}
	}
	walk(root)

	return tables
}

// repeatTableHeaderOnPages clones one table's thead onto its continuation
// pages, shifting the body rows down by the header height.
//
//nolint:cyclop // table header repetition algorithm
func repeatTableHeaderOnPages(res *Result, tblBox *box, contentH float64) {
	nHdr := tblBox.headerRows
	if nHdr > len(tblBox.rows) {
		nHdr = len(tblBox.rows)
	}

	hdrFirst, hdrLast, hdrTop, hdrH := rowSpan(tblBox.rows[:nHdr], res)
	if hdrFirst < 0 || hdrH <= 0 {
		return
	}

	firstPage, ok := checkedFlowPageOfY(tblBox.y, contentH)
	if !ok {
		return
	}

	pages := headerContinuationPages(tblBox, firstPage, res, contentH)

	// Process low pages first so later continuation shifts see stable Y.
	pageList := sortedPageKeys(pages)
	for _, page := range pageList {
		if page <= firstPage {
			continue
		}

		pageTop := float64(page) * contentH
		shiftFrom, shiftTo, bodyTop := tableBodyRange(tblBox, page, res, contentH)

		if shiftFrom >= 0 && bodyTop >= 0 && bodyTop < pageTop+hdrH-0.5 {
			dy := pageTop + hdrH - bodyTop
			if dy > 0 {
				shiftFlowY(res, shiftFrom, shiftTo, bodyTop-layoutSlack, dy)
				// Table cells are not always in the flow-box index, so
				// shiftFlowY alone can leave cell.y behind the ops.
				shiftTableBodyBoxesFrom(tblBox, page, contentH, dy)
			}
		}

		placeSliverBodyBelowHeader(res, tblBox, pageTop, hdrH)
		// Guaranteed clearance: body must start at/after the header band.
		ensureBodyBelowRepeatedHeader(res, tblBox, page, pageTop, hdrH, contentH)
		cloneHeaderOps(res, hdrFirst, hdrLast, hdrTop, pageTop)
	}
}

// shiftTableBodyBoxesFrom updates cell.y for body rows whose top is on or
// after page, matching a shiftFlowY applied to those rows' ops.
func shiftTableBodyBoxesFrom(tblBox *box, page int, contentH, dy float64) {
	if tblBox == nil || dy == 0 || contentH <= 0 {
		return
	}

	for _, row := range tblBox.rows[tblBox.headerRows:] {
		_, _, top, _, ok := rowOpGeometry(row)
		if !ok {
			continue
		}

		topPage, pageOK := checkedFlowPageOfY(top, contentH)
		if !pageOK || topPage < page {
			continue
		}

		shiftTableRowBoxes(row, dy)
	}
}

func sortedPageKeys(pages map[int]bool) []int {
	out := make([]int, 0, len(pages))
	for page := range pages {
		out = append(out, page)
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}

	return out
}

// ensureBodyBelowRepeatedHeader pushes the first body ops on a continuation
// page down when they still sit inside the repeated thead band.
func ensureBodyBelowRepeatedHeader(
	res *Result, tblBox *box, page int, pageTop, hdrH, contentH float64,
) {
	if res == nil || tblBox == nil || hdrH <= 0 {
		return
	}

	minTop := pageTop + hdrH
	shiftFrom, shiftTo, bodyTop := tableBodyRange(tblBox, page, res, contentH)
	if shiftFrom < 0 || bodyTop < 0 {
		return
	}
	if bodyTop >= minTop-layoutEpsilon {
		return
	}

	dy := minTop - bodyTop
	if dy > 0 {
		shiftFlowY(res, shiftFrom, shiftTo, bodyTop-layoutSlack, dy)
		shiftTableBodyBoxesFrom(tblBox, page, contentH, dy)
	}
}

const (
	tableSliverLookback      = 8.0
	tableSliverInspectHeight = 20.0
)

// placeSliverBodyBelowHeader pulls whole body rows that still intersect the
// repeated-thead band (or the lookback sliver above the page) down below the
// header. It must move each row's full op range together: shifting only the
// ops that sit inside the band splits borders from text (fixture-60 page-3
// empty gap under thead with body text starting a row-height lower).
func placeSliverBodyBelowHeader(res *Result, tblBox *box, pageTop, hdrH float64) {
	if res == nil || tblBox == nil || hdrH <= 0 {
		return
	}

	target := pageTop + hdrH
	loBound := pageTop - tableSliverLookback
	hiBound := pageTop + hdrH

	for _, row := range tblBox.rows[tblBox.headerRows:] {
		first, last := rowOpRange(row)
		if first < 0 || last < first {
			continue
		}

		rowTop := -1.0
		intersects := false
		for idx := first; idx <= last && idx < len(res.Ops); idx++ {
			posY := res.Ops[idx].Y
			if rowTop < 0 || posY < rowTop {
				rowTop = posY
			}
			if posY >= loBound && posY < hiBound {
				intersects = true
			}
		}

		if !intersects || rowTop < 0 {
			continue
		}

		dy := target - rowTop
		if dy <= layoutSlack {
			continue
		}

		shiftFlowY(res, first, last, rowTop-layoutSlack, dy)
		shiftTableRowBoxes(row, dy)
	}
}

func accumulateSliverOp(paintOp Op, idx int, loBound, hiBound float64, fromIdx, toIdx *int, top, bottom *float64) {
	posY := paintOp.Y
	if posY < loBound || posY >= hiBound {
		return
	}

	if *fromIdx < 0 || idx < *fromIdx {
		*fromIdx = idx
	}

	if idx > *toIdx {
		*toIdx = idx
	}

	if *top < 0 || posY < *top {
		*top = posY
	}

	if bot := posY + opInkHeight(paintOp); bot > *bottom {
		*bottom = bot
	}
}

// sliverBodyOps is the body-row paint sitting in [pageTop-8, pageTop+hdrH).
func sliverBodyOps(tblBox *box, pageTop, hdrH float64, res *Result) (int, int, float64, float64) {
	fromIdx, toIdx := -1, -1
	top, bottom := -1.0, -1.0
	loBound := pageTop - tableSliverLookback
	hiBound := pageTop + hdrH

	for _, row := range tblBox.rows[tblBox.headerRows:] {
		first, last := rowOpRange(row)
		if first < 0 {
			continue
		}

		for idx := first; idx <= last && idx < len(res.Ops); idx++ {
			accumulateSliverOp(res.Ops[idx], idx, loBound, hiBound, &fromIdx, &toIdx, &top, &bottom)
		}
	}

	return fromIdx, toIdx, top, bottom
}

func accumulateNextBodyOp(posY float64, idx int, pageTop float64, fromIdx *int, top *float64) {
	if posY < pageTop-0.5 {
		return
	}

	if *fromIdx < 0 || idx < *fromIdx {
		*fromIdx = idx
	}

	if *top < 0 || posY < *top {
		*top = posY
	}
}

// nextBodyAfterSliver is the first body op on this page that is not in the sliver.
func nextBodyAfterSliver(tblBox *box, pageTop float64, sliverFrom, sliverTo int, res *Result) (int, float64) {
	fromIdx := -1
	top := -1.0

	for _, row := range tblBox.rows[tblBox.headerRows:] {
		first, last := rowOpRange(row)
		if first < 0 {
			continue
		}

		for idx := first; idx <= last && idx < len(res.Ops); idx++ {
			if idx < sliverFrom || idx > sliverTo {
				accumulateNextBodyOp(res.Ops[idx].Y, idx, pageTop, &fromIdx, &top)
			}
		}
	}

	return fromIdx, top
}

func shiftTableBoxesInOpRange(tblBox *box, fromIdx, toIdx int, deltaY float64) {
	if tblBox == nil || deltaY == 0 {
		return
	}

	var walk func(*box)
	walk = func(boxNode *box) {
		if boxNode == nil {
			return
		}

		if boxNode.opStart <= boxNode.opEnd && boxNode.opStart >= fromIdx && boxNode.opEnd <= toIdx {
			boxNode.y += deltaY
		}

		for _, child := range boxNode.children {
			walk(child)
		}
	}

	for _, row := range tblBox.rows {
		for _, cell := range row {
			walk(cell)
		}
	}
}

// headerContinuationPages is the set of pages holding table body rows.
func headerContinuationPages(tblBox *box, _ int, res *Result, contentH float64) map[int]bool {
	pages := map[int]bool{}

	for _, row := range tblBox.rows[tblBox.headerRows:] {
		top := rowYBounds(row, res)
		if top < 0 {
			continue
		}

		page, ok := checkedFlowPageOfY(top, contentH)
		if ok {
			pages[page] = true

			off := math.Mod(top, contentH)
			if off < 0 {
				off += contentH
			}

			if off > contentH-tableSliverLookback {
				nextTop := float64(page+1) * contentH

				sliverFrom, _, sliverTop, _ := sliverBodyOps(tblBox, nextTop, tableSliverInspectHeight, res)
				if sliverFrom >= 0 && sliverTop >= 0 {
					pages[page+1] = true
				}
			}
		}
	}

	return pages
}

// tableBodyRange returns the op range and top Y of the body rows that start
// on the given page.
func tableBodyRange(tblBox *box, page int, res *Result, contentH float64) (int, int, float64) {
	shiftFrom, shiftTo := -1, -1
	bodyTop := -1.0

	for _, row := range tblBox.rows[tblBox.headerRows:] {
		face, lst := rowOpRange(row)
		if face < 0 {
			continue
		}

		top := rowYBounds(row, res)

		topPage, ok := checkedFlowPageOfY(top, contentH)
		if !ok || topPage < page {
			continue
		}

		if bodyTop < 0 || top < bodyTop {
			bodyTop = top
		}

		if shiftFrom < 0 || face < shiftFrom {
			shiftFrom = face
		}

		if lst > shiftTo {
			shiftTo = lst
		}
	}

	return shiftFrom, shiftTo, bodyTop
}

// cloneHeaderOps copies the thead op range to the page top. The clone count
// is known up front, so the display list grows once instead of per op.
// Clones are Pinned so later page-break-before suffix shifts (by op index)
// cannot pull them back onto the previous page.
func cloneHeaderOps(res *Result, hdrFirst, hdrLast int, hdrTop, pageTop float64) {
	if hdrFirst < 0 || hdrFirst > hdrLast || hdrFirst >= len(res.Ops) {
		return
	}

	if hdrLast >= len(res.Ops) {
		hdrLast = len(res.Ops) - 1
	}

	start := len(res.Ops)
	res.Ops = append(res.Ops, make([]Op, hdrLast-hdrFirst+1)...)

	for hdrIndex := hdrFirst; hdrIndex <= hdrLast; hdrIndex++ {
		op := res.Ops[hdrIndex]
		op.Y = pageTop + (op.Y - hdrTop)
		op.Pinned = true
		// Clones are in-flow page furniture, not position:fixed stamps.
		op.Fixed = false
		res.Ops[start+hdrIndex-hdrFirst] = op
	}

	// Header clones extend the display list after the page index was built.
	// Drop the cached buckets so later pagination shifts see the new ops.
	invalidateFlowIndex(res)
}

func rowOpRange(row []*box) (int, int) {
	first, last := -1, -1

	for _, cell := range row {
		if cell.opStart <= cell.opEnd {
			if first < 0 || cell.opStart < first {
				first = cell.opStart
			}

			if cell.opEnd > last {
				last = cell.opEnd
			}
		}
	}

	return first, last
}

func rowYBounds(row []*box, res *Result) float64 {
	first, last := rowOpRange(row)
	if first < 0 || first >= len(res.Ops) {
		return -1
	}

	top, bottom := opBottomEdge(res.Ops[first])

	for k := first + 1; k <= last && k < len(res.Ops); k++ {
		posY, bot := opBottomEdge(res.Ops[k])
		if posY < top {
			top = posY
		}

		if bot > bottom {
			bottom = bot
		}
	}

	return rowCellBounds(row, top, bottom)
}

// opBottomEdge returns an op's top Y and approximate bottom Y.
func opBottomEdge(paintOp Op) (float64, float64) {
	posY := paintOp.Y

	height := opInkHeight(paintOp)

	return posY, posY + height
}

// rowCellBounds returns the row band top for rowYBounds. It keeps the
// op-derived top and must not pull top upward from cell.y: after ops are
// shifted by pagination, stale cell.y would report the wrong page for
// header-repeat body detection (fixture-60 page-2 thead overlap).
func rowCellBounds(row []*box, top, bottom float64) float64 {
	_ = row
	_ = bottom

	return top
}

func rowSpan(rows [][]*box, res *Result) (int, int, float64, float64) {
	first, last, top, bottom, ok := rowsBandGeometry(rows, res)
	if !ok {
		return -1, -1, 0, 0
	}

	height := bottom - top
	if height >= 1 {
		return first, last, top, height
	}

	return first, last, top, sumRowHeights(rows)
}

// rowsBandGeometry returns the op range and Y band of the rows.
func rowsBandGeometry(rows [][]*box, res *Result) (int, int, float64, float64, bool) {
	first, last := -1, -1
	top, bottom := 0.0, 0.0
	set := false

	for _, row := range rows {
		rowFirst, rowLast, rowTop, rowBottom, rowOK := rowBandExtent(row, res)
		if !rowOK {
			continue
		}

		first, last = extendBandRange(first, last, rowFirst, rowLast)
		top, bottom, set = extendBandBounds(top, bottom, rowTop, rowBottom, set)
	}

	return first, last, top, bottom, first >= 0 && set
}

// rowBandExtent returns the op range and Y band of one row.
func rowBandExtent(row []*box, _ *Result) (int, int, float64, float64, bool) {
	first, last, top, bottom, ok := rowOpGeometry(row)
	if !ok {
		return -1, -1, 0, 0, false
	}

	return first, last, top, bottom, true
}

// extendBandRange widens the band's op range with one row's range.
func extendBandRange(first, last, rowFirst, rowLast int) (int, int) {
	if first < 0 || rowFirst < first {
		first = rowFirst
	}

	if rowLast > last {
		last = rowLast
	}

	return first, last
}

// extendBandBounds widens the band's Y bounds with one row's band.
func extendBandBounds(top, bottom, rowTop, rowBottom float64, set bool) (float64, float64, bool) {
	if !set || rowTop < top {
		top = rowTop
	}

	if !set || rowBottom > bottom {
		bottom = rowBottom
	}

	return top, bottom, true
}

// sumRowHeights falls back to the sum of the cells' heights when the op band
// is too thin to measure (rows without op Y spread).
func sumRowHeights(rows [][]*box) float64 {
	height := 0.0

	for _, row := range rows {
		rowH := 0.0
		for _, cell := range row {
			if cell.height > rowH {
				rowH = cell.height
			}
		}

		height += rowH
	}

	return height
}
