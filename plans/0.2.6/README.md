# Plans - v0.2.6 (CSS coverage)

| File / Folder | Role |
|---------------|------|
| [48-canonical-0.2.6-css-coverage.md](48-canonical-0.2.6-css-coverage.md) | Canonical v0.2.6 execution ledger, phases 48-56 |
| [phases/](phases) | Per-phase atomic checklists |
| [catalog/](catalog) | Frozen CSS catalogs plus `mapping.json` vs current engine |
| [AGENTS.md](AGENTS.md) | Agent rules for this ledger |

Workflow: [`../../skills/phase-wise-checklist/SKILLS.md`](../../skills/phase-wise-checklist/SKILLS.md)

Predecessor: [`../0.2.5/40-canonical-0.2.5-python-bindings.md`](../0.2.5/40-canonical-0.2.5-python-bindings.md) (python bindings, complete 2026-08-26).

WOFF2 sidecar: KB cites `plans/0.2.6/woff2-metric-aliases/` as complete 2026-08-20. That directory is not in this worktree. `internal/pdf/woff.go` still rejects WOFF2. This CSS ledger does not re-open WOFF2 unless an amendment says so. See canonical Out of scope.

## Scope in one line

Catalog-driven print CSS coverage: download the spec property list, map it onto what `internal/css` and `internal/layout` actually do, then raise how often authored templates hit Implemented, without claiming browser print.

## Verification

- `make test`, `make lint`, `make claim-scan`, `make golden`, `make build` green 2026-08-27 on `feature/026-extended-css-support`
- Mapping counts: `catalog/coverage-summary.json`
- VERSION still 0.2.5. Leftovers are `[~]` in the canonical ledger (`page: ident`, `@page` margin boxes, fixture-gated flex/float depth)
