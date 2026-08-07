package layout

import (
	"context"
	"errors"
	"testing"

	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

func TestContainerStateEqualityIncludesFontSize(t *testing.T) {
	a := sizeContainer{inlineSize: 240, fontSize: 12, names: "card"}
	if !sameSizeContainerState(a, a) {
		t.Fatal("identical container states should converge")
	}

	if sameSizeContainerState(a, sizeContainer{inlineSize: 240, fontSize: 14, names: "card"}) {
		t.Fatal("font-size changes must trigger a container-query recascade")
	}
}

func TestSplitCrossingRectsRemapsBoxRangeAndPreservesIdentity(t *testing.T) {
	root := &box{node: &html.Node{}, opStart: 0, opEnd: 0}
	res := &Result{root: root, Ops: []Op{{Kind: OpFillRect, X: 10, Y: 40, W: 20, H: 30}}}
	splitCrossingRects(res, 50, nil)

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

func TestUsedImageSizeUsesOneAspectAndConstraintPolicy(t *testing.T) {
	root, err := html.Parse(`<img width="100" src="x">`)
	if err != nil {
		t.Fatal(err)
	}

	var img *html.Node

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n == nil {
			return
		}

		if n.Name == "img" {
			img = n

			return
		}

		for _, child := range n.Children {
			walk(child)
		}
	}
	walk(root)

	if img == nil {
		t.Fatal("parsed image missing")
	}

	e := &engine{opts: Options{Width: 300}, scale: 1, imgMaxW: 80}
	ref := &imageRef{w: 400, h: 200}
	base := ResolvedStyle{Width: -1, WidthPercent: -1, Height: -1, HeightPercent: -1, MaxWidth: -1, MaxWidthPercent: -1, MaxHeight: -1}

	got := e.usedImageSize(img, base, ref)
	if got.w != 75 || got.h != 37.5 {
		t.Fatalf("auto image size = %.2fx%.2f, want 75x37.5", got.w, got.h)
	}

	css := base
	css.Width = 120

	got = e.usedImageSize(img, css, ref)
	if got.w != 120 || got.h != 60 {
		t.Fatalf("CSS image size = %.2fx%.2f, want 120x60", got.w, got.h)
	}

	e.imgMaxW = 240
	css = base
	css.WidthPercent = 50
	css.MaxHeight = 30

	got = e.usedImageSize(img, css, ref)
	if got.w != 60 || got.h != 30 {
		t.Fatalf("percent/max-height image size = %.2fx%.2f, want 60x30", got.w, got.h)
	}
}

func TestLayoutContextHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	root, err := html.Parse(`<html><body><p>cancel me</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = LayoutContext(ctx, root, Options{Width: 300, Height: 300})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("LayoutContext error = %v, want context.Canceled", err)
	}
}

func TestPaintContextHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	doc := pdf.NewDocument()
	res := &Result{Width: 100, Height: 100, Ops: []Op{{Kind: OpFillRect, W: 10, H: 10}}}

	err := PaintContext(ctx, doc, res, PaintOptions{PageWidth: 100, PageHeight: 100})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("PaintContext error = %v, want context.Canceled", err)
	}
}

func TestShiftOpsOnlyMaintainsFlowIndex(t *testing.T) {
	res := &Result{
		Ops:          []Op{{Y: 10}, {Y: 20}},
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
	res := &Result{
		Ops:          []Op{{Y: 110}, {Y: 210}},
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

func BenchmarkUsedImageSize(b *testing.B) {
	e := &engine{opts: Options{Width: 640}, scale: 1, imgMaxW: 320}

	root, err := html.Parse(`<img width="400" src="x">`)
	if err != nil {
		b.Fatal(err)
	}

	var img *html.Node

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n == nil || img != nil {
			return
		}

		if n.Name == "img" {
			img = n

			return
		}

		for _, child := range n.Children {
			walk(child)
		}
	}
	walk(root)

	st := ResolvedStyle{Width: -1, WidthPercent: -1, Height: -1, HeightPercent: -1, MaxWidth: -1, MaxWidthPercent: -1, MaxHeight: -1}
	ref := &imageRef{w: 800, h: 400}

	b.ReportAllocs()

	for range b.N {
		_ = e.usedImageSize(img, st, ref)
	}
}

func BenchmarkDisplayListIdentity10kOps100Pages(b *testing.B) {
	makeResult := func() *Result {
		ops := make([]Op, 10_000)
		for i := range ops {
			ops[i] = Op{
				Kind: OpFillRect,
				X:    float64(i % 100),
				Y:    float64(i/100) * 100,
				W:    1,
				H:    1,
				ID:   uint64(i + 1),
			}
		}

		return &Result{
			Width: 640, Height: 10_000, Ops: ops,
			root: &box{opStart: 0, opEnd: len(ops) - 1, h: 10_000},
		}
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		res := makeResult()
		doc := pdf.NewDocument()

		if err := PaintContext(b.Context(), doc, res, PaintOptions{PageWidth: 640, PageHeight: 100}); err != nil {
			b.Fatal(err)
		}

		if doc.PageCount() != 100 {
			b.Fatalf("page count = %d, want 100", doc.PageCount())
		}
	}
}
