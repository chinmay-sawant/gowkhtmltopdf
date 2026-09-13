package imageout

import (
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/png"
	"io"
)

// fastPNGDeflateLevel is the deflate level for the fast encoder. Level 2
// keeps Go's shared initDeflate allocation profile (BestSpeed allocates a
// separate 256 KiB token buffer and fast-deflate tables, which pushed B/op
// over the image ceiling) while measuring the same encode wall time on the
// tile workload; see results/image/image-time.md.
const fastPNGDeflateLevel = 2

// fastPNGChunkBytes is the IDAT chunk payload size. Multiple IDAT chunks are
// legal PNG and concatenate into one zlib stream; 8 KiB keeps the chunk
// framing overhead near zero while holding less scratch memory than the
// image/png encoder's row buffers (the 250/500 tile B/op ceiling is tight).
const fastPNGChunkBytes = 8 << 10

// PNG color types 2 (truecolor) and 6 (truecolor with alpha), matching what
// image/png writes for an *image.NRGBA source. The channel counts are the
// bytes per pixel for those color types.
const (
	pngColorTruecolor = 2
	pngColorAlpha     = 6
	pngChannelsRGB    = 3
	pngChannelsRGBA   = 4
	pngSignature      = "\x89PNG\r\n\x1a\n"
)

// encodePNG writes img as PNG. *image.NRGBA canvases at or above the
// directRasterPixels boundary (the same large-canvas line the direct raster
// branch documents) take encodeFastPNG; everything else keeps image/png so
// small outputs stay byte-for-byte identical. opaque is the caller's
// all-alpha-255 guarantee; false makes the fast path scan before choosing a
// color type, so a wrong hint can only be conservative, never lossy.
func encodePNG(w io.Writer, img image.Image, opaque bool) error {
	if nrgba, ok := img.(*image.NRGBA); ok && fastPNGCanvas(nrgba.Bounds()) {
		return encodeFastPNG(w, nrgba, opaque)
	}

	if err := png.Encode(w, img); err != nil {
		return fmt.Errorf("image/png: %w", err)
	}

	return nil
}

// fastPNGCanvas reports whether a canvas takes the fast encoder. The pixel
// area rule mirrors directRaster: large canvases already paint directly, and
// the per-row filter selection image/png pays is what makes encoding them the
// next bottleneck.
func fastPNGCanvas(bounds image.Rectangle) bool {
	return bounds.Dx()*bounds.Dy() >= directRasterPixels
}

// encodeFastPNG writes src as PNG without image/png's per-row adaptive filter
// selection, which dominates large-canvas encodes (see image-time.md: 20 to
// 32 percent of the 250/500 tile benchmarks in image/png.filter alone). Every
// row uses filter type 0 (None) and the deflate stream runs at
// fastPNGDeflateLevel. Output pixels are bit-identical to image/png's because
// PNG filters and compression levels are lossless; what changes is encoded
// size, which the results table records honestly (about +50 percent on the
// tile workload).
//
// The encoder is deliberately narrow: only *image.NRGBA (the canvas type
// rasterizeContext returns) and only canvases at or above directRasterPixels.
// Smaller images keep image/png byte-for-byte, so this fast path is a
// large-canvas decision like the direct raster branch, not a global output
// policy change.
func encodeFastPNG(writer io.Writer, src *image.NRGBA, opaque bool) error {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	opaqueKnown := opaque
	if !opaqueKnown {
		opaqueKnown = src.Opaque()
	}

	encoder, err := startFastPNG(writer, width, height, opaqueKnown)
	if err != nil {
		return err
	}

	for rowIndex := range height {
		offset := src.PixOffset(bounds.Min.X, bounds.Min.Y+rowIndex)
		if err := encoder.writeNRGBARow(src.Pix[offset:]); err != nil {
			return fmt.Errorf("png deflate row %d: %w", rowIndex, err)
		}
	}

	return encoder.close()
}

// pngRowEncoder writes one PNG from sequential NRGBA rows. Strip raster and
// the full-canvas fast path share this so filter None, the deflate level, and
// IDAT framing stay one implementation.
type pngRowEncoder struct {
	idat     fastIDATWriter
	deflater *zlib.Writer
	row      []byte
	width    int
	opaque   bool
}

// startFastPNG writes the PNG signature and IHDR, then returns a row encoder.
func startFastPNG(writer io.Writer, width, height int, opaque bool) (*pngRowEncoder, error) {
	channels, colorType := pngColorPlan(opaque)

	if _, err := io.WriteString(writer, pngSignature); err != nil {
		return nil, fmt.Errorf("png signature: %w", err)
	}

	if err := writePNGHeader(writer, width, height, colorType); err != nil {
		return nil, err
	}

	encoder := &pngRowEncoder{ //nolint:exhaustruct // idat and deflater are attached below
		row:    make([]byte, 1+width*channels),
		width:  width,
		opaque: opaque,
	}
	encoder.row[0] = 0 // filter type None
	encoder.idat.w = writer

	deflater, err := zlib.NewWriterLevel(&encoder.idat, fastPNGDeflateLevel)
	if err != nil {
		return nil, fmt.Errorf("png deflate writer: %w", err)
	}

	encoder.deflater = deflater

	return encoder, nil
}

// writeNRGBARow appends one packed NRGBA row to the deflate stream. pix must
// hold at least width*4 bytes of tightly packed NRGBA.
func (e *pngRowEncoder) writeNRGBARow(pix []byte) error {
	raw := e.row[1:]

	if e.opaque {
		packOpaqueRGBRow(raw, pix, e.width)
	} else {
		copy(raw, pix[:e.width*4])
	}

	if _, err := e.deflater.Write(e.row); err != nil {
		return fmt.Errorf("png deflate row: %w", err)
	}

	return nil
}

func (e *pngRowEncoder) close() error {
	if err := e.deflater.Close(); err != nil {
		return fmt.Errorf("png deflate close: %w", err)
	}

	if err := e.idat.flush(); err != nil {
		return err
	}

	return writePNGChunk(e.idat.w, "IEND", nil)
}

// pngColorPlan returns the PNG channel count and color type for an opaque or
// translucent NRGBA canvas.
func pngColorPlan(opaque bool) (int, byte) {
	if opaque {
		return pngChannelsRGB, pngColorTruecolor
	}

	return pngChannelsRGBA, pngColorAlpha
}

// writePNGHeader writes the 13-byte IHDR chunk for an 8-bit canvas.
func writePNGHeader(writer io.Writer, width, height int, colorType byte) error {
	var ihdr [13]byte

	//nolint:gosec // canvas dimensions are bounded by validateRasterSize (<= 16384)
	binary.BigEndian.PutUint32(ihdr[0:], uint32(width))
	//nolint:gosec // canvas dimensions are bounded by validateRasterSize (<= 16384)
	binary.BigEndian.PutUint32(ihdr[4:], uint32(height))
	ihdr[8] = 8 // bit depth
	ihdr[9] = colorType

	if err := writePNGChunk(writer, "IHDR", ihdr[:]); err != nil {
		return fmt.Errorf("png IHDR: %w", err)
	}

	return nil
}

// packOpaqueRGBRow copies an opaque NRGBA row into an RGB row, dropping the
// alpha byte. Four pixels are packed with 32/64-bit reads and writes so the
// per-pixel byte shuffle runs at roughly memory speed; the tail falls back to
// per-channel copies. src must hold at least width*4 bytes.
//
//nolint:mnd // the constants are the fixed 24/16/8-bit PNG RGB byte layout
func packOpaqueRGBRow(dst, src []byte, width int) {
	offset := 0
	out := 0
	remaining := width

	for ; remaining >= 4; remaining -= 4 {
		word0 := binary.LittleEndian.Uint32(src[offset:])
		word1 := binary.LittleEndian.Uint32(src[offset+4:])
		word2 := binary.LittleEndian.Uint32(src[offset+8:])
		word3 := binary.LittleEndian.Uint32(src[offset+12:])

		binary.LittleEndian.PutUint64(dst[out:],
			uint64(word0&0xFFFFFF)|uint64(word1&0xFFFFFF)<<24|uint64(word2&0xFFFF)<<48)
		binary.LittleEndian.PutUint32(dst[out+8:],
			(word2>>16)&0xFF|(word3&0xFFFFFF)<<8)

		offset += 16
		out += 12
	}

	for ; remaining > 0; remaining-- {
		dst[out] = src[offset]
		dst[out+1] = src[offset+1]
		dst[out+2] = src[offset+2]
		offset += 4
		out += 3
	}
}

// fastIDATWriter frames compressed bytes into IDAT chunks as they stream.
type fastIDATWriter struct {
	w   io.Writer
	buf [fastPNGChunkBytes]byte
	n   int
}

func (c *fastIDATWriter) Write(payload []byte) (int, error) {
	total := 0

	for len(payload) > 0 {
		n := copy(c.buf[c.n:], payload)
		c.n += n
		payload = payload[n:]
		total += n

		if c.n == len(c.buf) {
			if err := c.flush(); err != nil {
				return total, err
			}
		}
	}

	return total, nil
}

func (c *fastIDATWriter) flush() error {
	if c.n == 0 {
		return nil
	}

	if err := writePNGChunk(c.w, "IDAT", c.buf[:c.n]); err != nil {
		return err
	}

	c.n = 0

	return nil
}

// writePNGChunk writes one length/type/data/CRC chunk.
func writePNGChunk(writer io.Writer, kind string, data []byte) error {
	var header [8]byte

	//nolint:gosec // chunk payloads are capped at fastPNGChunkBytes (8 KiB)
	binary.BigEndian.PutUint32(header[0:4], uint32(len(data)))
	copy(header[4:], kind)

	if _, err := writer.Write(header[:]); err != nil {
		return fmt.Errorf("png chunk %s header: %w", kind, err)
	}

	if len(data) > 0 {
		if _, err := writer.Write(data); err != nil {
			return fmt.Errorf("png chunk %s data: %w", kind, err)
		}
	}

	crc := crc32.Update(0, crc32.IEEETable, []byte(kind))
	crc = crc32.Update(crc, crc32.IEEETable, data)

	var tail [4]byte

	binary.BigEndian.PutUint32(tail[:], crc)

	if _, err := writer.Write(tail[:]); err != nil {
		return fmt.Errorf("png chunk %s crc: %w", kind, err)
	}

	return nil
}
