# Plans — v0.2.5 (Font resolution follow-up)

| File / Folder | Role |
|---------------|------|
| [font/](font/README.md) | **Canonical font resolution, fallback, and compliance track** (phases 01–08) |
| [font/00-canonical-font-resolution-plan.md](font/00-canonical-font-resolution-plan.md) | Contract, current-state findings, phase map, definition of done |

Predecessor: [../0.2.4/README.md](../0.2.4/README.md) (Document API + CLI + external benches; phases 31–39 complete).

Product framing: [../../documentation/fonts.md](../../documentation/fonts.md),
[../../documentation/compatibility-matrix.md](../../documentation/compatibility-matrix.md),
[../../knowledge-base/wiki/syntheses/fonts-and-typography.md](../../knowledge-base/wiki/syntheses/fonts-and-typography.md).

## Scope in one line

Make CSS font resolution honest and useful: exact supplied faces win when
their internal family name matches; author stacks and deterministic bundled
generics continue otherwise; host Fontconfig aliases never enter default
output; every selected face still satisfies the existing PDF writer and
claiming-profile gates.

## Status

**Planned.** `VERSION` remains `0.2.4` until this track (or a sibling 0.2.5
track) ships and release prep updates it. No checklist row in `font/` is
complete until code, fixtures, compliance proof, and docs agree.
