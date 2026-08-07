package layout

import (
	"math"
	"testing"
)

func TestJustifyCapsRiversBesideFloat(t *testing.T) {
	s := sheet(t, `
.wrap { width: 300pt }
.box { float: right; width: 180pt; height: 80pt; background: #ccc }
p { text-align: justify; font-size: 12pt }
`)
	res := layoutHTML(t, `<html><body><div class="wrap">
<div class="box">x</div>
<p>one two three four</p>
</div></body></html>`, s)
	texts := opsOfKind(res, OpText)

	var line []Op

	for _, op := range texts {
		if op.Text == "x" {
			continue
		}

		if len(line) == 0 {
			line = append(line, op)

			continue
		}

		if math.Abs(op.Y-line[0].Y) < 0.5 {
			line = append(line, op)
		}
	}

	if len(line) < 2 {
		t.Fatalf("expected multi-word line beside float, got %+v", texts)
	}
	// Gaps between consecutive words must stay modest (no 50pt+ rivers).
	for i := 1; i < len(line); i++ {
		gap := line[i].X - (line[i-1].X + line[i-1].W)
		if gap > 20 {
			t.Errorf("word gap %v between %q and %q exceeds 20pt (river)", gap, line[i-1].Text, line[i].Text)
		}
	}
}
