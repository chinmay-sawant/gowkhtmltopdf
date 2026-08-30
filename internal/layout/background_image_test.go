//nolint:testpackage,wsl,varnamelen,exhaustruct,cyclop,err113,usetesting,goconst // image chrome probes
package layout

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

func TestBackgroundImageSrc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		raw  string
		want string
	}{
		{raw: "", want: ""},
		{raw: "none", want: ""},
		{raw: "NONE", want: ""},
		{raw: `url("x.png")`, want: "x.png"},
		{raw: `url('x.png')`, want: "x.png"},
		{raw: "url(x.png)", want: "x.png"},
		{raw: `url( "x.png" )`, want: "x.png"},
		{raw: "x.png", want: "x.png"},
		{raw: `url("a.png"), url("b.png")`, want: "a.png"},
		{raw: "none, url(a.png)", want: ""},
		{raw: "linear-gradient(red, blue)", want: ""},
		{raw: "repeating-linear-gradient(#000, #fff)", want: ""},
		{raw: "inherit", want: ""},
		{raw: `url("a,b.png"), url(c.png)`, want: "a,b.png"},
	}

	for _, tc := range tests {
		t.Run(tc.raw, func(t *testing.T) {
			t.Parallel()

			if got := backgroundImageSrc(tc.raw); got != tc.want {
				t.Fatalf("backgroundImageSrc(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestBackgroundImagePaints(t *testing.T) {
	t.Parallel()

	png := tinyPNG(10, 20)
	var fetched []string
	eng := newBackgroundImageEngine(func(src string) ([]byte, error) {
		fetched = append(fetched, src)

		return png, nil
	})
	sty := ResolvedStyle{BackgroundImage: `url("x.png")`} //nolint:exhaustruct // paint field under test
	eng.prependChrome(0, &box{}, sty, 10, 20, 100, 50)

	imageOp := wantSingleDeferredImage(t, eng)
	if imageOp.X != 10 || imageOp.Y != 20 || imageOp.W != 100 || imageOp.H != 50 {
		t.Errorf("image rect = %v,%v %vx%v, want 10,20 100x50",
			imageOp.X, imageOp.Y, imageOp.W, imageOp.H)
	}

	if imageOp.ImgW != 10 || imageOp.ImgH != 20 {
		t.Errorf("intrinsic = %dx%d, want 10x20", imageOp.ImgW, imageOp.ImgH)
	}

	if !bytes.Equal(imageOp.Image, png) {
		t.Error("image bytes do not match provided PNG")
	}

	if len(fetched) != 1 || fetched[0] != "x.png" {
		t.Errorf("fetched = %q, want [x.png]", fetched)
	}
}

func TestBackgroundImagePaintsOverColor(t *testing.T) {
	t.Parallel()

	png := tinyPNG(4, 4)
	eng := newBackgroundImageEngine(func(string) ([]byte, error) {
		return png, nil
	})
	sty := ResolvedStyle{ //nolint:exhaustruct // color then image paint order
		BGColor:         [4]float64{1, 0, 0, 1},
		BackgroundImage: `url("x.png")`,
	}
	eng.prependChrome(0, &box{}, sty, 0, 0, 80, 40)

	if len(eng.deferredChrome) != 1 {
		t.Fatalf("deferred chrome entries = %d, want 1", len(eng.deferredChrome))
	}

	ops := eng.deferredChrome[0].ops
	if len(ops) < 2 || ops[0].Kind != OpFillRect || ops[1].Kind != OpImage {
		t.Fatalf("chrome ops = %+v, want fill then image", ops)
	}

	if ops[1].W != 80 || ops[1].H != 40 {
		t.Errorf("image size = %vx%v, want 80x40", ops[1].W, ops[1].H)
	}
}

func TestBackgroundImageEmptySkips(t *testing.T) {
	t.Parallel()

	called := false
	eng := newBackgroundImageEngine(func(string) ([]byte, error) {
		called = true

		return nil, errors.New("should not fetch")
	})
	eng.prependChrome(0, &box{}, ResolvedStyle{}, 0, 0, 100, 50) //nolint:exhaustruct // empty style

	if called {
		t.Fatal("empty BackgroundImage fetched an image")
	}

	if len(eng.deferredChrome) != 0 {
		t.Fatalf("deferred chrome = %+v, want none", eng.deferredChrome)
	}
}

func TestBackgroundImageMissingSkips(t *testing.T) {
	t.Parallel()

	eng := newBackgroundImageEngine(func(string) ([]byte, error) {
		return nil, errors.New("missing")
	})
	sty := ResolvedStyle{BackgroundImage: "missing.png"} //nolint:exhaustruct // missing fetch
	eng.prependChrome(0, &box{}, sty, 0, 0, 100, 50)

	if len(eng.deferredChrome) != 0 {
		t.Fatalf("missing image still painted chrome = %+v", eng.deferredChrome)
	}

	if len(eng.ops) != 0 {
		t.Fatalf("missing image emitted ops = %+v", eng.ops)
	}
}

func TestBackgroundImageGradientRenders(t *testing.T) {
	t.Parallel()

	called := false
	eng := newBackgroundImageEngine(func(string) ([]byte, error) {
		called = true

		return tinyPNG(2, 2), nil
	})
	sty := ResolvedStyle{BackgroundImage: "linear-gradient(red, blue)"} //nolint:exhaustruct // gradient
	eng.prependChrome(0, &box{}, sty, 0, 0, 100, 50)

	if called {
		t.Fatal("gradient unexpectedly fetched an external image")
	}

	if len(eng.deferredChrome) == 0 {
		t.Fatal("gradient did not paint any chrome")
	}
}

func TestBackgroundImageMultiLayer(t *testing.T) {
	t.Parallel()

	var fetched []string
	eng := newBackgroundImageEngine(func(src string) ([]byte, error) {
		fetched = append(fetched, src)

		return tinyPNG(2, 2), nil
	})
	sty := ResolvedStyle{BackgroundImage: `url("a.png"), url("b.png")`} //nolint:exhaustruct // two layers
	eng.prependChrome(0, &box{}, sty, 0, 0, 40, 20)

	if len(fetched) != 2 || fetched[0] != "b.png" || fetched[1] != "a.png" {
		t.Errorf("fetched = %q, want [b.png a.png]", fetched)
	}
}

func TestBackgroundImageNoBackgroundFlag(t *testing.T) {
	t.Parallel()

	called := false
	eng := newBackgroundImageEngine(func(string) ([]byte, error) {
		called = true

		return tinyPNG(2, 2), nil
	})
	eng.opts.Background = false
	sty := ResolvedStyle{BackgroundImage: `url("x.png")`} //nolint:exhaustruct // gated by Background
	eng.prependChrome(0, &box{}, sty, 0, 0, 100, 50)

	if called {
		t.Fatal("Background=false still fetched")
	}

	if len(eng.deferredChrome) != 0 {
		t.Fatalf("Background=false painted chrome = %+v", eng.deferredChrome)
	}
}

func TestBackgroundImageLayoutPaints(t *testing.T) {
	t.Parallel()

	png := tinyPNG(10, 20)
	cssSheet := sheet(t, `div.bg { background-image: url("x.png"); width: 100pt; height: 50pt }`)
	root := mustParse(t, `<html><body><div class="bg">hi</div></body></html>`)
	opts := Options{ //nolint:exhaustruct // layoutHTML viewport plus image fetch
		Width: testViewport, Height: 800, Sheets: []*css.Stylesheet{cssSheet}, Background: true,
		Images: func(src string) ([]byte, error) {
			if src != "x.png" {
				t.Errorf("unexpected image src %q", src)
			}

			return png, nil
		},
	}

	styles, containers, err := resolveStylesForLayoutContext(context.Background(), root, opts)
	if err != nil {
		t.Fatalf("resolve styles: %v", err)
	}

	stampBackgroundImageIfEmpty(styles, `url("x.png")`)

	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	res, err := finalizeResult(
		newEngine(
			context.Background(),
			opts,
			faces,
			faces.Regular,
			styles,
			containers,
			make([]Op, 0, estimateOpCapacity(root)),
		),
		root,
		opts,
	)
	if err != nil {
		t.Fatalf("layout: %v", err)
	}

	imgs := opsOfKind(res, OpImage)
	if len(imgs) != 1 {
		t.Fatalf("image ops = %d, want 1 in %+v", len(imgs), res.Ops)
	}

	if imgs[0].W != 100 || imgs[0].H != 50 {
		t.Errorf("image size = %vx%v, want 100x50", imgs[0].W, imgs[0].H)
	}

	if imgs[0].ImgW != 10 || imgs[0].ImgH != 20 {
		t.Errorf("intrinsic = %dx%d, want 10x20", imgs[0].ImgW, imgs[0].ImgH)
	}

	if !bytes.Equal(imgs[0].Image, png) {
		t.Error("image bytes do not match provided PNG")
	}
}

func TestBackgroundImageBarePath(t *testing.T) {
	t.Parallel()

	png := tinyPNG(8, 8)
	var fetched []string
	eng := newBackgroundImageEngine(func(src string) ([]byte, error) {
		fetched = append(fetched, src)

		return png, nil
	})
	sty := ResolvedStyle{BackgroundImage: "logo.png"} //nolint:exhaustruct // sibling may store a bare path
	eng.prependChrome(0, &box{}, sty, 0, 0, 30, 30)
	imageOp := wantSingleDeferredImage(t, eng)

	if !bytes.Equal(imageOp.Image, png) {
		t.Error("image bytes do not match provided PNG")
	}

	if len(fetched) != 1 || fetched[0] != "logo.png" {
		t.Errorf("fetched = %q, want [logo.png]", fetched)
	}
}

func newBackgroundImageEngine(images func(string) ([]byte, error)) *engine {
	return &engine{ //nolint:exhaustruct // test engine
		opts: Options{ //nolint:exhaustruct // Images + Background only
			Images:     images,
			Background: true,
		},
		scale: 1,
	}
}

func stampBackgroundImageIfEmpty(styles map[*html.Node]*ResolvedStyle, raw string) {
	for node, st := range styles {
		if node == nil || st == nil || node.Name != divElementName {
			continue
		}

		if st.BackgroundImage != "" {
			continue
		}

		copied := *st
		copied.BackgroundImage = raw
		styles[node] = &copied
	}
}

func wantSingleDeferredImage(t *testing.T, eng *engine) Op {
	t.Helper()

	if len(eng.deferredChrome) != 1 {
		t.Fatalf("deferred chrome entries = %d, want 1", len(eng.deferredChrome))
	}

	var imageOp Op
	count := 0

	for _, op := range eng.deferredChrome[0].ops {
		if op.Kind != OpImage {
			continue
		}

		count++
		imageOp = op
	}

	if count != 1 {
		t.Fatalf("image ops = %d, want 1 in %+v", count, eng.deferredChrome[0].ops)
	}

	return imageOp
}
