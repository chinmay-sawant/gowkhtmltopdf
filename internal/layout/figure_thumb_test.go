//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// Wiki thumbs use figure[typeof~='mw:File/Thumb']{display:table;float:right}.
// Routing those through the table layout path drops nested <img> content.
func TestFigureThumbDisplayTableKeepsImage(t *testing.T) { //nolint:cyclop,funlen
	t.Parallel()
	// 1x1 PNG
	png := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
		0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x05, 0xfe, 0xd4, 0xef, 0x00, 0x00,
		0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}

	cssSheet := sheet(t, `
body { margin: 0; font-size: 10pt; }
figure[typeof~="mw:File/Thumb"] {
  display: table;
  float: right;
  clear: right;
  margin: 0.5em 0 0.5em 1.4em;
  border: 1px solid #ccc;
}
img { display: block; }
p { margin: 0.5em 0; }
`)

	root, err := html.Parse(`<html><body>
<p>Before the thumb float with enough words to wrap beside the image when it is placed.</p>
<figure typeof="mw:File/Thumb" class="mw-default-size">
<a href="/wiki/File:x.jpg"><img class="mw-file-element" width="120" height="80" src="thumb.png"></a>
<figcaption>Cast photo caption</figcaption>
</figure>
<p>After the figure the paragraph should continue in the remaining line width next to the float.</p>
</body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 400, Height: 600, Sheets: []*css.Stylesheet{cssSheet},
		Background: true,
		Images: func(src string) ([]byte, error) {
			if strings.Contains(src, "thumb.png") {
				return png, nil
			}

			return nil, err
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var imgs, captions int

	for _, op := range res.Ops {
		if op.Kind == OpImage && len(op.Image) > 0 {
			imgs++
		}

		if op.Kind == OpText && strings.Contains(op.Text, "caption") {
			captions++
		}
	}

	if imgs == 0 {
		t.Fatal("expected floated figure thumb image op (display:table must not drop img)")
	}

	if captions == 0 {
		t.Fatal("expected figcaption text")
	}
}
