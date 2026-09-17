package svg

import (
	"bytes"
	"html"
	"image"
	"strconv"
	"strings"

	"github.com/tdewolff/canvas"
)

// tdewolff/canvas's SVG parser tracks only font-family and font-size: its
// getFontFace loads FontRegular and never reads font-style or font-weight.
// Styled text therefore rasterizes upright regular (cplusplus.com's wordmark
// asks for bold italic), so prepareCanvasInput rewrites those <text> elements
// into outline <path> elements shaped with the requested face. Plain text is
// left untouched, byte for byte.
//
// The outlines come from canvas's own font loading and shaping, so metrics
// match the rasterizer, including faux bold/italic when the host has no face
// with the requested style.

const (
	textTagName    = "text"
	pathTagName    = "path"
	xAttr          = "x"
	yAttr          = "y"
	dxAttr         = "dx"
	dyAttr         = "dy"
	rotateAttr     = "rotate"
	fontAttr       = "font"
	styleAttr      = "style"
	fontSizeAttr   = "font-size"
	fontStyleAttr  = "font-style"
	fontWeightAttr = "font-weight"
	textAnchorAttr = "text-anchor"
	textLengthAttr = "textlength"
	lengthAdjAttr  = "lengthadjust"
	fontVariant    = "font-variant"

	defaultSVGFontFamily = "serif"
	defaultSVGFontSize   = 16.0

	thinFontWeight    = 100
	lightFontWeight   = 300
	regularFontWeight = 400
	boldFontWeight    = 700
	blackFontWeight   = 900

	weightThinLimit      = 150
	weightExtraLightMax  = 250
	weightLightMax       = 350
	weightRegularMax     = 450
	weightMediumMax      = 550
	weightSemiBoldMax    = 650
	weightBoldMax        = 750
	weightExtraBoldMax   = 850
	percentDenominator   = 100.0
	picaPerInch          = 6.0
	pointPerInch         = 72.0
	quarterMillimeter    = 0.25
	centimeterPerInch    = 10.0
	maxInlineStyleLength = 1 << 10
	defaultNestingCap    = 8
	defaultAttrCap       = 8
	closeTagPrefixLen    = 2
	quotedAttrMinLen     = 2
	midpointDivisor      = 2.0
)

// tagAttr is one start-tag attribute. The raw form keeps the source quoting
// so untouched attributes can be re-emitted byte-for-byte.
type tagAttr struct {
	name string
	raw  string
}

// textStyleState is the inherited SVG text style at one point in the tree.
type textStyleState struct {
	family string
	size   float64
	sizeOK bool
	weight int
	italic bool
	anchor string
}

// textOutlineRewrite is one <text> element to replace with an outline path.
type textOutlineRewrite struct {
	start, end int
	content    string
	family     string
	size       float64
	weight     int
	italic     bool
	anchor     string
	x, y       float64
	attrs      []tagAttr
}

func (state textStyleState) styled() bool {
	return state.weight != regularFontWeight || state.italic
}

func defaultTextStyleState() textStyleState {
	return textStyleState{
		family: defaultSVGFontFamily,
		size:   defaultSVGFontSize,
		sizeOK: true,
		weight: regularFontWeight,
		italic: false,
		anchor: "start",
	}
}

// noRewrite returns an empty rewrite with every field set so exhaustruct sees
// a complete literal; callers must check the bool result instead.
func noRewrite() textOutlineRewrite {
	return textOutlineRewrite{
		start:   0,
		end:     0,
		content: "",
		family:  "",
		size:    0,
		weight:  regularFontWeight,
		italic:  false,
		anchor:  "",
		x:       0,
		y:       0,
		attrs:   nil,
	}
}

// outlineStyledText returns data with every styled <text> element replaced by
// a <path> outline. Input without styled text comes back unchanged.
func outlineStyledText(data []byte) []byte {
	userW, userH, canvasW, canvasH, view := rootCanvasFrame(data)

	rewrites := styledTextRewrites(data, userW, userH)
	if len(rewrites) == 0 {
		return data
	}

	for idx := len(rewrites) - 1; idx >= 0; idx-- {
		outline, ok := outlineStyledTextPath(rewrites[idx], canvasW, canvasH, view)
		if !ok {
			continue
		}

		data = replaceByteSpan(data, rewrites[idx].start, rewrites[idx].end,
			buildTextPathTag(rewrites[idx].attrs, outline))
	}

	return data
}

func replaceByteSpan(data []byte, start, end int, replacement []byte) []byte {
	out := make([]byte, 0, len(data)-(end-start)+len(replacement))
	out = append(out, data[:start]...)
	out = append(out, replacement...)
	out = append(out, data[end:]...)

	return out
}

// rootCanvasFrame mirrors canvas's parseViewBox and init for the root <svg>:
// it returns the user-space and millimeter canvas sizes plus the view matrix
// that maps user units to millimeters.
func rootCanvasFrame(data []byte) (float64, float64, float64, float64, canvas.Matrix) {
	view := canvas.Identity

	root, ok := rootSVGElement(data)
	if !ok {
		return 0, 0, 0, 0, view
	}

	attrs := parseTagAttrs(data, elementNameEnd(data, root.start+1), root.end)
	values := make(map[string]string, len(attrs))

	for _, attr := range attrs {
		values[strings.ToLower(attr.name)] = attrValue(attr)
	}

	viewX, viewY, viewW, viewH := 0.0, 0.0, 0.0, 0.0

	if parts := splitNums(values[lowerViewBoxName]); len(parts) >= viewBoxNumParts {
		viewX, viewY, viewW, viewH = parts[0], parts[1], parts[2], parts[3]
	}

	canvasW := canvasRootDimension(values["width"], viewW)
	canvasH := canvasRootDimension(values["height"], viewH)

	if viewW > 0 && viewH > 0 {
		view = canvas.Identity.Scale(canvasW/viewW, canvasH/viewH).Translate(-viewX, -viewY)
	}

	userW := canvasW * cssDPI / mmPerInch
	userH := canvasH * cssDPI / mmPerInch

	return userW, userH, canvasW, canvasH, view
}

// canvasRootDimension mirrors canvas's parseViewBox rule: a unitless root
// width/height is treated as millimeters, while a viewBox-only root scales
// through 96dpi CSS pixels.
func canvasRootDimension(raw string, viewBoxSize float64) float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasSuffix(raw, "%") {
		return viewBoxSize * mmPerInch / cssDPI
	}

	value, ok := parseSVGUserLength(raw, 1.0)
	if !ok {
		return viewBoxSize * mmPerInch / cssDPI
	}

	return value
}

// styledTextRewrites scans the SVG markup for <text> elements that request a
// non-regular font weight or an italic style, tracking inherited text
// properties through the element stack.
func styledTextRewrites(data []byte, userW, userH float64) []textOutlineRewrite {
	rewrites := make([]textOutlineRewrite, 0, 1)

	state := defaultTextStyleState()
	stack := make([]textStyleState, 0, defaultNestingCap)

	for pos := 0; pos < len(data); {
		lt := bytes.IndexByte(data[pos:], '<')
		if lt < 0 {
			break
		}

		start := pos + lt

		nextPos, nextState, nextStack, more := scanStyledTextAt(
			data, start, state, stack, userW, userH, &rewrites)
		if !more {
			break
		}

		pos, state, stack = nextPos, nextState, nextStack
	}

	return rewrites
}

// scanStyledTextAt advances one markup token starting at start. ok is false
// when the scanner must stop (truncated or malformed tag).
func scanStyledTextAt(
	data []byte,
	start int,
	state textStyleState,
	stack []textStyleState,
	userW, userH float64,
	rewrites *[]textOutlineRewrite,
) (int, textStyleState, []textStyleState, bool) {
	if next, ok := markupEnd(data, start); ok {
		return next, state, stack, true
	}

	if start+1 >= len(data) {
		return start, state, stack, false
	}

	switch data[start+1] {
	case '!', '?':
		return start + 1, state, stack, true
	case '/':
		return popStyledTextEndTag(data, start, state, stack)
	}

	return pushStyledTextStartTag(data, start, state, stack, userW, userH, rewrites)
}

func popStyledTextEndTag(
	data []byte,
	start int,
	state textStyleState,
	stack []textStyleState,
) (int, textStyleState, []textStyleState, bool) {
	tagClose := tagEnd(data, start)
	if tagClose < 0 {
		return start, state, stack, false
	}

	if len(stack) > 0 {
		state = stack[len(stack)-1]
		stack = stack[:len(stack)-1]
	}

	return tagClose, state, stack, true
}

func pushStyledTextStartTag(
	data []byte,
	start int,
	state textStyleState,
	stack []textStyleState,
	userW, userH float64,
	rewrites *[]textOutlineRewrite,
) (int, textStyleState, []textStyleState, bool) {
	nameStart := start + 1
	nameEnd := elementNameEnd(data, nameStart)
	name := data[nameStart:nameEnd]

	tagClose := tagEnd(data, start)
	if tagClose < 0 {
		return start, state, stack, false
	}

	attrs := parseTagAttrs(data, nameEnd, tagClose)
	selfClosing := data[tagClose-closeTagPrefixLen] == '/'
	local := applyTextStyleAttrs(state, attrs)

	if bytes.EqualFold(name, []byte(textTagName)) {
		span, ok := elementSpan(data, start, textTagName)
		if ok {
			if rewrite, rewritten := textOutlineFor(data, span, tagClose, local, attrs, userW, userH); rewritten {
				*rewrites = append(*rewrites, rewrite)
			}

			return span.end, state, stack, true
		}
	}

	if !selfClosing {
		stack = append(stack, state)
		state = local
	}

	return tagClose, state, stack, true
}

// textOutlineFor turns one <text> element into a rewrite when it is styled,
// holds plain text only, and carries parseable coordinates.
func textOutlineFor(
	data []byte,
	span byteSpan,
	contentStart int,
	style textStyleState,
	attrs []tagAttr,
	userW, userH float64,
) (textOutlineRewrite, bool) {
	if !style.styled() || !style.sizeOK {
		return noRewrite(), false
	}

	content, contentOK := plainTextElementContent(data[contentStart:span.end])
	if !contentOK {
		return noRewrite(), false
	}

	posX, posY, posOK := textOutlinePosition(attrs, userW, userH)
	if !posOK {
		return noRewrite(), false
	}

	return textOutlineRewrite{
		start:   span.start,
		end:     span.end,
		content: content,
		family:  style.family,
		size:    style.size,
		weight:  style.weight,
		italic:  style.italic,
		anchor:  style.anchor,
		x:       posX,
		y:       posY,
		attrs:   attrs,
	}, true
}

func plainTextElementContent(inner []byte) (string, bool) {
	openIdx := bytes.IndexByte(inner, '<')
	prefixEnd := min(openIdx+closeTagPrefixLen, len(inner))

	if openIdx < 0 || !bytes.HasPrefix(bytes.ToLower(inner[openIdx:prefixEnd]), []byte("</")) {
		return "", false
	}

	content := html.UnescapeString(string(inner[:openIdx]))
	if strings.TrimSpace(content) == "" {
		return "", false
	}

	return content, true
}

func textOutlinePosition(attrs []tagAttr, userW, userH float64) (float64, float64, bool) {
	posX, posY := 0.0, 0.0

	for _, attr := range attrs {
		switch strings.ToLower(attr.name) {
		case xAttr:
			value, ok := parseSVGUserLength(attrValue(attr), userW)
			if !ok {
				return 0, 0, false
			}

			posX = value
		case yAttr:
			value, ok := parseSVGUserLength(attrValue(attr), userH)
			if !ok {
				return 0, 0, false
			}

			posY = value
		case dxAttr, dyAttr, rotateAttr:
			return 0, 0, false
		}
	}

	return posX, posY, true
}

// outlineStyledTextPath shapes the rewrite with canvas's own font machinery
// and returns user-space SVG path data for the glyph outlines.
func outlineStyledTextPath(
	rewrite textOutlineRewrite,
	canvasW, canvasH float64,
	view canvas.Matrix,
) (string, bool) {
	style := canvasFontStyle(rewrite.weight, rewrite.italic)

	family, ok := loadStyledFontFamily(rewrite.family, style)
	if !ok {
		return "", false
	}

	face := family.Face(rewrite.size*(pointPerInch/mmPerInch), style)
	text := canvas.NewTextLine(face, rewrite.content, textAlignFor(rewrite.anchor))

	renderer := &outlineRenderer{width: canvasW, height: canvasH, paths: nil}
	ctx := canvas.NewContext(renderer)
	ctx.SetCoordSystem(canvas.CartesianIV)
	ctx.SetView(view)
	ctx.DrawText(rewrite.x, rewrite.y, text)

	if len(renderer.paths) == 0 {
		return "", false
	}

	// Paths are parsed as frame*(path coords) while text adds a ReflectY for
	// its y-up glyph space; inverting the full frame lands in user space.
	inverse := canvas.Identity.ReflectYAbout(canvasH / midpointDivisor).Mul(view).Inv()

	var outline strings.Builder

	for _, path := range renderer.paths {
		outline.WriteString(path.Transform(inverse).ToSVG())
	}

	if outline.Len() == 0 {
		return "", false
	}

	return outline.String(), true
}

// loadStyledFontFamily loads the exact requested face when the host has a
// distinct bold/italic file, and otherwise falls back to the regular face so
// FontFamily.Face can synthesize faux weight and slant.
func loadStyledFontFamily(list string, style canvas.FontStyle) (*canvas.FontFamily, bool) {
	wanted, wantedOK := canvas.FindSystemFont(list, style)
	regular, regularOK := canvas.FindSystemFont(list, canvas.FontRegular)

	family := canvas.NewFontFamily(list)

	switch {
	case wantedOK && wanted != regular:
		if err := family.LoadSystemFont(list, style); err != nil {
			return nil, false
		}
	case regularOK:
		if err := family.LoadSystemFont(list, canvas.FontRegular); err != nil {
			return nil, false
		}
	default:
		return nil, false
	}

	return family, true
}

// canvasFontStyle maps a CSS font weight to the matching canvas style.
func canvasFontStyle(weight int, italic bool) canvas.FontStyle {
	style := canvasFontWeightStyle(weight)
	if italic {
		style |= canvas.FontItalic
	}

	return style
}

func canvasFontWeightStyle(weight int) canvas.FontStyle {
	switch {
	case weight <= weightThinLimit:
		return canvas.FontThin
	case weight <= weightExtraLightMax:
		return canvas.FontExtraLight
	case weight <= weightLightMax:
		return canvas.FontLight
	case weight <= weightRegularMax:
		return canvas.FontRegular
	case weight <= weightMediumMax:
		return canvas.FontMedium
	case weight <= weightSemiBoldMax:
		return canvas.FontSemiBold
	case weight <= weightBoldMax:
		return canvas.FontBold
	case weight <= weightExtraBoldMax:
		return canvas.FontExtraBold
	default:
		return canvas.FontBlack
	}
}

func textAlignFor(anchor string) canvas.TextAlign {
	switch strings.ToLower(strings.TrimSpace(anchor)) {
	case "middle":
		return canvas.Center
	case "end":
		return canvas.Right
	default:
		return canvas.Left
	}
}

// outlineRenderer collects glyph outlines instead of painting them.
type outlineRenderer struct {
	width  float64
	height float64
	paths  []*canvas.Path
}

func (renderer *outlineRenderer) Size() (float64, float64) {
	return renderer.width, renderer.height
}

func (renderer *outlineRenderer) RenderPath(path *canvas.Path, _ canvas.Style, matrix canvas.Matrix) {
	if path == nil || path.Empty() {
		return
	}

	renderer.paths = append(renderer.paths, path.Copy().Transform(matrix))
}

func (renderer *outlineRenderer) RenderText(text *canvas.Text, matrix canvas.Matrix) {
	text.RenderTo(renderer, matrix, 0)
}

func (renderer *outlineRenderer) RenderImage(image.Image, canvas.Matrix) {}

// buildTextPathTag re-emits the <text> attributes that still apply to an
// outline path, dropping positional and text-layout attributes.
func buildTextPathTag(attrs []tagAttr, outline string) []byte {
	var tag bytes.Buffer

	tag.WriteString("<" + pathTagName)

	for _, attr := range attrs {
		if omitTextAttr(strings.ToLower(attr.name)) {
			continue
		}

		tag.WriteByte(' ')
		tag.WriteString(attr.name)

		if attr.raw == "" {
			continue
		}

		tag.WriteByte('=')
		tag.WriteString(attr.raw)
	}

	tag.WriteString(` d="`)
	tag.WriteString(outline)
	tag.WriteString(`"/>`)

	return tag.Bytes()
}

func omitTextAttr(name string) bool {
	switch name {
	case xAttr, yAttr, dxAttr, dyAttr, rotateAttr, "d",
		fontFamilyAttrName, fontSizeAttr, fontStyleAttr, fontWeightAttr,
		fontAttr, fontVariant, textAnchorAttr, textLengthAttr, lengthAdjAttr:
		return true
	default:
		return false
	}
}

// applyTextStyleAttrs applies one start tag's presentation attributes and
// inline style declarations on top of the inherited state.
func applyTextStyleAttrs(state textStyleState, attrs []tagAttr) textStyleState {
	styleValue := ""
	hasStyle := false

	for _, attr := range attrs {
		name := strings.ToLower(attr.name)
		if name == styleAttr {
			styleValue, hasStyle = attrValue(attr), true

			continue
		}

		state = applyOneTextStyle(state, name, attrValue(attr))
	}

	if !hasStyle || len(styleValue) > maxInlineStyleLength {
		return state
	}

	return applyInlineTextStyle(state, styleValue)
}

func applyInlineTextStyle(state textStyleState, styleValue string) textStyleState {
	for _, declaration := range strings.Split(styleValue, ";") {
		key, value, found := strings.Cut(declaration, ":")
		if !found {
			continue
		}

		state = applyOneTextStyle(state, strings.ToLower(strings.TrimSpace(key)), value)
	}

	return state
}

func applyOneTextStyle(state textStyleState, name, value string) textStyleState {
	switch name {
	case fontFamilyAttrName:
		state.family = strings.TrimSpace(value)
	case fontSizeAttr:
		state.size, state.sizeOK = parseSVGUserLength(value, state.size)
	case fontStyleAttr:
		state.italic = parseFontItalic(value, state.italic)
	case fontWeightAttr:
		state.weight = parseFontWeight(value, state.weight)
	case textAnchorAttr:
		state.anchor = strings.TrimSpace(value)
	}

	return state
}

func parseFontItalic(raw string, previous bool) bool {
	fields := strings.Fields(strings.ToLower(raw))
	if len(fields) == 0 {
		return previous
	}

	switch fields[0] {
	case "italic", "oblique":
		return true
	case "normal":
		return false
	default:
		return previous
	}
}

func parseFontWeight(raw string, previous int) int {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "normal":
		return regularFontWeight
	case "bold":
		return boldFontWeight
	case "bolder":
		if previous >= boldFontWeight {
			return blackFontWeight
		}

		return boldFontWeight
	case "lighter":
		if previous > lightFontWeight {
			return lightFontWeight
		}

		return thinFontWeight
	}

	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value < thinFontWeight || value > blackFontWeight {
		return previous
	}

	return value
}

// parseSVGUserLength converts an SVG length to user units the way canvas's
// parseDimension does: unitless and px are user units, absolute units convert
// through 96dpi, and percentages resolve against parent.
func parseSVGUserLength(raw string, parent float64) (float64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}

	numberEnd := svgNumberEnd(raw)
	if numberEnd == 0 {
		return 0, false
	}

	value, err := strconv.ParseFloat(raw[:numberEnd], 64)
	if err != nil {
		return 0, false
	}

	return svgLengthToUser(value, strings.ToLower(strings.TrimSpace(raw[numberEnd:])), parent)
}

func svgLengthToUser(value float64, unit string, parent float64) (float64, bool) {
	switch unit {
	case "", "px":
		return value, true
	case "cm":
		return value * centimeterPerInch * cssDPI / mmPerInch, true
	case "mm":
		return value * cssDPI / mmPerInch, true
	case "q":
		return value * quarterMillimeter * cssDPI / mmPerInch, true
	case "in":
		return value * cssDPI, true
	case "pc":
		return value * cssDPI / picaPerInch, true
	case "pt":
		return value * cssDPI / pointPerInch, true
	case "%":
		return value * parent / percentDenominator, true
	default:
		return 0, false
	}
}

// svgNumberEnd returns the length of the leading number in raw, or 0 when raw
// does not start with one.
func svgNumberEnd(raw string) int {
	end, digits := consumeSVGIntegerPart(raw, 0)
	end, digits = consumeSVGFractionPart(raw, end, digits)

	if !digits {
		return 0
	}

	return consumeSVGExponentPart(raw, end)
}

func consumeSVGIntegerPart(raw string, end int) (int, bool) {
	if end < len(raw) && (raw[end] == '+' || raw[end] == '-') {
		end++
	}

	digits := false

	for end < len(raw) && isASCIIDigit(raw[end]) {
		end++
		digits = true
	}

	return end, digits
}

func consumeSVGFractionPart(raw string, end int, digits bool) (int, bool) {
	if end >= len(raw) || raw[end] != '.' {
		return end, digits
	}

	end++

	for end < len(raw) && isASCIIDigit(raw[end]) {
		end++
		digits = true
	}

	return end, digits
}

func consumeSVGExponentPart(raw string, end int) int {
	if end >= len(raw) || (raw[end] != 'e' && raw[end] != 'E') {
		return end
	}

	exponentStart := end
	end++

	if end < len(raw) && (raw[end] == '+' || raw[end] == '-') {
		end++
	}

	exponentDigits := false

	for end < len(raw) && isASCIIDigit(raw[end]) {
		end++
		exponentDigits = true
	}

	if !exponentDigits {
		return exponentStart
	}

	return end
}

func isASCIIDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// parseTagAttrs reads the attributes of a start tag body, preserving the
// source quoting of each value. tagClose is the index just past the closing
// '>'.
func parseTagAttrs(data []byte, pos, tagClose int) []tagAttr {
	attrs := make([]tagAttr, 0, defaultAttrCap)

	for pos < tagClose {
		pos = skipSVGSpaces(data, pos)
		if pos >= tagClose || data[pos] == '/' || data[pos] == '>' {
			break
		}

		attr, next := readOneTagAttr(data, pos, tagClose)
		attrs = append(attrs, attr)
		pos = next
	}

	return attrs
}

func readOneTagAttr(data []byte, pos, tagClose int) (tagAttr, int) {
	nameStart := pos

	for pos < tagClose && !isSVGSpace(data[pos]) && data[pos] != '=' &&
		data[pos] != '/' && data[pos] != '>' {
		pos++
	}

	name := string(data[nameStart:pos])
	pos = skipSVGSpaces(data, pos)

	if pos >= tagClose || data[pos] != '=' {
		return tagAttr{name: name, raw: ""}, pos
	}

	pos = skipSVGSpaces(data, pos+1)

	value, next := attributeRawValue(data, pos, tagClose)
	if next <= pos {
		next = pos + 1
	}

	return tagAttr{name: name, raw: value}, next
}

// attributeRawValue returns one attribute value including its source quoting
// and the offset just past it.
func attributeRawValue(data []byte, pos, limit int) (string, int) {
	if pos >= limit {
		return "", pos
	}

	if data[pos] == '"' || data[pos] == '\'' {
		return quotedAttributeRawValue(data, pos, limit)
	}

	start := pos

	for pos < limit && !isSVGSpace(data[pos]) && data[pos] != '/' && data[pos] != '>' {
		pos++
	}

	return string(data[start:pos]), pos
}

func quotedAttributeRawValue(data []byte, pos, limit int) (string, int) {
	quote := data[pos]
	start := pos
	pos++

	for pos < limit && data[pos] != quote {
		pos++
	}

	if pos < limit {
		pos++
	}

	return string(data[start:pos]), pos
}

func attrValue(attr tagAttr) string {
	raw := attr.raw
	if len(raw) < quotedAttrMinLen {
		return raw
	}

	first := raw[0]
	if (first == '"' || first == '\'') && raw[len(raw)-1] == first {
		return raw[1 : len(raw)-1]
	}

	return raw
}
