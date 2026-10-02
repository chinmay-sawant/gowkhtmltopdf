## Summary

Adds `layout.DisplayList`, a second public entry over the placement `Lay`
already builds. Where `Lay` paints that placement to an `image.Image`,
`DisplayList` returns it as a retained list of vector operations and no image,
so a caller can replay the placement on its own canvas and keep text as glyphs
rather than pixels. Additive and backward compatible: `Lay` is unchanged and
remains the way to get a picture. One commit, 15 files, no dependency added.

The motivating consumer is a GPU host that wants to draw the placement as far
as possible instead of blitting one bitmap, but nothing in this repository
consumes the list yet. The replay backend lives on the consumer side so this
package keeps its no-CGO, two-dependency posture.

---

## Motivation / context

- `internal/layout` has always built a display list (`Result.Ops`, kinds
  `OpFillRect`, `OpStrokeRect`, `OpLine`, `OpText`, `OpImage`, `OpLinkURI`,
  `OpBullet`, `OpGridRun`) and then flattened it during paint. That list, and
  the font and glyph machinery behind it, were unreachable outside the module
  because of Go's `internal/` rule. The public `layout` package exposed only
  `Image()`, `Boxes()`, and `Size()`.
- The export is a thin public view rather than a second engine. `DisplayList`
  shares `imageout.LayoutResult` with `RenderLayout`, so both entries start from
  one placement and neither can drift from the other.
- No new modules. `TestDirectModuleAllowlist` still holds at two direct
  requires (`go-text/typesetting`, `tdewolff/canvas`), and this change adds no
  import outside the module.

---

## Changes

### Public surface (`layout/displaylist.go`)

- `DisplayList(ctx, *css.Document) (*Display, error)` lays out and stops before
  the paint step. It performs the same context, document, and option checks
  `Lay` performs and wraps failures the same way.
- `Display` carries `Ops` in source order, `Order` as paint-order indices,
  `Width` and `Height` in CSS pixels, and `PointsPerPixel` / `PixelPerPoint`
  for converting between the op coordinate space (points) and the canvas.
- Aliases: `DisplayOp` (the internal `Op`), `DisplayGroup` (`BlendGroup`), and
  `DisplayKind` (`OpKind`). A type alias does not carry constants, so the nine
  sequence kinds are restated, plus `DisplayOpNoop`.
- Helpers for painter decisions a replay would otherwise re-derive:
  `DisplayOrder`, `DisplayFakeBold`, and `DisplayTransformText`.

### Nil-safe payload access (`internal/layout/op_read.go`)

Rare payload lives on an embedded pointer to an unexported type, so a caller
outside the module cannot build or inspect it, and reading a promoted field on
an operation without one dereferences nil. `LinkURI`, `ImageBytes`, `ImageAlt`,
`Transform`, `BlendModeName`, `Opacity`, `Outline`, `TextTransformValue`, and
`NoFakeBoldValue` read it safely. `Opacity` delegates to the internal
`pdfPaintOpacity` so it reports the effective opacity the PDF and raster
painters use, including the "0 means no override" convention.

### Support changes

- `internal/pdf`: `Font.Bytes()` returns the raw SFNT face so a consumer can
  hand it to an independent shaper. The face is already parsed and cached, so
  this is an accessor, not new work.
- `internal/imageout`: `LayoutResult` is the front half of `RenderLayout`
  (validate, resolve the default font, lay out) with the rasterize step removed.
  `RenderLayout` now calls it, so the shared path is the tested path.
- `internal/imageout`: `CanvasHeight` exports the canvas-height resolution so a
  display-list consumer reproduces the same canvas without rasterizing.
- `internal/layout`: `OpKindNoop` names the deactivated-op sentinel. It was
  reachable only as the unexported `opKindNoop`, so a consumer iterating a page
  met kind 255 with no way to name it.

### Switch coverage

Exporting `OpKindNoop` brought it into the enum the `exhaustive` linter checks,
which flagged two production switches that were ignoring it implicitly. Both now
state it: `internal/imageout` (deactivated ops do not paint) and
`internal/convert` (deactivated ops never force a page wider).

### Docs

`layout/doc.go`, `documentation/library-api.md`, `documentation/architecture.md`,
`documentation/architecture/02-library-api.md`, and `CHANGELOG.md`.

---

## Honest limits

- **Canvas height can differ by one pixel from `Lay`.** `Lay` reads its size
  back off the painted picture, which rounds at the supersample factor, while
  `DisplayList` converts the placement height straight from points and
  truncates. Measured across roughly 310 documents: 61 mismatches, all delta
  `(0, 1)`, never in width. Documented in the package comment, the display
  struct, the API reference, and the changelog rather than papered over.
- **`DisplayList` skips the raster budget.** A canvas too large for `Lay` to
  rasterize still returns a display list, so a caller that hands it to a GPU
  applies its own size limit. Documented.
- **No replay backend here.** Nothing in this repository draws the operations.
- `Display.Height` aside, `Width` is exact, because the viewport width is an
  integer.

---

## Verification

- `go build ./...` clean.
- `go test ./... -count=1` passes across the whole module.
- `go test ./internal/pdf/ -run TestDirectModuleAllowlist` passes; dependency
  policy unchanged.
- `golangci-lint run ./...` reports no finding in any file this branch touches.
  The two pre-existing findings outside it (`buildBlock` at 61 lines, and
  `internal/layout/viewport_height_test.go`) are unchanged and predate this
  branch.
- `gofmt -l` clean on every touched file.
- `make claim-scan` clean.
- New tests cover: canvas agreement with `Lay`, paint order being a permutation,
  font bytes being loadable, ascent matching the op's point size, opacity
  folding, opacity of the zero value, every accessor on the zero value, every
  emitted kind being nameable, and overflow clipping producing a skippable
  deactivated op.
- `bash scripts/check-file-size.sh` was red before this branch on
  `internal/layout/layout.go` (2372 lines against a recorded 2362). This branch
  does not change that count.

### Known flake, not introduced here

`go test -race ./internal/layout/` fails intermittently on `master` as well.
The shared object is the `*go-text/typesetting/font.Face` cached per `pdf.Font`
under `gotOnce`; the default faces are a process-wide singleton, so parallel
tests shape through one face, and the lazy caches in typesetting v0.3.4 are
unsynchronized. Reproduced on the unmodified tree by moving this branch's files
aside. A fix needs per-face locking or one face per concurrent run; it is out of
scope for this branch and is not caused by it.

---

## Related issues

None. No issue was filed for this work.

## Diff

`scripts/pr-diff-stat.sh` output is in the PR body below the fold on GitHub.
