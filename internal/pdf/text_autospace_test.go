package pdf

import (
	"strings"
	"testing"
)

func TestTextShowLanguageFeaturesWithAutospaceEmitsPairAdjustment(t *testing.T) {
	t.Parallel()

	font := loadDejaVu(t)
	content := NewContent()
	content.UseEmbeddedFont("F0", font)
	content.BeginText()
	content.SetFont("F0", 12)
	content.TextShowLanguageFeaturesWithAutospace("汉A汉", "", "", 1.5)
	content.EndText()

	stream := string(content.Bytes())

	if !strings.Contains(stream, "TJ") {
		t.Fatalf("autospace text stream = %q, want TJ adjustment array", stream)
	}

	adjustments := strings.Count(stream, "-125")
	if adjustments != 2 {
		t.Fatalf("autospace text stream = %q, got %d -125 adjustments, want 2", stream, adjustments)
	}
}
