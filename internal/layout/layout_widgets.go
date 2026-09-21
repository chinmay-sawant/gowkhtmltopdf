package layout

import (
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// isInputCheckbox reports whether node is an input checkbox or radio control.
func isInputCheckbox(node *html.Node) bool {
	if node == nil || node.Name != htmlInput {
		return false
	}

	t := strings.ToLower(node.Attribute("type"))

	return t == "checkbox" || t == "radio"
}

// defaultCheckboxSize returns the outer square size of a checkbox or radio.
func defaultCheckboxSize(eng *engine, style ResolvedStyle) float64 {
	if style.Width >= 0 {
		return eng.scalePt(style.Width)
	}

	const defaultCheckboxDimensionPt = 9.75 // 13px standard form-control box

	return eng.scalePt(defaultCheckboxDimensionPt)
}

// applyCheckboxAutoSize sizes an auto-sized checkbox or radio to the default
// form-control box. Width is applied when the style leaves it unspecified;
// height likewise, and its content-flow endpoint is returned. The endpoint
// starts from padding-top + top border, matching buildBlock's curY origin.
func applyCheckboxAutoSize(eng *engine, style ResolvedStyle, boxNode *box, curY float64) float64 {
	chkSz := defaultCheckboxSize(eng, style)

	if style.Width < 0 && style.WidthPercent < 0 {
		boxNode.w = chkSz + eng.scalePt(style.PaddingLeft) + eng.scalePt(style.PaddingRight) +
			eng.scalePt(style.BorderLeft.Width) + eng.scalePt(style.BorderRight.Width)
	}

	if style.Height < 0 && style.HeightPercent < 0 {
		boxStyle := &style
		curY = chkSz + eng.scalePt(boxStyle.PaddingTop) + eng.scalePt(borderLayoutWidth(boxStyle, boxStyle.BorderTop))
	}

	return curY
}

// checkboxGeometry holds the scaled dimensions used when painting a checkbox or radio.
type checkboxGeometry struct {
	boxX, boxY, size, radius float64
}

// checkboxConstants holds the proportional factors for the check-mark strokes.
const (
	chkMaxSzPx       = 13    // maximum clamped size in px
	chkRadiusPx      = 2     // border-radius for checkbox squares
	chkDotInset      = 0.28  // radio dot inset fraction
	chkStrokeWidthPx = 1.4   // tick stroke width in px
	chkTick1X        = 0.22  // tick segment 1 start-x fraction
	chkTick1Y        = 0.50  // tick segment 1 start-y fraction
	chkTick1W        = 0.20  // tick segment 1 delta-x fraction
	chkTick1H        = 0.22  // tick segment 1 delta-y fraction
	chkTick2X        = 0.42  // tick segment 2 start-x fraction
	chkTick2Y        = 0.72  // tick segment 2 start-y fraction
	chkTick2W        = 0.36  // tick segment 2 delta-x fraction
	chkTick2H        = -0.46 // tick segment 2 delta-y fraction (upward)
	chkUncheckedGray = 0.46  // unchecked border gray level
)

// paintCheckboxGeometry computes box position, size, and corner radius.
func paintCheckboxGeometry(eng *engine, isRadio bool, leftX, topY, width, height float64) checkboxGeometry {
	size := width
	if height < size {
		size = height
	}

	maxSz := eng.scalePt(chkMaxSzPx)
	if size > maxSz {
		size = maxSz
	}

	boxX := leftX + (width-size)/two
	boxY := topY + (height-size)/two
	radius := eng.scalePt(chkRadiusPx)

	if isRadio {
		radius = size / two
	}

	return checkboxGeometry{boxX: boxX, boxY: boxY, size: size, radius: radius}
}

// paintCheckedCheckbox paints the filled checkbox (or radio) for a checked input.
func (e *engine) paintCheckedCheckbox(isRadio bool, color [3]float64, geo checkboxGeometry) {
	e.add(Op{ //nolint:exhaustruct // intentional zero fields
		Kind:   OpFillRect,
		X:      geo.boxX,
		Y:      geo.boxY,
		W:      geo.size,
		H:      geo.size,
		R:      color[0],
		G:      color[1],
		B:      color[2],
		Alpha:  1,
		Radius: geo.radius,
	})

	if isRadio {
		e.paintRadioDot(geo)
	} else {
		e.paintCheckTick(geo)
	}
}

// paintRadioDot paints the inner white dot for a checked radio button.
func (e *engine) paintRadioDot(geo checkboxGeometry) {
	dotInset := geo.size * chkDotInset
	dotSize := geo.size - two*dotInset

	e.add(Op{ //nolint:exhaustruct // intentional zero fields
		Kind:   OpFillRect,
		X:      geo.boxX + dotInset,
		Y:      geo.boxY + dotInset,
		W:      dotSize,
		H:      dotSize,
		R:      1,
		G:      1,
		B:      1,
		Alpha:  1,
		Radius: dotSize / two,
	})
}

// paintCheckTick paints the white tick mark inside a checked checkbox.
func (e *engine) paintCheckTick(geo checkboxGeometry) {
	checkColor := [3]float64{1, 1, 1}
	tickW := e.scalePt(chkStrokeWidthPx)

	if tickW < 1 {
		tickW = 1
	}

	e.add(Op{ //nolint:exhaustruct // intentional zero fields
		Kind:  OpLine,
		X:     geo.boxX + chkTick1X*geo.size,
		Y:     geo.boxY + chkTick1Y*geo.size,
		W:     chkTick1W * geo.size,
		H:     chkTick1H * geo.size,
		R:     checkColor[0],
		G:     checkColor[1],
		B:     checkColor[2],
		Alpha: 1,
		Width: tickW,
	})
	e.add(Op{ //nolint:exhaustruct // intentional zero fields
		Kind:  OpLine,
		X:     geo.boxX + chkTick2X*geo.size,
		Y:     geo.boxY + chkTick2Y*geo.size,
		W:     chkTick2W * geo.size,
		H:     chkTick2H * geo.size,
		R:     checkColor[0],
		G:     checkColor[1],
		B:     checkColor[2],
		Alpha: 1,
		Width: tickW,
	})
}

// paintUncheckedCheckbox paints the empty checkbox border for an unchecked input.
func (e *engine) paintUncheckedCheckbox(geo checkboxGeometry) {
	e.add(Op{ //nolint:exhaustruct // intentional zero fields
		Kind:   OpFillRect,
		X:      geo.boxX,
		Y:      geo.boxY,
		W:      geo.size,
		H:      geo.size,
		R:      1,
		G:      1,
		B:      1,
		Alpha:  1,
		Radius: geo.radius,
	})
	e.add(Op{ //nolint:exhaustruct // intentional zero fields
		Kind:   OpStrokeRect,
		X:      geo.boxX,
		Y:      geo.boxY,
		W:      geo.size,
		H:      geo.size,
		R:      chkUncheckedGray,
		G:      chkUncheckedGray,
		B:      chkUncheckedGray,
		Alpha:  1,
		Width:  e.scalePt(1),
		Radius: geo.radius,
	})
}

// paintCheckboxWidget emits display-list operations for an input checkbox or radio.
func (e *engine) paintCheckboxWidget(
	node *html.Node, style ResolvedStyle, leftX, topY, width, height float64,
) {
	t := strings.ToLower(node.Attribute("type"))
	isRadio := t == "radio"

	_, isChecked := node.Attrs["checked"]
	if !isChecked {
		if v := node.Attribute("checked"); v != "" && v != "false" {
			isChecked = true
		}
	}

	geo := paintCheckboxGeometry(e, isRadio, leftX, topY, width, height)
	color := widgetValueColor(htmlInput, style)

	if isChecked {
		e.paintCheckedCheckbox(isRadio, color, geo)
	} else {
		e.paintUncheckedCheckbox(geo)
	}
}

// isInputRange reports whether node is an input range control.
func isInputRange(node *html.Node) bool {
	if node == nil || node.Name != htmlInput {
		return false
	}

	return strings.ToLower(node.Attribute("type")) == "range"
}

// isInputButton reports whether node is an input button control.
func isInputButton(node *html.Node) bool {
	if node == nil || node.Name != htmlInput {
		return false
	}

	return strings.ToLower(node.Attribute("type")) == "button"
}

// rangeThumbDiameterPx is the Chrome range thumb diameter in CSS px.
const rangeThumbDiameterPx = 15

// formControlFontSizePt is the Chrome UA form-control font size in points
// (13.333px). The style pipeline records no "author set font-size" flag, so the
// button label always uses the UA control size rather than the inherited body
// size.
const formControlFontSizePt = 10

// rangeThumbColor returns the range thumb fill: an authored accent-color when
// present, otherwise Chrome's default range accent #0075FF. The input fallback
// in widgetValueColor (0.46) is the rounded value checkboxes use; the range
// thumb keeps Chrome's exact 117/255.
func rangeThumbColor(style ResolvedStyle) [3]float64 {
	if style.AccentColorSet {
		return style.AccentColor
	}

	return [3]float64{0, 117.0 / 255.0, 1}
}

// paintRangeWidget paints the range thumb: a fully rounded circle centered in
// the control's border box. Chrome keeps the thumb visible for the fixture's
// appearance:none range input.
func (e *engine) paintRangeWidget(style ResolvedStyle, leftX, topY, width, height float64) {
	diameter := e.scalePt(pxToPt(rangeThumbDiameterPx))
	if diameter <= 0 || width <= 0 || height <= 0 {
		return
	}

	color := rangeThumbColor(style)

	e.add(Op{ //nolint:exhaustruct // intentional zero fields
		Kind:   OpFillRect,
		X:      leftX + (width-diameter)/two,
		Y:      topY + (height-diameter)/two,
		W:      diameter,
		H:      diameter,
		R:      color[0],
		G:      color[1],
		B:      color[2],
		Alpha:  1,
		Radius: diameter / two,
	})
}

// paintButtonWidget paints the input[type=button] value attribute centered in
// the control's border box at the UA form-control font size.
func (e *engine) paintButtonWidget(node *html.Node, style ResolvedStyle, leftX, topY, width, height float64) {
	text := node.Attribute("value")
	if text == "" {
		return
	}

	face := e.faceFor(&style)
	if face == nil {
		return
	}

	size := e.scalePt(formControlFontSizePt)
	if size <= 0 {
		return
	}

	textStyle := style
	textStyle.FontSize = formControlFontSizePt

	textW := e.measureTextFace(text, &textStyle)
	ascent := e.fontAscentFace(face, size)
	descent := e.fontDescentFace(face, size)

	e.add(Op{ //nolint:exhaustruct // intentional zero fields
		Kind: OpText, X: leftX + (width-textW)/two,
		Y: topY + (height+ascent-descent)/two, W: textW, H: ascent + descent,
		Text: text, Font: face, Size: size, InkDescent: descent,
		R: style.Color[0], G: style.Color[1], B: style.Color[2],
		Bold: style.FontWeight >= fontWeightBoldValue,
	})
}

// paintWidgetControl paints the native control face for value, range, button
// and checkbox widgets after the box height is final.
func (e *engine) paintWidgetControl(
	node *html.Node, style ResolvedStyle, boxNode *box, widget, chkWidget bool, posY float64,
) {
	switch {
	case widget:
		e.paintValueWidget(node, style, boxNode.x, posY, boxNode.w, boxNode.height)
	case chkWidget:
		e.paintCheckboxWidget(node, style, boxNode.x, posY, boxNode.w, boxNode.height)
	case isInputRange(node):
		e.paintRangeWidget(style, boxNode.x, posY, boxNode.w, boxNode.height)
	case isInputButton(node):
		e.paintButtonWidget(node, style, boxNode.x, posY, boxNode.w, boxNode.height)
	}
}
