## Summary

Rewrites the README Development section in plain voice: what the engine is,
who owns what between humans and AI tools, and where the proof lives. Also
clears the 100+ `make lint` findings on this branch (the stacked fixture
work predates the lint fixes that later landed on master via PR #66), so the
PR's CI lint gate goes green. No behavior change; all refactors are
provably neutral via `make test` and `make golden`.

---

## Motivation / context

- Plans: none (docs cleanup plus lint hygiene on the same branch)
- Issues: see **Related issues**

---

## Changes

### README Development section

- Replaces the dense "clean-room ... pairing ... for accelerated
  implementation" paragraph with short plain sentences.
- States the human/AI split directly: humans own architecture and the
  pipeline, AI tools help with drafts, golden-fixture checks, and tests.
- Points performance proof at `documentation/performance.md`,
  `testdata/golden/benchmarks/README.md`, and the CI perf budget test.
- Fixes grammar in the follow-up sentence about manual validation of the
  50+ sample templates.

### Lint cleanup (same branch, CI gate)

- Exhaustiveness: explicit `OpUnknown` / `NodeUnknown` / `line.Unknown`
  cases on every flagged switch (grouped with existing no-op arms;
  zero-value ops carry no ink and were already skipped).
- `err113`: static sentinel errors plus `%w` wraps in layout, paint,
  prepare, outline, and load; rendered messages are byte-identical.
- Dead code deleted: `(*runContext).renderObjects`, the unused
  `log` parameter of `drawHeadersFootersResult`,
  `oldCollectBodyNavigation`, and four unused css index helpers.
- Complexity refactors (behavior-neutral extractions): `settleBeforeAlways`
  plus `assignFlowPages` in pagination, `decodeImageBytes` in filters,
  `applyBodyLink` in internal links, `matchSubstringOp` plus
  `attrWantValue` in attribute matching, `buildAttrSelector` in the
  selector parser, `paintWidgetControl` in block building.
- Test hygiene: `t.Parallel` additions, short-name renames, line wraps,
  `//nolint:testpackage` per repo convention, focused-fixture
  `//nolint:exhaustruct` with reasons.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None |
| **Memory** | None |
| **Behavior / correctness** | None |
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

- [x] `make claim-scan` (clean, exit 0)
- [x] `make lint` (clean, exit 0, including frontend)
- [x] `make test` (full suite green, exit 0)
- [x] `make golden` (all 61 fixtures pass, exit 0)

### Commands

```sh
make claim-scan
```

---

## Screenshots / sample output

```
claim-scan: clean
```

---

## Related issues

- None

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`documentation`)
- [x] Related issues filled (none exist for this cleanup)
- [x] Filled body committed under `plans/PR/pr-readme-development-voice.md`

---

## Follow-ups (out of scope)

- None

---

## Reviewer checklist

- [ ] Wording matches the unslop pass, no new claims
- [ ] No unrelated changes in diff
- [ ] PR has assignee and labels
