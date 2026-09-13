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
	URI           string
	Image         []byte
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

func (op *Op) setImage(data []byte, width, height int, alt string) {
	extra := op.detachExtra()
	extra.Image = data
	extra.ImgW = width
	extra.ImgH = height
	extra.Alt = alt
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

func (op Op) withImage(data []byte, width, height int, alt string) Op {
	extra := op.detachedExtraCopy()
	extra.Image = data
	extra.ImgW = width
	extra.ImgH = height
	extra.Alt = alt
	op.opExtra = extra

	return op
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
