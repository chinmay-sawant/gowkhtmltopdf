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
// spanning, justify/align-items/self, subgrid inherit, and masonry packing
// (shortest-column) when one template axis is masonry.
func (e *engine) buildGrid(n *html.Node, st ResolvedStyle, availW, x, y float64) *box {
	st = applySubgridInherit(n, st, e)

	masonry := detectMasonryAxis(st.GridTemplateColumns, st.GridTemplateRows)
	switch masonry {
	case masonryBoth:
		// Honesty: both-axes masonry is not defined for packing; fall back to
		// ordinary dense auto-flow after stripping the masonry keywords.
		st.GridTemplateColumns = stripMasonryKeyword(st.GridTemplateColumns)
		st.GridTemplateRows = stripMasonryKeyword(st.GridTemplateRows)
	case masonryRows:
		// grid-template-rows: masonry — pack into column tracks.
		return e.buildMasonryPack(n, st, availW, x, y, true)
	case masonryCols:
		// grid-template-columns: masonry — pack into row tracks.
		return e.buildMasonryPack(n, st, availW, x, y, false)
	}

	ml := e.scalePt(st.MarginLeft)
	b := &box{node: n, style: st, kind: "block", x: x + ml, y: y}
	b.w = resolveUsedWidth(st, availW, e)
	contentW := b.w - e.scalePt(st.PaddingLeft) - e.scalePt(st.PaddingRight) -
		e.scalePt(st.BorderLeft.Width) - e.scalePt(st.BorderRight.Width)
	if contentW < 0 {
		contentW = 0
	}
	contentX := b.x + e.scalePt(st.BorderLeft.Width) + e.scalePt(st.PaddingLeft)
	contentStart := len(e.ops)
	cy := e.scalePt(st.PaddingTop) + e.scalePt(st.BorderTop.Width)

	rowGap, columnGap := e.gridGaps(st)

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
	for _, c := range n.Children {
		if c.Type != html.ElementNode {
			continue
		}
		if e.styles[c].Display == "none" {
			continue
		}
		kids = append(kids, c)
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
		for r := 0; r < rowSpan; r++ {
			ensure(row + r)
			for i := 0; i < colSpan; i++ {
				c := col + i
				if c >= len(cols) || occ[row+r][c] {
					return false
				}
			}
		}
		return true
	}
	mark := func(row, col, rowSpan, colSpan int) {
		for r := 0; r < rowSpan; r++ {
			ensure(row + r)
			for i := 0; i < colSpan; i++ {
				occ[row+r][col+i] = true
			}
		}
	}

	columnMajor, densePack := gridAutoFlowMode(st.GridAutoFlow)
	cursorRow, cursorCol := 0, 0
	nCols := len(cols)
	for _, kid := range kids {
		cs := e.styles[kid]
		colSpan := cs.GridColumnSpan
		if colSpan < 1 {
			colSpan = 1
		}
		rowSpan := cs.GridRowSpan
		if rowSpan < 1 {
			rowSpan = 1
		}
		colStart := cs.GridColumnStart - 1 // 0-based; -1 = auto
		rowStart := cs.GridRowStart - 1
		definite := false
		if name := strings.TrimSpace(cs.GridArea); name != "" {
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
	for _, p := range placed {
		cw := 0.0
		cx := contentX
		for j := 0; j < p.col; j++ {
			cx += cols[j] + columnGap
		}
		for j := 0; j < p.colSpan; j++ {
			cw += cols[p.col+j]
			if j > 0 {
				cw += columnGap
			}
		}
		was := e.noEmit
		e.noEmit = true
		mb := e.build(p.n, cw, cx, y+cy)
		e.noEmit = was
		prefH := 0.0
		if mb != nil {
			prefH = mb.h
		}
		pboxes = append(pboxes, placedBox{cell: p, cellW: cw, cx: cx, prefH: prefH})
		if !definiteRows && p.rowSpan == 1 && prefH > rows[p.row] {
			rows[p.row] = prefH
		}
	}

	if !definiteRows {
		for _, pb := range pboxes {
			if pb.rowSpan <= 1 {
				continue
			}
			sum := 0.0
			for r := 0; r < pb.rowSpan; r++ {
				sum += rows[pb.row+r]
				if r > 0 {
					sum += rowGap
				}
			}
			if pb.prefH > sum {
				extra := (pb.prefH - sum) / float64(pb.rowSpan)
				for r := 0; r < pb.rowSpan; r++ {
					rows[pb.row+r] += extra
				}
			}
		}
	}

	rowYs := make([]float64, numRows)
	rowYs[0] = cy
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
		pb := &pboxes[i]
		cellH := 0.0
		for r := 0; r < pb.rowSpan; r++ {
			cellH += rows[pb.row+r]
			if r > 0 {
				cellH += rowGap
			}
		}
		targetX := pb.cx
		targetY := y + rowYs[pb.row]

		cs := e.styles[pb.n]
		justify := cs.JustifySelf
		if justify == "" || justify == "auto" {
			justify = containerJustify
		}
		align := cs.AlignSelf
		if align == "" || align == "auto" {
			align = containerAlign
		}

		// Default stretch: border box fills the grid area (CSS Grid §10.2).
		buildH := -1.0
		if (align == "stretch" || align == "") && cellH > 0 &&
			cs.Height < 0 && cs.HeightPercent < 0 {
			buildH = cellH
		}

		var cb *box
		if buildH > 0 {
			prev := e.styles[pb.n]
			mod := prev
			mod.Height = buildH
			mod.HeightPercent = -1
			mod.BoxSizing = "border-box"
			e.styles[pb.n] = mod
			cb = e.build(pb.n, pb.cellW, pb.cx, targetY)
			e.styles[pb.n] = prev
		} else {
			cb = e.build(pb.n, pb.cellW, pb.cx, targetY)
		}
		if cb == nil {
			continue
		}
		pb.b = cb

		dx := targetX - pb.b.x
		dy := targetY - pb.b.y
		dx += gridAlignOffset(justify, pb.cellW, pb.b.w)
		dy += gridAlignOffset(align, cellH, pb.b.h)

		e.shiftBoxOps(pb.b, dx, dy)
		pb.b.x += dx
		pb.b.y += dy
		b.children = append(b.children, pb.b)
	}

	usedH := cy
	if numRows > 0 {
		usedH = rowYs[numRows-1] + rows[numRows-1]
	}
	usedH += e.scalePt(st.PaddingBottom)
	if st.Height >= 0 {
		h := e.scalePt(st.Height)
		if st.BoxSizing != "border-box" {
			h += e.scalePt(st.PaddingTop) + e.scalePt(st.PaddingBottom) +
				e.scalePt(st.BorderTop.Width) + e.scalePt(st.BorderBottom.Width)
		}
		if usedH < h {
			usedH = h
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
	b.h = usedH
	e.prependChrome(contentStart, st, b.x, y, b.w, b.h)
	return b
}

// buildMasonryPack implements report-engine masonry lite: items pack into the
// non-masonry axis tracks by shortest-stack packing.
//
// Honesty: this is not the full CSS Grid Level 3 masonry algorithm (no
// spanning across masonry tracks, no alignment into shared tracks, no
// reverse packing). Items are placed in source order into the currently
// shortest column (or row when packing along columns).
func (e *engine) buildMasonryPack(n *html.Node, st ResolvedStyle, availW, x, y float64, packIntoColumns bool) *box {
	ml := e.scalePt(st.MarginLeft)
	b := &box{node: n, style: st, kind: "block", x: x + ml, y: y}
	b.w = resolveUsedWidth(st, availW, e)
	contentW := b.w - e.scalePt(st.PaddingLeft) - e.scalePt(st.PaddingRight) -
		e.scalePt(st.BorderLeft.Width) - e.scalePt(st.BorderRight.Width)
	if contentW < 0 {
		contentW = 0
	}
	contentX := b.x + e.scalePt(st.BorderLeft.Width) + e.scalePt(st.PaddingLeft)
	contentStart := len(e.ops)
	cy := e.scalePt(st.PaddingTop) + e.scalePt(st.BorderTop.Width)
	rowGap, columnGap := e.gridGaps(st)

	var kids []*html.Node
	for _, c := range n.Children {
		if c.Type != html.ElementNode {
			continue
		}
		if e.styles[c].Display == "none" {
			continue
		}
		kids = append(kids, c)
	}

	if packIntoColumns {
		colRaw := stripMasonryKeyword(st.GridTemplateColumns)
		colDefs := parseGridTrackDefs(colRaw)
		if len(colDefs) == 0 {
			colDefs = []gridTrackDef{flexibleTrack(1), flexibleTrack(1), flexibleTrack(1)}
		}
		intrinsics := measureTrackIntrinsics(e, kids, len(colDefs), true)
		cols := resolveGridTrackSizes(colDefs, contentW, columnGap, e, intrinsics)
		stacks := make([]float64, len(cols))
		colX := make([]float64, len(cols))
		colX[0] = contentX
		for i := 1; i < len(cols); i++ {
			colX[i] = colX[i-1] + cols[i-1] + columnGap
		}

		for _, kid := range kids {
			ci := shortestStackIndex(stacks)
			was := e.noEmit
			e.noEmit = true
			mb := e.build(kid, cols[ci], colX[ci], y+cy+stacks[ci])
			e.noEmit = was
			prefH := 0.0
			if mb != nil {
				prefH = mb.h
			}
			cb := e.build(kid, cols[ci], colX[ci], y+cy+stacks[ci])
			if cb == nil {
				continue
			}
			dx := colX[ci] - cb.x
			dy := (y + cy + stacks[ci]) - cb.y
			e.shiftBoxOps(cb, dx, dy)
			cb.x += dx
			cb.y += dy
			b.children = append(b.children, cb)
			h := prefH
			if cb.h > h {
				h = cb.h
			}
			stacks[ci] += h + rowGap
		}
		maxStack := 0.0
		for _, s := range stacks {
			if s > maxStack {
				maxStack = s
			}
		}
		if maxStack > 0 {
			maxStack -= rowGap // last gap unused
		}
		usedH := cy + maxStack + e.scalePt(st.PaddingBottom)
		if st.Height >= 0 {
			h := e.scalePt(st.Height)
			if st.BoxSizing != "border-box" {
				h += e.scalePt(st.PaddingTop) + e.scalePt(st.PaddingBottom) +
					e.scalePt(st.BorderTop.Width) + e.scalePt(st.BorderBottom.Width)
			}
			if usedH < h {
				usedH = h
			}
		}
		b.h = usedH
		e.prependChrome(contentStart, st, b.x, y, b.w, b.h)
		return b
	}

	// Pack into row tracks (grid-template-columns: masonry).
	contentH := resolveContentHeight(st, e)
	rowRaw := stripMasonryKeyword(st.GridTemplateRows)
	rowDefs := parseGridTrackDefs(rowRaw)
	if len(rowDefs) == 0 {
		rowDefs = []gridTrackDef{flexibleTrack(1), flexibleTrack(1)}
	}
	rowSizeBase := contentH
	if rowSizeBase < 0 {
		rowSizeBase = contentW // honesty: auto-height masonry rows share contentW
	}
	intrinsics := measureTrackIntrinsics(e, kids, len(rowDefs), false)
	rows := resolveGridTrackSizes(rowDefs, rowSizeBase, rowGap, e, intrinsics)
	stacks := make([]float64, len(rows)) // used width per row
	rowY := make([]float64, len(rows))
	rowY[0] = cy
	for i := 1; i < len(rows); i++ {
		rowY[i] = rowY[i-1] + rows[i-1] + rowGap
	}

	for _, kid := range kids {
		ri := shortestStackIndex(stacks)
		remain := contentW - stacks[ri]
		if remain < 0 {
			remain = contentW
			stacks[ri] = 0
		}
		was := e.noEmit
		e.noEmit = true
		mb := e.build(kid, remain, contentX+stacks[ri], y+rowY[ri])
		e.noEmit = was
		itemW := remain
		if mb != nil && mb.w > 0 && mb.w < remain {
			itemW = mb.w
		}
		cb := e.build(kid, itemW, contentX+stacks[ri], y+rowY[ri])
		if cb == nil {
			continue
		}
		dx := contentX + stacks[ri] - cb.x
		dy := y + rowY[ri] - cb.y
		e.shiftBoxOps(cb, dx, dy)
		cb.x += dx
		cb.y += dy
		b.children = append(b.children, cb)
		stacks[ri] += cb.w + columnGap
	}
	usedH := cy
	if len(rows) > 0 {
		usedH = rowY[len(rows)-1] + rows[len(rows)-1]
	}
	usedH += e.scalePt(st.PaddingBottom)
	if st.Height >= 0 {
		h := e.scalePt(st.Height)
		if st.BoxSizing != "border-box" {
			h += e.scalePt(st.PaddingTop) + e.scalePt(st.PaddingBottom) +
				e.scalePt(st.BorderTop.Width) + e.scalePt(st.BorderBottom.Width)
		}
		if usedH < h {
			usedH = h
		}
	}
	b.h = usedH
	e.prependChrome(contentStart, st, b.x, y, b.w, b.h)
	return b
}

func shortestStackIndex(stacks []float64) int {
	best := 0
	for i := 1; i < len(stacks); i++ {
		if stacks[i] < stacks[best] {
			best = i
		}
	}
	return best
}

// --- Subgrid lite -----------------------------------------------------------

// applySubgridInherit treats display:subgrid as a nested grid that copy-inherits
// parent grid-template-columns/rows/areas when the child's templates are empty,
// "none", or the "subgrid" keyword.
//
// Honesty: no true shared track sizing across subtrees. Parent track sizes are
// not re-resolved jointly; only the template string is copied, then sized in
// the child's own containing block.
func applySubgridInherit(n *html.Node, st ResolvedStyle, e *engine) ResolvedStyle {
	if st.Display != "subgrid" {
		return st
	}
	st.Display = "grid"
	if n == nil || n.Parent == nil || e == nil || e.styles == nil {
		return st
	}
	ps, ok := e.styles[n.Parent]
	if !ok {
		return st
	}
	if isSubgridTemplateValue(st.GridTemplateColumns) {
		st.GridTemplateColumns = ps.GridTemplateColumns
	}
	if isSubgridTemplateValue(st.GridTemplateRows) {
		st.GridTemplateRows = ps.GridTemplateRows
	}
	if isSubgridTemplateValue(st.GridTemplateAreas) {
		st.GridTemplateAreas = ps.GridTemplateAreas
	}
	return st
}

func isSubgridTemplateValue(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "" || s == "none" || s == "subgrid"
}

// --- Masonry detection ------------------------------------------------------

type masonryAxis int

const (
	masonryNone masonryAxis = iota
	masonryRows             // grid-template-rows: masonry
	masonryCols             // grid-template-columns: masonry
	masonryBoth
)

func detectMasonryAxis(cols, rows string) masonryAxis {
	c := isMasonryKeyword(cols)
	r := isMasonryKeyword(rows)
	switch {
	case c && r:
		return masonryBoth
	case r:
		return masonryRows
	case c:
		return masonryCols
	default:
		return masonryNone
	}
}

func isMasonryKeyword(raw string) bool {
	return strings.ToLower(strings.TrimSpace(raw)) == "masonry"
}

func stripMasonryKeyword(raw string) string {
	if isMasonryKeyword(raw) {
		return ""
	}
	return raw
}

// --- Width / height helpers -------------------------------------------------

// resolveUsedWidth computes border-box width. WidthPercent against a
// non-positive (indefinite) availW is treated as auto (fill remaining).
func resolveUsedWidth(st ResolvedStyle, availW float64, e *engine) float64 {
	ml, mr := e.scalePt(st.MarginLeft), e.scalePt(st.MarginRight)
	w := availW - ml - mr
	if w < 0 {
		w = 0
	}
	if st.WidthPercent >= 0 {
		if availW > 0 && !math.IsInf(availW, 0) && availW < 1e12 {
			w = availW * st.WidthPercent / 100
		}
		// else: cyclic % → auto (keep fill-remaining w)
	} else if st.Width >= 0 {
		w = e.scalePt(st.Width)
		if st.BoxSizing != "border-box" {
			w += e.scalePt(st.PaddingLeft) + e.scalePt(st.PaddingRight) +
				e.scalePt(st.BorderLeft.Width) + e.scalePt(st.BorderRight.Width)
		}
	}
	return w
}

// resolveContentHeight returns definite content-box height, or -1 when auto.
// HeightPercent only resolves when Height was already made definite by a parent
// stretch; unresolved HeightPercent (indefinite CB) is treated as auto.
func resolveContentHeight(st ResolvedStyle, e *engine) float64 {
	if st.HeightPercent >= 0 && st.Height < 0 {
		// Cyclic % honesty: indefinite containing block → auto.
		return -1
	}
	if st.Height < 0 {
		return -1
	}
	h := e.scalePt(st.Height)
	if st.BoxSizing == "border-box" {
		h -= e.scalePt(st.PaddingTop) + e.scalePt(st.PaddingBottom) +
			e.scalePt(st.BorderTop.Width) + e.scalePt(st.BorderBottom.Width)
	}
	if h < 0 {
		h = 0
	}
	return h
}

// --- Gaps / alignment -------------------------------------------------------

// gridGaps returns scaled row/column gaps. When both longhands are unset (0),
// fall back to the Gap shorthand for both axes.
func (e *engine) gridGaps(st ResolvedStyle) (rowGap, columnGap float64) {
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
			return (cell - item) / 2
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
	out := gridTemplateAreasMap{names: map[string]gridAreaRect{}}
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.EqualFold(raw, "none") {
		return out
	}
	// Collect quoted strings: "a b" "c d" or 'a b'
	var rows [][]string
	for i := 0; i < len(raw); {
		for i < len(raw) && (raw[i] == ' ' || raw[i] == '\t' || raw[i] == '\n' || raw[i] == '\r') {
			i++
		}
		if i >= len(raw) {
			break
		}
		q := raw[i]
		if q != '"' && q != '\'' {
			// Unquoted token — skip (invalid lite)
			for i < len(raw) && raw[i] != ' ' && raw[i] != '\t' && raw[i] != '"' && raw[i] != '\'' {
				i++
			}
			continue
		}
		i++
		start := i
		for i < len(raw) && raw[i] != q {
			i++
		}
		cell := raw[start:i]
		if i < len(raw) {
			i++ // closing quote
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
	for r, row := range rows {
		for c, name := range row {
			if name == "." || strings.EqualFold(name, "none") {
				continue
			}
			b := acc[name]
			if b == nil {
				b = &bounds{r0: r, c0: c, r1: r, c1: c, seen: true}
				acc[name] = b
				continue
			}
			if r < b.r0 {
				b.r0 = r
			}
			if r > b.r1 {
				b.r1 = r
			}
			if c < b.c0 {
				b.c0 = c
			}
			if c > b.c1 {
				b.c1 = c
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
		return gridAreaRect{}, false
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

func flexibleTrack(fr float64) gridTrackDef {
	if fr <= 0 {
		fr = 1
	}
	return gridTrackDef{
		min: gridTrackSize{kind: trackAuto},
		max: gridTrackSize{kind: trackFr, val: fr},
	}
}

func autoTrack() gridTrackDef {
	return gridTrackDef{
		min: gridTrackSize{kind: trackAuto},
		max: gridTrackSize{kind: trackAuto},
	}
}

// parseGridTrackFixedMins returns fixed (non-fr) track sizes as minimums for
// auto-height grids. fr / unknown / intrinsic tracks yield 0.
func parseGridTrackFixedMins(raw string, e *engine) []float64 {
	defs := parseGridTrackDefs(raw)
	if len(defs) == 0 {
		return nil
	}
	out := make([]float64, len(defs))
	for i, d := range defs {
		if d.min.kind == trackFixed {
			out[i] = e.scalePt(d.min.val)
		}
	}
	return out
}

// parseGridTracks parses grid-template-columns/rows into resolved lengths.
// columnGap is subtracted from contentW before distributing fr tracks so
// (n tracks + n-1 gaps) fit the content box. Supports minmax(), fr, lengths,
// %, auto, min-content, max-content (intrinsics default to 0 without measure).
func parseGridTracks(raw string, contentW, columnGap float64, e *engine) []float64 {
	defs := parseGridTrackDefs(raw)
	if len(defs) == 0 {
		return nil
	}
	return resolveGridTrackSizes(defs, contentW, columnGap, e, nil)
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
		if len(parts) != 2 {
			return raw
		}
		n, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || n <= 0 || n >= 64 {
			return raw
		}
		track := strings.TrimSpace(parts[1])
		var b strings.Builder
		b.WriteString(raw[:idx])
		for i := 0; i < n; i++ {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(track)
		}
		b.WriteString(raw[end+1:])
		raw = b.String()
		lower = strings.ToLower(raw)
	}
}

func findMatchingParen(s string, openIdx int) int {
	depth := 0
	for i := openIdx; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func splitTopLevelComma(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// tokenizeGridTracks splits on whitespace but keeps function calls intact.
func tokenizeGridTracks(raw string) []string {
	var toks []string
	var b strings.Builder
	depth := 0
	flush := func() {
		if b.Len() == 0 {
			return
		}
		toks = append(toks, b.String())
		b.Reset()
	}
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		switch {
		case c == '(':
			depth++
			b.WriteByte(c)
		case c == ')':
			if depth > 0 {
				depth--
			}
			b.WriteByte(c)
		case (c == ' ' || c == '\t' || c == '\n' || c == '\r') && depth == 0:
			flush()
		default:
			b.WriteByte(c)
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
		if len(parts) == 2 {
			minS := parseTrackSize(strings.TrimSpace(parts[0]))
			maxS := parseTrackSize(strings.TrimSpace(parts[1]))
			// Spec: if max < min for fixed/fixed, use min for both (lite).
			if minS.kind == trackFixed && maxS.kind == trackFixed && maxS.val < minS.val {
				maxS = minS
			}
			return gridTrackDef{min: minS, max: maxS}
		}
	}
	sz := parseTrackSize(tok)
	if sz.kind == trackFr {
		return gridTrackDef{
			min: gridTrackSize{kind: trackAuto},
			max: sz,
		}
	}
	return gridTrackDef{min: sz, max: sz}
}

func parseTrackSize(tok string) gridTrackSize {
	tok = strings.TrimSpace(tok)
	lower := strings.ToLower(tok)
	switch lower {
	case "auto":
		return gridTrackSize{kind: trackAuto}
	case "min-content":
		return gridTrackSize{kind: trackMinContent}
	case "max-content":
		return gridTrackSize{kind: trackMaxContent}
	}
	if strings.HasSuffix(lower, "fr") {
		v, err := strconv.ParseFloat(strings.TrimSuffix(lower, "fr"), 64)
		if err != nil || v <= 0 {
			v = 1
		}
		return gridTrackSize{kind: trackFr, val: v}
	}
	if v, ok := lengthBox(tok, 12, 0, "auto"); ok && v >= 0 {
		// Percentages are re-resolved in resolveGridTrackSizes against the
		// definite container; store raw % as a sentinel via kind+val.
		if strings.HasSuffix(tok, "%") {
			pct, err := strconv.ParseFloat(strings.TrimSuffix(tok, "%"), 64)
			if err == nil {
				return gridTrackSize{kind: trackFixed, val: -pct} // negative → %
			}
		}
		return gridTrackSize{kind: trackFixed, val: v}
	}
	return gridTrackSize{kind: trackAuto}
}

type trackIntrinsic struct {
	minContent float64
	maxContent float64
}

// measureTrackIntrinsics estimates min/max-content contributions per track
// using text measure APIs. Spanning items contribute to the first track only
// (lite). axisColumns=true measures widths; false measures preferred heights.
func measureTrackIntrinsics(e *engine, kids []*html.Node, nTracks int, axisColumns bool) []trackIntrinsic {
	if nTracks < 1 {
		return nil
	}
	out := make([]trackIntrinsic, nTracks)
	if e == nil || len(kids) == 0 {
		return out
	}
	for i, kid := range kids {
		cs := e.styles[kid]
		ti := i % nTracks
		if axisColumns {
			mc := e.measureCellContent(kid, cs)
			if mc > out[ti].minContent {
				out[ti].minContent = mc
			}
			if mc > out[ti].maxContent {
				out[ti].maxContent = mc
			}
		} else {
			// Height intrinsic: single-line text approximation via font size.
			h := e.scalePt(cs.FontSize) * 1.2
			h += e.scalePt(cs.PaddingTop) + e.scalePt(cs.PaddingBottom) +
				e.scalePt(cs.BorderTop.Width) + e.scalePt(cs.BorderBottom.Width)
			if h > out[ti].minContent {
				out[ti].minContent = h
			}
			if h > out[ti].maxContent {
				out[ti].maxContent = h
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
	e *engine,
	intrinsics []trackIntrinsic,
) []float64 {
	n := len(defs)
	if n == 0 {
		return nil
	}
	gapTotal := 0.0
	if n > 1 {
		gapTotal = gap * float64(n-1)
	}
	definite := contentSize >= 0 && !math.IsNaN(contentSize) && !math.IsInf(contentSize, 0)

	base := make([]float64, n)
	limit := make([]float64, n)
	frCoef := make([]float64, n)
	frSum := 0.0

	for i, d := range defs {
		var intr trackIntrinsic
		if i < len(intrinsics) {
			intr = intrinsics[i]
		}
		base[i] = resolveTrackSide(d.min, contentSize, definite, e, intr, true)
		lim := resolveTrackSide(d.max, contentSize, definite, e, intr, false)
		if d.max.kind == trackFr {
			frCoef[i] = d.max.val
			if frCoef[i] <= 0 {
				frCoef[i] = 1
			}
			frSum += frCoef[i]
			limit[i] = math.Inf(1)
		} else if d.min.kind == trackFr && d.max.kind != trackFr {
			// Rare minmax(1fr, 200px): treat fr as flex with max cap.
			frCoef[i] = d.min.val
			if frCoef[i] <= 0 {
				frCoef[i] = 1
			}
			frSum += frCoef[i]
			base[i] = 0
			limit[i] = lim
		} else {
			limit[i] = lim
			if limit[i] < base[i] {
				limit[i] = base[i]
			}
		}
		// Auto max with auto/fixed min → growable to content (use max-content as soft limit).
		if d.max.kind == trackAuto || d.max.kind == trackMaxContent {
			if intr.maxContent > limit[i] || math.IsInf(limit[i], 1) {
				if d.max.kind == trackMaxContent && intr.maxContent > 0 {
					limit[i] = intr.maxContent
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

	out := make([]float64, n)
	if frSum > 0 && free > 0 {
		for i := range out {
			if frCoef[i] > 0 {
				grow := free * (frCoef[i] / frSum)
				out[i] = base[i] + grow
				if out[i] > limit[i] {
					out[i] = limit[i]
				}
			} else {
				out[i] = base[i]
				if out[i] > limit[i] {
					out[i] = limit[i]
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
	sz gridTrackSize,
	contentSize float64,
	definite bool,
	e *engine,
	intr trackIntrinsic,
	isMin bool,
) float64 {
	switch sz.kind {
	case trackFixed:
		if sz.val < 0 {
			// Percentage: cyclic honesty — indefinite container → auto (0 min / inf max).
			if !definite || contentSize < 0 {
				if isMin {
					return 0
				}
				return math.Inf(1)
			}
			pct := -sz.val
			return contentSize * pct / 100
		}
		if e != nil {
			return e.scalePt(sz.val)
		}
		return sz.val
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
