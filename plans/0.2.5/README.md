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

**Complete (implementation validated 2026-08-19).** Checklist rows in
`font/` are closed with `make test` / `make lint` and fixture-55 evidence.
`VERSION` remains `0.2.4` until release prep tags 0.2.5.

## Successor (do not reopen font phases 01–08)

WOFF2 decode and opt-in metric aliases are tracked under
[`plans/0.2.6/woff2-metric-aliases/`](../0.2.6/woff2-metric-aliases/README.md).
The no-flag Liberation-via-stack contract from this folder stays the baseline.
