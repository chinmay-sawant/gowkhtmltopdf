package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
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

// jpegImage returns (width, height) of a JPEG by scanning markers.
func jpegSize(data []byte) (int, int, error) {
	if !isJPEG(data) {
		return 0, 0, errors.New("image: not a JPEG")
	}
	pos := 2
	for pos+4 <= len(data) {
		if data[pos] != 0xFF {
			pos++
			continue
		}
		marker := data[pos+1]
		if marker == 0xD9 || marker == 0xDA { // EOI or SOS
			return 0, 0, errors.New("image: no SOF marker found")
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
			if w <= 0 || h <= 0 {
				return 0, 0, errors.New("image: bad JPEG dimensions")
			}
			return w, h, nil
		}
		pos += 2 + segLen
	}
	return 0, 0, errors.New("image: malformed JPEG")
}

// AddJPEGImage embeds a JPEG as a DCTDecode pass-through XObject and paints
// it into the rect. Errors are returned so the layout can fall back.
func (c *Content) AddJPEGImage(name string, x, y, w, h float64, data []byte) error {
	width, height, err := jpegSize(data)
	if err != nil {
		return err
	}
	// determine colorspace from SOF (3 = YCbCr, 1 = gray)
	components := 3
	if cw, err := jpegColorSpace(data); err == nil {
		components = cw
	}
	cs := "DeviceRGB"
	if components == 1 {
		cs = "DeviceGray"
	}
	dict := fmt.Sprintf("<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /%s /BitsPerComponent 8 /Filter /DCTDecode /Length %d >>",
		width, height, cs, len(data))
	ref := c.doc.newObject()
	c.doc.setDict(ref, dict)
	c.doc.setStream(ref, data)
	c.imageRefs[name] = &imageResource{ref: ref, width: width, height: height}
	c.imageUses[name] = ref
	c.Save()
	c.Transform(w, 0, 0, h, x, y)
	c.buf.WriteString("/" + name + " Do\n")
	c.Restore()
	return nil
}

// jpegColorSpace reports the SOF component count (1 = gray, 3 = YCbCr).
func jpegColorSpace(data []byte) (int, error) {
	pos := 2
	for pos+4 <= len(data) {
		if data[pos] != 0xFF {
			pos++
			continue
		}
		marker := data[pos+1]
		if marker == 0xD9 || marker == 0xDA {
			return 0, errors.New("image: no SOF")
		}
		if !jpegLengthMarkers(marker) {
			pos += 2
			continue
		}
		if marker >= 0xC0 && marker <= 0xCF && marker != 0xC4 && marker != 0xC8 && marker != 0xCC {
			return int(data[pos+9]), nil
		}
		segLen := int(data[pos+2])<<8 | int(data[pos+3])
		pos += 2 + segLen
	}
	return 0, errors.New("image: malformed JPEG")
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
	rgba := make([]byte, width*height*3)
	var mask []byte // 8-bit alpha, 0 = transparent
	for yy := 0; yy < height; yy++ {
		for xx := 0; xx < width; xx++ {
			r, g, b, a := img.At(bounds.Min.X+xx, bounds.Min.Y+yy).RGBA()
			off := (yy*width + xx) * 3
			rgba[off] = byte(r >> 8)
			rgba[off+1] = byte(g >> 8)
			rgba[off+2] = byte(b >> 8)
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

	raw := rgba
	dict := fmt.Sprintf("<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8",
		width, height)
	if c.doc.useCompression {
		raw = flateBytes(raw)
		dict += " /Filter /FlateDecode"
	}
	if hasAlpha {
		m := flateBytes(mask)
		maskRef := c.doc.newObject()
		c.doc.setDict(maskRef, fmt.Sprintf(
			"<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceGray /BitsPerComponent 8 /Filter /FlateDecode /Length %d >>",
			width, height, len(m)))
		c.doc.setStream(maskRef, m)
		dict += " /SMask " + maskRef
	}
	dict += fmt.Sprintf(" /Length %d >>", len(raw))
	ref := c.doc.newObject()
	c.doc.setDict(ref, dict)
	c.doc.setStream(ref, raw)
	c.imageRefs[name] = &imageResource{ref: ref, width: width, height: height}
	c.imageUses[name] = ref
	c.Save()
	c.Transform(w, 0, 0, h, x, y)
	c.buf.WriteString("/" + name + " Do\n")
	c.Restore()
	return nil
}
