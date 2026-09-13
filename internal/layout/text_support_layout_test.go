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

// paintedText concatenates every text op in paint order.
func paintedText(res *Result) string {
	var out strings.Builder

	for _, op := range res.Ops {
		if op.Kind == OpText {
			out.WriteString(op.Text)
		}
	}

	return out.String()
}

// TestUnicodeBidiOverrideKeepsSpaces proves a bidi-override reversal keeps the
// separator between two words. Word collection attaches the separator to the
// word it follows ("ABC " + "123"), so a naive reversal reorders the words and
// leaves the space trailing the run, where line trailing-space trimming drops
// it. The visual order must be 123, space, ABC.
func TestUnicodeBidiOverrideKeepsSpaces(t *testing.T) {
	t.Parallel()

	items := []inlineItem{
		{text: "ABC "},
		{text: "123"},
	}
	reverseInlineRange(items, 0)

	if items[0].text != "123" || items[1].text != " ABC" {
		t.Fatalf("reversed items = %q + %q, want 123 + space + ABC",
			items[0].text, items[1].text)
	}

	// Three words keep one separator per boundary.
	three := []inlineItem{
		{text: "one "},
		{text: "two "},
		{text: "three"},
	}
	reverseInlineRange(three, 0)

	var order strings.Builder

	for i := range three {
		order.WriteString(three[i].text)
	}

	if order.String() != "three two one" {
		t.Fatalf("three-word reversal = %q, want %q", order.String(), "three two one")
	}

	// A trailing separator uses the raw reversal path: the run already ends in
	// a space, and rotating would double the gap.
	trailing := []inlineItem{
		{text: "ABC "},
		{text: "123 "},
	}
	reverseInlineRange(trailing, 0)

	if trailing[0].text+trailing[1].text != "123 ABC " {
		t.Fatalf("trailing-separator reversal = %q, want %q",
			trailing[0].text+trailing[1].text, "123 ABC ")
	}

	// End to end: the painted run keeps the gap.
	res := layoutHTML(t, `<html><body><p style="margin:0;font-size:12pt">`+
		`<span style="direction:rtl;unicode-bidi:bidi-override">ABC 123</span>`+
		`</p></body></html>`)

	painted := paintedText(res)

	if !strings.Contains(painted, "123 ABC") {
		t.Fatalf("painted text %q missing %q", painted, "123 ABC")
	}

	// Trailing whitespace in the source must paint one gap, not two.
	trailingRes := layoutHTML(t, `<html><body><p style="margin:0;font-size:12pt">`+
		`<span style="direction:rtl;unicode-bidi:bidi-override">ABC 123 </span>`+
		`</p></body></html>`)
	trailingPainted := paintedText(trailingRes)

	if !strings.Contains(trailingPainted, "123 ABC") || strings.Contains(trailingPainted, "123  ABC") {
		t.Fatalf("trailing-whitespace painted text = %q, want one gap", trailingPainted)
	}
}

// TestTextOrientationUprightVsMixedPaint proves text-orientation:upright
// stacks each glyph unrotated down the column while the mixed default keeps
// the existing -90 degree run rotation.
func TestTextOrientationUprightVsMixedPaint(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body><p style="margin:0;font-size:14pt">`+
		`<span style="writing-mode:vertical-rl;text-orientation:upright">AB</span>`+
		`<span style="writing-mode:vertical-rl">CD</span>`+
		`</p></body></html>`)

	_, aOp, aFound := textOpIndex(res, "A")
	_, bOp, bFound := textOpIndex(res, "B")

	if !aFound || !bFound {
		t.Fatalf("upright AB produced glyph ops A=%v B=%v, want one op per glyph", aFound, bFound)
	}

	if diff := aOp.X - bOp.X; diff < -0.5 || diff > 0.5 {
		t.Errorf("upright glyph X = %.2f and %.2f, want one centered column", aOp.X, bOp.X)
	}

	if bOp.Y <= aOp.Y {
		t.Errorf("upright glyphs A y=%.2f B y=%.2f, want B below A", aOp.Y, bOp.Y)
	}

	if aOp.RotateDeg != 0 || bOp.RotateDeg != 0 {
		t.Errorf("upright RotateDeg = %v/%v, want 0", aOp.RotateDeg, bOp.RotateDeg)
	}

	_, mixed, found := textOpIndex(res, "CD")
	if !found {
		t.Fatal("no text op for CD")
	}

	if mixed.RotateDeg != -90 {
		t.Fatalf("mixed RotateDeg = %v, want -90", mixed.RotateDeg)
	}
}

// TestVerticalRLColumnAnchorsRightAndCombinedCellCenters proves a vertical-rl
// column anchors at the content box's right edge and a text-combine-upright
// cell centers itself in the column.
func TestVerticalRLColumnAnchorsRightAndCombinedCellCenters(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<div style="width:200pt;writing-mode:vertical-rl;text-orientation:upright;font-size:10pt">AB</div>`+
		`<div style="width:200pt;writing-mode:vertical-rl;text-combine-upright:digits 2;font-size:10pt">12</div>`+
		`</body></html>`)

	_, aOp, aFound := textOpIndex(res, "A")
	_, bOp, bFound := textOpIndex(res, "B")
	_, cell, cellFound := textOpIndex(res, "12")

	if !aFound || !bFound || !cellFound {
		t.Fatalf("missing ops: A=%v B=%v cell=%v", aFound, bFound, cellFound)
	}

	if aOp.X < 150 {
		t.Errorf("upright column X = %.2f, want near the 200pt block's right edge", aOp.X)
	}

	if bOp.X < 150 {
		t.Errorf("upright column B X = %.2f, want near the 200pt block's right edge", bOp.X)
	}

	if cell.X < 150 {
		t.Errorf("combined cell X = %.2f, want near the 200pt block's right edge", cell.X)
	}

	if cell.RotateDeg != 0 {
		t.Errorf("combined cell RotateDeg = %v, want 0", cell.RotateDeg)
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

	// The legacy spaces keyword maps onto skip-spaces:all, so the shorthand
	// paints the same per-word strokes with an interior gap.
	shorthandSpaces := underlineOps(layoutDecl("text-decoration-skip:spaces"))
	if len(shorthandSpaces) != two {
		t.Fatalf("shorthand spaces produced %d decoration lines, want 2 (one per word)", len(shorthandSpaces))
	}

	if shorthandSpaces[0].W >= none[0].W || shorthandSpaces[1].W >= none[0].W {
		t.Fatalf("shorthand spaces strokes W=%.2f and %.2f should be narrower than the none span W=%.2f",
			shorthandSpaces[0].W, shorthandSpaces[1].W, none[0].W)
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
