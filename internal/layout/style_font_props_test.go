//nolint:all // targeted unit tests for font prop cascade wiring
package layout

import (
	"context"
	"testing"
)

func TestFontPropsWave4(t *testing.T) {
	t.Parallel()

	ctx := &styleContext{
		ctx:       context.Background(),
		viewportW: 800,
	}
	style := initialStyle()
	raw := map[string]string{
		"font-feature-settings": `"liga" 1, "smcp" 1`,
		"font-kerning":          "none",
		"font-variant-caps":     "small-caps",
		"font-stretch":          "condensed",
		"font-size-adjust":      "0.58",
	}
	applyFontProps(&style, raw, 12, ctx)
	applyRestProps(&style, raw, ctx, nil)

	if style.FontFeatureSettings != `"liga" 1, "smcp" 1` {
		t.Fatalf("feature-settings = %q", style.FontFeatureSettings)
	}

	if style.FontKerning != "none" {
		t.Fatalf("kerning = %q", style.FontKerning)
	}

	if style.FontVariantCaps != "small-caps" {
		t.Fatalf("caps = %q", style.FontVariantCaps)
	}

	if style.FontWidth != 75 {
		t.Fatalf("stretch/width = %v", style.FontWidth)
	}

	if !style.FontSizeAdjustSet || style.FontSizeAdjust != 0.58 {
		t.Fatalf("size-adjust = set=%v value=%v", style.FontSizeAdjustSet, style.FontSizeAdjust)
	}
}
