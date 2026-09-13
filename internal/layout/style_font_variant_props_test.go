//nolint:all // targeted unit tests for the support-props font variant group
package layout

import (
	"context"
	"testing"
)

func langField(s *ResolvedStyle) string      { return s.FontLanguageOverride }
func opticalField(s *ResolvedStyle) string   { return s.FontOpticalSizing }
func paletteField(s *ResolvedStyle) string   { return s.FontPalette }
func variationField(s *ResolvedStyle) string { return s.FontVariationSettings }

// TestApplyFontVariantProps parses every property in the group. Invalid
// declarations are dropped: the field keeps its initial value instead of a
// half-parsed one.
func TestApplyFontVariantProps(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		prop  string
		value string
		want  string
		get   func(*ResolvedStyle) string
	}{
		{"language double quoted", "font-language-override", `"TRK"`, "TRK", langField},
		{"language single quoted", "font-language-override", `'dflt'`, "dflt", langField},
		{"language normal case-insensitive", "font-language-override", "NoRmAl", "normal", langField},
		{"language unquoted dropped", "font-language-override", "TRK", "normal", langField},
		{"language overlong dropped", "font-language-override", `"zh-Hant"`, "normal", langField},
		{"language empty dropped", "font-language-override", `""`, "normal", langField},
		{"language escape dropped", "font-language-override", `"T\RK"`, "normal", langField},
		{"optical auto case-insensitive", "font-optical-sizing", "AUTO", "auto", opticalField},
		{"optical none trimmed", "font-optical-sizing", " none ", "none", opticalField},
		{"optical invalid dropped", "font-optical-sizing", "yes", "auto", opticalField},
		{"palette normal case-insensitive", "font-palette", "Normal", "normal", paletteField},
		{"palette light", "font-palette", "light", "light", paletteField},
		{"palette dark case-insensitive", "font-palette", "DARK", "dark", paletteField},
		{"palette ident keeps case", "font-palette", "--Brand-2", "--Brand-2", paletteField},
		{"palette ident leading dash", "font-palette", "---custom", "---custom", paletteField},
		{"palette non-dashed dropped", "font-palette", "brand", "normal", paletteField},
		{"palette digit start dropped", "font-palette", "--2brand", "normal", paletteField},
		{"palette mix dropped", "font-palette", "palette-mix(light, dark)", "normal", paletteField},
		{"variation normal case-insensitive", "font-variation-settings", "NORMAL", "normal", variationField},
		{"variation one axis", "font-variation-settings", `"wght" 700`, `"wght" 700`, variationField},
		{
			"variation list canonical", "font-variation-settings",
			`'wdth' 87.5,"wght" 400`, `"wdth" 87.5, "wght" 400`, variationField,
		},
		{"variation leading dot", "font-variation-settings", `"slnt" .5`, `"slnt" 0.5`, variationField},
		{"variation exponent collapses", "font-variation-settings", `"GRAD" 1e2`, `"GRAD" 100`, variationField},
		{"variation negative", "font-variation-settings", `"slnt" -12.25`, `"slnt" -12.25`, variationField},
		{
			"variation duplicate last wins", "font-variation-settings",
			`"wght" 400, "wght" 700`, `"wght" 700`, variationField,
		},
		{"variation unquoted tag dropped", "font-variation-settings", `wght 700`, "normal", variationField},
		{"variation short tag dropped", "font-variation-settings", `"wg" 1`, "normal", variationField},
		{"variation long tag dropped", "font-variation-settings", `"wghtt" 1`, "normal", variationField},
		{"variation missing number dropped", "font-variation-settings", `"wght"`, "normal", variationField},
		{"variation unit number dropped", "font-variation-settings", `"wght" 7px`, "normal", variationField},
		{"variation hex number dropped", "font-variation-settings", `"wght" 0x10`, "normal", variationField},
		{"variation trailing comma dropped", "font-variation-settings", `"wght" 700,`, "normal", variationField},
		{"variation bare dot dropped", "font-variation-settings", `"wght" .`, "normal", variationField},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			style := initialStyle()

			if owned := applyFontVariantProps(&style, testCase.prop, testCase.value, 12, nil, nil, false); !owned {
				t.Fatalf("applyFontVariantProps(%q) not owned by the group", testCase.prop)
			}

			if got := testCase.get(&style); got != testCase.want {
				t.Fatalf("%s %q = %q, want %q", testCase.prop, testCase.value, got, testCase.want)
			}
		})
	}
}

// TestFontVariantPropsDispatchAndInheritance pins the cascade wiring: the
// dispatch table routes the four names to this group (not applyIgnoredGroup),
// and inheritProps copies the stored value when the child declares nothing.
func TestFontVariantPropsDispatchAndInheritance(t *testing.T) {
	t.Parallel()

	parent := initialStyle()
	parent.FontLanguageOverride = "TRK"
	parent.FontOpticalSizing = fontOpticalNone
	parent.FontPalette = fontPaletteDark
	parent.FontVariationSettings = `"wght" 700`

	child := initialStyle()
	inheritProps(&child, &parent, nil)

	if child.FontLanguageOverride != "TRK" ||
		child.FontOpticalSizing != fontOpticalNone ||
		child.FontPalette != fontPaletteDark ||
		child.FontVariationSettings != `"wght" 700` {
		t.Fatalf(
			"inherited fields = %q / %q / %q / %q",
			child.FontLanguageOverride, child.FontOpticalSizing,
			child.FontPalette, child.FontVariationSettings,
		)
	}

	ctx := &styleContext{ctx: context.Background(), viewportW: 800}

	applyStyleProp(&child, "font-language-override", `"ENG"`, 12, ctx, &parent, true)

	if child.FontLanguageOverride != "ENG" {
		t.Fatalf("declared font-language-override = %q, want ENG", child.FontLanguageOverride)
	}
}

// TestApplyFontVariantPropsIgnoresOtherFontProps keeps the group from
// swallowing font properties it does not own.
func TestApplyFontVariantPropsIgnoresOtherFontProps(t *testing.T) {
	t.Parallel()

	style := initialStyle()

	if applyFontVariantProps(&style, "font-feature-settings", `"liga" 1`, 12, nil, nil, false) {
		t.Fatal("font-feature-settings must not be owned by the font variant group")
	}
}

// TestFontShapingLanguageOverride pins the selection half of
// font-language-override: normal means no override, a parsed tag is returned
// unchanged. The shaper consumer lives in internal/pdf (shapingInput) and is
// fed from OpText.TextLanguage.
func TestFontShapingLanguageOverride(t *testing.T) {
	t.Parallel()

	style := initialStyle()

	if got := fontShapingLanguage(&style); got != "" {
		t.Fatalf("normal override = %q, want empty (no override)", got)
	}

	style.FontLanguageOverride = "TRK"

	if got := fontShapingLanguage(&style); got != "TRK" {
		t.Fatalf("override = %q, want TRK", got)
	}

	if got := fontShapingLanguage(nil); got != "" {
		t.Fatalf("nil style = %q, want empty", got)
	}
}
