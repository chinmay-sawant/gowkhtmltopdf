package convert //nolint:testpackage // white-box tests need unexported access

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// TestPipelineFinalizeStopsWhenCanceled proves Finalize honors cancellation
// before touching the document writer (CNV-04).
func TestPipelineFinalizeStopsWhenCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	p := &pdfPipeline{run: &runContext{}} //nolint:exhaustruct // cancelled ctx short-circuits before any state use

	err := p.Finalize(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Finalize error = %v, want context.Canceled", err)
	}
}

// TestDrawHeadersFootersStopsWhenCanceled proves the per-page HF pass stops on
// cancellation and reports it through the result failure path (CNV-05).
func TestDrawHeadersFootersStopsWhenCanceled(t *testing.T) {
	t.Parallel()

	doc := pdf.NewDocument()
	doc.AddPage(595, 842)

	plan, err := newPagePlan(nil, nil, 1, false)
	if err != nil {
		t.Fatalf("newPagePlan: %v", err)
	}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	result := drawHeadersFootersResult(ctx, nil, doc, &Request{}, plan, nil, io.Discard) //nolint:exhaustruct // zero request; cancelled ctx short-circuits
	if err := result.Err(); !errors.Is(err, context.Canceled) {
		t.Fatalf("Err = %v, want context.Canceled", err)
	}
}

// TestRunHonorsContextDeadline proves the convert boundary propagates an
// expired deadline instead of running unbounded (CNV-08). The overall
// conversion timeout is caller-owned; see Run's doc comment.
func TestRunHonorsContextDeadline(t *testing.T) {
	t.Parallel()

	cmd, _ := newCommand(t, `<html><body><p>deadline</p></body></html>`, "")
	cmd.Output = io.Discard

	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer cancel()

	err := Run(ctx, cmd, io.Discard, nil)
	if err == nil {
		t.Fatal("expected deadline error from expired context, got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want context.DeadlineExceeded", err)
	}
}
