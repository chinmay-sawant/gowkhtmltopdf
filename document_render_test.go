//nolint:exhaustruct,wsl,testpackage,usetesting,lll // same-package tests exercise the native adapter boundary.
package gowkhtmltopdf

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDocumentPDFAndImageProduceMagic(t *testing.T) {
	t.Parallel()

	document := NewDocument(Page{Source: HTML([]byte("<html><body><h1>hello</h1></body></html>"), "")})
	pdf, err := document.PDF(context.Background())
	if err != nil {
		t.Fatalf("PDF: %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("PDF prefix = %q", pdf[:min(len(pdf), 8)])
	}

	image, err := (&ImageDocument{Source: HTML([]byte("<html><body><h1>hello</h1></body></html>"), ""), Format: "png"}).Image(context.Background())
	if err != nil {
		t.Fatalf("Image: %v", err)
	}
	if !bytes.HasPrefix(image, []byte("\x89PNG\r\n\x1a\n")) {
		t.Fatalf("image is not PNG")
	}
}

func TestDocumentOutputAndOutlinePreflight(t *testing.T) {
	t.Parallel()

	document := NewDocument(Page{Source: File("report.html")})
	if err := document.WritePDF(context.Background(), nil); !errors.Is(err, ErrMissingPDFOutput) {
		t.Fatalf("nil output error = %v", err)
	}
	if err := document.WritePDFOutline(context.Background(), &bytes.Buffer{}, nil); !errors.Is(err, ErrMissingPDFOutlineOutput) {
		t.Fatalf("nil outline error = %v", err)
	}
	if err := document.WritePDF(context.Background(), &bytes.Buffer{}); err == nil {
		t.Fatal("expected file load error for nonexistent report")
	}
}

func TestDocumentOutlineAndContextCancellation(t *testing.T) {
	t.Parallel()

	document := NewDocument(Page{Source: HTML([]byte("<html><body><h1>Chapter</h1></body></html>"), "")})
	var pdfOutput bytes.Buffer
	var outlineOutput bytes.Buffer
	if err := document.WritePDFOutline(t.Context(), &pdfOutput, &outlineOutput); err != nil {
		t.Fatalf("WritePDFOutline: %v", err)
	}
	if !bytes.Contains(outlineOutput.Bytes(), []byte("<outline")) {
		t.Fatalf("outline output = %q, want outline XML", outlineOutput.Bytes())
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := document.WritePDF(ctx, &bytes.Buffer{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled WritePDF error = %v, want context.Canceled", err)
	}
}

func TestDocumentAdapterCopiesHTMLAtExecutionBoundary(t *testing.T) {
	t.Parallel()

	source := []byte("<html><body>before</body></html>")
	document := NewDocument(Page{Source: HTML(source, "")})
	source[15] = 'X'

	if err := document.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if got := string(document.Pages[0].Source.HTML); got == string(source) {
		t.Fatal("document source aliases caller bytes")
	}
}

func TestDocumentFontPathsReachPDF(t *testing.T) {
	t.Parallel()

	fontDir := t.TempDir()
	src := filepath.Join("internal", "pdf", "assets", "LiberationSans-Regular.ttf")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read asset: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fontDir, "Face.ttf"), data, 0o600); err != nil {
		t.Fatalf("write face: %v", err)
	}

	document := NewDocument(Page{Source: HTML([]byte(
		`<html><body style="font-family: 'Liberation Sans', sans-serif;"><p>DocFontPath</p></body></html>`,
	), "")})
	document.FontPaths = []string{fontDir}
	document.AllowLocalFiles = true

	pdf, err := document.PDF(t.Context())
	if err != nil {
		t.Fatalf("PDF: %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatal("not a PDF")
	}
	if !bytes.Contains(pdf, []byte("/FontFile2")) {
		t.Fatal("expected embedded font")
	}
}

func TestDocumentUseSystemFontsOption(t *testing.T) {
	t.Parallel()

	document := NewDocument(Page{Source: HTML([]byte(`<html><body><p>sys</p></body></html>`), "")})
	document.UseSystemFonts = true

	if err := document.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	pdf, err := document.PDF(t.Context())
	if err != nil {
		t.Fatalf("PDF with UseSystemFonts: %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatal("not a PDF")
	}
}
