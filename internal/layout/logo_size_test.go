package layout

import (
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

func TestLogoImgHonorsCSSWidth(t *testing.T) {
	t.Parallel()
	cssSheet := sheet(t, `body{margin:0;font-size:16px}`)
	// Tiny containing float must NOT crush a definite-width logo.
	htmlSrc := `<html><body>
<div style="width:40pt;float:left">
<img src="logo.svg" style="width: 8.75em; height: 1.375em;" alt="Wikipedia">
</div>
<p>text beside</p>
</body></html>`

	root, err := html.Parse(htmlSrc)
	if err != nil {
		t.Fatal(err)
	}

	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 140 22"><rect width="140" height="22" fill="#0e65c0"/></svg>`)
	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 500, Height: 400, Sheets: []*css.Stylesheet{cssSheet},
		Media: "print", Background: true,
		Images: func(_ string) ([]byte, error) { return svg, nil },
	})
	if err != nil {
		t.Fatal(err)
	}

	var imgW float64

	for _, op := range res.Ops {
		if op.Kind == OpImage {
			imgW = op.W
		}
	}
	// 8.75em at 16px = 140px = 105pt
	t.Logf("imgW=%.1f", imgW)

	if imgW < 80 {
		t.Fatalf("logo width %.1f crushed; want ~105pt from 8.75em", imgW)
	}
}
