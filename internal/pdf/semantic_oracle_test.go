//nolint:testpackage,cyclop,err113,exhaustruct,varnamelen,wsl,funlen,perfsprint,nlreturn // test-only parser
package pdf

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var (
	semanticObjectHeaderRE = regexp.MustCompile(`^(\d+) 0 obj\n`)
	semanticRefRE          = regexp.MustCompile(`(\d+) 0 R`)
	semanticResourceRE     = regexp.MustCompile(`/([A-Za-z][A-Za-z0-9_]*)\s+(\d+)\s+0\s+R`)
	semanticNumberRE       = regexp.MustCompile(`[-+]?\d+(?:\.\d+)?`)
	semanticLiteralRE      = regexp.MustCompile(`(?s)(\((?:\\.|[^\\)])*\))\s*Tj`)
	semanticHexRE          = regexp.MustCompile(`(?s)<([0-9A-Fa-f]*)>\s*Tj`)
)

// semanticPDF is deliberately smaller than a general PDF object model. It
// validates the structures emitted by this package and exposes only the
// semantic facts that the regression test needs.
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

func TestSemanticPDFOracle(t *testing.T) {
	t.Parallel()

	data := buildSemanticPDF(t)
	doc, err := parseSemanticPDF(data)
	if err != nil {
		t.Fatalf("parse generated PDF: %v", err)
	}

	if doc.version != Version {
		t.Errorf("PDF version = %q, want %q", doc.version, Version)
	}

	if len(doc.pages) != 2 {
		t.Fatalf("page count = %d, want 2", len(doc.pages))
	}

	assertFloatArray(t, doc.pages[0].mediaBox, [4]float64{0, 0, 612, 792})
	assertFloatArray(t, doc.pages[1].mediaBox, [4]float64{0, 0, 300, 400})

	if got, want := doc.pages[0].text, "alpha firstbeta second"; got != want {
		t.Errorf("page 1 extracted text = %q, want %q", got, want)
	}

	if got, want := doc.pages[1].text, "second page"; got != want {
		t.Errorf("page 2 extracted text = %q, want %q", got, want)
	}

	if got := doc.pages[0].fonts["F1"]; got == 0 {
		t.Fatal("page 1 is missing its F1 font resource")
	} else if !strings.Contains(doc.objects[got].dict, "/Subtype /TrueType") {
		t.Errorf("F1 resource object = %q, want an embedded TrueType font", doc.objects[got].dict)
	}

	if imageRef := doc.pages[0].images["Im1"]; imageRef == 0 {
		t.Fatal("page 1 is missing its Im1 image resource")
	} else if !strings.Contains(doc.objects[imageRef].dict, "/Subtype /Image") {
		t.Errorf("Im1 resource object = %q, want an image XObject", doc.objects[imageRef].dict)
	}

	if len(doc.pages[0].annots) != 2 {
		t.Fatalf("page 1 annotations = %d, want 2", len(doc.pages[0].annots))
	}

	if got, want := doc.pages[0].annots[0].uri, "https://example.com/report"; got != want {
		t.Errorf("external link URI = %q, want %q", got, want)
	}

	if got, want := doc.pages[0].annots[1].destPage, pageObjectRef(t, doc, 1); got != want {
		t.Errorf("internal link destination = %d, want page object %d", got, want)
	}

	assertInfoTitle(t, doc, "Semantic oracle")
	assertOutline(t, doc, "First page", pageObjectRef(t, doc, 0))
}

func TestSemanticPDFOracleRejectsTruncatedOutput(t *testing.T) {
	t.Parallel()

	data := buildSemanticPDF(t)
	startxref := bytes.Index(data, []byte("startxref"))
	if startxref < 0 {
		t.Fatal("generated PDF has no startxref marker")
	}

	truncated := append([]byte(nil), data[:startxref]...)
	truncated = append(truncated, []byte("\n%%EOF\n")...)

	if _, err := parseSemanticPDF(truncated); err == nil {
		t.Fatal("oracle accepted a truncated PDF with a header and EOF marker")
	}

	if _, err := parseSemanticPDF([]byte("%PDF-1.4\n%%EOF\n")); err == nil {
		t.Fatal("oracle accepted a header-only PDF with an EOF marker")
	}
}

func buildSemanticPDF(t *testing.T) []byte {
	t.Helper()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatalf("DefaultFont: %v", err)
	}

	doc := NewDocument()
	doc.SetInfo("Title", "Semantic oracle")

	first := doc.AddPage(612, 792)
	firstContent := first.Content()
	firstContent.UseEmbeddedFont("F1", fnt)
	firstContent.BeginText()
	firstContent.SetFont("F1", 12)
	firstContent.TextAt(40, 740)
	firstContent.TextShow("alpha first")
	firstContent.TextAt(40, 720)
	firstContent.TextShow("beta second")
	firstContent.EndText()
	if err := firstContent.AddPNGImage("Im1", 40, 620, 32, 16, makePNG(t, false)); err != nil {
		t.Fatalf("AddPNGImage: %v", err)
	}
	first.AddLinkURI([4]float64{40, 700, 160, 720}, "https://example.com/report")
	first.AddLinkDest([4]float64{40, 670, 160, 690}, 1, 20, 760)

	second := doc.AddPage(300, 400)
	secondContent := second.Content()
	secondContent.UseEmbeddedFont("F1", fnt)
	secondContent.BeginText()
	secondContent.SetFont("F1", 12)
	secondContent.TextAt(20, 360)
	secondContent.TextShow("second page")
	secondContent.EndText()

	doc.SetOutline(&Outline{ //nolint:exhaustruct // test fixture intentionally omits the root title.
		Children: []*Outline{{Title: "First page", PageRef: doc.PageRef(0), X: 40, Y: 740}},
	})

	var out bytes.Buffer
	if err := doc.Write(&out); err != nil {
		t.Fatalf("write semantic PDF: %v", err)
	}

	return out.Bytes()
}

func parseSemanticPDF(data []byte) (*semanticPDF, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty PDF")
	}

	firstLineEnd := bytes.IndexByte(data, '\n')
	if firstLineEnd <= len("%PDF-") {
		return nil, fmt.Errorf("missing PDF header")
	}

	header := string(bytes.TrimSuffix(data[:firstLineEnd], []byte("\r")))
	if !strings.HasPrefix(header, "%PDF-") {
		return nil, fmt.Errorf("bad PDF header %q", header)
	}

	trimmed := bytes.TrimRight(data, "\r\n")
	if !bytes.HasSuffix(trimmed, []byte("%%EOF")) {
		return nil, fmt.Errorf("missing EOF marker")
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
		return nil, fmt.Errorf("trailer has no /Root reference")
	}

	root, ok := objects[trailer.root]
	if !ok {
		return nil, fmt.Errorf("trailer /Root references missing object %d", trailer.root)
	}

	pagesRef, err := requiredRef(root.dict, "/Pages")
	if err != nil {
		return nil, fmt.Errorf("catalog: %w", err)
	}

	pagesTree, ok := objects[pagesRef]
	if !ok {
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
		return nil, fmt.Errorf("pages tree count = %d, kids = %d", count, len(pageRefs))
	}

	doc := &semanticPDF{ //nolint:exhaustruct // parser fills fields below.
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
		return 0, fmt.Errorf("missing startxref")
	}

	valueStart := markerPos + len(marker)
	valueEnd := bytes.IndexByte(data[valueStart:], '\n')
	if valueEnd < 0 {
		return 0, fmt.Errorf("startxref has no value")
	}

	value := strings.TrimSpace(string(data[valueStart : valueStart+valueEnd]))
	xrefPos, err := strconv.Atoi(value)
	if err != nil || xrefPos < 0 {
		return 0, fmt.Errorf("invalid startxref value %q", value)
	}

	if xrefPos >= len(data) || !bytes.HasPrefix(data[xrefPos:], []byte("xref\n")) {
		return 0, fmt.Errorf("startxref %d does not point to an xref table", xrefPos)
	}

	return xrefPos, nil
}

func parseXrefAndTrailer(data []byte, xrefPos int) (map[int]int, semanticTrailer, error) {
	section := data[xrefPos:]
	trailerPos := bytes.Index(section, []byte("trailer\n"))
	if trailerPos < 0 {
		return nil, semanticTrailer{}, fmt.Errorf("xref table has no trailer")
	}

	lines := strings.Split(strings.TrimRight(string(section[:trailerPos]), "\r\n"), "\n")
	if len(lines) < 3 || lines[0] != "xref" {
		return nil, semanticTrailer{}, fmt.Errorf("malformed xref table")
	}

	sectionHeader := strings.Fields(lines[1])
	if len(sectionHeader) != 2 || sectionHeader[0] != "0" {
		return nil, semanticTrailer{}, fmt.Errorf("malformed xref subsection")
	}

	count, err := strconv.Atoi(sectionHeader[1])
	if err != nil || count < 2 || len(lines) != count+2 {
		return nil, semanticTrailer{}, fmt.Errorf("xref entry count is invalid")
	}

	xref := make(map[int]int, count-1)
	for id := 1; id < count; id++ {
		fields := strings.Fields(lines[id+2])
		if len(fields) != 3 || fields[2] != "n" {
			return nil, semanticTrailer{}, fmt.Errorf("xref entry %d is not an in-use entry", id)
		}

		offset, err := strconv.Atoi(fields[0])
		if err != nil || offset <= 0 {
			return nil, semanticTrailer{}, fmt.Errorf("xref entry %d has invalid offset", id)
		}

		xref[id] = offset
	}

	trailerStart := xrefPos + trailerPos + len("trailer\n")
	trailerLineEnd := bytes.IndexByte(data[trailerStart:], '\n')
	if trailerLineEnd < 0 {
		return nil, semanticTrailer{}, fmt.Errorf("trailer has no dictionary")
	}

	trailerDict := string(data[trailerStart : trailerStart+trailerLineEnd])
	size, err := requiredInt(trailerDict, "/Size")
	if err != nil || size != count {
		return nil, semanticTrailer{}, fmt.Errorf("trailer /Size does not match xref")
	}

	root, err := requiredRef(trailerDict, "/Root")
	if err != nil {
		return nil, semanticTrailer{}, fmt.Errorf("trailer: %w", err)
	}

	info, _ := optionalRef(trailerDict, "/Info")

	return xref, semanticTrailer{size: size, root: root, info: info}, nil
}

func parseSemanticObjects(data []byte, xref map[int]int) (map[int]semanticObject, error) {
	objects := make(map[int]semanticObject, len(xref))

	for id, offset := range xref {
		if offset >= len(data) {
			return nil, fmt.Errorf("xref object %d starts past EOF", id)
		}

		objectData := data[offset:]
		headerMatch := semanticObjectHeaderRE.FindSubmatchIndex(objectData)
		if headerMatch == nil {
			return nil, fmt.Errorf("xref object %d does not point to an object header", id)
		}

		actualID, err := strconv.Atoi(string(objectData[headerMatch[2]:headerMatch[3]]))
		if err != nil || actualID != id {
			return nil, fmt.Errorf("xref object %d points to object %q", id, objectData[:min(len(objectData), 24)])
		}

		body := objectData[headerMatch[1]:]
		streamMark := bytes.Index(body, []byte("\nstream\n"))
		if streamMark < 0 {
			end := bytes.Index(body, []byte("\nendobj\n"))
			if end < 0 {
				return nil, fmt.Errorf("object %d has no endobj", id)
			}

			objects[id] = semanticObject{dict: string(body[:end]), stream: nil}
			continue
		}

		dict := string(body[:streamMark])
		length, err := requiredInt(dict, "/Length")
		if err != nil || length < 0 {
			return nil, fmt.Errorf("object %d has invalid stream length", id)
		}

		streamStart := streamMark + len("\nstream\n")
		streamEnd := streamStart + length
		if streamEnd > len(body) || !bytes.HasPrefix(body[streamEnd:], []byte("\nendstream\nendobj\n")) {
			return nil, fmt.Errorf("object %d has a truncated stream", id)
		}

		objects[id] = semanticObject{
			dict:   dict,
			stream: append([]byte(nil), body[streamStart:streamEnd]...),
		}
	}

	return objects, nil
}

func parseSemanticPage(objects map[int]semanticObject, pageRef int) (semanticPage, error) {
	pageObject, ok := objects[pageRef]
	if !ok {
		return semanticPage{}, fmt.Errorf("page object %d is missing", pageRef)
	}

	mediaBox, err := requiredNumberArray(pageObject.dict, "/MediaBox", 4)
	if err != nil {
		return semanticPage{}, err
	}

	contentsRef, err := requiredRef(pageObject.dict, "/Contents")
	if err != nil {
		return semanticPage{}, fmt.Errorf("contents: %w", err)
	}

	contents, ok := objects[contentsRef]
	if !ok {
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

	decoded := make([]byte, 0, len(value)/2)
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
			return nil, fmt.Errorf("annotation object %d is missing", ref)
		}

		annotation := semanticAnnotation{}
		if uri, ok := optionalLiteral(object.dict, "/URI"); ok {
			annotation.uri = uri
		}

		if dest, ok := optionalDestinationRef(object.dict); ok {
			annotation.destPage = dest
		}

		if annotation.uri == "" && annotation.destPage == 0 {
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

	return "", fmt.Errorf("dictionary %s is malformed", key)
}

func requiredRef(dict, key string) (int, error) {
	ref, ok := optionalRef(dict, key)
	if !ok {
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
		return nil, fmt.Errorf("missing %s array", key)
	}

	values := make([]float64, 0, want)
	for _, token := range semanticNumberRE.FindAllString(match[1], -1) {
		value, err := strconv.ParseFloat(token, 64)
		if err != nil {
			return nil, fmt.Errorf("%s contains invalid number %q", key, token)
		}

		values = append(values, value)
	}

	if len(values) != want {
		return nil, fmt.Errorf("%s has %d values, want %d", key, len(values), want)
	}

	return values, nil
}

func requiredInt(dict, key string) (int, error) {
	pattern := regexp.MustCompile(regexp.QuoteMeta(key) + `\s+(\d+)`)
	match := pattern.FindStringSubmatch(dict)
	if match == nil {
		return 0, fmt.Errorf("missing %s integer", key)
	}

	value, err := strconv.Atoi(match[1])
	if err != nil {
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

func pageObjectRef(t *testing.T, doc *semanticPDF, index int) int {
	t.Helper()

	pagesObject, ok := doc.objects[doc.root]
	if !ok {
		t.Fatalf("catalog object %d is missing", doc.root)
	}

	pagesRef, err := requiredRef(pagesObject.dict, "/Pages")
	if err != nil {
		t.Fatalf("catalog pages ref: %v", err)
	}

	pageTree, ok := doc.objects[pagesRef]
	if !ok {
		t.Fatalf("pages object %d is missing", pagesRef)
	}

	refs, err := requiredRefArray(pageTree.dict, "/Kids")
	if err != nil {
		t.Fatalf("page refs: %v", err)
	}

	if index < 0 || index >= len(refs) {
		t.Fatalf("page index %d is out of range", index)
	}

	return refs[index]
}

func assertInfoTitle(t *testing.T, doc *semanticPDF, want string) {
	t.Helper()

	info, ok := doc.objects[doc.info]
	if !ok {
		t.Fatalf("info object %d is missing", doc.info)
	}

	if got, ok := optionalLiteral(info.dict, "/Title"); !ok || got != want {
		t.Errorf("info title = %q, want %q", got, want)
	}
}

func assertOutline(t *testing.T, doc *semanticPDF, wantTitle string, wantPage int) {
	t.Helper()

	catalog := doc.objects[doc.root]
	outlinesRef, err := requiredRef(catalog.dict, "/Outlines")
	if err != nil {
		t.Fatalf("catalog outlines: %v", err)
	}

	outlines := doc.objects[outlinesRef]
	firstRef, err := requiredRef(outlines.dict, "/First")
	if err != nil {
		t.Fatalf("outline root: %v", err)
	}

	first := doc.objects[firstRef]
	if got, ok := optionalLiteral(first.dict, "/Title"); !ok || got != wantTitle {
		t.Errorf("outline title = %q, want %q", got, wantTitle)
	}

	if got, ok := optionalDestinationRef(first.dict); !ok || got != wantPage {
		t.Errorf("outline destination = %d, want page object %d", got, wantPage)
	}
}

func assertFloatArray(t *testing.T, got, want [4]float64) {
	t.Helper()

	for index := range got {
		if got[index] != want[index] {
			t.Errorf("MediaBox[%d] = %v, want %v", index, got[index], want[index])
		}
	}
}
