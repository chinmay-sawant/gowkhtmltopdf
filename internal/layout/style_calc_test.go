package layout

import "testing"

// TestWidthCalcPercentResolvesAgainstContainingBlock is the learncpp probe:
// calc(100% - 200px) inside a 400px (300pt) containing block is 200px (150pt),
// not a viewport-relative 518px. The percentage term defers to layout so it
// sees the containing block; mixed-unit calc without a percentage still
// resolves at cascade.
func TestWidthCalcPercentResolvesAgainstContainingBlock(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
	.box { width:400px }
	.probe { width: calc(100% - 200px) }
	.half { width: calc(50% - 10px) }
	.mixed { width: calc(20pt + 10px) }
	.full { width: calc(100% + 0px) }
	`)

	res := layoutHTML(t, `<html><body><div class="box">
	<div class="probe">p</div>
	<div class="half">h</div>
	<div class="mixed">m</div>
	<div class="full">f</div>
	</div></body></html>`, cssSheet)

	if got := boxWithClass(t, res, "box").w; !near(got, 300) {
		t.Fatalf("parent width = %.2fpt, want 300pt", got)
	}

	// 400px - 200px = 200px = 150pt. A viewport-relative resolution (test
	// viewport 500pt) would give 500 - 150 = 350pt.
	if got := boxWithClass(t, res, "probe").w; !near(got, 150) {
		t.Fatalf("width:calc(100%% - 200px) = %.2fpt, want 150pt (containing-block base)", got)
	}

	if got := boxWithClass(t, res, "half").w; !near(got, 142.5) {
		t.Fatalf("width:calc(50%% - 10px) = %.2fpt, want 142.5pt", got)
	}

	// Mixed units without a percentage keep the cascade-time resolution.
	if got := boxWithClass(t, res, "mixed").w; !near(got, 27.5) {
		t.Fatalf("width:calc(20pt + 10px) = %.2fpt, want 27.5pt", got)
	}

	if got := boxWithClass(t, res, "full").w; !near(got, 300) {
		t.Fatalf("width:calc(100%% + 0px) = %.2fpt, want 300pt", got)
	}
}

// TestFlexChildWidthCalcPercent covers the flex main-size consumers so a calc
// item is not resolved against the viewport there either.
func TestFlexChildWidthCalcPercent(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
	.row { display:flex; width:400px; margin:0 }
	.item { flex-shrink:0; width: calc(100% - 200px) }
	`)

	res := layoutHTML(t, `<html><body>
	<div class="row"><div class="item">x</div></div>
	</body></html>`, cssSheet)

	if got := boxWithClass(t, res, "item").w; !near(got, 150) {
		t.Fatalf("flex item width:calc(100%% - 200px) = %.2fpt, want 150pt", got)
	}
}

// TestPercentTopResolvesAgainstContainingBlock: percent insets defer to layout.
// A relative box with top:50% in an auto-height containing block keeps its
// static position instead of shifting by half the page viewport (the learncpp
// masthead probe moved 392.6pt too far). An absolute box with a definite
// 100pt containing block resolves 25% against the block.
func TestPercentTopResolvesAgainstContainingBlock(t *testing.T) {
	t.Parallel()

	shared := sheet(t, `
	.rel { position: relative; top: 50%; height: 10pt }
	.ctrl { height: 10pt }
	.cb { position: relative; height: 100pt }
	.abs { position: absolute; top: 25%; height: 10pt }
	`)

	withPct := layoutHTML(t, `<html><body><div class="rel">r</div></body></html>`, shared)
	control := layoutHTML(t, `<html><body><div class="ctrl">c</div></body></html>`, shared)

	relY := boxWithClass(t, withPct, "rel").y
	ctrlY := boxWithClass(t, control, "ctrl").y

	if !near(relY, ctrlY) {
		t.Fatalf("relative top:50%% shifted y by %.2fpt; must not resolve against the viewport height", relY-ctrlY)
	}

	absRes := layoutHTML(t, `<html><body><div class="cb"><div class="abs">a</div></div></body></html>`, shared)

	cbY := boxWithClass(t, absRes, "cb").y
	absY := boxWithClass(t, absRes, "abs").y

	if got := absY - cbY; !near(got, 25) {
		t.Fatalf("absolute top:25%% offset = %.2fpt, want 25pt against the 100pt containing block", got)
	}
}
