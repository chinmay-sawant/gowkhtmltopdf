package svg

import (
	"bytes"
	"sort"

	"github.com/tdewolff/canvas"
)

// tdewolff/canvas is the sole rasterizer, but it parses SVG in a single pass
// with exact-case names. Two inputs produced by this engine break that:
//
//   - The HTML pipeline lowercases element/attribute names, so an inline
//     <svg> reaches canvas as <lineargradient> with viewbox instead of
//     viewBox.
//   - A document may define paint servers after the shapes that reference
//     them (Programiz's sp_logo.svg does). Canvas resolves defs as it parses,
//     so a forward-referenced fill="url(#a)" keeps canvas's temporary black
//     fill.
//
// Canvas also panics when a font-family list matches no installed family
// (Roboto,arial on a host with neither); Rasterize's recover turns that into
// a lost image. CSS would fall back through the list, canvas leaves that to
// the caller.
//
// prepareCanvasInput rewrites the bytes so canvas can resolve all of it:
// same-document <use href="#id"> is inlined (external sprites need a fetch
// via ResolveUseReferences before Rasterize), gradient element names are
// restored to SVG case, defs and loose gradients are hoisted ahead of the
// shapes, the root viewbox is normalized, unresolvable font-family lists gain
// a generic fallback, and <text> elements with a bold or italic style become
// outline paths (canvas's parser ignores font-style and font-weight). Each
// rewrite is a no-op on input that does not need it.
func prepareCanvasInput(data []byte) []byte {
	// Same-document fragments only; nil fetch leaves external <use> alone.
	if resolved, err := ResolveUseReferences(data, nil); err == nil {
		data = resolved
	}

	data = normalizePaintServerCase(data)
	data = hoistPaintServers(data)
	data = normalizeRootViewBox(data)
	data = withFontFallbacks(data)
	data = outlineStyledText(data)

	return data
}

const (
	svgTagSVG            = "svg"
	svgTagDefs           = "defs"
	svgTagLinearGradient = "linearGradient"
	svgTagRadialGradient = "radialGradient"

	svgAttrViewBox     = "viewBox"
	lowerViewBoxName   = "viewbox"
	sansSerifFallback  = ",sans-serif"
	fontFamilyAttrName = "font-family"
)

// byteSpan is a half-open byte range [start, end).
type byteSpan struct {
	start int
	end   int
}

// noSpan reports a not-found span alongside ok=false. It sets both fields so
// exhaustruct sees a complete literal; callers must not read the value.
func noSpan() byteSpan {
	return byteSpan{start: 0, end: 0}
}

func isSVGSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\r' || c == '\n'
}

func isTagNameBoundary(c byte) bool {
	return isSVGSpace(c) || c == '/' || c == '>'
}

// asciiIndexFold returns the index of the first ASCII case-insensitive
// occurrence of needle in data, or -1.
func asciiIndexFold(data []byte, needle string) int {
	needleBytes := []byte(needle)

	for idx := 0; idx+len(needleBytes) <= len(data); idx++ {
		if bytes.EqualFold(data[idx:idx+len(needleBytes)], needleBytes) {
			return idx
		}
	}

	return -1
}

// tagEnd returns the index just past the '>' that closes the start tag at
// start, honoring quoted attribute values. It returns -1 when unterminated.
func tagEnd(data []byte, start int) int {
	for pos := start + 1; pos < len(data); pos++ {
		switch data[pos] {
		case '"', '\'':
			quote := data[pos]
			pos++

			for pos < len(data) && data[pos] != quote {
				pos++
			}

			if pos >= len(data) {
				return -1
			}
		case '>':
			return pos + 1
		}
	}

	return -1
}

// elementNameEnd returns the end of the element name that begins at start.
func elementNameEnd(data []byte, start int) int {
	pos := start
	for pos < len(data) && !isTagNameBoundary(data[pos]) {
		pos++
	}

	return pos
}

// markupEnd advances past the comment, CDATA section, or processing
// instruction that begins at start. It reports whether such a construct was
// recognized and closed.
func markupEnd(data []byte, start int) (int, bool) {
	constructs := []struct {
		open  string
		close string
	}{
		{"<!--", "-->"},
		{"<![CDATA[", "]]>"},
		{"<?", "?>"},
	}

	for _, construct := range constructs {
		if !bytes.HasPrefix(data[start:], []byte(construct.open)) {
			continue
		}

		end := bytes.Index(data[start:], []byte(construct.close))
		if end < 0 {
			return start, false
		}

		return start + end + len(construct.close), true
	}

	return start, false
}

// elementSpan returns the byte range of the element whose start tag begins at
// start, including nested same-name elements. A self-closing tag ends at its
// own '>'.
//
//nolint:cyclop // single-pass scanner: one branch per markup form it must tolerate
func elementSpan(data []byte, start int, name string) (byteSpan, bool) {
	end := tagEnd(data, start)
	if end < 0 {
		return noSpan(), false
	}

	if data[end-2] == '/' {
		return byteSpan{start, end}, true
	}

	depth := 1

	for idx := end; idx < len(data); {
		if data[idx] != '<' {
			idx++

			continue
		}

		if next, ok := markupEnd(data, idx); ok {
			idx = next

			continue
		}

		nameStart := idx + 1
		closing := false

		if nameStart < len(data) && data[nameStart] == '/' {
			nameStart++
			closing = true
		}

		nameEnd := elementNameEnd(data, nameStart)
		sameName := bytes.EqualFold(data[nameStart:nameEnd], []byte(name))

		tagClose := tagEnd(data, idx)
		if tagClose < 0 {
			return noSpan(), false
		}

		switch {
		case closing && sameName:
			depth--
			if depth == 0 {
				return byteSpan{start, tagClose}, true
			}
		case !closing && sameName && data[tagClose-2] != '/':
			depth++
		}

		idx = tagClose
	}

	return noSpan(), false
}

// rootSVGElement returns the span of the root <svg> start tag.
func rootSVGElement(data []byte) (byteSpan, bool) {
	for idx := 0; idx < len(data); {
		lt := bytes.IndexByte(data[idx:], '<')
		if lt < 0 {
			return noSpan(), false
		}

		start := idx + lt

		if next, ok := markupEnd(data, start); ok {
			idx = next

			continue
		}

		if start+1 < len(data) && data[start+1] != '/' && data[start+1] != '!' {
			nameEnd := elementNameEnd(data, start+1)
			if bytes.EqualFold(data[start+1:nameEnd], []byte(svgTagSVG)) {
				end := tagEnd(data, start)
				if end < 0 {
					return noSpan(), false
				}

				return byteSpan{start, end}, true
			}
		}

		idx = start + 1
	}

	return noSpan(), false
}

// elementSpanName returns the element name at the start of a span.
func elementSpanName(data []byte, span byteSpan) []byte {
	nameStart := span.start + 1
	if nameStart < len(data) && data[nameStart] == '/' {
		nameStart++
	}

	return data[nameStart:elementNameEnd(data, nameStart)]
}

// findElementSpans returns the spans of every element named in names in
// document order. Elements nested inside another match are not reported.
func findElementSpans(data []byte, names ...string) []byteSpan {
	var spans []byteSpan

	for idx := 0; idx < len(data); {
		lt := bytes.IndexByte(data[idx:], '<')
		if lt < 0 {
			break
		}

		start := idx + lt

		if next, ok := markupEnd(data, start); ok {
			idx = next

			continue
		}

		nameStart := start + 1
		if nameStart < len(data) && data[nameStart] == '/' {
			idx = start + 1

			continue
		}

		nameEnd := elementNameEnd(data, nameStart)
		name := data[nameStart:nameEnd]

		if matchesAnyName(name, names) {
			span, ok := elementSpan(data, start, string(name))
			if !ok {
				return spans
			}

			spans = append(spans, span)
			idx = span.end

			continue
		}

		idx = start + 1
	}

	return spans
}

func matchesAnyName(name []byte, names []string) bool {
	for _, want := range names {
		if bytes.EqualFold(name, []byte(want)) {
			return true
		}
	}

	return false
}

// dedupeSpans sorts spans by position and drops those nested inside an
// earlier one.
func dedupeSpans(spans []byteSpan) []byteSpan {
	sort.Slice(spans, func(i, j int) bool { return spans[i].start < spans[j].start })

	kept := make([]byteSpan, 0, len(spans))

	for _, span := range spans {
		if len(kept) > 0 && span.start < kept[len(kept)-1].end {
			continue
		}

		kept = append(kept, span)
	}

	return kept
}

// canonicalPaintServerTag returns the SVG canonical element name when name is
// a defs or gradient element in any case.
func canonicalPaintServerTag(name []byte) string {
	switch {
	case bytes.EqualFold(name, []byte(svgTagDefs)):
		return svgTagDefs
	case bytes.EqualFold(name, []byte(svgTagLinearGradient)):
		return svgTagLinearGradient
	case bytes.EqualFold(name, []byte(svgTagRadialGradient)):
		return svgTagRadialGradient
	}

	return ""
}

// normalizePaintServerCase restores the SVG canonical spelling of defs and
// gradient element names. The HTML pipeline lowercases them, and canvas only
// matches "linearGradient" / "radialGradient" exactly.
func normalizePaintServerCase(data []byte) []byte {
	var out bytes.Buffer

	cursor := 0
	changed := false

	for idx := 0; idx < len(data); {
		lt := bytes.IndexByte(data[idx:], '<')
		if lt < 0 {
			break
		}

		start := idx + lt
		if next, ok := markupEnd(data, start); ok {
			idx = next

			continue
		}

		nameStart := start + 1
		if nameStart < len(data) && data[nameStart] == '/' {
			nameStart++
		}

		nameEnd := elementNameEnd(data, nameStart)
		want := canonicalPaintServerTag(data[nameStart:nameEnd])

		if want != "" && string(data[nameStart:nameEnd]) != want {
			out.Write(data[cursor:nameStart])
			out.WriteString(want)

			cursor = nameEnd
			changed = true
		}

		end := tagEnd(data, start)
		if end < 0 {
			break
		}

		idx = end
	}

	if !changed {
		return data
	}

	out.Write(data[cursor:])

	return out.Bytes()
}

// hoistPaintServers moves <defs> elements and any loose gradient elements
// directly after the root <svg> start tag, so canvas has parsed every paint
// server before a shape can reference it.
func hoistPaintServers(data []byte) []byte {
	root, ok := rootSVGElement(data)
	if !ok || data[root.end-2] == '/' {
		return data
	}

	spans := findElementSpans(data, svgTagDefs)
	gradients := findElementSpans(data, svgTagLinearGradient, svgTagRadialGradient)

	if len(spans) == 0 && len(gradients) == 0 {
		return data
	}

	spans = dedupeSpans(append(spans, gradients...))

	var block bytes.Buffer

	loose := make([]byteSpan, 0, len(spans))

	for _, span := range spans {
		if bytes.EqualFold(elementSpanName(data, span), []byte(svgTagDefs)) {
			block.Write(data[span.start:span.end])

			continue
		}

		loose = append(loose, span)
	}

	if len(loose) > 0 {
		block.WriteString("<" + svgTagDefs + ">")

		for _, span := range loose {
			block.Write(data[span.start:span.end])
		}

		block.WriteString("</" + svgTagDefs + ">")
	}

	var body bytes.Buffer

	cursor := 0

	for _, span := range spans {
		body.Write(data[cursor:span.start])

		cursor = span.end
	}

	body.Write(data[cursor:])

	result := body.Bytes()

	out := make([]byte, 0, len(result)+block.Len())
	out = append(out, result[:root.end]...)
	out = append(out, block.Bytes()...)
	out = append(out, result[root.end:]...)

	return out
}

// attrHasEquals reports whether only SVG whitespace separates nameEnd from the
// '=' that follows it.
func attrHasEquals(data []byte, nameEnd, limit int) bool {
	pos := skipSVGSpaces(data, nameEnd)

	return pos < limit && data[pos] == '='
}

func skipSVGSpaces(data []byte, pos int) int {
	for pos < len(data) && isSVGSpace(data[pos]) {
		pos++
	}

	return pos
}

// viewBoxAttrSpan returns the span of the root viewBox attribute name, matched
// case-insensitively so a lowercased viewbox is found too.
func viewBoxAttrSpan(data []byte, root byteSpan) (byteSpan, bool) {
	scan := root.start

	for scan < root.end {
		rel := asciiIndexFold(data[scan:root.end], lowerViewBoxName)
		if rel < 0 {
			return noSpan(), false
		}

		start := scan + rel
		end := start + len(lowerViewBoxName)

		prevOK := start == root.start || isSVGSpace(data[start-1])
		if prevOK && attrHasEquals(data, end, root.end) {
			return byteSpan{start, end}, true
		}

		scan = end
	}

	return noSpan(), false
}

// normalizeRootViewBox rewrites a lowercased root viewbox attribute to
// viewBox, the only spelling canvas looks up.
func normalizeRootViewBox(data []byte) []byte {
	root, found := rootSVGElement(data)
	if !found {
		return data
	}

	attr, present := viewBoxAttrSpan(data, root)
	if !present || string(data[attr.start:attr.end]) == svgAttrViewBox {
		return data
	}

	out := make([]byte, 0, len(data))
	out = append(out, data[:attr.start]...)
	out = append(out, svgAttrViewBox...)
	out = append(out, data[attr.end:]...)

	return out
}

// attrValueSpan returns the quoted value range of the attribute whose name
// begins at attrStart and equals name.
func attrValueSpan(data []byte, attrStart int, name string) (byteSpan, bool) {
	if attrStart > 0 && !isSVGSpace(data[attrStart-1]) && data[attrStart-1] != '<' {
		return noSpan(), false
	}

	pos := skipSVGSpaces(data, attrStart+len(name))
	if pos >= len(data) || data[pos] != '=' {
		return noSpan(), false
	}

	pos = skipSVGSpaces(data, pos+1)
	if pos >= len(data) || (data[pos] != '"' && data[pos] != '\'') {
		return noSpan(), false
	}

	quote := data[pos]
	valueStart := pos + 1

	rel := bytes.IndexByte(data[valueStart:], quote)
	if rel < 0 {
		return noSpan(), false
	}

	return byteSpan{valueStart, valueStart + rel}, true
}

// fontFamilyResolves reports whether canvas can load at least one family from
// a CSS font-family list on this host.
func fontFamilyResolves(list string) bool {
	if list == "" {
		return true
	}

	_, ok := canvas.FindSystemFont(list, canvas.FontRegular)

	return ok
}

// withFontFallbacks appends sans-serif to font-family lists that resolve to no
// installed family. Canvas panics on those instead of falling back, which
// Rasterize can only turn into a lost image.
func withFontFallbacks(data []byte) []byte {
	var out bytes.Buffer

	cursor := 0
	scan := 0
	changed := false

	for scan < len(data) {
		rel := asciiIndexFold(data[scan:], fontFamilyAttrName)
		if rel < 0 {
			break
		}

		attrStart := scan + rel

		value, ok := attrValueSpan(data, attrStart, fontFamilyAttrName)
		if !ok {
			scan = attrStart + len(fontFamilyAttrName)

			continue
		}

		out.Write(data[cursor:value.end])

		cursor = value.end
		scan = value.end

		if !fontFamilyResolves(string(data[value.start:value.end])) {
			out.WriteString(sansSerifFallback)

			changed = true
		}
	}

	if !changed {
		return data
	}

	out.Write(data[cursor:])

	return out.Bytes()
}
