package layout

import (
	"math"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// TestTableCellTransformedChipStaysInStage locks the fixture-62 Effect-cell
// contract for rotate / scale / transform demos: a full-width centered stage
// keeps the transformed chip inside the stage after the stamped matrix is
// applied.
func TestTableCellTransformedChipStaysInStage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		style string
		label string
	}{
		{name: "rotate", style: "rotate:12deg", label: "rot"},
		{name: "scale", style: "scale:1.15", label: "sc"},
		{name: "transform", style: "transform:rotate(8deg) translate(4px,2px)", label: "xf"},
		{name: "transform-box", style: "transform-box:border-box;transform:rotate(8deg)", label: "box"},
		{name: "transform-origin", style: "transform-origin:left bottom;transform:rotate(10deg)", label: "orig"},
		{name: "translate", style: "translate:10px 4px", label: "mv"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assertChipInsideCenteredStage(t, tc.style, tc.label, 0)
		})
	}
}

// TestScaleTransformRestampAfterPagination is the fixture-62 #65 regression:
// pagination shifts box/op Y but must rebake transform origins, otherwise
// scale (and rotate) paint drifts out of the Effect stage on later pages.
func TestScaleTransformRestampAfterPagination(t *testing.T) {
	t.Parallel()

	const pageH = 400.0

	const margin = 36.0

	contentH := pageH - 2*margin

	doc := parseTestHTML(t, `<html><body style="margin:0"><table style="border-collapse:collapse;width:500px">`+
		scaleRestampRows()+`</table></body></html>`)

	res, err := Layout(doc, Options{
		Width:      560,
		Height:     pageH,
		Background: true,
	})

	if err != nil {
		t.Fatal(err)
	}

	if _, err := paginateOps(t.Context(), res, contentH); err != nil {
		t.Fatal(err)
	}

	staleDrift := assertScaleChipDriftBeforeRestamp(t, res)

	restampBoxTransforms(res.root, res.Ops)

	assertScaleChipRestamped(t, res, staleDrift)
}

// scaleRestampRows returns the fixture-62 #65 table body: 18 filler rows push
// the scale chip row far enough down that pagination shifts its box Y.
func scaleRestampRows() string {
	var rows string

	for range 18 {
		rows += `<tr><td style="border:1px solid #ccc;padding:8px">f</td>` +
			`<td style="border:1px solid #ccc;padding:8px;width:180px">x</td></tr>`
	}

	rows += `<tr>
<td style="border:1px solid #ccc;padding:6px;vertical-align:top">scale</td>
<td style="border:1px solid #ccc;padding:6px;vertical-align:top;width:180px">
<div style="padding:20px 24px;text-align:center;border:1px dashed #888;background:#f7f7f7">
<div style="scale:1.15;background:#fd8;padding:6px 8px;display:inline-block;border:1px solid #a60">sc</div>
</div>
</td></tr>`

	return rows
}

// assertScaleChipDriftBeforeRestamp locks the pre-rebake state: the chip is on
// the page with a stamped transform, and the painted center drift is returned
// to prove pagination shifted the scale origin.
func assertScaleChipDriftBeforeRestamp(t *testing.T, res *Result) float64 {
	t.Helper()

	stage := findXformStageFill(res)
	chip := findXformChipFill(res, "sc")

	if stage == nil || chip == nil {
		t.Fatal("missing stage or scale chip after pagination")
	}

	if !chip.XformSet {
		t.Fatal("scale chip missing transform stamp")
	}

	chipMidX := chip.X + chip.W/2
	chipMidY := chip.Y + chip.H/2
	staleX, staleY := chip.Xform.Apply(chipMidX, chipMidY)

	// Without rebake, a shifted scale origin drifts the painted center.
	staleDrift := math.Hypot(staleX-chipMidX, staleY-chipMidY)

	return staleDrift
}

// assertScaleChipRestamped locks the post-rebake contract: the chip keeps its
// transform stamp, its xformed center stays inside the stage, and the stale
// drift proves the test actually stressed the bug.
func assertScaleChipRestamped(t *testing.T, res *Result, staleDrift float64) {
	t.Helper()

	chip := findXformChipFill(res, "sc")

	if chip == nil || !chip.XformSet {
		t.Fatal("scale chip lost transform stamp after restamp")
	}

	chipMidX := chip.X + chip.W/2
	chipMidY := chip.Y + chip.H/2
	targetX, targetY := chip.Xform.Apply(chipMidX, chipMidY)

	stage := findXformStageFill(res)

	if stage == nil {
		t.Fatal("missing stage after restamp")
	}

	stageMidY := stage.Y + stage.H/2

	t.Logf("staleDrift=%.2f afterRestamp layout=(%.1f,%.1f) xformed=(%.1f,%.1f)",
		staleDrift, chipMidX, chipMidY, targetX, targetY)

	if staleDrift < 0.5 {
		t.Fatalf("expected stale scale origin drift after pagination, got %.3f (test not stressing the bug)", staleDrift)
	}

	assertRestampedBounds(t, targetX, targetY, chipMidX, chipMidY, stageMidY, stage)
}

func assertRestampedBounds(t *testing.T, targetX, targetY, chipMidX, chipMidY, stageMidY float64, stage *Op) {
	t.Helper()

	const edgePad = 4.0

	if targetX < stage.X+edgePad || targetX > stage.X+stage.W-edgePad ||
		targetY < stage.Y+edgePad || targetY > stage.Y+stage.H-edgePad {
		t.Fatalf("scale chip xformed center (%.1f,%.1f) outside stage after restamp", targetX, targetY)
	}

	if math.Abs(targetY-stageMidY) > stage.H*0.35 {
		t.Fatalf("scale chip midY=%.1f far from stage midY=%.1f after restamp", targetY, stageMidY)
	}

	if math.Hypot(targetX-chipMidX, targetY-chipMidY) > 1.0 {
		t.Fatalf("restamped scale should keep center fixed, got delta=(%.2f,%.2f)", targetX-chipMidX, targetY-chipMidY)
	}
}

func xformFillerRows(fillers int) string {
	var fillerRows string

	for range fillers {
		fillerRows += `<tr><td style="border:1px solid #ccc;padding:6px">f</td>` +
			`<td style="border:1px solid #ccc;padding:6px">x</td></tr>`
	}

	return fillerRows
}

func xformCenteredStageSrc(chipStyle, label, fillerRows string) string {
	return `<html><body style="margin:0">
<table style="border-collapse:collapse;table-layout:fixed;width:500px">
` + fillerRows + `
<tr>
<td style="border:1px solid #ccc;width:300px;padding:6px;vertical-align:top">desc</td>
<td style="border:1px solid #ccc;width:180px;padding:6px;vertical-align:top;background:#eef">
<div style="padding:20px 24px;text-align:center;border:1px dashed #888;background:#f7f7f7">
<div style="` + chipStyle + `;background:#fd8;padding:6px 8px;` +
		`display:inline-block;border:1px solid #a60">` + label + `</div>
</div>
</td>
</tr>
</table>
</body></html>`
}

func parseTestHTML(t *testing.T, src string) *html.Node {
	t.Helper()

	doc, err := html.Parse(src)

	if err != nil {
		t.Fatal(err)
	}

	return doc
}

func layoutXformStage(t *testing.T, src string) *Result {
	t.Helper()

	doc := parseTestHTML(t, src)

	res, err := Layout(doc, Options{
		Width:      560,
		Height:     800,
		Background: true,
	})

	if err != nil {
		t.Fatal(err)
	}

	return res
}

func requireChipCenterInStage(t *testing.T, stage, chip *Op, paintX, paintY float64) {
	t.Helper()

	stageMidX := stage.X + stage.W/2
	stageMidY := stage.Y + stage.H/2
	chipMidX := chip.X + chip.W/2

	const edgePad = 4.0

	if paintX < stage.X+edgePad || paintX > stage.X+stage.W-edgePad ||
		paintY < stage.Y+edgePad || paintY > stage.Y+stage.H-edgePad {
		t.Fatalf("transformed chip center (%.1f,%.1f) outside stage", paintX, paintY)
	}

	if math.Abs(chipMidX-stageMidX) > stage.W*0.25 {
		t.Fatalf("chip layout midX=%.1f far from stage midX=%.1f", chipMidX, stageMidX)
	}

	if math.Abs(paintY-stageMidY) > stage.H*0.35 {
		t.Fatalf("transformed chip midY=%.1f far from stage midY=%.1f", paintY, stageMidY)
	}
}

func assertChipInsideCenteredStage(t *testing.T, chipStyle, label string, fillers int) {
	t.Helper()

	fillerRows := xformFillerRows(fillers)
	src := xformCenteredStageSrc(chipStyle, label, fillerRows)
	res := layoutXformStage(t, src)

	stage := findXformStageFill(res)
	chip := findXformChipFill(res, label)

	if stage == nil {
		t.Fatal("missing xform-stage fill")
	}

	if chip == nil {
		t.Fatal("missing xform-chip fill")
	}

	if !chip.XformSet {
		t.Fatal("chip transform not stamped")
	}

	chipMidX := chip.X + chip.W/2
	chipMidY := chip.Y + chip.H/2
	paintX, paintY := chip.Xform.Apply(chipMidX, chipMidY)

	requireChipCenterInStage(t, stage, chip, paintX, paintY)
}

func isXformStageCandidate(fillOp *Op) bool {
	if fillOp.Kind != OpFillRect || fillOp.XformSet {
		return false
	}

	if fillOp.R < 0.95 || fillOp.G < 0.95 || fillOp.B < 0.95 || fillOp.W < 40 || fillOp.H < 30 {
		return false
	}

	return true
}

func findXformStageFill(res *Result) *Op {
	var best *Op

	for i := range res.Ops {
		fillOp := &res.Ops[i]

		if !isXformStageCandidate(fillOp) {
			continue
		}

		if best == nil || fillOp.W*fillOp.H > best.W*best.H {
			best = fillOp
		}
	}

	return best
}

func findXformLabelText(res *Result, label string) *Op {
	for i := range res.Ops {
		textOp := &res.Ops[i]

		if textOp.Kind == OpText && textOp.Text == label {
			return textOp
		}
	}

	return nil
}

func isXformChipCandidate(fillOp, textOp *Op) bool {
	if fillOp.Kind != OpFillRect || !fillOp.XformSet {
		return false
	}

	if fillOp.R < 0.9 || fillOp.G < 0.7 || fillOp.G > 0.95 {
		return false
	}

	if math.Abs(fillOp.Y-textOp.Y) >= 40 || math.Abs(fillOp.X-textOp.X) >= 40 {
		return false
	}

	return true
}

func findXformChipFill(res *Result, label string) *Op {
	textOp := findXformLabelText(res, label)

	if textOp == nil {
		return nil
	}

	for i := range res.Ops {
		fillOp := &res.Ops[i]

		if isXformChipCandidate(fillOp, textOp) {
			return fillOp
		}
	}

	return textOp
}
