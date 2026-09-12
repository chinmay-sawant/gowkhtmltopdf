package layout

import (
	"strings"
	"testing"
)

// textOpIndex returns the position and value of the first OpText with the
// exact text.
func textOpIndex(res *Result, text string) (int, Op, bool) {
	for idx := range res.Ops {
		if res.Ops[idx].Kind == OpText && res.Ops[idx].Text == text {
			return idx, res.Ops[idx], true
		}
	}

	return -1, Op{}, false
}

// underlineOps returns decoration strokes (OpLine with a positive width).
func underlineOps(res *Result) []Op {
	var lines []Op

	for _, op := range res.Ops {
		if op.Kind == OpLine && op.W > 0.2 {
			lines = append(lines, op)
		}
	}

	return lines
}

// textOpTrimmedIndex returns the position of the first OpText whose trimmed
// text matches.
func textOpTrimmedIndex(res *Result, text string) (int, bool) {
	for idx := range res.Ops {
		if res.Ops[idx].Kind == OpText && strings.TrimSpace(res.Ops[idx].Text) == text {
			return idx, true
		}
	}

	return -1, false
}

// TestUnicodeBidiIsolateScopesReversal proves unicode-bidi:isolate keeps its
// reversal inside the element: the Hebrew words reverse, while the Latin runs
// before and after the scope keep logical order (the document heuristic must
// not scramble the whole line).
func TestUnicodeBidiIsolateScopesReversal(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><p style="margin:0;font-size:12pt">`+
		`one <span style="direction:rtl;unicode-bidi:isolate">`+
		`<span style="color:#a00">אבג</span> <span style="color:#00a">דהו</span>`+
		`</span> two</p></body></html>`)

	hebrewFirst, found := textOpTrimmedIndex(res, "דהו")
	if !found {
		t.Fatal("no text op for the second Hebrew word")
	}

	hebrewSecond, found := textOpTrimmedIndex(res, "אבג")
	if !found {
		t.Fatal("no text op for the first Hebrew word")
	}

	if hebrewFirst > hebrewSecond {
		t.Fatalf("isolate scope did not reverse: דהו index %d after אבג index %d", hebrewFirst, hebrewSecond)
	}

	before, found := textOpTrimmedIndex(res, "one")
	if !found {
		t.Fatal("no text op for one")
	}

	after, found := textOpTrimmedIndex(res, "two")
	if !found {
		t.Fatal("no text op for two")
	}

	if before > hebrewFirst || after < hebrewSecond {
		t.Fatalf("Latin runs moved across the isolate scope: one=%d hebrew=[%d,%d] two=%d",
			before, hebrewFirst, hebrewSecond, after)
	}
}

// TestUnicodeBidiOverrideReversesRunOrder proves unicode-bidi:bidi-override
// reverses the visual run order under direction:rtl, while the same markup
// without the override keeps logical order.
func TestUnicodeBidiOverrideReversesRunOrder(t *testing.T) {
	t.Parallel()

	page := func(style string) *Result {
		return layoutHTML(t, `<html><body><p style="margin:0;font-size:12pt">`+
			`<span style="`+style+`">`+
			`<span style="color:#cc0000">ab</span> `+
			`<span style="color:#0000cc">cd</span>`+
			`</span></p></body></html>`)
	}

	override := page("direction:rtl;unicode-bidi:bidi-override")

	cdIdx, cdOp, found := textOpIndex(override, "cd")
	if !found {
		t.Fatal("override: no text op for cd")
	}

	abIdx, abOp, found := textOpIndex(override, "ab")
	if !found {
		t.Fatal("override: no text op for ab")
	}

	if cdIdx > abIdx {
		t.Fatalf("override: cd op index %d after ab index %d, want reversed", cdIdx, abIdx)
	}

	if cdOp.X >= abOp.X {
		t.Fatalf("override: cd.X = %.2f, ab.X = %.2f, want cd left of ab", cdOp.X, abOp.X)
	}

	plain := page("")

	abIdx, _, found = textOpIndex(plain, "ab")
	if !found {
		t.Fatal("plain: no text op for ab")
	}

	cdIdx, _, found = textOpIndex(plain, "cd")
	if !found {
		t.Fatal("plain: no text op for cd")
	}

	if abIdx > cdIdx {
		t.Fatalf("plain: ab op index %d after cd index %d, want logical order", abIdx, cdIdx)
	}
}

// TestTextOrientationUprightVsMixedPaint proves text-orientation:upright
// paints unrotated runs in a vertical writing mode while the mixed default
// keeps the existing -90 degree run rotation.
func TestTextOrientationUprightVsMixedPaint(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><p style="margin:0;font-size:14pt">`+
		`<span style="writing-mode:vertical-rl;text-orientation:upright">AB</span>`+
		`<span style="writing-mode:vertical-rl">CD</span>`+
		`</p></body></html>`)

	_, upright, found := textOpIndex(res, "AB")
	if !found {
		t.Fatal("no text op for AB")
	}

	if upright.RotateDeg != 0 {
		t.Fatalf("upright RotateDeg = %v, want 0", upright.RotateDeg)
	}

	_, mixed, found := textOpIndex(res, "CD")
	if !found {
		t.Fatal("no text op for CD")
	}

	if mixed.RotateDeg != -90 {
		t.Fatalf("mixed RotateDeg = %v, want -90", mixed.RotateDeg)
	}
}

// TestTextCombineUprightDigits proves text-combine-upright combines a whole
// digit run up to the digits cap into one upright cell and leaves over-cap or
// non-digit runs rotated.
func TestTextCombineUprightDigits(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><p style="margin:0;font-size:14pt">`+
		`<span style="color:#111;writing-mode:vertical-rl;text-combine-upright:digits 4">2026</span>`+
		`<span style="color:#222;writing-mode:vertical-rl;text-combine-upright:digits 4">20260</span>`+
		`<span style="color:#333;writing-mode:vertical-rl;text-combine-upright:all">AB</span>`+
		`<span style="color:#444;writing-mode:vertical-rl">CD</span>`+
		`</p></body></html>`)

	cases := []struct {
		text string
		want float32
	}{
		{"2026", 0},
		{"20260", -90},
		{"AB", 0},
		{"CD", -90},
	}

	for _, testCase := range cases {
		_, textOp, found := textOpIndex(res, testCase.text)
		if !found {
			t.Fatalf("no text op for %q", testCase.text)
		}

		if textOp.RotateDeg != testCase.want {
			t.Fatalf("RotateDeg(%q) = %v, want %v", testCase.text, textOp.RotateDeg, testCase.want)
		}
	}
}

// TestTextDecorationSkipSpacesSegments proves text-decoration-skip-spaces
// splits a nowrap run into per-word strokes for all, keeps one continuous
// stroke for none, and that the shorthand none matches the longhand.
func TestTextDecorationSkipSpacesSegments(t *testing.T) {
	t.Parallel()

	layoutDecl := func(decl string) *Result {
		return layoutHTML(t, `<html><body><p style="margin:0;font-size:14pt">`+
			`<span style="white-space:nowrap;text-decoration:underline;`+decl+`">alpha beta</span>`+
			`</p></body></html>`)
	}

	all := underlineOps(layoutDecl("text-decoration-skip-spaces:all"))
	if len(all) != two {
		t.Fatalf("skip-spaces all produced %d decoration lines, want 2 (one per word)", len(all))
	}

	none := underlineOps(layoutDecl("text-decoration-skip-spaces:none"))
	if len(none) != 1 {
		t.Fatalf("skip-spaces none produced %d decoration lines, want 1 continuous", len(none))
	}

	if none[0].W <= all[0].W || none[0].W <= all[1].W {
		t.Fatalf("none stroke W=%.2f should span both words (all segments %.2f and %.2f)",
			none[0].W, all[0].W, all[1].W)
	}

	shorthand := underlineOps(layoutDecl("text-decoration-skip:none"))
	if len(shorthand) != 1 {
		t.Fatalf("shorthand none produced %d decoration lines, want 1 continuous", len(shorthand))
	}
}

// TestTextDecorationSkipSpacesEndTrimsTrailingSpace proves skip-spaces:end
// drops a run's trailing space from the painted stroke while none paints it.
func TestTextDecorationSkipSpacesEndTrimsTrailingSpace(t *testing.T) {
	t.Parallel()

	layoutDecl := func(decl string) *Result {
		return layoutHTML(t, `<html><body><p style="margin:0;font-size:14pt">`+
			`<span style="white-space:pre;text-decoration:underline;`+decl+`">alpha </span>`+
			`<span>beta</span></p></body></html>`)
	}

	trimmed := underlineOps(layoutDecl("text-decoration-skip-spaces:end"))
	if len(trimmed) != 1 {
		t.Fatalf("skip-spaces end produced %d decoration lines, want 1", len(trimmed))
	}

	painted := underlineOps(layoutDecl("text-decoration-skip-spaces:none"))
	if len(painted) != 1 {
		t.Fatalf("skip-spaces none produced %d decoration lines, want 1", len(painted))
	}

	if painted[0].W <= trimmed[0].W+1 {
		t.Fatalf("none stroke W=%.2f did not include the trailing space over end W=%.2f",
			painted[0].W, trimmed[0].W)
	}
}

// TestTextDecorationInsetTrimsStroke proves text-decoration-inset moves the
// decoration endpoints inward by the inset on both sides.
func TestTextDecorationInsetTrimsStroke(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body>`+
		`<p style="margin:0;font-size:14pt"><span style="text-decoration:underline">abcdef</span></p>`+
		`<p style="margin:0;font-size:14pt">`+
		`<span style="text-decoration:underline;text-decoration-inset:3pt">abcdef</span></p>`+
		`</body></html>`)

	lines := underlineOps(res)
	if len(lines) != two {
		t.Fatalf("inset layout produced %d decoration lines, want 2", len(lines))
	}

	plain, inset := lines[0], lines[1]

	if inset.X < plain.X+2.5 {
		t.Fatalf("inset X=%.2f, plain X=%.2f; want start inset by 3pt", inset.X, plain.X)
	}

	if inset.W > plain.W-5.5 {
		t.Fatalf("inset W=%.2f, plain W=%.2f; want both endpoints inset by 3pt", inset.W, plain.W)
	}
}

// TestTextDecorationSkipSelfAndBoxSubsets proves the documented subsets:
// skip-self:skip-all suppresses the item's own stroke and skip-box:all breaks
// a continuous stroke at an item with inline padding.
func TestTextDecorationSkipSelfAndBoxSubsets(t *testing.T) {
	t.Parallel()

	page := func(middle string) *Result {
		return layoutHTML(t, `<html><body><p style="margin:0;font-size:14pt">`+
			`<span style="text-decoration:underline">ab</span>`+
			`<span style="text-decoration:underline;padding:0 6pt;`+middle+`">cd</span>`+
			`<span style="text-decoration:underline">ef</span>`+
			`</p></body></html>`)
	}

	if got := len(underlineOps(page(""))); got != 1 {
		t.Fatalf("continuous run = %d decoration lines, want 1", got)
	}

	if got := len(underlineOps(page("text-decoration-skip-box:all"))); got != two {
		t.Fatalf("skip-box all run = %d decoration lines, want 2", got)
	}

	skipSelf := layoutHTML(t, `<html><body><p style="margin:0;font-size:14pt">`+
		`<span style="text-decoration:underline;text-decoration-skip-self:skip-all">ab</span>`+
		`</p></body></html>`)

	if got := len(underlineOps(skipSelf)); got != 0 {
		t.Fatalf("skip-self skip-all = %d decoration lines, want 0", got)
	}
}
