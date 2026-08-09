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
