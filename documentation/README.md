# gowkhtmltopdf documentation

User and design docs for the pure-Go, stdlib-only HTML→PDF / HTML→image engine.

| Document | Purpose |
|----------|---------|
| [overview.md](overview.md) | What it is, who it’s for, pipeline at a glance |
| [fidelity.md](fidelity.md) | **Fidelity guide:** tiers, claims language, feature map, degrade rules |
| [getting-started.md](getting-started.md) | Install, first PDF, local files, HTTP URLs |
| [architecture.md](architecture.md) | Package map and conversion pipeline |
| [cli.md](cli.md) | `gowkhtmltopdf` / `gowkhtmltoimage` usage |
| [library-api.md](library-api.md) | Go library (`NewConverter`, settings) |
| [integration-security.md](integration-security.md) | **Gin/web apps:** SSRF, local files, preferred patterns (also vs wkhtmltopdf) |
| [samples.md](samples.md) | Golden fixtures and committed `output/` samples |
| [compatibility-matrix.md](compatibility-matrix.md) | Per-element / per-property support (normative contract) |
| [fonts.md](fonts.md) | Font discovery, Type0/CJK path, honest shaping limits |
| [THREAT-MODEL.md](THREAT-MODEL.md) | Security model and local-file ACL |
| [comparison-with-others/sebastiaanklippert-go-wkhtmltopdf.md](comparison-with-others/sebastiaanklippert-go-wkhtmltopdf.md) | vs SebastiaanKlippert/go-wkhtmltopdf (wrapper vs pure-Go engine) |

## Contributing

| Document | Purpose |
|----------|---------|
| [../CONTRIBUTIONS.md](../CONTRIBUTIONS.md) | Setup, tests, PR workflow, layout visual QA, doc update map |
| [../CHANGELOG.md](../CHANGELOG.md) | User-facing release notes |
| [../skills/PR/PR_TEMPLATE.md](../skills/PR/PR_TEMPLATE.md) | Pull request body template |

## Plans (implementation ledgers)

Phase-by-phase execution notes live under [`../plans/`](../plans/), not here:

- [Canonical rewrite plan (MVP 0–9)](../plans/00-canonical-pure-go-rewrite.md)
- [Post-MVP roadmap (10–23)](../plans/10-canonical-post-mvp-roadmap.md)
- [Per-phase ledgers](../plans/phases/)

## License

[MIT](../LICENSE) - Copyright (c) 2026 Chinmay Sawant.
