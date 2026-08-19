## Summary

Refresh the root README so the Library section describes the **shipped** 0.2.4 `Document` / `ImageDocument` API (no more “0.2.4 target”), and the Performance section matches the **2026-08-19** benchmark snapshot for 0.2.4 vs wkhtmltopdf plus WeasyPrint / Puppeteer highlights.

---

## Motivation / context

- Plans: n/a (README honesty pass after v0.2.4)
- Issues: see **Related issues**

After the tagline PR merged, the README still described the library as a “0.2.4 target” with old exports remaining, and still cited a 2026-08-14 / 0.2.1 wkhtmltopdf table. The checked-in performance docs already have the 2026-08-19 0.2.4 numbers.

---

## Changes

### Library section

- Renamed `## Library (0.2.4 target)` → `## Library`
- Removed “plan / target symbols / old exports remain until the hard break closes” wording
- Documented the shipped API; example includes `PageSize` and points at the migration guide

### Performance section

- Snapshot date **2026-08-19**, binary **0.2.4**
- Updated wkhtmltopdf comparison rows (2 / 10 / 100 / 500) from `cli-compare.md` / `documentation/performance.md`
- Added compact WeasyPrint and Puppeteer/Chrome speedup table
- Linked the full comparison artifacts under `testdata/golden/benchmarks/`

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Docs only — numbers copied from existing 2026-08-19 snapshots |
| **Memory** | None |
| **Behavior / correctness** | None |
| **API / CLI** | README wording only; API already shipped in 0.2.4 |
| **Dependencies** | None |
| **Binary size / build time** | None |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [x] Diff README against `documentation/performance.md` and `testdata/golden/benchmarks/cli-compare.md`
- [x] Confirm README no longer contains `Library (0.2.4 target)` or “old exports remain”
- [ ] Visual pass of rendered README on GitHub after PR opens

### Commands

```sh
# Numbers source of truth:
sed -n '44,106p' documentation/performance.md
sed -n '1,21p' testdata/golden/benchmarks/cli-compare.md
```

---

## Screenshots / sample output

README Performance now leads with:

```
Current snapshot (2026-08-19): … gowkhtmltopdf 0.2.4 versus wkhtmltopdf 0.12.6.1
2 pages: 17 ms vs 259 ms (15.5x)
```

---

## Related issues

- Relates to #54 (tagline copy landed; README still needed a numbers/API honesty pass)

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled
- [x] Filled body under `plans/PR/pr-readme-benchmarks-0.2.4.md`

---

## Follow-ups (out of scope)

- `documentation/getting-started.md` still has a `## Library (0.2.4 target)` heading with similar stale “hard break closes” prose — can be cleaned in a follow-up docs PR

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
