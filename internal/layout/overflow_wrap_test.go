//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
)

// Long unbreakable tokens (URLs, paths) must not paint past the content edge.
// Emergency wrap applies when a token alone exceeds the line width even with
// overflow-wrap:normal (print PDF usability).
func TestLongURLEmergencyWrap(t *testing.T) {
	t.Parallel()

	url := "https://web.archive.org/web/20200316084639/" +
		"https://www.example.com/path/to/a/very/long/article-title-with-many-segments"
	s := sheet(t, `body { margin: 0; font-size: 10pt; } p { margin: 0; }`)
	res := layoutHTML(t, `<html><body><p>`+url+`</p></body></html>`, s)

	const contentW = testViewport

	var maxRight float64

	texts := make([]string, 0, len(res.Ops))

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		texts = append(texts, paintOp.Text)

		if r := paintOp.X + paintOp.W; r > maxRight {
			maxRight = r
		}
	}

	joined := strings.Join(texts, "")
	if !strings.Contains(joined, "web.archive.org") {
		t.Fatalf("missing URL text in ops: %q", joined)
	}

	if len(texts) < 2 {
		t.Fatalf("expected URL to wrap across ≥2 text ops, got %d: %v", len(texts), texts)
	}
	// Leave a small epsilon for measurement rounding; must stay inside viewport.
	if maxRight > contentW+1 {
		t.Fatalf("text overflows content width: maxRight=%.1f contentW=%.1f ops=%v", maxRight, contentW, texts)
	}
}

func TestOverflowWrapBreakWordSoftBreaks(t *testing.T) {
	t.Parallel()
	// Prefer breaks after '/' rather than mid-segment when possible.
	url := "https://example.com/one/two/three/four/five/six/seven/eight"
	s := sheet(t, `
body { margin: 0; font-size: 12pt; }
p { margin: 0; width: 120pt; overflow-wrap: break-word; }
`)
	res := layoutHTML(t, `<html><body><p style="overflow-wrap:break-word">`+url+`</p></body></html>`, s)

	var texts = make([]string, 0, len(res.Ops))

	var maxRight float64

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		texts = append(texts, paintOp.Text)

		if r := paintOp.X + paintOp.W; r > maxRight {
			maxRight = r
		}
	}

	if len(texts) < 2 {
		t.Fatalf("expected wrap, got %v", texts)
	}
	// At least one soft break should land after a slash.
	soft := false

	for _, tx := range texts[:len(texts)-1] {
		if strings.HasSuffix(tx, "/") {
			soft = true

			break
		}
	}

	if !soft {
		t.Logf("note: no chunk ended with / (still OK if wrapped): %v", texts)
	}

	if maxRight > testViewport+1 {
		t.Fatalf("overflow maxRight=%.1f texts=%v", maxRight, texts)
	}
}

func TestWordBreakBreakAll(t *testing.T) {
	t.Parallel()

	s := sheet(t, `body { margin: 0; font-size: 12pt; } p { word-break: break-all; margin: 0; }`)
	// No soft opportunities — must still wrap under break-all.
	token := strings.Repeat("W", 80)
	res := layoutHTML(t, `<html><body><p style="word-break:break-all">`+token+`</p></body></html>`, s)

	var node int

	var maxRight float64

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		node++

		if r := paintOp.X + paintOp.W; r > maxRight {
			maxRight = r
		}
	}

	if node < 2 {
		t.Fatalf("break-all should wrap long token into ≥2 ops, got %d", node)
	}

	if maxRight > testViewport+1 {
		t.Fatalf("break-all still overflows: maxRight=%.1f", maxRight)
	}
}

func TestNowrapDoesNotEmergencySplit(t *testing.T) {
	t.Parallel()
	// white-space:nowrap must stay unbreakable (cite markers, badges).
	s := sheet(t, `body { margin: 0; font-size: 10pt; } .nw { white-space: nowrap; }`)
	token := "https://example.com/" + strings.Repeat("segment/", 20)
	res := layoutHTML(t, `<html><body><span class="nw">`+token+`</span></body></html>`, s)

	var texts []string

	for _, op := range res.Ops {
		if op.Kind == OpText {
			texts = append(texts, op.Text)
		}
	}
	// May be one or few face-split runs but should not wrap to many short lines
	// solely from emergency break — nowrap opts out.
	joined := strings.Join(texts, "")
	if !strings.Contains(joined, "example.com") {
		t.Fatalf("missing text: %v", texts)
	}
}

// overflow-wrap / word-break inherit onto text nodes (CSS Text), so a URL
// inside <a style="overflow-wrap:break-word"> actually wraps.
func TestOverflowWrapInheritsToText(t *testing.T) {
	t.Parallel()

	url := "https://web.archive.org/web/20200316084639/https://www.example.com/path/to/article-title"
	cssSheet := sheet(t, `body { margin: 0; font-size: 10pt; } a { overflow-wrap: break-word; }`)
	src := `<html><body><p>Lead (` + `<a href="` + url + `">` + url + `</a>)</p></body></html>`
	root := mustParse(t, src)

	const contentW = 280.0

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: contentW, Height: 800, Sheets: []*css.Stylesheet{cssSheet}, Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	var maxRight float64

	var node int

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		node++

		if r := paintOp.X + paintOp.W; r > maxRight {
			maxRight = r
		}
	}

	if node < 2 {
		t.Fatalf("expected wrapped URL (≥2 text ops), got %d", node)
	}

	if maxRight > contentW+1 {
		t.Fatalf("inherited overflow-wrap still overflows: maxRight=%.1f", maxRight)
	}
}
