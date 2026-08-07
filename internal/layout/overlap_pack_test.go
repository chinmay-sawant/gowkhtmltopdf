package layout

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

// TestNoCompactOverlapsBodyLines guards against packAvoidGaps / former
// compactAvoidInternalGaps + shiftOpsBelowY over-pulling body paragraphs so
// consecutive baselines collapse (wiki page-2 stacked paragraphs; ref lines
// with dy ≪ fontSize). Multi-paragraph body + avoid list must keep body line
// pitch ≥ 0.95·fontSize.
func TestNoCompactOverlapsBodyLines(t *testing.T) {
	const fontSize = 10.0
	s := sheet(t, fmt.Sprintf(`
body { margin: 0; font-size: %gpt; line-height: 1.25; }
p { margin: 0.6em 0; }
ol { margin: 0.5em 0; padding-left: 22pt; }
li { page-break-inside: avoid; margin: 0 0 0.35em 0; }
`, fontSize))

	var b strings.Builder
	b.WriteString(`<html><body>`)
	// Several multi-line body paragraphs that can sit near avoid boxes after
	// pagination shifts (the historical over-pack victim).
	for i := 0; i < 14; i++ {
		b.WriteString(fmt.Sprintf(
			`<p>Body paragraph %d with enough words that the line box wraps onto a second and sometimes a third line of text so we can measure consecutive baselines after paint packing. More filler about articles and biographies to keep width full.</p>`, i))
	}
	b.WriteString(`<ol>`)
	for i := 0; i < 18; i++ {
		b.WriteString(fmt.Sprintf(
			`<li id="cite-%d">"Reference title %d" (https://example.com/ref/%d/long-path). Publisher. 1 January 2020. Retrieved 2 January 2021.</li>`,
			i, i+1, i))
	}
	b.WriteString(`</ol>`)
	// More body after the list so packing cannot "heal" list holes by
	// collapsing following paragraphs.
	for i := 0; i < 10; i++ {
		b.WriteString(fmt.Sprintf(
			`<p>Trailing body %d continues the article with multi-line prose that must keep normal leading after any avoid-sibling packing pass.</p>`, i))
	}
	b.WriteString(`</body></html>`)

	root, err := html.Parse(b.String())
	if err != nil {
		t.Fatal(err)
	}
	const pageH = 500.0
	res, err := Layout(root, Options{
		Width: 420, Height: pageH, Sheets: []*css.Stylesheet{s},
		Media: "print", Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	doc := pdf.NewDocument()
	if err := Paint(doc, res, PaintOptions{
		PageWidth: 470, PageHeight: pageH + 40, MarginTop: 20, MarginBottom: 20,
	}); err != nil {
		t.Fatal(err)
	}

	// Collect body paragraph text baselines (exclude list-item ops via Locations).
	type liSpan struct{ y0, y1 float64 }
	var liSpans []liSpan
	for _, loc := range res.Locations {
		if loc.Node != nil && loc.Node.Name == "li" {
			liSpans = append(liSpans, liSpan{loc.Y - 1, loc.Y + loc.H + 2})
		}
	}
	inList := func(y float64) bool {
		for _, s := range liSpans {
			if y >= s.y0 && y <= s.y1 {
				return true
			}
		}
		return false
	}

	// Group non-list text ops into approximate lines by Y, per page, then
	// require consecutive body line dy ≥ 0.95·fontSize.
	minPitch := 0.95 * fontSize
	type lineY struct {
		y    float64
		size float64
	}
	var bodyLines []lineY
	for _, op := range res.Ops {
		if op.Kind != OpText || op.Text == "" {
			continue
		}
		if inList(op.Y) {
			continue
		}
		// Skip near-empty / marker-like runs.
		if len(strings.TrimSpace(op.Text)) < 8 {
			continue
		}
		merged := false
		for i := range bodyLines {
			if math.Abs(bodyLines[i].y-op.Y) <= 0.5 {
				merged = true
				if op.Size > bodyLines[i].size {
					bodyLines[i].size = op.Size
				}
				break
			}
		}
		if !merged {
			sz := op.Size
			if sz <= 0 {
				sz = fontSize
			}
			bodyLines = append(bodyLines, lineY{y: op.Y, size: sz})
		}
	}
	// Sort by Y.
	for i := 0; i < len(bodyLines); i++ {
		for j := i + 1; j < len(bodyLines); j++ {
			if bodyLines[j].y < bodyLines[i].y {
				bodyLines[i], bodyLines[j] = bodyLines[j], bodyLines[i]
			}
		}
	}
	if len(bodyLines) < 12 {
		t.Fatalf("expected many body lines, got %d", len(bodyLines))
	}

	contentH := pageH
	overlaps := 0
	minDy := 1e9
	for i := 1; i < len(bodyLines); i++ {
		prev, cur := bodyLines[i-1], bodyLines[i]
		// Only same-page consecutive body lines (paragraph stack / wrap).
		if int(prev.y/contentH) != int(cur.y/contentH) {
			continue
		}
		dy := cur.y - prev.y
		// Skip large paragraph gaps (new block well below previous).
		// Overlap bug shows dy of 0.6–5pt for 8–10pt font; wrapped lines
		// and adjacent paragraphs after collapse sit far below minPitch.
		if dy > fontSize*4 {
			continue
		}
		if dy < minDy {
			minDy = dy
		}
		need := minPitch
		if s := prev.size; s > 0 && 0.95*s > need {
			need = 0.95 * s
		}
		if dy < need {
			overlaps++
			t.Logf("body line pitch too tight: dy=%.2f (need ≥%.2f) at y=%.1f→%.1f page=%d",
				dy, need, prev.y, cur.y, int(cur.y/contentH))
		}
	}
	t.Logf("bodyLines=%d minDy=%.2f overlaps=%d", len(bodyLines), minDy, overlaps)
	if overlaps > 0 {
		t.Fatalf("packAvoidGaps over-pulled body lines: %d pairs with dy < 0.95*fontSize (minDy=%.2f)",
			overlaps, minDy)
	}
	if minDy < minPitch {
		t.Fatalf("minimum same-page body line pitch %.2f < 0.95*fontSize (%.2f)", minDy, minPitch)
	}
}
