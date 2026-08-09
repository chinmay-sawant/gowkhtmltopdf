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
	"errors"
	"fmt"
	"image/png"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/tdewolff/canvas"                      //nolint:depguard // sole SVG raster path, allowlisted (Makefile)
	"github.com/tdewolff/canvas/renderers/rasterizer" //nolint:depguard // sole SVG raster path, allowlisted (Makefile)
)

const (
	cssDPI          = 96.0
	mmPerInch       = 25.4
	viewBoxNumParts = 4
)

// Static errors returned by Rasterize; callers can match with errors.Is.
var (
	errNotSVG          = errors.New("svg: not SVG")
	errCanvasEmptySize = errors.New("svg canvas: empty size")
	errCanvasPanic     = errors.New("svg canvas: panic")
	errCanvasZeroPixel = errors.New("svg canvas: zero pixel size")
)

// Rasterize decodes SVG XML into a PNG image via tdewolff/canvas only.
// maxSide caps the longer edge in pixels (default 512).
// On failure (not SVG, parse/draw error, empty size, or canvas panic),
// returns err with nil pngBytes and zero w/h — callers must treat error
// as "no image". There is no second rasterizer or shell fallback.
func Rasterize(data []byte, maxSide int) ([]byte, int, int, error) {
	if maxSide <= 0 {
		maxSide = 512
	}

	if !looksLikeSVG(data) {
		return nil, 0, 0, errNotSVG
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
//
//nolint:nonamedreturns // defer-recover must override the result values
func rasterizeCanvas(data []byte, maxSide int) (pngBytes []byte, w, h int, err error) {
	defer func() {
		if r := recover(); r != nil {
			pngBytes, w, h = nil, 0, 0
			err = fmt.Errorf("%w: %v", errCanvasPanic, r)
		}
	}()

	svgCanvas, err := canvas.ParseSVG(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("svg canvas: %w", err)
	}

	canvasW, canvasH := svgCanvas.Size()
	if canvasW <= 0 || canvasH <= 0 {
		return nil, 0, 0, errCanvasEmptySize
	}

	// Intrinsic CSS-pixel size from viewBox / width / height attributes.
	targetW, targetH := svgCSSPixelSize(data, maxSide)
	dpmm := canvasDPMM(canvasW, canvasH, targetW, targetH)

	img := rasterizer.Draw(svgCanvas, canvas.DPMM(dpmm), nil)
	bounds := img.Bounds()

	pixW, pixH := bounds.Dx(), bounds.Dy()
	if pixW < 1 || pixH < 1 {
		return nil, 0, 0, errCanvasZeroPixel
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, 0, 0, fmt.Errorf("svg canvas: encode: %w", err)
	}

	return buf.Bytes(), pixW, pixH, nil
}

// canvasDPMM maps the canvas mm size to the target CSS-pixel size: it picks
// the resolution that keeps the longer edge within target, defaulting to CSS
// 96dpi when the mapping is degenerate.
func canvasDPMM(canvasW, canvasH float64, targetW, targetH int) float64 {
	dpmm := float64(targetW) / canvasW
	if alt := float64(targetH) / canvasH; alt > 0 && (dpmm <= 0 || math.Abs(alt-dpmm) > 1e-6) {
		// Prefer the resolution that keeps the longer edge within target.
		if float64(targetW) >= float64(targetH) {
			dpmm = float64(targetW) / canvasW
		} else {
			dpmm = float64(targetH) / canvasH
		}
	}

	if dpmm <= 0 {
		dpmm = cssDPI / mmPerInch
	}

	return dpmm
}

// svgCSSPixelSize returns the target raster size in CSS pixels (capped by
// maxSide), derived from the root SVG viewBox or width/height attributes.
// Only the root element is scanned; no shape parsing.
//
//nolint:cyclop,mnd // pixel size scaling with bounds check
func svgCSSPixelSize(data []byte, maxSide int) (int, int) {
	viewW, viewH := rootSVGSize(data)
	if math.IsNaN(viewW) || math.IsInf(viewW, 0) || viewW <= 0 {
		viewW = 100
	}

	if math.IsNaN(viewH) || math.IsInf(viewH, 0) || viewH <= 0 {
		viewH = 100
	}

	if maxSide <= 0 {
		maxSide = 512
	} else if maxSide > 4096 {
		maxSide = 4096
	}

	scale := 1.0
	if viewW > float64(maxSide) || viewH > float64(maxSide) {
		scale = float64(maxSide) / math.Max(viewW, viewH)
	}

	pixW := int(math.Ceil(viewW * scale))
	pixH := int(math.Ceil(viewH * scale))

	if pixW < 1 {
		pixW = 1
	} else if pixW > maxSide {
		pixW = maxSide
	}

	if pixH < 1 {
		pixH = 1
	} else if pixH > maxSide {
		pixH = maxSide
	}

	return pixW, pixH
}

// rootSVGSize reads viewBox / width / height from the first <svg> start tag.
func rootSVGSize(data []byte) (float64, float64) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	dec.AutoClose = xml.HTMLAutoClose
	dec.Entity = xml.HTMLEntity

	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return 0, 0
		}

		elem, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}

		if strings.ToLower(elem.Name.Local) != "svg" {
			continue
		}

		return svgSizeAttrs(elem)
	}

	return 0, 0
}

// svgSizeAttrs derives the intrinsic size from one root SVG element: the
// viewBox wins, falling back to width/height attributes (in CSS pixels).
func svgSizeAttrs(elem xml.StartElement) (float64, float64) {
	attrs := map[string]string{}
	for _, a := range elem.Attr {
		attrs[strings.ToLower(a.Name.Local)] = a.Value
	}

	width := parseLen(attrs["width"], 0)
	height := parseLen(attrs["height"], 0)

	sizeW, sizeH := width, height

	if vb := attrs["viewbox"]; vb != "" {
		parts := splitNums(vb)
		if len(parts) >= viewBoxNumParts {
			sizeW, sizeH = parts[2], parts[3]
		}
	}

	if sizeW <= 0 {
		sizeW = width
	}

	if sizeH <= 0 {
		sizeH = height
	}

	return sizeW, sizeH
}

func looksLikeSVG(data []byte) bool {
	s := strings.TrimSpace(string(data))
	if strings.HasPrefix(s, "\xef\xbb\xbf") {
		s = strings.TrimSpace(s[3:])
	}

	low := strings.ToLower(s)

	return strings.Contains(low, "<svg") || strings.HasPrefix(low, "<?xml")
}

func parseLen(raw string, def float64) float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return def
	}

	raw = strings.TrimSuffix(raw, "px")
	raw = strings.TrimSuffix(raw, "pt")

	f, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
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
