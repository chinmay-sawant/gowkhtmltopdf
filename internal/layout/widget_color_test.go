//nolint:testpackage // layoutHTML/sheet test helpers and Result.Ops internals are tested from the same package
package layout

import (
	"testing"
)

func TestProgressUsesAccentColorNotD03Token(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t,
		`<html><body><progress value="50" max="100"></progress></body></html>`,
		sheet(t, `progress { display:block; accent-color:#cc3366; width:100px; height:12px; }`),
	)

	var saw bool

	for _, op := range res.Ops {
		if op.Kind != OpFillRect {
			continue
		}

		if approx(op.R, 204.0/255) && approx(op.G, 51.0/255) && approx(op.B, 102.0/255) {
			saw = true

			break
		}
	}

	if !saw {
		t.Fatal("progress fill did not use authored accent-color #cc3366")
	}
}

func approx(got, want float64) bool {
	d := got - want
	if d < 0 {
		d = -d
	}

	return d < 0.02
}
