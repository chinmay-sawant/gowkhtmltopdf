package layout

import (
	"encoding/binary"
	"strconv"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func (e *engine) emitBorders(sty ResolvedStyle, posX, posY, boxW, boxH float64) {
	e.emitBorderLine(posX, posY, boxW, 0, e.scalePt(borderPaint(sty.BorderTop)), sty.BorderTop.Style,
		sty.BorderTop.Color[0], sty.BorderTop.Color[1], sty.BorderTop.Color[2])
	e.emitBorderLine(posX+boxW, posY, 0, boxH, e.scalePt(borderPaint(sty.BorderRight)), sty.BorderRight.Style,
		sty.BorderRight.Color[0], sty.BorderRight.Color[1], sty.BorderRight.Color[2])
	e.emitBorderLine(posX, posY+boxH, boxW, 0, e.scalePt(borderPaint(sty.BorderBottom)), sty.BorderBottom.Style,
		sty.BorderBottom.Color[0], sty.BorderBottom.Color[1], sty.BorderBottom.Color[2])
	e.emitBorderLine(posX, posY, 0, boxH, e.scalePt(borderPaint(sty.BorderLeft)), sty.BorderLeft.Style,
		sty.BorderLeft.Color[0], sty.BorderLeft.Color[1], sty.BorderLeft.Color[2])
}

// --- replaced elements ---

type imageUsedSize struct {
	w, h float64
}

// imageContainingWidth is the width used by percentage/max-width image
// constraints. imgMaxW is set by inline, float, and table-cell layout; the
// viewport is the fallback for ordinary block images.
func (e *engine) imageContainingWidth() float64 {
	if e.imgMaxW > 0 {
		return e.imgMaxW
	}

	return e.opts.Width
}

// usedImageSize is the single sizing policy for replaced images. It starts
// from intrinsic dimensions, applies HTML attributes, then CSS dimensions and
// finally max constraints while preserving the intrinsic aspect ratio for a
// one-dimensional constraint. The same helper is used by block, inline,
// float, and table intrinsic measurement paths.
func (e *engine) usedImageSize(
	node *html.Node, style ResolvedStyle, ref *imageRef,
) imageUsedSize {
	var size imageUsedSize
	if ref != nil {
		size.w = e.scalePt(pxToPt(float64(ref.w)))
		size.h = e.scalePt(pxToPt(float64(ref.h)))
	}

	attrW, attrH := e.imageAttrDims(node)
	if attrW > 0 {
		size.w = attrW
	}

	if attrH > 0 {
		size.h = attrH
	}

	size = applyImageAttrRatio(size, attrW, attrH, ref)

	cssW, cssH := style.Width >= 0, style.Height >= 0

	if style.WidthPercent >= 0 {
		if cb := e.imageContainingWidth(); cb > 0 {
			size.w = cb * style.WidthPercent / cssPercent
			cssW = true
		}
	} else if cssW {
		size.w = e.scalePt(style.Width)
	}

	if cssH {
		size.h = e.scalePt(style.Height)
	}

	size = applyImageCSSRatio(size, cssW, cssH, ref)
	size = clampImageWidth(size, e.imageMaxWidth(style, cssW))
	size = clampImageHeight(e, size, style)

	return size
}

// imageAttrDims reads scaled width/height attributes as used pixel dims.
func (e *engine) imageAttrDims(node *html.Node) (float64, float64) {
	if node == nil {
		return 0, 0
	}

	attrW := 0.0
	if v, err := strconv.Atoi(strings.TrimSpace(node.Attribute("width"))); err == nil && v > 0 {
		attrW = e.scalePt(pxToPt(float64(v)))
	}

	attrH := 0.0
	if v, err := strconv.Atoi(strings.TrimSpace(node.Attribute("height"))); err == nil && v > 0 {
		attrH = e.scalePt(pxToPt(float64(v)))
	}

	return attrW, attrH
}

// hasIntrinsic reports whether the image ref carries pixel dimensions.
func hasIntrinsic(ref *imageRef) bool {
	return ref != nil && ref.w > 0 && ref.h > 0
}

// applyImageAttrRatio fills the missing attribute dimension from the other
// attribute via the intrinsic aspect ratio.
func applyImageAttrRatio(size imageUsedSize, attrW, attrH float64, ref *imageRef) imageUsedSize {
	if !hasIntrinsic(ref) {
		return size
	}

	switch {
	case attrW > 0 && attrH == 0:
		size.h = attrW * float64(ref.h) / float64(ref.w)
	case attrH > 0 && attrW == 0:
		size.w = attrH * float64(ref.w) / float64(ref.h)
	}

	return size
}

// applyImageCSSRatio fills the missing CSS dimension from the other one via
// the intrinsic aspect ratio.
func applyImageCSSRatio(size imageUsedSize, cssW, cssH bool, ref *imageRef) imageUsedSize {
	if !hasIntrinsic(ref) {
		return size
	}

	switch {
	case cssW && !cssH:
		size.h = size.w * float64(ref.h) / float64(ref.w)
	case cssH && !cssW:
		size.w = size.h * float64(ref.w) / float64(ref.h)
	}

	return size
}

// clampImageWidth scales the size down to maxW preserving the aspect ratio.
func clampImageWidth(size imageUsedSize, maxW float64) imageUsedSize {
	if maxW >= 0 && size.w > maxW && size.w > 0 {
		factor := maxW / size.w
		size.w = maxW
		size.h *= factor
	}

	return size
}

// clampImageHeight scales the size down to max-height preserving the ratio.
func clampImageHeight(e *engine, size imageUsedSize, style ResolvedStyle) imageUsedSize {
	if style.MaxHeight < 0 {
		return size
	}

	maxH := e.scalePt(style.MaxHeight)
	if maxH >= 0 && size.h > maxH && size.h > 0 {
		factor := maxH / size.h
		size.w *= factor
		size.h = maxH
	}

	return size
}

// imageMaxWidth resolves the effective max-width constraint: CSS max-width,
// then max-width %, then the float/table/inline containing block for
// auto-sized images (a definite image width stays authoritative).
func (e *engine) imageMaxWidth(style ResolvedStyle, cssW bool) float64 {
	maxW := -1.0
	if style.MaxWidth >= 0 {
		maxW = e.scalePt(style.MaxWidth)
	}

	if style.MaxWidthPercent >= 0 {
		if cb := e.imageContainingWidth(); cb > 0 {
			pct := cb * style.MaxWidthPercent / cssPercent
			if maxW < 0 || pct < maxW {
				maxW = pct
			}
		}
	}

	if !cssW && e.imgMaxW > 0 && (maxW < 0 || e.imgMaxW < maxW) {
		maxW = e.imgMaxW
	}

	return maxW
}

func (e *engine) buildImage(node *html.Node, sty ResolvedStyle, posX, posY float64, paint bool) *box {
	boxNode := &box{ //nolint:exhaustruct // intentional zero fields
		node: node, style: e.stylePtr(node), kind: "replaced", x: posX, y: posY,
	}
	boxNode.img = e.resolveImage(node.Attribute("src"))
	size := e.usedImageSize(node, sty, boxNode.img)
	thumbImg := e.thumbImageInsideFigure(node)
	padL := e.scalePt(sty.PaddingLeft)
	padR := e.scalePt(sty.PaddingRight)
	padT := e.scalePt(sty.PaddingTop)
	padB := e.scalePt(sty.PaddingBottom)
	borderL := e.scalePt(borderPaint(sty.BorderLeft))
	borderR := e.scalePt(borderPaint(sty.BorderRight))
	borderT := e.scalePt(borderPaint(sty.BorderTop))
	borderB := e.scalePt(borderPaint(sty.BorderBottom))

	if thumbImg {
		// Figure already owns the outer rails; keep the bitmap flush so a
		// second inset frame does not double the thumb edge. The bottom
		// separator is painted when the image is placed on a line.
		boxNode.w, boxNode.height = size.w, size.h
	} else {
		boxNode.w = size.w + padL + padR + borderL + borderR
		boxNode.height = size.h + padT + padB + borderT + borderB
	}

	// Paint replaced images that are not deferred to the inline line box.
	// collectImageItem passes paint=false so emitInlineImage owns placement
	// (including display:block thumbs nested in inline <a href>).
	if paint {
		e.paintReplacedImage(boxNode, sty, posX, posY, size, thumbImg, borderL, padL, borderT, padT)
	}

	return boxNode
}

//nolint:cyclop // replaced image layout with border, padding and thumbnail handling
func (e *engine) paintReplacedImage(
	boxNode *box, sty ResolvedStyle, posX, posY float64,
	size imageUsedSize, thumbImg bool,
	borderL, padL, borderT, padT float64,
) {
	if boxNode.img == nil || boxNode.img.data == nil {
		return
	}

	inlineLevel := sty.Display == cssDisplayInline || sty.Display == cssDisplayInlineBlock ||
		sty.Display == displayInlineFlex || sty.Display == ""
	if sty.Float == cssDisplayNone && inlineLevel {
		return
	}

	imgX, imgY, imgW, imgH := posX, posY, size.w, size.h
	if !thumbImg {
		imgX += borderL + padL
		imgY += borderT + padT
	}

	alt := ""
	if boxNode != nil && boxNode.node != nil {
		alt = boxNode.node.Attribute("alt")
	}

	e.add(Op{ //nolint:exhaustruct // intentional zero fields
		Kind:   OpImage,
		X:      imgX,
		Y:      imgY,
		W:      imgW,
		H:      imgH,
		Image:  boxNode.img.data,
		ImgW:   boxNode.img.w,
		ImgH:   boxNode.img.h,
		IsJPEG: boxNode.img.isJPEG,
		Alt:    alt,
	})

	if thumbImg {
		e.emitThumbImageBottomSeparator(sty, posX, posY, size.w, size.h)

		return
	}

	e.prependChrome(len(e.ops)-1, boxNode, sty, posX, posY, boxNode.w, boxNode.height)
}

// emitThumbImageBottomSeparator paints the single bottom rule between a
// collapsed figure thumb bitmap and its caption. Left/right/top belong to the
// figure frame; emitting them again doubles the rails.
func (e *engine) emitThumbImageBottomSeparator(sty ResolvedStyle, posX, posY, width, height float64) {
	bottom := sty.BorderBottom
	if borderPaint(bottom) <= 0 || bottom.Style == cssDisplayNone {
		return
	}

	e.emitBorderLine(
		posX, posY+height, width, 0,
		e.scalePt(borderPaint(bottom)), bottom.Style,
		bottom.Color[0], bottom.Color[1], bottom.Color[2],
	)
}

func (e *engine) buildHR(n *html.Node, sty ResolvedStyle, availW, posX, posY float64) *box {
	boxNode := &box{ //nolint:exhaustruct // intentional zero fields
		node: n, style: e.stylePtr(n), kind: "replaced", x: posX, y: posY, w: availW,
	}
	if sty.Width >= 0 {
		boxNode.w = e.scalePt(sty.Width)
	}

	boxNode.height = e.scalePt(sty.BorderTop.Width) + e.scalePt(sty.BorderBottom.Width)
	if boxNode.height <= 0 {
		boxNode.height = 1
	}

	child := [3]float64{0, 0, 0}
	if sty.BorderTop.Style != cssDisplayNone {
		child = sty.BorderTop.Color
	}

	if boxNode.height > 0 {
		e.add(Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpFillRect, X: posX, Y: posY, W: boxNode.w, H: boxNode.height,
			R: child[0], G: child[1], B: child[2],
		})
	}

	return boxNode
}

// imageDims extracts pixel dimensions from PNG or JPEG bytes.
func imageDims(data []byte) (int, int, bool, bool) {
	if len(data) >= 24 && string(data[:8]) == "\x89PNG\r\n\x1a\n" {
		return int(binary.BigEndian.Uint32(data[16:20])), int(binary.BigEndian.Uint32(data[20:24])), false, true
	}

	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xD8 {
		return jpegDims(data)
	}

	return 0, 0, false, false
}

// jpegDims scans JPEG segment markers for a SOF segment carrying dimensions.
// Layout matches pdf/images.go jpegScan SOF field order: after the marker and
// 2-byte length, precision (1), height (2), width (2).
func jpegDims(data []byte) (int, int, bool, bool) {
	pos := 2
	for pos+4 <= len(data) {
		if data[pos] != byteMax {
			pos++

			continue
		}

		marker := data[pos+1]
		if marker == 0xD9 || marker == 0xDA {
			return 0, 0, false, false
		}

		if isSOFMarker(marker) {
			// SOF layout: marker, 2-byte length, precision, height, width.
			// Need through width (pos+8 inclusive).
			if pos+9 > len(data) {
				return 0, 0, false, false
			}

			height := int(binary.BigEndian.Uint16(data[pos+5 : pos+7]))
			width := int(binary.BigEndian.Uint16(data[pos+7 : pos+9]))

			if width <= 0 || height <= 0 {
				return 0, 0, false, false
			}

			return width, height, true, true
		}

		segLen := int(data[pos+2])<<byteShift | int(data[pos+3])
		if segLen < jpegSegHeaderLen {
			return 0, 0, false, false
		}

		pos += jpegSegHeaderLen + segLen
	}

	return 0, 0, false, false
}

// isSOFMarker reports whether marker is a JPEG start-of-frame segment that
// carries image dimensions (skips DHT/DAC/DNL in the 0xC0..0xCF range).
func isSOFMarker(marker byte) bool {
	return marker >= 0xC0 && marker <= 0xCF && marker != 0xC4 && marker != 0xC8 && marker != 0xCC
}

// --- tables ---
