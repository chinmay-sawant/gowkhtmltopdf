package layout

import (
	"math"
	"testing"
)

// cplusplus-tutorial-3: whitespace between display:inline-block <li> items
// inside a shrink-to-fit UL must not force one line per item. The intrinsic
// max-content measure skipped whitespace-only text nodes, so the UL was built
// a space narrower than its painted content and the second LI wrapped. The
// live #I_bar stayed 94.70pt tall (two rows) instead of one line.
func TestInlineBlockBreadcrumbWhitespaceStaysOnOneLine(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `.bar { line-height: 40px; background: #eee }
.bar ul { display: inline-block; margin-left: 50px }
.bar li { display: inline-block }`)

	control := layoutHTML(t,
		`<html><body><div class="bar"><ul><li>Tutorials : </li><li>C++ Language</li></ul></div></body></html>`,
		cssSheet)
	controlBar := findBoxByClass(t, control, "bar")

	for name, inner := range map[string]string{
		"newline": `<li>Tutorials : </li>` + "\n" + `<li>C++ Language</li>`,
		"space":   `<li>Tutorials : </li> <li>C++ Language</li>`,
		"none":    `<li>Tutorials : </li><li>C++ Language</li>`,
	} {
		res := layoutHTML(t, `<html><body><div class="bar"><ul>`+inner+`</ul></div></body></html>`, cssSheet)

		bar := findBoxByClass(t, res, "bar")
		if math.Abs(bar.height-controlBar.height) > 0.01 {
			t.Errorf("%s: breadcrumb bar height = %.2fpt, whitespace-free control "+
				"= %.2fpt; the inter-item space must not add a row", name, bar.height, controlBar.height)
		}

		firstY := math.NaN()

		for _, paintOp := range res.Ops {
			if paintOp.Kind != OpText || (paintOp.Text != "Tutorials :" && paintOp.Text != "C++ Language") {
				continue
			}

			if math.IsNaN(firstY) {
				firstY = paintOp.Y
			}

			if math.Abs(paintOp.Y-firstY) > 0.01 {
				t.Errorf("%s: %q paints at y=%.2f, first line at y=%.2f; want one row",
					name, paintOp.Text, paintOp.Y, firstY)
			}
		}

		if math.IsNaN(firstY) {
			t.Fatalf("%s: breadcrumb text ops not found", name)
		}
	}
}
