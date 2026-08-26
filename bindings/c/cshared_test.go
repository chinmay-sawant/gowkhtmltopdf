//go:build cgo

package main

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

const (
	testImageWidth  = 512
	testBadCopies   = 1001
	testFreeLoop    = 500
	testUnsupported = int32(2)
	testWrongSize   = int64(12345)
)

// basePDFOptions returns the all-defaults PDF option set without tripping
// the exhaustiveness linter on empty literals.
func basePDFOptions() pdfOptions {
	var opts pdfOptions

	return opts
}

// baseImageOptions returns the all-defaults image option set.
func baseImageOptions() imageOptions {
	var opts imageOptions

	return opts
}

// requirePDFShape asserts the structural PDF contract used by the golden
// corpus (%PDF- header, EOF marker, xref table, embedded subset font, at
// least one parseable page) and returns the extracted document text.
func requirePDFShape(t *testing.T, data []byte) string {
	t.Helper()

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatalf("output does not start with %%PDF-")
	}
	for _, needle := range []string{"%%EOF", "xref", "/FontFile2"} {
		if !bytes.Contains(data, []byte(needle)) {
			t.Errorf("output missing %q", needle)
		}
	}

	doc, err := pdf.ParseSemantic(data)
	if err != nil {
		t.Fatalf("ParseSemantic: %v", err)
	}
	if doc.PageCount() < 1 {
		t.Errorf("page count = %d, want >= 1", doc.PageCount())
	}

	return doc.DocumentText()
}

func TestCSharedInlineHelloPDF(t *testing.T) {
	status, data, message := runPDFWithContext(
		t.Context(), []byte("<!DOCTYPE html><h1>Hello</h1>"), basePDFOptions())

	if status != statusOK {
		t.Fatalf("status = %d, want %d (%s)", status, statusOK, message)
	}

	text := requirePDFShape(t, data)
	if !strings.Contains(text, "Hello") {
		t.Errorf("extracted text missing Hello; text=%q", text)
	}
}

func TestCSharedInvoiceFixture(t *testing.T) {
	html, err := os.ReadFile("../../testdata/golden/fixture-01-simple-invoice.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	status, data, message := runPDFWithContext(t.Context(), html, basePDFOptions())
	if status != statusOK {
		t.Fatalf("status = %d, want %d (%s)", status, statusOK, message)
	}

	text := requirePDFShape(t, data)
	if !strings.Contains(text, "Invoice") {
		t.Errorf("extracted text missing Invoice; text=%q", text)
	}
}

func TestCSharedOptionValidation(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*pdfOptions)
		want   int32
	}{
		{
			name:   "a4 page size converts",
			mutate: func(o *pdfOptions) { o.pageSize = "A4" },
			want:   statusOK,
		},
		{
			name:   "bogus page size rejected",
			mutate: func(o *pdfOptions) { o.pageSize = "bogus" },
			want:   statusInvalidArg,
		},
		{
			name:   "copies over limit rejected",
			mutate: func(o *pdfOptions) { o.copies = testBadCopies },
			want:   statusInvalidArg,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := basePDFOptions()
			tc.mutate(&opts)

			status, _, message := runPDFWithContext(
				t.Context(), []byte("<h1>options</h1>"), opts)
			if status != tc.want {
				t.Fatalf("status = %d, want %d (%s)", status, tc.want, message)
			}
		})
	}
}

func TestCSharedAbiGate(t *testing.T) {
	html := []byte("<h1>gate</h1>")

	if got := exportedABI(); got != int32(abiVersionValue) {
		t.Fatalf("abi version = %d, want %d", got, abiVersionValue)
	}

	status, data, message := probePDFExport(html, testUnsupported, -1)
	assertRejected(t, "abi_version", status, data, message)

	status, data, message = probePDFExport(html, int32(abiVersionValue), testWrongSize)
	assertRejected(t, "struct_size", status, data, message)

	status, data, message = probeStampedPDFExport(html)
	if status != statusOK || len(data) == 0 || message != "" {
		t.Fatalf("stamped struct: status = %d, len = %d (%s)", status, len(data), message)
	}

	requirePDFShape(t, data)
}

// assertRejected checks the failure conventions for one ABI gate probe.
func assertRejected(t *testing.T, label string, status int32, data []byte, message string) {
	t.Helper()

	if status != statusInvalidArg {
		t.Fatalf("%s: status = %d, want %d", label, status, statusInvalidArg)
	}
	if data != nil {
		t.Errorf("%s: expected empty payload on rejection", label)
	}
	if message == "" {
		t.Fatalf("%s: expected non-empty diagnostic", label)
	}
}

func TestCSharedNilHTMLRejected(t *testing.T) {
	var nilHTML []byte

	status, data, message := invokePDFExport(nilHTML, nil)

	assertRejected(t, "nil html", status, data, message)
}

func TestCSharedImagePNG(t *testing.T) {
	opts := baseImageOptions()
	opts.width = testImageWidth
	opts.format = "png"

	status, data, message := runImageWithContext(
		t.Context(), []byte("<h1>Badge</h1>"), opts)

	if status != statusOK {
		t.Fatalf("status = %d, want %d (%s)", status, statusOK, message)
	}
	if !bytes.HasPrefix(data, []byte("\x89PNG")) {
		t.Errorf("expected PNG signature, got %q", data[:min(8, len(data))])
	}
}

func TestCSharedCancelledContextTimesOut(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	status, data, message := runPDFWithContext(ctx, []byte("<h1>slow</h1>"), basePDFOptions())
	if status != statusTimeout {
		t.Fatalf("status = %d, want %d (%s)", status, statusTimeout, message)
	}
	if data != nil {
		t.Errorf("expected nil payload on cancellation")
	}
	if message == "" {
		t.Errorf("expected a diagnostic message on cancellation")
	}
}

func TestCSharedFreePairingLoop(t *testing.T) {
	if detail := smokeFreePairingLoop([]byte("<!DOCTYPE html><h1>Hello</h1>"),
		testFreeLoop); detail != "" {
		t.Fatalf("free pairing loop failed: %s", detail)
	}
}

func TestCSharedVersionAndLastError(t *testing.T) {
	if got := exportedVersion(); got != libVersion {
		t.Errorf("version = %q, want %q", got, libVersion)
	}

	opts := basePDFOptions()
	opts.pageSize = "bogus"

	status, _, _ := runPDFWithContext(t.Context(), []byte("<h1>x</h1>"), opts)
	if status != statusInvalidArg {
		t.Fatalf("status = %d, want %d", status, statusInvalidArg)
	}

	want := currentLastError()
	if lastErrorLength() == 0 || want == "" {
		t.Fatal("last error slot empty after failure")
	}

	buf := make([]byte, lastErrorLength()+1)
	written := copyLastErrorInto(buf)
	if written <= 0 || string(buf[:written]) != want {
		t.Errorf("buffer copy = %q, want %q", buf[:max(written, 0)], want)
	}

	if got := readLastErrorViaExport(); got != want {
		t.Errorf("exported reader = %q, want %q", got, want)
	}
}
