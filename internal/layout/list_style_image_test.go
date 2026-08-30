//nolint:testpackage,wsl,varnamelen,funlen,paralleltest,err113 // list-style-image probes
package layout

import (
	"bytes"
	"errors"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

func TestListStyleImage(t *testing.T) {
	t.Parallel()

	t.Run("apply", testListStyleImageApply)
	t.Run("paints", testListStyleImagePaints)
	t.Run("missingFallsBack", testListStyleImageMissingFallsBack)
	t.Run("noneClears", testListStyleImageNoneClears)
	t.Run("position", testListStyleImagePosition)
	t.Run("layout", testListStyleImageLayout)
}

func testListStyleImageApply(t *testing.T) {
	t.Parallel()

	tests := []struct {
		prop, value, want string
	}{
		{prop: "list-style-image", value: `url("x.png")`, want: "x.png"},
		{prop: "list-style-image", value: `url('x.png')`, want: "x.png"},
		{prop: "list-style-image", value: "url(x.png)", want: "x.png"},
		{prop: "list-style-image", value: `url( "x.png" )`, want: "x.png"},
		{prop: "list-style-image", value: "none", want: ""},
		{prop: "list-style-image", value: "NONE", want: ""},
		{prop: "list-style", value: `url("y.png")`, want: "y.png"},
		{prop: "list-style", value: `square url("y.png") inside`, want: "y.png"},
		{prop: "list-style", value: "none", want: ""},
		{prop: "list-style", value: "square inside", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.prop+"="+tc.value, func(t *testing.T) {
			t.Parallel()

			var sty ResolvedStyle
			if tc.prop == "list-style-image" {
				applyListStyleImageValue(&sty, tc.value)
			} else {
				applyListProps(&sty, tc.prop, tc.value)
			}

			if sty.ListStyleImage != tc.want {
				t.Fatalf("ListStyleImage = %q, want %q", sty.ListStyleImage, tc.want)
			}
		})
	}

	root := mustParse(t, `<html><body>
		<ul>
			<li class="img">a</li>
			<li class="none">b</li>
			<li class="sh">c</li>
		</ul>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.img { list-style-image: url("x.png") }
		.none { list-style-image: none }
		.sh { list-style: square url("y.png") inside }
	`)}, "print", testViewport, 800)

	if got := styleByClass(t, styles, "img").ListStyleImage; got != "x.png" {
		t.Fatalf("list-style-image url = %q, want x.png", got)
	}

	if got := styleByClass(t, styles, "none").ListStyleImage; got != "" {
		t.Fatalf("list-style-image none = %q, want empty", got)
	}

	sh := styleByClass(t, styles, "sh")
	if sh.ListStyleImage != "y.png" {
		t.Fatalf("list-style shorthand image = %q, want y.png", sh.ListStyleImage)
	}

	if sh.ListStyleType != "square" {
		t.Fatalf("list-style shorthand type = %q, want square", sh.ListStyleType)
	}

	if sh.ListStylePosition != "inside" {
		t.Fatalf("list-style shorthand position = %q, want inside", sh.ListStylePosition)
	}
}

func testListStyleImagePaints(t *testing.T) {
	t.Parallel()

	png := tinyPNG(10, 20)
	var fetched []string
	eng := newBackgroundImageEngine(func(src string) ([]byte, error) {
		fetched = append(fetched, src)

		return png, nil
	})
	sty := ResolvedStyle{ //nolint:exhaustruct // image marker fields under test
		ListStyleImage: `url("x.png")`,
		FontSize:       12,
	}
	eng.emitListMarker(nil, sty, 40, 30)

	imageOp := wantSingleImageOp(t, eng.ops)
	if imageOp.ImgW != 10 || imageOp.ImgH != 20 {
		t.Errorf("intrinsic = %dx%d, want 10x20", imageOp.ImgW, imageOp.ImgH)
	}

	if !near(imageOp.W, 7.5) || !near(imageOp.H, 15) {
		t.Errorf("used size = %vx%v, want 7.5x15", imageOp.W, imageOp.H)
	}

	if !bytes.Equal(imageOp.Image, png) {
		t.Error("image bytes do not match provided PNG")
	}

	if len(fetched) != 1 || fetched[0] != "x.png" {
		t.Errorf("fetched = %q, want [x.png]", fetched)
	}

	if bullets := opsOfKind(&Result{Ops: eng.ops}, OpBullet); len(bullets) != 0 { //nolint:exhaustruct // ops only
		t.Fatalf("image marker also painted type bullets = %+v", bullets)
	}
}

func testListStyleImageMissingFallsBack(t *testing.T) {
	t.Parallel()

	eng := newBackgroundImageEngine(func(string) ([]byte, error) {
		return nil, errors.New("missing")
	})
	sty := ResolvedStyle{ //nolint:exhaustruct // missing fetch falls back to disc
		ListStyleImage: "missing.png",
		ListStyleType:  listStyleDisc,
		FontSize:       12,
	}
	eng.emitListMarker(nil, sty, 40, 30)

	if images := opsOfKind(&Result{Ops: eng.ops}, OpImage); len(images) != 0 { //nolint:exhaustruct // ops only
		t.Fatalf("missing image still painted OpImage = %+v", images)
	}

	bullets := opsOfKind(&Result{Ops: eng.ops}, OpBullet) //nolint:exhaustruct // ops only
	if len(bullets) != 1 {
		t.Fatalf("missing image bullets = %+v, want 1 type marker", bullets)
	}

	if bullets[0].Text != bulletDisc {
		t.Fatalf("fallback marker text = %q, want disc", bullets[0].Text)
	}
}

func testListStyleImageNoneClears(t *testing.T) {
	t.Parallel()

	called := false
	eng := newBackgroundImageEngine(func(string) ([]byte, error) {
		called = true

		return tinyPNG(2, 2), nil
	})
	sty := ResolvedStyle{ //nolint:exhaustruct // none must not fetch
		ListStyleImage: "",
		ListStyleType:  listStyleDisc,
		FontSize:       12,
	}
	applyListStyleImageValue(&sty, `url("x.png")`)
	applyListStyleImageValue(&sty, "none")
	if sty.ListStyleImage != "" {
		t.Fatalf("none did not clear ListStyleImage = %q", sty.ListStyleImage)
	}

	eng.emitListMarker(nil, sty, 40, 30)

	if called {
		t.Fatal("list-style-image:none still fetched")
	}

	if images := opsOfKind(&Result{Ops: eng.ops}, OpImage); len(images) != 0 { //nolint:exhaustruct // ops only
		t.Fatalf("none painted OpImage = %+v", images)
	}

	if bullets := opsOfKind(&Result{Ops: eng.ops}, OpBullet); len(bullets) != 1 { //nolint:exhaustruct // ops only
		t.Fatalf("none bullets = %+v, want 1 type marker", bullets)
	}
}

func testListStyleImagePosition(t *testing.T) {
	t.Parallel()

	png := tinyPNG(10, 20)
	paint := func(position string) Op {
		eng := newBackgroundImageEngine(func(string) ([]byte, error) {
			return png, nil
		})
		sty := ResolvedStyle{ //nolint:exhaustruct // position vs image marker X
			ListStyleImage:    "x.png",
			ListStylePosition: position,
			FontSize:          12,
		}
		eng.emitListMarker(nil, sty, 40, 30)

		return wantSingleImageOp(t, eng.ops)
	}

	inside := paint(listPosInside)
	outside := paint(listPosOutside)

	if inside.X != 40 {
		t.Fatalf("inside image X=%.3f, want 40 (contentX)", inside.X)
	}

	if outside.X >= inside.X {
		t.Fatalf("outside image X=%.3f should hang left of inside X=%.3f", outside.X, inside.X)
	}

	wantOutside := listMarkerX(listPosOutside, 40, 12, 7.5)
	if !near(outside.X, wantOutside) {
		t.Fatalf("outside image X=%.3f, want %.3f", outside.X, wantOutside)
	}
}

func testListStyleImageLayout(t *testing.T) {
	t.Parallel()

	png := tinyPNG(10, 20)
	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
ul { margin: 0; padding-left: 24pt; }
li { list-style-image: url("x.png"); }
`)
	insideSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
ul { margin: 0; padding-left: 24pt; }
li { list-style-image: url("x.png"); list-style-position: inside; }
`)

	outside := layoutListStyleImage(t, listStylePositionFixture, png, "x.png", cssSheet)
	inside := layoutListStyleImage(t, listStylePositionFixture, png, "x.png", insideSheet)

	outsideImg := firstImage(t, outside)
	insideImg := firstImage(t, inside)
	contentLeft := listItemContentLeft(t, inside)

	if !bytes.Equal(outsideImg.Image, png) {
		t.Error("layout image bytes do not match provided PNG")
	}

	if bullets := opsOfKind(outside, OpBullet); len(bullets) != 0 {
		t.Fatalf("layout painted type bullets = %+v", bullets)
	}

	if insideImg.X < contentLeft {
		t.Fatalf("inside image X=%.3f is left of content edge %.3f", insideImg.X, contentLeft)
	}

	if outsideImg.X >= insideImg.X {
		t.Fatalf("outside image X=%.3f should hang left of inside X=%.3f", outsideImg.X, insideImg.X)
	}
}

func layoutListStyleImage(
	t *testing.T, src string, png []byte, wantSrc string, sheets ...*css.Stylesheet,
) *Result {
	t.Helper()

	root := mustParse(t, src)
	res, err := Layout(root, Options{ //nolint:exhaustruct // viewport plus image fetch
		Width: testViewport, Height: 800, Sheets: sheets, Background: true,
		Images: func(src string) ([]byte, error) {
			if wantSrc != "" && src != wantSrc {
				t.Errorf("unexpected image src %q", src)
			}

			return png, nil
		},
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	return res
}

func wantSingleImageOp(t *testing.T, ops []Op) Op {
	t.Helper()

	var imageOp Op
	count := 0

	for _, op := range ops {
		if op.Kind != OpImage {
			continue
		}

		count++
		imageOp = op
	}

	if count != 1 {
		t.Fatalf("image ops = %d, want 1 in %+v", count, ops)
	}

	return imageOp
}

func firstImage(t *testing.T, res *Result) Op {
	t.Helper()

	images := opsOfKind(res, OpImage)
	if len(images) != 1 {
		t.Fatalf("images = %+v, want 1", images)
	}

	return images[0]
}
