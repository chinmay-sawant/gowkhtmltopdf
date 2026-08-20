# Plans: v0.2.6 (WOFF2 + opt-in metric aliases)

| File / Folder | Role |
|---------------|------|
| [woff2-metric-aliases/](woff2-metric-aliases/README.md) | Track: WOFF2/Brotli decode + opt-in Fontconfig-style metric aliases |
| [woff2-metric-aliases/00-canonical-woff2-metric-aliases-plan.md](woff2-metric-aliases/00-canonical-woff2-metric-aliases-plan.md) | Contract, non-goals, phase map, definition of done |
| [woff2-metric-aliases/amendments/](woff2-metric-aliases/amendments/2026-08-20-woff2-brotli-allowlist.md) | Narrow allowlist exception for `andybalholm/brotli` |

Predecessor: closed [`plans/0.2.5/font/`](./0.2.5/font/README.md) (resolution,
fallback, embed preflight; phases 01-08). This release does not reopen that
ledger.

Product framing: [`documentation/fonts.md`](../documentation/fonts.md),
[`documentation/deferred.md`](../documentation/deferred.md),
[`knowledge-base/wiki/syntheses/fonts-and-typography.md`](../knowledge-base/wiki/syntheses/fonts-and-typography.md).

## Scope in one line

Decode WOFF2 `@font-face` sources under the existing ACL and size caps, and
add an opt-in metric-compatible family alias map (Georgia→Gelasio style) that
never becomes the no-flag default.

## Status

Planned (ledger filed 2026-08-20). Implementation checklists start unchecked.
Mark `[x]` only with command or artifact evidence in the phase file.
`VERSION` remains `0.2.4` until 0.2.5 release prep. This track targets the
minor after font-resolution ships.
