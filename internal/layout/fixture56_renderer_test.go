//nolint:testpackage,wsl,nlreturn,varnamelen,lll // fixture assertions inspect layout internals
package layout

import (
	"fmt"
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

func fixture56LeftRail(op Op) bool {
	return (op.Kind == OpLine && op.W == 0 && op.H > 0) ||
		(op.Kind == OpStrokeRect && op.StrokeMask == StrokeMaskLeft && op.H > 0)
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

// fixture56PageLead is the offset from the current page top. A float
// remainder just below a page multiple is treated as 0, not as ~contentH.
func fixture56PageLead(y, contentH float64) float64 {
	if contentH <= 0 {
		return y
	}

	offset := math.Mod(y, contentH)
	if offset < 0 {
		offset += contentH
	}

	if contentH-offset <= layoutSlack {
		return 0
	}

	return offset
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

func fixture56DAGRulePages(res *Result, rulesBox *box) []int { //nolint:cyclop // rule-to-page lookup is intentionally explicit
	ruleNeedles := []string{"parses", "neutral", "cli.Command", "Subresources", "parallel"}
	rulePages := make([]int, 0, len(ruleNeedles))

	for _, needle := range ruleNeedles {
		page := -1
		for i := rulesBox.opStart; i <= rulesBox.opEnd && i < len(res.Ops); i++ {
			if res.Ops[i].Kind != OpText || !strings.Contains(res.Ops[i].Text, needle) {
				continue
			}

			for candidatePage, indexes := range res.Pages {
				for _, index := range indexes {
					if index == i {
						page = candidatePage
						break
					}
				}
				if page >= 0 {
					break
				}
			}

			if page >= 0 {
				break
			}
		}

		rulePages = append(rulePages, page)
	}

	return rulePages
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
func TestFixture56RendererSeams(t *testing.T) { //nolint:maintidx,paralleltest // fixture seam assertions intentionally remain together
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

	for _, node := range fixture56Nodes(root, func(node *html.Node) bool {
		return strings.Contains(" "+fixture56Class(node)+" ", " div-card ")
	}) {
		boxNode := fixture56BoxByNode(res.root, node)
		found := false

		if boxNode != nil && boxNode.opStart >= 0 && boxNode.opEnd < len(res.Ops) {
			for _, op := range res.Ops[boxNode.opStart : boxNode.opEnd+1] {
				if op.Kind == OpStrokeRect && op.StrokeMask == StrokeMaskTop && op.Radius > 0 {
					found = true

					break
				}
			}
		}

		if !found {
			t.Fatalf("div-card %q has no rounded top-border operation", node.Text)
		}
	}

	for _, node := range fixture56Nodes(root, func(node *html.Node) bool {
		return strings.Contains(" "+fixture56Class(node)+" ", " d04-seam ")
	}) {
		boxNode := fixture56BoxByNode(res.root, node)
		found := false

		if boxNode != nil && boxNode.opStart >= 0 && boxNode.opEnd < len(res.Ops) {
			for _, op := range res.Ops[boxNode.opStart : boxNode.opEnd+1] {
				if op.Kind == OpStrokeRect && op.StrokeMask == StrokeMaskLeft && op.Radius > 0 {
					found = true

					break
				}
			}
		}

		if !found {
			t.Fatalf("d04-seam %q has no rounded left-border operation", node.Text)
		}
	}

	for _, node := range fixture56Nodes(root, func(node *html.Node) bool {
		return strings.Contains(" "+fixture56Class(node)+" ", " pipe-step ")
	}) {
		boxNode := fixture56BoxByNode(res.root, node)
		found := false

		if boxNode != nil && boxNode.opStart >= 0 && boxNode.opEnd < len(res.Ops) {
			for _, op := range res.Ops[boxNode.opStart : boxNode.opEnd+1] {
				if op.Kind == OpStrokeRect && op.StrokeMask == StrokeMaskTop && op.Radius > 0 {
					found = true

					break
				}
			}
		}

		if !found {
			t.Fatalf("pipe-step %q has no rounded top-border operation", node.Text)
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

	sources := fixture56Nodes(root, func(node *html.Node) bool { return fixture56Class(node) == "d02-src" })
	if len(sources) != 3 {
		t.Fatalf("D02 source cards = %d, want 3", len(sources))
	}

	first := fixture56BoxByNode(res.root, sources[0])
	second := fixture56BoxByNode(res.root, sources[1])
	third := fixture56BoxByNode(res.root, sources[2])
	if first == nil || second == nil || third == nil {
		t.Fatalf("D02 source card boxes missing: first=%+v second=%+v third=%+v", first, second, third)
	}

	if math.Abs(first.y-second.y) > 0.01 || third.y <= first.y+0.01 {
		t.Fatalf("D02 source cards did not wrap as authored: y=[%.2f %.2f %.2f]", first.y, second.y, third.y)
	}

	if third.w <= first.w {
		t.Fatalf("D02 wrapped card did not expand: first=%.2f third=%.2f", first.w, third.w)
	}
}

func TestFixture56PageBackgroundDoesNotUseHeroFill(t *testing.T) { //nolint:paralleltest,cyclop // renderer fixture uses shared font state
	root, sheet := loadFixture56(t)
	contentHeight := 841.89 - 2*28.35
	res, err := Layout(root, Options{ //nolint:exhaustruct // fixture uses the standard print layout path
		Width: 595.28 - 2*28.35, Height: contentHeight,
		Background: true, Sheets: []*css.Stylesheet{sheet}, Media: "print", Zoom: 0.98,
	})
	if err != nil {
		t.Fatal(err)
	}
	heroNode := fixture56Node(root, func(node *html.Node) bool { return node.Attribute("id") == "top" })
	pipelineNode := fixture56Node(root, func(node *html.Node) bool { return node.Attribute("id") == "pipeline" })
	hero := fixture56BoxByNode(res.root, heroNode)
	pipeline := fixture56BoxByNode(res.root, pipelineNode)
	if hero == nil || pipeline == nil {
		t.Fatalf("fixture page-1 boxes missing: hero=%+v pipeline=%+v", hero, pipeline)
	}

	const (
		navyR = 0.0784313725
		navyG = 0.1254901961
		navyB = 0.1686274510
		grayR = 0.9333333333
		grayG = 0.9490196078
		grayB = 0.9568627451
	)
	for _, op := range res.Ops {
		if op.Kind != OpFillRect || !fixture56HasRGB(op, navyR, navyG, navyB) ||
			math.Abs(op.X) > 0.01 || math.Abs(op.W-res.Width) > 0.01 {
			continue
		}
		if op.H > hero.height+0.01 {
			t.Fatalf("hero navy fill escaped its box: fill=%+v hero=%+v", op, hero)
		}
	}

	grayUnderPipeline := false
	pipelineBottom := pipeline.y + pipeline.height
	for _, op := range res.Ops {
		if op.Kind != OpFillRect || !fixture56HasRGB(op, grayR, grayG, grayB) ||
			math.Abs(op.X) > 0.01 || math.Abs(op.W-res.Width) > 0.01 {
			continue
		}
		if op.Y <= pipeline.y+0.01 && op.Y+op.H >= pipelineBottom-0.01 {
			grayUnderPipeline = true
			break
		}
	}
	if !grayUnderPipeline {
		t.Fatalf("page background does not cover pipeline area: pipeline=%+v", pipeline)
	}
}

func TestFixture56D02PageStartHasWhiteSectionBackground(t *testing.T) { //nolint:paralleltest,cyclop // renderer fixture uses shared font state
	root, sheet := loadFixture56(t)
	contentHeight := 841.89 - 2*28.35
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

	d02 := fixture56Node(root, func(node *html.Node) bool { return node.Attribute("id") == "domain-02" })
	d02Box := fixture56BoxByNode(res.root, d02)
	if d02Box == nil {
		t.Fatal("D02 section has no layout box")
	}
	for _, op := range res.Ops {
		if op.Kind == OpFillRect && fixture56HasRGB(op, 1, 1, 1) &&
			math.Abs(op.X-d02Box.x) < 0.01 && math.Abs(op.W-d02Box.w) < 0.01 &&
			op.Y <= d02Box.y+0.01 && op.Y+op.H > d02Box.y+1 {
			return
		}
	}
	t.Fatalf("D02 white section background does not reach page start: box=%+v", d02Box)
}

//nolint:cyclop,paralleltest // renderer fixture assertions intentionally stay together
func TestFixture56D02VerticalTabBaselineMatchesBoxTop(t *testing.T) {
	root, sheet := loadFixture56(t)
	contentHeight := 841.89 - 2*28.35
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

	tab := fixture56Node(root, func(node *html.Node) bool { return fixture56Class(node) == "d02-tab" })
	tabBox := fixture56BoxByNode(res.root, tab)
	if tabBox == nil || tabBox.opStart < 0 || tabBox.opEnd >= len(res.Ops) {
		t.Fatal("D02 vertical tab has no painted box")
	}

	backgroundTop := math.Inf(1)
	textBaseline := math.Inf(1)
	for _, op := range res.Ops[tabBox.opStart : tabBox.opEnd+1] {
		if op.Kind == OpFillRect && fixture56HasRGB(op, 0.0588, 0.4627, 0.4314) {
			backgroundTop = op.Y
		}
		if op.Kind == OpText && op.Text == "Root package" {
			textBaseline = op.Y
		}
	}

	if math.IsInf(backgroundTop, 1) || math.IsInf(textBaseline, 1) {
		t.Fatalf("D02 vertical tab paint missing: top=%.2f baseline=%.2f", backgroundTop, textBaseline)
	}
	if textBaseline-backgroundTop > 9 {
		t.Fatalf("D02 vertical tab baseline starts %.2fpt below box top, want <= 9pt", textBaseline-backgroundTop)
	}
}

func TestFixture56ArchitectureSectionsStartOnFreshPages(t *testing.T) { //nolint:paralleltest // renderer fixture uses shared font state
	root, sheet := loadFixture56(t)
	contentHeight := 841.89 - 2*28.35
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

	for _, id := range []string{
		"domain-02", "domain-03", "domain-04", "domain-05", "domain-06",
		"domain-07", "domain-08", "domain-09", "domain-10", "dependency-dag",
		"divergence", "security",
	} {
		node := fixture56Node(root, func(node *html.Node) bool { return node.Attribute("id") == id })
		location, ok := fixture56Location(res, node)
		if !ok {
			t.Fatalf("section %s has no painted location", id)
		}
		pageOffset := fixture56PageLead(location.Y, contentHeight)
		if pageOffset > 24 {
			t.Fatalf("section %s starts %.2fpt into page, want page top", id, pageOffset)
		}
	}
}

func TestFixture56DoesNotRepeatAncestorSideRails(t *testing.T) { //nolint:paralleltest,cyclop // renderer fixture uses shared font state
	root, sheet := loadFixture56(t)
	contentHeight := 841.89 - 2*28.35
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

	for _, op := range res.Ops {
		if !fixture56LeftRail(op) || op.H < contentHeight-1 ||
			(!nearLayout(op.X, 0) && !nearLayout(op.X, res.Width)) {
			continue
		}
		if fixture56HasRGB(op, 0.0784, 0.1255, 0.1686) {
			t.Fatalf("ancestor side rail repeats across pages: op=%+v", op)
		}
	}

	d09 := fixture56Node(root, func(node *html.Node) bool { return node.Attribute("id") == "domain-09" })
	d09Box := fixture56BoxByNode(res.root, d09)
	if d09Box == nil || d09Box.style.BorderRight.Width > 1.01 ||
		!fixture56HasRGBColor(d09Box.style.BorderRight.Color, 0.7961, 0.8353, 0.8824) {
		t.Fatalf("D09 right border does not match shared neutral frame: box=%+v", d09Box)
	}
}

func TestFixture56SectionRailsFlushWithFrame(t *testing.T) { //nolint:cyclop,paralleltest // fixture seam assertions intentionally remain together
	root, sheet := loadFixture56(t)
	contentHeight := 841.89 - 2*28.35
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

	for _, id := range []string{"dependency-dag", "divergence", "security"} {
		node := fixture56Node(root, func(node *html.Node) bool { return node.Attribute("id") == id })
		box := fixture56BoxByNode(res.root, node)
		if box == nil {
			t.Fatalf("section %s has no layout box", id)
		}

		var rail *Op
		for index := box.opStart; index <= box.opEnd && index < len(res.Ops); index++ {
			operation := &res.Ops[index]
			if fixture56LeftRail(*operation) && nearLayout(operation.X, box.x) {
				rail = operation
				break
			}
		}
		if rail == nil {
			t.Fatalf("section %s has no left rail", id)
		}
		if !nearLayout(rail.Y, box.y) || !nearLayout(rail.Y+rail.H, box.y+box.height) {
			t.Fatalf("section %s left rail is inset from frame: rail=%+v box=%+v", id, *rail, *box)
		}
	}
}

func TestFixture56DomainRailsStartAtFrameTop(t *testing.T) { //nolint:cyclop,paralleltest // fixture seam assertions intentionally remain together
	root, sheet := loadFixture56(t)
	contentHeight := 841.89 - 2*28.35
	res, err := Layout(root, Options{ //nolint:exhaustruct // fixture uses the standard print layout path
		Width: 595.28 - 2*28.35, Height: contentHeight,
		Background: true, Sheets: []*css.Stylesheet{sheet}, Media: "print", Zoom: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := Paint(pdf.NewDocument(), res, fixture56PaintOptions()); err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{"domain-06", "domain-08", "domain-09"} {
		node := fixture56Node(root, func(node *html.Node) bool { return node.Attribute("id") == id })
		box := fixture56BoxByNode(res.root, node)
		if box == nil {
			t.Fatalf("section %s has no layout box", id)
		}

		var firstRail *Op
		for index := box.opStart; index <= box.opEnd && index < len(res.Ops); index++ {
			operation := &res.Ops[index]
			if fixture56LeftRail(*operation) && nearLayout(operation.X, box.x) {
				firstRail = operation
				break
			}
		}
		if firstRail == nil || !nearLayout(firstRail.Y, box.y) {
			t.Fatalf("section %s left rail does not start at frame top: rail=%+v box=%+v", id, firstRail, *box)
		}
	}
}

func TestFixture56DAGStaysTogetherAtCLIPageGeometry(t *testing.T) { //nolint:paralleltest // renderer fixture uses shared font state
	root, sheet := loadFixture56(t)

	const (
		pageWidth  = 595.28
		pageHeight = 841.89
		margin     = 10 * 72.0 / 25.4
	)

	res, err := Layout(root, Options{ //nolint:exhaustruct // fixture uses the CLI print geometry
		Width: pageWidth - 2*margin, Height: pageHeight - 2*margin,
		Background: true, Sheets: []*css.Stylesheet{sheet}, Media: "print", Zoom: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := Paint(pdf.NewDocument(), res, PaintOptions{
		PageWidth: pageWidth, PageHeight: pageHeight,
		MarginTop: margin, MarginBottom: margin,
		MarginLeft: margin, MarginRight: margin,
	}); err != nil {
		t.Fatal(err)
	}

	dag := fixture56Node(root, func(node *html.Node) bool { return node.Attribute("id") == "dependency-dag" })
	dagBox := fixture56BoxByNode(res.root, dag)
	if dagBox == nil {
		t.Fatal("dependency DAG has no box")
	}

	rulePages := fixture56DAGRulePages(res, dagBox)
	for i, page := range rulePages {
		if page < 0 {
			t.Fatalf("dependency DAG rule %d has no painted page: pages=%v", i+1, rulePages)
		}
	}

	for i := 1; i < len(rulePages); i++ {
		if rulePages[i] != rulePages[0] {
			t.Fatalf("dependency DAG rules split across pages at CLI geometry: pages=%v", rulePages)
		}
	}
	if got := len(res.Pages); got < 19 || got > 21 {
		t.Fatalf("fixture page count = %d, want 19–21; DAG pages=%v y=%.2f h=%.2f", got, rulePages, dagBox.y, dagBox.height)
	}
}

// TestFixture56NotesCalloutsStayOnOnePage: .dom-notes asides that fit one
// page must not start on one page and finish on the next.
func TestFixture56NotesCalloutsStayOnOnePage(t *testing.T) { //nolint:paralleltest // renderer fixture uses shared font state
	root, sheet := loadFixture56(t)

	const (
		pageWidth  = 595.28
		pageHeight = 841.89
		margin     = 10 * 72.0 / 25.4
	)

	contentHeight := pageHeight - 2*margin
	res, err := Layout(root, Options{ //nolint:exhaustruct // fixture uses CLI print geometry
		Width: pageWidth - 2*margin, Height: contentHeight,
		Background: true, Sheets: []*css.Stylesheet{sheet}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := Paint(pdf.NewDocument(), res, PaintOptions{
		PageWidth: pageWidth, PageHeight: pageHeight,
		MarginTop: margin, MarginBottom: margin,
		MarginLeft: margin, MarginRight: margin,
	}); err != nil {
		t.Fatal(err)
	}

	notes := fixture56Nodes(root, func(node *html.Node) bool {
		return node.Name == "aside" && strings.Contains(fixture56Class(node), "dom-notes")
	})
	if len(notes) == 0 {
		t.Fatal("no .dom-notes asides in fixture-56")
	}

	split := 0

	for _, node := range notes {
		box := fixture56BoxByNode(res.root, node)
		if box == nil || box.height <= layoutSlack || box.height > contentHeight {
			continue
		}

		start := int((box.y + layoutSlack) / contentHeight)
		end := int((box.y + box.height - layoutSlack) / contentHeight)
		section := ""
		for ancestor := node.Parent; ancestor != nil; ancestor = ancestor.Parent {
			if id := ancestor.Attribute("id"); id != "" {
				section = id
				break
			}
		}
		t.Logf("notes %s y=%.2f h=%.2f pages=%d-%d", section, box.y, box.height, start+1, end+1)

		if end > start {
			split++
			t.Errorf("notes aside straddles pages %d-%d: y=%.2f h=%.2f section=%s",
				start+1, end+1, box.y, box.height, section)
		}
	}

	if split > 0 {
		t.Fatalf("%d .dom-notes asides split across a page boundary", split)
	}
}

func TestFixture56D04FlowFitsFourBoxesOnFirstLine(t *testing.T) { //nolint:cyclop,paralleltest // renderer fixture uses shared font state
	root, sheet := loadFixture56(t)
	const margin = 12 * 72 / 25.4
	const pageWidth = 595.28
	const pageHeight = 841.89
	contentWidth := pageWidth - 2*margin
	contentHeight := pageHeight - 2*margin
	res, err := Layout(root, Options{ //nolint:exhaustruct // fixture uses the standard print layout path
		Width: contentWidth, Height: contentHeight,
		Background: true, Sheets: []*css.Stylesheet{sheet}, Media: "print", Zoom: 0.98,
	})
	if err != nil {
		t.Fatal(err)
	}

	boxes := fixture56Nodes(root, func(node *html.Node) bool {
		return fixture56Class(node) == "d04-flowbox"
	})
	if len(boxes) != 5 {
		t.Fatalf("D04 flow boxes = %d, want 5", len(boxes))
	}
	firstRowBottom := 0.0
	for _, node := range boxes[:4] {
		boxNode := fixture56BoxByNode(res.root, node)
		if boxNode != nil && boxNode.y+boxNode.height > firstRowBottom {
			firstRowBottom = boxNode.y + boxNode.height
		}
	}

	for idx, node := range boxes[:4] {
		boxNode := fixture56BoxByNode(res.root, node)
		if boxNode == nil || boxNode.y >= firstRowBottom-0.01 {
			t.Fatalf("D04 flow box %d wrapped early: first row bottom=%.2f box=%+v", idx+1, firstRowBottom, boxNode)
		}
	}

	last := fixture56BoxByNode(res.root, boxes[4])
	if last == nil || last.y < firstRowBottom-0.01 {
		t.Fatalf("D04 fifth flow box stayed on first row: first bottom=%.2f last=%+v", firstRowBottom, last)
	}
}

//nolint:cyclop,funlen // staged pagination diagnostics intentionally log each phase
func TestFixture56D04ACLMatrixDebug(t *testing.T) { //nolint:paralleltest // fixture uses shared font state
	root, sheet := loadFixture56(t)
	res, err := Layout(root, Options{ //nolint:exhaustruct // fixture uses the standard print layout path
		Width: 595.28 - 2*28.35, Height: 841.89 - 2*28.35,
		Background: true, Sheets: []*css.Stylesheet{sheet}, Media: "print", Zoom: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	table := fixture56Node(root, func(node *html.Node) bool { return fixture56Class(node) == "d04-matrix" })
	tableBox := fixture56BoxByNode(res.root, table)
	if tableBox == nil {
		t.Fatal("ACL matrix has no layout box")
	}

	t.Logf("before table x=%.2f y=%.2f w=%.2f h=%.2f rows=%d children=%d", tableBox.x, tableBox.y, tableBox.w, tableBox.height, len(tableBox.rows), len(tableBox.children))
	for rowIdx, row := range tableBox.rows {
		if len(row) == 0 {
			t.Logf("row %d empty", rowIdx)
			continue
		}
		cell := row[0]
		t.Logf("before row %d cell y=%.2f h=%.2f contentH=%.2f op=[%d,%d]", rowIdx, cell.y, cell.height, cell.contentH, cell.opStart, cell.opEnd)
	}
	contentH := 841.89 - 2*28.35
	t.Logf("content height=%.2f", contentH)
	ensureFlowIndex(res, contentH)
	logRows := func(label string) {
		row := tableBox.rows[4][0]
		first, last, top, bottom, ok := rowOpGeometry(tableBox.rows[4])
		prev := tableBox.rows[3][0]
		layoutOut, hi := -1, -1
		if ok {
			layoutOut, hi = int(top/contentH), int(bottom/contentH)
		}
		t.Logf("%s row3=%.2f row4=%.2f h=%.2f ops=[%d,%d] geometry=[%d,%d %.2f..%.2f] pages=%d/%d", label, prev.y, row.y, row.height, row.opStart, row.opEnd, first, last, top, bottom, layoutOut, hi)
	}
	logRows("initial")
	for range 10 {
		if !beforeAlways(res, contentH) {
			break
		}
	}
	logRows("after before")
	snapCrossingTextOps(res, contentH)
	logRows("after snap")
	for iteration := range 10 {
		changed := avoidInside(res, contentH)
		logRows(fmt.Sprintf("fixpoint %d after avoid=%v", iteration, changed))
		if beforeAlways(res, contentH) {
			changed = true
		}
		logRows(fmt.Sprintf("fixpoint %d after before", iteration))
		if afterBreaks(res, contentH) {
			changed = true
		}
		logRows(fmt.Sprintf("fixpoint %d after after-breaks", iteration))
		if rowsIntact(res, contentH) {
			changed = true
		}
		logRows(fmt.Sprintf("fixpoint %d after rows", iteration))
		if keepHeadingWithNext(res, contentH) {
			changed = true
		}
		logRows(fmt.Sprintf("fixpoint %d after headings", iteration))
		if orphansWidows(res, contentH) {
			changed = true
		}
		logRows(fmt.Sprintf("fixpoint %d changed=%v", iteration, changed))
		if !changed {
			break
		}
	}
	if err := Paint(pdf.NewDocument(), res, fixture56PaintOptions()); err != nil {
		t.Fatal(err)
	}

	for rowIdx, row := range tableBox.rows {
		if len(row) > 0 {
			cell := row[0]
			t.Logf("after row %d cell y=%.2f h=%.2f contentH=%.2f op=[%d,%d]", rowIdx, cell.y, cell.height, cell.contentH, cell.opStart, cell.opEnd)
		}
	}
}

func TestFixture56D04ACLMatrixRowsStayContiguous(t *testing.T) { //nolint:paralleltest // fixture uses shared font state
	root, sheet := loadFixture56(t)
	const contentHeight = 841.89 - 2*28.35

	res, err := Layout(root, Options{ //nolint:exhaustruct // fixture uses the standard print layout path
		Width: 595.28 - 2*28.35, Height: contentHeight,
		Background: true, Sheets: []*css.Stylesheet{sheet}, Media: "print", Zoom: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := Paint(pdf.NewDocument(), res, fixture56PaintOptions()); err != nil {
		t.Fatal(err)
	}

	tableNode := fixture56Node(root, func(node *html.Node) bool { return fixture56Class(node) == "d04-matrix" })
	tableBox := fixture56BoxByNode(res.root, tableNode)
	if tableBox == nil || len(tableBox.rows) != 5 {
		t.Fatalf("ACL matrix rows = %d, want 5", len(tableBox.rows))
	}

	firstGap := rowYBounds(tableBox.rows[1], res) - rowYBounds(tableBox.rows[0], res)
	for rowIndex := 2; rowIndex < len(tableBox.rows); rowIndex++ {
		gap := rowYBounds(tableBox.rows[rowIndex], res) - rowYBounds(tableBox.rows[rowIndex-1], res)
		if gap > firstGap*1.5 {
			t.Fatalf("ACL matrix row gap before row %d = %.2fpt, first gap = %.2fpt", rowIndex, gap, firstGap)
		}
	}

	urlsNode := fixture56Node(root, func(node *html.Node) bool { return fixture56Class(node) == "d04-urls" })
	urlsBox := fixture56BoxByNode(res.root, urlsNode)
	if urlsBox == nil {
		t.Fatal("ACL URL paragraph has no layout box")
	}

	page := int(urlsBox.y / contentHeight)
	if urlsBox.y+urlsBox.height > float64(page+1)*contentHeight+1 {
		t.Fatalf("ACL URL paragraph crosses page boundary: y=%.2f h=%.2f page=%d", urlsBox.y, urlsBox.height, page)
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
		if fixture56LeftRail(op) && math.Abs(op.X-d01Box.x) < 0.01 && op.H > 0 &&
			op.G > 0.3 && op.B < 0.6 && op.Y+op.H > railBottom {
			railBottom = op.Y + op.H
		}
	}
	// Footer Y is the baseline; visible ink ends at Y+InkDescent (not Y+H,
	// which is baseline + full line-height and overshoots the line box).
	footerBottom := footer.Y + opVisibleInkHeight(footer)
	if railBottom < footerBottom-1 {
		t.Fatalf("D01 left rail ends at %.2f, before footer ink bottom %.2f", railBottom, footerBottom)
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

		if id == "d03-meter" || id == "d03-progress" {
			wantHeight := boxNode.style.FontSize * 0.98 // fixture layout uses Zoom .98
			if math.Abs(boxNode.height-wantHeight) > 0.5 {
				t.Fatalf("widget %s auto height = %.2f, want native font-sized %.2f", id, boxNode.height, wantHeight)
			}
		}
	}

	gauges := fixture56Node(root, func(node *html.Node) bool { return fixture56Class(node) == "d03-gauges" })
	gaugeBox := fixture56BoxByNode(res.root, gauges)
	if gaugeBox == nil || len(gaugeBox.children) != 2 {
		t.Fatalf("D03 gauge row = box=%+v children=%d, want two flex items", gaugeBox, len(gaugeBox.children))
	}

	for _, child := range gaugeBox.children {
		const authoredGaugeWidthPercent = 46.0
		wantWidth := gaugeBox.w * authoredGaugeWidthPercent / cssPercent
		if math.Abs(child.w-wantWidth) > 0.01 {
			t.Fatalf("D03 gauge width = %.2f, want authored %.2f%% of row %.2f", child.w, authoredGaugeWidthPercent, gaugeBox.w)
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
	rulePages := fixture56DAGRulePages(res, dagBox)
	for i, page := range rulePages {
		if page < 0 {
			t.Fatalf("dependency DAG rule %d has no painted page", i+1)
		}
	}
	for i := 1; i < len(rulePages); i++ {
		if rulePages[i] != rulePages[0] {
			t.Fatalf("dependency DAG rules split across pages: pages=%v", rulePages)
		}
	}
}
