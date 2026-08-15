package pdf

import (
	"fmt"
	"strconv"
	"strings"
)

// StructType represents standard structure type tags defined in ISO 32000-1 §14.8.4
// and ISO 32000-2 §14.8.4.
type StructType string

const (
	StructTypeDocument   StructType = "Document"
	StructTypePart       StructType = "Part"
	StructTypeArt        StructType = "Art"
	StructTypeSect       StructType = "Sect"
	StructTypeDiv        StructType = "Div"
	StructTypeH1         StructType = "H1"
	StructTypeH2         StructType = "H2"
	StructTypeH3         StructType = "H3"
	StructTypeH4         StructType = "H4"
	StructTypeH5         StructType = "H5"
	StructTypeH6         StructType = "H6"
	StructTypeP          StructType = "P"
	StructTypeL          StructType = "L"
	StructTypeLI         StructType = "LI"
	StructTypeLbl        StructType = "Lbl"
	StructTypeLBody      StructType = "LBody"
	StructTypeTable      StructType = "Table"
	StructTypeTR         StructType = "TR"
	StructTypeTH         StructType = "TH"
	StructTypeTD         StructType = "TD"
	StructTypeTHead      StructType = "THead"
	StructTypeTBody      StructType = "TBody"
	StructTypeTFoot      StructType = "TFoot"
	StructTypeCaption    StructType = "Caption"
	StructTypeSpan       StructType = "Span"
	StructTypeLink       StructType = "Link"
	StructTypeFigure     StructType = "Figure"
	StructTypeFormula    StructType = "Formula"
	StructTypeQuote      StructType = "Quote"
	StructTypeBlockQuote StructType = "BlockQuote"
	StructTypeCode       StructType = "Code"
)

// StructType aliases for backwards compatibility.
const (
	StructDocument   = StructTypeDocument
	StructPart       = StructTypePart
	StructArt        = StructTypeArt
	StructSect       = StructTypeSect
	StructDiv        = StructTypeDiv
	StructH1         = StructTypeH1
	StructH2         = StructTypeH2
	StructH3         = StructTypeH3
	StructH4         = StructTypeH4
	StructH5         = StructTypeH5
	StructH6         = StructTypeH6
	StructP          = StructTypeP
	StructL          = StructTypeL
	StructLI         = StructTypeLI
	StructLbl        = StructTypeLbl
	StructLBody      = StructTypeLBody
	StructTable      = StructTypeTable
	StructTR         = StructTypeTR
	StructTH         = StructTypeTH
	StructTD         = StructTypeTD
	StructTHead      = StructTypeTHead
	StructTBody      = StructTypeTBody
	StructTFoot      = StructTypeTFoot
	StructCaption    = StructTypeCaption
	StructSpan       = StructTypeSpan
	StructLink       = StructTypeLink
	StructFigure     = StructTypeFigure
	StructFormula    = StructTypeFormula
	StructQuote      = StructTypeQuote
	StructBlockQuote = StructTypeBlockQuote
	StructCode       = StructTypeCode
)

// contentRef pairs an MCID integer with its owning page, ensuring multi-page
// structure elements emit MCR (Marked Content Reference) dictionaries rather
// than bare integers that would resolve against the element's single /Pg.
type contentRef struct {
	page *Page
	mcid int
}

// StructElem represents a single logical structure element dictionary in the
// PDF document logical structure tree (ISO 32000-1 §14.7.2).
type StructElem struct {
	Tag           StructType
	Alt           string
	Lang          string
	ActualText    string
	Title         string
	TableScope    string
	ListNumbering string // PDF/UA-2 List attribute: "Disc", "Decimal", "Circle", "Square", "None"
	Page          *Page
	Kids          []*StructElem
	AnnotRef      objRef
	ref           objRef
	parent        *StructElem
	doc           *Document
	content       []contentRef
}

// StructTreeRoot represents the root of the document logical structure tree
// (ISO 32000-1 §14.7.2).
type StructTreeRoot struct {
	Children []*StructElem
	ref      objRef
	doc      *Document
}

// NewChild creates and appends a child StructElem under this structure element.
func (e *StructElem) NewChild(tag StructType) *StructElem {
	child := &StructElem{ //nolint:exhaustruct // intentional zero-value fields
		Tag:    tag,
		parent: e,
		doc:    e.doc,
		Page:   e.Page,
	}
	e.Kids = append(e.Kids, child)

	return child
}

// NewChild creates and appends a top-level StructElem under the StructTreeRoot.
func (r *StructTreeRoot) NewChild(tag StructType) *StructElem {
	elem := &StructElem{ //nolint:exhaustruct // intentional zero-value fields
		Tag: tag,
		doc: r.doc,
	}
	r.Children = append(r.Children, elem)

	return elem
}

// SetAlt sets the alternative text description (/Alt) for accessibility.
func (e *StructElem) SetAlt(alt string) { e.Alt = alt }

// SetLang sets the natural language override (/Lang) for this element subtree.
func (e *StructElem) SetLang(lang string) { e.Lang = lang }

// SetActualText sets the exact replacement text (/ActualText) for accessibility.
func (e *StructElem) SetActualText(text string) { e.ActualText = text }

// SetTitle sets the title text (/Title) for this structure element.
func (e *StructElem) SetTitle(title string) { e.Title = title }

// SetPage sets the owning page for this structure element.
func (e *StructElem) SetPage(p *Page) { e.Page = p }

// SetObjRef sets an indirect object reference (e.g. an annotation) as an OBJR child.
func (e *StructElem) SetObjRef(ref objRef, page *Page) {
	e.AnnotRef = ref
	if page != nil {
		e.Page = page
	}
}

// AddAnnot is an alias for SetObjRef.
func (e *StructElem) AddAnnot(ref objRef, page *Page) {
	e.SetObjRef(ref, page)
}

// SetAnnotation sets the annotation owning page on this structure element.
func (e *StructElem) SetAnnotation(page *Page, _ int) {
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
	if !d.IsUA() {
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

	var walk func(elem *StructElem)
	walk = func(elem *StructElem) {
		if elem == nil {
			return
		}

		if isHeadingTag(elem.Tag) {
			result = append(result, elem)
		}

		for _, kid := range elem.Kids {
			if kid != nil {
				walk(kid)
			}
		}
	}

	for _, child := range d.structTreeRoot.Children {
		if child != nil {
			walk(child)
		}
	}

	return result
}

// isHeadingTag reports whether tag is one of H1–H6.
//
//nolint:exhaustive // only heading tags H1–H6 are checked
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
	if elem == nil || p.doc == nil || !p.doc.IsUA() {
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
	if elem == nil {
		return
	}

	elem.ref = doc.newObject()
	for _, child := range elem.Kids {
		if child != nil {
			assignStructElemRefs(doc, child)
		}
	}
}

// pruneEmptyStructElems removes any unnecessary empty structure element subtree,
// while preserving required document, table, and list structural tags.
//
//nolint:exhaustive,varnamelen,wsl // in-place filtering of kids array
func pruneEmptyStructElems(elem *StructElem) bool {
	if elem == nil {
		return true
	}

	n := 0
	for _, kid := range elem.Kids {
		if kid != nil && !pruneEmptyStructElems(kid) {
			elem.Kids[n] = kid
			n++
		}
	}
	elem.Kids = elem.Kids[:n]

	switch elem.Tag {
	case StructTypeDocument, StructTypeTable, StructTypeTR, StructTypeTH, StructTypeTD,
		StructTypeL, StructTypeLI, StructTypeLBody, StructTypeCaption:
		return false
	default:
		return len(elem.Kids) == 0 && len(elem.content) == 0 && elem.AnnotRef == 0
	}
}

// finalizeStructure creates and serializes StructTreeRoot, ParentTree, and all StructElems.
//
//nolint:cyclop // structure root, namespace, prune, ParentTree, and serialize in one finalize pass
func (d *Document) finalizeStructure() error {
	if !d.isUA {
		return nil
	}

	if d.isUA2 {
		d.namespaceRef = d.newObject()
		d.setDict(d.namespaceRef, "<< /Type /Namespace /NS (http://iso.org/pdf2/ssn) >>")
	}

	if d.structTreeRoot == nil {
		d.structTreeRoot = &StructTreeRoot{doc: d} //nolint:exhaustruct // default structure root
		_ = d.structTreeRoot.NewChild(StructTypeDocument)
	}

	validRootKids := make([]*StructElem, 0, len(d.structTreeRoot.Children))

	for _, child := range d.structTreeRoot.Children {
		if child != nil && !pruneEmptyStructElems(child) {
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
		if child != nil {
			assignStructElemRefs(d, child)
		}
	}

	d.buildParentTree()
	d.serializeStructTreeRoot()

	for _, child := range d.structTreeRoot.Children {
		if child != nil {
			if err := d.serializeStructElem(child, d.structTreeRootRef); err != nil {
				return err
			}
		}
	}

	return nil
}

// buildParentTree constructs the ParentTree number tree dictionary mapping
// page StructParents indices to arrays of StructElem object references per MCID,
// and annotation StructParent indices to owning StructElem references.
//
//nolint:cyclop,funlen,gocognit,wsl // ParentTree number tree construction over pages and annotations
func (d *Document) buildParentTree() {
	for _, page := range d.pages {
		for i := range page.annots {
			if page.annots[i].annotRef == 0 {
				page.annots[i].annotRef = d.newObject()
			}
		}
	}

	annotToElem := make(map[objRef]*StructElem)

	var collectAnnotElems func(elem *StructElem)
	collectAnnotElems = func(elem *StructElem) {
		if elem == nil {
			return
		}

		if elem.AnnotRef != 0 {
			annotToElem[elem.AnnotRef] = elem
		}

		for _, kid := range elem.Kids {
			if kid != nil {
				collectAnnotElems(kid)
			}
		}
	}

	for _, child := range d.structTreeRoot.Children {
		if child != nil {
			collectAnnotElems(child)
		}
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

	var numsBuf strings.Builder
	nextStructParents := 0

	for _, page := range d.pages {
		if len(page.mcids) > 0 {
			structParents := nextStructParents
			nextStructParents++
			page.structParents = structParents
			page.hasStructParents = true

			if numsBuf.Len() > 0 {
				numsBuf.WriteByte(' ')
			}

			numsBuf.WriteString(strconv.Itoa(structParents))
			numsBuf.WriteString(" [ ")

			for j, elem := range page.mcids {
				if j > 0 {
					numsBuf.WriteByte(' ')
				}
				if elem != nil {
					numsBuf.WriteString(elem.ref.String())
				}
			}

			numsBuf.WriteString(" ]")
		}

		for i := range page.annots {
			a := &page.annots[i]
			if elem, ok := annotToElem[a.annotRef]; ok && elem != nil {
				annotStructParent := nextStructParents
				nextStructParents++
				a.structParent = annotStructParent
				a.hasStructParent = true

				if numsBuf.Len() > 0 {
					numsBuf.WriteByte(' ')
				}

				numsBuf.WriteString(strconv.Itoa(annotStructParent))
				numsBuf.WriteByte(' ')
				numsBuf.WriteString(elem.ref.String())
			}
		}
	}

	d.parentTreeNextKey = nextStructParents

	if numsBuf.Len() > 0 {
		d.parentTreeRef = d.newObject()
		d.setDict(d.parentTreeRef, fmt.Sprintf("<< /Nums [ %s ] >>", numsBuf.String()))
	}
}

// serializeStructTreeRoot serializes the /StructTreeRoot dictionary.
func (d *Document) serializeStructTreeRoot() {
	rootKids := make([]string, 0, len(d.structTreeRoot.Children))

	for _, child := range d.structTreeRoot.Children {
		if child != nil {
			rootKids = append(rootKids, child.ref.String())
		}
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

	if d.isUA2 && d.namespaceRef != 0 {
		rootDict = rootDict.add("/Namespaces", "["+d.namespaceRef.String()+"]")
	}

	d.setDict(d.structTreeRootRef, rootDict.String())
}

//nolint:cyclop,wsl // MCID vs MCR selection branches on elem content shape
func (d *Document) formatStructKids(elem *StructElem) string {
	var kItems []string
	if len(elem.Kids) > 0 {
		kItems = make([]string, 0, len(elem.Kids)+len(elem.content)+1)
		for _, kid := range elem.Kids {
			if kid != nil {
				kItems = append(kItems, kid.ref.String())
			}
		}
	} else if len(elem.content) > 0 || elem.AnnotRef != 0 {
		kItems = make([]string, 0, len(elem.content)+1)
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
			var mcrBuilder strings.Builder
			mcrBuilder.WriteString("<< /Type /MCR /Pg ")
			mcrBuilder.WriteString(contentRef.page.ref.String())
			mcrBuilder.WriteString(" /MCID ")
			mcrBuilder.WriteString(strconv.Itoa(contentRef.mcid))
			mcrBuilder.WriteString(" >>")
			kItems = append(kItems, mcrBuilder.String())
		} else {
			kItems = append(kItems, strconv.Itoa(contentRef.mcid))
		}
	}

	if elem.AnnotRef != 0 {
		targetPage := elem.Page
		if targetPage == nil && elem.parent != nil {
			targetPage = elem.parent.Page
		}

		var objrBuilder strings.Builder
		objrBuilder.WriteString("<< /Type /OBJR /Obj ")
		objrBuilder.WriteString(elem.AnnotRef.String())

		if targetPage != nil {
			objrBuilder.WriteString(" /Pg ")
			objrBuilder.WriteString(targetPage.ref.String())
		}

		objrBuilder.WriteString(" >>")
		kItems = append(kItems, objrBuilder.String())
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
func resolveListNumbering(elem *StructElem) string {
	if elem == nil || elem.Tag != StructTypeL {
		return ""
	}

	if elem.ListNumbering != "" {
		return elem.ListNumbering
	}

	if elem.doc == nil || !elem.doc.isUA2 {
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
		if kid == nil {
			continue
		}

		if kid.Tag == StructTypeLbl {
			return true
		}

		if kid.Tag == StructTypeLI {
			for _, grand := range kid.Kids {
				if grand != nil && grand.Tag == StructTypeLbl {
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
	if elem == nil {
		return nil
	}

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

	if d.isUA2 && elem.Tag == StructTypeDocument && d.namespaceRef != 0 {
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
		elemDict = elemDict.add("/A", fmt.Sprintf("<< /O /List /ListNumbering /%s >>", listNumbering))
	}

	d.setDict(elem.ref, elemDict.String())

	for _, kid := range elem.Kids {
		if kid != nil {
			if err := d.serializeStructElem(kid, elem.ref); err != nil {
				return err
			}
		}
	}

	return nil
}
