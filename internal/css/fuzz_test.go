package css_test

import (
	"testing"

	"gowkhtmltopdf/internal/css"
)

func FuzzParseCSS(f *testing.F) {
	seeds := []string{
		"body { color: #333; font-size: 14px; margin: 0; }",
		"@media print { .no-print { display: none; } }",
		"@container (min-width: 300px) { h2 { font-size: 1.5rem; } }",
		"div:has(> p) { border: 1px solid black; }",
		"@page { size: A4; margin: 10mm; }",
		"p::before { content: 'Note: '; font-weight: bold; }",
		"table, th, td { border-collapse: collapse; padding: 4px; }",
		":root { --main-color: blue; } div { color: var(--main-color); }",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(_ *testing.T, orig string) {
		if len(orig) > 64<<10 {
			return
		}
		// Must not panic on any CSS input.
		_, _ = css.Parse(orig)
	})
}
