# Pull Request

## Summary

Adds browser-local HTML conversion through WebAssembly for PDF, PNG, and JPEG output. The Try Live Demo page now loads a usable sample automatically, previews every generated page, supports downloads, and explains browser limits around system fonts and compliance validation.

## Motivation and context

The 0.2.6 WASM plan calls for a browser adapter that exercises the same Go rendering pipeline as the native tools:

- Plan: `plans/0.2.6/86-canonical-0.2.6-wasm.md`
- Related issue: No related issue was provided for this branch.

The live demo gives users a quick way to inspect output without uploading HTML. Native binaries and library integrations remain the path for system-font access and compliance validation.

## Changes

- Added the WASM conversion contract and JavaScript adapter for PDF, PNG, and JPEG output.
- Added image padding support with a 20px default for browser image output.
- Preserved transparent backgrounds for PNG output and composited JPEG output onto white.
- Added multi-page image rendering, page-aware previews, single-file downloads, and ZIP downloads for image pages.
- Added the Try Live Demo page with a wider layout, separate HTML and CSS editors, automatic sample loading, and automatic conversion when a sample or output format changes.
- Added five curated paired HTML/CSS samples, including multi-page operational and audit documents.
- Added an optional catalog of fixtures fetched from the GitHub golden directory. The curated samples remain available if GitHub is unavailable or rate-limited.
- Added a scrollable PDF preview and an image preview that shows rendered pages without PDF-only page controls.
- Renamed frontend-facing WASM demo labels, styles, scripts, and components to Live Demo terminology while retaining the `/wasm` route.
- Added the Fonts and compliance notice to the live demo.
- Updated documentation, generated static-site assets, CI configuration, and the WASM test target.

## Impact

| Area | Impact |
| --- | --- |
| Performance | WASM startup and rendering now run in the browser demo. The optional GitHub fixture catalog is fetched only when available. |
| Memory | Multi-page previews retain the rendered page blobs until the output is replaced or reset. |
| Behavior/correctness | Browser output now handles page padding, transparent PNG canvases, white JPEG backgrounds, page-aware previews, and ZIP offsets for separate image pages. |
| API/CLI | No native CLI or Go library breaking change. Adds a browser-facing WASM conversion contract. |
| Dependencies | No new direct Go dependency. Frontend dependencies remain managed by `frontend/package-lock.json`. |
| Binary/build | Includes the browser WASM binary and refreshed generated site output under `docs/`. |

## Breaking changes and migration

None for the native CLI or Go library. Frontend-facing internal WASM names were renamed to Live Demo terminology; the public `/wasm` route remains unchanged.

## Test plan

- [x] `make test`
- [x] `make lint`
- [x] `make claim-scan`
- [x] `make golden`
- [x] `npm --prefix frontend test`
- [x] `npm --prefix frontend run build`
- [x] `bash scripts/check-wasm-contract.sh`
- [ ] `make test-race` was not run in this PR preparation pass.
- [ ] `make wasm-test` reaches the default and curated conversion flows but currently stops at the padded-PNG browser assertion after repeated conversions: `padded PNG conversion timed out`, with the expected dimensions check still pending.

### Commands run

```text
make test
make lint
make claim-scan
make golden
npm --prefix frontend test
npm --prefix frontend run build
bash scripts/check-wasm-contract.sh
```

## Screenshots or sample output

No screenshot is attached. The browser smoke test drives the live page and verifies the route, generated assets, output controls, curated sample conversions, and image-output flow. Its current remaining failure is recorded above.

## Related issues

No related issue was provided for this branch.

## PR metadata checklist

- [x] Self-assigned with `--assignee @me`.
- [x] Labeled with `enhancement` and `documentation`.
- [x] PR body saved at `plans/PR/pr-wasm-live-demo.md`.
- [x] No issue reference was invented.

## Follow-ups

- Investigate the padded-PNG browser assertion timing and dimension mismatch after repeated conversions.
- Add a deterministic local fixture path for browser checks that need to exercise the remote golden catalog without depending on GitHub availability.

## Reviewer checklist

- [ ] Reviewed the WASM contract and native conversion boundary.
- [ ] Reviewed multi-page image rendering and ZIP generation.
- [ ] Reviewed the live demo interaction and generated static-site assets.
- [ ] Confirmed the remaining browser smoke-test failure is understood before merge.

## Diff stat

| Metric | Value |
| --- | ---: |
| Files changed | 98 |
| Insertions | 5,435 |
| Deletions | 162 |
