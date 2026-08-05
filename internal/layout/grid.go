package layout

import (
	"strconv"
	"strings"

	"gowkhtmltopdf/internal/html"
)

// buildGrid lays out a CSS grid Stage B lite: column/row tracks, independent
// gaps, auto-flow row, column/row spanning, and justify/align-items/self.
func (e *engine) buildGrid(n *html.Node, st ResolvedStyle, availW, x, y float64) *box {
	ml, mr := e.scalePt(st.MarginLeft), e.scalePt(st.MarginRight)
	b := &box{node: n, style: st, kind: "block", x: x + ml, y: y}
	b.w = availW - ml - mr
	if b.w < 0 {
		b.w = 0
	}
	if st.WidthPercent >= 0 {
		b.w = availW * st.WidthPercent / 100
	} else if st.Width >= 0 {
		b.w = e.scalePt(st.Width)
		if st.BoxSizing != "border-box" {
			b.w += e.scalePt(st.PaddingLeft) + e.scalePt(st.PaddingRight) +
				e.scalePt(st.BorderLeft.Width) + e.scalePt(st.BorderRight.Width)
		}
	}
	contentW := b.w - e.scalePt(st.PaddingLeft) - e.scalePt(st.PaddingRight) -
		e.scalePt(st.BorderLeft.Width) - e.scalePt(st.BorderRight.Width)
	if contentW < 0 {
		contentW = 0
	}
	contentX := b.x + e.scalePt(st.BorderLeft.Width) + e.scalePt(st.PaddingLeft)
	contentStart := len(e.ops)
	cy := e.scalePt(st.PaddingTop) + e.scalePt(st.BorderTop.Width)

	rowGap, columnGap := e.gridGaps(st)

	cols := parseGridTracks(st.GridTemplateColumns, contentW, columnGap, e)
	if len(cols) == 0 {
		cols = []float64{contentW}
	}

	contentH := -1.0
	if st.Height >= 0 {
		h := e.scalePt(st.Height)
		if st.BoxSizing == "border-box" {
			h -= e.scalePt(st.PaddingTop) + e.scalePt(st.PaddingBottom) +
				e.scalePt(st.BorderTop.Width) + e.scalePt(st.BorderBottom.Width)
		}
		if h < 0 {
			h = 0
		}
		contentH = h
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

	cursorRow, cursorCol := 0, 0
	for _, kid := range kids {
		cs := e.styles[kid]
		colSpan := cs.GridColumnSpan
		if colSpan < 1 {
			colSpan = 1
		}
		if colSpan > len(cols) {
			colSpan = len(cols)
		}
		rowSpan := cs.GridRowSpan
		if rowSpan < 1 {
			rowSpan = 1
		}
		colStart := cs.GridColumnStart - 1 // 0-based; -1 = auto
		rowStart := cs.GridRowStart - 1

		row, col := cursorRow, cursorCol
		switch {
		case rowStart >= 0 && colStart >= 0:
			row, col = rowStart, colStart
			for !freeAt(row, col, rowSpan, colSpan) {
				row++
			}
		case colStart >= 0:
			col = colStart
			for !freeAt(row, col, rowSpan, colSpan) {
				row++
			}
		case rowStart >= 0:
			row = rowStart
			col = 0
			for {
				if col+colSpan > len(cols) {
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
			for {
				if col+colSpan > len(cols) {
					col = 0
					row++
					continue
				}
				if freeAt(row, col, rowSpan, colSpan) {
					break
				}
				col++
			}
		}
		mark(row, col, rowSpan, colSpan)
		placed = append(placed, cell{n: kid, col: col, colSpan: colSpan, row: row, rowSpan: rowSpan})
		cursorRow, cursorCol = row, col+colSpan
		if cursorCol >= len(cols) {
			cursorCol = 0
			cursorRow++
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
	if numRows < 1 {
		numRows = 1
	}

	definiteRows := contentH >= 0 && strings.TrimSpace(st.GridTemplateRows) != "" &&
		strings.ToLower(strings.TrimSpace(st.GridTemplateRows)) != "none"
	var rows []float64
	if definiteRows {
		rows = parseGridTracks(st.GridTemplateRows, contentH, rowGap, e)
	}
	if len(rows) == 0 {
		rows = make([]float64, numRows)
		// Honor explicit fixed track minimums when height is auto.
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
		// Without this, span-2 cells stay content-sized (fixture-32 Tall span-2).
		buildH := -1.0
		if (align == "stretch" || align == "") && cellH > 0 && cs.Height < 0 {
			buildH = cellH
		}

		var cb *box
		if buildH > 0 {
			prev := e.styles[pb.n]
			mod := prev
			mod.Height = buildH
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
	// Content-box height is the content area only; cy already includes
	// padding-top + border-top, usedH includes padding-bottom. Ensure the
	// border box covers padding-top/border-top that sit above content tracks.
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

// parseGridTrackFixedMins returns fixed (non-fr) track sizes as minimums for
// auto-height grids. fr / unknown tracks yield 0.
func parseGridTrackFixedMins(raw string, e *engine) []float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "none" {
		return nil
	}
	if strings.HasPrefix(strings.ToLower(raw), "repeat(") {
		inner := raw[len("repeat("):]
		if i := strings.LastIndex(inner, ")"); i >= 0 {
			inner = inner[:i]
		}
		parts := strings.SplitN(inner, ",", 2)
		if len(parts) == 2 {
			n, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err == nil && n > 0 && n < 64 {
				track := strings.TrimSpace(parts[1])
				if strings.HasSuffix(track, "fr") || track == "" {
					return nil
				}
				if v, ok := lengthBox(track, 12, 0, "auto"); ok && v >= 0 {
					sv := e.scalePt(v)
					out := make([]float64, n)
					for i := range out {
						out[i] = sv
					}
					return out
				}
			}
		}
		return nil
	}
	toks := strings.Fields(raw)
	out := make([]float64, 0, len(toks))
	for _, t := range toks {
		t = strings.TrimSpace(t)
		if strings.HasSuffix(t, "fr") {
			out = append(out, 0)
			continue
		}
		if v, ok := lengthBox(t, 12, 0, "auto"); ok && v >= 0 {
			out = append(out, e.scalePt(v))
			continue
		}
		out = append(out, 0)
	}
	return out
}

// parseGridTracks parses a lite subset of grid-template-columns/rows.
// columnGap is subtracted from contentW before distributing fr/equal tracks
// so (n tracks + n-1 gaps) fit the content box.
func parseGridTracks(raw string, contentW, columnGap float64, e *engine) []float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "none" {
		return nil
	}
	if strings.HasPrefix(strings.ToLower(raw), "repeat(") {
		inner := raw[len("repeat("):]
		if i := strings.LastIndex(inner, ")"); i >= 0 {
			inner = inner[:i]
		}
		parts := strings.SplitN(inner, ",", 2)
		if len(parts) == 2 {
			n, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err == nil && n > 0 && n < 64 {
				return expandTracks(n, strings.TrimSpace(parts[1]), contentW, columnGap, e)
			}
		}
	}
	toks := strings.Fields(raw)
	type track struct {
		fr    float64
		fixed float64
	}
	var tracks []track
	frSum := 0.0
	fixedSum := 0.0
	for _, t := range toks {
		t = strings.TrimSpace(t)
		if strings.HasSuffix(t, "fr") {
			v, err := strconv.ParseFloat(strings.TrimSuffix(t, "fr"), 64)
			if err != nil || v <= 0 {
				v = 1
			}
			tracks = append(tracks, track{fr: v, fixed: -1})
			frSum += v
			continue
		}
		if v, ok := lengthBox(t, 12, contentW, "auto"); ok && v >= 0 {
			sv := e.scalePt(v)
			tracks = append(tracks, track{fixed: sv})
			fixedSum += sv
			continue
		}
		tracks = append(tracks, track{fr: 1, fixed: -1})
		frSum += 1
	}
	gapTotal := 0.0
	if len(tracks) > 1 {
		gapTotal = columnGap * float64(len(tracks)-1)
	}
	avail := contentW - fixedSum - gapTotal
	if avail < 0 {
		avail = 0
	}
	out := make([]float64, len(tracks))
	for i, tr := range tracks {
		if tr.fixed >= 0 {
			out[i] = tr.fixed
			continue
		}
		if frSum > 0 {
			out[i] = avail * (tr.fr / frSum)
		}
	}
	return out
}

func expandTracks(n int, track string, contentW, columnGap float64, e *engine) []float64 {
	out := make([]float64, n)
	gapTotal := 0.0
	if n > 1 {
		gapTotal = columnGap * float64(n-1)
	}
	avail := contentW - gapTotal
	if avail < 0 {
		avail = 0
	}
	if strings.HasSuffix(track, "fr") || track == "" {
		each := avail / float64(n)
		for i := range out {
			out[i] = each
		}
		return out
	}
	if v, ok := lengthBox(track, 12, contentW, "auto"); ok && v >= 0 {
		sv := e.scalePt(v)
		for i := range out {
			out[i] = sv
		}
		return out
	}
	each := avail / float64(n)
	for i := range out {
		out[i] = each
	}
	return out
}
