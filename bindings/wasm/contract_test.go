package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"strconv"
	"strings"
	"testing"
)

func TestDecodeRequestDefaultsToPDF(t *testing.T) {
	request, err := DecodeRequest(`{"html":"<h1>Hello</h1>"}`)
	if err != nil {
		t.Fatalf("DecodeRequest() error = %v", err)
	}

	if request.Mode != "pdf" {
		t.Fatalf("mode = %q, want pdf", request.Mode)
	}
}

func TestDecodeRequestSupportsImageModes(t *testing.T) {
	for _, mode := range []string{"png", "jpeg"} {
		t.Run(mode, func(t *testing.T) {
			request, err := DecodeRequest(`{"html":"<h1>Hello</h1>","mode":"` + mode + `"}`)
			if err != nil {
				t.Fatalf("DecodeRequest() error = %v", err)
			}
			if request.Mode != mode {
				t.Fatalf("mode = %q, want %q", request.Mode, mode)
			}
		})
	}
}

func TestDecodeRequestRejectsInvalidBoundary(t *testing.T) {
	largeHTML := strings.Repeat("x", maxHTMLBytes+1)
	cases := []struct {
		name string
		raw  string
		want error
	}{
		{name: "empty", raw: `{"html":""}`, want: errInvalidRequest},
		{name: "unknown field", raw: `{"html":"x","file":"local.html"}`, want: errInvalidRequest},
		{name: "URL source", raw: `{"html":"x","url":"https://example.test"}`, want: errInvalidRequest},
		{name: "unsupported mode", raw: `{"html":"x","mode":"webp"}`, want: errUnsupportedMode},
		{name: "large html", raw: mustJSON(Request{HTML: largeHTML}), want: errInputTooLarge},
		{name: "pdf image option", raw: `{"html":"x","mode":"pdf","width":10}`, want: errInvalidRequest},
		{name: "large image", raw: `{"html":"x","mode":"png","width":4097}`, want: errImageTooLarge},
		{name: "bad quality", raw: `{"html":"x","mode":"jpeg","quality":101}`, want: errInvalidRequest},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			_, err := DecodeRequest(test.raw)
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want errors.Is(..., %v)", err, test.want)
			}
		})
	}
}

func TestConvertProducesPDFPNGAndJPEG(t *testing.T) {
	for _, mode := range []string{"pdf", "png", "jpeg"} {
		t.Run(mode, func(t *testing.T) {
			request := Request{HTML: `<html><body><h1>WASM sample</h1><p>Inline HTML only.</p></body></html>`, Mode: mode}
			var progress []string
			result, err := Convert(context.Background(), request, func(phase string, percent int) {
				progress = append(progress, phase+":"+strconv.Itoa(percent))
			})
			if err != nil {
				t.Fatalf("Convert() error = %v", err)
			}
			if len(result.Bytes) == 0 || result.MIME == "" {
				t.Fatalf("empty result: %#v", result)
			}
			if len(progress) == 0 {
				t.Fatal("conversion emitted no progress")
			}

			switch mode {
			case "pdf":
				if !bytes.HasPrefix(result.Bytes, []byte("%PDF-")) || !bytes.Contains(result.Bytes, []byte("%%EOF")) {
					t.Fatal("result is not a complete PDF")
				}
				if result.MIME != "application/pdf" {
					t.Fatalf("MIME = %q, want application/pdf", result.MIME)
				}
			case "png", "jpeg":
				config, format, err := image.DecodeConfig(bytes.NewReader(result.Bytes))
				if err != nil {
					t.Fatalf("DecodeConfig() error = %v", err)
				}
				if format != mode && !(mode == "jpeg" && format == "jpeg") {
					t.Fatalf("format = %q, want %q", format, mode)
				}
				if config.Width != result.Width || config.Height != result.Height {
					t.Fatalf("dimensions = %dx%d, result = %dx%d", config.Width, config.Height, result.Width, result.Height)
				}
			}
		})
	}
}

func TestBrowserPNGConversionPreservesTransparentCanvas(t *testing.T) {
	result, err := Convert(t.Context(), Request{
		HTML:   `<html><body style="margin:0"><span style="color:#176b3a">WASM</span></body></html>`,
		Mode:   "png",
		Width:  128,
		Height: 64,
	}, nil)
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}

	decoded, _, err := image.Decode(bytes.NewReader(result.Bytes))
	if err != nil {
		t.Fatalf("image.Decode() error = %v", err)
	}

	_, _, _, alpha := decoded.At(result.Width-1, result.Height-1).RGBA()
	if alpha != 0 {
		t.Fatalf("blank PNG corner alpha = %d, want 0", alpha)
	}
}

func TestErrorResponseUsesStableCodes(t *testing.T) {
	if got := errorResponse(errInputTooLarge); got.Code != "resource_limit" {
		t.Fatalf("code = %q, want resource_limit", got.Code)
	}
	if got := errorResponse(errOutputTooLarge); got.Code != "resource_limit" {
		t.Fatalf("output limit code = %q, want resource_limit", got.Code)
	}
	if got := errorResponse(errImageTooLarge); got.Code != "resource_limit" {
		t.Fatalf("image limit code = %q, want resource_limit", got.Code)
	}
	if got := errorResponse(errUnsupportedMode); got.Code != "invalid_request" {
		t.Fatalf("code = %q, want invalid_request", got.Code)
	}
	if got := errorResponse(context.DeadlineExceeded); got.Code != "timeout" {
		t.Fatalf("code = %q, want timeout", got.Code)
	}
}

func TestConvertReportsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Convert(ctx, Request{HTML: "<h1>cancelled</h1>", Mode: "pdf"}, nil)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("Convert() error = %v, want context cancellation", err)
	}
}

func TestBrowserDocumentsUseInlineOnlyDefaults(t *testing.T) {
	request := Request{HTML: "<h1>sample</h1>", Mode: "pdf"}
	pdf := browserPDFDocument(request, nil)
	if pdf.AllowLocalFiles || pdf.UseSystemFonts || len(pdf.FontPaths) != 0 {
		t.Fatalf("PDF browser defaults allow native resources: %#v", pdf)
	}
	if pdf.Network == nil || len(pdf.Network.AllowedSchemes) != 0 {
		t.Fatalf("PDF browser network policy = %#v, want no allowed schemes", pdf.Network)
	}

	request.Mode = "png"
	imageDocument := browserImageDocument(request, nil)
	if imageDocument.AllowLocalFiles || imageDocument.UseSystemFonts || len(imageDocument.FontPaths) != 0 {
		t.Fatalf("image browser defaults allow native resources: %#v", imageDocument)
	}
	if imageDocument.Network == nil || len(imageDocument.Network.AllowedSchemes) != 0 {
		t.Fatalf("image browser network policy = %#v, want no allowed schemes", imageDocument.Network)
	}
}

func TestProgressHooksPreservePhaseAndOrder(t *testing.T) {
	var events []string
	onPhase, onValue := progressHooks(func(phase string, percent int) {
		events = append(events, phase+":"+strconv.Itoa(percent))
	})
	onPhase("Loading")
	onValue(25)
	onPhase("Rendering")
	onValue(100)

	want := []string{"Loading:0", "Loading:25", "Rendering:0", "Rendering:100"}
	if len(events) != len(want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
	for index := range want {
		if events[index] != want[index] {
			t.Fatalf("event %d = %q, want %q", index, events[index], want[index])
		}
	}
}

func mustJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}

	return string(data)
}
