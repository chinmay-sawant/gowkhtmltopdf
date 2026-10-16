package cli

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

func TestEngineFlagSelectsBlinkless(t *testing.T) {
	t.Parallel()

	cmd := parsePDF(t, "--engine", "blinkless", "-o", outPDF, "page.html")
	if cmd.Global.Engine != settings.EngineBlinkless {
		t.Fatalf("engine = %q, want %q", cmd.Global.Engine, settings.EngineBlinkless)
	}

	if _, err := Parse([]string{"--engine", "webkit", "-o", outPDF, "page.html"}, ModePDF); err == nil {
		t.Fatal("Parse accepted an unknown engine")
	}
}
