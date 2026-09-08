package layout

import "testing"

func TestBoxModelStyleProjection(t *testing.T) { //nolint:cyclop // each field group is checked explicitly
	t.Parallel()

	style := &ResolvedStyle{
		BoxSizing:         borderBox,
		Width:             100,
		WidthPercent:      25,
		MinWidth:          10,
		MinWidthPercent:   5,
		MaxWidth:          200,
		MaxWidthPercent:   90,
		Height:            80,
		HeightPercent:     40,
		MinHeight:         20,
		MinHeightPercent:  10,
		MaxHeight:         160,
		MaxHeightPercent:  75,
		MarginLeft:        3,
		MarginRight:       4,
		MarginLeftAuto:    true,
		MarginRightAuto:   true,
		PaddingTop:        5,
		PaddingRight:      6,
		PaddingBottom:     7,
		PaddingLeft:       8,
		BorderTop:         border{Width: 1},
		BorderRight:       border{Width: 2},
		BorderBottom:      border{Width: 3},
		BorderLeft:        border{Width: 4},
		BorderImageSource: "frame.png",
	}

	view := boxModelStyleOf(style)
	if view.boxSizing != borderBox || view.width != 100 || view.widthPercent != 25 ||
		view.minWidth != 10 || view.minWidthPercent != 5 || view.maxWidth != 200 ||
		view.maxWidthPercent != 90 {
		t.Fatalf("width projection = %#v", view)
	}

	if view.height != 80 || view.heightPercent != 40 || view.minHeight != 20 ||
		view.minHeightPercent != 10 || view.maxHeight != 160 || view.maxHeightPercent != 75 {
		t.Fatalf("height projection = %#v", view)
	}

	if view.marginLeft != 3 || view.marginRight != 4 || !view.marginLeftAuto || !view.marginRightAuto {
		t.Fatalf("margin projection = %#v", view)
	}

	if view.paddingTop != 5 || view.paddingRight != 6 || view.paddingBottom != 7 || view.paddingLeft != 8 ||
		view.borderTop.Width != 1 || view.borderRight.Width != 2 || view.borderBottom.Width != 3 ||
		view.borderLeft.Width != 4 || view.borderImageSource != "frame.png" {
		t.Fatalf("chrome projection = %#v", view)
	}
}
