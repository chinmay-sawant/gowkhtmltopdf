package app_test

import (
	"bytes"
	"errors"
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
