// Package svg provides SVG-as-image rasterization for <img src="*.svg">.
// Sole path: github.com/tdewolff/canvas (ParseSVG + rasterizer), which
// handles complex wiki logos (gradients, groups, clipPaths, arcs).
// On canvas failure, Rasterize returns a non-nil error and empty PNG
// (nil bytes, zero size); there is no in-tree fallback rasterizer and
// no ImageMagick/convert shell path.
//
// ponytail: canvas is sole SVG raster path; no second in-tree rasterizer.
package svg

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"image/png"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/tdewolff/canvas"
	"github.com/tdewolff/canvas/renderers/rasterizer"
)

// Rasterize decodes SVG XML into a PNG image via tdewolff/canvas only.
// maxSide caps the longer edge in pixels (default 512).
// On failure (not SVG, parse/draw error, empty size, or canvas panic),
// returns err with nil pngBytes and zero w/h — callers must treat error
// as "no image". There is no second rasterizer or shell fallback.
func Rasterize(data []byte, maxSide int) (pngBytes []byte, w, h int, err error) {
	if maxSide <= 0 {
		maxSide = 512
	}
	if !looksLikeSVG(data) {
		return nil, 0, 0, fmt.Errorf("svg: not SVG")
	}
	return rasterizeCanvas(data, maxSide)
}

// rasterizeCanvas uses tdewolff/canvas to parse SVG and rasterize to PNG.
// Canvas stores sizes in millimeters (unitless root width/height may be
// treated as mm). We pick resolution so the longer edge is the SVG's
// viewBox/width/height in CSS pixels (capped by maxSide), matching layout's
// 96dpi intrinsic-size model.
//
// Canvas can panic on some malformed paths; recover turns that into a
// clean error so <img src="bad.svg"> does not crash the converter.
func rasterizeCanvas(data []byte, maxSide int) (pngBytes []byte, w, h int, err error) {
	defer func() {
		if r := recover(); r != nil {
			pngBytes, w, h = nil, 0, 0
			err = fmt.Errorf("svg canvas: panic: %v", r)
		}
	}()

	c, err := canvas.ParseSVG(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("svg canvas: %w", err)
	}
	cw, ch := c.Size()
	if cw <= 0 || ch <= 0 {
		return nil, 0, 0, fmt.Errorf("svg canvas: empty size")
	}

	// Intrinsic CSS-pixel size from viewBox / width / height attributes.
	targetW, targetH := svgCSSPixelSize(data, maxSide)
	// Map canvas mm → target pixels.
	dpmm := float64(targetW) / cw
	if alt := float64(targetH) / ch; alt > 0 && (dpmm <= 0 || math.Abs(alt-dpmm) > 1e-6) {
		// Prefer the resolution that keeps the longer edge within target.
		if float64(targetW) >= float64(targetH) {
			dpmm = float64(targetW) / cw
		} else {
			dpmm = float64(targetH) / ch
		}
	}
	if dpmm <= 0 {
		dpmm = 96.0 / 25.4
	}

	img := rasterizer.Draw(c, canvas.DPMM(dpmm), nil)
	bounds := img.Bounds()
	pw, ph := bounds.Dx(), bounds.Dy()
	if pw < 1 || ph < 1 {
		return nil, 0, 0, fmt.Errorf("svg canvas: zero pixel size")
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), pw, ph, nil
}

// svgCSSPixelSize returns the target raster size in CSS pixels (capped by
// maxSide), derived from the root SVG viewBox or width/height attributes.
// Only the root element is scanned; no shape parsing.
func svgCSSPixelSize(data []byte, maxSide int) (int, int) {
	vw, vh := rootSVGSize(data)
	if vw <= 0 {
		vw = 100
	}
	if vh <= 0 {
		vh = 100
	}
	scale := 1.0
	if vw > float64(maxSide) || vh > float64(maxSide) {
		scale = float64(maxSide) / math.Max(vw, vh)
	}
	pw := int(math.Ceil(vw * scale))
	ph := int(math.Ceil(vh * scale))
	if pw < 1 {
		pw = 1
	}
	if ph < 1 {
		ph = 1
	}
	return pw, ph
}

// rootSVGSize reads viewBox / width / height from the first <svg> start tag.
func rootSVGSize(data []byte) (vw, vh float64) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	dec.AutoClose = xml.HTMLAutoClose
	dec.Entity = xml.HTMLEntity
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, 0
		}
		el, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		if strings.ToLower(el.Name.Local) != "svg" {
			continue
		}
		attrs := map[string]string{}
		for _, a := range el.Attr {
			attrs[strings.ToLower(a.Name.Local)] = a.Value
		}
		width := parseLen(attrs["width"], 0)
		height := parseLen(attrs["height"], 0)
		if vb := attrs["viewbox"]; vb != "" {
			parts := splitNums(vb)
			if len(parts) >= 4 {
				vw, vh = parts[2], parts[3]
			}
		}
		if vw <= 0 {
			vw = width
		}
		if vh <= 0 {
			vh = height
		}
		return vw, vh
	}
	return 0, 0
}

func looksLikeSVG(data []byte) bool {
	s := strings.TrimSpace(string(data))
	if strings.HasPrefix(s, "\xef\xbb\xbf") {
		s = strings.TrimSpace(s[3:])
	}
	low := strings.ToLower(s)
	return strings.Contains(low, "<svg") || strings.HasPrefix(low, "<?xml")
}

func parseLen(s string, def float64) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	s = strings.TrimSuffix(s, "px")
	s = strings.TrimSuffix(s, "pt")
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return def
	}
	return f
}

func splitNums(s string) []float64 {
	s = strings.ReplaceAll(s, ",", " ")
	var out []float64
	for _, p := range strings.Fields(s) {
		f, err := strconv.ParseFloat(p, 64)
		if err == nil {
			out = append(out, f)
		}
	}
	return out
}
