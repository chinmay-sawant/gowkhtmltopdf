package layout

import (
	"strings"

	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

// opTagInfo holds structure tagging metadata for one display-list op.
type opTagInfo struct {
	elem *pdf.StructElem
	tag  pdf.StructType
}

// buildStructureTree creates the PDF/UA-1 logical structure tree from the
// laid-out box tree and returns a map from op index to the owning StructElem.
// Returns an error if a compliance rule is violated (e.g. <img> missing alt).
//
//nolint:cyclop,gocognit,varnamelen,wsl,nilnil,nlreturn // structure tree mapping over display list ops
func buildStructureTree(doc *pdf.Document, res *Result) (map[int]*opTagInfo, error) {
	if doc == nil || !doc.Policy().IsPDFUA1() || res == nil {
		return nil, nil
	}

	rootElem := doc.CreateStructTreeRoot()
	if rootElem == nil {
		return nil, nil
	}

	docElem := rootElem.NewChild(pdf.StructDocument)
	opMap := make(map[int]*opTagInfo, len(res.Ops))

	if res.root != nil {
		if err := walkBoxForStructure(res.root, docElem, opMap, doc); err != nil {
			return nil, err
		}
	}

	// Link ops and fallback text grouping:
	var currentP *pdf.StructElem
	for i := range res.Ops {
		op := &res.Ops[i]

		if op.Kind == OpLinkURI && op.URI != "" {
			parent := currentP
			if parent == nil {
				parent = docElem
			}
			linkElem := parent.NewChild(pdf.StructLink)
			opMap[i] = &opTagInfo{elem: linkElem, tag: linkElem.Tag}

			// If the next op is text for this link, associate it with the linkElem
			if i+1 < len(res.Ops) && res.Ops[i+1].Kind == OpText {
				opMap[i+1] = &opTagInfo{elem: linkElem, tag: linkElem.Tag}
			}
			continue
		}

		if op.Kind == OpImage {
			if op.Alt == "" && doc.Policy().IsPDFUA1() {
				return nil, pdf.ErrPDFUAMissingAlt
			}
			parent := currentP
			if parent == nil {
				parent = docElem
			}
			figElem := parent.NewChild(pdf.StructFigure)
			figElem.SetAlt(op.Alt)
			opMap[i] = &opTagInfo{elem: figElem, tag: figElem.Tag}
			continue
		}

		if op.Kind == OpText || op.Kind == OpBullet {
			if currentP == nil {
				currentP = docElem.NewChild(pdf.StructP)
			}
			opMap[i] = &opTagInfo{elem: currentP, tag: currentP.Tag}
		}
	}

	return opMap, nil
}

// walkBoxForStructure recursively traverses the box hierarchy and constructs
// ISO 32000-1 structure elements (H1..H6, P, Table, TR, TH, TD, L, LI, Figure, Link).
//
//nolint:cyclop,gocyclo,gocognit,nestif,funlen,varnamelen,wsl // recursive layout box walker for PDF structure
func walkBoxForStructure(b *box, parent *pdf.StructElem, opMap map[int]*opTagInfo, doc *pdf.Document) error {
	if b == nil {
		return nil
	}

	currentParent := parent
	var createdElem *pdf.StructElem

	if b.node != nil && b.node.Type == html.ElementNode {
		name := strings.ToLower(b.node.Name)

		switch name {
		case "h1", "h2", "h3", "h4", "h5", "h6":
			tag := pdf.StructType(strings.ToUpper(name))
			createdElem = parent.NewChild(tag)
			currentParent = createdElem
		case "p":
			createdElem = parent.NewChild(pdf.StructP)
			currentParent = createdElem
		case "table":
			createdElem = parent.NewChild(pdf.StructTable)
			currentParent = createdElem
			// Process table rows and cells if b.rows is populated
			if len(b.rows) > 0 {
				for _, row := range b.rows {
					trElem := createdElem.NewChild(pdf.StructTR)
					for _, cell := range row {
						cellTag := pdf.StructTD
						if cell.node != nil && strings.EqualFold(cell.node.Name, "th") {
							cellTag = pdf.StructTH
						}
						cellElem := trElem.NewChild(cellTag)
						if cell.opStart >= 0 && cell.opEnd >= cell.opStart {
							for i := cell.opStart; i <= cell.opEnd; i++ {
								if _, exists := opMap[i]; !exists {
									opMap[i] = &opTagInfo{elem: cellElem, tag: cellElem.Tag}
								}
							}
						}
						for _, child := range cell.children {
							if err := walkBoxForStructure(child, cellElem, opMap, doc); err != nil {
								return err
							}
						}
					}
				}
			}
		case "tr":
			createdElem = parent.NewChild(pdf.StructTR)
			currentParent = createdElem
		case "th":
			createdElem = parent.NewChild(pdf.StructTH)
			currentParent = createdElem
		case "td":
			createdElem = parent.NewChild(pdf.StructTD)
			currentParent = createdElem
		case "ul", "ol":
			createdElem = parent.NewChild(pdf.StructL)
			currentParent = createdElem
		case "li":
			createdElem = parent.NewChild(pdf.StructLI)
			currentParent = createdElem
		case cssTagImg:
			alt := b.node.Attribute("alt")
			if alt == "" && doc.Policy().IsPDFUA1() {
				return pdf.ErrPDFUAMissingAlt
			}
			createdElem = parent.NewChild(pdf.StructFigure)
			createdElem.SetAlt(alt)
			currentParent = createdElem
		case "a":
			createdElem = parent.NewChild(pdf.StructLink)
			currentParent = createdElem
		}
	}

	if createdElem != nil && b.opStart >= 0 && b.opEnd >= b.opStart {
		for i := b.opStart; i <= b.opEnd; i++ {
			if _, exists := opMap[i]; !exists {
				opMap[i] = &opTagInfo{elem: createdElem, tag: createdElem.Tag}
			}
		}
	}

	for _, child := range b.children {
		if err := walkBoxForStructure(child, currentParent, opMap, doc); err != nil {
			return err
		}
	}

	return nil
}
