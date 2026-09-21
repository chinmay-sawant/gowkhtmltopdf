//nolint:wsl,nlreturn,varnamelen,lll // white-box geometry regression tests
package layout

import (
	"math"
	"strings"
	"testing"
)

func TestFixture21ParagraphAfterForcedBreakStaysContiguous(t *testing.T) {
	t.Parallel()

	res, contentH := paintGoldenFixture(t, "fixture-21-detailed-report.html")
	needles := []string{"The HMI software team", "release 3.1.2.", "support procedures."}
	var ys []float64
	var pages []int
	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}
		for _, needle := range needles {
			if strings.Contains(op.Text, needle) {
				ys = append(ys, op.Y)
				pages = append(pages, int(op.Y/contentH))
				break
			}
		}
	}
	if len(ys) != len(needles) {
		t.Fatalf("paragraph lines = %d, want %d: ys=%v", len(ys), len(needles), ys)
	}
	for i := 1; i < len(pages); i++ {
		if pages[i] != pages[0] {
			t.Fatalf("second resource paragraph split across pages: pages=%v ys=%v", pages, ys)
		}
		if ys[i]-ys[i-1] > 2*1.2*9.5+1 {
			t.Fatalf("second resource paragraph has excess vertical gap: ys=%v", ys)
		}
	}
}

// Work-package table continues past page 1; thead must repeat on page 2 top,
// not get suffix-shifted back onto page 1 bottom by a later page-break-before.
func TestFixture21WorkPackageHeaderRepeatsOnContinuationPage(t *testing.T) { //nolint:cyclop
	t.Parallel()

	res, contentH := paintGoldenFixture(t, "fixture-21-detailed-report.html")

	var titleYs []float64
	var wp09Y float64
	var foundWP09 bool
	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}
		switch {
		case op.Text == "Title":
			titleYs = append(titleYs, op.Y)
		case op.Text == "WP-09":
			wp09Y = op.Y
			foundWP09 = true
		}
	}
	if !foundWP09 {
		t.Fatal("missing WP-09 body row")
	}
	if len(titleYs) < 2 {
		t.Fatalf("WP Title header occurrences = %d, want ≥2 (original + continuation)", len(titleYs))
	}

	wp09Page := int(wp09Y / contentH)
	if wp09Page < 1 {
		t.Fatalf("WP-09 on page %d, want continuation page ≥1 (y=%.2f)", wp09Page, wp09Y)
	}

	var contTitleY float64
	var foundCont bool
	for _, y := range titleYs {
		page := int(y / contentH)
		local := y - float64(page)*contentH
		// Bogus regression: clone dragged to page-0 bottom under body rows.
		if page == 0 && local > contentH-40 {
			t.Fatalf("thead clone landed on page-1 bottom at y=%.2f (local=%.2f); want continuation page top", y, local)
		}
		if page == wp09Page && local < 40 {
			if !foundCont || y < contTitleY {
				contTitleY = y
				foundCont = true
			}
		}
	}
	if !foundCont {
		t.Fatalf("no Title header near top of WP-09 page %d; titleYs=%v wp09Y=%.2f", wp09Page, titleYs, wp09Y)
	}
	if wp09Y < contTitleY+4 {
		t.Fatalf("WP-09 y=%.2f overlaps continuation header y=%.2f", wp09Y, contTitleY)
	}
}

func TestFixture23RepeatedHeaderHasNoVisualGap(t *testing.T) {
	t.Parallel()

	res, contentH := paintGoldenFixture(t, "fixture-23-thead-repeat.html")
	var table *box
	var find func(*box)
	find = func(b *box) {
		if b.kind == boxKindTable && table == nil {
			table = b
		}
		for _, child := range b.children {
			find(child)
		}
	}
	find(res.root)
	if table == nil || len(table.rows) < 38 || len(table.rows[0]) == 0 {
		t.Fatal("fixture-23 table rows missing")
	}

	headerBottom := table.rows[0][0].y + table.rows[0][0].height
	bodyTop := rowYBounds(table.rows[37], res)
	wantBodyTop := contentH + table.rows[0][0].height
	if math.Abs(bodyTop-wantBodyTop) > 0.5 {
		t.Fatalf("continuation body starts %.2fpt from repeated header band: body=%.2f want=%.2f header cell band=%.2f..%.2f", bodyTop-wantBodyTop, bodyTop, wantBodyTop, table.rows[0][0].y, headerBottom)
	}
}

func TestFixture64RepeatedHeaderKeepsCollapsedGridOnContinuationPage(t *testing.T) {
	t.Parallel()

	res, contentH := paintGoldenFixture(t, "fixture-64-next-72-props.html")
	tables := tableBoxes(res.root)
	if len(tables) != 1 {
		t.Fatalf("fixture-64 tables = %d, want 1", len(tables))
	}

	_, _, headerTop, headerH := rowSpan(tables[0].rows[:tables[0].headerRows], res)
	headerGrid := fixture64HeaderGrid(res, headerTop, headerH)
	if headerGrid == nil {
		t.Fatal("fixture-64 has no original header grid")
	}
	headerFills := fixture64HeaderFills(res, headerTop, headerH)
	if len(headerFills) == 0 {
		t.Fatal("fixture-64 has no original gray header fills")
	}

	assertFixture64RepeatedHeaderGrid(t, res, contentH, headerGrid, headerFills)
	assertFixture64ContinuationBottomRule(t, res, contentH, *headerGrid)
}

func TestFixture64TextAutospaceAddsIdeographAlphaGap(t *testing.T) {
	t.Parallel()

	res, _ := paintGoldenFixture(t, "fixture-64-next-72-props.html")
	var runs []Op
	for _, op := range res.Ops {
		if op.Kind == OpText && op.Text == "汉A汉A汉" {
			runs = append(runs, op)
		}
	}
	if len(runs) != 2 {
		t.Fatalf("fixture-64 text-autospace runs = %d, want 2: %+v", len(runs), runs)
	}
	if runs[0].TextAutospaceGap() != 0 || runs[1].TextAutospaceGap() <= 0 {
		t.Fatalf("fixture-64 text-autospace gaps = %.2f/%.2f, want 0/positive",
			runs[0].TextAutospaceGap(), runs[1].TextAutospaceGap())
	}

	if runs[1].W <= runs[0].W+1 {
		t.Fatalf("fixture-64 ideograph-alpha width = %.2f, no-autospace width = %.2f; want a visible gap", runs[1].W, runs[0].W)
	}
}

func assertFixture64RepeatedHeaderGrid(t *testing.T, res *Result, contentH float64, headerGrid *Op, headerFills []Op) {
	t.Helper()

	var foundContinuationGrid, foundContinuationFill bool
	for _, paintOp := range res.Ops {
		if !paintOp.Pinned {
			continue
		}

		page, ok := checkedFlowPageOfY(paintOp.Y, contentH)
		if !ok || page < 1 {
			continue
		}

		gridFound, fillFound := assertFixture64ContinuationOp(t, paintOp, contentH, page, *headerGrid, headerFills)
		foundContinuationGrid = foundContinuationGrid || gridFound
		foundContinuationFill = foundContinuationFill || fillFound
	}

	if !foundContinuationGrid {
		t.Fatal("fixture-64 has no repeated header grid on a continuation page")
	}
	if !foundContinuationFill {
		t.Fatal("fixture-64 has no repeated gray header fill on a continuation page")
	}
}

func assertFixture64ContinuationOp(t *testing.T, paintOp Op, contentH float64, page int, headerGrid Op, headerFills []Op) (bool, bool) {
	t.Helper()
	if paintOp.Kind == OpGridRun && paintOp.Grid != nil {
		assertFixture64GridGeometry(t, paintOp, headerGrid, page, contentH)

		return true, false
	}
	if paintOp.Kind != OpFillRect {
		return false, false
	}
	if !fixture64HasMatchingFill(headerFills, paintOp) {
		t.Fatalf("fixture-64 repeated gray header fill has mismatched left/width: x=%.2f w=%.2f", paintOp.X, paintOp.W)
	}

	return false, true
}

func fixture64HeaderGrid(res *Result, headerTop, headerH float64) *Op {
	for idx := range res.Ops {
		paintOp := &res.Ops[idx]
		if paintOp.Pinned || paintOp.Kind != OpGridRun || paintOp.Grid == nil {
			continue
		}
		if paintOp.Y >= headerTop-layoutEpsilon && paintOp.Y+paintOp.H <= headerTop+headerH+layoutEpsilon {
			return paintOp
		}
	}

	return nil
}

func fixture64HeaderFills(res *Result, headerTop, headerH float64) []Op {
	fills := make([]Op, 0, 4)
	for _, paintOp := range res.Ops {
		if paintOp.Pinned || paintOp.Kind != OpFillRect {
			continue
		}
		if paintOp.Y >= headerTop-layoutEpsilon && paintOp.Y+paintOp.H <= headerTop+headerH+layoutEpsilon {
			fills = append(fills, paintOp)
		}
	}

	return fills
}

func assertFixture64GridGeometry(t *testing.T, got, want Op, page int, contentH float64) {
	t.Helper()
	if math.Abs(got.X-want.X) > layoutEpsilon || math.Abs(got.W-want.W) > layoutEpsilon {
		t.Fatalf("fixture-64 repeated header grid left/width = %.2f/%.2f, want %.2f/%.2f", got.X, got.W, want.X, want.W)
	}
	if len(got.Grid.Segs) != len(want.Grid.Segs) {
		t.Fatalf("fixture-64 repeated header grid segments = %d, want %d", len(got.Grid.Segs), len(want.Grid.Segs))
	}

	pageTop := float64(page) * contentH
	for segIdx, seg := range got.Grid.Segs {
		wantSeg := want.Grid.Segs[segIdx]
		if seg.Y < pageTop-layoutEpsilon {
			t.Fatalf("fixture-64 repeated header grid segment %d stayed at y=%.2f for page %d; page top=%.2f", segIdx, seg.Y, page+1, pageTop)
		}
		if math.Abs(seg.X-wantSeg.X) > layoutEpsilon || math.Abs(seg.W-wantSeg.W) > layoutEpsilon {
			t.Fatalf("fixture-64 repeated header segment %d left/width = %.2f/%.2f, want %.2f/%.2f", segIdx, seg.X, seg.W, wantSeg.X, wantSeg.W)
		}
	}
}

func assertFixture64ContinuationBottomRule(t *testing.T, res *Result, contentH float64, headerGrid Op) {
	t.Helper()

	bottomY := fixture64ContinuationBottomY(res, contentH, headerGrid)
	if bottomY < 0 {
		t.Fatal("fixture-64 has no last body-row border band on page 2")
	}
	if fixture64HasClosingBottomRule(res, contentH, headerGrid, bottomY) {
		return
	}

	t.Fatalf("fixture-64 page-2 last body row has no closing bottom rule at y=%.2f", bottomY)
}

func fixture64ContinuationBottomY(res *Result, contentH float64, headerGrid Op) float64 {
	bottomY := -1.0
	for _, paintOp := range res.Ops {
		if !fixture64BodyGridOnPage(paintOp, contentH, headerGrid) {
			continue
		}
		for _, segment := range paintOp.Grid.Segs {
			if segment.W < 1 && segment.H > 2 {
				bottomY = math.Max(bottomY, segment.Y+segment.H)
			}
		}
	}

	return bottomY
}

func fixture64BodyGridOnPage(paintOp Op, contentH float64, headerGrid Op) bool {
	if paintOp.Pinned || paintOp.Kind != OpGridRun || paintOp.Grid == nil {
		return false
	}

	page, ok := checkedFlowPageOfY(paintOp.Y, contentH)
	return ok && page == 1 &&
		math.Abs(paintOp.X-headerGrid.X) <= layoutEpsilon &&
		math.Abs(paintOp.W-headerGrid.W) <= layoutEpsilon
}

func fixture64HasClosingBottomRule(res *Result, contentH float64, headerGrid Op, bottomY float64) bool {
	for _, paintOp := range res.Ops {
		if paintOp.Fixed || paintOp.Kind != OpLine || paintOp.H >= 1 {
			continue
		}
		page, ok := checkedFlowPageOfY(paintOp.Y, contentH)
		if !ok || page != 1 {
			continue
		}
		if math.Abs(paintOp.Y-bottomY) <= 0.5 &&
			math.Abs(paintOp.X-headerGrid.X) <= layoutEpsilon &&
			math.Abs(paintOp.W-headerGrid.W) <= layoutEpsilon {
			return true
		}
	}

	return false
}

func fixture64HasMatchingFill(headerFills []Op, got Op) bool {
	for _, want := range headerFills {
		if math.Abs(got.X-want.X) <= layoutEpsilon &&
			math.Abs(got.W-want.W) <= layoutEpsilon &&
			math.Abs(got.H-want.H) <= layoutEpsilon &&
			math.Abs(got.R-want.R) <= layoutEpsilon &&
			math.Abs(got.G-want.G) <= layoutEpsilon &&
			math.Abs(got.B-want.B) <= layoutEpsilon {
			return true
		}
	}

	return false
}

func TestFixture28FlexWrapGridItemsStayInFirstPageLayout(t *testing.T) {
	t.Parallel()

	res, contentH := paintGoldenFixture(t, "fixture-28-flex-wrap-grid-fixed.html")
	labels := []string{"A1", "A2", "A3", "A4", "G1", "G2", "G3", "G4"}
	positions := make(map[string]float64, len(labels))
	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}
		for _, label := range labels {
			if op.Text == label {
				positions[label] = op.Y
			}
		}
	}
	for _, label := range labels {
		y, ok := positions[label]
		if !ok {
			t.Fatalf("fixture-28 missing text %q", label)
		}
		if y >= contentH {
			t.Fatalf("fixture-28 label %q moved to page %d at y=%.2f", label, int(y/contentH), y)
		}
	}
	minY, maxY := positions[labels[0]], positions[labels[0]]
	for _, label := range labels[1:] {
		minY = math.Min(minY, positions[label])
		maxY = math.Max(maxY, positions[label])
	}
	if maxY-minY > 100 {
		t.Fatalf("fixture-28 wrapped/grid labels are vertically separated: positions=%v", positions)
	}
}

//nolint:cyclop // fixture assertions follow the fixture's request order
func TestFixture43CardsAndTheadDoNotOverlap(t *testing.T) {
	t.Parallel()

	res, contentH := paintGoldenFixture(t, "fixture-43-complex-dossier.html")
	needles := []string{"Northstar Atlas", "Product Launch Dossier"}
	var found []string
	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}
		for _, needle := range needles {
			if strings.Contains(op.Text, needle) {
				found = append(found, needle)
				if op.Y < 0 {
					t.Fatalf("fixture-43 %q painted at negative y=%.2f", needle, op.Y)
				}
			}
		}
	}
	if len(found) == 0 {
		t.Fatal("fixture-43 missing dossier title text")
	}

	var table *box
	var find func(*box)
	find = func(b *box) {
		if b == nil {
			return
		}
		if b.kind == boxKindTable && table == nil && len(b.rows) > 4 {
			table = b
		}
		for _, child := range b.children {
			find(child)
		}
	}
	find(res.root)
	if table == nil {
		return
	}
	headerBottom := table.rows[0][0].y + table.rows[0][0].height
	for _, row := range table.rows[1:] {
		if len(row) == 0 {
			continue
		}
		if row[0].y+0.5 < headerBottom && row[0].y < contentH {
			t.Fatalf("fixture-43 body row overlaps thead: y=%.2f headerBottom=%.2f", row[0].y, headerBottom)
		}
		break
	}
}

// Fixture-64 page 7: a sliver repair on page 6 dragged the whole following
// flow down by 63.65pt, and row 63 stayed that far under the repeated header
// because nothing pulls a row back up across a page boundary.
func TestFixture64ContinuationRowsStartAtHeaderBottom(t *testing.T) {
	t.Parallel()

	res, contentH := paintGoldenFixture(t, "fixture-64-next-72-props.html")

	table := fixture64BigTable(res)
	if table == nil {
		t.Fatal("fixture-64 table not found")
	}

	hdrFirst, hdrLast, _, hdrH := rowSpan(table.rows[:table.headerRows], res)
	if hdrFirst < 0 || hdrLast < hdrFirst || hdrH <= 0 {
		t.Fatalf("fixture-64 header band missing: first=%d last=%d h=%.2f", hdrFirst, hdrLast, hdrH)
	}

	firstTop := fixture64FirstBodyRowPerPage(table, res, contentH)
	if len(firstTop) < 5 {
		t.Fatalf("fixture-64 continuation pages = %d, want >= 5", len(firstTop))
	}

	for page, top := range firstTop {
		want := float64(page)*contentH + hdrH
		if top < want-0.75 || top > want+0.75 {
			t.Errorf("page %d: first body row top = %.2f, want repeated-header bottom %.2f", page+1, top, want)
		}
	}
}

func fixture64BigTable(res *Result) *box {
	for _, b := range flowBoxList(res) {
		if b.kind == boxKindTable && len(b.rows) > 60 && b.headerRows > 0 {
			return b
		}
	}

	return nil
}

// fixture64FirstBodyRowPerPage returns the topmost body row Y per page index,
// skipping page 0 (the table's first page starts below the masthead).
func fixture64FirstBodyRowPerPage(table *box, res *Result, contentH float64) map[int]float64 {
	firstTop := map[int]float64{}

	for _, row := range table.rows[table.headerRows:] {
		top := rowYBounds(row, res)
		if top < 0 {
			continue
		}

		page := int(top / contentH)
		if page == 0 {
			continue
		}

		if current, ok := firstTop[page]; !ok || top < current {
			firstTop[page] = top
		}
	}

	return firstTop
}
