package convert

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// TestBlinklessEngineFixture01 converts fixture-01 through the blinkless
// engine and asserts the structural contract the native golden path demands:
// PDF structure, exactly one page, an embedded TrueType subset, and the
// invoice text. This is the end-to-end proof for `--engine blinkless`.
func TestBlinklessEngineFixture01(t *testing.T) {
	t.Parallel()

	req := requestForFixture(t, "fixture-01-simple-invoice.html")
	req.Global.Engine = settings.EngineBlinkless

	data := runPDFWithLog(t, req, io.Discard)

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatalf("PDF prefix = %q", data[:min(len(data), 8)])
	}

	if !bytes.Contains(data, []byte("%%EOF")) {
		t.Fatal("PDF is missing its EOF marker")
	}

	if got := pageCount(data); got != 1 {
		t.Errorf("pages = %d, want 1", got)
	}

	if !bytes.Contains(data, []byte("/FontFile2")) {
		t.Error("no embedded TrueType subset (/FontFile2)")
	}

	semantic, err := pdf.ParseSemantic(data)
	if err != nil {
		t.Fatalf("ParseSemantic: %v", err)
	}

	text := semantic.DocumentText()

	for _, needle := range []string{"Acme Widgets", "Invoice", "234.40 EUR"} {
		if !strings.Contains(text, needle) {
			t.Errorf("document text missing %q", needle)
		}
	}
}

// TestBlinklessEngineFixture67CornerShapes converts fixture 67 through the
// blinkless engine and asserts its contract: PDF structure, the pinned page
// count, an embedded subset, and the audit needles. The shaped geometry
// itself is proven in internal/blinkengine (TestLayoutMapsCornerShapePath);
// this run proves the whole 34-example fixture converts end to end.
func TestBlinklessEngineFixture67CornerShapes(t *testing.T) {
	t.Parallel()

	req := requestForFixture(t, "fixture-67-coordinate-shapes.html")
	req.Global.Engine = settings.EngineBlinkless

	data := runPDFWithLog(t, req, io.Discard)

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatalf("PDF prefix = %q", data[:min(len(data), 8)])
	}

	if !bytes.Contains(data, []byte("%%EOF")) {
		t.Fatal("PDF is missing its EOF marker")
	}

	if got := pageCount(data); got != 6 {
		t.Errorf("pages = %d, want 6", got)
	}

	if !bytes.Contains(data, []byte("/FontFile2")) {
		t.Error("no embedded TrueType subset (/FontFile2)")
	}

	semantic, err := pdf.ParseSemantic(data)
	if err != nil {
		t.Fatalf("ParseSemantic: %v", err)
	}

	text := semantic.DocumentText()

	for _, needle := range []string{"COORDINATE-SHAPES", "corner-top-left-shape", "end-end = bottom-right"} {
		if !strings.Contains(text, needle) {
			t.Errorf("document text missing %q", needle)
		}
	}
}

// TestBlinklessEngineFixture68MaskBorderRegionsShapes converts fixture 68
// through the blinkless engine and asserts its contract: PDF structure, the
// pinned page count, an embedded subset, and the audit needles. The live
// regions and shapes effects are proven by the fixture's conversion plus the
// layout probes in internal/blinkengine; this run proves the whole 26-example
// fixture converts end to end. The mask-border rows are descriptor-only by
// design (this host does not replay mask pixels), so their presence is a text
// needle, not a painted frame.
func TestBlinklessEngineFixture68MaskBorderRegionsShapes(t *testing.T) {
	t.Parallel()

	req := requestForFixture(t, "fixture-68-mask-border-regions-shapes.html")
	req.Global.Engine = settings.EngineBlinkless

	data := runPDFWithLog(t, req, io.Discard)

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatalf("PDF prefix = %q", data[:min(len(data), 8)])
	}

	if !bytes.Contains(data, []byte("%%EOF")) {
		t.Fatal("PDF is missing its EOF marker")
	}

	if got := pageCount(data); got != 4 {
		t.Errorf("pages = %d, want 4", got)
	}

	if !bytes.Contains(data, []byte("/FontFile2")) {
		t.Error("no embedded TrueType subset (/FontFile2)")
	}

	semantic, err := pdf.ParseSemantic(data)
	if err != nil {
		t.Fatalf("ParseSemantic: %v", err)
	}

	text := semantic.DocumentText()

	for _, needle := range []string{
		"MASK-BORDER-REGIONS-SHAPES",
		"mask-border-source",
		"flow-into",
		"wrap-inside",
		"shape-inside",
		"shape-padding",
		"shape-image-threshold",
		"float-defer",
	} {
		if !strings.Contains(text, needle) {
			t.Errorf("document text missing %q", needle)
		}
	}
}

// TestBlinklessEngineAppliesLinkedStylesheet proves the bridge fetches a
// <link rel=stylesheet> with this module's loader and hands the CSS to
// blinkless: the external sheet hides the heading, so the rendered text must
// not contain it. Without the bridge the link is silently dropped and the
// heading stays visible.
func TestBlinklessEngineAppliesLinkedStylesheet(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	pagePath := filepath.Join(dir, "index.html")
	pageHTML := `<!DOCTYPE html><html><head>` +
		`<link rel="stylesheet" href="style.css">` +
		`</head><body><h1>Hidden heading</h1><p>Visible body</p></body></html>`

	if err := os.WriteFile(pagePath, []byte(pageHTML), 0o600); err != nil {
		t.Fatalf("write page: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "style.css"), []byte("h1 { display: none }"), 0o600); err != nil {
		t.Fatalf("write stylesheet: %v", err)
	}

	global := settings.DefaultPdfGlobal()
	global.Load.EnableLocalFileAccess = true
	global.Engine = settings.EngineBlinkless

	obj := settings.DefaultPdfObject()
	obj.Page = pagePath
	obj.Load.BlockLocalFileAccess = false

	data := runPDF(t, NewPDFRequest(global, []settings.PdfObject{obj}, nil, nil))

	semantic, err := pdf.ParseSemantic(data)
	if err != nil {
		t.Fatalf("ParseSemantic: %v", err)
	}

	text := semantic.DocumentText()

	if strings.Contains(text, "Hidden heading") {
		t.Errorf("linked stylesheet was not applied: heading still present in %q", text)
	}

	if !strings.Contains(text, "Visible body") {
		t.Errorf("document text missing body copy: %q", text)
	}
}

// TestBlinklessEngineRejectsUnknownName proves the request gate: an engine
// name outside native|blinkless fails validation before any loading.
func TestBlinklessEngineRejectsUnknownName(t *testing.T) {
	t.Parallel()

	req := requestForFixture(t, "fixture-01-simple-invoice.html")
	req.Global.Engine = "webkit"

	var out bytes.Buffer

	req.Output = &out

	err := Run(t.Context(), req, io.Discard, nil)
	if err == nil {
		t.Fatal("Run accepted an unknown engine")
	}

	if !strings.Contains(err.Error(), "invalid engine") {
		t.Fatalf("err = %v, want invalid engine", err)
	}
}
