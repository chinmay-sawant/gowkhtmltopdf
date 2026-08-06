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
	d := fixedDoc(t)
	d.AddPage(595.276, 841.89) // A4 portrait
	out := writePDF(t, d)
	s := string(out)

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
		if !strings.Contains(s, want) {
			t.Errorf("output missing %q", want)
		}
	}
	if !strings.HasPrefix(s, "%PDF-") {
		t.Errorf("output must start with PDF header")
	}
}

func TestDeterministicOutput(t *testing.T) {
	f, err := DefaultFont()
	if err != nil {
		t.Fatal(err)
	}
	build := func() []byte {
		d := fixedDoc(t)
		p := d.AddPage(200, 200)
		c := p.Content()
		c.UseEmbeddedFont("F1", f)
		c.BeginText()
		c.SetFont("F1", 12)
		c.TextAt(10, 20)
		c.TextShow("hello")
		c.EndText()
		d.SetOutline(&Outline{Title: "root", Children: []*Outline{{Title: "child"}}})
		return writePDF(t, d)
	}
	a, b := build(), build()
	if !bytes.Equal(a, b) {
		t.Errorf("output not deterministic")
	}
}

func TestXrefOffsets(t *testing.T) {
	f, err := DefaultFont()
	if err != nil {
		t.Fatal(err)
	}
	d := fixedDoc(t)
	p := d.AddPage(100, 100)
	p.Content().UseEmbeddedFont("F1", f)
	out := writePDF(t, d)

	// every n entry must point at the start of "N 0 obj"
	lines := strings.Split(string(out), "\n")
	xrefIdx := -1
	for i, l := range lines {
		if l == "xref" {
			xrefIdx = i
			break
		}
	}
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
	offsets := map[int]int{}
	for i := xrefIdx + 3; i < startxref-2; i++ {
		fields := strings.Fields(lines[i])
		if len(fields) != 3 || fields[1] != "00000" {
			continue
		}
		obj := i - xrefIdx - 2
		off, err := strconv.Atoi(fields[0])
		if err != nil {
			t.Fatalf("bad offset %q", fields[0])
		}
		offsets[obj] = off
	}
	for obj, off := range offsets {
		want := strconv.Itoa(obj) + " 0 obj"
		if !bytes.HasPrefix(out[off:], []byte(want)) {
			t.Errorf("object %d offset %d does not start with %q", obj, off, want)
		}
	}
}

func TestEmptyDocFails(t *testing.T) {
	d := NewDocument()
	var buf bytes.Buffer
	if err := d.Write(&buf); err == nil {
		t.Fatal("expected error for empty document")
	}
}

func TestContentOperators(t *testing.T) {
	d := fixedDoc(t)
	p := d.AddPage(300, 300)
	c := p.Content()
	c.Save()
	c.SetFillColor(0.5, 0.25, 0.75)
	c.Rect(10, 10, 50, 60)
	c.Fill()
	c.SetLineWidth(1.5)
	c.MoveTo(0, 0)
	c.LineTo(100, 100)
	c.Stroke()
	c.Restore()
	out := writePDF(t, d)
	s := string(out)
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
	_ = s
}

func TestTextStream(t *testing.T) {
	f, err := DefaultFont()
	if err != nil {
		t.Fatal(err)
	}
	d := fixedDoc(t)
	d.SetCompression(false)
	p := d.AddPage(300, 300)
	c := p.Content()
	c.UseEmbeddedFont("F1", f)
	c.BeginText()
	c.SetFont("F1", 14)
	c.TextLeading(18)
	c.TextAt(20, 280)
	c.TextShow("line one")
	c.TextNextLine()
	c.TextShow("line two")
	c.EndText()
	out := string(writePDF(t, d))
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
	d := fixedDoc(t)
	d.SetCompression(false)
	p := d.AddPage(200, 200)
	if err := p.Content().AddPNGImage("Im1", 10, 10, 100, 50, makePNG(t, false)); err != nil {
		t.Fatal(err)
	}
	out := string(writePDF(t, d))
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
	d := fixedDoc(t)
	d.SetCompression(false)
	p1 := d.AddPage(200, 200)
	p1.AddLinkURI([4]float64{10, 10, 110, 30}, "https://example.com")
	p1.AddLinkDest([4]float64{10, 40, 110, 60}, 1, 50, 150)
	d.AddPage(200, 200)
	out := string(writePDF(t, d))
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
	f, err := DefaultFont()
	if err != nil {
		t.Fatal(err)
	}
	sub, err := subsetFont(f, []rune("A "), subsetSimple)
	if err != nil {
		t.Fatal(err)
	}
	first, _, w := subsetWidths(sub, f.UnitsPerEm())
	g, ok := sub.glyphIDs['A']
	if !ok {
		t.Fatal("subset missing glyph for A")
	}
	raw := sub.widths[g]
	aWidth := w[int('A')-first]
	scaled := raw * 1000 / float64(f.UnitsPerEm())
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
	d := fixedDoc(t)
	d.SetCompression(false)
	d.AddPage(200, 200)
	child := &Outline{Title: "child", PageRef: "4 0 R", X: 10, Y: 100}
	d.SetOutline(&Outline{
		Title:    "root",
		Children: []*Outline{{Title: "first", Children: []*Outline{child}}, {Title: "second"}},
	})
	out := string(writePDF(t, d))
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
	i := strings.Index(out, "/Outlines")
	if i < 0 {
		return "(no /Outlines)"
	}
	end := i + 40
	if end > len(out) {
		end = len(out)
	}
	return out[i:end]
}

func TestInfoDict(t *testing.T) {
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
	a := &Outline{Title: "a", PageRef: "2 0 R", Y: 10}
	b := &Outline{Title: "b", PageRef: "2 0 R", Y: 5}
	c := &Outline{Title: "c", PageRef: "1 0 R", Y: 3}
	nodes := []*Outline{a, b, c}
	SortOutlines(nodes)
	if nodes[0] != c || nodes[1] != a || nodes[2] != b {
		t.Errorf("sort order wrong: %q %q %q", nodes[0].Title, nodes[1].Title, nodes[2].Title)
	}
	if got := outlineCount(&Outline{Children: []*Outline{{}, {Children: []*Outline{{}}}}}); got != 3 {
		t.Errorf("outlineCount = %d, want 3", got)
	}
}

func TestWriteToMemoryBuffer(t *testing.T) {
	d := fixedDoc(t)
	d.SetCompression(false)
	p := d.AddPage(200, 200)
	p.Content().TextShow("memory buffer")
	var buf bytes.Buffer
	if err := d.Write(&buf); err != nil {
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
	d := fixedDoc(t)
	pA := d.AddPage(100, 100)
	pB := d.AddPage(100, 100)
	pC := d.AddPage(100, 100)
	if err := d.ReorderPages([]int{2, 0, 1}); err != nil {
		t.Fatalf("ReorderPages: %v", err)
	}
	out := string(writePDF(t, d))
	kids := kidsRefs(t, out)
	want := []string{pC.ref, pA.ref, pB.ref}
	if strings.Join(kids, " ") != strings.Join(want, " ") {
		t.Errorf("/Kids = %v, want %v", kids, want)
	}
	// every page object still owns its original content stream
	for _, p := range []*Page{pA, pB, pC} {
		if !strings.Contains(out, "/Contents "+p.contentRef) {
			t.Errorf("page %s lost its content stream %s", p.ref, p.contentRef)
		}
	}
}

func TestDuplicatePage(t *testing.T) {
	d := fixedDoc(t)
	d.SetCompression(false)
	pA := d.AddPage(100, 100)
	pA.Content().TextShow("AAA")
	pB := d.AddPage(200, 200)
	pB.Content().TextShow("BBB")

	dup, err := d.DuplicatePage(0)
	if err != nil {
		t.Fatalf("DuplicatePage: %v", err)
	}
	if dup.Width() != 100 || dup.Height() != 100 {
		t.Errorf("duplicate size = %g x %g, want 100 x 100", dup.Width(), dup.Height())
	}
	if d.PageCount() != 3 {
		t.Fatalf("PageCount = %d, want 3", d.PageCount())
	}
	if dup.ref == pA.ref || dup.contentRef == pA.contentRef {
		t.Error("duplicate must have its own page and content objects")
	}

	out := string(writePDF(t, d))
	// kids must be [A B A'] and every page keeps its own content stream
	kids := kidsRefs(t, out)
	wantKids := []string{pA.ref, pB.ref, dup.ref}
	if strings.Join(kids, " ") != strings.Join(wantKids, " ") {
		t.Errorf("/Kids = %v, want %v", kids, wantKids)
	}
	for _, p := range []*Page{pA, pB, dup} {
		if !strings.Contains(out, "/Contents "+p.contentRef) {
			t.Errorf("page %s lost its content stream %s", p.ref, p.contentRef)
		}
	}
	// both copies of A paint the same text
	if c := bytes.Count([]byte(out), []byte("(AAA)")); c != 2 {
		t.Errorf("(AAA) appears %d times, want 2", c)
	}

	if _, err := d.DuplicatePage(5); err == nil {
		t.Error("DuplicatePage(5): expected error for out-of-range index")
	}
	if _, err := d.DuplicatePage(-1); err == nil {
		t.Error("DuplicatePage(-1): expected error for negative index")
	}
}

func TestReorderPagesValidation(t *testing.T) {
	d := fixedDoc(t)
	d.AddPage(100, 100)
	d.AddPage(100, 100)
	for _, order := range [][]int{
		{0},       // wrong length
		{0, 1, 2}, // wrong length
		{-1, 0},   // negative index
		{2, 0},    // out of range
		{0, 0},    // duplicate index
	} {
		if err := d.ReorderPages(order); err == nil {
			t.Errorf("ReorderPages(%v): expected error, got nil", order)
		}
	}
	// failed reorders must leave the page order untouched
	if got := kidsRefs(t, string(writePDF(t, d))); len(got) != 2 {
		t.Errorf("pages corrupted by failed reorders: %v", got)
	}
	// reordering after Write (finalize) must fail
	var buf bytes.Buffer
	if err := d.Write(&buf); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := d.ReorderPages([]int{1, 0}); err == nil {
		t.Error("ReorderPages after finalize: expected error, got nil")
	}
}
