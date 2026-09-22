package convert

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// Tolerances for the canonical output checks. The committed sample and the
// fresh conversion come from the same writer, so these absorb float
// formatting and font-metric noise, not layout drift.
const (
	outputOpsPosTolerance   = 0.5
	outputOpsSizeTolerance  = 0.05
	outputOpsColorTolerance = 0.01
	outputOpsWidthTolerance = 0.05
	outputOpsBoxTolerance   = 0.05
	outputOpsMaxReported    = 10
)

// readCommittedOps opens one committed sample under output/ and parses its
// page operations.
func readCommittedOps(t *testing.T, rel string) *pdf.PageOps {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "..", "output", rel))
	if err != nil {
		t.Fatalf("read committed sample %s: %v", rel, err)
	}

	ops, err := pdf.ParsePageOps(data)
	if err != nil {
		t.Fatalf("parse committed %s: %v", rel, err)
	}

	return ops
}

// freshOps converts a golden HTML through requestForFixture and parses its
// page operations.
func freshOps(t *testing.T, golden string) *pdf.PageOps {
	t.Helper()

	ops, err := pdf.ParsePageOps(runPDF(t, requestForFixture(t, golden)))
	if err != nil {
		t.Fatalf("parse fresh conversion of %s: %v", golden, err)
	}

	return ops
}

// assertPageOpsMatch fails when the fresh conversion draws anything the
// committed sample does not draw in the same order, within tolerance.
func assertPageOpsMatch(t *testing.T, committed, fresh *pdf.PageOps) {
	t.Helper()

	diffs := pageOpsDiff(committed, fresh)
	if len(diffs) == 0 {
		return
	}

	reported := diffs
	if len(reported) > outputOpsMaxReported {
		reported = reported[:outputOpsMaxReported]
	}

	t.Errorf("%d page-op mismatch(es) between committed and fresh; first %d:\n%s",
		len(diffs), len(reported), strings.Join(reported, "\n"))
}

// pageOpsDiff lists every field of the committed record that the fresh
// conversion reproduces outside tolerance.
func pageOpsDiff(committed, fresh *pdf.PageOps) []string {
	diffs := []string{}

	diffs = appendIfDiff(diffs, opsIntDiff("pages", committed.Pages, fresh.Pages))
	diffs = appendIfDiff(diffs, opsIntDiff("mediaboxes", len(committed.MediaBoxes), len(fresh.MediaBoxes)))
	diffs = appendIfDiff(diffs, opsIntDiff("texts", len(committed.Texts), len(fresh.Texts)))
	diffs = appendIfDiff(diffs, opsIntDiff("strokes", len(committed.Strokes), len(fresh.Strokes)))
	diffs = appendIfDiff(diffs, opsIntDiff("fills", len(committed.Fills), len(fresh.Fills)))
	diffs = appendIfDiff(diffs, opsIntDiff("images", len(committed.Images), len(fresh.Images)))

	for index := range min(len(committed.MediaBoxes), len(fresh.MediaBoxes)) {
		for axis := range 4 {
			diffs = appendIfDiff(diffs, opsNumberDiff(
				fmt.Sprintf("mediabox[%d][%d]", index, axis),
				committed.MediaBoxes[index][axis], fresh.MediaBoxes[index][axis], outputOpsBoxTolerance))
		}
	}

	for index := range min(len(committed.Texts), len(fresh.Texts)) {
		want, got := committed.Texts[index], fresh.Texts[index]
		prefix := fmt.Sprintf("text[%d]", index)
		diffs = appendIfDiff(diffs, opsIntDiff(prefix+" page", want.Page, got.Page))
		diffs = appendIfDiff(diffs, opsStringDiff(prefix+" text", want.Text, got.Text))
		diffs = appendIfDiff(diffs, opsNumberDiff(prefix+" x", want.X, got.X, outputOpsPosTolerance))
		diffs = appendIfDiff(diffs, opsNumberDiff(prefix+" y", want.Y, got.Y, outputOpsPosTolerance))
		diffs = appendIfDiff(diffs, opsNumberDiff(prefix+" size", want.Size, got.Size, outputOpsSizeTolerance))
		diffs = appendIfDiff(diffs, opsStringDiff(prefix+" font", want.Font, got.Font))
		diffs = appendIfDiff(diffs, opsColorDiff(prefix+" color", want.Color, got.Color))
	}

	diffs = append(diffs, strokeDiffs(committed.Strokes, fresh.Strokes)...)
	diffs = append(diffs, fillDiffs(committed.Fills, fresh.Fills)...)
	diffs = append(diffs, imageDiffs(committed.Images, fresh.Images)...)

	return diffs
}

func strokeDiffs(committed, fresh []pdf.StrokeSegment) []string {
	diffs := []string{}

	for index := range min(len(committed), len(fresh)) {
		want, got := committed[index], fresh[index]
		prefix := fmt.Sprintf("stroke[%d]", index)
		diffs = appendIfDiff(diffs, opsIntDiff(prefix+" page", want.Page, got.Page))
		diffs = appendIfDiff(diffs, opsNumberDiff(prefix+" x1", want.X1, got.X1, outputOpsPosTolerance))
		diffs = appendIfDiff(diffs, opsNumberDiff(prefix+" y1", want.Y1, got.Y1, outputOpsPosTolerance))
		diffs = appendIfDiff(diffs, opsNumberDiff(prefix+" x2", want.X2, got.X2, outputOpsPosTolerance))
		diffs = appendIfDiff(diffs, opsNumberDiff(prefix+" y2", want.Y2, got.Y2, outputOpsPosTolerance))
		diffs = appendIfDiff(diffs, opsNumberDiff(prefix+" width", want.Width, got.Width, outputOpsWidthTolerance))
		diffs = appendIfDiff(diffs, opsColorDiff(prefix+" color", want.Color, got.Color))
	}

	return diffs
}

func fillDiffs(committed, fresh []pdf.FillRect) []string {
	diffs := []string{}

	for index := range min(len(committed), len(fresh)) {
		want, got := committed[index], fresh[index]
		prefix := fmt.Sprintf("fill[%d]", index)
		diffs = appendIfDiff(diffs, opsIntDiff(prefix+" page", want.Page, got.Page))
		diffs = appendIfDiff(diffs, opsNumberDiff(prefix+" x", want.X, got.X, outputOpsPosTolerance))
		diffs = appendIfDiff(diffs, opsNumberDiff(prefix+" y", want.Y, got.Y, outputOpsPosTolerance))
		diffs = appendIfDiff(diffs, opsNumberDiff(prefix+" w", want.W, got.W, outputOpsPosTolerance))
		diffs = appendIfDiff(diffs, opsNumberDiff(prefix+" h", want.H, got.H, outputOpsPosTolerance))
		diffs = appendIfDiff(diffs, opsColorDiff(prefix+" color", want.Color, got.Color))
	}

	return diffs
}

func imageDiffs(committed, fresh []pdf.ImageBox) []string {
	diffs := []string{}

	for index := range min(len(committed), len(fresh)) {
		want, got := committed[index], fresh[index]
		prefix := fmt.Sprintf("image[%d]", index)
		diffs = appendIfDiff(diffs, opsIntDiff(prefix+" page", want.Page, got.Page))
		diffs = appendIfDiff(diffs, opsNumberDiff(prefix+" x", want.X, got.X, outputOpsPosTolerance))
		diffs = appendIfDiff(diffs, opsNumberDiff(prefix+" y", want.Y, got.Y, outputOpsPosTolerance))
		diffs = appendIfDiff(diffs, opsNumberDiff(prefix+" w", want.W, got.W, outputOpsPosTolerance))
		diffs = appendIfDiff(diffs, opsNumberDiff(prefix+" h", want.H, got.H, outputOpsPosTolerance))
	}

	return diffs
}

func appendIfDiff(diffs []string, diff string) []string {
	if diff == "" {
		return diffs
	}

	return append(diffs, diff)
}

func opsIntDiff(field string, want, got int) string {
	if want == got {
		return ""
	}

	return fmt.Sprintf("%s: committed %d, fresh %d", field, want, got)
}

func opsStringDiff(field, want, got string) string {
	if want == got {
		return ""
	}

	return fmt.Sprintf("%s: committed %q, fresh %q", field, want, got)
}

func opsNumberDiff(field string, want, got, tolerance float64) string {
	if math.Abs(want-got) <= tolerance {
		return ""
	}

	return fmt.Sprintf("%s: committed %.3f, fresh %.3f", field, want, got)
}

func opsColorDiff(field string, want, got [3]float64) string {
	if opsColorClose(want, got) {
		return ""
	}

	return fmt.Sprintf("%s: committed (%.3f, %.3f, %.3f), fresh (%.3f, %.3f, %.3f)",
		field, want[0], want[1], want[2], got[0], got[1], got[2])
}

func opsColorClose(want, got [3]float64) bool {
	return math.Abs(want[0]-got[0]) <= outputOpsColorTolerance &&
		math.Abs(want[1]-got[1]) <= outputOpsColorTolerance &&
		math.Abs(want[2]-got[2]) <= outputOpsColorTolerance
}

func opsBoxClose(want, got [4]float64) bool {
	for index := range 4 {
		if math.Abs(want[index]-got[index]) > outputOpsBoxTolerance {
			return false
		}
	}

	return true
}

// assertOpsTextRun pins one authored text feature measured with
// scripts/inspect_pdf_ops.py. page is 1-based.
//
//nolint:unparam // multi-page fixture tests use non-1 page values
func assertOpsTextRun(
	t *testing.T, ops *pdf.PageOps, page int, text string, xPos, yPos, size float64, font string, color [3]float64,
) {
	t.Helper()

	for _, run := range ops.Texts {
		if run.Page != page-1 || run.Text != text {
			continue
		}

		if math.Abs(run.X-xPos) > outputOpsPosTolerance || math.Abs(run.Y-yPos) > outputOpsPosTolerance {
			t.Errorf("text %q at (%.3f, %.3f), want (%.3f, %.3f)", text, run.X, run.Y, xPos, yPos)
		}

		if math.Abs(run.Size-size) > outputOpsSizeTolerance {
			t.Errorf("text %q size %.3f, want %.3f", text, run.Size, size)
		}

		if run.Font != font {
			t.Errorf("text %q font %q, want %q", text, run.Font, font)
		}

		if !opsColorClose(run.Color, color) {
			t.Errorf("text %q color (%.3f, %.3f, %.3f), want (%.3f, %.3f, %.3f)",
				text, run.Color[0], run.Color[1], run.Color[2], color[0], color[1], color[2])
		}

		return
	}

	t.Fatalf("no %q text run on page %d in the committed sample (%d runs)", text, page, len(ops.Texts))
}

// assertOpsStrokeSegment pins one authored rule. page is 1-based.
func assertOpsStrokeSegment(
	t *testing.T, ops *pdf.PageOps, page int,
	startX, startY, endX, endY float64, color [3]float64, width float64,
) {
	t.Helper()

	for _, segment := range ops.Strokes {
		if segment.Page != page-1 ||
			math.Abs(segment.X1-startX) > outputOpsPosTolerance || math.Abs(segment.Y1-startY) > outputOpsPosTolerance ||
			math.Abs(segment.X2-endX) > outputOpsPosTolerance || math.Abs(segment.Y2-endY) > outputOpsPosTolerance {
			continue
		}

		if math.Abs(segment.Width-width) > outputOpsWidthTolerance {
			t.Errorf("rule width %.3f, want %.3f", segment.Width, width)
		}

		if !opsColorClose(segment.Color, color) {
			t.Errorf("rule color (%.3f, %.3f, %.3f), want (%.3f, %.3f, %.3f)",
				segment.Color[0], segment.Color[1], segment.Color[2], color[0], color[1], color[2])
		}

		return
	}

	t.Fatalf("no rule (%.3f, %.3f) -> (%.3f, %.3f) on page %d in the committed sample (%d strokes)",
		startX, startY, endX, endY, page, len(ops.Strokes))
}

// assertOpsFillRect pins one authored filled box. page is 1-based.
func assertOpsFillRect(
	t *testing.T, ops *pdf.PageOps, page int,
	xPos, yPos, width, height float64, color [3]float64,
) {
	t.Helper()

	for _, fill := range ops.Fills {
		if fill.Page != page-1 ||
			math.Abs(fill.X-xPos) > outputOpsPosTolerance || math.Abs(fill.Y-yPos) > outputOpsPosTolerance ||
			math.Abs(fill.W-width) > outputOpsPosTolerance || math.Abs(fill.H-height) > outputOpsPosTolerance {
			continue
		}

		if !opsColorClose(fill.Color, color) {
			t.Errorf("fill color (%.3f, %.3f, %.3f), want (%.3f, %.3f, %.3f)",
				fill.Color[0], fill.Color[1], fill.Color[2], color[0], color[1], color[2])
		}

		return
	}

	t.Fatalf("no fill box (%.3f, %.3f) %.3fx%.3f on page %d in the committed sample (%d fills)",
		xPos, yPos, width, height, page, len(ops.Fills))
}

// assertOpsImageBox pins one authored image placement. page is 1-based.
//
//nolint:unused // shared by later fixture tests that author images
func assertOpsImageBox(t *testing.T, ops *pdf.PageOps, page int, xPos, yPos, width, height float64) {
	t.Helper()

	for _, box := range ops.Images {
		if box.Page != page-1 {
			continue
		}

		if math.Abs(box.X-xPos) <= outputOpsPosTolerance && math.Abs(box.Y-yPos) <= outputOpsPosTolerance &&
			math.Abs(box.W-width) <= outputOpsPosTolerance && math.Abs(box.H-height) <= outputOpsPosTolerance {
			return
		}
	}

	t.Fatalf("no image box (%.3f, %.3f) %.3fx%.3f on page %d in the committed sample (%d images)",
		xPos, yPos, width, height, page, len(ops.Images))
}
