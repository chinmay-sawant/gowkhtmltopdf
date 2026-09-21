//nolint:all // targeted unit tests for CSS Fonts feature/variant group
package layout

import (
	"context"
	"strings"
	"testing"
)

func TestApplyFontFeatureSettings(t *testing.T) {
	t.Parallel()

	style := initialStyle()
	if !applyFontFeatureProps(&style, "font-feature-settings", `"smcp" 1, "liga" on`, 12, nil, nil, false) {
		t.Fatal("font-feature-settings not owned")
	}

	if style.FontFeatureSettings != `"smcp" 1, "liga" 1` {
		t.Fatalf("settings = %q", style.FontFeatureSettings)
	}

	if !applyFontFeatureProps(&style, "font-feature-settings", "NORMAL", 12, nil, nil, false) {
		t.Fatal("normal not owned")
	}

	if style.FontFeatureSettings != "normal" {
		t.Fatalf("normal = %q", style.FontFeatureSettings)
	}
}

func TestFontKerningNone(t *testing.T) {
	t.Parallel()

	style := initialStyle()
	if !applyFontFeatureProps(&style, "font-kerning", "none", 12, nil, nil, false) {
		t.Fatal("font-kerning not owned")
	}

	if style.FontKerning != "none" {
		t.Fatalf("kerning = %q", style.FontKerning)
	}

	got := fontShapingFeatureSettings(&style)
	if !strings.Contains(got, `"kern" 0`) {
		t.Fatalf("feature settings = %q, want kern 0", got)
	}
}

func TestFontVariantCapsMapsToSmcp(t *testing.T) {
	t.Parallel()

	style := initialStyle()
	if !applyFontFeatureProps(&style, "font-variant-caps", "small-caps", 12, nil, nil, false) {
		t.Fatal("font-variant-caps not owned")
	}

	if style.FontVariantCaps != "small-caps" {
		t.Fatalf("caps = %q", style.FontVariantCaps)
	}

	got := fontShapingFeatureSettings(&style)
	if got != `"smcp" 1` {
		t.Fatalf("features = %q, want smcp 1", got)
	}
}

func TestFontVariantShorthandExpands(t *testing.T) {
	t.Parallel()

	style := initialStyle()
	if !applyFontFeatureProps(&style, "font-variant", "small-caps", 12, nil, nil, false) {
		t.Fatal("font-variant not owned")
	}

	if style.FontVariantCaps != "small-caps" {
		t.Fatalf("caps = %q after small-caps shorthand", style.FontVariantCaps)
	}

	style = initialStyle()
	if !applyFontFeatureProps(&style, "font-variant", "normal", 12, nil, nil, false) {
		t.Fatal("font-variant normal not owned")
	}

	if style.FontVariantCaps != "normal" || style.FontVariantLigatures != "normal" {
		t.Fatalf("normal expand = caps %q lig %q", style.FontVariantCaps, style.FontVariantLigatures)
	}
}

func TestFontFeatureSettingsReachShaper(t *testing.T) {
	t.Parallel()

	doc := parseTestHTML(t, `<html><body style="margin:0">`+
		`<p style="font-feature-settings: 'smcp' 1">features</p>`+
		`<p style="font-kerning: none">nokern</p>`+
		`<p>plain</p>`+
		`</body></html>`)

	res, err := Layout(doc, Options{Width: 500, Height: 400})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"features": `"smcp" 1`,
		"nokern":   `"kern" 0`,
		"plain":    "",
	}
	found := map[string]bool{}

	for i := range res.Ops {
		paintOp := &res.Ops[i]
		if paintOp.Kind != OpText {
			continue
		}

		for text, feats := range want {
			if !strings.Contains(paintOp.Text, text) {
				continue
			}

			found[text] = true

			if got := paintOp.FontFeatures(); got != feats {
				t.Errorf("OpText %q features = %q, want %q", text, got, feats)
			}
		}
	}

	for text := range want {
		if !found[text] {
			t.Errorf("no OpText for %q", text)
		}
	}
}

func TestFontFeaturePropsInherit(t *testing.T) {
	t.Parallel()

	parent := initialStyle()
	parent.FontFeatureSettings = `"liga" 0`
	parent.FontKerning = "none"
	parent.FontVariantCaps = "small-caps"

	child := initialStyle()
	inheritProps(&child, &parent, nil)

	if child.FontFeatureSettings != `"liga" 0` || child.FontKerning != "none" || child.FontVariantCaps != "small-caps" {
		t.Fatalf("inherited = %q / %q / %q", child.FontFeatureSettings, child.FontKerning, child.FontVariantCaps)
	}

	ctx := &styleContext{ctx: context.Background(), viewportW: 800}
	applyStyleProp(&child, "font-feature-settings", `"smcp" 1`, 12, ctx, &parent, true)

	if child.FontFeatureSettings != `"smcp" 1` {
		t.Fatalf("declared = %q", child.FontFeatureSettings)
	}
}
