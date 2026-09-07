//nolint:testpackage // test exercises unexported layout geometry.
package layout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func TestFixture60Border111StaysInsideBorderBox(t *testing.T) {
	t.Parallel()

	res, target := fixture60Property111Fixture(t)
	boxNode := fixture56BoxByNode(res.root, target)

	if boxNode == nil {
		t.Fatal("property 111 effect box not found")
	}

	right := fixture60RightBorder(t, res, boxNode)
	if right.LineInset != LineInsetRight {
		t.Fatalf("right border LineInset = %d, want %d", right.LineInset, LineInsetRight)
	}

	x, _, _, _, width := right.PaintLineGeometry() //nolint:dogsled // only the horizontal outer bound matters here
	if x+width/2 > boxNode.x+boxNode.w+0.001 {
		t.Fatalf("right border extends past border box: outer=%.4f boxRight=%.4f", x+width/2, boxNode.x+boxNode.w)
	}
}

func fixture60Property111Fixture(t *testing.T) (*Result, *html.Node) {
	t.Helper()

	rootDir, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(rootDir, "testdata/golden/fixture-60-implemented-props-a.html"))
	if err != nil {
		t.Fatal(err)
	}

	doc, err := html.Parse(string(raw))
	if err != nil {
		t.Fatal(err)
	}

	parsedSheet, err := css.Parse(extractStyleContent(doc))
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(doc, Options{ //nolint:exhaustruct // fixture geometry probe
		Width: 571.64, Height: 817.89, Background: true, Media: "print", Zoom: 1,
		Sheets: []*css.Stylesheet{parsedSheet},
	})
	if err != nil {
		t.Fatal(err)
	}

	target := fixture60Property111Node(doc)

	if target == nil {
		t.Fatal("property 111 effect node not found")
	}

	return res, target
}

func fixture60Property111Node(doc *html.Node) *html.Node {
	var target *html.Node

	doc.Walk(func(node *html.Node) {
		if target != nil || node.Type != html.ElementNode || node.Name != divElementName ||
			node.TextContent() != "thick right" {
			return
		}

		if node.Parent != nil && node.Parent.Parent != nil && strings.Contains(node.Parent.Parent.TextContent(), "111") {
			target = node
		}
	})

	return target
}

func fixture60RightBorder(t *testing.T, res *Result, boxNode *box) Op {
	t.Helper()

	for idx := boxNode.opStart; idx <= boxNode.opEnd && idx < len(res.Ops); idx++ {
		op := res.Ops[idx]
		if op.Kind == OpLine && op.W == 0 && op.H > 0 && op.R > 0.1 && op.B > 0.8 &&
			nearLayout(op.X, boxNode.x+boxNode.w) {
			return op
		}
	}

	t.Fatal("property 111 right border operation not found")

	return Op{} //nolint:exhaustruct // t.Fatal stops the test before this fallback
}
