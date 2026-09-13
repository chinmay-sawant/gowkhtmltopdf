//nolint:all // targeted unit tests
package layout

import (
	"testing"
)

func TestWaveBTextTruncationAndClamping(t *testing.T) {
	t.Parallel()

	var s ResolvedStyle
	applyAdvancedProps(&s, "text-overflow", "ellipsis", 12)
	applyAdvancedProps(&s, "line-clamp", "3", 12)
	applyAdvancedProps(&s, "max-lines", "5", 12)
	applyAdvancedProps(&s, "margin-trim", "block", 12)

	if s.TextOverflow != "ellipsis" {
		t.Errorf("TextOverflow = %q, want 'ellipsis'", s.TextOverflow)
	}
	if s.LineClamp != 3 {
		t.Errorf("LineClamp = %d, want 3", s.LineClamp)
	}
	if s.MaxLines != 5 {
		t.Errorf("MaxLines = %d, want 5", s.MaxLines)
	}
	if s.MarginTrim != "block" {
		t.Errorf("MarginTrim = %q, want 'block'", s.MarginTrim)
	}
}

func TestLineClampClearsWebkitBoxNowrap(t *testing.T) {
	t.Parallel()

	var s ResolvedStyle
	setDisplayKeyword(&s, "-webkit-box")
	if s.WhiteSpace != cssWhiteSpaceNowrap {
		t.Fatalf("webkit-box WhiteSpace=%q, want nowrap", s.WhiteSpace)
	}
	applyAdvancedProps(&s, "-webkit-line-clamp", "2", 12)
	if s.LineClamp != 2 {
		t.Fatalf("LineClamp=%d, want 2", s.LineClamp)
	}
	if s.WhiteSpace == cssWhiteSpaceNowrap {
		t.Fatal("line-clamp should clear webkit-box nowrap so multiple lines can form")
	}
}

func TestWordBreakOfLineBreakAnywhere(t *testing.T) {
	t.Parallel()

	st := ResolvedStyle{LineBreak: "anywhere", WhiteSpace: "normal"}
	if got := wordBreakOf(st); got != breakAll {
		t.Fatalf("wordBreakOf(line-break:anywhere)=%v, want breakAll", got)
	}
}

func TestWaveCBoxDecorationBreak(t *testing.T) {
	t.Parallel()

	var s ResolvedStyle
	applyAdvancedProps(&s, "box-decoration-break", "clone", 12)

	if s.BoxDecorationBreak != "clone" {
		t.Errorf("BoxDecorationBreak = %q, want 'clone'", s.BoxDecorationBreak)
	}
}

func TestWaveDTextDecorationSkipInkAndOverflowClipMargin(t *testing.T) {
	t.Parallel()

	var s ResolvedStyle
	applyAdvancedProps(&s, "text-decoration-skip-ink", "auto", 12)
	applyAdvancedProps(&s, "overflow-clip-margin-top", "10px", 12)
	applyAdvancedProps(&s, "overflow-clip-margin-right", "20pt", 12)

	if s.TextDecorationSkipInk != "auto" {
		t.Errorf("TextDecorationSkipInk = %q, want 'auto'", s.TextDecorationSkipInk)
	}
	if s.OverflowClipMarginTop != 7.5 {
		t.Errorf("OverflowClipMarginTop = %v, want 7.5", s.OverflowClipMarginTop)
	}
	if s.OverflowClipMarginRight != 20 {
		t.Errorf("OverflowClipMarginRight = %v, want 20", s.OverflowClipMarginRight)
	}
}
