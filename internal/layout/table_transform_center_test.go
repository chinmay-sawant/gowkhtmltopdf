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

	var rows string
	for i := 0; i < 18; i++ {
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

	doc, err := html.Parse(`<html><body style="margin:0"><table style="border-collapse:collapse;width:500px">` +
		rows + `</table></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	const pageH = 400.0
	const margin = 36.0
	contentH := pageH - 2*margin

	res, err := Layout(doc, Options{Width: 560, Height: pageH, Background: true})
	if err != nil {
		t.Fatal(err)
	}

	paginateOps(res, contentH)

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

	restampBoxTransforms(res.root, res.Ops)
	chip = findXformChipFill(res, "sc")
	if chip == nil || !chip.XformSet {
		t.Fatal("scale chip lost transform stamp after restamp")
	}
	chipMidX = chip.X + chip.W/2
	chipMidY = chip.Y + chip.H/2
	tx, ty := chip.Xform.Apply(chipMidX, chipMidY)
	stage = findXformStageFill(res)
	if stage == nil {
		t.Fatal("missing stage after restamp")
	}
	stageMidY := stage.Y + stage.H/2

	t.Logf("staleDrift=%.2f afterRestamp layout=(%.1f,%.1f) xformed=(%.1f,%.1f)",
		staleDrift, chipMidX, chipMidY, tx, ty)

	if staleDrift < 0.5 {
		t.Fatalf("expected stale scale origin drift after pagination, got %.3f (test not stressing the bug)", staleDrift)
	}

	const edgePad = 4.0
	if tx < stage.X+edgePad || tx > stage.X+stage.W-edgePad ||
		ty < stage.Y+edgePad || ty > stage.Y+stage.H-edgePad {
		t.Fatalf("scale chip xformed center (%.1f,%.1f) outside stage after restamp", tx, ty)
	}
	if math.Abs(ty-stageMidY) > stage.H*0.35 {
		t.Fatalf("scale chip midY=%.1f far from stage midY=%.1f after restamp", ty, stageMidY)
	}
	if math.Hypot(tx-chipMidX, ty-chipMidY) > 1.0 {
		t.Fatalf("restamped scale should keep center fixed, got delta=(%.2f,%.2f)", tx-chipMidX, ty-chipMidY)
	}
}

func assertChipInsideCenteredStage(t *testing.T, chipStyle, label string, fillers int) {
	t.Helper()

	var fillerRows string
	for i := 0; i < fillers; i++ {
		fillerRows += `<tr><td style="border:1px solid #ccc;padding:6px">f</td>` +
			`<td style="border:1px solid #ccc;padding:6px">x</td></tr>`
	}

	src := `<html><body style="margin:0">
<table style="border-collapse:collapse;table-layout:fixed;width:500px">
` + fillerRows + `
<tr>
<td style="border:1px solid #ccc;width:300px;padding:6px;vertical-align:top">desc</td>
<td style="border:1px solid #ccc;width:180px;padding:6px;vertical-align:top;background:#eef">
<div style="padding:20px 24px;text-align:center;border:1px dashed #888;background:#f7f7f7">
<div style="` + chipStyle + `;background:#fd8;padding:6px 8px;display:inline-block;border:1px solid #a60">` + label + `</div>
</div>
</td>
</tr>
</table>
</body></html>`

	doc, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(doc, Options{Width: 560, Height: 800, Background: true})
	if err != nil {
		t.Fatal(err)
	}

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

	stageMidX := stage.X + stage.W/2
	stageMidY := stage.Y + stage.H/2
	chipMidX := chip.X + chip.W/2
	chipMidY := chip.Y + chip.H/2
	tx, ty := chip.Xform.Apply(chipMidX, chipMidY)

	const edgePad = 4.0
	if tx < stage.X+edgePad || tx > stage.X+stage.W-edgePad ||
		ty < stage.Y+edgePad || ty > stage.Y+stage.H-edgePad {
		t.Fatalf("transformed chip center (%.1f,%.1f) outside stage", tx, ty)
	}
	if math.Abs(chipMidX-stageMidX) > stage.W*0.25 {
		t.Fatalf("chip layout midX=%.1f far from stage midX=%.1f", chipMidX, stageMidX)
	}
	if math.Abs(ty-stageMidY) > stage.H*0.35 {
		t.Fatalf("transformed chip midY=%.1f far from stage midY=%.1f", ty, stageMidY)
	}
}

func findXformStageFill(res *Result) *Op {
	var best *Op
	for i := range res.Ops {
		op := &res.Ops[i]
		if op.Kind != OpFillRect || op.XformSet {
			continue
		}
		if op.R < 0.95 || op.G < 0.95 || op.B < 0.95 || op.W < 40 || op.H < 30 {
			continue
		}
		if best == nil || op.W*op.H > best.W*best.H {
			best = op
		}
	}

	return best
}

func findXformChipFill(res *Result, label string) *Op {
	var textOp *Op
	for i := range res.Ops {
		op := &res.Ops[i]
		if op.Kind == OpText && op.Text == label {
			textOp = op
			break
		}
	}
	if textOp == nil {
		return nil
	}

	for i := range res.Ops {
		op := &res.Ops[i]
		if op.Kind != OpFillRect || !op.XformSet {
			continue
		}
		if op.R < 0.9 || op.G < 0.7 || op.G > 0.95 {
			continue
		}
		if math.Abs(op.Y-textOp.Y) < 40 && math.Abs(op.X-textOp.X) < 40 {
			return op
		}
	}

	return textOp
}
