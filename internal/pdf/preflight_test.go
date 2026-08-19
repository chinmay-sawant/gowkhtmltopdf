package pdf //nolint:testpackage // white-box PreflightEmbed

import "testing"

func TestPreflightEmbedLiberationLatinAndUnicode(t *testing.T) {
	t.Parallel()

	faces, err := LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	if err := PreflightEmbed(faces.Regular, []rune("Hello PDF")); err != nil {
		t.Fatalf("latin preflight: %v", err)
	}

	if err := PreflightEmbed(faces.UnicodeFallback, []rune("안녕★")); err != nil {
		t.Fatalf("unicode preflight: %v", err)
	}

	if err := PreflightEmbed(nil, []rune("A")); err == nil {
		t.Fatal("nil font must fail preflight")
	}
}
