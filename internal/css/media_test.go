package css

import "testing"

func TestMediaMatchesTypes(t *testing.T) {
	t.Parallel()

	const width, height = 538.0, 785.0 // ~A4 content

	cases := []struct {
		q, media string
		want     bool
	}{
		{"", "print", true},
		{"all", "print", true},
		{"print", "print", true},
		{"screen", "print", false},
		{"screen", "screen", true},
		{"only print", "print", true},
		{"not print", "print", false},
		{"not screen", "print", true},
		{"print, screen", "print", true},
		{"screen, print", "screen", true},
	}
	for _, c := range cases {
		if got := MediaMatches(c.q, c.media, width, height); got != c.want {
			t.Errorf("MediaMatches(%q,%q)=%v want %v", c.q, c.media, got, c.want)
		}
	}
}

func TestMediaMatchesSizeFeatures(t *testing.T) {
	t.Parallel()
	// 640px = 480pt; 500px = 375pt
	wide := 538.0
	narrow := 300.0
	height := 785.0

	if !MediaMatches("(min-width: 500px)", "print", wide, height) {
		t.Error("wide viewport should match min-width:500px")
	}

	if MediaMatches("(min-width: 500px)", "print", narrow, height) {
		t.Error("narrow viewport should not match min-width:500px")
	}

	if !MediaMatches("(min-width:640px)", "print", wide, height) {
		t.Error("538pt should match min-width:640px (480pt)")
	}

	if MediaMatches("screen and (min-width: 500px)", "print", wide, height) {
		t.Error("screen feature query must not match print media")
	}

	if !MediaMatches("print and (min-width: 400px)", "print", wide, height) {
		t.Error("print + matching min-width should apply")
	}

	if MediaMatches("print and (min-width: 800px)", "print", wide, height) {
		t.Error("print + failing min-width should not apply")
	}

	if MediaMatches("(prefers-color-scheme: dark)", "print", wide, height) {
		t.Error("unknown features must evaluate false")
	}

	if !MediaMatches("(orientation: portrait)", "print", wide, height) {
		t.Error("portrait A4 should match")
	}

	if MediaMatches("(orientation: landscape)", "print", wide, height) {
		t.Error("portrait A4 should not match landscape")
	}
}

func TestMediaMatchesEmptyMediaType(t *testing.T) {
	t.Parallel()
	// Legacy Options.Media == "": apply everything.
	if !MediaMatches("screen", "", 100, 100) {
		t.Error("empty mediaType should match all queries")
	}
}
