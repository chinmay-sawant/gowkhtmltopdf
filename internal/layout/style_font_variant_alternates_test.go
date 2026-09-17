//nolint:all // targeted unit tests for font-variant-alternates
package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

func TestFontVariantAlternatesMapsToHist(t *testing.T) {
	t.Parallel()

	style := initialStyle()
	if !applyFontFeatureProps(&style, "font-variant-alternates", "historical-forms", 12, nil, nil, false) {
		t.Fatal("font-variant-alternates not owned")
	}

	if style.FontVariantAlternates != "historical-forms" {
		t.Fatalf("stored = %q", style.FontVariantAlternates)
	}

	got := fontShapingFeatureSettings(&style)
	if got != `"hist" 1` {
		t.Fatalf("features = %q, want hist 1", got)
	}

	feats := pdf.ParseFontFeatureSettings(got)
	if len(feats) != 1 || feats[0].Tag.String() != "hist" || feats[0].Value != 1 {
		t.Fatalf("ParseFontFeatureSettings(%q) = %+v, want hist=1", got, feats)
	}
}

func TestFontVariantAlternatesStylisticMapsToSalt(t *testing.T) {
	t.Parallel()

	style := initialStyle()
	if !applyFontFeatureProps(&style, "font-variant-alternates", "stylistic(1)", 12, nil, nil, false) {
		t.Fatal("stylistic(1) not owned")
	}

	got := fontShapingFeatureSettings(&style)
	if got != `"salt" 1` {
		t.Fatalf("features = %q, want salt 1", got)
	}

	feats := pdf.ParseFontFeatureSettings(got)
	if len(feats) != 1 || feats[0].Tag.String() != "salt" || feats[0].Value != 1 {
		t.Fatalf("ParseFontFeatureSettings(%q) = %+v, want salt=1", got, feats)
	}
}

func TestFontVariantAlternatesStylesetAndSwash(t *testing.T) {
	t.Parallel()

	style := initialStyle()
	if !applyFontFeatureProps(&style, "font-variant-alternates", "styleset(2) swash(1)", 12, nil, nil, false) {
		t.Fatal("styleset/swash not owned")
	}

	got := fontShapingFeatureSettings(&style)
	if !strings.Contains(got, `"ss02" 1`) || !strings.Contains(got, `"swsh" 1`) {
		t.Fatalf("features = %q, want ss02 and swsh", got)
	}
}

func TestFontVariantAlternatesReachShaper(t *testing.T) {
	t.Parallel()

	doc := parseTestHTML(t, `<html><body style="margin:0">`+
		`<p style="font-variant-alternates: historical-forms">histforms</p>`+
		`<p style="font-variant-alternates: stylistic(1)">saltforms</p>`+
		`<p>plain</p>`+
		`</body></html>`)

	res, err := Layout(doc, Options{Width: 500, Height: 400})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{
		"histforms": `"hist" 1`,
		"saltforms": `"salt" 1`,
		"plain":     "",
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

func TestFontVariantAlternatesRejectsJunk(t *testing.T) {
	t.Parallel()

	style := initialStyle()
	if !applyFontFeatureProps(&style, "font-variant-alternates", "not-a-feature", 12, nil, nil, false) {
		t.Fatal("must own the property even when dropping junk")
	}

	if style.FontVariantAlternates != "normal" {
		t.Fatalf("junk stored = %q, want normal", style.FontVariantAlternates)
	}
}
