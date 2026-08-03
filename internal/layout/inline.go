package layout

import (
	"strings"

	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

// inlineItem is one atomic piece of inline content.
type inlineItem struct {
	text       string
	style      ResolvedStyle
	w, h       float64 // text: run width + line height; image: placed size
	ascent     float64
	descent    float64
	img        bool
	imgData    []byte
	imgJPEG    bool
	imgW       int
	imgH       int
	href       string
	forceBreak bool
	// block-in-inline: a laid-out block box whose ops live in
	// e.ops[opStart:opEnd] and need relocating when placed on a line.
	blockBox *box
	opStart  int
	opEnd    int
}

// layoutInline lays out inline content into line boxes and emits text/image
// ops. It returns the consumed height and records the first line's baseline
// on the box.
func (e *engine) layoutInline(b *box, nodes []*html.Node, availW, x, y float64) float64 {
	var items []inlineItem
	e.collectInline(nodes, &items)
	if len(items) == 0 {
		return 0
	}

	ly := y
	lastBreak := 0
	lineW := 0.0
	for i := 0; i < len(items); i++ {
		it := &items[i]
		if it.forceBreak {
			ly += e.emitLine(b, items, lastBreak, i, availW, x, ly)
			lastBreak = i + 1
			lineW = 0
			continue
		}
		if lineW > 0 && lineW+it.w > availW && !nowrap(it.style.WhiteSpace) {
			ly += e.emitLine(b, items, lastBreak, i, availW, x, ly)
			lastBreak = i
			lineW = 0
		}
		lineW += it.w
	}
	ly += e.emitLine(b, items, lastBreak, len(items), availW, x, ly)
	return ly - y
}

func nowrap(ws string) bool { return ws == "nowrap" }

// emitLine renders items[start:end) as one line and returns its height.
func (e *engine) emitLine(b *box, items []inlineItem, start, end int, availW, x, y float64) float64 {
	line := make([]inlineItem, end-start)
	copy(line, items[start:end])
	if len(line) == 0 {
		return 0
	}

	// trim trailing whitespace of the last run
	if !line[len(line)-1].img {
		last := &line[len(line)-1]
		trimmed := strings.TrimRight(last.text, " ")
		if trimmed != last.text {
			last.text = trimmed
			last.w = e.measureTextFace(trimmed, last.style)
		}
	}

	// Coalesce adjacent same-style text runs into one op so PDF/image paint
	// advances match layout (avoids word-by-word Tj gaps).
	line = coalesceTextItems(line)

	// line metrics
	maxAscent, maxDescent := 0.0, 0.0
	for i := range line {
		it := &line[i]
		if it.img {
			if it.h > maxAscent {
				maxAscent = it.h
			}
			continue
		}
		as := e.fontAscent(it.style.FontSize * e.scale)
		de := e.fontDescent(it.style.FontSize * e.scale)
		lh := lineHeightOf(&it.style) * e.scale
		extra := (lh - as - de) / 2
		if as+extra > maxAscent {
			maxAscent = as + extra
		}
		if de+extra > maxDescent {
			maxDescent = de + extra
		}
	}
	lh := maxAscent + maxDescent
	if lh <= 0 {
		lh = 1
	}
	baseline := y + maxAscent

	totalW := 0.0
	for i := range line {
		totalW += line[i].w
	}

	textAlign := "left"
	if b != nil && b.style.TextAlign != "" {
		textAlign = b.style.TextAlign
	}
	var lx float64
	switch textAlign {
	case "right":
		lx = x + availW - totalW
	case "center":
		lx = x + (availW-totalW)/2
	default:
		lx = x
	}

	for i := range line {
		it := &line[i]
		if it.blockBox != nil {
			dx := lx - it.blockBox.x
			dy := baseline - it.h - it.blockBox.y
			for k := it.opStart; k < it.opEnd; k++ {
				e.ops[k].X += dx
				e.ops[k].Y += dy
			}
			lx += it.blockBox.w
			continue
		}
		if it.img {
			top := baseline - it.h
			if it.imgData != nil {
				e.add(Op{Kind: OpImage, X: lx, Y: top, W: it.w, H: it.h,
					Image: it.imgData, ImgW: it.imgW, ImgH: it.imgH, IsJPEG: it.imgJPEG})
			}
			if it.href != "" {
				e.add(Op{Kind: OpLinkURI, X: lx, Y: top, W: it.w, H: it.h, URI: it.href})
			}
			lx += it.w
			continue
		}
		c := it.style.Color
		face := e.faceFor(it.style)
		e.add(Op{Kind: OpText, X: lx, Y: baseline, W: it.w, H: it.h,
			Text: it.text, Font: face, Size: it.style.FontSize * e.scale, Bold: it.style.FontWeight >= 700,
			R: c[0], G: c[1], B: c[2]})
		if it.style.TextDecoration == "underline" {
			e.add(Op{Kind: OpLine, X: lx, Y: baseline + it.descent*0.25, W: it.w, H: 0, R: c[0], G: c[1], B: c[2]})
		}
		if it.style.TextDecoration == "line-through" {
			e.add(Op{Kind: OpLine, X: lx, Y: baseline - it.ascent*0.3, W: it.w, H: 0, R: c[0], G: c[1], B: c[2]})
		}
		if it.href != "" {
			e.add(Op{Kind: OpLinkURI, X: lx, Y: baseline - it.ascent, W: it.w, H: it.ascent + it.descent, URI: it.href})
		}
		lx += it.w
	}

	if b != nil && b.firstBaseline == 0 {
		b.firstBaseline = baseline
	}
	return lh
}

// collectInline flattens inline child nodes into items.
func (e *engine) collectInline(nodes []*html.Node, out *[]inlineItem) {
	for _, n := range nodes {
		e.collectInlineNode(n, out)
	}
}

func (e *engine) collectInlineNode(n *html.Node, out *[]inlineItem) {
	st := e.styles[n]
	switch n.Type {
	case html.TextNode:
		if st.Display == "none" {
			return
		}
		switch st.WhiteSpace {
		case "pre":
			parts := strings.Split(n.Text, "\n")
			for i, p := range parts {
				if p != "" {
					*out = append(*out, e.textItem(p, st))
				}
				if i < len(parts)-1 {
					*out = append(*out, inlineItem{forceBreak: true})
				}
			}
		default:
			text := collapseWS(n.Text)
			if text == "" {
				return
			}
			for _, word := range strings.Fields(text) {
				*out = append(*out, e.textItem(word+" ", st))
			}
		}
	case html.ElementNode:
		if st.Display == "none" {
			return
		}
		if n.Name == "br" {
			*out = append(*out, inlineItem{forceBreak: true})
			return
		}
		if n.Name == "img" {
			ib := e.buildImage(n, st, 0, 0)
			*out = append(*out, inlineItem{
				img: true, imgData: ib.imgData, imgJPEG: ib.imgJPEG,
				imgW: ib.imgW, imgH: ib.imgH, w: ib.w, h: ib.h, style: st,
			})
			return
		}
		if st.Display == "inline" {
			href := ""
			if n.Name == "a" && isExternalHref(n.Attribute("href")) {
				href = n.Attribute("href")
			}
			before := len(*out)
			for _, c := range n.Children {
				e.collectInlineNode(c, out)
			}
			if href != "" {
				for i := before; i < len(*out); i++ {
					(*out)[i].href = href
				}
			}
			return
		}
		// block-level element inside inline context: lay out at a throwaway
		// offset, then shift its ops into the line when placed.
		opStart := len(e.ops)
		cb := e.build(n, availWForInline(), 0, 0)
		opEnd := len(e.ops)
		if cb != nil {
			*out = append(*out, inlineItem{
				img: true, w: cb.w, h: cb.h, style: st,
				blockBox: cb, opStart: opStart, opEnd: opEnd,
			})
		}
	}
}

// availWForInline is a generous width for block-in-inline measurement.
func availWForInline() float64 { return 1 << 30 }

func (e *engine) textItem(text string, st ResolvedStyle) inlineItem {
	w := e.measureTextFace(text, st)
	return inlineItem{text: text, style: st, w: w, h: lineHeightOf(&st) * e.scale}
}

// coalesceTextItems merges consecutive non-image text runs that share style
// and href so one text op paints the whole phrase.
func coalesceTextItems(line []inlineItem) []inlineItem {
	if len(line) < 2 {
		return line
	}
	out := make([]inlineItem, 0, len(line))
	out = append(out, line[0])
	for i := 1; i < len(line); i++ {
		cur := line[i]
		prev := &out[len(out)-1]
		if !cur.img && !prev.img && !cur.forceBreak && cur.href == prev.href &&
			sameInlineStyle(prev.style, cur.style) {
			prev.text += cur.text
			prev.w += cur.w
			continue
		}
		out = append(out, cur)
	}
	return out
}

func sameInlineStyle(a, b ResolvedStyle) bool {
	return a.FontSize == b.FontSize &&
		a.FontWeight == b.FontWeight &&
		a.FontItalic == b.FontItalic &&
		a.LetterSpacing == b.LetterSpacing &&
		a.Color == b.Color &&
		a.TextDecoration == b.TextDecoration &&
		a.WhiteSpace == b.WhiteSpace
}

func lineHeightOf(st *ResolvedStyle) float64 {
	if st.LineHeight > 0 {
		return st.LineHeight
	}
	return 1.2 * st.FontSize
}

func collapseWS(s string) string {
	var b strings.Builder
	prevSpace := true
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '\f' {
			if !prevSpace {
				b.WriteByte(' ')
				prevSpace = true
			}
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}
	return strings.TrimRight(b.String(), " ")
}

// measureText returns the width of s in points at the given size using the
// engine default face (for call sites without a style).
func (e *engine) measureText(s string, size float64) float64 {
	return e.measureWith(e.font, s, size, 0)
}

// measureTextFace measures s with the face selected for st.
func (e *engine) measureTextFace(s string, st ResolvedStyle) float64 {
	size := st.FontSize * e.scale
	return e.measureWith(e.faceFor(st), s, size, st.LetterSpacing*e.scale)
}

func (e *engine) measureWith(face *pdf.Font, s string, size, letterSpacing float64) float64 {
	if face == nil {
		face = e.font
	}
	var total float64
	n := 0
	for _, r := range s {
		total += face.AdvanceInPoints(r, size)
		n++
	}
	if letterSpacing != 0 && n > 0 {
		total += letterSpacing * float64(n)
	}
	return total
}

func (e *engine) fontAscent(size float64) float64 {
	return float64(e.font.Ascent()) * size / float64(e.font.UnitsPerEm())
}

func (e *engine) fontDescent(size float64) float64 {
	return float64(-e.font.Descent()) * size / float64(e.font.UnitsPerEm())
}

// isExternalHref reports whether a link target should become a URI
// annotation (http/https/mailto). Local links are out of scope for phase 04.
func isExternalHref(href string) bool {
	low := strings.ToLower(href)
	return strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://") || strings.HasPrefix(low, "mailto:")
}
