package css //nolint:testpackage // parser-internal helpers are part of this unit's contract

import "testing"

func TestParsePageMarginBoxes(t *testing.T) {
	t.Parallel()

	src := `@page {
		margin: 2cm;
		@top-left { content: "TL" }
		@top-center { content: "Header" }
		@top-right { content: "TR" }
		@bottom-left { content: "BL" }
		@bottom-center { content: 'BC' }
		@bottom-right { content: "BR" }
		size: A4;
	}`

	str := mustSheet(t, src)
	if len(str.Pages) != 1 {
		t.Fatalf("Pages = %+v, want 1", str.Pages)
	}

	got := str.Pages[0]
	if got.Margin != "2cm" || got.Size != "A4" {
		t.Fatalf("descriptors = %+v, want margin 2cm size A4", got)
	}

	want := PageMarginBoxes{
		TopLeft:      "TL",
		TopCenter:    "Header",
		TopRight:     "TR",
		BottomLeft:   "BL",
		BottomCenter: "BC",
		BottomRight:  "BR",
	}
	if got.Boxes != want {
		t.Errorf("Boxes = %+v, want %+v", got.Boxes, want)
	}
}

func TestParsePageMarginBoxesIgnoreFunctions(t *testing.T) {
	t.Parallel()

	str := mustSheet(t, `@page { @top-center { content: counter(page) } @top-left { content: "ok" } }`)
	if len(str.Pages) != 1 {
		t.Fatalf("Pages = %+v, want 1", str.Pages)
	}

	if str.Pages[0].Boxes.TopCenter != "" {
		t.Errorf("function content stored %q, want drop", str.Pages[0].Boxes.TopCenter)
	}

	if str.Pages[0].Boxes.TopLeft != "ok" {
		t.Errorf("TopLeft = %q, want ok", str.Pages[0].Boxes.TopLeft)
	}
}

func TestParsePageMarginBoxesNamedPage(t *testing.T) {
	t.Parallel()

	str := mustSheet(t, `@page chapter { margin: 1cm; @top-center { content: "Ch" } }`)
	if len(str.Pages) != 1 || str.Pages[0].Sel != "chapter" {
		t.Fatalf("Pages = %+v", str.Pages)
	}

	if str.Pages[0].Boxes != (PageMarginBoxes{}) { //nolint:exhaustruct // compare against the empty value
		t.Errorf("named page Boxes = %+v, want empty because only unnamed chrome is consumed", str.Pages[0].Boxes)
	}
}
