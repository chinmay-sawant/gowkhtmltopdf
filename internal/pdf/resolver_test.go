//nolint:testpackage // tests reach into unexported fingerprint / helpers
package pdf

import (
	"bytes"
	"strings"
	"testing"
)

func TestFontResolverGenericsAndLiberationNames(t *testing.T) {
	t.Parallel()

	faces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	resolver := NewFontResolver(faces, nil)

	if got := resolver.ResolveFamilyStyle([]string{"serif"}, 400, false); got != faces.Serif {
		t.Fatal("serif → Liberation Serif")
	}

	if got := resolver.ResolveFamilyStyle([]string{"sans-serif"}, 700, false); got != faces.Bold {
		t.Fatal("sans-serif bold → Liberation Sans Bold")
	}

	if got := resolver.ResolveFamilyStyle([]string{"monospace"}, 400, true); got != faces.MonoItalic {
		t.Fatal("monospace italic → Liberation Mono Italic")
	}

	if got := resolver.ResolveFamilyStyle([]string{"system-ui"}, 400, false); got != faces.UnicodeFallback {
		t.Fatal("system-ui → DejaVu")
	}

	if got := resolver.ResolveFamilyStyle([]string{"Liberation Serif"}, 700, true); got != faces.SerifBoldItalic {
		t.Fatal("liberation serif name → bundled BI")
	}
}

func TestFontResolverNoLegacyAliases(t *testing.T) {
	t.Parallel()

	faces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	resolver := NewFontResolver(faces, nil)

	if got := resolver.ResolveFamilyStyle([]string{"Georgia"}, 400, false); got != faces.Regular {
		t.Fatal("Georgia alone without registry → Liberation Sans terminal default, not Serif alias")
	}

	if got := resolver.ResolveFamilyStyle([]string{"Georgia", "serif"}, 400, false); got != faces.Serif {
		t.Fatal("Georgia, serif → Liberation Serif via generic")
	}

	if got := resolver.ResolveFamilyStyle([]string{"Arial", "sans-serif"}, 700, false); got != faces.Bold {
		t.Fatal("Arial, sans-serif → Liberation Sans Bold via generic")
	}

	if got := resolver.ResolveFamilyStyle([]string{"Courier New", "monospace"}, 700, false); got != faces.MonoBold {
		t.Fatal("Courier New, monospace → Mono Bold via generic")
	}
}

func TestFontResolverExactRegisteredFamily(t *testing.T) {
	t.Parallel()

	faces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	reg := NewRegistry()
	reg.AddFamilyAlias("Georgia", faces.Mono)

	resolver := NewFontResolver(faces, reg)
	if got := resolver.ResolveFamilyStyle([]string{"Georgia", "serif"}, 400, false); got != faces.Mono {
		t.Fatal("registered Georgia wins over later serif generic")
	}
}

func TestFontResolverAuthorStackContinuation(t *testing.T) {
	t.Parallel()

	faces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	reg := NewRegistry()
	reg.AddFamilyAlias("CustomStack", faces.SerifBold)

	resolver := NewFontResolver(faces, reg)
	got := resolver.ResolveFamilyStyle([]string{"Missing", "CustomStack", "sans-serif"}, 400, false)

	if got != faces.SerifBold {
		t.Fatal("missing named family continues to next author token")
	}
}

func TestFontResolverMarkUnavailableContinues(t *testing.T) {
	t.Parallel()

	faces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	var warned strings.Builder

	resolver := NewFontResolver(faces, nil)
	resolver.Warn = func(msg string) { warned.WriteString(msg) }

	resolver.MarkUnavailable(faces.Serif, "embed preflight failed")

	if got := resolver.ResolveFamilyStyle([]string{"serif", "sans-serif"}, 400, false); got != faces.Regular {
		t.Fatal("unavailable serif continues to sans-serif")
	}

	if warned.Len() == 0 {
		t.Fatal("expected unavailable diagnostic")
	}
}

func TestFontResolverRuneFallbackKeepsPrimaryWhenCovered(t *testing.T) {
	t.Parallel()

	faces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	resolver := NewFontResolver(faces, nil)
	primary := resolver.ResolveFamilyStyle([]string{"serif"}, 400, false)

	if primary == nil {
		t.Fatal("primary")
	}

	if got := resolver.ResolveRune([]string{"serif"}, 400, false, 'A', primary); got != primary {
		t.Fatal("Latin glyph must stay on primary")
	}

	star := resolver.ResolveRune([]string{"serif"}, 400, false, '★', primary)
	if star == nil || star.GlyphID('★') == 0 {
		t.Fatal("missing glyph must fall back to a covering face")
	}

	if star == primary {
		t.Fatal("star should not stay on primary Liberation Serif")
	}

	if star != faces.UnicodeFallback && star != faces.Regular {
		// DejaVu is the documented Unicode fallback; Liberation Sans may also cover some symbols.
		if star.GlyphID('★') == 0 {
			t.Fatalf("fallback face %q lacks star", star.PostScriptName)
		}
	}
}

func TestFontResolverDuplicateFamilyTieBreak(t *testing.T) {
	t.Parallel()

	base, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	raw := bytes.Clone(base.Regular.data)

	first, err := ParseTTF(append([]byte(nil), raw...))
	if err != nil {
		t.Fatal(err)
	}

	second, err := ParseTTF(append([]byte(nil), raw...))
	if err != nil {
		t.Fatal(err)
	}

	first.PostScriptName = "DupA"
	second.PostScriptName = "DupB"
	// Same style score; distinct fingerprints → stable identity order.
	first.fingerprint[0] = 1
	second.fingerprint[0] = 2

	left := NewRegistry()
	left.AddFamilyAlias("DupFace", second)
	left.AddFamilyAlias("DupFace", first)

	right := NewRegistry()
	right.AddFamilyAlias("DupFace", first)
	right.AddFamilyAlias("DupFace", second)

	gotLeft := NewFontResolver(base, left).ResolveFamilyStyle([]string{"DupFace"}, 400, false)
	gotRight := NewFontResolver(base, right).ResolveFamilyStyle([]string{"DupFace"}, 400, false)

	if gotLeft == nil || gotRight == nil {
		t.Fatal("expected DupFace resolution")
	}

	if gotLeft.PostScriptName != gotRight.PostScriptName {
		t.Fatalf("tie-break unstable: left=%q right=%q", gotLeft.PostScriptName, gotRight.PostScriptName)
	}

	if gotLeft.PostScriptName != "DupA" {
		t.Fatalf("want lower fingerprint DupA, got %q", gotLeft.PostScriptName)
	}
}

func TestFontResolverCacheIsolationSameDisplayName(t *testing.T) {
	t.Parallel()

	faces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	faceA, err := ParseTTF(bytes.Clone(faces.Regular.data))
	if err != nil {
		t.Fatal(err)
	}

	faceB, err := ParseTTF(bytes.Clone(faces.Bold.data))
	if err != nil {
		t.Fatal(err)
	}

	faceA.PostScriptName = "SameDisplay"
	faceB.PostScriptName = "SameDisplay"

	if bytes.Equal(faceA.fingerprint[:], faceB.fingerprint[:]) {
		t.Fatal("expected distinct fingerprints for different font bytes")
	}

	reg := NewRegistry()
	reg.AddFamilyAlias("SameDisplay", faceA)

	resolver := NewFontResolver(faces, reg)
	got := resolver.ResolveFamilyStyle([]string{"SameDisplay"}, 400, false)

	if got != faceA {
		t.Fatal("resolver must select the registered face instance")
	}

	reg2 := NewRegistry()
	reg2.AddFamilyAlias("SameDisplay", faceB)

	got2 := NewFontResolver(faces, reg2).ResolveFamilyStyle([]string{"SameDisplay"}, 400, false)
	if got2 != faceB {
		t.Fatal("distinct bytes under same display name must stay cache-isolated by face identity")
	}

	if bytes.Equal(got.fingerprint[:], got2.fingerprint[:]) {
		t.Fatal("selected faces must keep distinct fingerprints")
	}
}
