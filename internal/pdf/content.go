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
	return &Content{
		fontUses:  map[string]string{},
		fontFiles: map[string]*Font{},
		used:      map[string][]rune{},
		imageUses: map[string]string{},
		imageRefs: map[string]*imageResource{},
	}
}

// Bytes returns the raw (uncompressed) content stream.
func (c *Content) Bytes() []byte { return c.buf.Bytes() }

// cloneContent returns a copy of c that paints the same operators. Resource
// maps are copied rather than aliased: a duplicated page may add a resource
// after cloning and must not mutate the source page's resource dictionary.
// The parsed fonts themselves are immutable, so their pointers remain shared.
// A fresh bytes.Buffer is used because Buffer values must not be copied after
// use.
func cloneContent(c *Content) *Content {
	nc := &Content{
		fontUses:  cloneStringMap(c.fontUses),
		fontFiles: cloneFontMap(c.fontFiles),
		used:      cloneRuneMap(c.used),
		curFont:   c.curFont,
		curSize:   c.curSize,
		imageUses: cloneStringMap(c.imageUses),
		imageRefs: cloneImageMap(c.imageRefs),
		opacity:   c.opacity,
		doc:       c.doc,
	}
	nc.buf.Write(c.buf.Bytes())
	return nc
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
	for k, v := range src {
		if v == nil {
			dst[k] = nil
			continue
		}
		copy := *v
		dst[k] = &copy
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
func (c *Content) SetFillColor(r, g, b float64) {
	if c.doc != nil && c.doc.grayscale {
		v := 0.299*r + 0.587*g + 0.114*b // Rec.601 luma
		r, g, b = v, v, v
	}
	c.buf.WriteString(fmt.Sprintf("%s %s %s rg\n", num(r), num(g), num(b)))
}

// SetStrokeColor sets the stroke color (RGB, 0..1); grayscale is applied at
// this paint-time seam, same fold as SetFillColor.
func (c *Content) SetStrokeColor(r, g, b float64) {
	if c.doc != nil && c.doc.grayscale {
		v := 0.299*r + 0.587*g + 0.114*b // Rec.601 luma
		r, g, b = v, v, v
	}
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

// Stroke strokes the current path.
func (c *Content) Stroke() { c.buf.WriteString("S\n") }

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
	c.curSize = size
	c.buf.WriteString(fmt.Sprintf("/%s %s Tf\n", name, num(size)))
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

// TextMatrix sets the text matrix via Tm (a b c d e f).
func (c *Content) TextMatrix(a, b, cc, d, e, f float64) {
	c.buf.WriteString(fmt.Sprintf("%s %s %s %s %s %s Tm\n", num(a), num(b), num(cc), num(d), num(e), num(f)))
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
// subsetter. Mixed CJK+Latin is split: Unicode glyphs the face provides go
// through Type0; Latin that the face lacks (typical for CJK fallback fonts)
// is drawn with an embedded Liberation fallback so ASCII does not become tofu.
func (c *Content) TextShow(s string) {
	f := c.fontFiles[c.curFont]
	s = ShapeRun(s, f, c.curSize).Text
	if f == nil || !c.textNeedsType0(s) {
		c.textShowSimple(s)
		return
	}
	type run struct {
		s     string
		type0 bool
	}
	var runs []run
	var buf strings.Builder
	mode := -1 // -1 unset, 0 simple, 1 type0
	flush := func() {
		if buf.Len() == 0 {
			return
		}
		runs = append(runs, run{s: buf.String(), type0: mode == 1})
		buf.Reset()
	}
	for _, r := range s {
		has := f.GlyphID(r) != 0
		next := 0
		if r > 0xFF {
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
		buf.WriteRune(r)
	}
	flush()
	// Keep the caller's face as the Type0 source. Latin fallback may switch
	// curFont to FL; Type0 must still subset the original Unicode face, not FL_u.
	base := strings.TrimSuffix(c.curFont, "_u")
	size := c.curSize
	for _, rn := range runs {
		if rn.type0 {
			if c.curFont != base && c.curFont != base+"_u" {
				c.SetFont(base, size)
			}
			c.textShowType0(rn.s)
			continue
		}
		name := base
		if face := c.fontFiles[base]; face != nil {
			for _, r := range rn.s {
				if face.GlyphID(r) == 0 {
					name = c.ensureLatinFallback()
					break
				}
			}
		}
		if c.curFont != name {
			c.SetFont(name, size)
		}
		c.textShowSimple(rn.s)
	}
	if c.curFont != base {
		c.SetFont(base, size)
	}
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

func (c *Content) textShowSimple(s string) {
	for _, r := range s {
		if r > 0xFF {
			r = winAnsiFold(r)
		}
		if r > 0xFF {
			r = '?'
		}
		c.used[c.curFont] = append(c.used[c.curFont], r)
	}
	c.buf.WriteString(pdfString(s) + " Tj\n")
}

func (c *Content) textNeedsType0(s string) bool {
	if _, ok := c.fontFiles[c.curFont]; !ok {
		return false
	}
	for _, r := range s {
		if r > 0xFF {
			r = winAnsiFold(r)
		}
		if r > 0xFF {
			return true
		}
	}
	return false
}

func (c *Content) textShowType0(s string) {
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
	for _, r := range s {
		if r > 0xFFFF {
			r = '?'
		}
		c.used[uname] = append(c.used[uname], r)
	}
	c.buf.WriteString(pdfHexCIDs(s) + " Tj\n")
}

// resources

// fonts returns the map of font resource name to object ref, allocating the
// font objects and their dicts lazily. Embedded fonts are subset for the
// runes used on this content. A font whose subset fails fails the page: text
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
		if img.ref != 0 {
			out[name] = img.ref.String()
		}
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
