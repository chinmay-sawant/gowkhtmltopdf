//nolint:testpackage // benchmark exercises the registry's private face list.
package pdf

import "testing"

func BenchmarkFontNameLoadingCached(b *testing.B) {
	faces, err := LoadDefaultFaces()
	if err != nil {
		b.Fatal(err)
	}

	font := faces.Regular

	b.ReportAllocs()

	for b.Loop() {
		if len(font.LoadNames()) == 0 {
			b.Fatal("font has no cached names")
		}
	}
}

func BenchmarkRegistryFindWithGlyph(b *testing.B) {
	faces, err := LoadDefaultFaces()
	if err != nil {
		b.Fatal(err)
	}

	registry := NewRegistry()
	for _, face := range []*Font{
		faces.Regular, faces.Bold, faces.Italic, faces.BoldItalic,
		faces.Serif, faces.SerifBold, faces.Mono, faces.MonoBold,
		faces.UnicodeFallback, faces.UnicodeFallbackBold,
	} {
		registry.AddFont(face)
	}

	b.ReportAllocs()

	for b.Loop() {
		if registry.FindWithGlyph('★', 400, false) == nil {
			b.Fatal("registry did not find a fallback glyph")
		}
	}
}
