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

	_, _, _, ok := face.PaletteSolidFill("normal")
	if ok {
		t.Fatal("normal must not override CSS color")
	}

	lr, lg, lb, ok := face.PaletteSolidFill("light")
	if !ok {
		t.Fatal("light palette missing")
	}

	dr, dg, db, ok := face.PaletteSolidFill("dark")
	if !ok {
		t.Fatal("dark palette missing")
	}

	zr, zg, zb, ok := face.PaletteSolidFill("0")
	if !ok {
		t.Fatal("palette 0 missing")
	}

	if lr == dr && lg == dg && lb == db {
		t.Fatal("light and dark fills must differ")
	}

	if zr != lr || zg != lg || zb != lb {
		t.Fatalf("palette 0 = (%v,%v,%v), want light (%v,%v,%v)", zr, zg, zb, lr, lg, lb)
	}
}

func TestWritePaletteDemoTTF(t *testing.T) {
	if os.Getenv("WRITE_PALETTE_DEMO") == "" {
		t.Skip("set WRITE_PALETTE_DEMO=1 to refresh testdata")
	}

	raw, err := PaletteDemoTTF(assets.LiberationSansRegular())
	if err != nil {
		t.Fatalf("PaletteDemoTTF: %v", err)
	}

	dest := filepath.Join("..", "..", "testdata", "fonts", "implemented-audit", "PaletteDemo-Regular.ttf")
	if err := os.WriteFile(dest, raw, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
