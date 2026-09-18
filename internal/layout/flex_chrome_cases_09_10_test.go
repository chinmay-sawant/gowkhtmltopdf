//nolint:wsl // reference fixtures keep parsing and geometry assertions together
package layout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

type chromeRect struct {
	x, y, w, h float64
}

// TestChromeReferenceLegacyFlexFlowOrientations compares the assigned static
// case with the Chromium geometry recorded by
// third_party/blink/web_tests/css3/flexbox/flex-flow-orientations.html. The
// source matrix is reduced to one named direction, writing-mode, and
// reverse-flow case so the reference remains deterministic and reviewable.
func TestChromeReferenceLegacyFlexFlowOrientations(t *testing.T) {
	t.Parallel()

	src := readChromeCase(t, "legacy-flex-flow-orientations.html")
	res := layoutChromeCase(t, src)
	refs := map[string]chromeRect{
		"item-a": {x: 80, y: 0, w: 20, h: 20},
		"item-b": {x: 80, y: 20, w: 20, h: 20},
	}

	for class, want := range refs {
		got := findBoxByClass(t, res, class)
		assertChromeRect(t, class, chromeRect{
			x: got.x, y: got.y, w: got.w, h: got.height,
		}, want)
	}
}

// TestChromeReferenceLegacyFlexFlow compares the assigned static case with
// the Chromium geometry recorded by
// third_party/blink/web_tests/css3/flexbox/flex-flow.html. Chromium's RTL
// row-reverse example gives A, B, and C widths of 75pt, 350pt, and 75pt and
// starts them at 0pt, 75pt, and 525pt.
func TestChromeReferenceLegacyFlexFlow(t *testing.T) {
	t.Parallel()

	src := readChromeCase(t, "legacy-flex-flow.html")
	res := layoutChromeCase(t, src)
	refs := map[string]chromeRect{
		"item-a": {x: 0, y: 0, w: 75, h: 20},
		"item-b": {x: 75, y: 0, w: 350, h: 20},
		"item-c": {x: 525, y: 0, w: 75, h: 20},
	}

	for class, want := range refs {
		got := findBoxByClass(t, res, class)
		assertChromeRect(t, class, chromeRect{
			x: got.x, y: got.y, w: got.w, h: got.height,
		}, want)
	}
}

func readChromeCase(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("..", "..", "test", "Chrome", "cases", name)
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Chrome case %q: %v", name, err)
	}

	return string(src)
}

func layoutChromeCase(t *testing.T, src string) *Result {
	t.Helper()

	start := strings.Index(src, "<style>")
	end := strings.Index(src, "</style>")
	if start < 0 || end < start+len("<style>") {
		t.Fatalf("Chrome case has no inline stylesheet")
	}

	return layoutHTMLAtViewport(t, src, 1000, 800, sheet(t, src[start+len("<style>"):end]))
}

func layoutHTMLAtViewport(t *testing.T, src string, width, height float64, cssSheet *css.Stylesheet) *Result {
	t.Helper()

	res, err := Layout(mustParse(t, src), Options{
		Width: width, Height: height, Sheets: []*css.Stylesheet{cssSheet}, Background: true,
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	return res
}

func assertChromeRect(t *testing.T, class string, got, want chromeRect) {
	t.Helper()

	assertNearChrome(t, class+" x", got.x, want.x)
	assertNearChrome(t, class+" y", got.y, want.y)
	assertNearChrome(t, class+" width", got.w, want.w)
	assertNearChrome(t, class+" height", got.h, want.h)
}

func assertNearChrome(t *testing.T, label string, got, want float64) {
	t.Helper()

	const tolerance = 1.0
	if got < want-tolerance || got > want+tolerance {
		t.Errorf("%s = %.2fpt, want Chromium %.2fpt +/- %.2fpt", label, got, want, tolerance)
	}
}
