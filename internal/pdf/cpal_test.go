package pdf

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf/assets"
)

func TestPaletteDemoTTFHasCPAL(t *testing.T) {
	t.Parallel()

	raw, err := PaletteDemoTTF(assets.LiberationSansRegular())
	if err != nil {
		t.Fatalf("PaletteDemoTTF: %v", err)
	}

	face, err := ParseTTF(raw)
	if err != nil {
		t.Fatalf("ParseTTF: %v", err)
	}

	if !face.HasColorPalette() {
		t.Fatal("want COLR+CPAL")
	}

	assertPaletteDemoFamily(t, face)
	assertPaletteDemoNormal(t, face)
	assertPaletteDemoColors(t, face)
}

func assertPaletteDemoFamily(t *testing.T, face *Font) {
	t.Helper()

	names := face.LoadNames()
	found := false

	for _, name := range names {
		if name == PaletteDemoFamily {
			found = true
		}
	}

	if !found {
		t.Fatalf("family names %v, want %q", names, PaletteDemoFamily)
	}
}

func assertPaletteDemoNormal(t *testing.T, face *Font) {
	t.Helper()

	normalR, normalG, normalB, normalOK := face.PaletteSolidFill("normal")
	if normalOK || normalR != 0 || normalG != 0 || normalB != 0 {
		t.Fatal("normal must not override CSS color")
	}
}

func assertPaletteDemoColors(t *testing.T, face *Font) {
	t.Helper()

	lightR, lightG, lightB, lightOK := face.PaletteSolidFill("light")
	if !lightOK {
		t.Fatal("light palette missing")
	}

	darkR, darkG, darkB, darkOK := face.PaletteSolidFill("dark")
	if !darkOK {
		t.Fatal("dark palette missing")
	}

	zeroR, zeroG, zeroB, zeroOK := face.PaletteSolidFill("0")
	if !zeroOK {
		t.Fatal("palette 0 missing")
	}

	if lightR == darkR && lightG == darkG && lightB == darkB {
		t.Fatal("light and dark fills must differ")
	}

	if zeroR != lightR || zeroG != lightG || zeroB != lightB {
		t.Fatalf("palette 0 = (%v,%v,%v), want light (%v,%v,%v)", zeroR, zeroG, zeroB, lightR, lightG, lightB)
	}
}

func TestWritePaletteDemoTTF(t *testing.T) {
	t.Parallel()

	if os.Getenv("WRITE_PALETTE_DEMO") == "" {
		t.Skip("set WRITE_PALETTE_DEMO=1 to refresh testdata")
	}

	raw, err := PaletteDemoTTF(assets.LiberationSansRegular())
	if err != nil {
		t.Fatalf("PaletteDemoTTF: %v", err)
	}

	dest := filepath.Join("..", "..", "testdata", "fonts", "implemented-audit", "PaletteDemo-Regular.ttf")
	if err := os.WriteFile(dest, raw, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
