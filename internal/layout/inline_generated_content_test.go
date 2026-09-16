package layout

import (
	"strings"
	"testing"
)

// A collapsible trailing space in generated content separates the box from
// the next inline sibling. The cplusplus.com breadcrumb li::after " : " lost
// its trailing space, so "Tutorials :" and "C++ Language" painted on top of
// each other ("Tutorials :C++ Language").
func TestGeneratedContentTrailingSpaceSeparatesInlineBlocks(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `body { margin: 0 }
ul { margin: 0; padding: 0; list-style: none }
li { display: inline-block }
li::after { content: " : " }`)

	res := layoutHTML(t, `<html><body><ul><li><a>Tutorials</a></li><li>C++ Language</li></ul></body></html>`, cssSheet)

	var colon, next Op

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		if strings.HasPrefix(paintOp.Text, "C++") {
			next = paintOp

			break
		}

		if strings.HasSuffix(strings.TrimRight(paintOp.Text, " "), ":") {
			colon = paintOp
		}
	}

	if colon.Text == "" || next.Text == "" {
		t.Fatalf("missing breadcrumb ops: colon=%+v next=%+v", colon, next)
	}

	// Calibrate one space advance at the same font size with nowrap probes.
	probe := sheet(t, `body { margin: 0 } p { margin: 0 }`)
	withSpace := layoutHTML(t, `<html><body><p style="white-space:nowrap">x x</p></body></html>`, probe)
	withoutSpace := layoutHTML(t, `<html><body><p style="white-space:nowrap">xx</p></body></html>`, probe)
	spaceAdv := firstText(withSpace).W - firstText(withoutSpace).W

	gap := next.X - (colon.X + colon.W)
	if gap < spaceAdv-0.5 {
		t.Fatalf("gap after ':' is %.3fpt, want at least one space advance (%.3fpt); "+
			"generated trailing space must separate the items", gap, spaceAdv)
	}
}
