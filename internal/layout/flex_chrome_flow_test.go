package layout

import "testing"

// TestChromeFlexRowWrap converts the line-assignment part of the
// test/chrome wrapping cases into direct box geometry.
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

// TestChromeFlexMultilineWrapReverse covers case legacy-multiline
// (test/chrome/manifest.json goTarget layout-unit).
//
// Source: third_party/blink/web_tests/css3/flexbox/multiline.html
// Expected: wrap-reverse stacks wrapped lines from the cross-end, so the first
// line lands at the bottom of the fixed-height row while every line keeps its
// own cross size. Chromium's horizontal-tb/row/wrap-reverse block records a
// 60x45 container with line heights 10/5/20/10 and line tops 35/30/10/0. This
// port keeps the 60x45 container and the grow-to-fill line widths; each item
// spans its full line height so the assertion isolates line order, line
// membership, and cross offsets. The row-reverse variant reverses the item
// order inside each line, so the first item of a line starts at the right
// (main-start of row-reverse).
//
//nolint:cyclop,funlen,varnamelen // wrap-reverse geometry keeps short item labels
func TestChromeFlexMultilineWrapReverse(t *testing.T) {
	t.Parallel()

	const (
		wantLineOneY   = 35.0 // last wrap-reverse line sits at the cross-end (bottom)
		wantLineTwoY   = 20.0
		wantLineThreeY = 0.0
	)

	run := func(t *testing.T, direction string, wantAX, wantBX float64) {
		t.Helper()

		cssSheet := sheet(t, `
body { margin: 0 }
.row {
  display: flex;
  flex-direction: `+direction+`;
  flex-wrap: wrap-reverse;
  align-content: flex-start;
  width: 60pt;
  height: 45pt;
}
.item { flex: 1 0 25pt }
.item-a { height: 10pt }
.item-b { height: 10pt }
.item-c { flex: 1 0 25pt; height: 15pt }
.item-d { flex: 1 0 50pt; height: 20pt }
`)
		res := layoutHTML(t, `<html><body>
<div class="row">
  <div class="item item-a"></div>
  <div class="item item-b"></div>
  <div class="item item-c"></div>
  <div class="item item-d"></div>
</div>
</body></html>`, cssSheet)

		row := findBoxByClass(t, res, "row")
		a := findBoxByClass(t, res, "item-a")
		b := findBoxByClass(t, res, "item-b")
		c := findBoxByClass(t, res, "item-c")
		d := findBoxByClass(t, res, "item-d")

		if !near(row.w, 60) || !near(row.height, 45) {
			t.Fatalf("container size = %.2fx%.2f, want 60x45", row.w, row.height)
		}

		// Line 1 [A,B]: 2*25pt basis + 10pt free space -> 30pt each.
		// Line 2 [C]: grows 25pt -> 60pt. Line 3 [D]: grows 10pt -> 60pt.
		if !near(a.w, 30) || !near(b.w, 30) || !near(c.w, 60) || !near(d.w, 60) {
			t.Fatalf("item widths A/B/C/D = %.2f/%.2f/%.2f/%.2f, want 30/30/60/60",
				a.w, b.w, c.w, d.w)
		}

		if !near(a.height, 10) || !near(b.height, 10) || !near(c.height, 15) || !near(d.height, 20) {
			t.Fatalf("item heights A/B/C/D = %.2f/%.2f/%.2f/%.2f, want 10/10/15/20",
				a.height, b.height, c.height, d.height)
		}

		// wrap-reverse order: first line at y=35, then 20, then 0.
		if !near(a.y, row.y+wantLineOneY) || !near(b.y, row.y+wantLineOneY) {
			t.Fatalf("line 1 y positions A/B = %.2f/%.2f, want %.2f",
				a.y, b.y, row.y+wantLineOneY)
		}

		if !near(c.y, row.y+wantLineTwoY) {
			t.Fatalf("line 2 y = %.2f, want %.2f", c.y, row.y+wantLineTwoY)
		}

		if !near(d.y, row.y+wantLineThreeY) {
			t.Fatalf("line 3 y = %.2f, want %.2f", d.y, row.y+wantLineThreeY)
		}

		if !near(a.x, row.x+wantAX) || !near(b.x, row.x+wantBX) {
			t.Fatalf("line 1 x positions A/B = %.2f/%.2f, want %.2f/%.2f",
				a.x, b.x, row.x+wantAX, row.x+wantBX)
		}
	}

	t.Run("wrap-reverse", func(t *testing.T) {
		t.Parallel()

		run(t, "row", 0, 30)
	})

	t.Run("row-reverse-wrap-reverse", func(t *testing.T) {
		t.Parallel()

		run(t, "row-reverse", 30, 0)
	})
}

// TestChromeFlexAlignContentWrappedColumns covers case
// legacy-multiline-align-content-column (test/chrome/manifest.json goTarget
// layout-unit).
//
// Source: third_party/blink/web_tests/css3/flexbox/multiline-align-content-horizontal-column.html
// Expected: a 600pt-wide column flex container with flex-wrap: wrap puts the
// 20pt-basis item and the 5pt-basis item on separate 100pt-wide column lines
// (each line takes the container's 20pt main size, and min-width: 100pt is the
// line cross size). align-content places those lines on the cross (horizontal)
// axis: flex-start -> x 0 and 100, center -> 200 and 300, flex-end -> 400 and
// 500; wrap-reverse + flex-start swaps them to 500 and 400. Chromium's
// horizontal-tb/column/ltr block records exactly those offsets.
//
// Blocked: column flex ignores flex-wrap, so all items stay on one line and
// the cross-axis stretch hands each item the full 600pt instead of the 100pt
// line width (internal/layout/flex.go:1528 flowFlexColumn has no wrap path;
// internal/layout/flex.go:1616 flexColumnHeights packs a single line).
//
//nolint:varnamelen // fixture assertions use short item labels
func TestChromeFlexAlignContentWrappedColumns(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		wrap  string
		align string
		wantA float64
		wantB float64
	}{
		{wrap: "wrap", align: "flex-start", wantA: 0, wantB: 100},
		{wrap: "wrap", align: "center", wantA: 200, wantB: 300},
		{wrap: "wrap", align: "flex-end", wantA: 400, wantB: 500},
		{wrap: "wrap-reverse", align: "flex-start", wantA: 500, wantB: 400},
	} {
		t.Run(tc.wrap+"-"+tc.align, func(t *testing.T) {
			t.Parallel()

			cssSheet := sheet(t, `
body { margin: 0 }
.case {
  display: flex;
  flex-direction: column;
  flex-wrap: `+tc.wrap+`;
  align-content: `+tc.align+`;
  width: 600pt;
  height: 20pt;
}
.first { flex: 1 1 20pt; min-width: 100pt }
.second { flex: 1 1 5pt; min-width: 100pt }
`)
			res := layoutHTML(t, `<html><body>
<div class="case">
  <div class="first item-a"></div>
  <div class="second item-b"></div>
</div>
</body></html>`, cssSheet)

			caseBox := findBoxByClass(t, res, "case")
			a := findBoxByClass(t, res, "item-a")
			b := findBoxByClass(t, res, "item-b")

			if !near(a.height, 20) || !near(b.height, 20) {
				t.Fatalf("item main sizes = %.2f/%.2f, want 20/20 (each line takes the container main size)",
					a.height, b.height)
			}

			if !near(a.w, 100) || !near(b.w, 100) {
				t.Fatalf("item cross sizes = %.2f/%.2f, want 100/100 (line cross size)", a.w, b.w)
			}

			if !near(a.x, caseBox.x+tc.wantA) || !near(b.x, caseBox.x+tc.wantB) {
				t.Fatalf("line x offsets A/B = %.2f/%.2f, want %.2f/%.2f",
					a.x, b.x, caseBox.x+tc.wantA, caseBox.x+tc.wantB)
			}

			if !near(a.y, caseBox.y) || !near(b.y, caseBox.y) {
				t.Fatalf("item tops A/B = %.2f/%.2f, want %.2f", a.y, b.y, caseBox.y)
			}
		})
	}
}

// TestChromeFlexColumnReverseMultiline covers case
// wpt-column-reverse-multiline (test/chrome/manifest.json goTarget
// layout-unit).
//
// Source: third_party/blink/web_tests/external/wpt/css/css-flexbox/flex-column-reverse-multiline-item-position.html
// Expected: the 40pt item fills the first column-reverse line, the 80pt item
// wraps to line 2, and the auto height resolves to the taller line (80pt).
// Column-reverse packs each line from the main-start (the bottom) of that
// finalized 80pt height: the 40pt item sits at y = 40 and the 80pt item at
// y = 0, side by side on the cross axis at x = 0 and x = 50. Chromium's
// reference (flex-column-reverse-multiline-item-position-ref.html) pins the
// same 200x80 container with .short at (0,40) and .tall at (50,0).
//
// Blocked: column flex ignores flex-wrap and never forms a second line
// (internal/layout/flex.go:1528); the auto height then falls through
// resolveFlexColumnContentHeight (internal/layout/flex.go:1567) and stacks
// both items into one 120pt column clamped to max-height 100pt.
//
//nolint:varnamelen // fixture assertions use short item labels
func TestChromeFlexColumnReverseMultiline(t *testing.T) {
	t.Parallel()

	t.Skip("blocked: column flex has no wrap path; one clamped line replaces two wrapped lines")

	cssSheet := sheet(t, `
body { margin: 0 }
.case {
  display: flex;
  flex-direction: column-reverse;
  flex-wrap: wrap;
  align-content: flex-start;
  align-items: flex-start;
  width: 200pt;
  max-height: 100pt;
}
.short { width: 50pt; height: 40pt }
.tall { width: 50pt; height: 80pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="case">
  <div class="short item-a"></div>
  <div class="tall item-b"></div>
</div>
</body></html>`, cssSheet)

	caseBox := findBoxByClass(t, res, "case")
	a := findBoxByClass(t, res, "item-a")
	b := findBoxByClass(t, res, "item-b")

	if !near(caseBox.height, 80) {
		t.Fatalf("container height = %.2f, want 80 (taller wrapped line)", caseBox.height)
	}

	if !near(a.w, 50) || !near(a.height, 40) || !near(b.w, 50) || !near(b.height, 80) {
		t.Fatalf("item sizes = %.2fx%.2f and %.2fx%.2f, want 50x40 and 50x80",
			a.w, a.height, b.w, b.height)
	}

	if !near(a.x, caseBox.x) || !near(a.y, caseBox.y+40) {
		t.Fatalf("short item position = (%.2f, %.2f), want (%.2f, %.2f)",
			a.x, a.y, caseBox.x, caseBox.y+40)
	}

	if !near(b.x, caseBox.x+50) || !near(b.y, caseBox.y) {
		t.Fatalf("tall item position = (%.2f, %.2f), want (%.2f, %.2f)",
			b.x, b.y, caseBox.x+50, caseBox.y)
	}
}
