//nolint:all // inline SVG serialize+raster path
package layout

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/svg"
)

const cssTagSVG = "svg"

// buildInlineSVG rasterizes an inline <svg> subtree (with cascade presentation
// props baked into attributes). When paint is false, only size/imgRef are
// filled so the inline line placer can emit at the final position (same
// contract as buildImage); painting at (0,0) here left stray SVGs on the
// masthead and hid logo.png.
func (e *engine) buildInlineSVG(node *html.Node, sty ResolvedStyle, posX, posY float64, paint bool) *box {
	boxNode := &box{ //nolint:exhaustruct // intentional zero fields
		node: node, style: e.stylePtr(node), kind: "replaced", x: posX, y: posY,
	}

	data := e.serializeInlineSVG(node)
	ref := &imageRef{src: "#inline-svg"} //nolint:exhaustruct // synthetic raster
	if png, pw, ph, err := svg.Rasterize(data, 1024); err == nil && len(png) > 0 {
		ref.data, ref.w, ref.h = png, pw, ph
	}
	boxNode.img = ref

	size := e.usedInlineSVGSize(node, sty, ref)
	padL := e.scalePt(sty.PaddingLeft)
	padR := e.scalePt(sty.PaddingRight)
	padT := e.scalePt(sty.PaddingTop)
	padB := e.scalePt(sty.PaddingBottom)
	borderL := e.scalePt(borderPaint(sty.BorderLeft))
	borderR := e.scalePt(borderPaint(sty.BorderRight))
	borderT := e.scalePt(borderPaint(sty.BorderTop))
	borderB := e.scalePt(borderPaint(sty.BorderBottom))
	boxNode.w = size.w + padL + padR + borderL + borderR
	boxNode.height = size.h + padT + padB + borderT + borderB

	if paint && ref.data != nil && !e.noEmit {
		imgX := posX + borderL + padL
		imgY := posY + borderT + padT
		opStart := len(e.ops)
		e.add(Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpImage, X: imgX, Y: imgY, W: size.w, H: size.h,
			Image: ref.data, ImgW: ref.w, ImgH: ref.h,
		})
		e.prependChrome(opStart, boxNode, sty, posX, posY, boxNode.w, boxNode.height)
	}

	return boxNode
}

// usedInlineSVGSize prefers width/height attributes (CSS px), then the
// raster intrinsic size, then a small fallback.
func (e *engine) usedInlineSVGSize(node *html.Node, sty ResolvedStyle, ref *imageRef) imageUsedSize {
	// Attribute lengths are CSS px → pt, then zoomed like other style lengths.
	wAttr := e.scalePt(parseSVGLengthPx(node.Attribute("width")))
	hAttr := e.scalePt(parseSVGLengthPx(node.Attribute("height")))
	if sty.Width >= 0 {
		wAttr = e.scalePt(sty.Width)
	} else if sty.WidthPercent >= 0 && e.opts.Width > 0 {
		wAttr = e.opts.Width * sty.WidthPercent / 100
	}
	if sty.Height >= 0 {
		hAttr = e.scalePt(sty.Height)
	}

	intrW, intrH := 0.0, 0.0
	if ref != nil && ref.w > 0 && ref.h > 0 {
		intrW = e.scalePt(pxToPt(float64(ref.w)))
		intrH = e.scalePt(pxToPt(float64(ref.h)))
	}
	if wAttr <= 0 {
		wAttr = intrW
	}
	if hAttr <= 0 {
		hAttr = intrH
	}
	if wAttr <= 0 {
		wAttr = e.scalePt(pxToPt(64))
	}
	if hAttr <= 0 {
		hAttr = e.scalePt(pxToPt(28))
	}

	return imageUsedSize{w: wAttr, h: hAttr}
}

func parseSVGLengthPx(raw string) float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	raw = strings.TrimSuffix(raw, "px")
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 {
		return 0
	}

	return pxToPt(v)
}

// serializeInlineSVG emits SVG XML for canvas rasterization, baking resolved
// fill/stroke presentation props onto each element as attributes.
func (e *engine) serializeInlineSVG(node *html.Node) []byte {
	var b strings.Builder
	e.writeSVGNode(&b, node)

	return []byte(b.String())
}

func (e *engine) writeSVGNode(b *strings.Builder, node *html.Node) {
	if node == nil {
		return
	}
	switch node.Type {
	case html.TextNode:
		b.WriteString(escapeXML(node.Text))
	case html.ElementNode:
		b.WriteByte('<')
		b.WriteString(node.Name)
		written := map[string]bool{}
		for k, v := range node.Attrs {
			if k == "style" {
				continue
			}
			b.WriteByte(' ')
			b.WriteString(k)
			b.WriteString(`="`)
			b.WriteString(escapeXML(v))
			b.WriteByte('"')
			written[k] = true
		}
		if node.Name == cssTagSVG && !written["xmlns"] {
			b.WriteString(` xmlns="http://www.w3.org/2000/svg"`)
		}
		e.writeSVGPresentationAttrs(b, node, written)
		if len(node.Children) == 0 {
			b.WriteString("/>")
			return
		}
		b.WriteByte('>')
		for _, c := range node.Children {
			e.writeSVGNode(b, c)
		}
		b.WriteString("</")
		b.WriteString(node.Name)
		b.WriteByte('>')
	}
}

func (e *engine) writeSVGPresentationAttrs(b *strings.Builder, node *html.Node, written map[string]bool) {
	st := e.styles[node]
	if st == nil {
		return
	}
	if st.FillSet && !written["fill"] {
		if st.FillOpacity == 0 && st.Fill == [3]float64{} {
			b.WriteString(` fill="none"`)
		} else {
			fmt.Fprintf(b, ` fill="%s"`, cssColorHex(st.Fill))
		}
	}
	if st.FillOpacity >= 0 && st.FillOpacity < 1 && !written["fill-opacity"] {
		fmt.Fprintf(b, ` fill-opacity="%g"`, st.FillOpacity)
	}
	if st.StrokeSet && !written["stroke"] {
		fmt.Fprintf(b, ` stroke="%s"`, cssColorHex(st.Stroke))
	}
	if st.StrokeWidthSet && st.StrokeWidth > 0 && !written["stroke-width"] {
		fmt.Fprintf(b, ` stroke-width="%g"`, st.StrokeWidth)
	}
	if st.StrokeOpacity >= 0 && st.StrokeOpacity < 1 && !written["stroke-opacity"] {
		fmt.Fprintf(b, ` stroke-opacity="%g"`, st.StrokeOpacity)
	}
}

func cssColorHex(c [3]float64) string {
	r := int(c[0]*255 + 0.5)
	g := int(c[1]*255 + 0.5)
	bl := int(c[2]*255 + 0.5)
	if r < 0 {
		r = 0
	}
	if g < 0 {
		g = 0
	}
	if bl < 0 {
		bl = 0
	}
	if r > 255 {
		r = 255
	}
	if g > 255 {
		g = 255
	}
	if bl > 255 {
		bl = 255
	}

	return fmt.Sprintf("#%02x%02x%02x", r, g, bl)
}

func escapeXML(s string) string {
	replacer := strings.NewReplacer(
		`&`, "&amp;",
		`<`, "&lt;",
		`>`, "&gt;",
		`"`, "&quot;",
		`'`, "&apos;",
	)

	return replacer.Replace(s)
}

// collectInlineSVGItem flattens an inline <svg> into one replaced inline item.
// paint=false so emitInlineImage places the bitmap at the line position.
func (e *engine) collectInlineSVGItem(node *html.Node, sty ResolvedStyle, out *[]inlineItem) {
	svgBox := e.buildInlineSVG(node, sty, 0, 0, false)
	*out = append(*out, inlineItem{ //nolint:exhaustruct // intentional zero fields
		img: true, w: svgBox.w, h: svgBox.height, style: e.stylePtr(node),
		imgRef:  svgBox.img,
		marginL: e.scalePt(sty.MarginLeft), marginR: e.scalePt(sty.MarginRight),
	})
}
