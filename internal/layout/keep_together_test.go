//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// TestImplicitKeepTogetherCallout: an aside with its own chrome that starts
// near the page end and fits one page must move wholly, not split across the
// boundary. Fixture-56 .dom-notes cards were the motivating case.
func TestImplicitKeepTogetherCallout(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
div.spacer { height: 760pt; }
aside.note {
  background: #fff8eb;
  border-left: 4px solid #d97706;
  padding: 12px;
  margin: 0;
}
p { margin: 6px 0; }
`)
	res := layoutHTML(t, `<html><body>
<div class="spacer">x</div>
<aside class="note">
<p>KEEP-TOGETHER line one with enough words to wrap.</p>
<p>KEEP-TOGETHER line two with enough words to wrap.</p>
<p>KEEP-TOGETHER line three with enough words to wrap.</p>
<p>KEEP-TOGETHER line four with enough words to wrap.</p>
</aside>
</body></html>`, cssSheet)

	if err := Paint(pdf.NewDocument(), res, paintOpts()); err != nil {
		t.Fatal(err)
	}

	var pages []int

	for i, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "KEEP-TOGETHER") {
			pages = append(pages, pageOfIdx(t, res, i))
		}
	}

	if len(pages) < 2 {
		t.Fatalf("callout text ops = %d, want ≥2", len(pages))
	}

	for _, page := range pages[1:] {
		if page != pages[0] {
			t.Fatalf("callout split across pages %v", pages)
		}
	}

	if pages[0] != 1 {
		t.Fatalf("callout landed on page %d, want 1 (pushed off page 0)", pages[0])
	}
}

// TestPlainNestedBlockStillSplits: a generic wrapper without chrome is not
// an implicit keep-together unit; its children may straddle as before.
func TestPlainNestedBlockStillSplits(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
div.spacer { height: 760pt; }
div.plain { margin: 0; padding: 0; }
p { margin: 8px 0; }
`)
	res := layoutHTML(t, `<html><body>
<div class="spacer">x</div>
<div class="plain">
<p>PLAIN-SPLIT line one with enough words to wrap.</p>
<p>PLAIN-SPLIT line two with enough words to wrap.</p>
<p>PLAIN-SPLIT line three with enough words to wrap.</p>
<p>PLAIN-SPLIT line four with enough words to wrap.</p>
<p>PLAIN-SPLIT line five with enough words to wrap.</p>
<p>PLAIN-SPLIT line six with enough words to wrap.</p>
</div>
</body></html>`, cssSheet)

	if err := Paint(pdf.NewDocument(), res, paintOpts()); err != nil {
		t.Fatal(err)
	}

	seen := map[int]int{}

	for i, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "PLAIN-SPLIT") {
			seen[pageOfIdx(t, res, i)]++
		}
	}

	if len(seen) < 2 {
		t.Fatalf("plain wrapper stayed on one page %v, want a split", seen)
	}
}
