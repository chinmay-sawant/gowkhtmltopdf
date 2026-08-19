# Direct CLI comparison: gowkhtmltopdf vs Puppeteer

Process-level measurement. Each cell is the median of 3 timed runs after one warmup.
Wall time is measured around `/usr/bin/time`; RSS is peak resident set from `%M` (KiB).
Fixture: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/testdata/golden/benchmarks/templates/report.html.tmpl` (20 invoice rows per requested page).
Host: Linux 6.6.87.2-microsoft-standard-WSL2 x86_64 (24 CPUs); toolchain: go version go1.26.4 linux/amd64; gowkhtmltopdf: 0.2.4.
Ghostscript `gs` was present; rendered page counts were checked against the requested size.
gowkhtmltopdf used `--quiet --allow-local-files -o OUTPUT INPUT`; Puppeteer printed via headless Chrome (`/usr/bin/google-chrome`) with `format: A4`, `printBackground: true`, `preferCSSPageSize: true`.
Puppeteer RSS is the peak process-tree RSS (node driver + headless Chrome children) sampled from a `ps` snapshot every 0.02 s; gowkhtmltopdf RSS is `%M`.

- gowkhtmltopdf: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/bin/gowkhtmltopdf` (generic CLI)
- Puppeteer: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/scripts/puppeteer/print.sh` (puppeteer-core 24.43.1 + Google Chrome 143.0.7499.40)
- Reproduce: `./scripts/bench-external.sh --engines=puppeteer` (or `make bench`)

| Pages | Gowk time | Puppeteer time | Speedup | Gowk RSS | Puppeteer RSS | Gowk PDF bytes | Puppeteer PDF bytes |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 18 ms | 1.411 s | 77.30x | 23,808 KiB | 944,056 KiB | 34,068 | 102,932 |
| 10 | 32 ms | 1.548 s | 47.84x | 27,264 KiB | 1,019,896 KiB | 56,607 | 406,557 |
| 50 | 121 ms | 2.069 s | 17.06x | 43,008 KiB | 1,108,580 KiB | 164,461 | 1,934,728 |
| 100 | 199 ms | 2.145 s | 10.78x | 62,016 KiB | 1,245,988 KiB | 300,249 | 3,884,017 |
