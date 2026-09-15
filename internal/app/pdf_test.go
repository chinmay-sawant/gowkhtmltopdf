//nolint:all
package app_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/app"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/cli"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/errs"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/load"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

func TestBuildPDFRequestPreservesEngineContract(t *testing.T) {
	t.Parallel()

	cmd := &cli.Command{
		Global:  settings.DefaultPdfGlobal(),
		Objects: []settings.PdfObject{{Page: "inline:<html></html>"}},
	}

	var out, outline bytes.Buffer

	cmd.Global.DumpOutline = true

	req, err := app.BuildPDFRequest(cmd, &out, &outline)
	if err != nil {
		t.Fatalf("BuildPDFRequest: %v", err)
	}

	if req.Output != &out || req.OutlineOutput != &outline {
		t.Fatal("request did not retain explicit output sinks")
	}

	if !req.Global.DumpOutline {
		t.Fatal("dump outline flag was not retained on global settings")
	}
}

func TestBuildPDFRequestRejectsMissingOutput(t *testing.T) {
	t.Parallel()

	cmd := &cli.Command{Global: settings.DefaultPdfGlobal()}

	_, err := app.BuildPDFRequest(cmd, nil, nil)
	if err == nil {
		t.Fatal("expected missing output error")
	}

	if !errors.Is(err, convert.ErrMissingOutput) {
		t.Fatalf("error = %v, want %v", err, convert.ErrMissingOutput)
	}
}

func TestRunPDFValidatesBeforeOpeningOutput(t *testing.T) {
	t.Parallel()

	output := filepath.Join(t.TempDir(), "out.pdf")
	cmd := &cli.Command{
		Global: settings.DefaultPdfGlobal(),
		Output: output,
	}

	err := app.RunPDF(t.Context(), cmd, nil, nil, nil)
	if !errors.Is(err, convert.ErrNoRenderableObjects) {
		t.Fatalf("RunPDF() = %v, want errors.Is(..., %v)", err, convert.ErrNoRenderableObjects)
	}

	if _, statErr := os.Stat(output); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("output stat error = %v, want os.ErrNotExist", statErr)
	}
}

func TestNilCommandUsesCanonicalSentinel(t *testing.T) {
	t.Parallel()

	if _, err := app.BuildPDFRequest(nil, nil, nil); !errors.Is(err, app.ErrNilCommand) {
		t.Fatalf("BuildPDFRequest(nil) = %v, want errors.Is(..., %v)", err, app.ErrNilCommand)
	}

	if err := app.RunPDF(t.Context(), nil, nil, nil, nil); !errors.Is(err, app.ErrNilCommand) {
		t.Fatalf("RunPDF(nil command) = %v, want errors.Is(..., %v)", err, app.ErrNilCommand)
	}
	if app.ErrNilCommand != errs.ErrNilCommand {
		t.Fatal("app and errs nil-command sentinels must be identical")
	}
}

//nolint:wsl // assertions intentionally follow the side-effect checks.
func TestRunPDFRejectsOutlineAndPDFOnStdout(t *testing.T) {
	t.Parallel()

	global := settings.DefaultPdfGlobal()
	global.DumpOutline = true
	cmd := &cli.Command{
		Global: global,
		Objects: []settings.PdfObject{{
			Page: "inline:<html><body>stdout conflict</body></html>",
		}},
		Output: "-",
	}

	var outline bytes.Buffer
	err := app.RunPDF(t.Context(), cmd, nil, nil, &outline)
	if !errors.Is(err, app.ErrConflictingOutputSinks) {
		t.Fatalf("RunPDF error = %v, want %v", err, app.ErrConflictingOutputSinks)
	}
	if outline.Len() != 0 {
		t.Fatalf("outline bytes = %d, want zero on rejected request", outline.Len())
	}
}

//nolint:wsl // assertions intentionally follow the side-effect checks.
func TestRunPDFKeepsPDFFileAndOutlineXMLSeparate(t *testing.T) {
	t.Parallel()

	global := settings.DefaultPdfGlobal()
	global.DumpOutline = true
	output := filepath.Join(t.TempDir(), "out.pdf")
	cmd := &cli.Command{
		Global: global,
		Objects: []settings.PdfObject{{
			Page: "inline:<html><body><h1>Separate outputs</h1></body></html>",
		}},
		Output: output,
	}

	var outline bytes.Buffer
	if err := app.RunPDF(t.Context(), cmd, nil, nil, &outline); err != nil {
		t.Fatalf("RunPDF: %v", err)
	}

	pdf, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", output, err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Fatalf("PDF prefix = %q, want %%PDF-", pdf[:min(len(pdf), 16)])
	}
	if !bytes.HasPrefix(outline.Bytes(), []byte("<?xml")) || !bytes.Contains(outline.Bytes(), []byte("<outline")) {
		t.Fatalf("outline output = %q, want standalone XML outline", outline.String())
	}
}

func TestRunPDFRestrictNetworkBlocksLoopback(t *testing.T) {
	t.Parallel()

	cmd, err := cli.Parse([]string{
		"--restrict-network",
		"--quiet",
		"http://127.0.0.1/",
		"--output", filepath.Join(t.TempDir(), "out.pdf"),
	}, cli.ModePDF)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if !cmd.Global.Load.NetworkPolicySet || !cmd.Global.Load.NetworkBlockPrivate {
		t.Fatalf("parsed restrict-network fields = %+v", cmd.Global.Load)
	}

	runErr := app.RunPDF(t.Context(), cmd, nil, nil, nil)
	if !errors.Is(runErr, load.ErrNetworkPolicy) {
		t.Fatalf("RunPDF = %v, want errors.Is(..., load.ErrNetworkPolicy)", runErr)
	}
}

// failedConversionCommand builds a command that passes request validation and
// fails during conversion (the loopback fetch is rejected by the network
// policy), which is exactly the window where the old adapter had already
// truncated the output path.
func failedConversionCommand(t *testing.T, output string) *cli.Command {
	t.Helper()

	cmd, err := cli.Parse([]string{
		"--restrict-network",
		"--quiet",
		"http://127.0.0.1/",
		"--output", output,
	}, cli.ModePDF)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	return cmd
}

func TestRunPDFFailurePreservesExistingOutput(t *testing.T) {
	t.Parallel()

	output := filepath.Join(t.TempDir(), "out.pdf")
	previous := []byte("previous artifact bytes that a failed conversion must not clobber")

	if err := os.WriteFile(output, previous, 0o600); err != nil {
		t.Fatalf("seed output: %v", err)
	}

	runErr := app.RunPDF(t.Context(), failedConversionCommand(t, output), nil, nil, nil)
	if !errors.Is(runErr, load.ErrNetworkPolicy) {
		t.Fatalf("RunPDF = %v, want errors.Is(..., load.ErrNetworkPolicy)", runErr)
	}

	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", output, err)
	}

	if !bytes.Equal(got, previous) {
		t.Fatalf("a failed conversion clobbered the output: got %d bytes %q, want %d bytes",
			len(got), got, len(previous))
	}
}

func TestRunPDFFailureDoesNotCreateOutput(t *testing.T) {
	t.Parallel()

	output := filepath.Join(t.TempDir(), "out.pdf")

	runErr := app.RunPDF(t.Context(), failedConversionCommand(t, output), nil, nil, nil)
	if !errors.Is(runErr, load.ErrNetworkPolicy) {
		t.Fatalf("RunPDF = %v, want errors.Is(..., load.ErrNetworkPolicy)", runErr)
	}

	if _, statErr := os.Stat(output); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("failed conversion left an artifact at %q (stat error = %v)", output, statErr)
	}
}

// TestRunPDFRejectsBadOutputPathBeforeConversion keeps the early destination
// check: a missing parent directory fails before the render starts even though
// the file itself is no longer opened up front.
func TestRunPDFRejectsBadOutputPathBeforeConversion(t *testing.T) {
	t.Parallel()

	cmd := &cli.Command{
		Global: settings.DefaultPdfGlobal(),
		Objects: []settings.PdfObject{{
			Page: "inline:<html><body>output preflight</body></html>",
		}},
		Output: filepath.Join(t.TempDir(), "missing-dir", "out.pdf"),
	}

	err := app.RunPDF(t.Context(), cmd, nil, nil, nil)
	if err == nil || !strings.HasPrefix(err.Error(), "app: open output:") {
		t.Fatalf("RunPDF() = %v, want app: open output error", err)
	}
}

func TestDefaultTOCXSLDelegatesToConvert(t *testing.T) {
	t.Parallel()

	got := app.DefaultTOCXSL()
	want := convert.DefaultTOCXSL()

	if got != want {
		t.Fatalf("DefaultTOCXSL mismatch")
	}

	if !strings.Contains(got, "<xsl:stylesheet") {
		t.Fatalf("DefaultTOCXSL = %q, want stylesheet", got)
	}
}
