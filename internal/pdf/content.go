package pdf

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// Content builds a page content stream. Every emitted operator is recorded
// as a string; fonts and images used are registered for the page /Resources.
type Content struct {
	buf       bytes.Buffer
	fontUses  map[string]string // resource name -> font object ref (allocated at finalize)
	fontFiles map[string]*Font  // resource name -> parsed font (embedded)
	used      map[string][]rune // resource name -> runes seen
	curFont   string            // active font from last SetFont
	curSize   float64           // active font size from last SetFont
	imageUses map[string]string // resource name -> image object ref
	imageRefs map[string]*imageResource
	opacity   float64 // 0 disables
	doc       *Document
}

type imageResource struct {
	ref    objRef // indirect object ref (allocated in AddJPEGImage/AddPNGImage)
	width  int
	height int
}

// NewContent creates an empty content stream builder.
func NewContent() *Content {
	return &Content{ //nolint:exhaustruct // intentional zero-value fields
		fontUses:  map[string]string{},
		fontFiles: map[string]*Font{},
		used:      map[string][]rune{},
		imageUses: map[string]string{},
		imageRefs: map[string]*imageResource{},
	}
}

// Bytes returns the raw (uncompressed) content stream.
func (c *Content) Bytes() []byte { return c.buf.Bytes() }

func appendPDFNum(dst []byte, val float64) []byte {
	if val == float64(int(val)) {
		return strconv.AppendInt(dst, int64(int(val)), pdfNumBase)
	}

	const float64Bits = 64

	dst = strconv.AppendFloat(dst, val, 'f', pdfFloatPrec, float64Bits)
	for len(dst) > 0 && dst[len(dst)-1] == '0' {
		dst = dst[:len(dst)-1]
	}

	if len(dst) > 0 && dst[len(dst)-1] == '.' {
		dst = dst[:len(dst)-1]
	}

	return dst
}

// writePDFNums appends a short numeric operator directly into the content
// buffer. Keeping formatting on the stack avoids the temporary strings that
// fmt.Sprintf and num used to create for every path and text coordinate.
func (c *Content) writePDFNums(suffix string, count int, num1, num2, num3, num4, num5, num6 float64) {
	out := c.buf.AvailableBuffer()
	out = appendPDFNum(out, num1)

	if count > 1 {
		out = append(out, ' ')
		out = appendPDFNum(out, num2)
	}

	if count > pointComponents {
		out = append(out, ' ')
		out = appendPDFNum(out, num3)
	}

	if count > numArgsMin3 {
		out = append(out, ' ')
		out = appendPDFNum(out, num4)
	}

	if count > numArgsMin4 {
		out = append(out, ' ')
		out = appendPDFNum(out, num5)
	}

	if count > numArgsMin5 {
		out = append(out, ' ')
		out = appendPDFNum(out, num6)
	}

	out = append(out, suffix...)
	_, _ = c.buf.Write(out)
}

// cloneContent returns a copy of c that paints the same operators. Resource
// maps are copied rather than aliased: a duplicated page may add a resource
// after cloning and must not mutate the source page's resource dictionary.
// The parsed fonts themselves are immutable, so their pointers remain shared.
// A fresh bytes.Buffer is used because Buffer values must not be copied after
// use.
func cloneContent(cur *Content) *Content {
	ncVal := &Content{ //nolint:exhaustruct // intentional zero-value fields
		fontUses:  cloneStringMap(cur.fontUses),
		fontFiles: cloneFontMap(cur.fontFiles),
		used:      cloneRuneMap(cur.used),
		curFont:   cur.curFont,
		curSize:   cur.curSize,
		imageUses: cloneStringMap(cur.imageUses),
		imageRefs: cloneImageMap(cur.imageRefs),
		opacity:   cur.opacity,
		doc:       cur.doc,
	}
	ncVal.buf.Write(cur.buf.Bytes())

	return ncVal
}

func cloneStringMap(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}

	return dst
}

func cloneFontMap(src map[string]*Font) map[string]*Font {
	dst := make(map[string]*Font, len(src))
	for k, v := range src {
		dst[k] = v
	}

	return dst
}

func cloneRuneMap(src map[string][]rune) map[string][]rune {
	dst := make(map[string][]rune, len(src))
	for k, v := range src {
		dst[k] = append([]rune(nil), v...)
	}

	return dst
}

func cloneImageMap(src map[string]*imageResource) map[string]*imageResource {
	dst := make(map[string]*imageResource, len(src))

	for name, res := range src {
		if res == nil {
			dst[name] = nil

			continue
		}

		cloned := *res
		dst[name] = &cloned
	}

	return dst
}

// graphics state

// Save restores the graphics state stack.
func (c *Content) Save() { c.buf.WriteString("q\n") }

// Restore pops the graphics state stack.
func (c *Content) Restore() { c.buf.WriteString("Q\n") }

// SetFillColor sets the fill color (RGB, 0..1); grayscale is applied at this
// paint-time seam, which is what Document.SetGrayscale promises today.
func (c *Content) SetFillColor(red, green, blue float64) {
	if c.doc != nil && c.doc.grayscale {
		v := lumaR*red + lumaG*green + lumaB*blue // Rec.601 luma
		red, green, blue = v, v, v
	}

	c.writePDFNums(" rg\n", rgbComponents, red, green, blue, 0, 0, 0)
}

// SetStrokeColor sets the stroke color (RGB, 0..1); grayscale is applied at
// this paint-time seam, same fold as SetFillColor.
func (c *Content) SetStrokeColor(red, green, blue float64) {
	if c.doc != nil && c.doc.grayscale {
		v := lumaR*red + lumaG*green + lumaB*blue // Rec.601 luma
		red, green, blue = v, v, v
	}

	c.writePDFNums(" RG\n", rgbComponents, red, green, blue, 0, 0, 0)
}

// SetLineWidth sets the stroked line width in points.
func (c *Content) SetLineWidth(w float64) {
	c.writePDFNums(" w\n", 1, w, 0, 0, 0, 0, 0)
}

// SetOpacity sets the fill/stroke opacity (0..1); 0 resets to opaque.
func (c *Content) SetOpacity(opacity float64) {
	if opacity >= 1 || opacity <= 0 {
		c.opacity = 0

		return
	}

	c.opacity = opacity
	c.buf.WriteString("/opacity gs\n")
}

// paths

// MoveTo begins a new subpath at (x, y).
func (c *Content) MoveTo(x, y float64) {
	c.writePDFNums(" m\n", pointComponents, x, y, 0, 0, 0, 0)
}

// LineTo appends a line segment to (x, y).
func (c *Content) LineTo(x, y float64) {
	c.writePDFNums(" l\n", pointComponents, x, y, 0, 0, 0, 0)
}

// Rect appends a rectangle to the current path.
func (c *Content) Rect(x, y, w, h float64) {
	c.writePDFNums(" re\n", rectComponents, x, y, w, h, 0, 0)
}

// Fill paints the current path with the fill color.
func (c *Content) Fill() { c.buf.WriteString("f\n") }

// Stroke strokes the current path.
func (c *Content) Stroke() { c.buf.WriteString("S\n") }

// Clip sets the current path as the clipping region (non-zero winding).
func (c *Content) Clip() { c.buf.WriteString("W n\n") }

// coordinates

// Transform appends a 6-element matrix to the CTM.
func (c *Content) Transform(a, b, c2, d, e, f float64) {
	c.writePDFNums(" cm\n", matrixComponents, a, b, c2, d, e, f)
}

// text

// SetFont selects a registered font resource by name and size.
func (c *Content) SetFont(name string, size float64) {
	c.curFont = name
	c.curSize = size
	_ = c.buf.WriteByte('/')
	c.buf.WriteString(name)
	_ = c.buf.WriteByte(' ')
	c.writePDFNums(" Tf\n", 1, size, 0, 0, 0, 0, 0)
}

// UseEmbeddedFont registers a parsed TTF under a resource name. Runes drawn
// with this font are subset-embedded into the PDF.
func (c *Content) UseEmbeddedFont(name string, f *Font) {
	c.fontFiles[name] = f
	if c.fontUses[name] == "" {
		c.fontUses[name] = ""
	}
}

// BeginText starts a text object.
func (c *Content) BeginText() { c.buf.WriteString("BT\n") }

// EndText ends a text object.
func (c *Content) EndText() { c.buf.WriteString("ET\n") }

// TextAt sets the text position.
func (c *Content) TextAt(x, y float64) {
	c.writePDFNums(" Td\n", pointComponents, x, y, 0, 0, 0, 0)
}

// TextMatrix sets the text matrix via Tm (a b c d e f).
func (c *Content) TextMatrix(a, b, cc, d, e, f float64) {
	c.writePDFNums(" Tm\n", matrixComponents, a, b, cc, d, e, f)
}

// TextLeading sets the leading for TL/T*.
func (c *Content) TextLeading(leading float64) {
	c.writePDFNums(" TL\n", 1, leading, 0, 0, 0, 0, 0)
}

// TextNextLine moves to the next line (TL).
func (c *Content) TextNextLine() { c.buf.WriteString("T*\n") }

// TextRenderMode sets the text rendering mode (0 = fill, 2 = fill + stroke).
// Mode 2 with a small line width yields a fake bold.
func (c *Content) TextRenderMode(mode int) {
	c.buf.WriteString(strconv.Itoa(mode) + " Tr\n")
}

// textRun is one contiguous simple-or-Type0 segment of a shown string.
type textRun struct {
	s     string
	type0 bool
}

// TextShow draws a string in the current font, recording its runes for the
// subsetter. Mixed CJK+Latin is split: Unicode glyphs the face provides go
// through Type0; Latin that the face lacks (typical for CJK fallback fonts)
// is drawn with an embedded Liberation fallback so ASCII does not become tofu.
func (c *Content) TextShow(text string) {
	fnt := c.fontFiles[c.curFont]
	text = ShapeRun(text, fnt, c.curSize).Text

	if fnt == nil || !c.textNeedsType0(text) {
		c.textShowSimple(text)

		return
	}

	c.emitTextRuns(splitType0Runs(text, fnt))

	base := strings.TrimSuffix(c.curFont, "_u")
	if c.curFont != base {
		c.SetFont(base, c.curSize)
	}
}

// splitType0Runs breaks s into simple vs Type0 segments: codes above
// Latin-1 take the Type0 path, missing Latin glyphs on a CJK face fall back
// to the Liberation face.
func splitType0Runs(text string, fnt *Font) []textRun {
	var runs []textRun

	var buf strings.Builder

	mode := -1 // -1 unset, 0 simple, 1 type0
	flush := func() {
		if buf.Len() == 0 {
			return
		}

		runs = append(runs, textRun{s: buf.String(), type0: mode == 1})
		buf.Reset()
	}

	for _, rVal := range text {
		has := fnt.GlyphID(rVal) != 0

		next := 0
		if rVal > maxLatin1Code {
			next = 1
		} else if !has {
			next = 0 // missing Latin on CJK face → Liberation
		}

		if mode < 0 {
			mode = next
		} else if next != mode {
			flush()

			mode = next
		}

		buf.WriteRune(rVal)
	}

	flush()

	return runs
}

// emitTextRuns paints each run, keeping the caller's face as the Type0
// source. Latin fallback may switch curFont to FL; Type0 must still subset
// the original Unicode face, not FL_u.
func (c *Content) emitTextRuns(runs []textRun) {
	base := strings.TrimSuffix(c.curFont, "_u")
	size := c.curSize

	for _, runVal := range runs {
		if runVal.type0 {
			if c.curFont != base && c.curFont != base+"_u" {
				c.SetFont(base, size)
			}

			c.textShowType0(runVal.s)

			continue
		}

		name := c.runFallbackFont(base, runVal.s)

		if c.curFont != name {
			c.SetFont(name, size)
		}

		c.textShowSimple(runVal.s)
	}
}

// runFallbackFont picks the Liberation fallback when the base face lacks a
// glyph in the run.
func (c *Content) runFallbackFont(base, s string) string {
	if face := c.fontFiles[base]; face != nil {
		for _, r := range s {
			if face.GlyphID(r) == 0 {
				return c.ensureLatinFallback()
			}
		}
	}

	return base
}

func (c *Content) ensureLatinFallback() string {
	const name = "FL"
	if c.fontFiles[name] != nil {
		return name
	}

	lf, err := DefaultFont()
	if err != nil || lf == nil {
		return c.curFont
	}

	c.UseEmbeddedFont(name, lf)

	return name
}

func (c *Content) textShowSimple(str string) {
	for _, rVal := range str {
		if rVal > maxLatin1Code {
			rVal = winAnsiFold(rVal)
		}

		if rVal > maxLatin1Code {
			rVal = '?'
		}

		c.used[c.curFont] = append(c.used[c.curFont], rVal)
	}

	c.buf.WriteString(pdfString(str) + " Tj\n")
}

func (c *Content) textNeedsType0(s string) bool {
	if _, ok := c.fontFiles[c.curFont]; !ok {
		return false
	}

	for _, r := range s {
		if r > maxLatin1Code {
			r = winAnsiFold(r)
		}

		if r > maxLatin1Code {
			return true
		}
	}

	return false
}

func (c *Content) textShowType0(str string) {
	base := strings.TrimSuffix(c.curFont, "_u")
	uname := base + "_u"

	if f := c.fontFiles[base]; f != nil {
		c.UseEmbeddedFont(uname, f)
	} else if f := c.fontFiles[c.curFont]; f != nil && c.curFont == uname {
		c.UseEmbeddedFont(uname, f)
	}

	if c.curFont != uname {
		c.SetFont(uname, c.curSize)
	}

	for _, r := range str {
		if r > maxBMPCode {
			r = '?'
		}

		c.used[uname] = append(c.used[uname], r)
	}

	c.buf.WriteString(pdfHexCIDs(str) + " Tj\n")
}

// resources

// fonts returns the map of font resource name to object ref, allocating the
// font objects and their dicts lazily. Embedded fonts are subset for the
// runes used on this content. A font whose subset fails the page: text
// that names a missing /Resources entry renders invisible, so the error is
// propagated instead of dropped.
func (c *Content) fonts() (map[string]string, error) {
	out := map[string]string{}

	for name := range c.fontUses {
		f, ok := c.fontFiles[name]
		if !ok {
			continue
		}

		ref, err := c.doc.ensureFont(f, c.used[name])
		if err != nil {
			return nil, fmt.Errorf("embed font %s: %w", name, err)
		}

		c.fontUses[name] = ref.String()
		out[name] = ref.String()
	}

	return out, nil
}

// imageResources returns the map of image resource name to object ref.
// JPEG/PNG paths allocate the XObject eagerly in AddJPEGImage/AddPNGImage.
func (c *Content) imageResources() map[string]string {
	out := map[string]string{}

	for name, img := range c.imageRefs {
		if img == nil || img.ref == 0 {
			continue
		}

		out[name] = img.ref.String()
	}

	return out
}

// extGState returns the ExtGState dict for the page resources ("" when none).
func (c *Content) extGState() string {
	if c.opacity > 0 {
		return fmt.Sprintf("/ExtGState << /opacity << /CA %s /ca %s >> >>", num(c.opacity), num(c.opacity))
	}

	return ""
}
