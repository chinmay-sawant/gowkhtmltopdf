package app

import (
	"bytes"
	"errors"
	"testing"

	"gowkhtmltopdf/internal/cli"
	"gowkhtmltopdf/internal/convert"
	"gowkhtmltopdf/internal/settings"
)

func TestBuildPDFRequestPreservesEngineContract(t *testing.T) {
	cmd := &cli.Command{
		Global:  settings.DefaultPdfGlobal(),
		Objects: []settings.PdfObject{{Page: "inline:<html></html>"}},
	}

	var out, outline bytes.Buffer

	cmd.DumpOutline = true

	req, err := BuildPDFRequest(cmd, &out, &outline)
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
	cmd := &cli.Command{Global: settings.DefaultPdfGlobal()}

	_, err := BuildPDFRequest(cmd, nil, nil)
	if err == nil {
		t.Fatal("expected missing output error")
	}

	if !errors.Is(err, convert.ErrMissingOutput) {
		t.Fatalf("error = %v, want %v", err, convert.ErrMissingOutput)
	}
}
