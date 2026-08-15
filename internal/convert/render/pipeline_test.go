package render_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert/render"
)

type pipeline struct {
	steps       []string
	assembleErr error
}

func (p *pipeline) RenderObjects(context.Context) error {
	p.steps = append(p.steps, "objects")

	return nil
}

func (p *pipeline) Assemble(context.Context) error {
	p.steps = append(p.steps, "assemble")

	return p.assembleErr
}

func (p *pipeline) Finalize(context.Context) error {
	p.steps = append(p.steps, "finalize")

	return nil
}

func TestRunOrdersStages(t *testing.T) {
	t.Parallel()

	var testPipeline pipeline

	if err := render.Run(t.Context(), &testPipeline); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if want := []string{"objects", "assemble", "finalize"}; !reflect.DeepEqual(testPipeline.steps, want) {
		t.Fatalf("stage order = %v, want %v", testPipeline.steps, want)
	}
}

func TestRunStopsAfterStageError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("assemble failed") //nolint:err113 // local test sentinel

	var testPipeline pipeline
	testPipeline.assembleErr = wantErr

	if err := render.Run(t.Context(), &testPipeline); !errors.Is(err, wantErr) {
		t.Fatalf("Run() error = %v, want %v", err, wantErr)
	}

	if want := []string{"objects", "assemble"}; !reflect.DeepEqual(testPipeline.steps, want) {
		t.Fatalf("stage order = %v, want %v", testPipeline.steps, want)
	}
}

func TestRunRejectsNilContextAndPipeline(t *testing.T) {
	t.Parallel()

	var testPipeline pipeline
	//nolint:staticcheck // deliberately tests the nil-context guard
	if err := render.Run(nil, &testPipeline); !errors.Is(err, render.ErrNilContext) {
		t.Fatalf("nil context error = %v", err)
	}

	if err := render.Run(t.Context(), nil); !errors.Is(err, render.ErrNilPipeline) {
		t.Fatalf("nil pipeline error = %v", err)
	}
}
