## Summary

The site compatibility page now names all 72 properties from the unreleased next-72 wave, and the getting-started, CLI, fonts, and landing copy match the current engine.

---

## Motivation / context

- Plans: `plans/0.2.7/next-72-properties.json`
- Issues: see **Related issues**

The markdown matrix already listed every next-72 name. The page at `/documentation/compatibility` reads `frontend/src/data/content/page-compatibility.json`, which still used wildcards such as `font-variant*` and `border-clip family`.

---

## Changes

### Compatibility page

- Added a Next-72 table with one row per property: 53 Implemented, 19 Not implemented.
- Spelled the same names out in the existing CSS status rows, including `column-height` and `column-wrap`.

### Guides and landing

- Dropped unfinished 0.2.4 migration wording from getting started, the CLI, and the library API.
- Documented header, footer, `--simplify-dom`, and `--print-link-underline` on the CLI page, and fixed the Wikipedia sample command to use `--url` and `-o`.
- Landing lede now says more than 400 CSS properties. The catalog count is 407 implemented of 818.
- Rebuilt `docs/` so the published site matches the frontend source.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None |
| **Memory** | None |
| **Behavior / correctness** | Docs and the static site only. No engine change. |
| **API / CLI** | None |
| **Dependencies** | None |
| **Binary size / build time** | None |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [x] `npm --prefix frontend run lint`
- [x] `npm --prefix frontend run build` (writes `docs/`)
- [x] `make claim-scan`
- [ ] `make test` (not run; no Go changes)
- [ ] `make lint` / `go vet` (not run; no Go changes)
- [ ] `make build` (not run; no binary change)
- [ ] `make run` wall time vs baseline (not run; no engine change)

### Commands

```sh
npm --prefix frontend run lint
npm --prefix frontend run build
make claim-scan
```

---

## Screenshots / sample output

```
claim-scan: clean
src/data lint clean (11 content pages, 68 showcase items)
vite build copied into docs/
```

---

## Related issues

- No linked issue.

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled (none open for this docs pass)
- [x] Filled body committed under `plans/PR/pr-refresh-site-guides.md`

---

## Follow-ups (out of scope)

- The planned next-100 property list is not in the engine yet, so it is not on the compatibility page.
- Performance tables stay on the 2026-09-13 capture. Later allocation work has no new published matrix.

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
- [ ] Diff-stat-by-extension table pasted at the bottom

---

## Diff stat by extension

| Extension | Files | Insertions | Deletions |
| --- | ---: | ---: | ---: |
| `.html` | 1 | 1 | 1 |
| `.js` | 6 | 26 | 25 |
| `.json` | 4 | 400 | 11 |
| `.jsx` | 1 | 1 | 1 |
| `.md` | 8 | 246 | 79 |
| **Total** | **20** | **674** | **117** |
