package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strconv"
)

var (
	errImageNotJPEG         = errors.New("image: not a JPEG")
	errImageNoSOF           = errors.New("image: no SOF marker found")
	errImageBadDims         = errors.New("image: bad JPEG dimensions")
	errImageMalformed       = errors.New("image: malformed JPEG")
	errImageEmptyPNG        = errors.New("image: empty PNG")
	errImageEncodedTooLarge = errors.New("image: encoded body too large")
	errImageDecodedTooLarge = errors.New("image: decoded image too large")
)

const (
	maxEmbeddedImageDimension = 16_384
	maxEmbeddedImagePixels    = 16 * 1024 * 1024
	maxEmbeddedEncodedBytes   = 32 << 20
	maxEmbeddedDecodedBytes   = 128 << 20
	decodedImageBytesPerPixel = 8 // decoder plus RGB/alpha output working set
)

func validateEmbeddedImage(dataLen, width, height int) error {
	if dataLen > maxEmbeddedEncodedBytes {
		return fmt.Errorf(
			"%w: %d bytes, limit %d",
			errImageEncodedTooLarge,
			dataLen,
			maxEmbeddedEncodedBytes,
		)
	}

	if width <= 0 || height <= 0 || width > maxEmbeddedImageDimension || height > maxEmbeddedImageDimension {
		return fmt.Errorf(
			"%w: dimensions %dx%d exceed %dx%d",
			errImageDecodedTooLarge,
			width,
			height,
			maxEmbeddedImageDimension,
			maxEmbeddedImageDimension,
		)
	}

	pixels := int64(width) * int64(height)
	if pixels > maxEmbeddedImagePixels || pixels*decodedImageBytesPerPixel > maxEmbeddedDecodedBytes {
		return fmt.Errorf("%w: %d pixels exceed budget", errImageDecodedTooLarge, pixels)
	}

	return nil
}

// jpegMarkers is the set of JPEG marker bytes that precede a length field.
func jpegLengthMarkers(buf byte) bool {
	// SOF0-SOF15 (except DHT C4, DAC CC, RST0-D7, TEM 01), SOI D8, EOI D9,
	// SOS DA, DQT DB, DRI DD, DNL DC, DHP DE, EXP DF, APP0-APP15 E0-EF,
	// COM FE, DHT C4, DAC CC - all have a 2-byte length after the marker.
	switch {
	case buf == jpegTEM, buf == jpegSOI, buf == jpegEOI:
		return false
	case buf >= 0xC0 && buf <= 0xFE:
		return true
	}

	return false
}

// isJPEG reports whether data looks like a JPEG stream.
func isJPEG(data []byte) bool {
	return len(data) > 4 && data[0] == 0xFF && data[1] == 0xD8 && data[len(data)-2] == 0xFF && data[len(data)-1] == 0xD9
}

// isSOFMarker reports whether marker byte is a start-of-frame marker that
// carries dimensions (excluding DHT 0xC4, JPG 0xC8 and DAC 0xCC).
func isSOFMarker(marker byte) bool {
	return marker >= 0xC0 && marker <= 0xCF && marker != 0xC4 && marker != 0xC8 && marker != 0xCC
}

// isScanMarker reports whether marker is EOI (0xD9) or SOS (0xDA).
func isScanMarker(marker byte) bool {
	return marker == 0xD9 || marker == 0xDA
}

// jpegScan scans JPEG markers once and returns the SOF dimensions and
// component count (1 = gray, 3 = YCbCr). One marker walk serves both the
// /Width//Height//ColorSpace dict and the grayscale desaturation check.
func jpegScan(data []byte) (int, int, int, error) {
	if !isJPEG(data) {
		return 0, 0, 0, errImageNotJPEG
	}

	pos := 2
	for pos+4 <= len(data) {
		if data[pos] != jpegMarkerPrefix {
			pos++

			continue
		}

		marker := data[pos+1]
		if isScanMarker(marker) { // EOI or SOS
			return 0, 0, 0, errImageNoSOF
		}

		if !jpegLengthMarkers(marker) {
			pos += 2

			continue
		}

		if pos+4 > len(data) {
			break
		}

		segLen := int(data[pos+2])<<bitsPerByte | int(data[pos+3])

		if isSOFMarker(marker) {
			if pos+9 > len(data) {
				break
			}

			height := int(data[pos+5])<<bitsPerByte | int(data[pos+6])
			width := int(data[pos+7])<<bitsPerByte | int(data[pos+8])
			components := int(data[pos+9])

			if !validJPEGDims(width, height) {
				return 0, 0, 0, errImageBadDims
			}

			return width, height, components, nil
		}

		pos += uint16Bytes + segLen
	}

	return 0, 0, 0, errImageMalformed
}

// validJPEGDims reports whether the decoded SOF dimensions make sense.
func validJPEGDims(width, height int) bool {
	return width > 0 && height > 0
}

// AddJPEGImage embeds a JPEG as a DCTDecode pass-through XObject and paints
// it into the rect. Errors are returned so the layout can fall back.
func (c *Content) AddJPEGImage(name string, posX, posY, drawW, drawH float64, data []byte) error {
	if len(data) > maxEmbeddedEncodedBytes {
		return fmt.Errorf(
			"%w: %d bytes, limit %d",
			errImageEncodedTooLarge,
			len(data),
			maxEmbeddedEncodedBytes,
		)
	}

	width, height, components, err := jpegScan(data)
	if err != nil {
		return err
	}
	if err := validateEmbeddedImage(len(data), width, height); err != nil {
		return err
	}
	// Grayscale mode desaturates at embed time: decode, fold Rec.601 luma,
	// re-encode as a 1-component gray JPEG. Pass-through stays untouched.
	if c.doc != nil && c.doc.grayscale && components > 1 {
		if gray, err := grayJPEG(data); err == nil {
			data = gray
			components = 1
		}
	}

	csVal := "DeviceRGB"
	if components == 1 {
		csVal = "DeviceGray"
	}

	name = c.uniqueImageName(name)
	ref := c.doc.newObject()
	c.doc.setDict(ref, dict{}.add("/Type", "/XObject").
		add("/Subtype", "/Image").
		add("/Width", strconv.Itoa(width)).
		add("/Height", strconv.Itoa(height)).
		add("/ColorSpace", "/"+csVal).
		add("/BitsPerComponent", "8").
		add("/Filter", "/DCTDecode").
		add("/Length", strconv.Itoa(len(data))).String())
	c.doc.setStream(ref, data)
	c.imageRefs[name] = &imageResource{ref: ref, width: width, height: height}
	c.imageUses[name] = ref.String()
	c.Save()
	c.Transform(drawW, 0, 0, drawH, posX, posY)
	c.buf.WriteString("/" + name + " Do\n")
	c.Restore()

	return nil
}

// grayJPEG decodes a JPEG and re-encodes it as a grayscale JPEG. The decode
// is a best effort: a malformed stream keeps its original bytes.
func grayJPEG(data []byte) ([]byte, error) {
	if len(data) > maxEmbeddedEncodedBytes {
		return nil, fmt.Errorf(
			"%w: %d bytes, limit %d",
			errImageEncodedTooLarge,
			len(data),
			maxEmbeddedEncodedBytes,
		)
	}

	width, height, _, err := jpegScan(data)
	if err != nil {
		return nil, err
	}
	if err := validateEmbeddedImage(len(data), width, height); err != nil {
		return nil, err
	}

	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("image: decode: %w", err)
	}

	bounds := img.Bounds()
	if err := validateEmbeddedImage(len(data), bounds.Dx(), bounds.Dy()); err != nil {
		return nil, err
	}
	gray := image.NewGray(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))

	for yy := range bounds.Dy() {
		for xx := range bounds.Dx() {
			red, green, blue, _ := img.At(bounds.Min.X+xx, bounds.Min.Y+yy).RGBA()
			gray.SetGray(xx, yy, color.Gray{
				Y: uint8(lumaR*float64(red>>bitsPerByte) + lumaG*float64(green>>bitsPerByte) + lumaB*float64(blue>>bitsPerByte)),
			})
		}
	}

	buf := limitedImageBuffer{limit: maxEmbeddedEncodedBytes}
	if err := jpeg.Encode(&buf, gray, nil); err != nil {
		return nil, fmt.Errorf("image: encode: %w", err)
	}

	return buf.Bytes(), nil
}

type limitedImageBuffer struct {
	bytes.Buffer
	limit int
}

func (b *limitedImageBuffer) Write(data []byte) (int, error) {
	if len(data) > b.limit-b.Len() {
		return 0, errImageEncodedTooLarge
	}

	return b.Buffer.Write(data)
}

// AddPNGImage decodes PNG with image/png and embeds as a Flate RGB
// XObject (alpha channel becomes a soft-mask when present).
func (c *Content) AddPNGImage(name string, posX, posY, drawWidth, drawHeight float64, data []byte) error {
	if len(data) > maxEmbeddedEncodedBytes {
		return fmt.Errorf(
			"%w: %d bytes, limit %d",
			errImageEncodedTooLarge,
			len(data),
			maxEmbeddedEncodedBytes,
		)
	}

	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("image: config: %w", err)
	}
	if err := validateEmbeddedImage(len(data), cfg.Width, cfg.Height); err != nil {
		return err
	}

	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("image: %w", err)
	}

	bounds := img.Bounds()

	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return errImageEmptyPNG
	}
	if err := validateEmbeddedImage(len(data), width, height); err != nil {
		return err
	}

	grayscale := c.doc != nil && c.doc.grayscale
	rgba, hasAlpha := renderImagePixels(img, bounds, grayscale)

	var mask []byte // 8-bit alpha, 0 = transparent

	if hasAlpha {
		mask = renderAlphaMask(img, bounds)
	}

	name = c.uniqueImageName(name)
	c.emitPNGXObject(name, width, height, rgba, mask, hasAlpha)

	c.Save()
	c.Transform(drawWidth, 0, 0, drawHeight, posX, posY)
	c.buf.WriteString("/" + name + " Do\n")
	c.Restore()

	return nil
}

// emitPNGXObject builds the Flate RGB XObject (plus alpha soft-mask) and
// registers it on the content stream.
func (c *Content) emitPNGXObject(name string, width, height int, rgba, mask []byte, hasAlpha bool) {
	raw := rgba
	dct := dict{}.add("/Type", "/XObject").
		add("/Subtype", "/Image").
		add("/Width", strconv.Itoa(width)).
		add("/Height", strconv.Itoa(height)).
		add("/ColorSpace", "/DeviceRGB").
		add("/BitsPerComponent", "8")

	if c.doc.useCompression {
		raw = flateBytes(raw)
		dct = dct.add("/Filter", "/FlateDecode")
	}

	if hasAlpha {
		mVal := flateBytes(mask)
		maskRef := c.doc.newObject()
		c.doc.setDict(maskRef, dict{}.add("/Type", "/XObject").
			add("/Subtype", "/Image").
			add("/Width", strconv.Itoa(width)).
			add("/Height", strconv.Itoa(height)).
			add("/ColorSpace", "/DeviceGray").
			add("/BitsPerComponent", "8").
			add("/Filter", "/FlateDecode").
			add("/Length", strconv.Itoa(len(mVal))).String())
		c.doc.setStream(maskRef, mVal)
		dct = dct.add("/SMask", maskRef.String())
	}

	dct = dct.add("/Length", strconv.Itoa(len(raw)))
	ref := c.doc.newObject()
	c.doc.setDict(ref, dct.String())
	c.doc.setStream(ref, raw)
	c.imageRefs[name] = &imageResource{ref: ref, width: width, height: height}
	c.imageUses[name] = ref.String()
}

// renderImagePixels folds the image into an RGB byte slice, returning
// whether any pixel has alpha below fully opaque.
func renderImagePixels(img image.Image, bounds image.Rectangle, grayscale bool) ([]byte, bool) {
	width, height := bounds.Dx(), bounds.Dy()
	rgba := make([]byte, width*height*rgbChannels)
	hasAlpha := false

	for yy := range height {
		for xx := range width {
			red, green, blue, alpha := img.At(bounds.Min.X+xx, bounds.Min.Y+yy).RGBA()
			off := (yy*width + xx) * rgbChannels

			if grayscale {
				v := byte(lumaR*float64(red>>bitsPerByte) + lumaG*float64(green>>bitsPerByte) + lumaB*float64(blue>>bitsPerByte))
				rgba[off], rgba[off+1], rgba[off+2] = v, v, v
			} else {
				rgba[off] = byte(red >> bitsPerByte)
				rgba[off+1] = byte(green >> bitsPerByte)
				rgba[off+2] = byte(blue >> bitsPerByte)
			}

			if alpha < maxUint16Val {
				hasAlpha = true
			}
		}
	}

	return rgba, hasAlpha
}

// renderAlphaMask extracts an 8-bit alpha mask (0 = transparent).
func renderAlphaMask(img image.Image, bounds image.Rectangle) []byte {
	width, height := bounds.Dx(), bounds.Dy()
	mask := make([]byte, width*height)

	for yy := range height {
		for xx := range width {
			_, _, _, a := img.At(bounds.Min.X+xx, bounds.Min.Y+yy).RGBA()
			mask[yy*width+xx] = byte(a >> bitsPerByte)
		}
	}

	return mask
}

// uniqueImageName returns a page-local resource name that has not already
// been registered on this content stream. Header/footer bands and body paint
// can each start at I0; silently reusing that name makes the second image
// replace the first image in /Resources while both operators still say /I0.
// Suffixing the requested name keeps existing callers source-compatible while
// making the emitted operator and resource dictionary agree.
func (c *Content) uniqueImageName(name string) string {
	if name == "" {
		name = "I0"
	}

	if _, exists := c.imageRefs[name]; !exists {
		return name
	}

	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s_%d", name, i)
		if _, exists := c.imageRefs[candidate]; !exists {
			return candidate
		}
	}
}
