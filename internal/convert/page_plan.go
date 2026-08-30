//nolint:all
package convert

import (
	"fmt"
	"math"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert/render"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// percent rounds i/n to a 0-100 percentage.
func percent(i, n int) int {
	if n <= 0 {
		return progressComplete
	}

	return int(math.Round(float64(i) * float64(progressComplete) / float64(n)))
}



// pageOwner is one logical (pre-copy) page and the object that owns it.
type pageOwner struct {
	st    *objectState
	local int
}

// pagePlan is the single owner of the document's page-index model.
type pagePlan struct {
	owners      []pageOwner
	objectStart []int
	model       *render.Plan
	tocTotal    int
	copies      int
	collate     bool
}

// newPagePlan builds the logical owner list and delegates page-index mapping
// to the focused render module.
//
//nolint:cyclop,wsl // page ownership adapter
func newPagePlan(tocs, bodies []*objectState, copies int, collate bool) (*pagePlan, error) {
	if copies < 1 {
		copies = 1
	}

	if copies > maxConversionCopies {
		return nil, fmt.Errorf("%w: got %d, limit %d", errTooManyCopies, copies, maxConversionCopies)
	}

	logicalPages := 0
	for _, state := range tocs {
		logicalPages += state.tocPages
	}

	for _, state := range bodies {
		logicalPages += state.pages
	}

	if logicalPages > maxConversionPages {
		return nil, fmt.Errorf("%w: got %d, limit %d", errTooManyPages, logicalPages, maxConversionPages)
	}

	if logicalPages > 0 && copies > maxConversionPages/logicalPages {
		return nil, fmt.Errorf(
			"%w: %d pages x %d copies exceeds %d",
			errTooManyPages, logicalPages, copies, maxConversionPages,
		)
	}

	tocCounts := make([]int, len(tocs))
	bodyCounts := make([]int, len(bodies))
	for index, st := range tocs {
		tocCounts[index] = st.tocPages
	}
	for index, st := range bodies {
		bodyCounts[index] = st.pages
	}
	model, err := render.NewPlan(tocCounts, bodyCounts, copies, collate)
	if err != nil {
		return nil, fmt.Errorf("build render page plan: %w", err)
	}
	pagePlan := &pagePlan{ //nolint:exhaustruct // owners are populated below
		owners: make([]pageOwner, 0, logicalPages), objectStart: make([]int, 0, len(tocs)+len(bodies)),
		model: model, copies: copies, collate: collate,
	}
	for _, st := range tocs {
		pagePlan.objectStart = append(pagePlan.objectStart, len(pagePlan.owners))
		pagePlan.tocTotal += st.tocPages
		for i := range st.tocPages {
			pagePlan.owners = append(pagePlan.owners, pageOwner{st, i})
		}
	}
	for _, st := range bodies {
		pagePlan.objectStart = append(pagePlan.objectStart, len(pagePlan.owners))
		for i := range st.pages {
			pagePlan.owners = append(pagePlan.owners, pageOwner{st, i})
		}
	}

	return pagePlan, nil
}

// OwnerOf resolves the object that owns final page p.
//
//nolint:wsl // compatibility owner adapter
func (pp *pagePlan) OwnerOf(page int) (pageOwner, bool) {
	if pp == nil || pp.model == nil {
		return pageOwner{}, false //nolint:exhaustruct // invalid page
	}
	owner, ok := pp.model.OwnerOf(page)
	if !ok || owner.Object < 0 || owner.Object >= len(pp.objectStart) {
		return pageOwner{}, false //nolint:exhaustruct // invalid page
	}
	logical := pp.objectStart[owner.Object] + owner.Local
	if logical < 0 || logical >= len(pp.owners) {
		return pageOwner{}, false //nolint:exhaustruct // defensive guard
	}

	return pp.owners[logical], true
}

// Remap converts a logical destination page to the final page in srcPage's
// copy group.
func (pp *pagePlan) Remap(logicalDest, srcPage int) int {
	if pp == nil {
		return logicalDest
	}
	if pp.model == nil { //nolint:wsl
		count := len(pp.owners)
		if pp.copies <= 1 || count <= 0 {
			return logicalDest
		}
		if pp.collate { //nolint:wsl
			return (srcPage/count)*count + logicalDest
		}

		return logicalDest*pp.copies + srcPage%pp.copies
	}

	return pp.model.Remap(logicalDest, srcPage)
}

// LogicalN is the number of pre-copy pages.
func (pp *pagePlan) LogicalN() int {
	if pp == nil {
		return 0
	}

	return pp.model.LogicalN()
}

// Ranges returns contiguous per-object page spans in logical order.
func (pp *pagePlan) Ranges() []render.Range {
	if pp == nil || pp.model == nil {
		return nil
	}
	ranges := pp.model.Ranges() //nolint:wsl

	return ranges
}

func tocFirstOrder(tocs, bodies []*objectState) []int {
	order := make([]int, 0, len(tocs)+len(bodies))

	for _, tr := range tocs {
		for i := range tr.tocPages {
			order = append(order, tr.start+i)
		}
	}

	for _, bg := range bodies {
		for i := range bg.pages {
			order = append(order, bg.offset+i)
		}
	}

	return order
}

func materializeCopies(doc *pdf.Document, ranges []render.Range, copies int) error {
	if copies < 1 {
		return nil
	}

	if copies > maxConversionCopies {
		return fmt.Errorf("%w: got %d, limit %d", errTooManyCopies, copies, maxConversionCopies)
	}

	original := 0
	for _, span := range ranges {
		original += span.Count
	}

	if original == 0 {
		return nil
	}

	for copyIndex := 1; copyIndex < copies; copyIndex++ {
		for _, span := range ranges {
			for page := span.Start; page < span.Start+span.Count; page++ {
				if _, err := doc.DuplicatePage(page); err != nil {
					return fmt.Errorf("assemble copies: %w", err)
				}
			}
		}
	}

	return nil
}

func nonCollateOrder(ranges []render.Range, copies int) []int {
	return render.NonCollateOrder(ranges, copies)
}
