package layout

import (
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

// Wordmark and tagline must stack vertically (display:block), not sit on one row.
func TestWikiLogoWordmarkAboveTagline(t *testing.T) {
	s := sheet(t, `
@media all {
.mw-logo { display: flex; height: 100%; align-items: center; }
.mw-logo-icon { float: left; margin-right: 10px; display: none; width: 3.125em; height: 3.125em; }
.mw-logo-container { float: left; max-width: 120px; }
.mw-logo-container img { width: 100%; }
.mw-logo-wordmark { display: block; margin: 0 auto; }
.mw-logo-tagline { display: block; margin: 5px auto 0; }
}
@media all and (min-width: 640px) {
.mw-logo-icon { display: block; }
.mw-logo-container { max-width: none; }
.mw-logo-container img { width: auto; }
}
`)
	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
		0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x05, 0xfe, 0xd4, 0xef, 0x00, 0x00,
		0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}

	root, err := html.Parse(`<html><body>
<a class="mw-logo" href="/">
<img class="mw-logo-icon" width="50" height="50" src="icon.png">
<span class="mw-logo-container">
<img class="mw-logo-wordmark" alt="Wikipedia" src="word.png" width="140" height="22" style="width: 8.75em; height: 1.375em;">
<img class="mw-logo-tagline" alt="tag" src="tag.png" width="140" height="11" style="width: 8.75em; height: 0.6875em;">
</span>
</a>
</body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	styles := resolveStyles(root, []*css.Stylesheet{s}, "print", 700, 900)

	var word, tagline, cont *html.Node

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			cl := n.Attribute("class")
			if strings.Contains(cl, "wordmark") {
				word = n
			}

			if strings.Contains(cl, "tagline") {
				tagline = n
			}

			if strings.Contains(cl, "mw-logo-container") {
				cont = n
			}
		}

		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)

	if styles[word].Display != "block" {
		t.Fatalf("wordmark display=%q want block", styles[word].Display)
	}

	if styles[tagline].Display != "block" {
		t.Fatalf("tagline display=%q want block", styles[tagline].Display)
	}

	_ = cont

	res, err := Layout(root, Options{
		Width: 700, Height: 200, Sheets: []*css.Stylesheet{s}, Background: true,
		Images: func(src string) ([]byte, error) { return png, nil },
	})
	if err != nil {
		t.Fatal(err)
	}

	type iy struct{ y, x, w, h float64 }

	var imgs []iy

	for _, op := range res.Ops {
		if op.Kind == OpImage {
			imgs = append(imgs, iy{op.Y, op.X, op.W, op.H})
			t.Logf("img x=%.1f y=%.1f w=%.1f h=%.1f", op.X, op.Y, op.W, op.H)
		}
	}

	if len(imgs) < 3 {
		t.Fatalf("imgs=%d", len(imgs))
	}
	// wordmark then tagline (after icon): same X band, tagline below wordmark
	wm, tl := imgs[1], imgs[2]
	if absF(wm.x-tl.x) > 20 {
		t.Fatalf("wordmark/tagline not stacked in column: word x=%.1f tag x=%.1f (imgs=%v)", wm.x, tl.x, imgs)
	}

	if tl.y < wm.y+wm.h-1 {
		t.Fatalf("tagline not below wordmark: word y=%.1f h=%.1f tag y=%.1f", wm.y, wm.h, tl.y)
	}
}
