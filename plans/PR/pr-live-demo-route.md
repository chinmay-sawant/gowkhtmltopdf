## Summary

Rename the Try Live Demo page route from `/wasm` to `/live-demo` so the URL matches the product label. Keep `/wasm` as a redirect for old bookmarks, and update nav, docs, smoke tests, and the generated site.

---

## Motivation / context

- The page is labeled "Try Live Demo" but lived at `/#/wasm`, which reads like an implementation detail.
- Issues: none filed; user request to rename the route.

---

## Changes

### Frontend routing

- Register `/live-demo` as the Live Demo page in `frontend/src/App.jsx`.
- Redirect `/wasm` to `/live-demo`.
- Point SiteNav, landing CTA, and command palette at `/live-demo`.

### Tests and docs

- Update smoke and live-demo browser harness expectations.
- Document the `/live-demo` route in `documentation/wasm.md`, `getting-started.md`, and `deferred.md`.
- Rebuild committed `docs/` assets so the published site matches.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None |
| **Memory** | None |
| **Behavior / correctness** | Public site hash route changes from `/#/wasm` to `/#/live-demo`; old `/#/wasm` redirects |
| **API / CLI** | None |
| **Dependencies** | None |
| **Binary size / build time** | None |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| Site URL `/#/wasm` | Use `/#/live-demo`. Old `/#/wasm` links redirect automatically. |
| Asset path `wasm/` | Unchanged. Runtime still loads `frontend/public/wasm/`. |

---

## Test plan

- [x] `npm --prefix frontend run build`
- [x] `npm --prefix frontend run lint`
- [x] `npm --prefix frontend test` (smoke, including `/live-demo` route + `/wasm` redirect assertions)
- [x] `make claim-scan`
- [x] Puppeteer check: `/#/live-demo` loads, `/#/wasm` redirects, landing CTA navigates, showcase still loads, mobile viewport renders live demo
- [ ] `make test` (not required for this frontend-only route rename; run if CI asks)

### Commands

```sh
npm --prefix frontend run build
npm --prefix frontend run lint
npm --prefix frontend test
make claim-scan
```

---

## Screenshots / sample output

```
Puppeteer checks:
- live-demo route hash=#/live-demo, title="Convert HTML in your browser."
- wasm redirect hash=#/live-demo
- landing CTA "Try Live Demo" -> #/live-demo
- showcase still loads
- mobile live-demo page width ~359px at 390 viewport
```

---

## Related issues

- Relates to #68 (WASM live demo shipping)

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled with real ticket IDs
- [x] Filled body committed under `plans/PR/pr-live-demo-route.md`

---

## Follow-ups (out of scope)

- Renaming `frontend/public/wasm/` asset directory or `make wasm` targets

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
| `.js` | 6 | 17 | 17 |
| `.jsx` | 4 | 5 | 4 |
| `.md` | 4 | 131 | 3 |
| `.mjs` | 2 | 4 | 2 |
| **Total** | **17** | **158** | **27** |
