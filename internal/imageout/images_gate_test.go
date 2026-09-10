package imageout

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"io"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// TestRunRequestObjectWebImagesFalseDisablesFetch proves RenderObjects feeds
// settings.ResolveImages into the fetch gate: an object-level
// web.images=false keeps a data: image out of the output even though the
// global and image layers default to true.
func TestRunRequestObjectWebImagesFalseDisablesFetch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		objectWeb settings.Web
		wantRed   bool
	}{
		{name: "object enables", objectWeb: settings.Web{Images: true}, wantRed: true},
		{name: "object disables", objectWeb: settings.Web{Images: false}, wantRed: false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			raw := redPNG(t)
			page := `inline:<html><body><img src="data:image/png;base64,` +
				base64.StdEncoding.EncodeToString(raw) + `"></body></html>`

			global := settings.DefaultPdfGlobal()
			imageSettings := settings.DefaultImageGlobal()
			imageSettings.Width = 200

			object := settings.DefaultPdfObject()
			object.Page = page
			object.Web = testCase.objectWeb

			var out bytes.Buffer

			req := NewRequest(global, imageSettings, []settings.PdfObject{object}, &out)
			if err := RunRequest(t.Context(), req, io.Discard); err != nil {
				t.Fatalf("RunRequest: %v", err)
			}

			img, err := png.Decode(&out)
			if err != nil {
				t.Fatalf("decode png: %v", err)
			}

			want := color.NRGBA{R: 255, A: 255}
			count := countPixels(img, image.Rect(8, 8, 8+16, 8+16), want)

			if testCase.wantRed && count == 0 {
				t.Error("no red pixels in the <img> region; image should render")
			}

			if !testCase.wantRed && count != 0 {
				t.Errorf("found %d red pixels; object-level web.images=false must disable fetch", count)
			}
		})
	}
}

// TestRunRequestGlobalWebImagesGateControlsFetch proves RenderObjects feeds
// settings.ResolveImages into the fetch gate: the global web.images layer
// decides the gate while the image and object layers default to true.
func TestRunRequestGlobalWebImagesGateControlsFetch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		globalWeb settings.Web
		wantRed   bool
	}{
		{name: "global enables", globalWeb: settings.Web{Images: true}, wantRed: true},
		{name: "global disables", globalWeb: settings.Web{Images: false}, wantRed: false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			raw := redPNG(t)
			page := `inline:<html><body><img src="data:image/png;base64,` +
				base64.StdEncoding.EncodeToString(raw) + `"></body></html>`

			global := settings.DefaultPdfGlobal()
			global.Web = testCase.globalWeb

			imageSettings := settings.DefaultImageGlobal()
			imageSettings.Width = 200

			object := settings.DefaultPdfObject()
			object.Page = page

			var out bytes.Buffer

			req := NewRequest(global, imageSettings, []settings.PdfObject{object}, &out)
			if err := RunRequest(t.Context(), req, io.Discard); err != nil {
				t.Fatalf("RunRequest: %v", err)
			}

			img, err := png.Decode(&out)
			if err != nil {
				t.Fatalf("decode png: %v", err)
			}

			want := color.NRGBA{R: 255, A: 255}
			count := countPixels(img, image.Rect(8, 8, 8+16, 8+16), want)

			if testCase.wantRed && count == 0 {
				t.Error("no red pixels in the <img> region; image should render")
			}

			if !testCase.wantRed && count != 0 {
				t.Errorf("found %d red pixels; global-level web.images=false must disable fetch", count)
			}
		})
	}
}
