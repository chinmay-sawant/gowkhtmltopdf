package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Point is one point in PDF user space: points, origin at the bottom-left
// corner of the page, y grows up.
type Point struct {
	X float64
	Y float64
}

// TextRun is one shown string (Tj or TJ) with the page it was painted on.
// X and Y are the origin of the text matrix in page space. Color is the fill
// color in force, each channel in 0..1. Font is the resolved /BaseFont name
// without the leading slash, or "" when the font resource could not be read.
type TextRun struct {
	Page  int
	Text  string
	X     float64
	Y     float64
	Size  float64
	Font  string
	Color [3]float64
}

// StrokeSegment is one stroked line segment in page space, with the stroke
// color and line width in force.
type StrokeSegment struct {
	Page   int
	X1, Y1 float64
	X2, Y2 float64
	Color  [3]float64
	Width  float64
}

// FillRect is one filled subpath, recorded as its page-space bounding box
// with the fill color in force. The writer paints axis-aligned rectangles,
// so the bounding box is exact for this engine's output.
type FillRect struct {
	Page  int
	X, Y  float64
	W, H  float64
	Color [3]float64
}

// ImageBox is one image XObject placement in page space: the lower-left
// corner and the size of the unit square under the CTM at the Do.
type ImageBox struct {
	Page int
	X, Y float64
	W, H float64
}

// PageOps is the drawing record of a PDF produced by this package. Text
// answers what words exist, ParsePageOps answers where each word, rule,
// filled box, and image lands and what color it carries.
type PageOps struct {
	Pages      int
	MediaBoxes [][4]float64
	Texts      []TextRun
	Strokes    []StrokeSegment
	Fills      []FillRect
	Images     []ImageBox
}

// baseFontRE extracts the BaseFont name from a font object dictionary.
var baseFontRE = regexp.MustCompile(`/BaseFont\s*/([^\s/<>\[\]()]+)`)

// ParsePageOps inflates the content streams of every page, follows the text,
// color, path, and Do operators, and returns the drawing record in stream
// order. Text and paths inside a Form XObject are reported at their page
// position, not at their form-local position.
func ParsePageOps(data []byte) (*PageOps, error) {
	objects, rootRef, err := parseOpsObjects(data)
	if err != nil {
		return nil, err
	}

	pageRefs, err := opsPageTreeRefs(objects, rootRef)
	if err != nil {
		return nil, err
	}

	out := &PageOps{
		Pages:      len(pageRefs),
		MediaBoxes: make([][4]float64, 0, len(pageRefs)),
		Texts:      make([]TextRun, 0),
		Strokes:    make([]StrokeSegment, 0),
		Fills:      make([]FillRect, 0),
		Images:     make([]ImageBox, 0),
	}

	for index, pageRef := range pageRefs {
		if err := collectPageOps(objects, pageRef, index, out); err != nil {
			return nil, fmt.Errorf("page %d: %w", index+1, err)
		}
	}

	return out, nil
}

// parseOpsObjects parses the object graph without the page-content and
// annotation checks of parseSemanticPDF. A named /Dest on a link annotation
// is legal output that parseAnnotations rejects because it only accepts a
// page-reference destination; the ops reader must still read those files.
func parseOpsObjects(data []byte) (map[int]semanticObject, int, error) {
	if len(data) == 0 {
		//nolint:err113 // static parse error message
		return nil, 0, errors.New("empty PDF")
	}

	firstLineEnd := bytes.IndexByte(data, '\n')
	if firstLineEnd <= len("%PDF-") {
		//nolint:err113 // static parse error message
		return nil, 0, errors.New("missing PDF header")
	}

	header := string(bytes.TrimSuffix(data[:firstLineEnd], []byte("\r")))
	if !strings.HasPrefix(header, "%PDF-") {
		//nolint:err113 // dynamic header value in message
		return nil, 0, fmt.Errorf("bad PDF header %q", header)
	}

	trimmed := bytes.TrimRight(data, "\r\n")
	if !bytes.HasSuffix(trimmed, []byte("%%EOF")) {
		//nolint:err113 // static parse error message
		return nil, 0, errors.New("missing EOF marker")
	}

	xrefPos, err := parseStartXref(trimmed)
	if err != nil {
		return nil, 0, err
	}

	xref, trailer, err := parseXrefAndTrailer(data, xrefPos)
	if err != nil {
		return nil, 0, err
	}

	objects, err := parseSemanticObjects(data, xref)
	if err != nil {
		return nil, 0, err
	}

	if trailer.root == 0 {
		//nolint:err113 // static parse error message
		return nil, 0, errors.New("trailer has no /Root reference")
	}

	if _, found := objects[trailer.root]; !found {
		//nolint:err113 // dynamic object id in message
		return nil, 0, fmt.Errorf("trailer /Root references missing object %d", trailer.root)
	}

	return objects, trailer.root, nil
}

// opsPageTreeRefs returns the page object references in document order.
func opsPageTreeRefs(objects map[int]semanticObject, rootRef int) ([]int, error) {
	root, found := objects[rootRef]
	if !found {
		//nolint:err113 // dynamic object id in message
		return nil, fmt.Errorf("catalog object %d is missing", rootRef)
	}

	pagesRef, err := requiredRef(root.dict, "/Pages")
	if err != nil {
		return nil, fmt.Errorf("catalog: %w", err)
	}

	pagesTree, found := objects[pagesRef]
	if !found {
		//nolint:err113 // dynamic object id in message
		return nil, fmt.Errorf("catalog /Pages references missing object %d", pagesRef)
	}

	refs, err := requiredRefArray(pagesTree.dict, "/Kids")
	if err != nil {
		return nil, fmt.Errorf("pages tree: %w", err)
	}

	count, err := requiredInt(pagesTree.dict, "/Count")
	if err != nil {
		return nil, fmt.Errorf("pages tree: %w", err)
	}

	if count != len(refs) {
		//nolint:err113 // dynamic counts in message
		return nil, fmt.Errorf("pages tree count = %d, kids = %d", count, len(refs))
	}

	return refs, nil
}

// collectPageOps decodes one page's content stream and walks it.
func collectPageOps(objects map[int]semanticObject, pageRef, index int, out *PageOps) error {
	page, found := objects[pageRef]
	if !found {
		//nolint:err113 // dynamic object id in message
		return fmt.Errorf("page object %d is missing", pageRef)
	}

	mediaBox, err := requiredNumberArray(page.dict, "/MediaBox", 4) //nolint:mnd // MediaBox has four values
	if err != nil {
		return fmt.Errorf("mediabox: %w", err)
	}

	contentsRef, err := requiredRef(page.dict, "/Contents")
	if err != nil {
		return fmt.Errorf("contents: %w", err)
	}

	contents, found := objects[contentsRef]
	if !found || contents.stream == nil {
		//nolint:err113 // dynamic object id in message
		return fmt.Errorf("contents object %d is missing or is not a stream", contentsRef)
	}

	stream, err := decodeSemanticStream(contents)
	if err != nil {
		return fmt.Errorf("contents stream: %w", err)
	}

	resources, err := requiredDictionary(page.dict, "/Resources")
	if err != nil {
		resources = ""
	}

	out.MediaBoxes = append(out.MediaBoxes, [4]float64{mediaBox[0], mediaBox[1], mediaBox[2], mediaBox[3]})

	walker := opsWalker{
		objects:   objects,
		page:      index,
		out:       out,
		visited:   map[int]bool{},
		fontCache: map[string]string{},
	}
	walker.walk(stream, resources, identityOpsMatrix())

	return nil
}

// opsMatrix is a PDF transformation matrix [a b c d e f]. A point maps to
// (a*x + c*y + e, b*x + d*y + f).
type opsMatrix [matrixComponents]float64

func identityOpsMatrix() opsMatrix {
	return opsMatrix{1, 0, 0, 1, 0, 0}
}

func translationOpsMatrix(dx, dy float64) opsMatrix {
	return opsMatrix{1, 0, 0, 1, dx, dy}
}

// multiplyOpsMatrix returns outer x inner: inner applies first, then outer.
// This matches the PDF cm concatenation order (new CTM = CTM x cm).
func multiplyOpsMatrix(outer, inner opsMatrix) opsMatrix {
	return opsMatrix{
		outer[0]*inner[0] + outer[2]*inner[1],
		outer[1]*inner[0] + outer[3]*inner[1],
		outer[0]*inner[2] + outer[2]*inner[3],
		outer[1]*inner[2] + outer[3]*inner[3],
		outer[0]*inner[4] + outer[2]*inner[5] + outer[4],
		outer[1]*inner[4] + outer[3]*inner[5] + outer[5],
	}
}

func applyOpsMatrix(m opsMatrix, x, y float64) (float64, float64) {
	return m[0]*x + m[2]*y + m[4], m[1]*x + m[3]*y + m[5]
}

// opsState is the graphics and text state a q/Q pair saves and restores.
type opsState struct {
	ctm       opsMatrix
	fill      [3]float64
	stroke    [3]float64
	lineWidth float64
	fontName  string
	fontSize  float64
	leading   float64
	tm        opsMatrix
	tlm       opsMatrix
}

func newOpsState() opsState {
	return opsState{ //nolint:exhaustruct // zero-value colors are black by default
		ctm:       identityOpsMatrix(),
		lineWidth: 1,
		tm:        identityOpsMatrix(),
		tlm:       identityOpsMatrix(),
	}
}

// opsPath accumulates the current path as subpaths of points. A rectangle
// becomes a closed five-point subpath so stroking yields its four edges.
type opsPath struct {
	subpaths [][]Point
	current  []Point
	start    Point
}

func (p *opsPath) moveTo(x, y float64) {
	p.flush()
	p.current = []Point{{X: x, Y: y}}
	p.start = Point{X: x, Y: y}
}

func (p *opsPath) lineTo(xPos, yPos float64) {
	if len(p.current) == 0 {
		p.moveTo(xPos, yPos)

		return
	}

	p.current = append(p.current, Point{X: xPos, Y: yPos})
}

func (p *opsPath) rect(x, y, w, h float64) {
	p.flush()
	p.current = []Point{{X: x, Y: y}, {X: x + w, Y: y}, {X: x + w, Y: y + h}, {X: x, Y: y + h}, {X: x, Y: y}}
	p.start = Point{X: x, Y: y}
}

func (p *opsPath) close() {
	if len(p.current) == 0 {
		return
	}

	if last := p.current[len(p.current)-1]; last != p.start {
		p.current = append(p.current, p.start)
	}
}

func (p *opsPath) flush() {
	if len(p.current) > 1 {
		p.subpaths = append(p.subpaths, p.current)
	}

	p.current = nil
}

func (p *opsPath) reset() {
	p.subpaths = nil
	p.current = nil
}

// opsWalker walks one content stream. base carries the accumulated Form
// XObject placement from the page.
type opsWalker struct {
	objects   map[int]semanticObject
	page      int
	out       *PageOps
	visited   map[int]bool
	fontCache map[string]string
}

// walk interprets the drawing operators of stream. Nested Form XObjects
// recurse with their placement folded into base.
//
//nolint:cyclop,funlen,gocognit,gocyclo,maintidx // one content-stream operator dispatch loop
func (w *opsWalker) walk(stream []byte, resources string, base opsMatrix) {
	state := newOpsState()
	stack := make([]opsState, 0, opsStateStackCapacity)
	path := new(opsPath)
	operands := make([]opsToken, 0, opsOperandCapacity)

	for _, token := range tokenizeOpsStream(stream) {
		switch token.kind {
		case opsTokenArrayOpen:
			operands = append(operands, token)

			continue
		case opsTokenArrayClose:
			operands = collapseOpsArray(operands)

			continue
		case opsTokenString, opsTokenNumber, opsTokenName:
			operands = append(operands, token)

			continue
		case opsTokenWord:
		}

		switch token.text {
		case "q":
			stack = append(stack, state)
		case "Q":
			if n := len(stack); n > 0 {
				state = stack[n-1]
				stack = stack[:n-1]
			}
		case "cm":
			if matrix, ok := opsMatrixOperand(operands); ok {
				state.ctm = multiplyOpsMatrix(state.ctm, matrix)
			}
		case "rg":
			if values, ok := opsNumbers(operands, rgbComponents); ok {
				state.fill = [3]float64{values[0], values[1], values[2]}
			}
		case "RG":
			if values, ok := opsNumbers(operands, rgbComponents); ok {
				state.stroke = [3]float64{values[0], values[1], values[2]}
			}
		case "g":
			if value, ok := lastOpsNumber(operands); ok {
				state.fill = [3]float64{value, value, value}
			}
		case "G":
			if value, ok := lastOpsNumber(operands); ok {
				state.stroke = [3]float64{value, value, value}
			}
		case "k":
			if values, ok := opsNumbers(operands, opsCMYKComponents); ok {
				state.fill = cmykOpsColor(values)
			}
		case "K":
			if values, ok := opsNumbers(operands, opsCMYKComponents); ok {
				state.stroke = cmykOpsColor(values)
			}
		case "w":
			if value, ok := lastOpsNumber(operands); ok {
				state.lineWidth = value
			}
		case "BT":
			state.tm = identityOpsMatrix()
			state.tlm = identityOpsMatrix()
		case "TL":
			if value, ok := lastOpsNumber(operands); ok {
				state.leading = value
			}
		case "Td":
			if dx, dy, ok := opsNumberPair(operands); ok {
				state.tlm = multiplyOpsMatrix(translationOpsMatrix(dx, dy), state.tlm)
				state.tm = state.tlm
			}
		case "Tm":
			if matrix, ok := opsMatrixOperand(operands); ok {
				state.tm = matrix
				state.tlm = matrix
			}
		case "T*":
			state.tlm = multiplyOpsMatrix(translationOpsMatrix(0, -state.leading), state.tlm)
			state.tm = state.tlm
		case "Tf":
			if name, size, ok := opsFontOperand(operands); ok {
				state.fontName = name
				state.fontSize = size
			}
		case "Tj", "'", "\"":
			w.emitText(base, state, resources, lastOpsString(operands))
		case "TJ":
			w.emitText(base, state, resources, lastOpsString(operands))
		case "m":
			if x, y, ok := opsNumberPair(operands); ok {
				path.moveTo(x, y)
			}
		case "l":
			if x, y, ok := opsNumberPair(operands); ok {
				path.lineTo(x, y)
			}
		case "c", "v", "y":
			if x, y, ok := opsCurveEnd(operands); ok {
				path.lineTo(x, y)
			}
		case "re":
			if values, ok := opsNumbers(operands, rectComponents); ok {
				path.rect(values[0], values[1], values[2], values[3])
			}
		case "h":
			path.close()
		case "S", "s":
			if token.text == "s" {
				path.close()
			}

			path.flush()
			w.strokePath(base, state, path)
			path.reset()
		case "f", "F", "f*", "B", "B*", "b", "b*":
			if token.text == "b" || token.text == "b*" {
				path.close()
			}

			path.flush()
			w.fillPath(base, state, path)

			if strings.HasPrefix(token.text, "B") || strings.HasPrefix(token.text, "b") {
				w.strokePath(base, state, path)
			}

			path.reset()
		case "n", "W", "W*":
			if token.text == "n" {
				path.reset()
			}
		case "Do":
			if name, ok := lastOpsName(operands); ok {
				w.paintXObject(name, resources, base, state.ctm)
			}
		}

		operands = operands[:0]
	}
}

// emitText records one shown string at the text matrix origin. The page
// position is base x ctm applied to the translation component of tm.
func (w *opsWalker) emitText(base opsMatrix, state opsState, resources, text string) {
	if text == "" {
		return
	}

	pageMatrix := multiplyOpsMatrix(base, state.ctm)
	originX, originY := applyOpsMatrix(pageMatrix, state.tm[4], state.tm[5])

	w.out.Texts = append(w.out.Texts, TextRun{
		Page:  w.page,
		Text:  text,
		X:     originX,
		Y:     originY,
		Size:  state.fontSize,
		Font:  w.fontBaseName(state.fontName, resources),
		Color: state.fill,
	})
}

// fontBaseName resolves a Tf resource name to its /BaseFont, cached per
// resource dictionary.
func (w *opsWalker) fontBaseName(name, resources string) string {
	if name == "" {
		return ""
	}

	key := resources + "\x00" + name
	if cached, ok := w.fontCache[key]; ok {
		return cached
	}

	base := ""

	refs, err := resourceRefs(resources, "/Font")
	if err != nil {
		w.fontCache[key] = base

		return base
	}

	ref, fontRefFound := refs[name]
	if !fontRefFound {
		w.fontCache[key] = base

		return base
	}

	object, objectFound := w.objects[ref]
	if !objectFound {
		w.fontCache[key] = base

		return base
	}

	if match := baseFontRE.FindStringSubmatch(object.dict); match != nil {
		base = match[1]
	}

	w.fontCache[key] = base

	return base
}

// strokePath records every stroked subpath edge in page space.
func (w *opsWalker) strokePath(base opsMatrix, state opsState, path *opsPath) {
	matrix := multiplyOpsMatrix(base, state.ctm)

	for _, subpath := range path.subpaths {
		for index := 1; index < len(subpath); index++ {
			startX, startY := applyOpsMatrix(matrix, subpath[index-1].X, subpath[index-1].Y)
			endX, endY := applyOpsMatrix(matrix, subpath[index].X, subpath[index].Y)

			w.out.Strokes = append(w.out.Strokes, StrokeSegment{
				Page:  w.page,
				X1:    startX,
				Y1:    startY,
				X2:    endX,
				Y2:    endY,
				Color: state.stroke,
				Width: state.lineWidth,
			})
		}
	}
}

// fillPath records one bounding box per filled subpath in page space.
func (w *opsWalker) fillPath(base opsMatrix, state opsState, path *opsPath) {
	matrix := multiplyOpsMatrix(base, state.ctm)

	for _, subpath := range path.subpaths {
		minX, minY := math.Inf(1), math.Inf(1)
		maxX, maxY := math.Inf(-1), math.Inf(-1)

		for _, point := range subpath {
			x, y := applyOpsMatrix(matrix, point.X, point.Y)
			minX = math.Min(minX, x)
			minY = math.Min(minY, y)
			maxX = math.Max(maxX, x)
			maxY = math.Max(maxY, y)
		}

		w.out.Fills = append(w.out.Fills, FillRect{
			Page:  w.page,
			X:     minX,
			Y:     minY,
			W:     maxX - minX,
			H:     maxY - minY,
			Color: state.fill,
		})
	}
}

// paintXObject records an image box or recurses into a Form XObject. An
// image's page box is the unit square under the CTM in force at the Do.
func (w *opsWalker) paintXObject(name, resources string, base, ctm opsMatrix) {
	ref, found := nestedXObjectRef(name, resources)
	if !found || w.visited[ref] {
		return
	}

	object, found := w.objects[ref]
	if !found {
		return
	}

	switch {
	case strings.Contains(object.dict, "/Subtype /Image"):
		matrix := multiplyOpsMatrix(base, ctm)
		originX, originY := applyOpsMatrix(matrix, 0, 0)
		xAxisX, xAxisY := applyOpsMatrix(matrix, 1, 0)
		yAxisX, yAxisY := applyOpsMatrix(matrix, 0, 1)

		w.out.Images = append(w.out.Images, ImageBox{
			Page: w.page,
			X:    originX,
			Y:    originY,
			W:    math.Hypot(xAxisX-originX, xAxisY-originY),
			H:    math.Hypot(yAxisX-originX, yAxisY-originY),
		})
	case strings.Contains(object.dict, "/Subtype /Form"):
		stream, err := decodeSemanticStream(object)
		if err != nil {
			return
		}

		childResources, err := requiredDictionary(object.dict, "/Resources")
		if err != nil {
			childResources = ""
		}

		formMatrix := identityOpsMatrix()
		if values, err := requiredNumberArray(object.dict, "/Matrix", matrixComponents); err == nil {
			copy(formMatrix[:], values)
		}

		w.visited[ref] = true
		w.walk(stream, childResources, multiplyOpsMatrix(base, multiplyOpsMatrix(ctm, formMatrix)))
		delete(w.visited, ref)
	}
}

func cmykOpsColor(values []float64) [3]float64 {
	cyan, magenta, yellow, black := values[0], values[1], values[2], values[3]

	return [3]float64{(1 - cyan) * (1 - black), (1 - magenta) * (1 - black), (1 - yellow) * (1 - black)}
}

// opsTokenKind classifies one content-stream token.
type opsTokenKind uint8

const (
	opsTokenNumber opsTokenKind = iota
	opsTokenString
	opsTokenName
	opsTokenWord
	opsTokenArrayOpen
	opsTokenArrayClose
)

const (
	opsCMYKComponents       = 4
	opsPairComponents       = 2
	opsStateStackCapacity   = 8
	opsOperandCapacity      = 16
	opsTokenCapacityDivisor = 8
)

type opsToken struct {
	kind opsTokenKind
	num  float64
	text string
}

// tokenizeOpsStream scans stream into numbers, strings, names, array
// brackets, and operator words, skipping whitespace, comments, and
// dictionary brackets. Strings are decoded in place so a keyword inside a
// literal never reaches the operator dispatch.
//
//nolint:cyclop,funlen // token classification has one branch per PDF token kind
func tokenizeOpsStream(stream []byte) []opsToken {
	tokens := make([]opsToken, 0, len(stream)/opsTokenCapacityDivisor)

	for pos := 0; pos < len(stream); {
		cur := stream[pos]

		switch {
		case isOpsWhitespace(cur):
			pos++
		case cur == '%':
			for pos < len(stream) && stream[pos] != '\n' {
				pos++
			}
		case cur == '(':
			value, next := scanOpsLiteral(stream, pos)
			tokens = append(tokens, newOpsToken(opsTokenString, 0, value))
			pos = next
		case cur == '<' && pos+1 < len(stream) && stream[pos+1] == '<':
			pos += 2
		case cur == '<':
			value, next := scanOpsHex(stream, pos)
			tokens = append(tokens, newOpsToken(opsTokenString, 0, value))
			pos = next
		case cur == '[':
			tokens = append(tokens, newOpsToken(opsTokenArrayOpen, 0, ""))
			pos++
		case cur == ']':
			tokens = append(tokens, newOpsToken(opsTokenArrayClose, 0, ""))
			pos++
		case cur == '/':
			end := pos + 1
			for end < len(stream) && !isOpsDelimiter(stream[end]) {
				end++
			}

			tokens = append(tokens, newOpsToken(opsTokenName, 0, string(stream[pos+1:end])))
			pos = end
		case cur == '-' || cur == '+' || cur == '.' || (cur >= '0' && cur <= '9'):
			end := pos + 1
			for end < len(stream) && !isOpsDelimiter(stream[end]) {
				end++
			}

			if value, err := strconv.ParseFloat(string(stream[pos:end]), 64); err == nil {
				tokens = append(tokens, newOpsToken(opsTokenNumber, value, ""))
			}

			pos = end
		case isOpsRegular(cur):
			end := pos + 1
			for end < len(stream) && isOpsRegular(stream[end]) {
				end++
			}

			tokens = append(tokens, newOpsToken(opsTokenWord, 0, string(stream[pos:end])))
			pos = end
		default:
			pos++ // stray delimiters such as '>' need no token
		}
	}

	return tokens
}

// scanOpsLiteral returns the decoded literal string that starts at pos and
// the offset just past its closing parenthesis. Escape sequences consume
// their next byte, including a backslash-newline continuation.
func scanOpsLiteral(stream []byte, pos int) (string, int) {
	depth := 0

	for end := pos; end < len(stream); end++ {
		switch stream[end] {
		case '\\':
			end++
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return decodePDFLiteral(string(stream[pos : end+1])), end + 1
			}
		}
	}

	return "", len(stream)
}

// scanOpsHex returns the decoded hex string that starts at pos and the
// offset just past its closing angle bracket.
func scanOpsHex(stream []byte, pos int) (string, int) {
	end := pos + 1
	for end < len(stream) && stream[end] != '>' {
		end++
	}

	digits := make([]byte, 0, end-pos)

	for _, digit := range stream[pos+1 : end] {
		if !isOpsWhitespace(digit) {
			digits = append(digits, digit)
		}
	}

	return decodePDFHex(string(digits)), min(end+1, len(stream))
}

func newOpsToken(kind opsTokenKind, number float64, text string) opsToken {
	return opsToken{kind: kind, num: number, text: text}
}

func isOpsWhitespace(cur byte) bool {
	switch cur {
	case 0, '\t', '\n', '\f', '\r', ' ':
		return true
	}

	return false
}

func isOpsDelimiter(cur byte) bool {
	if isOpsWhitespace(cur) {
		return true
	}

	switch cur {
	case '(', ')', '<', '>', '[', ']', '{', '}', '/', '%':
		return true
	}

	return false
}

func isOpsRegular(cur byte) bool {
	return !isOpsDelimiter(cur)
}

// collapseOpsArray turns the operand run [ ... ] into one string token
// holding the concatenated shown text, so TJ sees a single operand.
func collapseOpsArray(operands []opsToken) []opsToken {
	open := -1

	for index := len(operands) - 1; index >= 0; index-- {
		if operands[index].kind == opsTokenArrayOpen {
			open = index

			break
		}
	}

	if open < 0 {
		return operands
	}

	var text strings.Builder

	for _, token := range operands[open+1:] {
		if token.kind == opsTokenString {
			text.WriteString(token.text)
		}
	}

	return append(operands[:open], newOpsToken(opsTokenString, 0, text.String()))
}

func lastOpsString(operands []opsToken) string {
	if len(operands) == 0 {
		return ""
	}

	last := operands[len(operands)-1]
	if last.kind != opsTokenString {
		return ""
	}

	return last.text
}

func lastOpsName(operands []opsToken) (string, bool) {
	if len(operands) == 0 {
		return "", false
	}

	last := operands[len(operands)-1]
	if last.kind != opsTokenName {
		return "", false
	}

	return last.text, true
}

func lastOpsNumber(operands []opsToken) (float64, bool) {
	if len(operands) == 0 {
		return 0, false
	}

	last := operands[len(operands)-1]
	if last.kind != opsTokenNumber {
		return 0, false
	}

	return last.num, true
}

// opsNumberPair returns the last two numbers as (first, second).
func opsNumberPair(operands []opsToken) (float64, float64, bool) {
	if len(operands) < opsPairComponents {
		return 0, 0, false
	}

	first, second := operands[len(operands)-2], operands[len(operands)-1]
	if first.kind != opsTokenNumber || second.kind != opsTokenNumber {
		return 0, 0, false
	}

	return first.num, second.num, true
}

// opsNumbers returns the last want operands as numbers.
func opsNumbers(operands []opsToken, want int) ([]float64, bool) {
	if len(operands) < want {
		return nil, false
	}

	values := make([]float64, 0, want)

	for index := len(operands) - want; index < len(operands); index++ {
		if operands[index].kind != opsTokenNumber {
			return nil, false
		}

		values = append(values, operands[index].num)
	}

	return values, true
}

// opsMatrixOperand returns the last six numbers as a matrix.
func opsMatrixOperand(operands []opsToken) (opsMatrix, bool) {
	if len(operands) < matrixComponents {
		return opsMatrix{}, false
	}

	var matrix opsMatrix

	for index := range matrixComponents {
		token := operands[len(operands)-matrixComponents+index]
		if token.kind != opsTokenNumber {
			return opsMatrix{}, false
		}

		matrix[index] = token.num
	}

	return matrix, true
}

// opsFontOperand returns the Tf name and size: the last two operands must be
// a name followed by a number.
func opsFontOperand(operands []opsToken) (string, float64, bool) {
	if len(operands) < opsPairComponents {
		return "", 0, false
	}

	name, size := operands[len(operands)-2], operands[len(operands)-1]
	if name.kind != opsTokenName || size.kind != opsTokenNumber {
		return "", 0, false
	}

	return name.text, size.num, true
}

// opsCurveEnd returns the endpoint of a c, v, or y curve: the last pair of
// numbers. Curves are approximated by their endpoint for placement checks.
func opsCurveEnd(operands []opsToken) (float64, float64, bool) {
	return opsNumberPair(operands)
}
