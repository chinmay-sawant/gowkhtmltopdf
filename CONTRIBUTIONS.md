# Contributing to gowkhtmltopdf

Thanks for helping improve this pure-Go HTML→PDF / HTML→image engine.

This document covers **how to contribute** (setup, tests, PRs, layout QA).
Product design, fidelity tiers, and the support matrix live under
[`documentation/`](documentation/README.md). Implementation ledgers live under
[`plans/`](plans/README.md).

---

## Ground rules

1. **Pure Go, no CGO** for the main path. Direct third-party modules stay on the
   allowlist enforced by `internal/pdf.TestDirectModuleAllowlist`
   (currently OpenType shaping + SVG raster via allowlisted deps — see
   `Makefile` / `go.mod`).
2. **Generic engine fixes** — prefer CSS/print semantics over site-specific
   (e.g. MediaWiki class) hacks. Operator policy belongs in CLI flags, not
   cascade overrides for one skin.
3. **Honesty** — update `documentation/compatibility-matrix.md` and
   `documentation/fidelity.md` when behavior claims change.
4. **Determinism** — same inputs should produce stable PDF structure (font
   subsets / timestamps may still differ; do not rely on byte-identical PDFs
   as golden masters unless a test explicitly does).

---

## Development setup

```sh
git clone https://github.com/chinmay-sawant/gowkhtmltopdf.git
cd gowkhtmltopdf
go test ./...
make build
```

| Target | Purpose |
|--------|---------|
| `make test` | Full `go test ./...` |
| `make lint` | `go vet` + gofmt check |
| `make build` | `bin/gowkhtmltopdf`, `bin/gowkhtmltoimage` |
| `make golden` | Golden fixture corpus (`internal/convert`) |
| `make samples` | Regenerate `output/` fixtures + optional live wiki smoke |
| `make fmt` | `gofmt -w .` |

Minimum: recent Go toolchain matching `go.mod`.

---

## Branch and PR workflow

1. Branch from **`master`** (default integration branch).
2. Keep commits focused; prefer conventional-style messages:
   `fix(layout): …`, `feat(css): …`, `docs: …`, `test: …`.
3. Open a PR **to `master`** with:
   - Self-assign (`--assignee @me`)
   - At least one label (`bug`, `enhancement`, `documentation`, …)
   - Body based on [`skills/PR/PR_TEMPLATE.md`](skills/PR/PR_TEMPLATE.md)
   - Optional filled copy under `plans/PR/pr-<slug>.md`
4. Do **not** merge your own PR unless maintainers agree.

```sh
gh pr create \
  --base master \
  --head "$(git branch --show-current)" \
  --title "fix(layout): short imperative description" \
  --body-file plans/PR/pr-<slug>.md \
  --assignee "@me" \
  --label bug \
  --label enhancement
```

Issue templates: [`skills/PR/ISSUE_TEMPLATE.md`](skills/PR/ISSUE_TEMPLATE.md).

---

## Where to change code

| Concern | Package |
|---------|---------|
| CLI / flags | `internal/cli`, `cmd/gowkhtmltopdf` |
| Load / HTTP / ACL | `internal/load` |
| HTML parse | `internal/html` |
| CSS cascade | `internal/css` |
| Layout, floats, tables, pagination | `internal/layout` |
| PDF write / fonts / images | `internal/pdf` |
| Image mode | `internal/imageout` |
| End-to-end convert | `internal/convert` |
| Public library API | root `gowkhtmltopdf` (`api.go`) |

Pipeline: **load → parse → style → layout → paginate/paint → PDF write**.

---

## Testing expectations

### Required for layout/print changes

```sh
go test ./internal/layout/ -count=1
go test ./internal/convert/ -run 'TestGoldenCorpus' -count=1
make lint
```

Add or extend unit tests next to the code you touch
(e.g. `internal/layout/*_test.go`). Prefer **regression tests** that fail
before the fix for pagination, tables, floats, and avoid-packing bugs.

### Visual QA (layout regressions)

Golden structure tests do **not** catch text overlap, underline noise, or
table chrome. For print/layout work:

1. Rebuild: `make build`
2. Regenerate samples if needed: `make samples` (network for wiki smoke)
3. Open `output/wiki-ana-de-armas.pdf` (or a focused fixture) in a real viewer
4. Check: no overlapping body lines, table borders continuous, ref underlines
   not dominating, no huge empty bands from `page-break-inside: avoid`

Optional live smoke (also part of `make samples`):

```sh
./bin/gowkhtmltopdf --use-system-fonts --zoom 0.666667 \
  'https://en.wikipedia.org/wiki/Ana_de_Armas' \
  output/wiki-ana-de-armas.pdf
```

See [`documentation/samples.md`](documentation/samples.md).

### Dangerous patterns (layout)

- **Document-global Y shifts** of all ops below a baseline (historically
  `shiftOpsBelowY` in gap packing) — easily interleaves body paragraphs.
  Prefer scoped shifts or `preferSplitOverBlank`-style policy.
- **Site-specific CSS** injected for one website skin — use operator flags.
- **Weakening** `page-break-inside: avoid` without tests for blank-page
  cascades on dense lists.

---

## Documentation updates

When your change affects user-visible behavior:

| Change type | Update |
|-------------|--------|
| CSS / element support | `documentation/compatibility-matrix.md` |
| Fidelity claims / tiers | `documentation/fidelity.md` |
| CLI flags | `documentation/cli.md` + README if featured |
| Library API | `documentation/library-api.md` |
| Samples / smoke | `documentation/samples.md` |
| Security / ACL | `documentation/THREAT-MODEL.md` |
| User-facing release notes | `CHANGELOG.md` (Unreleased or next version) |

Implementation plans under `plans/` are for maintainers/agents; keep them in
sync only when you close a phase item.

---

## Coding notes

- Run `gofmt` on all Go you touch (`make fmt` / `make lint`).
- Match existing package style; keep modules deep (small public surface).
- Do not commit secrets, large unrelated binaries, or personal paths.
- `output/*.pdf` sample updates are OK when intentionally refreshing smoke
  artifacts (call out in the PR).

---

## Reporting bugs

Include:

- gowkhtmltopdf version / commit
- Minimal HTML (or fixture path) and exact CLI / library call
- Expected vs actual (screenshot of PDF page helps for layout bugs)
- OS and Go version

Security issues: prefer private report if the bug is load/SSRF related; see
[`documentation/THREAT-MODEL.md`](documentation/THREAT-MODEL.md).

---

## License

By contributing, you agree that your contributions are licensed under the
project’s [MIT License](LICENSE).
