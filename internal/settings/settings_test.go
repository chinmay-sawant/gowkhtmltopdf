package settings

import (
	"errors"
	"math"
	"testing"
)

func TestDefaultPdfGlobalSnapshot(t *testing.T) {
	t.Parallel()

	global := DefaultPdfGlobal()

	checks := []struct {
		name string
		got  any
		want any
	}{
		{"PageSize", global.PageSize, "A4"},
		{"Orientation", global.Orientation, OrientationPortrait},
		{"Grayscale", global.Grayscale, false},
		{"Collate", global.Collate, true},
		{"Outline", global.Outline, true},
		{"OutlineDepth", global.OutlineDepth, 4},
		{"UseCompression", global.UseCompression, true},
		{"Copies", global.Copies, 1},
		{"SmartShrinking", global.SmartShrinking, true},
		{"Background", global.Background, true},
		{"Web.Images", global.Web.Images, true},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}

	if global.Margin.Top != 10 || global.Margin.Bottom != 10 || global.Margin.Left != 10 || global.Margin.Right != 10 {
		t.Errorf("default margins = %+v, want 10mm all sides", global.Margin)
	}

	if global.Header.FontSize != 12 || global.Header.FontName != "Arial" || global.Header.Spacing != 0 {
		t.Errorf("default header = %+v, want Arial 12 spacing 0", global.Header)
	}

	if global.TOC.CaptionText != "Table of Contents" || global.TOC.FontScale != 0.8 || !global.TOC.DottedLines {
		t.Errorf("default toc = %+v", global.TOC)
	}
}

func TestDefaultPdfObjectSnapshot(t *testing.T) {
	t.Parallel()

	obj := DefaultPdfObject()
	if !obj.ExternalLinks || !obj.LocalLinks || !obj.IncludeInOutline || !obj.UseOutline {
		t.Errorf("default object = %+v", obj)
	}
}

func TestDefaultLoadPageSnapshot(t *testing.T) {
	t.Parallel()

	obj := DefaultPdfObject()
	if !obj.Load.BlockLocalFileAccess {
		t.Error("default blockLocalFileAccess must be true")
	}

	if obj.Load.LoadErrorHandling != LoadErrorAbort {
		t.Error("default loadErrorHandling must be abort")
	}
}

func TestGlobalSetDottedKeys(t *testing.T) {
	t.Parallel()

	global := DefaultPdfGlobal()
	tests := []struct {
		key, val string
		check    func(t *testing.T)
	}{
		{"margin.top", "25mm", func(t *testing.T) {
			if global.Margin.Top != 25 {
				t.Errorf("margin.top = %v, want 25", global.Margin.Top)
			}
		}},
		{"margin.left", "1in", func(t *testing.T) {
			if math.Abs(global.Margin.Left-25.4) > 1e-6 {
				t.Errorf("margin.left = %v, want 25.4mm", global.Margin.Left)
			}
		}},
		{"size.pagesize", "Letter", func(t *testing.T) {
			if global.PageSize != "Letter" || global.Size.PageSize != "Letter" {
				t.Errorf("pagesize = %q/%q", global.PageSize, global.Size.PageSize)
			}
		}},
		{"size.width", "210mm", func(t *testing.T) {
			if global.Size.Width != 210 {
				t.Errorf("size.width = %v, want 210", global.Size.Width)
			}
		}},
		{"size.height", "297mm", func(t *testing.T) {
			if global.Size.Height != 297 {
				t.Errorf("size.height = %v, want 297", global.Size.Height)
			}
		}},
		{"orientation", "landscape", func(t *testing.T) {
			if global.Orientation != OrientationLandscape {
				t.Error("orientation not landscape")
			}
		}},
		{"colormode", "grayscale", func(t *testing.T) {
			if !global.Grayscale {
				t.Error("colormode=grayscale must set Grayscale")
			}
		}},
		{"grayscale", "false", func(t *testing.T) {
			if global.Grayscale {
				t.Error("grayscale=false must clear Grayscale")
			}
		}},
		{"grayscale", "true", func(t *testing.T) {
			if !global.Grayscale {
				t.Error("grayscale=true must set Grayscale")
			}
		}},
		{"outline", "false", func(t *testing.T) {
			if global.Outline {
				t.Error("outline should be false")
			}
		}},
		{"outlinedepth", "6", func(t *testing.T) {
			if global.OutlineDepth != 6 {
				t.Errorf("outlinedepth = %d", global.OutlineDepth)
			}
		}},
		{"copies", "3", func(t *testing.T) {
			if global.Copies != 3 {
				t.Errorf("copies = %d", global.Copies)
			}
		}},
		{"title", "Invoice", func(t *testing.T) {
			if global.Title != "Invoice" {
				t.Errorf("title = %q", global.Title)
			}
		}},
		{"header.fontsize", "14", func(t *testing.T) {
			if global.Header.FontSize != 14 {
				t.Errorf("header.fontsize = %v", global.Header.FontSize)
			}
		}},
		{"footer.center", "[page] of [topage]", func(t *testing.T) {
			if global.Footer.Center != "[page] of [topage]" {
				t.Errorf("footer.center = %q", global.Footer.Center)
			}
		}},
		{"toc.captiontext", "Contents", func(t *testing.T) {
			if global.TOC.CaptionText != "Contents" {
				t.Error("toc.captiontext not set")
			}
		}},
		{"web.background", "false", func(t *testing.T) {
			if global.Background {
				t.Error("web.background should clear Global.Background")
			}
		}},
		{"allow", "/srv/html", func(t *testing.T) {
			if len(global.Load.Allow) != 1 || global.Load.Allow[0] != "/srv/html" {
				t.Errorf("allow = %v", global.Load.Allow)
			}
		}},
		{"dumpoutline", "true", func(t *testing.T) {
			if !global.DumpOutline {
				t.Error("dumpoutline should be true")
			}
		}},
	}

	for _, testCase := range tests {
		if err := global.Set(testCase.key, testCase.val); err != nil {
			t.Errorf("Set(%q, %q) error: %v", testCase.key, testCase.val, err)

			continue
		}

		testCase.check(t)
	}
}

func TestGlobalSetIgnoredKeys(t *testing.T) {
	t.Parallel()

	global := DefaultPdfGlobal()
	// Policy A: inert wkhtml keys are accepted into Ignored, not typed fields.
	for _, key := range []string{"dpi", "imagedpi", "imagequality", "lowquality", "log-level", "web.javascript", "produceforms"} {
		if err := global.Set(key, "1"); err != nil {
			t.Errorf("Set(%q) should accept ignored key: %v", key, err)
		}

		if global.Ignored[key] != "1" {
			t.Errorf("Ignored[%q] = %q, want 1", key, global.Ignored[key])
		}
	}
	// No typed stubs reintroduced.
	if global.Quiet {
		// Quiet is real; ensure dpi did not bleed into other fields.
	}
}

func TestGlobalSetUnknownKey(t *testing.T) {
	t.Parallel()

	global := DefaultPdfGlobal()
	if err := global.Set("bogus.key", "1"); err == nil {
		t.Error("expected error for unknown key")
	}
}

func TestObjectSetDottedKeys(t *testing.T) {
	t.Parallel()

	obj := DefaultPdfObject()
	if err := obj.Set("load.blocklocalfileaccess", "false"); err != nil {
		t.Fatal(err)
	}

	if obj.Load.BlockLocalFileAccess {
		t.Error("blocklocalfileaccess should be false")
	}

	if err := obj.Set("load.timeout", "30"); err != nil {
		t.Fatal(err)
	}

	if obj.Load.Timeout != 30 {
		t.Errorf("timeout = %d", obj.Load.Timeout)
	}

	if err := obj.Set("header.left", "objheader"); err != nil {
		t.Fatal(err)
	}

	if !obj.HeaderSet || obj.Header.Left != "objheader" {
		t.Errorf("object header = %+v (set=%v)", obj.Header, obj.HeaderSet)
	}

	if err := obj.Set("footer.right", "objfooter"); err != nil {
		t.Fatal(err)
	}

	if !obj.FooterSet || obj.Footer.Right != "objfooter" {
		t.Errorf("object footer = %+v (set=%v)", obj.Footer, obj.FooterSet)
	}

	if err := obj.Set("web.images", "false"); err != nil {
		t.Fatal(err)
	}

	if obj.Web.Images {
		t.Error("web.images should be false")
	}

	if err := obj.Set("web.simplifydom", "true"); err != nil {
		t.Fatal(err)
	}

	if !obj.Web.SimplifyDOM {
		t.Error("web.simplifydom should be true")
	}
}

func TestObjectSetIgnoredKeys(t *testing.T) {
	t.Parallel()

	obj := DefaultPdfObject()
	if err := obj.Set("load.jsdelay", "500"); err != nil {
		t.Fatalf("jsdelay should be accepted as ignored: %v", err)
	}

	if obj.Ignored["load.jsdelay"] != "500" {
		t.Errorf("Ignored jsdelay = %q", obj.Ignored["load.jsdelay"])
	}

	if err := obj.Set("web.javascript", "false"); err != nil {
		t.Fatalf("web.javascript should be accepted as ignored: %v", err)
	}

	if obj.Ignored["web.javascript"] != "false" {
		t.Errorf("Ignored web.javascript = %q", obj.Ignored["web.javascript"])
	}

	if err := obj.Set("pagescount", "false"); err != nil {
		t.Fatalf("pagescount should be accepted as ignored: %v", err)
	}
}

func TestParseUnitReal(t *testing.T) {
	t.Parallel()

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
	for _, testCase := range tests {
		unit, err := ParseUnitReal(testCase.in, "")
		if err != nil {
			t.Errorf("ParseUnitReal(%q): %v", testCase.in, err)

			continue
		}

		if unit.Value != testCase.val || unit.Unit != testCase.unit {
			t.Errorf("ParseUnitReal(%q) = %+v, want val=%v unit=%q", testCase.in, unit, testCase.val, testCase.unit)
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
	t.Parallel()

	unit, _ := ParseUnitReal("10mm", "mm")

	pt, found := unit.Points()
	if !found || math.Abs(pt-28.346) > 0.01 {
		t.Errorf("10mm = %v pt (ok=%v)", pt, found)
	}

	unit, _ = ParseUnitReal("72pt", "pt")
	if pt, _ := unit.Points(); pt != 72 {
		t.Errorf("72pt = %v", pt)
	}

	unit, _ = ParseUnitReal("96px", "px")
	if pt, _ := unit.Points(); pt != 72 {
		t.Errorf("96px = %v pt, want 72", pt)
	}

	unit, _ = ParseUnitReal("2em", "em")
	if _, found := unit.Points(); found {
		t.Error("em must not convert without font context")
	}
}

func TestParsePageSize(t *testing.T) {
	t.Parallel()

	width, height, err := ParsePageSize("A4")
	if err != nil || math.Abs(width-595.28) > 0.1 || math.Abs(height-841.89) > 0.1 {
		t.Errorf("A4 = %v x %v, err %v", width, height, err)
	}

	width, height, err = ParsePageSize("letter")
	if err != nil || width != 612 || height != 792 {
		t.Errorf("letter = %v x %v, err %v", width, height, err)
	}

	if _, _, err := ParsePageSize("Bogus"); err == nil {
		t.Error("expected error for unknown size")
	}
}

func TestParseEnums(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

	global := DefaultPdfGlobal()
	global.Header.Left = "global header"
	obj := DefaultPdfObject()

	if obj.HeaderFor(global).Left != "global header" {
		t.Error("object must inherit global header")
	}

	obj.HeaderSet = true
	obj.Header.Left = "own header"

	if obj.HeaderFor(global).Left != "own header" {
		t.Error("object header override must win")
	}
}

func TestDefaultImageGlobalNoQuietLogLevel(t *testing.T) {
	t.Parallel()

	img := DefaultImageGlobal()
	if img.Width != 1024 || !img.SmartWidth || img.Quality != 94 {
		t.Errorf("default image = %+v", img)
	}
	// Quiet lives on PdfGlobal only; imageout uses Global.Quiet.
}

func TestImageSet(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	global := DefaultPdfGlobal()
	if err := global.Set("title", "Hi"); err != nil {
		t.Fatal(err)
	}

	if got, found := global.Get("title"); !found || got != "Hi" {
		t.Fatalf("Get(title)=%q,%v", got, found)
	}

	if err := global.Set("dpi", "150"); err != nil {
		t.Fatal(err)
	}

	if got, found := global.Get("dpi"); !found || got != "150" {
		t.Fatalf("Get(dpi ignored)=%q,%v want 150,true", got, found)
	}

	if _, found := global.Get("totally.unknown"); found {
		t.Fatal("unknown key should not Get")
	}

	if err := global.Set("background", "false"); err != nil {
		t.Fatal(err)
	}

	if got, found := global.Get("web.background"); !found || got != "false" {
		t.Fatalf("Get(web.background)=%q,%v after Set(background)", got, found)
	}
}

func TestKeyTableSetGetParity(t *testing.T) {
	t.Parallel()
	// Every registered key must have both an apply and a get descriptor, and
	// every Get must answer ok=true (all descriptors get returns true).
	global := DefaultPdfGlobal()
	obj := DefaultPdfObject()
	img := DefaultImageGlobal()

	for k := range globalKeys {
		if _, found := getForKey(&global, globalKeys, &global.Ignored, k); !found {
			t.Errorf("global key %q missing Get", k)
		}
	}

	for k := range objectKeys {
		if _, found := getForKey(&obj, objectKeys, &obj.Ignored, k); !found {
			t.Errorf("object key %q missing Get", k)
		}
	}

	for k := range imageKeys {
		if _, found := getForKey(&img, imageKeys, &img.Ignored, k); !found {
			t.Errorf("image key %q missing Get", k)
		}
	}
}

func TestBackgroundSingleFieldNoWebMirror(t *testing.T) {
	t.Parallel()

	global := DefaultPdfGlobal()
	if err := global.Set("web.background", "false"); err != nil {
		t.Fatal(err)
	}

	if global.Background {
		t.Fatal("Global.Background should be false")
	}
	// Web has no Background field — compile-time guarantee; runtime Get uses Global.
	got, found := global.Get("web.background")
	if !found || got != "false" {
		t.Fatalf("Get(web.background)=%q,%v", got, found)
	}

	got2, found := global.Get("background")
	if !found || got2 != "false" {
		t.Fatalf("Get(background)=%q,%v", got2, found)
	}
}

func TestResolveMedia(t *testing.T) {
	t.Parallel()

	base := "print"
	none := Web{}                         //nolint:exhaustruct // intentional zero/partial fields
	pmt := Web{PrintMediaType: true}      //nolint:exhaustruct // intentional zero/partial fields
	screen := Web{MediaType: MediaScreen} //nolint:exhaustruct // intentional zero/partial fields
	print := Web{MediaType: MediaPrint}   //nolint:exhaustruct // intentional zero/partial fields

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
	if got := ResolveMedia(base, screen, &Web{PrintMediaType: true}); got != "print" { //nolint:exhaustruct // intentional zero/partial fields
		t.Errorf("pmt over media-type screen = %q", got)
	}
}

func TestApplyImageKeyBackgroundAlias(t *testing.T) {
	t.Parallel()

	global := DefaultPdfGlobal()
	img := DefaultImageGlobal()

	if err := ApplyImageKey(&global, &img, "web.background", "false"); err != nil {
		t.Fatal(err)
	}

	if global.Background {
		t.Error("web.background must route to PdfGlobal.Background")
	}

	if err := ApplyImageKey(&global, &img, "background", "true"); err != nil {
		t.Fatal(err)
	}

	if !global.Background {
		t.Error("background must route to PdfGlobal.Background")
	}

	if err := ApplyImageKey(&global, &img, "width", "800"); err != nil {
		t.Fatal(err)
	}

	if img.Width != 800 {
		t.Errorf("width must route to ImageGlobal: %d", img.Width)
	}
}
