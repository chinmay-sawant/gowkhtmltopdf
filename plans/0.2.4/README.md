# Plans — v0.2.4 (Idiomatic Document API + CLI rethink) — Complete

| File / Folder | Role |
|---------------|------|
| [31-canonical-0.2.4-roadmap.md](31-canonical-0.2.4-roadmap.md) | **Canonical v0.2.4 execution ledger** (phases 31–39) |
| [phases/](phases) | Per-phase atomic checklists (API/CLI 31–38 + external benches 39) |

Workflow: [../../skills/phase-wise-checklist/SKILLS.md](../../skills/phase-wise-checklist/SKILLS.md)

Predecessor: [../0.2.3/README.md](../0.2.3/README.md) (module path / `go install`);
engine contracts from [../0.2.1/24-canonical-0.2.1-roadmap.md](../0.2.1/24-canonical-0.2.1-roadmap.md).

Product framing: [../../documentation/overview.md](../../documentation/overview.md),
[../../documentation/library-api.md](../../documentation/library-api.md),
[../../documentation/cli.md](../../documentation/cli.md),
[../../testdata/golden/benchmarks/README.md](../../testdata/golden/benchmarks/README.md).

## Scope in one line

Hard-break the public Go API from wkhtml-style dotted `Set`/`Converter` to a
struct-based **Document** model, redesign the CLI to match, and freeze the
three-engine external compare paths (wkhtmltopdf, WeasyPrint, Puppeteer).
Engine layout and PDF writer stay; outer contract + bench harness paths change.

## External benchmark paths (Phase 39)

| Target | Script / test | Results under `testdata/golden/benchmarks/` |
|--------|---------------|--------------------------------------------|
| `make bench-cli-compare` | wkhtmltopdf compare test | `cli-compare.md`, `cli-compare-results.csv` |
| `make bench` | `scripts/bench-external.sh` → `scripts/weasyprint/print.sh`, `scripts/puppeteer/print.sh` | `weasyprint-compare*`, `puppeteer-compare*` |
| `make bench-engine` | Internal conversion allocation matrix | documented in benchmarks README |
| `make bench-lib` | Public `Document.WritePDF` / `ImageDocument.WriteImage` allocation matrix | documented in benchmarks README |
