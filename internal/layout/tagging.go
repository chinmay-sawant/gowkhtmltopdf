package layout

import (
	"fmt"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// structureContext tracks sequential document structure state during layout box walking.
type structureContext struct {
	lastHeadingLevel int
}

func isSemanticOp(kind OpKind) bool {
	return kind == OpText || kind == OpBullet || kind == OpImage
}

const (
	scopeColumn = "Column"
	scopeRow    = "Row"
	scopeBoth   = "Both"
)

// buildStructureTree creates the PDF/UA logical structure tree from the
// laid-out box tree and assigns StructElems directly onto display-list ops.
// Returns an error if a compliance rule is violated (e.g. <img> missing alt).
func buildStructureTree(doc *pdf.Document, res *Result) error {
	if doc == nil || !doc.IsUA() || res == nil {
		return nil
	}

	rootElem := doc.CreateStructTreeRoot()
	if rootElem == nil {
		return nil
	}

	var docElem *pdf.StructElem
	if len(rootElem.Children) > 0 && rootElem.Children[0].Tag == pdf.StructDocument {
		docElem = rootElem.Children[0]
	} else {
		docElem = rootElem.NewChild(pdf.StructDocument)
	}

	sctx := &structureContext{lastHeadingLevel: 0}

	if res.root != nil {
		if err := walkBoxForStructure(res.root, docElem, res.Ops, doc, sctx); err != nil {
			return err
		}
	}

	return associateUnmappedOps(doc, res.Ops, docElem)
}

// associateUnmappedOps associates any display ops that were not mapped by the
// DOM box walk with appropriate semantic structure elements (e.g. Link, Figure, P).
//
//nolint:cyclop,gocyclo,funlen,gocognit,nestif,varnamelen,wsl // sequential fallback grouping over display list ops
func associateUnmappedOps(doc *pdf.Document, ops []Op, docElem *pdf.StructElem) error {
	var currentP *pdf.StructElem

	for i := range ops {
		op := &ops[i]

		if op.Kind == OpLinkURI && op.URI != "" {
			if op.StructElem != nil && op.StructElem.Tag == pdf.StructLink {
				if i > 0 && ops[i-1].Kind == OpText && ops[i-1].StructElem != op.StructElem {
					ops[i-1].StructElem = op.StructElem
				}
				currentP = nil

				continue
			}

			if op.StructElem != nil {
				linkElem := newLinkChild(op.StructElem)
				op.StructElem = linkElem
				if i > 0 && ops[i-1].Kind == OpText {
					ops[i-1].StructElem = linkElem
				}
				currentP = nil

				continue
			}

			if i > 0 && ops[i-1].Kind == OpText && ops[i-1].StructElem != nil {
				if ops[i-1].StructElem.Tag == pdf.StructLink {
					op.StructElem = ops[i-1].StructElem
					currentP = nil

					continue
				}

				linkElem := newLinkChild(ops[i-1].StructElem)
				ops[i-1].StructElem = linkElem
				op.StructElem = linkElem
				currentP = nil

				continue
			}

			parent := docElem
			if currentP != nil {
				parent = currentP
			}

			linkElem := newLinkChild(parent)
			op.StructElem = linkElem

			if i > 0 && ops[i-1].Kind == OpText && ops[i-1].StructElem == nil {
				ops[i-1].StructElem = linkElem
			}
			if i+1 < len(ops) && ops[i+1].Kind == OpText && ops[i+1].StructElem == nil {
				ops[i+1].StructElem = linkElem
			}

			currentP = nil

			continue
		}

		if op.Kind == OpImage {
			if op.StructElem == nil {
				if op.Alt == "" && doc.IsUA() {
					return pdf.ErrPDFUAMissingAlt
				}

				parent := currentP
				if parent == nil {
					parent = docElem
				}

				figElem := parent.NewChild(pdf.StructFigure)
				figElem.SetAlt(op.Alt)
				op.StructElem = figElem
			}

			currentP = nil

			continue
		}

		if op.Kind == OpText || op.Kind == OpBullet {
			if op.StructElem == nil {
				if currentP == nil {
					currentP = docElem.NewChild(pdf.StructP)
				}

				op.StructElem = currentP
			} else {
				currentP = nil
			}

			continue
		}

		currentP = nil
	}

	return nil
}

// resolveTableScope extracts the scope attribute or derives the scope for a TH element.
func resolveTableScope(node *html.Node) string {
	if node == nil {
		return scopeColumn
	}

	sAttr := strings.ToLower(node.Attribute("scope"))
	switch sAttr {
	case "row", "rowgroup":
		return scopeRow
	case "both":
		return scopeBoth
	case "col", "colgroup":
		return scopeColumn
	default:
		return scopeColumn
	}
}

func clampHeadingLevel(level, lastLevel int) int {
	const (
		minHeadingLevel = 1
		maxHeadingLevel = 6
	)

	if lastLevel == 0 {
		if level > minHeadingLevel {
			return minHeadingLevel
		}

		return level
	}

	if level > lastLevel+1 {
		level = lastLevel + 1
	}

	if level < minHeadingLevel {
		return minHeadingLevel
	}

	if level > maxHeadingLevel {
		return maxHeadingLevel
	}

	return level
}

// resolveHeadingTag resolves sequential heading level ensuring no skipped levels per PDF/UA-1 §7.4.2.
func resolveHeadingTag(name string, sctx *structureContext) pdf.StructType {
	rawLevel := 1
	if len(name) >= 2 && name[1] >= '1' && name[1] <= '6' {
		rawLevel = int(name[1] - '0')
	}

	level := rawLevel
	if sctx != nil {
		level = clampHeadingLevel(rawLevel, sctx.lastHeadingLevel)
		sctx.lastHeadingLevel = level
	}

	return pdf.StructType(fmt.Sprintf("H%d", level))
}

func isTableGroupTag(tag pdf.StructType) bool {
	return tag == pdf.StructTable || tag == pdf.StructTHead || tag == pdf.StructTBody || tag == pdf.StructTFoot
}

func isTableDirectChild(name string) bool {
	return name == "tr" || name == htmlCaption || name == htmlThead || name == htmlTbody || name == htmlTfoot
}

func isStructuralContainer(tag pdf.StructType) bool {
	return tag == pdf.StructL || tag == pdf.StructTable || tag == pdf.StructTR || tag == pdf.StructDocument
}

// ensureInlineParent returns a structure parent that may legally contain
// inline content such as Link or Figure. ISO 32000-1 / ISO 32005 list
// nesting allows only LI, L, or Caption under L, and only Lbl, LBody, or
// L under LI.
//
//nolint:exhaustive // only list and table parents need inline wrapper elements
func ensureInlineParent(parent *pdf.StructElem) *pdf.StructElem {
	if parent == nil {
		return nil
	}

	switch parent.Tag {
	case pdf.StructL:
		return parent.NewChild(pdf.StructLI).NewChild(pdf.StructLBody)
	case pdf.StructLI:
		return parent.NewChild(pdf.StructLBody)
	case pdf.StructTable:
		return parent.NewChild(pdf.StructTR).NewChild(pdf.StructTD)
	case pdf.StructTR:
		return parent.NewChild(pdf.StructTD)
	default:
		return parent
	}
}

func newLinkChild(parent *pdf.StructElem) *pdf.StructElem {
	parent = ensureInlineParent(parent)
	if parent == nil {
		return nil
	}

	return parent.NewChild(pdf.StructLink)
}

// tagHeading handles heading element creation (H1..H6).
func tagHeading(name string, parent *pdf.StructElem, sctx *structureContext) *pdf.StructElem {
	tag := resolveHeadingTag(name, sctx)

	return parent.NewChild(tag)
}

// tagTable handles table structure creation (Table, TR, TH, TD, Caption).
//
//nolint:cyclop,gocognit,varnamelen,wsl,lll // table and grid structure hierarchy mapping
func tagTable(b *box, parent *pdf.StructElem, ops []Op, doc *pdf.Document, sctx *structureContext) (*pdf.StructElem, error) {
	tableElem := parent.NewChild(pdf.StructTable)

	for _, child := range b.children {
		if child.node != nil && strings.EqualFold(child.node.Name, "caption") {
			captionElem := tableElem.NewChild(pdf.StructCaption)
			mapSemanticOps(child, captionElem, ops)

			for _, c := range child.children {
				if err := walkBoxForStructure(c, captionElem, ops, doc, sctx); err != nil {
					return nil, err
				}
			}
		}
	}

	if len(b.rows) > 0 {
		for _, row := range b.rows {
			trElem := tableElem.NewChild(pdf.StructTR)
			for _, cell := range row {
				cellTag := pdf.StructTD
				if cell.node != nil && strings.EqualFold(cell.node.Name, "th") {
					cellTag = pdf.StructTH
				}

				cellElem := trElem.NewChild(cellTag)
				if cellTag == pdf.StructTH {
					cellElem.SetTableScope(resolveTableScope(cell.node))
				}

				mapSemanticOps(cell, cellElem, ops)

				for _, child := range cell.children {
					if err := walkBoxForStructure(child, cellElem, ops, doc, sctx); err != nil {
						return nil, err
					}
				}
			}
		}
	}

	return tableElem, nil
}

// tagListItem handles list item structure creation (LI, Lbl, LBody).
//
//nolint:cyclop,varnamelen,wsl // list item structure hierarchy mapping
func tagListItem(b *box, parent *pdf.StructElem, ops []Op, doc *pdf.Document, sctx *structureContext) error {
	if parent.Tag != pdf.StructL {
		parent = parent.NewChild(pdf.StructL)
		parent.SetListNumbering("Disc")
	}

	liElem := parent.NewChild(pdf.StructLI)
	var lblElem *pdf.StructElem

	if b.opStart >= 0 && b.opEnd >= b.opStart {
		for i := b.opStart; i <= b.opEnd; i++ {
			if i < len(ops) && ops[i].Kind == OpBullet {
				if lblElem == nil {
					lblElem = liElem.NewChild(pdf.StructLbl)
				}
				ops[i].StructElem = lblElem
			}
		}
	}

	lbodyElem := liElem.NewChild(pdf.StructLBody)

	if b.opStart >= 0 && b.opEnd >= b.opStart {
		for i := b.opStart; i <= b.opEnd; i++ {
			if i >= len(ops) || ops[i].StructElem != nil {
				continue
			}

			isBodyText := isSemanticOp(ops[i].Kind) && ops[i].Kind != OpBullet
			if isBodyText || ops[i].Kind == OpLinkURI {
				ops[i].StructElem = lbodyElem
			}
		}
	}

	for _, child := range b.children {
		if err := walkBoxForStructure(child, lbodyElem, ops, doc, sctx); err != nil {
			return err
		}
	}

	return nil
}

// mapSemanticOps maps display list ops belonging to box b to targetElem if not already mapped.
//
//nolint:varnamelen // b is conventional layout box receiver/param across layout package
func mapSemanticOps(b *box, targetElem *pdf.StructElem, ops []Op) {
	if targetElem == nil || isStructuralContainer(targetElem.Tag) {
		return
	}

	if b.opStart < 0 || b.opEnd < b.opStart {
		return
	}

	for i := b.opStart; i <= b.opEnd; i++ {
		if i < len(ops) && (isSemanticOp(ops[i].Kind) || ops[i].Kind == OpLinkURI) {
			if ops[i].StructElem == nil {
				ops[i].StructElem = targetElem
			}
		}
	}
}

// walkBoxForStructure recursively traverses the box hierarchy and constructs
// ISO 32000-1 structure elements (H1..H6, P, Table, TR, TH, TD, L, LI, Lbl, LBody, Figure, Link).
//
//nolint:cyclop,gocyclo,funlen,gocognit,nestif,varnamelen,wsl // recursive layout box walker for PDF structure
func walkBoxForStructure(
	b *box, parent *pdf.StructElem,
	ops []Op, doc *pdf.Document, sctx *structureContext,
) error {
	if b == nil {
		return nil
	}

	currentParent := parent
	var createdElem *pdf.StructElem

	if b.node != nil && b.node.Type == html.ElementNode {
		name := strings.ToLower(b.node.Name)

		if parent.Tag == pdf.StructL && name != "li" && name != "caption" {
			parent = parent.NewChild(pdf.StructLI).NewChild(pdf.StructLBody)
			currentParent = parent
		} else if parent.Tag == pdf.StructTable && !isTableDirectChild(name) {
			parent = parent.NewChild(pdf.StructTR).NewChild(pdf.StructTD)
			currentParent = parent
		}

		switch name {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			createdElem = tagHeading(name, parent, sctx)
			currentParent = createdElem
		case "p":
			createdElem = parent.NewChild(pdf.StructP)
			currentParent = createdElem
		case "table":
			tblElem, err := tagTable(b, parent, ops, doc, sctx)
			if err != nil {
				return err
			}
			currentParent = tblElem
			if len(b.rows) > 0 {
				return nil
			}
		case "tr":
			if !isTableGroupTag(parent.Tag) {
				parent = parent.NewChild(pdf.StructTable)
			}
			createdElem = parent.NewChild(pdf.StructTR)
			currentParent = createdElem
		case "th":
			if parent.Tag == pdf.StructTH || parent.Tag == pdf.StructTD {
				createdElem = parent
			} else {
				if parent.Tag != pdf.StructTR {
					if !isTableGroupTag(parent.Tag) {
						parent = parent.NewChild(pdf.StructTable)
					}
					parent = parent.NewChild(pdf.StructTR)
				}
				createdElem = parent.NewChild(pdf.StructTH)
				createdElem.SetTableScope(resolveTableScope(b.node))
			}
			currentParent = createdElem
		case "td":
			if parent.Tag == pdf.StructTD || parent.Tag == pdf.StructTH {
				createdElem = parent
			} else {
				if parent.Tag != pdf.StructTR {
					if !isTableGroupTag(parent.Tag) {
						parent = parent.NewChild(pdf.StructTable)
					}
					parent = parent.NewChild(pdf.StructTR)
				}
				createdElem = parent.NewChild(pdf.StructTD)
			}
			currentParent = createdElem
		case "ul", "ol":
			if parent.Tag == pdf.StructL {
				parent = parent.NewChild(pdf.StructLI).NewChild(pdf.StructLBody)
			}
			createdElem = parent.NewChild(pdf.StructL)
			if name == "ol" {
				createdElem.SetListNumbering("Decimal")
			} else {
				createdElem.SetListNumbering("Disc")
			}
			currentParent = createdElem
		case "li":
			return tagListItem(b, parent, ops, doc, sctx)
		case cssTagImg:
			alt := b.node.Attribute("alt")
			if alt == "" && doc.IsUA() {
				return pdf.ErrPDFUAMissingAlt
			}
			createdElem = parent.NewChild(pdf.StructFigure)
			createdElem.SetAlt(alt)
			currentParent = createdElem
		case "a":
			createdElem = newLinkChild(parent)
			currentParent = createdElem
		}
	} else if parent.Tag == pdf.StructL {
		parent = parent.NewChild(pdf.StructLI).NewChild(pdf.StructLBody)
		currentParent = parent
	}

	for _, child := range b.children {
		if err := walkBoxForStructure(child, currentParent, ops, doc, sctx); err != nil {
			return err
		}
	}

	targetElem := createdElem
	if targetElem == nil && currentParent != nil && !isStructuralContainer(currentParent.Tag) {
		targetElem = currentParent
	}

	mapSemanticOps(b, targetElem, ops)

	return nil
}
