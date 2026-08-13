# gowkhtmltopdf documentation

User-facing guides for the pure-Go, no-cgo HTML→PDF and HTML→image engine.

Start here if you are new: **[getting-started.md](getting-started.md)**.
Start here if you want the product contract: **[overview.md](overview.md)**
and **[fidelity.md](fidelity.md)**.

## How to read this set

| If you want to… | Read |
|-----------------|------|
| Install, convert one file, embed the Go API | [getting-started.md](getting-started.md) |
| Know what the engine is (and is not) | [overview.md](overview.md), [fidelity.md](fidelity.md) |
| Run the CLI | [cli.md](cli.md) |
| Call the library from Go | [library-api.md](library-api.md) |
| Check whether a CSS property or HTML tag is supported | [compatibility-matrix.md](compatibility-matrix.md) |
| Understand fonts, CJK, Arabic, `@font-face` | [fonts.md](fonts.md) |
| Embed the converter in a web app safely | [integration-security.md](integration-security.md), [THREAT-MODEL.md](THREAT-MODEL.md) |
| See sample PDFs and golden fixtures | [samples.md](samples.md), [`../output/`](../output/) |
| Understand the pipeline and packages | [architecture.md](architecture.md), [architecture/](architecture/) |
| Compare with wkhtmltopdf wrappers / browsers | [comparison-with-others/](comparison-with-others/) |
| See what is deferred or out of scope | [deferred.md](deferred.md) |
| Read performance numbers | [performance.md](performance.md) |

## Guides

| Document | Purpose |
|----------|---------|
| [overview.md](overview.md) | Product overview, design principles, pipeline, binaries |
| [getting-started.md](getting-started.md) | Install, first PDF/PNG, local files, HTTP URLs, tests |
| [cli.md](cli.md) | `gowkhtmltopdf` / `gowkhtmltoimage` grammar, flags, pitfalls |
| [library-api.md](library-api.md) | Go API: `RunPDF` / `Converter`, settings, errors |
| [fidelity.md](fidelity.md) | Tiers, claims language, degrade rules, Phase 21 URL print |
| [compatibility-matrix.md](compatibility-matrix.md) | Normative per-element / per-property / per-flag contract |
| [fonts.md](fonts.md) | Bundled faces, `--font-path`, Type0/CID, shaping limits |
| [samples.md](samples.md) | Golden fixtures, `output/`, `make samples` / `make golden` |
| [performance.md](performance.md) | Benchmarks, CLI comparison, how to measure |
| [deferred.md](deferred.md) | Deferred features, workload priority, next gates |

## Architecture

| Document | Purpose |
|----------|---------|
| [architecture.md](architecture.md) | Package map and conversion pipeline (one page) |
| [architecture/README.md](architecture/README.md) | Deep-dive index (CLI → load → HTML → CSS → layout → PDF) |

The ten domain notes under [architecture/](architecture/) are source-grounded
and include `file:line` references. Prefer them when changing engine code.

## Security

| Document | Purpose |
|----------|---------|
| [THREAT-MODEL.md](THREAT-MODEL.md) | Trust boundary, local-file ACL, network policy, timeouts |
| [integration-security.md](integration-security.md) | Gin / HTTP embedding: SSRF, preferred patterns, isolated workers |

## Comparisons

| Document | Purpose |
|----------|---------|
| [comparison-with-others/sebastiaanklippert-go-wkhtmltopdf.md](comparison-with-others/sebastiaanklippert-go-wkhtmltopdf.md) | Wrapper around the wkhtmltopdf binary vs this in-process engine |
| [comparison-with-others/landscape-2026.md](comparison-with-others/landscape-2026.md) | 2026 landscape vs Chromium, wkhtmltopdf, WeasyPrint, Prince |

## Contributing and plans

| Document | Purpose |
|----------|---------|
| [../CONTRIBUTIONS.md](../CONTRIBUTIONS.md) | Setup, tests, PR workflow, layout visual QA, doc update map |
| [../CHANGELOG.md](../CHANGELOG.md) | User-facing release notes |
| [../skills/PR/PR_TEMPLATE.md](../skills/PR/PR_TEMPLATE.md) | Pull request body template |
| [../plans/README.md](../plans/README.md) | Implementation ledger index |
| [../plans/00-canonical-pure-go-rewrite.md](../plans/00-canonical-pure-go-rewrite.md) | MVP phases 0–9 |
| [../plans/10-canonical-post-mvp-roadmap.md](../plans/10-canonical-post-mvp-roadmap.md) | Post-MVP phases 10–23 |

## License

[MIT](../LICENSE) — Copyright (c) 2026 Chinmay Sawant.

Bundled Liberation and DejaVu fonts are SIL OFL / Bitstream Vera; see
[internal/pdf/assets/NOTICE](../internal/pdf/assets/NOTICE).
