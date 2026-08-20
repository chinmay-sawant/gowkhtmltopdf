# v0.2.6 WOFF2 Decode + Opt-in Metric Aliases Plan

> **Status:** planned (ledger filed 2026-08-20)  
> **Parent:** [`plans/0.2.6/README.md`](../README.md)  
> **Predecessor:** closed [`plans/0.2.5/font/`](../../0.2.5/font/README.md)  
> **Owner:** `internal/pdf` (decode + resolver), `internal/convert/prepare`
> (`@font-face`), settings/CLI/Document surfaces  
> **Constraint:** pure Go; no CGO, browser embed, Fontconfig library, or
> second PDF writer. Direct modules today:
> `go-text/typesetting` + `tdewolff/canvas`. WOFF2 requires the dated
> Brotli amendment in this folder before any first-party import.

## Purpose

Ship two follow-ups that the v0.2.5 resolution track left out:

1. **WOFF2 → SFNT decode** so local and HTTPS `@font-face` `.woff2` sources
   register like WOFF1 when ACL/network allow.
2. **Opt-in metric-compatible family aliases** (Fontconfig
   `30-metric-aliases`-style accepts such as `Georgia → Gelasio`) when
   substitute faces are already in the registry, without changing the
   no-flag default.

## Evidence sources

| Source | Use |
|--------|-----|
| [`documentation/fonts.md`](../../../documentation/fonts.md) | Shipped operator contract; WeasyPrint divergence |
| [`documentation/deferred.md`](../../../documentation/deferred.md) | Current WOFF2 / alias inventory rows |
| [`plans/0.2.5/font/`](../../0.2.5/font/README.md) | Closed resolver + embed-preflight contract |
| [`09-remote-webfonts.md`](../../0.2.0/phases/pending-phase-items/09-remote-webfonts.md) | HTTPS WOFF1 shipped; WOFF2 decode open |
| [`2026-08-05-gotext-typesetting.md`](../../0.2.0/amendments/2026-08-05-gotext-typesetting.md) | Amendment shape for allowlist exceptions |
| `internal/pdf/woff.go`, `resolver.go`, `registry.go` | Decode + resolution seams |
| `internal/convert/prepare/styles.go`, `fontface_test.go` | `@font-face` fetch / WOFF2 skip tests |
| `go.mod` | `andybalholm/brotli` already **indirect** via canvas |

## Package map (touch points)

| Area | Primary paths | Notes |
|------|---------------|-------|
| WOFF1 / ParseFontBytes | `internal/pdf/woff.go` | `wOF2` → `errWOFF2Unsupported` today |
| WOFF2 (new) | `internal/pdf/woff2.go` (proposed) | Brotli + table reconstruct → SFNT |
| Allowlist gate | `internal/pdf/shape_test.go` `TestDirectModuleAllowlist` | Add brotli when amendment lands |
| `@font-face` fetch | `internal/convert/prepare/styles.go` | Suffix-skips `.woff2` before fetch |
| Resolution | `internal/pdf/resolver.go` | Alias step after exact registry miss |
| Settings / CLI | `settings`, `cli/flags`, `document.go` | New opt-in bool, default false |

## Part A: WOFF2 / Brotli

### Problem

CDN and wiki `@font-face` stacks list `.woff2` first. Operators who only have
WOFF2 fall through to Liberation even when FetchSub would allow the bytes.

WOFF2 is not “WOFF1 + Brotli”: after inflate there is a transformed table
stream (notably `glyf`/`loca`) that must become normal SFNT before
`ParseTTF`.

### Goals

- Decode valid WOFF2 → TrueType SFNT under WOFF1-class caps.
- Wire through existing `ParseFontBytes` → `MergeFontFaces` (PDF + image).
- Promote `github.com/andybalholm/brotli` to a **direct** allowlisted require
  via amendment (already in the module graph via canvas).
- Honest degrade for malformed / OTTO / `fvar` / collections / bombs.

### Non-goals (WOFF2)

- CGO Brotli; any fourth unrelated direct module.
- Variable fonts, CFF/`OTTO`, EOT, `data:` sources.
- WOFF2 collections (reject with diagnostic in v1).
- Encoding WOFF2; bundling CDN corpora; Chrome pixel parity claims.

### Design sketch

```text
@font-face url(..woff2)
  → prepare.fetchFontFace (FetchSub ACL/timeout/body)
  → pdf.ParseFontBytes
       ├─ wOFF  → DecodeWOFF
       ├─ wOF2  → DecodeWOFF2 (Brotli + reconstruct)  [new]
       └─ SFNT  → ParseTTF (reject OTTO / fvar)
  → Registry
```

Remove `.woff2` from the unsupported suffix list; keep `.eot` and `data:`.

## Part B: Opt-in metric aliases

### Problem

Default `Georgia, serif` → Liberation Serif is the **intentional** v0.2.5
contract. Operators who want WeasyPrint-like host substitutes need an
explicit knob when Gelasio/Cousine/etc. are already discoverable.

### Goals

- Curated accept map (Fontconfig metric-alias subset), consulted only when
  opt-in is on **and** the substitute family exists in the **Registry**.
- Exact name-table match still wins before aliases.
- No FaceSet / bundled Liberation coupling for alias targets in v1
  (flag alone with empty registry is a no-op for aliases).

### Non-goals (aliases)

- Silent Fontconfig import; parsing host `fonts.conf` in v1.
- Making discovery or aliases the no-flag default.
- Full Fontconfig accept/weak-alias graph; reverse aliases as a product goal.
- Claiming WeasyPrint pixel parity.

### Recommended design (Option A)

Hardcoded map inside `FontResolver` when `UseMetricFontAliases` is true:

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

### Resolution order (flag on)

1. Exact registry / `@font-face` / `--font-path` match for the CSS token.
2. **NEW:** If opt-in and token is in the map, try each accept family via
   exact registry lookup (weight/style unchanged).
3. Continue author stack.
4. Generics / Liberation names / `system-ui`.
5. Terminal Liberation Sans; glyph + embed-preflight paths unchanged.

## Phase map

| Phase | File | Outcome |
|-------|------|---------|
| 01 | [phase-01-allowlist-policy-and-threat-model.md](phase-01-allowlist-policy-and-threat-model.md) | Brotli amendment + threat-model wording |
| 02 | [phase-02-woff2-decode-and-caps.md](phase-02-woff2-decode-and-caps.md) | `DecodeWOFF2` + caps / unit tests |
| 03 | [phase-03-font-face-and-prepare-integration.md](phase-03-font-face-and-prepare-integration.md) | Local + HTTPS `@font-face` WOFF2 |
| 04 | [phase-04-metric-alias-contract.md](phase-04-metric-alias-contract.md) | Resolver alias step + unit tests |
| 05 | [phase-05-settings-cli-library-surface.md](phase-05-settings-cli-library-surface.md) | Flag / setting / Document field |
| 06 | [phase-06-corpus-regression-and-engine-compare.md](phase-06-corpus-regression-and-engine-compare.md) | Convert corpus + fixture-55 notes |
| 07 | [phase-07-docs-honesty-and-claim-scan.md](phase-07-docs-honesty-and-claim-scan.md) | Docs, deferred, KB, claim-scan |
| 08 | [phase-08-validation-and-closure.md](phase-08-validation-and-closure.md) | `make test` / `make lint` / build gates |

Amendment: [amendments/2026-08-20-woff2-brotli-allowlist.md](amendments/2026-08-20-woff2-brotli-allowlist.md).

## Definition of done

- [ ] Brotli is a direct allowlisted require; `TestDirectModuleAllowlist` and
  `AGENTS.md` agree; `CGO_ENABLED=0` throughout.
- [ ] Valid WOFF2 `@font-face` (local + HTTPS under ACL) embeds like WOFF1;
  unsupported/malformed cases warn and fall back.
- [ ] No-flag `Georgia, serif` still → Liberation Serif via `serif`.
- [ ] Flag on + Gelasio in registry → Georgia selects Gelasio; exact Georgia
  still wins; missing substitute continues the stack.
- [ ] Docs / deferred / pending-09 / KB agree; no new PDF/A or PDF/UA claims.
- [ ] Phase 08 records `make test` && `make lint` && `make build` evidence;
  track README status → complete.

## Risks

| Risk | Mitigation |
|------|------------|
| Third direct module / allowlist politics | Dated amendment; brotli already indirect via canvas; CI allowlist test |
| WOFF2 `glyf`/`loca` transform bugs | Spec tests; reject unknown transforms/collections; round-trip fixtures |
| Decompress bombs | Mirror WOFF1 caps + LimitReader on Brotli output |
| Operators think the alias flag loads fonts | Docs + empty-registry no-op; aliases never touch FaceSet |
| Accidental default-on | Zero-value false tests; no-flag fixture-55 Liberation path frozen |

## Out of scope / deferred elsewhere

- `data:` and EOT `@font-face`
- Host `fonts.conf` XML parser
- Custom operator-editable alias files
- WOFF2 encode; variable fonts; CFF OpenType
- Reopening `plans/0.2.5/font/` phases 01-08
