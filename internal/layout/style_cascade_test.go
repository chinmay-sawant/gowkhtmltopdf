//nolint:testpackage // tests exercise the resolved style cascade
package layout

import (
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

func TestCascadeShorthandRespectsSourceOrder(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(`<html><body><h1>heading</h1><p>paragraph</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	sheetAfter := sheet(t, `h1, p { margin-top: 0 } h1 { margin: 25mm 0 7mm }`)
	styles := resolveStyles(root, []*css.Stylesheet{sheetAfter}, "print", testViewport, 800)
	heading := findElementByName(root, "h1")

	if got := styles[heading].MarginTop; got < 70 || got > 72 {
		t.Fatalf("later margin shorthand lost to earlier longhand: got %.2fpt", got)
	}

	root, err = html.Parse(`<html><body><h1>heading</h1></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	sheetBefore := sheet(t, `h1 { margin: 25mm 0 7mm } h1 { margin-top: 0 }`)
	styles = resolveStyles(root, []*css.Stylesheet{sheetBefore}, "print", testViewport, 800)
	heading = findElementByName(root, "h1")

	if got := styles[heading].MarginTop; got != 0 {
		t.Fatalf("later margin-top lost to earlier shorthand: got %.2fpt", got)
	}
}

// TestCascadeBorderShorthandOverridesEarlierBorderTop: a later border
// shorthand must beat an earlier border-top longhand (fixture-56 domain-03
// accent top vs .domains > section frame).
func TestCascadeBorderShorthandOverridesEarlierBorderTop(t *testing.T) {
	t.Parallel()

	// Equal specificity (type+class): later border shorthand must win.
	root, err := html.Parse(`<html><body><div class="domains"><section class="d03">x</section></div></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	cssSheet := sheet(t, `
section.d03 { border-top: 3px solid #2563eb }
.domains > section { border: 1px solid #cbd5e1; border-left: 4px solid #0f766e }
`)
	styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "print", testViewport, 800)
	sec := findElementByName(root, "section")
	sty := styles[sec]

	if sty.BorderTop.Width > 1.5 {
		t.Fatalf("border-top width = %.2f, want ≤1.5 after later border shorthand", sty.BorderTop.Width)
	}

	// #2563eb accent must not survive; #cbd5e1 frame is ~ (0.80, 0.84, 0.88).
	if sty.BorderTop.Color[0] < 0.5 && sty.BorderTop.Color[2] > 0.85 {
		t.Fatalf("border-top still accent blue (%.2f,%.2f,%.2f), want neutral frame",
			sty.BorderTop.Color[0], sty.BorderTop.Color[1], sty.BorderTop.Color[2])
	}

	if sty.BorderLeft.Width < 2.5 {
		t.Fatalf("border-left width = %.2f, want accent rail ≥2.5", sty.BorderLeft.Width)
	}
}

func findElementByName(root *html.Node, name string) *html.Node {
	if root.Type == html.ElementNode && root.Name == name {
		return root
	}

	for _, child := range root.Children {
		if found := findElementByName(child, name); found != nil {
			return found
		}
	}

	return nil
}
