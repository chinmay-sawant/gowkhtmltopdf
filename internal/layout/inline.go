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
	marginL    float64 // leading horizontal margin (e.g. span margin-left)
	marginR    float64 // trailing horizontal margin
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
// on the box. When floats is non-nil, each line re-queries exclusion at its
// canvas Y so text widens again after a float ends mid-paragraph.
func (e *engine) layoutInline(b *box, nodes []*html.Node, availW, x, y float64) float64 {
	return e.layoutInlineFloats(b, nodes, availW, x, y, nil)
}

func (e *engine) layoutInlineFloats(b *box, nodes []*html.Node, contentW, contentX, y float64, floats *floatState) float64 {
	var items []inlineItem
	oldMax := e.imgMaxW
	if contentW > 0 {
		e.imgMaxW = contentW
	}
	e.collectInline(nodes, &items)
	e.imgMaxW = oldMax
	if len(items) == 0 {
		return 0
	}

	ly := y
	i := 0
	for i < len(items) {
		lineX, lineW := contentX, contentW
		if floats != nil {
			lineX, lineW = floats.exclusion(contentX, contentW, 0, ly)
		}
		if lineW < 0 {
			lineW = 0
		}
		// Pack one line under current exclusion width.
		start := i
		lineAdv := 0.0
		for i < len(items) {
			it := &items[i]
			if it.forceBreak {
				break
			}
			adv := it.marginL + it.w + it.marginR
			// Always wrap to the next line when the next item does not fit.
			// white-space:nowrap must not glue an unbreakable span onto a line
			// that already has content and overflow into a float (wiki .IPA).
			if lineAdv > 0 && lineAdv+adv > lineW {
				break
			}
			// Empty line beside a float too narrow for this item: CSS2.1 §9.5
			// pushes the line box below the float and recomputes width.
			if lineAdv == 0 && adv > lineW && floats != nil && lineW < contentW-0.5 {
				if next := floats.clearY(ly); next > ly+0.5 {
					ly = next
					lineX, lineW = floats.exclusion(contentX, contentW, 0, ly)
					if lineW < 0 {
						lineW = 0
					}
					// Retry this item at the new Y (do not advance i).
					if adv > lineW && lineW < contentW-0.5 {
						// Still too narrow (e.g. both sides floated) — emit anyway.
					} else {
						continue
					}
				}
			}
			lineAdv += adv
			i++
		}
		end := i
		if i < len(items) && items[i].forceBreak {
			i++ // consume br
		}
		if end == start {
			// Single unbreakable item wider than line — still emit it.
			if i < len(items) && !items[i].forceBreak {
				i++
				end = i
			} else {
				continue
			}
		}
		lastLine := i >= len(items)
		ly += e.emitLine(b, items, start, end, lineW, lineX, ly, lastLine)
	}
	return ly - y
}

// emitLine renders items[start:end) as one line and returns its height.
// lastLine is true for the final line of the inline formatting context (used
// so text-align:justify leaves the last line start-aligned).
func (e *engine) emitLine(b *box, items []inlineItem, start, end int, availW, x, y float64, lastLine bool) float64 {
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

	textAlign := "left"
	if b != nil && b.style.TextAlign != "" {
		textAlign = b.style.TextAlign
	}

	// Coalesce adjacent same-style text runs into one op so PDF/image paint
	// advances match layout (avoids word-by-word Tj gaps). Skip when
	// justifying — gaps are distributed between word items.
	if textAlign != "justify" {
		line = coalesceTextItems(line)
	}

	// line metrics
	maxAscent, maxDescent := 0.0, 0.0
	for i := range line {
		it := &line[i]
		if it.img || it.blockBox != nil {
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
		totalW += line[i].marginL + line[i].w + line[i].marginR
	}

	var lx float64
	justifyGap := 0.0
	switch textAlign {
	case "right":
		lx = x + availW - totalW
	case "center":
		lx = x + (availW-totalW)/2
	case "justify":
		lx = x
		// Distribute leftover between word slots on non-final lines.
		// Cap expansion so short lines beside floats do not blow out to
		// huge rivers (Chrome-like readability; wiki print uses justify).
		if !lastLine && availW > totalW && len(line) > 1 {
			gaps := float64(len(line) - 1)
			raw := (availW - totalW) / gaps
			maxGap := 6.0 // pt
			for i := range line {
				if fs := line[i].style.FontSize * e.scale; fs > maxGap {
					maxGap = fs // up to 1em extra between words
				}
			}
			if raw <= maxGap*2 {
				if raw > maxGap {
					raw = maxGap // soft-cap rivers without abandoning justify
				}
				justifyGap = raw
			}
		}
	default:
		lx = x
	}

	for i := range line {
		it := &line[i]
		lx += it.marginL
		if it.blockBox != nil {
			dx := lx - it.blockBox.x
			dy := baseline - it.h - it.blockBox.y
			for k := it.opStart; k < it.opEnd; k++ {
				e.ops[k].X += dx
				e.ops[k].Y += dy
			}
			it.blockBox.x += dx
			it.blockBox.y += dy
			// Attach to parent so paint-time transforms/opacity stamp the subtree.
			if b != nil {
				b.children = append(b.children, it.blockBox)
			}
			lx += it.blockBox.w + it.marginR
			if i < len(line)-1 {
				lx += justifyGap
			}
			continue
		}
		if it.img {
			top := baseline - it.h
			va := it.style.VerticalAlign
			switch va {
			case "top":
				top = y
			case "middle":
				top = y + (lh-it.h)/2
			case "bottom":
				top = y + lh - it.h
			}
			if it.imgData != nil {
				e.add(Op{Kind: OpImage, X: lx, Y: top, W: it.w, H: it.h,
					Image: it.imgData, ImgW: it.imgW, ImgH: it.imgH, IsJPEG: it.imgJPEG})
			}
			if it.href != "" {
				e.add(Op{Kind: OpLinkURI, X: lx, Y: top, W: it.w, H: it.h, URI: it.href})
			}
			lx += it.w + it.marginR
			if i < len(line)-1 {
				lx += justifyGap
			}
			continue
		}
		c := it.style.Color
		size := it.style.FontSize * e.scale
		as := e.fontAscent(size)
		de := e.fontDescent(size)
		if as+de < size*0.5 {
			// Fallback when font metrics are missing — keep hit targets usable.
			as = size * 0.8
			de = size * 0.2
		}
		for _, run := range e.splitTextByFace(it.text, it.style) {
			e.add(Op{Kind: OpText, X: lx, Y: baseline, W: run.w, H: it.h,
				Text: run.text, Font: run.face, Size: size, Bold: it.style.FontWeight >= 700,
				R: c[0], G: c[1], B: c[2]})
			if it.style.TextDecoration == "underline" || (it.href != "" && it.style.TextDecoration != "line-through") {
				// Sit clearly below glyph descenders (~1–2mm visual gap).
				uy := baseline + de + size*0.22
				e.add(Op{Kind: OpLine, X: lx, Y: uy, W: run.w, H: 0, R: c[0], G: c[1], B: c[2]})
			}
			if it.style.TextDecoration == "line-through" {
				e.add(Op{Kind: OpLine, X: lx, Y: baseline - as*0.3, W: run.w, H: 0, R: c[0], G: c[1], B: c[2]})
			}
			if it.href != "" {
				e.add(Op{Kind: OpLinkURI, X: lx, Y: baseline - as, W: run.w, H: as + de, URI: it.href})
			}
			lx += run.w
		}
		lx += it.marginR
		if i < len(line)-1 {
			lx += justifyGap
		}
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
			// white-space:nowrap — keep the run unbreakable (wiki .reference
			// cite markers in narrow table columns).
			if st.WhiteSpace == "nowrap" {
				*out = append(*out, e.textItem(text, st))
				return
			}
			words := strings.Fields(text)
			for i, word := range words {
				// Space only between words of this text node — not after the
				// last token. Appending " " to every field made cite brackets
				// render as "[ 111 ]" and wrap/overlap in narrow Ref columns.
				if i < len(words)-1 {
					word += " "
				}
				*out = append(*out, e.textItem(word, st))
			}
			// Preserve a trailing word-separator when the source text node
			// ended with whitespace (so "foo <b>bar</b>" keeps the gap).
			if len(words) > 0 && len(n.Text) > 0 {
				last := n.Text[len(n.Text)-1]
				if last == ' ' || last == '\t' || last == '\n' || last == '\r' || last == '\f' {
					(*out)[len(*out)-1].text += " "
					(*out)[len(*out)-1].w = e.measureTextFace((*out)[len(*out)-1].text, st)
				}
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
				marginL: e.scalePt(st.MarginLeft), marginR: e.scalePt(st.MarginRight),
			})
			return
		}
		if st.Display == "inline-block" {
			avail := e.inlineBlockAvail(n, st)
			opStart := len(e.ops)
			cb := e.build(n, avail, 0, 0)
			opEnd := len(e.ops)
			if cb != nil {
				*out = append(*out, inlineItem{
					img: true, w: cb.w, h: cb.h, style: st,
					blockBox: cb, opStart: opStart, opEnd: opEnd,
					marginL: e.scalePt(st.MarginLeft), marginR: e.scalePt(st.MarginRight),
				})
			}
			return
		}
		if st.Display == "inline" {
			href := ""
			if n.Name == "a" {
				h := n.Attribute("href")
				if isExternalHref(h) || isInternalHref(h) {
					href = h
				}
			}
			before := len(*out)
			for _, c := range n.Children {
				e.collectInlineNode(c, out)
			}
			// Horizontal margins on inline elements (e.g. .co { margin-left: 10px }
			// after a logo) apply to the first/last generated items.
			if before < len(*out) {
				(*out)[before].marginL += e.scalePt(st.MarginLeft)
				(*out)[len(*out)-1].marginR += e.scalePt(st.MarginRight)
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

// inlineBlockAvail returns the containing-block width used to lay out an
// inline-block: specified width when present, otherwise shrink-to-fit capped
// at a generous max so auto-width badges size to their content.
func (e *engine) inlineBlockAvail(n *html.Node, st ResolvedStyle) float64 {
	if st.WidthPercent >= 0 {
		// Percent of viewport is a best-effort stand-in; real containing
		// block width is not threaded into collectInline.
		if e.opts.Width > 0 {
			return e.opts.Width * st.WidthPercent / 100
		}
	}
	if st.Width >= 0 {
		// buildBlock applies box-sizing to the specified width; pass enough
		// avail that auto-fill does not stretch a definite-width box.
		w := e.scalePt(st.Width)
		if st.BoxSizing != "border-box" {
			w += e.scalePt(st.PaddingLeft) + e.scalePt(st.PaddingRight) +
				e.scalePt(st.BorderLeft.Width) + e.scalePt(st.BorderRight.Width)
		}
		return w + e.scalePt(st.MarginLeft) + e.scalePt(st.MarginRight)
	}
	if isSizeContainer(st) {
		// Size containment: shrink-to-fit as-if-empty.
		intr := e.scalePt(st.PaddingLeft) + e.scalePt(st.PaddingRight) +
			e.scalePt(st.BorderLeft.Width) + e.scalePt(st.BorderRight.Width) +
			e.scalePt(st.MarginLeft) + e.scalePt(st.MarginRight)
		if intr < 1 {
			intr = 1
		}
		return intr
	}
	intr := e.measureCellContent(n, st)
	intr += e.scalePt(st.PaddingLeft) + e.scalePt(st.PaddingRight) +
		e.scalePt(st.BorderLeft.Width) + e.scalePt(st.BorderRight.Width) +
		e.scalePt(st.MarginLeft) + e.scalePt(st.MarginRight)
	if intr < 1 {
		intr = 1
	}
	return intr
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
			// first item keeps marginL; last item's marginR wins
			prev.marginR = cur.marginR
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

// measureTextFace measures s using per-rune CSS font-family fallback
// (same face selection as paint).
func (e *engine) measureTextFace(s string, st ResolvedStyle) float64 {
	size := st.FontSize * e.scale
	ls := st.LetterSpacing * e.scale
	var total float64
	n := 0
	for _, r := range s {
		face := e.faceForRune(st, r)
		if face == nil {
			face = e.font
		}
		total += face.AdvanceInPoints(r, size)
		n++
	}
	if ls != 0 && n > 0 {
		total += ls * float64(n)
	}
	return total
}

type faceRun struct {
	text string
	face *pdf.Font
	w    float64
}

// splitTextByFace splits s into contiguous runs that share the same face
// under CSS font-family fallback.
func (e *engine) splitTextByFace(s string, st ResolvedStyle) []faceRun {
	if s == "" {
		return nil
	}
	size := st.FontSize * e.scale
	var runs []faceRun
	var cur faceRun
	for _, r := range s {
		face := e.faceForRune(st, r)
		if face == nil {
			face = e.font
		}
		if cur.face == nil {
			cur = faceRun{face: face}
		} else if face != cur.face {
			runs = append(runs, cur)
			cur = faceRun{face: face}
		}
		cur.text += string(r)
		cur.w += face.AdvanceInPoints(r, size)
	}
	if cur.face != nil {
		runs = append(runs, cur)
	}
	return runs
}

func (e *engine) measureWith(face *pdf.Font, s string, size, letterSpacing float64) float64 {
	if face == nil {
		face = e.font
	}
	fallback := e.font // Liberation (or engine default) for glyphs the face lacks
	var total float64
	n := 0
	for _, r := range s {
		advFace := face
		if face.GlyphID(r) == 0 && fallback != nil && fallback.GlyphID(r) != 0 {
			advFace = fallback
		}
		total += advFace.AdvanceInPoints(r, size)
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
// annotation (http/https/mailto). Local same-document fragments are handled
// separately as internal GoTo links.
func isExternalHref(href string) bool {
	low := strings.ToLower(href)
	return strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://") || strings.HasPrefix(low, "mailto:")
}

// isInternalHref reports a same-document fragment link (#id).
func isInternalHref(href string) bool {
	return strings.HasPrefix(href, "#") && len(href) > 1
}
