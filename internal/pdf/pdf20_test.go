//nolint:testpackage,exhaustruct // tests reach into unexported state
package pdf

import (
	"bytes"
	"errors"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

//nolint:cyclop,funlen // four-version /ID matrix in one test
func TestPDF20TrailerID(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	makeDoc := func(title string, version PDFVersion) *Document {
		docObj, err := NewDocumentWithPolicy(WriterPolicy{Version: version})
		if err != nil {
			t.Fatalf("NewDocumentWithPolicy(%v): %v", version, err)
		}

		docObj.SetCreationTime(fixedTime)
		docObj.SetInfo("Title", title)
		docObj.AddPage(300, 400)

		return docObj
	}

	// 1. PDF 2.0 trailer has /ID [ <hex> <hex> ] with two 32-char entries.
	doc20A := makeDoc("Doc A", PDF20)

	var buf20A bytes.Buffer
	if err := doc20A.Write(&buf20A); err != nil {
		t.Fatalf("doc20A.Write: %v", err)
	}

	str20A := buf20A.String()

	trailerIdx := strings.Index(str20A, "trailer\n")
	if trailerIdx < 0 {
		t.Fatal("PDF 2.0 output missing trailer")
	}

	idRe := regexp.MustCompile(`/ID\s*\[\s*<([0-9A-Fa-f]{32})>\s*<([0-9A-Fa-f]{32})>\s*\]`)

	matches := idRe.FindStringSubmatch(str20A[trailerIdx:])
	if len(matches) != 3 {
		t.Fatalf("PDF 2.0 trailer /ID regex did not match:\n%s", str20A[trailerIdx:])
	}

	if matches[1] != matches[2] {
		t.Errorf("trailer /ID elements not equal: %q vs %q", matches[1], matches[2])
	}

	// 2. Determinism: two writes of the same 2.0 document are byte-identical
	// (the ID derives from injectable creation time + content, never
	// math/rand or time.Now).
	doc20B := makeDoc("Doc A", PDF20)

	var buf20B bytes.Buffer
	if err := doc20B.Write(&buf20B); err != nil {
		t.Fatalf("doc20B.Write: %v", err)
	}

	if !bytes.Equal(buf20A.Bytes(), buf20B.Bytes()) {
		t.Error("two writes of the same PDF 2.0 document are not byte-identical")
	}

	// 3. The ID participates in the version, so the same content under 1.7
	// gets a different ID.
	doc17 := makeDoc("Doc A", PDF17)

	var buf17 bytes.Buffer
	if err := doc17.Write(&buf17); err != nil {
		t.Fatalf("doc17.Write: %v", err)
	}

	matches17 := idRe.FindStringSubmatch(buf17.String())
	if len(matches17) != 3 {
		t.Fatalf("PDF 1.7 trailer /ID regex did not match")
	}

	if matches17[1] == matches[1] {
		t.Errorf("PDF 2.0 and 1.7 IDs are equal (%s); version must participate", matches[1])
	}

	// 4. 1.4 trailer has no /ID at all (bit-compatible with legacy output).
	doc14 := makeDoc("Doc A", PDF14)

	var buf14 bytes.Buffer
	if err := doc14.Write(&buf14); err != nil {
		t.Fatalf("doc14.Write: %v", err)
	}

	idx14 := strings.Index(buf14.String(), "trailer\n")
	if idx14 < 0 {
		t.Fatal("PDF 1.4 output missing trailer")
	}

	if strings.Contains(buf14.String()[idx14:], "/ID") {
		t.Errorf("PDF 1.4 trailer must not contain /ID: %s", buf14.String()[idx14:])
	}
}

//nolint:cyclop,funlen // tests the full 2.0 catalog + non-claiming XMP structure
func TestPDF20CatalogAndMetadataStream(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 8, 14, 15, 30, 0, 0, time.UTC)

	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF20})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF20): %v", err)
	}

	doc.SetCreationTime(fixedTime)
	doc.SetInfo("Title", "Test Document")
	page := doc.AddPage(200, 200)
	doc.SetOutline(&Outline{
		Title:    "root",
		Children: []*Outline{{Title: "child", PageRef: page.ref.String()}},
	})

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write: %v", err)
	}

	outStr := buf.String()

	// Catalog keeps /Type /Catalog + /Pages and references /Metadata;
	// outlines were finalized before the catalog so /Outlines resolves.
	catRe := regexp.MustCompile(
		`<< /Type /Catalog /Metadata (\d+ 0 R) /Pages \d+ 0 R /Outlines \d+ 0 R /PageMode /UseOutlines >>`)

	catMatch := catRe.FindStringSubmatch(outStr)
	if len(catMatch) != 2 {
		t.Fatalf("PDF 2.0 catalog does not match expected pattern:\n%s", outStr)
	}

	// Catalog must NOT carry /Version: the header is the sole version
	// authority (matching the #31 1.7 sibling).
	if strings.Contains(catMatch[0], "/Version") {
		t.Errorf("PDF 2.0 catalog must not contain /Version: %s", catMatch[0])
	}

	metaRef := catMatch[1]

	metaObjHeader := strings.Replace(metaRef, " 0 R", " 0 obj", 1)
	if !strings.Contains(outStr, metaObjHeader) {
		t.Fatalf("metadata object %s not found in PDF 2.0 output", metaObjHeader)
	}

	if !strings.Contains(outStr, "/Type /Metadata /Subtype /XML") {
		t.Error("PDF 2.0 metadata object missing /Type /Metadata /Subtype /XML")
	}

	// Well-formed xpacket framing and required Dublin Core / pdf fields.
	if !strings.Contains(outStr, `<?xpacket begin="`+"\xef\xbb\xbf"+`" id="W5M0MpCehiHzreSzNTczkc9d"?>`) {
		t.Error("PDF 2.0 XMP header missing or malformed")
	}

	if !strings.Contains(outStr, "<dc:format>application/pdf</dc:format>") {
		t.Error("PDF 2.0 XMP missing dc:format application/pdf")
	}

	if !strings.Contains(outStr, "<pdf:Producer>gowkhtmltopdf 2.0</pdf:Producer>") {
		t.Error("PDF 2.0 XMP missing pdf:Producer gowkhtmltopdf 2.0")
	}

	// Dates come from the injectable creation time.
	if !strings.Contains(outStr, "<xmp:CreateDate>2026-08-14T15:30:00Z</xmp:CreateDate>") {
		t.Error("PDF 2.0 XMP missing correct xmp:CreateDate")
	}

	if !strings.Contains(outStr, "<xmp:ModifyDate>2026-08-14T15:30:00Z</xmp:ModifyDate>") {
		t.Error("PDF 2.0 XMP missing correct xmp:ModifyDate")
	}

	if !strings.Contains(outStr, "<?xpacket end=\"w\"?>") {
		t.Error("PDF 2.0 XMP missing closing xpacket")
	}

	// Negative: the packet must not claim PDF/A or PDF/UA (#33 boundary).
	for _, forbidden := range []string{"pdfaid", "pdfuaid", "pdfaExtension"} {
		if strings.Contains(outStr, forbidden) {
			t.Errorf("PDF 2.0 metadata contains forbidden claim token %q", forbidden)
		}
	}
}

func TestPDF20InfoAndOutlineUTF8(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF20})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF20): %v", err)
	}

	doc.SetInfo("Title", "Annual Report — 2026")
	page := doc.AddPage(200, 200)
	doc.SetOutline(&Outline{
		Title:    "root",
		Children: []*Outline{{Title: "Section — Overview", PageRef: page.ref.String()}},
	})

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write: %v", err)
	}

	outStr := buf.String()

	// Info is kept on 2.0 (deprecated, not removed) with the policy Producer.
	if !strings.Contains(outStr, "/Producer (gowkhtmltopdf 2.0)") {
		t.Error("PDF 2.0 Info dictionary missing /Producer (gowkhtmltopdf 2.0)")
	}

	// The em dash pushes the title outside Latin-1, so it must be a UTF-8
	// text string: BOM EF BB BF + UTF-8 bytes, in hex form.
	if !strings.Contains(outStr, "/Title <EFBBBF") {
		t.Errorf("PDF 2.0 Info Title must be a UTF-8 hex string, got:\n%s", outStr)
	}

	// U+2014 is E2 80 94 in UTF-8.
	if !strings.Contains(outStr, "E28094") {
		t.Errorf("PDF 2.0 Title missing UTF-8 encoding of U+2014 (E28094):\n%s", outStr)
	}

	// It must not have been folded to '?' or a WinAnsi/PDFDocEncoding byte.
	if strings.Contains(outStr, "/Title (Annual Report ? 2026)") {
		t.Error("PDF 2.0 title was corrupted with '?'")
	}

	if strings.Contains(outStr, "/Title (Annual Report \\227 2026)") {
		t.Error("PDF 2.0 title was folded to Latin-1 instead of UTF-8")
	}

	// Outline titles use the same 2.0 text-string rule.
	if !strings.Contains(outStr, "<< /Title <EFBBBF") {
		t.Errorf("PDF 2.0 outline item Title must be a UTF-8 hex string:\n%s", outStr)
	}
}

func TestPDF20EmptyDocFails(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF20})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF20): %v", err)
	}

	var buf bytes.Buffer
	if err := doc.Write(&buf); !errors.Is(err, errPDFNoPages) {
		t.Fatalf("empty PDF 2.0 Write err = %v, want errPDFNoPages", err)
	}

	if buf.Len() != 0 {
		t.Errorf("empty PDF 2.0 Write produced %d bytes, want 0", buf.Len())
	}
}

// TestPDF20ShortWriterContract keeps the fails-closed writer guarantee on
// 2.0: a sink that silently short-writes must surface io.ErrShortWrite so
// xref offsets can never describe a truncated file.
func TestPDF20ShortWriterContract(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF20})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF20): %v", err)
	}

	doc.AddPage(200, 200)

	sw := &shortWriterMock{limit: 50, wrote: 0}
	if err := doc.Write(sw); !errors.Is(err, io.ErrShortWrite) {
		t.Errorf("doc.Write with short writer err = %v, want io.ErrShortWrite", err)
	}

	doc2, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF20})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF20): %v", err)
	}

	doc2.AddPage(200, 200)

	sw2 := &shortWriterMock{limit: 50, wrote: 0}
	if _, err := doc2.WriteTo(sw2); !errors.Is(err, io.ErrShortWrite) {
		t.Errorf("doc2.WriteTo with short writer err = %v, want io.ErrShortWrite", err)
	}
}

//nolint:cyclop,funlen // xref validation over a multi-object 2.0 document
func TestPDF20XrefOffsets(t *testing.T) {
	t.Parallel()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF20})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF20): %v", err)
	}

	doc.SetCreationTime(time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC))
	doc.SetInfo("Title", "PDF 2.0 Xref Offsets Test")

	page1 := doc.AddPage(300, 400)
	content1 := page1.Content()
	content1.UseEmbeddedFont("F1", fnt)
	content1.BeginText()
	content1.SetFont("F1", 12)
	content1.TextAt(20, 350)
	content1.TextShow("PDF 2.0 page one")
	content1.EndText()

	if err := content1.AddPNGImage("Im1", 20, 200, 50, 50, makePNG(t, false)); err != nil {
		t.Fatalf("AddPNGImage: %v", err)
	}

	page1.AddLinkURI([4]float64{20, 150, 120, 170}, "https://example.com/20")

	page2 := doc.AddPage(300, 400)
	content2 := page2.Content()
	content2.UseEmbeddedFont("F1", fnt)
	content2.BeginText()
	content2.SetFont("F1", 12)
	content2.TextAt(20, 350)
	content2.TextShow("PDF 2.0 page two")
	content2.EndText()

	doc.SetOutline(&Outline{
		Title: "root",
		Children: []*Outline{
			{Title: "Section 1", PageRef: doc.PageRef(0), X: 20, Y: 350},
			{Title: "Section 2", PageRef: doc.PageRef(1), X: 20, Y: 350},
		},
	})

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write: %v", err)
	}

	out := buf.Bytes()
	outStr := string(out)

	// Classic xref table only — no xref streams, no /Type /XRef.
	if !strings.Contains(outStr, "xref\n0 ") || !strings.Contains(outStr, "0000000000 65535 f \n") {
		t.Error("PDF 2.0 missing classic xref table")
	}

	if strings.Contains(outStr, "/Type /XRef") || strings.Contains(outStr, "/ObjStm") {
		t.Error("PDF 2.0 output must not contain xref/object streams")
	}

	lines := strings.Split(outStr, "\n")

	xrefIdx := findLine(lines)
	if xrefIdx < 0 {
		t.Fatal("no xref section found in PDF 2.0")
	}

	startxref := -1

	for i := xrefIdx + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "startxref") {
			startxref = i

			break
		}
	}

	if startxref < 0 {
		t.Fatal("no startxref found in PDF 2.0")
	}

	offsets := parseXrefEntries(t, lines, xrefIdx, startxref)
	if len(offsets) == 0 {
		t.Fatal("no xref entries parsed")
	}

	for obj, off := range offsets {
		want := strconv.Itoa(obj) + " 0 obj"
		if !bytes.HasPrefix(out[off:], []byte(want)) {
			t.Errorf("PDF 2.0 object %d offset %d does not start with %q", obj, off, want)
		}
	}
}

//nolint:dupl // PDF20 Type0 mirror of the 1.4/1.7 test; assert content is identical apart from version tokens
func TestType0CJKEmbeddingPDF20(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(droidSansFallbackPath)
	if err != nil {
		t.Skip("system CJK font not available:", err)
	}

	fVal, err := ParseTTF(data)
	if err != nil {
		t.Fatal(err)
	}

	fVal.PostScriptName = droidSansFallbackPSName

	dVal, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF20})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF20): %v", err)
	}

	dVal.SetCompression(false)
	p := dVal.AddPage(400, 200)
	cur := p.Content()
	cur.UseEmbeddedFont("F1", fVal)
	cur.BeginText()
	cur.SetFont("F1", 14)
	cur.TextAt(20, 100)
	cur.TextShow("你好世界")
	cur.EndText()

	out := string(writePDF(t, dVal))
	for _, want := range []string{
		"%PDF-2.0",
		"/Subtype /Type0",
		"/CIDFontType2",
		"/Encoding /Identity-H",
		"/ToUnicode",
		"begincmap",
		"beginbfchar",
		"<4F60597D4E16754C> Tj",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in PDF 2.0 Type0 output", want)
		}
	}
}

//nolint:cyclop // Latin/CJK fallback checks plus /FL resource assertions
func TestType0MixedLatinFallbackPDF20(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(droidSansFallbackPath)
	if err != nil {
		t.Skip("system CJK font not available:", err)
	}

	fVal, err := ParseTTF(data)
	if err != nil {
		t.Fatal(err)
	}

	fVal.PostScriptName = droidSansFallbackPSName

	dVal, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF20})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF20): %v", err)
	}

	dVal.SetCompression(false)
	p := dVal.AddPage(500, 200)
	cur := p.Content()
	cur.UseEmbeddedFont("F0", fVal)
	cur.BeginText()
	cur.SetFont("F0", 14)
	cur.TextAt(20, 100)
	cur.TextShow("Hello 中文 world")
	cur.EndText()

	out := string(writePDF(t, dVal))
	if !strings.HasPrefix(out, "%PDF-2.0") {
		t.Errorf("expected %%PDF-2.0 header")
	}

	if strings.Contains(out, "/FL_u") {
		t.Fatal("Latin fallback must not grow a Type0 sibling FL_u")
	}

	if !strings.Contains(out, "/F0_u") {
		t.Fatal("expected Type0 sibling F0_u for CJK run")
	}

	if !strings.Contains(out, "/FL ") {
		t.Fatal("expected Liberation Latin fallback resource FL")
	}

	if !strings.Contains(out, "(Hello )") && !strings.Contains(out, "(Hello)") {
		t.Fatal("expected simple-show Latin run")
	}

	if !strings.Contains(out, "<4E2D6587>") {
		t.Fatal("expected Identity-H CIDs for 中文")
	}
}

func TestJPEGAndPNGImagesPDF20(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF20})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF20): %v", err)
	}

	doc.SetCompression(false)

	page := doc.AddPage(200, 200)

	// JPEG stays a DCTDecode pass-through.
	if err := page.Content().AddJPEGImage("J1", 10, 10, 50, 30, makeJPEG(t)); err != nil {
		t.Fatalf("AddJPEGImage: %v", err)
	}

	// PNG with alpha keeps Flate + /SMask; DeviceRGB, no ICC rewrite.
	if err := page.Content().AddPNGImage("P1", 80, 10, 50, 30, makePNG(t, true)); err != nil {
		t.Fatalf("AddPNGImage: %v", err)
	}

	out := string(writePDF(t, doc))
	for _, want := range []string{
		"%PDF-2.0",
		"/Filter /DCTDecode",
		"/ColorSpace /DeviceRGB",
		"/SMask",
		"/J1 Do",
		"/P1 Do",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in PDF 2.0 image output", want)
		}
	}

	if strings.Contains(out, "/ICCBased") || strings.Contains(out, "/OutputIntents") {
		t.Error("PDF 2.0 image output must not contain ICC/OutputIntent (deferred to #33)")
	}

	// Page /Resources on 2.0 list real resources only — no /ProcSet entry
	// (the "/CIDInit /ProcSet findresource" text inside ToUnicode CMaps is
	// stream content, not a resource dict entry, and stays legal).
	if strings.Contains(out, "<< /ProcSet ") {
		t.Errorf("PDF 2.0 page resources must not contain /ProcSet:\n%s", out)
	}
}

func TestGrayscaleJPEGFoldPDF20(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF20})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF20): %v", err)
	}

	doc.SetGrayscale(true)
	doc.SetCompression(false)

	p := doc.AddPage(100, 100)
	if err := p.Content().AddJPEGImage("J1", 10, 10, 50, 30, makeJPEG(t)); err != nil {
		t.Fatal(err)
	}

	out := string(writePDF(t, doc))
	if !strings.Contains(out, "/ColorSpace /DeviceGray") {
		t.Error("grayscale JPEG in PDF 2.0 must be embedded as /DeviceGray")
	}

	if strings.Contains(out, "/ColorSpace /DeviceRGB") {
		t.Error("grayscale JPEG in PDF 2.0 must not stay /DeviceRGB")
	}
}

// buildPDF20RichDoc assembles a 2.0 document exercising graphics, opacity,
// text, JPEG + PNG, link annotations, and an outline hierarchy.
//
//nolint:funlen // one fixture must exercise every 2.0 emit path
func buildPDF20RichDoc(t *testing.T, fnt *Font) []byte {
	t.Helper()

	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF20})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF20): %v", err)
	}

	doc.SetCreationTime(time.Date(2026, 8, 14, 16, 0, 0, 0, time.UTC))
	doc.SetInfo("Title", "Rich Document — PDF 2.0 Test")

	page1 := doc.AddPage(600, 800)
	content1 := page1.Content()

	content1.Save()
	content1.SetFillColor(0.2, 0.4, 0.8)
	content1.Rect(50, 700, 200, 50)
	content1.Fill()
	content1.SetLineWidth(2.0)
	content1.MoveTo(50, 680)
	content1.LineTo(250, 680)
	content1.Stroke()
	content1.Restore()

	content1.Save()
	content1.SetOpacity(0.65)
	content1.SetFillColor(0.9, 0.2, 0.2)
	content1.Rect(70, 710, 100, 30)
	content1.Fill()
	content1.Restore()

	content1.UseEmbeddedFont("F1", fnt)
	content1.BeginText()
	content1.SetFont("F1", 14)
	content1.TextAt(50, 600)
	content1.TextShow("PDF 2.0 Text Heading")
	content1.EndText()

	if err := content1.AddJPEGImage("J1", 50, 450, 100, 80, makeJPEG(t)); err != nil {
		t.Fatalf("AddJPEGImage: %v", err)
	}

	if err := content1.AddPNGImage("P1", 200, 450, 100, 80, makePNG(t, true)); err != nil {
		t.Fatalf("AddPNGImage: %v", err)
	}

	page1.AddLinkURI([4]float64{50, 600, 250, 620}, "https://example.com/pdf20")
	page1.AddLinkDest([4]float64{50, 450, 150, 530}, 1, 50, 750)

	page2 := doc.AddPage(600, 800)
	content2 := page2.Content()
	content2.UseEmbeddedFont("F1", fnt)
	content2.BeginText()
	content2.SetFont("F1", 12)
	content2.TextAt(50, 750)
	content2.TextShow("Destination Page 2 Content")
	content2.EndText()

	doc.SetOutline(&Outline{
		Title: "root",
		Children: []*Outline{
			{Title: "Chapter 1 — Introduction", PageRef: page1.ref.String(), X: 50, Y: 750},
			{
				Title:   "Chapter 2 — Details",
				PageRef: page2.ref.String(),
				X:       50,
				Y:       750,
				Children: []*Outline{
					{Title: "Section 2.1 — Sub-topic", PageRef: page2.ref.String(), X: 50, Y: 600},
				},
			},
		},
	})

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write: %v", err)
	}

	return buf.Bytes()
}

//nolint:cyclop,funlen // comprehensive 2.0 structure assertions over one rich document
func TestPDF20RichDocument(t *testing.T) {
	t.Parallel()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatalf("DefaultFont: %v", err)
	}

	outBytes := buildPDF20RichDoc(t, fnt)
	outStr := string(outBytes)

	// Header, trailer /ID, catalog, and metadata.
	if !strings.HasPrefix(outStr, "%PDF-2.0\n%\xe2\xe3\xcf\xd3\n") {
		t.Errorf("invalid header prefix in PDF 2.0 rich doc")
	}

	if !strings.Contains(outStr, "trailer\n<< /Size ") || !strings.Contains(outStr, "/ID [ <") {
		t.Errorf("PDF 2.0 rich doc trailer missing /ID or invalid structure")
	}

	if !strings.Contains(outStr, "/Type /Catalog") ||
		!strings.Contains(outStr, "/Metadata ") ||
		!strings.Contains(outStr, "/Outlines ") {
		t.Errorf("PDF 2.0 catalog missing /Metadata or /Outlines")
	}

	if !strings.Contains(outStr, "/Type /Metadata /Subtype /XML") {
		t.Errorf("PDF 2.0 metadata stream missing /Type /Metadata /Subtype /XML")
	}

	if !strings.Contains(outStr, "<pdf:Producer>gowkhtmltopdf 2.0</pdf:Producer>") {
		t.Errorf("PDF 2.0 XMP missing correct Producer")
	}

	// Page resources: real resources only, ExtGState opacity present,
	// /ProcSet absent.
	for _, expectedRes := range []string{
		"/Font <<",
		"/XObject <<",
		"/ExtGState << /opacity << /CA 0.65 /ca 0.65 >> >>",
	} {
		if !strings.Contains(outStr, expectedRes) {
			t.Errorf("missing page resource entry %q", expectedRes)
		}
	}

	if strings.Contains(outStr, "<< /ProcSet ") {
		t.Errorf("PDF 2.0 page resources must not contain /ProcSet:\n%s", outStr)
	}

	// Annotations and outlines.
	wantURIAnnot := "/Subtype /Link /Rect [50 600 250 620] /Border [0 0 0] /F 4 " +
		"/A << /S /URI /URI (https://example.com/pdf20) >>"
	if !strings.Contains(outStr, wantURIAnnot) {
		t.Errorf("PDF 2.0 URI annotation missing or malformed")
	}

	if !strings.Contains(outStr, "/Type /Outlines") || !strings.Contains(outStr, "/PageMode /UseOutlines") {
		t.Errorf("PDF 2.0 outlines or /PageMode /UseOutlines missing")
	}

	// Outline titles with em dashes use UTF-8 text strings.
	if !strings.Contains(outStr, "<< /Title <EFBBBF") {
		t.Errorf("PDF 2.0 outline titles must be UTF-8 hex strings:\n%s", outStr)
	}

	// Semantic view reports version 2.0 and both pages.
	sem, err := ParseSemantic(outBytes)
	if err != nil {
		t.Fatalf("ParseSemantic(richDoc): %v", err)
	}

	if sem.Version != versionToken20 {
		t.Errorf("SemanticDoc.Version = %q, want %q", sem.Version, versionToken20)
	}

	if len(sem.Pages) != 2 {
		t.Fatalf("SemanticDoc.Pages count = %d, want 2", len(sem.Pages))
	}

	// Determinism across separate builds.
	if out2 := buildPDF20RichDoc(t, fnt); !bytes.Equal(outBytes, out2) {
		t.Error("two builds of the same PDF 2.0 rich document are not byte-identical")
	}
}

func TestPDF20FontCacheKeysVersionIndependent(t *testing.T) {
	t.Parallel()

	// Font subset bytes never consult the document version (fonts.go /
	// subset.go / fonttype0.go read no policy), so the subset cache key
	// stays version-independent by design: a 1.4 and a 2.0 document with
	// the same face and rune set produce the same key and identical
	// FontFile2 stream bytes.
	fnt, err := DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	build := func(policy WriterPolicy) (string, []byte) {
		docObj, err := NewDocumentWithPolicy(policy)
		if err != nil {
			t.Fatalf("NewDocumentWithPolicy(%v): %v", policy.Version, err)
		}

		docObj.SetCompression(false)
		p := docObj.AddPage(200, 200)
		cur := p.Content()
		cur.UseEmbeddedFont("F1", fnt)
		cur.BeginText()
		cur.SetFont("F1", 12)
		cur.TextAt(10, 20)
		cur.TextShow("hello")
		cur.EndText()

		out := writePDF(t, docObj)

		return docObj.fontKeys["F1"], out
	}

	key14, out14 := build(WriterPolicy{Version: PDF14})
	key20, out20 := build(WriterPolicy{Version: PDF20})

	if key14 == "" || key20 == "" {
		t.Fatal("subset cache key not populated for F1")
	}

	if key14 != key20 {
		t.Errorf("font cache key diverges between 1.4 (%q) and 2.0 (%q); emit bytes must not depend on version",
			key14, key20)
	}

	fontStream14 := fontFile2Stream(out14)
	fontStream20 := fontFile2Stream(out20)

	if !bytes.Equal(fontStream14, fontStream20) {
		t.Error("subset FontFile2 bytes differ between 1.4 and 2.0 documents")
	}
}

// fontFile2Stream returns the raw bytes of the first embedded FontFile2 stream.
func fontFile2Stream(out []byte) []byte {
	str := string(out)

	idx := strings.Index(str, "/FontFile2")
	if idx < 0 {
		return nil
	}

	sm := strings.Index(str[idx:], "\nstream\n")
	if sm < 0 {
		return nil
	}

	data := out[idx+sm+len("\nstream\n"):]

	end := bytes.Index(data, []byte("\nendstream"))
	if end < 0 {
		return nil
	}

	return data[:end]
}
