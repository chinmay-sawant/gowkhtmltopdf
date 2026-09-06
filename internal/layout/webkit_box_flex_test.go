//nolint:all // webkit-box flex mapping probes
package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func TestWebkitBoxDisplayMapsToFlex(t *testing.T) {
	t.Parallel()
	st := styleForDecl(t, "display: -webkit-box")
	if st.Display != displayFlex || !st.IsWebkitBox {
		t.Fatalf("display=%q IsWebkitBox=%v, want flex+true", st.Display, st.IsWebkitBox)
	}
	st = styleForDecl(t, "display: -webkit-inline-box")
	if st.Display != displayInlineFlex || !st.IsWebkitBox {
		t.Fatalf("inline-box display=%q IsWebkitBox=%v, want inline-flex+true", st.Display, st.IsWebkitBox)
	}
}

func TestWebkitBoxOrientVerticalStacks(t *testing.T) {
	t.Parallel()
	sheet, err := css.Parse(`
.box { display:-webkit-box; -webkit-box-orient:vertical; width:48px; border:1px solid #000; padding:4px; }
.chip { background:#cde; padding:2px 6px; }
`)
	if err != nil {
		t.Fatal(err)
	}
	root, err := html.Parse(`<html><body><div class="box"><span class="chip">A</span><span class="chip">B</span></div></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{ //nolint:exhaustruct
		Width: 200, Height: 200, Background: true, Sheets: []*css.Stylesheet{sheet},
	})
	if err != nil {
		t.Fatal(err)
	}
	var ay, by float64
	var gotA, gotB bool
	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}
		switch op.Text {
		case "A":
			ay, gotA = op.Y, true
		case "B":
			by, gotB = op.Y, true
		}
	}
	if !gotA || !gotB {
		t.Fatalf("missing text ops A/B")
	}
	if by <= ay {
		t.Fatalf("vertical orient: B Y=%.1f should be below A Y=%.1f", by, ay)
	}
}

func TestWebkitBoxFlexGrows(t *testing.T) {
	t.Parallel()
	sheet, err := css.Parse(`
.box { display:-webkit-box; -webkit-box-orient:horizontal; width:140px; border:1px solid #000; padding:4px; }
.grow { -webkit-box-flex:1; background:#cde; padding:4px 6px; }
.fix { background:#fd8; padding:4px 6px; }
`)
	if err != nil {
		t.Fatal(err)
	}
	root, err := html.Parse(`<html><body><div class="box"><span class="grow">flex</span><span class="fix">fix</span></div></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{ //nolint:exhaustruct
		Width: 200, Height: 200, Background: true, Sheets: []*css.Stylesheet{sheet},
	})
	if err != nil {
		t.Fatal(err)
	}
	var fx, fix float64
	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}
		switch op.Text {
		case "flex":
			fx = op.X
		case "fix":
			fix = op.X
		}
	}
	if fix-fx < 40 {
		t.Fatalf("box-flex grow: flex X=%.1f fix X=%.1f, expected grow to push fix right", fx, fix)
	}
}
