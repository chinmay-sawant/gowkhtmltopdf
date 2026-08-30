//nolint:all // targeted unit tests for Phase 80
package layout

import (
	"context"
	"testing"
)

func TestFontPropsWave4(t *testing.T) {
	t.Parallel()
	// YAGNI wave 4 fields removed: ensure cascade ignores them without panic.
	ctx := &styleContext{
		ctx:       context.Background(),
		viewportW: 800,
	}
	s := initialStyle()
	raw := map[string]string{
		"font-feature-settings": `"liga" 1, "smcp" 1`,
		"font-kerning":          "none",
		"font-variant-caps":     "small-caps",
		"font-stretch":          "condensed",
		"font-size-adjust":      "0.58",
	}
	applyFontProps(&s, raw, 12, ctx)
	// No assertions: fields are YAGNI and must not be stored.
}
