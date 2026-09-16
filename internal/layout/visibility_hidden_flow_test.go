package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// cplusplus.com #I_left sets visibility:hidden and height:0 but still occupied
// 1931.6pt of flow and painted its red/gray backgrounds (real-sites evidence
// 2026-09-16, cplusplus-tutorial rows 1-2). A hidden height:0 subtree must
// contribute no flow height and no ink.
func TestVisibilityHiddenHeightZeroSidebarCollapses(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
		body { margin: 10px; font-family: sans-serif; font-size: 12px }
		.left { background: #ff0000; visibility: hidden; height: 0 }
		.left .sect { background: #f8f8f8; border: 1px solid #abc }
	`)
	res := layoutHTML(t, `<html><body>
		<div class="left"><div class="sect"><p>HIDDEN-SIDEBAR-TEXT-ONE</p><p>HIDDEN-SIDEBAR-TEXT-TWO</p></div></div>
		<div style="background:#f0f0f0"><p>AFTER-LEFT</p></div>
	</body></html>`, cssSheet)

	left := findBoxByClass(t, res, "left")
	if !near(left.height, 0) {
		t.Fatalf("visibility:hidden height:0 sidebar height = %.2f, want 0", left.height)
	}

	if got := joinedText(res); strings.Contains(got, "HIDDEN-SIDEBAR") {
		t.Fatalf("hidden sidebar painted text: %q", got)
	}

	// Every op emitted while building the hidden subtree must stay
	// deactivated: the probe painted a red fill plus .sect gray fills and
	// 1pt #abc borders while the box was hidden.
	assertHiddenSubtreeNoopOps(t, res, left)
	assertNoRedFill(t, res)

	if lines := opsOfKind(res, OpLine); len(lines) != 0 {
		t.Fatalf("hidden sidebar painted %d border strokes: %+v", len(lines), lines)
	}

	afterY := textY(t, res, "AFTER-LEFT")

	// The collapsed sidebar must not shift the following sibling at all.
	control := layoutHTML(t, `<html><body>
		<div style="background:#f0f0f0"><p>AFTER-LEFT</p></div>
	</body></html>`, cssSheet)
	if controlY := textY(t, control, "AFTER-LEFT"); !near(afterY, controlY) {
		t.Fatalf("hidden sidebar moved AFTER-LEFT: with=%.2f without=%.2f", afterY, controlY)
	}
}

// Regression guard for cplusplus-tutorial row 2: after hidden backgrounds stop
// painting, the breadcrumb stays on page 1 with dark ink and no later fill
// covers it.
func TestHiddenSidebarDoesNotCoverBreadcrumb(t *testing.T) {
	t.Parallel()

	var rows strings.Builder
	for range 30 {
		rows.WriteString(`<div class="sect">HIDDEN-SIDEROW</div>`)
	}

	cssSheet := sheet(t, `
		body { margin: 0; font: 12px sans-serif }
		.left { background: #ff0000; visibility: hidden; height: 0 }
		.left .sect { background: #f8f8f8; border: 1px solid #abc; height: 40pt }
		.bar { background: #e8e8e8 }
	`)
	res := layoutHTML(t, `<html><body>
		<div class="left">`+rows.String()+`</div>
		<div class="main"><div class="bar">Tutorials : C++ Language</div></div>
	</body></html>`, cssSheet)

	doc := pdf.NewDocument()
	if err := Paint(doc, res, paintOpts()); err != nil {
		t.Fatal(err)
	}

	if got := pageOf(t, res, "Tutorials"); got != 0 {
		t.Fatalf("breadcrumb page = %d, want 0 (a hidden 0-height sidebar must not push content)", got)
	}

	textIdx, breadcrumb := -1, Op{}

	for i, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "Tutorials") {
			textIdx, breadcrumb = i, op

			break
		}
	}

	if textIdx < 0 {
		t.Fatal("no breadcrumb text op")
	}

	if breadcrumb.R+breadcrumb.G+breadcrumb.B > 1.5 {
		t.Fatalf("breadcrumb ink is not dark: rgb=(%.2f,%.2f,%.2f)", breadcrumb.R, breadcrumb.G, breadcrumb.B)
	}

	assertNoFillCoversText(t, res, textIdx, breadcrumb)
}

// fillCoversOpText reports whether a fill rect covers the advance box of a
// text op (baseline Y, ascent from H or Size).
func fillCoversOpText(fill, text Op) bool {
	top := text.Y
	if text.H > 0 {
		top = text.Y - text.H
	} else if text.Size > 0 {
		top = text.Y - text.Size
	}

	return fill.X <= text.X && fill.X+fill.W >= text.X+text.W &&
		fill.Y <= top && fill.Y+fill.H >= text.Y
}

// assertHiddenSubtreeNoopOps checks that every op emitted while building the
// hidden box's span stayed deactivated.
func assertHiddenSubtreeNoopOps(t *testing.T, res *Result, hidden *box) {
	t.Helper()

	for i := hidden.opStart; i <= hidden.opEnd && i < len(res.Ops); i++ {
		if res.Ops[i].Kind != opKindNoop {
			t.Fatalf("hidden sidebar op %d still paints: %+v", i, res.Ops[i])
		}
	}
}

// assertNoRedFill fails when an opaque red fill escaped a hidden subtree.
func assertNoRedFill(t *testing.T, res *Result) {
	t.Helper()

	for _, paintOp := range res.Ops {
		if paintOp.Kind == OpFillRect && paintOp.R > 0.7 && paintOp.G < 0.3 && paintOp.B < 0.3 {
			t.Fatalf("hidden sidebar painted red fill: %+v", paintOp)
		}
	}
}

// assertNoFillCoversText fails when an opaque fill after startIdx covers text.
func assertNoFillCoversText(t *testing.T, res *Result, startIdx int, text Op) {
	t.Helper()

	for i := startIdx + 1; i < len(res.Ops); i++ {
		fillOp := res.Ops[i]
		if fillOp.Kind != OpFillRect || fillOp.Alpha <= 0 {
			continue
		}

		if fillCoversOpText(fillOp, text) {
			t.Fatalf("fill painted after the breadcrumb covers it: %+v", fillOp)
		}
	}
}
