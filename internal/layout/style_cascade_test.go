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
