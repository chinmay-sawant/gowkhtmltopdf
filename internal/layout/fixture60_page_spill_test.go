package layout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

func TestFixture60BackgroundImagesStayWithTheirRows(t *testing.T) {
	rootDir, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(rootDir, "testdata/golden/fixture-60-implemented-props-a.html"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := html.Parse(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	styleText := extractStyleContent(doc)
	sheet, err := css.Parse(styleText)
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Join(rootDir, "testdata/golden")
	margin := 12 * 72 / 25.4
	pageW, pageH := 595.28, 841.89
	contentW := pageW - 2*margin
	contentH := pageH - 2*margin

	res, err := Layout(doc, Options{ //nolint:exhaustruct
		Width: contentW, Height: contentH, Background: true, Media: "print", Zoom: 0.995,
		Sheets: []*css.Stylesheet{sheet},
		Images: func(src string) ([]byte, error) {
			src = strings.TrimPrefix(src, "file://")
			if strings.HasPrefix(src, "data:") {
				return nil, os.ErrNotExist
			}
			if !filepath.IsAbs(src) {
				src = filepath.Join(base, src)
			}
			return os.ReadFile(src)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	pdfDoc := pdf.NewDocument()
	if err := Paint(pdfDoc, res, PaintOptions{ //nolint:exhaustruct
		PageWidth: pageW, PageHeight: pageH,
		MarginTop: margin, MarginBottom: margin, MarginLeft: margin, MarginRight: margin,
	}); err != nil {
		t.Fatal(err)
	}

	// Find Y of "background-image" property text and Y of align-content text
	var bgImageY, alignY float64
	var bgImagePage, alignPage int
	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}
		if op.Text == "background-image" {
			bgImageY = op.Y
			bgImagePage = int(op.Y / contentH)
		}
		if op.Text == "align-content" {
			alignY = op.Y
			alignPage = int(op.Y / contentH)
		}
	}
	t.Logf("align-content y=%.1f page=%d", alignY, alignPage)
	t.Logf("background-image y=%.1f page=%d", bgImageY, bgImagePage)

	// Background images whose Y falls on align-content's page but below would be OK;
	// images with Y near align-content while belonging to background rows are spill.
	// Count IsBackground images on align page vs bg page.
	imgsOnAlignPage := 0
	imgsOnBgPage := 0
	for _, op := range res.Ops {
		if op.Kind != OpImage {
			continue
		}
		p := int(op.Y / contentH)
		if p == alignPage {
			imgsOnAlignPage++
			t.Logf("  img on align page: y=%.1f x=%.1f w=%.1f h=%.1f bg=%v", op.Y, op.X, op.W, op.H, op.IsBackground)
		}
		if p == bgImagePage {
			imgsOnBgPage++
		}
	}
	t.Logf("images on align page=%d bg page=%d", imgsOnAlignPage, imgsOnBgPage)

	// align-content page should not have many background logos (only maybe filter img)
	if imgsOnAlignPage > 3 {
		t.Fatalf("too many images on align-content page (%d); likely Effect spill across rows", imgsOnAlignPage)
	}
}
