package layout

import "testing"

// TestChromeFlexRowWrap converts the line-assignment part of the
// test/Chrome wrapping cases into direct box geometry.
//
//nolint:wsl,varnamelen // fixture assertions use short item labels
func TestChromeFlexRowWrap(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0 }
.row {
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  align-content: flex-start;
  width: 100pt;
  height: 55pt;
  column-gap: 5pt;
  row-gap: 5pt;
}
.item { width: 45pt; height: 20pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="row">
  <div class="item item-a">A</div>
  <div class="item item-b">B</div>
  <div class="item item-c">C</div>
</div>
</body></html>`, cssSheet)

	row := findBoxByClass(t, res, "row")
	a := findBoxByClass(t, res, "item-a")
	b := findBoxByClass(t, res, "item-b")
	c := findBoxByClass(t, res, "item-c")

	if !near(a.w, 45) || !near(a.height, 20) {
		t.Fatalf("row item A size = %.2fx%.2f, want 45x20", a.w, a.height)
	}
	if !near(a.x, row.x) || !near(b.x, a.x+a.w+5) {
		t.Fatalf("first row positions A=%.2f B=%.2f, want A=%.2f B=%.2f", a.x, b.x, row.x, a.x+a.w+5)
	}
	if !near(a.y, b.y) {
		t.Fatalf("first row y positions A=%.2f B=%.2f, want equal", a.y, b.y)
	}
	if !near(c.x, a.x) || !near(c.y, a.y+a.height+5) {
		t.Fatalf("wrapped item C position = (%.2f, %.2f), want (%.2f, %.2f)",
			c.x, c.y, a.x, a.y+a.height+5)
	}
}

// TestChromeFlexReverseFlow converts row-reverse and column-reverse placement
// from the Chrome direction and flow cases into exact main-axis positions.
//
//nolint:varnamelen // fixture assertions use short item labels
func TestChromeFlexReverseFlow(t *testing.T) {
	t.Parallel()

	t.Run("row-reverse", func(t *testing.T) {
		t.Parallel()
		cssSheet := sheet(t, `
body { margin: 0 }
.row { display: flex; flex-direction: row-reverse; width: 90pt; height: 20pt }
.item { width: 30pt; height: 20pt }
`)
		res := layoutHTML(t, `<html><body>
<div class="row">
  <div class="item item-a">A</div>
  <div class="item item-b">B</div>
  <div class="item item-c">C</div>
</div>
</body></html>`, cssSheet)

		row := findBoxByClass(t, res, "row")
		a := findBoxByClass(t, res, "item-a")
		b := findBoxByClass(t, res, "item-b")
		c := findBoxByClass(t, res, "item-c")

		if !near(a.x, row.x+row.w-a.w) || !near(b.x, a.x-b.w) || !near(c.x, b.x-c.w) {
			t.Fatalf("row-reverse x positions A/B/C = %.2f/%.2f/%.2f, want right-to-left from %.2f",
				a.x, b.x, c.x, row.x+row.w)
		}
	})

	t.Run("column-reverse", func(t *testing.T) {
		t.Parallel()
		cssSheet := sheet(t, `
body { margin: 0 }
.column { display: flex; flex-direction: column-reverse; width: 30pt; height: 60pt }
.item { width: 30pt; height: 20pt }
`)
		res := layoutHTML(t, `<html><body>
<div class="column">
  <div class="item item-a">A</div>
  <div class="item item-b">B</div>
  <div class="item item-c">C</div>
</div>
</body></html>`, cssSheet)

		column := findBoxByClass(t, res, "column")
		a := findBoxByClass(t, res, "item-a")
		b := findBoxByClass(t, res, "item-b")
		c := findBoxByClass(t, res, "item-c")

		if !near(a.y, column.y+column.height-a.height) || !near(b.y, a.y-b.height) || !near(c.y, b.y-c.height) {
			t.Fatalf("column-reverse y positions A/B/C = %.2f/%.2f/%.2f, want bottom-to-top from %.2f",
				a.y, b.y, c.y, column.y+column.height)
		}
	})
}

// TestChromeFlexMultilineAlignContent converts the two-line align-content
// behavior from the Chrome multiline cases into exact cross-axis positions.
//
//nolint:wsl,varnamelen // fixture assertions use short item labels
func TestChromeFlexMultilineAlignContent(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0 }
.row {
  display: flex;
  flex-direction: row;
  flex-wrap: wrap;
  align-content: space-between;
  width: 100pt;
  height: 100pt;
}
.item { width: 45pt; height: 20pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="row">
  <div class="item item-a">A</div>
  <div class="item item-b">B</div>
  <div class="item item-c">C</div>
</div>
</body></html>`, cssSheet)

	row := findBoxByClass(t, res, "row")
	a := findBoxByClass(t, res, "item-a")
	b := findBoxByClass(t, res, "item-b")
	c := findBoxByClass(t, res, "item-c")

	if !near(a.y, row.y) || !near(b.y, row.y) {
		t.Fatalf("first align-content line y positions A/B = %.2f/%.2f, want %.2f", a.y, b.y, row.y)
	}
	wantSecondLineY := row.y + row.height - c.height
	if !near(c.y, wantSecondLineY) {
		t.Fatalf("second align-content line y = %.2f, want %.2f", c.y, wantSecondLineY)
	}
	if !near(b.x, a.x+a.w) || !near(c.x, a.x) {
		t.Fatalf("align-content line x positions A/B/C = %.2f/%.2f/%.2f, want second item beside A and C below A",
			a.x, b.x, c.x)
	}
}
