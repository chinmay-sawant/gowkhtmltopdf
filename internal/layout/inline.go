package layout

import (
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

const (
	cssTagA                      = "a"
	cssTagBR                     = "br"
	cssTagImg                    = "img"
	cssDisplayInline             = "inline"
	cssDisplayInlineBlock        = "inline-block"
	cssDisplayNone               = "none"
	cssWhiteSpaceNowrap          = "nowrap"
	cssWhiteSpacePre             = "pre"
	cssWhiteSpacePreWrap         = "pre-wrap"
	cssWhiteSpacePreLine         = "pre-line"
	writingModeHorizontalTB      = "horizontal-tb"
	cssTextAlignJustify          = "justify"
	cssVerticalAlignBottom       = "bottom"
	cssVerticalAlignMiddle       = "middle"
	cssVerticalAlignTop          = "top"
	cssTextDecorationLineThrough = "line-through"
	cssTextDecorationUnderline   = "underline"
	writingModeVerticalRL        = "vertical-rl"
	writingModeVerticalLR        = "vertical-lr"
	nonASCIIStart                = 0x80
)

// inlineItem is one atomic piece of inline content.
type inlineItem struct {
	text       string
	style      *ResolvedStyle
	w, h       float64 // text: run width + line height; image: placed size
	marginL    float64 // leading horizontal margin (e.g. span margin-left)
	marginR    float64 // trailing horizontal margin
	img        bool
	thumbImg   bool // img inside a collapsed wiki figure; outer frame owns L/R/T
	chrome     bool // text belongs to an inline element with its own decoration
	noSplit    bool // vertical writing-mode run must remain one rotated line
	imgRef     *imageRef
	alt        string
	href       string
	forceBreak bool
	// block-in-inline: a laid-out block box whose ops live in
	// e.ops[opStart:opEnd] and need relocating when placed on a line.
	blockBox *box
	opStart  int
	opEnd    int
}

func (e *engine) collectAndPrepareInlineItems(nodes []*html.Node, contentW float64) []inlineItem {
	items := e.acquireInlineItems()

	oldMax := e.imgMaxW
	oldCB := e.inlineCBW

	if contentW > 0 {
		e.imgMaxW = contentW
		e.inlineCBW = contentW
	}

	e.collectInline(nodes, &items)
	e.imgMaxW = oldMax
	e.inlineCBW = oldCB

	if len(items) >= two {
		items = squeezeInlineSpaces(items)
	}

	if len(items) >= two {
		items = separateAdjacentCites(items, e)
	}

	return items
}

// layoutInlineFloats lays out inline content into line boxes and emits
// text/image ops. It returns the consumed height and records the first line's
// baseline on the box. When floats is non-nil, each line re-queries exclusion
// at its canvas Y so text widens again after a float ends mid-paragraph.
//
//nolint:cyclop // hot path: per-line wrap against float exclusion zones
func (e *engine) layoutInlineFloats(
	boxNode *box, nodes []*html.Node, contentW, contentX, lineY float64,
	floats *floatState,
) float64 {
	items := e.collectAndPrepareInlineItems(nodes, contentW)
	defer e.releaseInlineItems(items)

	if len(items) == 0 {
		return 0
	}

	leftY := lineY

	idx := 0
	for idx < len(items) {
		lineX, lineW := e.lineBounds(floats, contentX, contentW, leftY)

		if idx == 0 && boxNode != nil && boxNode.style != nil && boxNode.style.TextIndent != 0 {
			indent := e.scalePt(boxNode.style.TextIndent)
			lineX += indent
			lineW -= indent

			if lineW < 0 {
				lineW = 0
			}
		}

		// Short remaining tail beside a float: if it fits as one full-width
		// line under the float, drop there instead of leaving an orphan in
		// the narrow column (e.g. wiki "big time."[71] left of a thumb).
		leftY, lineX, lineW = e.preferClearForTail(items, idx, lineX, lineW, contentX, contentW, leftY, floats)
		// Pack one line under current exclusion width.
		start := idx

		idx, lineX, lineW, leftY = e.packInlineLine(&items, start, lineX, lineW, leftY, contentX, contentW, floats)

		end := idx

		if idx < len(items) && items[idx].forceBreak {
			idx++ // consume br
		}

		if end == start {
			// Single unbreakable item wider than line — still emit it.
			if idx < len(items) && !items[idx].forceBreak {
				idx++
				end = idx
			} else {
				continue
			}
		}

		lastLine := idx >= len(items)
		leftY += e.emitLine(boxNode, items, start, end, lineW, lineX, leftY, lastLine)
	}

	return leftY - lineY
}

const maxPooledInlineItems = 256

// acquireInlineItems returns a reusable temporary item slice for one inline
// formatting context. Nested contexts consume separate entries from the same
// engine-local stack.
const initialInlineItemCapacity = 32

func (e *engine) acquireInlineItems() []inlineItem {
	if n := len(e.inlineItemPool); n > 0 {
		items := e.inlineItemPool[n-1]
		e.inlineItemPool = e.inlineItemPool[:n-1]

		return items[:0]
	}

	return make([]inlineItem, 0, initialInlineItemCapacity)
}

// releaseInlineItems returns a temporary item slice to the engine-local pool.
// Very large contexts are left for GC so one pathological line cannot pin a
// large backing array for the rest of the document.
func (e *engine) releaseInlineItems(items []inlineItem) {
	if cap(items) == 0 || cap(items) > maxPooledInlineItems {
		return
	}

	clear(items)
	e.inlineItemPool = append(e.inlineItemPool, items[:0])
}

// packInlineLine advances idx over the items that fit on one line under the
// current float exclusion, splitting overlong tokens as needed. It returns
// the next index and the updated line geometry (items may be replaced by the
// split, hence the pointer).
func (e *engine) packInlineLine(
	items *[]inlineItem, start int, lineX, lineW, leftY float64,
	contentX, contentW float64, floats *floatState,
) (int, float64, float64, float64) {
	idx := start
	lineAdv := 0.0

	for idx < len(*items) {
		item := &(*items)[idx]
		if item.forceBreak {
			break
		}
		// Split long unbreakable runs (URLs, paths, base64) that would
		// overflow the line. Honors overflow-wrap / word-break; also
		// emergency-breaks when a token is wider than the line so text
		// does not paint past the page edge (print PDF).
		// restMax for subsequent chunks uses contentW (full BFC width) so
		// pre-split fragments can reflow wider after a float ends.
		if parts := e.maybeSplitOverflow(*item, lineW, lineAdv, contentW); len(parts) > 1 {
			// Open space in-place for the split fragments. The former nested
			// append always allocated for the tail because chunkParts returns a
			// full-capacity slice, even when items already had enough room.
			oldLen := len(*items)
			extra := len(parts) - 1

			if oldLen+extra > cap(*items) {
				grown := make([]inlineItem, oldLen+extra)
				copy(grown, *items)
				*items = grown
			} else {
				*items = (*items)[:oldLen+extra]
			}

			copy((*items)[idx+len(parts):], (*items)[idx+1:oldLen])
			copy((*items)[idx:idx+len(parts)], parts)
			item = &(*items)[idx]
		}

		adv := item.marginL + item.w + item.marginR
		// Always wrap to the next line when the next item does not fit.
		// white-space:nowrap must not glue an unbreakable span onto a line
		// that already has content and overflow into a float (wiki .IPA).
		// Exception: never break before attaching punctuation / mid-cite
		// (")[37]" → ")\n[" or "[\n37]" or "saying.[\n7]").
		if lineAdv > 0 && lineAdv+adv > lineW+layoutEpsilon {
			idx, _ = e.glueStickyTail(*items, idx, start, adv)

			break
		}
		// Empty line beside a float too narrow for this item: CSS2.1 §9.5
		// pushes the line box below the float and recomputes width.
		if lineAdv == 0 {
			nY, nX, nW, retry := e.clearFloatBelow(lineX, lineW, leftY, adv, contentX, contentW, floats)
			leftY, lineX, lineW = nY, nX, nW

			if retry {
				continue
			}
		}

		lineAdv += adv
		idx++
	}

	return idx, lineX, lineW, leftY
}

// lineBounds returns the line origin and width under float exclusion at y.
func (e *engine) lineBounds(floats *floatState, contentX, contentW, lineY float64) (float64, float64) {
	if floats == nil {
		return contentX, contentW
	}

	lineX, lineW := floats.exclusion(contentX, contentW, 0, lineY)
	if lineW < 0 {
		lineW = 0
	}

	return lineX, lineW
}

// preferClearForTail drops the current line below active floats when the
// remaining tail fits one full-width line (see preferFloatClearForTail).
// Returns (lineY, lineX, lineW) to match the call site assignment order.
func (e *engine) preferClearForTail(
	items []inlineItem, idx int, lineX, lineW, contentX, contentW, lineY float64,
	floats *floatState,
) (float64, float64, float64) {
	if floats == nil || lineW >= contentW-0.5 {
		return lineY, lineX, lineW
	}

	if next, ok := e.preferFloatClearForTail(items, idx, contentW, lineW, lineY, floats); ok {
		nextX, nextW := e.lineBounds(floats, contentX, contentW, next)

		return next, nextX, nextW
	}

	return lineY, lineX, lineW
}

// maybeSplitOverflow splits an unbreakable text item that would overflow the
// current line; returns the replacement items, or nil when no split applies.
func (e *engine) maybeSplitOverflow(item inlineItem, lineW, lineAdv, contentW float64) []inlineItem {
	if item.img || item.blockBox != nil || item.text == "" {
		return nil
	}

	if item.noSplit {
		return nil
	}

	room := lineW - lineAdv
	if room < 0 {
		room = 0
	}

	restW := contentW
	if restW < lineW {
		restW = lineW
	}

	return e.breakOverflowItem(item, room, lineW, restW, lineAdv == 0)
}

// glueStickyTail advances idx across consecutive no-break items (cite
// clusters, IPA fragments) that may stick to the current line even when they
// overflow slightly; returns the new idx and accumulated line advance.
func (e *engine) glueStickyTail(items []inlineItem, idx, start int, adv float64) (int, float64) {
	if idx <= start || !noBreakBefore(items[idx-1], items[idx]) {
		return idx, adv
	}
	// Keep short sticky tails (cite ")", "[37]", commas) even if
	// they overflow slightly. Do NOT glue multi-em tokens
	// (bare URLs after "(") onto the line — that painted past
	// the page edge on reference lists.
	//
	// Nowrap-to-nowrap chains (wiki Ref cells "[127][128]") must
	// stay on one line even when the cell is a hair narrower
	// than the cluster; stacking at the same X was worse.
	emSize := items[idx].style.FontSize * e.scale
	if emSize < 1 {
		emSize = 10
	}

	if adv > glueLimit(items[idx-1], items[idx], emSize) {
		return idx, adv
	}

	return stickChain(items, idx+1, emSize, adv)
}

// glueLimit returns the max advance that may stick to the current line for
// the item pair: nowrap clusters (multi-cite / IPA fragments) may glue more.
func glueLimit(prev, cur inlineItem, emSize float64) float64 {
	limit := emSize * glueEmSoft

	if isNowrapCluster(prev, cur) {
		limit = emSize * maxGlueEm // multi-cite / IPA fragments
	}

	return limit
}

// stickChain advances idx across consecutive no-break items whose advance
// stays within the pair's glue limit.
func stickChain(items []inlineItem, idx int, emSize, lineAdv float64) (int, float64) {
	for idx < len(items) && !items[idx].forceBreak && noBreakBefore(items[idx-1], items[idx]) {
		acc := items[idx].marginL + items[idx].w + items[idx].marginR
		if acc > glueLimit(items[idx-1], items[idx], emSize) {
			break
		}

		lineAdv += acc
		idx++
	}

	return idx, lineAdv
}

// isNowrapCluster reports that both items are white-space:nowrap runs that
// must stay together on one line.
func isNowrapCluster(a, b inlineItem) bool {
	return a.style.WhiteSpace == cssWhiteSpaceNowrap && b.style.WhiteSpace == cssWhiteSpaceNowrap
}

// clearFloatBelow drops the current (empty) line below active floats when it
// is too narrow for the next item; returns (lineY, lineX, lineW, retry) to
// match the call site assignment order.
func (e *engine) clearFloatBelow(
	lineX, lineW, lineY, adv, contentX, contentW float64,
	floats *floatState,
) (float64, float64, float64, bool) {
	if floats == nil || adv <= lineW || lineW >= contentW-0.5 {
		return lineY, lineX, lineW, false
	}

	next := floats.clearY(lineY)
	if next <= lineY+0.5 {
		return lineY, lineX, lineW, false
	}

	nextX, nextW := e.lineBounds(floats, contentX, contentW, next)
	// Retry this item at the new Y (do not advance i).
	if nextW >= contentW-0.5 || adv <= nextW {
		return next, nextX, nextW, true
	}

	// Still too narrow (e.g. both sides floated) — emit anyway.
	return next, nextX, nextW, false
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
func (e *engine) breakOverflowItem(
	item inlineItem, remainW, fullLineW, restLineW float64,
	aloneOnLine bool,
) []inlineItem {
	if item.img || item.blockBox != nil || item.forceBreak || item.text == "" {
		return nil
	}

	pol := wordBreakOf(*item.style)
	if pol == breakNever {
		return nil
	}

	adv := item.marginL + item.w + item.marginR
	// Nothing to do when the whole item fits on this line.
	if adv <= remainW+0.01 {
		return nil
	}

	if !allowMidTokenBreak(pol, adv, fullLineW+layoutSlack, aloneOnLine) {
		return nil
	}

	firstRoom, restRoom, ok := breakRooms(remainW, fullLineW, restLineW, item.marginL, item.marginR, aloneOnLine)
	if !ok {
		return nil
	}

	chunks := e.breakToken(item.text, *item.style, firstRoom, restRoom)
	if len(chunks) <= 1 {
		return nil
	}

	return e.chunkParts(item, chunks)
}

// allowMidTokenBreak applies the CSS wrapping policy to decide whether the
// token may be split mid-word on the current line instead of wrapping whole.
func allowMidTokenBreak(pol wordBreakPolicy, adv, fullLineW float64, aloneOnLine bool) bool {
	tokenExceedsLine := adv > fullLineW
	// Mid-line: a normal / break-word token that fits a full next line must
	// wrap whole — not mid-break into a tight remainW (captions: "International").
	if pol != breakAll && !tokenExceedsLine {
		return false
	}
	// Emergency path (overflow-wrap:normal): only when alone on the line and
	// the token is wider than that line.
	if pol == breakNormal {
		return aloneOnLine && tokenExceedsLine
	}

	return true
}

// breakRooms computes the usable widths for the first and subsequent chunks
// of a split token, or reports that splitting is not viable.
func breakRooms(
	remainW, fullLineW, restLineW, marginL, marginR float64,
	aloneOnLine bool,
) (float64, float64, bool) {
	// Width available for the first chunk of text (exclude leading margin once).
	firstRoom := remainW - marginL
	if firstRoom < 1 {
		// No room on this line — let the outer packer wrap to the next line
		// unless we are already alone (then force a progressive split).
		if !aloneOnLine {
			return 0, 0, false
		}

		firstRoom = fullLineW - marginL - marginR
	}

	if firstRoom < 1 {
		firstRoom = fullLineW
	}

	if firstRoom < 1 {
		return 0, 0, false
	}

	if restLineW < fullLineW {
		restLineW = fullLineW
	}

	restRoom := restLineW - marginL - marginR
	if restRoom < 1 {
		restRoom = restLineW
	}

	if restRoom < 1 {
		return 0, 0, false
	}

	return firstRoom, restRoom, true
}

// chunkParts builds the replacement items for a split token, zeroing the
// margins on fragments that are not at the token's outer edge.
func (e *engine) chunkParts(item inlineItem, chunks []string) []inlineItem {
	out := make([]inlineItem, 0, len(chunks))

	for idx, chunk := range chunks {
		part := item
		part.text = chunk
		part.w = e.inlineTextWidth(chunk, item.style, item.chrome)

		if idx > 0 {
			part.marginL = 0
		}

		if idx < len(chunks)-1 {
			part.marginR = 0
		}

		out = append(out, part)
	}

	return out
}

// breakToken splits s into chunks that each fit firstMax (first piece) then
// restMax under wordBreakOf(st). Shared by inline overflow packing; soft-mode
// selection lives only here (and softModeOf) so measure and pack cannot drift.
func (e *engine) breakToken(cssSheet string, sty ResolvedStyle, firstMax, restMax float64) []string {
	if cssSheet == "" {
		return nil
	}

	pol := wordBreakOf(sty)
	if pol == breakNever {
		return []string{cssSheet}
	}

	return e.splitTextToWidth(cssSheet, sty, firstMax, restMax, softModeOf(pol))
}

// preferFloatClearForTail reports whether remaining inline content from i
// should drop below active floats so the last line(s) use full content width
// instead of sitting as a short orphan beside the float.
func (e *engine) preferFloatClearForTail(
	items []inlineItem, idx int, contentW, lineW, lineY float64,
	floats *floatState,
) (float64, bool) {
	if floats == nil || idx >= len(items) {
		return lineY, false
	}

	next := floats.clearY(lineY)

	if next-lineY <= halfRatio {
		return lineY, false
	}

	rem, estLH := tailRemaining(items, idx)
	if rem <= 0 {
		return lineY, false
	}

	if estLH < 1 {
		estLH = 12
	}
	// Only when the whole remaining tail fits one full-width line (true
	// short ending), not when a long paragraph still has room to wrap
	// usefully beside the float.
	if rem > contentW+1 {
		return lineY, false
	}
	// Near the float bottom, pull the last full-width-fitting chunk under the
	// float so we do not leave "…destined for the" beside + orphan
	// "big time."[71] under. Allow a slightly larger jump when the tail would
	// need multiple narrow lines but only one full-width line.
	maxGap := estLH * glueEmSoft
	if rem > lineW+0.01 {
		maxGap = estLH * glueEmHard
	}

	if next-lineY <= maxGap {
		return next, true
	}

	return lineY, false
}

// tailRemaining sums the advance of items[i:] up to the next hard break and
// estimates the line height of the tail.
func tailRemaining(items []inlineItem, i int) (float64, float64) {
	rem := 0.0
	estLH := 0.0

	for j := i; j < len(items); j++ {
		item := items[j]
		if item.forceBreak {
			// Hard break ends the tail we consider for a single-line clear.
			break
		}

		rem += item.marginL + item.w + item.marginR

		if estLH <= 0 && item.h > 0 {
			estLH = item.h
		}
	}

	return rem, estLH
}

// separateAdjacentCites inserts a thin space between consecutive citation
// markers ("][") so [90][91][92] are not painted as a cramped cluster. Does
// not touch spaces inside a single marker ([111]). Items are mutated in
// place (only the current item's text/w change), so no copy slice is needed.
func separateAdjacentCites(items []inlineItem, eng *engine) []inlineItem {
	if len(items) < two {
		return items
	}

	for i := 1; i < len(items); i++ {
		cur := &items[i]
		prev := &items[i-1]

		if isCiteBoundary(*prev, *cur) {
			// Prefer a hair space so markers stay visually tight but not
			// colliding; fall back to a normal space if measure fails.
			gap := "\u200a" // hair space

			cur.text = gap + cur.text
			if eng != nil {
				cur.w = eng.inlineTextWidth(cur.text, cur.style, cur.chrome)
			}
		}
	}

	return items
}

// isCiteBoundary reports that prev followed by cur are two adjacent citation
// markers ("][") that would paint as a cramped cluster.
func isCiteBoundary(prev, cur inlineItem) bool {
	if prev.img || cur.img || prev.forceBreak || cur.forceBreak {
		return false
	}

	if prev.blockBox != nil || cur.blockBox != nil {
		return false
	}

	pt := strings.TrimRight(prev.text, " ")
	ct := strings.TrimLeft(cur.text, " ")

	return strings.HasSuffix(pt, "]") && strings.HasPrefix(ct, "[") &&
		!strings.HasSuffix(prev.text, " ") && !strings.HasPrefix(cur.text, " ")
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
func (e *engine) splitTextToWidth(
	text string, style ResolvedStyle, firstMax, restMax float64, mode softBreakMode,
) []string {
	if text == "" {
		return nil
	}

	runes := []rune(text)
	if len(runes) <= 1 {
		return []string{text}
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
		node := e.fittingPrefix(runes, limit, style)
		// Prefer a soft wrap opportunity near the end of the fitting prefix
		// so URLs break after "/" etc. rather than mid-token when possible.
		if mode != softBreakNone && node < len(runes) && node > 1 {
			if soft := lastSoftBreak(runes[:node], mode); soft > 0 {
				node = soft
			}
		}

		out = append(out, string(runes[:node]))
		runes = runes[node:]
	}

	return out
}

// fittingPrefix returns the rune count of the longest prefix of runes that
// fits within limit, always at least 1 so splitting makes progress.
func (e *engine) fittingPrefix(runes []rune, limit float64, style ResolvedStyle) int {
	node := 0
	width := 0.0

	for node < len(runes) {
		// measureRuneFace avoids per-rune string(r) allocations.
		rowW := e.measureRuneFace(runes[node], style)
		if node > 0 && width+rowW > limit+0.01 {
			break
		}
		// Always take at least one rune so we make progress.
		if node == 0 && rowW > limit+0.01 {
			return 1
		}

		width += rowW
		node++
	}

	if node <= 0 {
		return 1
	}

	return node
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

func hidesPaint(style *ResolvedStyle) bool {
	if style == nil {
		return false
	}

	switch style.Visibility {
	case overflowHidden, borderCollapseValue:
		return true
	default:
		return false
	}
}

// emitLine renders items[start:end) as one line and returns its height.
// lastLine is true for the final line of the inline formatting context (used
// so text-align:justify leaves the last line start-aligned).
func (e *engine) emitLine(
	boxNode *box, items []inlineItem, start, end int,
	availW, startX, lineY float64, lastLine bool,
) float64 {
	line := items[start:end]
	if len(line) == 0 {
		return 0
	}

	// trim trailing whitespace of the last run
	e.trimTrailingSpace(line)

	textAlign := floatLeft
	if boxNode != nil && boxNode.style.TextAlign != "" {
		textAlign = boxNode.style.TextAlign
	}

	// Coalesce adjacent same-style text runs into one op so PDF/image paint
	// advances match layout (avoids word-by-word Tj gaps). Skip when
	// justifying — gaps are distributed between word items.
	if textAlign != cssTextAlignJustify {
		line = e.coalesceTextItems(line)
	}

	// line metrics
	lineH, baseline := e.lineMetrics(line, lineY)

	totalW := 0.0
	for i := range line {
		totalW += line[i].marginL + line[i].w + line[i].marginR
	}

	leftX, justifyGap := e.lineOriginAndGap(textAlign, startX, availW, totalW, line, lastLine)

	e.emitLineItems(boxNode, line, leftX, baseline, lineH, lineY, justifyGap)

	if boxNode != nil && boxNode.firstBaseline == 0 {
		boxNode.firstBaseline = baseline
	}

	return lineH
}

// emitLineItems paints each item of a line at the given baseline, flushing
// the accumulated underline run when the styling changes.
func (e *engine) emitLineItems(boxNode *box, line []inlineItem, leftX, baseline, lineH, lineY, justifyGap float64) {
	var und undRun

	for idx := range line {
		item := &line[idx]
		if item.forceBreak || item.style == nil {
			continue
		}

		leftX += item.marginL

		if hidesPaint(item.style) {
			und.flush(e)

			leftX += item.w + item.marginR

			if idx < len(line)-1 && isJustifyGapAfter(*item) {
				leftX += justifyGap
			}

			continue
		}

		switch {
		case item.blockBox != nil:
			leftX = e.emitInlineBlock(
				boxNode, item, leftX, lineY, lineH, baseline, justifyGap, idx < len(line)-1, &und,
			)
		case item.img:
			leftX = e.emitInlineImage(item, leftX, lineY, lineH, baseline, justifyGap, idx < len(line)-1, &und)
		default:
			leftX = e.emitInlineText(item, leftX, baseline, justifyGap, idx < len(line)-1, &und)
		}
	}

	und.flush(e)
}

// trimTrailingSpace drops trailing whitespace from the last run of a line.
// Only ASCII spaces are trimmed; each one contributes exactly its per-rune
// measured width (advance + letter-spacing), so the width is adjusted by
// subtraction instead of re-measuring the whole run.
func (e *engine) trimTrailingSpace(line []inlineItem) {
	if len(line) == 0 || line[len(line)-1].img {
		return
	}

	last := &line[len(line)-1]
	trimmed := strings.TrimRight(last.text, " ")

	if trimmed != last.text {
		spaceCount := len(last.text) - len(trimmed)
		last.text = trimmed
		last.w -= float64(spaceCount) * e.measureRuneFace(' ', *last.style)
	}
}

// lineMetrics returns the height of a line and the Y of its baseline.
//
//nolint:cyclop // line metrics combine inline item classes and chrome
func (e *engine) lineMetrics(line []inlineItem, lineY float64) (float64, float64) {
	maxAscent, maxDescent := 0.0, 0.0

	for i := range line {
		item := &line[i]
		if item.forceBreak || item.style == nil {
			continue
		}

		if item.img || item.blockBox != nil {
			ascent, descent := e.atomicInlineAlign(item)
			if ascent > maxAscent {
				maxAscent = ascent
			}

			if descent > maxDescent {
				maxDescent = descent
			}

			continue
		}

		ascent, descent := e.inlineFontMetrics(item.text, *item.style)
		lh := lineHeightOf(item.style) * e.scale

		extra := (lh - ascent - descent) / two
		itemAscent := ascent + extra
		itemDescent := descent + extra

		if item.chrome {
			itemAscent += e.inlineChromeTop(item.style)
			itemDescent += e.inlineChromeBottom(item.style)
		}

		if itemAscent > maxAscent {
			maxAscent = itemAscent
		}

		if itemDescent > maxDescent {
			maxDescent = itemDescent
		}
	}

	lineH := maxAscent + maxDescent
	if lineH <= 0 {
		lineH = 1
	}

	return lineH, lineY + maxAscent
}

// atomicInlineAlign is the ascent/descent a replaced or inline-block item
// contributes to the line. A length vertical-align raises (positive) or
// lowers (negative) the box relative to the baseline.
func (e *engine) atomicInlineAlign(item *inlineItem) (float64, float64) {
	if item.style != nil {
		switch item.style.VerticalAlign {
		case cssVerticalAlignTop, cssVerticalAlignMiddle, cssVerticalAlignBottom:
			return item.h, 0
		}
	}

	shift := 0.0
	if item.style != nil {
		shift = e.scalePt(item.style.VerticalAlignShift)
	}

	ascent := item.h + shift
	descent := -shift

	if ascent < 0 {
		descent -= ascent
		ascent = 0
	}

	if descent < 0 {
		ascent -= descent
		descent = 0
	}

	return ascent, descent
}

// lineOriginAndGap returns the x where the line content starts and the extra
// inter-word space added by text-align:justify (0 unless justifying).
func (e *engine) lineOriginAndGap(
	textAlign string, originX, availW, totalW float64,
	line []inlineItem, lastLine bool,
) (float64, float64) {
	switch textAlign {
	case floatRight:
		return originX + availW - totalW, 0
	case fxCenter:
		return originX + (availW-totalW)/two, 0
	case cssTextAlignJustify:
		return originX, e.justifyGapOf(line, availW, totalW, lastLine)
	default:
		return originX, 0
	}
}

// justifyGapOf computes the extra space added after each inter-word gap when
// text-align:justify stretches a non-last line.
// CSS justify expands inter-word spaces only — not every inline box
// boundary. Expanding after every item put rivers before commas, cites
// ("word [1]"), and apostrophes ("Roth 's") on wiki print pages.
func (e *engine) justifyGapOf(line []inlineItem, availW, totalW float64, lastLine bool) float64 {
	if lastLine || availW <= totalW || len(line) <= 1 {
		return 0
	}

	gaps := 0

	for i := range len(line) - 1 {
		if isJustifyGapAfter(line[i]) {
			gaps++
		}
	}

	if gaps == 0 {
		return 0
	}

	raw := (availW - totalW) / float64(gaps)

	maxGap := e.justifyMaxGap(line)

	if raw > maxGap*2 {
		return 0
	}

	if raw > maxGap {
		return maxGap // soft-cap rivers without abandoning justify
	}

	return raw
}

// justifyMaxGap returns the largest inter-word gap allowed for the line,
// up to 1em of the largest font size present.
func (e *engine) justifyMaxGap(line []inlineItem) float64 {
	maxGap := 6.0 // pt

	for i := range line {
		if fs := line[i].style.FontSize * e.scale; fs > maxGap {
			maxGap = fs // up to 1em extra between words
		}
	}

	return maxGap
}
