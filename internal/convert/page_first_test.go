package convert //nolint:testpackage // white-box tests need unexported access

import (
	"bytes"
	"math"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

const pageFirstCSS = `@page { margin: 10mm; size: A4 }
@page :first { margin: 40mm }
body, p { margin: 0; padding: 0; }`

var (
	objectHeaderRe    = regexp.MustCompile(`(?m)^(\d+) 0 obj\n`)
	pageContentsRefRe = regexp.MustCompile(`/Contents\s+(\d+)\s+0\s+R`)
	textPosRe         = regexp.MustCompile(`([\d.]+)\s+([\d.\-]+)\s+Td\n`)
)

func TestPageFirstMargins(t *testing.T) {
	t.Parallel()

	unnamedPt := 10 * mmToPt
	firstPt := 40 * mmToPt
	assertPageFirstGeom(t, unnamedPt, firstPt)
	assertPageFirstPDFText(t, unnamedPt, firstPt)
}

func assertPageFirstGeom(t *testing.T, unnamedPt, firstPt float64) {
	t.Helper()

	sheet, err := css.Parse(pageFirstCSS)
	if err != nil {
		t.Fatalf("css.Parse: %v", err)
	}

	if sheet.Page == nil || sheet.Page.Margin != "10mm" {
		t.Fatalf("unnamed @page margin = %+v, want 10mm", sheet.Page)
	}

	initGeom := hfGeom{ //nolint:exhaustruct // test initial geometry
		pageW: 595.28, pageH: 841.89,
		marginTop:    unnamedPt,
		marginBottom: unnamedPt,
		marginLeft:   unnamedPt,
		marginRight:  unnamedPt,
	}
	geom := applyCSSPageMargins(initGeom, []*css.Stylesheet{sheet})

	if math.Abs(geom.marginTop-unnamedPt) > 0.05 || math.Abs(geom.marginLeft-unnamedPt) > 0.05 {
		t.Errorf("unnamed margins = top %v left %v, want %v", geom.marginTop, geom.marginLeft, unnamedPt)
	}

	if geom.first == nil {
		t.Fatal("expected @page :first margins on geom")
	}

	if math.Abs(geom.first.top-firstPt) > 0.05 || math.Abs(geom.first.left-firstPt) > 0.05 {
		t.Errorf(":first margins = top %v left %v, want %v", geom.first.top, geom.first.left, firstPt)
	}
}

func assertPageFirstPDFText(t *testing.T, unnamedPt, firstPt float64) {
	t.Helper()

	html := `<!DOCTYPE html>
<html>
<head><style>` + pageFirstCSS + `</style></head>
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
	delta := firstPt - unnamedPt

	if page1Y >= page2Y {
		t.Errorf("page 1 text y=%.3f is not below page 2 y=%.3f (larger :first top margin)", page1Y, page2Y)
	}

	if math.Abs((page2Y-page1Y)-delta) > 2 {
		t.Errorf("page 1 vs 2 top origin delta = %.3f, want ~%.3f (40mm-10mm); y1=%.3f y2=%.3f",
			page2Y-page1Y, delta, page1Y, page2Y)
	}

	if page1X <= page2X {
		t.Errorf("page 1 text x=%.3f is not right of page 2 x=%.3f (larger :first left margin)", page1X, page2X)
	}

	if math.Abs((page1X-page2X)-delta) > 2 {
		t.Errorf("page 1 vs 2 left origin delta = %.3f, want ~%.3f (40mm-10mm); x1=%.3f x2=%.3f",
			page1X-page2X, delta, page1X, page2X)
	}
}

func firstPageTextPos(t *testing.T, data []byte, pageObj int) (float64, float64) {
	t.Helper()

	dict := objectBody(data, pageObj)
	if dict == nil {
		t.Fatalf("page object %d not found", pageObj)
	}

	ref := pageContentsRefRe.FindSubmatch(dict)
	if ref == nil {
		t.Fatalf("page object %d has no /Contents", pageObj)
	}

	contentsID, _ := strconv.Atoi(string(ref[1]))

	stream := objectStream(data, contentsID)
	if stream == nil {
		t.Fatalf("contents object %d has no stream", contentsID)
	}

	match := textPosRe.FindSubmatch(stream)
	if match == nil {
		t.Fatalf("contents object %d has no Td text position", contentsID)
	}

	posX, err := strconv.ParseFloat(string(match[1]), 64)
	if err != nil {
		t.Fatalf("parse Td x: %v", err)
	}

	posY, err := strconv.ParseFloat(string(match[2]), 64)
	if err != nil {
		t.Fatalf("parse Td y: %v", err)
	}

	return posX, posY
}

func objectBody(data []byte, id int) []byte {
	header := objectHeaderRe.FindAllSubmatchIndex(data, -1)
	want := []byte(strconv.Itoa(id))

	for _, loc := range header {
		if !bytes.Equal(data[loc[2]:loc[3]], want) {
			continue
		}

		body := data[loc[1]:]

		end := bytes.Index(body, []byte("\nendobj\n"))
		if end < 0 {
			return nil
		}

		return body[:end]
	}

	return nil
}

func objectStream(data []byte, id int) []byte {
	body := objectBody(data, id)
	if body == nil {
		return nil
	}

	mark := bytes.Index(body, []byte("\nstream\n"))
	if mark < 0 {
		return nil
	}

	rest := body[mark+len("\nstream\n"):]

	end := bytes.Index(rest, []byte("endstream"))
	if end < 0 {
		return nil
	}

	return rest[:end]
}
