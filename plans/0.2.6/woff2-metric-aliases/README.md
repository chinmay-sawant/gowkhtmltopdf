# v0.2.6: WOFF2 decode + opt-in metric aliases

> **Status:** complete — validated 2026-08-20
> **Parent:** [`plans/0.2.6/README.md`](../README.md)
> **Predecessor (closed):** [`plans/0.2.5/font/`](../../0.2.5/font/README.md)
> **Supersedes (remaining work):**
> [`plans/0.2.0/phases/pending-phase-items/09-remote-webfonts.md`](../../0.2.0/phases/pending-phase-items/09-remote-webfonts.md)
> for unfinished WOFF2/Brotli decode only
> **Constraint:** pure Go, `CGO_ENABLED=0`. Third direct module:
> `tdewolff/font` (see dated amendment in this folder).

This track closed two gaps left after v0.2.5 font-resolution:

1. **WOFF2.** Local and HTTPS `@font-face` `.woff2` decode and register like
   WOFF1 when ACL/network allow (`DecodeWOFF2` via `tdewolff/font`).
2. **Opt-in metric aliases.** `--use-metric-font-aliases` /
   `UseMetricFontAliases` (default false) can map Georgia→Gelasio-style
   accepts when the substitute is already in the registry. No-flag default
   stays author stack → generics → Liberation.

## Rules that do not move

| Rule | Meaning |
|------|---------|
| Default unchanged | No-flag `Georgia, serif` → Liberation Serif via `serif` |
| Aliases opt-in only | `--use-metric-font-aliases` / `UseMetricFontAliases` default false |
| Exact match wins | Real Georgia or `@font-face Georgia` beats any alias |
| No Fontconfig library | Curated pure-Go accept map. No cgo. No host `fonts.conf` parser in v1 |
| No reopen of 0.2.5 | Phases 01-08 in `plans/0.2.5/font/` stay closed |
| Profiles honest | No new PDF/A or PDF/UA claims from this track |

## Start here

- [00-canonical-woff2-metric-aliases-plan.md](00-canonical-woff2-metric-aliases-plan.md)
- [amendments/2026-08-20-woff2-brotli-allowlist.md](amendments/2026-08-20-woff2-brotli-allowlist.md)
- [phase-01-allowlist-policy-and-threat-model.md](phase-01-allowlist-policy-and-threat-model.md)
- [phase-02-woff2-decode-and-caps.md](phase-02-woff2-decode-and-caps.md)
- [phase-03-font-face-and-prepare-integration.md](phase-03-font-face-and-prepare-integration.md)
- [phase-04-metric-alias-contract.md](phase-04-metric-alias-contract.md)
- [phase-05-settings-cli-library-surface.md](phase-05-settings-cli-library-surface.md)
- [phase-06-corpus-regression-and-engine-compare.md](phase-06-corpus-regression-and-engine-compare.md)
- [phase-07-docs-honesty-and-claim-scan.md](phase-07-docs-honesty-and-claim-scan.md)
- [phase-08-validation-and-closure.md](phase-08-validation-and-closure.md)

## Related work

| Item | Location | Relationship |
|------|----------|--------------|
| Closed font resolution | [`plans/0.2.5/font/`](../../0.2.5/font/README.md) | Contract this track builds on |
| HTTPS WOFF1 / TTF `@font-face` | already shipped | Reuse `FetchSub` + `MergeFontFaces` |
| Pending-09 remote webfonts | [`09-remote-webfonts.md`](../../0.2.0/phases/pending-phase-items/09-remote-webfonts.md) | Remaining WOFF2 rows superseded here |
| Operator docs | [`documentation/fonts.md`](../../../documentation/fonts.md) | Update when phases ship |
| Deferred inventory | [`documentation/deferred.md`](../../../documentation/deferred.md) | Next-gate rows point here |

Creating this README is not implementation evidence. Mark phase rows `[x]`
only after recorded command outcomes or artifacts.
