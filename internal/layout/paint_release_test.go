package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// TestPaintReleasesPaginationIndexesAfterReaders pins the PDF-05 release
// point. After Paint:
//
//   - the six pagination-only indexes are nil, so headers and icons are not
//     retained through PDF finalization;
//   - Ops, Pages, Locations, boxes, and root survive, because conversion reads
//     them after Paint (convert.go:663-667 plus the link and outline passes);
//   - PageNames, the first post-Paint reader, still resolves named pages from
//     the retained box list;
//   - a later pagination pass rebuilds the indexes through ensureFlowIndex.
//
// If the release moved earlier than a live reader that rebuilds on demand, the
// output would still be correct but the paint pipeline would rebuild mid-pass.
// normalizeTableRowGaps reads res.flowPages directly, before the release
// point; the table pagination tests in this package protect that path.
func TestPaintReleasesPaginationIndexesAfterReaders(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
		.cover { page: cover }
		p { margin: 0 }
	`)
	res := layoutHTML(t, `<html><body>
		<div class="cover"><p>COVER</p></div>
		<div class="body"><p>BODYONE</p><p>BODYTWO</p></div>
	</body></html>`, cssSheet)
	doc := pdf.NewDocument()

	if err := Paint(doc, res, paintOpts()); err != nil {
		t.Fatal(err)
	}

	assertPaginationIndexesReleased(t, res)
	assertConversionInputsRetained(t, res)

	const contentH = 842.0

	names := PageNames(res, contentH)
	if len(names) == 0 || names[0] != "cover" {
		t.Fatalf("PageNames after release = %v, want first page cover", names)
	}

	ensureFlowIndex(res, contentH)

	if len(res.flowPageOf) != len(res.Ops) || len(res.flowPages) == 0 {
		t.Fatalf(
			"pagination index did not rebuild on demand: flowPageOf=%d ops=%d",
			len(res.flowPageOf), len(res.Ops),
		)
	}
}

// assertPaginationIndexesReleased requires all six pagination-only indexes to
// be nil after Paint, so PDF finalization cannot retain them.
func assertPaginationIndexesReleased(t *testing.T, res *Result) {
	t.Helper()

	released := map[string]bool{
		"flowPages":   res.flowPages != nil,
		"flowPageOf":  res.flowPageOf != nil,
		"flowPos":     res.flowPos != nil,
		"flowBoxes":   res.flowBoxes != nil,
		"flowBoxPage": res.flowBoxPage != nil,
		"flowBoxPos":  res.flowBoxPos != nil,
	}

	for name, live := range released {
		if live {
			t.Errorf("%s still populated after Paint, want released for PDF-05", name)
		}
	}
}

// assertConversionInputsRetained requires the fields conversion reads after
// Paint to survive the release.
func assertConversionInputsRetained(t *testing.T, res *Result) {
	t.Helper()

	if len(res.Ops) == 0 || len(res.Pages) == 0 || len(res.Locations) == 0 {
		t.Fatal("Paint released an op/page/location field that conversion reads")
	}

	if res.root == nil || len(res.boxes) == 0 {
		t.Fatal("Paint released the box tree that PageNames and a later paint read")
	}
}
