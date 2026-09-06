package layout

import (
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

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
	columnMajor, densePack bool, explicitRows int,
) []gridCell {
	// Implicit row band for column-major auto flow (grid-template-rows empty).
	implicitRows := areas.rows
	if explicitRows > implicitRows {
		implicitRows = explicitRows
	}

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
			*eng.stylePtr(kid), areas, occ, nCols, cursorRow, cursorCol,
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

	colStart, colSpan = resolveGridAxisSpan(colStart, sty.GridColumnEnd, colSpan)
	rowStart, rowSpan = resolveGridAxisSpan(rowStart, sty.GridRowEnd, rowSpan)

	colStart, colSpan = expandGridColumnEndSpan(colStart, colSpan, nCols, sty)

	if sty.GridRowEnd == -1 && rowStart >= 0 {
		// row -1 is best-effort: extend span if definite rows known elsewhere; handled via gridRowCount.
		_ = rowStart
	}

	rowStart, colStart, rowSpan, colSpan, definite := applyGridAreaSpan(
		rowStart, colStart, rowSpan, colSpan, sty, areas,
	)

	if colSpan > nCols {
		colSpan = nCols
	}

	if rowStart >= 0 && colStart >= 0 {
		definite = true
	}

	return rowStart, colStart, rowSpan, colSpan, definite
}

// expandGridColumnEndSpan expands a -1 column end (last grid line, nCols+1
// for columns) into the column count it implies.
func expandGridColumnEndSpan(colStart, colSpan, nCols int, sty ResolvedStyle) (int, int) {
	// -1 means last grid line (nCols+1 for columns). Expand span to end.
	if sty.GridColumnEnd != -1 {
		return colStart, colSpan
	}

	if colStart >= 0 {
		colSpan = nCols - colStart
		if colSpan < 1 {
			colSpan = 1
		}

		return colStart, colSpan
	}

	return 0, nCols
}

// applyGridAreaSpan overlays the named grid-area rect when the item names one
// present in areas. It returns the possibly updated spans plus definite.
func applyGridAreaSpan(
	rowStart, colStart, rowSpan, colSpan int, sty ResolvedStyle, areas gridTemplateAreasMap,
) (int, int, int, int, bool) {
	name := strings.TrimSpace(sty.GridArea)
	if name == "" {
		return rowStart, colStart, rowSpan, colSpan, false
	}

	rect, ok := resolveNamedGridArea(areas, name)
	if !ok {
		return rowStart, colStart, rowSpan, colSpan, false
	}

	return rect.row, rect.col, rect.rowSpan, rect.colSpan, true
}

func resolveGridAxisSpan(start, end, span int) (int, int) {
	if end <= 0 {
		return start, span
	}

	if start >= 0 && end-1 > start {
		return start, (end - 1) - start
	}

	if start < 0 {
		st := end - 1 - span
		if st < 0 {
			st = 0
		}

		return st, span
	}

	return start, span
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

	if sty.GridRowEnd == -1 {
		rowStart, rowSpan = resolveGridRowEndSpan(rowStart, minRows)
	}

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

// resolveGridRowEndSpan expands a -1 row end (last grid line, best-effort)
// against the known implicit row band.
func resolveGridRowEndSpan(rowStart, minRows int) (int, int) {
	if rowStart >= 0 {
		if rowStart < minRows {
			span := minRows - rowStart
			if span < 1 {
				span = 1
			}

			return rowStart, span
		}

		return rowStart, 1
	}

	if minRows < 1 {
		return 0, 1
	}

	return 0, minRows
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
