# v0.2.6: WOFF2 decode + opt-in metric aliases

> **Status:** planned (ledger filed 2026-08-20)
> **Parent:** [`plans/0.2.6/README.md`](../README.md)
> **Predecessor (closed):** [`plans/0.2.5/font/`](../../0.2.5/font/README.md)
> **Supersedes (remaining work):**
> [`plans/0.2.0/phases/pending-phase-items/09-remote-webfonts.md`](../../0.2.0/phases/pending-phase-items/09-remote-webfonts.md)
> for unfinished WOFF2/Brotli decode only
> **Constraint:** pure Go, `CGO_ENABLED=0`. A third direct module is allowed
> only via the dated Brotli amendment in this folder.

Two gaps remain after the v0.2.5 font-resolution track:

1. **WOFF2.** Modern `@font-face` stacks serve `.woff2` first. Today the
   engine skips them (`errWOFF2Unsupported` or a suffix short-circuit).
   HTTPS TTF/OTF/WOFF1 already work.
2. **Opt-in metric aliases.** WeasyPrint/Fontconfig may map `Georgia` to
   `Gelasio` when Gelasio is installed. Gowkhtmltopdf’s default must stay
   author stack → generics → Liberation. Operators who want the WeasyPrint-like
   path need an explicit flag, not a silent default change.

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
