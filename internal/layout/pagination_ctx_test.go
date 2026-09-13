package layout

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// tableStressHTML builds a table with nRows rows so layout spends its time in
// the table row loops (measure/place/emit) rather than in style resolution.
func tableStressHTML(nRows int) string {
	var buf strings.Builder

	buf.WriteString("<html><body><table>")

	for i := range nRows {
		_, _ = fmt.Fprintf(&buf, "<tr><td>row %d</td><td>%d</td><td>alpha beta gamma</td></tr>", i, i*7)
	}

	buf.WriteString("</table></body></html>")

	return buf.String()
}

// TestLayoutContextCancelDuringTableLayout proves the table row loops poll
// ctx (LAY-04): a cancel issued while a large table is being laid out aborts
// LayoutContext with context.Canceled instead of finishing the whole table.
func TestLayoutContextCancelDuringTableLayout(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(tableStressHTML(1500))
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)

	go func() {
		time.Sleep(2 * time.Millisecond)
		cancel()
	}()

	start := time.Now()

	go func() {
		opts := Options{Width: testViewport, Height: 800}
		_, err := LayoutContext(ctx, root, opts)
		done <- err
	}()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("LayoutContext error = %v, want context.Canceled", err)
		}

		if elapsed := time.Since(start); elapsed > 5*time.Second {
			t.Fatalf("cancel-during-table-layout took %v, want abort well under the full layout time", elapsed)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("LayoutContext ignored cancellation during table layout")
	}
}

// TestPaginateOpsHonorsCancellation proves paginateOps returns the ctx error
// (LAY-04): the fixpoint loops and the final display-list scan check ctx.
func TestPaginateOpsHonorsCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	res := &Result{
		Ops: []Op{
			{Kind: OpText, Y: 10, Size: 12},
			{Kind: OpText, Y: 120, Size: 12},
		},
	}

	if err := paginateOps(ctx, res, 100); !errors.Is(err, context.Canceled) {
		t.Fatalf("paginateOps error = %v, want context.Canceled", err)
	}
}

// TestPaintContextCancelDuringPagination proves PaintContext surfaces the
// pagination-phase cancellation instead of painting a partially paginated
// display list.
func TestPaintContextCancelDuringPagination(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	doc := pdf.NewDocument()
	res := &Result{
		Ops: []Op{
			{Kind: OpFillRect, W: 10, H: 10},
			{Kind: OpText, Y: 10, Size: 12},
		},
	}

	err := PaintContext(ctx, doc, res, PaintOptions{
		PageWidth: 100, PageHeight: 100,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("PaintContext error = %v, want context.Canceled", err)
	}
}

// TestFilterKindZeroIsUnknown proves the filter enum sentinel (LAY-02): the
// zero value is not a real filter, so an unassigned parsedFilter is a no-op
// instead of a blur.
func TestFilterKindZeroIsUnknown(t *testing.T) {
	t.Parallel()

	if filterKind(0) != filterUnknown {
		t.Fatalf("filterKind zero = %d, want filterUnknown", filterKind(0))
	}

	input := []byte("not-an-image")
	filters := []parsedFilter{{kind: 0}}

	if got := applyImageFilterToImage(input, filters); string(got) != string(input) {
		t.Fatal("zero-value filter kind must be a no-op on the image bytes")
	}
}

// TestTrackSizeKindZeroIsUnknown proves the grid track sentinel (LAY-02): a
// zero-value gridTrackSize resolves to zero rather than pretending to be a
// fixed track.
func TestTrackSizeKindZeroIsUnknown(t *testing.T) {
	t.Parallel()

	if trackSizeKind(0) != trackUnknown {
		t.Fatalf("trackSizeKind zero = %d, want trackUnknown", trackSizeKind(0))
	}

	var eng engine

	zeroTrack := gridTrackSize{}
	zeroIntrinsic := trackIntrinsic{}

	if got := resolveTrackSide(zeroTrack, 500, true, &eng, zeroIntrinsic, false); got != 0 {
		t.Fatalf("resolveTrackSide(zero) = %v, want 0", got)
	}
}

// TestBreakPolicyZeroDefaults document the intentional zero-value contracts
// (LAY-02): breakNormal and softBreakNone are the safe defaults, so a
// zero-valued policy never splits a token where the author did not allow it.
func TestBreakPolicyZeroDefaults(t *testing.T) {
	t.Parallel()

	if wordBreakPolicy(0) != breakNormal {
		t.Fatalf("wordBreakPolicy zero = %d, want breakNormal", wordBreakPolicy(0))
	}

	if softBreakMode(0) != softBreakNone {
		t.Fatalf("softBreakMode zero = %d, want softBreakNone", softBreakMode(0))
	}
}
