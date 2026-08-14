//nolint:testpackage,cyclop,varnamelen,wsl // test-only parser
package pdf

import (
	"bytes"
	"strings"
	"testing"
)

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

//nolint:funlen // oracle test with detailed assertions
func TestSemanticPDF17Oracle(t *testing.T) {
	t.Parallel()

	data := buildSemanticPDF17(t)
	doc, err := parseSemanticPDF(data)

	if err != nil {
		t.Fatalf("parse generated PDF 1.7: %v", err)
	}

	if doc.version != "1.7" {
		t.Errorf("PDF version = %q, want %q", doc.version, "1.7")
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

	if got, want := doc.pages[0].annots[0].uri, "https://example.com/report17"; got != want {
		t.Errorf("external link URI = %q, want %q", got, want)
	}

	if got, want := doc.pages[0].annots[1].destPage, pageObjectRef(t, doc, 1); got != want {
		t.Errorf("internal link destination = %d, want page object %d", got, want)
	}

	// Verify public ParseSemantic API
	semDoc, err := ParseSemantic(data)
	if err != nil {
		t.Fatalf("ParseSemantic: %v", err)
	}

	if semDoc.Version != "1.7" {
		t.Errorf("semDoc.Version = %q, want %q", semDoc.Version, "1.7")
	}

	if semDoc.PageCount() != 2 {
		t.Errorf("semDoc.PageCount() = %d, want 2", semDoc.PageCount())
	}

	if !semDoc.HasURI() {
		t.Error("semDoc.HasURI() = false, want true")
	}

	if !semDoc.HasImageXObject() {
		t.Error("semDoc.HasImageXObject() = false, want true")
	}

	if !semDoc.HasInternalDest() {
		t.Error("semDoc.HasInternalDest() = false, want true")
	}
}

func buildSemanticPDF17(t *testing.T) []byte {
	t.Helper()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatalf("DefaultFont: %v", err)
	}

	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17}) //nolint:exhaustruct // test policy
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF17): %v", err)
	}

	doc.SetInfo("Title", "Semantic oracle 1.7")

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

	first.AddLinkURI([4]float64{40, 700, 160, 720}, "https://example.com/report17")
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
		t.Fatalf("write semantic PDF 1.7: %v", err)
	}

	return out.Bytes()
}
