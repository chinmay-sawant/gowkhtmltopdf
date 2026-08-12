package cli //nolint:testpackage // tests exercise unexported flagTable/shortFlags

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/settings"
)

// errBoom is a plain non-http error used by ExitCode tests.
var errBoom = errors.New("boom")

const outPDF = "out.pdf"

func parse(t *testing.T, args ...string) *Command {
	t.Helper()

	cmd, err := Parse(args)
	if err != nil {
		t.Fatalf("Parse(%v): %v", args, err)
	}

	return cmd
}

func TestGlobalFlagsToSettings(t *testing.T) {
	t.Parallel()

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
		outPDF,
	)

	assertGlobalFlagSettings(t, cmd.Global)

	if cmd.Output != outPDF || len(cmd.Objects) != 1 || cmd.Objects[0].Page != "page.html" {
		t.Errorf("grammar: output=%q objects=%+v", cmd.Output, cmd.Objects)
	}
}

// assertGlobalFlagSettings pins the settings written by global flags.
func assertGlobalFlagSettings(t *testing.T, globalCfg settings.PdfGlobal) {
	t.Helper()

	if globalCfg.PageSize != "Letter" {
		t.Errorf("page-size = %q", globalCfg.PageSize)
	}

	if globalCfg.Orientation != settings.OrientationLandscape {
		t.Error("orientation")
	}

	if globalCfg.Margin.Top != 20 {
		t.Errorf("margin.top = %v", globalCfg.Margin.Top)
	}

	if globalCfg.Title != "Report" {
		t.Errorf("title = %q", globalCfg.Title)
	}

	if globalCfg.Copies != 3 {
		t.Errorf("copies = %d", globalCfg.Copies)
	}

	if globalCfg.Outline {
		t.Error("outline must be false")
	}

	if globalCfg.OutlineDepth != 2 {
		t.Errorf("outline-depth = %d", globalCfg.OutlineDepth)
	}
	// convert reads Global.Grayscale only.
	if !globalCfg.Grayscale {
		t.Error("grayscale must set Global.Grayscale")
	}

	if !globalCfg.Quiet {
		t.Error("quiet")
	}
}

func TestShortFlags(t *testing.T) {
	t.Parallel()

	cmd := parse(t, "-s", "A5", "-O", "Landscape", "-q", "-T", "15mm", "-c", "2", "-t", "T", "in.html", outPDF)
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
	t.Parallel()

	cmd := parse(t, "--page-size=A4", "--orientation=portrait", "in.html", outPDF)
	if cmd.Global.PageSize != "A4" {
		t.Error("= syntax page-size")
	}

	if cmd.Global.Orientation != settings.OrientationPortrait {
		t.Error("= syntax orientation")
	}
}

func TestBoolFlagValues(t *testing.T) {
	t.Parallel()

	cmd := parse(t, "--outline=false", "--outline", "in.html", outPDF)
	if !cmd.Global.Outline {
		t.Error("--outline=false then --outline must end true")
	}

	cmd = parse(t, "--no-outline", "in.html", outPDF)
	if cmd.Global.Outline {
		t.Error("--no-outline must set false")
	}
	// page-scoped flags bind to the first object (address remapping)
	cmd = parse(t, "--enable-local-file-access", "in.html", outPDF)
	if cmd.Objects[0].Load.BlockLocalFileAccess {
		t.Error("enable-local-file-access must land on first object")
	}

	if !cmd.Global.Load.EnableLocalFileAccess {
		t.Error("enable-local-file-access must set global")
	}

	cmd = parse(t, "--disable-local-file-access", "in.html", outPDF)
	if !cmd.Objects[0].Load.BlockLocalFileAccess {
		t.Error("disable-local-file-access")
	}
}

func TestMultiObjectGrammar(t *testing.T) {
	t.Parallel()

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

	assertCoverAndTOC(t, cmd.Objects[0], cmd.Objects[1])

	if got := cmd.Objects[2]; got.Page != "ch1.html" || !got.HeaderSet || got.Header.Left != "Chapter" {
		t.Errorf("ch1 = %+v (header %+v)", got, got.Header)
	}

	if got := cmd.Objects[3]; got.Page != "ch2.html" || got.HeaderSet {
		t.Errorf("ch2 = %+v", got)
	}

	if cmd.Output != "book.pdf" {
		t.Errorf("output = %q", cmd.Output)
	}
}

// assertCoverAndTOC pins the cover and toc object shapes.
func assertCoverAndTOC(t *testing.T, cover, toc settings.PdfObject) {
	t.Helper()

	if !cover.IsCover || cover.Page != "cover.html" || cover.IncludeInOutline {
		t.Errorf("cover = %+v", cover)
	}

	if !toc.IsTableOfContent {
		t.Error("toc object")
	}
}

func TestImplicitFirstPageObject(t *testing.T) {
	t.Parallel()

	cmd := parse(t, "--margin-top", "5mm", "a.html", "b.html", outPDF)
	if len(cmd.Objects) != 2 {
		t.Fatalf("objects = %d", len(cmd.Objects))
	}

	if cmd.Objects[0].Page != "a.html" || cmd.Objects[1].Page != "b.html" {
		t.Errorf("objects = %+v", cmd.Objects)
	}

	if cmd.Output != outPDF {
		t.Errorf("output = %q", cmd.Output)
	}
}

func TestStdinOutputAndInput(t *testing.T) {
	t.Parallel()

	cmd := parse(t, "page", "-", "-")
	if cmd.Objects[0].Page != "-" {
		t.Error("stdin input")
	}

	if cmd.Output != "-" {
		t.Error("stdin output")
	}
}

func TestPairFlags(t *testing.T) {
	t.Parallel()

	cmd := parse(t,
		"--cookie", "session", "abc123",
		"--custom-header", "X-Token", "secret",
		"--post", "q", "hello",
		"in.html", outPDF,
	)

	obj := cmd.Objects[0]
	if obj.Load.Cookies["session"] != "abc123" {
		t.Errorf("cookie = %v", obj.Load.Cookies)
	}

	if obj.Load.CustomHeaders["X-Token"] != "secret" {
		t.Errorf("custom-header = %v", obj.Load.CustomHeaders)
	}

	if len(obj.Load.Post) != 1 || obj.Load.Post[0].Name != "q" || obj.Load.Post[0].Value != "hello" {
		t.Errorf("post = %+v", obj.Load.Post)
	}
}

func TestHeaderFooterFlags(t *testing.T) {
	t.Parallel()

	cmd := parse(t,
		"--header-left", "L", "--header-right", "R", "--header-font-size", "14",
		"--header-line", "--footer-center", "[page]/[topage]",
		"--footer-spacing", "5",
		"in.html", outPDF,
	)

	globalCfg := cmd.Global
	if globalCfg.Header.Left != "L" || globalCfg.Header.Right != "R" || globalCfg.Header.FontSize != 14 {
		t.Errorf("header = %+v", globalCfg.Header)
	}

	if !globalCfg.Header.Line {
		t.Error("header-line")
	}

	if globalCfg.Footer.Center != "[page]/[topage]" || globalCfg.Footer.Spacing != 5 {
		t.Errorf("footer = %+v", globalCfg.Footer)
	}
}

func TestTOCFlags(t *testing.T) {
	t.Parallel()

	cmd := parse(t,
		"--toc-header-text", "Contents",
		"--toc-text-size-shrink", "0.5",
		"--disable-dotted-lines",
		"toc",
		"in.html", outPDF,
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
	t.Parallel()

	// Page-only flags after an object keyword land on that object.
	cmd := parse(t,
		"page",
		"--zoom", "1.5",
		"--load-error-handling", "ignore",
		"--print-media-type",
		"--username", "u", "--password", "p",
		"in.html", outPDF,
	)

	obj := cmd.Objects[0]
	if obj.Load.ZoomFactor != 1.5 {
		t.Errorf("zoom = %v", obj.Load.ZoomFactor)
	}

	if obj.Load.LoadErrorHandling != settings.LoadErrorIgnore {
		t.Error("load-error-handling")
	}

	if !obj.Load.PrintMediaType {
		t.Error("print-media-type on object Load")
	}

	if cmd.Image.Web.PrintMediaType {
		t.Error("print-media-type must not set Image.Web (single home is Global + obj.Load)")
	}

	if !cmd.Global.Web.PrintMediaType {
		t.Error("print-media-type must set Global.Web (convert mediaFor)")
	}

	if obj.Load.Username != "u" || obj.Load.Password != "p" {
		t.Errorf("auth = %q/%q", obj.Load.Username, obj.Load.Password)
	}
}

func TestPageOnlyFlagPreObjectPending(t *testing.T) {
	t.Parallel()

	// Pre-object page-only flags remap onto the first page (pending), matching
	// documented smoke recipes: `--zoom 0.67 url out.pdf`.
	cmd := parse(t, "--zoom", "2", "--username", "u", "--password", "p",
		"--timeout", "30", "--external-links", "--internal-links",
		"in.html", outPDF)

	assertPreObjectPending(t, cmd.Objects[0])

	// After an object keyword they land on the object.
	cmd = parse(t, "page", "--zoom", "2", "--external-links", "in.html", outPDF)
	assertPostKeywordPending(t, cmd.Objects[0])

	// Leading toc does not consume pending; zoom applies to the page after.
	cmd = parse(t, "--zoom", "1.5", "toc", "page", "in.html", outPDF)

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

// assertPreObjectPending pins pending page-only flags before any keyword.
func assertPreObjectPending(t *testing.T, obj settings.PdfObject) {
	t.Helper()

	if obj.Load.ZoomFactor != 2 {
		t.Errorf("pre-object zoom pending: got %v", obj.Load.ZoomFactor)
	}

	if obj.Load.Username != "u" || obj.Load.Password != "p" {
		t.Errorf("pre-object auth pending: %q/%q", obj.Load.Username, obj.Load.Password)
	}

	if obj.Load.Timeout != 30 {
		t.Errorf("pre-object timeout pending: got %v", obj.Load.Timeout)
	}

	if !obj.ExternalLinks {
		t.Error("pre-object external-links pending")
	}

	if !obj.LocalLinks {
		t.Error("pre-object internal-links (locallinks) pending")
	}
}

// assertPostKeywordPending pins page-only flags after a page keyword.
func assertPostKeywordPending(t *testing.T, obj settings.PdfObject) {
	t.Helper()

	if obj.Load.ZoomFactor != 2 {
		t.Error("zoom must land on the object after page keyword")
	}

	if !obj.ExternalLinks {
		t.Error("external-links must land on the object after page keyword")
	}
}

func TestGrayscaleSetsConvertField(t *testing.T) {
	t.Parallel()

	cmd := parse(t, "--grayscale", "in.html", outPDF)
	if !cmd.Global.Grayscale {
		t.Error("--grayscale must set Global.Grayscale (convert.SetGrayscale)")
	}

	cmd = parse(t, "--no-grayscale", "in.html", outPDF)
	if cmd.Global.Grayscale {
		t.Error("--no-grayscale must clear Global.Grayscale")
	}
}

func TestSmartShrinkingEnableDisable(t *testing.T) {
	t.Parallel()

	// Default is on; disable pair only (no bare --smart-shrinking).
	cmd := parse(t, "--disable-smart-shrinking", "in.html", outPDF)
	if cmd.Global.SmartShrinking {
		t.Error("disable-smart-shrinking")
	}

	cmd = parse(t, "--disable-smart-shrinking", "--enable-smart-shrinking", "in.html", outPDF)
	if !cmd.Global.SmartShrinking {
		t.Error("enable-smart-shrinking must re-enable")
	}

	if _, err := Parse([]string{"--smart-shrinking", "in.html", outPDF}); err == nil {
		t.Error("bare --smart-shrinking must be unknown (pair only)")
	}
}

func TestBackgroundPDFAndImage(t *testing.T) {
	t.Parallel()

	// Both convert and imageout read Global.Background.
	cmd := parse(t, "--no-background", "in.html", outPDF)
	if cmd.Global.Background {
		t.Error("no-background must clear Global.Background")
	}

	cmd = parse(t, "--no-background", "--background", "in.html", outPDF)
	if !cmd.Global.Background {
		t.Error("--background must set Global.Background")
	}
}

func TestDumpOutlineGlobalHome(t *testing.T) {
	t.Parallel()

	// Single home: Global settings; negation rides the value.
	cmd := parse(t, "--dump-outline", "in.html", outPDF)
	if !cmd.Global.DumpOutline {
		t.Error("--dump-outline must set Global.DumpOutline")
	}

	cmd = parse(t, "--no-dump-outline", "in.html", outPDF)
	if cmd.Global.DumpOutline {
		t.Error("--no-dump-outline must clear Global.DumpOutline")
	}

	cmd = parse(t, "--dump-default-toc-xsl")
	if !cmd.Global.DumpDefaultTOCXSL {
		t.Error("--dump-default-toc-xsl must set Global.DumpDefaultTOCXSL")
	}

	cmd = parse(t, "--dump-default-toc-xsl=false", "in.html", outPDF)
	if cmd.Global.DumpDefaultTOCXSL {
		t.Error("--dump-default-toc-xsl=false must clear Global.DumpDefaultTOCXSL")
	}
}

func TestDumpDefaultTOCXSLTerminalValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "input", args: []string{"--dump-default-toc-xsl", "in.html"}},
		{name: "output", args: []string{"--dump-default-toc-xsl", "in.html", outPDF}},
		{name: "outline", args: []string{"--dump-default-toc-xsl", "--dump-outline"}},
		{name: "malformed boolean", args: []string{"--dump-default-toc-xsl=maybe"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := Parse(tc.args); err == nil {
				t.Fatalf("Parse(%v) succeeded; want terminal validation error", tc.args)
			}
		})
	}

	if _, err := Parse([]string{"--dump-default-toc-xsl"}, ModeImage); err == nil {
		t.Fatal("--dump-default-toc-xsl accepted in image mode")
	}
}

func TestStubFlagsRemoved(t *testing.T) {
	t.Parallel()

	// Policy A: inert engine-less flags are rejected, not accepted no-ops.
	cases := [][]string{
		{"--dpi", "150", "in.html", outPDF},
		{"--image-dpi", "300", "in.html", outPDF},
		{"--image-quality", "80", "in.html", outPDF},
		{"--lowquality", "in.html", outPDF},
		{"--use-xserver", "in.html", outPDF},
		{"--cookie-jar", "jar.txt", "in.html", outPDF},
		{"--read-args-from-stdin", "in.html", outPDF},
		{"--log-level", "info", "in.html", outPDF},
		{"--javascript-delay", "1000", "in.html", outPDF},
		{"--window-status", "ready", "in.html", outPDF},
		{"--run-script", "x.js", "in.html", outPDF},
		{"--debug-javascript", "in.html", outPDF},
		{"--user-style-sheet", "s.css", "in.html", outPDF},
		{"--minimum-font-size", "8", "in.html", outPDF},
		{"--enable-plugins", "in.html", outPDF},
		{"--produce-forms", "in.html", outPDF},
		{"--enable-javascript", "in.html", outPDF},
		{"--stop-slow-scripts", "in.html", outPDF},
		{"--default-encoding", "utf-8", "in.html", outPDF},
		{"--custom-header-propagation", "in.html", outPDF},
	}
	for _, args := range cases {
		if _, err := Parse(args); err == nil {
			t.Errorf("stub flag %v must be unknown", args[0])
		}
	}
}

func TestSimplifyDOMFlag(t *testing.T) {
	t.Parallel()

	cmd := parse(t, "--simplify-dom", "in.html", outPDF)
	if !cmd.Global.Web.SimplifyDOM {
		t.Error("global web.simplifydom")
	}

	if !cmd.Objects[0].Web.SimplifyDOM {
		t.Error("object web.simplifydom")
	}

	cmd = parse(t, "--simplify-dom", "--no-simplify-dom", "in.html", outPDF)
	if cmd.Global.Web.SimplifyDOM || cmd.Objects[0].Web.SimplifyDOM {
		t.Error("--no-simplify-dom should clear the flag")
	}
}

func TestSimplifyDOMProfileFlag(t *testing.T) {
	t.Parallel()

	cmd := parse(t, "--simplify-dom", "--simplify-dom-profile", "mediawiki", "in.html", outPDF)
	if cmd.Global.Web.SimplifyDOMProfile != "mediawiki" {
		t.Errorf("profile=%q", cmd.Global.Web.SimplifyDOMProfile)
	}

	if cmd.Objects[0].Web.SimplifyDOMProfile != "mediawiki" {
		t.Errorf("object profile=%q", cmd.Objects[0].Web.SimplifyDOMProfile)
	}
}

func TestPrintLinkUnderlineFlag(t *testing.T) {
	t.Parallel()

	cmd := parse(t, "--print-link-underline", "in.html", outPDF)
	if !cmd.Global.Web.PrintLinkUnderline {
		t.Error("global printlinkunderline")
	}

	if !cmd.Objects[0].Web.PrintLinkUnderline {
		t.Error("object printlinkunderline")
	}
}

func TestUnknownFlagErrors(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]string{"--bogus-flag", "x", outPDF}); err == nil {
		t.Error("unknown flag must error")
	}

	if _, err := Parse([]string{"-Z", "x", outPDF}); err == nil {
		t.Error("unknown short flag must error")
	}
}

func TestDocFlags(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{{"--help"}, {"-h"}} {
		_, err := Parse(args)
		if !errors.Is(err, ErrHelp) {
			t.Errorf("Parse(%v) = %v, want ErrHelp", args, err)
		}
	}

	for _, args := range [][]string{{"--version"}, {"-V"}} {
		_, err := Parse(args)
		if !errors.Is(err, ErrVersion) {
			t.Errorf("Parse(%v) = %v, want ErrVersion", args, err)
		}
	}

	_, err := Parse([]string{"--license"})
	if !errors.Is(err, ErrLicense) {
		t.Errorf("license = %v", err)
	}
}

func TestDocFlagBoundaryBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want error
	}{
		{name: "help value remains invalid", args: []string{"--help=x"}, want: errInvalidBoolValue},
		{name: "help cannot be negated", args: []string{"--no-help", "in.html", outPDF}, want: errUnknownOption},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if _, err := Parse(testCase.args); !errors.Is(err, testCase.want) {
				t.Fatalf("Parse(%v) = %v, want errors.Is(..., %v)", testCase.args, err, testCase.want)
			}
		})
	}
}

func TestShortFlagClusterIsRejected(t *testing.T) {
	t.Parallel()

	_, err := Parse([]string{"-xyz", "in.html", outPDF})
	if !errors.Is(err, errUnknownOption) {
		t.Fatalf("Parse(-xyz ...) = %v, want errors.Is(..., %v)", err, errUnknownOption)
	}

	if !strings.Contains(err.Error(), "-xyz") {
		t.Fatalf("Parse(-xyz ...) = %v, want offending token", err)
	}
}

func TestPrintHelpUsesProductTruth(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name string
		mode Mode
		want string
	}{
		{name: "pdf", mode: ModePDF, want: "Convert HTML to PDF"},
		{name: "image", mode: ModeImage, want: "Convert HTML to PNG/JPEG image"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var help bytes.Buffer

			PrintHelp(&help, testCase.mode)

			got := help.String()
			if !strings.Contains(got, testCase.want) {
				t.Errorf("help missing %q:\n%s", testCase.want, got)
			}

			for _, forbidden := range []string{"Qt WebKit engine", "using the Qt WebKit"} {
				if strings.Contains(got, forbidden) {
					t.Errorf("help contains stale implementation claim %q", forbidden)
				}
			}

			if !strings.Contains(got, "No JavaScript, browser process") {
				t.Errorf("help missing browser/JavaScript boundary:\n%s", got)
			}
		})
	}
}

func TestEndOfOptions(t *testing.T) {
	t.Parallel()

	cmd := parse(t, "--", "page.html", outPDF)
	if cmd.Objects[0].Page != "page.html" || cmd.Output != outPDF {
		t.Errorf("-- handling: %+v", cmd.Objects)
	}
}

func TestImageFlags(t *testing.T) {
	t.Parallel()

	cmd := parse(t,
		"--width", "800", "--height", "600",
		"--crop-x", "10", "--crop-y", "20", "--crop-w", "100", "--crop-h", "50",
		"--quality", "80", "--format", "png", "--transparent", "--no-smart-width",
		"in.html", "out.png",
	)

	assertImageDimensions(t, cmd.Image)

	if img := cmd.Image; img.Quality != 80 || img.Format != "png" || !img.Transparent || img.SmartWidth {
		t.Errorf("image settings = %+v", img)
	}

	if cmd.Output != "out.png" {
		t.Errorf("output = %q", cmd.Output)
	}
}

// assertImageDimensions pins width/height and the crop rect.
func assertImageDimensions(t *testing.T, img settings.ImageGlobal) {
	t.Helper()

	if img.Width != 800 || img.Height != 600 {
		t.Errorf("width/height = %d/%d", img.Width, img.Height)
	}

	if img.Crop.Left != 10 || img.Crop.Top != 20 || img.Crop.Width != 100 || img.Crop.Height != 50 {
		t.Errorf("crop = %+v", img.Crop)
	}
}

func TestParseModeRejectsInapplicableFlags(t *testing.T) {
	t.Parallel()

	for name, spec := range flagTable {
		if spec.mod == ModeBoth {
			continue
		}

		mode := ModeImage
		if spec.mod == ModeImage {
			mode = ModePDF
		}

		_, err := Parse([]string{"--" + name, "input.html", "output"}, mode)
		if err == nil {
			t.Errorf("Parse(%q, %v) accepted an inapplicable flag", name, mode)

			continue
		}

		if !strings.Contains(err.Error(), "not supported") {
			t.Errorf("Parse(%q, %v) error = %v, want applicability error", name, mode, err)
		}
	}

	for name, spec := range shortFlags {
		if spec.mod == ModeBoth {
			continue
		}

		if _, err := Parse([]string{"-" + name, "input.html", "output"}, ModeImage); err == nil {
			t.Errorf("Parse(%q, image mode) accepted an inapplicable short flag", name)
		}
	}
}

func TestParseModeAcceptsApplicableFlags(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]string{"--page-size", "Letter", "input.html", "output.pdf"}, ModePDF); err != nil {
		t.Fatalf("PDF flag rejected in PDF mode: %v", err)
	}

	if _, err := Parse([]string{"--width", "800", "input.html", "output.png"}, ModeImage); err != nil {
		t.Fatalf("image flag rejected in image mode: %v", err)
	}

	if _, err := Parse([]string{"--quiet", "input.html", "output"}, ModePDF); err != nil {
		t.Fatalf("shared flag rejected in PDF mode: %v", err)
	}

	if _, err := Parse([]string{"--quiet", "input.html", "output"}, ModeImage); err != nil {
		t.Fatalf("shared flag rejected in image mode: %v", err)
	}

	if _, err := ParseMode([]string{"--width", "800", "input.html", "output.png"}, ModeImage); err != nil {
		t.Fatalf("ParseMode rejected image flag: %v", err)
	}
}

func TestParseModeValidation(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]string{"input.html", "output"}, 0); err == nil {
		t.Fatal("zero mode accepted")
	}

	if _, err := Parse([]string{"input.html", "output"}, Mode(8)); err == nil {
		t.Fatal("unknown mode accepted")
	}

	if _, err := Parse([]string{"input.html", "output"}, ModePDF, ModeImage); err == nil {
		t.Fatal("multiple modes accepted")
	}
}

func TestValidateNoInput(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]string{"toc", outPDF}); err == nil {
		t.Error("toc-only must error (no input page)")
	}
}

// Page-scoped flags before object keywords must not leave an empty ghost
// page when the next token is toc (or cover/page). They apply to the first
// real page instead; header flags before any object stay global.
func TestPageScopedBeforeTOCNoGhost(t *testing.T) {
	t.Parallel()

	cmd := parse(t,
		"--enable-local-file-access",
		"--header-left", "Demo",
		"--footer-center", "[page]",
		"toc",
		"in.html",
		outPDF,
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
	t.Parallel()

	cmd := parse(t, "--enable-local-file-access", "page", "in.html", outPDF)
	if len(cmd.Objects) != 1 {
		t.Fatalf("objects = %d, want 1; got %+v", len(cmd.Objects), cmd.Objects)
	}

	if cmd.Objects[0].Page != "in.html" || cmd.Objects[0].Load.BlockLocalFileAccess {
		t.Errorf("page = %+v", cmd.Objects[0])
	}
}

func TestExitCode(t *testing.T) {
	t.Parallel()

	if ExitCode(nil) != ExitError {
		t.Error("nil must exit 1")
	}

	if ExitCode(errBoom) != ExitError {
		t.Error("plain error must exit 1")
	}

	if ExitCode(&settings.HttpStatusError{Status: 404, URL: ""}) != 2 {
		t.Error("404 must exit 2")
	}

	if ExitCode(&settings.HttpStatusError{Status: 401, URL: ""}) != 3 {
		t.Error("401 must exit 3")
	}
	// Wrapped errors still resolve through errors.As.
	if ExitCode(fmt.Errorf("wrap: %w", &settings.HttpStatusError{Status: 404, URL: ""})) != 2 {
		t.Error("wrapped 404 must exit 2")
	}
}

func TestOpenOutputWriterPrecedence(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	cmd := &Command{ //nolint:exhaustruct // intentional zero/partial fields
		Output:       "/tmp/should-not-be-created-by-openoutput-test.pdf",
		OutputWriter: &buf,
	}

	writer, closeW, err := cmd.OpenOutput()
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = closeW() }()

	if writer != &buf {
		t.Fatal("OutputWriter must win over Output path")
	}

	if _, err := writer.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}

	if buf.String() != "x" {
		t.Fatalf("buf=%q", buf.String())
	}
}

func TestRestrictNetworkAndAllowHostFlags(t *testing.T) { //nolint:cyclop // table of flag combinations
	t.Parallel()

	cmd := parse(t, "--restrict-network", "page.html", outPDF)
	if !cmd.Global.Load.NetworkPolicySet {
		t.Fatal("--restrict-network must set NetworkPolicySet")
	}

	if !cmd.Global.Load.NetworkBlockPrivate || !cmd.Global.Load.NetworkBlockCrossHost {
		t.Fatal("--restrict-network must set BlockPrivate and BlockCrossHost")
	}

	if got := cmd.Global.Load.NetworkAllowedSchemes; len(got) != 2 ||
		got[0] != "http" || got[1] != "https" {
		t.Fatalf("AllowedSchemes = %v, want [http https]", got)
	}

	cmd = parse(t, "--allow-host", "reports.example.test", "page.html", outPDF)
	if !cmd.Global.Load.NetworkPolicySet {
		t.Fatal("--allow-host must set NetworkPolicySet")
	}

	if got := cmd.Global.Load.NetworkAllowedHosts; len(got) != 1 || got[0] != "reports.example.test" {
		t.Fatalf("AllowedHosts = %v", got)
	}

	cmd = parse(t,
		"--restrict-network",
		"--allow-host", "a.example.test",
		"--allow-host", "b.example.test",
		"page.html", outPDF,
	)
	if !cmd.Global.Load.NetworkBlockPrivate || len(cmd.Global.Load.NetworkAllowedHosts) != 2 {
		t.Fatalf("combined flags: %+v", cmd.Global.Load)
	}

	if _, ok := flagTable["restrict-network"]; !ok {
		t.Fatal("restrict-network missing from flag table")
	}

	if spec, ok := flagTable["allow-host"]; !ok || spec.mod != ModeBoth {
		t.Fatal("allow-host must be registered on ModeBoth")
	}
}

func TestCLIVersionMatchesVERSIONFile(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../../VERSION")
	if err != nil {
		t.Fatalf("read VERSION: %v", err)
	}

	want := strings.TrimSpace(string(raw))
	if Version != want && !strings.HasPrefix(Version, want+"-") {
		t.Fatalf("cli.Version = %q, want VERSION %q or a suffix-stamped form", Version, want)
	}
}
