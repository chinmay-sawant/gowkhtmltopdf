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
	fontDefs  map[string]string // resource name -> base font name (base-14)
	fontFiles map[string]*Font  // resource name -> parsed font (embedded)
	used      map[string][]rune // resource name -> runes seen
	curFont   string            // active font from last SetFont
	imageUses map[string]string // resource name -> image object ref
	imageRefs map[string]*imageResource
	opacity   float64 // 0 disables
	doc       *Document
}

type imageResource struct {
	ref    string // indirect object ref (allocated lazily)
	width  int
	height int
	data   []byte // raw RGBA
}

// NewContent creates an empty content stream builder.
func NewContent() *Content {
	return &Content{
		fontUses:  map[string]string{},
		fontDefs:  map[string]string{},
		fontFiles: map[string]*Font{},
		used:      map[string][]rune{},
		imageUses: map[string]string{},
		imageRefs: map[string]*imageResource{},
	}
}

// Bytes returns the raw (uncompressed) content stream.
func (c *Content) Bytes() []byte { return c.buf.Bytes() }

// graphics state

// Save restores the graphics state stack.
func (c *Content) Save() { c.buf.WriteString("q\n") }

// Restore pops the graphics state stack.
func (c *Content) Restore() { c.buf.WriteString("Q\n") }

// SetFillColor sets the fill color (RGB, 0..1).
func (c *Content) SetFillColor(r, g, b float64) {
	c.buf.WriteString(fmt.Sprintf("%s %s %s rg\n", num(r), num(g), num(b)))
}

// SetStrokeColor sets the stroke color (RGB, 0..1).
func (c *Content) SetStrokeColor(r, g, b float64) {
	c.buf.WriteString(fmt.Sprintf("%s %s %s RG\n", num(r), num(g), num(b)))
}

// SetLineWidth sets the stroked line width in points.
func (c *Content) SetLineWidth(w float64) {
	c.buf.WriteString(num(w) + " w\n")
}

// SetOpacity sets the fill/stroke opacity (0..1); 0 resets to opaque.
func (c *Content) SetOpacity(a float64) {
	if a >= 1 || a <= 0 {
		c.opacity = 0
		return
	}
	c.opacity = a
	c.buf.WriteString("/opacity gs\n")
}

// paths

// MoveTo begins a new subpath at (x, y).
func (c *Content) MoveTo(x, y float64) {
	c.buf.WriteString(fmt.Sprintf("%s %s m\n", num(x), num(y)))
}

// LineTo appends a line segment to (x, y).
func (c *Content) LineTo(x, y float64) {
	c.buf.WriteString(fmt.Sprintf("%s %s l\n", num(x), num(y)))
}

// Rect appends a rectangle to the current path.
func (c *Content) Rect(x, y, w, h float64) {
	c.buf.WriteString(fmt.Sprintf("%s %s %s %s re\n", num(x), num(y), num(w), num(h)))
}

// Fill paints the current path with the fill color.
func (c *Content) Fill() { c.buf.WriteString("f\n") }

// FillStroke fills and strokes the current path.
func (c *Content) FillStroke() { c.buf.WriteString("B\n") }

// Stroke strokes the current path.
func (c *Content) Stroke() { c.buf.WriteString("S\n") }

// ClosePath closes the current subpath.
func (c *Content) ClosePath() { c.buf.WriteString("h\n") }

// Clip sets the current path as the clipping region (non-zero winding).
func (c *Content) Clip() { c.buf.WriteString("W n\n") }

// coordinates

// Transform appends a 6-element matrix to the CTM.
func (c *Content) Transform(a, b, c2, d, e, f float64) {
	c.buf.WriteString(fmt.Sprintf("%s %s %s %s %s %s cm\n",
		num(a), num(b), num(c2), num(d), num(e), num(f)))
}

// text

// SetFont selects a registered font resource by name and size.
func (c *Content) SetFont(name string, size float64) {
	c.curFont = name
	c.buf.WriteString(fmt.Sprintf("/%s %s Tf\n", name, num(size)))
}

// UseFont registers a base-14 font under a resource name for later SetFont.
func (c *Content) UseFont(name, baseFont string) {
	c.fontDefs[name] = baseFont
	if c.fontUses[name] == "" {
		c.fontUses[name] = ""
	}
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
	c.buf.WriteString(fmt.Sprintf("%s %s Td\n", num(x), num(y)))
}

// TextLeading sets the leading for TL/T*.
func (c *Content) TextLeading(leading float64) {
	c.buf.WriteString(num(leading) + " TL\n")
}

// TextNextLine moves to the next line (TL).
func (c *Content) TextNextLine() { c.buf.WriteString("T*\n") }

// TextRenderMode sets the text rendering mode (0 = fill, 2 = fill + stroke).
// Mode 2 with a small line width yields a fake bold.
func (c *Content) TextRenderMode(mode int) {
	c.buf.WriteString(strconv.Itoa(mode) + " Tr\n")
}

// TextShow draws a string in the current font, recording its runes for the
// subsetter.
func (c *Content) TextShow(s string) {
	for _, r := range s {
		c.used[c.curFont] = append(c.used[c.curFont], r)
	}
	c.buf.WriteString(pdfString(s) + " Tj\n")
}

// TextShowAdj draws text with per-char adjustments (kerning offsets in 1/1000 em).
func (c *Content) TextShowAdj(s string, kern []int) {
	if len(kern) == 0 {
		c.TextShow(s)
		return
	}
	for _, r := range s {
		c.used[c.curFont] = append(c.used[c.curFont], r)
	}
	var b strings.Builder
	b.WriteByte('[')
	b.WriteString(pdfString(s))
	for _, k := range kern {
		b.WriteString(" " + strconv.Itoa(-k))
	}
	b.WriteString("] TJ\n")
	c.buf.WriteString(b.String())
}

// TextRise sets the text rise.
func (c *Content) TextRise(rise float64) {
	c.buf.WriteString(num(rise) + " Ts\n")
}

// TextHorizScale sets the horizontal text scaling (percent).
func (c *Content) TextHorizScale(scale float64) {
	c.buf.WriteString(num(scale) + " Tz\n")
}

// TextWordSpacing sets word spacing.
func (c *Content) TextWordSpacing(s float64) {
	c.buf.WriteString(num(s) + " Tw\n")
}

// TextCharSpacing sets character spacing.
func (c *Content) TextCharSpacing(s float64) {
	c.buf.WriteString(num(s) + " Tc\n")
}

// images

// DrawImage embeds raw RGBA pixels as a Flate-compressed image XObject and
// paints it into the rect (x, y, w, h) in PDF coords.
func (c *Content) DrawImage(name string, x, y, w, h float64, rgba []byte, width, height int) {
	c.imageRefs[name] = &imageResource{width: width, height: height, data: rgba}
	c.imageUses[name] = "" // resolved at finalize
	c.Save()
	c.Transform(w, 0, 0, h, x, y)
	c.buf.WriteString("/" + name + " Do\n")
	c.Restore()
}

// resources

// fonts returns the map of font resource name to object ref, allocating the
// font objects and their dicts lazily. Embedded fonts are subset for the
// runes used on this content.
func (c *Content) fonts() map[string]string {
	out := map[string]string{}
	for name := range c.fontUses {
		if f, ok := c.fontFiles[name]; ok {
			ref, err := c.doc.ensureFont(f, c.used[name])
			if err != nil {
				continue // skip broken font, layout should have caught it
			}
			c.fontUses[name] = ref
		} else if c.fontUses[name] == "" {
			c.fontUses[name] = c.doc.newObject()
			c.doc.setDict(c.fontUses[name], "<< /Type /Font /Subtype /Type1 /BaseFont /"+c.fontDefs[name]+" >>")
		}
		out[name] = c.fontUses[name]
	}
	return out
}

// imageResources returns the map of image resource name to object ref,
// allocating refs and emitting the image XObject lazily.
func (c *Content) imageResources() map[string]string {
	out := map[string]string{}
	for name, img := range c.imageRefs {
		if img.ref == "" {
			raw := img.data
			dict := fmt.Sprintf("<< /Type /XObject /Subtype /Image /Width %d /Height %d /ColorSpace /DeviceRGB /BitsPerComponent 8",
				img.width, img.height)
			if c.doc.useCompression {
				raw = flateBytes(raw)
				dict += " /Filter /FlateDecode"
			}
			dict += fmt.Sprintf(" /Length %d >>", len(raw))
			img.ref = c.doc.newObject()
			c.doc.setDict(img.ref, dict)
			c.doc.setStream(img.ref, raw)
		}
		out[name] = img.ref
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
