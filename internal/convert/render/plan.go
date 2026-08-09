// Package render owns document page ordering and copy materialization. The
// conversion package supplies object ownership; this module owns the smaller
// page-index interface used by links and headers/footers.
package render

import (
	"errors"
	"fmt"
	"slices"

	"gowkhtmltopdf/internal/pdf"
)

const maxCopies = 1_000

var errCopyLimit = errors.New("render: copy limit exceeded")

// Range is a half-open span [Start, Start+Count) in the pre-copy document.
type Range struct {
	Start int
	Count int
}

// Owner identifies a logical page by its object index and local page number.
type Owner struct {
	Object int
	Local  int
}

// Plan is the page-index model for logical pages and copy/collate mapping.
type Plan struct {
	owners  []Owner
	copies  int
	collate bool
}

// NewPlan builds the logical order from TOC page counts followed by body page
// counts. Object indexes are stable across the two input slices: TOCs first,
// then bodies.
func NewPlan(
	tocPages, bodyPages []int, copies int, collate bool,
) (*Plan, error) {
	if copies < 1 {
		copies = 1
	}

	if copies > maxCopies {
		return nil, fmt.Errorf("%w: got %d, limit %d", errCopyLimit, copies, maxCopies)
	}

	plan := &Plan{copies: copies, collate: collate} //nolint:exhaustruct // owners are appended below
	allPages := slices.Concat(tocPages, bodyPages)

	// Object order is part of the page-index contract.

	for object, pages := range allPages {
		for local := range pages {
			plan.owners = append(plan.owners, Owner{Object: object, Local: local})
		}
	}

	return plan, nil
}

// OwnerOf resolves a final document page to its logical owner.
//
//nolint:wsl,nlreturn // page-index branching
//nolint:wsl // page-index branching
func (p *Plan) OwnerOf(page int) (Owner, bool) {
	if p == nil || len(p.owners) == 0 {
		return Owner{}, false //nolint:exhaustruct // invalid page has no owner
	}

	var index int
	switch {
	case p.copies <= 1:
		index = page
	case p.collate:
		index = page % len(p.owners)
	default:
		index = page / p.copies
	}
	if index < 0 || index >= len(p.owners) {
		return Owner{}, false //nolint:exhaustruct // invalid page has no owner
	}
	return p.owners[index], true
}

// Remap converts a logical destination page to the final page in srcPage's
// copy group.
//
//nolint:wsl,nlreturn // copy permutation cases
//nolint:wsl // copy permutation cases
func (p *Plan) Remap(logicalDest, srcPage int) int {
	if p == nil || p.copies <= 1 || len(p.owners) == 0 {
		return logicalDest
	}
	if p.collate {
		return (srcPage/len(p.owners))*len(p.owners) + logicalDest
	}
	return logicalDest*p.copies + srcPage%p.copies
}

// LogicalN is the number of pre-copy pages.
func (p *Plan) LogicalN() int {
	if p == nil {
		return 0
	}

	return len(p.owners)
}

// Ranges returns contiguous per-object page spans in logical order.
//
//nolint:wsl,nlreturn // contiguous range scan
func (p *Plan) Ranges() []Range {
	if p == nil || len(p.owners) == 0 {
		return nil
	}

	ranges := make([]Range, 0, len(p.owners))
	start := 0
	current := p.owners[0].Object
	for index := 1; index < len(p.owners); index++ {
		if p.owners[index].Object == current {
			continue
		}
		ranges = append(ranges, Range{Start: start, Count: index - start})
		start = index
		current = p.owners[index].Object
	}
	ranges = append(ranges, Range{Start: start, Count: len(p.owners) - start})
	return ranges
}

// MaterializeCopies appends fresh page objects for each copy run.
func MaterializeCopies(
	doc *pdf.Document, ranges []Range, copies int,
) error {
	if copies < 1 {
		return nil
	}

	if copies > maxCopies {
		return fmt.Errorf("%w: got %d, limit %d", errCopyLimit, copies, maxCopies)
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

// NonCollateOrder builds the page permutation for non-collated copies.
//
//nolint:wsl,nlreturn // permutation loops are intentionally local
func NonCollateOrder(ranges []Range, copies int) []int {
	original := 0
	for _, span := range ranges {
		original += span.Count
	}
	order := make([]int, 0, original*copies)
	for _, span := range ranges {
		for copyIndex := range copies {
			for page := span.Start; page < span.Start+span.Count; page++ {
				order = append(order, page+copyIndex*original)
			}
		}
	}
	return order
}
