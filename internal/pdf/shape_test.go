package pdf

import (
	"strings"
	"testing"
	"unicode"
)

func TestShapeTextRTLReverse(t *testing.T) {
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

func TestArabicJoiningBehProducesConnectedForms(t *testing.T) {
	got := shapeArabicJoining("ب")
	if got == "ب" {
		t.Fatalf("expected presentation form, got %q U+%04X", got, []rune(got)[0])
	}
	got = shapeArabicJoining("بب")
	rs := []rune(got)
	if len(rs) != 2 {
		t.Fatalf("len=%d %q", len(rs), got)
	}
	if rs[0] != 0xFE91 {
		t.Errorf("first form U+%04X want FE91 (initial)", rs[0])
	}
	if rs[1] != 0xFE90 {
		t.Errorf("second form U+%04X want FE90 (final)", rs[1])
	}
}

func TestArabicLamAlefLigature(t *testing.T) {
	got := shapeArabicJoining("لا")
	rs := []rune(got)
	if len(rs) != 1 {
		t.Fatalf("lam-alef should ligate to 1 rune, got %d %q", len(rs), got)
	}
	if rs[0] < 0xFEF5 || rs[0] > 0xFEFC {
		t.Fatalf("got U+%04X, want Lam-Alef presentation", rs[0])
	}
}

func TestShapeTextArabicPipeline(t *testing.T) {
	s := ShapeText("اب")
	if s == "" {
		t.Fatal("empty")
	}
	for _, r := range s {
		if !(unicode.Is(unicode.Arabic, r) || (r >= 0xFB50 && r <= 0xFEFF)) {
			t.Fatalf("unexpected rune U+%04X in %q", r, s)
		}
	}
}

func TestIndicCombiningNotDroppedMidWord(t *testing.T) {
	s := ShapeText("क्")
	if !strings.Contains(s, "क") {
		t.Fatalf("lost base ka: %q", s)
	}
}
