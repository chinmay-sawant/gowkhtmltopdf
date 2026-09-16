package layout

import (
	"bytes"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// layoutLinkFixture lays out src with the given stylesheet and image provider.
func layoutLinkFixture(
	t *testing.T, src string, cssSheet *css.Stylesheet, png []byte,
) *Result {
	t.Helper()

	root, err := html.Parse(src)
	if err != nil {
		t.Fatalf("html.Parse: %v", err)
	}

	opts := Options{
		Width: 600, Height: 800, Background: true, Media: "print",
		Images: func(string) ([]byte, error) { return png, nil },
	}
	if cssSheet != nil {
		opts.Sheets = []*css.Stylesheet{cssSheet}
	}

	res, err := Layout(root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	return res
}

// TestImageOnlyAnchorEmitsLinkOp: an anchor whose only content is an image must
// emit one link op covering the image box. The anchor may be inline, blockified
// by author CSS (display:block / inline-block), or a floated wiki thumb
// (figure[typeof~='mw:File/Thumb'] > .mw-file-description { display:block }).
func TestImageOnlyAnchorEmitsLinkOp(t *testing.T) { //nolint:funlen // table-driven link-rect coverage
	t.Parallel()

	const (
		wantURI = "https://x.example/file"
		eps     = 0.5
	)

	png := tinyPNG(120, 80)

	cases := []struct {
		name string
		src  string
		css  string
	}{
		{
			name: "inline-anchor",
			src: `<html><body><p>before ` +
				`<a href="` + wantURI + `"><img src="a.png" width="120" height="80"></a> after</p></body></html>`,
		},
		{
			// tutorialspoint Tutorix banner: the anchor itself carries
			// style="display: block", and the img is display:block too.
			name: "block-anchor",
			src: `<html><body><div style="max-width: 920px">` +
				`<a href="` + wantURI + `" style="display: block; text-decoration: none;">` +
				`<img src="a.png" style="width: 100%; height: auto; display: block;"></a>` +
				`</div></body></html>`,
		},
		{
			// ana-de-armas wiki thumb: mediawiki.skinning sets
			// figure[typeof~='mw:File/Thumb'] > .mw-file-description
			// { display:block; position:relative }, and the figure floats.
			name: "wiki-figure-thumb",
			src: `<html><body>
<p>Before the thumb float with enough words to wrap beside the image.</p>
<figure typeof="mw:File/Thumb" class="mw-default-size mw-halign-right">
<a href="` + wantURI + `" class="mw-file-description">` +
				`<img src="a.png" width="120" height="80" class="mw-file-element"></a>
<figcaption>Cast photo caption</figcaption>
</figure>
<p>After the figure the paragraph continues.</p></body></html>`,
			css: `
figure[typeof~="mw:File/Thumb"] {
  display: table; text-align: center; border-collapse: collapse; line-height: 0;
  margin: 0.5em 0 1.3em 1.4em; clear: right; float: right;
}
figure[typeof~="mw:File/Thumb"] > .mw-file-description { display: block; position: relative; }
figure[typeof~="mw:File/Thumb"] > figcaption {
  display: table-caption; caption-side: bottom; border: 1px solid #c8ccd1;
  border-top: 0; font-size: 88.4%; line-height: 1.4; padding: 3px 6px;
}
.mw-file-element { display: block; }
`,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var cssSheet *css.Stylesheet
			if test.css != "" {
				cssSheet = sheet(t, test.css)
			}

			res := layoutLinkFixture(t, test.src, cssSheet, png)

			imgs := opsOfKind(res, OpImage)
			if len(imgs) != 1 {
				t.Fatalf("image ops = %d, want 1", len(imgs))
			}

			links := opsOfKind(res, OpLinkURI)
			if len(links) != 1 {
				t.Fatalf("link ops = %d, want exactly 1 (image-only anchor must emit one link op)", len(links))
			}

			if links[0].URI != wantURI {
				t.Errorf("link URI = %q, want %q", links[0].URI, wantURI)
			}

			img, link := imgs[0], links[0]

			covers := link.X <= img.X+eps && link.Y <= img.Y+eps &&
				link.X+link.W >= img.X+img.W-eps && link.Y+link.H >= img.Y+img.H-eps
			if !covers {
				t.Errorf("link rect (%.1f,%.1f,%.1f,%.1f) does not cover image rect (%.1f,%.1f,%.1f,%.1f)",
					link.X, link.Y, link.W, link.H, img.X, img.Y, img.W, img.H)
			}
		})
	}
}

// TestFirstPageAnchorEmitsURIAnnot: programiz-cpp page 1 emits no URI annots
// because every header anchor is blockified by author CSS (inline-block brand,
// flex CTA) and never collected by collectInlineSpan. The same anchor markup in
// normal inline flow works, so the first-page header shape must too.
func TestFirstPageAnchorEmitsURIAnnot(t *testing.T) { //nolint:cyclop,funlen
	t.Parallel()

	var filler strings.Builder
	for range 40 {
		filler.WriteString(`<p>Filler paragraph with several words to build up the page height for later content.</p>`)
	}

	const wantURI = "https://example.com/"

	cases := []struct {
		name string
		src  string
		css  string
	}{
		{
			name: "plain-header",
			src:  `<html><body><header><a href="` + wantURI + `">Brand</a></header>` + filler.String() + `</body></html>`,
		},
		{
			// programiz-cpp real rule: .brand a { display:inline-block }.
			name: "inline-block-header",
			src: `<html><body><header><span class="brand"><a href="` + wantURI + `">Brand</a></span></header>` +
				filler.String() + `</body></html>`,
			css: `.brand a { display: inline-block; width: 84px; height: auto; }`,
		},
		{
			// Flex/grid items are blockified even with display:inline.
			name: "flex-header",
			src: `<html><body><header class="nav"><a href="` + wantURI + `">Brand</a></header>` +
				filler.String() + `</body></html>`,
			css: `.nav { display: flex; }`,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var cssSheet *css.Stylesheet
			if test.css != "" {
				cssSheet = sheet(t, test.css)
			}

			root := mustParse(t, test.src)

			opts := Options{
				Width: 500, Height: 600, Background: true, Media: "print",
			}
			if cssSheet != nil {
				opts.Sheets = []*css.Stylesheet{cssSheet}
			}

			res, err := Layout(root, opts)
			if err != nil {
				t.Fatalf("Layout: %v", err)
			}

			doc := pdf.NewDocument()

			if err := Paint(doc, res, PaintOptions{PageWidth: 500, PageHeight: 600}); err != nil {
				t.Fatalf("Paint: %v", err)
			}

			var buf bytes.Buffer
			if err := doc.Write(&buf); err != nil {
				t.Fatalf("Write: %v", err)
			}

			sem, err := pdf.ParseSemantic(buf.Bytes())
			if err != nil {
				t.Fatalf("ParseSemantic: %v", err)
			}

			if len(sem.Pages) < 2 {
				t.Fatalf("pages = %d, want at least 2", len(sem.Pages))
			}

			annots := sem.Pages[0].Annots
			if len(annots) != 1 || annots[0].URI != wantURI {
				t.Fatalf("page 1 annots = %+v, want exactly one URI annot %q", annots, wantURI)
			}
		})
	}
}
