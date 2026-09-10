package imageout

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"slices"
	"sync"
)

// encode serializes img as PNG or JPEG. quality applies to JPEG only
// (1..100); PNG is lossless and ignores it. The scratch buffer is pooled and
// bounded by maxImageEncoded, and the encoded bytes are copied out before the
// buffer returns to the pool, so callers own an independent slice and a
// failed encode can never leak partial output.
func encode(img image.Image, format string, quality int) ([]byte, error) {
	buf := encodeBufferPool.Get().(*limitedImageBuffer) //nolint:forcetypeassert // the pool only stores this type
	buf.Reset()

	defer func() {
		buf.Reset()

		if cap(buf.Bytes()) <= maxImageEncoded {
			encodeBufferPool.Put(buf)
		}
	}()

	switch format {
	case formatPNG:
		if err := png.Encode(buf, img); err != nil {
			return nil, fmt.Errorf("png encode: %w", err)
		}
	case formatJPG:
		if err := jpeg.Encode(buf, ycbcr420FastPath(img), &jpeg.Options{Quality: clampJPEGQuality(quality)}); err != nil {
			return nil, fmt.Errorf("jpeg encode: %w", err)
		}
	default:
		return nil, fmt.Errorf("%w %q", errUnsupportedFmt, format)
	}

	return slices.Clone(buf.Bytes()), nil
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

// encodeBufferPool recycles encode scratch across conversions. encode drops
// buffers whose capacity outgrew maxImageEncoded instead of pinning them.
//
//nolint:gochecknoglobals // bounded encode scratch recycling
var encodeBufferPool = sync.Pool{
	New: func() any { return newLimitedImageBuffer(maxImageEncoded) },
}
