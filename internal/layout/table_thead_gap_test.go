//nolint:testpackage // tests exercise unexported table pagination helpers
package layout

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// Continuation-page body rows under a repeated thead must stay grid-adjacent.
// A paint gap between the first and second body row shows up as a white seam
// (fixture-60 props 51/52 and 104/105).
func TestFixture60TheadContinuationRowsHaveNoPaintGap(t *testing.T) {
	t.Parallel()

	rootDir, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(rootDir, "testdata/golden/fixture-60-implemented-props-a.html"))
	if err != nil {
		t.Fatal(err)
	}

	doc, err := html.Parse(string(raw))
	if err != nil {
		t.Fatal(err)
	}

	sheet, err := css.Parse(extractStyleContent(doc))
	if err != nil {
		t.Fatal(err)
	}

	base := filepath.Join(rootDir, "testdata/golden")
	margin := 12 * 72 / 25.4
	pageW, pageH := 595.28, 841.89
	contentW := pageW - 2*margin
	contentH := pageH - 2*margin

	res, err := Layout(doc, Options{ //nolint:exhaustruct
		Width: contentW, Height: contentH, Background: true, Media: "print", Zoom: 1,
		Sheets: []*css.Stylesheet{sheet},
		Images: func(src string) ([]byte, error) {
			src = strings.TrimPrefix(src, "file://")
			if strings.HasPrefix(src, "data:") {
				return nil, os.ErrNotExist
			}
			if !filepath.IsAbs(src) {
				src = filepath.Join(base, src)
			}

			return os.ReadFile(src)
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	pdfDoc := pdf.NewDocument()
	if err := Paint(pdfDoc, res, PaintOptions{ //nolint:exhaustruct
		PageWidth: pageW, PageHeight: pageH,
		MarginTop: margin, MarginBottom: margin, MarginLeft: margin, MarginRight: margin,
	}); err != nil {
		t.Fatal(err)
	}

	var table *box
	for _, b := range flowBoxList(res) {
		if b.kind == displayTable && len(b.rows) > 100 {
			table = b
			break
		}
	}
	if table == nil {
		t.Fatal("fixture-60 table not found")
	}

	seen := map[int]bool{}
	for ri := table.headerRows; ri+1 < len(table.rows); ri++ {
		_, _, top, bot, ok := rowPaintBand(table.rows[ri], res)
		if !ok {
			continue
		}
		page := int(top / contentH)
		if page == 0 || seen[page] {
			continue
		}
		seen[page] = true

		_, _, nextTop, _, nextOK := rowPaintBand(table.rows[ri+1], res)
		if !nextOK {
			t.Fatalf("page %d: second body row has no paint band", page)
		}
		if int(nextTop/contentH) != page {
			continue
		}

		gap := nextTop - bot
		if gap > 0.75 {
			t.Fatalf("page %d: paint gap %.3fpt between first body row %d and next (top=%.2f bot=%.2f nextTop=%.2f)",
				page, gap, ri, top, bot, nextTop)
		}
		if gap < -0.75 {
			t.Fatalf("page %d: paint overlap %.3fpt between first body row %d and next", page, gap, ri)
		}
	}

	if len(seen) < 3 {
		t.Fatalf("expected several continuation pages, got %d", len(seen))
	}
}

func TestRowPaintBandPrefersVerticalRules(t *testing.T) {
	t.Parallel()

	res := &Result{ //nolint:exhaustruct
		Ops: []Op{
			{Kind: OpText, X: 10, Y: 100, H: 12, Text: "a"},
			{Kind: OpLine, X: 0, Y: 90, W: 0, H: 40},  // vertical rule
			{Kind: OpLine, X: 50, Y: 90, W: 0, H: 40}, // vertical rule
			{Kind: OpText, X: 10, Y: 105, H: 10, Text: "b"},
		},
	}
	cell := &box{ //nolint:exhaustruct
		opStart: 0, opEnd: 3, y: 100, height: 30,
	}
	row := []*box{cell}

	_, _, top, bot, ok := rowPaintBand(row, res)
	if !ok {
		t.Fatal("expected paint band")
	}
	if math.Abs(top-90) > 1e-9 || math.Abs(bot-130) > 1e-9 {
		t.Fatalf("paint band = [%.2f,%.2f], want [90,130] from verticals", top, bot)
	}
}

// Last body row on a page whose next row continues on the following page must
// get a full-width bottom seal (fixture-60 pages ending at props 33 and 67).
func TestFixture60PageBottomRowsAreSealed(t *testing.T) {
	t.Parallel()

	rootDir, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(rootDir, "testdata/golden/fixture-60-implemented-props-a.html"))
	if err != nil {
		t.Fatal(err)
	}

	doc, err := html.Parse(string(raw))
	if err != nil {
		t.Fatal(err)
	}

	sheet, err := css.Parse(extractStyleContent(doc))
	if err != nil {
		t.Fatal(err)
	}

	base := filepath.Join(rootDir, "testdata/golden")
	margin := 12 * 72 / 25.4
	pageW, pageH := 595.28, 841.89
	contentW := pageW - 2*margin
	contentH := pageH - 2*margin

	res, err := Layout(doc, Options{ //nolint:exhaustruct
		Width: contentW, Height: contentH, Background: true, Media: "print", Zoom: 1,
		Sheets: []*css.Stylesheet{sheet},
		Images: func(src string) ([]byte, error) {
			src = strings.TrimPrefix(src, "file://")
			if strings.HasPrefix(src, "data:") {
				return nil, os.ErrNotExist
			}
			if !filepath.IsAbs(src) {
				src = filepath.Join(base, src)
			}

			return os.ReadFile(src)
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	pdfDoc := pdf.NewDocument()
	if err := Paint(pdfDoc, res, PaintOptions{ //nolint:exhaustruct
		PageWidth: pageW, PageHeight: pageH,
		MarginTop: margin, MarginBottom: margin, MarginLeft: margin, MarginRight: margin,
	}); err != nil {
		t.Fatal(err)
	}

	var table *box
	for _, b := range flowBoxList(res) {
		if b.kind == displayTable && len(b.rows) > 100 {
			table = b
			break
		}
	}
	if table == nil {
		t.Fatal("fixture-60 table not found")
	}

	findIdx := func(want string) int {
		for ri, row := range table.rows {
			first, last, _, _, ok := rowPaintBand(row, res)
			if !ok {
				continue
			}
			for i := first; i <= last && i < len(res.Ops); i++ {
				if res.Ops[i].Kind == OpText && res.Ops[i].Text == want {
					return ri
				}
			}
		}

		return -1
	}

	for _, want := range []string{"33", "67"} {
		ri := findIdx(want)
		if ri < 0 {
			t.Fatalf("idx %s not found", want)
		}
		_, _, _, bot, ok := rowPaintBand(table.rows[ri], res)
		if !ok {
			t.Fatalf("idx %s: no paint band", want)
		}
		hasFull := false
		for _, op := range res.Ops {
			if op.Kind != OpLine || op.H > 0.01 || op.W < 400 {
				continue
			}
			if math.Abs(op.Y-bot) <= 1.0 {
				hasFull = true
				break
			}
		}
		if !hasFull {
			t.Fatalf("idx %s: missing full-width bottom seal at y=%.2f", want, bot)
		}
	}
}
