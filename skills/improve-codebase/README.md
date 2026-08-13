# Improve-codebase skill pack

Architecture deepening, extension-seam honesty, and Go practices.
Findings become one phase-wise checklist via
`skills/phase-wise-checklist/SKILLS.md`.

This pack **reviews**. It does not implement, commit, or re-score a
ledger that is already canonical unless the user asks.

## Skills

| Skill | Slash | Question it answers |
|---|---|---|
| [SKILL.md](SKILL.md) | `/improve-codebase` | Run all three lenses and write the ledger |
| [architecture-deepening](architecture-deepening/SKILL.md) | `/improve-codebase-architecture` | Where is the module shallow or the DAG leaky? |
| [extension-seams](extension-seams/SKILL.md) | `/improve-codebase-extension` | Where does the next honest feature fork? |
| [go-practices](go-practices/SKILL.md) | `/improve-codebase-practices` | Where do errors, context, tests, or ownership lie? |

Shared finding shape: [references/finding-schema.md](references/finding-schema.md).
When the target is this repo: [references/gowkhtmltopdf.md](references/gowkhtmltopdf.md).

## Relation to other skills

| Skill | Owns |
|---|---|
| `codebase-design` | Vocabulary: module, interface, seam, adapter, depth |
| `phase-wise-checklist` | Ledger shape, statuses, evidence, `make lint`/`make test` gates |
| `ponytail` / `ponytail-review` | What to **delete** |
| `critical-go-review` | 5-agent API/memory/devil's-advocate wave |
| `perf-review` | Allocation / hot-path squeeze |
| `debug-html-template` | One visual PDF symptom |

Run ponytail **before** deepening so you do not solidify a stub surface.
Do not launch critical-go-review or perf-review from this pack.

## Output home

`plans/reviews/improve-codebase/<slug>-<YYYY-MM-DD>/phase-wise-checklist.md`

One new dated folder per wave. Never edit a closed historical ledger
except to mark a moved row `[~]` with a pointer.
