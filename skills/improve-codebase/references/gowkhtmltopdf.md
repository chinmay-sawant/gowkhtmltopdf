# Calibration — gowkhtmltopdf

Load this file only when the module path is `gowkhtmltopdf`. It is
context for the three lenses, not a second finding list. Do not re-file
closed CR/ARC/P-rows from `plans/reviews/improve-codebase/` unless
current source has regressed.

Authoritative prose lives in `documentation/architecture/` and
`CONTRIBUTING.md`. This file is the short map.

## Product ceiling

Controlled-report renderer. No JS, no CGO, no browser, no third-party
PDF/HTML/CSS libraries. Direct modules ⊆ `go-text/typesetting` +
`tdewolff/canvas` (`internal/pdf.TestDirectModuleAllowlist`). Flex/grid/
position are report features, not CSS completeness.

## Engine seam

| Mode | Job | Entry | Lifecycle |
|---|---|---|---|
| PDF | `convert.Request` | `convert.Run` | `render.Pipeline` |
| Image | `imageout.Request` | `imageout.RunRequest` | same pipeline; `Assemble` is a no-op |

Callers above the seam: root `api.go`, `internal/app`. `cmd/*` talks only
to `app` + `cli`. Image shares load → html → css → layout and forks at
paint/write.

## DAG (do not invert)

- `cmd` → `app` → `cli` → `settings`. `cli` never imports `cmd` or the root.
- Root `api.go` never imports `cli`. `convert` never imports `cli`.
- `prepare` / `render` / `islands` never import `convert`.
- `layout` / `pdf` never import `settings` or `cli`.
- Untrusted bytes enter only via `load.Loader.Load` and
  `load.ResourceContext.Fetch`.
- Known live leak to inspect, not to treat as a new invention:
  `imageout` → `convert` (prepare aliases / `ValidateRenderableObjects`).

## Contracts to preserve

- Clone at the public boundary (`settings/clone.go`, `AddObject`,
  `WithGlobal`, `toRequest`, `Output()`). CLI process-exclusive ownership
  of `cli.Command` is documented; do not add library sinks on it.
- Nil context is `errs.ErrNilContext`. Alias it; do not `errors.New` a
  second value with the same meaning.
- Public fluent nil / invalid `WithCopies` / `WithPageSize` **panic**.
  `Set` / `WithSetting` / `Convert` / `Run*` **return error**.
- Policy A (`internal/settings/doc.go`): typed field only if convert /
  load / imageout / layout consume it. Otherwise `Ignored`.
- `LibraryVersion` (wkhtml compat) ≠ `VERSION` (release).
- `render.Run` checks `ctx.Err()` between stages. Layout may store `ctx`
  with `containedctx` and poll at recursion boundaries — do not spread
  that pattern.
- Layout is single-goroutine, zero locks. Do not mutex `pdf.Document`.
- Image jobs are exactly one input (`imageout.ErrMultipleInputs`).
- Dual flag pair for local files: `enablelocalfileaccess` **and**
  `load.blocklocalfileaccess=false`.

## Extension tables (owner of the next row)

| Change class | Owner | Must also touch |
|---|---|---|
| CSS property | `layout` used values (`style.go`, `style_properties.go`, `style_cascade.go` `styleGroups` / `inheritableProps`) | optional `css/values.go`; consumer in layout/paint; matrix row |
| Selector / pseudo | `internal/css` parse **and** match together | unknown pseudos stay on the compound and never match |
| HTML element | `html` allowlist + `uaDecls` + `engine.build` | matrix §1 if it paints |
| Formatting context / pagination | `layout` (`buildInFlowDisplay`, `paint_flow.go`, `paint_pagination.go`) | new `OpKind` → `paint.go` **and** imageout raster + `PaintOrder` |
| CLI flag / dotted key | `settings` struct + `reflect.go` `register*` + `cli/flags.go` | engine read; `TestKeyTableSetGetParity`; cli.md |
| Typed `With*` | `settings/options.go` + thin `api.go` wrapper | `api_test.go`; library-api.md |
| PDF object | `internal/pdf` writer + convert wiring | needles, not byte-identical PDFs |
| Image-only knob | `imageout` + `ImageGlobal` | do not fork CSS/layout |
| Load / ACL | `internal/load` | THREAT-MODEL; deny stays default |

## Proof that is law here

- Non-doc change: `make lint` + `make test` before any `[x]`.
- Layout/print: also `go test ./internal/layout/ -count=1` and
  `go test ./internal/convert/ -run 'TestGoldenCorpus' -count=1`.
- New golden fixture needs a `fixturePageBounds` row or the walker fatals.
- Goldens are structural (`%PDF-`, xref, `/FontFile2`, page envelope,
  optional needles). They do not catch overlap — visual is `make samples`.
- Security is a load unit (allow **and** deny), never a golden PDF.
- `make claim-scan` forbids "stdlib-only", "zero third-party",
  "byte-identical PDF" claims.

## Do not nag (decided)

Dotted settings stay. No plugin system. No `x/net/html` swap. No CGO
HarfBuzz. No pixel-diff merge gate. No `gofumpt`. No mutex on layout or
`pdf.Document`. No context-on-every-struct. No live network in `make test`.
No site-specific MediaWiki cascade hacks. No document-global Y shifts.
ARC-03 / ARC-04 are documented follow-up, not fresh defects. R-01…R-04
are risks, not bugs. CR-01…CR-08 and ARC-01/02/05 stay closed unless
source regressed.
