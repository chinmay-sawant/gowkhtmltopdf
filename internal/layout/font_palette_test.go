package layout

import (
	"math"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf/assets"
)

func TestFontPaletteCPALFillDiffersFromNormal(t *testing.T) {
	t.Parallel()

	reg := paletteDemoRegistry(t)
	cssSheet := sheet(t, `
.base { font-family: "Palette Demo"; font-size: 18pt; color: #111111; }
.normal { font-palette: normal; }
.dark { font-palette: dark; }
.light { font-palette: light; }
.zero { font-palette: 0; }
`)
	html := `<html><body>
<span class="base normal">Ag</span>
<span class="base dark">Ag</span>
<span class="base light">Ag</span>
<span class="base zero">Ag</span>
</body></html>`
	res := layoutHTMLRegistry(t, html, reg, cssSheet)

	ops := textOpsNamed(res, "Ag")
	if len(ops) < 4 {
		t.Fatalf("text ops = %d, want 4", len(ops))
	}

	normal, dark, light, zero := ops[0], ops[1], ops[2], ops[3]

	if colorsNear(normal, dark, 0.05) {
		t.Fatalf("dark fill RGB=(%.3f,%.3f,%.3f) must differ from normal=(%.3f,%.3f,%.3f)",
			dark.R, dark.G, dark.B, normal.R, normal.G, normal.B)
	}

	if colorsNear(light, dark, 0.05) {
		t.Fatalf("light fill RGB=(%.3f,%.3f,%.3f) must differ from dark=(%.3f,%.3f,%.3f)",
			light.R, light.G, light.B, dark.R, dark.G, dark.B)
	}

	if !colorsNear(zero, light, 0.02) {
		t.Fatalf("palette 0 RGB=(%.3f,%.3f,%.3f) should match light=(%.3f,%.3f,%.3f)",
			zero.R, zero.G, zero.B, light.R, light.G, light.B)
	}

	if normal.R > 0.2 || normal.G > 0.2 || normal.B > 0.2 {
		t.Fatalf("normal should keep CSS #111, got RGB=(%.3f,%.3f,%.3f)", normal.R, normal.G, normal.B)
	}
}

func TestFontPaletteStaticFaceKeepsCSSColor(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.n { font-size: 18pt; color: #111111; font-palette: normal; }
.d { font-size: 18pt; color: #111111; font-palette: dark; }
`)
	resN := layoutHTML(t, `<html><body><span class="n">Ag</span></body></html>`, cssSheet)
	resD := layoutHTML(t, `<html><body><span class="d">Ag</span></body></html>`, cssSheet)
	n := textOpRGB(t, resN, "Ag")
	d := textOpRGB(t, resD, "Ag")

	if !colorsNear(n, d, 0.01) {
		t.Fatalf("static face dark RGB=(%.3f,%.3f,%.3f) should match normal=(%.3f,%.3f,%.3f)",
			d.R, d.G, d.B, n.R, n.G, n.B)
	}
}

func paletteDemoRegistry(t *testing.T) *pdf.Registry {
	t.Helper()

	raw, err := pdf.PaletteDemoTTF(assets.LiberationSansRegular())
	if err != nil {
		t.Fatalf("PaletteDemoTTF: %v", err)
	}

	face, err := pdf.ParseTTF(raw)
	if err != nil {
		t.Fatalf("ParseTTF: %v", err)
	}

	if !face.HasColorPalette() {
		t.Fatal("Palette Demo face must advertise COLR+CPAL")
	}

	reg := pdf.NewRegistry()
	reg.AddFamilyAlias(pdf.PaletteDemoFamily, face)

	return reg
}

func textOpsNamed(res *Result, needle string) []Op {
	out := []Op{}

	for _, op := range res.Ops {
		if op.Kind == OpText && op.Text == needle {
			out = append(out, op)
		}
	}

	return out
}

func textOpRGB(t *testing.T, res *Result, needle string) Op {
	t.Helper()

	for _, op := range res.Ops {
		if op.Kind == OpText && op.Text == needle {
			return op
		}
	}

	t.Fatalf("text op %q not found", needle)

	return Op{}
}

func colorsNear(a, b Op, eps float64) bool {
	return math.Abs(a.R-b.R) <= eps && math.Abs(a.G-b.G) <= eps && math.Abs(a.B-b.B) <= eps
}
