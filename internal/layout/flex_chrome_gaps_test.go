package layout

import "testing"

// TestChromeFlexColumnGapFlexibleChildren covers case wpt-gap-002-ltr.
// Source: chromium/third_party/blink/web_tests/external/wpt/css/css-flexbox/gap-002-ltr.html
// (manifest source third_party/blink/web_tests/external/wpt/css/css-flexbox/gap-002-ltr.html).
// Expected: the column gap is counted between items only. gap-002-ltr-ref.html
// draws the same boxes with a 20px top margin on every item but the first, so
// nothing is added before the first item or after the last item. Three empty
// flex: 1 1 auto children split the 200pt content height minus two 20pt gaps
// into three equal tracks of 160/3pt each.
func TestChromeFlexColumnGapFlexibleChildren(t *testing.T) {
	t.Parallel()

	const (
		gap      = 20.0
		contentH = 200.0
	)

	cssSheet := sheet(t, `
body { margin: 0 }
.case {
  display: flex;
  flex-direction: column;
  gap: 20pt;
  height: 200pt;
}
.case > div { flex: 1 1 auto }
`)
	res := layoutHTML(t, `<html><body>
<section class="case">
  <div class="item"></div>
  <div class="item"></div>
  <div class="item"></div>
</section>
</body></html>`, cssSheet)

	section := findBoxByClass(t, res, "case")
	items := classBoxes(res.root, "item")

	if len(items) != 3 {
		t.Fatalf("flex items = %d, want 3", len(items))
	}

	first, second, last := items[0], items[1], items[2]

	if !near(section.height, contentH) {
		t.Fatalf("section height = %.2f, want %.2f", section.height, contentH)
	}

	wantItemH := (contentH - 2*gap) / 3
	wantY := []float64{
		section.y,
		section.y + wantItemH + gap,
		section.y + 2*(wantItemH+gap),
	}

	for idx, item := range items {
		if !near(item.height, wantItemH) || !near(item.y, wantY[idx]) {
			t.Errorf("item[%d] y/height = %.3f/%.3f, want %.3f/%.3f",
				idx, item.y, item.height, wantY[idx], wantItemH)
		}
	}

	assertColumnGapSpacing(t, section, first, second, last, contentH, gap)
}

// assertColumnGapSpacing checks the outer edges and the two inter-item gaps of
// a three-item column flex with the given gap.
func assertColumnGapSpacing(t *testing.T, section, first, second, last *box, contentH, gap float64) {
	t.Helper()

	// No gap before the first item: it starts at the section content top.
	if !near(first.y, section.y) {
		t.Fatalf("first item y = %.2f, want content top %.2f", first.y, section.y)
	}
	// The 20pt gap appears only between adjacent items.
	if !near(second.y-(first.y+first.height), gap) || !near(last.y-(second.y+second.height), gap) {
		t.Fatalf("item gaps AB=%.2f BC=%.2f, want %.2f",
			second.y-(first.y+first.height), last.y-(second.y+second.height), gap)
	}
	// No gap after the last item: it ends at the section content bottom.
	if !near(last.y+last.height, section.y+contentH) {
		t.Fatalf("last item bottom = %.2f, want content bottom %.2f",
			last.y+last.height, section.y+contentH)
	}
}

// TestChromeFlexGapGeometryWrapping covers case blink-gap-decorations-basic.
// Source: chromium/third_party/blink/renderer/core/layout/flex/flex_layout_algorithm_test.cc
// (manifest source third_party/blink/renderer/core/layout/flex/flex_layout_algorithm_test.cc),
// TEST_F(FlexLayoutAlgorithmTest, GapDecorationsBasic) at line 143.
// Expected: six fixed 50pt items with 10pt column and row gaps wrap into two
// rows of three inside the 170pt content width. Chromium reports the row gap
// centered at y=57 and column gaps at x=57 and x=117, with the content box
// spanning x 2..172 and y 2..112 in the 2pt border box, which is 174x114.
// The last item therefore ends on the content corner with no trailing gap.
//
//nolint:cyclop,funlen // six-item wrap geometry reads as one fixture
func TestChromeFlexGapGeometryWrapping(t *testing.T) {
	t.Parallel()

	const (
		border   = 2.0
		itemSize = 50.0
		gap      = 10.0
	)

	cssSheet := sheet(t, `
body { margin: 0 }
.case {
  display: flex;
  flex-wrap: wrap;
  column-gap: 10pt;
  row-gap: 10pt;
  border: 2pt solid #608ba8;
  width: 170pt;
}
.case > div {
  flex-shrink: 1;
  width: 50pt;
  height: 50pt;
}
`)
	res := layoutHTML(t, `<html><body>
<div class="case">
  <div class="item">One</div>
  <div class="item">Two</div>
  <div class="item">Three</div>
  <div class="item">Four</div>
  <div class="item">Five</div>
  <div class="item">Six</div>
</div>
</body></html>`, cssSheet)

	caseBox := findBoxByClass(t, res, "case")
	items := classBoxes(res.root, "item")

	if len(items) != 6 {
		t.Fatalf("flex items = %d, want 6", len(items))
	}

	one, two, three := items[0], items[1], items[2]
	four, five, six := items[3], items[4], items[5]

	wantW := 2*border + 3*itemSize + 2*gap
	wantH := 2*border + 2*itemSize + gap

	if !near(caseBox.w, wantW) || !near(caseBox.height, wantH) {
		t.Fatalf("container size = %.2fx%.2f, want %.2fx%.2f", caseBox.w, caseBox.height, wantW, wantH)
	}

	for idx, item := range items {
		if !near(item.w, itemSize) || !near(item.height, itemSize) {
			t.Errorf("item[%d] size = %.2fx%.2f, want %.2fx%.2f",
				idx, item.w, item.height, itemSize, itemSize)
		}
	}

	contentX := caseBox.x + border
	rowOneY := caseBox.y + border
	rowTwoY := rowOneY + itemSize + gap
	wantX := []float64{contentX, contentX + itemSize + gap, contentX + 2*(itemSize+gap)}

	for idx, item := range []*box{one, two, three} {
		if !near(item.x, wantX[idx]) || !near(item.y, rowOneY) {
			t.Errorf("row 1 item[%d] = (%.2f, %.2f), want (%.2f, %.2f)",
				idx, item.x, item.y, wantX[idx], rowOneY)
		}
	}

	for idx, item := range []*box{four, five, six} {
		if !near(item.x, wantX[idx]) || !near(item.y, rowTwoY) {
			t.Errorf("row 2 item[%d] = (%.2f, %.2f), want (%.2f, %.2f)",
				idx, item.x, item.y, wantX[idx], rowTwoY)
		}
	}

	// Main gaps sit between columns on each row and between the two rows.
	if !near(two.x-(one.x+one.w), gap) || !near(three.x-(two.x+two.w), gap) {
		t.Fatalf("column gaps = %.2f/%.2f, want %.2f",
			two.x-(one.x+one.w), three.x-(two.x+two.w), gap)
	}

	if !near(four.y-(one.y+one.height), gap) {
		t.Fatalf("row gap = %.2f, want %.2f", four.y-(one.y+one.height), gap)
	}

	// No trailing gap: the last item reaches the content corner.
	contentRight := caseBox.x + caseBox.w - border
	contentBottom := caseBox.y + caseBox.height - border

	if !near(six.x+six.w, contentRight) || !near(six.y+six.height, contentBottom) {
		t.Fatalf("last item bottom-right = (%.2f, %.2f), want content corner (%.2f, %.2f)",
			six.x+six.w, six.y+six.height, contentRight, contentBottom)
	}
}
