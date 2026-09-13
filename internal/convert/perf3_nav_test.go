package convert

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
)

func TestPerf3NavWithID(t *testing.T) {
	t.Parallel()

	body := `<html><body><p id="target">here</p><p><a href="#target">jump</a></p></body></html>`
	cmd, _ := newCommand(t, body, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.UseCompression = false

	data := runPDF(t, cmd)
	if !bytes.Contains(data, []byte("/Dest [")) {
		t.Fatal("internal link destination missing")
	}

	if !bytes.Contains(data, []byte("/URI")) && !bytes.Contains(data, []byte("/Dest [")) {
		t.Fatal("expected a fragment link annotation")
	}
}

func TestPerf3NavWithoutIDsConverts(t *testing.T) {
	t.Parallel()

	cmd, _ := newCommand(t, `<html><body><p>no ids no links</p></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	data := runPDF(t, cmd)

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("output is not a PDF")
	}

	if n := pageCount(data); n != 1 {
		t.Fatalf("pages = %d, want 1", n)
	}
}

func TestCollectBodyNavigationHonorsHasIDs(t *testing.T) {
	t.Parallel()

	target := &html.Node{Attrs: map[string]string{"id": "target"}}
	nav := collectBodyNavigation(&layout.Result{
		MaxContentX: 100,
		HasIDs:      true,
		Locations: []layout.ElementLocation{
			{Node: target, Page: 1, X: 2, Y: 3, W: 4, H: 5},
		},
	})

	loc, ok := nav.ids["target"]
	if !ok {
		t.Fatal("target id was not collected")
	}

	if loc.Node != nil {
		t.Fatal("projected destination retained its DOM node")
	}

	if loc.Page != 1 || loc.X != 2 || loc.Y != 3 {
		t.Fatalf("destination = %#v", loc)
	}
}

func TestCollectBodyNavigationEmptySkipsMaps(t *testing.T) {
	t.Parallel()

	nav := collectBodyNavigation(&layout.Result{
		Width:       100,
		MaxContentX: 100,
		Ops:         make([]layout.Op, 64),
		Locations:   make([]layout.ElementLocation, 64),
	})
	if nav.ids != nil || nav.idElems != nil || nav.links != nil {
		t.Fatalf("empty census allocated nav maps: %#v", nav)
	}
}

//nolint:paralleltest // testing.AllocsPerRun panics during parallel tests.
func TestCollectBodyNavigationEmptyDoesNotAllocateMaps(t *testing.T) {
	res := &layout.Result{
		Width:       100,
		MaxContentX: 100,
		Ops:         make([]layout.Op, 512),
		Locations:   make([]layout.ElementLocation, 512),
	}

	allocs := testing.AllocsPerRun(50, func() {
		nav := collectBodyNavigation(res)
		if nav.ids != nil || nav.idElems != nil {
			t.Fatal("empty census allocated maps")
		}
	})
	if allocs != 0 {
		t.Fatalf("allocs = %v, want 0", allocs)
	}
}
