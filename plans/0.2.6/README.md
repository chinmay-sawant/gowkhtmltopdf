# Plans - v0.2.6 (CSS coverage)

| File / Folder | Role |
|---------------|------|
| [48-canonical-0.2.6-css-coverage.md](48-canonical-0.2.6-css-coverage.md) | Canonical v0.2.6 execution ledger, phases 48-78 |
| [phases/](phases) | Per-phase atomic checklists |
| [catalog/](catalog) | Frozen CSS catalogs plus `mapping.json` vs current engine |
| [property-counts.md](property-counts.md) | Property counts (implemented / partial / unsupported / ignored) |
| [ignored-inventory.json](ignored-inventory.json) | Phase 68-78 ownership of all 247 Ignored names |
| [phases/phase-57-partial-to-implemented-catchup.md](phases/phase-57-partial-to-implemented-catchup.md) | Phase 57-67: Partial to Implemented (closed) |
| [phases/phase-68-ignored-inventory-policy.md](phases/phase-68-ignored-inventory-policy.md) | Phase 68-78: Ignored to browser-level print |
| [AGENTS.md](AGENTS.md) | Agent rules for this ledger |
| [HONESTY-GATES.md](HONESTY-GATES.md) | Anti catalog-only close rules + flip packet |
| [implemented-honesty-pass.md](implemented-honesty-pass.md) | 2026-08-28 audit of Implemented rows (152 kept / 14 demoted) |
| [review/](review) | Post-ship architecture + ponytail ledger for commit `48e06dbc` |
| [agy-review/](agy-review) | Go Design Patterns and Go Code Style review ledgers |

Workflow: [`../../skills/phase-wise-checklist/SKILLS.md`](../../skills/phase-wise-checklist/SKILLS.md)

Predecessor: [`../0.2.5/40-canonical-0.2.5-python-bindings.md`](../0.2.5/40-canonical-0.2.5-python-bindings.md) (python bindings, complete 2026-08-26).

WOFF2 sidecar: KB cites `plans/0.2.6/woff2-metric-aliases/` as complete 2026-08-20. That directory is not in this worktree. `internal/pdf/woff.go` still rejects WOFF2. This CSS ledger does not re-open WOFF2 unless an amendment says so. See canonical Out of scope.

## Scope in one line

Catalog-driven CSS coverage: map the WebRef property list onto the engine, finish Partials (phases 57-67, done), then reopen Ignored for browser-level print (phases 68-78).

## Verification

- `make test`, `make lint`, `make claim-scan`, `make golden`, `make build` green 2026-08-27 on `feature/026-extended-css-support`
- Mapping counts: `catalog/coverage-summary.json`
- VERSION still 0.2.5. Leftovers are `[~]` in the canonical ledger (fixture-gated flex/float depth). Phase 54 `page: ident` and `@page` margin boxes shipped lite (see phase-54).
