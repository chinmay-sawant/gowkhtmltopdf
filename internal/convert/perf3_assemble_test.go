package convert

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

func TestAssembleSkipsEmptyChrome(t *testing.T) {
	t.Parallel()

	doc := pdf.NewDocument()
	doc.AddPage(595, 842)
	doc.AddPage(595, 842)

	state := &objectState{}
	state.pages = 2
	plan, err := newPagePlan(nil, []*objectState{state}, 1, false)

	if err != nil {
		t.Fatalf("newPagePlan: %v", err)
	}

	req := &Request{
		Now: func() time.Time {
			t.Fatal("now() must not run when header and footer have no content")

			return time.Time{}
		},
	}

	result := drawHeadersFootersResult(t.Context(), nil, doc, req, plan, nil)
	if err := result.Err(); err != nil {
		t.Fatalf("empty chrome Err = %v, want nil", err)
	}
}

func TestAssembleEmptyChromeProducesPDF(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html><html><body>
<table style="width:100%;border-collapse:collapse">
<thead><tr><th>Line</th><th>SKU</th></tr></thead>
<tbody>
<tr><td>1</td><td>SKU-0001</td></tr>
<tr><td>2</td><td>SKU-0002</td></tr>
</tbody>
</table>
<p style="page-break-before:always">page two</p>
</body></html>`

	cmd, _ := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))
	data := runPDF(t, cmd)

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("output is not a PDF")
	}

	if n := pageCount(data); n < 2 {
		t.Fatalf("pages = %d, want >= 2", n)
	}
}

func TestAssembleObjectHeaderStillDraws(t *testing.T) {
	t.Parallel()

	cmd, _ := newCommand(t, `<html><body><p>body</p></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Objects[0].HeaderSet = true
	cmd.Objects[0].Header.Left = "OBJHDR"
	cmd.Global.UseCompression = false

	data := runPDF(t, cmd)
	if !bytes.Contains(data, []byte("(OBJHDR) Tj")) {
		t.Fatal("object-level header was skipped")
	}
}
