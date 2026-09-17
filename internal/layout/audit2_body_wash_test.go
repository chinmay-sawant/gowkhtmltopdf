package layout

import (
	"fmt"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// gobyexample-8: a page created by paginating past the body's layout bottom
// must still carry the body/html paper wash to the content-box bottom.
// Mirror TestFixture56ShortPageKeepsPaperWashToBottom with a tinted body.
func TestBodyPaperWashCoversContinuationPage(t *testing.T) {
	t.Parallel()

	const (
		pageW  = 400.0
		pageH  = 200.0
		margin = 20.0
	)

	contentH := pageH - 2*margin
	res, doc := layoutBodyWashFixture(t, pageW, pageH, margin, contentH)

	if doc.PageCount() < 3 {
		t.Fatalf("page count = %d, want >= 3 so the footer sits alone on a continuation page",
			doc.PageCount())
	}

	lastPage := doc.PageCount() - 1
	pageTop := float64(lastPage) * contentH
	pageBot := pageTop + contentH
	bestBot := paperWashBottomOnPage(res, pageTop, pageBot)

	if bestBot < pageBot-2 {
		t.Fatalf("continuation page %d paper wash bottom = %.2f, want >= %.2f (full content height)",
			lastPage+1, bestBot, pageBot-2)
	}
}

func layoutBodyWashFixture(t *testing.T, pageW, pageH, margin, contentH float64) (*Result, *pdf.Document) {
	t.Helper()

	var blocks strings.Builder

	for blockIdx := range 2 {
		blocks.WriteString(fmt.Sprintf(`<div class="tall">BLOCK%d</div>`, blockIdx))
	}

	cssSheet := sheet(t, `
html, body { margin: 0; background: #dddddd }
body { font-size: 12pt }
.tall { height: 150pt; margin: 0 }
.foot { page-break-before: always }
`)

	root := mustParse(t, `<html><body>`+blocks.String()+
		`<div class="foot">FOOTER</div></body></html>`)

	res, err := Layout(root, Options{
		Width: pageW - 2*margin, Height: contentH,
		Background: true, Sheets: []*css.Stylesheet{cssSheet}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}

	doc := pdf.NewDocument()
	if err := Paint(doc, res, PaintOptions{
		PageWidth: pageW, PageHeight: pageH,
		MarginTop: margin, MarginBottom: margin, MarginLeft: margin, MarginRight: margin,
	}); err != nil {
		t.Fatal(err)
	}

	return res, doc
}

// paperWashBottomOnPage returns the lowest bottom edge of a #dddddd paper-wash
// fill that intersects [pageTop, pageBot].
func paperWashBottomOnPage(res *Result, pageTop, pageBot float64) float64 {
	const paperR, paperG, paperB = 0.867, 0.867, 0.867

	bestBot := 0.0

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpFillRect {
			continue
		}

		if absRGB(paintOp.R, paperR) || absRGB(paintOp.G, paperG) || absRGB(paintOp.B, paperB) {
			continue
		}

		if paintOp.Y+paintOp.H <= pageTop+1 || paintOp.Y >= pageBot-1 {
			continue
		}

		if bot := paintOp.Y + paintOp.H; bot > bestBot {
			bestBot = bot
		}
	}

	return bestBot
}

func absRGB(got, want float64) bool {
	d := got - want
	if d < 0 {
		d = -d
	}

	return d > 0.05
}
