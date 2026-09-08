## Summary

Rewrites the README Development section in plain voice: what the engine is,
who owns what between humans and AI tools, and where the proof lives. No
behavior change, docs only.

---

## Motivation / context

- Plans: none (small docs cleanup)
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
- [ ] `make test` (not run: docs-only change, no Go files touched)
- [ ] `make lint` (not run: docs-only change)
- [ ] `make golden` (not run: docs-only change)

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
