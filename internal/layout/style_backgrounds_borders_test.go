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
