package layout

import "testing"

// TestCapTablePageBreaksSkipsTransformedRails is the fixture-62 prop 63
// regression: a rotated chip's border rails (XformSet) must not merge with the
// parent stage's dashed border fragments in a page-bottom seal cluster.
//
// The stage's two dashed side rails end 0.075pt before the chip's two solid
// side rails. Both land in one roundY bucket, so before the collector skipped
// transformed ops the cluster had n=4, span=190 and sealPageBottomClusters
// appended a solid full-span line at the chip's rail end. The line painted
// axis-aligned because seal ops are appended after transform stamping.
func TestCapTablePageBreaksSkipsTransformedRails(t *testing.T) {
	t.Parallel()

	const contentH = 200.0

	stageGray := 0.533
	chipBrownR, chipBrownG := 0.667, 0.4

	res := &Result{}
	res.Ops = []Op{
		// Stage dashed side rails: fragments ending at y=300.15.
		{Kind: OpLine, X: 10, Y: 297.9, W: 0, H: 2.25, Width: 0.75, R: stageGray, G: stageGray, B: stageGray},
		{Kind: OpLine, X: 200, Y: 297.9, W: 0, H: 2.25, Width: 0.75, R: stageGray, G: stageGray, B: stageGray},
		// Rotated chip rails: same end bucket, but painted under an Xform.
		{Kind: OpLine, X: 80, Y: 276.35, W: 0, H: 23.8, Width: 0.75, R: chipBrownR, G: chipBrownG, B: 0, XformSet: true},
		{Kind: OpLine, X: 130, Y: 276.35, W: 0, H: 23.8, Width: 0.75, R: chipBrownR, G: chipBrownG, B: 0, XformSet: true},
		// Chip bottom border: the partial horizontal that keeps coverage false.
		{Kind: OpLine, X: 80, Y: 300.15, W: 50, H: 0, Width: 0.75, R: chipBrownR, G: chipBrownG, B: 0},
	}

	verts, _ := collectBorderSegmentOps(res.Ops)
	if len(verts) != 2 {
		t.Fatalf("collectBorderSegmentOps kept %d vertical segments, want 2 (XformSet rails must be skipped)", len(verts))
	}

	before := len(res.Ops)

	capTablePageBreaks(res, contentH)

	for _, paintOp := range res.Ops[before:] {
		if paintOp.Kind == OpLine && paintOp.H == 0 && paintOp.W > 100 {
			t.Fatalf(
				"capTablePageBreaks appended a full-span seal over transformed rails: X=%.2f Y=%.2f W=%.2f (%.3f,%.3f,%.3f)",
				paintOp.X, paintOp.Y, paintOp.W, paintOp.R, paintOp.G, paintOp.B)
		}
	}
}
