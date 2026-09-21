package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// TestChromeFlexAutoMarginFillKeepsDefiniteHeight covers the auto-margin
// pagination defect. A page-tall leading block pushes the flex case to a later page and
// leaves its trailing container border as orphan chrome, so the page-level
// strip pass runs and reports a removal. The tighten pass that follows then
// shortened the auto-margin item's 30pt background fill to the last baseline
// plus 8pt of padding (27.573pt) while the item box stayed 30pt tall, so the
// painted fill no longer matched its box.
func TestChromeFlexAutoMarginFillKeepsDefiniteHeight(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
html, body { margin: 0; padding: 0; }
.lead { height: 700pt; background: #eeeeee; }
.case { display: flex; width: 300pt; height: 80pt; border: 1pt solid #222; }
.item { width: 40pt; height: 30pt; font-size: 8pt; line-height: 1.4; }
.auto { margin: auto; background: #9ec5fe; }
`)

	res := layoutHTML(t, `<html><body><div class="lead"></div>`+
		`<div class="case"><div class="item auto">A</div><div class="item">B</div></div>`+
		`</body></html>`, cssSheet)

	items := classBoxes(res.root, "item")
	if len(items) != 2 {
		t.Fatalf("item boxes = %d, want 2", len(items))
	}

	auto := items[0]

	if !near(auto.height, 30) {
		t.Fatalf("auto item height = %.3f, want 30", auto.height)
	}

	if err := Paint(pdf.NewDocument(), res, PaintOptions{PageWidth: 500, PageHeight: 400}); err != nil {
		t.Fatalf("paint: %v", err)
	}

	fill := findFillOpForBox(res, auto)

	if fill == nil {
		t.Fatalf("no fill for auto item box (%.2f, %.2f, %.2f, %.2f)",
			auto.x, auto.y, auto.w, auto.height)
	}

	if !near(fill.Y, auto.y) {
		t.Errorf("auto fill Y = %.3f, want box y %.3f", fill.Y, auto.y)
	}

	if !near(fill.H, auto.height) {
		t.Errorf("auto fill H = %.3f, want box height %.3f (seal tightened a definite flex item fill)",
			fill.H, auto.height)
	}
}

// findFillOpForBox returns the first fill operation drawn over item's box, or
// nil when the result has no matching fill.
func findFillOpForBox(res *Result, item *box) *Op {
	for i := range res.Ops {
		paintOp := &res.Ops[i]
		if paintOp.Kind == OpFillRect && near(paintOp.X, item.x) && near(paintOp.W, item.w) &&
			paintOp.Y >= item.y-1 && paintOp.Y <= item.y+1 {
			return paintOp
		}
	}

	return nil
}

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

// TestChromeFlexMultilineAlignSelf covers case legacy-multiline-align-self
// from third_party/blink/web_tests/css3/flexbox/multiline-align-self.html.
//
// Expected behavior: each wrapped line resolves its own cross size before
// align-self places its items. The first line below is 30pt tall and the
// second line is 20pt tall, so flex-start sits at the line top, center sits at
// line top + (line cross - item height)/2, flex-end sits at line top + line
// cross - item height, and stretch fills the line. Chromium's baseline group
// of empty items uses the largest margin-top + height as the line baseline,
// which puts both baseline items 5pt below the first line top and 6pt/0pt
// below the second line top.
//
//nolint:funlen // adapted fixture plus a baseline probe
func TestChromeFlexMultilineAlignSelf(t *testing.T) {
	t.Parallel()

	t.Run("per-line-cross-size", func(t *testing.T) {
		t.Parallel()

		cssSheet := sheet(t, `
html, body { margin: 0; padding: 0; }
.row { display: flex; flex-flow: row wrap; align-content: flex-start; width: 160pt; }
.item { width: 40pt; }
.l1-start { height: 20pt; align-self: flex-start; }
.l1-center { height: 10pt; align-self: center; }
.l1-end { height: 30pt; align-self: flex-end; }
.l1-stretch { height: 10pt; align-self: stretch; }
.l2-start { height: 20pt; align-self: flex-start; }
.l2-center { height: 10pt; align-self: center; }
.l2-end { height: 10pt; align-self: flex-end; }
.l2-stretch { align-self: stretch; }
`)
		res := layoutHTML(t, `<html><body>
<div class="row">
  <div class="item l1-start"></div>
  <div class="item l1-center"></div>
  <div class="item l1-end"></div>
  <div class="item l1-stretch"></div>
  <div class="item l2-start"></div>
  <div class="item l2-center"></div>
  <div class="item l2-end"></div>
  <div class="item l2-stretch"></div>
</div>
</body></html>`, cssSheet)

		row := findBoxByClass(t, res, "row")

		type itemWant struct {
			name       string
			box        *box
			x, y, w, h float64
		}

		wants := []itemWant{
			{name: "l1-start", box: findBoxByClass(t, res, "l1-start"), x: row.x, y: row.y, w: 40, h: 20},
			{name: "l1-center", box: findBoxByClass(t, res, "l1-center"), x: row.x + 40, y: row.y + 10, w: 40, h: 10},
			{name: "l1-end", box: findBoxByClass(t, res, "l1-end"), x: row.x + 80, y: row.y, w: 40, h: 30},
			{name: "l1-stretch", box: findBoxByClass(t, res, "l1-stretch"), x: row.x + 120, y: row.y, w: 40, h: 10},
			{name: "l2-start", box: findBoxByClass(t, res, "l2-start"), x: row.x, y: row.y + 30, w: 40, h: 20},
			{name: "l2-center", box: findBoxByClass(t, res, "l2-center"), x: row.x + 40, y: row.y + 35, w: 40, h: 10},
			{name: "l2-end", box: findBoxByClass(t, res, "l2-end"), x: row.x + 80, y: row.y + 40, w: 40, h: 10},
			// The auto-height stretch item proves line 2 uses its own 20pt
			// cross size: a flex-start copy would also sit at row.y + 30.
			{name: "l2-stretch", box: findBoxByClass(t, res, "l2-stretch"), x: row.x + 120, y: row.y + 30, w: 40, h: 20},
		}

		for _, want := range wants {
			if !near(want.box.x, want.x) || !near(want.box.y, want.y) ||
				!near(want.box.w, want.w) || !near(want.box.height, want.h) {
				t.Errorf(
					"%s box = (%.2f, %.2f, %.2f, %.2f), want (%.2f, %.2f, %.2f, %.2f)",
					want.name,
					want.box.x, want.box.y, want.box.w, want.box.height,
					want.x, want.y, want.w, want.h,
				)
			}
		}
	})

	t.Run("baseline", func(t *testing.T) {
		t.Parallel()

		// Chromium's expectations from the same source file: an empty baseline
		// item is measured at its bottom border edge, so the line baseline is
		// the largest margin-top + height in the baseline group.
		cssSheet := sheet(t, `
html, body { margin: 0; padding: 0; }
.row { display: flex; flex-flow: row wrap; align-content: flex-start; width: 120pt; }
.item { width: 40pt; }
.tall1 { height: 30pt; }
.base1 { height: 10pt; align-self: baseline; }
.base2 { height: 10pt; margin-top: 5pt; align-self: baseline; }
.tall2 { height: 20pt; }
.base3 { height: 10pt; align-self: baseline; }
.base4 { height: 16pt; align-self: baseline; }
`)
		res := layoutHTML(t, `<html><body>
<div class="row">
  <div class="item tall1">t1</div>
  <div class="item base1"></div>
  <div class="item base2"></div>
  <div class="item tall2">t2</div>
  <div class="item base3"></div>
  <div class="item base4"></div>
</div>
</body></html>`, cssSheet)

		row := findBoxByClass(t, res, "row")
		base1 := findBoxByClass(t, res, "base1")
		base2 := findBoxByClass(t, res, "base2")
		base3 := findBoxByClass(t, res, "base3")
		base4 := findBoxByClass(t, res, "base4")

		if !near(base1.y, row.y+5) || !near(base2.y, row.y+5) {
			t.Errorf("line 1 baseline y = %.2f/%.2f, want %.2f/%.2f",
				base1.y, base2.y, row.y+5, row.y+5)
		}

		if !near(base3.y, row.y+36) || !near(base4.y, row.y+30) {
			t.Errorf("line 2 baseline y = %.2f/%.2f, want %.2f/%.2f",
				base3.y, base4.y, row.y+36, row.y+30)
		}
	})
}

// TestChromeFlexAlignItemsStretchMargins covers case wpt-align-items-stretch
// from third_party/blink/web_tests/external/wpt/css/css-flexbox/
// flexbox_align-items-stretch-2.html.
//
// Expected behavior: align-items: stretch fills the line cross size only up to
// the item's cross-axis margins, so an item with margin-top equal to the
// container height gets a used height of 0 while its margin box still spans
// the line. An item with an explicit height keeps that height, and flex: none
// keeps every item at its 64pt main size.
//
//nolint:funlen,cyclop // three item probes share one fixture
func TestChromeFlexAlignItemsStretchMargins(t *testing.T) {
	t.Parallel()

	const (
		containerH = 96.0
		itemW      = 64.0
		itemH      = 24.0
	)

	cssSheet := sheet(t, `
html, body { margin: 0; padding: 0; }
.case { display: flex; align-items: stretch; width: 240pt; height: 96pt; }
.item { width: 64pt; flex: none; }
.pushed { margin-top: 96pt; }
.sized { height: 24pt; }
`)
	res := layoutHTML(t, `<html><body>
<div class="case">
  <div class="item pushed">PASS</div>
  <div class="item normal"></div>
  <div class="item sized">x</div>
</div>
</body></html>`, cssSheet)

	container := findBoxByClass(t, res, "case")

	t.Run("stretch-fills-line", func(t *testing.T) {
		t.Parallel()

		normal := findBoxByClass(t, res, "normal")
		if !near(container.w, 240) || !near(normal.w, itemW) || !near(normal.height, containerH) {
			t.Errorf("container w = %.2f, stretched item size = %.2fx%.2f, want 240 and %.2fx%.2f",
				container.w, normal.w, normal.height, itemW, containerH)
		}

		if !near(normal.x, container.x+itemW) || !near(normal.y, container.y) {
			t.Errorf("stretched item position = (%.2f, %.2f), want (%.2f, %.2f)",
				normal.x, normal.y, container.x+itemW, container.y)
		}
	})

	t.Run("explicit-height-kept", func(t *testing.T) {
		t.Parallel()

		sized := findBoxByClass(t, res, "sized")
		if !near(sized.height, itemH) || !near(sized.w, itemW) {
			t.Errorf("explicitly sized item = %.2fx%.2f, want %.2fx%.2f",
				sized.w, sized.height, itemW, itemH)
		}

		if !near(sized.x, container.x+2*itemW) || !near(sized.y, container.y) {
			t.Errorf("explicitly sized item position = (%.2f, %.2f), want (%.2f, %.2f)",
				sized.x, sized.y, container.x+2*itemW, container.y)
		}
	})

	t.Run("stretch-respects-margin", func(t *testing.T) {
		t.Parallel()

		// Chromium computes the stretched cross size as the line cross size
		// minus the item's cross-axis margins, so a 96pt margin inside the
		// 96pt tall container leaves a 0pt border box at container.y + 96.
		pushed := findBoxByClass(t, res, "pushed")
		if !near(pushed.height, 0) || !near(pushed.w, itemW) {
			t.Errorf("margin-pushed item size = %.2fx%.2f, want %.2fx0",
				pushed.w, pushed.height, itemW)
		}

		if !near(pushed.x, container.x) || !near(pushed.y, container.y+containerH) {
			t.Errorf("margin-pushed item position = (%.2f, %.2f), want (%.2f, %.2f)",
				pushed.x, pushed.y, container.x, container.y+containerH)
		}
	})
}

// TestChromeFlexColumnAutoMargins covers case wpt-auto-margins-column from
// third_party/blink/web_tests/external/wpt/css/css-flexbox/auto-margins-003.html.
//
// Expected behavior: in a 400pt wide column flex container, an item with
// margin: 0 auto and a definite width splits its free cross-axis space between
// the two auto margins, so its center is the container center. An item with
// align-self: center lands at the same center. Both keep their specified
// sizes, and the second item starts one item height below the first because
// the main axis is vertical. An auto-width item with auto margins keeps its
// shrink-to-fit width instead of stretching to the container cross size.
//
//nolint:cyclop,funlen // three cross-axis centering cases share one fixture
func TestChromeFlexColumnAutoMargins(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
html, body { margin: 0; padding: 0; }
.case { display: flex; flex-direction: column; width: 400pt; height: 200pt; }
.by-margins { width: 120pt; height: 30pt; margin: 0 auto; }
.by-align { width: 160pt; height: 30pt; align-self: center; }
.by-auto { margin: 0 auto; }
`)
	res := layoutHTML(t, `<html><body>
<div class="case">
  <div class="by-margins">centeredWithMargins</div>
  <div class="by-align">centeredWithAlignSelf</div>
  <div class="by-auto">narrow</div>
</div>
</body></html>`, cssSheet)

	container := findBoxByClass(t, res, "case")

	t.Run("align-self-center", func(t *testing.T) {
		t.Parallel()

		item := findBoxByClass(t, res, "by-align")
		wantX := container.x + (container.w-160)/2

		if !near(item.x, wantX) || !near(item.w, 160) {
			t.Errorf("align-self: center item x/w = %.2f/%.2f, want %.2f/160", item.x, item.w, wantX)
		}

		if !near(item.x+item.w/2, container.x+container.w/2) {
			t.Errorf("align-self: center item center x = %.2f, want %.2f",
				item.x+item.w/2, container.x+container.w/2)
		}

		if !near(item.y, container.y+30) || !near(item.height, 30) {
			t.Errorf("align-self: center item y/h = %.2f/%.2f, want %.2f/30",
				item.y, item.height, container.y+30)
		}
	})

	t.Run("auto-margins-center", func(t *testing.T) {
		t.Parallel()

		// Chromium centers the item through its two auto cross-axis margins:
		// the free space (400 - 120) splits evenly, so x = container.x + 140
		// and the item center equals the container center.
		item := findBoxByClass(t, res, "by-margins")
		wantX := container.x + (container.w-120)/2

		if !near(item.x, wantX) || !near(item.w, 120) {
			t.Errorf("auto-margin item x/w = %.2f/%.2f, want %.2f/120", item.x, item.w, wantX)
		}

		if !near(item.x+item.w/2, container.x+container.w/2) {
			t.Errorf("auto-margin item center x = %.2f, want %.2f",
				item.x+item.w/2, container.x+container.w/2)
		}

		if !near(item.y, container.y) || !near(item.height, 30) {
			t.Errorf("auto-margin item y/h = %.2f/%.2f, want %.2f/30",
				item.y, item.height, container.y)
		}

		// An auto-width auto-margin item must not stretch to the line cross
		// size (Flexbox L1 8.5): its used width stays shrink-to-fit and the
		// auto margins center that used width inside the container.
		autoWidth := findBoxByClass(t, res, "by-auto")
		if autoWidth.w <= 0 || autoWidth.w >= container.w {
			t.Errorf("auto-width auto-margin item w = %.2f, want shrink-to-fit inside (0, %.2f)",
				autoWidth.w, container.w)
		}

		if !near(autoWidth.x+autoWidth.w/2, container.x+container.w/2) {
			t.Errorf("auto-width auto-margin item center x = %.2f, want %.2f",
				autoWidth.x+autoWidth.w/2, container.x+container.w/2)
		}
	})
}
