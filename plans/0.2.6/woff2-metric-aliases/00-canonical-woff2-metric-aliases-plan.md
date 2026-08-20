# v0.2.6 WOFF2 Decode + Opt-in Metric Aliases Plan

> **Status:** complete — validated 2026-08-20
> **Parent:** [`plans/0.2.6/README.md`](../README.md)
> **Predecessor:** closed [`plans/0.2.5/font/`](../../0.2.5/font/README.md)
> **Owner:** `internal/pdf` (decode + resolver), `internal/convert/prepare`
> (`@font-face`), settings/CLI/Document surfaces
> **Constraint:** pure Go; no CGO, browser embed, Fontconfig library, or
> second PDF writer. Direct modules after this track:
> `go-text/typesetting`, `tdewolff/canvas`, and `tdewolff/font` (WOFF2).
> Amendment: [amendments/2026-08-20-woff2-brotli-allowlist.md](amendments/2026-08-20-woff2-brotli-allowlist.md).

## Purpose

Ship two follow-ups that the v0.2.5 resolution track left out:

1. **WOFF2 → SFNT decode** so local and HTTPS `@font-face` `.woff2` sources
   register like WOFF1 when ACL/network allow.
2. **Opt-in metric-compatible family aliases** (Fontconfig
   `30-metric-aliases`-style accepts such as `Georgia → Gelasio`) when
   substitute faces are already in the registry, without changing the
   no-flag default.

## What shipped (summary)

| Item | Outcome |
|------|---------|
| WOFF2 decode | `pdf.DecodeWOFF2` via `tdewolff/font.ParseWOFF2`; wired in `ParseFontBytes` |
| `@font-face` | `.woff2` no longer suffix-skipped; `.eot` / `data:` still skipped |
| Allowlist | Third direct: `github.com/tdewolff/font` (Brotli stays **indirect** via font) |
| Metric aliases | `FontResolver.UseMetricFontAliases` + curated Registry-only accept map |
| Operator surface | CLI `--use-metric-font-aliases`, dotted `usemetricfontalias`, library `UseMetricFontAliases` (default **false**) |
| Default path | Unchanged: no-flag `Georgia, serif` → Liberation Serif via `serif` |

## Evidence sources

| Source | Use |
|--------|-----|
| [`documentation/fonts.md`](../../../documentation/fonts.md) | Shipped operator contract |
| [`documentation/deferred.md`](../../../documentation/deferred.md) | WOFF2 shipped; aliases opt-in |
| [`plans/0.2.5/font/`](../../0.2.5/font/README.md) | Closed resolver + embed-preflight contract |
| [`09-remote-webfonts.md`](../../0.2.0/phases/pending-phase-items/09-remote-webfonts.md) | Remaining WOFF2 rows superseded |
| `internal/pdf/woff2.go`, `woff.go`, `resolver.go` | Decode + alias seams |
| `internal/convert/prepare/styles.go`, `fontface_test.go` | `@font-face` WOFF2 embed / bad-skip |
| `go.mod` / `TestDirectModuleAllowlist` | Three directs: typesetting, canvas, font |
| `testdata/fonts/LiberationSans-Regular.woff2` | Round-trip / `@font-face` fixture |

## Package map (shipped)

| Area | Primary paths | Notes |
|------|---------------|-------|
| WOFF1 / WOFF2 / ParseFontBytes | `internal/pdf/woff.go`, `woff2.go` | `wOF2` → `DecodeWOFF2` → SFNT → `ParseTTF` |
| Allowlist gate | `internal/pdf/shape_test.go` | Allows typesetting + canvas + font |
| `@font-face` fetch | `internal/convert/prepare/styles.go` | Skips `.eot` / `data:` only |
| Resolution | `internal/pdf/resolver.go` | Alias step after exact registry miss when flag on |
| Settings / CLI / library | `settings`, `cli/flags`, `document.go` | `UseMetricFontAliases` default false |
| Layout / convert / imageout | `layout.Options`, `convert`, `imageout` | Flag plumbed into `FontResolver` |

## Part A: WOFF2

### Design (shipped)

```text
@font-face url(..woff2)
  → prepare.fetchFontFace (FetchSub ACL/timeout/body)
  → pdf.ParseFontBytes
       ├─ wOFF  → DecodeWOFF
       ├─ wOF2  → DecodeWOFF2 (tdewolff/font.ParseWOFF2)  [shipped]
       └─ SFNT  → ParseTTF (reject OTTO / fvar)
  → Registry
```

Allowlist choice vs the original “promote brotli only” sketch: promoting
`tdewolff/font` reuses a working `ParseWOFF2` (glyf/loca reconstruct) already
in the canvas graph. `andybalholm/brotli` remains indirect via font. See the
dated amendment in this folder.

### Non-goals (unchanged)

- CGO Brotli; fourth unrelated direct module
- Variable fonts, CFF/`OTTO`, EOT, `data:` sources
- WOFF2 collections (rejected)
- Encoding WOFF2; CDN corpora; Chrome pixel parity claims

## Part B: Opt-in metric aliases

### Design (shipped)

Hardcoded map inside `FontResolver` when `UseMetricFontAliases` is true.
Targets resolve against **Registry only**, never bundled `FaceSet`.

| CSS token (miss) | Try registry families (order) |
|------------------|-------------------------------|
| `georgia` | `Gelasio` |
| `courier new` | `Cousine` |
| `times new roman` | `Tinos` |
| `arial` | `Arimo` |
| `cambria` | `Caladea` |
| `calibri` | `Carlito` |

### Operator surface (defaults off)

| Surface | Name | Default |
|---------|------|---------|
| CLI | `--use-metric-font-aliases` | false |
| Dotted setting | `usemetricfontalias` | false |
| Library | `Document` / `ImageDocument` `.UseMetricFontAliases` | false |

Discovery stays separate: `--font-path` / `--use-system-fonts` / `@font-face`
must supply the substitute face. The alias flag alone does not scan disk.

### Resolution order (flag on)

1. Exact registry / `@font-face` / `--font-path` match for the CSS token.
2. If opt-in and token is in the map, try each accept family via exact
   registry lookup (weight/style unchanged).
3. Continue author stack.
4. Generics / Liberation names / `system-ui`.
5. Terminal Liberation Sans; glyph + embed-preflight paths unchanged.

## Phase map

| Phase | File | Outcome |
|-------|------|---------|
| 01 | [phase-01-allowlist-policy-and-threat-model.md](phase-01-allowlist-policy-and-threat-model.md) | Allowlist + threat-model wording |
| 02 | [phase-02-woff2-decode-and-caps.md](phase-02-woff2-decode-and-caps.md) | `DecodeWOFF2` + unit tests |
| 03 | [phase-03-font-face-and-prepare-integration.md](phase-03-font-face-and-prepare-integration.md) | Local + HTTPS `@font-face` WOFF2 |
| 04 | [phase-04-metric-alias-contract.md](phase-04-metric-alias-contract.md) | Resolver alias step + unit tests |
| 05 | [phase-05-settings-cli-library-surface.md](phase-05-settings-cli-library-surface.md) | Flag / setting / Document field |
| 06 | [phase-06-corpus-regression-and-engine-compare.md](phase-06-corpus-regression-and-engine-compare.md) | Convert corpus + fixture notes |
| 07 | [phase-07-docs-honesty-and-claim-scan.md](phase-07-docs-honesty-and-claim-scan.md) | Docs, deferred, honesty |
| 08 | [phase-08-validation-and-closure.md](phase-08-validation-and-closure.md) | `make test` / `make lint` / build |

## Definition of done

- [x] Allowlist updated: `TestDirectModuleAllowlist` and `AGENTS.md` agree on
  typesetting + canvas + `tdewolff/font`; `CGO_ENABLED=0` throughout.
- [x] Valid WOFF2 `@font-face` embeds like WOFF1; malformed cases warn and
  fall back (`TestDecodeWOFF2RoundTrip`, `TestFontFaceWOFF2Embed`,
  `TestFontFaceBadWOFF2Skipped`).
- [x] No-flag `Georgia, serif` still → Liberation Serif via `serif`
  (`TestFontResolverNoLegacyAliases`, alias-off cases).
- [x] Flag on + Gelasio in registry → Georgia selects Gelasio; exact Georgia
  still wins; missing substitute continues the stack
  (`TestFontResolverMetricAliasesOptIn`).
- [x] Docs / deferred / pending-09 agree; no new PDF/A or PDF/UA claims.
- [x] Phase 08 records `make test` && `make lint` evidence; track README
  status → complete.

## Validation outcomes (2026-08-20)

```
$ CGO_ENABLED=0 make test
go test ./...
(ok — all packages)

$ CGO_ENABLED=0 make lint
golangci-lint run ./...
(exit 0)

$ CGO_ENABLED=0 go build -o /tmp/gowkhtmltopdf-check ./cmd/gowkhtmltopdf
$ CGO_ENABLED=0 go build -o /tmp/gowkhtmltoimage-check ./cmd/gowkhtmltoimage
(BUILD_OK)
```

Focused proofs: `go test ./internal/pdf ./internal/convert ./internal/cli
./internal/layout ./internal/imageout` covering DecodeWOFF2, `@font-face`
WOFF2 embed, metric-alias resolver cases, and CLI flag parse.

## Risks (post-ship)

| Risk | Mitigation |
|------|------------|
| Third direct module | Dated amendment; font already in canvas graph; CI allowlist test |
| WOFF2 reconstruct bugs | Use maintained `tdewolff/font.ParseWOFF2`; fixture + embed tests |
| Operators think the alias flag loads fonts | Docs: discovery separate; empty-registry alias no-op |
| Accidental default-on | Zero-value false; alias-off tests |

## Out of scope / deferred elsewhere

- `data:` and EOT `@font-face`
- Host `fonts.conf` XML parser
- Custom operator-editable alias files
- WOFF2 encode; variable fonts; CFF OpenType
- Reopening `plans/0.2.5/font/` phases 01-08
