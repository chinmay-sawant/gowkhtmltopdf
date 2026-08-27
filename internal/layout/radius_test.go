//nolint:testpackage,wsl // radius slash / elliptical longhand proofs
package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

func TestRadiusSlash(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="slash">x</div>
		<div class="circle">x</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.slash { border-radius: 10pt / 5pt }
		.circle { border-radius: 8pt }
	`)}, "print", testViewport, 800)

	slash := styleByClass(t, styles, "slash")
	assertCornerRadiusXY(t, slash.BorderRadiusTopLeft, slash.BorderRadiusTopLeftY, 10, 5, "slash TL")
	assertCornerRadiusXY(t, slash.BorderRadiusTopRight, slash.BorderRadiusTopRightY, 10, 5, "slash TR")
	assertCornerRadiusXY(t, slash.BorderRadiusBottomRight, slash.BorderRadiusBottomRightY, 10, 5, "slash BR")
	assertCornerRadiusXY(t, slash.BorderRadiusBottomLeft, slash.BorderRadiusBottomLeftY, 10, 5, "slash BL")

	circle := styleByClass(t, styles, "circle")
	if !near(circle.BorderRadius, 8) {
		t.Fatalf("circular shorthand BorderRadius = %.3f, want 8", circle.BorderRadius)
	}

	if !near(circle.BorderRadiusTopLeftY, 0) {
		t.Fatalf("circular shorthand Y = %.3f, want 0 (same as X at paint)", circle.BorderRadiusTopLeftY)
	}

	cssSheet := sheet(t, `
body { margin: 0 }
.slash { width: 40pt; height: 20pt; background: #f00; border-radius: 10pt / 5pt }
`)
	res := layoutHTML(t, `<html><body><div class="slash">x</div></body></html>`, cssSheet)
	fill := redFillOp(t, res.Ops)
	if !near(fill.RadiusTopLeft, 10) || !near(fill.RadiusTopLeftY, 5) {
		t.Fatalf("slash fill radii X=%.3f Y=%.3f, want 10 / 5", fill.RadiusTopLeft, fill.RadiusTopLeftY)
	}
}

func TestRadiusEllipticalLonghand(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="slash">x</div>
		<div class="space">x</div>
		<div class="round">x</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.slash { border-top-left-radius: 10pt / 5pt }
		.space { border-top-right-radius: 8pt 4pt }
		.round { border-bottom-right-radius: 6pt }
	`)}, "print", testViewport, 800)

	slash := styleByClass(t, styles, "slash")
	assertCornerRadiusXY(t, slash.BorderRadiusTopLeft, slash.BorderRadiusTopLeftY, 10, 5, "longhand slash")

	space := styleByClass(t, styles, "space")
	assertCornerRadiusXY(t, space.BorderRadiusTopRight, space.BorderRadiusTopRightY, 8, 4, "longhand space")

	round := styleByClass(t, styles, "round")
	if !near(round.BorderRadiusBottomRight, 6) {
		t.Fatalf("circular longhand X = %.3f, want 6", round.BorderRadiusBottomRight)
	}

	if !near(round.BorderRadiusBottomRightY, 0) {
		t.Fatalf("circular longhand Y = %.3f, want 0", round.BorderRadiusBottomRightY)
	}

	res := layoutHTML(t, `<html><body><div class="slash">x</div></body></html>`, sheet(t, `
body { margin: 0 }
.slash { width: 40pt; height: 20pt; background: #f00; border-top-left-radius: 10pt / 5pt }
`))
	fill := redFillOp(t, res.Ops)
	if !near(fill.RadiusTopLeft, 10) || !near(fill.RadiusTopLeftY, 5) {
		t.Fatalf("longhand fill radii X=%.3f Y=%.3f, want 10 / 5", fill.RadiusTopLeft, fill.RadiusTopLeftY)
	}
}

func assertCornerRadiusXY(t *testing.T, radiusX, radiusY, wantX, wantY float64, label string) {
	t.Helper()

	if !near(radiusX, wantX) || !near(radiusY, wantY) {
		t.Fatalf("%s radius = %.3f / %.3f, want %.3f / %.3f", label, radiusX, radiusY, wantX, wantY)
	}
}

func redFillOp(t *testing.T, ops []Op) Op {
	t.Helper()

	for _, op := range ops {
		if op.Kind == OpFillRect && op.R > 0.9 {
			return op
		}
	}

	t.Fatal("missing red fill op")

	return Op{} //nolint:exhaustruct // unreachable
}
