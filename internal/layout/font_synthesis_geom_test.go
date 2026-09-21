//nolint:wsl,nlreturn,unparam // compact white-box checks keep each proof direct
package layout

import (
	"path/filepath"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

func TestFontSynthesisPositionSubScales(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.auto { font-variant-position: sub; font-synthesis-position: auto; font-size: 20pt; }
.none { font-variant-position: sub; font-synthesis-position: none; font-size: 20pt; }
`)
	resAuto := layoutHTML(t, `<html><body><span class="auto">2</span></body></html>`, cssSheet)
	resNone := layoutHTML(t, `<html><body><span class="none">2</span></body></html>`, cssSheet)

	sizeAuto := textOpSize(t, resAuto, "2")
	sizeNone := textOpSize(t, resNone, "2")
	if sizeAuto >= sizeNone*0.9 {
		t.Fatalf("sub+auto size=%.1f should be smaller than none=%.1f", sizeAuto, sizeNone)
	}
}

func TestFontSynthesisSmallCapsUppercases(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.auto { font-variant-caps: small-caps; font-synthesis-small-caps: auto; font-size: 16pt; }
.none { font-variant-caps: small-caps; font-synthesis-small-caps: none; font-size: 16pt; }
`)
	resAuto := layoutHTML(t, `<html><body><span class="auto">ag</span></body></html>`, cssSheet)
	resNone := layoutHTML(t, `<html><body><span class="none">ag</span></body></html>`, cssSheet)

	if got := joinedPaintText(resAuto); got != "AG" {
		t.Fatalf("small-caps auto paint text %q, want AG", got)
	}
	if got := joinedPaintText(resNone); got != "ag" {
		t.Fatalf("small-caps none paint text %q, want ag", got)
	}
}

func TestFontSynthesisStyleFakeOblique(t *testing.T) {
	t.Parallel()

	audit := filepath.Join("..", "..", "testdata", "fonts", "implemented-audit")
	reg := pdf.ScanFontDirs([]string{audit})
	cssSheet := sheet(t, `
.auto { font-family: "Cactus Classical Serif"; font-style: italic; font-synthesis-style: auto; }
.none { font-family: "Cactus Classical Serif"; font-style: italic; font-synthesis-style: none; }
`)
	resAuto := layoutHTMLRegistry(t, `<html><body><span class="auto">Ag</span></body></html>`, reg, cssSheet)
	resNone := layoutHTMLRegistry(t, `<html><body><span class="none">Ag</span></body></html>`, reg, cssSheet)

	if !textOpFakeOblique(t, resAuto) {
		t.Fatal("italic+auto on upright-only face should FakeOblique")
	}
	if textOpFakeOblique(t, resNone) {
		t.Fatal("italic+none must not FakeOblique")
	}
}

func TestFontWidthCondensesAdvance(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.wide { font-size: 20pt; font-width: 100%; }
.nar { font-size: 20pt; font-width: condensed; }
`)
	resW := layoutHTML(t, `<html><body><span class="wide">MMMM</span></body></html>`, cssSheet)
	resN := layoutHTML(t, `<html><body><span class="nar">MMMM</span></body></html>`, cssSheet)
	wW := textOpWidth(t, resW, "MMMM")
	wN := textOpWidth(t, resN, "MMMM")
	if wN >= wW*0.9 {
		t.Fatalf("condensed width=%.1f should be < normal=%.1f", wN, wW)
	}
}

func TestFontStretchCondensesAdvance(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.wide { font-size: 20pt; font-stretch: normal; }
.nar { font-size: 20pt; font-stretch: condensed; }
`)
	resW := layoutHTML(t, `<html><body><span class="wide">MMMM</span></body></html>`, cssSheet)
	resN := layoutHTML(t, `<html><body><span class="nar">MMMM</span></body></html>`, cssSheet)
	wW := textOpWidth(t, resW, "MMMM")
	wN := textOpWidth(t, resN, "MMMM")
	if wN >= wW*0.9 {
		t.Fatalf("font-stretch condensed width=%.1f should be < normal=%.1f", wN, wW)
	}
}

func layoutHTMLRegistry(t *testing.T, src string, reg *pdf.Registry, sheets ...*css.Stylesheet) *Result {
	t.Helper()
	root, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{
		Width: testViewport, Height: 800, Sheets: sheets, Background: true, Registry: reg,
	})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func textOpSize(t *testing.T, res *Result, needle string) float64 {
	t.Helper()
	for _, op := range res.Ops {
		if op.Kind == OpText && op.Text == needle {
			return op.Size
		}
	}
	t.Fatalf("text op %q not found", needle)
	return 0
}

func textOpWidth(t *testing.T, res *Result, needle string) float64 {
	t.Helper()
	for _, op := range res.Ops {
		if op.Kind == OpText && op.Text == needle {
			return op.W
		}
	}
	t.Fatalf("text op %q not found", needle)
	return 0
}

func textOpFakeOblique(t *testing.T, res *Result) bool {
	t.Helper()
	for _, op := range res.Ops {
		if op.Kind == OpText {
			return op.FakeOblique
		}
	}
	t.Fatal("no text op")
	return false
}
