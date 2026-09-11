package convert

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/outline"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// idMatchSlopPt is the y-band slop when matching struct-tree elements to id
// anchor locations: element boxes recorded during layout can sit slightly
// above or below the location entry for the same anchor.
const idMatchSlopPt = 20

// minLinkRectSide is the floor applied to degenerate link rectangles so the
// annotation stays visible and clickable.
const minLinkRectSide = 10

// bodyLinkIntent is the information from a same-document link operation that
// later needs document-wide destinations. It deliberately omits the display
// operation and any source DOM pointer.
type bodyLinkIntent struct {
	uri  string
	loc  layout.ElementLocation
	elem *pdf.StructElem
}

// bodyNavigation is the compact post-paint projection of a body Result used
// by local-link and header/footer passes. Keeping it on objectState lets the
// complete display list, box tree, locations, and DOM be collected once paint
// and heading collection have finished.
type bodyNavigation struct {
	ids     map[string]layout.ElementLocation
	idElems map[string]*pdf.StructElem
	links   []bodyLinkIntent
}

// collectBodyNavigation copies the body navigation metadata that later
// document-wide passes need. Later duplicate ids overwrite earlier ones,
// matching the historical scan of Result.Locations. The copied locations have
// nil Node pointers so they do not keep the parsed document alive.
//
// Layout-produced results set MaxContentX > 0. When Paint also left HasIDs
// and HasFragmentLinks false, the report-shaped path returns without the
// two maps or the struct-op index. Hand-built Result values keep MaxContentX
// at 0 and still scan; maps are allocated only on the first id.
//
//nolint:cyclop,varnamelen,funlen // element location and structure element collection
func collectBodyNavigation(res *layout.Result) bodyNavigation {
	if res == nil {
		return bodyNavigation{} //nolint:exhaustruct // intentional zero-value projection
	}

	if res.MaxContentX > 0 && !res.HasIDs && !res.HasFragmentLinks {
		return bodyNavigation{} //nolint:exhaustruct // intentional zero-value projection
	}

	var (
		nav         bodyNavigation
		structOps   []structOpEntry
		structReady bool
	)

	for _, loc := range res.Locations {
		if loc.Node == nil {
			continue
		}

		id := loc.Node.Attribute("id")
		if id == "" {
			continue
		}

		if nav.ids == nil {
			nav.ids = make(map[string]layout.ElementLocation)
			nav.idElems = make(map[string]*pdf.StructElem)
		}

		if !structReady {
			structOps = indexStructOps(res.Ops)
			structReady = true
		}

		loc.Node = nil
		nav.ids[id] = loc

		lo, hi := loc.Y, loc.Y+loc.H+idMatchSlopPt

		first := sort.Search(len(structOps), func(i int) bool { return structOps[i].y >= lo })
		for first < len(structOps) && structOps[first].y <= hi {
			nav.idElems[id] = structOps[first].elem

			break
		}
	}

	for _, oper := range res.Ops {
		if oper.Kind != layout.OpLinkURI || !strings.HasPrefix(oper.URI, "#") {
			continue
		}

		nav.links = append(nav.links, bodyLinkIntent{
			uri: oper.URI,
			loc: layout.ElementLocation{ //nolint:exhaustruct // intentional zero-value fields
				Page: -1,
				X:    oper.X,
				Y:    oper.Y,
				W:    oper.W,
				H:    oper.H,
			},
			elem: oper.StructElem,
		})
	}

	return nav
}

type structOpEntry struct {
	y    float64
	elem *pdf.StructElem
}

func indexStructOps(ops []layout.Op) []structOpEntry {
	structOps := make([]structOpEntry, 0)

	for i := range ops {
		if ops[i].StructElem != nil {
			structOps = append(structOps, structOpEntry{y: ops[i].Y, elem: ops[i].StructElem})
		}
	}

	sort.SliceStable(structOps, func(a, b int) bool { return structOps[a].y < structOps[b].y })

	return structOps
}

// stripLinkURIs neutralizes external (http/https/mailto) link ops in place.
// Same-document fragment links (#id) are left for applyInternalLinks.
// Neutralization uses layout.DeactivateOp so every painter skips the op
// while keeping its slot (box tree op indices stay valid).
func stripLinkURIs(ops []layout.Op) []layout.Op {
	for i := range ops {
		if ops[i].Kind == layout.OpLinkURI && !strings.HasPrefix(ops[i].URI, "#") {
			layout.DeactivateOp(&ops[i])
		}
	}

	return ops
}

// tocAnchorLocations indexes the element boxes of one laid-out TOC document
// by their data-wk-target attribute (the heading anchor of the entry).
func tocAnchorLocations(_ *html.Node, res *layout.Result) map[string]layout.ElementLocation {
	out := map[string]layout.ElementLocation{}

	for _, loc := range res.Locations {
		if loc.Node == nil || loc.Node.Type != html.ElementNode {
			continue
		}

		if target := loc.Node.Attribute("data-wk-target"); target != "" {
			out[target] = loc
		}
	}

	return out
}

// attachLinkStructElem ensures a link annotation is referenced in the PDF/UA structure tree.
func attachLinkStructElem(doc *pdf.Document, page *pdf.Page, elem *pdf.StructElem, annotRef pdf.ObjRef) {
	if doc == nil || !doc.IsUA() || page == nil || annotRef == 0 {
		return
	}

	if elem != nil {
		elem.SetObjRef(annotRef, page)

		return
	}

	root := doc.StructTreeRoot()
	if root == nil {
		return
	}

	var docElem *pdf.StructElem
	if len(root.Children) > 0 {
		docElem = root.Children[0]
	} else {
		docElem = root.NewChild(pdf.StructDocument)
	}

	linkElem := docElem.NewChild(pdf.StructLink)
	linkElem.SetObjRef(annotRef, page)
}

// applyTOCLinks wires the TOC link annotations once every page exists, using
// the final page indices.
//
// Forward links (toc.forward-links): each TOC entry div becomes a GoTo
// annotation to its heading's page and position. Back links (toc.back-links):
// each body heading becomes a GoTo annotation back to its TOC entry. Both are
// skipped when the anchor side is missing.
//
// Arbitrary same-document <a href="#frag"> links in body HTML are wired by
// applyInternalLinks: layout emits OpLinkURI with a "#frag" URI for inline
// anchors that have a paint box (text runs), and convert resolves them to
// GoTo destinations via element id / heading locations.
//
//nolint:gocognit,cyclop,funlen,lll,wsl,nestif // forward/back link passes over entry locations
func applyTOCLinks(ctx context.Context, doc *pdf.Document, tocs []*objectState, bodies []*objectState, tocTotal int, headings []*outline.Heading, warn func(format string, args ...any)) error {
	if len(tocs) == 0 {
		return nil
	}

	byAnchor := map[string]*outline.Heading{}

	for _, h := range headings {
		if h.Anchor != "" {
			byAnchor[h.Anchor] = h
		}
	}

	var headingMap map[*outline.Heading]*pdf.StructElem
	if doc != nil && doc.IsUA() {
		headingMap = headingStructIdentity(doc, bodies, tocTotal)
	}

	for _, trVal := range tocs {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("toc links: %w", err)
		}

		if !trVal.toc.ForwardLinks && !trVal.toc.BackLinks {
			continue
		}

		entryLocs := tocAnchorLocations(trVal.tocRoot, trVal.tocRes)
		for anchor, eloc := range entryLocs {
			hVal := byAnchor[anchor]
			if hVal == nil {
				if warn != nil {
					warn("object %d: toc anchor %q has no matching heading; link skipped", trVal.idx+1, anchor)
				}

				continue
			}

			srcPage := doc.PageAt(trVal.start + eloc.Page)
			if srcPage == nil {
				if warn != nil {
					warn("object %d: toc anchor %q source page %d missing; link skipped", trVal.idx+1, anchor, trVal.start+eloc.Page)
				}

				continue
			}

			docPage := hVal.DocPage

			if trVal.toc.ForwardLinks {
				// TOC entry → heading
				destX, destY := headingDest(hVal, bodies)
				annotRef := srcPage.AddLinkDest(trVal.geom.pdfRect(eloc), doc.PageAt(tocTotal+docPage), destX, destY)
				attachLinkStructElem(doc, srcPage, nil, annotRef)
				if annotRef != 0 && headingMap != nil {
					if targetElem := headingMap[hVal]; targetElem != nil {
						srcPage.SetLinkDestStruct(targetElem)
					}
				}
			}

			if trVal.toc.BackLinks {
				// heading → TOC entry
				destPage := trVal.start + eloc.Page
				destX, destY := trVal.geom.pdfXY(eloc)

				if stVal := bodyStateFor(bodies, docPage); stVal != nil {
					if page := doc.PageAt(tocTotal + docPage); page != nil {
						locPage := docPage - stVal.offset
						hLoc := layout.ElementLocation{ //nolint:exhaustruct // intentional zero-value fields
							Page: locPage, X: hVal.X, Y: hVal.Y, W: hVal.W, H: hVal.H,
						}
						annotRef := page.AddLinkDest(stVal.geom.pdfRect(hLoc), doc.PageAt(destPage), destX, destY)
						attachLinkStructElem(doc, page, nil, annotRef)
					} else if warn != nil {
						warn("object %d: toc back link target page %d missing; link skipped", trVal.idx+1, tocTotal+docPage)
					}
				} else if warn != nil {
					warn("object %d: toc back link target object for page %d missing; link skipped", trVal.idx+1, docPage)
				}
			}
		}
	}

	return nil
}

// headingDest returns the PDF destination (x, y-up) of a heading's top-left
// corner using the heading's own geometry (no location scan).
func headingDest(hVal *outline.Heading, bodies []*objectState) (float64, float64) {
	docPage := hVal.DocPage

	state := bodyStateFor(bodies, docPage)
	if state == nil {
		return 0, 0
	}

	locPage := docPage - state.offset

	loc := layout.ElementLocation{ //nolint:exhaustruct // intentional zero-value fields
		Page: locPage, X: hVal.X, Y: hVal.Y, W: hVal.W, H: hVal.H,
	}

	return state.geom.pdfXY(loc)
}

// bodyIDDest is one body element destination keyed by id attribute.
type bodyIDDest struct {
	st   *objectState
	loc  layout.ElementLocation
	elem *pdf.StructElem
}

// buildBodyIDIndex maps element id attributes across body objects to their
// layout locations. Later occurrences of the same id overwrite earlier ones
// (matches the prior applyInternalLinks loop).
func buildBodyIDIndex(bodies []*objectState) map[string]bodyIDDest {
	idLoc := map[string]bodyIDDest{}

	for _, state := range bodies {
		if state == nil {
			continue
		}

		for id, loc := range state.navigation.ids {
			idLoc[id] = bodyIDDest{st: state, loc: loc, elem: state.navigation.idElems[id]}
		}
	}

	return idLoc
}

// logicalDestPage is the pre-copy document page index of a body id dest.
func logicalDestPage(dest bodyIDDest, tocTotal int) int {
	return tocTotal + dest.st.offset + dest.loc.Page
}

// remapPageForCopies maps the pre-copy (logical) destination page 1 onto the
// final document page index in the same copy group as srcPage, in a two-page
// document. Thin wrapper over pagePlan.Remap kept for unit tests.
func remapPageForCopies(srcPage, copies int, collate bool) int {
	// The wrapper models a two-page logical document.
	const logicalPages = 2

	plan := &pagePlan{ //nolint:exhaustruct // intentional zero-value fields
		copies:  copies,
		collate: collate,
		owners:  make([]pageOwner, logicalPages),
	}

	return plan.Remap(1, srcPage)
}

func applyInternalLinks(
	ctx context.Context, doc *pdf.Document, bodies []*objectState, tocTotal int, warn func(format string, args ...any),
) error {
	if doc == nil {
		return nil
	}

	idLoc := buildBodyIDIndex(bodies)

	for _, state := range bodies {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("internal links: %w", err)
		}

		if state == nil || state.geom.contentH <= 0 {
			continue
		}

		useLocal := state.obj.LocalLinks

		for _, link := range state.navigation.links {
			applyBodyLink(doc, state, link, useLocal, idLoc, tocTotal, warn)
		}
	}

	return nil
}

// applyBodyLink wires one same-document fragment link to its destination,
// warning and skipping when the anchor or source page is missing.
func applyBodyLink(
	doc *pdf.Document, state *objectState, link bodyLinkIntent, useLocal bool,
	idLoc map[string]bodyIDDest, tocTotal int, warn func(format string, args ...any),
) {
	frag := strings.TrimPrefix(link.uri, "#")

	if !useLocal || frag == "" {
		return
	}

	dest, ok := idLoc[frag]
	if !ok {
		if warn != nil {
			warn("object %d: link #%s has no matching element id; link skipped", state.idx+1, frag)
		}

		return
	}

	srcPageIdx := tocTotal + state.offset + int(link.loc.Y/state.geom.contentH)

	srcPage := doc.PageAt(srcPageIdx)
	if srcPage == nil {
		if warn != nil {
			warn("object %d: link #%s source page %d missing; link skipped", state.idx+1, frag, srcPageIdx)
		}

		return
	}

	srcLoc := link.loc
	srcLoc.Page = int(link.loc.Y / state.geom.contentH)

	// Degenerate boxes would be invisible and unclickable; clamp to a floor.
	srcLoc.H = max(srcLoc.H, minLinkRectSide)
	srcLoc.W = max(srcLoc.W, minLinkRectSide)

	destPage := logicalDestPage(dest, tocTotal)
	dx, dy := dest.st.geom.pdfXY(dest.loc)
	annotRef := srcPage.AddLinkDest(state.geom.pdfRect(srcLoc), doc.PageAt(destPage), dx, dy)
	attachLinkStructElem(doc, srcPage, link.elem, annotRef)

	if annotRef != 0 && dest.elem != nil {
		srcPage.SetLinkDestStruct(dest.elem)
	}
}
