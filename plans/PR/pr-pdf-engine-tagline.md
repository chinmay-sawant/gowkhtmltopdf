## Summary

Reposition product copy from “HTML template engine” to **“A PDF engine based on HTML templates … — without any wrappers”** across the README, documentation, docs site, and meta tags so the primary claim leads with PDF output rather than HTML templating.

---

## Motivation / context

- Plans: n/a (copy/positioning refresh)
- Issues: see **Related issues**

The landing hero and surrounding docs still framed the project as an HTML template engine first. The preferred product sentence leads with PDF generation from HTML templates and explicitly states there are no wrappers.

---

## Changes

### Product tagline

- Landing hero: `A PDF engine based on HTML templates for invoices, certificates, storybooks, posters, statements, and tables — without any wrappers.`
- README intro rewritten to the same PDF-engine-first framing
- Overview, fidelity, CLI, deferred, performance, and comparison docs aligned

### Docs site / frontend

- Overview, About, and Compatibility content JSON updated
- `frontend/index.html` and rebuilt `docs/index.html` meta / Open Graph / Twitter descriptions updated
- `npm run build` regenerated committed `docs/` assets

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None |
| **Memory** | None |
| **Behavior / correctness** | None — docs and site copy only |
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

- [x] `npm --prefix frontend run build`
- [x] `npm --prefix frontend run test`
- [ ] Spot-check landing hero and meta description text after deploy
- [ ] Confirm no remaining “An HTML template engine for invoices…” on live product surfaces (historical `plans/` / changelog entries intentionally left alone)

### Commands

```sh
npm --prefix frontend run build
npm --prefix frontend run test
```

---

## Screenshots / sample output

Landing lede now reads:

```
A PDF engine based on HTML templates for invoices, certificates, storybooks, posters, statements, and tables — without any wrappers.
```

---

## Related issues

- Refs none (docs/copy refresh; no tracking issue)

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled
- [x] Filled body under `plans/PR/pr-pdf-engine-tagline.md`

---

## Follow-ups (out of scope)

- Rewriting historical release notes under `plans/0.2.*/` and older CHANGELOG entries that still say “HTML template engine”

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
