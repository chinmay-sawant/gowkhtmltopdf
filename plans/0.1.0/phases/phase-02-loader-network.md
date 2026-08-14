# Phase 02 - Resource Loader & Network

> **Parent:** `plans/00-canonical-pure-go-rewrite.md`  
> **Status:** complete (2026-08-03)  
> **Estimated effort:** 3–6 weeks solo  
> **Depends on:** Phase 1 settings  
> **Unblocks:** Phase 4 resource fetch; Phase 6 HTML HF loads

---

## Overview

Reimplement MultiPageLoader **orchestration** in pure Go: concurrent loads, cookies, proxy, auth, local ACL, error policies. **Not** a browser - after fetch, hand HTML/CSS/image bytes to layout.

## Executive Summary

Loading semantics are well-bounded (~3–8 person-weeks). Rendering is not. MVP: `jsdelay` as post-load sleep; `windowStatus`/`runScript` warn-and-ignore until JS exists (it will not in MVP).

---

## Checklist

### 2.1 Types (`internal/load`)
- [x] `Loader` with add resource (URL, LoadPage, optional inline HTML)
- [x] `Resource` result: body, final URL, content-type, HTTP code, skip flag, error
- [x] Progress callback aggregation - `OnProgress` hook; full aggregate progress wired in Phase 5 convert

### 2.2 Input kinds
- [x] Inline HTML → in-memory with synthetic `inline:` base (documented choice: in-memory + base URL)
- [x] stdin `-` → read all → same as inline (wired in convert Phase 5)
- [x] Local path if exists → `file://` absolute
- [x] Absolute URL / host:port / default `http://` guess (`guessUrlFromString` parity tests)
- [x] data: URLs supported for primary input

### 2.3 HTTP
- [x] GET default; POST with urlencoded from `PostItem`
- [x] Custom headers on main request; repeat flag stored (`repeatCustomHeaders` for subresources)
- [x] Basic auth (username/password)
- [~] Cookie jar load/save file format - in-memory jar + per-request cookies live; jar-file persistence deferred (Phase 9)
- [x] Per-load cookies applied
- [~] Proxy http(s) wired via `net/http` transport; SOCKS5 deferred (stdlib-only; no `golang.org/x/net/proxy`)
- [x] Client cert PEM key+crt (`ApplyCert`)
- [x] TLS: hardened - cert errors NOT silently ignored; `--insecure` flag opt-in
- [~] Optional disk cache directory - deferred (Phase 9)
- [x] Timeouts, max body size, max redirects

### 2.4 Local ACL
- [x] Default deny local file reads
- [x] Allow main resource path + `--allow` prefixes (parent walk like Qt)
- [x] Subresource file: checks against allowlist

### 2.5 Subresource fetch (for layout)
- [x] API: resolve relative URL against base; fetch CSS/images with same policy
- [~] Media vs non-media error handling by extension list - load-error policy applies to all resources; media-extension refinement deferred (Phase 9)

### 2.6 Wait stubs
- [x] `jsdelay`: sleep after primary load complete
- [x] `windowStatus`: log warning, no-op
- [x] `runScript`: log warning, no-op

### 2.7 Tests
- [x] httptest: headers, auth, cookies, POST, error codes
- [x] ACL unit tests with temp dirs
- [x] Concurrent multi-URL completion
- [x] Exit code mapping helper: 404→2, 401→3

### 2.8 Closure
- [x] `make test` / `make lint` pass
- [x] Proof commands recorded in parent ledger when closed
      Evidence 2026-08-03: `go test ./internal/load/` 13 tests pass; `make test` + `make lint` green.

---

## Dependencies

| Upstream | Role |
|----------|------|
| `multipageloader.cc` | Behavior source of truth |
| Phase 1 `LoadPage` | Settings |

## Risks

- Cookie domain/path weak semantics in upstream extras - document Go behavior.
- SOCKS5 via pure stdlib: use `golang.org/x/net/proxy` is **forbidden** under stdlib-only → implement HTTP proxy first; SOCKS5 may be `[~]` deferred or hand-rolled.
