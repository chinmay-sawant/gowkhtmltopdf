//nolint:wsl,varnamelen,nlreturn,cyclop,gocognit,lll // table-driven fixture geometry stays together
package layout

import "testing"

func chromeFlexFixture(t *testing.T, name string) *Result {
	t.Helper()

	src := readChromeCase(t, name)

	return layoutChromeCase(t, src)
}

// TestChromeFlexAlgorithmMargins adapts
// third_party/blink/web_tests/css3/flexbox/flex-algorithm-with-margins.html.
// It proves that positive free space goes to auto margins, negative free space
// still flexes the fixed item, and container padding moves the content box.
func TestChromeFlexAlgorithmMargins(t *testing.T) {
	t.Parallel()

	res := chromeFlexFixture(t, "legacy-flex-algorithm-margins.html")

	t.Run("positive-free-space", func(t *testing.T) {
		t.Parallel()

		first := findBoxByID(res.root, "positive-first")
		auto := findBoxByID(res.root, "positive-auto")
		last := findBoxByID(res.root, "positive-last")
		if first == nil || auto == nil || last == nil {
			t.Fatal("positive auto-margin boxes missing")
		}

		if !near(first.w, 100) || !near(auto.w, 100) || !near(last.w, 100) {
			t.Fatalf("positive widths = %.2f/%.2f/%.2f, want 100/100/100", first.w, auto.w, last.w)
		}
		if !near(first.x, 0) || !near(auto.x, 250) || !near(last.x, 500) {
			t.Fatalf("positive x positions = %.2f/%.2f/%.2f, want 0/250/500", first.x, auto.x, last.x)
		}
	})

	t.Run("negative-free-space", func(t *testing.T) {
		t.Parallel()

		first := findBoxByID(res.root, "negative-first")
		auto := findBoxByID(res.root, "negative-auto")
		last := findBoxByID(res.root, "negative-last")
		if first == nil || auto == nil || last == nil {
			t.Fatal("negative auto-margin boxes missing")
		}

		if !near(first.w, 150) || !near(auto.w, 300) || !near(last.w, 150) {
			t.Fatalf("negative widths = %.2f/%.2f/%.2f, want 150/300/150", first.w, auto.w, last.w)
		}
		if !near(first.x, 0) || !near(auto.x, 150) || !near(last.x, 450) {
			t.Fatalf("negative x positions = %.2f/%.2f/%.2f, want 0/150/450", first.x, auto.x, last.x)
		}
	})

	t.Run("container-padding", func(t *testing.T) {
		t.Parallel()

		container := findBoxByID(res.root, "padded-free-space")
		first := findBoxByID(res.root, "padded-first")
		last := findBoxByID(res.root, "padded-last")
		if container == nil || first == nil || last == nil {
			t.Fatal("padded auto-margin boxes missing")
		}

		if !near(container.w, 800) || !near(first.w, 300) || !near(last.w, 300) {
			t.Fatalf("padded widths = container %.2f, items %.2f/%.2f, want 800/300/300", container.w, first.w, last.w)
		}
		if !near(first.x, 100) || !near(last.x, 400) {
			t.Fatalf("padded x positions = %.2f/%.2f, want 100/400", first.x, last.x)
		}
	})
}

// TestChromeColumnsAutoSize adapts
// third_party/blink/web_tests/css3/flexbox/columns-auto-size.html.
// It proves that an auto-height column includes child content and margins,
// while min/max and vertical padding constrain the final column size.
func TestChromeColumnsAutoSize(t *testing.T) { //nolint:gocyclo,funlen // explicit subcases keep each sizing branch visible
	t.Parallel()

	res := chromeFlexFixture(t, "legacy-columns-auto-size.html")

	t.Run("content", func(t *testing.T) {
		t.Parallel()

		container := findBoxByID(res.root, "column-content")
		first := findBoxByID(res.root, "content-first")
		second := findBoxByID(res.root, "content-second")
		third := findBoxByID(res.root, "content-third")
		if container == nil || first == nil || second == nil || third == nil {
			t.Fatal("content column boxes missing")
		}

		if !near(container.height, 30) {
			t.Fatalf("content column height = %.2f, want 30", container.height)
		}
		if !near(first.height, 10) || !near(second.height, 10) || !near(third.height, 10) {
			t.Fatalf("content item heights = %.2f/%.2f/%.2f, want 10/10/10", first.height, second.height, third.height)
		}
		if !near(first.y, 0) || !near(second.y, 10) || !near(third.y, 20) {
			t.Fatalf("content y positions = %.2f/%.2f/%.2f, want 0/10/20", first.y, second.y, third.y)
		}
	})

	t.Run("margins", func(t *testing.T) {
		t.Parallel()

		container := findBoxByID(res.root, "column-margins")
		first := findBoxByID(res.root, "margins-first")
		second := findBoxByID(res.root, "margins-second")
		third := findBoxByID(res.root, "margins-third")
		if container == nil || first == nil || second == nil || third == nil {
			t.Fatal("margin column boxes missing")
		}

		if !near(container.height, 70) {
			t.Fatalf("margin column height = %.2f, want 70; item y=%.2f/%.2f/%.2f", container.height, first.y, second.y, third.y)
		}
		firstY := first.y - container.y
		secondY := second.y - container.y
		thirdY := third.y - container.y
		if !near(firstY, 10) || !near(secondY, 20) || !near(thirdY, 50) {
			t.Fatalf("margin relative y positions = %.2f/%.2f/%.2f, want 10/20/50", firstY, secondY, thirdY)
		}
		if !near(third.height, 20) {
			t.Fatalf("margin padded item height = %.2f, want 20", third.height)
		}
	})

	t.Run("max-height", func(t *testing.T) {
		t.Parallel()

		container := findBoxByID(res.root, "column-max-constraints")
		first := findBoxByID(res.root, "max-first")
		second := findBoxByID(res.root, "max-second")
		if container == nil || first == nil || second == nil {
			t.Fatal("max-height column boxes missing")
		}

		// Chrome keeps the negative free-space split fractional. Its CSS-pixel
		// values convert to approximately 8.5pt for both items.
		if !near(container.height, 17) || !near(first.height, 8.5) || !near(second.height, 8.5) {
			t.Fatalf("max-height geometry = container %.2f, items %.2f/%.2f, want 17/8.5/8.5", container.height, first.height, second.height)
		}
		firstY := first.y - container.y
		secondY := second.y - container.y
		if !near(firstY, 0) || !near(secondY, 8.5) {
			t.Fatalf("max-height relative y positions = %.2f/%.2f, want 0/8.5", firstY, secondY)
		}
	})

	t.Run("padding-and-max-height", func(t *testing.T) {
		t.Parallel()

		container := findBoxByID(res.root, "column-padding-constraints")
		first := findBoxByID(res.root, "padding-first")
		second := findBoxByID(res.root, "padding-second")
		if container == nil || first == nil || second == nil {
			t.Fatal("padding constraint boxes missing")
		}

		// Chrome preserves the 31pt content height as two 15.5pt items.
		if !near(container.height, 33) || !near(first.height, 15.5) || !near(second.height, 15.5) {
			t.Fatalf("padding geometry = container %.2f, items %.2f/%.2f, want 33/15.5/15.5", container.height, first.height, second.height)
		}
		firstY := first.y - container.y
		secondY := second.y - container.y
		if !near(firstY, 1) || !near(secondY, 16.5) {
			t.Fatalf("padding relative y positions = %.2f/%.2f, want 1/16.5", firstY, secondY)
		}
	})
}

func findBoxByID(root *box, id string) *box {
	if root == nil {
		return nil
	}
	if root.node != nil && root.node.Attribute("id") == id {
		return root
	}
	for _, child := range root.children {
		if found := findBoxByID(child, id); found != nil {
			return found
		}
	}
	return nil
}
