package imageout

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"io"
	"slices"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// Compile-time check that the encode scratch takes image/jpeg's direct-write
// path (Flush plus io.Writer) instead of the bufio fallback.
var _ interface {
	io.Writer
	Flush() error
} = (*limitedImageBuffer)(nil)

// oversizedEncodeImage reports a JPEG-rejecting width without ever being
// sampled: image/jpeg rejects dimensions >= 1<<16 before reading pixels.
type oversizedEncodeImage struct{}

func (oversizedEncodeImage) ColorModel() color.Model { return color.RGBAModel }

func (oversizedEncodeImage) Bounds() image.Rectangle {
	return image.Rect(0, 0, 1<<16, 1)
}

func (oversizedEncodeImage) At(int, int) color.Color {
	panic("encode must reject the bounds before sampling pixels")
}

// TestWriteEncodedOutputFailedEncodeWritesNothing proves the no-partial-output
// guarantee: when the encoder fails, req.Output receives nothing.
func TestWriteEncodedOutputFailedEncodeWritesNothing(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	req := &Request{
		Image:  settings.ImageGlobal{Format: formatJPG},
		Output: &output,
	}

	err := writeEncodedOutput(t.Context(), req, oversizedEncodeImage{}, io.Discard)
	if err == nil {
		t.Fatal("writeEncodedOutput succeeded for an oversized image, want an encode error")
	}

	if output.Len() != 0 {
		t.Fatalf("failed encode wrote %d bytes to the output, want 0", output.Len())
	}
}

// TestLimitedImageBufferRejectsOverflow checks the write limit: a write that
// would cross limit fails atomically and the buffer keeps its contents.
func TestLimitedImageBufferRejectsOverflow(t *testing.T) {
	t.Parallel()

	buf := newLimitedImageBuffer(4)
	if _, err := buf.Write([]byte("abc")); err != nil {
		t.Fatalf("Write within limit: %v", err)
	}

	written, err := buf.Write([]byte("de"))
	if !errors.Is(err, errEncodedTooLarge) {
		t.Fatalf("Write past limit error = %v, want %v", err, errEncodedTooLarge)
	}

	if written != 0 || buf.String() != "abc" {
		t.Fatalf("failed Write changed the buffer: written=%d contents=%q", written, buf.String())
	}

	if err := buf.Flush(); err != nil {
		t.Fatalf("Flush = %v, want nil", err)
	}
}

// TestEncodeCopiesOutOfThePooledBuffer proves the pooled scratch cannot leak
// between calls: the bytes returned by one encode stay intact after the next.
func TestEncodeCopiesOutOfThePooledBuffer(t *testing.T) {
	t.Parallel()

	first := makeYCbCrProbeImage(image.Rect(0, 0, 32, 24), true)

	firstBytes, err := encode(first, formatJPG, 80, false)
	if err != nil {
		t.Fatalf("first encode: %v", err)
	}

	snapshot := slices.Clone(firstBytes)

	second := makeYCbCrProbeImage(image.Rect(0, 0, 32, 24), false)
	if _, err := encode(second, formatJPG, 80, false); err != nil {
		t.Fatalf("second encode: %v", err)
	}

	if !bytes.Equal(firstBytes, snapshot) {
		t.Fatal("a later encode mutated an earlier result")
	}
}
