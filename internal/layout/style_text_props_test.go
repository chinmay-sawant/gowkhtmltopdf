//nolint:cyclop,lll,wsl,varnamelen,exhaustruct,usetesting,testpackage // targeted unit tests for Phase 80
package layout

import (
	"context"
	"testing"
)

func TestTextPropsWave3(t *testing.T) {
	t.Parallel()

	ctx := &styleContext{
		ctx:       context.Background(),
		err:       nil,
		work:      0,
		sheets:    nil,
		viewportW: 800,
	}

	s := initialStyle()
	raw := map[string]string{
		"text-align-last":      "center",
		"tab-size":             "4",
		"text-wrap-mode":       "nowrap",
		"text-wrap-style":      "balance",
		"white-space-collapse": "preserve",
		"white-space-trim":     "discard-inner",
		"hyphens":              "manual",
		"hyphenate-character":  "~",
		"text-justify":         "inter-word",
		"line-break":           "strict",
	}
	applyRestProps(&s, raw, ctx, nil)

	if s.TextAlignLast != "center" {
		t.Errorf("expected TextAlignLast 'center', got %q", s.TextAlignLast)
	}
	if s.TabSize != 4 {
		t.Errorf("expected TabSize 4, got %v", s.TabSize)
	}
	if s.TextWrapMode != "nowrap" || s.TextWrapStyle != "balance" {
		t.Errorf("expected text-wrap mode/style nowrap/balance, got mode=%q, style=%q", s.TextWrapMode, s.TextWrapStyle)
	}
	if s.WhiteSpaceCollapse != "preserve" || s.WhiteSpaceTrim != "discard-inner" {
		t.Errorf("expected white-space collapse/trim preserve/discard-inner, got %q, %q", s.WhiteSpaceCollapse, s.WhiteSpaceTrim)
	}
	if s.Hyphens != "manual" || s.HyphenateCharacter != "~" {
		t.Errorf("expected hyphens manual/~, got %q, %q", s.Hyphens, s.HyphenateCharacter)
	}
	if s.TextJustify != "inter-word" || s.LineBreak != "strict" {
		t.Errorf("expected text-justify inter-word / line-break strict, got %q, %q", s.TextJustify, s.LineBreak)
	}
}

func TestTextDecorationPropsWave3(t *testing.T) {
	t.Parallel()

	ctx := &styleContext{
		ctx:       context.Background(),
		err:       nil,
		work:      0,
		sheets:    nil,
		viewportW: 800,
	}

	s := initialStyle()
	raw := map[string]string{
		"text-decoration-line":      "underline",
		"text-decoration-color":     "red",
		"text-decoration-style":     "dashed",
		"text-decoration-thickness": "2pt",
		"text-underline-offset":     "3pt",
		"text-underline-position":   "under",
		"text-shadow":               "1pt 2pt 3pt blue",
	}
	applyRestProps(&s, raw, ctx, nil)

	if s.TextDecorationLine != "underline" || s.TextDecoration != cssTextDecorationUnderline {
		t.Errorf("mismatch on decoration line: line=%q, dec=%q", s.TextDecorationLine, s.TextDecoration)
	}
	if s.TextDecorationColor != [3]float64{1, 0, 0} || !s.TextDecorationColorSet {
		t.Errorf("mismatch on decoration color: color=%v, set=%v", s.TextDecorationColor, s.TextDecorationColorSet)
	}
	if s.TextDecorationStyle != "dashed" || s.TextDecorationThickness != 2 {
		t.Errorf("mismatch on decoration style/thickness: style=%q, th=%v", s.TextDecorationStyle, s.TextDecorationThickness)
	}
	if s.TextUnderlineOffset != 3 || s.TextUnderlinePosition != "under" {
		t.Errorf("mismatch on underline offset/pos: off=%v, pos=%q", s.TextUnderlineOffset, s.TextUnderlinePosition)
	}
	if !s.TextShadowSet || s.TextShadowX != 1 || s.TextShadowY != 2 || s.TextShadowBlur != 3 ||
		s.TextShadowColor != [3]float64{0, 0, 1} {
		t.Errorf("mismatch on text shadow: set=%v, x=%v, y=%v, blur=%v, color=%v",
			s.TextShadowSet, s.TextShadowX, s.TextShadowY, s.TextShadowBlur, s.TextShadowColor)
	}
}
