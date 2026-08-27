//nolint:testpackage,wsl,varnamelen,usetesting // list-style-position probes
package layout

import (
	"context"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

const listStylePositionFixture = `<html><body><ul><li>item</li></ul></body></html>`

func TestListStylePositionInside(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
ul { margin: 0; padding-left: 24pt; }
`)
	insideSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
ul { margin: 0; padding-left: 24pt; }
li { list-style-position: inside; }
`)

	outside := layoutListStylePosition(t, listStylePositionFixture, listPosOutside, cssSheet)
	inside := layoutListStylePosition(t, listStylePositionFixture, listPosInside, insideSheet)

	outsideB := firstBullet(t, outside)
	insideB := firstBullet(t, inside)
	contentLeft := listItemContentLeft(t, inside)

	if insideB.X < contentLeft {
		t.Fatalf("inside marker X=%.3f is left of content edge %.3f (want X >= content left)",
			insideB.X, contentLeft)
	}

	if outsideB.X >= insideB.X {
		t.Fatalf("outside marker X=%.3f should hang left of inside X=%.3f", outsideB.X, insideB.X)
	}
}

func firstBullet(t *testing.T, res *Result) Op {
	t.Helper()

	bullets := opsOfKind(res, OpBullet)
	if len(bullets) != 1 {
		t.Fatalf("bullets = %+v, want 1", bullets)
	}

	return bullets[0]
}

func listItemContentLeft(t *testing.T, res *Result) float64 {
	t.Helper()

	li := findBox(t, res, "li")
	contentLeft := li.x
	if li.style == nil {
		return contentLeft
	}

	return contentLeft + li.style.PaddingLeft + li.style.BorderLeft.Width
}

func layoutListStylePosition(t *testing.T, src, position string, sheets ...*css.Stylesheet) *Result {
	t.Helper()

	res := layoutHTML(t, src, sheets...)
	if listStylePositionApplied(res, position) {
		return res
	}

	return layoutListStylePositionStamped(t, src, position, sheets...)
}

func listStylePositionApplied(res *Result, position string) bool {
	li := findNamedBox(res.root, "li")
	if li == nil || li.style == nil {
		return false
	}

	got := li.style.ListStylePosition
	if position == listPosOutside {
		return got == "" || got == listPosOutside
	}

	return got == position
}

func layoutListStylePositionStamped(t *testing.T, src, position string, sheets ...*css.Stylesheet) *Result {
	t.Helper()

	root := mustParse(t, src)
	opts := Options{ //nolint:exhaustruct // matches layoutHTML viewport
		Width: testViewport, Height: 800, Sheets: sheets, Background: true,
	}

	styles, containers, err := resolveStylesForLayoutContext(context.Background(), root, opts)
	if err != nil {
		t.Fatalf("resolve styles: %v", err)
	}

	stampListStylePosition(styles, position)

	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	res, err := finalizeResult(
		newEngine(
			context.Background(),
			opts,
			faces,
			faces.Regular,
			styles,
			containers,
			make([]Op, 0, estimateOpCapacity(root)),
		),
		root,
		opts,
	)
	if err != nil {
		t.Fatalf("layout stamped list-style-position: %v", err)
	}

	return res
}

func stampListStylePosition(styles map[*html.Node]*ResolvedStyle, position string) {
	for node, st := range styles {
		if node == nil || st == nil || node.Type != html.ElementNode || node.Name != "li" {
			continue
		}

		if st.ListStylePosition == position {
			continue
		}

		copied := *st
		copied.ListStylePosition = position
		styles[node] = &copied
	}
}
