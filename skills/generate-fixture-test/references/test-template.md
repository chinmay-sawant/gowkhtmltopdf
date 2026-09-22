# Fixture test template

Use this reference after measuring the exact PDF. Replace every placeholder
with values from that fixture's inspector output. Do not reuse a coordinate or
page number from another test.

```go
package convert

import "testing"

func TestOutputFixtureNNSlug(t *testing.T) {
	t.Parallel()

	committed := readCommittedOps(t, "fixture-NN-slug.pdf")
	fresh := freshOps(t, "fixture-NN-slug.html")

	assertPageOpsMatch(t, committed, fresh)

	// Values below came from scripts/inspect_pdf_ops.py for this PDF.
	assertOpsTextRun(
		t, committed, 1, "Unique title", measuredX, measuredY, measuredSize,
		"MeasuredBaseFont", [3]float64{red / 255, green / 255, blue / 255},
	)
	assertOpsStrokeSegment(
		t, committed, 1,
		measuredX1, measuredY1, measuredX2, measuredY2,
		[3]float64{red / 255, green / 255, blue / 255}, measuredWidth,
	)

	if committed.Pages != measuredPages {
		t.Errorf("pages = %d, want %d", committed.Pages, measuredPages)
	}

	if len(committed.MediaBoxes) != measuredPages ||
		!opsBoxClose(committed.MediaBoxes[0], [4]float64{minX, minY, maxX, maxY}) {
		t.Errorf("mediaboxes = %v, want the measured boxes", committed.MediaBoxes)
	}

	if len(committed.Texts) != measuredTextRuns {
		t.Errorf("text runs = %d, want %d", len(committed.Texts), measuredTextRuns)
	}

	if len(committed.Strokes) != measuredStrokes {
		t.Errorf("strokes = %d, want %d", len(committed.Strokes), measuredStrokes)
	}

	if len(committed.Fills) != measuredFills {
		t.Errorf("fills = %d, want %d", len(committed.Fills), measuredFills)
	}

	if len(committed.Images) != measuredImages {
		t.Errorf("images = %d, want %d", len(committed.Images), measuredImages)
	}
}
```

## Choosing assertions

Use only assertions that match the HTML and the measured PDF.

- Text: choose unique strings. Include page, origin, size, BaseFont, and fill
  color.
- Rule or border: choose a measured stroked segment. Include endpoints, width,
  and stroke color.
- Fill: choose a measured filled rectangle. Include page box and fill color.
- Image: choose the measured placement box. Confirm the HTML really authors
  the image.
- Page: pin the measured page count and MediaBox.
- Counts: pin them when the PDF is simple enough that an unexpected operation
  should be a failure. Counts are supporting evidence, not a replacement for
  authored anchors.

For a multi-page PDF, repeat text or drawing assertions with the measured page
number. Put at least one assertion on the last page when the fixture has a
stable last-page feature.

## Measurement record

Before writing the test, keep a short working record in the response or local
notes allowed by the user's scope:

```text
PDF: output/fixture-NN-slug.pdf
HTML: testdata/golden/fixture-NN-slug.html
Pages: N
MediaBoxes: ...
Text runs: N
Strokes: N
Fills: N
Images: N
Anchors:
  page 1 | text | "..." | x=... y=... size=... font=... color=#......
  page N | rule | x1=... y1=... x2=... y2=... width=... color=#......
```

The working record is evidence for the test values. It is not a substitute for
the test or for the independent PDF inspection.
