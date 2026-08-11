//nolint:testpackage,wsl,nlreturn,varnamelen,lll // fixture diagnostics inspect layout internals
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

	return ElementLocation{Node: nil, Page: 0, X: 0, Y: 0, W: 0, H: 0}, false
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

	return PaintOptions{
		PageWidth: pageWidth, PageHeight: pageHeight,
		MarginTop: margin, MarginBottom: margin,
		MarginLeft: margin, MarginRight: margin,
	}
}

//nolint:gocognit,gocyclo,cyclop,funlen // fixture seam assertions intentionally remain together
func TestFixture56RendererSeams(t *testing.T) { //nolint:paralleltest // renderer fixture uses shared font state
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
	boxNode := fixture56BoxByNode(res.root, heroLegend)

	if boxNode == nil || boxNode.height > 2*styles[heroLegend].LineHeight {
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

//nolint:cyclop,funlen // fixture composition assertions intentionally remain together
func TestFixture56PageComposition(t *testing.T) { //nolint:paralleltest // renderer fixture uses shared font state
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
	firstRowPage, foundFirst := fixture56TextPage(res, rowTexts[0])

	if !foundFirst {
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

	d02Location, foundD02 := fixture56Location(res, d02)
	if !foundD02 {
		t.Fatal("D02 section has no painted location")
	}

	pageOffset := math.Mod(d02Location.Y, contentHeight)
	if math.Min(pageOffset, contentHeight-pageOffset) > 2 {
		t.Fatalf("D02 section starts %.2fpt into page, want forced page start", pageOffset)
	}
}

//nolint:gocognit,gocyclo,cyclop,funlen,maintidx // fixture seam assertions intentionally remain together
func TestFixture56PaginationChromeAndWidgetGeometry(t *testing.T) { //nolint:paralleltest // fixture uses shared font state
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

	d01 := fixture56Node(root, func(node *html.Node) bool { return node.Attribute("id") == "domain-01" })
	d01Box := fixture56BoxByNode(res.root, d01)
	footerText := "Position: argv → settings → Request (left edge of the pipeline) · " +
		"see documentation/architecture/01-entrypoints-cli.md"
	var footer Op
	foundFooter := false
	for _, op := range res.Ops {
		if op.Kind == OpText && strings.TrimSpace(op.Text) == footerText {
			footer = op
			foundFooter = true
			break
		}
	}
	if !foundFooter || d01Box == nil {
		t.Fatalf("D01 footer/box missing: footer=%v box=%+v", foundFooter, d01Box)
	}

	railBottom := 0.0
	for i := d01Box.opStart; i <= d01Box.opEnd && i < len(res.Ops); i++ {
		op := res.Ops[i]
		if op.Kind == OpLine && math.Abs(op.X-d01Box.x) < 0.01 && op.H > 0 &&
			op.G > 0.3 && op.B < 0.6 && op.Y+op.H > railBottom {
			railBottom = op.Y + op.H
		}
	}
	if railBottom < footer.Y+footer.H-1 {
		t.Fatalf("D01 left rail ends at %.2f, before footer bottom %.2f", railBottom, footer.Y+footer.H)
	}

	for _, node := range fixture56Nodes(root, func(node *html.Node) bool { return fixture56Class(node) == "d02-engine" }) {
		boxNode := fixture56BoxByNode(res.root, node)
		if boxNode == nil {
			t.Fatalf("D02 engine has no box: %q", node.Text)
		}
		var text Op
		foundText := false
		for i := boxNode.opStart; i <= boxNode.opEnd && i < len(res.Ops); i++ {
			if res.Ops[i].Kind == OpText {
				text = res.Ops[i]
				foundText = true
				break
			}
		}
		if !foundText {
			t.Fatalf("D02 engine has no text op: %q", node.Text)
		}
		leftGap := text.X - boxNode.x
		rightGap := boxNode.x + boxNode.w - (text.X + text.W)
		if math.Abs(leftGap-rightGap) > 1 {
			t.Fatalf("D02 engine padding is asymmetric: box=%+v text=%+v left=%.2f right=%.2f", boxNode, text, leftGap, rightGap)
		}
	}

	for _, id := range []string{"d03-meter", "d03-progress", "d0n-progress"} {
		node := fixture56Node(root, func(node *html.Node) bool {
			return node.Attribute("id") == id || node.Attribute("class") == id
		})
		boxNode := fixture56BoxByNode(res.root, node)
		if boxNode == nil {
			t.Fatalf("widget %s has no box", id)
		}
		foundFill := false
		for i := boxNode.opStart; i <= boxNode.opEnd && i < len(res.Ops); i++ {
			op := res.Ops[i]
			if op.Kind != OpFillRect || op.W <= 0 || op.H <= 0 || op.W >= boxNode.w-1 {
				continue
			}
			if id == "d0n-progress" && !(op.G > 0.35 && op.G > op.R*1.5 && op.G > op.B*1.5) {
				continue
			}
			if op.H > 4.5 {
				t.Fatalf("widget %s value fill is too thick: op=%+v", id, op)
			}
			foundFill = true
		}
		if !foundFill {
			t.Fatalf("widget %s has no thin value fill", id)
		}
	}

	var noteBox *box
	for _, node := range fixture56Nodes(root, func(node *html.Node) bool { return fixture56Class(node) == "dom-notes" }) {
		candidate := fixture56BoxByNode(res.root, node)
		if candidate == nil {
			continue
		}
		for i := candidate.opStart; i <= candidate.opEnd && i < len(res.Ops); i++ {
			if res.Ops[i].Kind == OpText && strings.Contains(res.Ops[i].Text, "Security: no script execution") {
				noteBox = candidate
				break
			}
		}
		if noteBox != nil {
			break
		}
	}
	if noteBox == nil {
		t.Fatal("security note has no box")
	}
	securityTextY := -1.0
	for i := noteBox.opStart; i <= noteBox.opEnd && i < len(res.Ops); i++ {
		if res.Ops[i].Kind == OpText && strings.Contains(res.Ops[i].Text, "Security: no script execution") {
			securityTextY = res.Ops[i].Y
			break
		}
	}
	if securityTextY < 0 {
		t.Fatal("security note text has no paint op")
	}
	borderBottom := 0.0
	borderTop := math.MaxFloat64
	for i := noteBox.opStart; i <= noteBox.opEnd && i < len(res.Ops); i++ {
		op := res.Ops[i]
		if op.Kind == OpLine && math.Abs(op.X-noteBox.x) < 0.01 && op.H > 0 && op.R > 0.7 && op.G > 0.3 && op.B < 0.1 {
			if op.Y < borderTop {
				borderTop = op.Y
			}
			if op.Y+op.H > borderBottom {
				borderBottom = op.Y + op.H
			}
		}
	}
	if borderTop > securityTextY+1 || borderBottom < securityTextY+1 {
		t.Fatalf("security note rail is not aligned with text: top=%.2f bottom=%.2f "+
			"textY=%.2f", borderTop, borderBottom, securityTextY)
	}

	dag := fixture56Node(root, func(node *html.Node) bool { return node.Attribute("id") == "dependency-dag" })
	dagBox := fixture56BoxByNode(res.root, dag)
	if dagBox == nil {
		t.Fatal("dependency DAG has no box")
	}
	// Text paint is word-oriented, so use stable words from each rule rather
	// than requiring a whole sentence in one OpText record.
	ruleNeedles := []string{"parses", "neutral", "cli.Command", "Subresources", "parallel"}
	rulePages := make([]int, 0, len(ruleNeedles))
	for _, needle := range ruleNeedles {
		page := -1
		for i := dagBox.opStart; i <= dagBox.opEnd && i < len(res.Ops); i++ {
			if res.Ops[i].Kind != OpText || !strings.Contains(res.Ops[i].Text, needle) {
				continue
			}
			for candidatePage, indexes := range res.Pages {
				for _, index := range indexes {
					if index == i {
						page = candidatePage
					}
				}
			}
			if page >= 0 {
				break
			}
		}
		if page < 0 {
			t.Fatalf("dependency DAG rule %q has no painted page", needle)
		}
		rulePages = append(rulePages, page)
	}
	for i := 1; i < len(rulePages); i++ {
		if rulePages[i] != rulePages[0] {
			t.Fatalf("dependency DAG rules split across pages: pages=%v", rulePages)
		}
	}
}
