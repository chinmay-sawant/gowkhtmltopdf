//nolint:testpackage // tests reach into unexported state
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
	// Em dash (U+2014) folds to ASCII hyphen for WinAnsi output.
	if pdfString("\u2014") != "(-)" {
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
