# Phase 02 — Resource Loader & Network

> **Parent:** `plans/00-canonical-pure-go-rewrite.md`  
> **Status:** not started  
> **Estimated effort:** 3–6 weeks solo  
> **Depends on:** Phase 1 settings  
> **Unblocks:** Phase 4 resource fetch; Phase 6 HTML HF loads

---

## Overview

Reimplement MultiPageLoader **orchestration** in pure Go: concurrent loads, cookies, proxy, auth, local ACL, error policies. **Not** a browser — after fetch, hand HTML/CSS/image bytes to layout.

## Executive Summary

Loading semantics are well-bounded (~3–8 person-weeks). Rendering is not. MVP: `jsdelay` as post-load sleep; `windowStatus`/`runScript` warn-and-ignore until JS exists (it will not in MVP).

---

## Checklist

### 2.1 Types (`internal/load`)
- [ ] `Loader` with add resource (URL, LoadPage, optional inline HTML)
- [ ] `Resource` result: body, final URL, content-type, HTTP code, skip flag, error
- [ ] Progress callback aggregation (average of N resources)

### 2.2 Input kinds
- [ ] Inline HTML → temp file or in-memory with synthetic `file://` base (document choice; prefer in-memory + base URL)
- [ ] stdin `-` → read all → same as inline
- [ ] Local path if exists → `file://` absolute
- [ ] Absolute URL / host:port / default `http://` guess (`guessUrlFromString` parity tests)
- [ ] data: URLs supported for primary input

### 2.3 HTTP
- [ ] GET default; POST with urlencoded or multipart from `PostItem`
- [ ] Custom headers on main request; optional repeat on subresources
- [ ] Basic auth (username/password)
- [ ] Cookie jar load/save file format (document Netscape-ish subset)
- [ ] Per-load cookies applied
- [ ] Proxy http/socks5 + bypass host list
- [ ] Client cert PEM key+crt
- [ ] TLS: configurable — **do not silently ignore all cert errors by default** (upstream does; we harden with flag `--insecure` if needed)
- [ ] Optional disk cache directory (simple file cache)
- [ ] Timeouts, max body size, max redirects

### 2.4 Local ACL
- [ ] Default deny local file reads
- [ ] Allow main resource path + `--allow` prefixes (parent walk like Qt)
- [ ] Subresource file: checks against allowlist

### 2.5 Subresource fetch (for layout)
- [ ] API: resolve relative URL against base; fetch CSS/images with same policy
- [ ] Media vs non-media error handling by extension list (css,js,svg,png,jpg,jpeg,gif)

### 2.6 Wait stubs
- [ ] `jsdelay`: sleep after primary load complete
- [ ] `windowStatus`: log warning, no-op
- [ ] `runScript`: log warning, no-op

### 2.7 Tests
- [ ] httptest: headers, auth, cookies, POST, error codes
- [ ] ACL unit tests with temp dirs
- [ ] Concurrent multi-URL completion
- [ ] Exit code mapping helper: 404→2, 401→3

### 2.8 Closure
- [ ] `make test` / `make lint` pass
- [ ] Proof commands recorded in parent ledger when closed

---

## Dependencies

| Upstream | Role |
|----------|------|
| `multipageloader.cc` | Behavior source of truth |
| Phase 1 `LoadPage` | Settings |

## Risks

- Cookie domain/path weak semantics in upstream extras — document Go behavior.
- SOCKS5 via pure stdlib: use `golang.org/x/net/proxy` is **forbidden** under stdlib-only → implement HTTP proxy first; SOCKS5 may be `[~]` deferred or hand-rolled.
