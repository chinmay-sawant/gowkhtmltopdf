package layout

import (
	"strings"
	"testing"
)

// audit2 flex percentage columns. Each test asserts painted text geometry
// (whole words, column x positions), not box internals.

// learn-cpp-org-3: Bootstrap .row > .col-3 + .col-9. The .col-3 item carries
// flex: 0 0 25% plus max-width: 25%. Its "Output" heading must paint as a
// single whole word in a 25-percent column. Before the fix the percent
// max-width was applied a second time against the item's already-assigned
// flex width, so the column collapsed to 6.25 percent ("Outp" / "ut").
func TestFlexPercentColumnKeepsWordWhole(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-family: sans-serif; font-size: 10.86pt }
.row { display: flex; flex-wrap: wrap }
.col-3 { flex: 0 0 25%; max-width: 25% }
.col-9 { flex: 0 0 75%; max-width: 75%; text-align: right }
`)

	res := layoutHTML(t,
		`<html><body><div class="row">`+
			`<div class="col-3"><h5>Output</h5></div>`+
			`<div class="col-9">Expected Output</div></div></body></html>`,
		cssSheet)

	// The h5 sits at the column's left edge. Its whole text is "Output"; the
	// wrap bug split it into fragments at the same x on two lines.
	var h5Runs []Op

	col9Right := 0.0

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}

		if paintOp.X < 50 {
			h5Runs = append(h5Runs, paintOp)
		} else if strings.Contains(paintOp.Text, "Expected") {
			col9Right = paintOp.X + paintOp.W
		}
	}

	if len(h5Runs) != 1 || strings.TrimSpace(h5Runs[0].Text) != "Output" {
		texts := make([]string, 0, len(h5Runs))
		for _, run := range h5Runs {
			texts = append(texts, run.Text)
		}

		t.Fatalf("h5 text layer runs = %q, want one run \"Output\" (the 25%% column "+
			"must not collapse)", texts)
	}

	// The .col-9 text is right-aligned to the row's right edge (500pt). A
	// collapsed col-9 (75% of 75%) would end near x=281.
	if col9Right < 495 {
		t.Fatalf("col-9 text right edge = %.2f, want about 500 (row right edge); "+
			"the percent max-width was applied twice", col9Right)
	}
}

// learn-cpp-org-3 control: .col-2/.col-10 and .col-4/.col-8 resolve the same
// percentage bases. The right column starts where the left column ends.
func TestFlexPercentColumnBases(t *testing.T) {
	t.Parallel()

	cases := []struct {
		left, right       string
		wantRightFraction float64
	}{
		{"col-2", "col-10", 0.16666667},
		{"col-4", "col-8", 0.33333333},
	}

	for _, testCase := range cases {
		t.Run(testCase.left, func(t *testing.T) {
			t.Parallel()

			cssSheet := sheet(t, `
body { margin: 0; font-family: sans-serif; font-size: 12pt }
.row { display: flex; flex-wrap: wrap }
.col-2 { flex: 0 0 16.666667%; max-width: 16.666667% }
.col-10 { flex: 0 0 83.333333%; max-width: 83.333333% }
.col-4 { flex: 0 0 33.333333%; max-width: 33.333333% }
.col-8 { flex: 0 0 66.666667%; max-width: 66.666667% }
`)

			res := layoutHTML(t,
				`<html><body><div class="row">`+
					`<div class="`+testCase.left+`"><span>L</span></div>`+
					`<div class="`+testCase.right+`"><span>R</span></div></div></body></html>`,
				cssSheet)

			wantX := testViewport * testCase.wantRightFraction
			foundR := false

			for _, paintOp := range res.Ops {
				if paintOp.Kind != OpText {
					continue
				}

				if strings.TrimSpace(paintOp.Text) == "R" {
					foundR = true

					if diff := paintOp.X - wantX; diff > 2 || diff < -2 {
						t.Fatalf("%s text x=%.2f, want %.2f (column start = "+
							"percent of the row)", testCase.right, paintOp.X, wantX)
					}
				}
			}

			if !foundR {
				t.Fatal("no right-column text op")
			}
		})
	}
}
