# fix-log/fix-pdf-fonts.md — agent fix-pdf-fonts (wave 1)

Rows: P5-03 P5-04 P5-05(pdf side) P5-06 P5-07 + optional stray-note folds.
Owns: `internal/pdf/*`, `internal/pdf/assets/*`, `internal/svg/*`.

## Per-CID status

### P5-03 — SetGrayscale paint-time seam — **done (implemented)**
**Decision:** implement the paint-time grayscale seam (recommended option in the
row and contract #9), rather than deleting the setter. Rationale: the flag is
threaded through the whole settings surface (`settings/reflect.go` colormode
mapping, `cli --grayscale`, `api.go`, `convert.go:159-160`) and cli/settings
tests already pin it; removing it would ripple outside this package. The seam
was real but misplaced — color flows through `Content` at paint time.

Changes:
- `content.go`: `SetFillColor`/`SetStrokeColor` fold Rec.601 luma
  (0.299r+0.587g+0.114b) when `c.doc != nil && c.doc.grayscale`.
- `images.go`: XObjects desaturate at embed time — PNG pixels folded in the
  existing decode loop (keeps alpha soft-mask); RGB JPEGs decoded with
  stdlib `image/jpeg`, luma-folded, re-encoded as 1-component gray JPEG
  (best effort; decode failure keeps the pass-through stream). Dict then
  declares `/ColorSpace /DeviceGray` (1 component) as before.
- `convert.go:159-160` verified: it already calls `doc.SetGrayscale(cmd.Global.Grayscale)` —
  the flag now has its promised effect; file left untouched (other agent's).
- New tests: `TestSetFillColorGrayscale` (equal r,g,b lines + off-mode
  unchanged), `TestGrayscaleGetter`, `TestGrayscalePNGFold`,
  `TestGrayscaleJPEGFold`, `TestGrayscalePNGAlphaKept`.

### P5-04 — typed objRef + dict builder — **done**
- `pdf.go`: `type objRef int` with `String()` "N 0 R"; `newObject() objRef`,
  `setDict(r objRef, s string)`, `setStream(r objRef, raw []byte)`; `refID`
  panic-helper deleted (refs cannot be malformed by construction).
- `type dict []string` with `add(k, v ...string)` / `String()` ("<< … >>")
  per the Future snippet; now used for the font/descriptor/stream/image
  dicts in `fonttype0.go`, `images.go`, and the catalog/info dicts in
  `pdf.go`. Info-dict key order preserved (Title..Keywords before
  Creator/Producer/Dates) so output stays byte-identical.
- Internal refs typed: `Page.ref`/`contentRef`, `annotation.annotRef`,
  `Outline.refStr`, `imageResource.ref`, `fontCache` values, `Document`
  catalog/info refs, `ensureToUnicode` return.
- Exported surface kept as strings (no convert/layout churn): `PageRef`,
  `Outline.PageRef`, `AddLinkURI`, `AddLinkDest` unchanged.
- `finalizeOutlines`/`buildOutlineItems` now validate `PageRef` via
  `parseRef` and range-check against allocated objects, returning an error
  instead of emitting a bogus /Dest (hypothesis validated:
  `PageRef("999999 0 R")` now fails `Write`).
- Bonus fold (p5-01 "Not raised" note): `Write` no longer records xref
  offsets for objects whose dict was never set (was recording the offset of
  the *next* object — latent corruption).
- New test: `TestOutlineBadPageRefFails` (bad refs error, empty ref = no
  /Dest, valid ref emits /Dest).

### P5-05 — derived font data onto *Font (pdf side) — **done**
- Deleted package-level `gotextFaceCache sync.Map` + `gotextFaceEntry`.
- `Font` gains `gotOnce sync.Once` + `gotFace *gtfont.Face` and
  `revOnce sync.Once` + `rev map[uint16]rune` (fonts.go; sync + gtfont
  imports only, no new dependency).
- `gotextFace` is now a `*Font` method building the face once (nil on
  failure, cached via sync.Once); `reverseCmap()` builds once from the
  immutable `f.cmap` instead of per text op (CJK faces ~30k-entry maps).
- Raster glyph atlas (`internal/imageout/ttfraster.go`) is
  fix-imageout-wave2's row — untouched.
- Tests: existing shape/subset suites pass (reverseCmap behavior identical).

### P5-06 — collapse simple+type0 embed pipelines — **done**
- `ensureFont(f, used) (objRef, error)` with a `type0 bool` mode switch per
  the Future snippet; cache key `v%d|name|runes` (mode|name|runes, per-Document).
- Shared `embedFontFile(f, sub, fontName)` (FontFile2 stream + FontDescriptor)
  — single copy of the ~25-line constructor that was duplicated with a
  one-string difference.
- `emitSimple` / `emitType0` tails; `widthsInEm(sub, upm)` is the single
  home of units→1000-em conversion feeding both `/Widths` (`subsetWidths`
  now indexes it) and the Type0 `/W` array.
- Deleted the now-dead `embeddedFont` struct (fontpdf.go).
- Dicts byte-identical to the old hand-stringed output (verified by
  reconstruction; in-package tests substring-green).
- Base-14 path untouched (no separate implementation exists; all fonts embed).
- Cache behavior preserved: `TestFontCacheSharedAcrossPages` still sees
  exactly one `/FontFile2`.

### P5-07 — stop swallowing the font-embed error — **done (pdf side)**
- `Content.fonts()` now returns `(map[string]string, error)` and wraps
  failures as `fmt.Errorf("embed font %s: %w", name, err)`; the
  `continue // skip broken font` swallow is gone.
- `finalizePage` returns the error; `finalize` propagates it into
  `Document.Write`'s return value (Write already returned finalize's error).
- New test `TestFontEmbedErrorPropagates`: an empty `*Font` (subsetting must
  fail) makes `Write` return an error wrapping "embed font F1".
- The `_ = c.AddJPEGImage(...)` discard sites in `internal/layout/paint.go`
  are fix-layout's row (P5-07 layout side) — not my package; no marker
  needed per the row text. drawImage error propagation is theirs.

### Stray notes fold-in (p5-01 "Not raised", optional) — **done, all contained**
- `Font.FamilyNames()` side-effect → explicit `LoadNames()` (documented
  deliberate PostScriptName fill); `FamilyNames()` delegates to it so
  `internal/layout` test callers keep working; `Registry.AddFont` calls
  `LoadNames` directly.
- `nfcNormalize` → `stripOrphanMark` (behavior unchanged; comment notes it
  is a best-effort stand-in, not full NFC).
- `jpegSize`/`jpegColorSpace` → one `jpegScan(data) (w, h, components, err)`
  marker-walk (images.go + image_test.go updated).
- Cross-note (Font embeds + Registry live here): no action beyond rows.

## Files changed

- `internal/pdf/pdf.go` — objRef, dict, typed refs, PageRef validation,
  error propagation, xref empty-dict fix, catalog/info dicts.
- `internal/pdf/content.go` — grayscale fold, `fonts() (map, error)`.
- `internal/pdf/images.go` — objRef, jpegScan, grayscale JPEG/PNG fold.
- `internal/pdf/fonttype0.go` — collapsed ensureFont/embedFontFile/emit*.
- `internal/pdf/fontpdf.go` — widthsInEm, embeddedFont removed, ensureToUnicode objRef.
- `internal/pdf/shape_gotext.go` — Font-local face cache, once-built reverseCmap.
- `internal/pdf/fonts.go` — Font cache fields, LoadNames.
- `internal/pdf/shape.go` — stripOrphanMark rename.
- `internal/pdf/registry.go` — LoadNames call.
- Tests: `pdf_test.go`, `image_test.go`, `fonttype0_test.go`.
- `internal/svg/*` — no changes needed (validated only).

## Validation

- `go build ./internal/pdf/... ./internal/svg/...` — clean.
- `go test -count=1 ./internal/pdf/... ./internal/svg/...` — ok (no golden
  diffs; grayscale only changes output when the flag is set, P5-04/05/06/07
  behavior-neutral for valid documents; pdf tests include new grayscale,
  outline-validation, jpegScan and embed-error tests).
- `go vet ./internal/pdf/... ./internal/svg/...` — clean.
- Repo-wide `go build ./...` currently fails in `internal/css` (undefined
  `err`, pre-existing, owned by fix-css) and `internal/convert`
  (`settings.PdfGlobal.Allow`/`EnableLocalFileAccess` missing, in-flight
  fix-settings-cli P2-07 work) — both pre-existing/parallel, not caused by
  this agent.

## Remaining markers

- None added. All work stayed inside owned packages; the layout
  `_ = AddJPEGImage`/`AddPNGImage` discard sites belong to fix-layout's own
  P5-07(layout side) row and `ttfraster.go` glyphCache to fix-imageout-wave2's
  P5-05 row — deferred by ownership, not by marker.
