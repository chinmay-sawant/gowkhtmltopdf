//nolint:varnamelen,funlen,cyclop,mnd,wsl,intrange,nlreturn,goconst,gocognit,dupl,unparam,exhaustive // filter
package layout

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"strconv"
	"strings"
)

type filterKind int

const (
	filterBlur filterKind = iota
	filterOpacity
	filterDropShadow
	filterGrayscale
	filterBrightness
	filterContrast
	filterInvert
	filterSepia
	filterSaturate
	filterHueRotate
)

type parsedFilter struct {
	kind       filterKind
	val        float64 // opacity, radius, amount, angle
	dropShadow parsedBoxShadow
}

func parseFilterList(value string, current [3]float64, fsize float64) []parsedFilter {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, cssDisplayNone) {
		return nil
	}

	var filters []parsedFilter
	rest := value

	for {
		rest = strings.TrimSpace(rest)
		if rest == "" {
			break
		}

		name, args, next, ok := splitTransformFunc(rest)
		if !ok {
			break
		}

		lowName := strings.ToLower(name)
		args = strings.TrimSpace(args)

		switch lowName {
		case "blur":
			if r, ok := plainLength(args, fsize, 0); ok && r >= 0 {
				filters = append(filters, parsedFilter{kind: filterBlur, val: r}) //nolint:exhaustruct
			}
		case "opacity":
			if op, ok := parseOpacityValue(args); ok {
				filters = append(filters, parsedFilter{kind: filterOpacity, val: op}) //nolint:exhaustruct
			}
		case "drop-shadow":
			if shadow, ok := parseBoxShadowLayer(args, current, fsize); ok {
				filters = append(filters, parsedFilter{kind: filterDropShadow, dropShadow: shadow}) //nolint:exhaustruct
			}
		case "grayscale":
			v := parseFilterAmount(args, 1.0)
			filters = append(filters, parsedFilter{kind: filterGrayscale, val: v}) //nolint:exhaustruct
		case "brightness":
			v := parseFilterAmount(args, 1.0)
			filters = append(filters, parsedFilter{kind: filterBrightness, val: v}) //nolint:exhaustruct
		case "contrast":
			v := parseFilterAmount(args, 1.0)
			filters = append(filters, parsedFilter{kind: filterContrast, val: v}) //nolint:exhaustruct
		case "invert":
			v := parseFilterAmount(args, 1.0)
			filters = append(filters, parsedFilter{kind: filterInvert, val: v}) //nolint:exhaustruct
		case "sepia":
			v := parseFilterAmount(args, 1.0)
			filters = append(filters, parsedFilter{kind: filterSepia, val: v}) //nolint:exhaustruct
		case "saturate":
			v := parseFilterAmount(args, 1.0)
			filters = append(filters, parsedFilter{kind: filterSaturate, val: v}) //nolint:exhaustruct
		case "hue-rotate":
			if deg, ok := parseAngleDeg(args); ok {
				filters = append(filters, parsedFilter{kind: filterHueRotate, val: deg}) //nolint:exhaustruct
			}
		}

		rest = next
	}

	return filters
}

func parseFilterAmount(args string, defaultVal float64) float64 {
	args = strings.TrimSpace(args)
	if args == "" {
		return defaultVal
	}
	if strings.HasSuffix(args, "%") {
		if v, err := strconv.ParseFloat(strings.TrimSuffix(args, "%"), 64); err == nil {
			return v / 100.0
		}
	}
	if v, err := strconv.ParseFloat(args, 64); err == nil {
		return v
	}
	return defaultVal
}

// buildGaussianKernel generates a 1D normalized Gaussian convolution kernel.
func buildGaussianKernel(radius float64) []float64 {
	sigma := math.Max(radius*0.57735, 0.5)
	kRadius := int(math.Ceil(3 * sigma))
	if kRadius > 25 {
		kRadius = 25
	}
	if kRadius < 1 {
		kRadius = 1
	}

	size := 2*kRadius + 1
	kernel := make([]float64, size)
	twoSigmaSq := 2 * sigma * sigma
	sum := 0.0

	for i := -kRadius; i <= kRadius; i++ {
		w := math.Exp(-float64(i*i) / twoSigmaSq)
		kernel[i+kRadius] = w
		sum += w
	}

	for i := range kernel {
		kernel[i] /= sum
	}

	return kernel
}

// applyGaussianBlur performs a 2D separable Gaussian blur on an RGBA image.
func applyGaussianBlur(src *image.NRGBA, radius float64) *image.NRGBA {
	if radius <= 0 || src == nil {
		return src
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 1 && h <= 1 {
		return src
	}

	kernel := buildGaussianKernel(radius)
	kRadius := (len(kernel) - 1) / 2

	// Horizontal pass
	temp := image.NewNRGBA(bounds)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var rSum, gSum, bSum, aSum float64
			for k := -kRadius; k <= kRadius; k++ {
				ix := x + k
				if ix < 0 {
					ix = 0
				} else if ix >= w {
					ix = w - 1
				}
				c := src.NRGBAAt(ix, y)
				weight := kernel[k+kRadius]
				rSum += float64(c.R) * weight
				gSum += float64(c.G) * weight
				bSum += float64(c.B) * weight
				aSum += float64(c.A) * weight
			}
			temp.SetNRGBA(x, y, color.NRGBA{
				R: uint8(math.Round(clamp01(rSum/255.0) * 255.0)),
				G: uint8(math.Round(clamp01(gSum/255.0) * 255.0)),
				B: uint8(math.Round(clamp01(bSum/255.0) * 255.0)),
				A: uint8(math.Round(clamp01(aSum/255.0) * 255.0)),
			})
		}
	}

	// Vertical pass
	dst := image.NewNRGBA(bounds)
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			var rSum, gSum, bSum, aSum float64
			for k := -kRadius; k <= kRadius; k++ {
				iy := y + k
				if iy < 0 {
					iy = 0
				} else if iy >= h {
					iy = h - 1
				}
				c := temp.NRGBAAt(x, iy)
				weight := kernel[k+kRadius]
				rSum += float64(c.R) * weight
				gSum += float64(c.G) * weight
				bSum += float64(c.B) * weight
				aSum += float64(c.A) * weight
			}
			dst.SetNRGBA(x, y, color.NRGBA{
				R: uint8(math.Round(clamp01(rSum/255.0) * 255.0)),
				G: uint8(math.Round(clamp01(gSum/255.0) * 255.0)),
				B: uint8(math.Round(clamp01(bSum/255.0) * 255.0)),
				A: uint8(math.Round(clamp01(aSum/255.0) * 255.0)),
			})
		}
	}

	return dst
}

func toNRGBA(img image.Image) *image.NRGBA {
	if nrgba, ok := img.(*image.NRGBA); ok {
		return nrgba
	}
	bounds := img.Bounds()
	nrgba := image.NewNRGBA(bounds)
	draw.Draw(nrgba, bounds, img, bounds.Min, draw.Src)
	return nrgba
}

// applyImageFilterToImage decodes PNG/JPEG, applies filters (blur, grayscale, etc.), and returns PNG bytes.
func applyImageFilterToImage(imgBytes []byte, filters []parsedFilter) []byte {
	if len(imgBytes) == 0 || len(filters) == 0 {
		return imgBytes
	}

	srcImg, _, err := image.Decode(bytes.NewReader(imgBytes))
	if err != nil {
		// try jpeg / png explicitly
		srcImg, err = png.Decode(bytes.NewReader(imgBytes))
		if err != nil {
			srcImg, err = jpeg.Decode(bytes.NewReader(imgBytes))
			if err != nil {
				return imgBytes
			}
		}
	}

	nrgba := toNRGBA(srcImg)
	bounds := nrgba.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	for _, f := range filters {
		switch f.kind {
		case filterBlur:
			nrgba = applyGaussianBlur(nrgba, f.val)
		case filterGrayscale:
			amount := clamp01(f.val)
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					c := nrgba.NRGBAAt(x, y)
					gray := 0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)
					r := float64(c.R)*(1-amount) + gray*amount
					g := float64(c.G)*(1-amount) + gray*amount
					b := float64(c.B)*(1-amount) + gray*amount
					nrgba.SetNRGBA(x, y, color.NRGBA{
						R: uint8(math.Round(r)),
						G: uint8(math.Round(g)),
						B: uint8(math.Round(b)),
						A: c.A,
					})
				}
			}
		case filterInvert:
			amount := clamp01(f.val)
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					c := nrgba.NRGBAAt(x, y)
					invR := 255.0 - float64(c.R)
					invG := 255.0 - float64(c.G)
					invB := 255.0 - float64(c.B)
					r := float64(c.R)*(1-amount) + invR*amount
					g := float64(c.G)*(1-amount) + invG*amount
					b := float64(c.B)*(1-amount) + invB*amount
					nrgba.SetNRGBA(x, y, color.NRGBA{
						R: uint8(math.Round(r)),
						G: uint8(math.Round(g)),
						B: uint8(math.Round(b)),
						A: c.A,
					})
				}
			}
		case filterOpacity:
			op := clamp01(f.val)
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					c := nrgba.NRGBAAt(x, y)
					nrgba.SetNRGBA(x, y, color.NRGBA{
						R: c.R,
						G: c.G,
						B: c.B,
						A: uint8(math.Round(float64(c.A) * op)),
					})
				}
			}
		}
	}

	var out bytes.Buffer
	if err := png.Encode(&out, nrgba); err != nil {
		return imgBytes
	}
	return out.Bytes()
}
