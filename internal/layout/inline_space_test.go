//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// Adjacent inlines separated by a leading-space text node ("</a> in") must
// keep that space after collapseWS/Fields (wiki "Reeves in", "Cuba Spain").
func TestLeadingSpaceBetweenInlines(t *testing.T) {
	t.Parallel()

	s := sheet(t, `body { margin: 0; font-size: 10pt; } a { color: blue; }`)
	res := layoutHTML(t, `<html><body>
<p><a href="/w">Reeves</a> in her first film. <a href="/c">Cuba</a> <a href="/s">Spain</a></p>
</body></html>`, s)

	var got string

	for _, op := range res.Ops {
		if op.Kind == OpText {
			got += op.Text
		}
	}

	if strings.Contains(got, "Reevesin") {
		t.Fatalf("missing space after link: %q", got)
	}

	if !strings.Contains(got, "Reeves in") {
		t.Fatalf("expected 'Reeves in' in %q", got)
	}

	if strings.Contains(got, "CubaSpain") {
		t.Fatalf("missing space between links: %q", got)
	}
}

// Inline backgrounds must stop at the decorated text. They must not cover
// following plain text when a padded inline element is followed by punctuation
// and another word on the same line.
func TestInlineChromeDoesNotCoverFollowingText(t *testing.T) { //nolint:cyclop
	t.Parallel()

	htmlSrc := `<html><body><p>Booleans coerce liberally ` +
		`(<samp>"" / true / 1 / yes / on</samp>); unknown keys fail loudly.</p></body></html>`
	cssSrc := `
body { margin: 0; font-size: 10pt; }
samp { font-family: monospace; background: #efe9dc; padding: 0.05em 0.3em; border-radius: 4px; }
`
	res := layoutHTML(t, htmlSrc, sheet(t, cssSrc))

	var sampText, followingText, chrome Op

	for _, paintOp := range res.Ops {
		switch {
		case paintOp.Kind == OpText && strings.Contains(paintOp.Text, `"" / true / 1 / yes / on`):
			sampText = paintOp
		case paintOp.Kind == OpText && strings.Contains(paintOp.Text, "); unknown"):
			followingText = paintOp
		}
	}

	for _, paintOp := range res.Ops {
		matchesChrome := paintOp.Kind == OpFillRect &&
			fixture56HasRGB(paintOp, 0.9373, 0.9137, 0.8627) &&
			paintOp.X < sampText.X+0.1 &&
			paintOp.Y < sampText.Y &&
			paintOp.Y+paintOp.H > sampText.Y-paintOp.H
		if !matchesChrome {
			continue
		}

		chrome = paintOp

		break
	}

	if sampText.Text == "" || followingText.Text == "" || chrome.W == 0 {
		t.Fatalf("missing inline boundary ops: samp=%+v following=%+v chrome=%+v",
			sampText, followingText, chrome)
	}

	if chrome.X+chrome.W > followingText.X+4 {
		t.Fatalf(
			"inline chrome overlaps following text: chrome right=%.2f following text x=%.2f",
			chrome.X+chrome.W,
			followingText.X,
		)
	}
}

// page-break-after:avoid must not land a heading on top of body text that
// already snapped to the next page (wiki Career subsection page-2 overlap).
// The overlapping text is often a PRIOR paragraph's continuation, not the
// heading's following sibling — so clearance must clear the whole page-top band.
func TestPageBreakAfterAvoidNoOverwrite(t *testing.T) { //nolint:cyclop,funlen
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
h3 { font-size: 12pt; margin: 8pt 0 4pt; page-break-after: avoid; }
p { margin: 0 0 6pt 0; }
`)

	var paras strings.Builder
	for range 40 {
		paras.WriteString(`<p>Line of career body text that fills the page so the next heading is forced ` +
			`across a boundary with following copy.</p>`)
	}

	src := `<html><body>` + paras.String() + `
<h3>Transition to Hollywood and breakthrough</h3>
<p>had to start her career again from scratch with more words here.</p>
</body></html>`

	root, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	const pageH = 400.0

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 400, Height: pageH, Sheets: []*css.Stylesheet{cssSheet}, Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	paginateOps(res, pageH)

	var headingY, bodyY float64
	headingY, bodyY = -1, -1

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		if strings.Contains(paintOp.Text, "Transition to Hollywood") {
			headingY = paintOp.Y
		}

		if strings.Contains(paintOp.Text, "from scratch") {
			bodyY = paintOp.Y
		}
	}

	if headingY < 0 || bodyY < 0 {
		t.Fatalf("missing ops headingY=%.1f bodyY=%.1f", headingY, bodyY)
	}

	hPage := int(headingY / pageH)
	bPage := int(bodyY / pageH)

	if hPage != bPage {
		t.Fatalf("heading page %d body page %d", hPage, bPage)
	}

	pageStart := float64(hPage) * pageH
	if headingY < pageStart-0.5 {
		t.Fatalf("heading y=%.1f above page start %.1f", headingY, pageStart)
	}

	if bodyY < headingY+14 {
		t.Fatalf("body y=%.1f overlaps heading y=%.1f (want ≥14pt clearance)", bodyY, headingY)
	}
}
