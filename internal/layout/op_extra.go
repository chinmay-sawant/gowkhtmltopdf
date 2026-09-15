package layout

import "github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"

// emptyExtra is the shared zero extra for ops that never set a rare payload.
// Writers must detach before mutating so the singleton stays zero.
var emptyExtra = &opExtra{} //nolint:exhaustruct,gochecknoglobals // immutable zero singleton shared by all ops

// opExtra holds display-list payloads that are empty on the report fixture
// (no images, URIs, transforms, or structure tags on typical fill/text/grid
// ops). Embedding it keeps field names (op.URI, op.Image) while the hot Op
// record stays at 256 bytes.
type opExtra struct {
	URI   string
	Image []byte
	// Src is the fetch source of Image. Synthetic rasters (gradients, inline
	// SVG, border-image slices) leave it empty; paint then labels the op
	// "inline" instead of naming a fetched resource.
	Src           string
	ImgW, ImgH    int
	Alt           string
	Xform         Matrix2D
	BlendMode     string
	PaintOpacity  float64
	StructElem    *pdf.StructElem
	TextTransform string
	TextLanguage  string
	// BlendGroup is the owning CSS element group (mix-blend-mode or
	// isolation: isolate). GroupMark flags begin/end boundary markers that
	// carry the group without painting.
	BlendGroup *BlendGroup
	GroupMark  uint8
	// ZChain is the op's innermost stacking-context frame (nil in the root
	// context). paint_order.go compares these chains so ancestor chrome
	// paints below descendant content.
	ZChain *paintCtxFrame
	// IsChrome marks background/border/shadow ops emitted by prependChrome.
	IsChrome bool
	// ChromeDepth counts the stacking contexts active when the chrome was
	// emitted (0 for root-level chrome and for every non-chrome op).
	ChromeDepth int
}

// setZChain records the op's innermost stacking-context frame. Root-context
// ops keep a nil chain.
func (op *Op) setZChain(frame *paintCtxFrame) {
	if frame == nil {
		return
	}

	op.detachExtra().ZChain = frame
}

// zChain returns the op's innermost stacking-context frame, or nil when the
// op was emitted in the root context.
func (op Op) zChain() *paintCtxFrame {
	if op.opExtra == nil {
		return nil
	}

	return op.opExtra.ZChain
}

// setChrome marks the op as element chrome and records the stacking-context
// depth it was emitted at for diagnostics.
func (op *Op) setChrome(depth int) {
	extra := op.detachExtra()
	extra.IsChrome = true
	extra.ChromeDepth = depth
}

// isChrome reports whether the op is element chrome (background, border, or
// box shadow emitted by prependChrome).
func (op Op) isChrome() bool {
	return op.opExtra != nil && op.opExtra.IsChrome
}

// chromeDepth returns the stacked-context depth recorded for chrome ops; 0
// for non-chrome ops and root-level chrome.
func (op Op) chromeDepth() int {
	if op.opExtra == nil {
		return 0
	}

	return op.opExtra.ChromeDepth
}

func (op *Op) detachExtra() *opExtra {
	if op.opExtra == nil || op.opExtra == emptyExtra {
		extra := new(opExtra)
		op.opExtra = extra

		return extra
	}

	return op.opExtra
}

func (op Op) detachedExtraCopy() *opExtra {
	if op.opExtra == nil || op.opExtra == emptyExtra {
		return new(opExtra)
	}

	cp := *op.opExtra
	cp.Image = append([]byte(nil), cp.Image...)

	return &cp
}

func (op *Op) bindEmptyExtra() {
	if op.opExtra == nil {
		op.opExtra = emptyExtra
	}
}

// BindEmptyExtra points a nil extra at the shared zero extra so readers in
// other packages can load BlendMode, Xform, and Image without panicking.
//
//nolint:stylecheck // Op methods already mix op/paintOp receivers
func (op *Op) BindEmptyExtra() { op.bindEmptyExtra() }

func (op *Op) setURI(uri string) {
	if uri == "" {
		if op.opExtra == nil || op.opExtra == emptyExtra {
			return
		}

		op.detachExtra().URI = ""

		return
	}

	op.detachExtra().URI = uri
}

// SetURI writes the link target, allocating a unique extra if needed.
func (op *Op) SetURI(uri string) { op.setURI(uri) }

// SetBlendMode writes the CSS blend mode, allocating a unique extra if needed.
func (op *Op) SetBlendMode(mode string) { op.setBlendMode(mode) }

// SetXform writes the baked CSS transform and marks it set.
func (op *Op) SetXform(matrix Matrix2D) {
	op.setXform(matrix)
	op.XformSet = true
}

// SetTextTransform writes the CSS text-transform, allocating a unique extra if needed.
func (op *Op) SetTextTransform(value string) { op.setTextTransform(value) }

// SetTextLanguage writes the CSS font-language-override tag, allocating a
// unique extra if needed. Empty and "normal" mean no shaping override.
func (op *Op) SetTextLanguage(value string) { op.setTextLanguage(value) }

// TextLanguage returns the op's OpenType language override, or "" when the op
// carries none. Ops that never bound an extra read as no override.
func (op Op) TextLanguage() string {
	if op.opExtra == nil {
		return ""
	}

	return op.opExtra.TextLanguage
}

// SetPaintOpacity writes element opacity, allocating a unique extra if needed.
func (op *Op) SetPaintOpacity(value float64) { op.setPaintOpacity(value) }

// Clone returns a copy whose extra payload does not alias op.
func (op Op) Clone() Op {
	op.opExtra = op.detachedExtraCopy()

	return op
}

func (op *Op) setImage(data []byte, width, height int, alt, src string) {
	extra := op.detachExtra()
	extra.Image = data
	extra.ImgW = width
	extra.ImgH = height
	extra.Alt = alt
	extra.Src = src
}

func (op *Op) setBlendMode(mode string) {
	if mode == "" || mode == blendNormal {
		if op.opExtra == nil || op.opExtra == emptyExtra {
			return
		}

		op.detachExtra().BlendMode = mode

		return
	}

	op.detachExtra().BlendMode = mode
}

func (op *Op) setTextTransform(value string) {
	if value == "" || value == textTransformNone {
		if op.opExtra == nil || op.opExtra == emptyExtra {
			return
		}

		op.detachExtra().TextTransform = value

		return
	}

	op.detachExtra().TextTransform = value
}

func (op *Op) setTextLanguage(value string) {
	if value == fontVariantNormal {
		value = ""
	}

	if value == "" {
		if op.opExtra == nil || op.opExtra == emptyExtra {
			return
		}

		op.detachExtra().TextLanguage = ""

		return
	}

	op.detachExtra().TextLanguage = value
}

func (op *Op) setPaintOpacity(value float64) {
	if value <= 0 || value >= 1 {
		if op.opExtra == nil || op.opExtra == emptyExtra {
			return
		}

		op.detachExtra().PaintOpacity = 0

		return
	}

	op.detachExtra().PaintOpacity = value
}

func (op *Op) setXform(matrix Matrix2D) {
	op.detachExtra().Xform = matrix
}

func (op *Op) setStructElem(elem *pdf.StructElem) {
	if elem == nil {
		if op.opExtra == nil || op.opExtra == emptyExtra {
			return
		}

		op.detachExtra().StructElem = nil

		return
	}

	op.detachExtra().StructElem = elem
}

func (op Op) withImage(data []byte, width, height int, alt, src string) Op {
	extra := op.detachedExtraCopy()
	extra.Image = data
	extra.ImgW = width
	extra.ImgH = height
	extra.Alt = alt
	extra.Src = src
	op.opExtra = extra

	return op
}

// ImageSrc returns the fetch source of the op's image payload, or "" when the
// payload was generated in-process (gradients, inline SVG, border slices).
func (op Op) ImageSrc() string {
	if op.opExtra == nil {
		return ""
	}

	return op.opExtra.Src
}

// imageSrcLogLimit caps a src echoed into warnings and embed errors so a
// data: URI cannot write a megabyte-scale log line.
const imageSrcLogLimit = 200

// truncateImageSrc caps src for log and error text.
func truncateImageSrc(src string) string {
	if len(src) <= imageSrcLogLimit {
		return src
	}

	return src[:imageSrcLogLimit] + "..."
}

func (op Op) withURI(uri string) Op {
	if uri == "" {
		return op
	}

	extra := op.detachedExtraCopy()
	extra.URI = uri
	op.opExtra = extra

	return op
}

func (op Op) withBlendMode(mode string) Op {
	if mode == "" || mode == blendNormal {
		return op
	}

	extra := op.detachedExtraCopy()
	extra.BlendMode = mode
	op.opExtra = extra

	return op
}

func (op Op) withTextTransform(value string) Op {
	if value == "" || value == textTransformNone {
		return op
	}

	extra := op.detachedExtraCopy()
	extra.TextTransform = value
	op.opExtra = extra

	return op
}

func (op Op) withTextLanguage(value string) Op {
	if value == "" || value == fontVariantNormal {
		return op
	}

	extra := op.detachedExtraCopy()
	extra.TextLanguage = value
	op.opExtra = extra

	return op
}

func cloneOpExtra(src *opExtra) *opExtra {
	if src == nil || src == emptyExtra {
		return src
	}

	cp := *src
	cp.Image = append([]byte(nil), src.Image...)
	cp.StructElem = nil

	return &cp
}
