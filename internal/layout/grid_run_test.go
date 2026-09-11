package layout

// resLineOps expands a result's display list into line ops, unpacking grid
// runs so display-list geometry assertions keep seeing one entry per painted
// line. Order is preserved.
func resLineOps(res *Result) []Op {
	out := make([]Op, 0, len(res.Ops))

	visitLineOps(res.Ops, func(line Op) {
		out = append(out, line)
	})

	return out
}
