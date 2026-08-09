package layout

import (
	"math"
	"strconv"
	"strings"

	"gowkhtmltopdf/internal/html"
)

// Grid auto-flow keywords (grid-auto-flow longhand values).
const (
	gridFlowColumnDense = "column dense"
	gridFlowDense       = "dense"
	gridFlowRowDense    = "row dense"
)

// buildGrid lays out a CSS grid Stage B/C subset: column/row tracks (incl.
// minmax/fr/auto/min-content/max-content lite), independent gaps, template
// areas + named grid-area, auto-flow row/column (sparse or dense), column/row
// spanning, and justify/align-items/self.
//
// ponytail: display:subgrid → ordinary grid (no parent template inherit).
// ponytail: grid-template-*: masonry keyword stripped → dense auto-flow
// (no L3 shortest-stack pack). Upgrade if report templates need either.
func (e *engine) buildGrid(node *html.Node, sty ResolvedStyle, availW, posX, posY float64) *box {
	if sty.Display == displaySubgrid {
		sty.Display = displayGrid
	}

	// Masonry keyword → empty track list → auto-flow dense grid.
	sty.GridTemplateColumns = stripMasonryKeyword(sty.GridTemplateColumns)
	sty.GridTemplateRows = stripMasonryKeyword(sty.GridTemplateRows)

	ml := e.scalePt(sty.MarginLeft)
	boxNode := &box{ //nolint:exhaustruct // intentional zero fields
		node: node, style: e.stylePtr(node), kind: displayBlock, x: posX + ml, y: posY,
	}
	boxNode.w = resolveUsedWidth(sty, availW, e)
	contentX, contentW := e.contentBox(boxNode.x, boxNode.w, sty)
	contentStart := len(e.ops)
	curY := e.scalePt(sty.PaddingTop) + e.scalePt(sty.BorderTop.Width)

	rowGap, columnGap := e.styleGaps(sty)

	areas := parseGridTemplateAreas(sty.GridTemplateAreas)
	colDefs := gridColumnDefs(sty.GridTemplateColumns, areas.cols)

	kids := collectGridKids(e, node)

	// Intrinsic measure lite for min-content / max-content column mins.
	colIntrinsics := measureTrackIntrinsics(e, kids, len(colDefs), true)

	cols := resolveGridTrackSizes(colDefs, contentW, columnGap, e, colIntrinsics)
	if len(cols) == 0 {
		cols = []float64{contentW}
	}

	contentH := resolveContentHeight(sty, e)

	columnMajor, densePack := gridAutoFlowMode(sty.GridAutoFlow)
	placed := placeGridItems(e, kids, areas, newGridOccupation(len(cols)), len(cols), columnMajor, densePack)

	numRows := gridRowCount(placed, areas)

	rowTemplate := strings.TrimSpace(sty.GridTemplateRows)
	definiteRows := contentH >= 0 && rowTemplate != "" &&
		strings.ToLower(rowTemplate) != cssDisplayNone

	rows, numRows := resolveGridRows(e, sty, kids, numRows, contentH, rowGap, definiteRows)
	pboxes := measureGridPreferredHeights(e, placed, cols, columnGap, rowGap, contentX, curY, posY, rows, definiteRows)

	rowYs := emitGridBoxes(e, sty, boxNode, pboxes, rows, rowGap, posY, curY)

	usedH := curY
	if numRows > 0 {
		usedH = rowYs[numRows-1] + rows[numRows-1]
	}

	usedH = resolveGridUsedHeight(e, sty, usedH, contentH)

	boxNode.height = usedH
	e.prependChrome(contentStart, boxNode, sty, boxNode.x, posY, boxNode.w, boxNode.height)

	return boxNode
}

// gridColumnDefs expands the column track list, padding with flexible tracks
// when fewer defs than template-area columns are declared.
func gridColumnDefs(raw string, areaCols int) []gridTrackDef {
	colDefs := parseGridTrackDefs(raw)

	if len(colDefs) == 0 {
		n := areaCols
		if n < 1 {
			n = 1
		}

		colDefs = make([]gridTrackDef, n)
		for i := range colDefs {
			colDefs[i] = flexibleTrack(1)
		}

		return colDefs
	}

	if need := areaCols - len(colDefs); need > 0 {
		padded := make([]gridTrackDef, areaCols)
		copy(padded, colDefs)

		for i := len(colDefs); i < areaCols; i++ {
			padded[i] = flexibleTrack(1)
		}

		return padded
	}

	return colDefs
}

// collectGridKids returns the element children that participate in grid
// layout (display:none is skipped).
func collectGridKids(eng *engine, node *html.Node) []*html.Node {
	kids := make([]*html.Node, 0, len(node.Children))

	for _, child := range node.Children {
		if child.Type != html.ElementNode {
			continue
		}

		if eng.styles[child].Display == cssDisplayNone {
			continue
		}

		kids = append(kids, child)
	}

	return kids
}

// gridCell is a placed grid item with its resolved grid coordinates.
type gridCell struct {
	n            *html.Node
	col, colSpan int
	row, rowSpan int
}

// gridOccupation tracks placed grid cells so auto-flow items skip overlaps.
type gridOccupation struct {
	nCols int
	byRow map[int]map[int]bool
}

func newGridOccupation(nCols int) *gridOccupation {
	return &gridOccupation{nCols: nCols, byRow: map[int]map[int]bool{}}
}

func (o *gridOccupation) ensure(row int) {
	if o.byRow[row] == nil {
		o.byRow[row] = map[int]bool{}
	}
}

func (o *gridOccupation) freeAt(row, col, rowSpan, colSpan int) bool {
	for r := range rowSpan {
		o.ensure(row + r)

		for i := range colSpan {
			c := col + i
			if c >= o.nCols || o.byRow[row+r][c] {
				return false
			}
		}
	}

	return true
}

func (o *gridOccupation) mark(row, col, rowSpan, colSpan int) {
	for r := range rowSpan {
		o.ensure(row + r)

		for i := range colSpan {
			o.byRow[row+r][col+i] = true
		}
	}
}

// placeGridItems runs CSS Grid auto-placement over the collected kids and
// returns the placed cells, growing the implicit row band as needed.
func placeGridItems(
	eng *engine,
	kids []*html.Node,
	areas gridTemplateAreasMap,
	occ *gridOccupation,
	nCols int,
	columnMajor, densePack bool,
) []gridCell {
	// Implicit row band for column-major auto flow (grid-template-rows empty).
	implicitRows := areas.rows
	if implicitRows < 1 {
		implicitRows = (len(kids) + nCols - 1) / nCols
	}

	minRows := implicitRows
	if minRows < 1 {
		minRows = 1
	}

	placed := make([]gridCell, 0, len(kids))
	cursorRow, cursorCol := 0, 0

	for _, kid := range kids {
		row, col, rowSpan, colSpan := planGridItemPlacement(
			*eng.styles[kid], areas, occ, nCols, cursorRow, cursorCol,
			columnMajor, densePack, minRows,
		)
		occ.mark(row, col, rowSpan, colSpan)

		placed = append(placed, gridCell{n: kid, col: col, colSpan: colSpan, row: row, rowSpan: rowSpan})

		if columnMajor {
			cursorRow = row + rowSpan
			cursorCol = col

			if implicitRows > 0 && cursorRow >= implicitRows {
				cursorRow = 0
				cursorCol++
			}
		} else {
			cursorRow, cursorCol = row, col+colSpan
			if cursorCol >= nCols {
				cursorCol = 0
				cursorRow++
			}
		}
	}

	return placed
}

// gridItemSpans resolves the row/col start (0-based, -1 = auto) and span for
// one item. definite is true when both axes have an explicit or area-derived
// position.
func gridItemSpans(sty ResolvedStyle, areas gridTemplateAreasMap, nCols int) (int, int, int, int, bool) {
	colSpan := sty.GridColumnSpan
	if colSpan < 1 {
		colSpan = 1
	}

	rowSpan := sty.GridRowSpan
	if rowSpan < 1 {
		rowSpan = 1
	}

	colStart := sty.GridColumnStart - 1 // 0-based; -1 = auto
	rowStart := sty.GridRowStart - 1
	definite := false

	if name := strings.TrimSpace(sty.GridArea); name != "" {
		if rect, ok := resolveNamedGridArea(areas, name); ok {
			rowStart, colStart = rect.row, rect.col
			rowSpan, colSpan = rect.rowSpan, rect.colSpan
			definite = true
		}
	}

	if colSpan > nCols {
		colSpan = nCols
	}

	if rowStart >= 0 && colStart >= 0 {
		definite = true
	}

	return rowStart, colStart, rowSpan, colSpan, definite
}

// planGridItemPlacement plans the row/col and span for one grid item.
func planGridItemPlacement(
	sty ResolvedStyle,
	areas gridTemplateAreasMap,
	occ *gridOccupation,
	nCols, cursorRow, cursorCol int,
	columnMajor, densePack bool,
	minRows int,
) (int, int, int, int) {
	rowStart, colStart, rowSpan, colSpan, definite := gridItemSpans(sty, areas, nCols)

	var row, col int

	switch {
	case definite:
		row, col = placeGridItemDefinite(occ, rowStart, colStart, rowSpan, colSpan)
	case colStart >= 0:
		row, col = placeGridItemColumnPinned(occ, colStart, cursorRow, densePack, rowSpan, colSpan)
	case rowStart >= 0:
		row, col = placeGridItemRowPinned(occ, rowStart, nCols, rowSpan, colSpan)
	default:
		row, col = placeGridItemAutoFlow(occ, nCols, cursorRow, cursorCol, columnMajor, densePack, rowSpan, colSpan, minRows)
	}

	return row, col, rowSpan, colSpan
}

// placeGridItemDefinite drops an explicitly positioned item into its cell,
// pushing down on conflict.
func placeGridItemDefinite(occ *gridOccupation, rowStart, colStart, rowSpan, colSpan int) (int, int) {
	row, col := rowStart, colStart
	if row < 0 {
		row = 0
	}

	if col < 0 {
		col = 0
	}

	for !occ.freeAt(row, col, rowSpan, colSpan) {
		row++
	}

	return row, col
}

// placeGridItemColumnPinned places an item with an explicit column, starting
// at the auto-flow cursor unless dense packing.
func placeGridItemColumnPinned(
	occ *gridOccupation,
	colStart, cursorRow int,
	densePack bool,
	rowSpan, colSpan int,
) (int, int) {
	row := 0
	if !densePack {
		row = cursorRow
	}

	for !occ.freeAt(row, colStart, rowSpan, colSpan) {
		row++
	}

	return row, colStart
}

// placeGridItemRowPinned places an item with an explicit row, scanning
// columns left-to-right and wrapping to the next row on overflow.
func placeGridItemRowPinned(occ *gridOccupation, rowStart, nCols int, rowSpan, colSpan int) (int, int) {
	row, col := rowStart, 0

	for {
		if col+colSpan > nCols {
			col = 0
			row++

			continue
		}

		if occ.freeAt(row, col, rowSpan, colSpan) {
			break
		}

		col++
	}

	return row, col
}

// placeGridItemAutoFlow places an item by the grid-auto-flow cursor.
func placeGridItemAutoFlow(
	occ *gridOccupation,
	nCols, cursorRow, cursorCol int,
	columnMajor, densePack bool,
	rowSpan, colSpan, minRows int,
) (int, int) {
	startRow, startCol := cursorRow, cursorCol
	if densePack {
		startRow, startCol = 0, 0
	}

	if columnMajor {
		return findGridSlotColumnMajor(occ.freeAt, nCols, startRow, startCol, rowSpan, colSpan, minRows)
	}

	return findGridSlotRowMajor(occ.freeAt, nCols, startRow, startCol, rowSpan, colSpan)
}

// gridRowCount derives the implicit row count from placed items and areas.
func gridRowCount(placed []gridCell, areas gridTemplateAreasMap) int {
	maxRow := 0

	for _, p := range placed {
		end := p.row + p.rowSpan - 1
		if end > maxRow {
			maxRow = end
		}
	}

	numRows := maxRow + 1
	if areas.rows > numRows {
		numRows = areas.rows
	}

	if numRows < 1 {
		numRows = 1
	}

	return numRows
}

// resolveGridRows sizes the row tracks, returning the final row count.
func resolveGridRows(
	eng *engine,
	sty ResolvedStyle,
	kids []*html.Node,
	numRows int,
	contentH, rowGap float64,
	definiteRows bool,
) ([]float64, int) {
	var rows []float64

	if definiteRows {
		rowDefs := parseGridTrackDefs(sty.GridTemplateRows)
		// Pad/truncate defs to placed row count when template is shorter.
		for len(rowDefs) < numRows {
			rowDefs = append(rowDefs, flexibleTrack(1))
		}

		rowIntrinsics := measureTrackIntrinsics(eng, kids, len(rowDefs), false)
		rows = resolveGridTrackSizes(rowDefs, contentH, rowGap, eng, rowIntrinsics)
	}

	switch {
	case len(rows) == 0:
		rows = make([]float64, numRows)

		if mins := parseGridTrackFixedMins(sty.GridTemplateRows, eng); len(mins) > 0 {
			for i := 0; i < numRows && i < len(mins); i++ {
				if mins[i] > 0 {
					rows[i] = mins[i]
				}
			}
		}
	case len(rows) < numRows:
		rows = padGridRowSizes(rows, numRows)
	case len(rows) > numRows:
		numRows = len(rows)
	}

	return rows, numRows
}

// padGridRowSizes extends a row-size slice to n entries, zero-filling.
func padGridRowSizes(rows []float64, n int) []float64 {
	padded := make([]float64, n)
	copy(padded, rows)

	return padded
}

// gridPlacedBox is a measured grid item with its cell geometry.
type gridPlacedBox struct {
	gridCell
	b         *box
	cellW, cx float64
	prefH     float64
}

// gridCellExtent returns the content width and x offset for one placed cell.
func gridCellExtent(cols []float64, columnGap, contentX float64, page gridCell) (float64, float64) {
	contW := 0.0

	curX := contentX
	for j := range page.col {
		curX += cols[j] + columnGap
	}

	for j := range page.colSpan {
		contW += cols[page.col+j]
		if j > 0 {
			contW += columnGap
		}
	}

	return contW, curX
}

// measureGridPreferredHeights measures items without emitting ops so auto
// row tracks can be sized before final placement.
func measureGridPreferredHeights(
	eng *engine,
	placed []gridCell,
	cols []float64,
	columnGap, rowGap, contentX, curY, posY float64,
	rows []float64,
	definiteRows bool,
) []gridPlacedBox {
	pboxes := make([]gridPlacedBox, 0, len(placed))

	for _, page := range placed {
		contW, curX := gridCellExtent(cols, columnGap, contentX, page)

		was := eng.noEmit
		eng.noEmit = true
		mb := eng.build(page.n, contW, curX, posY+curY)
		eng.noEmit = was

		prefH := 0.0
		if mb != nil {
			prefH = mb.height
		}

		pboxes = append(pboxes, gridPlacedBox{
			gridCell: page,
			b:        nil,
			cellW:    contW,
			cx:       curX,
			prefH:    prefH,
		})

		if !definiteRows && page.rowSpan == 1 && prefH > rows[page.row] {
			rows[page.row] = prefH
		}
	}

	if !definiteRows {
		growSpanningGridRows(rows, pboxes, rowGap)
	}

	return pboxes
}

// growSpanningGridRows distributes extra height across the rows a spanning
// item occupies so its preferred height fits.
func growSpanningGridRows(rows []float64, pboxes []gridPlacedBox, rowGap float64) {
	for _, pbox := range pboxes {
		if pbox.rowSpan <= 1 {
			continue
		}

		sum := 0.0
		for r := range pbox.rowSpan {
			sum += rows[pbox.row+r]
			if r > 0 {
				sum += rowGap
			}
		}

		if pbox.prefH > sum {
			extra := (pbox.prefH - sum) / float64(pbox.rowSpan)
			for r := range pbox.rowSpan {
				rows[pbox.row+r] += extra
			}
		}
	}
}

// emitGridBoxes builds and positions each item, returning the row y-offsets.
func emitGridBoxes(
	eng *engine,
	sty ResolvedStyle,
	boxNode *box,
	pboxes []gridPlacedBox,
	rows []float64,
	rowGap, posY, curY float64,
) []float64 {
	rowYs := make([]float64, len(rows))
	rowYs[0] = curY

	for r := 1; r < len(rows); r++ {
		rowYs[r] = rowYs[r-1] + rows[r-1] + rowGap
	}

	containerJustify := sty.JustifyItems
	if containerJustify == "" {
		containerJustify = fxStretch
	}

	containerAlign := sty.AlignItems
	if containerAlign == "" {
		containerAlign = fxStretch
	}

	for i := range pboxes {
		emitGridItem(eng, boxNode, &pboxes[i], rows, rowGap, posY, rowYs, containerJustify, containerAlign)
	}

	return rowYs
}

// emitGridItem builds one item's box and shifts it into its cell.
func emitGridItem(
	eng *engine,
	boxNode *box,
	pbox *gridPlacedBox,
	rows []float64,
	rowGap, posY float64,
	rowYs []float64,
	containerJustify, containerAlign string,
) {
	cellH := gridItemCellHeight(*pbox, rows, rowGap)

	targetX := pbox.cx
	targetY := posY + rowYs[pbox.row]

	cstate := eng.styles[pbox.n]
	justify := gridItemJustify(*cstate, containerJustify)
	align := gridItemAlign(*cstate, containerAlign)

	buildH := gridStretchBuildHeight(align, cellH, *cstate)

	var cblock *box

	if buildH > 0 {
		override := *cstate
		override.Height = buildH
		override.HeightPercent = -1
		override.BoxSizing = borderBox
		cblock = eng.buildWithStyle(pbox.n, &override, pbox.cellW, pbox.cx, targetY)
	} else {
		cblock = eng.build(pbox.n, pbox.cellW, pbox.cx, targetY)
	}

	if cblock == nil {
		return
	}

	pbox.b = cblock

	deltaX := targetX - pbox.b.x
	deltaY := targetY - pbox.b.y
	deltaX += gridAlignOffset(justify, pbox.cellW, pbox.b.w)
	deltaY += gridAlignOffset(align, cellH, pbox.b.height)

	eng.shiftBoxOps(pbox.b, deltaX, deltaY)
	pbox.b.x += deltaX
	pbox.b.y += deltaY
	boxNode.children = append(boxNode.children, pbox.b)
}

// gridItemCellHeight sums the track sizes a spanning item occupies (with gaps).
func gridItemCellHeight(pbox gridPlacedBox, rows []float64, rowGap float64) float64 {
	cellH := 0.0

	for r := range pbox.rowSpan {
		cellH += rows[pbox.row+r]
		if r > 0 {
			cellH += rowGap
		}
	}

	return cellH
}

// gridItemJustify resolves the effective justify-self for one item.
func gridItemJustify(cstate ResolvedStyle, container string) string {
	justify := cstate.JustifySelf
	if justify == "" || justify == overflowAuto {
		justify = container
	}

	return justify
}

// gridItemAlign resolves the effective align-self for one item.
func gridItemAlign(cstate ResolvedStyle, container string) string {
	align := cstate.AlignSelf
	if align == "" || align == overflowAuto {
		align = container
	}

	return align
}

// gridStretchBuildHeight returns the box height used to stretch a grid item
// into its cell, or -1 when the item is not stretched.
func gridStretchBuildHeight(align string, cellH float64, cstate ResolvedStyle) float64 {
	// Default stretch: border box fills the grid area (CSS Grid §10.2).
	if (align == fxStretch || align == "") && cellH > 0 &&
		cstate.Height < 0 && cstate.HeightPercent < 0 {
		return cellH
	}

	return -1
}

// resolveGridUsedHeight bumps the used height to the definite height and the
// min border-box floor.
func resolveGridUsedHeight(eng *engine, sty ResolvedStyle, usedH, contentH float64) float64 {
	usedH += eng.scalePt(sty.PaddingBottom)

	if sty.Height >= 0 {
		height := eng.scalePt(sty.Height)
		if sty.BoxSizing != borderBox {
			height += eng.scalePt(sty.PaddingTop) + eng.scalePt(sty.PaddingBottom) +
				eng.scalePt(sty.BorderTop.Width) + eng.scalePt(sty.BorderBottom.Width)
		}

		if usedH < height {
			usedH = height
		}
	}

	minBorderH := eng.scalePt(sty.PaddingTop) + eng.scalePt(sty.BorderTop.Width) +
		eng.scalePt(sty.PaddingBottom) + eng.scalePt(sty.BorderBottom.Width)
	if contentH >= 0 {
		minBorderH += contentH
	}

	if usedH < minBorderH {
		usedH = minBorderH
	}

	return usedH
}

// stripMasonryKeyword clears a lone "masonry" track list so layout falls
// through to ordinary dense auto-flow (no L3 pack).
func stripMasonryKeyword(raw string) string {
	if strings.ToLower(strings.TrimSpace(raw)) == "masonry" {
		return ""
	}

	return raw
}

// --- Width / height helpers -------------------------------------------------

// resolveUsedWidth computes border-box width. WidthPercent against a
// non-positive (indefinite) availW is treated as auto (fill remaining).
// Shared by flex/grid/multicol (block keeps its own min/max/margin-auto path).
func resolveUsedWidth(sty ResolvedStyle, availW float64, engN *engine) float64 {
	ml, mr := engN.scalePt(sty.MarginLeft), engN.scalePt(sty.MarginRight)

	width := availW - ml - mr
	if width < 0 {
		width = 0
	}

	if sty.WidthPercent >= 0 {
		// Cyclic % → auto (keep fill-remaining width).
		if availW > 0 && !math.IsInf(availW, 0) && availW < 1e12 {
			width = availW * sty.WidthPercent / cssPercent
		}
	} else if sty.Width >= 0 {
		width = engN.scalePt(sty.Width)
		if sty.BoxSizing != borderBox {
			width += engN.scalePt(sty.PaddingLeft) + engN.scalePt(sty.PaddingRight) +
				engN.scalePt(sty.BorderLeft.Width) + engN.scalePt(sty.BorderRight.Width)
		}
	}

	return width
}

// resolveContentHeight returns definite content-box height, or -1 when auto.
// HeightPercent only resolves when Height was already made definite by a parent
// stretch; unresolved HeightPercent (indefinite CB) is treated as auto.
// Shared by flex/grid/multicol.
func resolveContentHeight(sty ResolvedStyle, engN *engine) float64 {
	if sty.HeightPercent >= 0 && sty.Height < 0 {
		// Cyclic % honesty: indefinite containing block → auto.
		return -1
	}

	if sty.Height < 0 {
		return -1
	}

	height := engN.scalePt(sty.Height)
	if sty.BoxSizing == borderBox {
		height -= engN.scalePt(sty.PaddingTop) + engN.scalePt(sty.PaddingBottom) +
			engN.scalePt(sty.BorderTop.Width) + engN.scalePt(sty.BorderBottom.Width)
	}

	if height < 0 {
		height = 0
	}

	return height
}

// --- Gaps / alignment -------------------------------------------------------

// styleGaps returns scaled row/column gaps. When both longhands are unset (0),
// fall back to the Gap shorthand for both axes. Shared by flex and grid.
func (e *engine) styleGaps(sty ResolvedStyle) (float64, float64) {
	if sty.RowGap == 0 && sty.ColumnGap == 0 {
		g := e.scalePt(sty.Gap)

		return g, g
	}

	return e.scalePt(sty.RowGap), e.scalePt(sty.ColumnGap)
}

// gridAlignOffset returns the inline/block offset for start/end/center/stretch.
// stretch is treated as start (lite).
func gridAlignOffset(value string, cell, item float64) float64 {
	switch value {
	case fxFlexEnd, fxEnd, cssVerticalAlignBottom, floatRight:
		if cell > item {
			return cell - item
		}
	case fxCenter:
		if cell > item {
			return (cell - item) / two
		}
	}

	return 0
}

// --- Areas + auto-flow placement (kept separate from track parsing) ---------

// gridAreaRect is a 0-based rectangle covering a named template area.
type gridAreaRect struct {
	row, col, rowSpan, colSpan int
}

// gridTemplateAreasMap holds the parsed grid-template-areas name → rect map.
type gridTemplateAreasMap struct {
	names      map[string]gridAreaRect
	rows, cols int
}

// parseGridTemplateAreas parses quoted area rows into a name map.
// Tokens "none", ".", and empty cells are holes (no name).
func parseGridTemplateAreas(raw string) gridTemplateAreasMap {
	out := gridTemplateAreasMap{names: map[string]gridAreaRect{}} //nolint:exhaustruct // intentional zero fields
	raw = strings.TrimSpace(raw)

	if raw == "" || strings.EqualFold(raw, cssDisplayNone) {
		return out
	}

	// Collect quoted strings: "a b" "c d" or 'a b'
	rows := collectGridAreaRows(raw)
	if len(rows) == 0 {
		return out
	}

	out.rows = len(rows)
	for _, r := range rows {
		if len(r) > out.cols {
			out.cols = len(r)
		}
	}

	// Pad short rows with "." so indexing is safe.
	for i := range rows {
		for len(rows[i]) < out.cols {
			rows[i] = append(rows[i], ".")
		}
	}

	out.names = accumulateGridAreaBounds(rows)

	return out
}

// collectGridAreaRows extracts the quoted template-area rows from raw text.
func collectGridAreaRows(raw string) [][]string {
	var rows [][]string

	for idx := 0; idx < len(raw); {
		for idx < len(raw) && isTrackWhitespace(raw[idx]) {
			idx++
		}

		if idx >= len(raw) {
			break
		}

		quote := raw[idx]
		if quote != '"' && quote != '\'' {
			// Unquoted token — skip (invalid lite)
			idx = skipGridUnquotedToken(raw, idx)

			continue
		}

		cell, nextIdx := scanGridAreaCell(raw, idx, quote)
		idx = nextIdx

		toks := strings.Fields(cell)
		if len(toks) == 0 {
			continue
		}

		rows = append(rows, toks)
	}

	return rows
}

// skipGridUnquotedToken advances past an unquoted template-area token.
func skipGridUnquotedToken(raw string, idx int) int {
	for idx < len(raw) && raw[idx] != ' ' && raw[idx] != '\t' && raw[idx] != '"' && raw[idx] != '\'' {
		idx++
	}

	return idx
}

// scanGridAreaCell advances to the closing quote and returns the cell text
// plus the index just past the closing quote.
func scanGridAreaCell(raw string, idx int, quote byte) (string, int) {
	idx++
	start := idx

	for idx < len(raw) && raw[idx] != quote {
		idx++
	}

	cell := raw[start:idx]

	if idx < len(raw) {
		idx++ // closing quote
	}

	return cell, idx
}

// gridAreaBounds tracks the running bounds of one named template area.
type gridAreaBounds struct {
	r0, c0, r1, c1 int
	seen           bool
}

// accumulateGridAreaBounds folds each area name onto its bounding rectangle.
func accumulateGridAreaBounds(rows [][]string) map[string]gridAreaRect {
	acc := map[string]*gridAreaBounds{}

	for runic, row := range rows {
		for child, name := range row {
			if name == "." || strings.EqualFold(name, cssDisplayNone) {
				continue
			}

			cur := acc[name]
			if cur == nil {
				acc[name] = &gridAreaBounds{r0: runic, c0: child, r1: runic, c1: child, seen: true}

				continue
			}

			extendGridAreaBounds(cur, runic, child)
		}
	}

	out := make(map[string]gridAreaRect, len(acc))
	for name, b := range acc {
		out[name] = gridAreaRect{
			row:     b.r0,
			col:     b.c0,
			rowSpan: b.r1 - b.r0 + 1,
			colSpan: b.c1 - b.c0 + 1,
		}
	}

	return out
}

// extendGridAreaBounds grows a bounds rect to include (row, col).
func extendGridAreaBounds(cur *gridAreaBounds, runic, child int) {
	if runic < cur.r0 {
		cur.r0 = runic
	}

	if runic > cur.r1 {
		cur.r1 = runic
	}

	if child < cur.c0 {
		cur.c0 = child
	}

	if child > cur.c1 {
		cur.c1 = child
	}
}

// resolveNamedGridArea looks up a custom-ident in the areas map.
func resolveNamedGridArea(areas gridTemplateAreasMap, name string) (gridAreaRect, bool) {
	if areas.names == nil {
		return gridAreaRect{}, false //nolint:exhaustruct // intentional zero fields
	}

	rect, ok := areas.names[name]

	return rect, ok
}

// gridAutoFlowMode returns column-major and dense flags from GridAutoFlow.
func gridAutoFlowMode(flow string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(flow)) {
	case fxCol:
		return true, false
	case gridFlowColumnDense:
		return true, true
	case gridFlowDense, gridFlowRowDense:
		return false, true
	default:
		return false, false
	}
}

type gridFreeFn func(row, col, rowSpan, colSpan int) bool

// findGridSlotRowMajor searches row-major from (startRow, startCol).
func findGridSlotRowMajor(free gridFreeFn, nCols, startRow, startCol, rowSpan, colSpan int) (int, int) {
	row, col := startRow, startCol

	for {
		if col+colSpan > nCols {
			col = 0
			row++

			continue
		}

		if free(row, col, rowSpan, colSpan) {
			return row, col
		}

		col++
	}
}

// findGridSlotColumnMajor searches column-major with an expanding implicit
// row limit (needed when grid-template-rows is empty). Sparse callers pass a
// cursor; dense callers pass (0,0). minRows is the initial row band.
func findGridSlotColumnMajor(free gridFreeFn, nCols, startRow, startCol, rowSpan, colSpan, minRows int) (int, int) {
	if colSpan > nCols {
		colSpan = nCols
	}

	if minRows < rowSpan {
		minRows = rowSpan
	}

	for maxRows := minRows; maxRows < 4096; maxRows++ {
		for col := 0; col+colSpan <= nCols; col++ {
			for row := 0; row+rowSpan <= maxRows; row++ {
				if row < startRow || (row == startRow && col < startCol) {
					continue
				}

				if free(row, col, rowSpan, colSpan) {
					return row, col
				}
			}
		}
	}

	return findGridSlotRowMajor(free, nCols, startRow, startCol, rowSpan, colSpan)
}

// --- Track parsing (minmax / fr / intrinsic) --------------------------------

type trackSizeKind int

const (
	trackFixed trackSizeKind = iota
	trackFr
	trackAuto
	trackMinContent
	trackMaxContent
)

// gridTrackSize is one side of a track (min or max).
type gridTrackSize struct {
	kind trackSizeKind
	val  float64 // pt for fixed (pre-scale raw pt), or fr coefficient
}

// gridTrackDef is minmax(min, max); a bare size is stored as minmax(size, size)
// except fr → minmax(auto, fr) per CSS Grid.
type gridTrackDef struct {
	min, max gridTrackSize
}

func flexibleTrack(frac float64) gridTrackDef {
	if frac <= 0 {
		frac = 1
	}

	return gridTrackDef{
		min: gridTrackSize{kind: trackAuto}, //nolint:exhaustruct // intentional zero fields
		max: gridTrackSize{kind: trackFr, val: frac},
	}
}

// parseGridTrackFixedMins returns fixed (non-fr) track sizes as minimums for
// auto-height grids. fr / unknown / intrinsic tracks yield 0.
func parseGridTrackFixedMins(raw string, eng *engine) []float64 {
	defs := parseGridTrackDefs(raw)
	if len(defs) == 0 {
		return nil
	}

	out := make([]float64, len(defs))

	for i, d := range defs {
		if d.min.kind == trackFixed {
			out[i] = eng.scalePt(d.min.val)
		}
	}

	return out
}

// parseGridTracks parses grid-template-columns/rows into resolved lengths.
// columnGap is subtracted from contentW before distributing fr tracks so
// (n tracks + n-1 gaps) fit the content box. Supports minmax(), fr, lengths,
// %, auto, min-content, max-content (intrinsics default to 0 without measure).
func parseGridTracks(raw string, contentW, columnGap float64, eng *engine) []float64 {
	defs := parseGridTrackDefs(raw)
	if len(defs) == 0 {
		return nil
	}

	return resolveGridTrackSizes(defs, contentW, columnGap, eng, nil)
}

// parseGridTrackDefs tokenizes and expands repeat()/minmax() into track defs.
func parseGridTrackDefs(raw string) []gridTrackDef {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, cssDisplayNone) || strings.EqualFold(raw, "masonry") {
		return nil
	}

	raw = expandRepeatFunctions(raw)

	toks := tokenizeGridTracks(raw)
	if len(toks) == 0 {
		return nil
	}

	out := make([]gridTrackDef, 0, len(toks))
	for _, t := range toks {
		out = append(out, parseOneTrackDef(t))
	}

	return out
}

// expandRepeatFunctions replaces repeat(N, <track-list>) with N copies.
func expandRepeatFunctions(raw string) string {
	lower := strings.ToLower(raw)

	for {
		idx := strings.Index(lower, "repeat(")
		if idx < 0 {
			return raw
		}

		start := idx + len("repeat(")

		end := findMatchingParen(raw, start-1)
		if end < 0 {
			return raw
		}

		inner := raw[start:end]

		parts := splitTopLevelComma(inner)
		if len(parts) != two {
			return raw
		}

		node, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || node <= 0 || node >= 64 {
			return raw
		}

		track := strings.TrimSpace(parts[1])

		var boxNode strings.Builder

		boxNode.WriteString(raw[:idx])

		for i := range node {
			if i > 0 {
				boxNode.WriteByte(' ')
			}

			boxNode.WriteString(track)
		}

		boxNode.WriteString(raw[end+1:])
		raw = boxNode.String()
		lower = strings.ToLower(raw)
	}
}

func findMatchingParen(s string, openIdx int) int {
	depth := 0

	for idx := openIdx; idx < len(s); idx++ {
		switch s[idx] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return idx
			}
		}
	}

	return -1
}

func splitTopLevelComma(cssSheet string) []string {
	var parts []string

	depth := 0
	start := 0

	for idx := range len(cssSheet) {
		switch cssSheet[idx] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, cssSheet[start:idx])
				start = idx + 1
			}
		}
	}

	parts = append(parts, cssSheet[start:])

	return parts
}

// isTrackWhitespace reports CSS whitespace between grid track tokens.
func isTrackWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// tokenizeGridTracks splits on whitespace but keeps function calls intact.
func tokenizeGridTracks(raw string) []string {
	var toks []string

	var boxNode strings.Builder

	depth := 0
	flush := func() {
		if boxNode.Len() == 0 {
			return
		}

		toks = append(toks, boxNode.String())
		boxNode.Reset()
	}

	for i := range len(raw) {
		child := raw[i]

		switch {
		case child == '(':
			depth++

			boxNode.WriteByte(child)
		case child == ')':
			if depth > 0 {
				depth--
			}

			boxNode.WriteByte(child)
		case isTrackWhitespace(child) && depth == 0:
			flush()
		default:
			boxNode.WriteByte(child)
		}
	}

	flush()

	return toks
}

func parseOneTrackDef(tok string) gridTrackDef {
	tok = strings.TrimSpace(tok)
	lower := strings.ToLower(tok)

	if strings.HasPrefix(lower, "minmax(") && strings.HasSuffix(tok, ")") {
		inner := tok[len("minmax(") : len(tok)-1]

		parts := splitTopLevelComma(inner)
		if len(parts) == two {
			minS := parseTrackSize(strings.TrimSpace(parts[0]))
			maxS := parseTrackSize(strings.TrimSpace(parts[1]))
			// Spec: if max < min for fixed/fixed, use min for both (lite).
			if minS.kind == trackFixed && maxS.kind == trackFixed && maxS.val < minS.val {
				maxS = minS
			}

			return gridTrackDef{min: minS, max: maxS}
		}
	}

	size := parseTrackSize(tok)
	if size.kind == trackFr {
		return gridTrackDef{
			min: gridTrackSize{kind: trackAuto}, //nolint:exhaustruct // intentional zero fields
			max: size,
		}
	}

	return gridTrackDef{min: size, max: size}
}

// parseFrSize parses a "Nfr" track size; ok=false when tok is not fr.
func parseFrSize(tok string) (gridTrackSize, bool) {
	lower := strings.ToLower(tok)
	if !strings.HasSuffix(lower, "fr") {
		return gridTrackSize{}, false //nolint:exhaustruct // intentional zero fields
	}

	v, err := strconv.ParseFloat(strings.TrimSuffix(lower, "fr"), 64)
	if err != nil || v <= 0 {
		v = 1
	}

	return gridTrackSize{kind: trackFr, val: v}, true
}

// parseTrackPct parses a percentage track size into the negative fixed
// sentinel used by resolveTrackSide; ok=false when tok is not a valid %.
func parseTrackPct(tok string) (gridTrackSize, bool) {
	if !strings.HasSuffix(tok, "%") {
		return gridTrackSize{}, false //nolint:exhaustruct // intentional zero fields
	}

	pct, err := strconv.ParseFloat(strings.TrimSuffix(tok, "%"), 64)
	if err != nil {
		return gridTrackSize{}, false //nolint:exhaustruct // intentional zero fields
	}

	return gridTrackSize{kind: trackFixed, val: -pct}, true
}

func parseTrackSize(tok string) gridTrackSize {
	tok = strings.TrimSpace(tok)

	lower := strings.ToLower(tok)
	switch lower {
	case overflowAuto:
		return gridTrackSize{kind: trackAuto} //nolint:exhaustruct // intentional zero fields
	case "min-content":
		return gridTrackSize{kind: trackMinContent} //nolint:exhaustruct // intentional zero fields
	case "max-content":
		return gridTrackSize{kind: trackMaxContent} //nolint:exhaustruct // intentional zero fields
	}

	if size, ok := parseFrSize(lower); ok {
		return size
	}

	if val, ok := lengthBox(tok, defaultFontSizePt, 0, overflowAuto); ok && val >= 0 {
		// Percentages are re-resolved in resolveGridTrackSizes against the
		// definite container; store raw % as a sentinel via kind+val.
		if pctSize, ok := parseTrackPct(tok); ok {
			return pctSize
		}

		return gridTrackSize{kind: trackFixed, val: val}
	}

	return gridTrackSize{kind: trackAuto} //nolint:exhaustruct // intentional zero fields
}

type trackIntrinsic struct {
	minContent float64
	maxContent float64
}

// measureTrackIntrinsics estimates min/max-content contributions per track
// using text measure APIs. Spanning items contribute to the first track only
// (lite). axisColumns=true measures widths; false measures preferred heights.
func measureTrackIntrinsics(eng *engine, kids []*html.Node, nTracks int, axisColumns bool) []trackIntrinsic {
	if nTracks < 1 {
		return nil
	}

	out := make([]trackIntrinsic, nTracks)
	if eng == nil || len(kids) == 0 {
		return out
	}

	for i, kid := range kids {
		cstate := eng.styles[kid]
		tidx := i % nTracks

		var val float64
		if axisColumns {
			val = eng.measureCellContent(kid, *cstate)
		} else {
			// Height intrinsic: single-line text approximation via font size.
			val = eng.scalePt(cstate.FontSize) * defaultLineHeightRatio
			val += eng.scalePt(cstate.PaddingTop) + eng.scalePt(cstate.PaddingBottom) +
				eng.scalePt(cstate.BorderTop.Width) + eng.scalePt(cstate.BorderBottom.Width)
		}

		if val > out[tidx].minContent {
			out[tidx].minContent = val
		}

		if val > out[tidx].maxContent {
			out[tidx].maxContent = val
		}
	}

	return out
}

// gridTrackPlan holds the resolved base/limit sizes and fr factors for the
// tracks of one axis.
type gridTrackPlan struct {
	base, limit, frCoef []float64
	frSum               float64
}

// planGridTrackSides resolves each track's base/limit sizes and fr factors.
func planGridTrackSides(
	defs []gridTrackDef,
	contentSize float64,
	definite bool,
	eng *engine,
	intrinsics []trackIntrinsic,
) gridTrackPlan {
	node := len(defs)

	plan := gridTrackPlan{
		base:   make([]float64, node),
		limit:  make([]float64, node),
		frCoef: make([]float64, node),
		frSum:  0,
	}

	for idx, def := range defs {
		var intr trackIntrinsic
		if idx < len(intrinsics) {
			intr = intrinsics[idx]
		}

		plan.base[idx] = resolveTrackSide(def.min, contentSize, definite, eng, intr, true)
		lim := resolveTrackSide(def.max, contentSize, definite, eng, intr, false)

		switch {
		case def.max.kind == trackFr:
			plan.frCoef[idx] = def.max.val
			if plan.frCoef[idx] <= 0 {
				plan.frCoef[idx] = 1
			}

			plan.frSum += plan.frCoef[idx]
			plan.limit[idx] = math.Inf(1)
		case def.min.kind == trackFr:
			// Rare minmax(1fr, 200px): treat fr as flex with max cap.
			plan.frCoef[idx] = def.min.val
			if plan.frCoef[idx] <= 0 {
				plan.frCoef[idx] = 1
			}

			plan.frSum += plan.frCoef[idx]
			plan.base[idx] = 0
			plan.limit[idx] = lim
		default:
			plan.limit[idx] = lim
			if plan.limit[idx] < plan.base[idx] {
				plan.limit[idx] = plan.base[idx]
			}
		}
		// Auto max with auto/fixed min → growable to content (use max-content as soft limit).
		applyAutoSoftLimit(plan.limit, def, intr, idx)
	}

	return plan
}

// applyAutoSoftLimit caps growable auto/max-content tracks at their measured
// max-content size.
func applyAutoSoftLimit(limit []float64, def gridTrackDef, intr trackIntrinsic, idx int) {
	if def.max.kind != trackAuto && def.max.kind != trackMaxContent {
		return
	}

	if intr.maxContent <= limit[idx] && !math.IsInf(limit[idx], 1) {
		return
	}

	if def.max.kind == trackMaxContent && intr.maxContent > 0 {
		limit[idx] = intr.maxContent
	}
}

// distributeGridTracks shares leftover space between fr tracks, or between
// growable auto tracks when no fr tracks exist.
func distributeGridTracks(defs []gridTrackDef, base, limit, frCoef []float64, frSum, free float64) []float64 {
	out := make([]float64, len(defs))

	if frSum > 0 && free > 0 {
		for idx := range out {
			out[idx] = base[idx]
			if frCoef[idx] > 0 {
				out[idx] += free * (frCoef[idx] / frSum)
			}

			if out[idx] > limit[idx] {
				out[idx] = limit[idx]
			}
		}

		return out
	}

	return distributeAutoGridTracks(defs, base, limit, free, frSum)
}

// isAutoTrackKind reports whether a track can absorb leftover space.
func isAutoTrackKind(kind trackSizeKind) bool {
	return kind == trackAuto || kind == trackMaxContent || kind == trackMinContent
}

// distributeAutoGridTracks shares leftover space equally among auto tracks.
func distributeAutoGridTracks(defs []gridTrackDef, base, limit []float64, free, frSum float64) []float64 {
	out := make([]float64, len(defs))
	autoIdx := []int{}

	for i, d := range defs {
		out[i] = base[i]

		if isAutoTrackKind(d.max.kind) {
			autoIdx = append(autoIdx, i)
		}
	}

	if free > 0 && len(autoIdx) > 0 && frSum == 0 {
		each := free / float64(len(autoIdx))
		for _, i := range autoIdx {
			out[i] += each
			if out[i] > limit[i] && !math.IsInf(limit[i], 1) {
				out[i] = limit[i]
			}
		}
	}

	return out
}

// sanitizeGridTrackSizes clamps NaN/negative track sizes to zero.
func sanitizeGridTrackSizes(out []float64) {
	for i := range out {
		if out[i] < 0 || math.IsNaN(out[i]) {
			out[i] = 0
		}
	}
}

// resolveGridTrackSizes distributes free space with fr, honoring minmax floors.
// Percent mins/maxes require a definite contentSize (>=0); otherwise % → auto.
func resolveGridTrackSizes(
	defs []gridTrackDef,
	contentSize, gap float64,
	eng *engine,
	intrinsics []trackIntrinsic,
) []float64 {
	node := len(defs)
	if node == 0 {
		return nil
	}

	gapTotal := 0.0
	if node > 1 {
		gapTotal = gap * float64(node-1)
	}

	definite := contentSize >= 0 && !math.IsNaN(contentSize) && !math.IsInf(contentSize, 0)

	plan := planGridTrackSides(defs, contentSize, definite, eng, intrinsics)

	fixedSum := 0.0
	for i := range plan.base {
		fixedSum += plan.base[i]
	}

	free := contentSize - gapTotal - fixedSum
	if !definite {
		free = 0
	}

	if free < 0 {
		free = 0
	}

	out := distributeGridTracks(defs, plan.base, plan.limit, plan.frCoef, plan.frSum, free)
	sanitizeGridTrackSizes(out)

	return out
}

// resolveTrackSide resolves one min or max track size.
// pctSentinel: trackFixed with val < 0 stores -percent.
func resolveTrackSide(
	size gridTrackSize,
	contentSize float64,
	definite bool,
	eng *engine,
	intr trackIntrinsic,
	isMin bool,
) float64 {
	switch size.kind {
	case trackFixed:
		return resolveTrackFixedSide(size, contentSize, definite, eng, isMin)
	case trackFr:
		if isMin {
			return 0
		}

		return math.Inf(1)
	case trackMinContent:
		return intr.minContent
	case trackMaxContent:
		return intr.maxContent
	case trackAuto:
		if isMin {
			return intr.minContent // auto min ≈ min-content lite
		}

		return math.Inf(1)
	}

	return 0
}

// resolveTrackFixedSide resolves a fixed (or percentage) track side.
// Percentage: cyclic honesty — indefinite container → auto (0 min / inf max).
func resolveTrackFixedSide(size gridTrackSize, contentSize float64, definite bool, eng *engine, isMin bool) float64 {
	if size.val < 0 {
		if !definite || contentSize < 0 {
			if isMin {
				return 0
			}

			return math.Inf(1)
		}

		pct := -size.val

		return contentSize * pct / cssPercent
	}

	if eng != nil {
		return eng.scalePt(size.val)
	}

	return size.val
}
