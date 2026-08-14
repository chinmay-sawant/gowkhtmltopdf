# gocorepdfengine — Phase Plans

Source: [baseplan/base-pdf-engine-pdfa4-pdfua2-plan.md](./baseplan/base-pdf-engine-pdfa4-pdfua2-plan.md)

Each phase is a **checklist plan** you can execute independently. Complete phases in order unless noted.

| Phase | File | Goal | Gate |
|-------|------|------|------|
| 1 | [phase-01-core-pdf20-writer.md](./phase-01-core-pdf20-writer.md) | Minimal PDF 2.0 shell | Unit / open in viewer |
| 2 | [phase-02-layout-primitives.md](./phase-02-layout-primitives.md) | Text, tables, multi-page, images | Visual fixtures |
| 3 | [phase-03-font-embedding.md](./phase-03-font-embedding.md) | TTF subset + Type0 embed | Glyph/width tests |
| 4 | [phase-04-pdfa4-compliance.md](./phase-04-pdfa4-compliance.md) | PDF/A-4 archival objects | **veraPDF `-f 4`** |
| 5 | [phase-05-pdfua2-tagging.md](./phase-05-pdfua2-tagging.md) | PDF/UA-2 structure tree | **veraPDF `-f ua2`** |
| 6 | [phase-06-performance-pooling.md](./phase-06-performance-pooling.md) | Speed / memory parity | Bench (after 4+5 green) |
| 7 | [phase-07-optional-product-features.md](./phase-07-optional-product-features.md) | Sign, encrypt, forms | Separate product gates |
| 8 | [phase-08-zerodha-benchmark.md](./phase-08-zerodha-benchmark.md) | Zerodha-style JSON→model→layout bench (cache on/off) | Local engine only |

**Template field contract:** [guides/TEMPLATE_REFERENCE.md](../guides/TEMPLATE_REFERENCE.md) (full `config`/`elements` shape; phase 8 maps domain JSON → layout).

**Default compliant profile (end of phase 5):** PDF 2.0 + PDF/A-4 + PDF/UA-2.

**Out of scope:** HTTP API, frontend, bindings, **gopdfsuit as a dependency**, merge/redact product surface.

## Architecture reviews

| Date | File | Overall |
|------|------|--------:|
| 2026-07-25 | [reviews/improve-codebase-architecture/2026-07-25-architecture-review.md](./reviews/improve-codebase-architecture/2026-07-25-architecture-review.md) | **6.2 / 10** |

## Ponytail reviews (leanness / over-engineering)

| Date | File | Overall |
|------|------|--------:|
| 2026-07-25 | [pontail/ponytail-ultra-2026-07-25.md](./pontail/ponytail-ultra-2026-07-25.md) | **6.3 / 10** |

## Compliance + Zerodha harness

```bash
make install-verapdf
make test-verify-pdfs
make bench-zerodha              # cache ON
make bench-zerodha-uncached     # rebuild model each iter
make bench-zerodha-nocomply
```

See [`../compliance/README.md`](../compliance/README.md) and [`../sampledata/zerodha/README.md`](../sampledata/zerodha/README.md).
