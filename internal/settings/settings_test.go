package settings //nolint:testpackage // exercises unexported key tables via getForKey

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
		{"PdfVersion", global.PdfVersion, ""},
		{"PdfProfile", global.PdfProfile, ""},
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

	assertDefaultMargins(t, global.Margin)
	assertDefaultHeader(t, global.Header)
	assertDefaultTOC(t, global.TOC)
}

func assertDefaultMargins(t *testing.T, m Margin) {
	t.Helper()

	if m.Top != 10 || m.Bottom != 10 || m.Left != 10 || m.Right != 10 {
		t.Errorf("default margins = %+v, want 10mm all sides", m)
	}
}

func assertDefaultHeader(t *testing.T, h HeaderFooter) {
	t.Helper()

	if h.FontSize != 12 || h.FontName != "Arial" || h.Spacing != 0 {
		t.Errorf("default header = %+v, want Arial 12 spacing 0", h)
	}
}

func assertDefaultTOC(t *testing.T, toc TableOfContent) {
	t.Helper()

	if toc.CaptionText != "Table of Contents" || toc.FontScale != 0.8 || !toc.DottedLines {
		t.Errorf("default toc = %+v", toc)
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
	for _, tc := range globalDottedGeometryChecks(&global) {
		runDottedCheck(t, &global, tc)
	}

	for _, tc := range globalDottedTextChecks(&global) {
		runDottedCheck(t, &global, tc)
	}
}

type dottedCheck struct {
	key, val string
	desc     string
	check    func() bool
}

func runDottedCheck(t *testing.T, global *PdfGlobal, dotted dottedCheck) {
	t.Helper()

	if err := global.Set(dotted.key, dotted.val); err != nil {
		t.Errorf("Set(%q, %q) error: %v", dotted.key, dotted.val, err)

		return
	}

	if !dotted.check() {
		t.Errorf("%s: Set(%q, %q) check failed", dotted.desc, dotted.key, dotted.val)
	}
}

func globalDottedGeometryChecks(global *PdfGlobal) []dottedCheck {
	return []dottedCheck{
		{
			key: "margin.top", val: "25mm", desc: "margin.top must be 25mm",
			check: func() bool { return global.Margin.Top == 25 },
		},
		{
			key: "margin.left", val: "1in", desc: "margin.left must be 25.4mm",
			check: func() bool { return math.Abs(global.Margin.Left-25.4) < 1e-6 },
		},
		{
			key: "size.pagesize", val: "Letter", desc: "pagesize must be Letter on both homes",
			check: func() bool { return global.PageSize == "Letter" },
		},
		{
			key: "size.width", val: "210mm", desc: "size.width must be 210mm",
			check: func() bool { return global.Size.Width == 210 },
		},
		{
			key: "size.height", val: "297mm", desc: "size.height must be 297mm",
			check: func() bool { return global.Size.Height == 297 },
		},
		{
			key: "orientation", val: "landscape", desc: "orientation must be landscape",
			check: func() bool { return global.Orientation == OrientationLandscape },
		},
		{
			key: "colormode", val: "grayscale", desc: "colormode=grayscale must set Grayscale",
			check: func() bool { return global.Grayscale },
		},
		{
			key: "grayscale", val: sFalse, desc: "grayscale=false must clear Grayscale",
			check: func() bool { return !global.Grayscale },
		},
		{
			key: "grayscale", val: "true", desc: "grayscale=true must set Grayscale",
			check: func() bool { return global.Grayscale },
		},
		{
			key: "outline", val: sFalse, desc: "outline must be false",
			check: func() bool { return !global.Outline },
		},
	}
}

func globalDottedTextChecks(global *PdfGlobal) []dottedCheck {
	return []dottedCheck{
		{
			key: "outlinedepth", val: "6", desc: "outlinedepth must be 6",
			check: func() bool { return global.OutlineDepth == 6 },
		},
		{
			key: "copies", val: "3", desc: "copies must be 3",
			check: func() bool { return global.Copies == 3 },
		},
		{
			key: "title", val: "Invoice", desc: "title must be Invoice",
			check: func() bool { return global.Title == "Invoice" },
		},
		{
			key: "header.fontsize", val: "14", desc: "header.fontsize must be 14",
			check: func() bool { return global.Header.FontSize == 14 },
		},
		{
			key: "footer.center", val: "[page] of [topage]", desc: "footer.center must be set",
			check: func() bool { return global.Footer.Center == "[page] of [topage]" },
		},
		{
			key: "toc.captiontext", val: "Contents", desc: "toc.captiontext must be set",
			check: func() bool { return global.TOC.CaptionText == "Contents" },
		},
		{
			key: "web.background", val: sFalse, desc: "web.background must clear Global.Background",
			check: func() bool { return !global.Background },
		},
		{
			key: "allow", val: "/srv/html", desc: "allow must be in Load.Allow",
			check: func() bool { return len(global.Load.Allow) == 1 && global.Load.Allow[0] == "/srv/html" },
		},
		{
			key: "dumpoutline", val: "true", desc: "dumpoutline must be true",
			check: func() bool { return global.DumpOutline },
		},
	}
}

func TestGlobalSetIgnoredKeys(t *testing.T) {
	t.Parallel()

	global := DefaultPdfGlobal()
	// Policy A: inert wkhtml keys are accepted into Ignored, not typed fields.
	for _, key := range []string{
		"dpi", "imagedpi", "imagequality", "lowquality", "log-level", "web.javascript", "produceforms",
	} {
		if err := global.Set(key, "1"); err != nil {
			t.Errorf("Set(%q) should accept ignored key: %v", key, err)
		}

		if global.Ignored[key] != "1" {
			t.Errorf("Ignored[%q] = %q, want 1", key, global.Ignored[key])
		}
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
	setKey(t, &obj, "load.blocklocalfileaccess", sFalse)

	if obj.Load.BlockLocalFileAccess {
		t.Error("blocklocalfileaccess should be false")
	}

	setKey(t, &obj, "load.timeout", "30")

	if obj.Load.Timeout != 30 {
		t.Errorf("timeout = %d", obj.Load.Timeout)
	}

	setKey(t, &obj, "header.left", "objheader")

	if !obj.HeaderSet || obj.Header.Left != "objheader" {
		t.Errorf("object header = %+v (set=%v)", obj.Header, obj.HeaderSet)
	}

	setKey(t, &obj, "footer.right", "objfooter")

	if !obj.FooterSet || obj.Footer.Right != "objfooter" {
		t.Errorf("object footer = %+v (set=%v)", obj.Footer, obj.FooterSet)
	}

	setKey(t, &obj, "web.images", sFalse)

	if obj.Web.Images {
		t.Error("web.images should be false")
	}

	setKey(t, &obj, "web.simplifydom", "true")

	if !obj.Web.SimplifyDOM {
		t.Error("web.simplifydom should be true")
	}
}

func setKey(t *testing.T, obj *PdfObject, key, val string) {
	t.Helper()

	if err := obj.Set(key, val); err != nil {
		t.Fatalf("Set(%q, %q): %v", key, val, err)
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

	if err := obj.Set("web.javascript", sFalse); err != nil {
		t.Fatalf("web.javascript should be accepted as ignored: %v", err)
	}

	if obj.Ignored["web.javascript"] != sFalse {
		t.Errorf("Ignored web.javascript = %q", obj.Ignored["web.javascript"])
	}

	if err := obj.Set("pagescount", sFalse); err != nil {
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
	// Quiet lives on PdfGlobal only; imageout uses Global.Quiet.
	if img.Width != 1024 || !img.SmartWidth || img.Quality != 94 {
		t.Errorf("default image = %+v", img)
	}
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

	if err := img.Set("web.images", sFalse); err != nil {
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

	getMust(t, &global, "title", "Hi")

	if err := global.Set("dpi", "150"); err != nil {
		t.Fatal(err)
	}

	getMust(t, &global, "dpi", "150")
	getMissing(t, &global, "totally.unknown")

	if err := global.Set("background", sFalse); err != nil {
		t.Fatal(err)
	}

	getMust(t, &global, "web.background", sFalse)
}

func getMust(t *testing.T, g *PdfGlobal, key, want string) {
	t.Helper()

	got, found := g.Get(key)
	if !found || got != want {
		t.Fatalf("Get(%q) = %q,%v want %q,true", key, got, found, want)
	}
}

func getMissing(t *testing.T, g *PdfGlobal, key string) {
	t.Helper()

	if _, found := g.Get(key); found {
		t.Fatalf("Get(%q) must not be found", key)
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
	if err := global.Set("web.background", sFalse); err != nil {
		t.Fatal(err)
	}

	if global.Background {
		t.Fatal("Global.Background should be false")
	}
	// Web has no Background field — compile-time guarantee; runtime Get uses Global.
	got, found := global.Get("web.background")
	if !found || got != sFalse {
		t.Fatalf("Get(web.background)=%q,%v", got, found)
	}

	got2, found := global.Get("background")
	if !found || got2 != sFalse {
		t.Fatalf("Get(background)=%q,%v", got2, found)
	}
}

func TestResolveMedia(t *testing.T) {
	t.Parallel()

	base := sPrint
	none := Web{}                            //nolint:exhaustruct // intentional zero/partial fields
	pmt := Web{PrintMediaType: true}         //nolint:exhaustruct // intentional zero/partial fields
	screen := Web{MediaType: MediaScreen}    //nolint:exhaustruct // intentional zero/partial fields
	printMedia := Web{MediaType: MediaPrint} //nolint:exhaustruct // intentional zero/partial fields

	if got := ResolveMedia(base, none, nil); got != sPrint {
		t.Errorf("default PDF = %q", got)
	}

	if got := ResolveMedia(sScreen, none, nil); got != sScreen {
		t.Errorf("default image = %q", got)
	}

	if got := ResolveMedia(base, pmt, nil); got != sPrint {
		t.Errorf("global print-media-type = %q", got)
	}

	if got := ResolveMedia(base, none, &pmt); got != sPrint {
		t.Errorf("obj print-media-type = %q", got)
	}

	if got := ResolveMedia(base, screen, nil); got != sScreen {
		t.Errorf("global media-type screen = %q", got)
	}

	if got := ResolveMedia(base, none, &screen); got != sScreen {
		t.Errorf("obj media-type screen = %q", got)
	}
	// obj wins over global media-type.
	if got := ResolveMedia(base, screen, &printMedia); got != sPrint {
		t.Errorf("obj media-type print over global screen = %q", got)
	}
	// print-media-type override wins over media-type.
	if got := ResolveMedia(base, screen, &pmt); got != sPrint {
		t.Errorf("pmt over media-type screen = %q", got)
	}
}

func TestApplyImageKeyBackgroundAlias(t *testing.T) {
	t.Parallel()

	global := DefaultPdfGlobal()
	img := DefaultImageGlobal()

	if err := ApplyImageKey(&global, &img, "web.background", sFalse); err != nil {
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

func TestParsePDFVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input   string
		want    string
		wantErr error
	}{
		{"", "1.4", nil},
		{"1.4", "1.4", nil},
		{"1.7", "1.7", nil},
		{" 1.7 ", "1.7", nil},
		{"2.0", "2.0", nil},
		{" 2.0 ", "2.0", nil},
		{"9.9", "", ErrInvalidPDFVersion},
		{"invalid", "", ErrInvalidPDFVersion},
		{"1.5", "", ErrInvalidPDFVersion},
	}

	for _, testCase := range tests {
		got, err := ParsePDFVersion(testCase.input)
		if testCase.wantErr != nil {
			if !errors.Is(err, testCase.wantErr) {
				t.Errorf("ParsePDFVersion(%q) error = %v, wantErr %v", testCase.input, err, testCase.wantErr)
			}

			continue
		}

		if err != nil {
			t.Errorf("ParsePDFVersion(%q) unexpected error: %v", testCase.input, err)
		}

		if got != testCase.want {
			t.Errorf("ParsePDFVersion(%q) = %q, want %q", testCase.input, got, testCase.want)
		}
	}
}

//nolint:cyclop // sequential getter/setter checks for multiple versions
func TestGlobalPdfVersionSetting(t *testing.T) {
	t.Parallel()

	global := DefaultPdfGlobal()

	// Default Get returns 1.4.
	if got, ok := global.Get("pdfversion"); !ok || got != "1.4" {
		t.Fatalf("initial Get(pdfversion) = %q, %v; want %q, true", got, ok, "1.4")
	}

	// Set valid 1.7.
	if err := global.Set("pdfversion", "1.7"); err != nil {
		t.Fatalf("Set(pdfversion, 1.7): %v", err)
	}

	if global.PdfVersion != "1.7" {
		t.Fatalf("global.PdfVersion = %q, want 1.7", global.PdfVersion)
	}

	if got, ok := global.Get("pdfversion"); !ok || got != "1.7" {
		t.Fatalf("Get(pdfversion) = %q, %v; want %q, true", got, ok, "1.7")
	}

	// Set valid 1.4.
	if err := global.Set("pdfversion", "1.4"); err != nil {
		t.Fatalf("Set(pdfversion, 1.4): %v", err)
	}

	if global.PdfVersion != "1.4" {
		t.Fatalf("global.PdfVersion = %q, want 1.4", global.PdfVersion)
	}

	// Set 2.0 succeeds and stores the canonical value.
	if err := global.Set("pdfversion", "2.0"); err != nil {
		t.Fatalf("Set(pdfversion, 2.0): %v", err)
	}

	if global.PdfVersion != "2.0" {
		t.Fatalf("global.PdfVersion = %q, want 2.0", global.PdfVersion)
	}

	if got, ok := global.Get("pdfversion"); !ok || got != "2.0" {
		t.Fatalf("Get(pdfversion) = %q, %v; want %q, true", got, ok, "2.0")
	}

	// Set invalid returns ErrInvalidPDFVersion.
	if err := global.Set("pdfversion", "9.9"); !errors.Is(err, ErrInvalidPDFVersion) {
		t.Fatalf("Set(pdfversion, 9.9) error = %v, want ErrInvalidPDFVersion", err)
	}
}

//nolint:cyclop,funlen // comprehensive test matrix for PDF profile settings
func TestGlobalPdfProfileSetting(t *testing.T) {
	t.Parallel()

	global := DefaultPdfGlobal()

	// Default Get returns "".
	if got, ok := global.Get("pdfprofile"); !ok || got != "" {
		t.Fatalf("initial Get(pdfprofile) = %q, %v; want %q, true", got, ok, "")
	}

	validCases := []struct {
		input         string
		wantCanonical string
	}{
		{"a3a-ua1", ProfilePDFA3aPDFUA1},
		{"PDF/A-3a+PDF/UA-1", ProfilePDFA3aPDFUA1},
		{"pdf/a-3a+pdf/ua-1", ProfilePDFA3aPDFUA1},
		{"a3a", ProfilePDFA3a},
		{"pdf/a-3a", ProfilePDFA3a},
		{"PDF/A-3a", ProfilePDFA3a},
		{"ua1", ProfilePDFUA1},
		{"pdf/ua-1", ProfilePDFUA1},
		{"PDF/UA-1", ProfilePDFUA1},
		{"a4-ua2", ProfilePDFA4PDFUA2},
		{"PDF/A-4+PDF/UA-2", ProfilePDFA4PDFUA2},
		{"pdf/a-4+pdf/ua-2", ProfilePDFA4PDFUA2},
		{"a4", ProfilePDFA4},
		{"pdf/a-4", ProfilePDFA4},
		{"PDF/A-4", ProfilePDFA4},
		{"ua2", ProfilePDFUA2},
		{"pdf/ua-2", ProfilePDFUA2},
		{"PDF/UA-2", ProfilePDFUA2},
		{"", ProfileNone},
	}

	for _, testCase := range validCases {
		if err := global.Set("pdfprofile", testCase.input); err != nil {
			t.Fatalf("Set(pdfprofile, %q): %v", testCase.input, err)
		}

		if global.PdfProfile != testCase.wantCanonical {
			t.Fatalf("global.PdfProfile after Set(%q) = %q, want %q", testCase.input, global.PdfProfile, testCase.wantCanonical)
		}

		if got, ok := global.Get("pdfprofile"); !ok || got != testCase.wantCanonical {
			t.Fatalf("Get(pdfprofile) after Set(%q) = %q, %v; want %q, true", testCase.input, got, ok, testCase.wantCanonical)
		}
	}

	invalidCases := []struct {
		input   string
		wantErr error
	}{
		{"pdfa-1b", ErrProfilePDFA1Unsupported},
		{"a1", ErrProfilePDFA1Unsupported},
		{"pdf/a-1", ErrProfilePDFA1Unsupported},
		{"unknown", ErrInvalidPDFProfile},
		{"garbage", ErrInvalidPDFProfile},
	}

	for _, testCase := range invalidCases {
		err := global.Set("pdfprofile", testCase.input)
		if err == nil {
			t.Fatalf("Set(pdfprofile, %q) succeeded, want error wrapping %v", testCase.input, testCase.wantErr)
		}

		if !errors.Is(err, testCase.wantErr) {
			t.Fatalf("Set(pdfprofile, %q) error = %v, want error wrapping %v", testCase.input, err, testCase.wantErr)
		}
	}
}
