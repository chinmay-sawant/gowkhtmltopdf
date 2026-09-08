# Release checklist

Read this before cutting any release. No release ships unless every hard
gate below exits 0. This file is the single checklist; `AGENTS.md`
points here so the checks appear on every release attempt.

## Version sources

`VERSION` is the single source of truth. It is a bare semver string
with no `v` prefix (`VERSION:1`).

| Surface | Where it lives | What it reports |
|---|---|---|
| Binary version | `internal/cli/help.go:15` (`var Version`), stamped by `Makefile:76` (`CLI_VERSION_LDFLAGS`) and `release.yml:68` | `bin/gowkhtmltopdf --version` prints `Name:` + `Version:` |
| Go library compat id | `api.go:23` (`LibraryVersion = "0.12.7-dev"`), banner in `api.go:26` (`Version()`) | Upstream wkhtmltopdf settings-surface id, not the project release. Never bump it on release. |
| C ABI release | `bindings/c/include/gowkhtmltopdf.h:105` (`GOWKHTMLTOPDF_VERSION`), compat id at `:106` (`GOWKHTMLTOPDF_LIBRARY_VERSION`), runtime stamp via `bindings/c/main.go:18` (`libVersion`, `Makefile:81`) | `gowkhtmltopdf_version()` must equal `VERSION`; `gowkhtmltopdf_abi_version()` is `1` |
| Python package | `bindings/python/pyproject.toml:7` (`version`), runtime copy in `bindings/python/src/gowkhtmltopdf/__init__.py:68` (`__version__`), compat id at `:72` (`library_version`) | `import gowkhtmltopdf; gowkhtmltopdf.__version__` must equal `VERSION`. Checked by `scripts/check_versions.sh:1` |

Bump these together: `VERSION`, `internal/cli/help.go` default,
`bindings/c/include/gowkhtmltopdf.h`, `bindings/python/pyproject.toml`
plus `__init__.py`, `CHANGELOG.md`. Then run `make check-versions`.

## What ships today

- Static Go binaries `cmd/gowkhtmltopdf` and `cmd/gowkhtmltoimage`
  (`CGO_ENABLED=0`, linux/windows/darwin x amd64/arm64 via `release.yml`).
- Go library `Document` / `ImageDocument` at the repo root (`api.go`,
  `document.go`).
- Python bindings (in-process, opt-in cgo): frozen C ABI v1 in
  `bindings/c` plus `ctypes` package in `bindings/python`
  (`documentation/python.md`). Built with `make c-shared`, tested with
  `make python-binding-test`, published to PyPI on `v*` tags via
  `publish-pypi.yml`.

## What does not ship

- WASM / browser runtime. There is no `GOOS=js` / `GOARCH=wasm` target,
  no `bindings/wasm`, no `syscall/js` bridge. The engine runs as native
  binaries, a Go library, or the Python-loaded shared library only.
  A future WASM port that runs in the browser is possible but not
  planned and not claimed anywhere. Do not list it as supported in
  release notes, `README.md`, `documentation/`, or frontend content
  (`make claim-scan` polices these surfaces). Once a WASM implementation
  lands, it becomes a release artifact too: build it, version-stamp it
  from `VERSION`, cover it with tests, and ship it with every release.

## Hard gates (all must pass)

Run in this order. Stop on the first failure.

```sh
make check-versions
make test
make golden
make claim-scan
make lint
```

Then the static build plus version-stamp assertion (mirrors CI):

```sh
make build
test "$(./bin/gowkhtmltopdf --version | sed -n 's/^Version:[[:space:]]*//p' | tr -d '[:space:]')" = "$(tr -d '[:space:]' < VERSION)"
CGO_ENABLED=0 go build ./...
```

When the release touches `bindings/` or Python packaging, also run:

```sh
CGO_ENABLED=1 make c-shared
make python-binding-test
```

When the release touches `frontend/src/data/content/`, also verify the
docs site builds clean with no dirty `docs/` output (CI fails otherwise):

```sh
npm ci --prefix frontend
npm --prefix frontend run build
git status --porcelain -- docs frontend/dist
```

Race job (`make test-race`) runs in CI on hot packages
(`internal/convert`, `internal/layout`, `internal/pdf`,
`internal/imageout`, `internal/load`). Run it locally before release
when layout, convert, pdf, imageout, or load changed.

## Cut steps

1. Bump version files together (see table above). Move `CHANGELOG.md`
   `## Unreleased` into a dated `## <ver> (YYYY-MM-DD)` section.
2. Drop `unreleased <ver>` language from generic docs per
   `skills/release-note/SKILL.md` (README, `documentation/`, frontend
   content), rebuild `docs/` when site text changed.
3. Run all hard gates above. Paste exit codes into the PR body.
4. Open a `chore/release-<ver>` PR to `master` using
   `skills/PR/PR_TEMPLATE.md`, self-assign, add at least one label.
5. After merge, only on explicit approval: `git tag v<ver> &&
   git push origin v<ver>`. `VERSION` must match the tag without the
   `v` or `release.yml` refuses to publish. PyPI publish runs from the
   same tag (`publish-pypi.yml` checks versions plus `twine check`).
6. Paste the GitHub Release body from `plans/<ver>/PR/release-v<ver>.md`.

See also: `CONTRIBUTING.md` (cutting a release),
`skills/release-note/SKILL.md` (full promote flow),
`.github/workflows/release.yml` (tag-gated binary build).
