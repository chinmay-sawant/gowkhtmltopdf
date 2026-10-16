//nolint:wsl,usetesting // same-package tests exercise the native adapter boundary.
package gowkhtmltopdf

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// TestDocumentPDFBlinklessEngine proves the public library path: a Document
// with Engine "blinkless" converts HTML through the blinkless engine and
// still produces a valid PDF with the document text.
func TestDocumentPDFBlinklessEngine(t *testing.T) {
	t.Parallel()

	document := NewDocument(Page{Source: HTML([]byte(
		"<html><body><h1>Blinkless path</h1><p>Total: 234.40 EUR</p></body></html>"), "")})
	document.Engine = "blinkless"

	out, err := document.PDF(context.Background())
	if err != nil {
		t.Fatalf("PDF: %v", err)
	}

	if !bytes.HasPrefix(out, []byte("%PDF-")) {
		t.Fatalf("PDF prefix = %q", out[:min(len(out), 8)])
	}

	semantic, err := pdf.ParseSemantic(out)
	if err != nil {
		t.Fatalf("ParseSemantic: %v", err)
	}

	text := semantic.DocumentText()

	for _, needle := range []string{"Blinkless path", "234.40 EUR"} {
		if !strings.Contains(text, needle) {
			t.Errorf("document text missing %q", needle)
		}
	}
}

func TestDocumentValidateRejectsUnknownEngine(t *testing.T) {
	t.Parallel()

	document := NewDocument(Page{Source: HTML([]byte("<p>x</p>"), "")})
	document.Engine = "webkit"

	if err := document.Validate(); !errors.Is(err, settings.ErrInvalidEngine) {
		t.Fatalf("Validate err = %v, want invalid engine", err)
	}
}
