//nolint:testpackage // white-box test exercises the layout-to-PDF paint seam
package layout

import (
	"bytes"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

//nolint:gocognit,cyclop,funlen,wsl,exhaustruct,varnamelen,nlreturn
func TestFixture55MastheadPreservesLetterSpacing(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile(filepath.Join("..", "..", "testdata", "golden", "fixture-55-lantern-cooperative-report.html"))
	if err != nil {
		t.Fatal(err)
	}

	root, err := html.Parse(string(data))
	if err != nil {
		t.Fatal(err)
	}

	var sheets []*css.Stylesheet
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Name == styleElement {
			var source strings.Builder
			for _, child := range node.Children {
				if child.Type == html.TextNode {
					source.WriteString(child.Text)
				}
			}
			sheet, parseErr := css.Parse(source.String())
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			sheets = append(sheets, sheet)
		}
		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(root)

	res, err := Layout(root, Options{Width: 538, Height: 785, Sheets: sheets, Media: "print", Background: true})
	if err != nil {
		t.Fatal(err)
	}

	doc := pdf.NewDocument()
	doc.SetCompression(false)
	if err := Paint(doc, res, PaintOptions{PageWidth: 595, PageHeight: 842}); err != nil {
		t.Fatalf("Paint: %v", err)
	}

	var pdfData bytes.Buffer
	if err := doc.Write(&pdfData); err != nil {
		t.Fatalf("write PDF: %v", err)
	}
	if !bytes.Contains(pdfData.Bytes(), []byte("0.975 Tc")) {
		t.Fatal("PDF masthead is missing the CSS letter-spacing operator")
	}
	if !bytes.Contains(pdfData.Bytes(), []byte("FIELD OPERATIONS")) {
		t.Fatal("PDF masthead is missing text-transform uppercase output")
	}

	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(strings.ToLower(op.Text), "field operations") {
			if op.TextTransform != textTransformUppercase {
				t.Fatalf("masthead text-transform = %q, want uppercase", op.TextTransform)
			}

			want := 0.13 * 7.5
			if math.Abs(op.LetterSpacing-want) > 0.001 {
				t.Fatalf("masthead letter spacing = %.3fpt, want %.3fpt", op.LetterSpacing, want)
			}
			return
		}
	}

	t.Fatal("masthead text op not found")
}
