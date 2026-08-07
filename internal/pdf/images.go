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

// jpegScan scans JPEG markers once and returns the SOF dimensions and
// component count (1 = gray, 3 = YCbCr). One marker walk serves both the
// /Width//Height//ColorSpace dict and the grayscale desaturation check.
func jpegScan(data []byte) (w, h, components int, err error) {
	if !isJPEG(data) {
		return 0, 0, 0, errors.New("image: not a JPEG")
	}

	pos := 2
	for pos+4 <= len(data) {
		if data[pos] != jpegMarkerPrefix {
			pos++

			continue
		}

		marker := data[pos+1]
		if marker == 0xD9 || marker == 0xDA { // EOI or SOS
			return 0, 0, 0, errors.New("image: no SOF marker found")
		}

		if !jpegLengthMarkers(marker) {
			pos += 2

			continue
		}

		if pos+4 > len(data) {
			break
		}

		segLen := int(data[pos+2])<<bitsPerByte | int(data[pos+3])

		if marker >= 0xC0 && marker <= 0xCF && marker != 0xC4 && marker != 0xC8 && marker != 0xCC {
			if pos+9 > len(data) {
				break
			}

			height := int(data[pos+5])<<bitsPerByte | int(data[pos+6])
			width := int(data[pos+7])<<bitsPerByte | int(data[pos+8])
			components = int(data[pos+9])

			if width <= 0 || height <= 0 {
				return 0, 0, 0, errors.New("image: bad JPEG dimensions")
			}

			return width, height, components, nil
		}

		pos += uint16Bytes + segLen
	}

	return 0, 0, 0, errors.New("image: malformed JPEG")
}

// AddJPEGImage embeds a JPEG as a DCTDecode pass-through XObject and paints
// it into the rect. Errors are returned so the layout can fall back.
func (c *Content) AddJPEGImage(name string, posX, posY, drawW, drawH float64, data []byte) error {
	width, height, components, err := jpegScan(data)
	if err != nil {
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
	img, err := jpeg.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	gray := image.NewGray(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))

	for yy := range bounds.Dy() {
		for xx := range bounds.Dx() {
			red, green, blue, _ := img.At(bounds.Min.X+xx, bounds.Min.Y+yy).RGBA()
			gray.SetGray(xx, yy, color.Gray{
				Y: uint8(lumaR*float64(red>>bitsPerByte) + lumaG*float64(green>>bitsPerByte) + lumaB*float64(blue>>bitsPerByte)),
			})
		}
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, gray, nil); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// AddPNGImage decodes PNG with image/png and embeds as a Flate RGB
// XObject (alpha channel becomes a soft-mask when present).
func (c *Content) AddPNGImage(name string, x, y, w, h float64, data []byte) error {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("image: %w", err)
	}

	bounds := img.Bounds()

	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return errors.New("image: empty PNG")
	}

	hasAlpha := false
	grayscale := c.doc != nil && c.doc.grayscale
	rgba := make([]byte, width*height*rgbChannels)

	var mask []byte // 8-bit alpha, 0 = transparent

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

	if hasAlpha {
		mask = make([]byte, width*height)

		for yy := range height {
			for xx := range width {
				_, _, _, a := img.At(bounds.Min.X+xx, bounds.Min.Y+yy).RGBA()
				mask[yy*width+xx] = byte(a >> bitsPerByte)
			}
		}
	}

	name = c.uniqueImageName(name)
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
	c.Save()
	c.Transform(w, 0, 0, h, x, y)
	c.buf.WriteString("/" + name + " Do\n")
	c.Restore()

	return nil
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
