package convert //nolint:testpackage // white-box benchmark of body navigation collection

import (
	"fmt"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// largeNavigationResult builds a result with idsCount id locations and
// opsCount display ops (every third op carries a structure element).
func largeNavigationResult(idsCount, opsCount int) *layout.Result {
	ops := make([]layout.Op, opsCount)

	for i := range ops {
		ops[i].Y = float64(i * 10)
		if i%3 == 0 {
			ops[i].StructElem = &pdf.StructElem{} //nolint:exhaustruct // bench needs only the pointer
		}
	}

	locs := make([]layout.ElementLocation, idsCount)

	for i := range locs {
		n := &html.Node{Attrs: map[string]string{"id": fmt.Sprintf("id-%d", i)}} //nolint:exhaustruct // bench needs only the id
		locs[i] = layout.ElementLocation{Node: n, Y: float64(i*400 + 5), H: 20}
	}

	return &layout.Result{Locations: locs, Ops: ops} //nolint:exhaustruct // bench needs only navigation fields
}

// oldCollectBodyNavigation mirrors the pre-index per-id op rescan.
func oldCollectBodyNavigation(res *layout.Result) bodyNavigation {
	if res == nil {
		return bodyNavigation{}
	}

	nav := bodyNavigation{
		ids:     make(map[string]layout.ElementLocation),
		idElems: make(map[string]*pdf.StructElem),
	}

	for _, loc := range res.Locations {
		if loc.Node == nil {
			continue
		}

		if id := loc.Node.Attribute("id"); id != "" {
			loc.Node = nil
			nav.ids[id] = loc

			for i := range res.Ops {
				op := &res.Ops[i]
				if op.StructElem != nil && op.Y >= loc.Y && op.Y <= loc.Y+loc.H+20 {
					nav.idElems[id] = op.StructElem

					break
				}
			}
		}
	}

	return nav
}

func BenchmarkCollectBodyNavigationLargeDoc(b *testing.B) {
	res := largeNavigationResult(500, 20000)

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		nav := collectBodyNavigation(res)
		if len(nav.ids) != 500 {
			b.Fatal("wrong id count")
		}
	}
}
