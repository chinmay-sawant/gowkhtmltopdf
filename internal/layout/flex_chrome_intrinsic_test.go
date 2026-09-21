package layout

import "testing"

// TestChromeFlexAutoColumnIntrinsicSizing covers case legacy-columns-auto-size
// (goTarget layout-unit).
//
// Source: third_party/blink/web_tests/css3/flexbox/columns-auto-size.html
// Expected: automatic column sizes include content, margins, padding, and
// min/max constraints. The subtests port the Chromium fixture's auto-height
// column cases: content-based item heights, padding inside the border box,
// main-axis margins in the container height, and the max-height clamp.
//
//nolint:funlen,cyclop,gocognit,maintidx // five Chromium column fixtures share one case
func TestChromeFlexAutoColumnIntrinsicSizing(t *testing.T) {
	t.Parallel()

	t.Run("content-heights", func(t *testing.T) {
		t.Parallel()

		// Chromium fixture 1: a 10pt flex basis, a 10pt height, and a 10pt
		// child inside an auto-height wrapper all resolve to 10pt items in a
		// 30pt container at offsets 0/10/20.
		cssSheet := sheet(t, `
body { margin: 0 }
.case { display: flex; flex-direction: column; width: 400pt }
.item-a { flex: 1 0 10pt }
.item-b { height: 10pt }
`)
		res := layoutHTML(t, `<html><body>
<div class="case">
  <div class="item-a"></div>
  <div class="item-b"></div>
  <div class="item-c"><div class="child-c" style="height: 10pt"></div></div>
</div>
</body></html>`, cssSheet)

		container := findBoxByClass(t, res, "case")
		itemA := findBoxByClass(t, res, "item-a")
		itemB := findBoxByClass(t, res, "item-b")
		itemC := findBoxByClass(t, res, "item-c")
		childC := findBoxByClass(t, res, "child-c")

		if !near(itemA.height, 10) || !near(itemB.height, 10) || !near(itemC.height, 10) {
			t.Fatalf("item heights = %.2f/%.2f/%.2f, want 10/10/10",
				itemA.height, itemB.height, itemC.height)
		}

		if !near(itemA.y, container.y) || !near(itemB.y, container.y+10) || !near(itemC.y, container.y+20) {
			t.Fatalf("item tops = %.2f/%.2f/%.2f, want %.2f/%.2f/%.2f",
				itemA.y, itemB.y, itemC.y, container.y, container.y+10, container.y+20)
		}

		if !near(container.height, 30) {
			t.Fatalf("container height = %.2f, want 30", container.height)
		}

		if !near(childC.height, 10) || !near(childC.y, itemC.y) {
			t.Fatalf("wrapper child = %.2f tall at %.2f, want 10 at %.2f",
				childC.height, childC.y, itemC.y)
		}
	})

	t.Run("padding-heights", func(t *testing.T) {
		t.Parallel()

		// Chromium fixture 3: used item heights are border-box sizes, so the
		// padded item resolves to 20pt (10pt padding-top + 10pt child) while
		// the first two stay at 10pt. Margins do not change used heights.
		cssSheet := sheet(t, `
body { margin: 0 }
.case { display: flex; flex-direction: column; width: 400pt }
.item-a { flex: 1 0 10pt; margin-top: 10pt }
.item-b { height: 10pt; margin-bottom: 20pt }
.item-c { padding-top: 10pt }
`)
		res := layoutHTML(t, `<html><body>
<div class="case">
  <div class="item-a"></div>
  <div class="item-b"></div>
  <div class="item-c"><div class="child-c" style="height: 10pt"></div></div>
</div>
</body></html>`, cssSheet)

		itemA := findBoxByClass(t, res, "item-a")
		itemB := findBoxByClass(t, res, "item-b")
		itemC := findBoxByClass(t, res, "item-c")
		childC := findBoxByClass(t, res, "child-c")

		if !near(itemA.height, 10) || !near(itemB.height, 10) {
			t.Fatalf("item heights = %.2f/%.2f, want 10/10", itemA.height, itemB.height)
		}

		if !near(itemC.height, 20) {
			t.Fatalf("padded item height = %.2f, want 20 (10pt padding + 10pt child)", itemC.height)
		}

		if !near(childC.height, 10) || !near(childC.y, itemC.y+10) {
			t.Fatalf("padded item child = %.2f tall at %.2f, want 10 at %.2f",
				childC.height, childC.y, itemC.y+10)
		}
	})

	t.Run("margin-offsets", func(t *testing.T) {
		t.Parallel()

		// Chromium fixture 3, exact geometry: the first item's 10pt top margin
		// and the second item's 20pt bottom margin push the item tops to
		// 10/20/50 and grow the auto-height container to 70pt. The engine
		// drops main-axis margins, so the strict assertion is blocked.
		t.Skip("blocked: column flex ignores main-axis margins: base sizes exclude them " +
			"(internal/layout/flex.go:1436) and placement adds only heights (internal/layout/flex.go:1777)")

		cssSheet := sheet(t, `
body { margin: 0 }
.case { display: flex; flex-direction: column; width: 400pt }
.item-a { flex: 1 0 10pt; margin-top: 10pt }
.item-b { height: 10pt; margin-bottom: 20pt }
.item-c { padding-top: 10pt }
`)
		res := layoutHTML(t, `<html><body>
<div class="case">
  <div class="item-a"></div>
  <div class="item-b"></div>
  <div class="item-c"><div class="child-c" style="height: 10pt"></div></div>
</div>
</body></html>`, cssSheet)

		container := findBoxByClass(t, res, "case")
		itemA := findBoxByClass(t, res, "item-a")
		itemB := findBoxByClass(t, res, "item-b")
		itemC := findBoxByClass(t, res, "item-c")
		childC := findBoxByClass(t, res, "child-c")

		if !near(itemA.y, container.y+10) || !near(itemB.y, container.y+20) || !near(itemC.y, container.y+50) {
			t.Fatalf("item tops = %.2f/%.2f/%.2f, want %.2f/%.2f/%.2f",
				itemA.y, itemB.y, itemC.y, container.y+10, container.y+20, container.y+50)
		}

		if !near(childC.y, container.y+60) {
			t.Fatalf("padded item child top = %.2f, want %.2f", childC.y, container.y+60)
		}

		if !near(container.height, 70) {
			t.Fatalf("container height = %.2f, want 70 (item margins add to the auto height)", container.height)
		}
	})

	t.Run("max-height-clamp-includes-padding", func(t *testing.T) {
		t.Parallel()

		// Chromium fixture 8: a 30pt content-box max-height with 1pt
		// padding-top and 2pt padding-bottom clamps the border box to 33pt,
		// and the first item starts 1pt below the container top.
		cssSheet := sheet(t, `
body { margin: 0 }
.case {
  display: flex;
  flex-direction: column;
  width: 400pt;
  min-height: 5pt;
  max-height: 30pt;
  padding-top: 1pt;
  padding-bottom: 2pt;
}
.item { min-height: 0; flex: 0 1 auto }
.inner { height: 20pt }
`)
		res := layoutHTML(t, `<html><body>
<div class="case">
  <div class="item"><div class="inner"></div></div>
  <div class="item"><div class="inner"></div></div>
</div>
</body></html>`, cssSheet)

		container := findBoxByClass(t, res, "case")
		items := classBoxes(res.root, "item")

		if len(items) != 2 {
			t.Fatalf("item boxes = %d, want 2", len(items))
		}

		if !near(container.height, 33) {
			t.Fatalf("container height = %.2f, want 33 (30pt max-height + 3pt padding)", container.height)
		}

		if !near(items[0].y, container.y+1) {
			t.Fatalf("first item top = %.2f, want %.2f (container top + padding-top)",
				items[0].y, container.y+1)
		}
	})

	t.Run("min-height-zero-shrink", func(t *testing.T) {
		t.Parallel()

		// Chromium fixture 8, exact geometry: the two 20pt items share the
		// 30pt content height and resolve to 15pt each at offsets 1 and 16.
		// The engine keeps the content floor and flexes against min-height.
		t.Skip("blocked: explicit min-height:0 does not override the content floor " +
			"(internal/layout/flex.go:1503); auto height flexes against min-height (internal/layout/flex.go:1569)")

		cssSheet := sheet(t, `
body { margin: 0 }
.case {
  display: flex;
  flex-direction: column;
  width: 400pt;
  min-height: 5pt;
  max-height: 30pt;
  padding-top: 1pt;
  padding-bottom: 2pt;
}
.item { min-height: 0; flex: 0 1 auto }
.inner { height: 20pt }
`)
		res := layoutHTML(t, `<html><body>
<div class="case">
  <div class="item"><div class="inner"></div></div>
  <div class="item"><div class="inner"></div></div>
</div>
</body></html>`, cssSheet)

		container := findBoxByClass(t, res, "case")
		items := classBoxes(res.root, "item")

		if len(items) != 2 {
			t.Fatalf("item boxes = %d, want 2", len(items))
		}

		wantTops := []float64{container.y + 1, container.y + 16}
		for i := range items {
			if !near(items[i].height, 15) || !near(items[i].y, wantTops[i]) {
				t.Fatalf("item[%d] = height %.2f at %.2f, want 15 at %.2f",
					i, items[i].height, items[i].y, wantTops[i])
			}
		}
	})
}

// TestChromeFlexContainerMaxContentContribution covers case
// wpt-flex-container-max-content (goTarget chrome-reference).
//
// Source: third_party/blink/web_tests/external/wpt/css/css-flexbox/flex-container-max-content-001.html
// Expected: the row flex container's max-content width is the sum of its
// items' outer max-content contributions: content + padding + border +
// margins. Chromium declares width:max-content on the container; the Go
// geometry uses justify-self:center, the engine's fit-content path, which
// resolves to max-content while the containing block is wider. WPT's Ahem
// text is replaced by a definite-width inline-block so the contribution is
// font-independent. This wave does not launch Chromium.
//
//nolint:funlen // the passing and blocked contribution variants share one case
func TestChromeFlexContainerMaxContentContribution(t *testing.T) {
	t.Parallel()

	t.Run("padding-and-border-contributions", func(t *testing.T) {
		t.Parallel()

		// 30pt and 40pt content boxes: 30+3+3+2+2 = 40pt and 40+3+3+2+2 =
		// 50pt, so the container max-content width is 90pt and both items
		// keep their outer sizes side by side. The container itself has no
		// padding or border, so its border box equals the 90pt content sum.
		cssSheet := sheet(t, `
body { margin: 0 }
.case { display: flex; width: max-content; justify-self: center }
.item { padding: 3pt; border: 2pt solid #0aa }
.ink { display: inline-block; width: 30pt; height: 10pt }
.wide { width: 40pt }
`)
		res := layoutHTML(t, `<html><body>
<div class="case">
  <div class="item item-a"><span class="ink"></span></div>
  <div class="item item-b"><span class="ink wide"></span></div>
</div>
</body></html>`, cssSheet)

		container := findBoxByClass(t, res, "case")
		itemA := findBoxByClass(t, res, "item-a")
		itemB := findBoxByClass(t, res, "item-b")

		if !near(container.w, 90) {
			t.Fatalf("container max-content width = %.2f, want 90", container.w)
		}

		if !near(itemA.w, 40) || !near(itemB.w, 50) {
			t.Fatalf("item outer widths = %.2f/%.2f, want 40/50", itemA.w, itemB.w)
		}

		if !near(itemB.x, itemA.x+itemA.w) {
			t.Fatalf("second item x = %.2f, want %.2f after the first", itemB.x, itemA.x+itemA.w)
		}
	})

	t.Run("item-margins", func(t *testing.T) {
		t.Parallel()

		// Chromium's WPT box model adds margin:5px on every side, so the same
		// two items contribute 50pt and 60pt and the container resolves to
		// 110pt. The engine's intrinsic sum omits item margins.
		t.Skip("blocked: flex intrinsic max-content omits item margins " +
			"(internal/layout/flex.go:657; the used-basis path adds them at internal/layout/flex.go:609)")

		cssSheet := sheet(t, `
body { margin: 0 }
.case { display: flex; width: max-content; justify-self: center }
.item { margin: 5pt; padding: 3pt; border: 2pt solid #0aa }
.ink { display: inline-block; width: 30pt; height: 10pt }
.wide { width: 40pt }
`)
		res := layoutHTML(t, `<html><body>
<div class="case">
  <div class="item item-a"><span class="ink"></span></div>
  <div class="item item-b"><span class="ink wide"></span></div>
</div>
</body></html>`, cssSheet)

		container := findBoxByClass(t, res, "case")

		if !near(container.w, 110) {
			t.Fatalf("container max-content width = %.2f, want 110 (sum of outer contributions)", container.w)
		}
	})
}

// TestChromeFlexContainerMinContentContribution covers case
// wpt-flex-container-min-content (goTarget chrome-reference).
//
// Source: third_party/blink/web_tests/external/wpt/css/css-flexbox/flex-container-min-content-001.html
// Expected: a width:min-content row flex container resolves to the largest
// outer min-content contribution of its items, not their sum. With the WPT
// box model (margin 5pt, padding 3pt, border 2pt) the 30pt and 40pt content
// boxes contribute 50pt and 60pt, so the container is 60pt wide. Chromium
// imposes min-content with a float plus width:min-content; the Go conversion
// uses width:min-content alone on an in-flow container. This wave does not
// launch Chromium.
func TestChromeFlexContainerMinContentContribution(t *testing.T) {
	t.Parallel()

	t.Skip("blocked: width:min-content is ignored (internal/layout/style_properties.go:577); " +
		"flex containers have no min-content contribution (internal/layout/layout_measure.go:452)")

	cssSheet := sheet(t, `
body { margin: 0 }
.case { display: flex; width: min-content }
.item { margin: 5pt; padding: 3pt; border: 2pt solid #0aa }
.ink { display: inline-block; width: 30pt; height: 10pt }
.wide { width: 40pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="case">
  <div class="item item-a"><span class="ink"></span></div>
  <div class="item item-b"><span class="ink wide"></span></div>
</div>
</body></html>`, cssSheet)

	container := findBoxByClass(t, res, "case")
	itemA := findBoxByClass(t, res, "item-a")
	itemB := findBoxByClass(t, res, "item-b")

	if !near(container.w, 60) {
		t.Fatalf("min-content container width = %.2f, want 60 (largest outer contribution)", container.w)
	}

	// Under the 60pt container the items keep their content-based outer sizes
	// and overflow instead of shrinking below the automatic minimum.
	if !near(itemA.w, 40) || !near(itemB.w, 50) {
		t.Fatalf("item widths = %.2f/%.2f, want 40/50", itemA.w, itemB.w)
	}
}
