## Summary

Rebuild the committed browser WASM artifact from the post-#71 tree. The copies at `frontend/public/wasm/gowkhtmltopdf.wasm` and `docs/wasm/gowkhtmltopdf.wasm` were last built at `a42f26e` (0.2.6 release), before PR #71 changed `internal/layout/op_ownership.go`, so the live demo was serving a layout engine without the mitered-border fix.

---

## Motivation / context

- The live demo loads `frontend/public/wasm/gowkhtmltopdf.wasm`; the published site loads `docs/wasm/gowkhtmltopdf.wasm`. Both are checked in.
- PR #71 changed layout code that is compiled into that artifact, so the committed binary fell behind the tree.
- Plans: `plans/0.2.6/86-canonical-0.2.6-wasm.md` (WASM-ASSET-01/02, artifact ownership). `RELEASE.md` holds the release stamp check.
- Issues: none filed; follow-up to #71.

---

## Changes

### Browser artifact

- Run `make wasm` to rebuild `frontend/public/wasm/gowkhtmltopdf.wasm` with `-X main.wasmVersion=0.2.6` (30,027,708 -> 30,029,477 bytes, sha256 `6dcf3fbd...` -> `5c65373d...`).
- Run `npm --prefix frontend run build` so `docs/wasm/gowkhtmltopdf.wasm` carries the same bytes. `git status` shows no other `docs/` changes.

---

## Impact

| Area | Impact |
|------|--------|
| **Performance** | None |
| **Memory** | None |
| **Behavior / correctness** | Browser PDF/PNG/JPEG output now includes the #71 mitered-border chrome fix; the version stamp stays 0.2.6 |
| **API / CLI** | None; the JS bridge contract is unchanged |
| **Dependencies** | None |
| **Binary size / build time** | WASM artifact grows 1,769 bytes |

---

## Breaking changes / migration

| Item | Migration |
|------|-----------|
| None | - |

---

## Test plan

- [x] `bash scripts/check-wasm-contract.sh`
- [x] `npm --prefix frontend run build`
- [x] `make wasm-test` (contract, WASM build, frontend lint, build, smoke, and browser live demo for PDF, PNG, and JPEG)
- [ ] `make test` / `make lint` / race (no Go source changed; CI runs them on this PR)

### Commands

```sh
make wasm
npm --prefix frontend run build
make wasm-test
```

---

## Screenshots / sample output

```
sha256 frontend/public/wasm/gowkhtmltopdf.wasm
  before: 6dcf3fbd3363636509ba5f8164d46f483d7cf3c9003c1cd4550dba81d6542a51
  after:  5c65373defcc9d634bd7e98d721aa88da3995a714311e5f05ab9273d28756fc5
  docs copy matches the frontend copy after the build

make wasm-test:
  ok   github.com/chinmay-sawant/gowkhtmltopdf/bindings/wasm
  WASM contract tests and JS/WASM build passed.
  All frontend smoke tests passed successfully!
  Live demo browser smoke passed for PDF, PNG, JPEG, validation, and mobile layout.
```

---

## Related issues

- Relates to #71

---

## PR metadata checklist (author)

- [x] Self-assigned (`--assignee @me`)
- [x] Labels applied
- [x] Related issues filled with real ticket IDs
- [x] Filled body committed under `plans/PR/pr-rebuild-wasm.md`

---

## Follow-ups (out of scope)

- No CI check compares the committed WASM against a fresh build; consider one if the artifact drifts again.

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
| `.md` | 1 | 120 | 0 |
| `.wasm` | 2 | Binary | Binary |
| **Total** | **3** | **120** | **0** |
