package pdf

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
	"time"
)

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
	build := func() []byte {
		d := fixedDoc(t)
		p := d.AddPage(200, 200)
		c := p.Content()
		c.UseFont("F1", "Helvetica")
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
	d := fixedDoc(t)
	p := d.AddPage(100, 100)
	p.Content().UseFont("F1", "Times-Roman")
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
	d := fixedDoc(t)
	d.SetCompression(false)
	p := d.AddPage(300, 300)
	c := p.Content()
	c.UseFont("F1", "Courier")
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
		"/Type /Font /Subtype /Type1 /BaseFont /Courier",
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
	// 2x1 gradient
	rgba := []byte{255, 0, 0, 255, 0, 0, 255, 255}
	p.Content().DrawImage("Im1", 10, 10, 100, 50, rgba, 2, 1)
	out := string(writePDF(t, d))
	for _, want := range []string{
		"/Im1 Do",
		"/Subtype /Image",
		"/Width 2 /Height 1",
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
