//nolint:cyclop,exhaustruct,wsl,lll,err113,testpackage // parser table tests intentionally cover private grammar state.
package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

const (
	outPDF = "out.pdf"
	outPNG = "out.png"
)

func parsePDF(t *testing.T, args ...string) *Command {
	t.Helper()

	cmd, err := Parse(args, ModePDF)
	if err != nil {
		t.Fatalf("Parse PDF %v: %v", args, err)
	}

	return cmd
}

func parseImage(t *testing.T, args ...string) *Command {
	t.Helper()

	cmd, err := Parse(args, ModeImage)
	if err != nil {
		t.Fatalf("Parse image %v: %v", args, err)
	}

	return cmd
}

const testReportTitle = "Report"

func TestDocumentGrammarBuildsPagesFromPositionalFiles(t *testing.T) {
	t.Parallel()

	cmd := parsePDF(t,
		"--page-size", "Letter",
		"--orientation", "Landscape",
		"--margin-top", "20mm",
		"--title", "Report",
		"--outline-depth", "2",
		"--allow-local-files",
		"-o", outPDF,
		"page-1.html", "page-2.html",
	)

	if cmd.Output != outPDF || len(cmd.Objects) != 2 {
		t.Fatalf("output=%q objects=%d, want %q and two pages", cmd.Output, len(cmd.Objects), outPDF)
	}
	if cmd.Objects[0].Page != "page-1.html" || cmd.Objects[1].Page != "page-2.html" {
		t.Fatalf("pages = %+v", cmd.Objects)
	}
	if cmd.Global.PageSize != "Letter" || cmd.Global.Orientation != settings.OrientationLandscape {
		t.Fatalf("geometry = %+v", cmd.Global)
	}
	if cmd.Global.Margin.Top != 20 || cmd.Global.Title != testReportTitle || cmd.Global.OutlineDepth != 2 {
		t.Fatalf("global settings = %+v", cmd.Global)
	}
	if !cmd.Global.Load.EnableLocalFileAccess || cmd.Objects[0].Load.BlockLocalFileAccess {
		t.Fatalf("local-file policy = global=%v object-block=%v", cmd.Global.Load.EnableLocalFileAccess, cmd.Objects[0].Load.BlockLocalFileAccess)
	}
}

func TestExplicitSources(t *testing.T) {
	t.Parallel()

	inline := `<html><body><h1>Inline</h1></body></html>`
	cmd := parsePDF(t, "--html", inline, "--output", outPDF)
	if len(cmd.Objects) != 1 || string(cmd.Objects[0].Load.InlineHTML) != inline {
		t.Fatalf("html source = %+v", cmd.Objects)
	}

	cmd = parsePDF(t, "--url", "https://example.test/report", "-o", outPDF)
	if len(cmd.Objects) != 1 || cmd.Objects[0].Page != "https://example.test/report" {
		t.Fatalf("url source = %+v", cmd.Objects)
	}
}

func TestCoverTOCAndBodyOrdering(t *testing.T) {
	t.Parallel()

	cmd := parsePDF(t,
		"-o", outPDF,
		"--toc",
		"--cover", "cover.html",
		"--header-left", testReportTitle,
		"chapter-1.html", "chapter-2.html",
	)
	if len(cmd.Objects) != 4 {
		t.Fatalf("objects = %d, want cover, toc, and two pages: %+v", len(cmd.Objects), cmd.Objects)
	}
	cover := cmd.Objects[0]
	if !cover.IsCover || cover.Page != "cover.html" {
		t.Fatalf("cover = %+v", cover)
	}
	if cover.IncludeInOutline {
		t.Fatal("cover must be excluded from outline by default")
	}
	if !cover.HeaderSet || !cover.FooterSet {
		t.Fatalf("cover HF override bits = header:%v footer:%v", cover.HeaderSet, cover.FooterSet)
	}
	if cover.Header.Left != "" || cover.Header.Center != "" || cover.Header.Right != "" ||
		cover.Footer.Left != "" || cover.Footer.Center != "" || cover.Footer.Right != "" ||
		len(cover.Header.Replace) != 0 || len(cover.Footer.Replace) != 0 {
		t.Fatalf("cover must stamp empty HF (no global inherit): header=%+v footer=%+v", cover.Header, cover.Footer)
	}
	toc := cmd.Objects[1]
	if !toc.IsTableOfContent || toc.UseOutline || toc.IncludeInOutline {
		t.Fatalf("toc defaults = %+v", toc)
	}
	if cmd.Objects[2].Page != "chapter-1.html" || cmd.Objects[3].Page != "chapter-2.html" {
		t.Fatalf("body pages = %+v", cmd.Objects[2:])
	}
	if cmd.Global.Header.Left != testReportTitle {
		t.Fatalf("global header should remain set for body inherit: %+v", cmd.Global.Header)
	}
}

func TestSourceConflictsAndOutputRequirement(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want error
	}{
		{name: "missing output", args: []string{"page.html"}, want: ErrMissingOutput},
		{name: "html and url", args: []string{"--html", "<p>x</p>", "--url", "https://example.test", "-o", outPDF}, want: ErrConflictingInputs},
		{name: "html and files", args: []string{"--html", "<p>x</p>", "page.html", "-o", outPDF}, want: ErrConflictingInputs},
		{name: "url and files", args: []string{"--url", "https://example.test", "page.html", "-o", outPDF}, want: ErrConflictingInputs},
		{name: "duplicate output", args: []string{"-o", "one.pdf", "--output", "two.pdf", "page.html"}, want: ErrDuplicateOutput},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := Parse(testCase.args, ModePDF)
			if !errors.Is(err, testCase.want) {
				t.Fatalf("Parse(%v) = %v, want errors.Is(..., %v)", testCase.args, err, testCase.want)
			}
		})
	}
}

func TestRemovedObjectGrammarAndSetEscapeHatch(t *testing.T) {
	t.Parallel()

	for _, token := range []string{"page", "cover", "toc"} {
		if _, err := Parse([]string{"-o", outPDF, token, "input.html"}, ModePDF); !errors.Is(err, ErrLegacyObjectSyntax) {
			t.Errorf("token %q error = %v, want legacy syntax error", token, err)
		}
	}
	if _, err := Parse([]string{"--set", "title=bad", "-o", outPDF, "input.html"}, ModePDF); err == nil {
		t.Error("--set must not be accepted by the new CLI")
	}
}

func TestGoFriendlyGlobalFlags(t *testing.T) {
	t.Parallel()

	cmd := parsePDF(t,
		"--pdf-version", "1.7",
		"--pdf-profile", "a3a",
		"--header-left", testReportTitle,
		"--footer-center", "[page]/[topage]",
		"--font-path", "/fonts/one",
		"--font-path", "/fonts/two",
		"--no-outline",
		"--grayscale",
		"-q",
		"-o", outPDF,
		"input.html",
	)

	if cmd.Global.PdfVersion != "1.7" || cmd.Global.PdfProfile != settings.ProfilePDFA3a {
		t.Fatalf("pdf settings = %+v", cmd.Global)
	}
	if cmd.Global.Header.Left != testReportTitle || cmd.Global.Footer.Center != "[page]/[topage]" {
		t.Fatalf("header/footer = %+v / %+v", cmd.Global.Header, cmd.Global.Footer)
	}
	if len(cmd.Global.FontPaths) != 2 || cmd.Global.Outline || !cmd.Global.Grayscale || !cmd.Global.Quiet {
		t.Fatalf("global flags = %+v", cmd.Global)
	}
}

func TestImageGrammarAndOptions(t *testing.T) {
	t.Parallel()

	cmd := parseImage(t,
		"--html", "<html><body>image</body></html>",
		"--width", "800", "--height", "600", "--quality", "80",
		"--format", "png", "--no-smart-width", "-o", outPNG,
	)
	if cmd.Output != outPNG || len(cmd.Objects) != 1 || len(cmd.Objects[0].Load.InlineHTML) == 0 {
		t.Fatalf("image source/output = %+v / %q", cmd.Objects, cmd.Output)
	}
	if cmd.Image.Width != 800 || cmd.Image.Height != 600 || cmd.Image.Quality != 80 || cmd.Image.Format != "png" || cmd.Image.SmartWidth {
		t.Fatalf("image settings = %+v", cmd.Image)
	}

	if _, err := Parse([]string{"--cover", "cover.html", "-o", outPNG, "input.html"}, ModeImage); err == nil {
		t.Error("--cover must not be accepted in image mode")
	}
}

func TestBooleanAndShortFlagSyntax(t *testing.T) {
	t.Parallel()

	cmd := parsePDF(t, "--outline=false", "--outline", "-s", "A5", "-O", "Landscape", "-o", outPDF, "input.html")
	if !cmd.Global.Outline || cmd.Global.PageSize != "A5" || cmd.Global.Orientation != settings.OrientationLandscape {
		t.Fatalf("boolean/short flags = %+v", cmd.Global)
	}
	if _, err := Parse([]string{"--outline=maybe", "-o", outPDF, "input.html"}, ModePDF); !errors.Is(err, errInvalidBoolValue) {
		t.Fatalf("invalid bool error = %v", err)
	}
}

func TestHelpUsesDocumentGrammar(t *testing.T) {
	t.Parallel()

	for _, mode := range []Mode{ModePDF, ModeImage} {
		var help bytes.Buffer
		PrintHelp(&help, mode)
		got := help.String()
		for _, want := range []string{"-o, --output", "--html", "--url"} {
			if !strings.Contains(got, want) {
				t.Errorf("mode %v help missing %q:\n%s", mode, want, got)
			}
		}
		if mode == ModePDF {
			for _, want := range []string{"--cover", "--toc"} {
				if !strings.Contains(got, want) {
					t.Errorf("PDF help missing %q:\n%s", want, got)
				}
			}
		}
		for _, forbidden := range []string{"OBJECT is one of", "last positional argument is the output", "--set"} {
			if strings.Contains(got, forbidden) {
				t.Errorf("help contains stale grammar %q:\n%s", forbidden, got)
			}
		}
	}
}

func TestTerminalActionsAndModeValidation(t *testing.T) {
	t.Parallel()

	cmd, err := Parse([]string{"--dump-default-toc-xsl"}, ModePDF)
	if err != nil || !cmd.Global.DumpDefaultTOCXSL {
		t.Fatalf("terminal action = %#v, %v", cmd, err)
	}
	if _, err := Parse([]string{"--dump-default-toc-xsl", "-o", outPDF, "input.html"}, ModePDF); !errors.Is(err, ErrTerminalConflict) {
		t.Fatalf("terminal conflict = %v", err)
	}
	if _, err := Parse([]string{"--dump-default-toc-xsl", "--dump-outline"}, ModePDF); !errors.Is(err, ErrTerminalConflict) {
		t.Fatalf("dump-outline conflict = %v", err)
	}
	if _, err := Parse([]string{"--dump-default-toc-xsl"}, ModeImage); err == nil {
		t.Fatal("PDF terminal action accepted in image mode")
	}
}

func TestExitCodeAndOutputWriter(t *testing.T) {
	t.Parallel()

	errBoom := errors.New("boom")
	if ExitCode(nil) != ExitError || ExitCode(errBoom) != ExitError {
		t.Fatal("plain errors must use exit code 1")
	}
	if ExitCode(&settings.HttpStatusError{Status: 404}) != 2 || ExitCode(&settings.HttpStatusError{Status: 401}) != 3 {
		t.Fatal("HTTP status exit code mapping changed")
	}

	var buf bytes.Buffer
	cmd := &Command{Output: "/tmp/unused.pdf", OutputWriter: &buf} //nolint:exhaustruct // sink precedence only
	writer, closeWriter, err := cmd.OpenOutput()
	if err != nil || writer != &buf {
		t.Fatalf("OpenOutput = %v, %v", writer, err)
	}
	if err := closeWriter(); err != nil {
		t.Fatal(err)
	}
}

func TestCLIVersionMatchesVERSIONFile(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile("../../VERSION")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSpace(string(raw))
	if Version != want && !strings.HasPrefix(Version, want+"-") {
		t.Fatalf("cli.Version = %q, want VERSION %q or stamped suffix", Version, want)
	}
}

func TestParseModeValidation(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]string{"-o", outPDF, "input.html"}, 0); err == nil {
		t.Fatal("zero mode accepted")
	}
	if _, err := Parse([]string{"-o", outPDF, "input.html"}, Mode(8)); err == nil {
		t.Fatal("unknown mode accepted")
	}
	if _, err := Parse([]string{"-o", outPDF, "input.html"}, ModePDF, ModeImage); err == nil {
		t.Fatal("multiple modes accepted")
	}
}

func TestInvalidPDFOptions(t *testing.T) {
	t.Parallel()

	if _, err := Parse([]string{"--pdf-version", "9.9", "-o", outPDF, "input.html"}, ModePDF); !errors.Is(err, settings.ErrInvalidPDFVersion) {
		t.Fatalf("invalid version = %v", err)
	}
	if _, err := Parse([]string{"--pdf-profile", "invalid", "-o", outPDF, "input.html"}, ModePDF); !errors.Is(err, settings.ErrInvalidPDFProfile) {
		t.Fatalf("invalid profile = %v", err)
	}
	if _, err := Parse([]string{"--dpi", "150", "-o", outPDF, "input.html"}, ModePDF); err == nil {
		t.Fatal("Policy A dpi flag accepted")
	}
}

func TestErrorFormattingKeepsOffendingFlag(t *testing.T) {
	t.Parallel()

	_, err := Parse([]string{"--unknown", "-o", outPDF, "input.html"}, ModePDF)
	if err == nil || !strings.Contains(fmt.Sprint(err), "--unknown") {
		t.Fatalf("unknown flag error = %v", err)
	}
}
