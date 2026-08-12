package pdf

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var (
	semanticObjectHeaderRE = regexp.MustCompile(`^(\d+) 0 obj\n`)
	semanticRefRE          = regexp.MustCompile(`(\d+) 0 R`)
	semanticResourceRE     = regexp.MustCompile(`/([A-Za-z][A-Za-z0-9_]*)\s+(\d+)\s+0\s+R`)
	semanticNumberRE       = regexp.MustCompile(`[-+]?\d+(?:\.\d+)?`)
	semanticLiteralRE      = regexp.MustCompile(`(?s)(\((?:\\.|[^\\)])*\))\s*Tj`)
	semanticHexRE          = regexp.MustCompile(`(?s)<([0-9A-Fa-f]*)>\s*Tj`)
)

// SemanticDoc is a small, production-safe view of a PDF this package emits.
// It exposes document-order page text, embedded image XObjects, and
// URI / GoTo annotations so callers can assert authored content without
// depending on a full PDF object model.
type SemanticDoc struct {
	Version string
	Pages   []SemanticPage
}

// SemanticPage is one page of a SemanticDoc.
type SemanticPage struct {
	MediaBox [4]float64
	Text     string
	Fonts    map[string]int
	Images   map[string]int
	Annots   []SemanticAnnot
}

// SemanticAnnot is a URI or internal destination annotation.
type SemanticAnnot struct {
	URI      string
	DestPage int
}

// ParseSemantic parses a PDF produced by this package.
func ParseSemantic(data []byte) (*SemanticDoc, error) {
	doc, err := parseSemanticPDF(data)
	if err != nil {
		return nil, err
	}

	out := &SemanticDoc{
		Version: doc.version,
		Pages:   make([]SemanticPage, 0, len(doc.pages)),
	}

	for _, page := range doc.pages {
		annots := make([]SemanticAnnot, 0, len(page.annots))
		for _, annot := range page.annots {
			annots = append(annots, SemanticAnnot{URI: annot.uri, DestPage: annot.destPage})
		}

		out.Pages = append(out.Pages, SemanticPage{
			MediaBox: page.mediaBox,
			Text:     page.text,
			Fonts:    page.fonts,
			Images:   page.images,
			Annots:   annots,
		})
	}

	return out, nil
}

// PageCount returns the number of pages in the document.
func (d *SemanticDoc) PageCount() int {
	if d == nil {
		return 0
	}

	return len(d.Pages)
}

// DocumentText concatenates extracted page text in document order.
func (d *SemanticDoc) DocumentText() string {
	if d == nil {
		return ""
	}

	var text strings.Builder
	for _, page := range d.Pages {
		text.WriteString(page.Text)
	}

	return text.String()
}

// HasURI reports whether any page carries a URI annotation.
func (d *SemanticDoc) HasURI() bool {
	if d == nil {
		return false
	}

	for _, page := range d.Pages {
		for _, annot := range page.Annots {
			if annot.URI != "" {
				return true
			}
		}
	}

	return false
}

// HasImageXObject reports whether any page references an image XObject.
func (d *SemanticDoc) HasImageXObject() bool {
	if d == nil {
		return false
	}

	for _, page := range d.Pages {
		if len(page.Images) > 0 {
			return true
		}
	}

	return false
}

// HasInternalDest reports whether any page carries a GoTo / Dest annotation.
func (d *SemanticDoc) HasInternalDest() bool {
	if d == nil {
		return false
	}

	for _, page := range d.Pages {
		for _, annot := range page.Annots {
			if annot.DestPage != 0 {
				return true
			}
		}
	}

	return false
}

// semanticPDF is the internal object model used by ParseSemantic and the
// same-package oracle tests.
type semanticPDF struct {
	version string
	objects map[int]semanticObject
	root    int
	info    int
	pages   []semanticPage
}

type semanticObject struct {
	dict   string
	stream []byte
}

type semanticPage struct {
	mediaBox [4]float64
	text     string
	fonts    map[string]int
	images   map[string]int
	annots   []semanticAnnotation
}

type semanticAnnotation struct {
	uri      string
	destPage int
}

//nolint:cyclop,funlen // sequential PDF parse passes share the same byte slice
func parseSemanticPDF(data []byte) (*semanticPDF, error) {
	if len(data) == 0 {
		//nolint:err113 // static parse error message
		return nil, errors.New("empty PDF")
	}

	firstLineEnd := bytes.IndexByte(data, '\n')
	if firstLineEnd <= len("%PDF-") {
		//nolint:err113 // static parse error message
		return nil, errors.New("missing PDF header")
	}

	header := string(bytes.TrimSuffix(data[:firstLineEnd], []byte("\r")))
	if !strings.HasPrefix(header, "%PDF-") {
		//nolint:err113 // dynamic header value in message
		return nil, fmt.Errorf("bad PDF header %q", header)
	}

	trimmed := bytes.TrimRight(data, "\r\n")
	if !bytes.HasSuffix(trimmed, []byte("%%EOF")) {
		//nolint:err113 // static parse error message
		return nil, errors.New("missing EOF marker")
	}

	xrefPos, err := parseStartXref(trimmed)
	if err != nil {
		return nil, err
	}

	xref, trailer, err := parseXrefAndTrailer(data, xrefPos)
	if err != nil {
		return nil, err
	}

	objects, err := parseSemanticObjects(data, xref)
	if err != nil {
		return nil, err
	}

	if trailer.root == 0 {
		//nolint:err113 // static parse error message
		return nil, errors.New("trailer has no /Root reference")
	}

	root, found := objects[trailer.root]
	if !found {
		//nolint:err113 // dynamic object id in message
		return nil, fmt.Errorf("trailer /Root references missing object %d", trailer.root)
	}

	pagesRef, err := requiredRef(root.dict, "/Pages")
	if err != nil {
		return nil, fmt.Errorf("catalog: %w", err)
	}

	pagesTree, ok := objects[pagesRef]
	if !ok {
		//nolint:err113 // dynamic object id in message
		return nil, fmt.Errorf("catalog /Pages references missing object %d", pagesRef)
	}

	pageRefs, err := requiredRefArray(pagesTree.dict, "/Kids")
	if err != nil {
		return nil, fmt.Errorf("pages tree: %w", err)
	}

	count, err := requiredInt(pagesTree.dict, "/Count")
	if err != nil {
		return nil, fmt.Errorf("pages tree: %w", err)
	}

	if count != len(pageRefs) {
		//nolint:err113 // dynamic counts in message
		return nil, fmt.Errorf("pages tree count = %d, kids = %d", count, len(pageRefs))
	}

	doc := &semanticPDF{
		version: strings.TrimPrefix(header, "%PDF-"),
		objects: objects,
		root:    trailer.root,
		info:    trailer.info,
		pages:   make([]semanticPage, 0, len(pageRefs)),
	}

	for _, pageRef := range pageRefs {
		page, err := parseSemanticPage(objects, pageRef)
		if err != nil {
			return nil, fmt.Errorf("page %d: %w", len(doc.pages)+1, err)
		}

		doc.pages = append(doc.pages, page)
	}

	if doc.info != 0 {
		if _, ok := objects[doc.info]; !ok {
			//nolint:err113 // dynamic object id in message
			return nil, fmt.Errorf("trailer /Info references missing object %d", doc.info)
		}
	}

	return doc, nil
}

type semanticTrailer struct {
	size int
	root int
	info int
}

func parseStartXref(data []byte) (int, error) {
	marker := []byte("startxref\n")
	markerPos := bytes.LastIndex(data, marker)

	if markerPos < 0 {
		//nolint:err113 // static parse error message
		return 0, errors.New("missing startxref")
	}

	valueStart := markerPos + len(marker)
	valueEnd := bytes.IndexByte(data[valueStart:], '\n')

	if valueEnd < 0 {
		//nolint:err113 // static parse error message
		return 0, errors.New("startxref has no value")
	}

	value := strings.TrimSpace(string(data[valueStart : valueStart+valueEnd]))
	xrefPos, err := strconv.Atoi(value)

	if err != nil || xrefPos < 0 {
		//nolint:err113 // dynamic value in message
		return 0, fmt.Errorf("invalid startxref value %q", value)
	}

	if xrefPos >= len(data) || !bytes.HasPrefix(data[xrefPos:], []byte("xref\n")) {
		//nolint:err113 // dynamic offset in message
		return 0, fmt.Errorf("startxref %d does not point to an xref table", xrefPos)
	}

	return xrefPos, nil
}

//nolint:cyclop // xref and trailer parse passes share the same byte slice
func parseXrefAndTrailer(data []byte, xrefPos int) (map[int]int, semanticTrailer, error) {
	section := data[xrefPos:]
	trailerPos := bytes.Index(section, []byte("trailer\n"))

	if trailerPos < 0 {
		//nolint:err113 // static parse error message
		return nil, semanticTrailer{}, errors.New("xref table has no trailer")
	}

	lines := strings.Split(strings.TrimRight(string(section[:trailerPos]), "\r\n"), "\n")
	if len(lines) < 3 || lines[0] != "xref" {
		//nolint:err113 // static parse error message
		return nil, semanticTrailer{}, errors.New("malformed xref table")
	}

	sectionHeader := strings.Fields(lines[1])
	if len(sectionHeader) != 2 || sectionHeader[0] != "0" {
		//nolint:err113 // static parse error message
		return nil, semanticTrailer{}, errors.New("malformed xref subsection")
	}

	count, err := strconv.Atoi(sectionHeader[1])
	if err != nil || count < 2 || len(lines) != count+2 {
		//nolint:err113 // static parse error message
		return nil, semanticTrailer{}, errors.New("xref entry count is invalid")
	}

	xref := make(map[int]int, count-1)

	for objectID := 1; objectID < count; objectID++ {
		fields := strings.Fields(lines[objectID+2])
		if len(fields) != 3 || fields[2] != "n" {
			//nolint:err113 // dynamic object id in message
			return nil, semanticTrailer{}, fmt.Errorf("xref entry %d is not an in-use entry", objectID)
		}

		offset, err := strconv.Atoi(fields[0])
		if err != nil || offset <= 0 {
			//nolint:err113 // dynamic object id in message
			return nil, semanticTrailer{}, fmt.Errorf("xref entry %d has invalid offset", objectID)
		}

		xref[objectID] = offset
	}

	trailerStart := xrefPos + trailerPos + len("trailer\n")
	trailerLineEnd := bytes.IndexByte(data[trailerStart:], '\n')

	if trailerLineEnd < 0 {
		//nolint:err113 // static parse error message
		return nil, semanticTrailer{}, errors.New("trailer has no dictionary")
	}

	trailerDict := string(data[trailerStart : trailerStart+trailerLineEnd])
	size, err := requiredInt(trailerDict, "/Size")

	if err != nil || size != count {
		//nolint:err113 // static parse error message
		return nil, semanticTrailer{}, errors.New("trailer /Size does not match xref")
	}

	root, err := requiredRef(trailerDict, "/Root")
	if err != nil {
		return nil, semanticTrailer{}, fmt.Errorf("trailer: %w", err)
	}

	info, _ := optionalRef(trailerDict, "/Info")

	return xref, semanticTrailer{size: size, root: root, info: info}, nil
}

//nolint:cyclop // per-object parse passes share the same byte slice
func parseSemanticObjects(data []byte, xref map[int]int) (map[int]semanticObject, error) {
	const errorPreviewBytes = 24

	objects := make(map[int]semanticObject, len(xref))

	for objectID, offset := range xref {
		if offset >= len(data) {
			//nolint:err113 // dynamic object id in message
			return nil, fmt.Errorf("xref object %d starts past EOF", objectID)
		}

		objectData := data[offset:]
		headerMatch := semanticObjectHeaderRE.FindSubmatchIndex(objectData)

		if headerMatch == nil {
			//nolint:err113 // dynamic object id in message
			return nil, fmt.Errorf("xref object %d does not point to an object header", objectID)
		}

		actualID, err := strconv.Atoi(string(objectData[headerMatch[2]:headerMatch[3]]))
		if err != nil || actualID != objectID {
			//nolint:err113 // dynamic object id in message
			return nil, fmt.Errorf("xref object %d points to object %q",
				objectID, objectData[:min(len(objectData), errorPreviewBytes)])
		}

		body := objectData[headerMatch[1]:]
		streamMark := bytes.Index(body, []byte("\nstream\n"))

		if streamMark < 0 {
			end := bytes.Index(body, []byte("\nendobj\n"))
			if end < 0 {
				//nolint:err113 // dynamic object id in message
				return nil, fmt.Errorf("object %d has no endobj", objectID)
			}

			objects[objectID] = semanticObject{dict: string(body[:end]), stream: nil}

			continue
		}

		dict := string(body[:streamMark])
		length, err := requiredInt(dict, "/Length")

		if err != nil || length < 0 {
			//nolint:err113 // dynamic object id in message
			return nil, fmt.Errorf("object %d has invalid stream length", objectID)
		}

		streamStart := streamMark + len("\nstream\n")
		streamEnd := streamStart + length

		if streamEnd > len(body) || !bytes.HasPrefix(body[streamEnd:], []byte("\nendstream\nendobj\n")) {
			//nolint:err113 // dynamic object id in message
			return nil, fmt.Errorf("object %d has a truncated stream", objectID)
		}

		objects[objectID] = semanticObject{
			dict:   dict,
			stream: append([]byte(nil), body[streamStart:streamEnd]...),
		}
	}

	return objects, nil
}

//nolint:cyclop // contents, resources, and annots parse the same page object
func parseSemanticPage(objects map[int]semanticObject, pageRef int) (semanticPage, error) {
	pageObject, found := objects[pageRef]
	if !found {
		//nolint:err113 // dynamic object id in message
		return semanticPage{}, fmt.Errorf("page object %d is missing", pageRef)
	}

	mediaBox, err := requiredNumberArray(pageObject.dict, "/MediaBox", 4) //nolint:mnd // MediaBox has four values
	if err != nil {
		return semanticPage{}, err
	}

	contentsRef, err := requiredRef(pageObject.dict, "/Contents")
	if err != nil {
		return semanticPage{}, fmt.Errorf("contents: %w", err)
	}

	contents, ok := objects[contentsRef]
	if !ok {
		//nolint:err113 // dynamic object id in message
		return semanticPage{}, fmt.Errorf("contents object %d is missing", contentsRef)
	}

	stream, err := decodeSemanticStream(contents)
	if err != nil {
		return semanticPage{}, fmt.Errorf("contents stream: %w", err)
	}

	resources, err := requiredDictionary(pageObject.dict, "/Resources")
	if err != nil {
		return semanticPage{}, err
	}

	fonts, err := resourceRefs(resources, "/Font")
	if err != nil {
		return semanticPage{}, fmt.Errorf("fonts: %w", err)
	}

	images, err := resourceRefs(resources, "/XObject")
	if err != nil {
		return semanticPage{}, fmt.Errorf("images: %w", err)
	}

	for _, ref := range append(refValues(fonts), refValues(images)...) {
		if _, ok := objects[ref]; !ok {
			//nolint:err113 // dynamic object id in message
			return semanticPage{}, fmt.Errorf("resource object %d is missing", ref)
		}
	}

	annots, err := parseAnnotations(objects, pageObject.dict)
	if err != nil {
		return semanticPage{}, err
	}

	return semanticPage{
		mediaBox: [4]float64{mediaBox[0], mediaBox[1], mediaBox[2], mediaBox[3]},
		text:     extractSemanticText(stream),
		fonts:    fonts,
		images:   images,
		annots:   annots,
	}, nil
}

func decodeSemanticStream(object semanticObject) ([]byte, error) {
	if !strings.Contains(object.dict, "/Filter /FlateDecode") {
		return object.stream, nil
	}

	reader, err := zlib.NewReader(bytes.NewReader(object.stream))
	if err != nil {
		return nil, fmt.Errorf("flate reader: %w", err)
	}

	decoded, readErr := io.ReadAll(reader)
	closeErr := reader.Close()

	if readErr != nil {
		return nil, fmt.Errorf("flate stream: %w", readErr)
	}

	if closeErr != nil {
		return nil, fmt.Errorf("close flate stream: %w", closeErr)
	}

	return decoded, nil
}

func extractSemanticText(stream []byte) string {
	var text strings.Builder

	streamText := string(stream)

	for _, match := range semanticLiteralRE.FindAllStringSubmatch(streamText, -1) {
		text.WriteString(decodePDFLiteral(match[1]))
	}

	for _, match := range semanticHexRE.FindAllStringSubmatch(streamText, -1) {
		text.WriteString(decodePDFHex(match[1]))
	}

	return text.String()
}

//nolint:cyclop // escape sequence decoding shares the literal bytes
func decodePDFLiteral(value string) string {
	value = strings.TrimSuffix(strings.TrimPrefix(value, "("), ")")
	decoded := make([]byte, 0, len(value))

	for pos := 0; pos < len(value); pos++ {
		if value[pos] != '\\' || pos+1 >= len(value) {
			decoded = append(decoded, value[pos])

			continue
		}

		pos++
		switch value[pos] {
		case 'n':
			decoded = append(decoded, '\n')
		case 'r':
			decoded = append(decoded, '\r')
		case 't':
			decoded = append(decoded, '\t')
		case 'b':
			decoded = append(decoded, '\b')
		case 'f':
			decoded = append(decoded, '\f')
		case '\n':
		case '\r':
			if pos+1 < len(value) && value[pos+1] == '\n' {
				pos++
			}
		case '(', ')', '\\':
			decoded = append(decoded, value[pos])
		default:
			if value[pos] >= '0' && value[pos] <= '7' {
				end := pos + 1
				for end < len(value) && end < pos+3 && value[end] >= '0' && value[end] <= '7' {
					end++
				}

				parsed, err := strconv.ParseUint(value[pos:end], 8, 8)
				if err == nil {
					decoded = append(decoded, byte(parsed))
					pos = end - 1
				}
			} else {
				decoded = append(decoded, value[pos])
			}
		}
	}

	return string(decoded)
}

func decodePDFHex(value string) string {
	if len(value)%2 != 0 {
		value += "0"
	}

	decoded := make([]byte, 0, len(value)/2) //nolint:mnd // two hex digits per byte

	for pos := 0; pos < len(value); pos += 2 {
		parsed, err := strconv.ParseUint(value[pos:pos+2], 16, 8)
		if err != nil {
			return ""
		}

		decoded = append(decoded, byte(parsed))
	}

	return string(decoded)
}

func parseAnnotations(objects map[int]semanticObject, pageDict string) ([]semanticAnnotation, error) {
	refs, _ := optionalRefArray(pageDict, "/Annots")

	annotations := make([]semanticAnnotation, 0, len(refs))

	for _, ref := range refs {
		object, ok := objects[ref]
		if !ok {
			//nolint:err113 // dynamic object id in message
			return nil, fmt.Errorf("annotation object %d is missing", ref)
		}

		annotation := semanticAnnotation{} //nolint:exhaustruct // parser fills remaining fields below
		if uri, ok := optionalLiteral(object.dict, "/URI"); ok {
			annotation.uri = uri
		}

		if dest, ok := optionalDestinationRef(object.dict); ok {
			annotation.destPage = dest
		}

		if annotation.uri == "" && annotation.destPage == 0 {
			//nolint:err113 // dynamic object id in message
			return nil, fmt.Errorf("annotation object %d has no URI or destination", ref)
		}

		annotations = append(annotations, annotation)
	}

	return annotations, nil
}

func resourceRefs(dict, key string) (map[string]int, error) {
	resourceDict, err := requiredDictionary(dict, key)
	if err != nil {
		if strings.Contains(err.Error(), "missing") {
			return map[string]int{}, nil
		}

		return nil, err
	}

	refs := make(map[string]int)

	for _, match := range semanticResourceRE.FindAllStringSubmatch(resourceDict, -1) {
		ref, err := strconv.Atoi(match[2])
		if err != nil {
			//nolint:err113 // dynamic resource name in message
			return nil, fmt.Errorf("resource %q has invalid reference", match[1])
		}

		refs[match[1]] = ref
	}

	return refs, nil
}

func requiredDictionary(dict, key string) (string, error) {
	marker := key + " <<"
	start := strings.Index(dict, marker)

	if start < 0 {
		//nolint:err113 // dynamic key in message
		return "", fmt.Errorf("dictionary %s is missing", key)
	}

	valueStart := start + len(key) + 1
	depth := 0

	for pos := valueStart; pos+1 < len(dict); pos++ {
		switch dict[pos : pos+2] {
		case "<<":
			depth++
			pos++
		case ">>":
			depth--
			if depth == 0 {
				return dict[valueStart+2 : pos], nil
			}

			pos++
		}
	}

	//nolint:err113 // dynamic key in message
	return "", fmt.Errorf("dictionary %s is malformed", key)
}

func requiredRef(dict, key string) (int, error) {
	ref, ok := optionalRef(dict, key)
	if !ok {
		//nolint:err113 // dynamic key in message
		return 0, fmt.Errorf("missing %s reference", key)
	}

	return ref, nil
}

func optionalRef(dict, key string) (int, bool) {
	pattern := regexp.MustCompile(regexp.QuoteMeta(key) + `\s+(\d+)\s+0\s+R`)
	match := pattern.FindStringSubmatch(dict)

	if match == nil {
		return 0, false
	}

	ref, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, false
	}

	return ref, true
}

func requiredRefArray(dict, key string) ([]int, error) {
	refs, ok := optionalRefArray(dict, key)
	if !ok {
		//nolint:err113 // dynamic key in message
		return nil, fmt.Errorf("missing %s array", key)
	}

	return refs, nil
}

func optionalRefArray(dict, key string) ([]int, bool) {
	pattern := regexp.MustCompile(regexp.QuoteMeta(key) + `\s*\[([^\]]*)\]`)
	match := pattern.FindStringSubmatch(dict)

	if match == nil {
		return nil, false
	}

	refs := make([]int, 0)

	for _, refMatch := range semanticRefRE.FindAllStringSubmatch(match[1], -1) {
		ref, err := strconv.Atoi(refMatch[1])
		if err != nil {
			return nil, false
		}

		refs = append(refs, ref)
	}

	return refs, true
}

func requiredNumberArray(dict, key string, want int) ([]float64, error) {
	pattern := regexp.MustCompile(regexp.QuoteMeta(key) + `\s*\[([^\]]*)\]`)
	match := pattern.FindStringSubmatch(dict)

	if match == nil {
		//nolint:err113 // dynamic key in message
		return nil, fmt.Errorf("missing %s array", key)
	}

	values := make([]float64, 0, want)

	for _, token := range semanticNumberRE.FindAllString(match[1], -1) {
		value, err := strconv.ParseFloat(token, 64)
		if err != nil {
			//nolint:err113 // dynamic key and token in message
			return nil, fmt.Errorf("%s contains invalid number %q", key, token)
		}

		values = append(values, value)
	}

	if len(values) != want {
		//nolint:err113 // dynamic counts in message
		return nil, fmt.Errorf("%s has %d values, want %d", key, len(values), want)
	}

	return values, nil
}

func requiredInt(dict, key string) (int, error) {
	pattern := regexp.MustCompile(regexp.QuoteMeta(key) + `\s+(\d+)`)
	match := pattern.FindStringSubmatch(dict)

	if match == nil {
		//nolint:err113 // dynamic key in message
		return 0, fmt.Errorf("missing %s integer", key)
	}

	value, err := strconv.Atoi(match[1])
	if err != nil {
		//nolint:err113 // dynamic key in message
		return 0, fmt.Errorf("%s has invalid integer", key)
	}

	return value, nil
}

func optionalLiteral(dict, key string) (string, bool) {
	pattern := regexp.MustCompile(regexp.QuoteMeta(key) + `\s+(\((?:\\.|[^\\)])*\))`)
	match := pattern.FindStringSubmatch(dict)

	if match == nil {
		return "", false
	}

	return decodePDFLiteral(match[1]), true
}

func optionalDestinationRef(dict string) (int, bool) {
	pattern := regexp.MustCompile(`/Dest\s*\[\s*(\d+)\s+0\s+R`)
	match := pattern.FindStringSubmatch(dict)

	if match == nil {
		return 0, false
	}

	ref, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, false
	}

	return ref, true
}

func refValues(refs map[string]int) []int {
	values := make([]int, 0, len(refs))
	for _, ref := range refs {
		values = append(values, ref)
	}

	return values
}
