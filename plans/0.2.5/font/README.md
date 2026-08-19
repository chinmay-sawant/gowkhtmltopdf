# v0.2.5 Font Resolution and Compliance

> **Status:** planned — implementation not started
> **Parent:** [`plans/0.2.5/README.md`](../README.md)
> **Predecessor:** completed v0.2.4 Document API / CLI work
> ([`plans/0.2.4/README.md`](../../0.2.4/README.md), phases 31–39 closed)
> **Evidence sources:** `documentation/fonts.md`,
> `knowledge-base/wiki/syntheses/fonts-and-typography.md`,
> codebase scan refreshed 2026-08-19 (see canonical §Current-state findings)

This track makes CSS font resolution predictable and useful for real-world
documents while preserving the existing pure-Go PDF writer and its compliance
contracts. It is driven by the fixture-55 comparison:

| Renderer | `Georgia` in the compared artifact |
|----------|------------------------------------|
| Chrome on the supplied Windows host | Actual subsetted `Georgia` |
| WeasyPrint on this Linux host | Fontconfig-selected `Gelasio Italic` |
| Gowkhtmltopdf without font flags | Bundled `Liberation Serif` |

The goal is not to make host font discovery the default. The goal is to make
the resolution contract explicit:

1. An explicitly supplied, supported face wins when its internal family name
   matches the CSS family.
2. CSS family stacks continue in author order when a named face is absent.
3. Generic families use deterministic bundled faces.
4. Host aliases such as Fontconfig's `Georgia → Gelasio` are not imported into
   the renderer's default behavior.
5. Every selected face remains parseable, subsettable, embeddable, and valid
   under the existing PDF/A and PDF/UA profiles.

## Start here

- [00-canonical-font-resolution-plan.md](00-canonical-font-resolution-plan.md)
- [phase-01-resolution-contract.md](phase-01-resolution-contract.md)
- [phase-02-resolver-seam.md](phase-02-resolver-seam.md)
- [phase-03-discovery-and-font-face.md](phase-03-discovery-and-font-face.md)
- [phase-04-font-format-and-asset-policy.md](phase-04-font-format-and-asset-policy.md)
- [phase-05-pdf-embedding-and-compliance.md](phase-05-pdf-embedding-and-compliance.md)
- [phase-06-corpus-and-engine-comparison.md](phase-06-corpus-and-engine-comparison.md)
- [phase-07-docs-and-operator-contract.md](phase-07-docs-and-operator-contract.md)
- [phase-08-validation-and-closure.md](phase-08-validation-and-closure.md)

## Related work (out of this track)

| Item | Location | Relationship |
|------|----------|--------------|
| HTTPS/local TTF/OTF/WOFF1 `@font-face` | already shipped | Reuse; do not re-plan as unimplemented |
| WOFF2 / Brotli decode | [`plans/0.2.0/phases/pending-phase-items/09-remote-webfonts.md`](../../0.2.0/phases/pending-phase-items/09-remote-webfonts.md) | Separate allowlist-amendment epic; not required for resolution closure |
| Normative operator docs | [`documentation/fonts.md`](../../../documentation/fonts.md) | Authoritative until Phase 7 lands |
| Compiled KB synthesis | [`knowledge-base/wiki/syntheses/fonts-and-typography.md`](../../../knowledge-base/wiki/syntheses/fonts-and-typography.md) | Path retargeted to this folder; Phase 7 still syncs contract text after implementation |

All implementation checklist rows in this folder intentionally remain
unchecked until the corresponding code, rendered artifacts, and compliance
proof exist. Creating or relocating this plan does not constitute
implementation evidence.
