package layout

import "testing"

// TestChromeFlexVerticalWritingAlignment ports test/chrome case
// legacy-flex-align-vertical-writing.
//
// Source: chromium/third_party/blink/web_tests/css3/flexbox/flex-align-vertical-writing-mode.html
// Expected: vertical writing modes remap physical positions while preserving
// logical alignment. buildFlex maps flex axes from physical top and left and
// never reads WritingMode (internal/layout/flex.go:84), so the vertical-rl
// assertions are kept strict and skipped. The horizontal-tb subtests below run
// the same logical alignment decisions on the supported axis.
//
//nolint:cyclop,funlen,wsl,gocognit,gocyclo // Chromium fixture geometry keeps short labels
func TestChromeFlexVerticalWritingAlignment(t *testing.T) {
	t.Parallel()

	t.Run("horizontal-tb-logical-alignment", func(t *testing.T) {
		t.Parallel()

		cssSheet := sheet(t, `
body { margin: 0 }
.flexcase { display: flex; flex-direction: column; width: 100pt; height: 100pt }
.box { height: 10pt }
.box-start { align-self: flex-start; width: 50pt }
.box-center { align-self: center; width: 50pt }
.box-end { align-self: flex-end; width: 50pt }
.box-zero { align-self: center }
`)
		res := layoutHTML(t, `<html><body>
<div class="flexcase">
  <div class="box box-start"></div>
  <div class="box box-center"></div>
  <div class="box box-end"></div>
  <div class="box box-zero"></div>
  <div class="box box-stretch"></div>
</div>
</body></html>`, cssSheet)

		flexcase := findBoxByClass(t, res, "flexcase")
		start := findBoxByClass(t, res, "box-start")
		center := findBoxByClass(t, res, "box-center")
		end := findBoxByClass(t, res, "box-end")
		zero := findBoxByClass(t, res, "box-zero")
		stretch := findBoxByClass(t, res, "box-stretch")

		if !near(flexcase.w, 100) || !near(flexcase.height, 100) {
			t.Fatalf("container size = %.2fx%.2f, want 100x100", flexcase.w, flexcase.height)
		}
		if !near(start.w, 50) || !near(start.x, flexcase.x) {
			t.Fatalf("flex-start item x=%.2f w=%.2f, want x=%.2f w=50", start.x, start.w, flexcase.x)
		}
		if !near(center.w, 50) || !near(center.x, flexcase.x+25) {
			t.Fatalf("center item x=%.2f w=%.2f, want x=%.2f w=50", center.x, center.w, flexcase.x+25)
		}
		if !near(end.w, 50) || !near(end.x, flexcase.x+50) {
			t.Fatalf("flex-end item x=%.2f w=%.2f, want x=%.2f w=50", end.x, end.w, flexcase.x+50)
		}
		if !near(zero.w, 0) || !near(zero.x, flexcase.x+50) {
			t.Fatalf("center zero-width item x=%.2f w=%.2f, want x=%.2f w=0", zero.x, zero.w, flexcase.x+50)
		}
		if !near(stretch.w, 100) || !near(stretch.x, flexcase.x) {
			t.Fatalf("stretched item x=%.2f w=%.2f, want x=%.2f w=100", stretch.x, stretch.w, flexcase.x)
		}
		if !near(start.y, flexcase.y) || !near(center.y, flexcase.y+10) ||
			!near(end.y, flexcase.y+20) || !near(zero.y, flexcase.y+30) ||
			!near(stretch.y, flexcase.y+40) {
			t.Fatalf("main-axis y = %.2f/%.2f/%.2f/%.2f/%.2f, want +0/+10/+20/+30/+40 from %.2f",
				start.y, center.y, end.y, zero.y, stretch.y, flexcase.y)
		}
	})

	t.Run("horizontal-tb-row-stretch", func(t *testing.T) {
		t.Parallel()

		// Source container 1: three flex: 1 items under the default
		// align-items: stretch. In horizontal-tb the shared cross size is the
		// physical height, so each item stretches to 100pt and the main axis
		// splits 100pt three ways.
		cssSheet := sheet(t, `
body { margin: 0 }
.flexcase { display: flex; width: 100pt; height: 100pt }
.one { flex: 1 }
`)
		res := layoutHTML(t, `<html><body>
<div class="flexcase">
  <div class="one"></div>
  <div class="one"></div>
  <div class="one"></div>
</div>
</body></html>`, cssSheet)

		flexcase := findBoxByClass(t, res, "flexcase")
		items := classBoxes(res.root, "one")
		if len(items) != 3 {
			t.Fatalf("stretch items = %d, want 3", len(items))
		}
		for idx, item := range items {
			if !near(item.w, 100.0/3) || !near(item.height, 100) {
				t.Fatalf("item %d size = %.2fx%.2f, want 33.33x100", idx, item.w, item.height)
			}
			if !near(item.x, flexcase.x+100.0/3*float64(idx)) || !near(item.y, flexcase.y) {
				t.Fatalf("item %d at (%.2f, %.2f), want (%.2f, %.2f)",
					idx, item.x, item.y, flexcase.x+100.0/3*float64(idx), flexcase.y)
			}
		}
	})

	t.Run("vertical-rl-cross-axis-remap", func(t *testing.T) {
		t.Parallel()

		// Chromium flex-align-vertical-writing-mode.html container 5, with the
		// source's auto block size pinned to the 100px it resolves to: in
		// vertical-rl the cross axis runs right to left, so align-self:
		// flex-start maps to the right edge, center to the middle, and flex-end
		// to the left edge.
		// Blocked: buildFlex never consults WritingMode when it maps the main
		// and cross axes (internal/layout/flex.go:84).
		cssSheet := sheet(t, `
body { margin: 0 }
.flexcase { display: flex; width: 100pt; height: 100pt; writing-mode: vertical-rl }
.vbox { flex: 1 }
.v-start { align-self: flex-start }
.v-start-fifty { align-self: flex-start; width: 50pt }
.v-center { align-self: center; width: 50pt }
.v-end { align-self: flex-end; width: 50pt }
`)
		res := layoutHTML(t, `<html><body>
<div class="flexcase">
  <div class="vbox v-start"></div>
  <div class="vbox v-start-fifty"></div>
  <div class="vbox v-center"></div>
  <div class="vbox v-end"></div>
</div>
</body></html>`, cssSheet)

		flexcase := findBoxByClass(t, res, "flexcase")
		start := findBoxByClass(t, res, "v-start")
		startFifty := findBoxByClass(t, res, "v-start-fifty")
		center := findBoxByClass(t, res, "v-center")
		end := findBoxByClass(t, res, "v-end")

		if !near(flexcase.w, 100) {
			t.Fatalf("vertical-rl container width = %.2f, want 100", flexcase.w)
		}
		if !near(start.w, 0) || !near(start.x, flexcase.x+100) {
			t.Fatalf("flex-start item x=%.2f w=%.2f, want x=%.2f w=0", start.x, start.w, flexcase.x+100)
		}
		if !near(startFifty.w, 50) || !near(startFifty.x, flexcase.x+50) {
			t.Fatalf("flex-start 50 wide item x=%.2f w=%.2f, want x=%.2f w=50",
				startFifty.x, startFifty.w, flexcase.x+50)
		}
		if !near(center.w, 50) || !near(center.x, flexcase.x+25) {
			t.Fatalf("center item x=%.2f w=%.2f, want x=%.2f w=50", center.x, center.w, flexcase.x+25)
		}
		if !near(end.w, 50) || !near(end.x, flexcase.x) {
			t.Fatalf("flex-end item x=%.2f w=%.2f, want x=%.2f w=50", end.x, end.w, flexcase.x)
		}
	})
}

// TestChromeFlexFlowOrientations ports test/chrome case
// legacy-flex-flow-orientations.
//
// Source: chromium/third_party/blink/web_tests/css3/flexbox/flex-flow-orientations.html
// Expected: direction, writing mode, row or column, and reverse flow change
// logical start and end positions. The source runs a generated matrix of two
// 20x20 items inside a 100x100 flex container; the table below uses its exact
// expected offsets. Forward flows in horizontal-tb ltr are supported. Reverse
// flows pack the reversed item list at main-start instead of main-end, and
// direction: rtl plus vertical writing modes are ignored, so those rows are
// asserted strictly and skipped.
//
//nolint:cyclop,funlen,wsl,gocognit,gocyclo,maintidx,varnamelen // Chromium expectation matrix keeps short labels
func TestChromeFlexFlowOrientations(t *testing.T) {
	t.Parallel()

	orientCSS := `
body { margin: 0 }
.flexcase { display: flex; width: 100pt; height: 100pt }
.item { flex: none; width: 20pt; height: 20pt }
`
	orientHTML := `<html><body><div class="flexcase">` +
		`<div class="item item-a"></div><div class="item item-b"></div>` +
		`</div></body></html>`

	t.Run("horizontal-tb-ltr-row", func(t *testing.T) {
		t.Parallel()

		res := layoutHTML(t, orientHTML, sheet(t, orientCSS))
		flexcase := findBoxByClass(t, res, "flexcase")
		a := findBoxByClass(t, res, "item-a")
		b := findBoxByClass(t, res, "item-b")

		if !near(a.w, 20) || !near(a.height, 20) || !near(b.w, 20) || !near(b.height, 20) {
			t.Fatalf("item sizes = %.1fx%.1f %.1fx%.1f, want 20x20 each", a.w, a.height, b.w, b.height)
		}
		if !near(a.x, flexcase.x) || !near(b.x, flexcase.x+20) {
			t.Fatalf("row item x = %.2f/%.2f, want %.2f/%.2f", a.x, b.x, flexcase.x, flexcase.x+20)
		}
		if !near(a.y, flexcase.y) || !near(b.y, flexcase.y) {
			t.Fatalf("row item y = %.2f/%.2f, want %.2f", a.y, b.y, flexcase.y)
		}
	})

	t.Run("horizontal-tb-ltr-column", func(t *testing.T) {
		t.Parallel()

		res := layoutHTML(t, orientHTML, sheet(t, orientCSS+".flexcase { flex-direction: column }"))
		flexcase := findBoxByClass(t, res, "flexcase")
		a := findBoxByClass(t, res, "item-a")
		b := findBoxByClass(t, res, "item-b")

		if !near(a.x, flexcase.x) || !near(b.x, flexcase.x) {
			t.Fatalf("column item x = %.2f/%.2f, want %.2f", a.x, b.x, flexcase.x)
		}
		if !near(a.y, flexcase.y) || !near(b.y, flexcase.y+20) {
			t.Fatalf("column item y = %.2f/%.2f, want %.2f/%.2f", a.y, b.y, flexcase.y, flexcase.y+20)
		}
	})

	t.Run("horizontal-tb-ltr-row-reverse", func(t *testing.T) {
		t.Parallel()

		// Chromium horizontal-tb/row-reverse/ltr: item-a at (80, 0), item-b at
		// (60, 0). Blocked: flowFlexRow reverses the item list and then packs it
		// at main-start, so free space stays at main-end
		// (internal/layout/flex.go:306, internal/layout/flex.go:1101).
		res := layoutHTML(t, orientHTML, sheet(t, orientCSS+".flexcase { flex-direction: row-reverse }"))
		flexcase := findBoxByClass(t, res, "flexcase")
		a := findBoxByClass(t, res, "item-a")
		b := findBoxByClass(t, res, "item-b")

		if !near(a.x, flexcase.x+80) || !near(b.x, flexcase.x+60) {
			t.Fatalf("row-reverse item x = %.2f/%.2f, want %.2f/%.2f", a.x, b.x, flexcase.x+80, flexcase.x+60)
		}
		if !near(a.y, flexcase.y) || !near(b.y, flexcase.y) {
			t.Fatalf("row-reverse item y = %.2f/%.2f, want %.2f", a.y, b.y, flexcase.y)
		}
	})

	t.Run("horizontal-tb-ltr-column-reverse", func(t *testing.T) {
		t.Parallel()

		// Chromium horizontal-tb/column-reverse/ltr: item-a at (0, 80), item-b
		// at (0, 60). Blocked by the same reversed-list packing as row-reverse
		// (internal/layout/flex.go:1539).
		res := layoutHTML(t, orientHTML, sheet(t, orientCSS+".flexcase { flex-direction: column-reverse }"))
		flexcase := findBoxByClass(t, res, "flexcase")
		a := findBoxByClass(t, res, "item-a")
		b := findBoxByClass(t, res, "item-b")

		if !near(a.x, flexcase.x) || !near(b.x, flexcase.x) {
			t.Fatalf("column-reverse item x = %.2f/%.2f, want %.2f", a.x, b.x, flexcase.x)
		}
		if !near(a.y, flexcase.y+80) || !near(b.y, flexcase.y+60) {
			t.Fatalf("column-reverse item y = %.2f/%.2f, want %.2f/%.2f", a.y, b.y, flexcase.y+80, flexcase.y+60)
		}
	})

	t.Run("horizontal-tb-rtl-matrix", func(t *testing.T) {
		t.Parallel()

		// Chromium expectations horizontal-tb/rtl, item-a then item-b:
		// row (80,0) (60,0); row-reverse (0,0) (20,0); column (80,0) (80,20);
		// column-reverse (80,80) (80,60). Blocked: flex.go ignores
		// style.Direction when it places items (internal/layout/flex.go:84).
		orient := []struct {
			flow   string
			x1, y1 float64
			x2, y2 float64
		}{
			{flow: "row", x1: 80, y1: 0, x2: 60, y2: 0},
			{flow: "row-reverse", x1: 0, y1: 0, x2: 20, y2: 0},
			{flow: "column", x1: 80, y1: 0, x2: 80, y2: 20},
			{flow: "column-reverse", x1: 80, y1: 80, x2: 80, y2: 60},
		}
		for _, want := range orient {
			res := layoutHTML(t, orientHTML,
				sheet(t, orientCSS+".flexcase { direction: rtl; flex-direction: "+want.flow+" }"))
			flexcase := findBoxByClass(t, res, "flexcase")
			a := findBoxByClass(t, res, "item-a")
			b := findBoxByClass(t, res, "item-b")

			if !near(a.x, flexcase.x+want.x1) || !near(a.y, flexcase.y+want.y1) ||
				!near(b.x, flexcase.x+want.x2) || !near(b.y, flexcase.y+want.y2) {
				t.Fatalf("rtl %s items = (%.2f, %.2f)/(%.2f, %.2f), want (%.2f, %.2f)/(%.2f, %.2f)",
					want.flow, a.x, a.y, b.x, b.y,
					flexcase.x+want.x1, flexcase.y+want.y1, flexcase.x+want.x2, flexcase.y+want.y2)
			}
		}
	})

	t.Run("vertical-writing-mode-matrix", func(t *testing.T) {
		t.Parallel()

		// Chromium expectations for vertical-lr and vertical-rl from the same
		// source table. In vertical-lr row the main axis is vertical and the
		// cross axis left to right; vertical-rl flips the cross axis.
		// Blocked: buildFlex ignores writing-mode entirely
		// (internal/layout/flex.go:84, internal/layout/flex.go:89).
		vertical := []struct {
			mode      string
			flow      string
			direction string
			x1, y1    float64
			x2, y2    float64
		}{
			{mode: "vertical-lr", flow: "column", direction: "ltr", x1: 0, y1: 0, x2: 20, y2: 0},
			{mode: "vertical-lr", flow: "column", direction: "rtl", x1: 0, y1: 80, x2: 20, y2: 80},
			{mode: "vertical-lr", flow: "column-reverse", direction: "ltr", x1: 80, y1: 0, x2: 60, y2: 0},
			{mode: "vertical-lr", flow: "column-reverse", direction: "rtl", x1: 80, y1: 80, x2: 60, y2: 80},
			{mode: "vertical-lr", flow: "row", direction: "ltr", x1: 0, y1: 0, x2: 0, y2: 20},
			{mode: "vertical-lr", flow: "row", direction: "rtl", x1: 0, y1: 80, x2: 0, y2: 60},
			{mode: "vertical-lr", flow: "row-reverse", direction: "ltr", x1: 0, y1: 80, x2: 0, y2: 60},
			{mode: "vertical-lr", flow: "row-reverse", direction: "rtl", x1: 0, y1: 0, x2: 0, y2: 20},
			{mode: "vertical-rl", flow: "column", direction: "ltr", x1: 80, y1: 0, x2: 60, y2: 0},
			{mode: "vertical-rl", flow: "column", direction: "rtl", x1: 80, y1: 80, x2: 60, y2: 80},
			{mode: "vertical-rl", flow: "column-reverse", direction: "ltr", x1: 0, y1: 0, x2: 20, y2: 0},
			{mode: "vertical-rl", flow: "column-reverse", direction: "rtl", x1: 0, y1: 80, x2: 20, y2: 80},
			{mode: "vertical-rl", flow: "row", direction: "ltr", x1: 80, y1: 0, x2: 80, y2: 20},
			{mode: "vertical-rl", flow: "row", direction: "rtl", x1: 80, y1: 80, x2: 80, y2: 60},
			{mode: "vertical-rl", flow: "row-reverse", direction: "ltr", x1: 80, y1: 80, x2: 80, y2: 60},
			{mode: "vertical-rl", flow: "row-reverse", direction: "rtl", x1: 80, y1: 0, x2: 80, y2: 20},
		}
		for _, want := range vertical {
			cssText := orientCSS + ".flexcase { writing-mode: " + want.mode +
				"; direction: " + want.direction + "; flex-direction: " + want.flow + " }"
			res := layoutHTML(t, orientHTML, sheet(t, cssText))
			flexcase := findBoxByClass(t, res, "flexcase")
			a := findBoxByClass(t, res, "item-a")
			b := findBoxByClass(t, res, "item-b")
			if !near(a.x, flexcase.x+want.x1) || !near(a.y, flexcase.y+want.y1) ||
				!near(b.x, flexcase.x+want.x2) || !near(b.y, flexcase.y+want.y2) {
				t.Fatalf("%s %s %s items = (%.2f, %.2f)/(%.2f, %.2f), want (%.2f, %.2f)/(%.2f, %.2f)",
					want.mode, want.direction, want.flow, a.x, a.y, b.x, b.y,
					flexcase.x+want.x1, flexcase.y+want.y1, flexcase.x+want.x2, flexcase.y+want.y2)
			}
		}
	})
}

// TestChromeFlexFlowDirectionPadding ports test/chrome case legacy-flex-flow.
//
// Source: chromium/third_party/blink/web_tests/css3/flexbox/flex-flow.html
// Expected: logical padding and margins stay attached to logical edges across
// reverse directions and axes. The supported subtests use the source's column
// containers, where items fill the 600pt main size exactly, so reverse packing
// and flex basis 1:2:1 resolve to the Chromium heights and y offsets. The row
// containers depend on fixed margins being counted in free space, and the rtl
// and vertical containers depend on direction and writing-mode mapping, both
// of which are unsupported, so those rows are asserted strictly and skipped.
//
//nolint:cyclop,funlen,wsl,gocognit,gocyclo,maintidx // Chromium fixture geometry keeps short labels
func TestChromeFlexFlowDirectionPadding(t *testing.T) {
	t.Parallel()

	columnCSS := `
body { margin: 0 }
.flexcase { display: flex; width: 600pt; height: 600pt }
.lead { flex: 1 0 0; margin: auto 200pt auto 150pt }
.mid { flex: 2 0 0; padding-inline-start: 200pt }
.tail { flex: 1 0 0; margin-inline-end: 100pt }
`
	columnHTML := `<html><body>
<div class="flexcase">
  <div class="lead"></div>
  <div class="mid"><div class="inner"></div></div>
  <div class="tail"></div>
</div>
</body></html>`

	t.Run("column-ltr-basis-heights", func(t *testing.T) {
		t.Parallel()

		res := layoutHTML(t, columnHTML, sheet(t, columnCSS+".flexcase { flex-direction: column }"))
		flexcase := findBoxByClass(t, res, "flexcase")
		lead := findBoxByClass(t, res, "lead")
		mid := findBoxByClass(t, res, "mid")
		tail := findBoxByClass(t, res, "tail")
		inner := findBoxByClass(t, res, "inner")
		if !near(lead.height, 150) || !near(lead.y, flexcase.y) {
			t.Fatalf("lead = h%.2f y%.2f, want h150 y%.2f", lead.height, lead.y, flexcase.y)
		}
		if !near(mid.height, 300) || !near(mid.y, flexcase.y+150) {
			t.Fatalf("mid = h%.2f y%.2f, want h300 y%.2f", mid.height, mid.y, flexcase.y+150)
		}
		if !near(tail.height, 150) || !near(tail.y, flexcase.y+450) {
			t.Fatalf("tail = h%.2f y%.2f, want h150 y%.2f", tail.height, tail.y, flexcase.y+450)
		}
		if !near(inner.x, flexcase.x+200) || !near(inner.y, flexcase.y+150) {
			t.Fatalf("padding-inline-start inner at (%.2f, %.2f), want (%.2f, %.2f)",
				inner.x, inner.y, flexcase.x+200, flexcase.y+150)
		}
	})

	t.Run("column-reverse-ltr-basis-heights", func(t *testing.T) {
		t.Parallel()

		res := layoutHTML(t, columnHTML, sheet(t, columnCSS+".flexcase { flex-direction: column-reverse }"))
		flexcase := findBoxByClass(t, res, "flexcase")
		lead := findBoxByClass(t, res, "lead")
		mid := findBoxByClass(t, res, "mid")
		tail := findBoxByClass(t, res, "tail")
		inner := findBoxByClass(t, res, "inner")

		if !near(lead.height, 150) || !near(lead.y, flexcase.y+450) {
			t.Fatalf("column-reverse lead = h%.2f y%.2f, want h150 y%.2f", lead.height, lead.y, flexcase.y+450)
		}
		if !near(mid.height, 300) || !near(mid.y, flexcase.y+150) {
			t.Fatalf("column-reverse mid = h%.2f y%.2f, want h300 y%.2f", mid.height, mid.y, flexcase.y+150)
		}
		if !near(tail.height, 150) || !near(tail.y, flexcase.y) {
			t.Fatalf("column-reverse tail = h%.2f y%.2f, want h150 y%.2f", tail.height, tail.y, flexcase.y)
		}
		if !near(inner.x, flexcase.x+200) || !near(inner.y, flexcase.y+150) {
			t.Fatalf("column-reverse inner at (%.2f, %.2f), want (%.2f, %.2f)",
				inner.x, inner.y, flexcase.x+200, flexcase.y+150)
		}
	})

	t.Run("row-ltr-padding-and-margins", func(t *testing.T) {
		t.Parallel()

		// Chromium row container 1: lead w75 x0, mid w350 x75 with inner x275,
		// tail w75 x425. Blocked: fixed margins are excluded from the row free
		// space computation and dropped from the item position
		// (internal/layout/flex.go:1049, internal/layout/flex.go:1294).
		rowCSS := `
body { margin: 0 }
.flexcase { display: flex; flex-direction: row; width: 600pt }
.lead { flex: 1 0 0; margin: 0 auto }
.mid { flex: 2 0 0; padding-inline-start: 200pt }
.tail { flex: 1 0 0; margin-inline-end: 100pt }
`
		res := layoutHTML(t, columnHTML, sheet(t, rowCSS))
		flexcase := findBoxByClass(t, res, "flexcase")
		lead := findBoxByClass(t, res, "lead")
		mid := findBoxByClass(t, res, "mid")
		tail := findBoxByClass(t, res, "tail")
		inner := findBoxByClass(t, res, "inner")

		if !near(lead.w, 75) || !near(lead.x, flexcase.x) {
			t.Fatalf("row lead = w%.2f x%.2f, want w75 x%.2f", lead.w, lead.x, flexcase.x)
		}
		if !near(mid.w, 350) || !near(mid.x, flexcase.x+75) || !near(inner.x, flexcase.x+275) {
			t.Fatalf("row mid = w%.2f x%.2f inner%.2f, want w350 x%.2f inner%.2f",
				mid.w, mid.x, inner.x, flexcase.x+75, flexcase.x+275)
		}
		if !near(tail.w, 75) || !near(tail.x, flexcase.x+425) {
			t.Fatalf("row tail = w%.2f x%.2f, want w75 x%.2f", tail.w, tail.x, flexcase.x+425)
		}
	})

	t.Run("column-rtl-logical-edges", func(t *testing.T) {
		t.Parallel()

		// Chromium column rtl container: lead x50 y0, mid inner x0 y150,
		// tail x100 y450. The cross-axis margins are resolved against the RTL
		// line edges, while the logical padding starts on the physical right.
		rtlCSS := columnCSS + `
.flexcase { flex-direction: column; direction: rtl }
.lead { margin: auto 100pt auto 50pt }
`
		res := layoutHTML(t, columnHTML, sheet(t, rtlCSS))
		flexcase := findBoxByClass(t, res, "flexcase")
		lead := findBoxByClass(t, res, "lead")
		mid := findBoxByClass(t, res, "mid")
		tail := findBoxByClass(t, res, "tail")
		inner := findBoxByClass(t, res, "inner")

		if !near(lead.x, flexcase.x+50) || !near(lead.y, flexcase.y) {
			t.Fatalf("rtl column lead = (%.2f, %.2f), want (%.2f, %.2f)",
				lead.x, lead.y, flexcase.x+50, flexcase.y)
		}
		if !near(mid.y, flexcase.y+150) || !near(inner.x, flexcase.x) || !near(inner.y, flexcase.y+150) {
			t.Fatalf("rtl column mid y%.2f inner (%.2f, %.2f), want y%.2f inner (%.2f, %.2f)",
				mid.y, inner.x, inner.y, flexcase.y+150, flexcase.x, flexcase.y+150)
		}
		if !near(tail.x, flexcase.x+100) || !near(tail.y, flexcase.y+450) {
			t.Fatalf("rtl column tail = (%.2f, %.2f), want (%.2f, %.2f)",
				tail.x, tail.y, flexcase.x+100, flexcase.y+450)
		}
	})

	t.Run("row-rtl-and-reverse-matrix", func(t *testing.T) {
		t.Parallel()

		// Chromium row containers 2 to 4: rtl lead x525, mid x175 inner x175,
		// tail x100; row-reverse lead x525, mid x175 inner x375, tail x0; rtl
		// row-reverse lead x0, mid x75 inner x75, tail x525. Blocked: rtl is
		// ignored and reverse flow packs at main-start
		// (internal/layout/flex.go:84, internal/layout/flex.go:306).
		rows := []struct {
			name   string
			extra  string
			lx, mx float64
			ix, tx float64
		}{
			{name: "rtl", extra: " direction: rtl", lx: 525, mx: 175, ix: 175, tx: 100},
			{name: "row-reverse", extra: " flex-direction: row-reverse", lx: 525, mx: 175, ix: 375, tx: 0},
			{name: "rtl-row-reverse", extra: " direction: rtl; flex-direction: row-reverse", lx: 0, mx: 75, ix: 75, tx: 525},
		}
		for _, want := range rows {
			res := layoutHTML(t, columnHTML, sheet(t, "body { margin: 0 }\n"+
				".flexcase { display: flex; width: 600pt;"+want.extra+" }\n"+
				".lead { flex: 1 0 0; margin: 0 auto }\n"+
				".mid { flex: 2 0 0; padding-inline-start: 200pt }\n"+
				".tail { flex: 1 0 0; margin-inline-end: 100pt }\n"))
			flexcase := findBoxByClass(t, res, "flexcase")
			lead := findBoxByClass(t, res, "lead")
			mid := findBoxByClass(t, res, "mid")
			tail := findBoxByClass(t, res, "tail")
			inner := findBoxByClass(t, res, "inner")

			if !near(lead.x, flexcase.x+want.lx) || !near(mid.x, flexcase.x+want.mx) ||
				!near(inner.x, flexcase.x+want.ix) || !near(tail.x, flexcase.x+want.tx) {
				t.Fatalf("%s x = lead%.2f mid%.2f inner%.2f tail%.2f, want %.2f/%.2f/%.2f/%.2f",
					want.name, lead.x, mid.x, inner.x, tail.x,
					flexcase.x+want.lx, flexcase.x+want.mx, flexcase.x+want.ix, flexcase.x+want.tx)
			}
		}
	})

	t.Run("vertical-writing-mode-columns", func(t *testing.T) {
		t.Parallel()

		// Chromium vertical-lr column container: first item x0 w500, second
		// item x500 y100 w100; vertical-rl mirrors the cross axis.
		// Blocked: buildFlex never reads writing-mode
		// (internal/layout/flex.go:84).
		res := layoutHTML(t, `<html><body>
<div class="flexcase">
  <div class="wide"></div>
  <div class="narrow"></div>
</div>
</body></html>`, sheet(t, `
body { margin: 0 }
.flexcase { display: flex; writing-mode: vertical-lr; flex-direction: column; width: 600pt; height: 600pt }
.wide { flex: 1 0 0; min-width: 300pt }
.narrow { flex: 1 0 200pt; max-width: 100pt; margin: 100pt 0 50pt 0 }
`))
		flexcase := findBoxByClass(t, res, "flexcase")
		wide := findBoxByClass(t, res, "wide")
		narrow := findBoxByClass(t, res, "narrow")

		if !near(wide.w, 500) || !near(wide.x, flexcase.x) {
			t.Fatalf("vertical-lr wide = w%.2f x%.2f, want w500 x%.2f", wide.w, wide.x, flexcase.x)
		}
		if !near(narrow.w, 100) || !near(narrow.x, flexcase.x+500) || !near(narrow.y, flexcase.y+100) {
			t.Fatalf("vertical-lr narrow = w%.2f (%.2f, %.2f), want w100 (%.2f, %.2f)",
				narrow.w, narrow.x, narrow.y, flexcase.x+500, flexcase.y+100)
		}
	})
}

// TestChromeFlexFlowAutoMarginsReverse ports test/chrome case
// legacy-flex-flow-auto-margins.
//
// Source: chromium/third_party/blink/web_tests/css3/flexbox/flex-flow-auto-margins.html
// Expected: logical auto margins consume free space through direction, writing
// mode, and column-reverse. The engine resolves explicit auto margins on the
// four physical sides and maps logical margins through direction and writing
// mode, so the source's 13/17/2pt margin boxes are asserted strictly.
// TestChromeFlexCase13AutoMarginsFixture pins the same fixture's wrapper
// geometry and the painted markers.
//
//nolint:cyclop,funlen,maintidx // Chromium fixture geometry keeps short labels
func TestChromeFlexFlowAutoMarginsReverse(t *testing.T) {
	t.Parallel()

	wrapHTML := `<html><body><div class="wrap"><div class="flexcase"><div class="item"></div></div></div></body></html>`

	t.Run("row-inline-auto", func(t *testing.T) {
		t.Parallel()

		// Physical row container, margin: 0 auto. Free main space is
		// 100 - 20 = 80pt, split evenly by the two auto inline margins.
		cssSheet := sheet(t, `
body { margin: 0 }
.flexcase { display: flex; width: 100pt; height: 100pt }
.item { width: 20pt; height: 20pt; margin: 0 auto }
`)
		res := layoutHTML(t, `<html><body><div class="flexcase"><div class="item"></div></div></body></html>`, cssSheet)
		flexcase := findBoxByClass(t, res, "flexcase")
		item := findBoxByClass(t, res, "item")

		if !near(item.x, flexcase.x+40) || !near(item.y, flexcase.y) {
			t.Fatalf("inline auto item = (%.2f, %.2f), want (%.2f, %.2f)",
				item.x, item.y, flexcase.x+40, flexcase.y)
		}
	})

	t.Run("row-block-auto", func(t *testing.T) {
		t.Parallel()

		// margin: auto 0. The block-start and block-end auto margins split the
		// 80pt cross free space, centering the item on the block axis.
		cssSheet := sheet(t, `
body { margin: 0 }
.flexcase { display: flex; width: 100pt; height: 100pt }
.item { width: 20pt; height: 20pt; margin: auto 0 }
`)
		res := layoutHTML(t, `<html><body><div class="flexcase"><div class="item"></div></div></body></html>`, cssSheet)
		flexcase := findBoxByClass(t, res, "flexcase")
		item := findBoxByClass(t, res, "item")

		if !near(item.x, flexcase.x) || !near(item.y, flexcase.y+40) {
			t.Fatalf("block auto item = (%.2f, %.2f), want (%.2f, %.2f)",
				item.x, item.y, flexcase.x, flexcase.y+40)
		}
	})

	t.Run("row-block-start-auto", func(t *testing.T) {
		t.Parallel()

		// margin: auto 0 0 0. Only the block-start margin is auto, so it takes
		// all 80pt of cross free space and pins the item to the block-end edge.
		cssSheet := sheet(t, `
body { margin: 0 }
.flexcase { display: flex; width: 100pt; height: 100pt }
.item { width: 20pt; height: 20pt; margin: auto 0 0 0 }
`)
		res := layoutHTML(t, `<html><body><div class="flexcase"><div class="item"></div></div></body></html>`, cssSheet)
		flexcase := findBoxByClass(t, res, "flexcase")
		item := findBoxByClass(t, res, "item")

		if !near(item.x, flexcase.x) || !near(item.y, flexcase.y+80) {
			t.Fatalf("block-start auto item = (%.2f, %.2f), want (%.2f, %.2f)",
				item.x, item.y, flexcase.x, flexcase.y+80)
		}
	})

	t.Run("row-all-auto-centers", func(t *testing.T) {
		t.Parallel()

		// margin: auto on every side. Main free space 80pt splits 40/40 and
		// cross free space also centers the item.
		cssSheet := sheet(t, `
body { margin: 0 }
.flexcase { display: flex; width: 100pt; height: 100pt }
.item { width: 20pt; height: 20pt; margin: auto }
`)
		res := layoutHTML(t, `<html><body><div class="flexcase"><div class="item"></div></div></body></html>`, cssSheet)
		flexcase := findBoxByClass(t, res, "flexcase")
		item := findBoxByClass(t, res, "item")

		if !near(item.x, flexcase.x+40) || !near(item.y, flexcase.y+40) {
			t.Fatalf("all-auto item = (%.2f, %.2f), want (%.2f, %.2f)",
				item.x, item.y, flexcase.x+40, flexcase.y+40)
		}
	})

	t.Run("column-reverse-main-auto", func(t *testing.T) {
		t.Parallel()

		// column-reverse with margin: auto 0. The vertical auto margins split
		// the 80pt main free space, so the item stays centered in the reversed
		// flow.
		cssSheet := sheet(t, `
body { margin: 0 }
.flexcase { display: flex; flex-direction: column-reverse; width: 100pt; height: 100pt }
.item { width: 20pt; height: 20pt; margin: auto 0 }
`)
		res := layoutHTML(t, `<html><body><div class="flexcase"><div class="item"></div></div></body></html>`, cssSheet)
		flexcase := findBoxByClass(t, res, "flexcase")
		item := findBoxByClass(t, res, "item")

		if !near(item.x, flexcase.x) || !near(item.y, flexcase.y+40) {
			t.Fatalf("column-reverse auto item = (%.2f, %.2f), want (%.2f, %.2f)",
				item.x, item.y, flexcase.x, flexcase.y+40)
		}
	})

	t.Run("horizontal-tb-ltr-physical-row", func(t *testing.T) {
		t.Parallel()

		// Chromium physical horizontal-tb/ltr/row: item at (80, 23) relative to
		// the inline-block wrapper, from margin: 13pt auto 17pt auto inside a
		// 100x100 flex box offset by (40, 10).
		cssSheet := sheet(t, `
body { margin: 0 }
.wrap { position: relative; display: inline-block }
.flexcase { display: flex; flex-direction: row; margin: 10pt 20pt 30pt 40pt; width: 100pt; height: 100pt }
.item { width: 20pt; height: 20pt; margin: 13pt auto 17pt auto }
`)
		res := layoutHTML(t, wrapHTML, cssSheet)
		wrap := findBoxByClass(t, res, "wrap")
		item := findBoxByClass(t, res, "item")

		if !near(item.x, wrap.x+80) || !near(item.y, wrap.y+23) {
			t.Fatalf("physical row item = (%.2f, %.2f), want (%.2f, %.2f)",
				item.x, item.y, wrap.x+80, wrap.y+23)
		}
	})

	t.Run("horizontal-tb-ltr-logical-row", func(t *testing.T) {
		t.Parallel()

		// Chromium logical horizontal-tb/ltr/row: item at (118, 73). The
		// inline-end 2pt and block-end 17pt fixed margins shrink the free space
		// that the two auto margins consume.
		cssSheet := sheet(t, `
body { margin: 0 }
.wrap { position: relative; display: inline-block }
.flexcase {
  display: flex; flex-direction: row;
  margin-block-start: 10pt; margin-block-end: 30pt;
  margin-inline-start: 40pt; margin-inline-end: 20pt;
  width: 100pt; height: 100pt;
}
.item {
  width: 20pt; height: 20pt;
  margin-block-start: auto; margin-block-end: 17pt;
  margin-inline-start: auto; margin-inline-end: 2pt;
}
`)
		res := layoutHTML(t, wrapHTML, cssSheet)
		wrap := findBoxByClass(t, res, "wrap")
		item := findBoxByClass(t, res, "item")

		if !near(item.x, wrap.x+118) || !near(item.y, wrap.y+73) {
			t.Fatalf("logical ltr row item = (%.2f, %.2f), want (%.2f, %.2f)",
				item.x, item.y, wrap.x+118, wrap.y+73)
		}
	})

	t.Run("horizontal-tb-rtl-logical-row", func(t *testing.T) {
		t.Parallel()

		// Chromium logical horizontal-tb/rtl/row: item at (22, 73). The
		// inline-start auto margin resolves on the right under rtl.
		cssSheet := sheet(t, `
body { margin: 0 }
.wrap { position: relative; display: inline-block }
.flexcase {
  display: flex; flex-direction: row; direction: rtl;
  margin-block-start: 10pt; margin-block-end: 30pt;
  margin-inline-start: 40pt; margin-inline-end: 20pt;
  width: 100pt; height: 100pt;
}
.item {
  width: 20pt; height: 20pt;
  margin-block-start: auto; margin-block-end: 17pt;
  margin-inline-start: auto; margin-inline-end: 2pt;
}
`)
		res := layoutHTML(t, wrapHTML, cssSheet)
		wrap := findBoxByClass(t, res, "wrap")
		item := findBoxByClass(t, res, "item")

		if !near(item.x, wrap.x+22) || !near(item.y, wrap.y+73) {
			t.Fatalf("logical rtl row item = (%.2f, %.2f), want (%.2f, %.2f)",
				item.x, item.y, wrap.x+22, wrap.y+73)
		}
	})

	t.Run("horizontal-tb-ltr-physical-column-reverse", func(t *testing.T) {
		t.Parallel()

		// Chromium physical horizontal-tb/ltr/column-reverse: item at
		// (80, 73) from the same 13pt auto 17pt auto margins.
		cssSheet := sheet(t, `
body { margin: 0 }
.wrap { position: relative; display: inline-block }
.flexcase {
  display: flex; flex-direction: column-reverse;
  margin: 10pt 20pt 30pt 40pt; width: 100pt; height: 100pt;
}
.item { width: 20pt; height: 20pt; margin: 13pt auto 17pt auto }
`)
		res := layoutHTML(t, wrapHTML, cssSheet)
		wrap := findBoxByClass(t, res, "wrap")
		item := findBoxByClass(t, res, "item")

		if !near(item.x, wrap.x+80) || !near(item.y, wrap.y+73) {
			t.Fatalf("physical column-reverse item = (%.2f, %.2f), want (%.2f, %.2f)",
				item.x, item.y, wrap.x+80, wrap.y+73)
		}
	})

	t.Run("vertical-lr-ltr-logical-row", func(t *testing.T) {
		t.Parallel()

		// Chromium logical vertical-lr/ltr/row: item at (73, 118). Logical
		// margins attach to the inline (vertical) and block (horizontal) axes.

		cssSheet := sheet(t, `
body { margin: 0 }
.wrap { position: relative; display: inline-block }
.flexcase {
  display: flex; flex-direction: row; writing-mode: vertical-lr;
  margin-block-start: 10pt; margin-block-end: 30pt;
  margin-inline-start: 40pt; margin-inline-end: 20pt;
  width: 100pt; height: 100pt;
}
.item {
  width: 20pt; height: 20pt;
  margin-block-start: auto; margin-block-end: 17pt;
  margin-inline-start: auto; margin-inline-end: 2pt;
}
`)
		res := layoutHTML(t, wrapHTML, cssSheet)
		wrap := findBoxByClass(t, res, "wrap")
		item := findBoxByClass(t, res, "item")

		if !near(item.x, wrap.x+73) || !near(item.y, wrap.y+118) {
			t.Fatalf("logical vertical-lr item = (%.2f, %.2f), want (%.2f, %.2f)",
				item.x, item.y, wrap.x+73, wrap.y+118)
		}
	})
}

// TestChromeFlexBaselineAlignment ports test/chrome case
// legacy-flex-align-baseline.
//
// Source: chromium/third_party/blink/web_tests/css3/flexbox/flex-align-baseline.html
// Expected: items with different margins share the expected baseline across
// flows and writing modes. The source sanity check asserts the physical edge
// for each direction, including the RTL start edge in a column.
//
//nolint:cyclop,funlen,wsl,gocognit // Chromium fixture geometry keeps short labels
func TestChromeFlexBaselineAlignment(t *testing.T) {
	t.Parallel()

	t.Run("horizontal-tb-flows", func(t *testing.T) {
		t.Parallel()

		flows := []struct {
			name  string
			isCol bool
		}{
			{name: "row"},
			{name: "row-reverse"},
			{name: "column", isCol: true},
			{name: "column-reverse", isCol: true},
		}
		dirs := []string{"ltr", "rtl"}
		for _, flow := range flows {
			for _, dir := range dirs {
				t.Run(flow.name+"-"+dir, func(t *testing.T) {
					t.Parallel()

					cssSheet := sheet(t, `
body { margin: 0 }
.flexcase { display: flex; margin: 120pt; width: 100pt; height: 100pt; align-items: baseline }
.flexcase > div { height: 110pt; width: 110pt }
.row { flex-direction: row }
.row-reverse { flex-direction: row-reverse }
.column { flex-direction: column }
.column-reverse { flex-direction: column-reverse }
.ltr { direction: ltr }
.rtl { direction: rtl }
`)
					res := layoutHTML(t, `<html><body>
<div class="flexcase `+flow.name+` `+dir+`">
  <div class="first"><div style="display: inline-block"></div></div>
  <div class="second" style="margin-top: 20pt"><div style="display: inline-block"></div></div>
</div>
</body></html>`, cssSheet)

					flexcase := findBoxByClass(t, res, "flexcase")
					first := findBoxByClass(t, res, "first")
					second := findBoxByClass(t, res, "second")
					// The 110pt items plus the second item's 20pt main-axis
					// margin shrink into the 100pt container: 40 in the flow
					// direction, 110 across it.
					if flow.isCol {
						if !near(first.w, 110) || !near(first.height, 40) {
							t.Fatalf("%s item size = %.1fx%.1f, want 110x40", flow.name, first.w, first.height)
						}
					} else if !near(first.w, 50) || !near(first.height, 110) {
						t.Fatalf("%s item size = %.1fx%.1f, want 50x110", flow.name, first.w, first.height)
					}
					if flow.isCol {
						wantX := flexcase.x
						if dir == "rtl" {
							wantX -= 10
						}
						if !near(first.x, wantX) || !near(second.x, wantX) {
							t.Fatalf("%s baseline x = %.2f/%.2f, want %.2f",
								flow.name, first.x, second.x, wantX)
						}
					} else if !near(first.y, flexcase.y+20) || !near(second.y, flexcase.y+20) {
						t.Fatalf("%s baseline y = %.2f/%.2f, want %.2f",
							flow.name, first.y, second.y, flexcase.y+20)
					}
				})
			}
		}
	})

	t.Run("vertical-writing-mode-flows", func(t *testing.T) {
		t.Parallel()

		// Source sanity check for vertical-lr and vertical-rl: the writing mode
		// rotates which physical edge proves the shared baseline. For vertical
		// modes with a row the cross axis is horizontal, so offsetLeft must
		// match; with a column the cross axis is vertical, so offsetTop must
		// match.
		// Blocked: buildFlex ignores writing-mode
		// (internal/layout/flex.go:84).
		modes := []string{"vertical-lr", "vertical-rl"}
		flows := []struct {
			name  string
			isCol bool
		}{
			{name: "row"},
			{name: "row-reverse"},
			{name: "column", isCol: true},
			{name: "column-reverse", isCol: true},
		}
		dirs := []string{"ltr", "rtl"}
		for _, mode := range modes {
			for _, flow := range flows {
				for _, dir := range dirs {
					cssSheet := sheet(t, `
body { margin: 0 }
.flexcase { display: flex; margin: 20pt; width: 100pt; height: 100pt; align-items: baseline; writing-mode: `+mode+` }
.flexcase > div { height: 110pt; width: 110pt }
.row { flex-direction: row }
.row-reverse { flex-direction: row-reverse }
.column { flex-direction: column }
.column-reverse { flex-direction: column-reverse }
.ltr { direction: ltr }
.rtl { direction: rtl }
`)
					res := layoutHTML(t, `<html><body>
<div class="flexcase `+flow.name+` `+dir+`">
  <div class="first"><div style="display: inline-block"></div></div>
  <div class="second" style="margin-top: 20pt"><div style="display: inline-block"></div></div>
</div>
</body></html>`, cssSheet)

					first := findBoxByClass(t, res, "first")
					second := findBoxByClass(t, res, "second")
					// isHorizontalFlow: for vertical writing modes the column
					// flow puts the baseline on the vertical axis.
					isHorizontalFlow := flow.isCol
					if isHorizontalFlow {
						if !near(first.y, second.y) {
							t.Fatalf("%s %s baseline y = %.2f/%.2f, want equal",
								mode, flow.name, first.y, second.y)
						}
					} else if !near(first.x, second.x) {
						t.Fatalf("%s %s baseline x = %.2f/%.2f, want equal",
							mode, flow.name, first.x, second.x)
					}
				}
			}
		}
	})
}

// TestChromeFlexRTLColumnWrapReverse ports test/chrome case
// wpt-rtl-flow-reverse.
//
// Source: chromium/third_party/blink/web_tests/external/wpt/css/css-flexbox/flexbox_rtl-flow-reverse.html
// Reference: flexbox_rtl-flow-reverse-ref.html
// Expected: RTL plus column wrap-reverse reverses logical column order without
// losing item order within a column. The reference grid keeps "one" and "two"
// in the first physical column and "three" and "four" in the second. The
// engine has no column wrapping (flowFlexColumn ignores FlexWrap) and ignores
// direction: rtl, so the strict assertion below is skipped.
//
//nolint:cyclop,wsl // Chromium fixture geometry keeps short labels
func TestChromeFlexRTLColumnWrapReverse(t *testing.T) {
	t.Parallel()

	t.Run("rtl-column-wrap-reverse", func(t *testing.T) {
		t.Parallel()

		// Source spans are 8em wide and 1em margins inside a 20em x 8em box.
		// Pinning font-size 10pt and line-height 12pt gives 80pt spans with
		// 10pt margins and 12pt line boxes; three 32pt outer heights overflow
		// the 80pt content height, so the items wrap into two columns of two.
		// Expected offsets relative to the border box: one (11, 11),
		// two (11, 43), three (111, 11), four (111, 43).
		// Blocked: flowFlexColumn has no wrap handling
		// (internal/layout/flex.go:1528) and direction: rtl is ignored
		// (internal/layout/flex.go:84).
		cssSheet := sheet(t, `
body { margin: 0 }
.flexcase {
  display: flex;
  flex-flow: column wrap-reverse;
  direction: rtl;
  width: 20em;
  height: 8em;
  margin: 1em 0;
  border: 1pt solid #888;
  font-size: 10pt;
  line-height: 12pt;
}
.flexcase span { width: 8em; margin: 1em }
`)
		res := layoutHTML(t, `<html><body>
<div class="flexcase">
  <span class="span-one">one</span>
  <span class="span-two">two</span>
  <span class="span-three">three</span>
  <span class="span-four">four</span>
</div>
</body></html>`, cssSheet)

		flexcase := findBoxByClass(t, res, "flexcase")
		one := findBoxByClass(t, res, "span-one")
		two := findBoxByClass(t, res, "span-two")
		three := findBoxByClass(t, res, "span-three")
		four := findBoxByClass(t, res, "span-four")

		if !near(flexcase.w, 202) || !near(flexcase.height, 82) {
			t.Fatalf("wrap-reverse container = %.2fx%.2f, want 202x82", flexcase.w, flexcase.height)
		}
		if !near(one.x, flexcase.x+11) || !near(one.y, flexcase.y+11) {
			t.Fatalf("one at (%.2f, %.2f), want (%.2f, %.2f)", one.x, one.y, flexcase.x+11, flexcase.y+11)
		}
		if !near(two.x, flexcase.x+11) || !near(two.y, flexcase.y+43) {
			t.Fatalf("two at (%.2f, %.2f), want (%.2f, %.2f)", two.x, two.y, flexcase.x+11, flexcase.y+43)
		}
		if !near(three.x, flexcase.x+111) || !near(three.y, flexcase.y+11) {
			t.Fatalf("three at (%.2f, %.2f), want (%.2f, %.2f)",
				three.x, three.y, flexcase.x+111, flexcase.y+11)
		}
		if !near(four.x, flexcase.x+111) || !near(four.y, flexcase.y+43) {
			t.Fatalf("four at (%.2f, %.2f), want (%.2f, %.2f)",
				four.x, four.y, flexcase.x+111, flexcase.y+43)
		}
	})
}

// TestChromeFlexWritingModeMatrix ports test/chrome case wpt-writing-mode-006.
//
// Source: chromium/third_party/blink/web_tests/external/wpt/css/css-flexbox/flexbox-writing-mode-006.html
// Reference: flexbox-writing-mode-006-ref.html
// Expected: row, column, reverse, and wrap-reverse combinations preserve
// logical block ordering in vertical writing modes. The reference lays the
// items out as a 2x2 physical grid: row wrap gives item2/item4 on top and
// item1/item3 below; each of the eight combinations maps to one grid. The
// horizontal-tb subtests below run the same wrap and wrap-reverse line
// ordering on the supported axes; vertical-lr with direction: rtl is skipped
// because buildFlex ignores writing-mode and direction.
//
//nolint:funlen,wsl // Chromium fixture geometry keeps short labels
func TestChromeFlexWritingModeMatrix(t *testing.T) {
	t.Parallel()

	matrixCSS := `
body { margin: 0 }
.flexcase { display: flex; width: 40pt; height: 30pt; border: 1pt solid #888 }
.item { width: 20pt; height: 15pt }
`
	matrixHTML := `<html><body>
<div class="flexcase">
  <div class="item item-one"></div><div class="item item-two"></div>
  <div class="item item-three"></div><div class="item item-four"></div>
</div>
</body></html>`

	t.Run("horizontal-tb-ltr-row-wrap", func(t *testing.T) {
		t.Parallel()

		res := layoutHTML(t, matrixHTML, sheet(t, matrixCSS+".flexcase { flex-flow: row wrap }"))
		assertWrapGrid(t, res, "wrap", [4]float64{1, 1, 16, 16})
	})

	t.Run("horizontal-tb-ltr-row-wrap-reverse", func(t *testing.T) {
		t.Parallel()

		res := layoutHTML(t, matrixHTML, sheet(t, matrixCSS+".flexcase { flex-flow: row wrap-reverse }"))
		assertWrapGrid(t, res, "wrap-reverse", [4]float64{16, 16, 1, 1})
	})

	t.Run("vertical-lr-rtl-matrix", func(t *testing.T) {
		t.Parallel()

		// Expected physical grid from flexbox-writing-mode-006-ref.html, with
		// offsets relative to the container content box: row wrap item1 (0,15)
		// item2 (0,0) item3 (20,15) item4 (20,0); row wrap-reverse item1
		// (20,15) item2 (20,0) item3 (0,15) item4 (0,0); row-reverse wrap item1
		// (0,0) item2 (0,15) item3 (20,0) item4 (20,15); row-reverse wrap-reverse
		// item1 (20,0) item2 (20,15) item3 (0,0) item4 (0,15); column wrap item1
		// (0,15) item2 (20,15) item3 (0,0) item4 (20,0); column wrap-reverse
		// item1 (0,0) item2 (20,0) item3 (0,15) item4 (20,15); column-reverse
		// wrap item1 (20,15) item2 (0,15) item3 (20,0) item4 (0,0); and
		// column-reverse wrap-reverse item1 (20,0) item2 (0,0) item3 (20,15)
		// item4 (0,15).
		grid := []struct {
			flow   string
			x1, y1 float64
			x2, y2 float64
			x3, y3 float64
			x4, y4 float64
		}{
			{flow: "row wrap", x1: 0, y1: 15, x2: 0, y2: 0, x3: 20, y3: 15, x4: 20, y4: 0},
			{flow: "row wrap-reverse", x1: 20, y1: 15, x2: 20, y2: 0, x3: 0, y3: 15, x4: 0, y4: 0},
			{flow: "row-reverse wrap", x1: 0, y1: 0, x2: 0, y2: 15, x3: 20, y3: 0, x4: 20, y4: 15},
			{flow: "row-reverse wrap-reverse", x1: 20, y1: 0, x2: 20, y2: 15, x3: 0, y3: 0, x4: 0, y4: 15},
			{flow: "column wrap", x1: 0, y1: 15, x2: 20, y2: 15, x3: 0, y3: 0, x4: 20, y4: 0},
			{flow: "column wrap-reverse", x1: 0, y1: 0, x2: 20, y2: 0, x3: 0, y3: 15, x4: 20, y4: 15},
			{flow: "column-reverse wrap", x1: 20, y1: 15, x2: 0, y2: 15, x3: 20, y3: 0, x4: 0, y4: 0},
			{flow: "column-reverse wrap-reverse", x1: 20, y1: 0, x2: 0, y2: 0, x3: 20, y3: 15, x4: 0, y4: 15},
		}
		for _, want := range grid {
			res := layoutHTML(t, matrixHTML,
				sheet(t, matrixCSS+".flexcase { writing-mode: vertical-lr; direction: rtl; flex-flow: "+want.flow+" }"))
			flexcase := findBoxByClass(t, res, "flexcase")
			one := findBoxByClass(t, res, "item-one")
			two := findBoxByClass(t, res, "item-two")
			three := findBoxByClass(t, res, "item-three")
			four := findBoxByClass(t, res, "item-four")

			got := [8]float64{one.x, one.y, two.x, two.y, three.x, three.y, four.x, four.y}
			expected := [8]float64{
				flexcase.x + 1 + want.x1, flexcase.y + 1 + want.y1,
				flexcase.x + 1 + want.x2, flexcase.y + 1 + want.y2,
				flexcase.x + 1 + want.x3, flexcase.y + 1 + want.y3,
				flexcase.x + 1 + want.x4, flexcase.y + 1 + want.y4,
			}
			for idx := range got {
				if !near(got[idx], expected[idx]) {
					t.Fatalf("vertical-lr rtl %s item %d coordinate %.2f, want %.2f",
						want.flow, idx/2+1, got[idx], expected[idx])
				}
			}
		}
	})
}

// assertWrapGrid checks the 2x2 item grid of the writing-mode matrix for one
// flex-flow value. wantY holds each item's y offset relative to the container
// content box; the column x offsets alternate 1/21.
func assertWrapGrid(t *testing.T, res *Result, label string, wantY [4]float64) {
	t.Helper()

	flexcase := findBoxByClass(t, res, "flexcase")
	wantX := [4]float64{1, 21, 1, 21}
	names := [4]string{"one", "two", "three", "four"}
	items := [4]*box{
		findBoxByClass(t, res, "item-one"),
		findBoxByClass(t, res, "item-two"),
		findBoxByClass(t, res, "item-three"),
		findBoxByClass(t, res, "item-four"),
	}

	for idx, item := range items {
		if !near(item.x, flexcase.x+wantX[idx]) || !near(item.y, flexcase.y+wantY[idx]) {
			t.Fatalf("%s %s at (%.2f, %.2f), want (%.2f, %.2f)",
				label, names[idx], item.x, item.y,
				flexcase.x+wantX[idx], flexcase.y+wantY[idx])
		}
	}
}
