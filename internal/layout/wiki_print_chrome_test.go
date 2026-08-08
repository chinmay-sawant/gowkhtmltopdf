//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"strings"
	"testing"
)

func TestWikiPrintChromeHidden(t *testing.T) { //nolint:cyclop,funlen
	t.Parallel()

	cssSheet := sheet(t, `
@media print {
  #mw-navigation, .noprint, .mw-jump-link, nav, #footer { display: none }
  .firstHeading { font-size: 25pt }
}
a:link { color: #0645ad }
`)
	html := `<html><body>
<nav class="vector">Main page</nav>
<div id="mw-navigation">Sidebar</div>
<div class="noprint">Donate</div>
<h1 class="firstHeading">Ana de Armas</h1>
<p>She is a <a href="/wiki/Cuba">Cuban</a> actress.</p>
<footer id="footer">Footer chrome</footer>
</body></html>`
	res := layoutHTML(t, html, cssSheet)

	var texts []string

	for _, op := range res.Ops {
		if op.Kind == OpText {
			texts = append(texts, strings.TrimSpace(op.Text))
		}
	}

	joined := strings.Join(texts, " | ")
	t.Log(joined)

	for _, bad := range []string{"Main page", "Sidebar", "Donate", "Footer chrome"} {
		for _, tx := range texts {
			if strings.Contains(tx, bad) {
				t.Errorf("chrome %q still painted in %q", bad, tx)
			}
		}
	}

	foundTitle, foundLink := false, false

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		if strings.Contains(paintOp.Text, "Ana de Armas") {
			foundTitle = true
		}

		if strings.Contains(paintOp.Text, "Cuban") {
			foundLink = true

			if paintOp.B < 0.5 {
				t.Errorf("Cuban link color = (%v,%v,%v), want blue", paintOp.R, paintOp.G, paintOp.B)
			}
		}
	}

	if !foundTitle {
		t.Error("missing title")
	}

	if !foundLink {
		t.Error("missing link text")
	}
}
