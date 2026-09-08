//nolint:testpackage,exhaustruct,wsl,lll,cyclop // layout regressions use internal state and explicit fixtures
package layout

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

func TestContainerStateEqualityIncludesFontSize(t *testing.T) {
	t.Parallel()

	a := sizeContainer{inlineSize: 240, fontSize: 12, names: "card"}
	if !sameSizeContainerState(a, a) {
		t.Fatal("identical container states should converge")
	}

	if sameSizeContainerState(a, sizeContainer{inlineSize: 240, fontSize: 14, names: "card"}) {
		t.Fatal("font-size changes must trigger a container-query recascade")
	}
}

func TestSplitCrossingRectsRemapsBoxRangeAndPreservesIdentity(t *testing.T) {
	t.Parallel()

	root := &box{ //nolint:exhaustruct // intentional zero fields
		node:    &html.Node{}, //nolint:exhaustruct // intentional zero fields
		opStart: 0,
		opEnd:   0,
	}

	res := &Result{ //nolint:exhaustruct // intentional zero fields
		root: root,
		Ops: []Op{
			{Kind: OpFillRect, X: 10, Y: 40, W: 20, H: 30}, //nolint:exhaustruct // intentional zero fields
		},
	}
	splitCrossingRects(res, 50)

	if len(res.Ops) != 2 {
		t.Fatalf("split produced %d ops, want 2", len(res.Ops))
	}

	if root.opStart != 0 || root.opEnd != 1 {
		t.Fatalf("box range = [%d,%d], want [0,1]", root.opStart, root.opEnd)
	}

	if res.Ops[0].ID == 0 || res.Ops[0].ID != res.Ops[1].ID {
		t.Fatalf("fragment identities = %d,%d, want one stable non-zero identity", res.Ops[0].ID, res.Ops[1].ID)
	}
}

func TestUsedImageSizeUsesOneAspectAndConstraintPolicy(t *testing.T) { //nolint:cyclop,funlen
	t.Parallel()

	root, err := html.Parse(`<img width="100" src="x">`)
	if err != nil {
		t.Fatal(err)
	}

	var img *html.Node

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node == nil {
			return
		}

		if node.Name == "img" {
			img = node

			return
		}

		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(root)

	if img == nil {
		t.Fatal("parsed image missing")
	}

	eng := &engine{ //nolint:exhaustruct // intentional zero fields
		opts:    Options{Width: 300}, //nolint:exhaustruct // intentional zero fields
		scale:   1,
		imgMaxW: 80,
	}

	ref := &imageRef{ //nolint:exhaustruct // intentional zero fields
		w: 400,
		h: 200,
	}

	base := ResolvedStyle{ //nolint:exhaustruct // intentional zero fields
		Width: -1, WidthPercent: -1, Height: -1, HeightPercent: -1,
		MaxWidth: -1, MaxWidthPercent: -1, MaxHeight: -1,
	}

	got := eng.usedImageSize(img, base, ref)
	if got.w != 75 || got.h != 37.5 {
		t.Fatalf("auto image size = %.2fx%.2f, want 75x37.5", got.w, got.h)
	}

	css := base
	css.Width = 120

	got = eng.usedImageSize(img, css, ref)
	if got.w != 120 || got.h != 60 {
		t.Fatalf("CSS image size = %.2fx%.2f, want 120x60", got.w, got.h)
	}

	eng.imgMaxW = 240
	css = base
	css.WidthPercent = 50
	css.MaxHeight = 30

	got = eng.usedImageSize(img, css, ref)
	if got.w != 60 || got.h != 30 {
		t.Fatalf("percent/max-height image size = %.2fx%.2f, want 60x30", got.w, got.h)
	}
}

func TestLayoutContextHonorsCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	root, err := html.Parse(`<html><body><p>cancel me</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = LayoutContext(ctx, root, Options{Width: 300, Height: 300}) //nolint:exhaustruct // intentional zero fields
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("LayoutContext error = %v, want context.Canceled", err)
	}
}

func TestLayoutContextCancelsBlockingImageResolver(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><img src="blocked"></body></html>`)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	started := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		_, err := LayoutContext(ctx, root, Options{
			Width: 300, Height: 300,
			ImagesContext: func(resolveCtx context.Context, _ string) ([]byte, error) {
				close(started)
				<-resolveCtx.Done()

				return nil, resolveCtx.Err()
			},
		}) //nolint:exhaustruct // blocking image callback is the only special option
		done <- err
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("image resolver did not start")
	}

	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("LayoutContext error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("LayoutContext remained blocked after image resolver cancellation")
	}
}

func TestCloneResultDropsDocumentOwnedStructureElements(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><h1>Title</h1><p>Body</p></body></html>`)
	res, err := LayoutContext(t.Context(), root, Options{Width: 300, Height: 300}) //nolint:exhaustruct
	if err != nil {
		t.Fatalf("LayoutContext: %v", err)
	}

	first, err := pdf.NewDocumentWithPolicy(pdf.WriterPolicy{
		Version:            pdf.PDF17,
		ConformanceProfile: pdf.ProfilePDFUA1,
	})
	if err != nil {
		t.Fatalf("first document: %v", err)
	}
	if err := Paint(first, res, PaintOptions{PageWidth: 300, PageHeight: 300}); err != nil {
		t.Fatalf("first Paint: %v", err)
	}

	clone := CloneResult(res)
	for idx, op := range clone.Ops {
		if op.StructElem != nil {
			t.Fatalf("clone op %d retained source structure element", idx)
		}
	}

	second, err := pdf.NewDocumentWithPolicy(pdf.WriterPolicy{
		Version:            pdf.PDF17,
		ConformanceProfile: pdf.ProfilePDFUA1,
	})
	if err != nil {
		t.Fatalf("second document: %v", err)
	}
	if err := Paint(second, clone, PaintOptions{PageWidth: 300, PageHeight: 300}); err != nil {
		t.Fatalf("second Paint: %v", err)
	}

	firstElems := structureElements(first.StructTreeRoot())
	secondElems := structureElements(second.StructTreeRoot())
	if len(firstElems) == 0 || len(secondElems) == 0 {
		t.Fatalf("structure tree sizes = %d and %d, want non-empty trees", len(firstElems), len(secondElems))
	}
	if len(firstElems) != len(secondElems) {
		t.Fatalf("structure tree sizes = %d and %d, want equal trees", len(firstElems), len(secondElems))
	}
	for elem := range firstElems {
		if secondElems[elem] {
			t.Fatal("cloned result shared a structure element with the source document")
		}
	}
}

func TestRepeatedTaggedPaintRebuildsDocumentStructure(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body><h1>Title</h1><p>Body</p></body></html>`)
	res, err := LayoutContext(t.Context(), root, Options{Width: 300, Height: 300}) //nolint:exhaustruct
	if err != nil {
		t.Fatalf("LayoutContext: %v", err)
	}

	first, err := pdf.NewDocumentWithPolicy(pdf.WriterPolicy{
		Version:            pdf.PDF17,
		ConformanceProfile: pdf.ProfilePDFUA1,
	})
	if err != nil {
		t.Fatalf("first document: %v", err)
	}
	if err := Paint(first, res, PaintOptions{PageWidth: 300, PageHeight: 300}); err != nil {
		t.Fatalf("first Paint: %v", err)
	}

	second, err := pdf.NewDocumentWithPolicy(pdf.WriterPolicy{
		Version:            pdf.PDF17,
		ConformanceProfile: pdf.ProfilePDFUA1,
	})
	if err != nil {
		t.Fatalf("second document: %v", err)
	}
	if err := Paint(second, res, PaintOptions{PageWidth: 300, PageHeight: 300}); err != nil {
		t.Fatalf("repeated Paint: %v", err)
	}

	firstElems := structureElements(first.StructTreeRoot())
	secondElems := structureElements(second.StructTreeRoot())
	if len(firstElems) != len(secondElems) || len(secondElems) < 3 {
		t.Fatalf("repeated structure tree sizes = %d and %d, want equal trees with headings and paragraphs", len(firstElems), len(secondElems))
	}
}

func structureElements(root *pdf.StructTreeRoot) map[*pdf.StructElem]bool {
	elems := map[*pdf.StructElem]bool{}
	var walk func(*pdf.StructElem)
	walk = func(elem *pdf.StructElem) {
		if elem == nil {
			return
		}

		elems[elem] = true
		for _, child := range elem.Kids {
			walk(child)
		}
	}
	if root != nil {
		for _, elem := range root.Children {
			walk(elem)
		}
	}

	return elems
}

func TestPaintContextHonorsCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	doc := pdf.NewDocument()
	res := &Result{ //nolint:exhaustruct // intentional zero fields
		Width: 100, Height: 100,
		Ops: []Op{
			{Kind: OpFillRect, W: 10, H: 10}, //nolint:exhaustruct // intentional zero fields
		},
	}

	err := PaintContext(ctx, doc, res, PaintOptions{ //nolint:exhaustruct // intentional zero fields
		PageWidth: 100, PageHeight: 100,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("PaintContext error = %v, want context.Canceled", err)
	}
}

func TestShiftOpsOnlyMaintainsFlowIndex(t *testing.T) {
	t.Parallel()

	res := &Result{ //nolint:exhaustruct // intentional zero fields
		Ops:          []Op{{Y: 10}, {Y: 20}}, //nolint:exhaustruct // intentional zero fields
		flowPageSize: 100,
	}

	shiftOpsOnly(res, 0, 0, 100)

	if got := res.Ops[0].Y; got != 110 {
		t.Fatalf("shifted operation Y = %.1f, want 110", got)
	}

	if got := res.flowPageOf[0]; got != 1 {
		t.Fatalf("shifted operation page = %d, want 1", got)
	}

	if got := res.flowPageOf[1]; got != 0 {
		t.Fatalf("unchanged operation page = %d, want 0", got)
	}
}

func TestShiftFlowYNegativeMaintainsFlowIndex(t *testing.T) {
	t.Parallel()

	res := &Result{ //nolint:exhaustruct // intentional zero fields
		Ops:          []Op{{Y: 110}, {Y: 210}}, //nolint:exhaustruct // intentional zero fields
		flowPageSize: 100,
	}

	shiftFlowY(res, 2, 1, 100, -100)

	if got := res.Ops[0].Y; got != 10 {
		t.Fatalf("first operation Y = %.1f, want 10", got)
	}

	if got := res.Ops[1].Y; got != 110 {
		t.Fatalf("second operation Y = %.1f, want 110", got)
	}

	if got := res.flowPageOf[0]; got != 0 {
		t.Fatalf("first operation page = %d, want 0", got)
	}

	if got := res.flowPageOf[1]; got != 1 {
		t.Fatalf("second operation page = %d, want 1", got)
	}
}

func TestGeneratedPaginationOpsInvalidateFlowIndex(t *testing.T) {
	t.Parallel()

	res := &Result{ //nolint:exhaustruct // intentional zero fields
		Ops:          []Op{{Kind: OpLine, Y: 10}}, //nolint:exhaustruct // intentional zero fields
		flowPageOf:   []int{0},
		flowPages:    [][]int{{0}},
		flowPos:      []int{0},
		flowPageSize: 100,
	}

	cloneHeaderOps(res, 0, 0, 10, 100)

	if res.flowPageOf != nil || res.flowPages != nil || res.flowPos != nil {
		t.Fatal("header cloning must invalidate cached flow indexes")
	}
}

func BenchmarkUsedImageSize(b *testing.B) {
	eng := &engine{opts: Options{Width: 640}, scale: 1, imgMaxW: 320} //nolint:exhaustruct // intentional zero fields

	root, err := html.Parse(`<img width="400" src="x">`)
	if err != nil {
		b.Fatal(err)
	}

	var img *html.Node

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node == nil || img != nil {
			return
		}

		if node.Name == "img" {
			img = node

			return
		}

		for _, child := range node.Children {
			walk(child)
		}
	}
	walk(root)

	baseStyle := ResolvedStyle{ //nolint:exhaustruct // intentional zero fields
		Width: -1, WidthPercent: -1, Height: -1, HeightPercent: -1,
		MaxWidth: -1, MaxWidthPercent: -1, MaxHeight: -1,
	}

	ref := &imageRef{ //nolint:exhaustruct // intentional zero fields
		w: 800,
		h: 400,
	}

	b.ReportAllocs()

	for b.Loop() {
		_ = eng.usedImageSize(img, baseStyle, ref)
	}
}

func BenchmarkDisplayListIdentity10kOps100Pages(b *testing.B) {
	makeResult := func() *Result {
		ops := make([]Op, 10_000)
		for idx := range ops {
			ops[idx] = Op{ //nolint:exhaustruct // intentional zero fields
				Kind: OpFillRect,
				X:    float64(idx % 100),
				Y:    float64(idx/100) * 100,
				W:    1,
				H:    1,
				ID:   uint64(idx) + 1, //nolint:gosec // idx is a non-negative range index
			}
		}

		return &Result{ //nolint:exhaustruct // intentional zero fields
			Width: 640, Height: 10_000, Ops: ops,
			root: &box{opStart: 0, opEnd: len(ops) - 1, height: 10_000}, //nolint:exhaustruct // intentional zero fields
		}
	}

	b.ReportAllocs()

	for b.Loop() {
		res := makeResult()
		doc := pdf.NewDocument()

		if err := PaintContext(b.Context(), doc, res, PaintOptions{ //nolint:exhaustruct // intentional zero fields
			PageWidth: 640, PageHeight: 100,
		}); err != nil {
			b.Fatal(err)
		}

		if doc.PageCount() != 100 {
			b.Fatalf("page count = %d, want 100", doc.PageCount())
		}
	}
}
