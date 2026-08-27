package convert //nolint:testpackage // white-box tests need unexported access

import (
	"bytes"
	"math"
	"path/filepath"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

// Lite named pages: CSS page:ident is stored and inherited as the used value.
// A sibling whose used name differs gets break-before:always. Named @page
// margins apply on pages that start a box with that name. Size stays unnamed.
// Continuation-only fragments without a new named box keep unnamed/:left/:right/:first.

const pageNamedCSS = `@page { margin: 10mm; size: A4 }
@page chapter { margin: 40mm }
body, p { margin: 0; padding: 0; }
.ch { page: chapter }`

func TestPageNamedMargins(t *testing.T) {
	t.Parallel()

	unnamedPt := 10 * mmToPt
	namedPt := 40 * mmToPt

	sheet, err := css.Parse(pageNamedCSS)
	if err != nil {
		t.Fatalf("css.Parse: %v", err)
	}

	geom := applyCSSPageMargins(sideInitGeom(unnamedPt), []*css.Stylesheet{sheet})
	if geom.named == nil {
		t.Fatal("expected named @page chapter margins")
	}

	ch, ok := geom.named["chapter"]
	if !ok || ch == nil {
		t.Fatalf("named map = %v, want chapter", geom.named)
	}

	if math.Abs(ch.top-namedPt) > 0.05 || math.Abs(ch.left-namedPt) > 0.05 {
		t.Errorf("chapter margins = top %v left %v, want %v", ch.top, ch.left, namedPt)
	}

	html := `<!DOCTYPE html>
<html>
<head><style>` + pageNamedCSS + `</style></head>
<body>
<p>PAGEONE</p>
<div class="ch"><p>PAGETWO</p></div>
</body>
</html>`

	cmd, _ := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.UseCompression = false
	cmd.Global.Outline = false
	data := runPDF(t, cmd)

	if n := pageCount(data); n < 2 {
		t.Fatalf("pages = %d, want >= 2 (named page change breaks)", n)
	}

	kids := pageKidsRefs(data)
	if len(kids) < 2 {
		t.Fatalf("Kids page refs = %v, want >= 2", kids)
	}

	page1X, page1Y := firstPageTextPos(t, data, kids[0])
	page2X, page2Y := firstPageTextPos(t, data, kids[1])
	delta := namedPt - unnamedPt

	if page2Y >= page1Y {
		t.Errorf("named page y=%.3f is not below unnamed y=%.3f", page2Y, page1Y)
	}

	if math.Abs((page1Y-page2Y)-delta) > 2 {
		t.Errorf("top origin delta = %.3f, want ~%.3f; y1=%.3f y2=%.3f",
			page1Y-page2Y, delta, page1Y, page2Y)
	}

	if page2X <= page1X {
		t.Errorf("named page x=%.3f is not right of unnamed x=%.3f", page2X, page1X)
	}

	if math.Abs((page2X-page1X)-delta) > 2 {
		t.Errorf("left origin delta = %.3f, want ~%.3f; x1=%.3f x2=%.3f",
			page2X-page1X, delta, page1X, page2X)
	}
}

func TestPageNameBreak(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<html>
<head><style>
body, p { margin: 0; padding: 0; }
.ch { page: chapter }
</style></head>
<body>
<p>PAGEONE</p>
<div class="ch"><p>PAGETWO</p></div>
</body>
</html>`

	cmd, _ := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.UseCompression = false
	cmd.Global.Outline = false
	data := runPDF(t, cmd)

	if n := pageCount(data); n < 2 {
		t.Fatalf("pages = %d, want >= 2 after page:chapter sibling", n)
	}
}

func TestPageMarginBoxes(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<html>
<head><style>
@page {
  margin: 20mm;
  @top-center { content: "HDRBOX" }
  @bottom-left { content: "FTRBOX" }
}
body, p { margin: 0; }
</style></head>
<body><p>BODYTEXT</p></body>
</html>`

	cmd, _ := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.UseCompression = false
	cmd.Global.Outline = false
	data := runPDF(t, cmd)

	if !bytes.Contains(data, []byte("HDRBOX")) {
		t.Error("@top-center content missing from PDF header chrome")
	}

	if !bytes.Contains(data, []byte("FTRBOX")) {
		t.Error("@bottom-left content missing from PDF footer chrome")
	}

	if !bytes.Contains(data, []byte("BODYTEXT")) {
		t.Error("body text missing")
	}
}

func TestPageMarginBoxesCLIWins(t *testing.T) {
	t.Parallel()

	html := `<!DOCTYPE html>
<html>
<head><style>
@page { @top-center { content: "CSSHDR" } }
</style></head>
<body><p>x</p></body>
</html>`

	cmd, _ := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.Header.Center = "CLIHDR"
	cmd.Global.UseCompression = false
	cmd.Global.Outline = false
	data := runPDF(t, cmd)

	if !bytes.Contains(data, []byte("CLIHDR")) {
		t.Error("CLI --header-center should win over @top-center")
	}

	if bytes.Contains(data, []byte("CSSHDR")) {
		t.Error("CSS @top-center filled a CLI-occupied header slot")
	}
}
