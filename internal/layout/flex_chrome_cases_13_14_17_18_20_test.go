//nolint:wsl // fixture contract checks keep source markers adjacent
package layout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

//nolint:gochecknoglobals // immutable fixture metadata is shared by subtests
var chromeFixtureContracts13To20 = []struct {
	name    string
	fixture string
	markers []string
}{
	{
		name:    "legacy-flex-flow-auto-margins",
		fixture: "case-13-legacy-flex-flow-auto-margins.html",
		markers: []string{
			"flex-flow-auto-margins.html",
			".physical > div { margin: 13px auto 17px auto; }",
			"margin-inline-start: auto",
			"vertical-lr ltr row logical",
			"vertical-rl rtl column-reverse logical",
		},
	},
	{
		name:    "legacy-flex-align-baseline",
		fixture: "case-14-legacy-flex-align-baseline.html",
		markers: []string{
			"flex-align-baseline.html",
			"align-items: baseline",
			"margin-top:20px",
			"vertical-lr ltr row",
			"vertical-rl rtl column-reverse",
		},
	},
	{
		name:    "wpt-flex-basis-011",
		fixture: "case-17-wpt-flex-basis-011.html",
		markers: []string{
			"flex-basis-011.html",
			"flex: 1 0 100%",
			"class=\"flexbox column\"",
			"<div>AAA</div>",
			"<div>BBB</div>",
		},
	},
	{
		name:    "wpt-rtl-flow-reverse",
		fixture: "case-18-wpt-rtl-flow-reverse.html",
		markers: []string{
			"flexbox_rtl-flow-reverse.html",
			"flex-flow: column wrap-reverse",
			"direction: rtl",
			"class=\"span-one\">one</span>",
			"class=\"span-four\">four</span>",
		},
	},
	{
		name:    "wpt-min-size-auto-overflow-clip",
		fixture: "case-20-wpt-min-size-auto-overflow-clip.html",
		markers: []string{
			"min-size-auto-overflow-clip.html",
			"overflow:clip",
			"width: 100px",
			"width:150px;height:50px",
			"min-size-auto-overflow-clip-ref.html",
		},
	},
}

// TestChromeFlexCase13AutoMarginsFixture is the geometry and display-list
// regression for case-13. Two engine defects made the inline-block wrappers
// diverge from the Chromium reference: the shrink-to-fit width counted a
// specified-width flex child's horizontal margins twice, and the BFC-root
// auto height dropped the trailing child margin. A third defect zeroed the
// vertical-rl reverse-flow marker in the orphan-row seal because its center
// sits below the last title ink.
func TestChromeFlexCase13AutoMarginsFixture(t *testing.T) {
	t.Parallel()

	res := layoutChromeFlexCase0102(t, "case-13-legacy-flex-flow-auto-margins.html")
	containers, markers := case13FixtureBoxes(res.root)

	if len(containers) != 5 || len(markers) != 5 {
		t.Fatalf("case-13 boxes: containers=%d markers=%d, want 5 each", len(containers), len(markers))
	}

	// Chromium reference sizes in pt (border-box), one per data-case branch:
	// the wrapper includes the flex child's margins exactly once.
	wantSizes := [][2]float64{{120, 105}, {120, 105}, {120, 105}, {105, 120}, {105, 120}}

	for index, container := range containers {
		if !near(container.w, wantSizes[index][0]) || !near(container.height, wantSizes[index][1]) {
			t.Errorf("container[%d] = %.2fx%.2f, want %.2fx%.2f",
				index, container.w, container.height, wantSizes[index][0], wantSizes[index][1])
		}

		// 20px markers stay square even in the vertical-rl reverse branch.
		if !near(markers[index].w, 15) || !near(markers[index].height, 15) {
			t.Errorf("marker[%d] = %.2fx%.2f, want 15x15", index, markers[index].w, markers[index].height)
		}
	}

	doc := pdf.NewDocument()

	if err := Paint(doc, res, paintOpts()); err != nil {
		t.Fatalf("paint case-13: %v", err)
	}

	if got := visibleBlueFillCount(res.Ops); got != 5 {
		t.Fatalf("blue marker fills after paint = %d, want 5 (orphan-row seal dropped one)", got)
	}
}

// markerInlineStyle13 identifies the five case-13 marker divs.
const markerInlineStyle13 = "height:20px;width:20px"

// case13FixtureBoxes returns the five wrapper boxes (data-case) and the five
// 20px marker boxes in document order.
func case13FixtureBoxes(root *box) ([]*box, []*box) {
	var containers, markers []*box

	var walk func(*box)

	walk = func(boxNode *box) {
		if boxNode == nil {
			return
		}

		if boxNode.node != nil {
			if boxNode.node.Attribute("data-case") != "" {
				containers = append(containers, boxNode)
			}

			if boxNode.node.Name == divElementName && boxNode.node.Attribute("style") == markerInlineStyle13 {
				markers = append(markers, boxNode)
			}
		}

		for _, child := range boxNode.children {
			walk(child)
		}
	}

	walk(root)

	return containers, markers
}

// visibleBlueFillCount counts blue fills that still have a paintable height
// after the pagination and seal passes.
func visibleBlueFillCount(ops []Op) int {
	count := 0

	for _, paintOp := range ops {
		if paintOp.Kind != OpFillRect || paintOp.R > 0.1 || paintOp.G > 0.1 || paintOp.B < 0.9 {
			continue
		}

		if paintOp.H > 1 {
			count++
		}
	}

	return count
}

// TestChromeCases13To20FixtureContracts keeps the five investigated fixtures
// tied to their Chromium interaction. Their layout assertions live in the
// focused geometry tests.
func TestChromeCases13To20FixtureContracts(t *testing.T) {
	t.Parallel()

	for _, tc := range chromeFixtureContracts13To20 {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertChromeFixtureContract(t, tc.fixture, tc.markers)
		})
	}
}

func assertChromeFixtureContract(t *testing.T, fixture string, markers []string) {
	t.Helper()

	path := filepath.Join("..", "..", "test", "chrome", "cases", fixture)
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	text := string(source)
	if !strings.HasPrefix(text, "<!doctype html>") {
		t.Fatal("fixture must start with a doctype")
	}
	if !strings.Contains(text, "Port status: completed") {
		t.Fatal("fixture must record a completed layout gate")
	}
	if strings.Contains(text, "<script") {
		t.Fatal("fixture must remain a static HTML input")
	}

	for _, marker := range markers {
		if !strings.Contains(text, marker) {
			t.Errorf("fixture is missing source marker %q", marker)
		}
	}
}
