package layout

import (
	"fmt"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

// TestAvoidListItemsNoCascadingGaps: dense page-break-inside:avoid list items
// near page boundaries must not cascade expanding empty bands between
// consecutive items (wiki .mw-references-columns li{page-break-inside:avoid}).
// Prefer splitting short avoid boxes over blanking large mid-page remainders
// (preferSplitOverBlank) so same-page inter-item start pitch stays near
// natural multi-line height (≤ ~2.5 line heights beyond content).
func TestAvoidListItemsNoCascadingGaps(t *testing.T) {
	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
ol { margin: 0; padding-left: 24pt; }
li { page-break-inside: avoid; margin: 0 0 0.4em 0; }
p { margin: 0.4em 0; }
`)

	var boxNode strings.Builder

	boxNode.WriteString(`<html><body>`)
	// Push the list so items straddle multiple page boundaries.
	for i := range 28 {
		boxNode.WriteString(fmt.Sprintf(
			`<p>Filler paragraph %d with enough words to approach the natural page-break zone for list pagination testing.</p>`, i))
	}

	boxNode.WriteString(`<ol start="1">`)

	for i := range 35 {
		boxNode.WriteString(fmt.Sprintf(
			`<li id="r%d">"Citation title number %d with a fairly long path" (https://example.com/article/%d/long-path-name-here). Journal Name. 12 December 2022. Archived from the original. Retrieved 12 December 2022.</li>`,
			i, i+1, i))
	}

	boxNode.WriteString(`</ol></body></html>`)

	root, err := html.Parse(boxNode.String())
	if err != nil {
		t.Fatal(err)
	}

	const pageH = 750.0

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 538, Height: pageH, Sheets: []*css.Stylesheet{cssSheet},
		Media: "print", Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	doc := pdf.NewDocument()
	po := PaintOptions{PageWidth: 595, PageHeight: pageH + 50, MarginTop: 25, MarginBottom: 25} //nolint:exhaustruct // intentional zero fields

	if err := Paint(doc, res, po); err != nil {
		t.Fatal(err)
	}

	contentH := pageH // Paint margins subtract from PageHeight; here content = pageH

	// Collect list-item top Y from bullets (decimal markers).
	var starts []float64

	for _, op := range res.Ops {
		if op.Kind == OpBullet {
			starts = append(starts, op.Y)
		}
	}

	if len(starts) < 20 {
		// Fall back to li locations if markers missing.
		for _, loc := range res.Locations {
			if loc.Node != nil && loc.Node.Name == "li" {
				starts = append(starts, loc.Y)
			}
		}
	}

	if len(starts) < 10 {
		t.Fatalf("expected many list starts, got %d", len(starts))
	}
	// Sort.
	for i := 0; i < len(starts); i++ {
		for j := i + 1; j < len(starts); j++ {
			if starts[j] < starts[i] {
				starts[i], starts[j] = starts[j], starts[i]
			}
		}
	}

	// Inter-item empty air (next start − prev location bottom) via Locations.
	type liY struct{ y, h float64 }

	var lis []liY

	for _, loc := range res.Locations {
		if loc.Node != nil && loc.Node.Name == "li" {
			lis = append(lis, liY{loc.Y, loc.H})
		}
	}

	for i := 0; i < len(lis); i++ {
		for j := i + 1; j < len(lis); j++ {
			if lis[j].y < lis[i].y {
				lis[i], lis[j] = lis[j], lis[i]
			}
		}
	}

	maxSamePageGap := 0.0
	bigGaps := 0

	for idx := 1; idx < len(starts); idx++ {
		if int(starts[idx-1]/contentH) != int(starts[idx]/contentH) {
			continue
		}

		gap := starts[idx] - starts[idx-1]
		if gap > maxSamePageGap {
			maxSamePageGap = gap
		}
		// Natural multi-line item + margin is ~25–45pt; cascading avoid
		// left 100–150pt bands on wiki references. Residual 26–38pt bands
		// are still too airy vs Chrome.
		if gap > 55 {
			bigGaps++

			t.Logf("large gap %.1f between items at y=%.1f and y=%.1f (page %d)",
				gap, starts[idx-1], starts[idx], int(starts[idx]/contentH))
		}
	}

	maxEmpty := 0.0
	emptyBig := 0

	for idx := 1; idx < len(lis); idx++ {
		if int(lis[idx].y/contentH) != int(lis[idx-1].y/contentH) {
			continue
		}

		empty := lis[idx].y - (lis[idx-1].y + lis[idx-1].h)
		if empty > maxEmpty {
			maxEmpty = empty
		}
		// Pure empty between items should stay near margin (0.4em ≈ 4pt) +
		// a line of slack; 2.5 line-heights at 10pt/1.2 ≈ 30pt is the cap.
		if empty > 30 {
			emptyBig++

			t.Logf("empty air %.1f between li y=%.1f h=%.1f and y=%.1f",
				empty, lis[idx-1].y, lis[idx-1].h, lis[idx].y)
		}
	}

	t.Logf("list starts=%d maxSamePageGap=%.1f bigGaps>55=%d maxEmpty=%.1f empty>30=%d",
		len(starts), maxSamePageGap, bigGaps, maxEmpty, emptyBig)

	if maxSamePageGap > 55 {
		t.Fatalf("cascading avoid-inside gaps: max same-page inter-item start gap %.1f (want ≤55)", maxSamePageGap)
	}

	if bigGaps > 1 {
		t.Fatalf("%d same-page inter-item start gaps >55pt (want ≤1)", bigGaps)
	}

	if maxEmpty > 30 {
		t.Fatalf("avoid-item empty air: max same-page empty %.1f (want ≤30 ≈ 2.5 line heights)", maxEmpty)
	}

	if emptyBig > 1 {
		t.Fatalf("%d same-page empty gaps >30pt (want ≤1)", emptyBig)
	}
}

// TestPreferSplitOverBlankUnit checks the shared blank-band heuristic.
func TestPreferSplitOverBlankUnit(t *testing.T) {
	t.Parallel()
	contentH := 800.0
	// Half-page remaining always prefers split.
	if !preferSplitOverBlank(401, 40, contentH) {
		t.Fatal("remaining > half page should prefer split")
	}
	// Short list item with ~100pt remaining → split (no 100pt blank band).
	if !preferSplitOverBlank(100, 30, contentH) {
		t.Fatal("short box with large remaining should prefer split")
	}
	// Short box with 40pt remaining still prefers split (maxBlank ~15).
	if !preferSplitOverBlank(40, 30, contentH) {
		t.Fatal("short box with mid remaining should prefer split")
	}
	// 20pt remaining on a 30pt box: maxBlank = max(14, 15) = 15 → split.
	if !preferSplitOverBlank(20, 30, contentH) {
		t.Fatal("short box with 20pt remaining should prefer split")
	}
	// Near page end: small remaining, short box → keep together (move).
	if preferSplitOverBlank(12, 30, contentH) {
		t.Fatal("near page end small remaining should allow keep-together move")
	}
	// Tall box near mid (remaining < half, box large): short-box rule N/A.
	if preferSplitOverBlank(300, 400, contentH) {
		t.Fatal("tall box with remaining < half should not match short-box rule")
	}
}
