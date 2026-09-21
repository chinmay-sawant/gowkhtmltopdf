//nolint:wsl,varnamelen,nlreturn // direct fixture traversal keeps the source-to-box mapping visible
package layout

import (
	"os"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func loadChromeFlexCase0102(t *testing.T, name string) (*html.Node, []*css.Stylesheet) {
	t.Helper()

	source, err := os.ReadFile("../../test/chrome/cases/" + name)
	if err != nil {
		t.Fatalf("read Chrome Flex case %q: %v", name, err)
	}

	root, err := html.Parse(string(source))
	if err != nil {
		t.Fatalf("parse Chrome Flex case %q: %v", name, err)
	}

	var sheets []*css.Stylesheet
	root.Walk(func(node *html.Node) {
		if node.Type != html.ElementNode || node.Name != "style" {
			return
		}

		var source string
		for _, child := range node.Children {
			if child.Type == html.TextNode {
				source += child.Text
			}
		}

		styleSheet, err := css.Parse(source)
		if err != nil {
			t.Fatalf("parse stylesheet in Chrome Flex case %q: %v", name, err)
		}
		sheets = append(sheets, styleSheet)
	})

	return root, sheets
}

func chromeFlexCaseBoxByID0102(t *testing.T, root *box, id string) *box {
	t.Helper()

	var found *box
	var walk func(*box)
	walk = func(node *box) {
		if found != nil || node == nil {
			return
		}
		if node.node != nil && node.node.Attribute("id") == id {
			found = node
			return
		}
		for _, child := range node.children {
			walk(child)
		}
	}
	walk(root)

	if found == nil {
		t.Fatalf("Chrome Flex case box %q is missing", id)
	}

	return found
}

func layoutChromeFlexCase0102(t *testing.T, name string) *Result {
	t.Helper()

	root, sheets := loadChromeFlexCase0102(t, name)
	res, err := Layout(root, Options{
		Width: testViewport, Height: 800, Sheets: sheets, Background: true,
	})
	if err != nil {
		t.Fatalf("layout Chrome Flex case %q: %v", name, err)
	}

	return res
}

// TestChromeFlexCase01LegacyAlgorithm covers the legacy Chromium fixture's
// positive grow and scaled negative shrink branches using its static input.
func TestChromeFlexCase01LegacyAlgorithm(t *testing.T) {
	t.Parallel()

	res := layoutChromeFlexCase0102(t, "case-01-legacy-flex-algorithm.html")

	for _, testCase := range []struct {
		id   string
		want float64
	}{
		{id: "basis-grow-a", want: 75},
		{id: "basis-grow-b", want: 150},
		{id: "basis-grow-c", want: 225},
		{id: "scaled-shrink-a", want: 187.5},
		{id: "scaled-shrink-b", want: 112.5},
		{id: "scaled-shrink-c", want: 150},
	} {
		box := chromeFlexCaseBoxByID0102(t, res.root, testCase.id)
		if !near(box.w, testCase.want) {
			t.Errorf("case %s width = %.2fpt, want %.2fpt", testCase.id, box.w, testCase.want)
		}
	}
}

// TestChromeFlexCase02LegacyAlgorithmMinmax covers max-width freezing and
// redistribution from the legacy Chromium min/max fixture.
func TestChromeFlexCase02LegacyAlgorithmMinmax(t *testing.T) {
	t.Parallel()

	res := layoutChromeFlexCase0102(t, "case-02-legacy-flex-algorithm-minmax.html")

	for _, testCase := range []struct {
		id   string
		want float64
	}{
		{id: "minmax-frozen", want: 75},
		{id: "minmax-redistributed-a", want: 187.5},
		{id: "minmax-redistributed-b", want: 187.5},
	} {
		box := chromeFlexCaseBoxByID0102(t, res.root, testCase.id)
		if !near(box.w, testCase.want) {
			t.Errorf("case %s width = %.2fpt, want %.2fpt", testCase.id, box.w, testCase.want)
		}
	}
}
