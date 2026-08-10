//nolint:testpackage // fixture diagnostics and regression checks inspect layout internals
package layout

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

func loadFixture56(t *testing.T) (*html.Node, *css.Stylesheet) {
	t.Helper()

	base := filepath.Join("..", "..", "testdata", "golden")
	htmlData, err := os.ReadFile(filepath.Join(base, "fixture-56-architecture-diagram.html"))
	if err != nil {
		t.Fatal(err)
	}

	cssData, err := os.ReadFile(filepath.Join(base, "fixture-56-architecture-diagram.css"))
	if err != nil {
		t.Fatal(err)
	}

	root, err := html.Parse(string(htmlData))
	if err != nil {
		t.Fatal(err)
	}

	sheet, err := css.Parse(string(cssData))
	if err != nil {
		t.Fatal(err)
	}

	return root, sheet
}

func fixture56Node(root *html.Node, predicate func(*html.Node) bool) *html.Node {
	var found *html.Node

	root.Walk(func(node *html.Node) {
		if found == nil && predicate(node) {
			found = node
		}
	})

	return found
}

func fixture56Class(node *html.Node) string {
	if node == nil {
		return ""
	}

	return node.Attribute("class")
}

func fixture56BoxByNode(root *box, target *html.Node) *box {
	if root == nil {
		return nil
	}

	if root.node == target {
		return root
	}

	for _, child := range root.children {
		if found := fixture56BoxByNode(child, target); found != nil {
			return found
		}
	}

	return nil
}

func fixture56Nodes(root *html.Node, predicate func(*html.Node) bool) []*html.Node {
	var found []*html.Node

	root.Walk(func(node *html.Node) {
		if predicate(node) {
			found = append(found, node)
		}
	})

	return found
}

func fixture56HasRGB(op Op, r, g, b float64) bool {
	return math.Abs(op.R-r) < 0.01 && math.Abs(op.G-g) < 0.01 && math.Abs(op.B-b) < 0.01
}

func fixture56HasRGBColor(color [3]float64, r, g, b float64) bool {
	return math.Abs(color[0]-r) < 0.01 && math.Abs(color[1]-g) < 0.01 && math.Abs(color[2]-b) < 0.01
}

func fixture56HasFill(res *Result, target *box, predicate func(Op) bool) bool {
	if target == nil || target.opStart < 0 || target.opEnd >= len(res.Ops) {
		return false
	}

	for _, op := range res.Ops[target.opStart : target.opEnd+1] {
		if op.Kind == OpFillRect && predicate(op) {
			return true
		}
	}

	return false
}

func TestFixture56RendererSeams(t *testing.T) {
	root, sheet := loadFixture56(t)
	opts := Options{ //nolint:exhaustruct // fixture uses the standard print layout path
		Width: 595.28 - 2*28.35, Height: 841.89 - 2*28.35,
		Background: true, Sheets: []*css.Stylesheet{sheet}, Media: "print",
	}

	styles, _ := resolveStylesForLayout(root, opts)
	res, err := Layout(root, opts)
	if err != nil {
		t.Fatal(err)
	}

	maxExtent := res.Width
	for _, op := range res.Ops {
		if (op.Kind == OpFillRect || op.Kind == OpStrokeRect || op.Kind == OpImage) && op.X+op.W > maxExtent {
			maxExtent = op.X + op.W
		}
	}
	if maxExtent > res.Width+0.01 {
		t.Fatalf("fixture paints outside its definite width: extent %.2f > %.2f", maxExtent, res.Width)
	}

	heroLegend := fixture56Node(root, func(node *html.Node) bool { return fixture56Class(node) == "hero-legend" })
	if boxNode := fixture56BoxByNode(res.root, heroLegend); boxNode == nil || boxNode.height > 2*styles[heroLegend].LineHeight {
		t.Fatalf("hero legend wrapped unexpectedly: box=%+v", boxNode)
	}

	for _, node := range fixture56Nodes(root, func(node *html.Node) bool { return fixture56Class(node) == "dom-pkgs" }) {
		boxNode := fixture56BoxByNode(res.root, node)
		if boxNode == nil {
			t.Fatalf("package grid has no layout box")
		}

		for _, child := range boxNode.children {
			if child.x < boxNode.x-0.01 || child.x+child.w > boxNode.x+boxNode.w+0.01 {
				t.Fatalf("package grid child overflows: child=%+v parent=%+v", child, boxNode)
			}
		}
	}

	for _, className := range []string{"pipe-input", "pipe-core", "pipe-output"} {
		node := fixture56Node(root, func(node *html.Node) bool {
			return strings.Contains(" "+fixture56Class(node)+" ", " "+className+" ")
		})
		if node == nil || styles[node].BorderTop.Color == [3]float64{0, 0, 0} {
			t.Fatalf("pipeline class %q lost its colored top border", className)
		}
	}

	security := fixture56Node(root, func(node *html.Node) bool { return fixture56Class(node) == "security" })
	if security == nil || !fixture56HasRGBColor(styles[security].BorderLeft.Color, 0.7059, 0.1373, 0.0941) {
		t.Fatalf("security rail color = %v, want #b42318", styles[security].BorderLeft.Color)
	}

	details := fixture56Nodes(root, func(node *html.Node) bool { return fixture56Class(node) == "d06-details" })
	if len(details) != 2 {
		t.Fatalf("details count = %d, want 2", len(details))
	}
	for _, node := range details {
		boxNode := fixture56BoxByNode(res.root, node)
		if _, open := node.Attrs["open"]; open {
			if boxNode == nil || len(boxNode.children) < 2 {
				t.Fatalf("open details lost content: attrs=%v box=%+v", node.Attrs, boxNode)
			}
		} else if boxNode == nil || len(boxNode.children) != 1 {
			t.Fatalf("closed details expanded in print layout: box=%+v", boxNode)
		}
	}

	for _, id := range []string{"d03-meter", "d03-progress"} {
		node := fixture56Node(root, func(node *html.Node) bool { return node.Attribute("id") == id })
		if !fixture56HasFill(res, fixture56BoxByNode(res.root, node), func(op Op) bool {
			return op.W > 0 && op.H > 0
		}) {
			t.Fatalf("%s has no value fill", id)
		}
	}

	codeFill, markFill := false, false
	for _, op := range res.Ops {
		codeFill = codeFill || (op.Kind == OpFillRect && fixture56HasRGB(op, 0.9373, 0.9137, 0.8627))
		markFill = markFill || (op.Kind == OpFillRect && fixture56HasRGB(op, 0.9529, 0.8627, 0.6902))
	}
	if !codeFill || !markFill {
		t.Fatalf("inline chrome fills missing: code=%v mark=%v", codeFill, markFill)
	}
}
