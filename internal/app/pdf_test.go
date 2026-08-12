package app_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"gowkhtmltopdf/internal/app"
	"gowkhtmltopdf/internal/cli"
	"gowkhtmltopdf/internal/convert"
	"gowkhtmltopdf/internal/settings"
)

func TestBuildPDFRequestPreservesEngineContract(t *testing.T) {
	t.Parallel()

	cmd := &cli.Command{ //nolint:exhaustruct // intentional zero/partial fields
		Global:  settings.DefaultPdfGlobal(),
		Objects: []settings.PdfObject{{Page: "inline:<html></html>"}}, //nolint:exhaustruct // intentional zero/partial fields
	}

	var out, outline bytes.Buffer

	cmd.DumpOutline = true

	req, err := app.BuildPDFRequest(cmd, &out, &outline)
	if err != nil {
		t.Fatalf("BuildPDFRequest: %v", err)
	}

	if req.Output != &out || req.OutlineOutput != &outline {
		t.Fatal("request did not retain explicit output sinks")
	}

	if !req.Global.DumpOutline {
		t.Fatal("legacy command dump flag was not projected to global settings")
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
