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

	res, table, contentH := layoutFixture60(t)

	assertContinuationRowsHaveNoPaintGap(t, table, res, contentH)
}

func layoutFixture60(t *testing.T) (*Result, *box, float64) {
	t.Helper()

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

	parsedSheet, err := css.Parse(extractStyleContent(doc))
	if err != nil {
		t.Fatal(err)
	}

	base := filepath.Join(rootDir, "testdata/golden")
	res, contentH := layoutFixture60Result(t, doc, parsedSheet, base)
	table := findFixture60Table(t, res)

	return res, table, contentH
}

func layoutFixture60Result(
	t *testing.T,
	doc *html.Node,
	parsedSheet *css.Stylesheet,
	base string,
) (*Result, float64) {
	t.Helper()

	margin := 12 * 72 / 25.4
	pageW, pageH := 595.28, 841.89
	contentW := pageW - 2*margin
	contentH := pageH - 2*margin

	res, err := Layout(doc, Options{ //nolint:exhaustruct
		Width: contentW, Height: contentH, Background: true, Media: "print", Zoom: 1,
		Sheets: []*css.Stylesheet{parsedSheet},
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

	if err := Paint(pdfDoc, res, PaintOptions{
		PageWidth: pageW, PageHeight: pageH,
		MarginTop: margin, MarginBottom: margin, MarginLeft: margin, MarginRight: margin,
	}); err != nil {
		t.Fatal(err)
	}

	return res, contentH
}

func findFixture60Table(t *testing.T, res *Result) *box {
	t.Helper()

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

	return table
}

func assertContinuationRowsHaveNoPaintGap(t *testing.T, table *box, res *Result, contentH float64) {
	t.Helper()

	seen := map[int]bool{}

	for rowIdx := table.headerRows; rowIdx+1 < len(table.rows); rowIdx++ {
		top, bot, page, first := firstContinuationRowPage(table, res, rowIdx, contentH, seen)
		if !first {
			continue
		}

		assertNoPaintGap(t, table, res, rowIdx, page, top, bot, contentH)
	}

	if len(seen) < 3 {
		t.Fatalf("expected several continuation pages, got %d", len(seen))
	}
}

func firstContinuationRowPage(
	table *box,
	res *Result,
	rowIdx int,
	contentH float64,
	seen map[int]bool,
) (float64, float64, int, bool) {
	_, _, top, bot, ok := rowPaintBand(table.rows[rowIdx], res)
	if !ok {
		return 0, 0, 0, false
	}

	page := int(top / contentH)
	if page == 0 || seen[page] {
		return 0, 0, 0, false
	}

	seen[page] = true

	return top, bot, page, true
}

func assertNoPaintGap(
	t *testing.T,
	table *box,
	res *Result,
	rowIdx, page int,
	top, bot, contentH float64,
) {
	t.Helper()

	nextFirst, nextLast, nextTop, _, nextOK := rowPaintBand(table.rows[rowIdx+1], res)
	_ = nextFirst
	_ = nextLast

	if !nextOK {
		t.Fatalf("page %d: second body row has no paint band", page)
	}

	if int(nextTop/contentH) != page {
		return
	}

	gap := nextTop - bot
	if gap > 0.75 {
		t.Fatalf("page %d: paint gap %.3fpt between first body row %d and next (top=%.2f bot=%.2f nextTop=%.2f)",
			page, gap, rowIdx, top, bot, nextTop)
	}

	if gap < -0.75 {
		t.Fatalf("page %d: paint overlap %.3fpt between first body row %d and next", page, gap, rowIdx)
	}
}

func TestRowPaintBandPrefersVerticalRules(t *testing.T) {
	t.Parallel()

	res := &Result{ //nolint:exhaustruct
		Ops: []Op{
			{Kind: OpText, X: 10, Y: 100, H: 12, Text: "a"}, //nolint:exhaustruct // band probe
			{Kind: OpLine, X: 0, Y: 90, W: 0, H: 40},        //nolint:exhaustruct // vertical rule
			{Kind: OpLine, X: 50, Y: 90, W: 0, H: 40},       //nolint:exhaustruct // vertical rule
			{Kind: OpText, X: 10, Y: 105, H: 10, Text: "b"}, //nolint:exhaustruct // band probe
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

	res, table, _ := layoutFixture60(t)

	assertBottomRowsSealed(t, table, res)
}

func assertBottomRowsSealed(t *testing.T, table *box, res *Result) {
	t.Helper()

	for _, want := range []string{"33", "67"} {
		assertBottomRowSealed(t, table, res, want)
	}
}

func assertBottomRowSealed(t *testing.T, table *box, res *Result, want string) {
	t.Helper()

	rowIdx := findRowWithText(table, res, want)
	if rowIdx < 0 {
		t.Fatalf("idx %s not found", want)
	}

	bandFirst, bandLast, bandTop, bot, ok := rowPaintBand(table.rows[rowIdx], res)
	_ = bandFirst
	_ = bandLast
	_ = bandTop

	if !ok {
		t.Fatalf("idx %s: no paint band", want)
	}

	if !hasFullWidthSeal(res, bot) {
		t.Fatalf("idx %s: missing full-width bottom seal at y=%.2f", want, bot)
	}
}

func findRowWithText(table *box, res *Result, want string) int {
	for rowIdx, row := range table.rows {
		first, last, _, _, ok := rowPaintBand(row, res)
		if !ok {
			continue
		}

		for i := first; i <= last && i < len(res.Ops); i++ {
			if res.Ops[i].Kind == OpText && res.Ops[i].Text == want {
				return rowIdx
			}
		}
	}

	return -1
}

func hasFullWidthSeal(res *Result, bot float64) bool {
	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpLine || paintOp.H > 0.01 || paintOp.W < 400 {
			continue
		}

		if math.Abs(paintOp.Y-bot) <= 1.0 {
			return true
		}
	}

	return false
}
