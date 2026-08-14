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
	// StructTypeLbl is a list item label (bullet or item number).
	StructTypeLbl StructType = "Lbl"
	// StructTypeLBody is a list item body container.
	StructTypeLBody StructType = "LBody"
	// StructTypeCaption is a caption element for a table or figure.
	StructTypeCaption StructType = "Caption"
	// StructTypeTHead is a table head row group.
	StructTypeTHead StructType = "THead"
	// StructTypeTBody is a table body row group.
	StructTypeTBody StructType = "TBody"
	// StructTypeTFoot is a table foot row group.
	StructTypeTFoot StructType = "TFoot"
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
	StructLbl      = StructTypeLbl
	StructLBody    = StructTypeLBody
	StructCaption  = StructTypeCaption
	StructTHead    = StructTypeTHead
	StructTBody    = StructTypeTBody
	StructTFoot    = StructTypeTFoot
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

// contentRef is one marked-content sequence owned by a StructElem on a page.
// MCIDs are page-local; multi-page elements must serialize as MCR dictionaries
// (ISO 32000-1 §14.7.4.2) rather than bare integers under a single /Pg.
type contentRef struct {
	page *Page
	mcid int
}

// StructElem represents a single structure element in the ISO 32000-1 structure tree.
type StructElem struct {
	doc        *Document
	ref        objRef
	parent     *StructElem
	Tag        StructType
	Page       *Page
	Kids       []*StructElem
	content    []contentRef
	Alt        string
	Lang       string
	Title      string
	ActualText string
	AnnotRef   objRef
	TableScope string
	// ListNumbering is the PDF List attribute value for /L elements
	// (Disc, Circle, Square, Decimal, UpperRoman, LowerRoman, UpperAlpha, LowerAlpha).
	// Required by PDF/UA-2 when list items contain /Lbl children.
	ListNumbering string
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

// SetObjRef links an annotation indirect object and page to this StructElem.
func (e *StructElem) SetObjRef(ref objRef, page *Page) {
	e.AnnotRef = ref
	if page != nil {
		e.Page = page
	}
}

// AddAnnot links an annotation indirect object and page to this StructElem.
func (e *StructElem) AddAnnot(ref objRef, page *Page) {
	e.AnnotRef = ref
	if page != nil {
		e.Page = page
	}
}

// SetTableScope sets the table header cell scope (/Column, /Row, or /Both) for a TH element.
func (e *StructElem) SetTableScope(scope string) {
	e.TableScope = scope
}

// SetListNumbering sets the List attribute /ListNumbering for an /L element
// (e.g. "Disc", "Decimal"). Empty leaves the attribute unset.
func (e *StructElem) SetListNumbering(value string) {
	e.ListNumbering = value
}

// AddMCID appends a marked content identifier belonging to this element.
// The MCID is associated with e.Page when set; prefer Page.AllocMCID so the
// page is recorded correctly for multi-page elements.
func (e *StructElem) AddMCID(mcid int) {
	e.content = append(e.content, contentRef{page: e.Page, mcid: mcid})
}

// Ref returns the indirect object reference string for this StructElem.
func (e *StructElem) Ref() string {
	return e.ref.String()
}

// CreateStructTreeRoot creates (or returns) the document's structure tree root.
// When the document policy does not specify PDF/UA-1 or PDF/UA-2, this returns nil.
func (d *Document) CreateStructTreeRoot() *StructTreeRoot {
	if !d.policy.IsPDFUA1() && !d.policy.IsPDFUA2() {
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

// HeadingStructElems returns all H1–H6 structure elements in document order.
// This is used by the outline builder to associate outline items with their
// heading StructElem for PDF/UA-2 structure destinations.
func (d *Document) HeadingStructElems() []*StructElem {
	if d.structTreeRoot == nil {
		return nil
	}

	var result []*StructElem

	var walk func(e *StructElem)
	walk = func(e *StructElem) {
		if isHeadingTag(e.Tag) {
			result = append(result, e)
		}

		for _, kid := range e.Kids {
			walk(kid)
		}
	}

	for _, child := range d.structTreeRoot.Children {
		walk(child)
	}

	return result
}

// isHeadingTag reports whether tag is one of H1–H6.
func isHeadingTag(tag StructType) bool {
	switch tag {
	case StructTypeH1, StructTypeH2, StructTypeH3, StructTypeH4, StructTypeH5, StructTypeH6:
		return true
	default:
		return false
	}
}

// AllocMCID allocates the next sequential Marked Content ID on this page
// and associates it with the owning StructElem for ParentTree resolution.
// Returns -1 if the document policy does not enable PDF/UA structure.
func (p *Page) AllocMCID(elem *StructElem) int {
	if elem == nil || p.doc == nil || (!p.doc.policy.IsPDFUA1() && !p.doc.policy.IsPDFUA2()) {
		return -1
	}

	mcid := len(p.mcids)
	p.mcids = append(p.mcids, elem)
	elem.content = append(elem.content, contentRef{page: p, mcid: mcid})

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

// pruneEmptyStructElems removes any unnecessary empty structure element subtree,
// while preserving required document, table, and list structural tags.
func pruneEmptyStructElems(elem *StructElem) bool {
	validKids := make([]*StructElem, 0, len(elem.Kids))

	for _, kid := range elem.Kids {
		if !pruneEmptyStructElems(kid) {
			validKids = append(validKids, kid)
		}
	}

	elem.Kids = validKids

	//nolint:exhaustive // explicit preserve list for table, list, and doc structural elements
	switch elem.Tag {
	case StructTypeDocument, StructTypeTable, StructTypeTR, StructTypeTH, StructTypeTD,
		StructTypeL, StructTypeLI, StructTypeLBody, StructTypeCaption:
		return false
	default:
		return len(elem.Kids) == 0 && len(elem.content) == 0 && elem.AnnotRef == 0
	}
}

// finalizeStructure creates and serializes StructTreeRoot, ParentTree, and all StructElems.
func (d *Document) finalizeStructure() error {
	if !d.policy.IsPDFUA1() && !d.policy.IsPDFUA2() {
		return nil
	}

	if d.policy.IsPDFUA2() {
		d.namespaceRef = d.newObject()
		d.setDict(d.namespaceRef, "<< /Type /Namespace /NS (http://iso.org/pdf2/ssn) >>")
	}

	if d.structTreeRoot == nil {
		d.structTreeRoot = &StructTreeRoot{doc: d} //nolint:exhaustruct // default structure root
		_ = d.structTreeRoot.NewChild(StructTypeDocument)
	}

	validRootKids := make([]*StructElem, 0, len(d.structTreeRoot.Children))

	for _, child := range d.structTreeRoot.Children {
		if !pruneEmptyStructElems(child) {
			validRootKids = append(validRootKids, child)
		}
	}

	if len(validRootKids) == 0 {
		docElem := d.structTreeRoot.NewChild(StructTypeDocument)
		validRootKids = []*StructElem{docElem}
	}

	d.structTreeRoot.Children = validRootKids

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
// page StructParents indices to arrays of StructElem object references per MCID,
// and annotation StructParent indices to owning StructElem references.
//
//nolint:cyclop,funlen,gocognit // ParentTree number tree construction over pages and annotations
func (d *Document) buildParentTree() {
	for _, page := range d.pages {
		for i := range page.annots {
			if page.annots[i].annotRef == 0 {
				page.annots[i].annotRef = d.newObject()
			}
		}
	}

	annotToElem := make(map[objRef]*StructElem)

	var collectAnnotElems func(e *StructElem)
	collectAnnotElems = func(e *StructElem) {
		if e.AnnotRef != 0 {
			annotToElem[e.AnnotRef] = e
		}

		for _, kid := range e.Kids {
			collectAnnotElems(kid)
		}
	}

	for _, child := range d.structTreeRoot.Children {
		collectAnnotElems(child)
	}

	var fallbackDocElem *StructElem
	if len(d.structTreeRoot.Children) > 0 {
		fallbackDocElem = d.structTreeRoot.Children[0]
	}

	for _, page := range d.pages {
		for i := range page.annots {
			ref := page.annots[i].annotRef
			if _, exists := annotToElem[ref]; !exists && fallbackDocElem != nil {
				linkElem := fallbackDocElem.NewChild(StructTypeLink)
				linkElem.SetObjRef(ref, page)
				assignStructElemRefs(d, linkElem)
				annotToElem[ref] = linkElem
			}
		}
	}

	numsParts := make([]string, 0, len(d.pages)+len(d.pages))
	nextStructParents := 0

	for _, page := range d.pages {
		if len(page.mcids) > 0 {
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

		for i := range page.annots {
			a := &page.annots[i]
			if elem, ok := annotToElem[a.annotRef]; ok {
				annotStructParent := nextStructParents
				nextStructParents++
				a.structParent = annotStructParent
				a.hasStructParent = true

				numsParts = append(numsParts, fmt.Sprintf("%d %s", annotStructParent, elem.ref.String()))
			}
		}
	}

	d.parentTreeNextKey = nextStructParents

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
		rootDict = rootDict.add("/ParentTree", d.parentTreeRef.String()).
			add("/ParentTreeNextKey", strconv.Itoa(d.parentTreeNextKey))
	}

	if d.policy.IsPDFUA2() && d.namespaceRef != 0 {
		rootDict = rootDict.add("/Namespaces", "["+d.namespaceRef.String()+"]")
	}

	d.setDict(d.structTreeRootRef, rootDict.String())
}

//nolint:cyclop // MCID vs MCR selection branches on elem content shape
func (d *Document) formatStructKids(elem *StructElem) string {
	kItems := make([]string, 0, len(elem.Kids)+len(elem.content)+1)
	for _, kid := range elem.Kids {
		kItems = append(kItems, kid.ref.String())
	}

	// Bare MCID integers are only valid relative to the element's single /Pg.
	// Content spanning multiple pages (or a page other than /Pg) must use MCR
	// dictionaries so each MCID is bound to its owning page.
	useMCR := contentNeedsMCR(elem)

	for _, contentRef := range elem.content {
		if contentRef.page == nil {
			kItems = append(kItems, strconv.Itoa(contentRef.mcid))

			continue
		}

		if useMCR || contentRef.page != elem.Page {
			kItems = append(kItems, fmt.Sprintf(
				"<< /Type /MCR /Pg %s /MCID %d >>", contentRef.page.ref.String(), contentRef.mcid,
			))
		} else {
			kItems = append(kItems, strconv.Itoa(contentRef.mcid))
		}
	}

	if elem.AnnotRef != 0 {
		pgRef := ""

		targetPage := elem.Page
		if targetPage == nil && elem.parent != nil {
			targetPage = elem.parent.Page
		}

		if targetPage != nil {
			pgRef = " /Pg " + targetPage.ref.String()
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

// contentNeedsMCR reports whether elem's marked content spans more than one page.
func contentNeedsMCR(elem *StructElem) bool {
	if elem == nil || len(elem.content) == 0 {
		return false
	}

	var first *Page

	for _, contentRef := range elem.content {
		if contentRef.page == nil {
			continue
		}

		if first == nil {
			first = contentRef.page

			continue
		}

		if contentRef.page != first {
			return true
		}
	}

	return false
}

// resolveListNumbering returns the ListNumbering name to emit on an /L element.
// Explicit values win; otherwise under PDF/UA-2, lists that have Lbl children
// default to Disc so validators do not see an implicit None.
func resolveListNumbering(elem *StructElem) string {
	if elem == nil || elem.Tag != StructTypeL {
		return ""
	}

	if elem.ListNumbering != "" {
		return elem.ListNumbering
	}

	if elem.doc == nil || !elem.doc.policy.IsPDFUA2() {
		return ""
	}

	if listHasLbl(elem) {
		return "Disc"
	}

	return ""
}

func listHasLbl(elem *StructElem) bool {
	if elem == nil {
		return false
	}

	for _, kid := range elem.Kids {
		if kid.Tag == StructTypeLbl {
			return true
		}

		if kid.Tag == StructTypeLI {
			for _, grand := range kid.Kids {
				if grand.Tag == StructTypeLbl {
					return true
				}
			}
		}
	}

	return false
}

// serializeStructElem serializes a single StructElem dictionary and its children.
//
//nolint:cyclop // sequential structure element serialization
func (d *Document) serializeStructElem(elem *StructElem, parentRef objRef) error {
	var elemDict dict
	elemDict = elemDict.add("/Type", "/StructElem").
		add("/S", "/"+string(elem.Tag)).
		add("/P", parentRef.String())

	if elem.Page != nil {
		elemDict = elemDict.add("/Pg", elem.Page.ref.String())
	}

	kidsStr := d.formatStructKids(elem)
	if kidsStr == "" {
		kidsStr = "[]"
	}

	elemDict = elemDict.add("/K", kidsStr)

	if d.policy.IsPDFUA2() && elem.Tag == StructTypeDocument && d.namespaceRef != 0 {
		elemDict = elemDict.add("/NS", d.namespaceRef.String())
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

	if elem.TableScope != "" {
		elemDict = elemDict.add("/A", fmt.Sprintf("<< /O /Table /Scope /%s >>", elem.TableScope))
	} else if elem.Tag == StructTypeTH {
		elemDict = elemDict.add("/A", "<< /O /Table /Scope /Column >>")
	} else if listNumbering := resolveListNumbering(elem); listNumbering != "" {
		// PDF/UA-2 8.2.5.25: when LI contains Lbl, ListNumbering must be present and not None.
		elemDict = elemDict.add("/A", fmt.Sprintf("<< /O /List /ListNumbering /%s >>", listNumbering))
	}

	d.setDict(elem.ref, elemDict.String())

	for _, kid := range elem.Kids {
		if err := d.serializeStructElem(kid, elem.ref); err != nil {
			return err
		}
	}

	return nil
}
