package convert //nolint:testpackage // white-box tests need unexported access

import (
	"math"
	"path/filepath"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

// LTR print: page 1 is :right (recto), even pages are :left, odd pages :right.
// :first overrides :left/:right on page 1.

const pageSideCSS = `@page { margin: 10mm; size: A4 }
@page :right { margin: 20mm }
@page :left { margin: 40mm }
body, p { margin: 0; padding: 0; }`

const pageSideFirstCSS = `@page { margin: 10mm; size: A4 }
@page :right { margin: 20mm }
@page :left { margin: 40mm }
@page :first { margin: 5mm }
body, p { margin: 0; padding: 0; }`

func TestPageLeftRightMargins(t *testing.T) {
	t.Parallel()

	unnamedPt := 10 * mmToPt
	rightPt := 20 * mmToPt
	leftPt := 40 * mmToPt
	assertPageSideGeom(t, unnamedPt, rightPt, leftPt)
	assertPageSidePDFText(t, pageSideCSS, rightPt, leftPt)
}

func TestPageFirstWinsOverLeftRight(t *testing.T) {
	t.Parallel()

	firstPt := 5 * mmToPt
	leftPt := 40 * mmToPt
	assertPageFirstOverSideGeom(t, firstPt, leftPt)
	assertPageSidePDFText(t, pageSideFirstCSS, firstPt, leftPt)
}

func assertPageSideGeom(t *testing.T, unnamedPt, rightPt, leftPt float64) {
	t.Helper()

	sheet, err := css.Parse(pageSideCSS)
	if err != nil {
		t.Fatalf("css.Parse: %v", err)
	}

	geom := applyCSSPageMargins(sideInitGeom(unnamedPt), []*css.Stylesheet{sheet})
	if math.Abs(geom.marginTop-unnamedPt) > 0.05 {
		t.Errorf("unnamed top = %v, want %v", geom.marginTop, unnamedPt)
	}

	if geom.right == nil {
		t.Fatal("expected @page :right margins")
	}

	if math.Abs(geom.right.top-rightPt) > 0.05 || math.Abs(geom.right.left-rightPt) > 0.05 {
		t.Errorf(":right = top %v left %v, want %v", geom.right.top, geom.right.left, rightPt)
	}

	if geom.left == nil {
		t.Fatal("expected @page :left margins")
	}

	if math.Abs(geom.left.top-leftPt) > 0.05 || math.Abs(geom.left.left-leftPt) > 0.05 {
		t.Errorf(":left = top %v left %v, want %v", geom.left.top, geom.left.left, leftPt)
	}

	rightL, rightT := geom.pageMargins(0)
	if math.Abs(rightL-rightPt) > 0.05 || math.Abs(rightT-rightPt) > 0.05 {
		t.Errorf("page 1 (recto) margins = %v, %v, want :right %v", rightL, rightT, rightPt)
	}

	leftL, leftT := geom.pageMargins(1)
	if math.Abs(leftL-leftPt) > 0.05 || math.Abs(leftT-leftPt) > 0.05 {
		t.Errorf("page 2 (verso) margins = %v, %v, want :left %v", leftL, leftT, leftPt)
	}
}

func assertPageFirstOverSideGeom(t *testing.T, firstPt, leftPt float64) {
	t.Helper()

	sheet, err := css.Parse(pageSideFirstCSS)
	if err != nil {
		t.Fatalf("css.Parse: %v", err)
	}

	unnamedPt := 10 * mmToPt
	geom := applyCSSPageMargins(sideInitGeom(unnamedPt), []*css.Stylesheet{sheet})

	if geom.first == nil {
		t.Fatal("expected @page :first margins")
	}

	firstL, firstT := geom.pageMargins(0)
	if math.Abs(firstL-firstPt) > 0.05 || math.Abs(firstT-firstPt) > 0.05 {
		t.Errorf("page 1 margins = %v, %v, want :first %v", firstL, firstT, firstPt)
	}

	leftL, leftT := geom.pageMargins(1)
	if math.Abs(leftL-leftPt) > 0.05 || math.Abs(leftT-leftPt) > 0.05 {
		t.Errorf("page 2 margins = %v, %v, want :left %v", leftL, leftT, leftPt)
	}
}

func sideInitGeom(unnamedPt float64) hfGeom {
	return hfGeom{ //nolint:exhaustruct // test initial geometry
		pageW:        595.28,
		pageH:        841.89,
		marginTop:    unnamedPt,
		marginBottom: unnamedPt,
		marginLeft:   unnamedPt,
		marginRight:  unnamedPt,
	}
}

func assertPageSidePDFText(t *testing.T, pageCSS string, page1Pt, page2Pt float64) {
	t.Helper()

	html := `<!DOCTYPE html>
<html>
<head><style>` + pageCSS + `</style></head>
<body>
<p>PAGEONE</p>
<div style="page-break-before:always"></div>
<p>PAGETWO</p>
</body>
</html>`

	cmd, _ := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.UseCompression = false
	cmd.Global.Outline = false
	data := runPDF(t, cmd)

	if n := pageCount(data); n < 2 {
		t.Fatalf("pages = %d, want >= 2", n)
	}

	kids := pageKidsRefs(data)
	if len(kids) < 2 {
		t.Fatalf("Kids page refs = %v, want >= 2", kids)
	}

	page1X, page1Y := firstPageTextPos(t, data, kids[0])
	page2X, page2Y := firstPageTextPos(t, data, kids[1])
	delta := page2Pt - page1Pt

	if delta > 0.5 {
		if page2Y >= page1Y {
			t.Errorf("page 2 y=%.3f is not below page 1 y=%.3f (larger :left top)", page2Y, page1Y)
		}

		if math.Abs((page1Y-page2Y)-delta) > 2 {
			t.Errorf("top origin delta = %.3f, want ~%.3f; y1=%.3f y2=%.3f",
				page1Y-page2Y, delta, page1Y, page2Y)
		}

		if page2X <= page1X {
			t.Errorf("page 2 x=%.3f is not right of page 1 x=%.3f (larger :left left)", page2X, page1X)
		}

		if math.Abs((page2X-page1X)-delta) > 2 {
			t.Errorf("left origin delta = %.3f, want ~%.3f; x1=%.3f x2=%.3f",
				page2X-page1X, delta, page1X, page2X)
		}

		return
	}

	if page1Y >= page2Y {
		t.Errorf("page 1 y=%.3f is not below page 2 y=%.3f (larger page-1 top)", page1Y, page2Y)
	}

	if math.Abs((page2Y-page1Y)-(-delta)) > 2 {
		t.Errorf("top origin delta = %.3f, want ~%.3f; y1=%.3f y2=%.3f",
			page2Y-page1Y, -delta, page1Y, page2Y)
	}

	if page1X <= page2X {
		t.Errorf("page 1 x=%.3f is not right of page 2 x=%.3f (larger page-1 left)", page1X, page2X)
	}

	if math.Abs((page1X-page2X)-(-delta)) > 2 {
		t.Errorf("left origin delta = %.3f, want ~%.3f; x1=%.3f x2=%.3f",
			page1X-page2X, -delta, page1X, page2X)
	}
}
