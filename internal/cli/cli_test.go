package cli

import (
	"math"
	"testing"

	"gowkhtmltopdf/internal/settings"
)

func parse(t *testing.T, args ...string) *Command {
	t.Helper()
	cmd, err := Parse(args, nil)
	if err != nil {
		t.Fatalf("Parse(%v): %v", args, err)
	}
	return cmd
}

func TestGlobalFlagsToSettings(t *testing.T) {
	cmd := parse(t,
		"--page-size", "Letter",
		"--orientation", "Landscape",
		"--margin-top", "20mm",
		"--title", "Report",
		"--copies", "3",
		"--no-outline",
		"--outline-depth", "2",
		"--dpi", "150",
		"--grayscale",
		"-q",
		"page.html",
		"out.pdf",
	)
	g := cmd.Global
	if g.PageSize != "Letter" {
		t.Errorf("page-size = %q", g.PageSize)
	}
	if g.Orientation != settings.OrientationLandscape {
		t.Error("orientation")
	}
	if g.Margin.Top != 20 {
		t.Errorf("margin.top = %v", g.Margin.Top)
	}
	if g.Title != "Report" {
		t.Errorf("title = %q", g.Title)
	}
	if g.Copies != 3 {
		t.Errorf("copies = %d", g.Copies)
	}
	if g.Outline {
		t.Error("outline must be false")
	}
	if g.OutlineDepth != 2 {
		t.Errorf("outline-depth = %d", g.OutlineDepth)
	}
	if g.DPI != 150 {
		t.Errorf("dpi = %d", g.DPI)
	}
	if g.ColorMode != settings.ColorModeGrayscale {
		t.Error("grayscale")
	}
	if !g.Quiet {
		t.Error("quiet")
	}
	if cmd.Output != "out.pdf" || len(cmd.Objects) != 1 || cmd.Objects[0].Page != "page.html" {
		t.Errorf("grammar: output=%q objects=%+v", cmd.Output, cmd.Objects)
	}
}

func TestShortFlags(t *testing.T) {
	cmd := parse(t, "-s", "A5", "-O", "Landscape", "-q", "-T", "15mm", "-c", "2", "-t", "T", "in.html", "out.pdf")
	if cmd.Global.PageSize != "A5" {
		t.Errorf("page-size = %q", cmd.Global.PageSize)
	}
	if cmd.Global.Orientation != settings.OrientationLandscape {
		t.Error("orientation")
	}
	if cmd.Global.Margin.Top != 15 {
		t.Errorf("margin-top = %v", cmd.Global.Margin.Top)
	}
	if cmd.Global.Copies != 2 {
		t.Errorf("copies = %d", cmd.Global.Copies)
	}
	if cmd.Global.Title != "T" {
		t.Errorf("title = %q", cmd.Global.Title)
	}
	if !cmd.Global.Quiet {
		t.Error("quiet")
	}
}

func TestFlagEqualsSyntax(t *testing.T) {
	cmd := parse(t, "--page-size=A4", "--orientation=portrait", "in.html", "out.pdf")
	if cmd.Global.PageSize != "A4" {
		t.Error("= syntax page-size")
	}
	if cmd.Global.Orientation != settings.OrientationPortrait {
		t.Error("= syntax orientation")
	}
}

func TestBoolFlagValues(t *testing.T) {
	cmd := parse(t, "--outline=false", "--outline", "in.html", "out.pdf")
	if !cmd.Global.Outline {
		t.Error("--outline=false then --outline must end true")
	}
	cmd = parse(t, "--no-outline", "in.html", "out.pdf")
	if cmd.Global.Outline {
		t.Error("--no-outline must set false")
	}
	// page-scoped web flags bind to the first object (address remapping)
	cmd = parse(t, "--enable-javascript", "in.html", "out.pdf")
	if !cmd.Objects[0].Web.JavaScript {
		t.Error("enable-javascript must land on first object")
	}
	cmd = parse(t, "--disable-javascript", "in.html", "out.pdf")
	if cmd.Objects[0].Web.JavaScript {
		t.Error("disable-javascript")
	}
}

func TestMultiObjectGrammar(t *testing.T) {
	cmd := parse(t,
		"cover", "cover.html",
		"toc",
		"page", "--header-left", "Chapter", "ch1.html",
		"page", "ch2.html",
		"book.pdf",
	)
	if len(cmd.Objects) != 4 {
		t.Fatalf("objects = %d, want 4", len(cmd.Objects))
	}
	cover, toc, ch1, ch2 := cmd.Objects[0], cmd.Objects[1], cmd.Objects[2], cmd.Objects[3]
	if !cover.IsCover || cover.Page != "cover.html" {
		t.Errorf("cover = %+v", cover)
	}
	if cover.IncludeInOutline {
		t.Error("cover includeInOutline must be false")
	}
	if !toc.IsTableOfContent {
		t.Error("toc object")
	}
	if ch1.Page != "ch1.html" || !ch1.HeaderSet || ch1.Header.Left != "Chapter" {
		t.Errorf("ch1 = %+v (header %+v)", ch1, ch1.Header)
	}
	if ch2.Page != "ch2.html" || ch2.HeaderSet {
		t.Errorf("ch2 = %+v", ch2)
	}
	if cmd.Output != "book.pdf" {
		t.Errorf("output = %q", cmd.Output)
	}
}

func TestImplicitFirstPageObject(t *testing.T) {
	cmd := parse(t, "--margin-top", "5mm", "a.html", "b.html", "out.pdf")
	if len(cmd.Objects) != 2 {
		t.Fatalf("objects = %d", len(cmd.Objects))
	}
	if cmd.Objects[0].Page != "a.html" || cmd.Objects[1].Page != "b.html" {
		t.Errorf("objects = %+v", cmd.Objects)
	}
	if cmd.Output != "out.pdf" {
		t.Errorf("output = %q", cmd.Output)
	}
}

func TestStdinOutputAndInput(t *testing.T) {
	cmd := parse(t, "page", "-", "-")
	if cmd.Objects[0].Page != "-" {
		t.Error("stdin input")
	}
	if cmd.Output != "-" {
		t.Error("stdin output")
	}
}

func TestPairFlags(t *testing.T) {
	cmd := parse(t,
		"--cookie", "session", "abc123",
		"--custom-header", "X-Token", "secret",
		"--post", "q", "hello",
		"in.html", "out.pdf",
	)
	o := cmd.Objects[0]
	if o.Load.Cookies["session"] != "abc123" {
		t.Errorf("cookie = %v", o.Load.Cookies)
	}
	if o.Load.CustomHeaders["X-Token"] != "secret" {
		t.Errorf("custom-header = %v", o.Load.CustomHeaders)
	}
	if len(o.Load.Post) != 1 || o.Load.Post[0].Name != "q" || o.Load.Post[0].Value != "hello" {
		t.Errorf("post = %+v", o.Load.Post)
	}
}

func TestHeaderFooterFlags(t *testing.T) {
	cmd := parse(t,
		"--header-left", "L", "--header-right", "R", "--header-font-size", "14",
		"--header-line", "--footer-center", "[page]/[topage]",
		"--footer-spacing", "5",
		"in.html", "out.pdf",
	)
	g := cmd.Global
	if g.Header.Left != "L" || g.Header.Right != "R" || g.Header.FontSize != 14 {
		t.Errorf("header = %+v", g.Header)
	}
	if !g.Header.Line {
		t.Error("header-line")
	}
	if g.Footer.Center != "[page]/[topage]" || g.Footer.Spacing != 5 {
		t.Errorf("footer = %+v", g.Footer)
	}
}

func TestTOCFlags(t *testing.T) {
	cmd := parse(t,
		"--toc-header-text", "Contents",
		"--toc-text-size-shrink", "0.5",
		"--disable-dotted-lines",
		"toc",
		"in.html", "out.pdf",
	)
	if cmd.Global.TOC.CaptionText != "Contents" {
		t.Errorf("caption = %q", cmd.Global.TOC.CaptionText)
	}
	if math.Abs(cmd.Global.TOC.FontScale-0.5) > 1e-9 {
		t.Errorf("fontScale = %v", cmd.Global.TOC.FontScale)
	}
	if cmd.Global.TOC.DottedLines {
		t.Error("dotted-lines must be disabled")
	}
}

func TestLoadFlags(t *testing.T) {
	cmd := parse(t,
		"--javascript-delay", "1500",
		"--zoom", "1.5",
		"--load-error-handling", "ignore",
		"--print-media-type",
		"--username", "u", "--password", "p",
		"in.html", "out.pdf",
	)
	o := cmd.Objects[0]
	if o.Load.JSDelay != 1500 {
		t.Errorf("jsdelay = %d", o.Load.JSDelay)
	}
	if o.Load.ZoomFactor != 1.5 {
		t.Errorf("zoom = %v", o.Load.ZoomFactor)
	}
	if o.Load.LoadErrorHandling != settings.LoadErrorIgnore {
		t.Error("load-error-handling")
	}
	if !o.Load.PrintMediaType {
		t.Error("print-media-type")
	}
	if o.Load.Username != "u" || o.Load.Password != "p" {
		t.Errorf("auth = %q/%q", o.Load.Username, o.Load.Password)
	}
}

func TestUnknownFlagErrors(t *testing.T) {
	if _, err := Parse([]string{"--bogus-flag", "x", "out.pdf"}, nil); err == nil {
		t.Error("unknown flag must error")
	}
	if _, err := Parse([]string{"-Z", "x", "out.pdf"}, nil); err == nil {
		t.Error("unknown short flag must error")
	}
}

func TestDocFlags(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"-h"}} {
		_, err := Parse(args, nil)
		if err != ErrHelp {
			t.Errorf("Parse(%v) = %v, want ErrHelp", args, err)
		}
	}
	for _, args := range [][]string{{"--version"}, {"-V"}} {
		_, err := Parse(args, nil)
		if err != ErrVersion {
			t.Errorf("Parse(%v) = %v, want ErrVersion", args, err)
		}
	}
	_, err := Parse([]string{"--license"}, nil)
	if err != ErrLicense {
		t.Errorf("license = %v", err)
	}
}

func TestEndOfOptions(t *testing.T) {
	cmd := parse(t, "--", "page.html", "out.pdf")
	if cmd.Objects[0].Page != "page.html" || cmd.Output != "out.pdf" {
		t.Errorf("-- handling: %+v", cmd.Objects)
	}
}

func TestImageFlags(t *testing.T) {
	cmd := parse(t,
		"--width", "800", "--height", "600",
		"--crop-x", "10", "--crop-y", "20", "--crop-w", "100", "--crop-h", "50",
		"--quality", "80", "--format", "png", "--transparent", "--no-smart-width",
		"in.html", "out.png",
	)
	img := cmd.Image
	if img.Width != 800 || img.Height != 600 {
		t.Errorf("width/height = %d/%d", img.Width, img.Height)
	}
	if img.Crop.Left != 10 || img.Crop.Top != 20 || img.Crop.Width != 100 || img.Crop.Height != 50 {
		t.Errorf("crop = %+v", img.Crop)
	}
	if img.Quality != 80 || img.Format != "png" || !img.Transparent || img.SmartWidth {
		t.Errorf("image settings = %+v", img)
	}
	if cmd.Output != "out.png" {
		t.Errorf("output = %q", cmd.Output)
	}
}

func TestValidateNoInput(t *testing.T) {
	if _, err := Parse([]string{"toc", "out.pdf"}, nil); err == nil {
		t.Error("toc-only must error (no input page)")
	}
}
