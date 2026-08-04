package layout

import (
	"os"
	"path/filepath"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

func TestCJKFontFamilyFallback(t *testing.T) {
	notoPath := filepath.Join("..", "..", "testdata", "fonts")
	droidPath := "/usr/share/fonts/truetype/droid"
	if _, err := os.Stat(droidPath); err != nil {
		t.Skip("droid fonts not installed")
	}
	if _, err := os.Stat(filepath.Join(notoPath, "NotoSansKR-HangulSubset.ttf")); err != nil {
		t.Skip("testdata Noto subset missing")
	}
	reg := pdf.ScanFontDirs([]string{notoPath, droidPath})
	s := sheet(t, `body { font-family: "Droid Sans Fallback", "Noto Sans KR", sans-serif; font-size: 14pt }`)
	src := `<html><body>
<p>汉字与假名：東京都、上海、深圳。</p>
<p>안녕하세요. 한글 테스트.</p>
</body></html>`
	root, err := html.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{
		Width: testViewport, Height: 800,
		Sheets: []*css.Stylesheet{s}, Background: true, Registry: reg,
	})
	if err != nil {
		t.Fatal(err)
	}
	var sawHan, sawHangul bool
	for _, op := range res.Ops {
		if op.Kind != OpText || op.Font == nil {
			continue
		}
		for _, r := range op.Text {
			switch r {
			case '汉', '圳':
				sawHan = true
				if op.Font.GlyphID(r) == 0 {
					t.Fatalf("rune %c drawn with face lacking glyph (%s)", r, op.Font.PostScriptName)
				}
			case '테', '안':
				sawHangul = true
				if op.Font.GlyphID(r) == 0 {
					t.Fatalf("rune %c drawn with face lacking glyph (%s)", r, op.Font.PostScriptName)
				}
			}
		}
	}
	if !sawHan || !sawHangul {
		t.Fatalf("missing runs sawHan=%v sawHangul=%v", sawHan, sawHangul)
	}
}
