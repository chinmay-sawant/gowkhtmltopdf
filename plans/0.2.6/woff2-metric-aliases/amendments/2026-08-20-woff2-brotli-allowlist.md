# Plan amendment: allow `andybalholm/brotli` for WOFF2 decode

> Date: 2026-08-20  
> Branch: `feature/0.2.5-font-resolution` (plan filed; implement on follow-up)  
> Amends: product allowlist in `AGENTS.md`, `TestDirectModuleAllowlist`,  
> shaping amendment’s “no Brotli” wording for WOFF2 only  
> Execution ledger: [`./00-canonical-woff2-metric-aliases-plan.md`](./00-canonical-woff2-metric-aliases-plan.md)

---

## Decision

**Exception (narrow):** the runtime may depend on **exactly one additional**
third-party Go module for WOFF2 decompression:

| Allowed | Forbidden (still) |
|---------|-------------------|
| [`github.com/andybalholm/brotli`](https://github.com/andybalholm/brotli) | CGO / `google/brotli` cbrotli |
| Pure Go, `CGO_ENABLED=0` | Any other new direct `go.mod` require under this amendment |
| Direct require of the version already in the canvas graph | Fontconfig, HarfBuzz, second PDF writer, browser embeds |

Existing directs remain: `go-text/typesetting`, `tdewolff/canvas`. After this
amendment lands in code, the allowlist is **three** modules.

All other product code remains **stdlib + in-tree + the three directs**.
This does **not** open a general dependency free-for-all.

## Why this exception

- Stdlib has no Brotli decoder; WOFF2 requires one.
- `andybalholm/brotli` is already an **indirect** dependency via
  `tdewolff/canvas` (`go.mod`). Promoting it to direct does not expand the
  typical downloaded graph; it only authorizes a first-party import.
- Pure Go; keeps static `CGO_ENABLED=0` builds.

## Alternatives considered (do **not** implement unless this amendment is revised)

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. Promote `andybalholm/brotli` (chosen)** | Already in graph; pure Go; maintained | Third direct module | **Adopt for WOFF2 only** |
| **B. In-tree Brotli port** | Zero new directs | Large surface; maintenance burden | Rejected near-term |
| **C. CGO `google/brotli`** | Reference impl | Violates no-cgo | Rejected |
| **D. Require operators to pre-decode to TTF/WOFF1** | No dep change | Breaks CDN `@font-face` stacks | Rejected as product default |
| **E. Rely on canvas’s transitive import without promoting** | No go.mod change | Blank import / allowlist test forbids; unclear ownership | Rejected |

## What this does **not** change

- Metric aliases remain a separate pure-Go feature; they need **no** new module.
- Variable fonts (`fvar`), CFF/`OTTO`, EOT, and `data:` `@font-face` stay rejected.
- PDF/A and PDF/UA profiles stay opt-in and unchanged.
- Discovery (`--font-path` / `--use-system-fonts`) stays opt-in.
- The closed v0.2.5 resolution contract (no-flag Liberation via stack) stays.

## Acceptance (when Phase 01-02 land in code)

- [ ] `go.mod` lists `github.com/andybalholm/brotli` as a **direct** require
- [ ] `TestDirectModuleAllowlist` allows exactly: typesetting, canvas, brotli
- [ ] `AGENTS.md` allowlist sentence updated
- [ ] `CGO_ENABLED=0` still green
- [ ] No other third-party direct modules added under this amendment
- [ ] Docs: “WOFF2 via andybalholm/brotli”; stop saying “Brotli not allowlisted”

## Supersedes

- Remaining open WOFF2/Brotli rows in
  [`09-remote-webfonts.md`](../../../0.2.0/phases/pending-phase-items/09-remote-webfonts.md)
  move execution to `plans/0.2.6/woff2-metric-aliases/`.
- Shaping amendment
  [`2026-08-05-gotext-typesetting.md`](../../../0.2.0/amendments/2026-08-05-gotext-typesetting.md)
  stated allowing typesetting does not by itself require WOFF2/Brotli. That
  remains true historically; **this** amendment is the separate WOFF2 exception.
