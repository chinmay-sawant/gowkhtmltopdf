package layout

import "testing"

func TestPaintChromeStyleViewCopiesOnlyChromeState(t *testing.T) {
	t.Parallel()

	boxNode := &box{
		style: &ResolvedStyle{
			PaddingBottom: 12,
			Float:         floatLeft,
			Position:      positionFixed,
			BorderLeft: border{
				Width: 2,
				Style: borderStyleDashed,
			},
			BorderRight: border{Width: 3},
		},
	}

	view, ok := paintChromeStyleOf(boxNode)
	if !ok {
		t.Fatal("paintChromeStyleOf returned no view")
	}

	if view.paddingBottom != 12 || view.float != floatLeft || view.position != positionFixed {
		t.Fatalf("view chrome fields = %#v", view)
	}

	if view.borderLeft.Width != 2 || view.borderLeft.Style != borderStyleDashed || view.borderRight.Width != 3 {
		t.Fatalf("view borders = %#v", view)
	}

	boxNode.style.PaddingBottom = 24

	if view.paddingBottom != 12 {
		t.Errorf("view changed after source mutation: paddingBottom = %v", view.paddingBottom)
	}
}
