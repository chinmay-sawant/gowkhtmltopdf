//nolint:cyclop,lll,wsl,varnamelen,exhaustruct,usetesting,testpackage // targeted unit tests for Phase 80
package layout

import (
	"context"
	"testing"
)

func TestFontPropsWave4(t *testing.T) {
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
		"font-feature-settings":     `"liga" 1, "smcp" 1`,
		"font-kerning":              "none",
		"font-variant-caps":         "small-caps",
		"font-variant-ligatures":    "common-ligatures",
		"font-variant-numeric":      "tabular-nums",
		"font-variant-position":     "sub",
		"font-variant-east-asian":   "simplified",
		"font-variant-emoji":        "text",
		"font-variant-alternates":   "historical-forms",
		"font-stretch":              "condensed",
		"font-synthesis-weight":     "none",
		"font-synthesis-style":      "none",
		"font-synthesis-small-caps": "auto",
		"font-synthesis-position":   "none",
		"font-size-adjust":          "0.58",
	}
	applyFontProps(&s, raw, 12, ctx)

	if s.FontFeatureSettings != `"liga" 1, "smcp" 1` {
		t.Errorf("mismatch on font-feature-settings: %q", s.FontFeatureSettings)
	}
	if s.FontKerning != "none" {
		t.Errorf("mismatch on font-kerning: %q", s.FontKerning)
	}
	if s.FontVariantCaps != "small-caps" || s.FontVariantLigatures != "common-ligatures" || s.FontVariantNumeric != "tabular-nums" {
		t.Errorf("mismatch on font-variant: caps=%q, lig=%q, num=%q", s.FontVariantCaps, s.FontVariantLigatures, s.FontVariantNumeric)
	}
	if s.FontVariantPosition != "sub" || s.FontVariantEastAsian != "simplified" || s.FontVariantEmoji != "text" || s.FontVariantAlternates != "historical-forms" {
		t.Errorf("mismatch on font-variant pos/ea/emoji/alt: pos=%q, ea=%q, emoji=%q, alt=%q",
			s.FontVariantPosition, s.FontVariantEastAsian, s.FontVariantEmoji, s.FontVariantAlternates)
	}
	if s.FontStretch != "condensed" {
		t.Errorf("mismatch on font-stretch: %q", s.FontStretch)
	}
	if s.FontSynthesisWeight || s.FontSynthesisStyle || !s.FontSynthesisSmallCaps || s.FontSynthesisPosition {
		t.Errorf("mismatch on font-synthesis: weight=%v, style=%v, smcp=%v, pos=%v",
			s.FontSynthesisWeight, s.FontSynthesisStyle, s.FontSynthesisSmallCaps, s.FontSynthesisPosition)
	}
	if s.FontSizeAdjust != 0.58 {
		t.Errorf("mismatch on font-size-adjust: %v", s.FontSizeAdjust)
	}
}
