# gowkhtmltopdf documentation

User and design docs for the pure-Go, stdlib-only HTML→PDF / HTML→image engine.

| Document | Purpose |
|----------|---------|
| [overview.md](overview.md) | What it is, who it’s for, pipeline at a glance |
| [getting-started.md](getting-started.md) | Install, first PDF, local files, HTTP URLs |
| [architecture.md](architecture.md) | Package map and conversion pipeline |
| [cli.md](cli.md) | `gowkhtmltopdf` / `gowkhtmltoimage` usage |
| [library-api.md](library-api.md) | Go library (`NewConverter`, settings) |
| [integration-security.md](integration-security.md) | **Gin/web apps:** SSRF, local files, preferred patterns (also vs wkhtmltopdf) |
| [samples.md](samples.md) | Golden fixtures and committed `output/` samples |
| [compatibility-matrix.md](compatibility-matrix.md) | Per-element / per-property support |
| [THREAT-MODEL.md](THREAT-MODEL.md) | Security model and local-file ACL |

## Plans (implementation ledgers)

Phase-by-phase execution notes live under [`../plans/`](../plans/), not here:

- [Canonical rewrite plan](../plans/00-canonical-pure-go-rewrite.md)
- [Per-phase ledgers](../plans/phases/)

## License

[MIT](../LICENSE) — Copyright (c) 2026 Chinmay Sawant.
