package main

import (
	"bytes"
	"encoding/json"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

type fixtureManifest struct {
	Fixture string                   `json:"fixture"`
	Outputs map[string]fixtureOutput `json:"outputs"`
}

type fixtureOutput struct {
	MIME        string   `json:"mime"`
	PageMin     int      `json:"pageMin"`
	PageMax     int      `json:"pageMax"`
	Width       int      `json:"width"`
	Height      int      `json:"height"`
	MinBytes    int      `json:"minBytes"`
	TextNeedles []string `json:"textNeedles"`
}

func TestWASMFixtureManifest(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "testdata", "wasm")
	manifestData, err := os.ReadFile(filepath.Join(fixtureDir, "manifest.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	var manifest fixtureManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	html, err := os.ReadFile(filepath.Join(fixtureDir, manifest.Fixture))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	for mode, expectation := range manifest.Outputs {
		t.Run(mode, func(t *testing.T) {
			request := Request{HTML: string(html), Mode: mode}
			if mode != "pdf" {
				request.Width = expectation.Width
				request.Height = expectation.Height
			}
			result, err := Convert(t.Context(), request, nil)
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}
			if result.MIME != expectation.MIME {
				t.Fatalf("MIME = %q, want %q", result.MIME, expectation.MIME)
			}
			if len(result.Bytes) < expectation.MinBytes {
				t.Fatalf("output bytes = %d, want at least %d", len(result.Bytes), expectation.MinBytes)
			}

			if mode == "pdf" {
				assertFixturePDF(t, result.Bytes, expectation)
				return
			}

			assertFixtureImage(t, result.Bytes, result.Width, result.Height, mode, expectation)
		})
	}
}

func assertFixturePDF(t *testing.T, output []byte, expectation fixtureOutput) {
	t.Helper()

	semantic, err := pdf.ParseSemantic(output)
	if err != nil {
		t.Fatalf("ParseSemantic() error = %v", err)
	}
	if semantic.PageCount() < expectation.PageMin || semantic.PageCount() > expectation.PageMax {
		t.Fatalf("page count = %d, want %d..%d", semantic.PageCount(), expectation.PageMin, expectation.PageMax)
	}
	text := semantic.DocumentText()
	for _, needle := range expectation.TextNeedles {
		if !strings.Contains(text, needle) {
			t.Fatalf("PDF text missing %q in %q", needle, text)
		}
	}
}

func assertFixtureImage(t *testing.T, output []byte, width, height int, mode string, expectation fixtureOutput) {
	t.Helper()

	config, format, err := image.DecodeConfig(bytes.NewReader(output))
	if err != nil {
		t.Fatalf("DecodeConfig() error = %v", err)
	}
	if format != mode {
		t.Fatalf("format = %q, want %q", format, mode)
	}
	if config.Width != expectation.Width || config.Height != expectation.Height {
		t.Fatalf("decoded dimensions = %dx%d, want %dx%d", config.Width, config.Height, expectation.Width, expectation.Height)
	}
	if width != expectation.Width || height != expectation.Height {
		t.Fatalf("result dimensions = %dx%d, want %dx%d", width, height, expectation.Width, expectation.Height)
	}
}
