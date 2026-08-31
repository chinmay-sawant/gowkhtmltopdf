package layout

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func (e *engine) collectInline(nodes []*html.Node, out *[]inlineItem) {
	start := len(*out)
	for _, n := range nodes {
		e.collectInlineNode(n, out)
	}
	// Lite RTL: mirror inline order when the containing block is dir:rtl.
	// Full bidi (unicode bidi algorithm, embedding levels) is out of scope
	// for print; this keeps e.g. <div dir="rtl">hello <b>world</b></div>
	// from painting in strict LTR order while staying a single left-to-right
	// layout pass.
	if len(*out)-start > 1 && hasRTLRun(*out, start) {
		reverseInlineRange(*out, start)
	}
}

func hasRTLRun(items []inlineItem, start int) bool {
	for i := start; i < len(items); i++ {
		if items[i].style != nil && items[i].style.Direction == "rtl" {
			return true
		}
		if items[i].forceBreak || items[i].blockBox != nil || items[i].img {
			continue
		}
	}
	return false
}

func reverseInlineRange(items []inlineItem, start int) {
	// Reverse per hard-break segment so <br> stays as line terminator
	// after mirroring. Segments between breaks are reversed independently.
	segStart := start
	for i := start; i <= len(items); i++ {
		isEnd := i == len(items)
		isBreak := !isEnd && items[i].forceBreak
		if isEnd || isBreak {
			// reverse [segStart, i) (exclusive of break)
			for l, r := segStart, i-1; l < r; l, r = l+1, r-1 {
				items[l], items[r] = items[r], items[l]
			}
			if isBreak {
				segStart = i + 1
			}
		}
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
//
//nolint:cyclop // whitespace, writing-mode, and inline decoration are one collection pass
func (e *engine) collectInlineText(node *html.Node, sty ResolvedStyle, out *[]inlineItem) {
	if sty.Display == cssDisplayNone {
		return
	}

	start := len(*out)
	parent := node.Parent
	vertical := parent != nil && parent.Type == html.ElementNode && isVerticalWritingMode(e.styleVal(parent).WritingMode)

	if vertical {
		verticalStyle := sty
		verticalStyle.WhiteSpace = cssWhiteSpaceNowrap
		e.collectWrappedText(node, verticalStyle, out)

		for idx := start; idx < len(*out); idx++ {
			(*out)[idx].style = &verticalStyle
			(*out)[idx].noSplit = true
		}
	} else {
		atomicChrome := e.inlineChromeIsAtomic(node)

		switch {
		case atomicChrome:
			// Keep padded/outlined inline labels together when they fit on
			// the next line. Splitting each word into a separate decorated
			// item produces a sequence of tiny pills and can strand the last
			// word on a new line even though the complete label fits.
			atomicStyle := sty
			atomicStyle.WhiteSpace = cssWhiteSpaceNowrap
			e.collectWrappedText(node, atomicStyle, out)
		case sty.WhiteSpace == cssWhiteSpacePre,
			sty.WhiteSpace == cssWhiteSpacePreWrap,
			sty.WhiteSpace == cssWhiteSpacePreLine:
			e.collectPreservingNewlines(node, sty, out)
		default:
			e.collectWrappedText(node, sty, out)
		}
	}

	if !e.inlineChromeApplies(node) {
		return
	}

	for idx := start; idx < len(*out); idx++ {
		e.enableInlineChrome(&(*out)[idx])
	}
}

func isVerticalWritingMode(mode string) bool {
	return mode == writingModeVerticalRL || mode == writingModeVerticalLR
}

// inlineChromeApplies reports whether text belongs to an inline formatting
// context whose decoration must be painted by the inline text emitter. A
// flex/grid item is laid out as a blockified box even when its authored
// display value is inline; that box already owns its padding, background, and
// border. Painting inline chrome for its text would emit a second rounded
// frame and inflate the item's measured size.
func (e *engine) inlineChromeApplies(node *html.Node) bool {
	parent := node.Parent
	if parent == nil || parent.Type != html.ElementNode {
		return false
	}

	pStyle := e.stylePtr(parent)
	if pStyle.Display != cssDisplayInline || pStyle.Position != "static" || isVerticalWritingMode(pStyle.WritingMode) {
		return false
	}

	container := parent.Parent
	if container == nil || container.Type != html.ElementNode {
		return true
	}

	switch e.stylePtr(container).Display {
	case displayFlex, displayInlineFlex, displayGrid, displayInlineGrid, displaySubgrid:
		return false
	default:
		return true
	}
}

func (e *engine) inlineChromeIsAtomic(node *html.Node) bool {
	if !e.inlineChromeApplies(node) {
		return false
	}

	parent := node.Parent
	if parent == nil {
		return false
	}

	if parent.Name != "mark" {
		return false
	}

	style := e.styleVal(parent)

	return style.PaddingLeft > 0 || style.PaddingRight > 0 || inlineHasBorder(style)
}

// collectPreservingNewlines splits on newlines for pre / pre-wrap / pre-line.
// pre keeps each line as one unbreakable run; pre-wrap preserves spaces and
// wraps; pre-line collapses spaces, preserves newlines, and wraps.
func (e *engine) collectPreservingNewlines(node *html.Node, sty ResolvedStyle, out *[]inlineItem) {
	style := e.stylePtr(node)
	text := node.Text
	collapse := sty.WhiteSpace == cssWhiteSpacePreLine
	wrap := sty.WhiteSpace != cssWhiteSpacePre

	for start := 0; ; {
		end := strings.IndexByte(text[start:], '\n')
		last := end < 0

		var line string
		if last {
			line = text[start:]
		} else {
			line = text[start : start+end]
		}

		e.emitWhiteSpaceLine(line, style, collapse, wrap, out)

		if last {
			return
		}

		*out = append(*out, inlineItem{forceBreak: true}) //nolint:exhaustruct // intentional zero fields
		start += end + 1
	}
}

func (e *engine) emitWhiteSpaceLine(line string, style *ResolvedStyle, collapse, wrap bool, out *[]inlineItem) {
	if collapse {
		line = collapseWS(line)
	}

	if line == "" {
		return
	}

	if !wrap {
		*out = append(*out, e.textItem(line, style))

		return
	}

	if collapse {
		e.emitCollapsedWords(line, style, out)

		return
	}

	e.emitPreservedWrap(line, style, out)
}

func (e *engine) emitCollapsedWords(text string, style *ResolvedStyle, out *[]inlineItem) {
	wordStart := 0

	for wordStart < len(text) {
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

		end := wordEnd
		if wordEnd < len(text) {
			end = wordEnd + 1
		}

		*out = append(*out, e.textItem(text[wordStart:end], style))
		wordStart = end
	}
}

func (e *engine) emitPreservedWrap(line string, style *ResolvedStyle, out *[]inlineItem) {
	idx := 0

	for idx < len(line) {
		if line[idx] == ' ' || line[idx] == '\t' {
			end := idx + 1
			for end < len(line) && (line[end] == ' ' || line[end] == '\t') {
				end++
			}

			*out = append(*out, e.textItem(line[idx:end], style))
			idx = end

			continue
		}

		end := idx + 1
		for end < len(line) && line[end] != ' ' && line[end] != '\t' {
			end++
		}

		*out = append(*out, e.textItem(line[idx:end], style))
		idx = end
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
			if len(*out) == 0 || !(*out)[len(*out)-1].img || (*out)[len(*out)-1].blockBox != nil {
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
		e.preserveTrailingGap(node, out)
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
func (e *engine) preserveTrailingGap(node *html.Node, out *[]inlineItem) {
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
	item.w = e.inlineTextWidth(item.text, e.stylePtr(node), item.chrome)
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

	if sty.Display == cssDisplayInlineBlock || sty.Display == displayInlineFlex ||
		sty.Display == displayInlineGrid {
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
	// Measure only: emitInlineImage places the bitmap (and thumb separator).
	imgBox := e.buildImage(node, sty, 0, 0, false)
	*out = append(*out, inlineItem{ //nolint:exhaustruct // intentional zero fields
		img:      true,
		thumbImg: e.thumbImageInsideFigure(node),
		imgRef:   imgBox.img,
		alt:      node.Attribute("alt"),
		w:        imgBox.w,
		h:        imgBox.height,
		style:    e.stylePtr(node),
		marginL:  e.scalePt(sty.MarginLeft),
		marginR:  e.scalePt(sty.MarginRight),
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

	if src := e.pseudoContentURL(node, pseudoBefore); src != "" {
		if ref := e.resolveImage(src); ref != nil && ref.data != nil {
			pstyle := e.pseudoStyle(node, pseudoBefore, sty)
			imgW := e.scalePt(pxToPt(float64(ref.w)))
			imgH := e.scalePt(pxToPt(float64(ref.h)))
			if imgW <= 0 {
				imgW = e.scalePt(pstyle.FontSize)
			}
			if imgH <= 0 {
				imgH = e.scalePt(pstyle.FontSize)
			}
			*out = append(*out, inlineItem{img: true, imgRef: ref, w: imgW, h: imgH, style: pstyle}) //nolint:exhaustruct // pseudo url image
		}
	} else if txt := e.pseudoContent(node, pseudoBefore); txt != "" {
		item := e.textItem(txt, e.pseudoStyle(node, pseudoBefore, sty))
		e.enableInlineChrome(&item)
		*out = append(*out, item)
	}

	for _, c := range node.Children {
		e.collectInlineNode(c, out)
	}

	if src := e.pseudoContentURL(node, pseudoAfter); src != "" {
		if ref := e.resolveImage(src); ref != nil && ref.data != nil {
			pstyle := e.pseudoStyle(node, pseudoAfter, sty)
			imgW := e.scalePt(pxToPt(float64(ref.w)))
			imgH := e.scalePt(pxToPt(float64(ref.h)))
			if imgW <= 0 {
				imgW = e.scalePt(pstyle.FontSize)
			}
			if imgH <= 0 {
				imgH = e.scalePt(pstyle.FontSize)
			}
			*out = append(*out, inlineItem{img: true, imgRef: ref, w: imgW, h: imgH, style: pstyle}) //nolint:exhaustruct // pseudo url image
		}
	} else if txt := e.pseudoContent(node, pseudoAfter); txt != "" {
		item := e.textItem(txt, e.pseudoStyle(node, pseudoAfter, sty))
		e.enableInlineChrome(&item)
		*out = append(*out, item)
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
			return cbW * sty.WidthPercent / 100
		}

		if e.opts.Width > 0 {
			return e.opts.Width * sty.WidthPercent / 100
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

	// measureCellContent already returns the max-content border-box width,
	// including horizontal padding and borders. Add only the outer margins;
	// adding the chrome again makes inline-block pills grow by a second set of
	// padding/border widths and leaves misleading empty space on the right.
	intr := e.measureCellContent(nodeN, sty) +
		e.scalePt(sty.MarginLeft) + e.scalePt(sty.MarginRight)

	if intr < 1 {
		intr = 1
	}

	return intr
}

// availWForInline is a generous width for block-in-inline measurement.
func availWForInline() float64 { return 1 << 30 }

func (e *engine) textItem(text string, style *ResolvedStyle) inlineItem {
	textWidth := e.measureTextFace(transformInlineText(text, style.TextTransform), style)
	lineHeight := lineHeightOf(style) * e.scale

	if isVerticalWritingMode(style.WritingMode) {
		// In a vertical writing context the text advance runs along the
		// block axis. The inline line still occupies only one glyph-width
		// column; using the full measured string width here makes centered
		// labels shift left by half their length.
		textWidth = lineHeight
	}

	return inlineItem{ //nolint:exhaustruct // intentional zero fields
		text: text, style: style, w: textWidth, h: lineHeight,
	}
}

func (e *engine) enableInlineChrome(item *inlineItem) {
	if item == nil || item.style == nil || item.forceBreak || item.chrome {
		return
	}

	item.chrome = true
	item.w += e.inlineChromeLeft(item.style) + e.inlineChromeRight(item.style)
	item.h += e.inlineChromeTop(item.style) + e.inlineChromeBottom(item.style)
}

func (e *engine) inlineTextWidth(text string, st *ResolvedStyle, chrome bool) float64 {
	w := e.measureTextFace(text, st)
	if chrome {
		w += e.inlineChromeLeft(st) + e.inlineChromeRight(st)
	}

	return w
}

func (e *engine) inlineChromeLeft(st *ResolvedStyle) float64 {
	return e.scalePt(st.PaddingLeft) + e.scalePt(st.BorderLeft.Width)
}

func (e *engine) inlineChromeRight(st *ResolvedStyle) float64 {
	return e.scalePt(st.PaddingRight) + e.scalePt(st.BorderRight.Width)
}

func (e *engine) inlineChromeTop(st *ResolvedStyle) float64 {
	return e.scalePt(st.PaddingTop) + e.scalePt(st.BorderTop.Width)
}

func (e *engine) inlineChromeBottom(st *ResolvedStyle) float64 {
	return e.scalePt(st.PaddingBottom) + e.scalePt(st.BorderBottom.Width)
}

// TransformInlineText applies CSS text-transform (uppercase, lowercase, capitalize).
func TransformInlineText(text, transform string) string {
	return transformInlineText(text, transform)
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
	if len(items) < 2 {
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
func (e *engine) coalesceTextItems(line []inlineItem) []inlineItem {
	if len(line) < 2 {
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
			cur.chrome == line[writeIdx-1].chrome &&
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
			if prev.chrome {
				// Each collected word includes its own inline padding/border in
				// w. A coalesced run paints one shared chrome box, so remove the
				// duplicate chrome contributed by the newly merged item.
				prev.w -= e.inlineChromeLeft(prev.style) + e.inlineChromeRight(prev.style)
			}

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

func sameInlineStyle(acc, boxN *ResolvedStyle) bool { //nolint:cyclop
	if acc == nil || boxN == nil {
		return acc == boxN
	}

	return acc.FontSize == boxN.FontSize &&
		acc.FontWeight == boxN.FontWeight &&
		acc.FontItalic == boxN.FontItalic &&
		acc.famHash == boxN.famHash &&
		acc.LineHeight == boxN.LineHeight &&
		acc.TextTransform == boxN.TextTransform &&
		acc.LetterSpacing == boxN.LetterSpacing &&
		acc.WordSpacing == boxN.WordSpacing &&
		acc.Visibility == boxN.Visibility &&
		acc.Color == boxN.Color &&
		acc.BGColor == boxN.BGColor &&
		acc.PaddingTop == boxN.PaddingTop && acc.PaddingRight == boxN.PaddingRight &&
		acc.PaddingBottom == boxN.PaddingBottom && acc.PaddingLeft == boxN.PaddingLeft &&
		acc.BorderTop == boxN.BorderTop && acc.BorderRight == boxN.BorderRight &&
		acc.BorderBottom == boxN.BorderBottom && acc.BorderLeft == boxN.BorderLeft &&
		acc.TextDecoration == boxN.TextDecoration &&
		acc.WhiteSpace == boxN.WhiteSpace
}

func lineHeightOf(st *ResolvedStyle) float64 {
	if st.LineHeight > 0 {
		return st.LineHeight
	}

	return 1.2 * st.FontSize
}

func borderPaint(side border) float64 {
	if side.PaintWidth > 0 {
		return side.PaintWidth
	}

	return side.Width
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
