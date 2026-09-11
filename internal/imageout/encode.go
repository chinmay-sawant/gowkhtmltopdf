package imageout

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"slices"
	"sync"
)

// encode serializes img as PNG or JPEG and returns owned bytes. quality applies
// to JPEG only (1..100); PNG is lossless and ignores it. opaque reports a
// caller guarantee that every pixel alpha is 255; a false report makes the PNG
// path scan the canvas, so the hint can only save the scan, never drop alpha.
//
// encodeBufferPool bounds the scratch at maxImageEncoded and encode copies the
// result out before the buffer returns to the pool, so callers own an
// independent slice and a failed encode can never leak partial output. Callers
// that can write the bytes out before returning (writeEncodedOutput) use
// encodeInto with acquireEncodeBuffer to skip that copy.
func encode(img image.Image, format string, quality int, opaque bool) ([]byte, error) {
	buf := acquireEncodeBuffer()
	defer releaseEncodeBuffer(buf)

	if err := encodeInto(buf, img, format, quality, opaque); err != nil {
		return nil, err
	}

	return slices.Clone(buf.Bytes()), nil
}

// encodeInto serializes img into buf. The buffer belongs to the caller, who
// must release it through releaseEncodeBuffer after the bytes are no longer
// needed; nothing here resets or returns it.
func encodeInto(buf *limitedImageBuffer, img image.Image, format string, quality int, opaque bool) error {
	switch format {
	case formatPNG:
		if err := encodePNG(buf, img, opaque); err != nil {
			return fmt.Errorf("png encode: %w", err)
		}
	case formatJPG:
		if err := jpeg.Encode(buf, ycbcr420FastPath(img), &jpeg.Options{Quality: clampJPEGQuality(quality)}); err != nil {
			return fmt.Errorf("jpeg encode: %w", err)
		}
	default:
		return fmt.Errorf("%w %q", errUnsupportedFmt, format)
	}

	return nil
}

// acquireEncodeBuffer takes one pooled scratch buffer, already reset.
func acquireEncodeBuffer() *limitedImageBuffer {
	buf := encodeBufferPool.Get().(*limitedImageBuffer) //nolint:forcetypeassert // the pool only stores this type
	buf.Reset()

	return buf
}

// releaseEncodeBuffer clears buf and returns it to the pool unless its capacity
// outgrew maxImageEncoded, which would pin more memory than the cap allows.
func releaseEncodeBuffer(buf *limitedImageBuffer) {
	buf.Reset()

	if cap(buf.Bytes()) <= maxImageEncoded {
		encodeBufferPool.Put(buf)
	}
}

// clampJPEGQuality bounds a requested JPEG quality to the 1..100 range. PNG
// ignores quality, so only the JPEG path calls this.
func clampJPEGQuality(quality int) int {
	if quality < 1 {
		return 1
	}

	if quality > qualityMaxPercent {
		return qualityMaxPercent
	}

	return quality
}

// ycbcr420FastPath returns the 4:2:0 planes image/jpeg reads directly when
// the NRGBA conversion is byte-exact; non-NRGBA inputs (and odd origins) keep
// the encoder's own At-based conversion.
func ycbcr420FastPath(img image.Image) image.Image {
	nrgba, ok := img.(*image.NRGBA)
	if !ok {
		return img
	}

	if ycbcr := nrgbaToYCbCr420(nrgba); ycbcr != nil {
		return ycbcr
	}

	return img
}

// limitedImageBuffer is an io.Writer that refuses to grow past limit bytes.
// Flush makes it satisfy image/jpeg's internal writer interface, so the
// encoder writes straight into the buffer instead of adding a bufio.Writer.
type limitedImageBuffer struct {
	bytes.Buffer
	limit int
}

// Flush reports success: bytes.Buffer holds no buffered state to drain.
func (*limitedImageBuffer) Flush() error { return nil }

func (b *limitedImageBuffer) Write(data []byte) (int, error) {
	if len(data) > b.limit-b.Len() {
		return 0, errEncodedTooLarge
	}

	n, err := b.Buffer.Write(data)
	if err != nil {
		return n, fmt.Errorf("write buffer: %w", err)
	}

	return n, nil
}

// newLimitedImageBuffer returns one encode scratch buffer.
func newLimitedImageBuffer(limit int) *limitedImageBuffer {
	return &limitedImageBuffer{Buffer: bytes.Buffer{}, limit: limit}
}

// encodeBufferPool recycles encode scratch across conversions.
// releaseEncodeBuffer drops buffers whose capacity outgrew maxImageEncoded
// instead of pinning them.
//
//nolint:gochecknoglobals // bounded encode scratch recycling
var encodeBufferPool = sync.Pool{
	New: func() any { return newLimitedImageBuffer(maxImageEncoded) },
}
