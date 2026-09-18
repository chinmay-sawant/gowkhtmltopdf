package pdf

import (
	"os"
	"path/filepath"
	"testing"
)

func gowkVarTTF(t *testing.T) []byte {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", "fonts", "implemented-audit", "GowkVar-VF.ttf")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read GowkVar-VF: %v", err)
	}

	return data
}

func parseGowkVar(t *testing.T) *Font {
	t.Helper()

	face, err := ParseTTF(gowkVarTTF(t))
	if err != nil {
		t.Fatalf("ParseTTF GowkVar: %v", err)
	}

	if !face.HasVariationAxes() {
		t.Fatal("GowkVar-VF has no fvar")
	}

	return face
}

func TestInstanceWghtChangesAdvanceAndOutline(t *testing.T) {
	t.Parallel()

	src := parseGowkVar(t)
	defAdv := src.Advance('A')
	defContours := src.GlyphContours('A')

	inst := src.Instance([]Variation{{Tag: "wght", Value: 900}})
	if inst == nil || inst == src {
		t.Fatal("Instance(wght 900) returned the default face")
	}

	if inst.HasVariationAxes() {
		t.Fatal("instanced face still advertises fvar")
	}

	gotAdv := inst.Advance('A')
	if gotAdv <= defAdv {
		t.Fatalf("wght 900 advance = %g, want > default %g", gotAdv, defAdv)
	}

	gotContours := inst.GlyphContours('A')
	if outlineWidth(gotContours) <= outlineWidth(defContours) {
		t.Fatalf("wght 900 outline width = %g, want > default %g", outlineWidth(gotContours), outlineWidth(defContours))
	}
}

func TestInstanceOpszChangesAdvance(t *testing.T) {
	t.Parallel()

	src := parseGowkVar(t)
	defAdv := src.Advance('A')

	inst := src.Instance([]Variation{{Tag: "opsz", Value: 72}})
	if inst == src {
		t.Fatal("Instance(opsz 72) returned the default face")
	}

	if got := inst.Advance('A'); got <= defAdv {
		t.Fatalf("opsz 72 advance = %g, want > default %g", got, defAdv)
	}
}

func TestInstanceDefaultCoordsReturnsReceiver(t *testing.T) {
	t.Parallel()

	src := parseGowkVar(t)
	if got := src.Instance([]Variation{{Tag: "wght", Value: 400}}); got != src {
		t.Fatal("default wght 400 must keep the receiver")
	}
}

func outlineWidth(contours [][]GlyphPoint) float64 {
	if len(contours) == 0 || len(contours[0]) == 0 {
		return 0
	}

	minX, maxX := contours[0][0].X, contours[0][0].X

	for _, contour := range contours {
		for _, point := range contour {
			if point.X < minX {
				minX = point.X
			}

			if point.X > maxX {
				maxX = point.X
			}
		}
	}

	return maxX - minX
}
