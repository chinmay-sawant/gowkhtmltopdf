# 0.2.6 Ponytail audit - synthesis

> **Canonical ledger:** [0.2.6-ponytail-audit.md](0.2.6-ponytail-audit.md)
> **Status:** review complete. Implementation is not started.
> **Review date:** 2026-09-12

## Result

Four read-only review packets found 46 current-tree candidates. This audit activates 40 of them because the requested working set is 30-40 items. The six remaining leads need an explicit performance or encoded-output decision first, so they are not checklist rows.

| Area | Active findings | Packet |
|---|---:|---|
| Public boundaries | 12 | [01-public-boundaries.md](01-public-boundaries.md) |
| CSS matcher and parser | 5 | [02-css-convert.md](02-css-convert.md) |
| Conversion pipeline | 7 | [02-css-convert.md](02-css-convert.md) |
| Layout | 12 | [03-layout.md](03-layout.md) |
| PDF and image output | 4 | [04-output.md](04-output.md) |
| Total | 40 | [0.2.6-ponytail-audit.md](0.2.6-ponytail-audit.md) |

The strongest deletion is the unused `ParallelLayout` experiment at `internal/layout/parallel.go:17-454`. It has only a test caller. The most repeated pattern is state that reaches parsing, inheritance, and generated style interning but has no layout or paint reader. That work must retain the fields with current readers, such as `MarginTrim` and `TextDecorationSkipInk`.

## Review rules

- The scan is limited to complexity. A possible bug, security concern, or slow path is not a Ponytail finding.
- A row needs a live source location, a caller check, a smaller replacement, and a focused proof command.
- Current `ponytail:` comments are deliberate ceilings. They were screened out unless the code no longer matches the comment.
- Completed rows in the August Ponytail ledger were not copied forward.

## Qualified output leads

`04-output.md` preserves six useful leads outside the active ledger: fast-PNG canvas encoding, supersample-buffer caching, parallel Flate compression, external-font caching, the raster paint-order facade, and a rune cache-key reducer. Each can change speed, retained memory, or encoded bytes. They need a measured decision before becoming a normal cleanup row.

## Evidence and validation

The packet reports cite current files and exact line ranges. No source changed during this audit. Under the phase-wise checklist rules, documentation-only work does not run `make test` or `make lint`; those commands are required when an implementation row is closed.

net: about -3,500 source and test lines possible, -0 dependencies possible.
