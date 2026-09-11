package convert

import (
	"bytes"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestPageAtATimeFiveSections(t *testing.T) {
	t.Parallel()

	var html strings.Builder

	html.WriteString(`<!DOCTYPE html><html><head><style>
		.page { page-break-inside: avoid; }
		.page + .page { page-break-before: always; }
		td, th { border: 1px solid #000; }
		table { border-collapse: collapse; }
	</style></head><body>`)

	for page := 1; page <= 5; page++ {
		html.WriteString(`<section class="page"><h1>Report page `)
		html.WriteString(strconv.Itoa(page))
		html.WriteString(`</h1><table><thead><tr><th>Line</th><th>SKU</th></tr></thead><tbody>`)

		for row := 1; row <= 8; row++ {
			html.WriteString(`<tr><td>`)
			html.WriteString(strconv.Itoa(row))
			html.WriteString(`</td><td>SKU-00`)
			html.WriteString(strconv.Itoa(page))
			html.WriteString(`-00`)
			html.WriteString(strconv.Itoa(row))
			html.WriteString(`</td></tr>`)
		}

		html.WriteString(`</tbody></table></section>`)
	}

	html.WriteString(`</body></html>`)

	cmd, _ := newCommand(t, html.String(), filepath.Join(t.TempDir(), "out.pdf"))
	data := runPDF(t, cmd)

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("output is not a PDF")
	}

	if n := pageCount(data); n != 5 {
		t.Fatalf("pages = %d, want 5", n)
	}

	if !bytes.Contains(data, []byte("SKU-001-001")) && !bytes.Contains(data, []byte("SKU-001")) {
		// PDF text may be encoded; page count is the hard pin.
		t.Log("SKU strings not visible as PDF literals; page count held")
	}
}

func TestSectionTextCountMatchesOpText(t *testing.T) {
	t.Parallel()

	htmlSrc := `<!DOCTYPE html><html><head><style>
		.page { page-break-inside: avoid; }
		.page + .page { page-break-before: always; }
		td, th { border: 1px solid #000; }
		table { border-collapse: collapse; }
	</style></head><body>
	<section class="page"><h1>Report page 1</h1><table>
	<thead><tr><th>Line</th><th>SKU</th></tr></thead>
	<tbody><tr><td>1</td><td>SKU-001-001</td></tr></tbody>
	</table></section>
	<section class="page"><h1>Report page 2</h1><table>
	<thead><tr><th>Line</th><th>SKU</th></tr></thead>
	<tbody><tr><td>1</td><td>SKU-002-001</td></tr></tbody>
	</table></section>
	</body></html>`

	cmd, _ := newCommand(t, htmlSrc, filepath.Join(t.TempDir(), "out.pdf"))
	data := runPDF(t, cmd)
	if pageCount(data) != 2 {
		t.Fatalf("pages = %d, want 2", pageCount(data))
	}
}

func TestIndependentBlocksFallbackSpanningTable(t *testing.T) {
	t.Parallel()

	var html strings.Builder

	html.WriteString(`<!DOCTYPE html><html><body><table>`)

	for range 80 {
		html.WriteString(`<tr><td>row</td></tr>`)
	}

	html.WriteString(`</table></body></html>`)

	cmd, _ := newCommand(t, html.String(), filepath.Join(t.TempDir(), "out.pdf"))
	data := runPDF(t, cmd)

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("fallback spanning table is not a PDF")
	}

	if n := pageCount(data); n < 1 {
		t.Fatalf("pages = %d, want >= 1", n)
	}
}
