# chore: surface CONTRIBUTING.md and exclude JS from language stats

## Summary

Rename `CONTRIBUTIONS.md` to `CONTRIBUTING.md` so GitHub actually surfaces the contributing guide, and vendor `*.mjs` / `*.cjs` in `.gitattributes` so leftover frontend scripts stop showing up as JavaScript in the language bar.

---

## Motivation / context

- Plans: none (follow-up to #37)
- Issues: see **Related issues**
- GitHub only recognizes `CONTRIBUTING.md` (root, `.github/`, or `docs/`) for the Contributing tab, sidebar link, `/contribute` page, and the “view contributing guidelines” prompt on new issues and PRs. The existing guide was named `CONTRIBUTIONS.md`, so none of that fired.
- After #37, GitHub language stats reported exactly 5,220 bytes of JavaScript — `frontend/scripts/copy-to-docs.mjs` (707) + `frontend/scripts/smoke-test.mjs` (4513). `*.js` was already vendored; `.mjs` was not.

---

## Changes

### Contributing guide

- Rename `CONTRIBUTIONS.md` → `CONTRIBUTING.md` (content unchanged).
- Update live links in `README.md`, `documentation/README.md`, `documentation/getting-started.md`, `documentation/samples.md`, and the two skill references that pointed at the old name.
- Point historical plan PR notes at the new path so those links do not 404.

### Linguist / language stats

- Mark `*.mjs` and `*.cjs` as `linguist-vendored` next to the existing `*.js` rule.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None |
| **Memory** | None |
| **Behavior / correctness** | None. Conversion, CLI, and library are unchanged. |
| **API / CLI** | None |
| **Dependencies** | None |
| **Binary size / build time** | None |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| Bookmark or clone-side link to `CONTRIBUTIONS.md` | Use `CONTRIBUTING.md` |

---

## Test plan

- [x] `git check-attr linguist-vendored -- frontend/scripts/copy-to-docs.mjs frontend/scripts/smoke-test.mjs` → `set`
- [x] Confirmed GitHub languages API previously reported `JavaScript: 5220`, matching the two unvendored `.mjs` files
- [x] No remaining `CONTRIBUTIONS.md` references (`rg CONTRIBUTIONS` is empty)
- [ ] GitHub Contributing tab / sidebar appear after merge to `master`
- [ ] Language bar drops JavaScript after linguist reprocesses `master`

### Commands

```sh
git check-attr linguist-vendored -- frontend/scripts/copy-to-docs.mjs frontend/scripts/smoke-test.mjs
rg CONTRIBUTIONS
```

---

## Screenshots / sample output

```
frontend/scripts/copy-to-docs.mjs: linguist-vendored: set
frontend/scripts/smoke-test.mjs: linguist-vendored: set
```

---

## Related issues

- Relates to #37

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled with real ticket IDs
- [x] Filled body committed under `plans/0.2.0/PR/pr-github-community-meta.md`

---

## Follow-ups (out of scope)

- None

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in the diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
