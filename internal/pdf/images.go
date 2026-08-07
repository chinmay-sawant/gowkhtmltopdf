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
func jpegLengthMarkers(b byte) bool {
	// SOF0-SOF15 (except DHT C4, DAC CC, RST0-D7, TEM 01), SOI D8, EOI D9,
	// SOS DA, DQT DB, DRI DD, DNL DC, DHP DE, EXP DF, APP0-APP15 E0-EF,
	// COM FE, DHT C4, DAC CC - all have a 2-byte length after the marker.
	switch {
	case b == 0x01, b == 0xD8, b == 0xD9:
		return false
	case b >= 0xC0 && b <= 0xFE:
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
		if data[pos] != 0xFF {
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
		segLen := int(data[pos+2])<<8 | int(data[pos+3])
		if marker >= 0xC0 && marker <= 0xCF && marker != 0xC4 && marker != 0xC8 && marker != 0xCC {
			if pos+9 > len(data) {
				break
			}
			h := int(data[pos+5])<<8 | int(data[pos+6])
			w := int(data[pos+7])<<8 | int(data[pos+8])
			components = int(data[pos+9])
			if w <= 0 || h <= 0 {
				return 0, 0, 0, errors.New("image: bad JPEG dimensions")
			}
			return w, h, components, nil
		}
		pos += 2 + segLen
	}
	return 0, 0, 0, errors.New("image: malformed JPEG")
}

// AddJPEGImage embeds a JPEG as a DCTDecode pass-through XObject and paints
// it into the rect. Errors are returned so the layout can fall back.
func (c *Content) AddJPEGImage(name string, x, y, w, h float64, data []byte) error {
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
	cs := "DeviceRGB"
	if components == 1 {
		cs = "DeviceGray"
	}
	name = c.uniqueImageName(name)
	ref := c.doc.newObject()
	c.doc.setDict(ref, dict{}.add("/Type", "/XObject").
		add("/Subtype", "/Image").
		add("/Width", strconv.Itoa(width)).
		add("/Height", strconv.Itoa(height)).
		add("/ColorSpace", "/"+cs).
		add("/BitsPerComponent", "8").
		add("/Filter", "/DCTDecode").
		add("/Length", strconv.Itoa(len(data))).String())
	c.doc.setStream(ref, data)
	c.imageRefs[name] = &imageResource{ref: ref, width: width, height: height}
	c.imageUses[name] = ref.String()
	c.Save()
	c.Transform(w, 0, 0, h, x, y)
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
	b := img.Bounds()
	gray := image.NewGray(image.Rect(0, 0, b.Dx(), b.Dy()))
	for yy := 0; yy < b.Dy(); yy++ {
		for xx := 0; xx < b.Dx(); xx++ {
			r, g, bl, _ := img.At(b.Min.X+xx, b.Min.Y+yy).RGBA()
			gray.SetGray(xx, yy, color.Gray{
				Y: uint8(0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(bl>>8)),
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
	rgba := make([]byte, width*height*3)
	var mask []byte // 8-bit alpha, 0 = transparent
	for yy := 0; yy < height; yy++ {
		for xx := 0; xx < width; xx++ {
			r, g, b, a := img.At(bounds.Min.X+xx, bounds.Min.Y+yy).RGBA()
			off := (yy*width + xx) * 3
			if grayscale {
				v := byte(0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8))
				rgba[off], rgba[off+1], rgba[off+2] = v, v, v
			} else {
				rgba[off] = byte(r >> 8)
				rgba[off+1] = byte(g >> 8)
				rgba[off+2] = byte(b >> 8)
			}
			if a < 0xFFFF {
				hasAlpha = true
			}
		}
	}
	if hasAlpha {
		mask = make([]byte, width*height)
		for yy := 0; yy < height; yy++ {
			for xx := 0; xx < width; xx++ {
				_, _, _, a := img.At(bounds.Min.X+xx, bounds.Min.Y+yy).RGBA()
				mask[yy*width+xx] = byte(a >> 8)
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
		m := flateBytes(mask)
		maskRef := c.doc.newObject()
		c.doc.setDict(maskRef, dict{}.add("/Type", "/XObject").
			add("/Subtype", "/Image").
			add("/Width", strconv.Itoa(width)).
			add("/Height", strconv.Itoa(height)).
			add("/ColorSpace", "/DeviceGray").
			add("/BitsPerComponent", "8").
			add("/Filter", "/FlateDecode").
			add("/Length", strconv.Itoa(len(m))).String())
		c.doc.setStream(maskRef, m)
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
