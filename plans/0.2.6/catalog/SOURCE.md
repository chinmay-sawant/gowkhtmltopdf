# CSS catalog sources (frozen 2026-08-27)

This folder is the inventory for `plans/0.2.6` CSS coverage. It is not loaded by the engine at runtime.

## Why these files

MDN is deprecating `mdn/data` in favor of W3C webref. Webref is extracted from CSS specs every few hours and published weekly as `@webref/css`. That is the universe of names.

W3C also publishes a process-status index: every property, every spec that defines it, REC / CR / WD / ED. CSS Snapshot 2026 is policy, not a property list. The old CSS Print Profile is abandoned. Do not use it.

## Files

| File | Role | Upstream |
|------|------|----------|
| `webref-css.json` | Primary catalog: properties, at-rules, selectors, functions, types | `https://unpkg.com/@webref/css/css.json` (captured 2026-08-27; package 8.7.3 on npm that day) |
| `w3c-all-properties.json` | Maturity overlay: property + spec URL + REC/CR/WD/ED | `https://www.w3.org/Style/CSS/all-properties.en.json` |
| `mdn-properties.json` | Overlay: `mdn_url`, `groups`, `status` (standard / experimental / obsolete) | `https://raw.githubusercontent.com/mdn/data/master/css/properties.json` |
| `mdn-units.json` | Units. Webref has none. | `https://raw.githubusercontent.com/mdn/data/master/css/units.json` |
| `mapping.json` | Ours: each name mapped to engine status against `internal/layout` + `internal/css` | generated 2026-08-27 |
| `coverage-summary.json` | Counts from that mapping | generated 2026-08-27 |

Human index: `README.md`.

## SHA-256 (this freeze)

```
b26a0501c6ee972ca343d2f91be620aaef0c719ec5602a2a70f317fd22135d75  webref-css.json
b5afe6f4c6e3e670bf5e27564e216fe2092579e4872fe49eaade5f970c96825d  w3c-all-properties.json
c03b7ea3c22cb3aa6b7a154b22ff4fb8d34ada743e754fe749399cba6bc31c74  mdn-properties.json
4034c340172ce697ef6d8f7246d788486146e8eb35735cbd9c57e6175bb8f81d  mdn-units.json
d923b259384c7983b3bdb68c783c82ca6913f31f588df99714569cae1f1f6b6e  mapping.json
d21051aef1cd1b5d2ee92bc39a94c52625e4067717632f068b37b3619ce7e3f2  coverage-summary.json
```

## License

- webref: MIT, Copyright 2020 World Wide Web Consortium
- mdn/data: CC0-1.0
- W3C all-properties: W3C document license
- `mapping.json`: this repo

## Refresh rule

Do not follow `main` on a whim. Pin a date in this file. Re-run the generator in Phase 48 only when bumping the freeze. New names land as `unsupported` or `ignored`. Never mark `implemented` from a catalog bump.

Human pages: [W3C all CSS properties](https://www.w3.org/Style/CSS/all-properties.en.html), [CSS Snapshot](https://www.w3.org/TR/CSS/), [webref](https://github.com/w3c/webref), [MDN CSS reference](https://developer.mozilla.org/en-US/docs/Web/CSS/Reference).
