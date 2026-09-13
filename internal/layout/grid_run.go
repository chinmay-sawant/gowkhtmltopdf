package layout

import (
	"math"
	"slices"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// GridSeg is one line segment of an OpGridRun, in canvas coordinates.
// Segments reproduce exactly the OpLine entries a collapsed table row used to
// append one by one, so painters replay the same PDF path operators in the
// same order and output bytes stay identical.
type GridSeg struct {
	X, Y, W, H float64
	Width      float64
	R, G, B    float64
	LineInset  uint8
}

// GridRun holds the ordered line segments of one table row's collapsed grid.
// The owning Op carries the run's bounding box and its first segment's stroke
// metadata for readers that only need coarse geometry.
type GridRun struct {
	Segs []GridSeg
}

// asLine returns segment idx as a standalone OpLine, copying the run's shared
// identity and paint metadata so per-line passes see the same shape they saw
// before the grid was batched.
func (run *GridRun) asLine(owner *Op, idx int) Op {
	seg := &run.Segs[idx]
	line := *owner
	line.Kind = OpLine
	line.X, line.Y, line.W, line.H = seg.X, seg.Y, seg.W, seg.H
	line.Width, line.R, line.G, line.B = seg.Width, seg.R, seg.G, seg.B
	line.LineInset = seg.LineInset
	line.Grid = nil

	return line
}

// forEachLine calls visit for every line an op paints: a plain op visits
// itself, a grid run visits each segment as a standalone OpLine in emission
// order.
func (paintOp Op) forEachLine(visit func(line Op)) {
	if paintOp.Kind != OpGridRun || paintOp.Grid == nil {
		visit(paintOp)

		return
	}

	for idx := range paintOp.Grid.Segs {
		visit(paintOp.Grid.asLine(&paintOp, idx))
	}
}

// visitLineOps calls visit once for every line op in ops, expanding grid runs
// in emission order.
func visitLineOps(ops []Op, visit func(line Op)) {
	for idx := range ops {
		ops[idx].forEachLine(visit)
	}
}

// forEachLineIndex visits (op index, segment index) for every line in
// [from,to]; segment index is -1 for a plain line op.
func forEachLineIndex(ops []Op, from, last int, visit func(opIdx, segIdx int)) {
	if from < 0 {
		from = 0
	}

	if last >= len(ops) {
		last = len(ops) - 1
	}

	for idx := from; idx <= last; idx++ {
		if ops[idx].Kind == OpGridRun && ops[idx].Grid != nil {
			for segIdx := range ops[idx].Grid.Segs {
				visit(idx, segIdx)
			}

			continue
		}

		visit(idx, -1)
	}
}

// lineViewAt returns the line at (opIdx, segIdx) as a standalone Op copy.
func lineViewAt(ops []Op, opIdx, segIdx int) Op {
	if segIdx < 0 || ops[opIdx].Grid == nil {
		return ops[opIdx]
	}

	return ops[opIdx].Grid.asLine(&ops[opIdx], segIdx)
}

// addLineAtY shifts one line inside its op by deltaY. Plain ops move the op;
// grid runs move the addressed segment and the run bounding box.
func addLineAtY(ops []Op, opIdx, segIdx int, deltaY float64) {
	if segIdx < 0 || ops[opIdx].Grid == nil {
		ops[opIdx].Y += deltaY

		return
	}

	ops[opIdx].Grid.Segs[segIdx].Y += deltaY
	recomputeGridRunBounds(&ops[opIdx])
}

// extendLineBottom grows the addressed line so its bottom reaches bottom.
func extendLineBottom(ops []Op, opIdx, segIdx int, bottom float64) {
	if segIdx < 0 || ops[opIdx].Grid == nil {
		if lineBottom := ops[opIdx].Y + ops[opIdx].H; lineBottom < bottom {
			ops[opIdx].H += bottom - lineBottom
		}

		return
	}

	seg := &ops[opIdx].Grid.Segs[segIdx]
	if lineBottom := seg.Y + seg.H; lineBottom < bottom {
		seg.H += bottom - lineBottom
	}

	recomputeGridRunBounds(&ops[opIdx])
}

// recomputeGridRunBounds rebuilds a run's bounding box after segment edits.
func recomputeGridRunBounds(paintOp *Op) {
	if paintOp.Grid == nil || len(paintOp.Grid.Segs) == 0 {
		return
	}

	minX, minY := paintOp.Grid.Segs[0].X, paintOp.Grid.Segs[0].Y
	maxX, maxY := minX+paintOp.Grid.Segs[0].W, minY+paintOp.Grid.Segs[0].H

	for idx := 1; idx < len(paintOp.Grid.Segs); idx++ {
		seg := &paintOp.Grid.Segs[idx]
		minX = math.Min(minX, seg.X)
		minY = math.Min(minY, seg.Y)
		maxX = math.Max(maxX, seg.X+seg.W)
		maxY = math.Max(maxY, seg.Y+seg.H)
	}

	paintOp.X, paintOp.Y, paintOp.W, paintOp.H = minX, minY, maxX-minX, maxY-minY
}

// addGridRun appends one row's grid segments as a single OpGridRun. The run
// carries the bounding box and the first segment's stroke style so generic
// geometry passes keep working; painters walk every segment in order. The
// segments are copied out of the caller's scratch buffer at exact size, so
// appending never reallocates a growing run.
func (e *engine) addGridRun(segs []GridSeg) {
	if len(segs) == 0 || e.checkContext() || e.noEmit {
		return
	}

	minX, minY := segs[0].X, segs[0].Y
	maxX, maxY := segs[0].X+segs[0].W, segs[0].Y+segs[0].H

	for idx := 1; idx < len(segs); idx++ {
		seg := &segs[idx]

		if seg.X < minX {
			minX = seg.X
		}

		if seg.Y < minY {
			minY = seg.Y
		}

		if seg.X+seg.W > maxX {
			maxX = seg.X + seg.W
		}

		if seg.Y+seg.H > maxY {
			maxY = seg.Y + seg.H
		}
	}

	e.nextOpID++

	blend := e.blendMode
	if blend == blendNormal {
		blend = ""
	}

	gridOp := (Op{ //nolint:exhaustruct // intentional zero fields
		ID: e.nextOpID, Kind: OpGridRun,
		X: minX, Y: minY, W: maxX - minX, H: maxY - minY,
		R: segs[0].R, G: segs[0].G, B: segs[0].B, Width: segs[0].Width,
		Grid:       &GridRun{Segs: slices.Clone(segs)},
		ZIndex:     e.zIndex,
		ZIndexSet:  e.zIndexSet,
		Positioned: e.positioned,
	}).withBlendMode(blend)
	gridOp.bindEmptyExtra()
	gridOp.setBlendGroup(e.blendGroup)

	e.ops = append(e.ops, gridOp)
}

// shiftGridRunY moves every segment of a run down the canvas, mirroring the
// whole-op Y shift pagination applies to plain ops.
func shiftGridRunY(paintOp *Op, deltaY float64) {
	for idx := range paintOp.Grid.Segs {
		paintOp.Grid.Segs[idx].Y += deltaY
	}
}

// shiftOpY moves an op down the canvas. A grid run moves every segment with
// its bounding box, so per-segment geometry never drifts behind the run's
// page assignment.
func shiftOpY(paintOp *Op, deltaY float64) {
	paintOp.Y += deltaY

	if paintOp.Grid != nil {
		shiftGridRunY(paintOp, deltaY)
	}
}

// shiftOpX moves an op right on the canvas, carrying grid segments with it.
func shiftOpX(paintOp *Op, deltaX float64) {
	paintOp.X += deltaX

	if paintOp.Grid == nil {
		return
	}

	for idx := range paintOp.Grid.Segs {
		paintOp.Grid.Segs[idx].X += deltaX
	}
}

// drawGridRun replays one row's grid segments as individual lines, in the
// order they were emitted, so the PDF content stream matches the pre-batch
// output exactly.
func (p *pagePainter) drawGridRun(runOp *Op, target *pdf.Content) {
	if runOp.Grid == nil {
		return
	}

	for idx := range runOp.Grid.Segs {
		line := runOp.Grid.asLine(runOp, idx)
		drawLine(target, &line, p.pageN, p.contentH, p.opts, p.pageH)
	}
}
