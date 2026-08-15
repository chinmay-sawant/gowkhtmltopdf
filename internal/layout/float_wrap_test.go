//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// After a float ends, subsequent lines must reclaim full content width
// (CSS2.1 line-box shortening), not stay narrow for the whole paragraph.
func TestFloatTextReclaimsFullWidthBelow(t *testing.T) {
	t.Parallel()

	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
		0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x05, 0xfe, 0xd4, 0xef, 0x00, 0x00,
		0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}

	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
p { margin: 0; }
figure { float: left; width: 80pt; margin: 0 8pt 4pt 0; }
img { display: block; width: 80pt; height: 60pt; }
`)
	// One long paragraph that starts beside the float and continues below it.
	words := strings.Repeat("word ", 80)

	root, err := html.Parse(`<html><body>
<figure><img width="80" height="60" src="t.png"><figcaption>c</figcaption></figure>
<p>` + words + `</p>
</body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 360, Height: 400, Sheets: []*css.Stylesheet{cssSheet}, Background: true,
		Images: func(string) ([]byte, error) { return png, nil },
	})
	if err != nil {
		t.Fatal(err)
	}

	var texts []Op

	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "word") {
			texts = append(texts, op)
		}
	}

	if len(texts) < 4 {
		t.Fatalf("expected multi-line text, got %d", len(texts))
	}
	// Group by approximate Y; last line should start near x=0 (full width),
	// first line should start right of the float (~88pt).
	first := texts[0]
	last := texts[len(texts)-1]

	if first.X < 50 {
		t.Fatalf("first line x=%.1f, want beside float (>=50)", first.X)
	}

	if last.X > 20 {
		t.Fatalf("last line x=%.1f, want full-width reclaim (~0) after float", last.X)
	}
}

// Short end-of-paragraph tails must not sit as orphans in the narrow column
// beside a float when they fit as one full-width line under the float.
func TestFloatOrphanTailClearsBelow(t *testing.T) {
	t.Parallel()

	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
		0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x05, 0xfe, 0xd4, 0xef, 0x00, 0x00,
		0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}

	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
p { margin: 0; }
figure { float: left; width: 100pt; margin: 0 8pt 4pt 0; }
img { display: block; width: 100pt; height: 120pt; }
`)
	// Enough text to wrap several lines beside a tall float, ending with a
	// short marker that must reclaim full width under the float rather than
	// sit alone in the narrow column (wiki "big time."[71]).
	lead := strings.Repeat("words beside the float image ", 18)

	root, err := html.Parse(`<html><body>
<figure><img width="100" height="120" src="t.png"><figcaption>cap</figcaption></figure>
<p>` + lead + `big time."[71]</p>
</body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 360, Height: 500, Sheets: []*css.Stylesheet{cssSheet}, Background: true,
		Images: func(string) ([]byte, error) { return png, nil },
	})
	if err != nil {
		t.Fatal(err)
	}

	var tail *Op

	for i := range res.Ops {
		op := &res.Ops[i]
		if op.Kind == OpText && strings.Contains(op.Text, "big") {
			tail = op
		}
	}

	if tail == nil {
		t.Fatal("missing big time tail text")
	}
	// Full-width reclaim: left edge near content origin, not stuck at ~108pt
	// (float width + margin).
	if tail.X > 25 {
		t.Fatalf("orphan tail x=%.1f, want full-width under float (~0), not beside float", tail.X)
	}
}
