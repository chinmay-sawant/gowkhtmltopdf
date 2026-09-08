//nolint:testpackage // tests exercise the finalization and write seams.
package imageout

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

type reviewShortWriter struct {
	bytes.Buffer
	limit int
}

func (w *reviewShortWriter) Write(data []byte) (int, error) {
	remaining := w.limit - w.Len()
	if remaining <= 0 {
		return 0, nil
	}

	if len(data) > remaining {
		data = data[:remaining]
	}

	n, err := w.Buffer.Write(data)
	if err != nil {
		return n, fmt.Errorf("write test buffer: %w", err)
	}

	return n, nil
}

func TestWriteEncodedOutputRejectsSilentShortWrites(t *testing.T) {
	t.Parallel()

	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, G: 0, B: 0, A: 255})

	for _, format := range []string{formatPNG, formatJPG} {
		t.Run(format, func(t *testing.T) {
			t.Parallel()

			out := &reviewShortWriter{Buffer: bytes.Buffer{}, limit: 1}
			req := &Request{
				Global:  settings.DefaultPdfGlobal(),
				Image:   settings.ImageGlobal{Format: format, Quality: 80}, //nolint:exhaustruct // focused output settings
				Objects: []settings.PdfObject{},
				Now:     nil,
				Output:  out,
			}

			err := writeEncodedOutput(t.Context(), req, img, io.Discard)
			if !errors.Is(err, io.ErrShortWrite) {
				t.Fatalf("writeEncodedOutput error = %v, want io.ErrShortWrite", err)
			}
		})
	}
}

func TestImagePipelineFinalizeChecksCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	out := &bytes.Buffer{}
	pipeline := &imagePipeline{
		req: &Request{
			Global:  settings.DefaultPdfGlobal(),
			Image:   settings.ImageGlobal{Format: formatPNG}, //nolint:exhaustruct // focused output settings
			Objects: []settings.PdfObject{},
			Now:     nil,
			Output:  out,
		},
		obj:      nil,
		loader:   nil,
		font:     nil,
		registry: nil,
		log:      io.Discard,
		img:      image.NewNRGBA(image.Rect(0, 0, 2, 2)),
	}

	err := pipeline.Finalize(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Finalize error = %v, want context.Canceled", err)
	}

	if out.Len() != 0 {
		t.Fatalf("Finalize wrote %d bytes after cancellation, want 0", out.Len())
	}
}
