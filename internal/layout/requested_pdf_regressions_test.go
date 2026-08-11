//nolint:testpackage // these checks inspect renderer geometry and display-list ownership
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

func classNodes(root *html.Node, className string) []*html.Node {
	var nodes []*html.Node

	root.Walk(func(node *html.Node) {
		if strings.Contains(" "+node.Attribute("class")+" ", " "+className+" ") {
			nodes = append(nodes, node)
		}
	})

	return nodes
}

//nolint:wsl // recursive tree walk keeps node matching and child traversal adjacent
func classBoxes(root *box, className string) []*box {
	var boxes []*box

	var walk func(*box)
	walk = func(node *box) {
		if node == nil {
			return
		}
		if node.node != nil && strings.Contains(" "+node.node.Attribute("class")+" ", " "+className+" ") {
			boxes = append(boxes, node)
		}
		for _, child := range node.children {
			walk(child)
		}
	}
	walk(root)

	return boxes
}

//nolint:wsl // fixture loading is a linear parse-and-layout setup
func linkedFixtureLayout(t *testing.T, name, stylesheet string) (*html.Node, *Result) {
	t.Helper()

	base := filepath.Join("..", "..", "testdata", "golden")
	htmlData, err := os.ReadFile(filepath.Join(base, name))
	if err != nil {
		t.Fatal(err)
	}

	cssData, err := os.ReadFile(filepath.Join(base, stylesheet))
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

	res, err := Layout(root, Options{ //nolint:exhaustruct // fixture uses print geometry
		Width: 595.28 - 2*28.35, Height: 841.89 - 2*28.35,
		Background: true, Sheets: []*css.Stylesheet{sheet}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}

	return root, res
}

//nolint:wsl // assertions intentionally follow the fixture's page contract
func TestFixture03SummaryTableStaysWithHeading(t *testing.T) {
	t.Parallel()

	res, _ := paintGoldenFixture(t, "fixture-03-multi-page-invoice.html")

	summaryPage := pageOf(t, res, "1. Summary")
	openingPage := pageOf(t, res, "Opening balance")
	transactionsPage := pageOf(t, res, "2. Transactions")
	notesPage := pageOf(t, res, "3. Notes")

	if summaryPage != openingPage {
		t.Fatalf("summary table detached from heading: heading page=%d table page=%d", summaryPage, openingPage)
	}
	if transactionsPage != summaryPage+1 || notesPage != transactionsPage+1 {
		t.Fatalf(
			"section pages = summary %d, transactions %d, notes %d; want consecutive sections",
			summaryPage, transactionsPage, notesPage,
		)
	}
	if len(res.Pages) != 4 {
		t.Fatalf("pages = %d, want 4 for intro plus three forced sections", len(res.Pages))
	}
}

//nolint:wsl // poster geometry assertions mirror the authored stacking order
func TestFixture49PosterMetaDoesNotOverlapCopy(t *testing.T) {
	t.Parallel()

	root, res := linkedFixtureLayout(t, "fixture-49-night-train-poster.html", "theme-print-stories.css")
	copyBox := fixture56BoxByNode(res.root, classNodes(root, "poster-copy")[0])
	meta := fixture56BoxByNode(res.root, classNodes(root, "poster-meta")[0])

	if copyBox == nil || meta == nil {
		t.Fatal("poster copy/meta boxes missing")
	}

	var lastCopyBottom float64
	for _, child := range copyBox.children {
		if child != meta && child.y+child.height > lastCopyBottom {
			lastCopyBottom = child.y + child.height
		}
	}
	if meta.y < lastCopyBottom-0.01 {
		t.Fatalf("poster meta overlaps copy: meta top=%.2f, copy bottom=%.2f", meta.y, lastCopyBottom)
	}
}

//nolint:cyclop,wsl // each assertion names a distinct visual contract
func TestFixture52PassHeadersStayInsideCards(t *testing.T) {
	t.Parallel()

	res, _ := paintGoldenFixture(t, "fixture-52-airline-boarding-pass.html")
	passes := classBoxes(res.root, "pass-top")
	kinds := classBoxes(res.root, "kind")
	if len(passes) != len(kinds) || len(passes) == 0 {
		t.Fatalf("pass headers/kinds = %d/%d", len(passes), len(kinds))
	}

	for idx, kindNode := range kinds {
		passBox := passes[idx]
		kindBox := kindNode
		if passBox == nil || kindBox == nil {
			t.Fatalf("pass %d missing layout box", idx)
		}
		for _, op := range res.Ops[passBox.opStart : passBox.opEnd+1] {
			if op.Kind == OpText && strings.Contains(strings.ToLower(op.Text), "boarding pass") {
				if op.X < passBox.x-0.01 || op.X+op.W > passBox.x+passBox.w+0.01 {
					t.Fatalf("pass %d label ink overflows header: op=%+v header=%+v", idx, op, passBox)
				}
			}
		}
		if kindBox.x < passBox.x-0.01 || kindBox.x+kindBox.w > passBox.x+passBox.w+0.01 {
			t.Fatalf("pass %d kind overflows header: kind=%+v header=%+v", idx, kindBox, passBox)
		}
	}
}

//nolint:cyclop,wsl // table ownership checks are intentionally explicit
func TestFixture55StatusPillsStayInsideCells(t *testing.T) {
	t.Parallel()

	res, _ := paintGoldenFixture(t, "fixture-55-lantern-cooperative-report.html")
	statuses := classBoxes(res.root, "status")
	if len(statuses) == 0 {
		t.Fatal("status pills missing")
	}

	for idx, statusBox := range statuses {
		if statusBox == nil || statusBox.node == nil {
			t.Fatalf("status %d missing pill node", idx)
		}

		inRouteTable := false
		for node := statusBox.node; node != nil; node = node.Parent {
			if strings.Contains(" "+node.Attribute("class")+" ", " route-table ") {
				inRouteTable = true

				break
			}
		}
		if !inRouteTable {
			continue
		}
		cellNode := statusBox.node.Parent
		for cellNode != nil && cellNode.Name != "td" && cellNode.Name != "th" {
			cellNode = cellNode.Parent
		}
		cellBox := fixture56BoxByNode(res.root, cellNode)
		if cellBox == nil {
			t.Fatalf("status %d missing pill/cell box", idx)
		}
		if statusBox.x < cellBox.x-0.01 || statusBox.x+statusBox.w > cellBox.x+cellBox.w+0.01 {
			t.Fatalf("status %d overflows cell: status=%+v cell=%+v", idx, statusBox, cellBox)
		}
	}
}

//nolint:wsl // assertions intentionally keep the matched box and its contract together
func TestFixture55MastheadTextStaysInsideRightHeader(t *testing.T) {
	t.Parallel()

	res, _ := paintGoldenFixture(t, "fixture-55-lantern-cooperative-report.html")
	eyebrows := classBoxes(res.root, "eyebrow")
	if len(eyebrows) == 0 {
		t.Fatal("masthead eyebrow missing")
	}

	eyebrow := eyebrows[0]
	for _, op := range res.Ops[eyebrow.opStart : eyebrow.opEnd+1] {
		if op.Kind != OpText {
			continue
		}
		if op.X < eyebrow.x-0.01 || op.X+op.W > eyebrow.x+eyebrow.w+0.01 {
			t.Fatalf("masthead text escapes right header: op=%+v eyebrow=%+v", op, eyebrow)
		}
	}
}

//nolint:wsl // page geometry setup and assertions are one visual contract
func TestFixture55PageBackgroundReachesFooter(t *testing.T) {
	t.Parallel()

	res, _ := paintGoldenFixture(t, "fixture-55-lantern-cooperative-report.html")
	pages := classBoxes(res.root, "page")
	if len(pages) != 3 {
		t.Fatalf("page boxes = %d, want 3", len(pages))
	}

	for idx, pageBox := range pages {
		if math.Abs(pageBox.height-pageBox.style.MinHeight) > 0.01 {
			t.Fatalf("page %d height = %.2f, want full min-height %.2f", idx+1, pageBox.height, pageBox.style.MinHeight)
		}
	}
}

//nolint:wsl // plan geometry setup and assertions are one visual contract
func TestFixture55ActionCopySharesStatusRowTop(t *testing.T) {
	t.Parallel()

	res, _ := paintGoldenFixture(t, "fixture-55-lantern-cooperative-report.html")
	items := classBoxes(res.root, "plan-item")
	copies := classBoxes(res.root, "plan-copy")
	statuses := classBoxes(res.root, "status")
	if len(items) != 4 || len(copies) != 4 || len(statuses) < 9 {
		t.Fatalf("plan geometry items=%d copies=%d statuses=%d", len(items), len(copies), len(statuses))
	}

	if math.Abs(copies[0].y-statuses[5].y) > 0.01 {
		t.Fatalf("Old station copy/status tops diverge: copy=%.2f status=%.2f", copies[0].y, statuses[5].y)
	}
}

//nolint:cyclop,funlen,wsl // this fixture combines page, flow, and paint contracts
func TestFixture56CaptionAndFlowStayVisible(t *testing.T) {
	t.Parallel()

	root, sheet := loadFixture56(t)
	res, err := Layout(root, Options{ //nolint:exhaustruct // fixture uses standard print layout
		Width: 595.28 - 2*28.35, Height: 841.89 - 2*28.35,
		Background: true, Sheets: []*css.Stylesheet{sheet}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}

	captionPage := -1
	for idx, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(strings.ToLower(op.Text), "exit-code contract") {
			captionPage = idx

			break
		}
	}
	if captionPage < 0 {
		t.Fatal("exit-code contract caption missing from display list")
	}

	flowBoxes := classBoxes(res.root, "d01-flowstep")
	if len(flowBoxes) != 5 {
		t.Fatalf("flow boxes = %d, want 5", len(flowBoxes))
	}
	flowYs := make([]float64, 0, len(flowBoxes))
	for _, flowBox := range flowBoxes {
		flowYs = append(flowYs, flowBox.y)
	}
	for idx := 1; idx < len(flowYs); idx++ {
		if flowYs[idx] != flowYs[0] {
			t.Fatalf("flow labels wrapped unexpectedly: ys=%v", flowYs)
		}
	}

	tab := classBoxes(res.root, "d02-tab")
	if len(tab) != 1 {
		t.Fatalf("vertical tab boxes = %d, want one", len(tab))
	}
	var tabText []Op
	for _, op := range res.Ops[tab[0].opStart : tab[0].opEnd+1] {
		if op.Kind == OpText {
			tabText = append(tabText, op)
		}
	}
	if len(tabText) != 1 {
		t.Fatalf("vertical tab text ops = %d, want one", len(tabText))
	}
	vertical := tabText[0]
	if vertical.RotateDeg != -90 || vertical.X < tab[0].x-0.01 || vertical.X > tab[0].x+tab[0].w+0.01 ||
		vertical.Y < tab[0].y-0.01 || vertical.Y+vertical.W > tab[0].y+tab[0].height+1 {
		t.Fatalf("vertical tab text is outside its rail: op=%+v tab=%+v", vertical, tab[0])
	}

	var pure []Op
	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "stdlib + internal only") {
			pure = append(pure, op)
		}
	}
	if len(pure) != 1 || !strings.Contains(pure[0].Text, "only") {
		t.Fatalf("pure-Go label split across inline runs: ops=%+v", pure)
	}
}
