package pdf //nolint:testpackage // registry tie-breaks require private face identity fields.

import "testing"

//nolint:wsl // registry permutations and assertions are intentionally adjacent.
func TestRegistryFindWithGlyphUsesStableFaceIdentityTieBreak(t *testing.T) {
	t.Parallel()

	base := testFont(t)
	first, err := ParseTTF(append([]byte(nil), base.data...))
	if err != nil {
		t.Fatalf("ParseTTF first: %v", err)
	}
	first.PostScriptName = "FaceA"
	first.fingerprint[0] = 1
	second, err := ParseTTF(append([]byte(nil), base.data...))
	if err != nil {
		t.Fatalf("ParseTTF second: %v", err)
	}
	second.PostScriptName = "FaceB"
	second.fingerprint[0] = 2

	left := NewRegistry()
	left.AddFamilyAlias("second", second)
	left.AddFamilyAlias("first", first)

	right := NewRegistry()
	right.AddFamilyAlias("first", first)
	right.AddFamilyAlias("second", second)

	if got := left.FindWithGlyph('A', 400, false); got != first {
		if got == nil {
			t.Fatal("left registry returned nil")

			return
		}
		t.Fatalf("left registry selected %q, want FaceA", got.PostScriptName)
	}
	if got := right.FindWithGlyph('A', 400, false); got != first {
		if got == nil {
			t.Fatal("right registry returned nil")

			return
		}
		t.Fatalf("right registry selected %q, want FaceA", got.PostScriptName)
	}
}
