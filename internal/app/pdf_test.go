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

	cmd := &cli.Command{ //nolint:exhaustruct // intentional zero/partial fields
		Global:  settings.DefaultPdfGlobal(),
		Objects: []settings.PdfObject{{Page: "inline:<html></html>"}}, //nolint:exhaustruct // intentional zero/partial fields
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

	cmd := &cli.Command{Global: settings.DefaultPdfGlobal()} //nolint:exhaustruct // intentional zero/partial fields

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
	cmd := &cli.Command{ //nolint:exhaustruct // focused invalid command
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

	if _, err := app.BuildPDFRequest(nil, nil, nil); !errors.Is(err, errs.ErrNilCommand) {
		t.Fatalf("BuildPDFRequest(nil) = %v, want errors.Is(..., %v)", err, errs.ErrNilCommand)
	}

	if err := app.RunPDF(t.Context(), nil, nil, nil, nil); !errors.Is(err, errs.ErrNilCommand) {
		t.Fatalf("RunPDF(nil command) = %v, want errors.Is(..., %v)", err, errs.ErrNilCommand)
	}
}

//nolint:wsl // assertions intentionally follow the side-effect checks.
func TestRunPDFRejectsOutlineAndPDFOnStdout(t *testing.T) {
	t.Parallel()

	global := settings.DefaultPdfGlobal()
	global.DumpOutline = true
	cmd := &cli.Command{ //nolint:exhaustruct // intentional zero/partial fields
		Global: global,
		Objects: []settings.PdfObject{{ //nolint:exhaustruct // intentional zero/partial fields
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
	cmd := &cli.Command{ //nolint:exhaustruct // intentional zero/partial fields
		Global: global,
		Objects: []settings.PdfObject{{ //nolint:exhaustruct // intentional zero/partial fields
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
		filepath.Join(t.TempDir(), "out.pdf"),
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
