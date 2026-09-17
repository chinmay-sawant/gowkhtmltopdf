package layout

import (
	"strings"
	"testing"
)

func TestTextBoxTrimBothShrinksHalfLeading(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:20pt;line-height:2">Hg</p></body></html>`)
	trimmed := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:20pt;line-height:2;`+
		`text-box:trim-both cap alphabetic">Hg</p></body></html>`)

	if trimmed.Height >= plain.Height-0.5 {
		t.Fatalf("trim-both height %v not smaller than untrimmed %v", trimmed.Height, plain.Height)
	}

	sty := initialStyle()
	applyTextBoxProps(&sty, "text-box", "trim-both cap alphabetic", 12, nil, nil, false)

	if sty.TextBoxTrim != "trim-both" || sty.TextBoxEdgeOver != "cap" || sty.TextBoxEdgeUnder != "alphabetic" {
		t.Fatalf("shorthand apply: trim=%q over=%q under=%q", sty.TextBoxTrim, sty.TextBoxEdgeOver, sty.TextBoxEdgeUnder)
	}
}

func TestTextAutospaceIdeographAlpha(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:16pt;text-autospace:no-autospace">漢A</p>`+
		`</body></html>`)
	spaced := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;font-size:16pt;text-autospace:ideograph-alpha">漢A</p>`+
		`</body></html>`)

	wp := firstTextWidth(t, plain, "漢A")
	ws := firstTextWidth(t, spaced, "漢A")

	if ws <= wp+0.1 {
		t.Fatalf("ideograph-alpha width %v should exceed no-autospace width %v", ws, wp)
	}

	sty := initialStyle()
	applyTextAutospaceProps(&sty, "text-autospace", "ideograph-alpha", 12, nil, nil, false)
	applyTextAutospaceProps(&sty, "text-spacing-trim", "trim-start", 12, nil, nil, false)
	applyTextAutospaceProps(&sty, "text-spacing", "trim-start", 12, nil, nil, false)

	if sty.TextAutospace != "ideograph-alpha" || sty.TextSpacingTrim != "trim-start" {
		t.Fatalf("autospace/spacing apply: autospace=%q trim=%q spacing=%q",
			sty.TextAutospace, sty.TextSpacingTrim, sty.TextSpacing)
	}
}

func TestTextGroupAlignCenter(t *testing.T) {
	t.Parallel()

	start := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:200pt;font-size:12pt;text-group-align:none">Hi</p>`+
		`</body></html>`)
	center := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:200pt;font-size:12pt;text-group-align:center">Hi</p>`+
		`</body></html>`)

	xs := firstTextX(t, start, "Hi")
	xc := firstTextX(t, center, "Hi")

	if xc <= xs+1 {
		t.Fatalf("text-group-align:center X %v should be right of start X %v", xc, xs)
	}
}

func TestTextSpacingTrimApply(t *testing.T) {
	t.Parallel()

	sty := initialStyle()
	if !applyTextAutospaceProps(&sty, "text-fit", "scale", 12, nil, nil, false) {
		t.Fatal("text-fit apply should consume the declaration")
	}

	if sty.TextFit != "scale" {
		t.Fatalf("TextFit=%q, want scale (stored; scale-search consumer not shipped)", sty.TextFit)
	}
}

func TestSoftHyphenUsesHyphenateCharacter(t *testing.T) {
	t.Parallel()

	// Narrow box forces a break at the soft hyphen; hyphenate-character "~"
	shy := "super\u00ADcalifragilistic"
	res := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:48pt;font-size:14pt;hyphens:manual;hyphenate-character:'~'">`+
		shy+`</p></body></html>`)

	var sawTilde bool

	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "~") {
			sawTilde = true

			break
		}
	}

	if !sawTilde {
		var texts []string
		for _, op := range res.Ops {
			if op.Kind == OpText {
				texts = append(texts, op.Text)
			}
		}

		t.Fatalf("expected hyphenate-character '~' in a text op, got %q", texts)
	}
}

func TestHyphenateLimitChars(t *testing.T) {
	t.Parallel()

	// SHY after 1 letter: before=1 fails min-before=3.
	blocked := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:40pt;font-size:14pt;hyphens:manual;`+
		`hyphenate-limit-chars:6 3 3;hyphenate-character:'~'">`+
		"a\u00ADbcdefghij</p></body></html>")

	allowed := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:40pt;font-size:14pt;hyphens:manual;`+
		`hyphenate-limit-chars:6 3 3;hyphenate-character:'~'">`+
		"abc\u00ADdefghij</p></body></html>")

	if textOpsContain(blocked, "~") {
		t.Fatal("hyphenate-limit-chars should refuse SHY with only 1 letter before")
	}

	if !textOpsContain(allowed, "~") {
		t.Fatal("hyphenate-limit-chars should allow SHY with 3 letters before and after")
	}

	sty := initialStyle()
	applyHyphenationProps(&sty, "hyphenate-limit-chars", "6 3 3", 12, nil, nil, false)
	applyHyphenationProps(&sty, "hyphenate-limit-zone", "8%", 12, &styleContext{viewportW: 100}, nil, false)
	applyHyphenationProps(&sty, "hyphenate-limit-lines", "2", 12, nil, nil, false)
	applyHyphenationProps(&sty, "hyphenate-limit-last", "always", 12, nil, nil, false)
	applyHyphenationProps(&sty, "hanging-punctuation", "first", 12, nil, nil, false)

	if sty.HyphenateLimitMinWord != 6 || sty.HyphenateLimitMinBefore != 3 || sty.HyphenateLimitMinAfter != 3 {
		t.Fatalf("limit-chars got %d %d %d", sty.HyphenateLimitMinWord, sty.HyphenateLimitMinBefore, sty.HyphenateLimitMinAfter)
	}

	if sty.HyphenateLimitZonePct != 8 || sty.HyphenateLimitLines != 2 ||
		sty.HyphenateLimitLast != "always" || sty.HangingPunctuation != "first" {
		t.Fatalf("limit/hanging apply failed: zonePct=%v lines=%d last=%q hang=%q",
			sty.HyphenateLimitZonePct, sty.HyphenateLimitLines, sty.HyphenateLimitLast, sty.HangingPunctuation)
	}
}

func TestHangingPunctuationFirst(t *testing.T) {
	t.Parallel()

	plain := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:120pt;font-size:14pt;hanging-punctuation:none">"Hello</p>`+
		`</body></html>`)
	hang := layoutHTML(t, `<html><body style="margin:0">`+
		`<p style="margin:0;width:120pt;font-size:14pt;hanging-punctuation:first">"Hello</p>`+
		`</body></html>`)

	xp := firstTextX(t, plain, `"Hello`)
	xh := firstTextX(t, hang, `"Hello`)

	if xh >= xp-0.1 {
		t.Fatalf("hanging-punctuation:first X %v should be left of plain X %v", xh, xp)
	}
}

func firstTextWidth(t *testing.T, res *Result, text string) float64 {
	t.Helper()

	for _, op := range res.Ops {
		if op.Kind == OpText && op.Text == text {
			return op.W
		}
	}

	t.Fatalf("no text op %q", text)

	return 0
}

func firstTextX(t *testing.T, res *Result, text string) float64 {
	t.Helper()

	for _, op := range res.Ops {
		if op.Kind == OpText && op.Text == text {
			return op.X
		}
	}

	t.Fatalf("no text op %q", text)

	return 0
}

func textOpsContain(res *Result, needle string) bool {
	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, needle) {
			return true
		}
	}

	return false
}
