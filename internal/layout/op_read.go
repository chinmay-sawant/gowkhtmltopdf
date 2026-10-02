package layout

// Nil-safe accessors for the opExtra payload. The fields on opExtra are
// promoted through an embedded pointer, so reading one directly on an op that
// never bound a rare payload dereferences a nil pointer. A display-list
// consumer outside this module cannot construct an opExtra to guard with, so
// every rare-payload read crosses one of these methods instead. They are the
// supported way for an external replay engine to read that payload.

// LinkURI returns the op's link target, or "" when it carries none.
func (op Op) LinkURI() string {
	if op.opExtra == nil {
		return ""
	}

	return op.opExtra.URI
}

// ImageBytes returns the op's encoded image payload with its pixel bounds.
// The slice aliases the op's buffer; read it, never write it. It returns nil
// for every op that is not an image.
func (op Op) ImageBytes() ([]byte, int, int) {
	if op.opExtra == nil {
		return nil, 0, 0
	}

	return op.opExtra.Image, op.opExtra.ImgW, op.opExtra.ImgH
}

// ImageAlt returns the op's alt text, or "" when it carries none.
func (op Op) ImageAlt() string {
	if op.opExtra == nil {
		return ""
	}

	return op.opExtra.Alt
}

// Transform returns the op's baked 2D transform. It returns the identity when
// the op has none; check XformSet to tell "identity" from "unset".
func (op Op) Transform() Matrix2D {
	if op.opExtra == nil {
		return IdentityMatrix()
	}

	return op.opExtra.Xform
}

// BlendModeName returns the op's CSS mix-blend-mode value, or "" for the
// default normal blending.
func (op Op) BlendModeName() string {
	if op.opExtra == nil {
		return ""
	}

	return op.opExtra.BlendMode
}

// Opacity returns the element opacity the op was painted with, folded with the
// op's own alpha exactly as the PDF and raster painters fold it.
//
// The engine stores 0 for "no opacity override" and treats only values
// strictly between 0 and 1 as a real opacity, so this reports the same
// effective number the painters used: 1 when nothing was set, and the product
// of the element opacity and the op alpha when both were.
func (op Op) Opacity() float64 {
	return pdfPaintOpacity(&op, true)
}

// Outline reports whether the op is a CSS outline operation. Outlines paint
// above descendant content, unlike ordinary backgrounds and borders.
func (op Op) Outline() bool {
	if op.opExtra == nil {
		return false
	}

	return op.opExtra.IsOutline
}

// TextTransformValue returns the CSS text-transform value the op carries, or
// "" when it carries none. It is spelled with a Value suffix rather than
// TextTransform because a method of that name would shadow the promoted
// opExtra field at the in-package call site in paint.go that reads it.
func (op Op) TextTransformValue() string {
	if op.opExtra == nil {
		return ""
	}

	return op.opExtra.TextTransform
}

// NoFakeBoldValue reports whether the op forbids synthesizing bold weight.
// The suffix keeps the accessor name distinct from the opExtra field of the
// same name, so the two read as one concept without one shadowing the other.
func (op Op) NoFakeBoldValue() bool {
	if op.opExtra == nil {
		return false
	}

	return op.opExtra.NoFakeBold
}

// Group and GroupBoundary already exist on *Op and are nil-safe, so a blend
// group is read through those rather than through a duplicate here.
