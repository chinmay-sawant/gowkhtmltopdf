# Current architecture rating — 7.9 / 10

The score is an explicit weighted assessment of the current source at commit
`37f5ee2`, not a headline copied from the prior leanness review.

| Area | Weight | Score | Evidence and deduction |
|---|---:|---:|---|
| Public API / CLI seam | 20% | 8.5 | `convert.Request` is useful, but parser mode is not enforced, image uses a PDF-owned union, and CLI adapters remain inside deep modules. |
| Dependency direction / module depth | 20% | 8.3 | `load`, `outline`, settings, layout, and PDF are genuine deep modules; engine-to-CLI and layout-to-PDF-shaped data remain leaks. |
| State ownership / context / concurrency | 20% | 7.6 | converter state is documented as non-concurrent, but snapshots are shallow and cancellation stops before layout/raster. |
| Logging / errors / output contracts | 15% | 8.0 | shared line severity and error wrapping are good; direct stdout and inconsistent nil-output semantics remain. |
| Fixtures / golden tests / benchmarks | 15% | 7.0 | broad structural golden coverage and passing gates exist; cross-mode parity, collision, context, and release benchmark evidence are incomplete. |
| Examples / documentation locality | 10% | 8.0 | contracts are documented in many places and current plans are detailed; some comments describe future or partial seams. |
| **Weighted total** | **100%** | **7.93 → 7.9** | `8.5×.20 + 8.3×.20 + 7.6×.20 + 8.0×.15 + 7.0×.15 + 8.0×.10 = 7.93` |

## What would move the score

- Phase 1 should raise seam separation by making mode, output, CLI-adapter, and
  snapshot invariants explicit.
- Phase 2 should raise locality by concentrating document/resource preparation.
- Phases 3–4 should raise correctness and leverage only after regression tests prove
  stable container, display-list, resource, and font behavior.

The existing 9.5/10 ponytail score is not substituted into this calculation: it
measures leanness, while this score measures architectural depth and seam quality.
