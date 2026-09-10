package imageout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
)

// TestScaledRadiiXYParity pins imageout's raster radii to layout.OpRadiiXY
// plus pxPerPt scaling for the three op shapes the raster painter sees:
// uniform shorthand, per-corner longhands, and a uniform Y-only radius. The
// resolver lives in layout; this test fails if imageout grows a second copy
// with different fallback rules.
func TestScaledRadiiXYParity(t *testing.T) {
	t.Parallel()

	const pxPerPt = 2.0

	tests := []struct {
		name         string
		op           layout.Op
		wantX, wantY [4]float64
	}{
		{
			name:  "uniform shorthand copies X to Y",
			op:    layout.Op{Radius: 8},
			wantX: [4]float64{8, 8, 8, 8},
			wantY: [4]float64{8, 8, 8, 8},
		},
		{
			name: "corner longhands keep both axes",
			op: layout.Op{
				RadiusTopLeft: 10, RadiusTopLeftY: 5,
				RadiusTopRight: 10, RadiusTopRightY: 5,
				RadiusBottomRight: 10, RadiusBottomRightY: 5,
				RadiusBottomLeft: 10, RadiusBottomLeftY: 5,
			},
			wantX: [4]float64{10, 10, 10, 10},
			wantY: [4]float64{5, 5, 5, 5},
		},
		{
			name:  "uniform Y pairs with uniform X",
			op:    layout.Op{Radius: 8, RadiusY: 4},
			wantX: [4]float64{8, 8, 8, 8},
			wantY: [4]float64{4, 4, 4, 4},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			gotX, gotY := scaledRadiiXY(&testCase.op, pxPerPt)
			refX, refY := layout.OpRadiiXY(&testCase.op)

			for corner := range testCase.wantX {
				wantScaledX := testCase.wantX[corner] * pxPerPt
				wantScaledY := testCase.wantY[corner] * pxPerPt

				if gotX[corner] != wantScaledX || gotY[corner] != wantScaledY {
					t.Errorf("corner %d scaled radii = %v/%v, want %v/%v",
						corner, gotX[corner], gotY[corner], wantScaledX, wantScaledY)
				}

				if gotX[corner] != refX[corner]*pxPerPt || gotY[corner] != refY[corner]*pxPerPt {
					t.Errorf("corner %d diverges from layout.OpRadiiXY: got %v/%v, want %v/%v scaled",
						corner, gotX[corner], gotY[corner], refX[corner]*pxPerPt, refY[corner]*pxPerPt)
				}
			}
		})
	}
}
