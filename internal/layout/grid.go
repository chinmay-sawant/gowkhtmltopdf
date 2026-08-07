package layout

import (
	"math"
	"strconv"
	"strings"

	"gowkhtmltopdf/internal/html"
)

// buildGrid lays out a CSS grid Stage B/C subset: column/row tracks (incl.
// minmax/fr/auto/min-content/max-content lite), independent gaps, template
// areas + named grid-area, auto-flow row/column (sparse or dense), column/row
// spanning, and justify/align-items/self.
//
// ponytail: display:subgrid → ordinary grid (no parent template inherit).
// ponytail: grid-template-*: masonry keyword stripped → dense auto-flow
// (no L3 shortest-stack pack). Upgrade if report templates need either.
func (e *engine) buildGrid(n *html.Node, st ResolvedStyle, availW, x, y float64) *box {
	if st.Display == "subgrid" {
		st.Display = "grid"
	}
	// Masonry keyword → empty track list → auto-flow dense grid.
	st.GridTemplateColumns = stripMasonryKeyword(st.GridTemplateColumns)
	st.GridTemplateRows = stripMasonryKeyword(st.GridTemplateRows)

	ml := e.scalePt(st.MarginLeft)
	boxNode := &box{node: n, style: st, kind: "block", x: x + ml, y: y} //nolint:exhaustruct // intentional zero fields
	boxNode.w = resolveUsedWidth(st, availW, e)
	contentX, contentW := e.contentBox(boxNode.x, boxNode.w, st)
	contentStart := len(e.ops)
	curY := e.scalePt(st.PaddingTop) + e.scalePt(st.BorderTop.Width)

	rowGap, columnGap := e.styleGaps(st)

	colDefs := parseGridTrackDefs(st.GridTemplateColumns)
	areas := parseGridTemplateAreas(st.GridTemplateAreas)

	if len(colDefs) == 0 {
		n := areas.cols
		if n < 1 {
			n = 1
		}

		colDefs = make([]gridTrackDef, n)
		for i := range colDefs {
			colDefs[i] = flexibleTrack(1)
		}
	} else {
		for len(colDefs) < areas.cols {
			colDefs = append(colDefs, flexibleTrack(1))
		}
	}

	var kids []*html.Node

	for _, child := range n.Children {
		if child.Type != html.ElementNode {
			continue
		}

		if e.styles[child].Display == "none" {
			continue
		}

		kids = append(kids, child)
	}

	// Intrinsic measure lite for min-content / max-content column mins.
	colIntrinsics := measureTrackIntrinsics(e, kids, len(colDefs), true)

	cols := resolveGridTrackSizes(colDefs, contentW, columnGap, e, colIntrinsics)
	if len(cols) == 0 {
		cols = []float64{contentW}
	}

	contentH := resolveContentHeight(st, e)

	type cell struct {
		n            *html.Node
		col, colSpan int
		row, rowSpan int
	}

	var placed []cell

	occ := map[int]map[int]bool{}
	ensure := func(row int) {
		if occ[row] == nil {
			occ[row] = map[int]bool{}
		}
	}
	freeAt := func(row, col, rowSpan, colSpan int) bool {
		for r := range rowSpan {
			ensure(row + r)

			for i := range colSpan {
				c := col + i
				if c >= len(cols) || occ[row+r][c] {
					return false
				}
			}
		}

		return true
	}
	mark := func(row, col, rowSpan, colSpan int) {
		for r := range rowSpan {
			ensure(row + r)

			for i := range colSpan {
				occ[row+r][col+i] = true
			}
		}
	}

	columnMajor, densePack := gridAutoFlowMode(st.GridAutoFlow)
	cursorRow, cursorCol := 0, 0
	nCols := len(cols)

	for _, kid := range kids {
		cstate := e.styles[kid]

		colSpan := cstate.GridColumnSpan
		if colSpan < 1 {
			colSpan = 1
		}

		rowSpan := cstate.GridRowSpan
		if rowSpan < 1 {
			rowSpan = 1
		}

		colStart := cstate.GridColumnStart - 1 // 0-based; -1 = auto
		rowStart := cstate.GridRowStart - 1
		definite := false

		if name := strings.TrimSpace(cstate.GridArea); name != "" {
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

		var row, col int

		switch {
		case definite:
			row, col = rowStart, colStart
			if row < 0 {
				row = 0
			}

			if col < 0 {
				col = 0
			}

			for !freeAt(row, col, rowSpan, colSpan) {
				row++
			}
		case colStart >= 0:
			col = colStart

			row = 0
			if !densePack {
				row = cursorRow
			}

			for !freeAt(row, col, rowSpan, colSpan) {
				row++
			}
		case rowStart >= 0:
			row = rowStart
			col = 0

			for {
				if col+colSpan > nCols {
					col = 0
					row++

					continue
				}

				if freeAt(row, col, rowSpan, colSpan) {
					break
				}

				col++
			}
		default:
			startRow, startCol := cursorRow, cursorCol
			if densePack {
				startRow, startCol = 0, 0
			}

			if columnMajor {
				minRows := areas.rows
				if minRows < 1 {
					minRows = (len(kids) + nCols - 1) / nCols
					if minRows < 1 {
						minRows = 1
					}
				}

				row, col = findGridSlotColumnMajor(freeAt, nCols, startRow, startCol, rowSpan, colSpan, minRows)
			} else {
				row, col = findGridSlotRowMajor(freeAt, nCols, startRow, startCol, rowSpan, colSpan)
			}
		}

		mark(row, col, rowSpan, colSpan)
		placed = append(placed, cell{n: kid, col: col, colSpan: colSpan, row: row, rowSpan: rowSpan})

		if columnMajor {
			cursorRow = row + rowSpan
			cursorCol = col

			flowRows := areas.rows
			if flowRows < 1 {
				flowRows = (len(kids) + nCols - 1) / nCols
			}

			if flowRows > 0 && cursorRow >= flowRows {
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

	rowTemplate := strings.TrimSpace(st.GridTemplateRows)
	definiteRows := contentH >= 0 && rowTemplate != "" &&
		strings.ToLower(rowTemplate) != "none"

	var rows []float64

	if definiteRows {
		rowDefs := parseGridTrackDefs(st.GridTemplateRows)
		// Pad/truncate defs to placed row count when template is shorter.
		for len(rowDefs) < numRows {
			rowDefs = append(rowDefs, flexibleTrack(1))
		}

		rowIntrinsics := measureTrackIntrinsics(e, kids, len(rowDefs), false)
		rows = resolveGridTrackSizes(rowDefs, contentH, rowGap, e, rowIntrinsics)
	}

	if len(rows) == 0 {
		rows = make([]float64, numRows)

		if mins := parseGridTrackFixedMins(st.GridTemplateRows, e); len(mins) > 0 {
			for i := 0; i < numRows && i < len(mins); i++ {
				if mins[i] > 0 {
					rows[i] = mins[i]
				}
			}
		}
	} else if len(rows) < numRows {
		for len(rows) < numRows {
			rows = append(rows, 0)
		}
	} else if len(rows) > numRows {
		numRows = len(rows)
	}

	type placedBox struct {
		cell
		b         *box
		cellW, cx float64
		prefH     float64
	}

	var pboxes []placedBox

	// Measure preferred heights without emitting (needed for auto row tracks).
	for _, page := range placed {
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

		was := e.noEmit
		e.noEmit = true
		mb := e.build(page.n, contW, curX, y+curY)
		e.noEmit = was

		prefH := 0.0
		if mb != nil {
			prefH = mb.height
		}

		pboxes = append(pboxes, placedBox{cell: page, cellW: contW, cx: curX, prefH: prefH}) //nolint:exhaustruct // intentional zero fields

		if !definiteRows && page.rowSpan == 1 && prefH > rows[page.row] {
			rows[page.row] = prefH
		}
	}

	if !definiteRows {
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

	rowYs := make([]float64, numRows)
	rowYs[0] = curY

	for r := 1; r < numRows; r++ {
		rowYs[r] = rowYs[r-1] + rows[r-1] + rowGap
	}

	containerJustify := st.JustifyItems
	if containerJustify == "" {
		containerJustify = "stretch"
	}

	containerAlign := st.AlignItems
	if containerAlign == "" {
		containerAlign = "stretch"
	}

	for i := range pboxes {
		pbox := &pboxes[i]
		cellH := 0.0

		for r := range pbox.rowSpan {
			cellH += rows[pbox.row+r]
			if r > 0 {
				cellH += rowGap
			}
		}

		targetX := pbox.cx
		targetY := y + rowYs[pbox.row]

		cstate := e.styles[pbox.n]

		justify := cstate.JustifySelf
		if justify == "" || justify == "auto" {
			justify = containerJustify
		}

		align := cstate.AlignSelf
		if align == "" || align == "auto" {
			align = containerAlign
		}

		// Default stretch: border box fills the grid area (CSS Grid §10.2).
		buildH := -1.0
		if (align == "stretch" || align == "") && cellH > 0 &&
			cstate.Height < 0 && cstate.HeightPercent < 0 {
			buildH = cellH
		}

		var cblock *box

		if buildH > 0 {
			prev := e.styles[pbox.n]
			mod := prev
			mod.Height = buildH
			mod.HeightPercent = -1
			mod.BoxSizing = "border-box"
			e.styles[pbox.n] = mod
			cblock = e.build(pbox.n, pbox.cellW, pbox.cx, targetY)
			e.styles[pbox.n] = prev
		} else {
			cblock = e.build(pbox.n, pbox.cellW, pbox.cx, targetY)
		}

		if cblock == nil {
			continue
		}

		pbox.b = cblock

		deltaX := targetX - pbox.b.x
		deltaY := targetY - pbox.b.y
		deltaX += gridAlignOffset(justify, pbox.cellW, pbox.b.w)
		deltaY += gridAlignOffset(align, cellH, pbox.b.height)

		e.shiftBoxOps(pbox.b, deltaX, deltaY)
		pbox.b.x += deltaX
		pbox.b.y += deltaY
		boxNode.children = append(boxNode.children, pbox.b)
	}

	usedH := curY
	if numRows > 0 {
		usedH = rowYs[numRows-1] + rows[numRows-1]
	}

	usedH += e.scalePt(st.PaddingBottom)

	if st.Height >= 0 {
		height := e.scalePt(st.Height)
		if st.BoxSizing != "border-box" {
			height += e.scalePt(st.PaddingTop) + e.scalePt(st.PaddingBottom) +
				e.scalePt(st.BorderTop.Width) + e.scalePt(st.BorderBottom.Width)
		}

		if usedH < height {
			usedH = height
		}
	}

	minBorderH := e.scalePt(st.PaddingTop) + e.scalePt(st.BorderTop.Width) +
		e.scalePt(st.PaddingBottom) + e.scalePt(st.BorderBottom.Width)
	if contentH >= 0 {
		minBorderH += contentH
	}

	if usedH < minBorderH {
		usedH = minBorderH
	}

	boxNode.height = usedH
	e.prependChrome(contentStart, boxNode, st, boxNode.x, y, boxNode.w, boxNode.height)

	return boxNode
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
		if availW > 0 && !math.IsInf(availW, 0) && availW < 1e12 {
			width = availW * sty.WidthPercent / cssPercent
		}
		// else: cyclic % → auto (keep fill-remaining w)
	} else if sty.Width >= 0 {
		width = engN.scalePt(sty.Width)
		if sty.BoxSizing != "border-box" {
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
	if sty.BoxSizing == "border-box" {
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
func (e *engine) styleGaps(st ResolvedStyle) (rowGap, columnGap float64) {
	if st.RowGap == 0 && st.ColumnGap == 0 {
		g := e.scalePt(st.Gap)

		return g, g
	}

	return e.scalePt(st.RowGap), e.scalePt(st.ColumnGap)
}

// gridAlignOffset returns the inline/block offset for start/end/center/stretch.
// stretch is treated as start (lite).
func gridAlignOffset(value string, cell, item float64) float64 {
	switch value {
	case "end", "flex-end", "right", "bottom":
		if cell > item {
			return cell - item
		}
	case "center":
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

	if raw == "" || strings.EqualFold(raw, "none") {
		return out
	}
	// Collect quoted strings: "a b" "c d" or 'a b'
	var rows [][]string

	for idx := 0; idx < len(raw); {
		for idx < len(raw) && (raw[idx] == ' ' || raw[idx] == '\t' || raw[idx] == '\n' || raw[idx] == '\r') {
			idx++
		}

		if idx >= len(raw) {
			break
		}

		query := raw[idx]
		if query != '"' && query != '\'' {
			// Unquoted token — skip (invalid lite)
			for idx < len(raw) && raw[idx] != ' ' && raw[idx] != '\t' && raw[idx] != '"' && raw[idx] != '\'' {
				idx++
			}

			continue
		}

		idx++
		start := idx

		for idx < len(raw) && raw[idx] != query {
			idx++
		}

		cell := raw[start:idx]

		if idx < len(raw) {
			idx++ // closing quote
		}

		toks := strings.Fields(cell)
		if len(toks) == 0 {
			continue
		}

		rows = append(rows, toks)
	}

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

	type bounds struct {
		r0, c0, r1, c1 int
		seen           bool
	}

	acc := map[string]*bounds{}

	for runic, row := range rows {
		for child, name := range row {
			if name == "." || strings.EqualFold(name, "none") {
				continue
			}

			boxNode := acc[name]
			if boxNode == nil {
				boxNode = &bounds{r0: runic, c0: child, r1: runic, c1: child, seen: true}
				acc[name] = boxNode

				continue
			}

			if runic < boxNode.r0 {
				boxNode.r0 = runic
			}

			if runic > boxNode.r1 {
				boxNode.r1 = runic
			}

			if child < boxNode.c0 {
				boxNode.c0 = child
			}

			if child > boxNode.c1 {
				boxNode.c1 = child
			}
		}
	}

	for name, b := range acc {
		out.names[name] = gridAreaRect{
			row:     b.r0,
			col:     b.c0,
			rowSpan: b.r1 - b.r0 + 1,
			colSpan: b.c1 - b.c0 + 1,
		}
	}

	return out
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
func gridAutoFlowMode(flow string) (columnMajor, dense bool) {
	switch strings.ToLower(strings.TrimSpace(flow)) {
	case "column":
		return true, false
	case "column dense":
		return true, true
	case "dense", "row dense":
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
	if raw == "" || strings.EqualFold(raw, "none") || strings.EqualFold(raw, "masonry") {
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
		case (child == ' ' || child == '\t' || child == '\n' || child == '\r') && depth == 0:
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

func parseTrackSize(tok string) gridTrackSize {
	tok = strings.TrimSpace(tok)

	lower := strings.ToLower(tok)
	switch lower {
	case "auto":
		return gridTrackSize{kind: trackAuto} //nolint:exhaustruct // intentional zero fields
	case "min-content":
		return gridTrackSize{kind: trackMinContent} //nolint:exhaustruct // intentional zero fields
	case "max-content":
		return gridTrackSize{kind: trackMaxContent} //nolint:exhaustruct // intentional zero fields
	}

	if strings.HasSuffix(lower, "fr") {
		v, err := strconv.ParseFloat(strings.TrimSuffix(lower, "fr"), 64)
		if err != nil || v <= 0 {
			v = 1
		}

		return gridTrackSize{kind: trackFr, val: v}
	}

	if val, ok := lengthBox(tok, defaultFontSizePt, 0, "auto"); ok && val >= 0 {
		// Percentages are re-resolved in resolveGridTrackSizes against the
		// definite container; store raw % as a sentinel via kind+val.
		if strings.HasSuffix(tok, "%") {
			pct, err := strconv.ParseFloat(strings.TrimSuffix(tok, "%"), 64)
			if err == nil {
				return gridTrackSize{kind: trackFixed, val: -pct} // negative → %
			}
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

		if axisColumns {
			minC := eng.measureCellContent(kid, cstate)
			if minC > out[tidx].minContent {
				out[tidx].minContent = minC
			}

			if minC > out[tidx].maxContent {
				out[tidx].maxContent = minC
			}
		} else {
			// Height intrinsic: single-line text approximation via font size.
			height := eng.scalePt(cstate.FontSize) * defaultLineHeightRatio
			height += eng.scalePt(cstate.PaddingTop) + eng.scalePt(cstate.PaddingBottom) +
				eng.scalePt(cstate.BorderTop.Width) + eng.scalePt(cstate.BorderBottom.Width)

			if height > out[tidx].minContent {
				out[tidx].minContent = height
			}

			if height > out[tidx].maxContent {
				out[tidx].maxContent = height
			}
		}
	}

	return out
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

	base := make([]float64, node)
	limit := make([]float64, node)
	frCoef := make([]float64, node)
	frSum := 0.0

	for idx, declN := range defs {
		var intr trackIntrinsic
		if idx < len(intrinsics) {
			intr = intrinsics[idx]
		}

		base[idx] = resolveTrackSide(declN.min, contentSize, definite, eng, intr, true)
		lim := resolveTrackSide(declN.max, contentSize, definite, eng, intr, false)

		if declN.max.kind == trackFr {
			frCoef[idx] = declN.max.val
			if frCoef[idx] <= 0 {
				frCoef[idx] = 1
			}

			frSum += frCoef[idx]
			limit[idx] = math.Inf(1)
		} else if declN.min.kind == trackFr && declN.max.kind != trackFr {
			// Rare minmax(1fr, 200px): treat fr as flex with max cap.
			frCoef[idx] = declN.min.val
			if frCoef[idx] <= 0 {
				frCoef[idx] = 1
			}

			frSum += frCoef[idx]
			base[idx] = 0
			limit[idx] = lim
		} else {
			limit[idx] = lim
			if limit[idx] < base[idx] {
				limit[idx] = base[idx]
			}
		}
		// Auto max with auto/fixed min → growable to content (use max-content as soft limit).
		if declN.max.kind == trackAuto || declN.max.kind == trackMaxContent {
			if intr.maxContent > limit[idx] || math.IsInf(limit[idx], 1) {
				if declN.max.kind == trackMaxContent && intr.maxContent > 0 {
					limit[idx] = intr.maxContent
				}
			}
		}
	}

	fixedSum := 0.0
	for i := range base {
		fixedSum += base[i]
	}

	free := contentSize - gapTotal - fixedSum
	if !definite {
		free = 0
	}

	if free < 0 {
		free = 0
	}

	out := make([]float64, node)
	if frSum > 0 && free > 0 {
		for idx := range out {
			if frCoef[idx] > 0 {
				grow := free * (frCoef[idx] / frSum)

				out[idx] = base[idx] + grow
				if out[idx] > limit[idx] {
					out[idx] = limit[idx]
				}
			} else {
				out[idx] = base[idx]
				if out[idx] > limit[idx] {
					out[idx] = limit[idx]
				}
			}
		}
	} else {
		// No fr: if all auto and definite, share leftover equally among auto maxes.
		autoIdx := []int{}

		for i, d := range defs {
			out[i] = base[i]

			if d.max.kind == trackAuto || d.max.kind == trackMaxContent || d.max.kind == trackMinContent {
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
	}

	for i := range out {
		if out[i] < 0 || math.IsNaN(out[i]) {
			out[i] = 0
		}
	}

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
		if size.val < 0 {
			// Percentage: cyclic honesty — indefinite container → auto (0 min / inf max).
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
	case trackFr:
		if isMin {
			return 0
		}

		return math.Inf(1)
	case trackMinContent:
		return intr.minContent
	case trackMaxContent:
		if isMin {
			return intr.maxContent
		}

		return intr.maxContent
	case trackAuto:
		if isMin {
			return intr.minContent // auto min ≈ min-content lite
		}

		return math.Inf(1)
	}

	return 0
}
