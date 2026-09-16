package layout

import (
	"math"
	"testing"
)

// Pretty-printed whitespace inside a white-space:nowrap inline-block must
// collapse and trim like the one-line form. The learn-cpp.org dock measured
// the indented Run button at 2.4x to 6x its painted width (158.5pt vs 26.2pt)
// because intrinsic sizing measured the raw whitespace runs.
func TestInlineBlockNowrapWhitespaceDoesNotInflateWidth(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `body { margin: 0 }
button { display: inline-block; white-space: nowrap; font-size: 12pt; padding: 2pt 6pt; border: 0 }`)

	oneLine := layoutHTML(t, `<html><body><button>Run</button></body></html>`, cssSheet)
	indented := layoutHTML(t, "<html><body><button>\n        Run\n</button></body></html>", cssSheet)

	oneBox := findBox(t, oneLine, "button")
	indentBox := findBox(t, indented, "button")

	if math.Abs(oneBox.w-indentBox.w) > 0.01 {
		t.Fatalf("indented button width %.3fpt, want the one-line width %.3fpt; "+
			"collapsed whitespace must not count in intrinsic sizing", indentBox.w, oneBox.w)
	}
}
