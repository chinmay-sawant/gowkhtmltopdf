package layout

import (
	"strings"
	"testing"
)

// TestFontLanguageOverrideReachesOpText proves the layout half of the
// font-language-override consumer: the parsed OpenType tag rides the emitted
// OpText payload, while "normal" and an undeclared property leave the accessor
// empty. internal/pdf maps the raw tag to BCP47 at shaping time.
func TestFontLanguageOverrideReachesOpText(t *testing.T) {
	t.Parallel()

	doc := parseTestHTML(t, `<html><body style="margin:0">`+
		`<p style="font-language-override: 'TRK'">turkish</p>`+
		`<p style="font-language-override: normal">plain</p>`+
		`<p>unset</p>`+
		`</body></html>`)

	res, err := Layout(doc, Options{Width: 500, Height: 400})
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]string{"turkish": "TRK", "plain": "", "unset": ""}
	found := make(map[string]bool, len(want))

	for i := range res.Ops {
		paintOp := &res.Ops[i]
		if paintOp.Kind != OpText {
			continue
		}

		for text, lang := range want {
			if !strings.Contains(paintOp.Text, text) {
				continue
			}

			found[text] = true

			if got := paintOp.TextLanguage(); got != lang {
				t.Errorf("OpText %q language = %q, want %q", text, got, lang)
			}
		}
	}

	for text := range want {
		if !found[text] {
			t.Errorf("no OpText emitted for %q", text)
		}
	}
}

// TestFontLanguageOverrideReachesShadowOpText covers the second OpText
// creation site: a text-shadow copy must carry the same override as its
// primary run so both paint with one shaping language.
func TestFontLanguageOverrideReachesShadowOpText(t *testing.T) {
	t.Parallel()

	doc := parseTestHTML(t, `<html><body style="margin:0">`+
		`<p style="font-language-override: 'SRB';text-shadow:1px 1px 2px #000">shadowed</p>`+
		`</body></html>`)

	res, err := Layout(doc, Options{Width: 500, Height: 400})
	if err != nil {
		t.Fatal(err)
	}

	seen := 0

	for i := range res.Ops {
		paintOp := &res.Ops[i]
		if paintOp.Kind != OpText || !strings.Contains(paintOp.Text, "shadowed") {
			continue
		}

		seen++

		if got := paintOp.TextLanguage(); got != "SRB" {
			t.Fatalf("shadow OpText %d language = %q, want SRB", seen, got)
		}
	}

	if seen < 2 {
		t.Fatalf("text-shadow emitted %d OpText ops, want primary plus shadow", seen)
	}
}
