package pdf_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// TestConvertedCurlyQuotesSurviveContentAndToUnicode converts a small HTML
// document with curly quotes and walks the full pipeline. The page text must
// carry the WinAnsi curly quote bytes (0x91-0x94) and /ToUnicode must map them
// back to U+2018-U+201D, which is what extraction tools read. The old fold
// wrote ASCII bytes and mapped them to ' and ", so both the glyph and the
// extracted text were straight.
func TestConvertedCurlyQuotesSurviveContentAndToUnicode(t *testing.T) {
	t.Parallel()

	const html = `<!DOCTYPE html>
<html><head><meta charset="utf-8"><title>quotes</title></head>
<body><p>won’t “this” and ‘that’</p></body></html>`

	path := filepath.Join(t.TempDir(), "quotes.html")
	if err := os.WriteFile(path, []byte(html), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	obj := settings.DefaultPdfObject()
	obj.Page = path
	obj.Load.BlockLocalFileAccess = false

	global := settings.DefaultPdfGlobal()
	global.Load.EnableLocalFileAccess = true
	global.PageSize = "A4"
	global.Margin = settings.DefaultMargins()
	global.Background = true
	global.UseCompression = false // keep content streams and CMaps searchable

	var out bytes.Buffer

	req := convert.NewPDFRequest(global, []settings.PdfObject{obj}, &out, nil)
	if err := convert.Run(t.Context(), req, discardWriter{}, nil); err != nil {
		t.Fatalf("convert.Run: %v", err)
	}

	data := out.Bytes()

	doc, err := pdf.ParseSemantic(data)
	if err != nil {
		t.Fatalf("ParseSemantic: %v", err)
	}

	// ParseSemantic returns simple-font Tj bytes verbatim, so the page text
	// carries the WinAnsi code of each curly quote. A folded stream would
	// carry ASCII bytes here instead.
	text := doc.DocumentText()

	for _, want := range []string{"won\x92t", "\x93this\x94", "\x91that\x92"} {
		if !strings.Contains(text, want) {
			t.Errorf("page text missing WinAnsi curly quote bytes %q; text=%q", want, text)
		}
	}

	// /ToUnicode is the text layer extraction tools read: byte to curly code
	// point, so pdftotext and PyMuPDF report the curly characters.
	for _, pair := range []string{"<91> <2018>", "<92> <2019>", "<93> <201C>", "<94> <201D>"} {
		if !bytes.Contains(data, []byte(pair)) {
			t.Errorf("converted PDF missing /ToUnicode mapping %s", pair)
		}
	}
}
