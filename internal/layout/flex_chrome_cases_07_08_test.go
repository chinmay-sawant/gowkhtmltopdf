//nolint:wsl,nlreturn // direct fixture traversal keeps the reference geometry visible
package layout

import (
	"os"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func layoutChromeFlexCase0708(t *testing.T, name string) *Result {
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

		styleSheet, err := css.Parse(node.TextContent())
		if err != nil {
			t.Fatalf("parse stylesheet in Chrome Flex case %q: %v", name, err)
		}
		sheets = append(sheets, styleSheet)
	})

	res, err := Layout(root, Options{Width: testViewport, Height: 800, Sheets: sheets, Background: true})
	if err != nil {
		t.Fatalf("layout Chrome Flex case %q: %v", name, err)
	}

	return res
}

// TestChromeFlexCase07ColumnCrossAxisCenter proves the adapted
// flex-align.html case at the layout-unit boundary. Each auto-sized child
// must have its center on the column container's cross-axis center.
func TestChromeFlexCase07ColumnCrossAxisCenter(t *testing.T) {
	t.Parallel()

	res := layoutChromeFlexCase0708(t, "case-07-legacy-flex-align.html")
	containers := classBoxes(res.root, "case")
	items := classBoxes(res.root, "item")
	if len(containers) != 1 || len(items) != 2 {
		t.Fatalf("case 7 boxes: containers=%d items=%d, want 1/2", len(containers), len(items))
	}

	container := containers[0]
	wantCenter := container.x + container.w/2
	for index, item := range items {
		if item.w >= container.w {
			t.Fatalf("case 7 item %d width=%.2f, want auto-sized child narrower than container %.2f", index, item.w, container.w)
		}

		gotCenter := item.x + item.w/2
		if !near(gotCenter, wantCenter) {
			t.Fatalf("case 7 item %d center=%.2f, want container center %.2f", index, gotCenter, wantCenter)
		}
	}
}

// TestChromeFlexCase08VerticalWritingReference records the Chromium geometry
// for flex-align-vertical-writing-mode.html. In vertical-rl writing mode, a
// row flex direction follows the vertical inline axis, while flex-end places
// the items at the physical left edge for the source's default direction.
// The assertion is based on the exact Chromium fixture geometry.
func TestChromeFlexCase08VerticalWritingReference(t *testing.T) {
	t.Parallel()

	res := layoutChromeFlexCase0708(t, "case-08-legacy-flex-align-vertical-writing.html")
	containers := classBoxes(res.root, "case")
	items := classBoxes(res.root, "item")
	if len(containers) != 1 || len(items) != 3 {
		t.Fatalf("case 8 boxes: containers=%d items=%d, want 1/3", len(containers), len(items))
	}

	container := containers[0]
	for index, item := range items {
		if !near(item.x, container.x) {
			t.Fatalf("case 8 item %d left edge=%.2f, want Chromium left cross edge %.2f", index, item.x, container.x)
		}
		if index == 0 {
			if !near(item.y, container.y) {
				t.Fatalf("case 8 first item y=%.2f, want Chromium main-axis start %.2f", item.y, container.y)
			}
			continue
		}

		previous := items[index-1]
		if !near(item.y, previous.y+previous.height) {
			t.Fatalf("case 8 item %d y=%.2f, want previous bottom %.2f", index, item.y, previous.y+previous.height)
		}
	}
}
