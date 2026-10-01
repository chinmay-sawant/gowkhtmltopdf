## Summary

Another module can parse HTML, apply CSS, and lay the document out without writing a PDF. `html.Parse` keeps the tree, `css.Apply` styles that tree, and `layout.Lay` returns the boxes and the painted image. `Document` and `ImageDocument` stay the PDF and image writers.

---

## Motivation / context

- Plans: none. go-gpui needs the same parse, style, and layout path this engine uses.
- Issues: see **Related issues**

---

## Changes

### HTML

- `html.Parse` parses the document and keeps the engine tree.
- `Find` returns an element by id.

### CSS

- `css.Parse` parses one stylesheet.
- `css.Apply` collects `<style>` from the parsed document and appends `Options.Extra`.
- Linked style sheets and images are not fetched.

### Layout

- `layout.Lay` places the styled document and paints that placement.
- `Boxes` are CSS pixels. `Image` is the painted picture.
- Pagination and PDF writing are not called.

### Screen and markup

- `screen.Render` is `html.Parse`, then `css.Apply`, then `layout.Lay`, then a PNG encode.
- `markup.Parse` stays a detached copy of the tree. `css.Apply` does not accept it.

### Documentation

- `documentation/library-api.md`, `documentation/overview.md`, `documentation/architecture.md`, and the library-api deep dive describe the three packages.
- `README.md` and `CONTRIBUTING.md` point at the same path.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | The layout path walks element boxes for callers that ask for them. PDF conversion is unchanged. |
| **Memory** | A layout call keeps the picture and the box list for that call. |
| **Behavior / correctness** | PDF and image output for existing callers is unchanged. |
| **API / CLI** | New packages `html`, `css`, and `layout`. `screen` and `markup` stay. Root `Document` and `ImageDocument` are unchanged. |
| **Dependencies** | None added. |
| **Binary size / build time** | No new dependency. Three small packages. |

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
make claim-scan
```

`make lint` and `make test` exited 0 on the engine packages. `make claim-scan` exited 0 after the documentation edit. `make build` and `make run` were not part of this change.

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

- go-gpui calls these packages from `Page.Redraw`. That app is a separate repository.

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
| `.go` | 23 | 1312 | 41 |
| `.md` | 8 | 206 | 3 |
| `.txt` | 1 | 0 | 1 |
| **Total** | **32** | **1518** | **45** |
