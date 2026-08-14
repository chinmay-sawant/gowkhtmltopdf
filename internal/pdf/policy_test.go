//nolint:testpackage,exhaustruct // tests verify internal serialization and policy behavior
package pdf

import (
	"bytes"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

//nolint:funlen // table-driven policy validation cases
func TestPolicyValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		policy  WriterPolicy
		wantErr error
	}{
		{
			name:    "default zero value is valid PDF14",
			policy:  WriterPolicy{}, //nolint:exhaustruct // testing zero-value behavior
			wantErr: nil,
		},
		{
			name:    "explicit PDF14 is valid",
			policy:  WriterPolicy{Version: PDF14},
			wantErr: nil,
		},
		{
			name:    "PDF17 is valid",
			policy:  WriterPolicy{Version: PDF17},
			wantErr: nil,
		},
		{
			name:    "PDF20 returns reserved error mentioning issue 32",
			policy:  WriterPolicy{Version: PDF20},
			wantErr: ErrReservedPDF20,
		},
		{
			name:    "negative version returns unsupported",
			policy:  WriterPolicy{Version: PDFVersion(-1)},
			wantErr: ErrUnsupportedPDFVersion,
		},
		{
			name:    "future version beyond PDF20 returns unsupported",
			policy:  WriterPolicy{Version: PDFVersion(99)},
			wantErr: ErrUnsupportedPDFVersion,
		},
		{
			name:    "encryption requested returns ErrEncryptionUnsupported",
			policy:  WriterPolicy{Version: PDF17, Encryption: true},
			wantErr: ErrEncryptionUnsupported,
		},
		{
			name:    "forms requested returns ErrFormsUnsupported",
			policy:  WriterPolicy{Version: PDF17, Forms: true},
			wantErr: ErrFormsUnsupported,
		},
		{
			name:    "signatures requested returns ErrSignaturesUnsupported",
			policy:  WriterPolicy{Version: PDF17, Signatures: true},
			wantErr: ErrSignaturesUnsupported,
		},
		{
			name:    "object streams requested returns ErrObjectStreamsUnsupported",
			policy:  WriterPolicy{Version: PDF17, ObjectStreams: true},
			wantErr: ErrObjectStreamsUnsupported,
		},
		{
			name:    "conformance profile PDF/A-1b requested returns ErrPDFA1Unsupported",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: "PDF/A-1b"},
			wantErr: ErrPDFA1Unsupported,
		},
		{
			name:    "conformance profile unknown string returns ErrUnknownConformanceProfile",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: "PDF/X-3"},
			wantErr: ErrUnknownConformanceProfile,
		},
		{
			name:    "conformance profile PDF/A-4 requested returns ErrConformanceProfilesUnsupported",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: "PDF/A-4"},
			wantErr: ErrConformanceProfilesUnsupported,
		},
		{
			name:    "conformance profile PDF/UA-2 requested returns ErrConformanceProfilesUnsupported",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: "PDF/UA-2"},
			wantErr: ErrConformanceProfilesUnsupported,
		},
		{
			name:    "conformance profile PDF/UA-1 on PDF17 is valid",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: ProfilePDFUA1},
			wantErr: nil,
		},
		{
			name:    "conformance profile PDF/A-3a on PDF17 is valid",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: ProfilePDFA3a},
			wantErr: nil,
		},
		{
			name:    "conformance profile Dual on PDF17 is valid",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: ProfileDualA3aUA1},
			wantErr: nil,
		},
		{
			name:    "conformance profile PDF/UA-1 on PDF14 returns ErrProfileRequiresPDF17",
			policy:  WriterPolicy{Version: PDF14, ConformanceProfile: ProfilePDFUA1},
			wantErr: ErrProfileRequiresPDF17,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := testCase.policy.Validate()
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("Validate() err = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

func TestPolicyHeaderAndProducerStrings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		policy       WriterPolicy
		wantHeader   string
		wantProducer string
	}{
		{
			policy:       WriterPolicy{}, //nolint:exhaustruct // zero value
			wantHeader:   "1.4",
			wantProducer: "gowkhtmltopdf 1.4",
		},
		{
			policy:       WriterPolicy{Version: PDF14},
			wantHeader:   "1.4",
			wantProducer: "gowkhtmltopdf 1.4",
		},
		{
			policy:       WriterPolicy{Version: PDF17},
			wantHeader:   versionToken17,
			wantProducer: "gowkhtmltopdf 1.7",
		},
		{
			policy:       WriterPolicy{Version: PDF20},
			wantHeader:   "2.0",
			wantProducer: "gowkhtmltopdf 2.0",
		},
	}

	for _, testCase := range tests {
		if got := testCase.policy.HeaderVersion(); got != testCase.wantHeader {
			t.Errorf("HeaderVersion(%v) = %q, want %q", testCase.policy.Version, got, testCase.wantHeader)
		}

		if got := testCase.policy.ProducerVersion(); got != testCase.wantProducer {
			t.Errorf("ProducerVersion(%v) = %q, want %q", testCase.policy.Version, got, testCase.wantProducer)
		}
	}
}

func TestNewDocumentWithPolicy(t *testing.T) {
	t.Parallel()

	doc14, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF14})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF14) error: %v", err)
	}

	if doc14.Policy().Version != PDF14 {
		t.Errorf("doc14.Policy().Version = %v, want %v", doc14.Policy().Version, PDF14)
	}

	doc17, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF17) error: %v", err)
	}

	if doc17.Policy().Version != PDF17 {
		t.Errorf("doc17.Policy().Version = %v, want %v", doc17.Policy().Version, PDF17)
	}

	if _, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF20}); !errors.Is(err, ErrReservedPDF20) {
		t.Fatalf("NewDocumentWithPolicy(PDF20) err = %v, want %v", err, ErrReservedPDF20)
	}

	if _, err := NewDocumentWithPolicy(WriterPolicy{Version: PDFVersion(-1)}); !errors.Is(err, ErrUnsupportedPDFVersion) {
		t.Fatalf("NewDocumentWithPolicy(invalid) err = %v, want %v", err, ErrUnsupportedPDFVersion)
	}

	defaultDoc := NewDocument()
	if defaultDoc.Policy().Version != PDF14 {
		t.Errorf("NewDocument().Policy().Version = %v, want %v", defaultDoc.Policy().Version, PDF14)
	}
}

func TestPDF17HeaderEmissionAndSemantic(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy: %v", err)
	}

	doc.AddPage(200, 200)

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write: %v", err)
	}

	out := buf.Bytes()
	wantPrefix := "%PDF-1.7\n%\xe2\xe3\xcf\xd3\n"

	if !strings.HasPrefix(string(out), wantPrefix) {
		t.Errorf("PDF 1.7 output prefix mismatch:\ngot:  %q\nwant: %q",
			string(out[:min(len(out), len(wantPrefix)+10)]), wantPrefix)
	}

	sem, err := ParseSemantic(out)
	if err != nil {
		t.Fatalf("ParseSemantic: %v", err)
	}

	if sem.Version != versionToken17 {
		t.Errorf("SemanticDoc.Version = %q, want %q", sem.Version, versionToken17)
	}
}

//nolint:cyclop,funlen // tests multiple deterministic trailer /ID properties in sequence
func TestTrailerIDBehavior(t *testing.T) {
	t.Parallel()

	// 1. PDF 1.4 document must not have /ID in trailer.
	doc14 := NewDocument()
	doc14.AddPage(200, 200)

	var buf14 bytes.Buffer
	if err := doc14.Write(&buf14); err != nil {
		t.Fatalf("doc14.Write: %v", err)
	}

	str14 := buf14.String()
	trailerIdx14 := strings.Index(str14, "trailer\n")

	if trailerIdx14 < 0 {
		t.Fatal("doc14 missing trailer")
	}

	if strings.Contains(str14[trailerIdx14:], "/ID") {
		t.Errorf("PDF 1.4 trailer should not contain /ID, found: %s", str14[trailerIdx14:])
	}

	// 2. PDF 1.7 document must have /ID [ <hex> <hex> ] with two identical 32-char hex values.
	fixedTime := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	makeDoc17 := func(title string) *Document {
		docObj, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
		if err != nil {
			t.Fatalf("NewDocumentWithPolicy: %v", err)
		}

		docObj.SetCreationTime(fixedTime)
		docObj.SetInfo("Title", title)
		docObj.AddPage(300, 400)

		return docObj
	}

	doc17A := makeDoc17("Doc A")

	var buf17A bytes.Buffer
	if err := doc17A.Write(&buf17A); err != nil {
		t.Fatalf("doc17A.Write: %v", err)
	}

	str17A := buf17A.String()
	trailerIdx17 := strings.Index(str17A, "trailer\n")

	if trailerIdx17 < 0 {
		t.Fatal("doc17A missing trailer")
	}

	idRe := regexp.MustCompile(`/ID\s*\[\s*<([0-9A-Fa-f]{32})>\s*<([0-9A-Fa-f]{32})>\s*\]`)
	matches := idRe.FindStringSubmatch(str17A[trailerIdx17:])

	if len(matches) != 3 {
		t.Fatalf("PDF 1.7 trailer /ID regex did not match:\n%s", str17A[trailerIdx17:])
	}

	if matches[1] != matches[2] {
		t.Errorf("trailer /ID elements not equal: %q vs %q", matches[1], matches[2])
	}

	// 3. Determinism: identical docs produce identical bytes.
	doc17B := makeDoc17("Doc A")

	var buf17B bytes.Buffer
	if err := doc17B.Write(&buf17B); err != nil {
		t.Fatalf("doc17B.Write: %v", err)
	}

	if !bytes.Equal(buf17A.Bytes(), buf17B.Bytes()) {
		t.Error("two writes of the same PDF 1.7 document are not byte-identical")
	}

	// 4. Changing title changes the ID.
	doc17C := makeDoc17("Doc Different Title")

	var buf17C bytes.Buffer
	if err := doc17C.Write(&buf17C); err != nil {
		t.Fatalf("doc17C.Write: %v", err)
	}

	matchesC := idRe.FindStringSubmatch(buf17C.String())
	if len(matchesC) != 3 {
		t.Fatalf("doc17C trailer /ID regex did not match")
	}

	if matchesC[1] == matches[1] {
		t.Errorf("expected different IDs for different titles, got same ID: %s", matchesC[1])
	}
}

//nolint:cyclop,funlen // tests complete catalog and XMP packet structure in detail
func TestCatalogAndMetadataStream(t *testing.T) {
	t.Parallel()

	fixedTime := time.Date(2026, 8, 14, 15, 30, 0, 0, time.UTC)

	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy: %v", err)
	}

	doc.SetCreationTime(fixedTime)
	doc.SetInfo("Title", "Test Document")
	doc.AddPage(200, 200)

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write: %v", err)
	}

	outStr := buf.String()

	// Verify catalog has /Metadata reference.
	catRe := regexp.MustCompile(`<< /Type /Catalog /Metadata (\d+ 0 R) /Pages \d+ 0 R >>`)
	catMatch := catRe.FindStringSubmatch(outStr)

	if len(catMatch) != 2 {
		t.Fatalf("catalog does not match expected pattern with /Metadata:\n%s", outStr)
	}

	metaRef := catMatch[1]
	metaObjHeader := strings.Replace(metaRef, " 0 R", " 0 obj", 1)

	if !strings.Contains(outStr, metaObjHeader) {
		t.Fatalf("metadata object %s not found in PDF output", metaObjHeader)
	}

	// Verify /Type /Metadata /Subtype /XML.
	if !strings.Contains(outStr, "/Type /Metadata /Subtype /XML") {
		t.Error("metadata object missing /Type /Metadata /Subtype /XML")
	}

	// Verify XMP content elements.
	if !strings.Contains(outStr, `<?xpacket begin="`+"\xef\xbb\xbf"+`" id="W5M0MpCehiHzreSzNTczkc9d"?>`) {
		t.Error("XMP header missing or malformed")
	}

	if !strings.Contains(outStr, "<dc:format>application/pdf</dc:format>") {
		t.Error("XMP missing dc:format")
	}

	if !strings.Contains(outStr, "<pdf:Producer>gowkhtmltopdf 1.7</pdf:Producer>") {
		t.Error("XMP missing pdf:Producer")
	}

	if !strings.Contains(outStr, "<xmp:CreateDate>2026-08-14T15:30:00Z</xmp:CreateDate>") {
		t.Error("XMP missing correct xmp:CreateDate")
	}

	if !strings.Contains(outStr, "<rdf:li xml:lang=\"x-default\">Test Document</rdf:li>") {
		t.Error("XMP missing dc:title")
	}

	if !strings.Contains(outStr, "<?xpacket end=\"w\"?>") {
		t.Error("XMP missing closing xpacket")
	}

	// Negative assertion: Must not contain PDF/A or PDF/UA claims.
	for _, forbidden := range []string{"pdfaid", "pdfuaid", "pdfaExtension"} {
		if strings.Contains(outStr, forbidden) {
			t.Errorf("PDF 1.7 metadata contains forbidden claim token %q", forbidden)
		}
	}

	// Catalog must NOT contain /Version or /Extensions.
	if strings.Contains(catMatch[0], "/Version") || strings.Contains(catMatch[0], "/Extensions") {
		t.Errorf("Catalog dictionary must not contain /Version or /Extensions, got: %s", catMatch[0])
	}
}

//nolint:cyclop // tests Info dictionary and outline UTF-16BE / Latin-1 paths
func TestPDF17InfoAndOutlineUnicodeUTF16BE(t *testing.T) {
	t.Parallel()

	// 1. PDF 1.7 with Unicode title containing em dash U+2014.
	doc17, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy: %v", err)
	}

	doc17.SetInfo("Title", "Annual Report — 2026")
	page := doc17.AddPage(200, 200)

	doc17.SetOutline(&Outline{ //nolint:exhaustruct // intentional zero-value fields
		Title: "root",
		Children: []*Outline{
			{Title: "Section — Overview", PageRef: page.ref.String()},
		},
	})

	var buf17 bytes.Buffer
	if err := doc17.Write(&buf17); err != nil {
		t.Fatalf("doc17.Write: %v", err)
	}

	str17 := buf17.String()

	// Em dash U+2014 in UTF-16BE is 0x2014.
	// "Annual Report — 2026" contains \u2014, so it must be formatted as <FEFF...> hex string.
	if !strings.Contains(str17, "/Title <FEFF") {
		t.Errorf("PDF 1.7 info dict Title should be encoded as UTF-16BE hex string, got:\n%s", str17)
	}

	// Check that em dash 2014 is present in the hex string and not folded to ? or Latin1 \227.
	if strings.Contains(str17, "/Title (Annual Report ? 2026)") {
		t.Errorf("PDF 1.7 title was corrupted with '?'")
	}

	if strings.Contains(str17, "/Title (Annual Report \\227 2026)") {
		t.Errorf("PDF 1.7 title was folded to Latin-1 instead of UTF-16BE")
	}

	// Check outline Title is also UTF-16BE hex.
	if !strings.Contains(str17, "<< /Title <FEFF") {
		t.Errorf("PDF 1.7 outline item Title should be encoded as UTF-16BE hex string, got:\n%s", str17)
	}

	// Check Producer is gowkhtmltopdf 1.7.
	if !strings.Contains(str17, "/Producer (gowkhtmltopdf 1.7)") {
		t.Errorf("PDF 1.7 Info dictionary missing /Producer (gowkhtmltopdf 1.7)")
	}

	// 2. PDF 1.4 with em dash folds to Latin-1 \227.
	doc14 := NewDocument()
	doc14.SetInfo("Title", "Annual Report — 2026")
	doc14.AddPage(200, 200)

	var buf14 bytes.Buffer
	if err := doc14.Write(&buf14); err != nil {
		t.Fatalf("doc14.Write: %v", err)
	}

	str14 := buf14.String()
	if !strings.Contains(str14, "/Title (Annual Report \\227 2026)") {
		t.Errorf("PDF 1.4 title should fold to Latin-1 \\227, got:\n%s", str14)
	}

	if !strings.Contains(str14, "/Producer (gowkhtmltopdf 1.4)") {
		t.Errorf("PDF 1.4 Info dictionary missing /Producer (gowkhtmltopdf 1.4)")
	}
}

func TestClassicXrefAndNoPagesPDF17(t *testing.T) {
	t.Parallel()

	// Empty document should return errPDFNoPages.
	emptyDoc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy: %v", err)
	}

	var emptyBuf bytes.Buffer
	if err := emptyDoc.Write(&emptyBuf); !errors.Is(err, errPDFNoPages) {
		t.Fatalf("emptyDoc.Write err = %v, want errPDFNoPages", err)
	}

	// Document with page has classic xref table.
	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy: %v", err)
	}

	doc.AddPage(100, 100)

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write: %v", err)
	}

	str := buf.String()
	if !strings.Contains(str, "xref\n0 ") {
		t.Error("PDF 1.7 missing classic xref table")
	}

	if !strings.Contains(str, "0000000000 65535 f \n") {
		t.Error("PDF 1.7 missing free object xref entry 0")
	}

	if !strings.Contains(str, "startxref\n") || !strings.Contains(str, "%%EOF") {
		t.Errorf("PDF 1.7 missing startxref or %%%%EOF")
	}
}

//nolint:funlen // table-driven tests for all unsupported feature gates failing closed
func TestFeatureGatesFailClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		policy  WriterPolicy
		wantErr error
	}{
		{
			name:    "encryption fails closed",
			policy:  WriterPolicy{Version: PDF17, Encryption: true},
			wantErr: ErrEncryptionUnsupported,
		},
		{
			name:    "forms fail closed",
			policy:  WriterPolicy{Version: PDF17, Forms: true},
			wantErr: ErrFormsUnsupported,
		},
		{
			name:    "signatures fail closed",
			policy:  WriterPolicy{Version: PDF17, Signatures: true},
			wantErr: ErrSignaturesUnsupported,
		},
		{
			name:    "object streams fail closed",
			policy:  WriterPolicy{Version: PDF17, ObjectStreams: true},
			wantErr: ErrObjectStreamsUnsupported,
		},
		{
			name:    "conformance PDF/A-1b fails closed",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: "PDF/A-1b"},
			wantErr: ErrPDFA1Unsupported,
		},
		{
			name:    "conformance PDF/A-4 fails closed",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: "PDF/A-4"},
			wantErr: ErrConformanceProfilesUnsupported,
		},
		{
			name:    "conformance PDF/UA-2 fails closed",
			policy:  WriterPolicy{Version: PDF17, ConformanceProfile: "PDF/UA-2"},
			wantErr: ErrConformanceProfilesUnsupported,
		},
		{
			name:    "conformance profile on PDF14 fails closed",
			policy:  WriterPolicy{Version: PDF14, ConformanceProfile: ProfilePDFUA1},
			wantErr: ErrProfileRequiresPDF17,
		},
		{
			name:    "PDF 2.0 reserved fails closed",
			policy:  WriterPolicy{Version: PDF20},
			wantErr: ErrReservedPDF20,
		},
		{
			name:    "invalid PDF version fails closed",
			policy:  WriterPolicy{Version: PDFVersion(99)},
			wantErr: ErrUnsupportedPDFVersion,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// 1. NewDocumentWithPolicy rejects the policy directly.
			_, err := NewDocumentWithPolicy(testCase.policy)
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("NewDocumentWithPolicy(%+v) err = %v, want %v", testCase.policy, err, testCase.wantErr)
			}

			// 2. A document constructed with NewDocument and mutated policy fails closed before WriteTo produces bytes.
			doc := NewDocument()
			doc.policy = testCase.policy
			doc.AddPage(100, 100)

			if err := doc.Validate(); !errors.Is(err, testCase.wantErr) {
				t.Errorf("doc.Validate() err = %v, want %v", err, testCase.wantErr)
			}

			var buf bytes.Buffer

			bytesWritten, writeErr := doc.WriteTo(&buf)
			if !errors.Is(writeErr, testCase.wantErr) {
				t.Fatalf("doc.WriteTo err = %v, want %v", writeErr, testCase.wantErr)
			}

			if bytesWritten != 0 {
				t.Errorf("doc.WriteTo returned bytesWritten = %d on error, want 0", bytesWritten)
			}

			if buf.Len() != 0 {
				t.Errorf("doc.WriteTo produced %d bytes in output buffer on error, want 0", buf.Len())
			}

			// 3. doc.Write also fails with the sentinel error and empty buffer.
			var bufWrite bytes.Buffer

			if err := doc.Write(&bufWrite); !errors.Is(err, testCase.wantErr) {
				t.Fatalf("doc.Write err = %v, want %v", err, testCase.wantErr)
			}

			if bufWrite.Len() != 0 {
				t.Errorf("doc.Write produced %d bytes on error, want 0", bufWrite.Len())
			}
		})
	}
}

//nolint:cyclop,funlen // xref offset validation involves building a document and checking every entry
func TestPDF17XrefOffsets(t *testing.T) {
	t.Parallel()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF17): %v", err)
	}

	doc.SetCreationTime(time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC))
	doc.SetInfo("Title", "PDF 1.7 Xref Offsets Test")

	page1 := doc.AddPage(300, 400)
	content1 := page1.Content()
	content1.UseEmbeddedFont("F1", fnt)
	content1.BeginText()
	content1.SetFont("F1", 12)
	content1.TextAt(20, 350)
	content1.TextShow("PDF 1.7 page one")
	content1.EndText()

	if err := content1.AddPNGImage("Im1", 20, 200, 50, 50, makePNG(t, false)); err != nil {
		t.Fatalf("AddPNGImage: %v", err)
	}

	page1.AddLinkURI([4]float64{20, 150, 120, 170}, "https://example.com/17")

	page2 := doc.AddPage(300, 400)
	content2 := page2.Content()
	content2.UseEmbeddedFont("F1", fnt)
	content2.BeginText()
	content2.SetFont("F1", 12)
	content2.TextAt(20, 350)
	content2.TextShow("PDF 1.7 page two")
	content2.EndText()

	doc.SetOutline(&Outline{ //nolint:exhaustruct // test outline
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
	lines := strings.Split(string(out), "\n")
	xrefIdx := findLine(lines, "xref")

	if xrefIdx < 0 {
		t.Fatal("no xref section found in PDF 1.7")
	}

	startxref := -1

	for i := xrefIdx + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "startxref") {
			startxref = i

			break
		}
	}

	if startxref < 0 {
		t.Fatal("no startxref found in PDF 1.7")
	}

	offsets := parseXrefEntries(t, lines, xrefIdx, startxref)
	if len(offsets) == 0 {
		t.Fatal("no xref entries parsed")
	}

	for obj, off := range offsets {
		want := strconv.Itoa(obj) + " 0 obj"
		if !bytes.HasPrefix(out[off:], []byte(want)) {
			t.Errorf("PDF 1.7 object %d offset %d does not start with %q", obj, off, want)
		}
	}
}

type shortWriterMock struct {
	limit int
	wrote int
}

func (s *shortWriterMock) Write(payload []byte) (int, error) {
	if s.wrote >= s.limit {
		return 0, nil
	}

	rem := s.limit - s.wrote
	bytesWrote := len(payload)

	if bytesWrote > rem {
		bytesWrote = rem
	}

	s.wrote += bytesWrote

	return bytesWrote, nil
}

func TestPDF17ShortWriterContract(t *testing.T) {
	t.Parallel()

	doc, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF17): %v", err)
	}

	doc.AddPage(200, 200)

	sw := &shortWriterMock{limit: 50, wrote: 0}
	errWrite := doc.Write(sw)

	if !errors.Is(errWrite, io.ErrShortWrite) {
		t.Errorf("doc.Write with short writer err = %v, want io.ErrShortWrite", errWrite)
	}

	// Recreate document to test WriteTo
	doc2, err := NewDocumentWithPolicy(WriterPolicy{Version: PDF17})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy(PDF17): %v", err)
	}

	doc2.AddPage(200, 200)

	sw2 := &shortWriterMock{limit: 50, wrote: 0}
	_, errWriteTo := doc2.WriteTo(sw2)

	if !errors.Is(errWriteTo, io.ErrShortWrite) {
		t.Errorf("doc2.WriteTo with short writer err = %v, want io.ErrShortWrite", errWriteTo)
	}
}

func TestDefaultNewDocumentAsserts14(t *testing.T) {
	t.Parallel()

	doc := NewDocument()
	if doc.Policy().Version != PDF14 {
		t.Fatalf("NewDocument().Policy().Version = %v, want %v", doc.Policy().Version, PDF14)
	}

	doc.SetInfo("Title", "Default 1.4 Doc")
	doc.AddPage(200, 200)

	var buf bytes.Buffer
	if err := doc.Write(&buf); err != nil {
		t.Fatalf("doc.Write: %v", err)
	}

	outStr := buf.String()

	// 1. Must start with %PDF-1.4
	if !strings.HasPrefix(outStr, "%PDF-1.4\n%\xe2\xe3\xcf\xd3\n") {
		t.Errorf("Default PDF output header mismatch: %q", outStr[:min(len(outStr), 25)])
	}

	// 2. Catalog must NOT contain /Metadata
	catRe := regexp.MustCompile(`<<\s*/Type\s*/Catalog[^>]*>>`)
	catMatch := catRe.FindString(outStr)

	if strings.Contains(catMatch, "/Metadata") {
		t.Errorf("Default PDF 1.4 catalog contains /Metadata: %s", catMatch)
	}

	// 3. Trailer must NOT contain /ID
	trailerIdx := strings.Index(outStr, "trailer\n")
	if trailerIdx < 0 {
		t.Fatal("missing trailer")
	}

	if strings.Contains(outStr[trailerIdx:], "/ID") {
		t.Errorf("Default PDF 1.4 trailer contains /ID: %s", outStr[trailerIdx:])
	}

	// 4. Producer must be gowkhtmltopdf 1.4
	if !strings.Contains(outStr, "/Producer (gowkhtmltopdf 1.4)") {
		t.Errorf("Default PDF 1.4 Info missing /Producer (gowkhtmltopdf 1.4)")
	}

	// 5. SemanticDoc.Version reports "1.4"
	sem, err := ParseSemantic(buf.Bytes())
	if err != nil {
		t.Fatalf("ParseSemantic: %v", err)
	}

	if sem.Version != "1.4" {
		t.Errorf("sem.Version = %q, want %q", sem.Version, "1.4")
	}
}
