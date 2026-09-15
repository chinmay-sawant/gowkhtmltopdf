## Summary

- `paintDecoration` forced an underline on every `a[href]` even when the cascade resolved to `text-decoration: none`, so links the author explicitly left undecorated still printed with a rule under them.
- Decoration now comes from the resolved style, and forced PDF link affordance stays fully behind the existing opt-in `--print-link-underline` flag.

---

## Motivation / context

- Plans: none; direct bug fix.
- Issues: see **Related issues**.
- The force rule predates `--print-link-underline`. Once that flag existed, the layout path was the remaining place that ignored author `text-decoration`.

---

## Changes

### `internal/layout/inline_paint.go`

- Replaced the `forceLinkUnderline(item)` href check in `paintDecoration` with `hasUnderline(style)`, which reads the cascade: `TextDecoration == underline` or `TextDecorationLine` containing `underline`.
- Removed `forceLinkUnderline`. Forced underlines for operators continue through the `--print-link-underline` path in style resolution (`internal/layout/style.go`, `internal/layout/style_memo.go`), which this PR does not change.
- Kept the existing guards: a visible `border-bottom` still suppresses the CSS underline, and line-through / overline links keep only those decorations.

### Tests (`internal/layout`)

- `link_print_repro_test.go`: a `text-decoration: none` sheet now asserts zero underline `OpLine` ops.
- `underline_coalesce_test.go`: renamed and reworked `TestUnderlineCoalesceHrefForceAcrossChunks` to `TestUnderlineCoalesceHrefAcrossChunks` with an explicit CSS underline, and added `TestLinkTextDecorationNoneHonored` covering `none` on a mailto and an https link.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None; one helper swap in the paint path. |
| **Memory** | None. |
| **Behavior / correctness** | Links with `text-decoration: none` no longer print underlines. Previous forced behavior is available with `--print-link-underline`. |
| **API / CLI** | None; the flag already exists. |
| **Dependencies** | None. |
| **Binary size / build time** | Negligible. |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| Documents that relied on the implicit forced link underline change appearance | Pass `--print-link-underline` to restore the previous output. |

---

## Test plan

- [x] `go build ./...`
- [x] `go vet ./internal/layout/`
- [x] `go test ./internal/layout/ -count=1`
- [x] `make golden`
- [x] `make test` (full suite)
- [x] `make lint`

### Commands

```sh
go build ./...
go vet ./internal/layout/
go test ./internal/layout/ -count=1
make golden
make test
make lint
```

---

## Screenshots / sample output

```
ok  	github.com/chinmay-sawant/gowkhtmltopdf/internal/layout	3.161s
ok  	github.com/chinmay-sawant/gowkhtmltopdf/internal/convert	5.704s
golangci-lint v1.64.8: clean
size-check: clean (3 allowlisted over-limit files)
frontend: src/data lint clean (11 content pages, 61 showcase items)
```

---

## Related issues

- None filed; behavior was reported and fixed directly.

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied (`bug`)
- [x] Related issues filled with real ticket IDs
- [x] Filled body committed under `plans/PR/pr-honor-text-decoration-none.md`

---

## Follow-ups (out of scope)

- None.

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
| `.go` | 3 | 65 | 31 |
| `.md` | 1 | 126 | 0 |
| **Total** | **4** | **191** | **31** |
