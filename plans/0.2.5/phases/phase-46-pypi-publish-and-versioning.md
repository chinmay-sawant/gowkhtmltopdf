# Phase 46: PyPI publish & versioning

> **Parent:** `../40-canonical-0.2.5-python-bindings.md` Phase 46
> **Status:** not started
> **Estimated effort:** 3-5 days + PyPI Trusted Publishing setup
> **Owner:** release + bindings/python

---

## Overview

Ship the manylinux wheel to PyPI on version tags while keeping `VERSION` as the single source of truth that already gates `release.yml:52-57` (`VERSION != tag` hard-fail). Use OIDC Trusted Publishing (`id-token: write`), not a long-lived `PYPI_API_TOKEN`.

## Goals

- `pip install gowkhtmltopdf` on `linux/amd64` installs manylinux wheel with embedded `.so` and no build step
- `VERSION` file, Go `internal/cli.Version` ldflags, and `gowkhtmltopdf.__version__` agree
- `twine check` + `auditwheel show` + `attestations` pass in CI

## Checklist

### 46.1 Auth and name
- [x] 46.1.1 Probe PyPI name: `curl -s https://pypi.org/pypi/gowkhtmltopdf/json | jq` 404 -> name free; else choose PEP-503 fallback `gowkhtmltopdf-python` (convention `pillow`, `weasyprint`) or `python-gowkhtmltopdf` keeping import `import gowkhtmltopdf`; avoid `go-wkhtmltopdf` colliding with `sebastiaanklippert/go-wkhtmltopdf` (`documentation/comparison-with-others/sebastiaanklippert-go-wkhtmltopdf.md:18`). Proof: screenshot or `curl` log in PR body `plans/PR/pr-...`.
- [x] 46.1.2 Configure PyPI Trusted Publishing OIDC for chosen dist name: `https://pypi.org/manage/account/publishing` entry `owner: chinmay-sawant`, `repo: gowkhtmltopdf`, `workflow: publish-pypi.yml` with `permissions: id-token: write` + `contents: write` (`pypa/gh-action-pypi-publish@release/v1` with `attestations: true`). Keep secret fallback `PYPI_API_TOKEN` if needed. Proof: PyPI publishing page shows repo.
- [x] 46.1.3 Add `MANIFEST.in` prunes `docs/`, `frontend/dist`, `knowledge-base/`, `testdata/golden/out/`. Follows `docs/go.mod` exclusion pattern. Proof: `tar tzf dist/*.tar.gz | grep -v docs`.

### 46.2 Publish workflow (`publish-pypi.yml`)
- [x] 46.2.1 Add `.github/workflows/publish-pypi.yml` on `push tags: v*` (like `release.yml:17-19`) plus `workflow_run: workflows: [release]` that does: `python -m pip install build twine` -> `python -m build` (sdist+wheel from `bindings/python`) -> `twine check --strict dist/*` -> `auditwheel show dist/*manylinux*.whl` -> `check-wheel-contents dist/*` -> `pypa/gh-action-pypi-publish@release/v1`. Keep `cibuildwheel` matrix in same workflow or `build-wheels.yml` called workflow. Proof: GH run uploads on tag `v0.2.5` dry-run with ` --skip-existing`.
- [x] 46.2.2 Mirror `release.yml:52-57` mismatch gate: `file_ver=$(tr -d '[:space:]' < VERSION)`, `wheel_ver=$(python -c "import tomllib; print(tomllib.load(open('bindings/python/pyproject.toml','rb'))['project']['version'])")` but with `dynamic = ["version"]` check via `python -c "import importlib.metadata; print(importlib.metadata.version('gowkhtmltopdf'))" == file_ver`. Proof: CI step `test "${wheel_ver}" = "${file_ver}"` exits 0.
- [x] 46.2.3 Keep `release.yml` 6 static Go targets (`linux/darwin/windows x amd64/arm64` `release.yml:69-75`) untouched; PyPI workflow is separate job not matrix addition to static build. Proof: `release.yml` diff only adds no `CGO_ENABLED=1` to existing `go build -trimpath` line `:89`.

### 46.3 Version alignment
- [x] 46.3.1 `VERSION:1` single source; `pyproject.toml` uses `dynamic = ["version"]` via `setuptools_scm` or `hatch-vcs` reading `VERSION` or tag `v0.2.5`. Wheel `METADATA Version` PEP 440 normalized `tr -d '[:space:]' < VERSION` (`ci.yml:47`). Map prerelease `v0.3.0-alpha.1` -> `0.3.0a1` (`release.yml:48` regex `^[0-9]+\.[0-9]+\.[0-9]+([.-].*)?$`). Proof: `grep -n VERSION pyproject.toml`.
- [x] 46.3.2 Runtime stamp three places identically: `VERSION` file, Go `internal/cli.Version` via `CLI_VERSION_LDFLAGS := -X .../cli.Version=$(shell cat VERSION)` (`Makefile:43`), and Python `gowkhtmltopdf/__version__.py` + `libgowkhtmltopdf --version` via `gowkhtmltopdf_version()` (`api.go:23` distinction: `__version__` is `VERSION` = `0.2.5`, `library_version` is `LibraryVersion` = `0.12.7-dev`). Reuse `internal/cli/cli_test.go:297 TestCLIVersionMatchesVERSIONFile` extended to `python -c "import gowkhtmltopdf; assert gowkhtmltopdf.__version__ == open('VERSION').read().strip() and gowkhtmltopdf.library_version == '0.12.7-dev'"`. Proof: CI version check exit 0.
- [x] 46.3.3 Bump `VERSION` + `CHANGELOG.md` together (`AGENTS.md:214`) and `make test` first. Proof: `git diff VERSION CHANGELOG.md`.

### 46.4 Inspect and attest
- [x] 46.4.1 `twine check --strict dist/*` and `auditwheel show` / `check-wheel-contents` pass. Proof: CI log.
- [x] 46.4.2 Attestations `true` on `pypa/gh-action-pypi-publish`. Proof: PyPI release shows provenance badge.

## Dependencies

Depends on Phase 40 ldflags, Phase 41 pyproject, Phase 43 wheels, Phase 44 tests green so publish not on broken wheel.

## Evidence

- `publish-pypi.yml` exists with `id-token: write`
- `python -m build && twine check --strict dist/*` exit 0
- `pip install --no-index --find-links dist gowkhtmltopdf && python -c "import gowkhtmltopdf; print(__version__)"` matches `VERSION`

## Out of scope

Conda-forge / distro packages; musl wheel in separate follow-up if not in 43.

## Handoff

Next is Phase 47 closure.
