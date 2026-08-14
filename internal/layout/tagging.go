package layout

import (
	"fmt"
	"strings"

	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

// opTagInfo holds structure tagging metadata for one display-list op.
type opTagInfo struct {
	elem *pdf.StructElem
	tag  pdf.StructType
}

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
// laid-out box tree and returns a map from op index to the owning StructElem.
// Returns an error if a compliance rule is violated (e.g. <img> missing alt).
//
//nolint:cyclop,gocognit,varnamelen,wsl,nilnil,funlen,nestif // structure tree mapping over display list ops
func buildStructureTree(doc *pdf.Document, res *Result) (map[int]*opTagInfo, error) {
	if doc == nil || (!doc.Policy().IsPDFUA1() && !doc.Policy().IsPDFUA2()) || res == nil {
		return nil, nil
	}

	rootElem := doc.CreateStructTreeRoot()
	if rootElem == nil {
		return nil, nil
	}

	docElem := rootElem.NewChild(pdf.StructDocument)
	opMap := make(map[int]*opTagInfo, len(res.Ops))
	sctx := &structureContext{lastHeadingLevel: 0}

	if res.root != nil {
		if err := walkBoxForStructure(res.root, docElem, opMap, res.Ops, doc, sctx); err != nil {
			return nil, err
		}
	}

	// Link ops and fallback text grouping. Unmapped text shares a P only
	// across a contiguous run; any mapped semantic op ends the run so we do
	// not accumulate one document-wide mega-P (invalid multi-page /K MCIDs).
	var currentP *pdf.StructElem
	for i := range res.Ops {
		op := &res.Ops[i]

		if op.Kind == OpLinkURI && op.URI != "" {
			parent := docElem
			if i+1 < len(res.Ops) && opMap[i+1] != nil && opMap[i+1].elem != nil {
				parent = opMap[i+1].elem
			} else if currentP != nil {
				parent = currentP
			}

			linkElem := parent.NewChild(pdf.StructLink)
			opMap[i] = &opTagInfo{elem: linkElem, tag: linkElem.Tag}

			// If the next op is text for this link, associate it with the linkElem
			if i+1 < len(res.Ops) && res.Ops[i+1].Kind == OpText {
				opMap[i+1] = &opTagInfo{elem: linkElem, tag: linkElem.Tag}
			}
			currentP = nil

			continue
		}

		if op.Kind == OpImage {
			if _, exists := opMap[i]; !exists {
				if op.Alt == "" && (doc.Policy().IsPDFUA1() || doc.Policy().IsPDFUA2()) {
					return nil, pdf.ErrPDFUAMissingAlt
				}
				parent := currentP
				if parent == nil {
					parent = docElem
				}
				figElem := parent.NewChild(pdf.StructFigure)
				figElem.SetAlt(op.Alt)
				opMap[i] = &opTagInfo{elem: figElem, tag: figElem.Tag}
			}
			currentP = nil

			continue
		}

		if op.Kind == OpText || op.Kind == OpBullet {
			if _, exists := opMap[i]; !exists {
				if currentP == nil {
					currentP = docElem.NewChild(pdf.StructP)
				}
				opMap[i] = &opTagInfo{elem: currentP, tag: currentP.Tag}
			} else {
				// Structured mapping ends the fallback run.
				currentP = nil
			}

			continue
		}

		// Non-text ops (fills, strokes, chrome) break fallback grouping.
		currentP = nil
	}

	for i, info := range opMap {
		if info != nil && i >= 0 && i < len(res.Ops) {
			res.Ops[i].StructElem = info.elem
		}
	}

	return opMap, nil
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

// walkBoxForStructure recursively traverses the box hierarchy and constructs
// ISO 32000-1 structure elements (H1..H6, P, Table, TR, TH, TD, L, LI, Lbl, LBody, Figure, Link).
//
//nolint:cyclop,gocyclo,gocognit,nestif,funlen,varnamelen,wsl,maintidx // recursive layout box walker for PDF structure
func walkBoxForStructure(
	b *box, parent *pdf.StructElem, opMap map[int]*opTagInfo,
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
			tag := resolveHeadingTag(name, sctx)
			createdElem = parent.NewChild(tag)
			currentParent = createdElem
		case "p":
			createdElem = parent.NewChild(pdf.StructP)
			currentParent = createdElem
		case "table":
			createdElem = parent.NewChild(pdf.StructTable)
			// 1. Process caption first if present in b.children
			for _, child := range b.children {
				if child.node != nil && strings.EqualFold(child.node.Name, "caption") {
					captionElem := createdElem.NewChild(pdf.StructCaption)
					if child.opStart >= 0 && child.opEnd >= child.opStart {
						for i := child.opStart; i <= child.opEnd; i++ {
							if i < len(ops) && isSemanticOp(ops[i].Kind) {
								if _, exists := opMap[i]; !exists {
									opMap[i] = &opTagInfo{elem: captionElem, tag: captionElem.Tag}
								}
							}
						}
					}
					for _, c := range child.children {
						if err := walkBoxForStructure(c, captionElem, opMap, ops, doc, sctx); err != nil {
							return err
						}
					}
				}
			}
			// 2. Process table rows and cells if b.rows is populated
			if len(b.rows) > 0 {
				for _, row := range b.rows {
					trElem := createdElem.NewChild(pdf.StructTR)
					for _, cell := range row {
						cellTag := pdf.StructTD
						if cell.node != nil && strings.EqualFold(cell.node.Name, "th") {
							cellTag = pdf.StructTH
						}
						cellElem := trElem.NewChild(cellTag)
						if cellTag == pdf.StructTH {
							cellElem.SetTableScope(resolveTableScope(cell.node))
						}
						if cell.opStart >= 0 && cell.opEnd >= cell.opStart {
							for i := cell.opStart; i <= cell.opEnd; i++ {
								if i < len(ops) && isSemanticOp(ops[i].Kind) {
									if _, exists := opMap[i]; !exists {
										opMap[i] = &opTagInfo{elem: cellElem, tag: cellElem.Tag}
									}
								}
							}
						}
						for _, child := range cell.children {
							if err := walkBoxForStructure(child, cellElem, opMap, ops, doc, sctx); err != nil {
								return err
							}
						}
					}
				}

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
				currentParent = createdElem
			} else {
				if parent.Tag != pdf.StructTR {
					if !isTableGroupTag(parent.Tag) {
						parent = parent.NewChild(pdf.StructTable)
					}
					parent = parent.NewChild(pdf.StructTR)
				}
				createdElem = parent.NewChild(pdf.StructTH)
				createdElem.SetTableScope(resolveTableScope(b.node))
				currentParent = createdElem
			}
		case "td":
			if parent.Tag == pdf.StructTD || parent.Tag == pdf.StructTH {
				createdElem = parent
				currentParent = createdElem
			} else {
				if parent.Tag != pdf.StructTR {
					if !isTableGroupTag(parent.Tag) {
						parent = parent.NewChild(pdf.StructTable)
					}
					parent = parent.NewChild(pdf.StructTR)
				}
				createdElem = parent.NewChild(pdf.StructTD)
				currentParent = createdElem
			}
		case "ul", "ol":
			if parent.Tag == pdf.StructL {
				parent = parent.NewChild(pdf.StructLI).NewChild(pdf.StructLBody)
			}
			createdElem = parent.NewChild(pdf.StructL)
			// PDF/UA-2 requires /ListNumbering when LIs carry /Lbl children.
			if name == "ol" {
				createdElem.SetListNumbering("Decimal")
			} else {
				createdElem.SetListNumbering("Disc")
			}
			currentParent = createdElem
		case "li":
			if parent.Tag != pdf.StructL {
				parent = parent.NewChild(pdf.StructL)
				parent.SetListNumbering("Disc")
			}
			liElem := parent.NewChild(pdf.StructLI)
			var lblElem *pdf.StructElem

			// Map bullet marker op to Lbl
			if b.opStart >= 0 && b.opEnd >= b.opStart {
				for i := b.opStart; i <= b.opEnd; i++ {
					if i < len(ops) && ops[i].Kind == OpBullet {
						if lblElem == nil {
							lblElem = liElem.NewChild(pdf.StructLbl)
						}
						opMap[i] = &opTagInfo{elem: lblElem, tag: lblElem.Tag}
					}
				}
			}

			// Create LBody for list item content and child boxes
			lbodyElem := liElem.NewChild(pdf.StructLBody)

			// Map direct item text ops (excluding OpBullet) to LBody
			if b.opStart >= 0 && b.opEnd >= b.opStart {
				for i := b.opStart; i <= b.opEnd; i++ {
					if i < len(ops) && isSemanticOp(ops[i].Kind) && ops[i].Kind != OpBullet {
						if _, exists := opMap[i]; !exists {
							opMap[i] = &opTagInfo{elem: lbodyElem, tag: lbodyElem.Tag}
						}
					}
				}
			}

			// Recurse on children with lbodyElem as parent (so nested L, P, etc. live in LBody)
			for _, child := range b.children {
				if err := walkBoxForStructure(child, lbodyElem, opMap, ops, doc, sctx); err != nil {
					return err
				}
			}

			return nil
		case cssTagImg:
			alt := b.node.Attribute("alt")
			if alt == "" && (doc.Policy().IsPDFUA1() || doc.Policy().IsPDFUA2()) {
				return pdf.ErrPDFUAMissingAlt
			}
			createdElem = parent.NewChild(pdf.StructFigure)
			createdElem.SetAlt(alt)
			currentParent = createdElem
		case "a":
			createdElem = parent.NewChild(pdf.StructLink)
			currentParent = createdElem
		}
	} else if parent.Tag == pdf.StructL {
		parent = parent.NewChild(pdf.StructLI).NewChild(pdf.StructLBody)
		currentParent = parent
	}

	targetElem := createdElem
	if targetElem == nil && currentParent != nil && !isStructuralContainer(currentParent.Tag) {
		targetElem = currentParent
	}

	if targetElem != nil && !isStructuralContainer(targetElem.Tag) && b.opStart >= 0 && b.opEnd >= b.opStart {
		for i := b.opStart; i <= b.opEnd; i++ {
			if i < len(ops) && isSemanticOp(ops[i].Kind) {
				if _, exists := opMap[i]; !exists {
					opMap[i] = &opTagInfo{elem: targetElem, tag: targetElem.Tag}
				}
			}
		}
	}

	for _, child := range b.children {
		if err := walkBoxForStructure(child, currentParent, opMap, ops, doc, sctx); err != nil {
			return err
		}
	}

	return nil
}
