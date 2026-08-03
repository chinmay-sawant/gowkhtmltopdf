package gowkhtmltopdf

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeHTML writes a temp HTML file and returns its path.
func writeHTML(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "input.html")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}
	return path
}

// newPDFConverter returns a converter wired for local-file input: global
// enable flag on and object-level block flag off (the frozen ACL pair).
func newPDFConverter(t *testing.T, path string) *Converter {
	t.Helper()
	c := NewConverter()
	if err := c.Global().Set("enablelocalfileaccess", "true"); err != nil {
		t.Fatalf("global set: %v", err)
	}
	obj := NewObjectSettings().SetPage(path)
	if err := obj.Set("load.blocklocalfileaccess", "false"); err != nil {
		t.Fatalf("object set: %v", err)
	}
	c.AddObject(obj)
	return c
}

func TestConvertPDFToBytes(t *testing.T) {
	path := writeHTML(t, "<html><body><h1>Hello</h1><p>world</p></body></html>")
	c := newPDFConverter(t, path)
	if err := c.Convert(context.Background()); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	data := c.Output()
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatalf("output does not start with %%PDF- (got %q)", data[:min(len(data), 8)])
	}
	if !bytes.Contains(data, []byte("/Type /Page")) {
		t.Error("output does not contain /Type /Page")
	}
	// A fresh converter has no output yet.
	if got := NewConverter().Output(); got != nil {
		t.Errorf("fresh Converter.Output() = %d bytes, want nil", len(got))
	}
}

func TestGlobalSettingsGetSetRoundTrip(t *testing.T) {
	cases := []struct{ name, value string }{
		{"size.pagesize", "Letter"},
		{"orientation", "Landscape"},
		{"margin.top", "12.5"},
		{"web.background", "false"},
		{"dpi", "150"},
		{"title", "Round Trip"},
		{"enablelocalfileaccess", "true"},
		{"header.center", "[title]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g := NewGlobalSettings()
			if err := g.Set(tc.name, tc.value); err != nil {
				t.Fatalf("Set(%q): %v", tc.name, err)
			}
			got, ok := g.Get(tc.name)
			if !ok {
				t.Fatalf("Get(%q) not found after Set", tc.name)
			}
			if got != tc.value {
				t.Errorf("Get(%q) = %q, want %q", tc.name, got, tc.value)
			}
		})
	}

	// Defaults are readable without any Set call.
	g := NewGlobalSettings()
	for name, want := range map[string]string{
		"size.pagesize":  "A4",
		"orientation":    "Portrait",
		"margin.top":     "10",
		"margin.bottom":  "10",
		"web.background": "true",
	} {
		got, ok := g.Get(name)
		if !ok || got != want {
			t.Errorf("default Get(%q) = %q, %v; want %q, true", name, got, ok, want)
		}
	}

	if _, ok := g.Get("bogus.key"); ok {
		t.Error("Get(bogus.key) found, want not found")
	}
	if err := g.Set("bogus.key", "x"); err == nil {
		t.Error("Set(bogus.key) succeeded, want error")
	}
	if err := g.Set("margin.top", "not-a-length"); err == nil {
		t.Error("Set(margin.top, not-a-length) succeeded, want error")
	}
}

func TestObjectSettingsGetSet(t *testing.T) {
	o := NewObjectSettings().SetPage("in.html")
	if got, _ := o.Get("page"); got != "in.html" {
		t.Errorf("Get(page) = %q, want in.html", got)
	}
	if err := o.Set("load.blocklocalfileaccess", "false"); err != nil {
		t.Fatalf("Set(load.blocklocalfileaccess): %v", err)
	}
	if got, ok := o.Get("load.blocklocalfileaccess"); !ok || got != "false" {
		t.Errorf("Get(load.blocklocalfileaccess) = %q, %v; want false, true", got, ok)
	}
	if err := o.Set("footer.right", "[page]"); err != nil {
		t.Fatalf("Set(footer.right): %v", err)
	}
	if got, ok := o.Get("footer.right"); !ok || got != "[page]" {
		t.Errorf("Get(footer.right) = %q, %v; want [page], true", got, ok)
	}
	if err := o.Set("unknown.key", "x"); err == nil {
		t.Error("Set(unknown.key) succeeded, want error")
	}
}

func TestConvertContextCancel(t *testing.T) {
	path := writeHTML(t, "<html><body><p>cancel me</p></body></html>")
	c := newPDFConverter(t, path)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.Convert(ctx); err == nil {
		t.Fatal("Convert with canceled context succeeded, want error")
	}
}

func TestConverterCallbacks(t *testing.T) {
	path := writeHTML(t, "<html><body><h1>callbacks</h1></body></html>")
	c := newPDFConverter(t, path)

	var phases, infos []string
	var progs []int
	c.OnPhase = func(p string) { phases = append(phases, p) }
	c.OnProgress = func(p int) { progs = append(progs, p) }
	c.OnInfo = func(line string) { infos = append(infos, line) }

	if err := c.Convert(context.Background()); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if len(phases) == 0 {
		t.Error("OnPhase never called")
	}
	if len(progs) == 0 {
		t.Error("OnProgress never called")
	}
	if len(infos) == 0 {
		t.Error("OnInfo never called")
	}
	if last := progs[len(progs)-1]; last != 100 {
		t.Errorf("last progress = %d, want 100", last)
	}
	if !bytes.HasPrefix(c.Output(), []byte("%PDF-")) {
		t.Error("Output after Convert is not a PDF")
	}
}

func TestImageConverterPNG(t *testing.T) {
	path := writeHTML(t, `<html><body><div style="width:120px;height:80px;background-color:#336699"></div></body></html>`)
	c := NewImageConverter()
	if err := c.Global().Set("enablelocalfileaccess", "true"); err != nil {
		t.Fatalf("global set: %v", err)
	}
	c.AddObject(path)
	if err := c.Object().Set("load.blocklocalfileaccess", "false"); err != nil {
		t.Fatalf("object set: %v", err)
	}
	if err := c.Set("width", "200"); err != nil {
		t.Fatalf("Set(width): %v", err)
	}
	if err := c.Convert(context.Background()); err != nil {
		t.Fatalf("Convert: %v", err)
	}

	img, err := png.Decode(bytes.NewReader(c.Output()))
	if err != nil {
		t.Fatalf("output is not a decodable PNG: %v", err)
	}
	b := img.Bounds()
	if b.Dx() != 200 {
		t.Errorf("width = %d, want 200 (viewport)", b.Dx())
	}
	if b.Dy() < 80 {
		t.Errorf("height = %d, want >= 80 (content + body margins)", b.Dy())
	}
	// The 120x80 #336699 div sits below the 8px body margin.
	if got := pixelAt(img, 60, 40); got != (color.NRGBA{R: 0x33, G: 0x66, B: 0x99, A: 0xff}) {
		t.Errorf("div pixel = %v, want #336699", got)
	}
}

func TestImageConverterJPEG(t *testing.T) {
	path := writeHTML(t, `<html><body><p>hello image</p></body></html>`)
	c := NewImageConverter()
	if err := c.Global().Set("enablelocalfileaccess", "true"); err != nil {
		t.Fatalf("global set: %v", err)
	}
	c.AddObject(path)
	if err := c.Object().Set("load.blocklocalfileaccess", "false"); err != nil {
		t.Fatalf("object set: %v", err)
	}
	if err := c.Set("format", "jpg"); err != nil {
		t.Fatalf("Set(format): %v", err)
	}
	if err := c.Convert(context.Background()); err != nil {
		t.Fatalf("Convert: %v", err)
	}
	if _, err := jpeg.Decode(bytes.NewReader(c.Output())); err != nil {
		t.Fatalf("output is not a decodable JPEG: %v", err)
	}
}

func TestVersion(t *testing.T) {
	v := Version()
	if !strings.Contains(v, LibraryVersion) {
		t.Errorf("Version() = %q, want it to contain LibraryVersion %q", v, LibraryVersion)
	}
}

func TestImageConverterNeedsPage(t *testing.T) {
	c := NewImageConverter()
	if err := c.Convert(context.Background()); err == nil {
		t.Fatal("Convert without a page succeeded, want error")
	}
}

func TestConverterNeedsObject(t *testing.T) {
	if err := NewConverter().Convert(context.Background()); err == nil {
		t.Fatal("Convert without objects succeeded, want error")
	}
}

// pixelAt returns the NRGBA colour at (x, y).
func pixelAt(img image.Image, x, y int) color.NRGBA {
	return color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
}
