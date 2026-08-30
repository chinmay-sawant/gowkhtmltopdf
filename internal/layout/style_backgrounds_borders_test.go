//nolint:wsl,varnamelen,exhaustruct,usetesting,testpackage // targeted unit tests for Phase 80
package layout

import (
	"context"
	"testing"
)

func TestBorderSideStyles(t *testing.T) {
	t.Parallel()

	ctx := &styleContext{
		ctx:       context.Background(),
		err:       nil,
		work:      0,
		sheets:    nil,
		viewportW: 800,
	}

	s := initialStyle()
	raw := map[string]string{
		"border-top-style":    "dashed",
		"border-right-style":  "dotted",
		"border-bottom-style": "solid",
		"border-left-style":   "none",
	}
	applyRestProps(&s, raw, ctx, nil)

	if s.BorderTop.Style != "dashed" || s.BorderRight.Style != "dotted" ||
		s.BorderBottom.Style != "solid" || s.BorderLeft.Style != "none" {
		t.Errorf("mismatch on border side styles: top=%v, right=%v, bot=%v, left=%v",
			s.BorderTop.Style, s.BorderRight.Style, s.BorderBottom.Style, s.BorderLeft.Style)
	}
}

func TestBoxShadowLonghands(t *testing.T) {
	t.Parallel()

	ctx := &styleContext{
		ctx:       context.Background(),
		err:       nil,
		work:      0,
		sheets:    nil,
		viewportW: 800,
	}

	s := initialStyle()
	raw := map[string]string{
		"box-shadow-offset":   "5pt 10pt",
		"box-shadow-blur":     "15pt",
		"box-shadow-spread":   "2pt",
		"box-shadow-color":    "red",
		"box-shadow-position": "inset",
	}
	applyRestProps(&s, raw, ctx, nil)

	if s.BoxShadowX != 5 || s.BoxShadowY != 10 || s.BoxShadowBlur != 15 || s.BoxShadowSpread != 2 {
		t.Errorf("mismatch on box shadow offset/blur/spread: x=%v, y=%v, blur=%v, spread=%v",
			s.BoxShadowX, s.BoxShadowY, s.BoxShadowBlur, s.BoxShadowSpread)
	}
	if s.BoxShadowColor != [3]float64{1, 0, 0} || !s.BoxShadowInset {
		t.Errorf("mismatch on box shadow color/inset: color=%v, inset=%v", s.BoxShadowColor, s.BoxShadowInset)
	}
}

func TestBackgroundLonghands(t *testing.T) {
	t.Parallel()

	ctx := &styleContext{
		ctx:       context.Background(),
		err:       nil,
		work:      0,
		sheets:    nil,
		viewportW: 800,
	}

	s := initialStyle()
	raw := map[string]string{
		"background-image":      "url(header.png)",
		"background-position-x": "center",
		"background-position-y": "top",
		"background-size":       "cover",
		"background-repeat":     "no-repeat",
		"background-clip":       "padding-box",
		"background-origin":     "content-box",
		"background-attachment": "fixed",
	}
	applyRestProps(&s, raw, ctx, nil)

	if s.BackgroundImage != "header.png" || s.BackgroundPosX != "center" || s.BackgroundPosY != "top" {
		t.Errorf("mismatch on bg image/pos: img=%q, posX=%q, posY=%q", s.BackgroundImage, s.BackgroundPosX, s.BackgroundPosY)
	}
	if s.BackgroundSize != "cover" || s.BackgroundRepeat != "no-repeat" {
		t.Errorf("mismatch on bg size/repeat: size=%q, repeat=%q", s.BackgroundSize, s.BackgroundRepeat)
	}
	if s.BackgroundClip != "padding-box" || s.BackgroundOrigin != "content-box" || s.BackgroundAttachment != "fixed" {
		t.Errorf("mismatch on bg clip/origin/attachment: clip=%q, origin=%q, attachment=%q",
			s.BackgroundClip, s.BackgroundOrigin, s.BackgroundAttachment)
	}
}

func TestBorderImageProps(t *testing.T) {
	t.Parallel()

	ctx := &styleContext{
		ctx:       context.Background(),
		err:       nil,
		work:      0,
		sheets:    nil,
		viewportW: 800,
	}

	s := initialStyle()
	raw := map[string]string{
		"border-image-source": "url(border.png)",
		"border-image-slice":  "30",
		"border-image-width":  "10pt",
		"border-image-outset": "2pt",
		"border-image-repeat": "round",
	}
	applyRestProps(&s, raw, ctx, nil)

	if s.BorderImageSource != "url(border.png)" || s.BorderImageSlice != "30" ||
		s.BorderImageWidth != "10pt" || s.BorderImageOutset != "2pt" || s.BorderImageRepeat != "round" {
		t.Errorf("mismatch on border image props: %+v", s)
	}
}
