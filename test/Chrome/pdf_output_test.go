package chrome_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

const chromePDFOutputDir = "pdf"

// TestChromeCasePDFOutputs renders every Chrome case into a local PDF so the
// current product output can be inspected alongside the case HTML. The PDF
// is intentionally not a pass/fail oracle for layout geometry. The focused
// layout tests own those assertions, including the cases currently blocked by
// known engine gaps.
func TestChromeCasePDFOutputs(t *testing.T) {
	t.Parallel()

	manifest := readManifest(t)

	if err := os.MkdirAll(chromePDFOutputDir, 0o755); err != nil {
		t.Fatalf("create PDF output directory: %v", err)
	}

	for _, item := range manifest.Cases {
		writeChromeCasePDF(t, item)
	}
}

func writeChromeCasePDF(t *testing.T, item manifestCase) {
	t.Helper()

	fixturePath, err := filepath.Abs(item.Fixture)
	if err != nil {
		t.Fatalf("resolve fixture path: %v", err)
	}

	if _, err := os.Stat(fixturePath); err != nil {
		t.Fatalf("stat fixture %q: %v", item.Fixture, err)
	}

	global := settings.DefaultPdfGlobal()
	global.Title = item.ID
	global.Load.EnableLocalFileAccess = true

	object := settings.DefaultPdfObject()
	object.Page = fixturePath
	object.Load.BlockLocalFileAccess = false

	var output bytes.Buffer
	request := convert.NewPDFRequest(
		global,
		[]settings.PdfObject{object},
		&output,
		nil,
	)

	if err := convert.Run(t.Context(), request, io.Discard, nil); err != nil {
		t.Fatalf("convert case %q: %v", item.ID, err)
	}

	document, err := pdf.ParseSemantic(output.Bytes())
	if err != nil {
		t.Fatalf("parse generated PDF for %q: %v", item.ID, err)
	}

	if document.PageCount() == 0 {
		t.Fatalf("generated PDF for %q has no pages", item.ID)
	}

	if strings.HasPrefix(item.ID, "legacy-") && strings.TrimSpace(document.DocumentText()) == "" {
		t.Fatalf("legacy case %q generated a PDF with no visible text", item.ID)
	}

	outputPath := filepath.Join(chromePDFOutputDir, item.ID+".pdf")
	if err := os.WriteFile(outputPath, output.Bytes(), 0o600); err != nil {
		t.Fatalf("write %s: %v", outputPath, err)
	}

	t.Logf("wrote %s (%d pages)", outputPath, document.PageCount())
}
