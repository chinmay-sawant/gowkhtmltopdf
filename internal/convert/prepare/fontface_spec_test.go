package prepare_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert/prepare"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/load"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// TestMergeFontFacesCarriesUnicodeRanges proves the @font-face descriptors
// parsed by internal/css reach the PDF registry: the latin-ext face is
// registered first (the learncpp.com order) and the latin face second, and the
// primary lookup must return the declared latin face while the per-code-point
// lookup returns each face where its unicode-range applies. The two files have
// different advance widths, so the selected face is observable.
func TestMergeFontFacesCarriesUnicodeRanges(t *testing.T) {
	t.Parallel()

	assetsDir := filepath.Join("..", "..", "..", "internal", "pdf", "assets")
	server := httptest.NewServer(http.FileServer(http.Dir(assetsDir)))
	t.Cleanup(server.Close)

	loader, err := load.NewLoaderWithError(settings.LoadGlobal{})
	if err != nil {
		t.Fatalf("new loader: %v", err)
	}

	resources := prepare.NewResourceContext(loader, server.URL+"/", settings.DefaultLoadPage())

	sheets := []*css.Stylesheet{{FontFaces: []css.FontFace{
		{
			Family:        "Custom",
			Src:           `url(LiberationMono-Regular.ttf) format("truetype")`,
			Weight:        400,
			UnicodeRanges: []css.UnicodeRange{{Lo: 0x0100, Hi: 0x02AF}},
		},
		{
			Family:        "Custom",
			Src:           `url(LiberationSans-Regular.ttf) format("truetype")`,
			Weight:        400,
			UnicodeRanges: []css.UnicodeRange{{Lo: 0x0000, Hi: 0x00FF}, {Lo: 0x2000, Hi: 0x206F}},
		},
	}}}

	var log bytes.Buffer

	registry := resources.MergeFontFaces(t.Context(), pdf.NewRegistry(), sheets, 1, &log)
	if registry == nil {
		t.Fatalf("MergeFontFaces returned nil; log=%q", log.String())
	}

	mono := readTestFace(t, filepath.Join(assetsDir, "LiberationMono-Regular.ttf"))
	sans := readTestFace(t, filepath.Join(assetsDir, "LiberationSans-Regular.ttf"))

	if mono.Advance('m') == sans.Advance('m') {
		t.Fatal("fixture faces have equal advance widths; this test cannot tell them apart")
	}

	primary := registry.Lookup([]string{"Custom"}, 400, false)
	assertFaceWidth(t, "primary face", primary, sans.Advance('m'))

	latinRune := registry.LookupRune([]string{"Custom"}, 400, false, 'p')
	assertFaceWidth(t, "LookupRune('p')", latinRune, sans.Advance('m'))

	extRune := registry.LookupRune([]string{"Custom"}, 400, false, 0x0100)
	assertFaceWidth(t, "LookupRune(U+0100)", extRune, mono.Advance('m'))
}

// assertFaceWidth fails when the face is nil or its 'm' advance does not match
// want, which is how this test tells the mono and sans files apart.
func assertFaceWidth(t *testing.T, label string, face *pdf.Font, want float64) {
	t.Helper()

	if face == nil {
		t.Fatalf("%s: face is nil", label)
	}

	if face.Advance('m') != want {
		t.Errorf("%s: advance('m') = %v, want %v", label, face.Advance('m'), want)
	}
}

func readTestFace(t *testing.T, path string) *pdf.Font {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	face, err := pdf.ParseFontBytes(data)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	return face
}
