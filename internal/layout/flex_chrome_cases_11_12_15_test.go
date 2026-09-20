//nolint:wsl // direct fixture traversal keeps the reference geometry visible
package layout

import "testing"

func chromeCaseChild111215(t *testing.T, container *box, class string) *box {
	t.Helper()

	items := classBoxes(container, class)
	if len(items) != 1 {
		t.Fatalf("Chrome Flex child %q count = %d, want 1", class, len(items))
	}

	return items[0]
}

func assertChromeCaseSize111215(t *testing.T, item *box, wantX, wantW, wantH float64) {
	t.Helper()

	if !near(item.x, wantX) || !near(item.w, wantW) || !near(item.height, wantH) {
		t.Errorf("%s box = x %.2f size %.2fx%.2f, want x %.2f size %.2fx%.2f",
			item.node.Attribute("class"), item.x, item.w, item.height,
			wantX, wantW, wantH)
	}
}

func assertChromeCaseBox111215(t *testing.T, item *box, wantX, wantY, wantW, wantH float64) {
	t.Helper()

	if !near(item.x, wantX) || !near(item.y, wantY) || !near(item.w, wantW) || !near(item.height, wantH) {
		t.Errorf("%s box = (%.2f, %.2f, %.2f, %.2f), want (%.2f, %.2f, %.2f, %.2f)",
			item.node.Attribute("class"), item.x, item.y, item.w, item.height,
			wantX, wantY, wantW, wantH)
	}
}

// TestChromeCase11LegacyMultiline covers the horizontal-tb row wrap-reverse
// branch from Chromium's multiline.html source case.
func TestChromeCase11LegacyMultiline(t *testing.T) {
	t.Parallel()

	res := layoutChromeFlexCase0708(t, "case-11-legacy-multiline.html")
	container := findBoxByID(res.root, "case-11")
	if container == nil {
		t.Fatal("Chrome Flex case box \"case-11\" is missing")
	}
	if !near(container.w, 60) || !near(container.height, 45) {
		t.Fatalf("container size = %.2fx%.2f, want 60x45", container.w, container.height)
	}

	assertChromeCaseSize111215(t, chromeCaseChild111215(t, container, "child-1"), container.x, 30, 5)
	assertChromeCaseBox111215(t, chromeCaseChild111215(t, container, "child-2"), container.x+30, container.y+35, 30, 10)
	assertChromeCaseBox111215(t, chromeCaseChild111215(t, container, "child-3"), container.x, container.y+30, 60, 5)
	assertChromeCaseSize111215(t, chromeCaseChild111215(t, container, "child-4"), container.x, 70, 20)
	assertChromeCaseBox111215(t, chromeCaseChild111215(t, container, "child-5"), container.x, container.y, 60, 10)

	t.Run("wrap-reverse-cross-axis-placement", func(t *testing.T) {
		t.Parallel()

		child := chromeCaseChild111215(t, container, "child-1")
		if !near(child.y, container.y+40) {
			t.Fatalf("child-1 y = %.2f, want %.2f", child.y, container.y+40)
		}
	})

	t.Run("negative-main-axis-overflow", func(t *testing.T) {
		t.Parallel()

		child := chromeCaseChild111215(t, container, "child-4")
		if !near(child.x, container.x) || !near(child.w, 70) {
			t.Fatalf("child-4 geometry = (%.2f, %.2f), want x %.2f width 70", child.x, child.w, container.x)
		}
	})
}

// TestChromeCase12LegacyMultilineAlignContentColumn covers Chromium's
// column-wrap align-content source case. Two 100pt columns are centered in
// the 600pt cross axis.
func TestChromeCase12LegacyMultilineAlignContentColumn(t *testing.T) {
	t.Parallel()

	res := layoutChromeFlexCase0708(t, "case-12-legacy-multiline-align-content-column.html")
	container := findBoxByID(res.root, "case-12")
	if container == nil {
		t.Fatal("Chrome Flex case box \"case-12\" is missing")
	}
	if !near(container.w, 600) || !near(container.height, 20) {
		t.Fatalf("container size = %.2fx%.2f, want 600x20", container.w, container.height)
	}

	for idx, wantX := range []float64{200, 300} {
		child := chromeCaseChild111215(t, container, []string{"child-1", "child-2"}[idx])
		if !near(child.x, container.x+wantX) || !near(child.y, container.y) ||
			!near(child.w, 100) || !near(child.height, 20) {
			t.Errorf("child-%d box = (%.2f, %.2f, %.2f, %.2f), want (%.2f, %.2f, 100, 20)",
				idx+1, child.x, child.y, child.w, child.height,
				container.x+wantX, container.y)
		}
	}
}

// TestChromeCase15LegacyMultilineAlignSelf covers the per-line align-self
// values in the horizontal-tb row branch, including baseline.
//
//nolint:cyclop // one fixture table covers each named align-self branch
func TestChromeCase15LegacyMultilineAlignSelf(t *testing.T) {
	t.Parallel()

	res := layoutChromeFlexCase0708(t, "case-15-legacy-multiline-align-self.html")
	container := findBoxByID(res.root, "case-15")
	if container == nil {
		t.Fatal("Chrome Flex case box \"case-15\" is missing")
	}
	if !near(container.w, 70) || !near(container.height, 60) {
		t.Fatalf("container size = %.2fx%.2f, want 70x60", container.w, container.height)
	}

	wants := []struct {
		class   string
		x, y, h float64
	}{
		{class: "child-1", x: 0, y: 0, h: 10},
		{class: "child-2", x: 10, y: 10, h: 10},
		{class: "child-3", x: 20, y: 20, h: 10},
		{class: "child-6", x: 50, y: 0, h: 30},
		{class: "child-7", x: 60, y: 0, h: 30},
		{class: "child-8", x: 0, y: 30, h: 10},
		{class: "child-9", x: 10, y: 40, h: 10},
		{class: "child-10", x: 20, y: 50, h: 10},
		{class: "child-13", x: 50, y: 30, h: 30},
		{class: "child-14", x: 60, y: 30, h: 30},
	}

	for _, want := range wants {
		item := chromeCaseChild111215(t, container, want.class)
		if !near(item.x, container.x+want.x) || !near(item.y, container.y+want.y) ||
			!near(item.w, 10) || !near(item.height, want.h) {
			t.Errorf("%s box = (%.2f, %.2f, %.2f, %.2f), want (%.2f, %.2f, 10, %.2f)",
				want.class, item.x, item.y, item.w, item.height,
				container.x+want.x, container.y+want.y, want.h)
		}
	}

	t.Run("baseline", func(t *testing.T) {
		t.Parallel()

		base1 := chromeCaseChild111215(t, container, "child-4")
		base2 := chromeCaseChild111215(t, container, "child-5")
		if !near(base1.y, container.y+5) || !near(base2.y, container.y+5) {
			t.Fatalf("baseline items y = %.2f/%.2f, want %.2f/%.2f",
				base1.y, base2.y, container.y+5, container.y+5)
		}
	})
}
