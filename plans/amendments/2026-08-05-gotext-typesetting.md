# Plan amendment: allow `go-text/typesetting` for complex-script shaping

> Date: 2026-08-05  
> Branch: `feature/tier-2-pending-2`  
> Amends: [`2026-08-04-shaping-stdlib.md`](2026-08-04-shaping-stdlib.md),  
> `plans/phases/phase-19-fonts-i18n.md`, product constraint in  
> `plans/10-canonical-post-mvp-roadmap.md`  
> Execution ledger: [`plans/phases/subplans-tier-2/shaping-gotext-typesetting.md`](../phases/subplans-tier-2/shaping-gotext-typesetting.md)

---

## Decision

**Exception (narrow):** the runtime may depend on **exactly one** third-party Go
module for OpenType shaping:

| Allowed | Forbidden (still) |
|---------|-------------------|
| [`github.com/go-text/typesetting`](https://github.com/go-text/typesetting) | CGO / HarfBuzz C library |
| Pure Go, `CGO_ENABLED=0` | Any other `go.mod` require (PDF, HTML, CSS, Brotli, …) |
| | Chrome / WebKit / Playwright embeds |

All other product code remains **stdlib + in-tree**. This does **not** open the
door to a general dependency free-for-all.

## Why this exception

In-tree presentation-form Arabic + NFC Indic is honest but not production
OpenType. Full GSUB/GPOS in-tree is HarfBuzz-sized. `go-text/typesetting` is
pure Go (used by Fyne/Gio/Ebitengine), provides real shaping, and keeps
`CGO_ENABLED=0`.

## Alternatives considered (do **not** implement unless this amendment is revised)

| Option | Pros | Cons | Verdict |
|--------|------|------|---------|
| **A. `go-text/typesetting` (chosen)** | Pure Go; GSUB/GPOS; active; no cgo | First `go.mod` dep; API still evolving (v0) | **Adopt for shaping only** |
| **B. In-tree GSUB/GPOS port** | Zero deps | Multi-month–year; high defect risk | Rejected for near-term |
| **C. `boxesandglue/textshape`** | HarfBuzz port; PDF-oriented | Another module; overlapping with A | Keep as fallback if A fails evaluation |
| **D. `KarpelesLab/gofreetype` shape** | Broad font stack | Larger surface; not chosen | Document only |
| **E. CGO HarfBuzz** | Gold standard | Violates no-cgo / static builds | Rejected |

## What this does **not** change

- **`--font-path` / system fonts / local TTF `@font-face`** remain the way to
  supply faces (including CJK). No full Noto CJK bundle in the default binary.
- **WOFF/WOFF2 decode** is orthogonal (see shaping subplan § WOFF clarification).
  Allowing typesetting does **not** by itself require WOFF2 or Brotli.
- Presentation-form Arabic path may remain as fallback until typesetting is wired
  and proven.

## Acceptance (when the shaping subplan lands)

- [x] `go.mod` lists **only** `github.com/go-text/typesetting` (and its
      transitive module graph as resolved by Go) beyond the main module —
      document the allowlist in CI (`TestDirectModuleAllowlist`)
- [x] `CGO_ENABLED=0` still green
- [x] Docs honesty: “OpenType shaping via go-text/typesetting”; drop “no OT”
  claims once shipped
- [x] No other third-party PDF/HTML/CSS libraries added under this amendment

## Supersedes

[`2026-08-04-shaping-stdlib.md`](2026-08-04-shaping-stdlib.md) decision “no
third-party Go module for shaping” is **superseded** for shaping only. That
file’s Arabic/Hangul interim mechanisms remain valid until typesetting wiring
ships.
