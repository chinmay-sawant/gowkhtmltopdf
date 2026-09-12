package gowkhtmltopdf_test

import (
	"bytes"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

const (
	perf3OutputBytesPin = 1420537
	perf3PageCountPin   = 500
)

// TestPerf3OutputBytesPin is the PERF3-03 equivalence pin for the 500-page
// report on the generic library path. Document.WritePDF builds convert.NewPDFRequest
// (document.go). HTML is the same report fixture the public benchmark uses
// (libraryBenchmarkReportHTML).
func TestPerf3OutputBytesPin(t *testing.T) {
	t.Parallel()

	document := libraryBenchmarkPDFDocument(perf3PageCountPin)

	var output bytes.Buffer
	if err := document.WritePDF(t.Context(), &output); err != nil {
		t.Fatalf("Document.WritePDF: %v", err)
	}

	pdfBytes := output.Bytes()
	if !bytes.HasPrefix(pdfBytes, []byte("%PDF-")) {
		t.Fatal("output is missing the %PDF- header")
	}

	if got := len(pdfBytes); got != perf3OutputBytesPin {
		t.Fatalf("output-bytes = %d, want %d", got, perf3OutputBytesPin)
	}

	doc, err := pdf.ParseSemantic(pdfBytes)
	if err != nil {
		t.Fatalf("parse semantic PDF: %v", err)
	}

	if got := doc.PageCount(); got != perf3PageCountPin {
		t.Fatalf("page count = %d, want %d", got, perf3PageCountPin)
	}

	if err := validateLibraryPDFOutput(pdfBytes, perf3PageCountPin); err != nil {
		t.Fatalf("ordered text needles: %v", err)
	}
}
