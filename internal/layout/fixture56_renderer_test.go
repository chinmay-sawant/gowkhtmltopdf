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
	"gowkhtmltopdf/internal/pdf"
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

func fixture56RoundedStrokes(res *Result, target *box) int {
	if target == nil || target.opStart < 0 || target.opEnd >= len(res.Ops) {
		return 0
	}

	count := 0
	for _, op := range res.Ops[target.opStart : target.opEnd+1] {
		if op.Kind == OpStrokeRect && op.Radius > 0 {
			count++
		}
	}

	return count
}

func fixture56TextOps(res *Result, target *box) int {
	if target == nil || target.opStart < 0 || target.opEnd >= len(res.Ops) {
		return 0
	}

	count := 0
	for _, op := range res.Ops[target.opStart : target.opEnd+1] {
		if op.Kind == OpText {
			count++
		}
	}

	return count
}

func fixture56Location(res *Result, target *html.Node) (ElementLocation, bool) {
	for _, location := range res.Locations {
		if location.Node == target {
			return location, true
		}
	}

	return ElementLocation{}, false
}

func fixture56TextPage(res *Result, text string) (int, bool) {
	for page, opIndexes := range res.Pages {
		for _, opIndex := range opIndexes {
			if opIndex >= 0 && opIndex < len(res.Ops) && res.Ops[opIndex].Kind == OpText &&
				strings.TrimSpace(res.Ops[opIndex].Text) == text {
				return page, true
			}
		}
	}

	return 0, false
}

func fixture56PaintOptions() PaintOptions {
	const (
		pageWidth  = 595.28
		pageHeight = 841.89
		margin     = 28.35
	)

	return PaintOptions{ //nolint:exhaustruct // fixture uses the standard print margins
		PageWidth: pageWidth, PageHeight: pageHeight,
		MarginTop: margin, MarginBottom: margin,
		MarginLeft: margin, MarginRight: margin,
	}
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

	for _, node := range fixture56Nodes(root, func(node *html.Node) bool {
		return strings.Contains(" "+fixture56Class(node)+" ", " hero-legend-item ")
	}) {
		if got := fixture56RoundedStrokes(res, fixture56BoxByNode(res.root, node)); got != 1 {
			t.Fatalf("hero legend item %q has %d rounded strokes, want one", node.Text, got)
		}
	}

	tab := fixture56Node(root, func(node *html.Node) bool { return fixture56Class(node) == "d02-tab" })
	tabBox := fixture56BoxByNode(res.root, tab)
	if got := fixture56TextOps(res, tabBox); got != 1 {
		t.Fatalf("vertical D02 rail has %d text ops, want one unwrapped run", got)
	}
}

func TestFixture56PageComposition(t *testing.T) {
	root, sheet := loadFixture56(t)
	const contentHeight = 841.89 - 2*28.35

	res, err := Layout(root, Options{ //nolint:exhaustruct // fixture uses the standard print layout path
		Width: 595.28 - 2*28.35, Height: contentHeight,
		Background: true, Sheets: []*css.Stylesheet{sheet}, Media: "print", Zoom: 0.98,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := Paint(pdf.NewDocument(), res, fixture56PaintOptions()); err != nil {
		t.Fatal(err)
	}

	d01Exit := fixture56Node(root, func(node *html.Node) bool {
		return fixture56Class(node) == "d01-exit"
	})
	bodyRows := fixture56Nodes(root, func(node *html.Node) bool {
		return node.Name == "tr" && node.Parent != nil && node.Parent.Name == "tbody" &&
			node.Parent.Parent == d01Exit
	})
	if len(bodyRows) != 4 {
		t.Fatalf("D01 exit body rows = %d, want 4", len(bodyRows))
	}
	tableBox := fixture56BoxByNode(res.root, d01Exit)
	if tableBox == nil || len(tableBox.rows) != 5 {
		t.Fatalf("D01 exit painted table rows = %d, want 5 including header", len(tableBox.rows))
	}
	firstGap := rowYBounds(tableBox.rows[1], res) - rowYBounds(tableBox.rows[0], res)
	for rowIndex := 2; rowIndex < len(tableBox.rows); rowIndex++ {
		gap := rowYBounds(tableBox.rows[rowIndex], res) - rowYBounds(tableBox.rows[rowIndex-1], res)
		if gap > firstGap*1.5 {
			t.Fatalf("D01 exit row gap before row %d = %.2fpt, first gap = %.2fpt", rowIndex, gap, firstGap)
		}
	}

	rowTexts := []string{"ok", "error", "HTTP 404", "HTTP 401"}
	firstRowPage, ok := fixture56TextPage(res, rowTexts[0])
	if !ok {
		t.Fatalf("D01 exit first body row text %q has no painted location", rowTexts[0])
	}

	for _, text := range rowTexts[1:] {
		page, found := fixture56TextPage(res, text)
		if !found {
			t.Fatalf("D01 exit row text %q has no painted location", text)
		}

		if page != firstRowPage {
			t.Fatalf("D01 exit rows split across pages: first=%d current=%d", firstRowPage, page)
		}
	}

	d02 := fixture56Node(root, func(node *html.Node) bool { return node.Attribute("id") == "domain-02" })
	d02Location, ok := fixture56Location(res, d02)
	if !ok {
		t.Fatal("D02 section has no painted location")
	}
	pageOffset := math.Mod(d02Location.Y, contentHeight)
	if math.Min(pageOffset, contentHeight-pageOffset) > 2 {
		t.Fatalf("D02 section starts %.2fpt into page, want forced page start", pageOffset)
	}
}
