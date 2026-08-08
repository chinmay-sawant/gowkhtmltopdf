//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

func TestRealBoldFaceOps(t *testing.T) { //nolint:cyclop
	t.Parallel()

	root, err := html.Parse(`<html><body><p>plain <b>bold</b> <i>italic</i></p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 400, Height: 400, Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	var sawBoldFace, sawItalicFace, sawRegular bool

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText || paintOp.Font == nil {
			continue
		}

		switch paintOp.Font.PostScriptName {
		case "LiberationSans-Bold":
			sawBoldFace = true

			if !paintOp.Bold {
				t.Error("bold op should set Bold flag")
			}
		case "LiberationSans-Italic":
			sawItalicFace = true
		case "LiberationSans":
			sawRegular = true
		}
	}

	if !sawBoldFace {
		t.Error("expected LiberationSans-Bold on <b>")
	}

	if !sawItalicFace {
		t.Error("expected LiberationSans-Italic on <i>")
	}

	if !sawRegular {
		t.Error("expected regular face on plain text")
	}
}

func TestCoalesceSameStyleWords(t *testing.T) {
	t.Parallel()

	root, err := html.Parse(`<html><body><p>one two three</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{Width: 500, Height: 200}) //nolint:exhaustruct // intentional zero fields
	if err != nil {
		t.Fatal(err)
	}

	texts := opsOfKind(res, OpText)
	if len(texts) != 1 {
		t.Fatalf("coalesced line should be 1 op, got %d: %+v", len(texts), texts)
	}

	if texts[0].Text != "one two three" {
		t.Errorf("text = %q", texts[0].Text)
	}
}

func TestNthChildZebraSheet(t *testing.T) {
	t.Parallel()

	sheet, err := css.Parse(`tr:nth-child(even) td { background-color: #eee }`)
	if err != nil {
		t.Fatal(err)
	}

	root, err := html.Parse(`<html><body><table>
		<tr><td>a</td></tr>
		<tr><td>b</td></tr>
		<tr><td>c</td></tr>
	</table></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 300, Height: 400,
		Sheets:     []*css.Stylesheet{sheet},
		Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	// even rows get a fill; odd rows may not
	fills := 0

	for _, op := range res.Ops {
		if op.Kind == OpFillRect && op.R > 0.8 && op.G > 0.8 && op.B > 0.8 {
			fills++
		}
	}

	if fills < 1 {
		t.Errorf("expected zebra background fill(s), got %d fills total among ops", fills)
	}
}
