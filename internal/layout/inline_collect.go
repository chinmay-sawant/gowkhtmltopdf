package layout

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"gowkhtmltopdf/internal/html"
)

func (e *engine) collectInline(nodes []*html.Node, out *[]inlineItem) {
	for _, n := range nodes {
		e.collectInlineNode(n, out)
	}
}

func (e *engine) collectInlineNode(node *html.Node, out *[]inlineItem) {
	sty := e.styleVal(node)

	switch node.Type {
	case html.TextNode:
		e.collectInlineText(node, sty, out)
	case html.ElementNode:
		e.collectInlineElement(node, sty, out)
	case html.CommentNode, html.DoctypeNode:
		return
	}
}

// collectInlineText flattens one text node under its white-space mode.
func (e *engine) collectInlineText(node *html.Node, sty ResolvedStyle, out *[]inlineItem) {
	if sty.Display == cssDisplayNone {
		return
	}

	switch sty.WhiteSpace {
	case cssWhiteSpacePre:
		e.collectPreText(node, sty, out)
	default:
		e.collectWrappedText(node, sty, out)
	}
}

// collectPreText splits a white-space:pre node on newlines.
func (e *engine) collectPreText(node *html.Node, _ ResolvedStyle, out *[]inlineItem) {
	style := e.stylePtr(node)
	text := node.Text

	for start := 0; ; {
		end := strings.IndexByte(text[start:], '\n')
		if end < 0 {
			p := text[start:]
			if p != "" {
				*out = append(*out, e.textItem(p, style))
			}

			return
		}

		end += start

		p := text[start:end]
		if p != "" {
			*out = append(*out, e.textItem(p, style))
		}

		*out = append(*out, inlineItem{forceBreak: true}) //nolint:exhaustruct // intentional zero fields
		start = end + 1
	}
}

// collectWrappedText flattens a normal white-space text node into word items.
//
//nolint:cyclop,funlen // hot path: word-scan with whitespace/nowrap edge cases
func (e *engine) collectWrappedText(node *html.Node, sty ResolvedStyle, out *[]inlineItem) {
	// Whitespace-only text nodes still separate adjacent inlines
	// (wiki "Cuba"+" "+"Spain" / pretty-printed newlines between
	// anchors). collapseWS would drop them. Skip only after a
	// replaced element so `<img>\n<span margin-left>` does not add
	// a space on top of the margin (TestLogoTitleGap).
	if !hasNonHTMLSpace(node.Text) {
		if node.Text != "" {
			if len(*out) == 0 || !(*out)[len(*out)-1].img {
				*out = append(*out, e.textItem(" ", e.stylePtr(node)))
			}
		}

		return
	}

	text := collapseWS(node.Text)
	if text == "" {
		return
	}

	nodeStyle := e.stylePtr(node)
	// white-space:nowrap — keep the run unbreakable (wiki .reference
	// cite markers in narrow table columns).
	if sty.WhiteSpace == cssWhiteSpaceNowrap {
		*out = append(*out, e.textItem(text, nodeStyle))

		return
	}

	// Walk collapsed words without strings.Fields / word+" " copies.
	// Substrings of `text` share the underlying bytes; inter-word spaces
	// stay attached to the preceding word ("hello ", "world") so paint
	// and justify see the same trailing-space convention as before.
	//
	// collapseWS strips a leading space that still separates this node
	// from the previous inline ("</a> in" → "Reeves"+"in"). Re-introduce
	// one when the source began with whitespace and the prior item does
	// not already end with a space. Do not insert a space before attaching
	// punctuation (", . ) ]") — pretty-printed "</a>\n," must stay "Award,".
	needLead := false

	if len(node.Text) > 0 && isWSSpaceByte(node.Text[0]) && len(*out) > 0 {
		prev := &(*out)[len(*out)-1]
		if !prev.forceBreak && !strings.HasSuffix(prev.text, " ") {
			needLead = true
		}
	}

	startOut := len(*out)
	wordStart := 0

	for wordStart < len(text) {
		// Skip any residual spaces (collapseWS normally leaves single
		// separators only between words).
		for wordStart < len(text) && text[wordStart] == ' ' {
			wordStart++
		}

		if wordStart >= len(text) {
			break
		}

		wordEnd := wordStart
		for wordEnd < len(text) && text[wordEnd] != ' ' {
			wordEnd++
		}

		// Include one trailing space when another word follows.
		end := wordEnd
		if wordEnd < len(text) {
			end = wordEnd + 1
		}

		word := text[wordStart:end]
		if startOut == len(*out) && needLead && !isAttachPunct(word) {
			word = " " + word
		}

		*out = append(*out, e.textItem(word, nodeStyle))
		wordStart = end
	}
	// Preserve a trailing word-separator when the source text node
	// ended with whitespace (so "foo <b>bar</b>" keeps the gap).
	if len(*out) > startOut {
		e.preserveTrailingGap(node, sty, out)
	}
}

// isWSSpaceByte reports that b is one of the CSS white-space characters.
func isWSSpaceByte(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\f':
		return true
	}

	return false
}

// preserveTrailingGap keeps a trailing word separator when the source text
// node ended with whitespace.
func (e *engine) preserveTrailingGap(node *html.Node, sty ResolvedStyle, out *[]inlineItem) {
	if len(*out) == 0 || len(node.Text) == 0 {
		return
	}

	last := node.Text[len(node.Text)-1]
	if !isWSSpaceByte(last) {
		return
	}

	item := &(*out)[len(*out)-1]
	if strings.HasSuffix(item.text, " ") {
		return
	}

	item.text += " "
	item.w = e.measureTextFace(item.text, sty)
}

// collectInlineElement flattens one element node into inline items.
func (e *engine) collectInlineElement(node *html.Node, sty ResolvedStyle, out *[]inlineItem) {
	if sty.Display == cssDisplayNone {
		return
	}

	if node.Name == cssTagBR {
		*out = append(*out, inlineItem{forceBreak: true}) //nolint:exhaustruct // intentional zero fields

		return
	}

	if node.Name == cssTagImg {
		e.collectImageItem(node, sty, out)

		return
	}

	if sty.Display == cssDisplayInlineBlock {
		e.collectInlineBlockItem(node, sty, out)

		return
	}

	if sty.Display == cssDisplayInline {
		e.collectInlineSpan(node, sty, out)

		return
	}
	// block-level element inside inline context: lay out at a throwaway
	// offset, then shift its ops into the line when placed.
	opStart := len(e.ops)
	cblock := e.build(node, availWForInline(), 0, 0)
	opEnd := len(e.ops)

	if cblock != nil {
		*out = append(*out, inlineItem{ //nolint:exhaustruct // intentional zero fields
			img: true, w: cblock.w, h: cblock.height, style: e.stylePtr(node),
			blockBox: cblock, opStart: opStart, opEnd: opEnd,
		})
	}
}

// collectImageItem flattens an <img> element into one inline item.
func (e *engine) collectImageItem(node *html.Node, sty ResolvedStyle, out *[]inlineItem) {
	ib := e.buildImage(node, sty, 0, 0)
	*out = append(*out, inlineItem{ //nolint:exhaustruct // intentional zero fields
		img: true, imgRef: ib.img, w: ib.w, h: ib.height, style: e.stylePtr(node),
		marginL: e.scalePt(sty.MarginLeft), marginR: e.scalePt(sty.MarginRight),
	})
}

// collectInlineBlockItem lays out a display:inline-block element at its
// containing-block width and flattens it into one inline item.
func (e *engine) collectInlineBlockItem(node *html.Node, sty ResolvedStyle, out *[]inlineItem) {
	avail := e.inlineBlockAvail(node, sty, e.inlineCBW)
	opStart := len(e.ops)
	cblock := e.build(node, avail, 0, 0)
	opEnd := len(e.ops)

	if cblock != nil {
		*out = append(*out, inlineItem{ //nolint:exhaustruct // intentional zero fields
			img: true, w: cblock.w, h: cblock.height, style: e.stylePtr(node),
			blockBox: cblock, opStart: opStart, opEnd: opEnd,
			marginL: e.scalePt(sty.MarginLeft), marginR: e.scalePt(sty.MarginRight),
		})
	}
}

// collectInlineSpan flattens a display:inline element (e.g. <a>), applying
// hrefs, pseudo-content and horizontal margins to the generated items.
func (e *engine) collectInlineSpan(node *html.Node, sty ResolvedStyle, out *[]inlineItem) {
	href := ""

	if node.Name == cssTagA {
		h := node.Attribute("href")
		if isExternalHref(h) || isInternalHref(h) {
			href = h
		}
	}

	before := len(*out)

	if txt := e.pseudoContent(node, "before"); txt != "" {
		*out = append(*out, e.textItem(txt, e.stylePtr(node)))
	}

	for _, c := range node.Children {
		e.collectInlineNode(c, out)
	}

	if txt := e.pseudoContent(node, "after"); txt != "" {
		*out = append(*out, e.textItem(txt, e.stylePtr(node)))
	}
	// Horizontal margins on inline elements (e.g. .co { margin-left: 10px }
	// after a logo) apply to the first/last generated items.
	if before < len(*out) {
		(*out)[before].marginL += e.scalePt(sty.MarginLeft)
		(*out)[len(*out)-1].marginR += e.scalePt(sty.MarginRight)
	}

	if href != "" {
		for i := before; i < len(*out); i++ {
			(*out)[i].href = href
		}
	}
}

// inlineBlockAvail returns the containing-block width used to lay out an
// inline-block: specified width when present, otherwise shrink-to-fit capped
// at a generous max so auto-width badges size to their content.
func (e *engine) inlineBlockAvail(nodeN *html.Node, sty ResolvedStyle, cbW float64) float64 {
	if sty.WidthPercent >= 0 {
		// Prefer the inline formatting-context width; fall back to viewport.
		if cbW > 0 {
			return cbW * sty.WidthPercent / cssPercent
		}

		if e.opts.Width > 0 {
			return e.opts.Width * sty.WidthPercent / cssPercent
		}
	}

	if sty.Width >= 0 {
		// buildBlock applies box-sizing to the specified width; pass enough
		// avail that auto-fill does not stretch a definite-width box.
		width := e.scalePt(sty.Width)
		if sty.BoxSizing != borderBox {
			width += e.scalePt(sty.PaddingLeft) + e.scalePt(sty.PaddingRight) +
				e.scalePt(sty.BorderLeft.Width) + e.scalePt(sty.BorderRight.Width)
		}

		return width + e.scalePt(sty.MarginLeft) + e.scalePt(sty.MarginRight)
	}

	if isSizeContainer(sty) {
		// Size containment: shrink-to-fit as-if-empty.
		intr := e.scalePt(sty.PaddingLeft) + e.scalePt(sty.PaddingRight) +
			e.scalePt(sty.BorderLeft.Width) + e.scalePt(sty.BorderRight.Width) +
			e.scalePt(sty.MarginLeft) + e.scalePt(sty.MarginRight)
		if intr < 1 {
			intr = 1
		}

		return intr
	}

	intr := e.measureCellContent(nodeN, sty)
	intr += e.scalePt(sty.PaddingLeft) + e.scalePt(sty.PaddingRight) +
		e.scalePt(sty.BorderLeft.Width) + e.scalePt(sty.BorderRight.Width) +
		e.scalePt(sty.MarginLeft) + e.scalePt(sty.MarginRight)

	if intr < 1 {
		intr = 1
	}

	return intr
}

// availWForInline is a generous width for block-in-inline measurement.
func availWForInline() float64 { return 1 << maxIntShift }

func (e *engine) textItem(text string, st *ResolvedStyle) inlineItem {
	w := e.measureTextFace(text, *st)

	return inlineItem{ //nolint:exhaustruct // intentional zero fields
		text: text, style: st, w: w, h: lineHeightOf(st) * e.scale,
	}
}

func transformInlineText(text, transform string) string {
	switch transform {
	case textTransformUppercase:
		return strings.ToUpper(text)
	case textTransformLowercase:
		return strings.ToLower(text)
	case textTransformCapitalize:
		return capitalizeInlineText(text)
	default:
		return text
	}
}

//nolint:wsl,varnamelen // small Unicode word-start transform helper
func capitalizeInlineText(text string) string {
	var out strings.Builder
	out.Grow(len(text))
	upperNext := true

	for _, r := range text {
		if unicode.IsLetter(r) {
			if upperNext {
				out.WriteString(strings.ToUpper(string(r)))
			} else {
				out.WriteRune(r)
			}
			upperNext = false

			continue
		}

		out.WriteRune(r)
		upperNext = unicode.IsSpace(r)
	}

	return out.String()
}

// squeezeInlineSpaces drops artificial space items that sit immediately before
// attaching punctuation (pretty-printed "</a>\n," → "Award ,") or that are
// redundant after a trailing space already on the previous item. Survivors
// are compacted in place; the surviving prefix is returned.
func squeezeInlineSpaces(items []inlineItem) []inlineItem {
	if len(items) < two {
		return items
	}

	writeIdx := 0

	for i := range items {
		if dropSpaceItem(items, i, items[:writeIdx]) {
			continue // drop space before "," / ")" / "]" …
		}

		items[writeIdx] = items[i]
		writeIdx++
	}

	return items[:writeIdx]
}

// dropSpaceItem reports whether the pure-space item at items[i] should be
// dropped: before attaching punctuation or citation brackets, or after a
// trailing space already on the previous item.
func dropSpaceItem(items []inlineItem, idx int, out []inlineItem) bool {
	item := items[idx]
	if item.img || item.forceBreak || item.blockBox != nil {
		return false
	}

	if strings.TrimSpace(item.text) != "" {
		return false
	}
	// Pure space item.
	if dropBeforeNext(items, idx) {
		return true
	}

	if len(out) == 0 {
		return false
	}

	return strings.HasSuffix(out[len(out)-1].text, " ") // redundant double space
}

// dropBeforeNext reports that the space item at items[i] is followed by
// attaching punctuation or a citation bracket that should not be spaced.
func dropBeforeNext(items []inlineItem, i int) bool {
	if i+1 >= len(items) {
		return false
	}

	next := items[i+1]
	if !next.img && !next.forceBreak && isAttachPunct(next.text) {
		return true
	}

	return strings.HasPrefix(strings.TrimSpace(next.text), "[")
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
func isAttachPunct(cssSheet string) bool {
	cssSheet = strings.TrimSpace(cssSheet)
	if cssSheet == "" {
		return false
	}

	r, _ := utf8.DecodeRuneInString(cssSheet)
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

	if isReplacedContent(prev) || isReplacedContent(cur) {
		return false
	}
	// Keep nowrap runs together (wiki .reference / .IPA pieces).
	if isNowrapCluster(prev, cur) {
		return true
	}

	count := strings.TrimSpace(cur.text)
	if count == "" {
		return false
	}
	// Do not start a line with closing/attach punctuation.
	if isAttachPunct(count) {
		return true
	}
	// Digits that continue a cite: "[" then "37" then "]".
	if digitCiteGlue(prev, cur) {
		return true
	}
	// Do not break after opening brackets / quotes.
	return endsWithOpenBracket(strings.TrimRight(prev.text, " "))
}

// digitCiteGlue reports that cur is a digit run continuing a citation begun
// on prev ("[" then "37" then "]").
func digitCiteGlue(prev, cur inlineItem) bool {
	if !isAllDigits(strings.TrimSpace(cur.text)) {
		return false
	}

	pt := strings.TrimSpace(prev.text)

	return strings.HasSuffix(pt, "[") || isAllDigits(pt)
}

// isReplacedContent reports that the item is a replaced or atomic inline
// (image or laid-out inline-block) rather than text.
func isReplacedContent(it inlineItem) bool {
	return it.img || it.blockBox != nil
}

// endsWithOpenBracket reports that s ends with an opening bracket or quote.
func endsWithOpenBracket(s string) bool {
	if s == "" {
		return false
	}

	r, _ := utf8.DecodeLastRuneInString(s)
	switch r {
	case '[', '(', '{', '"', '\'', '\u201c' /* “ */, '\u2018': /* ‘ */
		return true
	}

	return false
}

func isAllDigits(cssSheet string) bool {
	if cssSheet == "" {
		return false
	}

	for i := range len(cssSheet) {
		if cssSheet[i] < '0' || cssSheet[i] > '9' {
			return false
		}
	}

	return true
}

// isAllWhitespace reports that s consists only of Unicode whitespace — the
// same emptiness test as strings.TrimSpace(s) == "" without the TrimSpace
// string header allocation.
func isAllWhitespace(s string) bool {
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return false
		}
	}

	return true
}

// coalesceTextItems merges consecutive non-image text runs that share style
// and href so one text op paints the whole phrase. Each merged string is
// built once with a Builder (no per-merge string churn); items are compacted
// in place, returning the surviving prefix.
//
//nolint:cyclop // per-item merge decision: image/break/href/style compare plus in-place compaction
func coalesceTextItems(line []inlineItem) []inlineItem {
	if len(line) < two {
		return line
	}

	var builder strings.Builder

	writeIdx := 0
	merged := false

	for i := range line {
		cur := line[i]
		mergeable := writeIdx > 0 &&
			!cur.img && !line[writeIdx-1].img && !cur.forceBreak &&
			cur.href == line[writeIdx-1].href &&
			sameInlineStyle(line[writeIdx-1].style, cur.style)

		if mergeable {
			if !merged {
				merged = true

				builder.WriteString(line[writeIdx-1].text)
			}

			builder.WriteString(cur.text)

			prev := &line[writeIdx-1]
			// first item keeps marginL; last item's marginR wins
			prev.w += cur.w
			prev.marginR = cur.marginR

			continue
		}

		if merged {
			line[writeIdx-1].text = builder.String()
			builder.Reset()

			merged = false
		}

		line[writeIdx] = cur
		writeIdx++
	}

	if merged {
		line[writeIdx-1].text = builder.String()
	}

	return line[:writeIdx]
}

func sameInlineStyle(acc, boxN *ResolvedStyle) bool {
	if acc == nil || boxN == nil {
		return acc == boxN
	}

	return acc.FontSize == boxN.FontSize &&
		acc.FontWeight == boxN.FontWeight &&
		acc.FontItalic == boxN.FontItalic &&
		acc.TextTransform == boxN.TextTransform &&
		acc.LetterSpacing == boxN.LetterSpacing &&
		acc.Color == boxN.Color &&
		acc.TextDecoration == boxN.TextDecoration &&
		acc.WhiteSpace == boxN.WhiteSpace
}

func lineHeightOf(st *ResolvedStyle) float64 {
	if st.LineHeight > 0 {
		return st.LineHeight
	}

	return defaultLineHeightRatio * st.FontSize
}

func collapseWS(src string) string { //nolint:cyclop // hot path: byte-wise whitespace collapsing
	// Fast path: already a single-space-collapsed ASCII-friendly run with
	// no tab/newline/formfeed and no double spaces. Skip Builder entirely.
	if collapseWSNoop(src) {
		return strings.TrimRight(src, " ")
	}

	var boxNode strings.Builder

	boxNode.Grow(len(src))

	prevSpace := true

	for pos := 0; pos < len(src); {
		curByte := src[pos]
		// ASCII whitespace (HTML space set).
		if curByte == ' ' || curByte == '\t' || curByte == '\n' || curByte == '\r' || curByte == '\f' {
			if !prevSpace {
				boxNode.WriteByte(' ')

				prevSpace = true
			}

			pos++

			continue
		}
		// ASCII ink: WriteByte avoids utf8.AppendRune per character.
		if curByte < nonASCIIStart {
			boxNode.WriteByte(curByte)

			prevSpace = false
			pos++

			continue
		}
		// Multi-byte UTF-8: copy the full rune bytes without WriteRune→AppendRune.
		_, size := utf8.DecodeRuneInString(src[pos:])
		if size < 1 {
			size = 1
		}

		boxNode.WriteString(src[pos : pos+size])

		prevSpace = false
		pos += size
	}

	return strings.TrimRight(boxNode.String(), " ")
}

// collapseWSNoop reports that s needs no whitespace collapsing (only possible
// trailing spaces remain, handled by TrimRight).
func collapseWSNoop(s string) bool {
	prevSpace := false

	for i := range len(s) {
		c := s[i]
		if c == '\t' || c == '\n' || c == '\r' || c == '\f' {
			return false
		}

		if c == ' ' {
			if prevSpace {
				return false
			}

			prevSpace = true

			continue
		}

		prevSpace = false
	}

	return true
}
