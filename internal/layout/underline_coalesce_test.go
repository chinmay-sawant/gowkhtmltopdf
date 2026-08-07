//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"math"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

// countHorizUnderlines returns OpLine strokes that look like text underlines
// (horizontal, non-zero width, zero height).
func countHorizUnderlines(ops []Op) int {
	node := 0

	for _, op := range ops {
		if op.Kind == OpLine && op.H == 0 && op.W > 0.5 {
			node++
		}
	}

	return node
}

// TestUnderlineCoalesceMultiFaceSameHref: bold + normal chunks inside one
// <a href> on a single line must produce ONE underline OpLine, not one per
// nested style item / face run.
func TestUnderlineCoalesceMultiFaceSameHref(t *testing.T) { //nolint:cyclop
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
a { color: #0645ad; text-decoration: underline; }
b { font-weight: 700; }
`)
	// Nested <b> + following text share href but not sameInlineStyle, so
	// coalesceTextItems leaves two items — underline coalescing must still
	// emit a single stroke for the logical link run.
	root, err := html.Parse(`<html><body><p>` +
		`<a href="https://example.com/page"><b>Bold</b> title text here</a></p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 500, Height: 200, Sheets: []*css.Stylesheet{cssSheet}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}

	var textOps int

	for _, op := range res.Ops {
		if op.Kind == OpText && strings.TrimSpace(op.Text) != "" {
			textOps++
		}
	}

	if textOps < 2 {
		t.Fatalf("expected ≥2 non-empty text ops (bold+normal), got %d", textOps)
	}

	n := countHorizUnderlines(res.Ops)
	if n != 1 {
		t.Fatalf("same-href multi-chunk line: want 1 underline OpLine, got %d (textOps=%d)", n, textOps)
	}
	// Stroke must be capped for print density.
	for _, op := range res.Ops {
		if op.Kind == OpLine && op.H == 0 && op.W > 0.5 {
			if op.Width < 0.25-1e-6 || op.Width > 0.45+1e-6 {
				t.Errorf("underline Width=%.3f outside [0.25, 0.45]", op.Width)
			}
		}
	}
}

// TestUnderlineCoalesceHrefForceAcrossChunks: PDF affordance underlines
// (href set, author decoration none) also coalesce multi-chunk same-href.
func TestUnderlineCoalesceHrefForceAcrossChunks(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 11pt; }
a { color: #0645ad; text-decoration: none; }
i { font-style: italic; }
`)

	root, err := html.Parse(`<html><body><p><a href="https://web.archive.org/web/2020/https://example.com/long-path">` +
		`<i>Archive</i> https://web.archive.org/web/2020/https://example.com/long-path</a></p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 600, Height: 200, Sheets: []*css.Stylesheet{cssSheet}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Single line (wide viewport): expect one underline for the whole link.
	n := countHorizUnderlines(res.Ops)
	if n != 1 {
		t.Fatalf("href-force multi-chunk one line: want 1 underline, got %d", n)
	}
}

// TestUnderlineSkipWhitespaceOnly: a lone space item must not invent its own
// underline stroke; spaces inside a link extend the active run instead.
func TestUnderlineSkipWhitespaceOnly(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
a { color: blue; text-decoration: underline; }
`)
	// Pretty-printed: text nodes "Hello", whitespace, "World" inside one <a>.
	root, err := html.Parse(`<html><body><p><a href="https://example.com/x">Hello
World</a></p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 400, Height: 100, Sheets: []*css.Stylesheet{cssSheet}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}

	n := countHorizUnderlines(res.Ops)
	if n != 1 {
		t.Fatalf("link with internal whitespace: want 1 underline, got %d", n)
	}
}

// TestUnderlineStrokeWidthClamp: large and small font sizes stay in [0.25, 0.45].
func TestUnderlineStrokeWidthClamp(t *testing.T) { //nolint:cyclop
	t.Parallel()

	cases := []struct {
		name string
		css  string
	}{
		{"small", `a { text-decoration: underline; font-size: 8pt; }`},
		{"large", `a { text-decoration: underline; font-size: 24pt; }`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cssSheet := sheet(t, tc.css)

			root, err := html.Parse(`<html><body><a href="https://example.com">gyp</a></body></html>`)
			if err != nil {
				t.Fatal(err)
			}

			res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
				Width: 200, Height: 80, Sheets: []*css.Stylesheet{cssSheet}, Media: "print",
			})
			if err != nil {
				t.Fatal(err)
			}

			found := false

			for _, op := range res.Ops {
				if op.Kind == OpLine && op.H == 0 && op.W > 0 {
					found = true

					if op.Width < 0.25-1e-9 || op.Width > 0.45+1e-9 {
						t.Fatalf("Width=%.4f outside clamp", op.Width)
					}
				}
			}

			if !found {
				t.Fatal("no underline")
			}
		})
	}
}

// TestUnderlineStrokeWidthUnit matches helper directly.
func TestUnderlineStrokeWidthUnit(t *testing.T) {
	t.Parallel()

	if g := underlineStrokeWidth(4); math.Abs(g-0.25) > 1e-9 {
		t.Errorf("small em: got %.3f want 0.25", g)
	}

	if g := underlineStrokeWidth(20); math.Abs(g-0.45) > 1e-9 {
		// 20*0.05=1.0 → clamp 0.45
		t.Errorf("large em: got %.3f want 0.45", g)
	}

	if g := underlineStrokeWidth(8); math.Abs(g-0.4) > 1e-9 {
		// 8*0.05=0.4
		t.Errorf("mid em: got %.3f want 0.40", g)
	}
}

// TestUnderlineWrappedURLOnePerLine: a long bare URL that wraps still gets
// at most one underline per line (not per soft-break fragment stacked).
func TestUnderlineWrappedURLOnePerLine(t *testing.T) {
	t.Parallel()

	url := "https://web.archive.org/web/20200316084639/https://www.example.com/path/to/article-title-extra"
	cssSheet := sheet(t, `body { margin: 0; font-size: 10pt; } `+
		`a { overflow-wrap: break-word; text-decoration: underline; color: #0645ad; }`)
	src := `<html><body><p><a href="` + url + `">` + url + `</a></p></body></html>`

	root, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	const contentW = 220.0

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: contentW, Height: 800, Sheets: []*css.Stylesheet{cssSheet}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}
	// Group text ops by baseline Y → number of wrapped lines.
	type ykey int

	lines := map[ykey]bool{}

	for _, op := range res.Ops {
		if op.Kind == OpText && strings.TrimSpace(op.Text) != "" {
			lines[ykey(math.Round(op.Y*10))] = true
		}
	}

	nTextLines := len(lines)
	if nTextLines < 2 {
		t.Fatalf("expected wrapped URL (≥2 lines), got %d", nTextLines)
	}

	nUnder := countHorizUnderlines(res.Ops)
	// One underline per text line for a single bare URL link.
	if nUnder > nTextLines {
		t.Fatalf("underlines=%d > text lines=%d (face/chunk fragmentation)", nUnder, nTextLines)
	}

	if nUnder < 1 {
		t.Fatal("expected at least one underline")
	}

	t.Logf("wrapped URL: textLines=%d underlines=%d", nTextLines, nUnder)
}
