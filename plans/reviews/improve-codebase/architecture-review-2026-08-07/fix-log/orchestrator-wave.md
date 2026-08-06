# Orchestrator wave — 2026-08-07 (medium effort, 4 agents)

Adhered to `fix-contract.md`: **no git commands**, package ownership, FIX-REVIEW for deferred rows, fix logs under this directory.

## Agents

| Agent | Result |
|---|---|
| fix-convert (primary) | convert package green; MUST + most SHOULD rows |
| fix-convert-deep (secondary) | P1-1 Request, pagePlan, CollectSheets, paintCount partial |
| fix-layout | P4-01..04/06 done; P4-05/07 partial; P3-01/03/04/05; P5-01/07; DeactivateOp |
| fix-imageout-wave2 | P2-07/P1-4/P2-08/09; CollectSheets consume; RunRequest; line.Emit; atlas bound |

## Validation (orchestrator)

```
go build ./...   # exit 0
go vet ./...     # exit 0
go test ./... -count=1   # all packages ok
```

No commits made.

## Still open (next wave)

- **P1-1 remainder / P1-8 / P6-03 mains** — api.go + cmd/* drop `cli.Command` / use Request + cli.ExitCode (fix-engine-migration-wave2)
- **P2-04 full wiring** — remaining library/CLI paths for InlineHTML if any
- **P2-10** — objectState slim
- **P2-12 remainder** — paintCount error surface
- **P4-05/P4-07** — full text-wrap unify; deferred chrome
- **P5-01 imageout** — raster consume StyleOf/FakeBoldFor
- **P5-02** — full shared page-assembly pipe
- **P5-05** — per-Render atlas ownership
- Residual FIX-REVIEW: css ParseColor var path; default-encoding ignored

## Phase checklists

Updated in `phases/phase-0[1-6]-*.md` to match landed work.
