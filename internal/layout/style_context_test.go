package layout //nolint:testpackage // cancellation benchmark exercises the unexported style resolver.

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

type cancelAfterStyleChecks struct {
	checks int
	after  int
}

func (c *cancelAfterStyleChecks) Deadline() (time.Time, bool) { return time.Time{}, false }

func (c *cancelAfterStyleChecks) Done() <-chan struct{} { return nil }

func (c *cancelAfterStyleChecks) Err() error {
	c.checks++
	if c.checks > c.after {
		return context.Canceled
	}

	return nil
}

func (c *cancelAfterStyleChecks) Value(any) any { return nil }

func styleStressFixture(tb testing.TB) (*html.Node, *css.Stylesheet) {
	tb.Helper()

	root := &html.Node{ //nolint:exhaustruct // stress tree only needs element fields
		Type: html.ElementNode, Name: "body", Attrs: map[string]string{},
	}
	for range 2048 {
		child := &html.Node{ //nolint:exhaustruct // stress tree only needs element fields
			Type:   html.ElementNode,
			Name:   "div",
			Attrs:  map[string]string{"class": "item"},
			Parent: root,
		}
		root.Children = append(root.Children, child)
	}

	var source strings.Builder
	for i := range 1024 {
		_, _ = source.WriteString(".item[data-rule-")
		_ = source.WriteByte(byte('a' + i%26))
		_, _ = source.WriteString("] { color: #123456; } ")
	}
	// Include a matching selector so the fixture exercises both candidate
	// rejection and declaration application paths.
	_, _ = source.WriteString(".item { color: #abcdef; }")

	sheet, err := css.Parse(source.String())
	if err != nil {
		tb.Fatalf("css.Parse: %v", err)
	}

	return root, sheet
}

func TestResolveStylesContextStopsDuringCascade(t *testing.T) {
	t.Parallel()
	root, sheet := styleStressFixture(t)
	ctx := &cancelAfterStyleChecks{after: 40} //nolint:exhaustruct // test context only needs threshold

	_, err := resolveStylesWithContext(ctx, root, Options{ //nolint:exhaustruct // focused style options
		Sheets: sheetSlice(sheet), Width: testViewport, Height: 800,
	}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("resolveStylesWithContext error = %v, want context.Canceled", err)
	}
}

//nolint:wsl // benchmark timing boundaries intentionally surround the loop.
func BenchmarkStyleResolutionContext(b *testing.B) {
	root, sheet := styleStressFixture(b)
	opts := Options{Sheets: sheetSlice(sheet), Width: testViewport, Height: 800} //nolint:exhaustruct
	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		if _, err := resolveStylesWithContext(b.Context(), root, opts, nil); err != nil {
			b.Fatal(err)
		}
	}
}

func sheetSlice(sheet *css.Stylesheet) []*css.Stylesheet { return []*css.Stylesheet{sheet} }
