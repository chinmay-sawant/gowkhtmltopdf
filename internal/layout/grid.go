package layout

import (
	"strconv"
	"strings"

	"gowkhtmltopdf/internal/html"
)

// buildGrid lays out a simple CSS grid: equal or fixed tracks from
// grid-template-columns, with gap. Auto-flow is row. Nested grid and
// spanning are out of scope.
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

	cols := parseGridTracks(st.GridTemplateColumns, contentW, e)
	if len(cols) == 0 {
		cols = []float64{contentW}
	}
	gap := e.scalePt(st.Gap)
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

	col := 0
	rowY := cy
	rowH := 0.0
	for i, kid := range kids {
		cw := cols[col]
		cx := contentX
		for j := 0; j < col; j++ {
			cx += cols[j] + gap
		}
		cb := e.build(kid, cw, cx, y+rowY)
		if cb != nil {
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
		col++
		if col >= len(cols) || i == len(kids)-1 {
			cy = rowY + rowH
			if i < len(kids)-1 {
				cy += gap
			}
			rowY = cy
			rowH = 0
			col = 0
		}
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
func parseGridTracks(raw string, contentW float64, e *engine) []float64 {
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
				return expandTracks(n, strings.TrimSpace(parts[1]), contentW, e)
			}
		}
	}
	toks := strings.Fields(raw)
	type track struct {
		fr    float64 // >0 means fr weight
		fixed float64 // >=0 means definite pt width; -1 unset
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
	avail := contentW - fixedSum
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

func expandTracks(n int, track string, contentW float64, e *engine) []float64 {
	out := make([]float64, n)
	if strings.HasSuffix(track, "fr") || track == "" {
		each := contentW / float64(n)
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
	each := contentW / float64(n)
	for i := range out {
		out[i] = each
	}
	return out
}
