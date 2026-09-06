## Summary

Fixture-61 rows 108-111 applied list properties to plain `div` elements, so the audit PDF proved nothing about lists. Rewriting those cells with real `ul`/`ol` demos exposed an engine bug: `ul { list-style: none }` still painted discs. This PR fixes the cascade and ships the corrected fixture plus a regression test.

---

## Motivation / context

- Plans: `plans/PR/issue-list-style-cascade-body.md`
- Issues: see **Related issues**

---

## Changes

### Engine fix

- `internal/layout/style_cascade.go`: added `expandListStyleDeclaration`, called from `applyCascadeDeclaration`.
- The `list-style` shorthand now expands into `list-style-type`, `list-style-position`, and `list-style-image` at cascade time, carrying the shorthand's own origin, specificity, and order. Previously the shorthand and the UA `ul { list-style-type: disc }` longhand coexisted as separate raw keys, so the later-applied UA longhand won regardless of origin.
- Only components present in the value expand; omitted parts keep the prior cascade result. Same-origin source order holds both ways between shorthand and longhand.

### Regression test

- `internal/layout/style_cascade_test.go`: added `TestCascadeListStyleShorthandBeatsUADisc`.
- Covers author-sheet shorthand vs UA disc, `li` inheritance, inline style, zero `OpBullet` paint proof, and source order in both directions (`square` vs `decimal`).

### Fixture + sample

- `testdata/golden/fixture-61-implemented-props-b.html`: rows 108-111 now use real lists (`none`, `square` fallback for image `none`, `inside` with wrapping text, `decimal` on `ol`).
- `output/fixture-61-implemented-props-b.pdf`: regenerated with the audit font path.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | Negligible: tokenizes the short `list-style` value once per declaration. |
| **Memory** | Negligible: one small short-lived slice per `list-style` declaration. |
| **Behavior / correctness** | Author `list-style` on `ul`/`ol` now beats the UA default instead of losing to it. |
| **API / CLI** | None. |
| **Dependencies** | None. |
| **Binary size / build time** | None. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [x] `go build ./...` clean.
- [x] `go test ./internal/layout -count=1` green (includes the new test).
- [x] `go test ./internal/convert -run 'TestGoldenCorpusAllFixtures/fixture-61-implemented-props-b' -count=1` passes (8 pages, envelope 5-8).
- [x] Regenerated PDF text check: all demo texts present (`no bullet one/two`, `square one/two`, `inside marker flows`, `first/second item` with `1.` `2.` markers).
- [ ] Full golden suite has one known failure: `fixture-57-vanguard-telemetry-audit` expects 9 pages but renders 37. It fails identically with this fix reverted, so it is pre-existing and unrelated.

---

## Screenshots / sample output

`output/fixture-61-implemented-props-b.pdf` regenerated in-tree; rows 108-111 now show no bullets, square bullets, inside bullets with wrapped text, and decimal `1.` `2.` numbering.

---

## Related issues

- Closes #63

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled with real ticket IDs
- [x] Filled body committed under `plans/PR/pr-<slug>.md` when process-gated

---

## Follow-ups (out of scope)

- `list-style-image` does not inherit parent to child (pre-existing gap, separate change).
- No full shorthand reset: omitted components keep the prior cascade result instead of resetting to initial.
- Fixture-61 grid rows 74 and 76-81 apply item-placement props to the container; output is correct but the demos are vacuous.

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
