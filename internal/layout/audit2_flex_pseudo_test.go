package layout

import (
	"strings"
	"testing"
)

// programiz-cpp-11: generated ::before/::after on display:flex hosts must
// become flex items (and abs/fixed pseudos still paint). Before the fix,
// flexChildren only collected element and anonymous text children, so the
// print banner, header domain, and a.btn href suffixes never reached the
// display list. Block hosts with the same rules already painted.

func TestFlexPseudoContentPaints(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-family: sans-serif; font-size: 12pt }
header {
	display: flex; flex-direction: column;
}
header:before {
	display: block;
	content: "BEFORE BANNER";
}
header:after {
	content: "AFTER DOMAIN";
}
.row { display: flex; flex-direction: row; align-items: center }
a.btn {
	display: flex;
	color: #fff;
	background: #0556f3;
	padding: 8pt 12pt;
	text-decoration: none;
}
a.btn:after {
	content: " (" attr(href) ")";
}
`)

	res := layoutHTML(t,
		`<html><body>`+
			`<header><nav>NAV</nav></header>`+
			`<div class="row"><a class="btn" href="https://example.com/learn">Learn more</a></div>`+
			`</body></html>`,
		cssSheet)

	joined := ""

	for _, op := range res.Ops {
		if op.Kind == OpText {
			joined += op.Text + " "
		}
	}

	for _, needle := range []string{"BEFORE BANNER", "AFTER DOMAIN", "Learn more", "(https://example.com/learn)"} {
		if !strings.Contains(joined, needle) {
			t.Fatalf("flex pseudo text %q missing; text layer = %q", needle, joined)
		}
	}
}
