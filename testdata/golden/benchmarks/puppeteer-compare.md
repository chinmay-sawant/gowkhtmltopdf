# Direct CLI comparison: gowkhtmltopdf vs Puppeteer

Process-level measurement. Each cell is the median of 3 timed runs after one warmup.
Wall time is measured around `/usr/bin/time`; RSS is peak resident set from `%M` (KiB).
gowkhtmltopdf used `--quiet --enable-local-file-access`; Puppeteer printed via headless Chrome with `format: A4`, `printBackground: true`, `preferCSSPageSize: true`.
Puppeteer RSS is the peak process-tree RSS (node driver + headless Chrome children) sampled from a `ps` snapshot every 0.02 s; gowkhtmltopdf RSS is `%M`.

- gowkhtmltopdf: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/bin/gowkhtmltopdf` (generic CLI)
- Puppeteer: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/scripts/puppeteer/print.sh` (puppeteer-core 24.43.1 + Google Chrome 143.0.7499.40)
- Reproduce: `./scripts/bench-external.sh --engines=puppeteer` (or `make bench-external`)

| Pages | Gowk time | Puppeteer time | Speedup | Gowk RSS | Puppeteer RSS | Gowk PDF bytes | Puppeteer PDF bytes |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 28 ms | 1.563 s | 55.42x | 23,424 KiB | 990,672 KiB | 34,068 | 102,932 |
| 10 | 45 ms | 1.569 s | 34.51x | 26,496 KiB | 1,013,908 KiB | 56,607 | 406,557 |
| 50 | 125 ms | 1.952 s | 15.59x | 41,664 KiB | 1,109,484 KiB | 164,461 | 1,934,728 |
| 100 | 220 ms | 2.358 s | 10.73x | 62,208 KiB | 1,227,144 KiB | 300,249 | 3,884,017 |
