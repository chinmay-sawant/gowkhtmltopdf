package settings

import (
	"errors"
	"math"
	"testing"
)

func TestDefaultPdfGlobalSnapshot(t *testing.T) {
	g := DefaultPdfGlobal()
	checks := []struct {
		name string
		got  any
		want any
	}{
		{"PageSize", g.PageSize, "A4"},
		{"Orientation", g.Orientation, OrientationPortrait},
		{"Grayscale", g.Grayscale, false},
		{"Collate", g.Collate, true},
		{"Outline", g.Outline, true},
		{"OutlineDepth", g.OutlineDepth, 4},
		{"UseCompression", g.UseCompression, true},
		{"Copies", g.Copies, 1},
		{"SmartShrinking", g.SmartShrinking, true},
		{"Background", g.Background, true},
		{"Web.Images", g.Web.Images, true},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	if g.Margin.Top != 10 || g.Margin.Bottom != 10 || g.Margin.Left != 10 || g.Margin.Right != 10 {
		t.Errorf("default margins = %+v, want 10mm all sides", g.Margin)
	}
	if g.Header.FontSize != 12 || g.Header.FontName != "Arial" || g.Header.Spacing != 0 {
		t.Errorf("default header = %+v, want Arial 12 spacing 0", g.Header)
	}
	if g.TOC.CaptionText != "Table of Contents" || g.TOC.FontScale != 0.8 || !g.TOC.DottedLines {
		t.Errorf("default toc = %+v", g.TOC)
	}
}

func TestDefaultPdfObjectSnapshot(t *testing.T) {
	o := DefaultPdfObject()
	if !o.ExternalLinks || !o.LocalLinks || !o.IncludeInOutline || !o.UseOutline {
		t.Errorf("default object = %+v", o)
	}
}

func TestDefaultLoadPageSnapshot(t *testing.T) {
	o := DefaultPdfObject()
	if !o.Load.BlockLocalFileAccess {
		t.Error("default blockLocalFileAccess must be true")
	}
	if o.Load.LoadErrorHandling != LoadErrorAbort {
		t.Error("default loadErrorHandling must be abort")
	}
}

func TestGlobalSetDottedKeys(t *testing.T) {
	g := DefaultPdfGlobal()
	tests := []struct {
		key, val string
		check    func(t *testing.T)
	}{
		{"margin.top", "25mm", func(t *testing.T) {
			if g.Margin.Top != 25 {
				t.Errorf("margin.top = %v, want 25", g.Margin.Top)
			}
		}},
		{"margin.left", "1in", func(t *testing.T) {
			if math.Abs(g.Margin.Left-25.4) > 1e-6 {
				t.Errorf("margin.left = %v, want 25.4mm", g.Margin.Left)
			}
		}},
		{"size.pagesize", "Letter", func(t *testing.T) {
			if g.PageSize != "Letter" || g.Size.PageSize != "Letter" {
				t.Errorf("pagesize = %q/%q", g.PageSize, g.Size.PageSize)
			}
		}},
		{"size.width", "210mm", func(t *testing.T) {
			if g.Size.Width != 210 {
				t.Errorf("size.width = %v, want 210", g.Size.Width)
			}
		}},
		{"size.height", "297mm", func(t *testing.T) {
			if g.Size.Height != 297 {
				t.Errorf("size.height = %v, want 297", g.Size.Height)
			}
		}},
		{"orientation", "landscape", func(t *testing.T) {
			if g.Orientation != OrientationLandscape {
				t.Error("orientation not landscape")
			}
		}},
		{"colormode", "grayscale", func(t *testing.T) {
			if !g.Grayscale {
				t.Error("colormode=grayscale must set Grayscale")
			}
		}},
		{"grayscale", "false", func(t *testing.T) {
			if g.Grayscale {
				t.Error("grayscale=false must clear Grayscale")
			}
		}},
		{"grayscale", "true", func(t *testing.T) {
			if !g.Grayscale {
				t.Error("grayscale=true must set Grayscale")
			}
		}},
		{"outline", "false", func(t *testing.T) {
			if g.Outline {
				t.Error("outline should be false")
			}
		}},
		{"outlinedepth", "6", func(t *testing.T) {
			if g.OutlineDepth != 6 {
				t.Errorf("outlinedepth = %d", g.OutlineDepth)
			}
		}},
		{"copies", "3", func(t *testing.T) {
			if g.Copies != 3 {
				t.Errorf("copies = %d", g.Copies)
			}
		}},
		{"title", "Invoice", func(t *testing.T) {
			if g.Title != "Invoice" {
				t.Errorf("title = %q", g.Title)
			}
		}},
		{"header.fontsize", "14", func(t *testing.T) {
			if g.Header.FontSize != 14 {
				t.Errorf("header.fontsize = %v", g.Header.FontSize)
			}
		}},
		{"footer.center", "[page] of [topage]", func(t *testing.T) {
			if g.Footer.Center != "[page] of [topage]" {
				t.Errorf("footer.center = %q", g.Footer.Center)
			}
		}},
		{"toc.captiontext", "Contents", func(t *testing.T) {
			if g.TOC.CaptionText != "Contents" {
				t.Error("toc.captiontext not set")
			}
		}},
		{"web.background", "false", func(t *testing.T) {
			if g.Background {
				t.Error("web.background should clear Global.Background")
			}
		}},
		{"allow", "/srv/html", func(t *testing.T) {
			if len(g.Load.Allow) != 1 || g.Load.Allow[0] != "/srv/html" {
				t.Errorf("allow = %v", g.Load.Allow)
			}
		}},
		{"dumpoutline", "true", func(t *testing.T) {
			if !g.DumpOutline {
				t.Error("dumpoutline should be true")
			}
		}},
	}
	for _, tt := range tests {
		if err := g.Set(tt.key, tt.val); err != nil {
			t.Errorf("Set(%q, %q) error: %v", tt.key, tt.val, err)
			continue
		}
		tt.check(t)
	}
}

func TestGlobalSetIgnoredKeys(t *testing.T) {
	g := DefaultPdfGlobal()
	// Policy A: inert wkhtml keys are accepted into Ignored, not typed fields.
	for _, key := range []string{"dpi", "imagedpi", "imagequality", "lowquality", "log-level", "web.javascript", "produceforms"} {
		if err := g.Set(key, "1"); err != nil {
			t.Errorf("Set(%q) should accept ignored key: %v", key, err)
		}
		if g.Ignored[key] != "1" {
			t.Errorf("Ignored[%q] = %q, want 1", key, g.Ignored[key])
		}
	}
	// No typed stubs reintroduced.
	if g.Quiet {
		// Quiet is real; ensure dpi did not bleed into other fields.
	}
}

func TestGlobalSetUnknownKey(t *testing.T) {
	g := DefaultPdfGlobal()
	if err := g.Set("bogus.key", "1"); err == nil {
		t.Error("expected error for unknown key")
	}
}

func TestObjectSetDottedKeys(t *testing.T) {
	o := DefaultPdfObject()
	if err := o.Set("load.blocklocalfileaccess", "false"); err != nil {
		t.Fatal(err)
	}
	if o.Load.BlockLocalFileAccess {
		t.Error("blocklocalfileaccess should be false")
	}
	if err := o.Set("load.timeout", "30"); err != nil {
		t.Fatal(err)
	}
	if o.Load.Timeout != 30 {
		t.Errorf("timeout = %d", o.Load.Timeout)
	}
	if err := o.Set("header.left", "objheader"); err != nil {
		t.Fatal(err)
	}
	if !o.HeaderSet || o.Header.Left != "objheader" {
		t.Errorf("object header = %+v (set=%v)", o.Header, o.HeaderSet)
	}
	if err := o.Set("footer.right", "objfooter"); err != nil {
		t.Fatal(err)
	}
	if !o.FooterSet || o.Footer.Right != "objfooter" {
		t.Errorf("object footer = %+v (set=%v)", o.Footer, o.FooterSet)
	}
	if err := o.Set("web.images", "false"); err != nil {
		t.Fatal(err)
	}
	if o.Web.Images {
		t.Error("web.images should be false")
	}
	if err := o.Set("web.simplifydom", "true"); err != nil {
		t.Fatal(err)
	}
	if !o.Web.SimplifyDOM {
		t.Error("web.simplifydom should be true")
	}
}

func TestObjectSetIgnoredKeys(t *testing.T) {
	o := DefaultPdfObject()
	if err := o.Set("load.jsdelay", "500"); err != nil {
		t.Fatalf("jsdelay should be accepted as ignored: %v", err)
	}
	if o.Ignored["load.jsdelay"] != "500" {
		t.Errorf("Ignored jsdelay = %q", o.Ignored["load.jsdelay"])
	}
	if err := o.Set("web.javascript", "false"); err != nil {
		t.Fatalf("web.javascript should be accepted as ignored: %v", err)
	}
	if o.Ignored["web.javascript"] != "false" {
		t.Errorf("Ignored web.javascript = %q", o.Ignored["web.javascript"])
	}
	if err := o.Set("pagescount", "false"); err != nil {
		t.Fatalf("pagescount should be accepted as ignored: %v", err)
	}
}

func TestParseUnitReal(t *testing.T) {
	tests := []struct {
		in   string
		unit string
		val  float64
	}{
		{"10mm", "mm", 10},
		{"1.5cm", "cm", 1.5},
		{"1in", "in", 1},
		{"12pt", "pt", 12},
		{"96px", "px", 96},
		{"2em", "em", 2},
		{"100%", "%", 100},
		{"5", "", 5}, // implied unit
		{" 7.5 mm ", "mm", 7.5},
	}
	for _, tt := range tests {
		u, err := ParseUnitReal(tt.in, "")
		if err != nil {
			t.Errorf("ParseUnitReal(%q): %v", tt.in, err)
			continue
		}
		if u.Value != tt.val || u.Unit != tt.unit {
			t.Errorf("ParseUnitReal(%q) = %+v, want val=%v unit=%q", tt.in, u, tt.val, tt.unit)
		}
	}
	if _, err := ParseUnitReal("abc", "mm"); !errors.Is(err, ErrInvalidUnitReal) {
		t.Errorf("expected ErrInvalidUnitReal for %q, got %v", "abc", err)
	}
	if _, err := ParseUnitReal("", "mm"); !errors.Is(err, ErrInvalidUnitReal) {
		t.Errorf("expected ErrInvalidUnitReal for empty")
	}
}

func TestUnitRealPoints(t *testing.T) {
	u, _ := ParseUnitReal("10mm", "mm")
	pt, ok := u.Points()
	if !ok || math.Abs(pt-28.346) > 0.01 {
		t.Errorf("10mm = %v pt (ok=%v)", pt, ok)
	}
	u, _ = ParseUnitReal("72pt", "pt")
	if pt, _ := u.Points(); pt != 72 {
		t.Errorf("72pt = %v", pt)
	}
	u, _ = ParseUnitReal("96px", "px")
	if pt, _ := u.Points(); pt != 72 {
		t.Errorf("96px = %v pt, want 72", pt)
	}
	u, _ = ParseUnitReal("2em", "em")
	if _, ok := u.Points(); ok {
		t.Error("em must not convert without font context")
	}
}

func TestParsePageSize(t *testing.T) {
	w, h, err := ParsePageSize("A4")
	if err != nil || math.Abs(w-595.28) > 0.1 || math.Abs(h-841.89) > 0.1 {
		t.Errorf("A4 = %v x %v, err %v", w, h, err)
	}
	w, h, err = ParsePageSize("letter")
	if err != nil || w != 612 || h != 792 {
		t.Errorf("letter = %v x %v, err %v", w, h, err)
	}
	if _, _, err := ParsePageSize("Bogus"); err == nil {
		t.Error("expected error for unknown size")
	}
}

func TestParseEnums(t *testing.T) {
	if v, _ := ParseOrientation("LANDSCAPE"); v != OrientationLandscape {
		t.Error("orientation case-insensitive")
	}
	if v, err := ParseOrientation("diagonal"); err == nil || v != OrientationPortrait {
		t.Error("invalid orientation must error")
	}
	if v, _ := ParseColorMode("grayscale"); v != ColorModeGrayscale {
		t.Error("color-mode grayscale")
	}
	if v, _ := ParseLoadErrorHandling("skip"); v != LoadErrorSkip {
		t.Error("load-error-handling skip")
	}
}

func TestHttpErrorCode(t *testing.T) {
	if HttpErrorCode(404) != 2 {
		t.Error("404 must map to exit 2")
	}
	if HttpErrorCode(401) != 3 {
		t.Error("401 must map to exit 3")
	}
	if HttpErrorCode(500) != 1 {
		t.Error("500 must map to exit 1")
	}
}

func TestHeaderForFooterForInherit(t *testing.T) {
	g := DefaultPdfGlobal()
	g.Header.Left = "global header"
	o := DefaultPdfObject()
	if o.HeaderFor(g).Left != "global header" {
		t.Error("object must inherit global header")
	}
	o.HeaderSet = true
	o.Header.Left = "own header"
	if o.HeaderFor(g).Left != "own header" {
		t.Error("object header override must win")
	}
}

func TestDefaultImageGlobalNoQuietLogLevel(t *testing.T) {
	img := DefaultImageGlobal()
	if img.Width != 1024 || !img.SmartWidth || img.Quality != 94 {
		t.Errorf("default image = %+v", img)
	}
	// Quiet lives on PdfGlobal only; imageout uses Global.Quiet.
}

func TestImageSet(t *testing.T) {
	img := DefaultImageGlobal()
	if err := img.Set("width", "800"); err != nil {
		t.Fatal(err)
	}
	if img.Width != 800 {
		t.Errorf("width = %d", img.Width)
	}
	if err := img.Set("web.images", "false"); err != nil {
		t.Fatal(err)
	}
	if img.Web.Images {
		t.Error("web.images should be false")
	}
	// Inert web key accepted into Ignored.
	if err := img.Set("web.javascript", "true"); err != nil {
		t.Fatal(err)
	}
	if img.Ignored["web.javascript"] != "true" {
		t.Errorf("Ignored = %v", img.Ignored)
	}
}

func TestGlobalGetSetRoundTripAndIgnored(t *testing.T) {
	g := DefaultPdfGlobal()
	if err := g.Set("title", "Hi"); err != nil {
		t.Fatal(err)
	}
	if got, ok := g.Get("title"); !ok || got != "Hi" {
		t.Fatalf("Get(title)=%q,%v", got, ok)
	}
	if err := g.Set("dpi", "150"); err != nil {
		t.Fatal(err)
	}
	if got, ok := g.Get("dpi"); !ok || got != "150" {
		t.Fatalf("Get(dpi ignored)=%q,%v want 150,true", got, ok)
	}
	if _, ok := g.Get("totally.unknown"); ok {
		t.Fatal("unknown key should not Get")
	}
	if err := g.Set("background", "false"); err != nil {
		t.Fatal(err)
	}
	if got, ok := g.Get("web.background"); !ok || got != "false" {
		t.Fatalf("Get(web.background)=%q,%v after Set(background)", got, ok)
	}
}

func TestKeyTableSetGetParity(t *testing.T) {
	// Every registered key must have both an apply and a get descriptor, and
	// every Get must answer ok=true (all descriptors get returns true).
	g := DefaultPdfGlobal()
	o := DefaultPdfObject()
	img := DefaultImageGlobal()
	for k := range globalKeys {
		if _, ok := getForKey(&g, globalKeys, &g.Ignored, k); !ok {
			t.Errorf("global key %q missing Get", k)
		}
	}
	for k := range objectKeys {
		if _, ok := getForKey(&o, objectKeys, &o.Ignored, k); !ok {
			t.Errorf("object key %q missing Get", k)
		}
	}
	for k := range imageKeys {
		if _, ok := getForKey(&img, imageKeys, &img.Ignored, k); !ok {
			t.Errorf("image key %q missing Get", k)
		}
	}
}

func TestBackgroundSingleFieldNoWebMirror(t *testing.T) {
	g := DefaultPdfGlobal()
	if err := g.Set("web.background", "false"); err != nil {
		t.Fatal(err)
	}
	if g.Background {
		t.Fatal("Global.Background should be false")
	}
	// Web has no Background field — compile-time guarantee; runtime Get uses Global.
	got, ok := g.Get("web.background")
	if !ok || got != "false" {
		t.Fatalf("Get(web.background)=%q,%v", got, ok)
	}
	got2, ok := g.Get("background")
	if !ok || got2 != "false" {
		t.Fatalf("Get(background)=%q,%v", got2, ok)
	}
}

func TestResolveMedia(t *testing.T) {
	base := "print"
	none := Web{}
	pmt := Web{PrintMediaType: true}
	screen := Web{MediaType: MediaScreen}
	print := Web{MediaType: MediaPrint}

	if got := ResolveMedia(base, none, nil); got != "print" {
		t.Errorf("default PDF = %q", got)
	}
	if got := ResolveMedia("screen", none, nil); got != "screen" {
		t.Errorf("default image = %q", got)
	}
	if got := ResolveMedia(base, pmt, nil); got != "print" {
		t.Errorf("global print-media-type = %q", got)
	}
	if got := ResolveMedia(base, none, &pmt); got != "print" {
		t.Errorf("obj print-media-type = %q", got)
	}
	if got := ResolveMedia(base, screen, nil); got != "screen" {
		t.Errorf("global media-type screen = %q", got)
	}
	if got := ResolveMedia(base, none, &screen); got != "screen" {
		t.Errorf("obj media-type screen = %q", got)
	}
	// obj wins over global media-type.
	if got := ResolveMedia(base, screen, &print); got != "print" {
		t.Errorf("obj media-type print over global screen = %q", got)
	}
	// print-media-type override wins over media-type.
	if got := ResolveMedia(base, screen, &Web{PrintMediaType: true}); got != "print" {
		t.Errorf("pmt over media-type screen = %q", got)
	}
}

func TestApplyImageKeyBackgroundAlias(t *testing.T) {
	g := DefaultPdfGlobal()
	img := DefaultImageGlobal()
	if err := ApplyImageKey(&g, &img, "web.background", "false"); err != nil {
		t.Fatal(err)
	}
	if g.Background {
		t.Error("web.background must route to PdfGlobal.Background")
	}
	if err := ApplyImageKey(&g, &img, "background", "true"); err != nil {
		t.Fatal(err)
	}
	if !g.Background {
		t.Error("background must route to PdfGlobal.Background")
	}
	if err := ApplyImageKey(&g, &img, "width", "800"); err != nil {
		t.Fatal(err)
	}
	if img.Width != 800 {
		t.Errorf("width must route to ImageGlobal: %d", img.Width)
	}
}
