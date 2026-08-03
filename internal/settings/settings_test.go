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
		{"ColorMode", g.ColorMode, ColorModeColor},
		{"DPI", g.DPI, 96},
		{"ImageDPI", g.ImageDPI, 600},
		{"ImageQuality", g.ImageQuality, 94},
		{"Collate", g.Collate, true},
		{"Outline", g.Outline, true},
		{"OutlineDepth", g.OutlineDepth, 4},
		{"UseCompression", g.UseCompression, true},
		{"Copies", g.Copies, 1},
		{"SmartShrinking", g.SmartShrinking, true},
		{"Background", g.Background, true},
		{"Web.Images", g.Web.Images, true},
		{"Web.Plugins", g.Web.Plugins, false},
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
	if !o.ExternalLinks || !o.LocalLinks || !o.IncludeInOutline || !o.PagesCount || o.ProduceForms {
		t.Errorf("default object = %+v", o)
	}
}

func TestDefaultLoadPageSnapshot(t *testing.T) {
	o := DefaultPdfObject()
	if o.Load.JSDelay != 200 {
		t.Errorf("default jsdelay = %d, want 200", o.Load.JSDelay)
	}
	if !o.Load.BlockLocalFileAccess {
		t.Error("default blockLocalFileAccess must be true")
	}
	if !o.Load.StopSlowScripts {
		t.Error("default stopSlowScripts must be true")
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
		{"orientation", "landscape", func(t *testing.T) {
			if g.Orientation != OrientationLandscape {
				t.Error("orientation not landscape")
			}
		}},
		{"colormode", "grayscale", func(t *testing.T) {
			if g.ColorMode != ColorModeGrayscale {
				t.Error("colormode not grayscale")
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
			if g.Web.Background {
				t.Error("web.background should be false")
			}
		}},
		{"allow", "/srv/html", func(t *testing.T) {
			if len(g.Allow) != 1 || g.Allow[0] != "/srv/html" {
				t.Errorf("allow = %v", g.Allow)
			}
		}},
		{"imagedpi", "300", func(t *testing.T) {
			if g.ImageDPI != 300 {
				t.Errorf("imagedpi = %d", g.ImageDPI)
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

func TestGlobalSetUnknownKey(t *testing.T) {
	g := DefaultPdfGlobal()
	if err := g.Set("bogus.key", "1"); err == nil {
		t.Error("expected error for unknown key")
	}
}

func TestObjectSetDottedKeys(t *testing.T) {
	o := DefaultPdfObject()
	if err := o.Set("load.jsdelay", "500"); err != nil {
		t.Fatal(err)
	}
	if o.Load.JSDelay != 500 {
		t.Errorf("jsdelay = %d", o.Load.JSDelay)
	}
	if err := o.Set("load.blocklocalfileaccess", "false"); err != nil {
		t.Fatal(err)
	}
	if o.Load.BlockLocalFileAccess {
		t.Error("blocklocalfileaccess should be false")
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

func TestUnitRealFormat(t *testing.T) {
	u, _ := ParseUnitReal("10mm", "mm")
	if got := u.FormatUnitReal(); got != "10mm" {
		t.Errorf("FormatUnitReal = %q", got)
	}
	u, _ = ParseUnitReal("7.5", "mm")
	if got := u.FormatUnitReal(); got != "7.5" {
		t.Errorf("FormatUnitReal implied = %q", got)
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
	if v, _ := ParseLogLevel("debug"); v != LogDebug {
		t.Error("log-level debug")
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
