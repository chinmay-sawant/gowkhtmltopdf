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

	for idx := range locs {
		nodeAttrs := map[string]string{"id": fmt.Sprintf("id-%d", idx)}
		n := &html.Node{Attrs: nodeAttrs} //nolint:exhaustruct // bench needs only the id
		loc := layout.ElementLocation{    //nolint:exhaustruct // bench needs only navigation fields
			Node: n, Y: float64(idx*400 + 5), H: 20,
		}
		locs[idx] = loc
	}

	return &layout.Result{Locations: locs, Ops: ops} //nolint:exhaustruct // bench needs only navigation fields
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
