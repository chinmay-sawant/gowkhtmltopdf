//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

func TestMediaFeatureQueryPrint(t *testing.T) { //nolint:cyclop,funlen
	t.Parallel()

	cssSheet := sheet(t, `
@media screen { .screen-only { display: none } }
@media (min-width: 400px) { .wide { color: #00ff00 } }
@media (min-width: 2000px) { .huge { display: none } }
@media print and (max-width: 100px) { .tight { display: none } }
.base { color: #000000 }
`)
	src := `<html><body>
<p class="screen-only">ScreenRule</p>
<p class="wide">WideGreen</p>
<p class="huge">HugeKeep</p>
<p class="tight">TightKeep</p>
</body></html>`

	root, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	// Viewport 500pt — matches min-width 400px (300pt), not 2000px / max 100px.
	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: testViewport, Height: 800, Sheets: []*css.Stylesheet{cssSheet},
		Media: "print", Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	var texts []string

	for _, op := range res.Ops {
		if op.Kind == OpText {
			texts = append(texts, strings.TrimSpace(op.Text))
		}
	}

	joined := strings.Join(texts, " | ")
	t.Log(joined)

	for _, want := range []string{"ScreenRule", "WideGreen", "HugeKeep", "TightKeep"} {
		found := false

		for _, tx := range texts {
			if strings.Contains(tx, want) {
				found = true

				break
			}
		}

		if !found {
			t.Errorf("missing text %q in %q", want, joined)
		}
	}

	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "WideGreen") {
			if op.G < 0.9 {
				t.Errorf("WideGreen color G=%v, want green from min-width match", op.G)
			}
		}
	}
}
