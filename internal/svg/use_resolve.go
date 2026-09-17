package svg

import (
	"bytes"
	"strings"
)

const (
	svgTagUse    = "use"
	svgTagSymbol = "symbol"
	svgTagG      = "g"

	svgAttrHref      = "href"
	svgAttrXLinkHref = "xlink:href"
	svgAttrID        = "id"
)

// useTransferAttrs are copied from <use> onto the inlined wrapper so
// positioning and presentation survive the rewrite.
var useTransferAttrs = []string{ //nolint:gochecknoglobals // fixed attr allowlist
	"x", "y", "width", "height", "transform",
	"fill", "stroke", "stroke-width", "stroke-linecap", "stroke-linejoin",
	"stroke-opacity", "fill-opacity", "opacity", "class", "style",
}

// ResolveUseReferences inlines <use href|xlink:href="...#id"> by replacing
// each reference with a <g> or nested <svg> that holds the target geometry.
//
// Fragment-only values (#id) resolve against data itself. Values with a URL
// before the hash call fetch once per distinct URL; a nil fetch leaves those
// <use> elements unchanged. Missing ids, fetch failures, and malformed markup
// leave the original <use> in place (no panic). Nested <use> inside a fetched
// sprite is not expanded.
//
// Layout should call this before Rasterize when Options.Images can fetch
// external sprite files:
//
//	data, _ = svg.ResolveUseReferences(data, eng.resolveImageData)
//	svg.Rasterize(data, maxSide)
func ResolveUseReferences(data []byte, fetch func(string) ([]byte, error)) ([]byte, error) {
	if len(data) == 0 || !bytesContainsFold(data, []byte("<use")) {
		return data, nil
	}

	uses := findElementSpans(data, svgTagUse)
	if len(uses) == 0 {
		return data, nil
	}

	cache := map[string][]byte{}
	out := data

	// Rewrite from the end so earlier spans keep valid offsets.
	for i := len(uses) - 1; i >= 0; i-- {
		span := uses[i]
		if span.end > len(out) || span.start >= span.end {
			continue
		}

		replacement, ok := resolveOneUse(out, span, fetch, cache)
		if !ok {
			continue
		}

		buf := make([]byte, 0, len(out)-span.end+span.start+len(replacement))
		buf = append(buf, out[:span.start]...)
		buf = append(buf, replacement...)
		buf = append(buf, out[span.end:]...)
		out = buf
	}

	return out, nil
}

func resolveOneUse(
	data []byte,
	span byteSpan,
	fetch func(string) ([]byte, error),
	cache map[string][]byte,
) ([]byte, bool) {
	tagClose := tagEnd(data, span.start)
	if tagClose < 0 || tagClose > span.end {
		return nil, false
	}

	nameEnd := elementNameEnd(data, span.start+1)
	attrs := parseTagAttrs(data, nameEnd, tagClose)

	ref := useHrefValue(attrs)
	if ref == "" {
		return nil, false
	}

	url, fragID := splitUseRef(ref)
	if fragID == "" {
		return nil, false
	}

	source := data

	if url != "" {
		fetched, ok := loadUseDocument(url, fetch, cache)
		if !ok {
			return nil, false
		}

		source = fetched
	}

	target, ok := findElementByID(source, fragID)
	if !ok {
		return nil, false
	}

	return buildUseReplacement(attrs, source, target), true
}

func useHrefValue(attrs []tagAttr) string {
	var href, xlink string

	for _, attr := range attrs {
		switch strings.ToLower(attr.name) {
		case svgAttrHref:
			href = strings.TrimSpace(attrValue(attr))
		case svgAttrXLinkHref:
			xlink = strings.TrimSpace(attrValue(attr))
		}
	}

	if href != "" {
		return href
	}

	return xlink
}

func splitUseRef(ref string) (string, string) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", ""
	}

	hash := strings.IndexByte(ref, '#')
	if hash < 0 {
		return ref, ""
	}

	return ref[:hash], ref[hash+1:]
}

func loadUseDocument(
	url string,
	fetch func(string) ([]byte, error),
	cache map[string][]byte,
) ([]byte, bool) {
	if fetch == nil {
		return nil, false
	}

	if cached, ok := cache[url]; ok {
		return cached, cached != nil
	}

	body, err := fetch(url)
	if err != nil || len(body) == 0 || !looksLikeSVG(body) {
		cache[url] = nil

		return nil, false
	}

	cache[url] = body

	return body, true
}

// findElementByID returns the span of the first element whose id attribute
// equals elementID (SVG ids are matched exactly as written in the markup).
func findElementByID(data []byte, elementID string) (byteSpan, bool) {
	if elementID == "" {
		return noSpan(), false
	}

	for idx := 0; idx < len(data); {
		lt := bytes.IndexByte(data[idx:], '<')
		if lt < 0 {
			break
		}

		start := idx + lt
		nextIdx, span, found := matchElementIDAt(data, start, elementID)

		switch {
		case found:
			return span, true
		case nextIdx < 0:
			return noSpan(), false
		default:
			idx = nextIdx
		}
	}

	return noSpan(), false
}

// matchElementIDAt inspects the markup starting at start. It returns the next
// scan index, and when found is true the matching element span.
func matchElementIDAt(data []byte, start int, elementID string) (int, byteSpan, bool) {
	if next, ok := markupEnd(data, start); ok {
		return next, noSpan(), false
	}

	if start+1 >= len(data) || data[start+1] == '/' || data[start+1] == '!' {
		return start + 1, noSpan(), false
	}

	nameEnd := elementNameEnd(data, start+1)
	tagClose := tagEnd(data, start)

	if tagClose < 0 {
		return -1, noSpan(), false
	}

	attrs := parseTagAttrs(data, nameEnd, tagClose)
	if attrNamed(attrs, svgAttrID) != elementID {
		return tagClose, noSpan(), false
	}

	name := string(data[start+1 : nameEnd])
	span, ok := elementSpan(data, start, name)

	if !ok {
		return -1, noSpan(), false
	}

	return span.end, span, true
}

func attrNamed(attrs []tagAttr, name string) string {
	for _, attr := range attrs {
		if strings.EqualFold(attr.name, name) {
			return attrValue(attr)
		}
	}

	return ""
}

func buildUseReplacement(useAttrs []tagAttr, source []byte, target byteSpan) []byte {
	tagClose := tagEnd(source, target.start)
	if tagClose < 0 || tagClose > target.end {
		return nil
	}

	nameEnd := elementNameEnd(source, target.start+1)
	name := string(source[target.start+1 : nameEnd])
	targetAttrs := parseTagAttrs(source, nameEnd, tagClose)

	if strings.EqualFold(name, svgTagSymbol) {
		content := stripElementIDs(innerElementContent(source, tagClose, target.end, name))

		return buildSymbolUseReplacement(useAttrs, targetAttrs, content)
	}

	// Non-symbol target: clone the element inside a group. x/y become a
	// translate when set, matching SVG <use> positioning.
	return buildClonedUseReplacement(useAttrs, source[target.start:target.end])
}

func innerElementContent(source []byte, tagClose, end int, name string) []byte {
	if tagClose >= end {
		return nil
	}

	content := source[tagClose:end]
	closeTag := "</" + name + ">"

	if len(content) >= len(closeTag) &&
		bytes.EqualFold(content[len(content)-len(closeTag):], []byte(closeTag)) {
		content = content[:len(content)-len(closeTag)]
	}

	return content
}

func buildSymbolUseReplacement(useAttrs, targetAttrs []tagAttr, content []byte) []byte {
	var buf bytes.Buffer

	viewBox := attrNamed(targetAttrs, svgAttrViewBox)
	if viewBox == "" {
		viewBox = attrNamed(targetAttrs, lowerViewBoxName)
	}

	wrapper := svgTagG
	if viewBox != "" {
		wrapper = svgTagSVG
	}

	buf.WriteString("<")
	buf.WriteString(wrapper)
	writeSelectedAttrs(&buf, useAttrs, useTransferAttrs)

	if viewBox != "" {
		buf.WriteString(` viewBox="`)
		buf.WriteString(escapeXMLAttr(viewBox))
		buf.WriteByte('"')
	}

	buf.WriteByte('>')
	buf.Write(content)
	buf.WriteString("</")
	buf.WriteString(wrapper)
	buf.WriteByte('>')

	return buf.Bytes()
}

func buildClonedUseReplacement(useAttrs []tagAttr, targetMarkup []byte) []byte {
	var buf bytes.Buffer

	buf.WriteString("<")
	buf.WriteString(svgTagG)
	writeUsePositionAttrs(&buf, useAttrs)
	buf.WriteByte('>')
	buf.Write(targetMarkup)
	buf.WriteString("</")
	buf.WriteString(svgTagG)
	buf.WriteByte('>')

	return buf.Bytes()
}

func writeUsePositionAttrs(buf *bytes.Buffer, useAttrs []tagAttr) {
	useX := attrNamed(useAttrs, "x")
	useY := attrNamed(useAttrs, "y")
	transform := attrNamed(useAttrs, "transform")

	if useX == "" && useY == "" {
		writeSelectedAttrs(buf, useAttrs, useTransferAttrs)

		return
	}

	if useX == "" {
		useX = "0"
	}

	if useY == "" {
		useY = "0"
	}

	buf.WriteString(` transform="`)

	if transform != "" {
		buf.WriteString(escapeXMLAttr(transform))
		buf.WriteByte(' ')
	}

	buf.WriteString("translate(")
	buf.WriteString(escapeXMLAttr(useX))
	buf.WriteByte(',')
	buf.WriteString(escapeXMLAttr(useY))
	buf.WriteString(`)"`)
	writeSelectedAttrs(buf, useAttrs, useTransferAttrsExceptXYTransform)
}

var useTransferAttrsExceptXYTransform = []string{ //nolint:gochecknoglobals // fixed attr allowlist
	"width", "height",
	"fill", "stroke", "stroke-width", "stroke-linecap", "stroke-linejoin",
	"stroke-opacity", "fill-opacity", "opacity", "class", "style",
}

func writeSelectedAttrs(buf *bytes.Buffer, attrs []tagAttr, names []string) {
	have := map[string]tagAttr{}
	for _, attr := range attrs {
		have[strings.ToLower(attr.name)] = attr
	}

	for _, name := range names {
		attr, ok := have[name]
		if !ok || attrValue(attr) == "" {
			continue
		}

		buf.WriteByte(' ')
		buf.WriteString(name)
		buf.WriteString(`="`)
		buf.WriteString(escapeXMLAttr(attrValue(attr)))
		buf.WriteByte('"')
	}
}

// stripElementIDs removes id="..." from inlined markup so repeated <use> of
// the same symbol does not emit duplicate ids.
func stripElementIDs(data []byte) []byte {
	if !bytesContainsFold(data, []byte("id")) {
		return data
	}

	var out bytes.Buffer

	cursor := 0

	for idx := 0; idx < len(data); {
		lt := bytes.IndexByte(data[idx:], '<')
		if lt < 0 {
			break
		}

		start := idx + lt
		nextIdx, rewritten := stripIDAtTag(data, start, cursor, &out)

		if nextIdx < 0 {
			break
		}

		if rewritten {
			cursor = nextIdx
		}

		idx = nextIdx
	}

	if cursor == 0 {
		return data
	}

	out.Write(data[cursor:])

	return out.Bytes()
}

// stripIDAtTag advances past the tag at start. When the tag has an id, it
// appends the preceding bytes and a rewritten start tag to out and reports
// rewritten=true so the caller updates its copy cursor.
func stripIDAtTag(data []byte, start, cursor int, out *bytes.Buffer) (int, bool) {
	if next, ok := markupEnd(data, start); ok {
		return next, false
	}

	if start+1 >= len(data) || data[start+1] == '/' || data[start+1] == '!' {
		return start + 1, false
	}

	tagClose := tagEnd(data, start)
	if tagClose < 0 {
		return -1, false
	}

	nameEnd := elementNameEnd(data, start+1)
	attrs := parseTagAttrs(data, nameEnd, tagClose)
	idName := idAttrName(attrs)

	if idName == "" {
		return tagClose, false
	}

	out.Write(data[cursor:start])
	writeStartTagWithoutAttr(out, data, start, nameEnd, tagClose, attrs, idName)

	return tagClose, true
}

func idAttrName(attrs []tagAttr) string {
	for _, attr := range attrs {
		if strings.EqualFold(attr.name, svgAttrID) {
			return attr.name
		}
	}

	return ""
}

func writeStartTagWithoutAttr(
	out *bytes.Buffer,
	data []byte,
	start, nameEnd, tagClose int,
	attrs []tagAttr,
	skipName string,
) {
	out.WriteByte('<')
	out.Write(data[start+1 : nameEnd])

	for _, attr := range attrs {
		if strings.EqualFold(attr.name, skipName) {
			continue
		}

		out.WriteByte(' ')
		out.WriteString(attr.name)

		if attr.raw != "" {
			out.WriteByte('=')
			out.WriteString(attr.raw)
		}
	}

	selfClosingOffset := 2
	if data[tagClose-selfClosingOffset] == '/' {
		out.WriteString("/>")

		return
	}

	out.WriteByte('>')
}

func escapeXMLAttr(value string) string {
	replacer := strings.NewReplacer(
		`&`, "&amp;",
		`<`, "&lt;",
		`"`, "&quot;",
	)

	return replacer.Replace(value)
}
