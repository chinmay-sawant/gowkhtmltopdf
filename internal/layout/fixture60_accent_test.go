//nolint:testpackage,wsl,cyclop // display-list regression assertions stay together for readability.
package layout

import (
	"math"
	"testing"
)

func TestFixture60AccentCheckboxTickAndAlignment(t *testing.T) {
	t.Parallel()

	res, _, _ := layoutFixture60(t)
	input, label := fixture60AccentBoxes(t, res)

	var fill Op
	ticks := make([]Op, 0, 2)
	for idx := input.opStart; idx <= input.opEnd && idx < len(res.Ops); idx++ {
		op := res.Ops[idx]
		if op.Kind == OpFillRect {
			fill = op
		}
		if op.Kind == OpLine && op.R == 1 && op.G == 1 && op.B == 1 {
			ticks = append(ticks, op)
		}
	}

	if fill.W <= 0 || fill.H <= 0 {
		t.Fatal("checked checkbox fill not found")
	}
	if len(ticks) != 2 {
		t.Fatalf("checkbox tick operations = %d, want 2", len(ticks))
	}
	if ticks[0].H <= 0 || ticks[1].H >= 0 {
		t.Fatalf("checkbox tick directions = %.3f, %.3f, want down then up", ticks[0].H, ticks[1].H)
	}

	firstEndX := ticks[0].X + ticks[0].W
	firstEndY := ticks[0].Y + ticks[0].H
	if math.Abs(firstEndX-ticks[1].X) > 0.001 || math.Abs(firstEndY-ticks[1].Y) > 0.001 {
		t.Fatalf("checkbox tick segments do not join: first end=(%.3f,%.3f), second start=(%.3f,%.3f)",
			firstEndX, firstEndY, ticks[1].X, ticks[1].Y)
	}

	textOps := make([]Op, 0, 2)
	for idx := label.opStart; idx <= label.opEnd && idx < len(res.Ops); idx++ {
		if res.Ops[idx].Kind == OpText {
			textOps = append(textOps, res.Ops[idx])
		}
	}
	if len(textOps) != 2 {
		t.Fatalf("green check text lines = %d, want 2", len(textOps))
	}

	textTop := textOps[0].Y - (textOps[0].H - textOps[0].InkDescent)
	textBottom := textOps[len(textOps)-1].Y + textOps[len(textOps)-1].InkDescent
	checkboxCenter := fill.Y + fill.H/2
	textCenter := (textTop + textBottom) / 2
	if math.Abs(checkboxCenter-textCenter) > 1.5 {
		t.Fatalf("checkbox center %.3f is misaligned with green check text center %.3f", checkboxCenter, textCenter)
	}
}

func fixture60AccentBoxes(t *testing.T, res *Result) (*box, *box) {
	t.Helper()

	var input, label *box
	for _, boxNode := range flowBoxList(res) {
		if boxNode.node == nil {
			continue
		}

		if boxNode.node.Name == htmlInput && boxNode.node.Attribute("type") == "checkbox" {
			input = boxNode
		}
		if boxNode.node.Name == "span" && boxNode.node.TextContent() == "green check" {
			label = boxNode
		}
	}
	if input == nil || label == nil {
		t.Fatalf("fixture 60 accent controls missing: input=%v label=%v", input != nil, label != nil)
	}

	return input, label
}
