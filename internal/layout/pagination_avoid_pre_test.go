package layout

import (
	"fmt"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// Programiz code card (real-sites evidence 2026-09-16, programiz-cpp row 6):
// pre{page-break-inside:avoid} split at the page boundary. The card top and
// the h4 title stayed on the previous page while the code text snapped to the
// next one, leaving an empty bordered card top behind.
//
// A pre taller than a third of the content height trips the "large explicit
// avoid box" split preference, so this fixture keeps the pre above that band
// (40 content-height percent) to pin the real shape.
func TestPreAvoidInsideMovesWholeToNextPage(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
div.filler { height: 780pt; }
pre { margin: 0; line-height: 17pt; border: 1px solid #999999; page-break-inside: avoid; }
`)

	lines := make([]string, 20)
	for i := range lines {
		lines[i] = fmt.Sprintf("PRE-LINE-%02d", i)
	}

	src := `<html><body><div class="filler">filler</div><pre>` +
		strings.Join(lines, "\n") + `</pre></body></html>`

	res := layoutHTML(t, src, cssSheet)
	doc := pdf.NewDocument()

	if err := Paint(doc, res, paintOpts()); err != nil {
		t.Fatal(err)
	}

	pages := textOpPages(t, res, "PRE-LINE-")

	if len(pages) != 1 {
		t.Fatalf("pre text on %d pages %v, want the whole pre on one page", len(pages), pages)
	}

	if _, ok := pages[1]; !ok {
		t.Fatalf("pre text pages = %v, want page 1 (pushed whole off page 0)", pages)
	}

	// The pre chrome must land with the text: no part of the pre box may stay
	// behind on page 0.
	pre := findBox(t, res, "pre")
	for i := pre.opStart; i <= pre.opEnd && i < len(res.Ops); i++ {
		if res.Ops[i].Kind != OpText && pageOfIdx(t, res, i) == 0 {
			t.Fatalf("pre chrome op %d still on page 0 after the avoid-inside move", i)
		}
	}
}

// textOpPages counts, per page, the text ops whose text starts with prefix.
func textOpPages(t *testing.T, res *Result, prefix string) map[int]int {
	t.Helper()

	pages := map[int]int{}

	for i, paintOp := range res.Ops {
		if paintOp.Kind == OpText && strings.HasPrefix(paintOp.Text, prefix) {
			pages[pageOfIdx(t, res, i)]++
		}
	}

	return pages
}
