# Plan amendment: allow `tdewolff/font` for WOFF2 decode

> Date: 2026-08-20
> Amends: product allowlist in `AGENTS.md`, `TestDirectModuleAllowlist`
> Execution ledger: [`../00-canonical-woff2-metric-aliases-plan.md`](../00-canonical-woff2-metric-aliases-plan.md)

---

## Decision

**Exception (narrow):** the runtime may depend on one additional third-party
Go module for WOFF2 → SFNT decode:

| Allowed | Forbidden (still) |
|---------|-------------------|
| [`github.com/tdewolff/font`](https://github.com/tdewolff/font) | CGO / `google/brotli` cbrotli |
| Pure Go `ParseWOFF2` (uses `andybalholm/brotli` transitively) | Any other new direct `go.mod` require under this amendment |
| Already in the canvas module graph | Fontconfig, HarfBuzz, second PDF writer, browser embeds |

Existing directs remain: `go-text/typesetting`, `tdewolff/canvas`. After this
amendment, the allowlist is **three** modules (typesetting, canvas, font).
Brotli stays an **indirect** require via `tdewolff/font` (same graph canvas
already pulled).

## Why this exception

- Stdlib has no Brotli decoder; WOFF2 needs Brotli plus glyf/loca reconstruct.
- `tdewolff/font` is already an indirect dependency via `tdewolff/canvas`.
  Promoting it authorizes `ParseWOFF2` without inventing a second WOFF2 stack.
- Pure Go; keeps static `CGO_ENABLED=0` builds.

## Alternatives considered

| Option | Verdict |
|--------|---------|
| **A. Promote `tdewolff/font` (chosen)** | Adopt for WOFF2 decode only |
| **B. Promote only `andybalholm/brotli` + in-tree reconstruct** | Rejected near-term (large glyf/loca surface; font already has it) |
| **C. CGO brotli** | Rejected (no-cgo) |

## Acceptance

- [x] `go.mod` lists `github.com/tdewolff/font` as a direct require
- [x] `TestDirectModuleAllowlist` allows typesetting + canvas + font
- [x] `AGENTS.md` allowlist sentence updated
- [x] `CGO_ENABLED=0` retained
- [x] No fourth unrelated direct module under this amendment
