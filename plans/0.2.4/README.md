# Plans — v0.2.4 (Idiomatic Document API + CLI rethink)

| File / Folder | Role |
|---------------|------|
| [31-canonical-0.2.4-roadmap.md](31-canonical-0.2.4-roadmap.md) | **Canonical v0.2.4 execution ledger** (phases 31–38) |
| [phases/](phases) | Per-phase atomic checklists |

Workflow: [../../skills/phase-wise-checklist/SKILLS.md](../../skills/phase-wise-checklist/SKILLS.md)

Predecessor: [../0.2.3/README.md](../0.2.3/README.md) (module path / `go install`);
engine contracts from [../0.2.1/24-canonical-0.2.1-roadmap.md](../0.2.1/24-canonical-0.2.1-roadmap.md).

Product framing: [../../documentation/overview.md](../../documentation/overview.md),
[../../documentation/library-api.md](../../documentation/library-api.md),
[../../documentation/cli.md](../../documentation/cli.md).

## Scope in one line

Hard-break the public Go API from wkhtml-style dotted `Set`/`Converter` to a
struct-based **Document** model, and redesign the CLI to match. Engine layout
and PDF writer stay; only the outer contract changes.
