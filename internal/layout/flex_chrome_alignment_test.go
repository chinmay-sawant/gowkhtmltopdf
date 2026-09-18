package layout

import "testing"

// TestChromeFlexJustifyContentFixedItems adapts
// third_party/blink/web_tests/css3/flexbox/flex-justify-content.html.
// It checks main-axis offsets for fixed-size children without repeating the
// column align-items:center coverage in flex_test.go.
//
//nolint:cyclop,funlen,wsl // table-driven fixture checks several justify modes
func TestChromeFlexJustifyContentFixedItems(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		justify string
		start   float64
		gap     float64
	}{
		{name: "flex-start", justify: "flex-start"},
		{name: "center", justify: "center"},
		{name: "flex-end", justify: "flex-end"},
		{name: "space-between", justify: "space-between"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cssSheet := sheet(t, `
html, body { margin: 0; padding: 0; }
.case { display: flex; width: 300pt; justify-content: `+testCase.justify+`; }
.item { width: 40pt; height: 30pt; }
`)
			res := layoutHTML(t, `<html><body>
<div class="case"><div class="item">A</div><div class="item">B</div></div>
</body></html>`, cssSheet)

			containers := classBoxes(res.root, "case")
			items := classBoxes(res.root, "item")
			if len(containers) != 1 || len(items) != 2 {
				t.Fatalf("boxes: containers=%d items=%d, want 1/2", len(containers), len(items))
			}

			container := containers[0]
			if !near(items[0].w, 40) || !near(items[1].w, 40) {
				t.Fatalf("fixed item widths = %.2f/%.2f, want 40/40", items[0].w, items[1].w)
			}

			free := container.w - items[0].w - items[1].w
			switch testCase.justify {
			case "center":
				testCase.start = free / 2
			case "flex-end":
				testCase.start = free
			case "space-between":
				testCase.gap = free
			}

			wantFirstX := container.x + testCase.start
			wantSecondX := wantFirstX + items[0].w + testCase.gap
			if !near(items[0].x, wantFirstX) || !near(items[1].x, wantSecondX) {
				t.Fatalf(
					"justify-content=%s positions = %.2f/%.2f, want %.2f/%.2f",
					testCase.justify,
					items[0].x,
					items[1].x,
					wantFirstX,
					wantSecondX,
				)
			}
		})
	}
}

// TestChromeFlexRowAutoMarginsConsumeMainAxisFreeSpace adapts
// third_party/blink/web_tests/external/wpt/css/css-flexbox/flexbox_margin-auto.html.
// Auto margins absorb the row's positive main-axis space before the remaining
// fixed child is placed. The same auto margins center the first child on the
// cross axis.
//
//nolint:wsl // fixture assertions keep the expected geometry together
func TestChromeFlexRowAutoMarginsConsumeMainAxisFreeSpace(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
html, body { margin: 0; padding: 0; }
.case { display: flex; width: 300pt; height: 80pt; }
.item { width: 40pt; height: 30pt; }
`)
	res := layoutHTML(t, `<html><body>
<div class="case"><div class="item auto" style="margin: auto">A</div><div class="item fixed">B</div></div>
</body></html>`, cssSheet)

	containers := classBoxes(res.root, "case")
	items := classBoxes(res.root, "item")
	if len(containers) != 1 || len(items) != 2 {
		t.Fatalf("boxes: containers=%d items=%d, want 1/2", len(containers), len(items))
	}

	container := containers[0]
	autoItem, fixedItem := items[0], items[1]
	freeX := container.w - autoItem.w - fixedItem.w
	freeY := container.height - autoItem.height
	wantAutoX := container.x + freeX/2
	wantFixedX := wantAutoX + autoItem.w + freeX/2
	wantAutoY := container.y + freeY/2

	if !near(autoItem.x, wantAutoX) || !near(fixedItem.x, wantFixedX) {
		t.Errorf(
			"row auto-margin positions = %.2f/%.2f, want %.2f/%.2f",
			autoItem.x,
			fixedItem.x,
			wantAutoX,
			wantFixedX,
		)
	}

	if !near(autoItem.y, wantAutoY) {
		t.Errorf("auto-margin cross-axis y = %.2f, want %.2f", autoItem.y, wantAutoY)
	}
}
