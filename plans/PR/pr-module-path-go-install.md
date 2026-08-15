## Summary

Fix `go install github.com/chinmay-sawant/gowkhtmltopdf/cmd/gowkhtmltopdf@v0.2.2`, which failed because `go.mod` declared `module gowkhtmltopdf`. The module path now matches the GitHub import path, and nested stub `go.mod` files keep `frontend/`, `output/`, and `docs/` out of the parent module zip.

---

## Motivation / context

- Plans: none (follow-up to the v0.2.2 release notes / Getting Started `go install` docs)
- Issues: see **Related issues**
- Reproduced error:

```
go: github.com/chinmay-sawant/gowkhtmltopdf/cmd/gowkhtmltopdf@v0.2.2: version constraints conflict:
        github.com/chinmay-sawant/gowkhtmltopdf@v0.2.2: parsing go.mod:
        module declares its path as: gowkhtmltopdf
                but was required as: github.com/chinmay-sawant/gowkhtmltopdf
```

---

## Changes

### Module path

- Root `go.mod`: `module github.com/chinmay-sawant/gowkhtmltopdf`
- Rewrite all in-repo imports from `"gowkhtmltopdf/..."` to `"github.com/chinmay-sawant/gowkhtmltopdf/..."`
- Stamp ldflags as `-X github.com/chinmay-sawant/gowkhtmltopdf/internal/cli.Version=…` (`Makefile`, release workflow, help comment, Getting Started)

### Slimmer `go install` zip

Nested stub modules (excluded from the parent module zip):

- `frontend/go.mod`
- `output/go.mod`
- `docs/go.mod`

`go install` of the CLIs no longer pulls the documentation site or committed sample PDFs.

### Docs

- Getting Started, library API, architecture CLI note, frontend Getting Started / Library API / command palette
- Release-note source: `plans/0.2.2/PR/release-v0.2.2.md` (`go install` + new import path)

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None |
| **Memory** | None |
| **Behavior / correctness** | Conversion unchanged. `go test ./...` no longer descends into `frontend/node_modules` |
| **API / CLI** | Library import path changes (see migration). CLI flags unchanged |
| **Dependencies** | None new |
| **Binary size / build time** | Unchanged locally. Proxy module zip drops frontend/output/docs (~100MB tracked) |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| `import "gowkhtmltopdf"` | `import "github.com/chinmay-sawant/gowkhtmltopdf"` |
| `go get gowkhtmltopdf@v0.2.2` | `go get github.com/chinmay-sawant/gowkhtmltopdf@<new-tag>` |
| `go install …@v0.2.2` | Still broken: tag `v0.2.2` has the old module line. Needs a new tag after this merges (e.g. `v0.2.3`) |

---

## Test plan

- [x] `go test ./...`
- [ ] `make lint`
- [ ] `CGO_ENABLED=0 go build -o bin/gowkhtmltopdf ./cmd/gowkhtmltopdf`
- [ ] After merge + new tag: `go install github.com/chinmay-sawant/gowkhtmltopdf/cmd/gowkhtmltopdf@v0.2.3`

### Commands

```sh
go test ./...
make lint
make build
```

---

## Screenshots / sample output

```
ok  github.com/chinmay-sawant/gowkhtmltopdf                  ...
ok  github.com/chinmay-sawant/gowkhtmltopdf/internal/convert ...
ok  github.com/chinmay-sawant/gowkhtmltopdf/internal/layout  ...
```

---

## Related issues

- Relates to the v0.2.2 release cut (#50) — `go install` documented there does not work until the module path matches

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled (no standalone ticket; relates to #50)
- [x] Filled body committed under `plans/PR/pr-module-path-go-install.md`

---

## Follow-ups (out of scope)

- Tag `v0.2.3` (or next patch) so `go install …@v0.2.3` hits the new `go.mod`
- Optionally exclude `testdata/` the same way (would need a replace so `go run ./testdata/golden/api` still works)

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in the diff
- [ ] Public API / CLI docs use the GitHub module path
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
