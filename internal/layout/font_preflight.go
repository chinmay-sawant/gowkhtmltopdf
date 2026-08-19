package layout

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/line"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// maxFontEmbedRelayouts bounds embed-preflight → MarkUnavailable → re-layout
// loops for one object. One retry is the common case (optional face fails,
// bundled fallback wins); a few more cover multi-face stacks.
const maxFontEmbedRelayouts = 4

// ErrNoEmbeddableFace is returned when every candidate face fails embed
// preflight and no usable fallback remains.
var ErrNoEmbeddableFace = errors.New("layout: no embeddable font remains for required text")

// embedPreflight is the subset/embed gate used before paint. Tests may
// replace it to force a preflight failure without inventing a second subsetter.
//
//nolint:gochecknoglobals // test seam for synthetic embed failures
var embedPreflight = pdf.PreflightEmbed

// SetEmbedPreflightForTest replaces the embed preflight function. Call the
// returned function to restore the previous hook.
func SetEmbedPreflightForTest(hook func(*pdf.Font, []rune) error) func() {
	prev := embedPreflight

	if hook == nil {
		embedPreflight = pdf.PreflightEmbed
	} else {
		embedPreflight = hook
	}

	return func() {
		embedPreflight = prev
	}
}

// WithFontPreflight lays out root, preflights every text face against the
// runes it paints, and re-lays-out when an optional face fails so metrics
// stay consistent with the face that will actually embed.
//
//nolint:cyclop,funlen // preflight loop + face loading branches
func WithFontPreflight(
	ctx context.Context,
	root *html.Node,
	opts Options,
	log io.Writer,
	objectLabel string,
) (*Result, error) {
	faces := opts.Faces
	if faces == nil {
		var err error

		faces, err = pdf.LoadDefaultFaces()
		if err != nil {
			return nil, fmt.Errorf("font preflight faces: %w", err)
		}

		opts.Faces = faces
	}

	resolver := opts.Resolver
	if resolver == nil {
		resolver = pdf.NewFontResolver(faces, opts.Registry)
		if log != nil && log != io.Discard {
			resolver.Warn = func(msg string) {
				line.Emit(log, line.Warn, "%s: %s", objectLabel, msg)
			}
		}
	}

	opts.Resolver = resolver

	var last *Result

	for attempt := range maxFontEmbedRelayouts {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("%s: font preflight canceled: %w", objectLabel, err)
		}

		lres, err := LayoutContext(ctx, root, opts)
		if err != nil {
			return nil, err
		}

		last = lres

		failed := preflightResultFonts(lres)
		if len(failed) == 0 {
			return lres, nil
		}

		for _, face := range failed {
			resolver.MarkUnavailable(face, "embed preflight failed")
			line.Emit(log, line.Warn,
				"%s: face %q failed embed preflight; selecting next CSS/bundled fallback and re-layout (attempt %d)",
				objectLabel, face.PostScriptName, attempt+1)
		}
	}

	if last != nil && len(preflightResultFonts(last)) == 0 {
		return last, nil
	}

	return nil, fmt.Errorf("%s: %w", objectLabel, ErrNoEmbeddableFace)
}

func preflightResultFonts(res *Result) []*pdf.Font {
	if res == nil {
		return nil
	}

	used := collectLayoutFontRunes(res.Ops)
	failed := make([]*pdf.Font, 0)

	for face, runes := range used {
		if err := embedPreflight(face, runes); err != nil {
			failed = append(failed, face)
		}
	}

	return failed
}

func collectLayoutFontRunes(ops []Op) map[*pdf.Font][]rune {
	out := make(map[*pdf.Font]map[rune]struct{})

	for i := range ops {
		paintOp := &ops[i]
		if paintOp.Kind != OpText || paintOp.Font == nil || paintOp.Text == "" {
			continue
		}

		set := out[paintOp.Font]
		if set == nil {
			set = make(map[rune]struct{})
			out[paintOp.Font] = set
		}

		for _, r := range paintOp.Text {
			set[r] = struct{}{}
		}
	}

	result := make(map[*pdf.Font][]rune, len(out))

	for face, set := range out {
		runes := make([]rune, 0, len(set))
		for r := range set {
			runes = append(runes, r)
		}

		result[face] = runes
	}

	return result
}
