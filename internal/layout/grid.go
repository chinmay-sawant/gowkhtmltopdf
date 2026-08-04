package layout

import (
	"strconv"
	"strings"

	"gowkhtmltopdf/internal/html"
)

// buildGrid lays out a CSS grid lite: tracks from grid-template-columns,
// gap, auto-flow row, and column spanning (grid-column: span N / start / end).
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
	gap := e.scalePt(st.Gap)

	cols := parseGridTracks(st.GridTemplateColumns, contentW, gap, e)
	if len(cols) == 0 {
		cols = []float64{contentW}
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

	// Occupancy map: row → set of used column indices.
	type cell struct {
		n         *html.Node
		col, span int
		row       int
	}
	var placed []cell
	occ := map[int]map[int]bool{}
	ensure := func(row int) {
		if occ[row] == nil {
			occ[row] = map[int]bool{}
		}
	}
	freeAt := func(row, col, span int) bool {
		ensure(row)
		for i := 0; i < span; i++ {
			c := col + i
			if c >= len(cols) || occ[row][c] {
				return false
			}
		}
		return true
	}
	mark := func(row, col, span int) {
		ensure(row)
		for i := 0; i < span; i++ {
			occ[row][col+i] = true
		}
	}

	cursorRow, cursorCol := 0, 0
	for _, kid := range kids {
		cs := e.styles[kid]
		span := cs.GridColumnSpan
		if span < 1 {
			span = 1
		}
		if span > len(cols) {
			span = len(cols)
		}
		start := cs.GridColumnStart - 1 // 0-based
		row, col := cursorRow, cursorCol
		if start >= 0 {
			col = start
			// find first row where this start+span fits
			for !freeAt(row, col, span) {
				row++
			}
		} else {
			for {
				if col+span > len(cols) {
					col = 0
					row++
					continue
				}
				if freeAt(row, col, span) {
					break
				}
				col++
			}
		}
		mark(row, col, span)
		placed = append(placed, cell{n: kid, col: col, span: span, row: row})
		cursorRow, cursorCol = row, col+span
		if cursorCol >= len(cols) {
			cursorCol = 0
			cursorRow++
		}
	}

	// Lay out by rows.
	maxRow := 0
	for _, p := range placed {
		if p.row > maxRow {
			maxRow = p.row
		}
	}
	rowY := cy
	for r := 0; r <= maxRow; r++ {
		rowH := 0.0
		for _, p := range placed {
			if p.row != r {
				continue
			}
			cw := 0.0
			cx := contentX
			for j := 0; j < p.col; j++ {
				cx += cols[j] + gap
			}
			for j := 0; j < p.span; j++ {
				cw += cols[p.col+j]
				if j > 0 {
					cw += gap
				}
			}
			cb := e.build(p.n, cw, cx, y+rowY)
			if cb == nil {
				continue
			}
			dx := cx - cb.x
			dy := (y + rowY) - cb.y
			e.shiftBoxOps(cb, dx, dy)
			cb.x += dx
			cb.y += dy
			if cb.h > rowH {
				rowH = cb.h
			}
			b.children = append(b.children, cb)
		}
		cy = rowY + rowH
		if r < maxRow {
			cy += gap
		}
		rowY = cy
	}
	cy += e.scalePt(st.PaddingBottom)
	if st.Height >= 0 {
		h := e.scalePt(st.Height)
		if st.BoxSizing != "border-box" {
			h += e.scalePt(st.PaddingTop) + e.scalePt(st.PaddingBottom) +
				e.scalePt(st.BorderTop.Width) + e.scalePt(st.BorderBottom.Width)
		}
		if cy < h {
			cy = h
		}
	}
	b.h = cy
	e.prependChrome(contentStart, st, b.x, y, b.w, b.h)
	return b
}

// parseGridTracks parses a lite subset of grid-template-columns.
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
