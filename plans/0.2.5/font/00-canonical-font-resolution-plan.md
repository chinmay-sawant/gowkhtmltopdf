# v0.2.5 Font Resolution, Fallback, and Compliance Plan

> **Status:** complete — validated 2026-08-19
> **Parent:** [`plans/0.2.5/README.md`](../README.md)
> **Predecessor:** [`plans/0.2.4/31-canonical-0.2.4-roadmap.md`](../../0.2.4/31-canonical-0.2.4-roadmap.md)
> (phases 31–39 remain closed; this track does not reopen them)
> **Owner:** `internal/pdf`, `internal/layout`, `internal/convert` (incl.
> `prepare` / HF), CLI and public `Document` / `ImageDocument` font options
> **Constraint:** pure Go; no CGO, browser embedding, Pango, Fontconfig, or
> second PDF writer. Direct modules remain allowlisted to
> `go-text/typesetting` and `tdewolff/canvas` only (`AGENTS.md`).

## Purpose

Make the font behavior honest and useful for two different real-world cases:

- deterministic conversion with only the bundled fonts available;
- exact author-selected fonts when a licensed, supported face is supplied by
  `--font-path`, `--use-system-fonts`, or `@font-face`.

The current PDF writer already performs the hard compliance work. This plan
connects font selection to that writer without weakening the existing gates or
turning WeasyPrint/Chrome into a default runtime dependency.

## Evidence sources

Authoritative project docs and plans win when wiki pages drift
(`knowledge-base/SCHEMA.md`). This plan was reconciled against:

| Source | Use |
|--------|-----|
| [`documentation/fonts.md`](../../../documentation/fonts.md) | Normative operator font contract |
| [`documentation/compatibility-matrix.md`](../../../documentation/compatibility-matrix.md) | Matrix rows to correct in Phase 7 (alias wording; HF font-name) |
| [`documentation/deferred.md`](../../../documentation/deferred.md) | Non-goals; HTTPS WOFF1 already works (row may still understate that) |
| [`knowledge-base/wiki/syntheses/fonts-and-typography.md`](../../../knowledge-base/wiki/syntheses/fonts-and-typography.md) | Compiled synthesis (path retargeted to this folder 2026-08-19) |
| [`knowledge-base/wiki/syntheses/roadmap.md`](../../../knowledge-base/wiki/syntheses/roadmap.md) | Lists this track under `plans/0.2.5/font/` |
| [`knowledge-base/raw/security-posture.md`](../../../knowledge-base/raw/security-posture.md) | `@font-face` as untrusted parse input; FetchSub SSRF/ACL |
| [`plans/0.2.0/phases/pending-phase-items/09-remote-webfonts.md`](../../0.2.0/phases/pending-phase-items/09-remote-webfonts.md) | WOFF2/Brotli remains a separate epic |
| Code under `internal/pdf`, `internal/layout`, `internal/convert/prepare`, `internal/css` | Reality check 2026-08-19 |

## Package map (today)

There is no `internal/font` package. Fonts are owned by `internal/pdf` and
consumed by layout / convert / imageout:

| Area | Primary paths | Key types / funcs |
|------|---------------|-------------------|
| Bundled assets | `internal/pdf/assets/` | Liberation Sans/Serif/Mono R/B/I/BI + DejaVu R/B |
| Parse / metrics | `internal/pdf/fonts.go` | `Font`, `ParseTTF` |
| WOFF1 | `internal/pdf/woff.go` | `ParseFontBytes`, `DecodeWOFF` |
| Bundled families | `internal/pdf/faces.go` | `FaceSet`, `LoadDefaultFaces`, `ResolveFamily` |
| Opt-in discovery | `internal/pdf/registry.go` | `Registry`, `ScanFontDirs`, `Lookup`, `FindWithGlyph` |
| Subset / embed | `subset.go`, `fonttype0.go`, `fontpdf.go` | `subsetFont`, `ensureFont` |
| Shaping | `shape.go`, `shape_gotext.go` | `ShapeTextFont` via `go-text/typesetting` |
| CSS `@font-face` | `internal/css/css.go` | `FontFace{Family,Src}` only today |
| Merge into registry | `internal/convert/prepare/styles.go` | `MergeFontFaces` / `fetchFontFace` |
| Layout selection | `internal/layout/layout.go` | `lookupFaceFor`, `faceForRune`, `facesWithGlyph` |
| HF fonts | `internal/convert/hf.go` | `resolveHFFont` (registry, else Liberation) |
| Public / CLI | `document.go`, `internal/cli/flags.go`, `internal/settings` | `FontPaths`, `UseSystemFonts` |

## Current-state findings

| Area | Current implementation | Consequence for this plan |
|------|------------------------|---------------------------|
| Bundled faces | `internal/pdf/assets` embeds Liberation Sans/Serif/Mono regular, bold, italic, bold-italic plus DejaVu fallback | Safe deterministic baseline; preserve it |
| Named CSS aliases | `FaceSet.ResolveFamily` maps Georgia, Times, Arial, Courier New, … to bundled families; `lookupFaceFor` hits this after registry miss | Can bypass later author-stack entries; not exact named fonts. Locked today by `TestFaceResolveFamilyAliases` |
| Registry lookup | `Registry.Lookup` is exact by internal family name; CSS generics expand | Keep exactness; do not import Fontconfig aliases |
| Optional discovery | `--font-path` / `--use-system-fonts` → `ScanFontDirs` depth 2, `.ttf`/`.otf` only; CFF/`OTTO` and parse failures skipped **silently** | Flag is opt-in file discovery, not Fontconfig; Phase 3/4 must add skip diagnostics. A bare *file* path is not first-class (ReadDir fails → silent miss) |
| Layout selection | `lookupFaceFor` / `faceForRune` / `facesWithGlyph` duplicate policy; no `FontResolver` type | Central resolver seam (Phase 2) required |
| `@font-face` fetch | Local + `http(s)` TTF/OTF/WOFF1 via `FetchSub` under ACL/timeout/body cap (**already shipped**) | Do not list HTTPS webfonts as unimplemented. WOFF2/EOT/`data:` skipped with warning |
| `@font-face` model | `css.FontFace` stores only `Family` + `Src`; weight/style are **not retained** (not merely ignored later) | Phase 3 must extend the CSS model, then register style metadata |
| Parsing | `ParseTTF` accepts TrueType outlines and rejects CFF/`OTTO`; WOFF1 decompress has size/overlap caps | Supported-format and operator diagnostics need an explicit contract |
| PDF embedding | `ensureFont` subsets and emits simple TrueType or Type0/CIDFontType2 with `FontFile2` and `ToUnicode`; cache key includes face fingerprint | Reuse this pipeline; do not add a second subsetter |
| Embed preflight | No preflight + re-layout path if subset/embed fails after metrics were used | Phase 5 must own convert orchestration for fallback re-layout |
| Compliance | PDF 1.7 + PDF/A-3a/PDF/UA-1 and PDF 2.0 + PDF/A-4/PDF/UA-2 implemented; default remains unclaimed PDF 1.4 | New faces must pass the same embed-or-fail and profile gates |
| Public API | `Document.FontPaths` / `UseSystemFonts` (and image equivalents) map to `settings.PdfGlobal` | No new public font option required for the first implementation |
| HF font name | `resolveHFFont` already consults the registry before Liberation | Compatibility matrix “always Liberation” wording is stale; Phase 7 must align docs |
| External engines | Chrome resolves OS-installed Georgia; WeasyPrint follows Linux Fontconfig (Gelasio on the fixture-55 host) | Differential tests must control the font **bytes**, not only the CSS family name |
| Docs drift | Matrix §2.3 understates that named aliases map to Liberation; deferred.md understates HTTPS WOFF1 | Phase 7 must fix docs; KB paths already retargeted |

## Target resolution contract

For each cascaded `font-family` list and requested weight/style:

1. Try an exact registered/document face for each family token in CSS order.
2. Select the requested weight/style from that family, with a documented
   deterministic nearest-face rule when the exact face is absent.
3. If no named face is available, continue through the remaining author
   family tokens rather than silently importing a host alias.
4. Resolve `serif`, `sans-serif`, and `monospace` to the bundled canonical
   families.
5. Use bundled DejaVu or an explicitly registered glyph-covering face only as
   a last-resort per-rune fallback.
6. Record enough face identity for layout caches and PDF subset caches to stay
   distinct when two files share a display name.

The exact default behavior for legacy aliases must be locked in Phase 1. The
recommended policy is that `Georgia` is exact only when a family named Georgia
is supplied; otherwise `Georgia, "Times New Roman", serif` ends at the
deterministic bundled serif fallback. The renderer must not promise that
Liberation Serif is the actual Georgia typeface.

## Automatic fallback and compliance rule

Fallback is part of the normal font contract; a missing optional font must not
become a user-visible conversion error when a valid fallback exists.

The fallback sequence is:

```text
exact supplied face
  → next author CSS family
  → bundled Liberation family for the generic family
  → bundled DejaVu / registered glyph-covering face for a missing rune
  → error only when no usable face remains
```

Load-time failures such as an invalid path, unsupported CFF/`OTTO` font,
unsupported WOFF2 source, malformed font, or missing requested style must be
treated as an unavailable candidate and continue through that sequence. The
operator should receive a warning or diagnostic, but conversion should
continue when the CSS stack or bundled fallback can satisfy the text.

Embedding failure needs an additional safety rule. A font must not be swapped
after layout has already used its metrics, because a fallback can change line
wrapping and pagination. The implementation must therefore preflight the
selected face for the actual used glyph set before committing paint/output;
when that preflight fails, it must select the next valid fallback and re-layout
the affected object before writing the final PDF. Only when every valid
fallback is exhausted may conversion return an error. A claiming PDF must
never be emitted with an unembedded or non-compliant font.

**Ownership note:** the resolver (Phase 2) marks candidates unavailable;
`internal/convert` owns the preflight → fallback → re-layout orchestration
relative to `RenderObjects` / `Assemble` / `Finalize` (Phase 5). Layout must
not silently swap faces mid-paint.

## Design seam

Create one internal font-resolution module around the existing `FaceSet` and
optional `Registry`. **Recommended placement:** types under `internal/pdf`
(where `FaceSet` / `Registry` already live), with layout calling a small
interface. Do not invent `internal/font` unless Phase 2 proves a cycle or
layering violation.

Its interface should be small enough for layout (and HF) to ask for a
family/style face and a glyph fallback without knowing whether the face came
from bundled assets, an explicit directory, or document `@font-face`.

The seam must own:

- family-stack order;
- exact registered-family lookup;
- generic-family mapping;
- weight/style selection;
- per-rune fallback;
- stable face identity and cache keys;
- optional operator diagnostics about skipped or selected faces.

The public `Document` and CLI interfaces remain unchanged unless a concrete
gap is proven by a phase gate.

## Supported-font policy

The implementation must remain honest about the current writer:

- supported: TrueType fonts and OpenType containers with TrueType outlines;
- supported after conversion: WOFF1;
- rejected or skipped: CFF/`OTTO`, WOFF2, EOT, and `data:` font sources;
- variable-font behavior must be explicitly tested and either supported with
  a stable instance policy or rejected with a clear diagnostic;
- every selected face must be subsettable and embedded as `FontFile2`;
- every emitted text font must retain `ToUnicode` mappings;
- a claiming PDF must fail before writing if a selected face cannot satisfy
  the profile's embedding requirements;
- no proprietary Georgia font is bundled without an explicit redistribution
  license and asset manifest.

**WOFF2 is out of this track.** Shipping WOFF2 requires an allowlist
amendment (Brotli or equivalent) tracked in
`plans/0.2.0/phases/pending-phase-items/09-remote-webfonts.md`. This resolution
plan keeps WOFF2 as a documented skip with a clear diagnostic.

## Phase map

| Phase | File | Outcome |
|------:|------|---------|
| 1 | [phase-01-resolution-contract.md](phase-01-resolution-contract.md) | Browser-like, deterministic family-stack contract and acceptance matrix |
| 2 | [phase-02-resolver-seam.md](phase-02-resolver-seam.md) | One internal resolver for family and glyph selection |
| 3 | [phase-03-discovery-and-font-face.md](phase-03-discovery-and-font-face.md) | Correct `--font-path`, system opt-in, and `@font-face` style handling |
| 4 | [phase-04-font-format-and-asset-policy.md](phase-04-font-format-and-asset-policy.md) | Supported-format matrix, fixtures, licensing, and failure diagnostics |
| 5 | [phase-05-pdf-embedding-and-compliance.md](phase-05-pdf-embedding-and-compliance.md) | Loaded faces pass simple/Type0 embedding and all compliance profiles |
| 6 | [phase-06-corpus-and-engine-comparison.md](phase-06-corpus-and-engine-comparison.md) | Fixture-55 and corpus evidence with controlled Chrome/Weasy comparisons |
| 7 | [phase-07-docs-and-operator-contract.md](phase-07-docs-and-operator-contract.md) | Public docs, CLI help, KB, diagnostics, and compatibility claims agree |
| 8 | [phase-08-validation-and-closure.md](phase-08-validation-and-closure.md) | Performance, full gates, artifacts, and plan closure |

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Existing `FaceSet`, `Registry`, CSS parser, layout, and PDF writer | All phases |
| Phase 1 contract | Resolver and input implementation |
| Phase 2 resolver seam | `@font-face`, layout, and corpus work |
| Phase 3/4 accepted faces | Embedding and compliance proof |
| Phase 5 compliant outputs | Corpus and external differential evidence |
| Phase 6 accepted visual behavior | Docs, KB, and closure |

## Non-goals

- Rebuilding the PDF writer or adding a second font subsetter.
- Making Chrome, WeasyPrint, Pango, or Fontconfig runtime dependencies.
- Making host system font discovery the default.
- Claiming pixel parity with a browser whose installed fonts differ.
- Bundling Microsoft Georgia without redistribution rights.
- Reopening the completed 0.2.4 Document API phases for unrelated changes.
- Implementing WOFF2/Brotli inside this track (separate pending item + allowlist amendment).
- Marking existing fixture goldens as updated before visual and compliance
  evidence is reviewed.

## Definition of done

- The same HTML has a documented result with no font flags, with an explicit
  font directory, and with `@font-face`.
- An actual supplied family wins over the bundled fallback when its internal
  family name matches the CSS name.
- An invalid or unembeddable optional face automatically falls through to a
  valid CSS/bundled fallback before layout is committed.
- Fallback re-layout is proven for a wrapping-sensitive document; no
  post-layout font swap silently changes painted metrics.
- Missing named families follow the author stack and then deterministic
  generic fallback; host aliases do not change default output.
- Regular, bold, italic, bold-italic, Latin-1, and Type0/Unicode cases are
  covered by tests.
- PDF 1.4, PDF 1.7, PDF/A-3a + PDF/UA-1, PDF 2.0, and PDF/A-4 + PDF/UA-2
  outputs retain valid embedding and Unicode maps.
- Fixture-55 and the selected corpus are visually reviewed with screenshots,
  text extraction, page counts, and embedded-font inspection.
- `documentation/fonts.md`, compatibility matrix, deferred notes, CLI/API
  docs, and `knowledge-base/` font pages agree with the shipped contract
  (including no remaining live `temps/font/` cites).
- All phase files contain command outcomes before any row is marked complete.

## Validation outcomes (2026-08-19)

```
$ CGO_ENABLED=0 make test
go test ./...
(ok — all packages)

$ CGO_ENABLED=0 make lint
golangci-lint run ./...
(exit 0)

$ CGO_ENABLED=0 go build -o /tmp/font-0.2.5-evidence/gowkhtmltopdf ./cmd/gowkhtmltopdf
$ CGO_ENABLED=0 go build -o /tmp/font-0.2.5-evidence/gowkhtmltoimage ./cmd/gowkhtmltoimage
$ gowkhtmltopdf -q --allow-local-files -o /tmp/font-0.2.5-evidence/f55-default.pdf \
    testdata/golden/fixture-55-lantern-cooperative-report.html
$ gowkhtmltopdf -q --allow-local-files -o /tmp/font-0.2.5-evidence/f55-default-b.pdf \
    testdata/golden/fixture-55-lantern-cooperative-report.html
$ cmp …/f55-default.pdf …/f55-default-b.pdf   # STABLE=yes
# PDF 1.4, FontFile2 + ToUnicode + Liberation present, 4 page objects, ~60 KiB
$ gowkhtmltopdf -q --allow-local-files --font-path internal/pdf/assets \
    -o /tmp/font-0.2.5-evidence/f55-fontpath.pdf …/fixture-55-….html
$ gowkhtmltoimage -q --allow-local-files -o /tmp/font-0.2.5-evidence/f55.png …/fixture-55-….html
```

Focused proofs: `go test ./internal/pdf ./internal/layout ./internal/convert ./internal/css ./internal/cli ./internal/imageout .`
Cover FontResolver, discovery diagnostics, `@font-face` weight/style, preflight re-layout, CLI/library font-path.

