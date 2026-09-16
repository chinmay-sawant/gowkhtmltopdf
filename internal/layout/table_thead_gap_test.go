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

func TestFixture60TheadRepeatsOnEveryContinuationPage(t *testing.T) {
	t.Parallel()

	res, table, contentH := layoutFixture60(t)
	headerPages := map[int]bool{}
	bodyPages := map[int]bool{}

	for _, op := range res.Ops {
		if op.Kind == OpText && op.Text == "Property" {
			headerPages[int(op.Y/contentH)] = true
		}
	}

	for _, row := range table.rows[table.headerRows:] {
		top := rowYBounds(row, res)
		if top >= 0 {
			bodyPages[int(top/contentH)] = true
		}
	}

	for page := range bodyPages {
		if page > 0 && !headerPages[page] {
			t.Errorf("continuation page %d has body rows but no repeated thead", page+1)
		}
	}
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

	res, err := Layout(doc, Options{
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
		if b.kind == boxKindTable && len(b.rows) > 100 {
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

	res := &Result{
		Ops: []Op{
			{Kind: OpText, X: 10, Y: 100, H: 12, Text: "a"},
			{Kind: OpLine, X: 0, Y: 90, W: 0, H: 40},
			{Kind: OpLine, X: 50, Y: 90, W: 0, H: 40},
			{Kind: OpText, X: 10, Y: 105, H: 10, Text: "b"},
		},
	}
	cell := &box{
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
// get a full-width bottom seal. Page-ending row indexes move whenever line
// breaking changes (fixture-60 pages previously ended at props 31 and 46), so
// derive them from the packed layout instead of pinning prop numbers.
func TestFixture60PageBottomRowsAreSealed(t *testing.T) {
	t.Parallel()

	res, table, contentH := layoutFixture60(t)

	lastByPage := map[int]int{}

	for rowIdx := table.headerRows; rowIdx+1 < len(table.rows); rowIdx++ {
		_, _, top, _, ok := rowPaintBand(table.rows[rowIdx], res)
		if !ok {
			continue
		}

		lastByPage[int(top/contentH)] = rowIdx
	}

	checked := 0

	for page, rowIdx := range lastByPage {
		_, _, _, bot, ok := rowPaintBand(table.rows[rowIdx], res)
		if !ok {
			continue
		}

		_, _, nextTop, _, nextOK := rowPaintBand(table.rows[rowIdx+1], res)
		if !nextOK || int(nextTop/contentH) == page {
			continue // the next row stays on this page: no bottom seal expected
		}

		if !hasFullWidthSeal(res, bot) {
			t.Fatalf("page %d bottom row idx %d: missing full-width seal at y=%.2f", page, rowIdx, bot)
		}

		checked++
	}

	if checked < 2 {
		t.Fatalf("checked %d page-bottom rows, want at least 2", checked)
	}
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
