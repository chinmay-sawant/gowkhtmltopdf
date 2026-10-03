package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// textOpWith returns the first OpText whose Text equals want.
func textOpWith(res *Result, want string) *Op {
	for idx := range res.Ops {
		if res.Ops[idx].Kind == OpText && res.Ops[idx].Text == want {
			return &res.Ops[idx]
		}
	}

	return nil
}

// firstTextOp returns the first OpText in the display list.
func firstTextOp(res *Result) *Op {
	for idx := range res.Ops {
		if res.Ops[idx].Kind == OpText {
			return &res.Ops[idx]
		}
	}

	return nil
}

func TestTextInputPaintsValueInsideBorderBox(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t,
		`<html><body><input type="text" value="hello"></body></html>`,
		sheet(t, `input { width: 120px; height: 20px; }`),
	)
	inputBox := findBox(t, res, htmlInput)

	op := textOpWith(res, "hello")
	if op == nil {
		t.Fatal("text input value not painted")
	}

	if !near(op.Size, formControlFontSizePt) {
		t.Fatalf("value size %.2fpt, want %.2fpt", op.Size, float64(formControlFontSizePt))
	}

	if op.X < inputBox.x-0.01 || op.X+op.W > inputBox.x+inputBox.w+0.01 {
		t.Fatalf("value spans %.2f..%.2f outside box %.2f..%.2f",
			op.X, op.X+op.W, inputBox.x, inputBox.x+inputBox.w)
	}

	centerY := op.Y - op.H/two + op.InkDescent
	if !near(centerY, inputBox.y+inputBox.height/two) {
		t.Fatalf("value center y %.2f, want %.2f", centerY, inputBox.y+inputBox.height/two)
	}
}

func TestTextInputPaintsValueWithAutoHeight(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t,
		`<html><body><input type="text" value="hi"></body></html>`,
		sheet(t, `input { width: 100px; }`),
	)

	if textOpWith(res, "hi") == nil {
		t.Fatal("auto-height input value not painted")
	}
}

func TestTextInputPaintsPlaceholderWhenEmpty(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t,
		`<html><body><input type="text" placeholder="name"></body></html>`,
		sheet(t, `input { width: 120px; height: 20px; }`),
	)

	if textOpWith(res, "name") == nil {
		t.Fatal("placeholder not painted for an empty value")
	}
}

func TestPasswordInputMasksValueWithBullets(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t,
		`<html><body><input type="password" value="abc"></body></html>`,
		sheet(t, `input { width: 120px; height: 20px; }`),
	)

	want := strings.Repeat(string(controlBulletRune), 3)

	op := textOpWith(res, want)
	if op == nil {
		t.Fatalf("password value not masked with %d bullets", 3)
	}

	if strings.Contains(op.Text, "abc") {
		t.Fatal("password value leaked into the display list")
	}
}

func TestTextInputTruncatesValueToContentBox(t *testing.T) {
	t.Parallel()

	const value = "abcdefghijklmnopqrstuvwxyz"

	res := layoutHTML(t,
		`<html><body><input type="text" value="`+value+`"></body></html>`,
		sheet(t, `input { width: 40px; height: 20px; }`),
	)
	inputBox := findBox(t, res, htmlInput)

	op := textOpWith(res, value)
	if op == nil {
		op = firstTextOp(res)
	}

	if op == nil {
		t.Fatal("truncated input value not painted")
	}

	if op.Text == value {
		t.Fatal("long value was not truncated")
	}

	if !strings.HasPrefix(value, op.Text) {
		t.Fatalf("truncated value %q is not a prefix of %q", op.Text, value)
	}

	if op.X+op.W > inputBox.x+inputBox.w+0.01 {
		t.Fatalf("truncated value ends at %.2f past box right edge %.2f",
			op.X+op.W, inputBox.x+inputBox.w)
	}
}

func TestNonTextInputsKeepOwnFaces(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t,
		`<html><body><input type="checkbox"><input type="radio"><input type="range" value="5"></body></html>`,
		nil,
	)

	for _, op := range res.Ops {
		if op.Kind == OpText {
			t.Fatalf("non-text input painted text %q", op.Text)
		}
	}
}

func TestIsTextInputTypes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		typ  string
		want bool
	}{
		{"", true}, {"text", true}, {"PASSWORD", true}, {"search", true},
		{"email", true}, {"tel", true}, {"url", true}, {"number", true},
		{"checkbox", false}, {"radio", false}, {"range", false}, {"button", false},
		{"submit", false}, {"reset", false}, {"file", false}, {"hidden", false},
	}

	for _, tc := range cases {
		node := &html.Node{Type: html.ElementNode, Name: htmlInput} //nolint:exhaustruct // only name and type matter
		if tc.typ != "" {
			node.Attrs = map[string]string{"type": tc.typ}
		}

		if got := isTextInput(node); got != tc.want {
			t.Errorf("isTextInput(type=%q) = %v, want %v", tc.typ, got, tc.want)
		}
	}
}

func TestButtonUADefaultFace(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t,
		`<html><body><button>Go</button></body></html>`,
		sheet(t, `button { width: 100px; }`),
	)
	buttonBox := findBox(t, res, "button")
	sty := buttonBox.style

	if sty.Display != cssDisplayInlineBlock {
		t.Fatalf("button display = %q, want inline-block", sty.Display)
	}

	if !near(sty.PaddingTop, pxToPt(1)) || !near(sty.PaddingLeft, pxToPt(6)) {
		t.Fatalf("button padding = T %.2f L %.2f, want 1px/6px", sty.PaddingTop, sty.PaddingLeft)
	}

	if !near(sty.BorderTop.Width, 1) {
		t.Fatalf("button border width = %.2f, want 1px", sty.BorderTop.Width)
	}

	if !near(sty.BGColor[0], 0xef/255.0) || sty.BGColor[3] == 0 {
		t.Fatalf("button background = %v, want opaque #efefef", sty.BGColor)
	}

	if sty.TextAlign != "center" {
		t.Fatalf("button text-align = %q, want center", sty.TextAlign)
	}

	op := textOpWith(res, "Go")
	if op == nil {
		t.Fatal("button label not painted")
	}

	centerX := op.X + op.W/two
	if !near(centerX, buttonBox.x+buttonBox.w/two) {
		t.Fatalf("button label center x %.2f, want %.2f", centerX, buttonBox.x+buttonBox.w/two)
	}
}

func TestAuthorOverridesButtonUADefaults(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t,
		`<html><body><button>Go</button></body></html>`,
		sheet(t, `button { display: block; padding: 0; border: none; background-color: #123456; text-align: right; }`),
	)
	sty := findBox(t, res, "button").style

	if sty.Display != displayBlock {
		t.Fatalf("author display = %q, want block", sty.Display)
	}

	if !near(sty.PaddingTop, 0) || !near(sty.PaddingLeft, 0) {
		t.Fatalf("author padding lost: T %.2f L %.2f", sty.PaddingTop, sty.PaddingLeft)
	}

	if !near(sty.BorderTop.Width, 0) {
		t.Fatalf("author border:none lost: width %.2f", sty.BorderTop.Width)
	}

	if !near(sty.BGColor[0], 0x12/255.0) {
		t.Fatalf("author background lost: %v", sty.BGColor)
	}

	if sty.TextAlign != "right" {
		t.Fatalf("author text-align lost: %q", sty.TextAlign)
	}
}

func TestAuthorBackgroundShorthandOverridesButtonUA(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t,
		`<html><body><button>Go</button></body></html>`,
		sheet(t, `button { background: #123456; }`),
	)
	sty := findBox(t, res, "button").style

	if !near(sty.BGColor[0], 0x12/255.0) || !near(sty.BGColor[2], 0x56/255.0) {
		t.Fatalf("author background shorthand lost: %v", sty.BGColor)
	}
}

func TestSelectAndTextareaUAFaces(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t,
		`<html><body><select><option>One</option></select><textarea>body</textarea></body></html>`,
		nil,
	)

	selectBox := findBox(t, res, "select")
	if selectBox.style.Display != cssDisplayInlineBlock {
		t.Fatalf("select display = %q, want inline-block", selectBox.style.Display)
	}

	if !near(selectBox.style.PaddingTop, pxToPt(1)) || !near(selectBox.style.PaddingLeft, pxToPt(2)) {
		t.Fatalf("select padding = T %.2f L %.2f, want 1px/2px",
			selectBox.style.PaddingTop, selectBox.style.PaddingLeft)
	}

	if !near(selectBox.style.BorderTop.Width, 1) || !near(selectBox.style.BGColor[0], 0xef/255.0) {
		t.Fatalf("select face = border %.2f bg %v, want 1px #efefef",
			selectBox.style.BorderTop.Width, selectBox.style.BGColor)
	}

	textareaBox := findBox(t, res, "textarea")
	if textareaBox.style.Display != cssDisplayInlineBlock {
		t.Fatalf("textarea display = %q, want inline-block", textareaBox.style.Display)
	}

	if !near(textareaBox.style.PaddingTop, pxToPt(2)) || !near(textareaBox.style.PaddingLeft, pxToPt(2)) {
		t.Fatalf("textarea padding = T %.2f L %.2f, want 2px/2px",
			textareaBox.style.PaddingTop, textareaBox.style.PaddingLeft)
	}

	if !near(textareaBox.style.BorderTop.Width, 1) || !near(textareaBox.style.BGColor[0], 1) {
		t.Fatalf("textarea face = border %.2f bg %v, want 1px #ffffff",
			textareaBox.style.BorderTop.Width, textareaBox.style.BGColor)
	}
}
