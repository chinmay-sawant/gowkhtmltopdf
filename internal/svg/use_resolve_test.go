package svg

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

const featherSearchSymbol = `<svg xmlns="http://www.w3.org/2000/svg">` +
	`<defs>` +
	`<symbol id="search" viewBox="0 0 24 24">` +
	`<circle cx="11" cy="11" r="8"></circle>` +
	`<line x1="21" y1="21" x2="16.65" y2="16.65"></line>` +
	`</symbol>` +
	`<symbol id="other" viewBox="0 0 24 24"><path d="M1 1h1"/></symbol>` +
	`</defs></svg>`

const searchUseDoc = `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" ` +
	`viewBox="0 0 24 24" fill="none" stroke="black" stroke-width="2">` +
	`<use href="sprite.svg#search"/>` +
	`</svg>`

var errSpriteNotFound = errors.New("not found")

func TestResolveUseReferencesExternal(t *testing.T) {
	t.Parallel()

	var fetched []string

	out, err := ResolveUseReferences([]byte(searchUseDoc), func(url string) ([]byte, error) {
		fetched = append(fetched, url)

		if url != "sprite.svg" {
			t.Fatalf("fetch url = %q, want sprite.svg", url)
		}

		return []byte(featherSearchSymbol), nil
	})

	if err != nil {
		t.Fatalf("ResolveUseReferences: %v", err)
	}

	if len(fetched) != 1 {
		t.Fatalf("fetch count = %d, want 1", len(fetched))
	}

	assertResolvedSearchGeometry(t, string(out))

	png, width, height, err := Rasterize(out, 64)
	if err != nil {
		t.Fatalf("Rasterize after resolve: %v", err)
	}

	if len(png) == 0 || width < 1 || height < 1 {
		t.Fatalf("empty raster after resolve: png=%d w=%d h=%d", len(png), width, height)
	}
}

func assertResolvedSearchGeometry(t *testing.T, got string) {
	t.Helper()

	if strings.Contains(got, "<use") {
		t.Fatalf("resolved SVG still contains <use>: %s", got)
	}

	if !strings.Contains(got, `viewBox="0 0 24 24"`) {
		t.Fatalf("missing symbol viewBox in rewrite: %s", got)
	}

	if !strings.Contains(got, `cx="11"`) || !strings.Contains(got, `x1="21"`) {
		t.Fatalf("missing search geometry after resolve: %s", got)
	}
}

func TestResolveUseReferencesXLinkHref(t *testing.T) {
	t.Parallel()

	doc := `<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">` +
		`<use xlink:href="sprite.svg#search"></use></svg>`

	out, err := ResolveUseReferences([]byte(doc), func(string) ([]byte, error) {
		return []byte(featherSearchSymbol), nil
	})
	if err != nil {
		t.Fatalf("ResolveUseReferences: %v", err)
	}

	got := string(out)

	if strings.Contains(strings.ToLower(got), "<use") {
		t.Fatalf("xlink:href use not inlined: %s", got)
	}

	if !strings.Contains(got, `cy="11"`) {
		t.Fatalf("missing geometry: %s", got)
	}
}

func TestResolveUseReferencesSameDocument(t *testing.T) {
	t.Parallel()

	doc := `<svg xmlns="http://www.w3.org/2000/svg" width="10" height="10" viewBox="0 0 10 10">` +
		`<defs><symbol id="box" viewBox="0 0 10 10">` +
		`<path d="M0 0H10V10H0Z"/>` +
		`</symbol></defs>` +
		`<use href="#box"/>` +
		`</svg>`

	out, err := ResolveUseReferences([]byte(doc), nil)
	if err != nil {
		t.Fatalf("ResolveUseReferences: %v", err)
	}

	got := string(out)

	if strings.Contains(got, `<use`) {
		t.Fatalf("same-document use not inlined: %s", got)
	}

	if !strings.Contains(got, `d="M0 0H10V10H0Z"`) {
		t.Fatalf("missing path geometry: %s", got)
	}

	// prepareCanvasInput / Rasterize must also inline same-document refs.
	png, _, _, err := Rasterize([]byte(doc), 64)
	if err != nil {
		t.Fatalf("Rasterize same-document use: %v", err)
	}

	if len(png) == 0 {
		t.Fatal("Rasterize returned empty PNG for same-document use")
	}
}

func TestResolveUseReferencesMissingID(t *testing.T) {
	t.Parallel()

	doc := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24">` +
		`<use href="#missing"/>` +
		`</svg>`)

	out, err := ResolveUseReferences(doc, nil)
	if err != nil {
		t.Fatalf("ResolveUseReferences missing id: %v", err)
	}

	if !bytes.Equal(out, doc) {
		t.Fatalf("missing id should leave markup unchanged:\ngot  %s\nwant %s", out, doc)
	}

	// Rasterize must not panic; empty/near-empty geometry is fine.
	png, width, height, err := Rasterize(doc, 32)
	if err != nil && !errors.Is(err, errCanvasEmptySize) && !errors.Is(err, errCanvasZeroPixel) {
		// Canvas may succeed with an empty-looking image or fail cleanly.
		t.Logf("Rasterize missing use: %v (tolerated); png=%d %dx%d", err, len(png), width, height)
	}
}

func TestResolveUseReferencesMissingSprite(t *testing.T) {
	t.Parallel()

	doc := []byte(searchUseDoc)

	out, err := ResolveUseReferences(doc, func(string) ([]byte, error) {
		return nil, errSpriteNotFound
	})

	if err != nil {
		t.Fatalf("ResolveUseReferences: %v", err)
	}

	if !bytes.Equal(out, doc) {
		t.Fatalf("fetch failure should leave <use> in place, got %s", out)
	}
}

func TestResolveUseReferencesNilFetchLeavesExternal(t *testing.T) {
	t.Parallel()

	doc := []byte(searchUseDoc)

	out, err := ResolveUseReferences(doc, nil)

	if err != nil {
		t.Fatalf("ResolveUseReferences: %v", err)
	}

	if !bytes.Equal(out, doc) {
		t.Fatalf("nil fetch should leave external use unchanged")
	}
}
