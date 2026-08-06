# Sample outputs

Committed samples produced by `make samples` (and optional URL smoke tests).
Regenerate anytime:

```sh
make samples
```

| File | Description |
|------|-------------|
| `fixture-01` … `fixture-21-*.pdf` | Golden HTML fixtures under `testdata/golden/` converted to PDF |
| `fixture-01-simple-invoice.png` | Same fixture via `gowkhtmltoimage` |
| `fixture-21-detailed-report.png` | Detailed report fixture via library image converter |
| `showcase-toc-hf-outline.pdf` | TOC + headers/footers + outline on `fixture-16` |
| `wiki-ana-de-armas.pdf` | Live Wikipedia **raw** smoke from `make samples` (no `--simplify-dom`; `--use-system-fonts`; needs network; soft-fail if offline). Not a chrome-stripped “pretty” print — see `documentation/cli.md` URL recipes. |


These are **viewer smoke artifacts**, not golden byte baselines. CI uses
`make golden` / structure assertions, not binary PDF equality against this folder.
