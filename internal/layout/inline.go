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
	imgRef     *imageRef
	href       string
	forceBreak bool
	// block-in-inline: a laid-out block box whose ops live in
	// e.ops[opStart:opEnd] and need relocating when placed on a line.
	blockBox *box
	opStart  int
	opEnd    int
}

// layoutInlineFloats lays out inline content into line boxes and emits
// text/image ops. It returns the consumed height and records the first line's
// baseline on the box. When floats is non-nil, each line re-queries exclusion
// at its canvas Y so text widens again after a float ends mid-paragraph.
func (e *engine) layoutInlineFloats(b *box, nodes []*html.Node, contentW, contentX, y float64, floats *floatState) float64 {
	var items []inlineItem
	oldMax := e.imgMaxW
	oldCB := e.inlineCBW
	if contentW > 0 {
		e.imgMaxW = contentW
		e.inlineCBW = contentW
	}
	e.collectInline(nodes, &items)
	e.imgMaxW = oldMax
	e.inlineCBW = oldCB
	items = squeezeInlineSpaces(items)
	items = separateAdjacentCites(items, e)
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
		// Short remaining tail beside a float: if it fits as one full-width
		// line under the float, drop there instead of leaving an orphan in
		// the narrow column (e.g. wiki "big time."[71] left of a thumb).
		if floats != nil && lineW < contentW-0.5 {
			if next, ok := e.preferFloatClearForTail(items, i, contentW, lineW, ly, floats); ok {
				ly = next
				lineX, lineW = floats.exclusion(contentX, contentW, 0, ly)
				if lineW < 0 {
					lineW = 0
				}
			}
		}
		// Pack one line under current exclusion width.
		start := i
		lineAdv := 0.0
		for i < len(items) {
			it := &items[i]
			if it.forceBreak {
				break
			}
			// Split long unbreakable runs (URLs, paths, base64) that would
			// overflow the line. Honors overflow-wrap / word-break; also
			// emergency-breaks when a token is wider than the line so text
			// does not paint past the page edge (print PDF).
			// restMax for subsequent chunks uses contentW (full BFC width) so
			// pre-split fragments can reflow wider after a float ends.
			if !it.img && it.blockBox == nil && it.text != "" {
				room := lineW - lineAdv
				if room < 0 {
					room = 0
				}
				restW := contentW
				if restW < lineW {
					restW = lineW
				}
				if parts := e.breakOverflowItem(*it, room, lineW, restW, lineAdv == 0); len(parts) > 1 {
					items = append(items[:i], append(parts, items[i+1:]...)...)
					it = &items[i]
				}
			}
			adv := it.marginL + it.w + it.marginR
			// Always wrap to the next line when the next item does not fit.
			// white-space:nowrap must not glue an unbreakable span onto a line
			// that already has content and overflow into a float (wiki .IPA).
			// Exception: never break before attaching punctuation / mid-cite
			// (")[37]" → not ")\n[" or "[\n37]" or "saying.[\n7]").
			if lineAdv > 0 && lineAdv+adv > lineW {
				if i > start && noBreakBefore(items[i-1], items[i]) {
					// Keep short sticky tails (cite ")", "[37]", commas) even if
					// they overflow slightly. Do NOT glue multi-em tokens
					// (bare URLs after "(") onto the line — that painted past
					// the page edge on reference lists.
					//
					// Nowrap-to-nowrap chains (wiki Ref cells "[127][128]") must
					// stay on one line even when the cell is a hair narrower
					// than the cluster; stacking at the same X was worse.
					em := items[i].style.FontSize * e.scale
					if em < 1 {
						em = 10
					}
					nowrapCluster := items[i-1].style.WhiteSpace == "nowrap" &&
						items[i].style.WhiteSpace == "nowrap"
					limit := em * 2.5
					if nowrapCluster {
						limit = em * 8 // multi-cite / IPA fragments
					}
					if adv <= limit {
						lineAdv += adv
						i++
						for i < len(items) && !items[i].forceBreak &&
							noBreakBefore(items[i-1], items[i]) {
							a := items[i].marginL + items[i].w + items[i].marginR
							nc := items[i-1].style.WhiteSpace == "nowrap" &&
								items[i].style.WhiteSpace == "nowrap"
							lim := em * 2.5
							if nc {
								lim = em * 8
							}
							if a > lim {
								break
							}
							lineAdv += a
							i++
						}
					}
				}
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

// breakOverflowItem splits a text item that cannot fit in remainW on the
// current line (or in fullLineW alone). Returns nil when no split is needed
// or allowed; otherwise a sequence of items that replace the original.
// restLineW is the width used for chunks after the first (typically the BFC
// content width so fragments reflow full-width after a float ends).
//
// Policy (generic, not site-specific):
//   - word-break:break-all / overflow-wrap:anywhere → fill remainW, any grapheme
//   - overflow-wrap:break-word → soft then grapheme, but only when the token is
//     wider than a full line (CSS: do not mid-break a word that fits the next line)
//   - white-space:nowrap → never split
//   - otherwise emergency-split when the token alone exceeds fullLineW so
//     paint cannot run past the content box (print engines commonly do this)
func (e *engine) breakOverflowItem(it inlineItem, remainW, fullLineW, restLineW float64, aloneOnLine bool) []inlineItem {
	if it.img || it.blockBox != nil || it.forceBreak || it.text == "" {
		return nil
	}
	pol := wordBreakOf(it.style)
	if pol == breakNever {
		return nil
	}
	adv := it.marginL + it.w + it.marginR
	// Nothing to do when the whole item fits on this line.
	if adv <= remainW+0.01 {
		return nil
	}
	tokenExceedsLine := adv > fullLineW+0.01
	// Mid-line: a normal / break-word token that fits a full next line must
	// wrap whole — not mid-break into a tight remainW (captions: "International").
	if pol != breakAll && !tokenExceedsLine {
		return nil
	}
	// Emergency path (overflow-wrap:normal): only when alone on the line and
	// the token is wider than that line.
	if pol == breakNormal {
		if !(aloneOnLine && tokenExceedsLine) {
			// Defer to next line where aloneOnLine emergency can apply.
			return nil
		}
	}
	// Width available for the first chunk of text (exclude leading margin once).
	firstRoom := remainW - it.marginL
	if firstRoom < 1 {
		// No room on this line — let the outer packer wrap to the next line
		// unless we are already alone (then force a progressive split).
		if !aloneOnLine {
			return nil
		}
		firstRoom = fullLineW - it.marginL - it.marginR
	}
	if firstRoom < 1 {
		firstRoom = fullLineW
	}
	if firstRoom < 1 {
		return nil
	}
	if restLineW < fullLineW {
		restLineW = fullLineW
	}
	restRoom := restLineW - it.marginL - it.marginR
	if restRoom < 1 {
		restRoom = restLineW
	}
	if restRoom < 1 {
		return nil
	}
	chunks := e.breakToken(it.text, it.style, firstRoom, restRoom)
	if len(chunks) <= 1 {
		return nil
	}
	out := make([]inlineItem, 0, len(chunks))
	for i, chunk := range chunks {
		part := it
		part.text = chunk
		part.w = e.measureTextFace(chunk, it.style)
		if i > 0 {
			part.marginL = 0
		}
		if i < len(chunks)-1 {
			part.marginR = 0
		}
		out = append(out, part)
	}
	return out
}

// breakToken splits s into chunks that each fit firstMax (first piece) then
// restMax under wordBreakOf(st). Shared by inline overflow packing; soft-mode
// selection lives only here (and softModeOf) so measure and pack cannot drift.
func (e *engine) breakToken(s string, st ResolvedStyle, firstMax, restMax float64) []string {
	if s == "" {
		return nil
	}
	pol := wordBreakOf(st)
	if pol == breakNever {
		return []string{s}
	}
	return e.splitTextToWidth(s, st, firstMax, restMax, softModeOf(pol))
}

// preferFloatClearForTail reports whether remaining inline content from i
// should drop below active floats so the last line(s) use full content width
// instead of sitting as a short orphan beside the float.
func (e *engine) preferFloatClearForTail(items []inlineItem, i int, contentW, lineW, ly float64, floats *floatState) (nextY float64, ok bool) {
	if floats == nil || i >= len(items) {
		return ly, false
	}
	next := floats.clearY(ly)
	gap := next - ly
	if gap <= 0.5 {
		return ly, false
	}
	var rem float64
	estLH := 0.0
	for j := i; j < len(items); j++ {
		it := items[j]
		if it.forceBreak {
			// Hard break ends the tail we consider for a single-line clear.
			break
		}
		rem += it.marginL + it.w + it.marginR
		if estLH <= 0 && it.h > 0 {
			estLH = it.h
		}
	}
	if rem <= 0 {
		return ly, false
	}
	if estLH < 1 {
		estLH = 12
	}
	// Only when the whole remaining tail fits one full-width line (true
	// short ending), not when a long paragraph still has room to wrap
	// usefully beside the float.
	if rem > contentW+1 {
		return ly, false
	}
	// Near the float bottom, pull the last full-width-fitting chunk under the
	// float so we do not leave "…destined for the" beside + orphan
	// "big time."[71] under. Allow a slightly larger jump when the tail would
	// need multiple narrow lines but only one full-width line.
	maxGap := estLH * 2.5
	if rem > lineW+0.01 {
		maxGap = estLH * 3.5
	}
	if gap <= maxGap {
		return next, true
	}
	return ly, false
}

// separateAdjacentCites inserts a thin space between consecutive citation
// markers ("][") so [90][91][92] are not painted as a cramped cluster. Does
// not touch spaces inside a single marker ([111]).
func separateAdjacentCites(items []inlineItem, e *engine) []inlineItem {
	if len(items) < 2 {
		return items
	}
	out := make([]inlineItem, 0, len(items))
	out = append(out, items[0])
	for i := 1; i < len(items); i++ {
		cur := items[i]
		prev := &out[len(out)-1]
		if !prev.img && !cur.img && !prev.forceBreak && !cur.forceBreak &&
			prev.blockBox == nil && cur.blockBox == nil {
			pt := strings.TrimRight(prev.text, " ")
			ct := strings.TrimLeft(cur.text, " ")
			if strings.HasSuffix(pt, "]") && strings.HasPrefix(ct, "[") &&
				!strings.HasSuffix(prev.text, " ") && !strings.HasPrefix(cur.text, " ") {
				// Prefer a hair space so markers stay visually tight but not
				// colliding; fall back to a normal space if measure fails.
				gap := "\u200a" // hair space
				cur.text = gap + cur.text
				if e != nil {
					cur.w = e.measureTextFace(cur.text, cur.style)
				}
			}
		}
		out = append(out, cur)
	}
	return out
}

// softBreakMode selects where splitTextToWidth may insert breaks inside a token.
type softBreakMode int

const (
	softBreakNone softBreakMode = iota // grapheme only (word-break:break-all)
	softBreakURL                       // emergency: / ? & = # % : etc. (not hyphen)
	softBreakWord                      // overflow-wrap:break-word: URL + hyphen + underscore
)

// splitTextToWidth breaks s into substrings that each fit max widths.
// firstMax is for the first chunk (remaining space on the current line);
// restMax is for subsequent chunks (full line content width).
func (e *engine) splitTextToWidth(s string, st ResolvedStyle, firstMax, restMax float64, mode softBreakMode) []string {
	if s == "" {
		return nil
	}
	runes := []rune(s)
	if len(runes) <= 1 {
		return []string{s}
	}
	var out []string
	for len(runes) > 0 {
		limit := restMax
		if len(out) == 0 {
			limit = firstMax
		}
		if limit < 1 {
			limit = restMax
		}
		// Find the longest prefix that fits limit.
		n := 0
		var w float64
		for n < len(runes) {
			rw := e.measureTextFace(string(runes[n]), st)
			if n > 0 && w+rw > limit+0.01 {
				break
			}
			// Always take at least one rune so we make progress.
			if n == 0 && rw > limit+0.01 {
				n = 1
				w = rw
				break
			}
			w += rw
			n++
		}
		if n <= 0 {
			n = 1
		}
		// Prefer a soft wrap opportunity near the end of the fitting prefix
		// so URLs break after "/" etc. rather than mid-token when possible.
		if mode != softBreakNone && n < len(runes) && n > 1 {
			if soft := lastSoftBreak(runes[:n], mode); soft > 0 {
				n = soft
			}
		}
		out = append(out, string(runes[:n]))
		runes = runes[n:]
	}
	return out
}

// lastSoftBreak returns a split index after a soft-wrap character near the
// end of runes, or 0 if none is suitable (keep at least 1 rune on each side).
func lastSoftBreak(runes []rune, mode softBreakMode) int {
	for i := len(runes) - 1; i >= 1; i-- {
		if isSoftWrapRune(runes[i-1], mode) {
			return i
		}
	}
	return 0
}

func isSoftWrapRune(r rune, mode softBreakMode) bool {
	switch r {
	case '/', '\\', '?', '&', '=', '%', '#', ':', ';', ',', '+', '~', '@',
		'!', '*', '(', ')', '[', ']', '{', '}':
		return true
	case '-', '_', '.':
		// Hyphen/underscore/dot: only for overflow-wrap soft breaks, not pure
		// emergency (keeps "inner-a" / file names intact when slightly tight).
		return mode == softBreakWord
	}
	return false
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
	// CSS justify expands inter-word spaces only — not every inline box
	// boundary. Expanding after every item put rivers before commas, cites
	// ("word [1]"), and apostrophes ("Roth 's") on wiki print pages.
	switch textAlign {
	case "right":
		lx = x + availW - totalW
	case "center":
		lx = x + (availW-totalW)/2
	case "justify":
		lx = x
		if !lastLine && availW > totalW && len(line) > 1 {
			gaps := 0
			for i := 0; i < len(line)-1; i++ {
				if isJustifyGapAfter(line[i]) {
					gaps++
				}
			}
			if gaps > 0 {
				raw := (availW - totalW) / float64(gaps)
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
		}
	default:
		lx = x
	}

	// Coalesce adjacent same-href (or same-decoration) underline segments on
	// this line into ONE OpLine. Multi-face items already share one span; this
	// also merges bold/italic/nested chunks and skips whitespace-only runs so
	// dense reference lists (wrapped archive.org URLs) do not paint a forest
	// of short double-rules.
	type undRun struct {
		active  bool
		x, y, w float64
		uw      float64
		r, g, b float64
		href    string
		hasHref bool
	}
	var und undRun
	flushUnd := func() {
		if und.active && und.w > 0.01 {
			e.add(Op{Kind: OpLine, X: und.x, Y: und.y, W: und.w, H: 0,
				Width: und.uw, R: und.r, G: und.g, B: und.b})
		}
		und = undRun{}
	}

	for i := range line {
		it := &line[i]
		lx += it.marginL
		if it.blockBox != nil {
			flushUnd()
			dx := lx - it.blockBox.x
			dy := baseline - it.h - it.blockBox.y
			e.shiftBoxOps(it.blockBox, dx, dy)
			it.blockBox.x += dx
			it.blockBox.y += dy
			// Attach to parent so paint-time transforms/opacity stamp the subtree.
			if b != nil {
				b.children = append(b.children, it.blockBox)
			}
			lx += it.blockBox.w + it.marginR
			if i < len(line)-1 && isJustifyGapAfter(*it) {
				lx += justifyGap
			}
			continue
		}
		if it.img {
			flushUnd()
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
			if it.imgRef != nil && it.imgRef.data != nil {
				e.add(Op{Kind: OpImage, X: lx, Y: top, W: it.w, H: it.h,
					Image: it.imgRef.data, ImgW: it.imgRef.w, ImgH: it.imgRef.h, IsJPEG: it.imgRef.isJPEG})
			}
			if it.href != "" {
				e.add(Op{Kind: OpLinkURI, X: lx, Y: top, W: it.w, H: it.h, URI: it.href})
			}
			lx += it.w + it.marginR
			if i < len(line)-1 && isJustifyGapAfter(*it) {
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
		runs := e.splitTextByFace(it.text, it.style)
		runStart := lx
		var runSpan float64
		for _, run := range runs {
			e.add(Op{Kind: OpText, X: lx, Y: baseline, W: run.w, H: it.h,
				Text: run.text, Font: run.face, Size: size, Bold: it.style.FontWeight >= 700,
				R: c[0], G: c[1], B: c[2]})
			if it.href != "" {
				e.add(Op{Kind: OpLinkURI, X: lx, Y: baseline - as, W: run.w, H: as + de, URI: it.href})
			}
			lx += run.w
			runSpan += run.w
		}
		// Decoration: one stroke per logical link run on this line (not per
		// face-run / nested style chunk). Thin stroke ~5% em, clamped for
		// dense reference print (min 0.25pt, max 0.45pt).
		// Force-underline a[href] for PDF affordance. Bare URL strings
		// (https://…, archive fragments) never get underlines — multi-line
		// ref lists were a forest of rules; titles/prose links still underline.
		bareURL := isBareURLText(it.text)
		wantUnderline := !bareURL && (it.style.TextDecoration == "underline" ||
			(it.href != "" && it.style.TextDecoration != "line-through"))
		wsOnly := strings.TrimSpace(it.text) == ""
		if runSpan > 0.01 && (wantUnderline || it.style.TextDecoration == "line-through") {
			uw := underlineStrokeWidth(size)
			if wantUnderline {
				// Sit clearly below glyph descenders (~1–2mm visual gap).
				uy := baseline + de + size*0.22
				if wsOnly {
					// Do not start an underline on whitespace-only, but extend
					// an active same-href run across inter-word spaces.
					if und.active && it.href != "" && und.hasHref && it.href == und.href {
						end := runStart + runSpan
						if end > und.x+und.w {
							und.w = end - und.x
						}
						if uw < und.uw {
							und.uw = uw
						}
					}
				} else if und.active && nearUndY(und.y, uy) &&
					((it.href != "" && und.hasHref && it.href == und.href) ||
						(it.href == "" && !und.hasHref)) &&
					// Allow justify rivers / margins between nested chunks
					// (up to ~2em) without splitting the underline.
					runStart <= und.x+und.w+size*2+0.5 {
					// Extend continuous underline over adjacent same-href text.
					end := runStart + runSpan
					if end > und.x+und.w {
						und.w = end - und.x
					}
					if uw < und.uw {
						und.uw = uw
					}
					// Prefer first chunk's color (title) when styles mix.
				} else {
					flushUnd()
					und = undRun{
						active: true, x: runStart, y: uy, w: runSpan, uw: uw,
						r: c[0], g: c[1], b: c[2],
						href: it.href, hasHref: it.href != "",
					}
				}
			} else {
				flushUnd()
			}
			if it.style.TextDecoration == "line-through" && !wsOnly {
				e.add(Op{Kind: OpLine, X: runStart, Y: baseline - as*0.3, W: runSpan, H: 0,
					Width: uw, R: c[0], G: c[1], B: c[2]})
			}
		} else if !wantUnderline {
			flushUnd()
		}
		lx += it.marginR
		if i < len(line)-1 && isJustifyGapAfter(*it) {
			lx += justifyGap
		}
	}
	flushUnd()

	if b != nil && b.firstBaseline == 0 {
		b.firstBaseline = baseline
	}
	return lh
}

// underlineStrokeWidth returns a light print-friendly underline thickness.
// ~5% em, clamped so small ref text stays visible without dense double-rules.
func underlineStrokeWidth(em float64) float64 {
	uw := em * 0.05
	if uw < 0.25 {
		uw = 0.25
	}
	if uw > 0.45 {
		uw = 0.45
	}
	return uw
}

// isBareURLText reports that s is essentially a URL (optional leading
// punctuation from "(https://…)" wrappers). Used to skip force-underlines on
// raw link text in reference lists.
func isBareURLText(s string) bool {
	t := strings.TrimSpace(s)
	t = strings.TrimLeft(t, "([\"' \t")
	if t == "" {
		return false
	}
	low := strings.ToLower(t)
	if strings.HasPrefix(low, "https://") || strings.HasPrefix(low, "http://") ||
		strings.HasPrefix(low, "www.") {
		return true
	}
	// Continuation fragment of a wrapped URL (no scheme on later lines).
	if strings.Contains(t, "/") && !strings.ContainsAny(t, " \t") &&
		(strings.Contains(t, ".com") || strings.Contains(t, ".org") ||
			strings.Contains(t, "archive.") || strings.Contains(t, ".html") ||
			strings.Contains(t, ".php") || strings.Count(t, "/") >= 2) {
		return true
	}
	return false
}

func nearUndY(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 0.5
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
			// Whitespace-only text nodes still separate adjacent inlines
			// (wiki "Cuba"+" "+"Spain" / pretty-printed newlines between
			// anchors). collapseWS would drop them. Skip only after a
			// replaced element so `<img>\n<span margin-left>` does not add
			// a space on top of the margin (TestLogoTitleGap).
			if strings.TrimSpace(n.Text) == "" {
				if n.Text != "" {
					if len(*out) == 0 || !(*out)[len(*out)-1].img {
						*out = append(*out, e.textItem(" ", st))
					}
				}
				return
			}
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
			// collapseWS / Fields strip a leading space that still separates
			// this node from the previous inline ("</a> in" → "Reeves"+"in").
			// Re-introduce one when the source began with whitespace and the
			// prior item does not already end with a space.
			// Do not insert a space before attaching punctuation (", . ) ]") —
			// pretty-printed "</a>\n," must stay "Award," not "Award ,".
			if len(words) > 0 && len(n.Text) > 0 && len(*out) > 0 {
				first := n.Text[0]
				if first == ' ' || first == '\t' || first == '\n' || first == '\r' || first == '\f' {
					prev := &(*out)[len(*out)-1]
					if !prev.forceBreak && !strings.HasSuffix(prev.text, " ") &&
						!isAttachPunct(words[0]) {
						words[0] = " " + words[0]
					}
				}
			}
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
				img: true, imgRef: ib.img, w: ib.w, h: ib.h, style: st,
				marginL: e.scalePt(st.MarginLeft), marginR: e.scalePt(st.MarginRight),
			})
			return
		}
		if st.Display == "inline-block" {
			avail := e.inlineBlockAvail(n, st, e.inlineCBW)
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
			if txt := e.pseudoContent(n, "before"); txt != "" {
				*out = append(*out, e.textItem(txt, st))
			}
			for _, c := range n.Children {
				e.collectInlineNode(c, out)
			}
			if txt := e.pseudoContent(n, "after"); txt != "" {
				*out = append(*out, e.textItem(txt, st))
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
func (e *engine) inlineBlockAvail(n *html.Node, st ResolvedStyle, cbW float64) float64 {
	if st.WidthPercent >= 0 {
		// Prefer the inline formatting-context width; fall back to viewport.
		if cbW > 0 {
			return cbW * st.WidthPercent / 100
		}
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

// squeezeInlineSpaces drops artificial space items that sit immediately before
// attaching punctuation (pretty-printed "</a>\n," → "Award ,") or that are
// redundant after a trailing space already on the previous item.
func squeezeInlineSpaces(items []inlineItem) []inlineItem {
	if len(items) < 2 {
		return items
	}
	out := make([]inlineItem, 0, len(items))
	for i := 0; i < len(items); i++ {
		it := items[i]
		if !it.img && !it.forceBreak && it.blockBox == nil && strings.TrimSpace(it.text) == "" {
			// Pure space item.
			if i+1 < len(items) {
				next := items[i+1]
				if !next.img && !next.forceBreak && isAttachPunct(next.text) {
					continue // drop space before "," / ")" / "]" …
				}
				if strings.HasPrefix(strings.TrimSpace(next.text), "[") {
					continue // drop space before citation bracket
				}
			}
			if len(out) > 0 && strings.HasSuffix(out[len(out)-1].text, " ") {
				continue // redundant double space
			}
		}
		out = append(out, it)
	}
	return out
}

// isJustifyGapAfter reports whether CSS text-align:justify may expand after it.
// Only inter-word spaces stretch — not boundaries before punctuation or cites.
func isJustifyGapAfter(it inlineItem) bool {
	if it.img || it.forceBreak || it.blockBox != nil {
		return false
	}
	return strings.HasSuffix(it.text, " ")
}

// isAttachPunct is true for tokens that glue to the previous word (no leading space).
func isAttachPunct(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	r := []rune(s)[0]
	switch r {
	case ',', '.', ';', ':', '!', '?', ')', ']', '}', '\'', '"', '%',
		'\u201d' /* ” */, '\u2019' /* ’ */, '\u2013' /* – */, '\u2014': /* — */
		return true
	}
	return false
}

// noBreakBefore reports that a line break between prev and cur would split
// punctuation or a citation marker unnaturally.
func noBreakBefore(prev, cur inlineItem) bool {
	if cur.forceBreak || prev.forceBreak {
		return false
	}
	if cur.img || prev.img || cur.blockBox != nil || prev.blockBox != nil {
		return false
	}
	// Keep nowrap runs together (wiki .reference / .IPA pieces).
	if prev.style.WhiteSpace == "nowrap" && cur.style.WhiteSpace == "nowrap" {
		return true
	}
	ct := strings.TrimSpace(cur.text)
	if ct == "" {
		return false
	}
	// Do not start a line with closing/attach punctuation.
	if isAttachPunct(ct) {
		return true
	}
	// Digits that continue a cite: "[" then "37" then "]".
	if isAllDigits(ct) {
		pt := strings.TrimSpace(prev.text)
		if strings.HasSuffix(pt, "[") || isAllDigits(pt) {
			return true
		}
	}
	// Do not break after opening brackets / quotes.
	pt := strings.TrimRight(prev.text, " ")
	if pt != "" {
		runes := []rune(pt)
		switch runes[len(runes)-1] {
		case '[', '(', '{', '"', '\'', '\u201c' /* “ */, '\u2018': /* ‘ */
			return true
		}
	}
	return false
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
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
