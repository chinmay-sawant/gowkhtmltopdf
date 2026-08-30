package layout

import (
	"math"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// resolveGridRows sizes the row tracks, returning the final row count.
func resolveGridRows(
	eng *engine,
	sty ResolvedStyle,
	kids []*html.Node,
	numRows int,
	contentH, rowGap float64,
	definiteRows bool,
) ([]float64, int) {
	var rows []float64

	if definiteRows {
		rowDefs := parseGridTrackDefs(sty.GridTemplateRows)
		// Pad/truncate defs to placed row count when template is shorter.
		for len(rowDefs) < numRows {
			rowDefs = append(rowDefs, flexibleTrack(1))
		}

		rowIntrinsics := measureTrackIntrinsics(eng, kids, len(rowDefs), false)
		rows = resolveGridTrackSizes(rowDefs, contentH, rowGap, eng, rowIntrinsics)
	}

	switch {
	case len(rows) == 0:
		rows = make([]float64, numRows)

		if mins := parseGridTrackFixedMins(sty.GridTemplateRows, eng); len(mins) > 0 {
			for i := 0; i < numRows && i < len(mins); i++ {
				if mins[i] > 0 {
					rows[i] = mins[i]
				}
			}
		}
	case len(rows) < numRows:
		rows = padGridRowSizes(rows, numRows)
	case len(rows) > numRows:
		numRows = len(rows)
	}

	return rows, numRows
}

// padGridRowSizes extends a row-size slice to n entries, zero-filling.
func padGridRowSizes(rows []float64, n int) []float64 {
	padded := make([]float64, n)
	copy(padded, rows)

	return padded
}

type trackIntrinsic struct {
	minContent float64
	maxContent float64
}

// measureTrackIntrinsics estimates min/max-content contributions per track
// using text measure APIs. Spanning items contribute to the first track only
// (lite). axisColumns=true measures widths; false measures preferred heights.
func measureTrackIntrinsics(eng *engine, kids []*html.Node, nTracks int, axisColumns bool) []trackIntrinsic {
	if nTracks < 1 {
		return nil
	}

	out := make([]trackIntrinsic, nTracks)
	if eng == nil || len(kids) == 0 {
		return out
	}

	for i, kid := range kids {
		cstate := eng.stylePtr(kid)
		tidx := i % nTracks

		var val float64
		if axisColumns {
			val = eng.measureCellContent(kid, *cstate)
		} else {
			// Height intrinsic: single-line text approximation via font size.
			val = eng.scalePt(cstate.FontSize) * 1.2
			val += eng.scalePt(cstate.PaddingTop) + eng.scalePt(cstate.PaddingBottom) +
				eng.scalePt(cstate.BorderTop.Width) + eng.scalePt(cstate.BorderBottom.Width)
		}

		if val > out[tidx].minContent {
			out[tidx].minContent = val
		}

		if val > out[tidx].maxContent {
			out[tidx].maxContent = val
		}
	}

	return out
}

// gridTrackPlan holds the resolved base/limit sizes and fr factors for the
// tracks of one axis.
type gridTrackPlan struct {
	base, limit, frCoef []float64
	frSum               float64
}

// planGridTrackSides resolves each track's base/limit sizes and fr factors.
func planGridTrackSides(
	defs []gridTrackDef,
	contentSize float64,
	definite bool,
	eng *engine,
	intrinsics []trackIntrinsic,
) gridTrackPlan {
	node := len(defs)

	plan := gridTrackPlan{
		base:   make([]float64, node),
		limit:  make([]float64, node),
		frCoef: make([]float64, node),
		frSum:  0,
	}

	for idx, def := range defs {
		var intr trackIntrinsic
		if idx < len(intrinsics) {
			intr = intrinsics[idx]
		}

		plan.base[idx] = resolveTrackSide(def.min, contentSize, definite, eng, intr, true)
		lim := resolveTrackSide(def.max, contentSize, definite, eng, intr, false)

		switch {
		case def.max.kind == trackFr:
			plan.frCoef[idx] = def.max.val
			if plan.frCoef[idx] <= 0 {
				plan.frCoef[idx] = 1
			}

			plan.frSum += plan.frCoef[idx]
			plan.limit[idx] = math.Inf(1)
		case def.min.kind == trackFr:
			// Rare minmax(1fr, 200px): treat fr as flex with max cap.
			plan.frCoef[idx] = def.min.val
			if plan.frCoef[idx] <= 0 {
				plan.frCoef[idx] = 1
			}

			plan.frSum += plan.frCoef[idx]
			plan.base[idx] = 0
			plan.limit[idx] = lim
		default:
			plan.limit[idx] = lim
			if plan.limit[idx] < plan.base[idx] {
				plan.limit[idx] = plan.base[idx]
			}
		}
		// Auto max with auto/fixed min -> growable to content (use max-content as soft limit).
		applyAutoSoftLimit(plan.limit, def, intr, idx)
	}

	return plan
}

// applyAutoSoftLimit caps growable auto/max-content tracks at their measured
// max-content size.
func applyAutoSoftLimit(limit []float64, def gridTrackDef, intr trackIntrinsic, idx int) {
	if def.max.kind != trackAuto && def.max.kind != trackMaxContent {
		return
	}

	if intr.maxContent <= limit[idx] && !math.IsInf(limit[idx], 1) {
		return
	}

	if def.max.kind == trackMaxContent && intr.maxContent > 0 {
		limit[idx] = intr.maxContent
	}
}

// distributeGridTracks shares leftover space between fr tracks, or between
// growable auto tracks when no fr tracks exist.
func distributeGridTracks(defs []gridTrackDef, base, limit, frCoef []float64, frSum, free float64) []float64 {
	out := make([]float64, len(defs))

	if frSum > 0 && free > 0 {
		for idx := range out {
			out[idx] = base[idx]
			if frCoef[idx] > 0 {
				out[idx] += free * (frCoef[idx] / frSum)
			}

			if out[idx] > limit[idx] {
				out[idx] = limit[idx]
			}
		}

		return out
	}

	return distributeAutoGridTracks(defs, base, limit, free, frSum)
}

// isAutoTrackKind reports whether a track can absorb leftover space.
func isAutoTrackKind(kind trackSizeKind) bool {
	return kind == trackAuto || kind == trackMaxContent || kind == trackMinContent
}

// distributeAutoGridTracks shares leftover space equally among auto tracks.
func distributeAutoGridTracks(defs []gridTrackDef, base, limit []float64, free, frSum float64) []float64 {
	out := make([]float64, len(defs))
	autoIdx := []int{}

	for i, d := range defs {
		out[i] = base[i]

		if isAutoTrackKind(d.max.kind) {
			autoIdx = append(autoIdx, i)
		}
	}

	if free > 0 && len(autoIdx) > 0 && frSum == 0 {
		each := free / float64(len(autoIdx))
		for _, i := range autoIdx {
			out[i] += each
			if out[i] > limit[i] && !math.IsInf(limit[i], 1) {
				out[i] = limit[i]
			}
		}
	}

	return out
}

// sanitizeGridTrackSizes clamps NaN/negative track sizes to zero.
func sanitizeGridTrackSizes(out []float64) {
	for i := range out {
		if out[i] < 0 || math.IsNaN(out[i]) {
			out[i] = 0
		}
	}
}

// resolveGridTrackSizes distributes free space with fr, honoring minmax floors.
// Percent mins/maxes require a definite contentSize (>=0); otherwise % -> auto.
func resolveGridTrackSizes( //nolint:cyclop // grid track sizing has independent definite/negative/fr branches
	defs []gridTrackDef,
	contentSize, gap float64,
	eng *engine,
	intrinsics []trackIntrinsic,
) []float64 {
	node := len(defs)
	if node == 0 {
		return nil
	}

	gapTotal := 0.0
	if node > 1 {
		gapTotal = gap * float64(node-1)
	}

	definite := contentSize >= 0 && !math.IsNaN(contentSize) && !math.IsInf(contentSize, 0)

	plan := planGridTrackSides(defs, contentSize, definite, eng, intrinsics)

	fixedSum := 0.0
	for i := range plan.base {
		fixedSum += plan.base[i]
	}

	free := contentSize - gapTotal - fixedSum
	if !definite {
		free = 0
	}

	if free < 0 {
		// A bare fr track is flexible when the definite grid container is
		// narrower than its intrinsic contributions. Let the fr tracks absorb
		// the available space instead of allowing an auto minimum to make the
		// whole grid overflow and trigger document-wide smart shrinking.
		if plan.frSum > 0 {
			fixedSum = 0

			for idx, coef := range plan.frCoef {
				if coef == 0 {
					fixedSum += plan.base[idx]

					continue
				}

				plan.base[idx] = 0
			}

			free = contentSize - gapTotal - fixedSum
		}

		if free < 0 {
			free = 0
		}
	}

	out := distributeGridTracks(defs, plan.base, plan.limit, plan.frCoef, plan.frSum, free)
	sanitizeGridTrackSizes(out)

	return out
}

// resolveTrackSide resolves one min or max track size.
// pctSentinel: trackFixed with val < 0 stores -percent.
func resolveTrackSide(
	size gridTrackSize,
	contentSize float64,
	definite bool,
	eng *engine,
	intr trackIntrinsic,
	isMin bool,
) float64 {
	switch size.kind {
	case trackFixed:
		return resolveTrackFixedSide(size, contentSize, definite, eng, isMin)
	case trackFr:
		if isMin {
			return 0
		}

		return math.Inf(1)
	case trackMinContent:
		return intr.minContent
	case trackMaxContent:
		return intr.maxContent
	case trackAuto:
		if isMin {
			return intr.minContent // auto min ~= min-content lite
		}

		return math.Inf(1)
	}

	return 0
}

// resolveTrackFixedSide resolves a fixed (or percentage) track side.
// Percentage: cyclic honesty - indefinite container -> auto (0 min / inf max).
func resolveTrackFixedSide(size gridTrackSize, contentSize float64, definite bool, eng *engine, isMin bool) float64 {
	if size.val < 0 {
		if !definite || contentSize < 0 {
			if isMin {
				return 0
			}

			return math.Inf(1)
		}

		pct := -size.val

		return contentSize * pct / 100
	}

	if eng != nil {
		return eng.scalePt(size.val)
	}

	return size.val
}
