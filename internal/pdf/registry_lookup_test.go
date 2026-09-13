package pdf

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

func TestFontFamilyKeysGenericsOnly(t *testing.T) {
	t.Parallel()

	if got := fontFamilyKeys("Georgia"); len(got) != 1 || got[0] != "georgia" {
		t.Fatalf("Georgia keys=%v want exact [georgia]", got)
	}

	if got := fontFamilyKeys("serif"); len(got) == 0 || got[0] != "liberation serif" {
		t.Fatalf("serif keys=%v want liberation serif first", got)
	}

	if got := fontFamilyKeys("sans-serif"); len(got) == 0 || got[0] != "liberation sans" {
		t.Fatalf("sans-serif keys=%v", got)
	}
}

// TestRegistryResolvesBundledDejaVuSans proves the default registry exposes
// the bundled DejaVu Sans fallback faces under their CSS family name without
// letting the generic sans-serif expansion switch to DejaVu, and that the
// resolved face still applies the Serbian (SRB) locl substitution.
func TestRegistryResolvesBundledDejaVuSans(t *testing.T) {
	t.Parallel()

	faces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatalf("LoadDefaultFaces: %v", err)
	}

	reg := RegistryFromGlobal(settings.DefaultPdfGlobal())
	if reg == nil {
		t.Fatal("RegistryFromGlobal returned nil")
	}

	regular := reg.Lookup([]string{"DejaVu Sans"}, 400, false)
	if regular != faces.UnicodeFallback {
		t.Fatalf("DejaVu Sans regular resolved to %v, want the bundled fallback face", regular)
	}

	bold := reg.Lookup([]string{"DejaVu Sans"}, 700, false)
	if bold != faces.UnicodeFallbackBold {
		t.Fatalf("DejaVu Sans bold resolved to %v, want the bundled fallback bold face", bold)
	}

	// The exact alias must not capture the generic sans-serif expansion.
	if got := reg.Lookup([]string{"sans-serif"}, 400, false); got != nil {
		t.Fatalf("sans-serif resolved to %v through the bundled exact alias", got)
	}

	plain := ShapeTextFontWithFeaturesLanguage("б", regular, nil, "")
	serbian := ShapeTextFontWithFeaturesLanguage("б", regular, nil, "SRB")

	if plain == serbian {
		t.Fatalf("SRB override did not change the shaped run for the fallback face: %q", plain)
	}
}

func TestLookupExactBeforeGeneric(t *testing.T) {
	t.Parallel()

	reg := ScanFontDirs(DefaultSystemFontDirs())
	if len(reg.byFamily["liberation serif"]) == 0 {
		t.Skip("Liberation Serif not installed")
	}

	if f := reg.Lookup([]string{"Georgia"}, 400, false); f != nil {
		for _, n := range f.FamilyNames() {
			if strings.Contains(strings.ToLower(n), "georgia") {
				t.Skip("system has Georgia; cannot assert miss")
			}
		}
	}

	fnt := reg.Lookup([]string{"Georgia", "Liberation Serif", "serif"}, 400, false)
	if fnt == nil {
		t.Fatal("expected Liberation Serif from author stack")
	}

	found := false

	for _, n := range fnt.FamilyNames() {
		low := strings.ToLower(n)
		if strings.Contains(low, "liberation") && strings.Contains(low, "serif") {
			found = true
		}
	}

	if !found {
		t.Fatalf("got face %v, want Liberation Serif", fnt.FamilyNames())
	}
}
