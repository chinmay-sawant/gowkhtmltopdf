# Exploration 03 — Loader & WebKit Surface

> **Agent:** explore · multipage loader  
> **Primary files:** `multipageloader.cc` (~853 LOC), `loadsettings.*`, `websettings.*`

---

## Load pipeline

1. `addResource` for each URL/inline/stdin  
2. Concurrent `ResourceObject::load`  
3. WebKit network + layout  
4. On finish: runScript → windowStatus poll → jsdelay (default 200ms)  
5. Or early finish on `window.print()`  
6. Aggregate `loadFinished`  

**No `document.ready` wait** beyond WebKit loadFinished.

## LoadPage defaults (important)

| Field | Default |
|-------|---------|
| jsdelay | 200 |
| blockLocalFileAccess | **true** |
| stopSlowScripts | true |
| loadErrorHandling | abort |
| mediaLoadErrorHandling | ignore |
| printMediaType | false |
| zoomFactor | 1.0 |

## Security notes

- Local files blocked by default; `--allow` allowlist  
- Upstream **ignores all SSL errors** — pure-Go rewrite should not copy blindly  
- Cookie extras lack strong domain/path scoping  

## WebKit surface to replace

HTML5 parse, CSS, layout, fonts, images, SVG, JS/DOM, print media, pagination.

## Effort bands (loader only vs engine)

| Scope | Person-weeks |
|-------|--------------|
| Loader orchestration parity | 3–8 |
| Report HTML/CSS subset engine | 40–100 |
| Full WebKit parity stdlib | 800–2000+ (not realistic) |
