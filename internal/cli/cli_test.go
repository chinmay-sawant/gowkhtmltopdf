package cli

import (
	"bytes"
	"fmt"
	"math"
	"testing"

	"gowkhtmltopdf/internal/settings"
)

func parse(t *testing.T, args ...string) *Command {
	t.Helper()
	cmd, err := Parse(args)
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
	// convert reads Global.Grayscale only.
	if !g.Grayscale {
		t.Error("grayscale must set Global.Grayscale")
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
	// page-scoped flags bind to the first object (address remapping)
	cmd = parse(t, "--enable-local-file-access", "in.html", "out.pdf")
	if cmd.Objects[0].Load.BlockLocalFileAccess {
		t.Error("enable-local-file-access must land on first object")
	}
	if !cmd.Global.Load.EnableLocalFileAccess {
		t.Error("enable-local-file-access must set global")
	}
	cmd = parse(t, "--disable-local-file-access", "in.html", "out.pdf")
	if !cmd.Objects[0].Load.BlockLocalFileAccess {
		t.Error("disable-local-file-access")
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
	// Page-only flags after an object keyword land on that object.
	cmd := parse(t,
		"page",
		"--zoom", "1.5",
		"--load-error-handling", "ignore",
		"--print-media-type",
		"--username", "u", "--password", "p",
		"in.html", "out.pdf",
	)
	o := cmd.Objects[0]
	if o.Load.ZoomFactor != 1.5 {
		t.Errorf("zoom = %v", o.Load.ZoomFactor)
	}
	if o.Load.LoadErrorHandling != settings.LoadErrorIgnore {
		t.Error("load-error-handling")
	}
	if !o.Load.PrintMediaType {
		t.Error("print-media-type on object Load")
	}
	if cmd.Image.Web.PrintMediaType {
		t.Error("print-media-type must not set Image.Web (single home is Global + obj.Load)")
	}
	if !cmd.Global.Web.PrintMediaType {
		t.Error("print-media-type must set Global.Web (convert mediaFor)")
	}
	if o.Load.Username != "u" || o.Load.Password != "p" {
		t.Errorf("auth = %q/%q", o.Load.Username, o.Load.Password)
	}
}

func TestPageOnlyFlagPreObjectPending(t *testing.T) {
	// Pre-object page-only flags remap onto the first page (pending), matching
	// documented smoke recipes: `--zoom 0.67 url out.pdf`.
	cmd := parse(t, "--zoom", "2", "--username", "u", "--password", "p",
		"--timeout", "30", "--external-links", "--internal-links",
		"in.html", "out.pdf")
	o := cmd.Objects[0]
	if o.Load.ZoomFactor != 2 {
		t.Errorf("pre-object zoom pending: got %v", o.Load.ZoomFactor)
	}
	if o.Load.Username != "u" || o.Load.Password != "p" {
		t.Errorf("pre-object auth pending: %q/%q", o.Load.Username, o.Load.Password)
	}
	if o.Load.Timeout != 30 {
		t.Errorf("pre-object timeout pending: got %v", o.Load.Timeout)
	}
	if !o.ExternalLinks {
		t.Error("pre-object external-links pending")
	}
	if !o.LocalLinks {
		t.Error("pre-object internal-links (locallinks) pending")
	}
	// After an object keyword they land on the object.
	cmd = parse(t, "page", "--zoom", "2", "--external-links", "in.html", "out.pdf")
	if cmd.Objects[0].Load.ZoomFactor != 2 {
		t.Error("zoom must land on the object after page keyword")
	}
	if !cmd.Objects[0].ExternalLinks {
		t.Error("external-links must land on the object after page keyword")
	}
	// Leading toc does not consume pending; zoom applies to the page after.
	cmd = parse(t, "--zoom", "1.5", "toc", "page", "in.html", "out.pdf")
	var body *settings.PdfObject
	for i := range cmd.Objects {
		if !cmd.Objects[i].IsTableOfContent {
			body = &cmd.Objects[i]
			break
		}
	}
	if body == nil || body.Load.ZoomFactor != 1.5 {
		t.Errorf("pre-object zoom after toc must land on body page; body=%v", body)
	}
}

func TestGrayscaleSetsConvertField(t *testing.T) {
	cmd := parse(t, "--grayscale", "in.html", "out.pdf")
	if !cmd.Global.Grayscale {
		t.Error("--grayscale must set Global.Grayscale (convert.SetGrayscale)")
	}
	cmd = parse(t, "--no-grayscale", "in.html", "out.pdf")
	if cmd.Global.Grayscale {
		t.Error("--no-grayscale must clear Global.Grayscale")
	}
}

func TestSmartShrinkingEnableDisable(t *testing.T) {
	// Default is on; disable pair only (no bare --smart-shrinking).
	cmd := parse(t, "--disable-smart-shrinking", "in.html", "out.pdf")
	if cmd.Global.SmartShrinking {
		t.Error("disable-smart-shrinking")
	}
	cmd = parse(t, "--disable-smart-shrinking", "--enable-smart-shrinking", "in.html", "out.pdf")
	if !cmd.Global.SmartShrinking {
		t.Error("enable-smart-shrinking must re-enable")
	}
	if _, err := Parse([]string{"--smart-shrinking", "in.html", "out.pdf"}); err == nil {
		t.Error("bare --smart-shrinking must be unknown (pair only)")
	}
}

func TestBackgroundPDFAndImage(t *testing.T) {
	// Both convert and imageout read Global.Background.
	cmd := parse(t, "--no-background", "in.html", "out.pdf")
	if cmd.Global.Background {
		t.Error("no-background must clear Global.Background")
	}
	cmd = parse(t, "--no-background", "--background", "in.html", "out.pdf")
	if !cmd.Global.Background {
		t.Error("--background must set Global.Background")
	}
}

func TestDumpOutlineGlobalHome(t *testing.T) {
	// Single home: Global settings; negation rides the value.
	cmd := parse(t, "--dump-outline", "in.html", "out.pdf")
	if !cmd.Global.DumpOutline {
		t.Error("--dump-outline must set Global.DumpOutline")
	}
	cmd = parse(t, "--no-dump-outline", "in.html", "out.pdf")
	if cmd.Global.DumpOutline {
		t.Error("--no-dump-outline must clear Global.DumpOutline")
	}
	cmd = parse(t, "--dump-default-toc-xsl", "in.html", "out.pdf")
	if !cmd.Global.DumpDefaultTOCXSL {
		t.Error("--dump-default-toc-xsl must set Global.DumpDefaultTOCXSL")
	}
	cmd = parse(t, "--dump-default-toc-xsl=false", "in.html", "out.pdf")
	if cmd.Global.DumpDefaultTOCXSL {
		t.Error("--dump-default-toc-xsl=false must clear Global.DumpDefaultTOCXSL")
	}
}

func TestStubFlagsRemoved(t *testing.T) {
	// Policy A: inert engine-less flags are rejected, not accepted no-ops.
	cases := [][]string{
		{"--dpi", "150", "in.html", "out.pdf"},
		{"--image-dpi", "300", "in.html", "out.pdf"},
		{"--image-quality", "80", "in.html", "out.pdf"},
		{"--lowquality", "in.html", "out.pdf"},
		{"--use-xserver", "in.html", "out.pdf"},
		{"--cookie-jar", "jar.txt", "in.html", "out.pdf"},
		{"--read-args-from-stdin", "in.html", "out.pdf"},
		{"--log-level", "info", "in.html", "out.pdf"},
		{"--javascript-delay", "1000", "in.html", "out.pdf"},
		{"--window-status", "ready", "in.html", "out.pdf"},
		{"--run-script", "x.js", "in.html", "out.pdf"},
		{"--debug-javascript", "in.html", "out.pdf"},
		{"--user-style-sheet", "s.css", "in.html", "out.pdf"},
		{"--minimum-font-size", "8", "in.html", "out.pdf"},
		{"--enable-plugins", "in.html", "out.pdf"},
		{"--produce-forms", "in.html", "out.pdf"},
		{"--enable-javascript", "in.html", "out.pdf"},
		{"--stop-slow-scripts", "in.html", "out.pdf"},
		{"--default-encoding", "utf-8", "in.html", "out.pdf"},
		{"--custom-header-propagation", "in.html", "out.pdf"},
	}
	for _, args := range cases {
		if _, err := Parse(args); err == nil {
			t.Errorf("stub flag %v must be unknown", args[0])
		}
	}
}

func TestSimplifyDOMFlag(t *testing.T) {
	cmd := parse(t, "--simplify-dom", "in.html", "out.pdf")
	if !cmd.Global.Web.SimplifyDOM {
		t.Error("global web.simplifydom")
	}
	if !cmd.Objects[0].Web.SimplifyDOM {
		t.Error("object web.simplifydom")
	}
	cmd = parse(t, "--simplify-dom", "--no-simplify-dom", "in.html", "out.pdf")
	if cmd.Global.Web.SimplifyDOM || cmd.Objects[0].Web.SimplifyDOM {
		t.Error("--no-simplify-dom should clear the flag")
	}
}

func TestSimplifyDOMProfileFlag(t *testing.T) {
	cmd := parse(t, "--simplify-dom", "--simplify-dom-profile", "mediawiki", "in.html", "out.pdf")
	if cmd.Global.Web.SimplifyDOMProfile != "mediawiki" {
		t.Errorf("profile=%q", cmd.Global.Web.SimplifyDOMProfile)
	}
	if cmd.Objects[0].Web.SimplifyDOMProfile != "mediawiki" {
		t.Errorf("object profile=%q", cmd.Objects[0].Web.SimplifyDOMProfile)
	}
}

func TestPrintLinkUnderlineFlag(t *testing.T) {
	cmd := parse(t, "--print-link-underline", "in.html", "out.pdf")
	if !cmd.Global.Web.PrintLinkUnderline {
		t.Error("global printlinkunderline")
	}
	if !cmd.Objects[0].Web.PrintLinkUnderline {
		t.Error("object printlinkunderline")
	}
}

func TestUnknownFlagErrors(t *testing.T) {
	if _, err := Parse([]string{"--bogus-flag", "x", "out.pdf"}); err == nil {
		t.Error("unknown flag must error")
	}
	if _, err := Parse([]string{"-Z", "x", "out.pdf"}); err == nil {
		t.Error("unknown short flag must error")
	}
}

func TestDocFlags(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"-h"}} {
		_, err := Parse(args)
		if err != ErrHelp {
			t.Errorf("Parse(%v) = %v, want ErrHelp", args, err)
		}
	}
	for _, args := range [][]string{{"--version"}, {"-V"}} {
		_, err := Parse(args)
		if err != ErrVersion {
			t.Errorf("Parse(%v) = %v, want ErrVersion", args, err)
		}
	}
	_, err := Parse([]string{"--license"})
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
	if _, err := Parse([]string{"toc", "out.pdf"}); err == nil {
		t.Error("toc-only must error (no input page)")
	}
}

// Page-scoped flags before object keywords must not leave an empty ghost
// page when the next token is toc (or cover/page). They apply to the first
// real page instead; header flags before any object stay global.
func TestPageScopedBeforeTOCNoGhost(t *testing.T) {
	cmd := parse(t,
		"--enable-local-file-access",
		"--header-left", "Demo",
		"--footer-center", "[page]",
		"toc",
		"in.html",
		"out.pdf",
	)
	if len(cmd.Objects) != 2 {
		t.Fatalf("objects = %d, want 2 (toc + page); got %+v", len(cmd.Objects), cmd.Objects)
	}
	if !cmd.Objects[0].IsTableOfContent {
		t.Errorf("object 0 should be toc: %+v", cmd.Objects[0])
	}
	if cmd.Objects[1].Page != "in.html" {
		t.Errorf("object 1 page = %q", cmd.Objects[1].Page)
	}
	if cmd.Objects[1].Load.BlockLocalFileAccess {
		t.Error("enable-local-file-access must land on the first real page")
	}
	if !cmd.Global.Load.EnableLocalFileAccess {
		t.Error("enable-local-file-access must set the global flag")
	}
	// Header/footer before any object keyword remain global so every object
	// (including the TOC) inherits them via HeaderFor/FooterFor.
	if cmd.Global.Header.Left != "Demo" || cmd.Global.Footer.Center != "[page]" {
		t.Errorf("global hf = header%+v footer%+v", cmd.Global.Header, cmd.Global.Footer)
	}
	if cmd.Objects[1].HeaderSet {
		t.Error("header must not bind to the page object when set before any object")
	}
}

func TestPageScopedBeforePageKeyword(t *testing.T) {
	cmd := parse(t, "--enable-local-file-access", "page", "in.html", "out.pdf")
	if len(cmd.Objects) != 1 {
		t.Fatalf("objects = %d, want 1; got %+v", len(cmd.Objects), cmd.Objects)
	}
	if cmd.Objects[0].Page != "in.html" || cmd.Objects[0].Load.BlockLocalFileAccess {
		t.Errorf("page = %+v", cmd.Objects[0])
	}
}

func TestExitCode(t *testing.T) {
	if ExitCode(nil) != ExitError {
		t.Error("nil must exit 1")
	}
	if ExitCode(fmt.Errorf("boom")) != ExitError {
		t.Error("plain error must exit 1")
	}
	if ExitCode(&settings.HttpStatusError{Status: 404}) != 2 {
		t.Error("404 must exit 2")
	}
	if ExitCode(&settings.HttpStatusError{Status: 401}) != 3 {
		t.Error("401 must exit 3")
	}
	// Wrapped errors still resolve through errors.As.
	if ExitCode(fmt.Errorf("wrap: %w", &settings.HttpStatusError{Status: 404})) != 2 {
		t.Error("wrapped 404 must exit 2")
	}
}

func TestOpenOutputWriterPrecedence(t *testing.T) {
	var buf bytes.Buffer
	cmd := &Command{
		Output:       "/tmp/should-not-be-created-by-openoutput-test.pdf",
		OutputWriter: &buf,
	}
	w, closeW, err := cmd.OpenOutput()
	if err != nil {
		t.Fatal(err)
	}
	defer closeW()
	if w != &buf {
		t.Fatal("OutputWriter must win over Output path")
	}
	if _, err := w.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "x" {
		t.Fatalf("buf=%q", buf.String())
	}
}
