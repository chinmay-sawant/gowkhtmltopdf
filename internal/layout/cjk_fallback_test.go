//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"os"
	"path/filepath"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

func TestCJKFontFamilyFallback(t *testing.T) { //nolint:cyclop
	t.Parallel()

	notoPath := filepath.Join("..", "..", "testdata", "fonts")

	droidPath := "/usr/share/fonts/truetype/droid"
	if _, err := os.Stat(droidPath); err != nil {
		t.Skip("droid fonts not installed")
	}

	if _, err := os.Stat(filepath.Join(notoPath, "NotoSansKR-HangulSubset.ttf")); err != nil {
		t.Skip("testdata Noto subset missing")
	}

	reg := pdf.ScanFontDirs([]string{notoPath, droidPath})
	cssSheet := sheet(t, `body { font-family: "Droid Sans Fallback", "Noto Sans KR", sans-serif; font-size: 14pt }`)
	src := `<html><body>
<p>汉字与假名：東京都、上海、深圳。</p>
<p>안녕하세요. 한글 테스트.</p>
</body></html>`

	root, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: testViewport, Height: 800,
		Sheets: []*css.Stylesheet{cssSheet}, Background: true, Registry: reg,
	})
	if err != nil {
		t.Fatal(err)
	}

	var sawHan, sawHangul bool

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText || paintOp.Font == nil {
			continue
		}

		for _, runic := range paintOp.Text {
			switch runic {
			case '汉', '圳':
				sawHan = true

				if paintOp.Font.GlyphID(runic) == 0 {
					t.Fatalf("rune %c drawn with face lacking glyph (%s)", runic, paintOp.Font.PostScriptName)
				}
			case '테', '안':
				sawHangul = true

				if paintOp.Font.GlyphID(runic) == 0 {
					t.Fatalf("rune %c drawn with face lacking glyph (%s)", runic, paintOp.Font.PostScriptName)
				}
			}
		}
	}

	if !sawHan || !sawHangul {
		t.Fatalf("missing runs sawHan=%v sawHangul=%v", sawHan, sawHangul)
	}
}
