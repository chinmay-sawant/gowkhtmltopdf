package render

import (
	"context"
	"errors"
	"fmt"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/errs"
)

// Pipeline is the narrow lifecycle seam for document rendering. The adapter
// owns mode-specific state; this package owns stage ordering and cancellation
// checks between stages.
type Pipeline interface {
	RenderObjects(ctx context.Context) error
	Assemble(ctx context.Context) error
	Finalize(ctx context.Context) error
}

var (
	ErrNilPipeline = errors.New("render: nil pipeline")
	ErrNilContext  = errs.ErrNilContext
)

// Run executes the rendering lifecycle in a fixed order. Checking the context
// between stages prevents a cancelled request from entering a later, more
// expensive phase after an earlier phase completed.
func Run(ctx context.Context, pipeline Pipeline) error {
	if ctx == nil {
		return ErrNilContext
	}

	if pipeline == nil {
		return ErrNilPipeline
	}

	stages := [...]func(context.Context) error{
		pipeline.RenderObjects,
		pipeline.Assemble,
		pipeline.Finalize,
	}

	for _, stage := range stages {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("render stage canceled: %w", err)
		}

		if err := stage(ctx); err != nil {
			return err
		}
	}

	if err := ctx.Err(); err != nil {
		return fmt.Errorf("render pipeline canceled: %w", err)
	}

	return nil
}
