## Summary

Promote **v0.2.3** as the current release: bump `VERSION` and `cli.Version`, add a dated changelog section, point Getting Started / the docs site at `go install …@v0.2.3`, and commit the GitHub release body (same product notes as v0.2.2, plus #50 and #51).

No converter behavior change. Same engine as [v0.2.2](https://github.com/chinmay-sawant/gowkhtmltopdf/releases/tag/v0.2.2).

---

## Motivation / context

- Plans: [`plans/0.2.3/README.md`](plans/0.2.3/README.md), [`plans/0.2.3/PR/release-v0.2.3.md`](plans/0.2.3/PR/release-v0.2.3.md)
- Release-prep shape matches [`plans/0.2.2/PR/pr-release-0.2.2.md`](plans/0.2.2/PR/pr-release-0.2.2.md)
- CONTRIBUTING requires `VERSION` + a dated `CHANGELOG.md` section before tagging `v0.2.3`
- Issues: see **Related issues**

---

## Changes

### Release metadata

- `VERSION` and `internal/cli.Version` `0.2.2` → `0.2.3`
- `CHANGELOG.md`: dated **0.2.3 (2026-08-15)**; empty `## Unreleased` kept
- Add [`plans/0.2.3/PR/release-v0.2.3.md`](plans/0.2.3/PR/release-v0.2.3.md) (v0.2.2 body + #50 / #51 + `go install @v0.2.3`)

### Docs and site

- README, overview, getting started, library API, landscape: current release is **v0.2.3**
- `go install` / `go get` examples use `@v0.2.3`
- Frontend Getting Started, About, Overview, Library API, command palette
- Feature history that says “shipped in 0.2.2” is left as 0.2.2

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None |
| **Memory** | None |
| **Behavior / correctness** | None in the converter. Version stamp and docs only |
| **API / CLI** | Import path already `github.com/chinmay-sawant/gowkhtmltopdf` (#51). Install pin is `@v0.2.3` |
| **Dependencies** | None |
| **Binary size / build time** | Unchanged. Tag `v0.2.3` stamps `internal/cli.Version` via the release workflow |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| `go install …@v0.2.2` | Use `@v0.2.3` |
| `go get …@v0.2.2` | Use `@v0.2.3` |
| `import "gowkhtmltopdf"` | Already `github.com/chinmay-sawant/gowkhtmltopdf` from #51 |

---

## Test plan

- [x] `VERSION` is `0.2.3`
- [x] `cli.Version` default is `0.2.3`
- [x] `CHANGELOG.md` has dated **0.2.3 (2026-08-15)**
- [x] Getting Started / library API / frontend install commands use `@v0.2.3`
- [ ] `make test`
- [ ] `make lint`
- [ ] After merge (not this PR): tag `v0.2.3` and paste [`plans/0.2.3/PR/release-v0.2.3.md`](plans/0.2.3/PR/release-v0.2.3.md)

### Commands

```sh
test "$(tr -d '[:space:]' < VERSION)" = "0.2.3"
rg -n 'go install github.com/chinmay-sawant/gowkhtmltopdf/cmd/gowkhtmltopdf@v0.2.3' \
  documentation/getting-started.md frontend/src/data/content/page-getting-started.json
```

---

## Screenshots / sample output

```
VERSION=0.2.3
CHANGELOG: ## 0.2.3 (2026-08-15)
go install github.com/chinmay-sawant/gowkhtmltopdf/cmd/gowkhtmltopdf@v0.2.3
```

---

## Related issues

- Relates to #29 (newer PDF versions and compliance epic)
- Relates to #50 (v0.2.2 release prep)
- Relates to #51 (GitHub module path)

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled with real ticket IDs
- [x] Filled body committed under `plans/0.2.3/PR/pr-release-0.2.3.md`

---

## Follow-ups (out of scope)

- Tag `v0.2.3` and publish the GitHub Release **after** this PR merges
- Do not retag `v0.2.2`

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in the diff
- [ ] Public API / CLI docs pin `@v0.2.3`
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
