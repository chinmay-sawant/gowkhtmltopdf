package pdf

import "testing"

func TestShapeTextRTLReverse(t *testing.T) {
	// Hebrew letters אבג (U+05D0 U+05D1 U+05D2) should reverse.
	in := "ab\u05d0\u05d1\u05d2cd"
	got := ShapeText(in)
	want := "ab\u05d2\u05d1\u05d0cd"
	if got != want {
		t.Fatalf("ShapeText = %q, want %q", got, want)
	}
	if ShapeText("hello") != "hello" {
		t.Fatal("LTR unchanged")
	}
}
