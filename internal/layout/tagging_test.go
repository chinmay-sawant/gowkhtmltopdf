//nolint:testpackage,exhaustruct,cyclop,funlen,varnamelen,wsl,lll,exhaustive,nlreturn // tests exercise unexported tagging internals
package layout

import (
	"testing"

	"gowkhtmltopdf/internal/pdf"
)

//nolint:cyclop,funlen,varnamelen,wsl,lll,exhaustive,nlreturn,exhaustruct
func TestStructureTreeTableHierarchy(t *testing.T) {
	t.Parallel()

	doc, err := pdf.NewDocumentWithPolicy(pdf.WriterPolicy{
		Version:            pdf.PDF17,
		ConformanceProfile: pdf.ProfilePDFUA1,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy: %v", err)
	}

	htmlStr := `<html><body>
<table>
  <caption>Sales Summary</caption>
  <tr><th scope="col">Quarter</th><th scope="col">Revenue</th></tr>
  <tr><th scope="row">Q1</th><td>$10,000</td></tr>
</table>
</body></html>`

	cssSheet := sheet(t, `
table { width: 300pt; border-collapse: collapse }
th, td { border: 1px solid #ccc; padding: 4pt }
`)
	res := layoutHTML(t, htmlStr, cssSheet)

	if err := buildStructureTree(doc, res); err != nil {
		t.Fatalf("buildStructureTree: %v", err)
	}

	root := doc.StructTreeRoot()
	if root == nil || len(root.Children) == 0 {
		t.Fatal("expected non-empty StructTreeRoot")
	}

	docElem := root.Children[0]
	if docElem.Tag != pdf.StructDocument {
		t.Fatalf("expected root child Document, got %s", docElem.Tag)
	}

	var tableElem *pdf.StructElem
	for _, child := range docElem.Kids {
		if child.Tag == pdf.StructTable {
			tableElem = child
			break
		}
	}

	if tableElem == nil {
		t.Fatal("missing Table StructElem under Document")
	}

	// 1. Table must contain ONLY Caption and TR elements (NO direct TH or TD)
	var captionElem *pdf.StructElem
	var trElems []*pdf.StructElem

	for _, child := range tableElem.Kids {
		switch child.Tag {
		case pdf.StructCaption:
			captionElem = child
		case pdf.StructTR:
			trElems = append(trElems, child)
		case pdf.StructTH, pdf.StructTD:
			t.Errorf("Table contains direct %s child which violates PDF/UA-1 structure rules", child.Tag)
		default:
			t.Errorf("Table contains unexpected child element %s", child.Tag)
		}
	}

	if captionElem == nil {
		t.Error("missing Caption StructElem under Table")
	}

	if len(trElems) != 2 {
		t.Fatalf("expected 2 TR elements under Table, got %d", len(trElems))
	}

	// 2. First TR must contain 2 TH elements with Scope = Column
	tr0 := trElems[0]
	if len(tr0.Kids) != 2 {
		t.Fatalf("expected 2 cells in row 0, got %d", len(tr0.Kids))
	}
	for i, cell := range tr0.Kids {
		if cell.Tag != pdf.StructTH {
			t.Errorf("row 0 cell %d tag = %s, want TH", i, cell.Tag)
		}
		if cell.TableScope != "Column" {
			t.Errorf("row 0 cell %d TableScope = %q, want Column", i, cell.TableScope)
		}
	}

	// 3. Second TR must contain TH (Scope = Row) and TD
	tr1 := trElems[1]
	if len(tr1.Kids) != 2 {
		t.Fatalf("expected 2 cells in row 1, got %d", len(tr1.Kids))
	}
	if tr1.Kids[0].Tag != pdf.StructTH || tr1.Kids[0].TableScope != "Row" {
		t.Errorf("row 1 cell 0 tag = %s, scope = %q, want TH with Row scope", tr1.Kids[0].Tag, tr1.Kids[0].TableScope)
	}
	if tr1.Kids[1].Tag != pdf.StructTD {
		t.Errorf("row 1 cell 1 tag = %s, want TD", tr1.Kids[1].Tag)
	}
}

//nolint:cyclop,funlen,varnamelen,wsl,lll,exhaustive,nlreturn,exhaustruct
func TestStructureTreeListTagging(t *testing.T) {
	t.Parallel()

	doc, err := pdf.NewDocumentWithPolicy(pdf.WriterPolicy{
		Version:            pdf.PDF17,
		ConformanceProfile: pdf.ProfilePDFUA1,
	})
	if err != nil {
		t.Fatalf("NewDocumentWithPolicy: %v", err)
	}

	htmlStr := `<html><body>
<ul>
  <li>First item</li>
  <li>Second item with nested:
    <ul>
      <li>Nested child</li>
    </ul>
  </li>
</ul>
</body></html>`

	cssSheet := sheet(t, `ul { margin-left: 20pt } li { margin: 2pt 0 }`)
	res := layoutHTML(t, htmlStr, cssSheet)

	if err := buildStructureTree(doc, res); err != nil {
		t.Fatalf("buildStructureTree: %v", err)
	}

	root := doc.StructTreeRoot()
	docElem := root.Children[0]

	var topList *pdf.StructElem
	for _, child := range docElem.Kids {
		if child.Tag == pdf.StructL {
			topList = child
			break
		}
	}

	if topList == nil {
		t.Fatal("missing top-level L StructElem under Document")
	}

	// Top list must have 2 LI children
	if len(topList.Kids) != 2 {
		t.Fatalf("expected 2 LI elements under L, got %d", len(topList.Kids))
	}

	for idx, li := range topList.Kids {
		if li.Tag != pdf.StructLI {
			t.Errorf("child %d of L is %s, want LI", idx, li.Tag)
		}

		// Each LI must contain ONLY Lbl and/or LBody
		for _, liChild := range li.Kids {
			if liChild.Tag != pdf.StructLbl && liChild.Tag != pdf.StructLBody {
				t.Errorf("LI %d contains direct child %s instead of Lbl or LBody", idx, liChild.Tag)
			}
		}
	}

	// LI 1 has a nested list. The nested L MUST be inside LBody, NOT directly under LI!
	li1 := topList.Kids[1]
	var lbody1 *pdf.StructElem
	for _, c := range li1.Kids {
		if c.Tag == pdf.StructLBody {
			lbody1 = c
			break
		}
	}

	if lbody1 == nil {
		t.Fatal("LI 1 is missing LBody element")
	}

	var nestedL *pdf.StructElem
	for _, c := range lbody1.Kids {
		if c.Tag == pdf.StructL {
			nestedL = c
			break
		}
	}

	if nestedL == nil {
		t.Fatal("nested L must be inside LBody element under LI")
	}

	if len(nestedL.Kids) != 1 || nestedL.Kids[0].Tag != pdf.StructLI {
		t.Fatalf("nested L should have 1 LI child, got %d", len(nestedL.Kids))
	}
}

//nolint:cyclop,funlen,varnamelen,wsl,lll,exhaustive,nlreturn,exhaustruct
func TestStructureTreeHeadingNormalization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		html     string
		wantTags []pdf.StructType
	}{
		{
			name: "Normal sequence H1 -> H2 -> H3",
			html: `<html><body><h1>Title</h1><h2>Section</h2><h3>Subsection</h3></body></html>`,
			wantTags: []pdf.StructType{
				pdf.StructH1, pdf.StructH2, pdf.StructH3,
			},
		},
		{
			name: "Skipped H2: H1 -> H3 -> H5 normalized to H1 -> H2 -> H3",
			html: `<html><body><h1>Title</h1><h3>Skipped to H3</h3><h5>Skipped to H5</h5></body></html>`,
			wantTags: []pdf.StructType{
				pdf.StructH1, pdf.StructH2, pdf.StructH3,
			},
		},
		{
			name: "Starting with H3 normalized to H1 -> H2",
			html: `<html><body><h3>Leading H3</h3><h4>Leading H4</h4></body></html>`,
			wantTags: []pdf.StructType{
				pdf.StructH1, pdf.StructH2,
			},
		},
		{
			name: "Descending jumps clamped while ascents allowed H1 -> H3 -> H1 -> H2",
			html: `<html><body><h1>Title</h1><h3>Clamped H3 to H2</h3><h1>New Chapter</h1><h2>Section</h2></body></html>`,
			wantTags: []pdf.StructType{
				pdf.StructH1, pdf.StructH2, pdf.StructH1, pdf.StructH2,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc, err := pdf.NewDocumentWithPolicy(pdf.WriterPolicy{
				Version:            pdf.PDF17,
				ConformanceProfile: pdf.ProfilePDFUA1,
			})
			if err != nil {
				t.Fatalf("NewDocumentWithPolicy: %v", err)
			}

			res := layoutHTML(t, tc.html, sheet(t, ""))
			if err := buildStructureTree(doc, res); err != nil {
				t.Fatalf("buildStructureTree: %v", err)
			}

			root := doc.StructTreeRoot()
			docElem := root.Children[0]

			var gotTags []pdf.StructType
			for _, child := range docElem.Kids {
				switch child.Tag {
				case pdf.StructH1, pdf.StructH2, pdf.StructH3, pdf.StructH4, pdf.StructH5, pdf.StructH6:
					gotTags = append(gotTags, child.Tag)
				}
			}

			if len(gotTags) != len(tc.wantTags) {
				t.Fatalf("got %d headings %v, want %d headings %v", len(gotTags), gotTags, len(tc.wantTags), tc.wantTags)
			}

			for i := range gotTags {
				if gotTags[i] != tc.wantTags[i] {
					t.Errorf("heading %d = %s, want %s", i, gotTags[i], tc.wantTags[i])
				}
			}
		})
	}
}

func dumpStructTree(elem *pdf.StructElem, indent string) string {
	if elem == nil {
		return indent + "<nil>\n"
	}

	out := indent + string(elem.Tag) + "\n"
	for _, kid := range elem.Kids {
		out += dumpStructTree(kid, indent+"  ")
	}

	return out
}

func dumpBoxTree(b *box, indent string) string {
	if b == nil {
		return indent + "<nil>\n"
	}

	name := "(anon)"
	if b.node != nil {
		name = b.node.Name
		if name == "" {
			name = "(text)"
		}
	}

	out := indent + name + " kind=" + b.kind + "\n"
	for _, child := range b.children {
		out += dumpBoxTree(child, indent+"  ")
	}

	return out
}

func countStructTags(elem *pdf.StructElem, tag pdf.StructType) int {
	if elem == nil {
		return 0
	}

	n := 0
	if elem.Tag == tag {
		n++
	}

	for _, kid := range elem.Kids {
		n += countStructTags(kid, tag)
	}

	return n
}

func assertListHierarchy(t *testing.T, elem *pdf.StructElem) {
	t.Helper()

	if elem == nil {
		return
	}

	switch elem.Tag {
	case pdf.StructL:
		for i, kid := range elem.Kids {
			switch kid.Tag {
			case pdf.StructLI, pdf.StructL, pdf.StructCaption:
			default:
				t.Errorf("L child %d is %s, want LI, L, or Caption", i, kid.Tag)
			}
		}
	case pdf.StructLI:
		for i, kid := range elem.Kids {
			switch kid.Tag {
			case pdf.StructLbl, pdf.StructLBody, pdf.StructL:
			default:
				t.Errorf("LI child %d is %s, want Lbl, LBody, or L", i, kid.Tag)
			}
		}
	}

	for _, kid := range elem.Kids {
		assertListHierarchy(t, kid)
	}
}

//nolint:cyclop,funlen,varnamelen,wsl,lll,exhaustive,nlreturn,exhaustruct
func TestStructureTreeListLinkHierarchy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		html string
		css  string
	}{
		{
			name: "li wrapping a fragment link",
			html: `<html><body>
<ol>
  <li><a href="#one">One</a></li>
  <li><a href="#two">Two</a></li>
</ol>
</body></html>`,
			css: `ol { margin-left: 20pt } li { margin: 2pt 0 }`,
		},
		{
			name: "multicol toc list of links",
			html: `<html><body>
<ol class="toc-list">
  <li><a href="#one">One</a></li>
  <li><a href="#two">Two</a></li>
  <li><a href="#three">Three</a></li>
</ol>
</body></html>`,
			css: `.toc-list { columns: 2; column-gap: 30px; } .toc-list li { margin: 2px 0; }`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			doc, err := pdf.NewDocumentWithPolicy(pdf.WriterPolicy{
				Version:            pdf.PDF17,
				ConformanceProfile: pdf.ProfilePDFUA1,
			})
			if err != nil {
				t.Fatalf("NewDocumentWithPolicy: %v", err)
			}

			res := layoutHTML(t, tc.html, sheet(t, tc.css))
			if err := buildStructureTree(doc, res); err != nil {
				t.Fatalf("buildStructureTree: %v", err)
			}

			root := doc.StructTreeRoot()
			if root == nil || len(root.Children) == 0 {
				t.Fatal("expected non-empty StructTreeRoot")
			}

			assertListHierarchy(t, root.Children[0])

			if links := countStructTags(root.Children[0], pdf.StructLink); links == 0 {
				t.Error("expected at least one Link in the structure tree")
			}

			if t.Failed() {
				t.Logf("box tree:\n%s", dumpBoxTree(res.root, ""))
				t.Logf("struct tree:\n%s", dumpStructTree(root.Children[0], ""))
			}
		})
	}
}
