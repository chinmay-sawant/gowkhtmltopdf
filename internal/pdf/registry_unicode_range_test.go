package pdf

import "testing"

// subsetFaceWithoutBasicLatin returns a parsed face whose cmap carries only
// U+0100 (Latin Extended-A). It mirrors the Google Fonts latin-ext files,
// which have no A/a/0-9 glyphs, while keeping real Liberation outlines.
func subsetFaceWithoutBasicLatin(t *testing.T) *Font {
	t.Helper()

	sub, err := subsetFont(testFont(t), []rune{0x0100}, subsetUnicode)
	if err != nil {
		t.Fatalf("subsetFont: %v", err)
	}

	face, err := ParseTTF(sub.data)
	if err != nil {
		t.Fatalf("ParseTTF subset: %v", err)
	}

	face.PostScriptName = "OpenSansLatinExt"

	return face
}

// TestRegistryLookupSkipsFaceWithoutPrimaryGlyphs pins the learncpp-1
// ordering: the latin-ext @font-face is declared first and the latin face
// second, so an order-only tie hands ordinary text to a face that has no
// basic Latin glyphs. Without unicode-range descriptors the glyph probes
// decide, and LookupRune still picks each face per code point.
func TestRegistryLookupSkipsFaceWithoutPrimaryGlyphs(t *testing.T) {
	t.Parallel()

	ext := subsetFaceWithoutBasicLatin(t)

	for _, r := range "programs" {
		if ext.GlyphID(r) != 0 {
			t.Fatalf("fixture face unexpectedly covers %q", r)
		}
	}

	latin := testFont(t)

	reg := NewRegistry()
	// Registration order mirrors the real page: latin-ext first.
	reg.AddFamilyAlias("Open Sans", ext)
	reg.AddFamilyAlias("Open Sans", latin)

	got := reg.Lookup([]string{"Open Sans"}, 400, false)
	if got == nil {
		t.Fatal("Lookup returned nil")
	}

	if got != latin {
		t.Fatalf("Lookup picked %q, want the latin face", got.PostScriptName)
	}

	for _, r := range "programs" {
		face := reg.LookupRune([]string{"Open Sans"}, 400, false, r)
		if face != latin {
			t.Fatalf("LookupRune(%q) picked %v, want the latin face", r, face)
		}
	}

	if face := reg.LookupRune([]string{"Open Sans"}, 400, false, 0x0100); face != ext {
		t.Fatalf("LookupRune(U+0100) picked %v, want the latin-ext face", face)
	}
}

// TestRegistryLookupRuneHonorsDeclaredRanges registers the same two face
// orderings with the Google Fonts unicode-range descriptors. The primary
// lookup must prefer the face whose declared range covers ordinary text, and
// LookupRune must return the declared face per code point (or nil when no
// face declares the code point).
func TestRegistryLookupRuneHonorsDeclaredRanges(t *testing.T) {
	t.Parallel()

	ext := subsetFaceWithoutBasicLatin(t)
	latin := testFont(t)

	reg := NewRegistry()
	// Ranges mirror the Open Sans Google Fonts split: latin-ext first, then
	// latin (which also carries general punctuation such as U+2019).
	reg.AddFamilyAliasSpec("Open Sans", ext, FaceSpec{
		Ranges: []UnicodeRange{{Lo: 0x0100, Hi: 0x02AF}},
	})
	reg.AddFamilyAliasSpec("Open Sans", latin, FaceSpec{
		Ranges: []UnicodeRange{{Lo: 0x0000, Hi: 0x00FF}, {Lo: 0x2000, Hi: 0x206F}},
	})

	if got := reg.Lookup([]string{"Open Sans"}, 400, false); got != latin {
		t.Fatalf("Lookup picked %v, want the declared latin face", got)
	}

	for _, r := range "programs" {
		if face := reg.LookupRune([]string{"Open Sans"}, 400, false, r); face != latin {
			t.Fatalf("LookupRune(%q) picked %v, want the latin face", r, face)
		}
	}

	if face := reg.LookupRune([]string{"Open Sans"}, 400, false, 0x0100); face != ext {
		t.Fatalf("LookupRune(U+0100) picked %v, want the latin-ext face", face)
	}

	if face := reg.LookupRune([]string{"Open Sans"}, 400, false, 0x2019); face != latin {
		t.Fatalf("LookupRune(U+2019) picked %v, want the latin face", face)
	}

	if face := reg.LookupRune([]string{"Open Sans"}, 400, false, '★'); face != nil {
		t.Fatalf("LookupRune(U+2605) picked %v, want nil outside both declared ranges", face)
	}
}

// TestRegistryLookupUsesDeclaredWeightAndStyle covers the other two
// @font-face descriptors: a face whose descriptor says 700 outranks one whose
// OS/2 table says 700, and an explicit font-style:normal overrides a file's
// italic macStyle bit.
func TestRegistryLookupUsesDeclaredWeightAndStyle(t *testing.T) {
	t.Parallel()

	declaredBold := weightedFont(t, 400)
	fileBold := weightedFont(t, 700)

	reg := NewRegistry()
	reg.AddFamilyAliasSpec("Weighted", declaredBold, FaceSpec{Weight: 700})
	reg.AddFamilyAliasSpec("Weighted", fileBold, FaceSpec{Weight: 400})

	if got := reg.Lookup([]string{"Weighted"}, 700, false); got != declaredBold {
		t.Fatalf("Lookup(700) picked %q, want the face declared at 700", got.PostScriptName)
	}

	if got := reg.Lookup([]string{"Weighted"}, 400, false); got != fileBold {
		t.Fatalf("Lookup(400) picked %q, want the face declared at 400", got.PostScriptName)
	}

	upright := testFontWithStyle(t, 0)
	fileItalic := testFontWithStyle(t, 2)

	styled := NewRegistry()
	// Declared italic on an upright file, declared normal on an italic file.
	styled.AddFamilyAliasSpec("Styled", upright, FaceSpec{Italic: true, StyleSet: true})
	styled.AddFamilyAliasSpec("Styled", fileItalic, FaceSpec{Italic: false, StyleSet: true})

	if got := styled.Lookup([]string{"Styled"}, 400, true); got != upright {
		t.Fatalf("Lookup(italic) picked the file's italic face, want the declared italic face")
	}

	if got := styled.Lookup([]string{"Styled"}, 400, false); got != fileItalic {
		t.Fatalf("Lookup(upright) picked the file's upright face, want the declared normal face")
	}
}
