//nolint:wsl,cyclop,funlen // direct fixture assertions keep each CSS behavior readable
package layout

import "testing"

// TestChromeCaseLegacyDefiniteMainSize adapts
// third_party/blink/web_tests/css3/flexbox/definite-main-size.html. A
// definite column height resolves the percentage child against the flexible
// item's used height instead of treating the percentage as zero.
func TestChromeCaseLegacyDefiniteMainSize(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
html, body { margin: 0; padding: 0; }
.case { display: flex; flex-direction: column; width: 300pt; height: 300pt; border: 3pt solid #222; }
.flex-one { flex: 1 1 0; min-height: 0; }
.percentage-child { width: 100%; height: 50%; }
.marker { width: 50pt; height: 50pt; }
.fixed { flex: 0 0 auto; width: 50pt; height: 50pt; }
`)
	res := layoutHTML(t, `<html><body>
<div class="case" id="definite-column">
  <div class="flex-one" id="flexible-column-item">
    <div class="percentage-child" id="percentage-child">
      <div class="marker">percentage marker</div>
    </div>
  </div>
  <div class="fixed" id="fixed-column-item">fixed marker</div>
</div>
</body></html>`, cssSheet)

	containers := classBoxes(res.root, "case")
	flexible := classBoxes(res.root, "flex-one")
	percentages := classBoxes(res.root, "percentage-child")
	markers := classBoxes(res.root, "marker")
	fixed := classBoxes(res.root, "fixed")
	if len(containers) != 1 || len(flexible) != 1 || len(percentages) != 1 || len(markers) != 1 || len(fixed) != 1 {
		t.Fatalf(
			"boxes: case=%d flexible=%d percentage=%d marker=%d fixed=%d, want 1 each",
			len(containers), len(flexible), len(percentages), len(markers), len(fixed),
		)
	}

	container := containers[0]
	flexItem := flexible[0]
	percentage := percentages[0]
	marker := markers[0]
	fixedItem := fixed[0]

	if !near(container.height, 306) {
		t.Fatalf("definite column outer height = %.2f, want 306 (300 content + 6 border)", container.height)
	}
	if !near(flexItem.height, 250) {
		t.Fatalf("flexible column item height = %.2f, want 250", flexItem.height)
	}
	if !near(percentage.height, 125) {
		t.Fatalf("percentage child height = %.2f, want 125 (50%% of 250)", percentage.height)
	}
	if !near(marker.height, 50) || !near(fixedItem.height, 50) {
		t.Fatalf("fixed heights = marker %.2f/fixed %.2f, want 50/50", marker.height, fixedItem.height)
	}
	if !near(percentage.y, flexItem.y) || !near(fixedItem.y, flexItem.y+flexItem.height) {
		t.Fatalf(
			"column positions: percentage y=%.2f flexible y=%.2f fixed y=%.2f, want nested and following",
			percentage.y, flexItem.y, fixedItem.y,
		)
	}
}

// TestChromeCaseLegacyJustifyContent adapts
// third_party/blink/web_tests/css3/flexbox/flex-justify-content.html. Three
// fixed children expose the free-space placement for each main-axis mode.
func TestChromeCaseLegacyJustifyContent(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
html, body { margin: 0; padding: 0; }
.case { display: flex; width: 600pt; height: 20pt; }
.item { flex: 0 0 100pt; height: 20pt; }
.start { justify-content: flex-start; }
.center { justify-content: center; }
.end { justify-content: flex-end; }
.between { justify-content: space-between; }
`)
	res := layoutHTML(t, `<html><body>
<div class="case start">
  <div class="item">start A</div><div class="item">start B</div><div class="item">start C</div>
</div>
<div class="case center">
  <div class="item">center A</div><div class="item">center B</div><div class="item">center C</div>
</div>
<div class="case end">
  <div class="item">end A</div><div class="item">end B</div><div class="item">end C</div>
</div>
<div class="case between">
  <div class="item">between A</div><div class="item">between B</div><div class="item">between C</div>
</div>
</body></html>`, cssSheet)

	testCases := []struct {
		name  string
		start float64
		gap   float64
	}{
		{name: "start", start: 0, gap: 0},
		{name: "center", start: 150, gap: 0},
		{name: "end", start: 300, gap: 0},
		{name: "between", start: 0, gap: 150},
	}

	containers := classBoxes(res.root, "case")
	items := classBoxes(res.root, "item")
	if len(containers) != len(testCases) || len(items) != len(testCases)*3 {
		t.Fatalf("boxes: cases=%d items=%d, want %d/%d", len(containers), len(items), len(testCases), len(testCases)*3)
	}

	for index, testCase := range testCases {
		container := containers[index]
		first := items[index*3]
		second := items[index*3+1]
		third := items[index*3+2]

		if !near(container.w, 600) {
			t.Fatalf("%s container width = %.2f, want 600", testCase.name, container.w)
		}
		for itemIndex, item := range []*box{first, second, third} {
			if !near(item.w, 100) {
				t.Fatalf("%s item[%d] width = %.2f, want 100", testCase.name, itemIndex, item.w)
			}
		}

		wantFirstX := container.x + testCase.start
		wantSecondX := wantFirstX + 100 + testCase.gap
		wantThirdX := wantSecondX + 100 + testCase.gap
		if !near(first.x, wantFirstX) || !near(second.x, wantSecondX) || !near(third.x, wantThirdX) {
			t.Fatalf(
				"%s positions = %.2f/%.2f/%.2f, want %.2f/%.2f/%.2f",
				testCase.name,
				first.x,
				second.x,
				third.x,
				wantFirstX,
				wantSecondX,
				wantThirdX,
			)
		}
	}
}
