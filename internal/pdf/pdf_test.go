//nolint:testpackage,exhaustruct // tests reach into unexported state
package pdf

import (
	"bytes"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// kidsRe matches the pages tree /Kids array.
var kidsRe = regexp.MustCompile(`/Kids \[([^\]]+)\]`)

func fixedDoc(t *testing.T) *Document {
	t.Helper()

	d := NewDocument()
	d.SetCreationTime(time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC))
	d.SetInfo("Title", "Smoke")

	return d
}

func writePDF(t *testing.T, d *Document) []byte {
	t.Helper()

	var buf bytes.Buffer
	if err := d.Write(&buf); err != nil {
		t.Fatalf("Write: %v", err)
	}

	return buf.Bytes()
}

func TestWriteHeaderAndTrailer(t *testing.T) {
	t.Parallel()
	d := fixedDoc(t)
	d.AddPage(595.276, 841.89) // A4 portrait
	out := writePDF(t, d)
	str := string(out)

	for _, want := range []string{
		"%PDF-1.4",
		"%\xe2\xe3\xcf\xd3",
		"xref",
		"trailer",
		"/Type /Catalog",
		"/Type /Pages",
		"/Type /Page",
		"/MediaBox [0 0 595.276 841.89]",
		"startxref",
		"%%EOF",
	} {
		if !strings.Contains(str, want) {
			t.Errorf("output missing %q", want)
		}
	}

	if !strings.HasPrefix(str, "%PDF-") {
		t.Errorf("output must start with PDF header")
	}
}

func TestDeterministicOutput(t *testing.T) {
	t.Parallel()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	build := func() []byte {
		data := fixedDoc(t)
		p := data.AddPage(200, 200)
		cur := p.Content()
		cur.UseEmbeddedFont("F1", fnt)
		cur.BeginText()
		cur.SetFont("F1", 12)
		cur.TextAt(10, 20)
		cur.TextShow("hello")
		cur.EndText()
		data.SetOutline(&Outline{ //nolint:exhaustruct // intentional zero-value fields
			Title:    "root",
			Children: []*Outline{{Title: "child"}},
		})

		return writePDF(t, data)
	}

	a, b := build(), build()
	if !bytes.Equal(a, b) {
		t.Errorf("output not deterministic")
	}
}

func TestXrefOffsets(t *testing.T) {
	t.Parallel()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	d := fixedDoc(t)
	p := d.AddPage(100, 100)
	p.Content().UseEmbeddedFont("F1", fnt)

	out := writePDF(t, d)

	// every n entry must point at the start of "N 0 obj"
	lines := strings.Split(string(out), "\n")
	xrefIdx := findLine(lines, "xref")

	if xrefIdx < 0 {
		t.Fatal("no xref")
	}

	startxref := -1

	for i := xrefIdx + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "startxref") {
			startxref = i

			break
		}
	}

	if startxref < 0 {
		t.Fatal("no startxref")
	}

	offsets := parseXrefEntries(t, lines, xrefIdx, startxref)

	for obj, off := range offsets {
		want := strconv.Itoa(obj) + " 0 obj"
		if !bytes.HasPrefix(out[off:], []byte(want)) {
			t.Errorf("object %d offset %d does not start with %q", obj, off, want)
		}
	}
}

// findLine returns the index of the first exact line match, or -1.
func findLine(lines []string, want string) int {
	for i, l := range lines {
		if l == want {
			return i
		}
	}

	return -1
}

// parseXrefEntries extracts the object→offset table from the xref section.
func parseXrefEntries(t *testing.T, lines []string, xrefIdx, startxref int) map[int]int {
	t.Helper()

	offsets := map[int]int{}

	for idx := xrefIdx + 3; idx < startxref-2; idx++ {
		fields := strings.Fields(lines[idx])
		if len(fields) != 3 || fields[1] != "00000" {
			continue
		}

		obj := idx - xrefIdx - 2

		off, err := strconv.Atoi(fields[0])
		if err != nil {
			t.Fatalf("bad offset %q", fields[0])
		}

		offsets[obj] = off
	}

	return offsets
}

func TestEmptyDocFails(t *testing.T) {
	t.Parallel()

	d := NewDocument()

	var buf bytes.Buffer

	if err := d.Write(&buf); err == nil {
		t.Fatal("expected error for empty document")
	}
}

func TestContentOperators(t *testing.T) {
	t.Parallel()
	data := fixedDoc(t)
	p := data.AddPage(300, 300)
	cur := p.Content()
	cur.Save()
	cur.SetFillColor(0.5, 0.25, 0.75)
	cur.Rect(10, 10, 50, 60)
	cur.Fill()
	cur.SetLineWidth(1.5)
	cur.MoveTo(0, 0)
	cur.LineTo(100, 100)
	cur.Stroke()
	cur.Restore()

	out := writePDF(t, data)
	str := string(out)
	// content stream is flate-compressed, so check compressed-free page by
	// disabling compression instead.
	d2 := fixedDoc(t)
	p2 := d2.AddPage(300, 300)
	p2.Content().Rect(1, 2, 3, 4)
	d2.SetCompression(false)

	out2 := writePDF(t, d2)
	if !strings.Contains(string(out2), "1 2 3 4 re") {
		t.Errorf("rect operator missing in uncompressed stream")
	}

	_ = str
}

func TestTextStream(t *testing.T) {
	t.Parallel()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	data := fixedDoc(t)
	data.SetCompression(false)
	p := data.AddPage(300, 300)
	cur := p.Content()
	cur.UseEmbeddedFont("F1", fnt)
	cur.BeginText()
	cur.SetFont("F1", 14)
	cur.TextLeading(18)
	cur.TextAt(20, 280)
	cur.TextShow("line one")
	cur.TextNextLine()
	cur.TextShow("line two")
	cur.EndText()

	out := string(writePDF(t, data))
	for _, want := range []string{
		"/F1 14 Tf",
		"20 280 Td",
		"(line one) Tj",
		"T*",
		"(line two) Tj",
		"/Type /Font /Subtype /TrueType",
		"/BaseFont /",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestImageXObject(t *testing.T) {
	t.Parallel()
	data := fixedDoc(t)
	data.SetCompression(false)

	p := data.AddPage(200, 200)
	if err := p.Content().AddPNGImage("Im1", 10, 10, 100, 50, makePNG(t, false)); err != nil {
		t.Fatal(err)
	}

	out := string(writePDF(t, data))
	for _, want := range []string{
		"/Im1 Do",
		"/Subtype /Image",
		"/Width 4 /Height 2",
		"/ColorSpace /DeviceRGB",
		"/XObject << /Im1",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestLinkAnnotations(t *testing.T) {
	t.Parallel()
	data := fixedDoc(t)
	data.SetCompression(false)
	p1 := data.AddPage(200, 200)
	p1.AddLinkURI([4]float64{10, 10, 110, 30}, "https://example.com")
	p1.AddLinkDest([4]float64{10, 40, 110, 60}, 1, 50, 150)
	data.AddPage(200, 200)

	out := string(writePDF(t, data))
	for _, want := range []string{
		"/Subtype /Link",
		"/A << /S /URI /URI (https://example.com) >>",
		"/Dest [",
		"/XYZ 50 150 null",
		"/Annots [",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestPDFStringLatin1NotUTF8(t *testing.T) {
	t.Parallel()
	// Middle dot must be one WinAnsi byte 0xB7, not UTF-8 C2 B7.
	got := pdfString("a·b")
	want := "(a\\267b)"

	if got != want {
		t.Errorf("pdfString(a·b) = %q, want %q", got, want)
	}
	// Bullet folds to middle dot.
	got = pdfString("•x")
	if got != "(\\267x)" {
		t.Errorf("pdfString(bullet) = %q, want (\\267x)", got)
	}
	// Em dash uses the PDFDocEncoding 0x97 byte rather than an ASCII hyphen.
	if pdfString("\u2014") != "(\\227)" {
		t.Errorf("pdfString(emdash) = %q", pdfString("\u2014"))
	}
}

func TestSubsetWidthsArePDFUnits(t *testing.T) {
	t.Parallel()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	sub, err := subsetFont(fnt, []rune("A "), subsetSimple)
	if err != nil {
		t.Fatal(err)
	}

	first, _, width := subsetWidths(sub, fnt.UnitsPerEm())

	g, ok := sub.glyphIDs['A']
	if !ok {
		t.Fatal("subset missing glyph for A")
	}

	raw := sub.widths[g]
	aWidth := width[int('A')-first]
	scaled := raw * 1000 / float64(fnt.UnitsPerEm())
	// 'A' is ~667 in 1000-unit em for Liberation Sans (TTF advance ~1366 at 2048/em).
	if aWidth < 200 || aWidth > 900 {
		t.Errorf("A width in PDF units = %v, want ~500-700", aWidth)
	}

	if aWidth != scaled {
		t.Errorf("width %v != scaled %v", aWidth, scaled)
	}

	if raw > 0 && aWidth >= raw {
		t.Errorf("PDF width %v should be smaller than raw TTF units %v", aWidth, raw)
	}
}

func TestOutlines(t *testing.T) {
	t.Parallel()
	data := fixedDoc(t)
	data.SetCompression(false)
	data.AddPage(200, 200)

	child := &Outline{ //nolint:exhaustruct // intentional zero-value fields
		Title:   "child",
		PageRef: "4 0 R",
		X:       10,
		Y:       100,
	}
	data.SetOutline(&Outline{ //nolint:exhaustruct // intentional zero-value fields
		Title:    "root",
		Children: []*Outline{{Title: "first", Children: []*Outline{child}}, {Title: "second"}},
	})

	out := string(writePDF(t, data))

	for _, want := range []string{
		"/Type /Outlines",
		"/PageMode /UseOutlines",
		"/Parent",
		"/Prev",
		"/Next",
		"(first)",
		"(second)",
		"/Dest [4 0 R /XYZ 10 100 null]",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
	// Catalog must reference the outlines object: `/Outlines <n> 0 R`, never
	// an empty value which leaves the dictionary malformed.
	if !regexp.MustCompile(`/Outlines \d+ 0 R`).MatchString(out) {
		t.Errorf("catalog missing /Outlines N 0 R; snippet around Outlines: %q",
			outlineSnippet(out))
	}

	if strings.Contains(out, "/Outlines  /PageMode") || strings.Contains(out, "/Outlines /PageMode") {
		t.Error("catalog has empty /Outlines value")
	}
}

func outlineSnippet(out string) string {
	idx := strings.Index(out, "/Outlines")
	if idx < 0 {
		return "(no /Outlines)"
	}

	end := idx + 40
	if end > len(out) {
		end = len(out)
	}

	return out[idx:end]
}

func TestInfoDict(t *testing.T) {
	t.Parallel()
	d := fixedDoc(t)
	d.AddPage(100, 100)

	out := string(writePDF(t, d))
	for _, want := range []string{
		"/Title (Smoke)",
		"/Producer (gowkhtmltopdf " + Version + ")",
		"/CreationDate (D:20260803120000Z)",
		"/Info ",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestOutlineCountAndSort(t *testing.T) {
	t.Parallel()

	arg := &Outline{Title: "a", PageRef: "2 0 R", Y: 10} //nolint:exhaustruct // intentional zero-value fields
	b := &Outline{Title: "b", PageRef: "2 0 R", Y: 5}    //nolint:exhaustruct // intentional zero-value fields
	c := &Outline{Title: "c", PageRef: "1 0 R", Y: 3}    //nolint:exhaustruct // intentional zero-value fields
	nodes := []*Outline{arg, b, c}
	SortOutlines(nodes)

	if nodes[0] != c || nodes[1] != arg || nodes[2] != b {
		t.Errorf("sort order wrong: %q %q %q", nodes[0].Title, nodes[1].Title, nodes[2].Title)
	}

	if got := outlineCount(&Outline{ //nolint:exhaustruct // intentional zero-value fields
		Children: []*Outline{{}, {Children: []*Outline{{}}}},
	}); got != 3 {
		t.Errorf("outlineCount = %d, want 3", got)
	}
}

func TestSetFillColorGrayscale(t *testing.T) {
	t.Parallel()
	// P5-03: Document.SetGrayscale must have its promised paint-time effect.
	// Fill/stroke colors fold through Rec.601 luma so r=g=b lines are equal.
	doc := fixedDoc(t)
	doc.SetGrayscale(true)
	doc.SetCompression(false)
	p := doc.AddPage(100, 100)
	c := p.Content()
	c.SetFillColor(1, 0, 0)   // pure red → luma 0.299
	c.SetStrokeColor(0, 1, 0) // pure green → luma 0.587

	out := string(writePDF(t, doc))
	if !strings.Contains(out, "0.299 0.299 0.299 rg") {
		t.Errorf("grayscale fill not folded to luma: %q", outlineSnippet(out))
	}

	if !strings.Contains(out, "0.587 0.587 0.587 RG") {
		t.Errorf("grayscale stroke not folded to luma: %q", outlineSnippet(out))
	}

	if strings.Contains(out, "1 0 0 rg") || strings.Contains(out, "0 1 0 RG") {
		t.Error("grayscale mode emitted the raw RGB color")
	}

	// Default (off) keeps colors untouched.
	d2Val := fixedDoc(t)
	d2Val.SetCompression(false)
	c2 := d2Val.AddPage(100, 100).Content()
	c2.SetFillColor(1, 0, 0)
	c2.SetStrokeColor(0, 1, 0)

	out2 := string(writePDF(t, d2Val))
	if !strings.Contains(out2, "1 0 0 rg") || !strings.Contains(out2, "0 1 0 RG") {
		t.Error("color mode must keep the raw RGB color")
	}
}

func TestGrayscaleGetter(t *testing.T) {
	t.Parallel()

	doc := NewDocument()
	if doc.Grayscale() {
		t.Error("new document must start in color mode")
	}

	doc.SetGrayscale(true)

	if !doc.Grayscale() {
		t.Error("SetGrayscale(true) must report Grayscale() == true")
	}
}

func TestOutlineBadPageRefFails(t *testing.T) {
	t.Parallel()
	// P5-04: a bogus PageRef must fail Write instead of emitting a corrupt
	// /Dest with no diagnostic.
	for _, ref := range []string{"", "999999 0 R", "garbage", "4 0 X", "0 0 R"} {
		data := fixedDoc(t)
		data.AddPage(200, 200)
		data.SetOutline(&Outline{ //nolint:exhaustruct // intentional zero-value fields
			Title:    "root",
			Children: []*Outline{{Title: "bad", PageRef: ref}},
		})

		var buf bytes.Buffer

		err := data.Write(&buf)
		if ref == "" {
			if err != nil {
				t.Errorf("empty PageRef: unexpected error %v", err)
			}

			continue
		}

		if err == nil {
			t.Errorf("PageRef %q: expected Write error, got nil", ref)
		}
	}
	// A valid ref still writes fine (page object 1).
	doc := fixedDoc(t)
	doc.SetCompression(false)
	doc.AddPage(200, 200)
	doc.SetOutline(&Outline{ //nolint:exhaustruct // intentional zero-value fields
		Title:    "root",
		Children: []*Outline{{Title: "ok", PageRef: "1 0 R", X: 5, Y: 6}},
	})

	out := string(writePDF(t, doc))
	if !strings.Contains(out, "/Dest [1 0 R /XYZ 5 6 null]") {
		t.Error("valid PageRef must emit /Dest")
	}
}

func TestWriteToMemoryBuffer(t *testing.T) {
	t.Parallel()
	data := fixedDoc(t)
	data.SetCompression(false)
	p := data.AddPage(200, 200)
	p.Content().TextShow("memory buffer")

	var buf bytes.Buffer
	if err := data.Write(&buf); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if buf.Len() == 0 {
		t.Fatal("buffer is empty")
	}

	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Fatal("buffer does not contain a PDF")
	}

	if !bytes.Contains(buf.Bytes(), []byte("(memory buffer)")) {
		t.Error("buffer output is missing the content stream")
	}
}

func TestMultipleWritesDeterministic(t *testing.T) {
	t.Parallel()

	data := fixedDoc(t)
	p := data.AddPage(200, 200)
	p.Content().TextShow("repeat write")

	var buf1, buf2 bytes.Buffer
	if err := data.Write(&buf1); err != nil {
		t.Fatalf("Write 1: %v", err)
	}

	if err := data.Write(&buf2); err != nil {
		t.Fatalf("Write 2: %v", err)
	}

	if !bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
		t.Error("consecutive Write calls produced non-deterministic output")
	}
}

// kidsRefs extracts the page refs listed in the pages tree /Kids array.
func kidsRefs(t *testing.T, out string) []string {
	t.Helper()

	m := kidsRe.FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("no /Kids array in output")
	}

	fields := strings.Fields(m[1])

	var refs []string

	for i := 0; i+2 < len(fields); i += 3 {
		refs = append(refs, strings.Join(fields[i:i+3], " "))
	}

	return refs
}

func TestReorderPagesKidsOrder(t *testing.T) {
	t.Parallel()
	data := fixedDoc(t)
	pageA := data.AddPage(100, 100)
	pBVal := data.AddPage(100, 100)
	pCVal := data.AddPage(100, 100)

	if err := data.ReorderPages([]int{2, 0, 1}); err != nil {
		t.Fatalf("ReorderPages: %v", err)
	}

	out := string(writePDF(t, data))
	kids := kidsRefs(t, out)
	want := []string{pCVal.ref.String(), pageA.ref.String(), pBVal.ref.String()}

	if strings.Join(kids, " ") != strings.Join(want, " ") {
		t.Errorf("/Kids = %v, want %v", kids, want)
	}
	// every page object still owns its original content stream
	for _, p := range []*Page{pageA, pBVal, pCVal} {
		if !strings.Contains(out, "/Contents "+p.contentRef.String()) {
			t.Errorf("page %s lost its content stream %s", p.ref, p.contentRef)
		}
	}
}

func TestDuplicatePage(t *testing.T) {
	t.Parallel()
	data := fixedDoc(t)
	data.SetCompression(false)
	pageA := data.AddPage(100, 100)
	pageA.Content().TextShow("AAA")

	pBVal := data.AddPage(200, 200)
	pBVal.Content().TextShow("BBB")

	dup, err := data.DuplicatePage(0)
	if err != nil {
		t.Fatalf("DuplicatePage: %v", err)
	}

	if dup.Width() != 100 || dup.Height() != 100 {
		t.Errorf("duplicate size = %g x %g, want 100 x 100", dup.Width(), dup.Height())
	}

	if data.PageCount() != 3 {
		t.Fatalf("PageCount = %d, want 3", data.PageCount())
	}

	if dup.ref == pageA.ref || dup.contentRef == pageA.contentRef {
		t.Error("duplicate must have its own page and content objects")
	}

	out := string(writePDF(t, data))
	verifyKidsAndContents(t, out, []*Page{pageA, pBVal, dup}, []string{
		pageA.ref.String(), pBVal.ref.String(), dup.ref.String(),
	})
	// both copies of A paint the same text
	if c := strings.Count(out, "(AAA)"); c != 2 {
		t.Errorf("(AAA) appears %d times, want 2", c)
	}

	if _, err := data.DuplicatePage(5); err == nil {
		t.Error("DuplicatePage(5): expected error for out-of-range index")
	}

	if _, err := data.DuplicatePage(-1); err == nil {
		t.Error("DuplicatePage(-1): expected error for negative index")
	}
}

// verifyKidsAndContents checks the pages tree /Kids order and that every
// page object still owns its content stream.
func verifyKidsAndContents(t *testing.T, out string, pages []*Page, wantKids []string) {
	t.Helper()

	kids := kidsRefs(t, out)

	if strings.Join(kids, " ") != strings.Join(wantKids, " ") {
		t.Errorf("/Kids = %v, want %v", kids, wantKids)
	}

	for _, p := range pages {
		if !strings.Contains(out, "/Contents "+p.contentRef.String()) {
			t.Errorf("page %s lost its content stream %s", p.ref, p.contentRef)
		}
	}
}

func TestDuplicatePageOwnsResourceMaps(t *testing.T) {
	t.Parallel()
	data := fixedDoc(t)
	data.SetCompression(false)

	page := data.AddPage(100, 100)
	if err := page.Content().AddPNGImage("I0", 0, 0, 10, 10, makePNG(t, false)); err != nil {
		t.Fatalf("source image: %v", err)
	}

	dup, err := data.DuplicatePage(0)
	if err != nil {
		t.Fatalf("DuplicatePage: %v", err)
	}

	if err := dup.Content().AddPNGImage("I1", 20, 20, 10, 10, makePNG(t, true)); err != nil {
		t.Fatalf("duplicate image: %v", err)
	}

	if _, ok := page.Content().imageRefs["I1"]; ok {
		t.Fatal("duplicate resource was added to source page")
	}

	if _, ok := dup.Content().imageRefs["I1"]; !ok {
		t.Fatal("duplicate page did not retain its own resource")
	}
}

func TestReorderPagesValidation(t *testing.T) {
	t.Parallel()
	data := fixedDoc(t)
	data.AddPage(100, 100)
	data.AddPage(100, 100)

	for _, order := range [][]int{
		{0},       // wrong length
		{0, 1, 2}, // wrong length
		{-1, 0},   // negative index
		{2, 0},    // out of range
		{0, 0},    // duplicate index
	} {
		if err := data.ReorderPages(order); err == nil {
			t.Errorf("ReorderPages(%v): expected error, got nil", order)
		}
	}
	// failed reorders must leave the page order untouched
	if got := kidsRefs(t, string(writePDF(t, data))); len(got) != 2 {
		t.Errorf("pages corrupted by failed reorders: %v", got)
	}
	// reordering after Write (finalize) must fail
	var buf bytes.Buffer
	if err := data.Write(&buf); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if err := data.ReorderPages([]int{1, 0}); err == nil {
		t.Error("ReorderPages after finalize: expected error, got nil")
	}
}

//nolint:funlen,cyclop,maintidx // comprehensive integration test covering rich PDF 1.7 feature set
func TestPDF17RichDocument(t *testing.T) {
	t.Parallel()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatalf("DefaultFont: %v", err)
	}

	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF17): %v", err)
	}

	fixedTime := time.Date(2026, 8, 14, 16, 0, 0, 0, time.UTC)
	doc.SetCreationTime(fixedTime)
	doc.SetInfo("Title", "Rich Document — PDF 1.7 Specification Test")
	doc.SetInfo("Author", "Test Author")
	doc.SetInfo("Subject", "Testing PDF 1.7 Features")
	doc.SetInfo("Keywords", "PDF, 1.7, Graphics, Fonts, Images")
	doc.SetInfo("Creator", "gowkhtmltopdf test suite")

	// Page 1: Graphics, Text, Transparency, Images, Link Annotations
	page1 := doc.AddPage(600, 800)
	content1 := page1.Content()

	// 1. Graphics & Paths
	content1.Save()
	content1.SetFillColor(0.2, 0.4, 0.8)
	content1.SetStrokeColor(0.1, 0.1, 0.1)
	content1.SetLineWidth(2.0)
	content1.SetLineCap(1)
	content1.Rect(50, 700, 200, 50)
	content1.Fill()
	content1.MoveTo(50, 680)
	content1.LineTo(250, 680)
	content1.CurveTo(270, 680, 290, 660, 300, 640)
	content1.Stroke()
	content1.Restore()

	// 2. ExtGState Opacity
	content1.Save()
	content1.SetOpacity(0.65)
	content1.SetFillColor(0.9, 0.2, 0.2)
	content1.Rect(70, 710, 100, 30)
	content1.Fill()
	content1.Restore()

	// 3. Text
	content1.UseEmbeddedFont("F1", fnt)
	content1.BeginText()
	content1.SetFont("F1", 14)
	content1.TextLeading(18)
	content1.TextAt(50, 600)
	content1.SetCharSpacing(0.5)
	content1.TextShow("PDF 1.7 Text Heading")
	content1.TextNextLine()
	content1.TextShow("Secondary line with numbers: 1234567890")
	content1.EndText()

	// 4. Images (JPEG pass-through and PNG with SMask)
	if err := content1.AddJPEGImage("J1", 50, 450, 100, 80, makeJPEG(t)); err != nil {
		t.Fatalf("AddJPEGImage: %v", err)
	}

	if err := content1.AddPNGImage("P1", 200, 450, 100, 80, makePNG(t, true)); err != nil {
		t.Fatalf("AddPNGImage: %v", err)
	}

	// 5. Link Annotations (URI + Page GoTo)
	page1.AddLinkURI([4]float64{50, 600, 250, 620}, "https://example.com/pdf17")
	page1.AddLinkDest([4]float64{50, 450, 150, 530}, 1, 50, 750)

	// Page 2: Additional page content for internal linking
	page2 := doc.AddPage(600, 800)
	content2 := page2.Content()
	content2.UseEmbeddedFont("F1", fnt)
	content2.BeginText()
	content2.SetFont("F1", 12)
	content2.TextAt(50, 750)
	content2.TextShow("Destination Page 2 Content")
	content2.EndText()

	// 6. Outline hierarchy
	rootOutline := &Outline{ //nolint:exhaustruct // intentional zero-value fields
		Title: "root",
		Children: []*Outline{
			{
				Title:   "Chapter 1 — Introduction",
				PageRef: page1.ref.String(),
				X:       50,
				Y:       750,
			},
			{
				Title:   "Chapter 2 — Details",
				PageRef: page2.ref.String(),
				X:       50,
				Y:       750,
				Children: []*Outline{
					{
						Title:   "Section 2.1 — Sub-topic",
						PageRef: page2.ref.String(),
						X:       50,
						Y:       600,
					},
				},
			},
		},
	}
	doc.SetOutline(rootOutline)

	// Serialize
	var buf bytes.Buffer

	bytesWritten, err := doc.WriteTo(&buf)
	if err != nil {
		t.Fatalf("doc.WriteTo: %v", err)
	}

	if bytesWritten != int64(buf.Len()) {
		t.Errorf("WriteTo count = %d, buf.Len() = %d", bytesWritten, buf.Len())
	}

	outBytes := buf.Bytes()
	outStr := string(outBytes)

	// Verify Header & Binary Comment
	if !strings.HasPrefix(outStr, "%PDF-1.7\n%\xe2\xe3\xcf\xd3\n") {
		t.Errorf("invalid header prefix in PDF 1.7 rich doc")
	}

	// Verify Trailer /ID and structure
	if !strings.Contains(outStr, "trailer\n<< /Size ") || !strings.Contains(outStr, "/ID [ <") {
		t.Errorf("PDF 1.7 rich doc trailer missing /ID or invalid structure")
	}

	// Verify Catalog & Metadata stream
	if !strings.Contains(outStr, "/Type /Catalog") ||
		!strings.Contains(outStr, "/Metadata ") ||
		!strings.Contains(outStr, "/Outlines ") {
		t.Errorf("Catalog missing /Metadata or /Outlines")
	}

	if !strings.Contains(outStr, "/Type /Metadata /Subtype /XML") {
		t.Errorf("Metadata stream missing /Type /Metadata /Subtype /XML")
	}

	if !strings.Contains(outStr, "<pdf:Producer>gowkhtmltopdf 1.7</pdf:Producer>") {
		t.Errorf("XMP metadata missing correct Producer")
	}

	// Verify Page Resources
	for _, expectedRes := range []string{
		"/ProcSet [/PDF /Text /ImageB /ImageC /ImageI]",
		"/Font <<",
		"/XObject <<",
		"/ExtGState << /opacity << /CA 0.65 /ca 0.65 >> >>",
	} {
		if !strings.Contains(outStr, expectedRes) {
			t.Errorf("missing page resource entry %q", expectedRes)
		}
	}

	// Verify Annotations
	wantURIAnnot := "/Subtype /Link /Rect [50 600 250 620] /A << /S /URI /URI (https://example.com/pdf17) >>"
	if !strings.Contains(outStr, wantURIAnnot) {
		t.Errorf("URI annotation missing or malformed")
	}

	if !strings.Contains(outStr, "/Dest ["+page2.ref.String()+" /XYZ 50 750 null]") {
		t.Errorf("GoTo destination annotation missing or malformed")
	}

	// Verify Outlines
	if !strings.Contains(outStr, "/Type /Outlines") || !strings.Contains(outStr, "/PageMode /UseOutlines") {
		t.Errorf("Outlines or /PageMode /UseOutlines missing")
	}

	// Verify Semantic representation
	sem, err := ParseSemantic(outBytes)
	if err != nil {
		t.Fatalf("ParseSemantic(richDoc): %v", err)
	}

	if sem.Version != versionToken17 {
		t.Errorf("SemanticDoc.Version = %q, want 1.7", sem.Version)
	}

	if len(sem.Pages) != 2 {
		t.Fatalf("SemanticDoc.Pages count = %d, want 2", len(sem.Pages))
	}

	if sem.Pages[0].MediaBox != [4]float64{0, 0, 600, 800} {
		t.Errorf("Page 1 MediaBox = %v, want [0 0 600 800]", sem.Pages[0].MediaBox)
	}

	if len(sem.Pages[0].Annots) != 2 {
		t.Errorf("Page 1 Annots count = %d, want 2", len(sem.Pages[0].Annots))
	}

	// Verify Determinism across multiple runs
	doc2, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy: %v", err)
	}

	doc2.SetCreationTime(fixedTime)
	doc2.SetInfo("Title", "Rich Document — PDF 1.7 Specification Test")
	doc2.SetInfo("Author", "Test Author")
	doc2.SetInfo("Subject", "Testing PDF 1.7 Features")
	doc2.SetInfo("Keywords", "PDF, 1.7, Graphics, Fonts, Images")
	doc2.SetInfo("Creator", "gowkhtmltopdf test suite")

	p2_1 := doc2.AddPage(600, 800)
	c2_1 := p2_1.Content()
	c2_1.Save()
	c2_1.SetFillColor(0.2, 0.4, 0.8)
	c2_1.SetStrokeColor(0.1, 0.1, 0.1)
	c2_1.SetLineWidth(2.0)
	c2_1.SetLineCap(1)
	c2_1.Rect(50, 700, 200, 50)
	c2_1.Fill()
	c2_1.MoveTo(50, 680)
	c2_1.LineTo(250, 680)
	c2_1.CurveTo(270, 680, 290, 660, 300, 640)
	c2_1.Stroke()
	c2_1.Restore()
	c2_1.Save()
	c2_1.SetOpacity(0.65)
	c2_1.SetFillColor(0.9, 0.2, 0.2)
	c2_1.Rect(70, 710, 100, 30)
	c2_1.Fill()
	c2_1.Restore()
	c2_1.UseEmbeddedFont("F1", fnt)
	c2_1.BeginText()
	c2_1.SetFont("F1", 14)
	c2_1.TextLeading(18)
	c2_1.TextAt(50, 600)
	c2_1.SetCharSpacing(0.5)
	c2_1.TextShow("PDF 1.7 Text Heading")
	c2_1.TextNextLine()
	c2_1.TextShow("Secondary line with numbers: 1234567890")
	c2_1.EndText()
	_ = c2_1.AddJPEGImage("J1", 50, 450, 100, 80, makeJPEG(t))
	_ = c2_1.AddPNGImage("P1", 200, 450, 100, 80, makePNG(t, true))

	p2_1.AddLinkURI([4]float64{50, 600, 250, 620}, "https://example.com/pdf17")
	p2_1.AddLinkDest([4]float64{50, 450, 150, 530}, 1, 50, 750)

	p2_2 := doc2.AddPage(600, 800)
	c2_2 := p2_2.Content()
	c2_2.UseEmbeddedFont("F1", fnt)
	c2_2.BeginText()
	c2_2.SetFont("F1", 12)
	c2_2.TextAt(50, 750)
	c2_2.TextShow("Destination Page 2 Content")
	c2_2.EndText()

	doc2.SetOutline(&Outline{ //nolint:exhaustruct // intentional zero-value fields
		Title: "root",
		Children: []*Outline{
			{
				Title:   "Chapter 1 — Introduction",
				PageRef: p2_1.ref.String(),
				X:       50,
				Y:       750,
			},
			{
				Title:   "Chapter 2 — Details",
				PageRef: p2_2.ref.String(),
				X:       50,
				Y:       750,
				Children: []*Outline{
					{
						Title:   "Section 2.1 — Sub-topic",
						PageRef: p2_2.ref.String(),
						X:       50,
						Y:       600,
					},
				},
			},
		},
	})

	var buf2 bytes.Buffer
	if err := doc2.Write(&buf2); err != nil {
		t.Fatalf("doc2.Write: %v", err)
	}

	if !bytes.Equal(outBytes, buf2.Bytes()) {
		t.Error("consecutive PDF 1.7 rich document writes are not byte-identical")
	}
}
