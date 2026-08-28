package layout

import (
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func (e *engine) buildTable(node *html.Node, style ResolvedStyle, availW, posX, posY float64) *box {
	// flatten row groups into rows; count leading header-group rows
	rows, headerRows := e.collectTableRows(node)
	rows = stripEmptyTableRows(rows)
	headerRows = resolveHeaderRows(rows, headerRows)

	tableBox := &box{ //nolint:exhaustruct // intentional zero fields
		node: node, style: e.stylePtr(node), kind: displayTable, x: posX, y: posY, headerRows: headerRows,
	}
	if len(rows) == 0 {
		return tableBox
	}

	placed, nCols := placeTableCells(rows)
	if nCols == 0 {
		return tableBox
	}
	// Every placed cell is appended once while measuring rows. Reserve the
	// exact backing array so large tables do not retain geometric-growth
	// copies of their child pointers.
	tableBox.children = make([]*box, 0, len(placed)+1)

	colW, colMin, colPct, colAbs, cellData := e.measureTableColumns(placed, nCols)

	// table width
	// border-collapse: collapse suppresses the separate-border gap so colspan
	// header rows and body cells share edges instead of looking double-lined.
	spacingH := e.tableHSpacing(style)
	spacingV := e.tableVSpacing(style)
	chrome := spacingH*float64(nCols+1) +
		e.scalePt(style.BorderLeft.Width) + e.scalePt(style.BorderRight.Width) +
		e.scalePt(style.PaddingLeft) + e.scalePt(style.PaddingRight)

	tableHint := e.tableWidthHint(style, availW)

	capNode := e.tableCaptionNode(node)

	var capStyle *ResolvedStyle
	if capNode != nil {
		capStyle = e.stylePtr(capNode)
	}

	side := captionSideValue(style, capStyle)
	gridHint, gridAvail, captionW := e.sideCaptionGridBudget(capNode, capStyle, side, tableHint, availW)
	colW, gridW := sizeTableColumns(tableColumnEnv{
		colMin: colMin, colW: colW, colPct: colPct, colAbs: colAbs,
		chrome: chrome, availW: gridAvail, tableW: gridHint,
		fixed: style.TableLayout == positionFixed && gridHint >= 0,
	})
	tableBox.w = gridW

	gridX := posX
	if side == floatLeft && captionW > 0 {
		gridX = posX + captionW
	}

	tableY := posY

	if !captionSideHorizontal(side) && side != cssVerticalAlignBottom {
		if caption := e.attachCaption(tableBox, capNode, gridW, posX, posY); caption != nil {
			tableY = posY + caption.height
		}
	}

	e.layoutTableGrid(tableBox, style, rows, cellData, colW, spacingH, spacingV, nCols, gridX, tableY)
	e.placeTableCaption(tableBox, capNode, side, captionW, gridW, posX, posY)

	return tableBox
}

func (e *engine) layoutTableGrid(
	tableBox *box, style ResolvedStyle, rows [][]*html.Node, cellData [][]*box,
	colW []float64, spacingH, spacingV float64, nCols int, posX, tableY float64,
) {
	padL := e.scalePt(style.PaddingLeft) + e.scalePt(style.BorderLeft.Width)
	rowHeights, rowTops, curY := e.measureTableRows(
		tableBox, rows, cellData, colW, spacingH, spacingV, nCols, posX, tableY, padL,
	)

	tableBox.rows = cellData
	tableHeight := curY + e.scalePt(style.PaddingBottom) + e.scalePt(style.BorderBottom.Width)
	tableBox.height = tableY + tableHeight - tableBox.y

	if style.BGColor[3] > 0 && e.opts.Background {
		e.add(Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpFillRect, X: posX, Y: tableY, W: tableBox.w, H: tableHeight,
			R: style.BGColor[0], G: style.BGColor[1], B: style.BGColor[2], Alpha: style.BGColor[3],
		})
	}

	e.emitTableCells(tableBox, style, posX, tableY, tableHeight, padL, colW, rowTops, rowHeights, cellData)
}

func (e *engine) attachCaption(tableBox *box, capNode *html.Node, tableW, posX, posY float64) *box {
	caption := e.buildCaptionAt(capNode, tableW, posX, posY)
	if caption == nil {
		return nil
	}

	tableBox.children = append(tableBox.children, caption)

	return caption
}

func captionSideValue(tableStyle ResolvedStyle, captionStyle *ResolvedStyle) string {
	if tableStyle.CaptionSide != "" {
		return tableStyle.CaptionSide
	}

	if captionStyle != nil {
		return captionStyle.CaptionSide
	}

	return ""
}

func captionSideHorizontal(side string) bool {
	return side == floatLeft || side == floatRight
}

func (e *engine) sideCaptionGridBudget(
	capNode *html.Node, capStyle *ResolvedStyle, side string, tableHint, availW float64,
) (float64, float64, float64) {
	gridHint, gridAvail := tableHint, availW
	if !captionSideHorizontal(side) || capNode == nil {
		return gridHint, gridAvail, 0
	}

	outer := tableHint
	if outer < 0 {
		outer = availW
	}

	captionW := e.sideCaptionUsedWidth(capNode, capStyle, outer)
	if captionW <= 0 {
		return gridHint, gridAvail, 0
	}

	if gridHint >= 0 {
		gridHint -= captionW
		if gridHint < 0 {
			gridHint = 0
		}
	}

	gridAvail -= captionW
	if gridAvail < 0 {
		gridAvail = 0
	}

	return gridHint, gridAvail, captionW
}

func (e *engine) sideCaptionUsedWidth(capNode *html.Node, capStyle *ResolvedStyle, outerW float64) float64 {
	if capNode == nil || outerW <= 0 {
		return 0
	}

	maxW := outerW * captionSideMaxFrac

	if capStyle != nil && capStyle.Width >= 0 {
		used := e.scalePt(capStyle.Width)
		if used > maxW {
			return maxW
		}

		return used
	}

	st := initialStyle()
	if capStyle != nil {
		st = *capStyle
	}

	_, maxC := e.measureCellMinMax(capNode, st)
	if maxC > maxW {
		return maxW
	}

	return maxC
}

func (e *engine) placeTableCaption(
	tableBox *box, capNode *html.Node, side string, captionW, gridW, posX, posY float64,
) {
	switch {
	case side == cssVerticalAlignBottom:
		if caption := e.attachCaption(tableBox, capNode, tableBox.w, posX, posY+tableBox.height); caption != nil {
			tableBox.height += caption.height
		}
	case captionSideHorizontal(side) && captionW > 0:
		capX := posX
		if side == floatRight {
			capX = posX + gridW
		}

		caption := e.attachCaption(tableBox, capNode, captionW, capX, posY)
		if caption == nil {
			return
		}

		tableBox.w = gridW + captionW
		if caption.height > tableBox.height {
			tableBox.height = caption.height
		}
	}
}

func (e *engine) tableCaptionNode(node *html.Node) *html.Node {
	for _, child := range node.Children {
		if child.Type != html.ElementNode {
			continue
		}

		style := e.stylePtr(child)
		if style.Display != cssDisplayNone &&
			(child.Name == htmlCaption || style.Display == displayTableCaption) {
			return child
		}
	}

	return nil
}

func (e *engine) buildCaptionAt(capNode *html.Node, width, posX, posY float64) *box {
	if capNode == nil {
		return nil
	}

	return e.build(capNode, width, posX, posY)
}

// tableHSpacing is the horizontal inter-cell gap: border-collapse suppresses it.
func (e *engine) tableHSpacing(st ResolvedStyle) float64 {
	if st.BorderCollapse == borderCollapseValue {
		return 0
	}

	return e.scalePt(st.BorderSpacing)
}

// tableVSpacing is the vertical inter-cell gap: border-collapse suppresses it.
func (e *engine) tableVSpacing(style ResolvedStyle) float64 {
	if style.BorderCollapse == borderCollapseValue {
		return 0
	}

	if style.BorderSpacingV != 0 {
		return e.scalePt(style.BorderSpacingV)
	}

	return e.scalePt(style.BorderSpacing)
}

// emitTableCells paints the cell backgrounds/borders and the collapsed grid
// row-by-row so a row's grid segments land in the same op index span as its
// cells (pagination moves them together).
//
//nolint:wsl // collapsed and separate border paths intentionally share one emitter
func (e *engine) emitTableCells(
	tableBox *box, sty ResolvedStyle, posX, posY, tableHeight, padL float64,
	colW, rowTops, rowHeights []float64, cellData [][]*box,
) {
	collapse := sty.BorderCollapse == borderCollapseValue
	// Separate borders: stroke the table box. Collapsed grids include the
	// outer perimeter — stroking both doubles the outer edge and leaves the
	// table chrome behind when only cell ops shift across pages.
	if !collapse {
		e.emitBorders(sty, posX, posY, tableBox.w, tableHeight)
	}

	lastNonEmpty := lastNonEmptyRow(rowHeights)

	if collapse {
		// Column boundaries are the same for every row; compute once instead
		// of reallocating nCols+1 floats per row inside emitCollapsedRowGrid.
		xList := gridColumnEdges(posX+padL, colW)

		for rowIdx, cells := range cellData {
			for _, cell := range cells {
				// Skip paint for collapsed empty rows (h≈0); content was
				// ink-less and would only re-inflate phantom bands.
				if cell.height > layoutSlack {
					e.emitCell(cell, true)
				}
			}

			if rowHeights[rowIdx] > layoutSlack {
				e.emitCollapsedRowGrid(tableBox, rowIdx, rowIdx == lastNonEmpty, xList, rowTops, rowHeights)
			}
		}

		return
	}

	for _, cell := range tableBox.children {
		if cell.kind != tableCellKind {
			continue
		}
		if cell.height > layoutSlack {
			e.emitCell(cell, false)
		}
	}
}

// tcell is one placed table cell: its source node, grid position and spans.
type tcell struct {
	node         *html.Node
	row, col     int
	cSpan, rSpan int
}

// collectTableRows flattens row groups into rows and counts leading
// header-group rows.
func (e *engine) collectTableRows(node *html.Node) ([][]*html.Node, int) {
	var rows [][]*html.Node

	headerRows := 0

	var collect func(n *html.Node, inHeader bool)
	collect = func(n *html.Node, inHeader bool) {
		for _, child := range n.Children {
			if child.Type != html.ElementNode {
				continue
			}

			cstate := e.stylePtr(child)
			if cstate.Display == cssDisplayNone || cstate.Visibility == "collapse" {
				continue
			}

			switch {
			case cstate.Display == displayTableRow:
				rows = append(rows, rowCellNodes(child, e))

				if inHeader {
					headerRows++
				}
			case cstate.Display == displayHeaderGroup:
				collect(child, true)
			case strings.HasSuffix(cstate.Display, "row-group"):
				collect(child, false)
			}
		}
	}
	collect(node, false)

	return rows, headerRows
}

// rowCellNodes returns the table-cell children of a <tr>.
func rowCellNodes(tr *html.Node, e *engine) []*html.Node {
	cells := make([]*html.Node, 0, len(tr.Children))

	for _, cell := range tr.Children {
		if cell.Type == html.ElementNode && e.stylePtr(cell).Display == displayTableCell {
			cells = append(cells, cell)
		}
	}

	return cells
}

// resolveHeaderRows fixes up the thead-derived header count after empty rows
// were stripped, falling back to a leading band of <th> cells.
func resolveHeaderRows(rows [][]*html.Node, headerRows int) int {
	if headerRows > len(rows) {
		headerRows = len(rows)
	}
	// If thead contributed only empty rows, fall through to leading-th.
	if headerRows > 0 {
		// Verify leading rows still look like headers (all th); otherwise
		// the count was empty-thead noise.
		if countLeadingTHRows(rows[:headerRows]) != headerRows {
			headerRows = 0
		}
	}

	// Tables without <thead> often still use a leading row of <th> cells as
	// column headers (common HTML pattern). Treat consecutive leading all-th
	// rows as repeating headers for multi-page tables — generic, not site CSS.
	if headerRows == 0 {
		headerRows = countLeadingTHRows(rows)
	}

	return headerRows
}

// placeTableCells assigns each cell a column index honoring rowspan holes and
// discovers the column count. Returns the placed cells and nCols.
func placeTableCells(rows [][]*html.Node) ([]tcell, int) {
	placed := make([]tcell, 0, tableCellCount(rows))

	nRows := len(rows)
	occupied := make([][]int, nRows) // per-row remaining coverage counts
	nCols := 0

	var rowCols int

	for rowI, runic := range rows {
		if occupied[rowI] == nil {
			occupied[rowI] = make([]int, nCols)
		}

		placed, rowCols = placeRowCells(occupied, rowI, runic, nRows, placed)

		if rowCols > nCols {
			nCols = rowCols
		}

		if len(occupied[rowI]) > nCols {
			nCols = len(occupied[rowI])
		}
	}

	// Normalize occupied rows to nCols.
	for ri := range occupied {
		for len(occupied[ri]) < nCols {
			occupied[ri] = append(occupied[ri], 0)
		}
	}

	return placed, nCols
}

func tableCellCount(rows [][]*html.Node) int {
	count := 0
	for _, row := range rows {
		count += len(row)
	}

	return count
}

// placeRowCells assigns one row's cells to columns, honoring rowspan holes.
func placeRowCells(
	occupied [][]int,
	rowI int,
	runic []*html.Node,
	nRows int,
	placed []tcell,
) ([]tcell, int) {
	nCols := 0
	cidx := 0

	for _, cellNode := range runic {
		cstate, rowS := colSpan(cellNode), cellRowSpan(cellNode)
		if cstate < 1 {
			cstate = 1
		}

		if rowS < 1 {
			rowS = 1
		}

		for cidx < len(occupied[rowI]) && occupied[rowI][cidx] > 0 {
			cidx++
		}

		for len(occupied[rowI]) < cidx+cstate {
			occupied[rowI] = append(occupied[rowI], 0)
		}

		for k := range cstate {
			occupied[rowI][cidx+k] = rowS // covered for rs rows including this one
		}

		markRowspanCoverage(occupied, rowI, cidx, cstate, rowS, nRows)

		placed = append(placed, tcell{node: cellNode, row: rowI, col: cidx, cSpan: cstate, rSpan: rowS})

		if end := cidx + cstate; end > nCols {
			nCols = end
		}

		cidx += cstate
	}

	return placed, nCols
}

// markRowspanCoverage records that a rowspan cell covers columns
// [cidx, cidx+cstate) for rowS rows below rowI.
func markRowspanCoverage(occupied [][]int, rowI, cidx, cstate, rowS, nRows int) {
	for rowR := 1; rowR < rowS && rowI+rowR < nRows; rowR++ {
		for len(occupied[rowI+rowR]) < cidx+cstate {
			occupied[rowI+rowR] = append(occupied[rowI+rowR], 0)
		}

		for k := range cstate {
			if occupied[rowI+rowR][cidx+k] < rowS-rowR {
				occupied[rowI+rowR][cidx+k] = rowS - rowR
			}
		}
	}
}

// measureTableColumns measures each cell's min/max-content width; colspan
// cells contribute their content width evenly across the spanned columns
// (min floor per col). Returns column hints and the per-row cell boxes.
func (e *engine) measureTableColumns(
	placed []tcell, nCols int,
) ([]float64, []float64, []float64, []float64, [][]*box) {
	colW := make([]float64, nCols)   // preferred = max-content
	colMin := make([]float64, nCols) // shrink floor = min-content
	colPct := make([]float64, nCols) // >=0 means width:% of table; -1 = auto
	colAbs := make([]float64, nCols) // >=0 means absolute width pt; -1 = auto

	for i := range colPct {
		colPct[i] = -1
		colAbs[i] = -1
	}

	nRows := 0
	for _, p := range placed {
		if p.row+1 > nRows {
			nRows = p.row + 1
		}
	}

	rowCounts := make([]int, nRows)
	for _, p := range placed {
		rowCounts[p.row]++
	}

	cellData := make([][]*box, nRows)
	for i := range cellData {
		cellData[i] = make([]*box, 0, rowCounts[i])
	}

	for _, page := range placed {
		cell := e.buildCell(page.node, page.col, page.cSpan)
		cell.row, cell.rowSpan = page.row, page.rSpan
		cellData[page.row] = append(cellData[page.row], cell)
		cstate := e.styleVal(page.node)

		switch {
		case page.cSpan == 1:
			applySingleCellColumn(cell, cstate, colW, colMin, colPct, colAbs, page.col, e)
		case page.cSpan > 1:
			distributeSpanColumns(cell, page, colW, colMin, nCols)
		}
	}

	return colW, colMin, colPct, colAbs, cellData
}

// applySingleCellColumn folds one non-spanning cell's width contribution and
// width hints into its column.
func applySingleCellColumn(
	cell *box, cstate ResolvedStyle, colW, colMin, colPct, colAbs []float64, col int, eng *engine,
) {
	if cell.contentW > colW[col] {
		colW[col] = cell.contentW
	}

	if cell.contentMin > colMin[col] {
		colMin[col] = cell.contentMin
	}

	if cstate.WidthPercent >= 0 && colPct[col] < 0 {
		colPct[col] = cstate.WidthPercent
	}

	if cstate.Width >= 0 && colAbs[col] < 0 {
		colAbs[col] = eng.scalePt(cstate.Width)
	}
}

// distributeSpanColumns spreads a colspan cell's width evenly across the
// spanned columns (min floor per col).
func distributeSpanColumns(cell *box, page tcell, colW, colMin []float64, nCols int) {
	var sumMax, sumMin float64
	for k := 0; k < page.cSpan && page.col+k < nCols; k++ {
		sumMax += colW[page.col+k]
		sumMin += colMin[page.col+k]
	}

	if cell.contentW > sumMax {
		extra := (cell.contentW - sumMax) / float64(page.cSpan)
		for k := 0; k < page.cSpan && page.col+k < nCols; k++ {
			colW[page.col+k] += extra
		}
	}

	if cell.contentMin > sumMin {
		extra := (cell.contentMin - sumMin) / float64(page.cSpan)
		for k := 0; k < page.cSpan && page.col+k < nCols; k++ {
			colMin[page.col+k] += extra
		}
	}
}

// tableWidthHint resolves the definite table border-box width hint (-1 = auto).
func (e *engine) tableWidthHint(st ResolvedStyle, availW float64) float64 {
	var hint float64 = -1 // auto
	if st.WidthPercent >= 0 {
		hint = availW * st.WidthPercent / cssPercent
	} else if st.Width >= 0 {
		hint = e.scalePt(st.Width)
		if hint > availW && availW > 0 {
			hint = availW
		}
	}

	return hint
}

// measureTableRows lays out every cell at its final column width and resolves
// row heights: single-row cells first, then rowspan growth, then final tops
// and cell heights. Returns rowHeights, rowTops and the content height.
func (e *engine) measureTableRows(
	tableBox *box, rows [][]*html.Node, cellData [][]*box, colW []float64,
	spacingH, spacingV float64, nCols int, posX, posY, padL float64,
) ([]float64, []float64, float64) {
	// rows stays for the layoutTable call contract; ink flags now live on the
	// cell boxes in cellData (recorded once at build time), so the row loop
	// below reads flags instead of re-walking each cell's subtree.
	_ = rows

	nRows := len(cellData)
	rowHeights := make([]float64, nRows)
	rowTops := make([]float64, nRows)
	curY := e.scalePt(tableBox.style.PaddingTop) + e.scalePt(tableBox.style.BorderTop.Width)
	// Measure each cell at its final column width; row height from single-row
	// cells first. Rowspan cells enlarge the spanned rows afterward.
	// Rows with no local cells (rowspan holes) or only ink-less cells stay at
	// height 0 until rowspan growth — do not invent a 1pt phantom band.
	for rowIdx, cells := range cellData {
		rowTops[rowIdx] = posY + curY
		rowH := e.measureRowCells(tableBox, cells, rowIdx, colW, spacingH, nCols, posX, padL, rowTops)
		// Collapse rows whose cells have no ink (only padding/borders of empty
		// th/td). Keep a hairline only when the row has cells that paint
		// borders in separate-border mode and measured some chrome — pure
		// empty content collapses to 0 so border-collapse grids do not draw
		// a phantom empty band above real headers.
		if rowH > 0 && rowCellsHaveNoInk(cells) {
			rowH = 0
		}

		rowHeights[rowIdx] = rowH

		if rowH > 0 || spacingV > 0 {
			curY += rowH + spacingV
		}
	}

	growRowspanRows(tableBox, nRows, rowHeights, spacingV)

	// Recompute tops and assign final cell heights after rowspan growth.
	curY = e.scalePt(tableBox.style.PaddingTop) + e.scalePt(tableBox.style.BorderTop.Width)
	for rowIdx := range rowHeights {
		rowTops[rowIdx] = posY + curY
		curY += rowHeights[rowIdx] + spacingV
	}

	assignFinalCellHeights(tableBox, nRows, rowHeights, rowTops, spacingV)

	return rowHeights, rowTops, curY
}

// measureRowCells sizes and measures the cells of one row at their final
// column widths, returning the row height (single-row cells only).
func (e *engine) measureRowCells(
	tableBox *box, cells []*box, rowIdx int, colW []float64,
	spacing float64, nCols int, posX, padL float64, rowTops []float64,
) float64 {
	rowH := 0.0

	for _, cell := range cells {
		if cell.rowSpan < 1 {
			cell.rowSpan = 1
		}

		cellW := 0.0
		for k := 0; k < cell.span && cell.col+k < nCols; k++ {
			cellW += colW[cell.col+k]
		}

		cellW += spacing * float64(cell.span-1)
		cell.w = cellW
		cell.x = posX + padL

		for c := 0; c < cell.col && c < nCols; c++ {
			cell.x += colW[c] + spacing
		}

		cell.y = rowTops[rowIdx]
		e.measureCellHeight(cell, cellW)

		if cell.rowSpan == 1 && cell.contentH > rowH {
			rowH = cell.contentH
		}

		tableBox.children = append(tableBox.children, cell)
	}

	return rowH
}

// growRowspanRows enlarges the spanned rows so rowspan cells fit their
// content across the whole band.
//
//nolint:wsl // table geometry guards intentionally stay adjacent to the scan
func growRowspanRows(tableBox *box, nRows int, rowHeights []float64, spacing float64) {
	for _, cell := range tableBox.children {
		if cell.kind != tableCellKind {
			continue
		}
		if cell.rowSpan <= 1 {
			continue
		}

		start := cell.row
		if start < 0 {
			continue
		}

		end := start + cell.rowSpan
		if end > nRows {
			end = nRows
		}

		sum := 0.0
		for rowIdx := start; rowIdx < end; rowIdx++ {
			sum += rowHeights[rowIdx]
			if rowIdx+1 < end {
				sum += spacing
			}
		}

		if cell.contentH > sum {
			extra := (cell.contentH - sum) / float64(end-start)
			for rowIdx := start; rowIdx < end; rowIdx++ {
				rowHeights[rowIdx] += extra
			}
		}
	}
}

// assignFinalCellHeights sets cell.y/height/rowBoxH from the resolved rows.
//
//nolint:wsl // table geometry guards intentionally stay adjacent to the scan
func assignFinalCellHeights(tb *box, nRows int, rowHeights, rowTops []float64, spacing float64) {
	for _, cell := range tb.children {
		if cell.kind != tableCellKind {
			continue
		}
		start := cell.row
		if start < 0 {
			continue
		}

		cell.y = rowTops[start]

		rs := cell.rowSpan
		if rs < 1 {
			rs = 1
		}

		end := start + rs
		if end > nRows {
			end = nRows
		}

		height := 0.0
		for ri := start; ri < end; ri++ {
			height += rowHeights[ri]
			if ri+1 < end {
				height += spacing
			}
		}

		cell.height = height
		cell.rowBoxH = rowHeights[start]
	}
}

// lastNonEmptyRow returns the index of the last row with nonzero height.
func lastNonEmptyRow(rowHeights []float64) int {
	last := -1

	for ri := range rowHeights {
		if rowHeights[ri] > layoutSlack {
			last = ri
		}
	}

	return last
}

// emitCollapsedRowGrid strokes the shared border-collapse grid for one table
// row (top edge + verticals; bottom edge when lastRow). Ops are appended
// immediately after that row's cells and folded into the row's op range.
// xList holds the precomputed column boundary positions (shared across rows).
func (e *engine) emitCollapsedRowGrid(
	tableBox *box, rowIdx int, lastRow bool, xList []float64, rowTops, rowHeights []float64,
) {
	if rowIdx < 0 || rowIdx >= len(rowHeights) || rowHeights[rowIdx] <= 0.01 || len(xList) < two {
		return
	}

	nCols := len(xList) - 1

	yStart := rowTops[rowIdx]
	yEnd := yStart + rowHeights[rowIdx]
	gridStart := len(e.ops)
	stroke := &rowGridStroker{e: e}
	// Top edge. Skip under rowspan continuations so a multi-row Year cell is
	// not bisected mid-table; paint.capTablePageBreaks re-seals full tops for
	// page fragments where those holes look open.
	emitGridTopEdges(stroke, tableBox, rowIdx, xList, yStart)
	// Verticals only exist where an adjacent cell declares a left/right side.
	emitGridVerticals(stroke, tableBox, rowIdx, nCols, xList, yStart, yEnd)

	if lastRow {
		emitGridBottomEdges(stroke, tableBox, rowIdx, xList, yEnd)
	}

	gridEnd := len(e.ops) - 1
	if gridEnd >= gridStart && rowIdx < len(tableBox.rows) {
		expandRowOpRange(tableBox.rows[rowIdx], gridStart, gridEnd)
	}
}

// gridColumnEdges returns the x positions of the nCols+1 column boundaries.
func gridColumnEdges(left float64, colW []float64) []float64 {
	nCols := len(colW)
	xList := make([]float64, nCols+1)
	xList[0] = left

	for i := range nCols {
		xList[i+1] = xList[i] + colW[i]
	}

	return xList
}

// emitGridTopEdges strokes the row's top edge, skipping rowspan
// continuations so a multi-row cell is not bisected mid-table.
func emitGridTopEdges(stroke *rowGridStroker, tableBox *box, rowIdx int, xList []float64, yStart float64) {
	for cidx := range len(xList) - 1 {
		if rowIdx > 0 && rowspanCovers(tableBox, rowIdx-1, rowIdx, cidx) {
			continue
		}

		if side, ok := horizontalTableBorder(tableBox, rowIdx, cidx); ok {
			stroke.hline(xList[cidx], xList[cidx+1], yStart, side)
		}
	}
}

// emitGridVerticals strokes the row's vertical edges.
func emitGridVerticals(
	stroke *rowGridStroker, tableBox *box, rowIdx, nCols int, xList []float64, yStart, yEnd float64,
) {
	for cidx := 0; cidx <= nCols; cidx++ {
		if cidx > 0 && cidx < nCols && colspanCovers(tableBox, rowIdx, cidx-1, cidx) {
			continue
		}

		if side, ok := verticalTableBorder(tableBox, rowIdx, cidx); ok {
			stroke.vline(xList[cidx], yStart, yEnd, side)
		}
	}
}

// emitGridBottomEdges strokes the bottom edge of the last row.
func emitGridBottomEdges(stroke *rowGridStroker, tableBox *box, rowIdx int, xList []float64, yEnd float64) {
	for ci := range len(xList) - 1 {
		if side, ok := horizontalTableBorder(tableBox, rowIdx+1, ci); ok {
			stroke.hline(xList[ci], xList[ci+1], yEnd, side)
		}
	}
}

// rowGridStroker appends horizontal/vertical grid border ops with the shared
// engine so collapsed rows stay in the row's op span.
type rowGridStroker struct{ e *engine }

func (s *rowGridStroker) hline(x0, x1, yy float64, side border) {
	if x1-x0 <= 0 || !borderVisible(side) {
		return
	}

	s.e.emitBorderLine(x0, yy, x1-x0, 0,
		s.e.scalePt(side.Width), side.Style, side.Color[0], side.Color[1], side.Color[2])
}

func (s *rowGridStroker) vline(xx, ya, yb float64, side border) {
	if yb-ya <= 0.01 || !borderVisible(side) {
		return
	}

	s.e.emitBorderLine(xx, ya, 0, yb-ya,
		s.e.scalePt(side.Width), side.Style, side.Color[0], side.Color[1], side.Color[2])
}

func borderVisible(side border) bool {
	return side.Width > 0 && side.Style != cssDisplayNone
}

//nolint:mnd // CSS 2.1 border-style precedence ranking
func borderStyleRank(style string) int {
	switch style {
	case "double":
		return 5
	case solidKeyword:
		return 4
	case "dashed":
		return 3
	case "dotted":
		return 2
	case "ridge", "outset", "groove", "inset":
		return 1
	default:
		return 0
	}
}

func resolveBorderConflict(firstBorder, secondBorder border) border {
	if firstBorder.Style == overflowHidden || secondBorder.Style == overflowHidden {
		return border{Style: overflowHidden} //nolint:exhaustruct // intentional zero fields
	}

	firstVisible := borderVisible(firstBorder)
	secondVisible := borderVisible(secondBorder)

	if !firstVisible || !secondVisible {
		return resolveOneVisibleBorder(firstBorder, secondBorder, firstVisible, secondVisible)
	}

	if firstBorder.Width != secondBorder.Width {
		if firstBorder.Width > secondBorder.Width {
			return firstBorder
		}

		return secondBorder
	}

	rank1 := borderStyleRank(firstBorder.Style)
	rank2 := borderStyleRank(secondBorder.Style)

	if rank1 != rank2 {
		if rank1 > rank2 {
			return firstBorder
		}

		return secondBorder
	}

	return secondBorder
}

func resolveOneVisibleBorder(firstBorder, secondBorder border, firstVisible, secondVisible bool) border {
	if !firstVisible && !secondVisible {
		return border{} //nolint:exhaustruct // intentional zero fields
	}

	if !firstVisible {
		return secondBorder
	}

	return firstBorder
}

//nolint:cyclop,dupl // horizontal and vertical border scanning are symmetric axis routines
func horizontalTableBorder(tableBox *box, boundary, col int) (border, bool) {
	var (
		best  border
		found bool
	)

	for _, cell := range tableBox.children {
		if cell.kind != tableCellKind {
			continue
		}

		if cell.col > col || cell.col+cell.span <= col {
			continue
		}

		if cell.row == boundary && borderVisible(cell.style.BorderTop) {
			if !found {
				best = cell.style.BorderTop
				found = true
			} else {
				best = resolveBorderConflict(best, cell.style.BorderTop)
			}
		}

		if cell.row+cell.rowSpan == boundary && borderVisible(cell.style.BorderBottom) {
			if !found {
				best = cell.style.BorderBottom
				found = true
			} else {
				best = resolveBorderConflict(best, cell.style.BorderBottom)
			}
		}
	}

	return best, found && borderVisible(best)
}

//nolint:cyclop,dupl // horizontal and vertical border scanning are symmetric axis routines
func verticalTableBorder(tableBox *box, row, boundary int) (border, bool) {
	var (
		best  border
		found bool
	)

	for _, cell := range tableBox.children {
		if cell.kind != tableCellKind {
			continue
		}

		if cell.row > row || cell.row+cell.rowSpan <= row {
			continue
		}

		if cell.col == boundary && borderVisible(cell.style.BorderLeft) {
			if !found {
				best = cell.style.BorderLeft
				found = true
			} else {
				best = resolveBorderConflict(best, cell.style.BorderLeft)
			}
		}

		if cell.col+cell.span == boundary && borderVisible(cell.style.BorderRight) {
			if !found {
				best = cell.style.BorderRight
				found = true
			} else {
				best = resolveBorderConflict(best, cell.style.BorderRight)
			}
		}
	}

	return best, found && borderVisible(best)
}

// expandRowOpRange includes [start,end] paint ops in every cell of the row so
// pagination shifts that use the row's op span also move the collapsed grid.
func expandRowOpRange(row []*box, start, end int) {
	if start > end || len(row) == 0 {
		return
	}

	for _, cell := range row {
		if cell == nil {
			continue
		}

		if cell.opStart > cell.opEnd {
			// Cell emitted nothing (empty); claim the grid ops alone.
			cell.opStart, cell.opEnd = start, end

			continue
		}

		if start < cell.opStart {
			cell.opStart = start
		}

		if end > cell.opEnd {
			cell.opEnd = end
		}
	}
}

// rowspanCovers reports whether some cell occupies column ci across the
// boundary between row above and row below (so the horizontal rule is omitted).
//
//nolint:wsl // table coverage predicates are intentionally sequential
func rowspanCovers(tb *box, above, below, cidx int) bool {
	for _, cell := range tb.children {
		if cell.kind != tableCellKind {
			continue
		}
		if cell.rowSpan <= 1 {
			continue
		}

		start := cell.row
		if start < 0 {
			continue
		}

		if start <= above && start+cell.rowSpan > below &&
			cell.col <= cidx && cell.col+cell.span > cidx {
			return true
		}
	}

	return false
}

func colspanCovers(tableBox *box, rowIdx, leftCol, rightCol int) bool {
	if rowIdx < 0 || rowIdx >= len(tableBox.rows) {
		return false
	}

	for _, cell := range tableBox.rows[rowIdx] {
		if cell.span > 1 && cell.col <= leftCol && cell.col+cell.span > rightCol {
			return true
		}
	}

	// Rowspan continuation rows have no local cell — find covering cell.
	return rowspanCellCovers(tableBox, rowIdx, leftCol, rightCol)
}

// rowspanCellCovers reports whether a rowspan>1 cell whose vertical range
// includes ri spans columns (leftCol, rightCol).
//
//nolint:wsl // table coverage predicates are intentionally sequential
func rowspanCellCovers(tableBox *box, rowIdx, leftCol, rightCol int) bool {
	for _, cell := range tableBox.children {
		if cell.kind != tableCellKind {
			continue
		}
		start := cell.row
		if start < 0 {
			continue
		}

		rs := cell.rowSpan
		if rs < 1 {
			rs = 1
		}

		if start <= rowIdx && start+rs > rowIdx &&
			cell.span > 1 && cell.col <= leftCol && cell.col+cell.span > rightCol {
			return true
		}
	}

	return false
}

// buildCell measures a table cell's min/max-content width (no ops emitted).
// Height is not final here: layoutCell must run again with the real column
// width after column sizing, or narrow max-content widths force false wraps
// and inflate row heights (empty bands under single-line cell text).
func (e *engine) buildCell(node *html.Node, col, span int) *box {
	cellStyle := e.stylePtr(node)
	cellBox := &box{ //nolint:exhaustruct // intentional zero fields
		node:   node,
		style:  cellStyle,
		kind:   tableCellKind,
		col:    col,
		span:   span,
		hasInk: nodeHasTableInk(node),
	}
	cellBox.contentMin, cellBox.contentW = e.measureCellMinMax(node, *cellStyle)

	return cellBox
}

// measureCellHeight lays out the cell at width (border-box) without emitting
// paint ops, and stores the result on b.contentH.
func (e *engine) measureCellHeight(boxNode *box, width float64) {
	was := e.noEmit
	e.noEmit = true
	// Preserve the caller's noEmit flag. Nested tables call this during an
	// outer measure pass; restoring false mid-measure leaked ops at wrong
	// positions (fixture-10 nested table borders/text).
	boxNode.contentH = e.layoutCell(boxNode.node, *boxNode.style, width)
	e.noEmit = was
}

// cellBG returns the background to paint for a cell: the cell's own color,
// or the parent table-row's background when the cell is transparent (CSS
// does not inherit background, but row backgrounds show through empty
// cells in browsers — required for tr.good / tr.warn / tr.bad).
func (e *engine) cellBG(cell *box) (float64, float64, float64, float64, bool) {
	style := cell.style
	if style.BGColor[3] > 0 {
		return style.BGColor[0], style.BGColor[1], style.BGColor[2], style.BGColor[3], true
	}

	if cell.node != nil && cell.node.Parent != nil {
		ps := e.stylePtr(cell.node.Parent)
		if ps.Display == displayTableRow && ps.BGColor[3] > 0 {
			return ps.BGColor[0], ps.BGColor[1], ps.BGColor[2], ps.BGColor[3], true
		}
	}

	return 0, 0, 0, 0, false
}

// emitCell paints a placed cell's background, borders and content.
// skipBorders is set for border-collapse tables whose grid is stroked once
// by the parent table (avoids doubled/gapped per-cell edges).
func (e *engine) emitCell(cell *box, skipBorders bool) {
	sty := *cell.style
	start := len(e.ops)

	if e.opts.Background {
		if r, g, bl, a, ok := e.cellBG(cell); ok {
			e.add(Op{ //nolint:exhaustruct // intentional zero fields
				Kind: OpFillRect, X: cell.x, Y: cell.y, W: cell.w, H: cell.height,
				R: r, G: g, B: bl, Alpha: a,
			})
		}
	}

	if !skipBorders {
		e.emitBorders(sty, cell.x, cell.y, cell.w, cell.height)
	}

	curX, contentW := e.contentBox(cell.x, cell.w, sty)
	curY := cell.y + e.scalePt(sty.PaddingTop) + e.scalePt(sty.BorderTop.Width)
	curY = cellVerticalAlignOffset(cell, curY)
	// flowChildren advances cy; cell content is rooted at absolute canvas y
	// (pass y=0, contentX=cx, cy=content top) so floats pack inside the cell
	// BFC. Pass the cell as parent so float/block children attach for tests.
	oldMax := e.imgMaxW

	if contentW > 0 {
		e.imgMaxW = contentW
	}

	enclose := e.pushBFCFloats(sty, curX, contentW)
	_ = e.flowChildren(cell, cell.node.Children, sty, contentW, curX, 0, curY)

	if enclose && e.bfcFloats != nil {
		// Cell border box already sized; floats are clipped to the cell BFC.
		_ = e.bfcFloats.extentCy(0, curY)
	}

	e.popBFCFloats(enclose)

	e.imgMaxW = oldMax
	// Rowspan cells with forced multi-line content (wiki Ref: [127]<br>[128]
	// in rowspan=2) pack lines at the top with normal line-height, so both
	// markers sit in the first row band and look overlapped. Spread line
	// boxes evenly across the full cell height when we have room.
	if cell.rowSpan > 1 {
		distributeRowspanLines(e.ops, start, len(e.ops), cell.y, cell.height,
			e.scalePt(sty.PaddingTop)+e.scalePt(sty.BorderTop.Width),
			e.scalePt(sty.PaddingBottom)+e.scalePt(sty.BorderBottom.Width))
	}

	cell.opStart, cell.opEnd = start, len(e.ops)-1
}

// cellVerticalAlignOffset shifts the content origin within the row box for
// vertical-align middle/bottom table cells.
func cellVerticalAlignOffset(cell *box, curY float64) float64 {
	extra := cell.height - cell.contentH
	if extra <= 0 {
		return curY
	}

	switch cell.style.VerticalAlign {
	case cssVerticalAlignMiddle:
		return curY + extra/two
	case cssVerticalAlignBottom:
		return curY + extra
	default:
		return curY
	}
}

// distributeRowspanLines remaps distinct text/bullet baselines in ops[start:end)
// so they span the cell's content box evenly (top line near top, bottom near
// bottom). Non-text ops (underlines, links) ride with the nearest baseline.
func distributeRowspanLines(ops []Op, start, end int, cellY, cellH, padTop, padBot float64) {
	if end <= start || cellH <= 0 || ops == nil {
		return
	}

	const yEps = 0.75

	bands := collectTextBands(ops, start, end, yEps)

	if len(bands) < two {
		return
	}
	// Sort bands top→bottom.
	sortBandsTopDown(bands)

	innerTop := cellY + padTop
	innerBot := cellY + cellH - padBot

	if innerBot-innerTop < minBoxPt {
		return
	}
	// Only redistribute when natural packing is much shorter than the cell
	// (typical rowspan>1 with few <br> lines).
	natural := bands[len(bands)-1].y - bands[0].y
	if natural >= (innerBot-innerTop)*0.55 {
		return
	}

	targets := interpolatedBandTargets(ops, bands, innerTop, innerBot)
	if targets == nil {
		return
	}
	// Map old baseline → dy, apply to all ops near that baseline.
	shifts := make([]bandShift, len(bands))
	for i, b := range bands {
		shifts[i] = bandShift{y0: b.y, dy: targets[i] - b.y}
	}

	applyBandShifts(ops, start, end, shifts, bandEmSize(ops, bands[0].idx))
}

// bandEmSize estimates the em size from the first band's text ops.
