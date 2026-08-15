//nolint:testpackage // tests exercise unexported layout pagination internals
package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

//nolint:lll,wsl // fixture assertion intentionally follows pagination ownership
func TestFixture56DomainSectionOmitsAccentTopRail(t *testing.T) { //nolint:paralleltest // fixture uses shared font state
	root, sheet := loadFixture56(t)
	const (
		pageWidth  = 595.28
		pageHeight = 841.89
		margin     = 12 * 72.0 / 25.4
	)
	contentHeight := pageHeight - 2*margin

	res, err := Layout(root, Options{ //nolint:exhaustruct // fixture uses the standard print geometry
		Width: pageWidth - 2*margin, Height: contentHeight,
		Background: true, Sheets: []*css.Stylesheet{sheet}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := Paint(pdf.NewDocument(), res, PaintOptions{
		PageWidth: pageWidth, PageHeight: pageHeight,
		MarginTop: margin, MarginBottom: margin,
		MarginLeft: margin, MarginRight: margin,
	}); err != nil {
		t.Fatal(err)
	}

	section := fixture56Node(root, func(node *html.Node) bool { return node.Attribute("id") == "domain-04" })
	boxNode := fixture56BoxByNode(res.root, section)
	if boxNode == nil {
		t.Fatal("domain-04 has no layout box")
	}

	topRails := make([]Op, 0, 1)
	for index := boxNode.opStart; index <= boxNode.opEnd && index < len(res.Ops); index++ {
		op := res.Ops[index]
		if op.Kind == OpStrokeRect && op.StrokeMask == StrokeMaskTop {
			topRails = append(topRails, op)
		}
	}
	if len(topRails) != 0 {
		t.Fatalf("domain-04 accent top rail fragments = %d, want none: %+v", len(topRails), topRails)
	}
}
