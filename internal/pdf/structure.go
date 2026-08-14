package pdf

import (
	"fmt"
	"strconv"
	"strings"
)

// StructType identifies an ISO 32000-1 standard structure element type.
type StructType string

const (
	// StructTypeDocument is the root grouping element for the document content.
	StructTypeDocument StructType = "Document"
	// StructTypeH1 is a first-level heading.
	StructTypeH1 StructType = "H1"
	// StructTypeH2 is a second-level heading.
	StructTypeH2 StructType = "H2"
	// StructTypeH3 is a third-level heading.
	StructTypeH3 StructType = "H3"
	// StructTypeH4 is a fourth-level heading.
	StructTypeH4 StructType = "H4"
	// StructTypeH5 is a fifth-level heading.
	StructTypeH5 StructType = "H5"
	// StructTypeH6 is a sixth-level heading.
	StructTypeH6 StructType = "H6"
	// StructTypeP is a paragraph.
	StructTypeP StructType = "P"
	// StructTypeTable is a table.
	StructTypeTable StructType = "Table"
	// StructTypeTR is a table row.
	StructTypeTR StructType = "TR"
	// StructTypeTH is a table header cell.
	StructTypeTH StructType = "TH"
	// StructTypeTD is a table data cell.
	StructTypeTD StructType = "TD"
	// StructTypeL is a list.
	StructTypeL StructType = "L"
	// StructTypeLI is a list item.
	StructTypeLI StructType = "LI"
	// StructTypeFigure is an illustration or graphic image.
	StructTypeFigure StructType = "Figure"
	// StructTypeLink is an interactive hypertext or internal link.
	StructTypeLink StructType = "Link"
	// StructTypeSpan is an inline text container.
	StructTypeSpan StructType = "Span"
	// StructTypeDiv is a generic block container.
	StructTypeDiv StructType = "Div"
	// StructTypeArtifact is non-structural pagination or background content.
	StructTypeArtifact StructType = "Artifact"
)

// Standard structure element type aliases.
const (
	StructDocument = StructTypeDocument
	StructH1       = StructTypeH1
	StructH2       = StructTypeH2
	StructH3       = StructTypeH3
	StructH4       = StructTypeH4
	StructH5       = StructTypeH5
	StructH6       = StructTypeH6
	StructP        = StructTypeP
	StructTable    = StructTypeTable
	StructTR       = StructTypeTR
	StructTH       = StructTypeTH
	StructTD       = StructTypeTD
	StructL        = StructTypeL
	StructLI       = StructTypeLI
	StructFigure   = StructTypeFigure
	StructLink     = StructTypeLink
	StructSpan     = StructTypeSpan
	StructDiv      = StructTypeDiv
	StructArtifact = StructTypeArtifact
)

// StructTreeRoot represents the root of a document's logical structure tree.
type StructTreeRoot struct {
	doc      *Document
	ref      objRef
	Children []*StructElem
}

// AddChild appends a top-level StructElem (typically /Document) to the root.
func (r *StructTreeRoot) AddChild(child *StructElem) {
	if child == nil {
		return
	}

	child.doc = r.doc
	child.parent = nil
	r.Children = append(r.Children, child)
}

// NewChild creates and appends a top-level StructElem to the root.
func (r *StructTreeRoot) NewChild(tag StructType) *StructElem {
	elem := &StructElem{ //nolint:exhaustruct // intentional zero-value fields
		doc: r.doc,
		Tag: tag,
	}
	r.AddChild(elem)

	return elem
}

// Ref returns the indirect object reference string for the structure tree root.
func (r *StructTreeRoot) Ref() string {
	return r.ref.String()
}

// StructElem represents a single structure element in the ISO 32000-1 structure tree.
type StructElem struct {
	doc        *Document
	ref        objRef
	parent     *StructElem
	Tag        StructType
	Page       *Page
	Kids       []*StructElem
	MCIDs      []int
	Alt        string
	Lang       string
	Title      string
	ActualText string
	AnnotRef   objRef
}

// AddChild appends a child StructElem to this element.
func (e *StructElem) AddChild(child *StructElem) {
	if child == nil {
		return
	}

	child.doc = e.doc
	child.parent = e
	e.Kids = append(e.Kids, child)
}

// NewChild creates and appends a child StructElem with the given tag.
func (e *StructElem) NewChild(tag StructType) *StructElem {
	elem := &StructElem{ //nolint:exhaustruct // intentional zero-value fields
		doc:    e.doc,
		Tag:    tag,
		parent: e,
	}
	e.Kids = append(e.Kids, elem)

	return elem
}

// SetPage assigns the page where this element's content appears.
func (e *StructElem) SetPage(page *Page) {
	e.Page = page
}

// SetAlt sets the alternative description text (for Figure, etc.).
func (e *StructElem) SetAlt(alt string) {
	e.Alt = alt
}

// SetLang sets the natural language tag for this element.
func (e *StructElem) SetLang(lang string) {
	e.Lang = lang
}

// SetTitle sets the element title text.
func (e *StructElem) SetTitle(title string) {
	e.Title = title
}

// SetActualText sets the exact text replacement for accessibility tools.
func (e *StructElem) SetActualText(text string) {
	e.ActualText = text
}

// SetAnnotRef links a link annotation indirect object to this StructElem.
func (e *StructElem) SetAnnotRef(ref objRef) {
	e.AnnotRef = ref
}

// SetAnnotation links the annotation at annotIndex on page to this StructElem.
func (e *StructElem) SetAnnotation(page *Page, annotIndex int) {
	if page == nil || annotIndex < 0 || annotIndex >= len(page.annots) {
		return
	}

	e.Page = page
	e.AnnotRef = page.annots[annotIndex].annotRef
}

// AddMCID appends a marked content identifier belonging to this element.
func (e *StructElem) AddMCID(mcid int) {
	e.MCIDs = append(e.MCIDs, mcid)
}

// Ref returns the indirect object reference string for this StructElem.
func (e *StructElem) Ref() string {
	return e.ref.String()
}

// CreateStructTreeRoot creates (or returns) the document's structure tree root.
// When the document policy does not specify PDF/UA-1, this returns nil.
func (d *Document) CreateStructTreeRoot() *StructTreeRoot {
	if !d.policy.IsPDFUA1() {
		return nil
	}

	if d.structTreeRoot == nil {
		d.structTreeRoot = &StructTreeRoot{ //nolint:exhaustruct // intentional zero-value fields
			doc: d,
		}
	}

	return d.structTreeRoot
}

// StructTreeRoot returns the document's logical structure tree root (nil if none).
func (d *Document) StructTreeRoot() *StructTreeRoot {
	return d.structTreeRoot
}

// AllocMCID allocates the next sequential Marked Content ID on this page
// and associates it with the owning StructElem for ParentTree resolution.
// Returns -1 if the document policy does not enable PDF/UA-1 structure.
func (p *Page) AllocMCID(elem *StructElem) int {
	if elem == nil || p.doc == nil || !p.doc.policy.IsPDFUA1() {
		return -1
	}

	mcid := len(p.mcids)
	p.mcids = append(p.mcids, elem)
	elem.MCIDs = append(elem.MCIDs, mcid)

	if elem.Page == nil {
		elem.Page = p
	}

	return mcid
}

// MCIDCount returns the number of Marked Content IDs allocated on this page.
func (p *Page) MCIDCount() int {
	return len(p.mcids)
}

// assignStructElemRefs recursively allocates indirect object IDs for all StructElems.
func assignStructElemRefs(doc *Document, elem *StructElem) {
	elem.ref = doc.newObject()
	for _, child := range elem.Kids {
		assignStructElemRefs(doc, child)
	}
}

// finalizeStructure creates and serializes StructTreeRoot, ParentTree, and all StructElems.
func (d *Document) finalizeStructure() error {
	if !d.policy.IsPDFUA1() {
		return nil
	}

	if d.structTreeRoot == nil {
		d.structTreeRoot = &StructTreeRoot{doc: d} //nolint:exhaustruct // default structure root
		_ = d.structTreeRoot.NewChild(StructTypeDocument)
	}

	d.structTreeRootRef = d.newObject()
	d.structTreeRoot.ref = d.structTreeRootRef

	for _, child := range d.structTreeRoot.Children {
		assignStructElemRefs(d, child)
	}

	d.buildParentTree()
	d.serializeStructTreeRoot()

	for _, child := range d.structTreeRoot.Children {
		if err := d.serializeStructElem(child, d.structTreeRootRef); err != nil {
			return err
		}
	}

	return nil
}

// buildParentTree constructs the ParentTree number tree dictionary mapping
// page StructParents indices to arrays of StructElem object references per MCID.
func (d *Document) buildParentTree() {
	numsParts := make([]string, 0, len(d.pages))
	nextStructParents := 0

	for _, page := range d.pages {
		if len(page.mcids) == 0 {
			continue
		}

		structParents := nextStructParents
		nextStructParents++
		page.structParents = structParents
		page.hasStructParents = true

		elemRefs := make([]string, 0, len(page.mcids))
		for _, elem := range page.mcids {
			elemRefs = append(elemRefs, elem.ref.String())
		}

		numsParts = append(numsParts, fmt.Sprintf("%d [ %s ]", structParents, strings.Join(elemRefs, " ")))
	}

	if len(numsParts) > 0 {
		d.parentTreeRef = d.newObject()
		d.setDict(d.parentTreeRef, fmt.Sprintf("<< /Nums [ %s ] >>", strings.Join(numsParts, " ")))
	}
}

// serializeStructTreeRoot serializes the /StructTreeRoot dictionary.
func (d *Document) serializeStructTreeRoot() {
	rootKids := make([]string, 0, len(d.structTreeRoot.Children))
	for _, child := range d.structTreeRoot.Children {
		rootKids = append(rootKids, child.ref.String())
	}

	var rootDict dict
	rootDict = rootDict.add("/Type", "/StructTreeRoot")

	if len(rootKids) == 1 {
		rootDict = rootDict.add("/K", rootKids[0])
	} else if len(rootKids) > 1 {
		rootDict = rootDict.add("/K", "["+strings.Join(rootKids, " ")+"]")
	}

	if d.parentTreeRef != 0 {
		rootDict = rootDict.add("/ParentTree", d.parentTreeRef.String())
	}

	d.setDict(d.structTreeRootRef, rootDict.String())
}

func (d *Document) formatStructKids(elem *StructElem) string {
	if len(elem.Kids) > 0 {
		kidRefs := make([]string, 0, len(elem.Kids))
		for _, kid := range elem.Kids {
			kidRefs = append(kidRefs, kid.ref.String())
		}

		if len(kidRefs) == 1 {
			return kidRefs[0]
		}

		return "[" + strings.Join(kidRefs, " ") + "]"
	}

	kItems := make([]string, 0, len(elem.MCIDs)+1)
	for _, mcid := range elem.MCIDs {
		kItems = append(kItems, strconv.Itoa(mcid))
	}

	if elem.AnnotRef != 0 {
		pgRef := ""
		if elem.Page != nil {
			pgRef = " /Pg " + elem.Page.ref.String()
		}

		kItems = append(kItems, fmt.Sprintf("<< /Type /OBJR /Obj %s%s >>", elem.AnnotRef.String(), pgRef))
	}

	if len(kItems) == 1 {
		return kItems[0]
	} else if len(kItems) > 1 {
		return "[" + strings.Join(kItems, " ") + "]"
	}

	return ""
}

// serializeStructElem serializes a single StructElem dictionary and its children.
func (d *Document) serializeStructElem(elem *StructElem, parentRef objRef) error {
	var elemDict dict
	elemDict = elemDict.add("/Type", "/StructElem").
		add("/S", "/"+string(elem.Tag)).
		add("/P", parentRef.String())

	if elem.Page != nil {
		elemDict = elemDict.add("/Pg", elem.Page.ref.String())
	}

	if kidsStr := d.formatStructKids(elem); kidsStr != "" {
		elemDict = elemDict.add("/K", kidsStr)
	}

	if elem.Alt != "" {
		elemDict = elemDict.add("/Alt", d.encodeTextString(elem.Alt))
	}

	if elem.Lang != "" {
		elemDict = elemDict.add("/Lang", pdfString(elem.Lang))
	}

	if elem.Title != "" {
		elemDict = elemDict.add("/Title", d.encodeTextString(elem.Title))
	}

	if elem.ActualText != "" {
		elemDict = elemDict.add("/ActualText", d.encodeTextString(elem.ActualText))
	}

	d.setDict(elem.ref, elemDict.String())

	for _, kid := range elem.Kids {
		if err := d.serializeStructElem(kid, elem.ref); err != nil {
			return err
		}
	}

	return nil
}
