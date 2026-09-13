package pdf

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf/assets"
	"github.com/go-text/typesetting/language"
)

// bundledDejaVu parses the in-tree DejaVu Sans face, which carries a Cyrillic
// SRB language system used to observe the override effect.
func bundledDejaVu(t *testing.T) *Font {
	t.Helper()

	fnt, err := ParseTTF(assets.UnicodeFallbackRegular())
	if err != nil {
		t.Fatalf("parse bundled DejaVuSans: %v", err)
	}

	if !fnt.hasGSUB() {
		t.Fatal("bundled DejaVuSans expected to have GSUB")
	}

	return fnt
}

// TestShapingLanguageOverride pins the OpenType tag to BCP47 mapping. Several
// raw tags would collide with unrelated BCP47 codes if only lowercased, so the
// mapped subset matters for language-system selection.
func TestShapingLanguageOverride(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		tag  string
		want language.Language
	}{
		{"empty", "", ""},
		{"normal", "normal", ""},
		{"normal case-insensitive", "NoRmAl", ""},
		{"padded", " TRK ", "tr"},
		{"turkish", "TRK", "tr"},
		{"turkish lowercase", "trk", "tr"},
		{"serbian", "SRB", "sr"},
		{"azerbaijani", "AZE", "az"},
		{"romanian", "ROM", "ro"},
		{"hungarian", "HUN", "hu"},
		{"polish legacy tag", "POL", "pl"},
		{"polish current tag", "PLK", "pl"},
		{"czech legacy tag", "CES", "cs"},
		{"czech current tag", "CSY", "cs"},
		{"slovenian", "SLV", "sl"},
		{"slovak", "SKY", "sk"},
		{"known tag passes through", "ENG", "eng"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := shapingLanguageOverride(testCase.tag); got != testCase.want {
				t.Fatalf("shapingLanguageOverride(%q) = %q, want %q", testCase.tag, got, testCase.want)
			}
		})
	}
}

// TestShapingInputCarriesLanguage observes the production shaping.Input
// builder: an override lands on Input.Language, and no override leaves the
// zero value so default shaping is unchanged.
func TestShapingInputCarriesLanguage(t *testing.T) {
	t.Parallel()

	fnt := bundledDejaVu(t)

	face, ok := fnt.gotextFace()
	if !ok {
		t.Fatal("go-text face unavailable")
	}

	withLang := shapingInput([]rune{'б'}, face, nil, shapingLanguageOverride("SRB"))
	if withLang.Language != language.Language("sr") {
		t.Fatalf("shaping input language = %q, want sr", withLang.Language)
	}

	noLang := shapingInput([]rune{'б'}, face, nil, shapingLanguageOverride(""))
	if noLang.Language != "" {
		t.Fatalf("empty override language = %q, want zero", noLang.Language)
	}
}

// TestShapeTextFontWithFeaturesLanguageAppliesLangSys proves the override is
// observably effectful with a bundled face, not just plumbed: bundled
// DejaVuSans GSUB has a Serbian (SRB) language system whose locl feature
// substitutes the glyph for U+0431, and that glyph reverse-maps to U+F6C5.
// Other bundled faces carry language systems this path cannot reverse-map
// (Liberation's SRB/MKD alt glyphs are not in its cmap), so for those tags
// the override stays shaper-visible through shaping.Input.Language only.
func TestShapeTextFontWithFeaturesLanguageAppliesLangSys(t *testing.T) {
	t.Parallel()

	fnt := bundledDejaVu(t)

	const cyrillicBe = "б" // U+0431

	plain := ShapeTextFontWithFeaturesLanguage(cyrillicBe, fnt, nil, "")
	if plain != cyrillicBe {
		t.Fatalf("no override = %q, want the input rune", plain)
	}

	serbian := ShapeTextFontWithFeaturesLanguage(cyrillicBe, fnt, nil, "SRB")
	if serbian == plain {
		t.Fatalf("SRB override did not change shaped output: %q", serbian)
	}

	if len([]rune(serbian)) != 1 {
		t.Fatalf("SRB output = %q, want one rune", serbian)
	}
}

// TestContentTextShowLanguageShapesWithOverride drives the writer entry point:
// TextShowLanguage reaches the shaper with the override, while TextShow keeps
// the unchanged default path.
func TestContentTextShowLanguageShapesWithOverride(t *testing.T) {
	t.Parallel()

	fnt := bundledDejaVu(t)

	render := func(lang string) string {
		content := NewContent()
		content.UseEmbeddedFont("F0", fnt)
		content.BeginText()
		content.SetFont("F0", 12)
		content.TextShowLanguage("б", lang)
		content.EndText()

		return string(content.Bytes())
	}

	plain := render("")
	serbian := render("SRB")

	if !strings.Contains(plain, "<0431>") {
		t.Fatalf("plain content missing nominal CID: %q", plain)
	}

	if !strings.Contains(serbian, "<F6C5>") {
		t.Fatalf("SRB content missing substituted CID: %q", serbian)
	}
}
