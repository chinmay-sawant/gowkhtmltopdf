## Summary

Unblock `make lint` on `master`. CI's `test + lint` job failed after #48 because `//nolint:revive` on `Page.AddLinkURI` and `Page.AddLinkDest` was unused under `nolintlint`. Those helpers now return the existing exported `ObjRef` alias instead of the unexported `objRef` name.

---

## Motivation / context

- Plans: none (lint-only follow-up)
- Issues: see **Related issues**
- CI: `master` run after #48 — `internal/pdf/pdf.go:453` and `:471`, unused `//nolint:revive` (`nolintlint`)

Local `revive` still flags `unexported-return` when the signature uses `objRef`. Returning `ObjRef` (`type ObjRef = objRef`) satisfies both environments without a suppression.

---

## Changes

### PDF page link helpers

- `Page.AddLinkURI` and `Page.AddLinkDest` return `ObjRef` instead of `objRef`
- Remove the unused `//nolint:revive` comments
- No call-site changes (`ObjRef` is an alias; convert/layout already use `pdf.ObjRef`)

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None |
| **Memory** | None |
| **Behavior / correctness** | None. Same underlying type |
| **API / CLI** | Internal writer signature uses the already-exported alias name. No CLI change |
| **Dependencies** | None |
| **Binary size / build time** | None |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | `type ObjRef = objRef`; existing `pdf.ObjRef` callers are unchanged |

---

## Test plan

- [x] `make lint` (golangci-lint v1.64.8, clean)
- [x] `go test ./internal/pdf/ ./internal/convert/ ./internal/layout/ -count=1`
- [ ] `make test` (full suite; CI)
- [ ] `CGO_ENABLED=0 go build` (CI static-build job)

### Commands

```sh
make lint
go test ./internal/pdf/ ./internal/convert/ ./internal/layout/ -count=1
```

CI jobs: `test + lint`, `static build (CGO_ENABLED=0)`, `race (hot packages)`, `frontend production build`.

---

## Screenshots / sample output

```
make lint
golangci-lint has version v1.64.8 built with go1.26.4
golangci-lint run ./...
# exit 0

go test ./internal/pdf/ ./internal/convert/ ./internal/layout/ -count=1
ok  gowkhtmltopdf/internal/pdf      2.139s
ok  gowkhtmltopdf/internal/convert  6.765s
ok  gowkhtmltopdf/internal/layout   3.928s
```

---

## Related issues

- None. No ticket for this CI lint failure. Relates to the `test + lint` job on `master` after #48.

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled (no ticket; noted above)
- [x] Filled body committed under `plans/PR/pr-lint-objref.md`

---

## Follow-ups (out of scope)

- Align local vs CI `revive` / default-exclusion behavior so unused-`nolint` cannot regress only on GitHub Actions

---

## Reviewer checklist

- [ ] Behavior matches summary and test plan
- [ ] No unrelated changes in diff
- [ ] Public API / CLI changes documented
- [ ] New rules have fixture coverage when applicable
- [ ] PR has assignee and labels
- [ ] Related issues use correct Closes/Relates keywords
- [ ] No secrets or generated artifacts committed
