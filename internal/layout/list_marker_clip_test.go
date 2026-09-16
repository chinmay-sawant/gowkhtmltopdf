package layout

import (
	"testing"
)

// The reference viewport clips content left of the page content box, so an
// outside disc that hangs entirely past the page edge does not paint.
// cplusplus.com resets ul{padding:0} and paints 25 visible markers Chrome
// shows none of (cplusplus-tutorial finding 4).
func TestOutsideListMarkerClippedAtPageEdge(t *testing.T) {
	t.Parallel()

	clipped := layoutHTML(t, `<html><body><div class="content"><ul><li><a>Compilers</a></li></ul></div></body></html>`,
		sheet(t, `body { margin: 0 } * { margin: 0; padding: 0 }
.content { padding-left: 5px }
ul { list-style-type: disc }`))

	if bullets := opsOfKind(clipped, OpBullet); len(bullets) != 0 {
		t.Fatalf("clipped marker painted = %+v, want none", bullets)
	}

	withRoom := layoutHTML(t, `<html><body><div class="content"><ul><li><a>Compilers</a></li></ul></div></body></html>`,
		sheet(t, `body { margin: 0 } * { margin: 0; padding: 0 }
.content { padding-left: 24pt }
ul { list-style-type: disc }`))

	bullets := opsOfKind(withRoom, OpBullet)
	if len(bullets) != 1 {
		t.Fatalf("marker with room = %+v, want 1", bullets)
	}

	if bullets[0].X < 0 {
		t.Fatalf("marker X=%.3f hangs past the page edge; want >= 0", bullets[0].X)
	}
}
