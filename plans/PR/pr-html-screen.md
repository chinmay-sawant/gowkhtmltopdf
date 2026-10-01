## Summary

A second Go module can now ask this engine for a PNG of an HTML screen and for the element rectangles on that PNG. The same call also exposes a small owned HTML tree. Document and ImageDocument stay at the module root.

---

## Motivation / context

- Plans: none. This is the public seam for the local go-gpui app.
- Issues: see **Related issues**

---

## Changes

### Screen frames

- `screen.Render` parses the HTML, lays it out at a fixed CSS-pixel size, and returns the PNG plus each element's id, tag, `data-action`, text, and box.
- `internal/layout/placed.go` walks those boxes in document order. It does not need PDF paint.

### Markup tree

- `markup.Parse` returns an owned HTML tree. It does not run scripts.

### File size

- `imageout.RenderLayout` moved into `internal/imageout/frame.go`. `imageout.go` is under the 2,000-line cap and left the allowlist. `AGENTS.md` lists the two files that are still over the cap.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | One extra walk of the element boxes when a caller asks for a screen frame. PDF conversion is unchanged. |
| **Memory** | The screen path keeps the PNG and the box list for that call. |
| **Behavior / correctness** | PDF and image output for existing callers is unchanged. |
| **API / CLI** | New packages `screen` and `markup`. Root `Document` and `ImageDocument` are unchanged. |
| **Dependencies** | None added. |
| **Binary size / build time** | No new dependency. Two small packages. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [x] `make test`
- [x] `make lint` / `go vet`
- [ ] `make build` (when binary output is part of the change)
- [ ] `make run` wall time vs baseline (hard &lt; 400ms; soft ±50ms of reference)
- [ ] `make reference-metrics` / gopdfsuit hard metrics if detector surface changed

### Commands

```sh
make lint
make test
```

Both exited 0 before this pull request was opened. `make build` and `make run` were not part of this change.

---

## Screenshots / sample output

```
(not a CLI output change)
```

---

## Related issues

- No tracked issue.

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [ ] Related issues filled with real ticket IDs
- [x] Filled body committed under `plans/PR/pr-<slug>.md` when process-gated

---

## Follow-ups (out of scope)

- The go-gpui window app lives in its own repository.

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
| `.go` | 9 | 634 | 41 |
| `.md` | 2 | 125 | 3 |
| `.txt` | 1 | 0 | 1 |
| **Total** | **12** | **759** | **45** |
