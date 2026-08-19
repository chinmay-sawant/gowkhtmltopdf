//nolint:testpackage,cyclop,wsl // layout resolver integration checks
package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// TestLayoutFontResolverMetricsMatchPaint proves faceFor / faceForRune use the
// Phase 1 contract and the same selected face for measured and painted glyphs.
func TestLayoutFontResolverMetricsMatchPaint(t *testing.T) {
	t.Parallel()

	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	cssSheet := sheet(t, `
body { margin: 0; font-family: Georgia, serif; font-size: 12pt; }
.mono { font-family: MissingMono, monospace; }
`)
	root, err := html.Parse(`<html><body>
<p id="serif">Hello</p>
<p class="mono" id="mono">Code</p>
</body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: testViewport, Height: 400,
		Sheets: []*css.Stylesheet{cssSheet}, Background: true, Faces: faces,
	})
	if err != nil {
		t.Fatal(err)
	}

	var sawSerif, sawMono bool

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText || paintOp.Font == nil {
			continue
		}

		switch {
		case paintOp.Font == faces.Serif || paintOp.Font == faces.SerifBold ||
			paintOp.Font == faces.SerifItalic || paintOp.Font == faces.SerifBoldItalic:
			sawSerif = true
		case paintOp.Font == faces.Mono || paintOp.Font == faces.MonoBold ||
			paintOp.Font == faces.MonoItalic || paintOp.Font == faces.MonoBoldItalic:
			sawMono = true
		}
	}

	if !sawSerif {
		t.Fatal("Georgia, serif text must paint with Liberation Serif (generic), not a legacy alias miss")
	}

	if !sawMono {
		t.Fatal("MissingMono, monospace must continue to Liberation Mono")
	}

	// Direct resolver agreement with layout's selected faces.
	resolver := pdf.NewFontResolver(faces, nil)
	if got := resolver.ResolveFamilyStyle([]string{"Georgia", "serif"}, 400, false); got != faces.Serif {
		t.Fatal("resolver contract drift vs layout paint face")
	}
}

func TestLayoutFontResolverExactRegistryWins(t *testing.T) {
	t.Parallel()

	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	reg := pdf.NewRegistry()
	reg.AddFamilyAlias("Georgia", faces.Mono)

	cssSheet := sheet(t, `body { margin: 0; font-family: Georgia, serif; font-size: 12pt; }`)
	root, err := html.Parse(`<html><body><p>Hello</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: testViewport, Height: 200,
		Sheets: []*css.Stylesheet{cssSheet}, Background: true, Faces: faces, Registry: reg,
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText || paintOp.Font == nil {
			continue
		}

		if paintOp.Font != faces.Mono {
			t.Fatalf("registered Georgia must win over serif generic, got %q", paintOp.Font.PostScriptName)
		}

		return
	}

	t.Fatal("expected text op")
}
