//nolint:testpackage,wsl,nlreturn,varnamelen,lll // white-box geometry regression tests
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
		if ys[i]-ys[i-1] > 2*defaultLineHeightRatio*9.5+1 {
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
		if b.kind == displayTable && table == nil {
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
		if b.kind == displayTable && table == nil && len(b.rows) > 4 {
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
