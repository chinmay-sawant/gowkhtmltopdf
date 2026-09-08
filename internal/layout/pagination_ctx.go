package layout

import "context"

// ctxPoll is a bounded-interval cancellation poller for row/line loops whose
// per-iteration work is smaller than one engine recursion boundary: table
// rows, flex lines, grid cells, and display-list scans. It mirrors the
// styleContext cadence (style.go pollContext): the context is touched once
// every 64 polls, keeping cancellation latency proportional to 64 iterations
// of row/line work instead of the whole pass.
//
// ctxPoll is single-goroutine (one engine pass) and not safe for concurrent
// use.
type ctxPoll struct {
	ctx  context.Context
	err  error
	work int
}

// newCtxPoll builds a poller for one loop pass. A nil ctx never stops,
// matching the legacy background-context adapters.
func newCtxPoll(ctx context.Context) *ctxPoll {
	return &ctxPoll{ctx: ctx}
}

// poll reports whether the loop must stop and caches the cause in err. The
// first 63 polls skip the context entirely.
func (p *ctxPoll) poll() bool {
	if p.err != nil {
		return true
	}

	if p.ctx == nil {
		return false
	}

	p.work++
	if p.work&63 != 0 {
		return false
	}

	p.err = p.ctx.Err()

	return p.err != nil
}
