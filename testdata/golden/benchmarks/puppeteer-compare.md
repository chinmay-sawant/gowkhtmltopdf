# Direct CLI comparison: gowkhtmltopdf vs Puppeteer

Process-level measurement. Each cell is the median of 3 timed runs after one warmup.
Wall time is measured around `/usr/bin/time`; RSS is peak resident set from `%M` (KiB).
gowkhtmltopdf used `--quiet --enable-local-file-access`; Puppeteer printed via headless Chrome with `format: A4`, `printBackground: true`, `preferCSSPageSize: true`.
Puppeteer RSS is the peak process-tree RSS (node driver + headless Chrome children) sampled from a `ps` snapshot every 0.02 s; gowkhtmltopdf RSS is `%M`.

- gowkhtmltopdf: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/bin/gowkhtmltopdf` (generic CLI)
- Puppeteer: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/scripts/puppeteer/print.sh` (puppeteer-core 24.43.1 + Google Chrome 143.0.7499.40)
- Reproduce: `./scripts/bench-external.sh --engines=puppeteer` (or `make bench`)

| Pages | Gowk time | Puppeteer time | Speedup | Gowk RSS | Puppeteer RSS | Gowk PDF bytes | Puppeteer PDF bytes |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 22 ms | 1.479 s | 67.27x | 24,192 KiB | 938,636 KiB | 34,068 | 102,932 |
| 10 | 45 ms | 1.698 s | 38.08x | 26,688 KiB | 1,013,204 KiB | 56,607 | 406,557 |
| 50 | 119 ms | 1.892 s | 15.86x | 42,816 KiB | 1,106,184 KiB | 164,461 | 1,934,728 |
| 100 | 272 ms | 2.623 s | 9.62x | 63,168 KiB | 1,243,636 KiB | 300,249 | 3,884,017 |
