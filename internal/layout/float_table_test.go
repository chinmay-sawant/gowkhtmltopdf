//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"math"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// TestFloatInsideTableCell: float packs inside a td BFC; neighbor cell uses
// vertical-align:top; clear:both on a block advances past the float.
func TestFloatInsideTableCell(t *testing.T) { //nolint:cyclop,funlen
	t.Parallel()

	cssSheet := sheet(t, `
table { width: 320pt; border-collapse: collapse }
td { vertical-align: top; border: 1px solid #000; padding: 4pt }
.icon { float: left; width: 40pt; height: 40pt; background-color: #ccc }
.clear { clear: both }
body { width: 400pt }
`)
	res := layoutHTML(t, `<html><body>
<table><tr>
<td class="cell"><div class="icon">I</div>wrap beside icon<div class="clear"></div>below</td>
<td class="neighbor">neighbor top</td>
</tr></table>
</body></html>`, cssSheet)

	var icon, clearBox, cell *box

	var walk func(b *box)
	walk = func(boxNode *box) {
		if boxNode.node != nil {
			switch boxNode.node.Attribute("class") {
			case "icon":
				icon = boxNode
			case "clear":
				clearBox = boxNode
			case "cell":
				cell = boxNode
			}
		}

		for _, c := range boxNode.children {
			walk(c)
		}
	}
	walk(res.root)

	if icon == nil || clearBox == nil || cell == nil {
		t.Fatalf("missing boxes icon=%v clear=%v cell=%v", icon != nil, clearBox != nil, cell != nil)
	}
	// Icon is a left float inside the cell content edge.
	if icon.x+0.01 < cell.x {
		t.Errorf("icon x=%.1f should be >= cell x=%.1f", icon.x, cell.x)
	}
	// Wrapping text sits to the right of the float (fill + text ops).
	var wrapX float64

	var sawWrap bool

	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "wrap") {
			wrapX, sawWrap = op.X, true
		}
	}

	if !sawWrap {
		t.Fatal("expected wrapping text op")
	}

	if wrapX+0.01 < icon.x+icon.w {
		t.Errorf("wrap text x=%.1f should be >= icon right edge %.1f", wrapX, icon.x+icon.w)
	}
	// clear:both block sits below the float margin box.
	if clearBox.y+0.01 < icon.y+icon.height {
		t.Errorf("clear y=%.1f should be >= icon bottom %.1f", clearBox.y, icon.y+icon.height)
	}
	// Neighbor cell shares the row top (vertical-align:top).
	var neighborY float64

	for _, op := range res.Ops {
		if op.Kind == OpText && op.Text == "neighbor" {
			neighborY = op.Y
		}
	}

	if math.Abs(neighborY-icon.y) > 20 {
		t.Errorf("neighbor text y=%.1f should be near icon top y=%.1f (top-aligned row)", neighborY, icon.y)
	}
}

// TestTableClearsFloat: an in-flow table after a float always clears below
// (no shrink-beside). Unsupported: Chrome-style squeeze beside the float.
func TestTableClearsFloat(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { width: 400pt }
.f { float: left; width: 100pt; height: 60pt; background-color: #ddd }
table { width: 200pt; border-collapse: collapse }
td { border: 1px solid #000; padding: 4pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="f">F</div>
<table><tr><td>TABLE</td></tr></table>
</body></html>`, cssSheet)

	var face, tblB *box

	var walk func(b *box)
	walk = func(boxNode *box) {
		if boxNode.node != nil {
			if boxNode.node.Attribute("class") == "f" {
				face = boxNode
			}

			if boxNode.node.Name == displayTable {
				tblB = boxNode
			}
		}

		for _, c := range boxNode.children {
			walk(c)
		}
	}
	walk(res.root)

	if face == nil || tblB == nil {
		t.Fatalf("missing boxes float=%v table=%v", face != nil, tblB != nil)
	}

	bottom := face.y + face.height
	if tblB.y+0.01 < bottom {
		t.Errorf("table y=%.1f should clear below float bottom %.1f (got shrink-beside)", tblB.y, bottom)
	}
	// Full containing-block origin (not squeezed to the right of the float).
	if tblB.x > face.x+face.w-1 {
		t.Errorf("table x=%.1f should start at content edge, not beside float (float right=%.1f)", tblB.x, face.x+face.w)
	}
}

// TestFloatOnTableCellBlockifies: float ≠ none on table-cell becomes block
// (CSS2.1 §9.7) and lays out without panic; anonymous table grid is best-effort.
func TestFloatOnTableCellBlockifies(t *testing.T) { //nolint:cyclop,funlen
	t.Parallel()

	cssSheet := sheet(t, `
.cell { float: left; display: table-cell; width: 80pt; height: 30pt; background-color: #eee }
.row { float: left; display: table-row; width: 50pt; height: 20pt; background-color: #ddd }
`)
	root := mustParse(t, `<html><body>
<div class="cell">A</div>
<div class="row">R</div>
<p>after</p>
</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "", testViewport, 800)

	var find func(n *html.Node, class string) *html.Node
	find = func(n *html.Node, class string) *html.Node {
		if n.Type == html.ElementNode && n.Attribute("class") == class {
			return n
		}

		for _, c := range n.Children {
			if hit := find(c, class); hit != nil {
				return hit
			}
		}

		return nil
	}

	cell := find(root, "cell")
	row := find(root, "row")

	if cell == nil || row == nil {
		t.Fatal("missing nodes")
	}

	if styles[cell].Display != displayBlock || styles[cell].Float != floatLeft {
		t.Fatalf("table-cell+float: display=%q float=%q, want block/left", styles[cell].Display, styles[cell].Float)
	}

	if styles[row].Display != displayBlock || styles[row].Float != floatLeft {
		t.Fatalf("table-row+float: display=%q float=%q, want block/left", styles[row].Display, styles[row].Float)
	}

	// Layout must not panic; floats participate as blocks.
	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: testViewport, Height: 800, Sheets: []*css.Stylesheet{cssSheet}, Background: true,
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	var aBox *box

	var walk func(b *box)
	walk = func(b *box) {
		if b.node != nil && b.node.Attribute("class") == "cell" {
			aBox = b
		}

		for _, c := range b.children {
			walk(c)
		}
	}
	walk(res.root)

	if aBox == nil {
		t.Fatal("expected floated cell box")
	}

	if aBox.kind == displayTable {
		t.Fatalf("blockified float should not build as empty table, kind=%s w=%.1f", aBox.kind, aBox.w)
	}

	if aBox.w < 70 {
		t.Errorf("floated block width=%.1f, want ~80", aBox.w)
	}
}

// TestFloatedTableKeepsDisplay: float on <table> keeps display:table (wrapper).
func TestFloatedTableKeepsDisplay(t *testing.T) {
	t.Parallel()

	s := sheet(t, `table.infobox { float: right; width: 100pt }`)
	root := mustParse(t, `<html><body><table class="infobox"><tr><td>X</td></tr></table></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{s}, "", testViewport, 800)

	var table *html.Node

	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Name == displayTable {
			table = n
		}

		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)

	if table == nil {
		t.Fatal("no table")
	}

	st := styles[table]
	if st.Display != displayTable || st.Float != floatRight {
		t.Fatalf("floated table: display=%q float=%q, want table/right", st.Display, st.Float)
	}
}
