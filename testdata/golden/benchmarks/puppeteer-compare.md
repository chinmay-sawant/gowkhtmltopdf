# Direct CLI comparison: gowkhtmltopdf vs Puppeteer

Process-level measurement. Each cell is the median of 3 timed runs after one warmup.
Wall time is measured around `/usr/bin/time`; RSS is peak resident set from `%M` (KiB).
Fixture: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/testdata/golden/benchmarks/templates/report.html.tmpl` (20 invoice rows per requested page).
Host: Linux 6.6.87.2-microsoft-standard-WSL2 x86_64 (24 CPUs); toolchain: go version go1.26.4 linux/amd64; gowkhtmltopdf: 0.2.5.
Ghostscript `gs` was present; rendered page counts were checked against the requested size.
gowkhtmltopdf used `--quiet --allow-local-files -o OUTPUT INPUT`; Puppeteer printed via headless Chrome (`/usr/bin/google-chrome`) with `format: A4`, `printBackground: true`, `preferCSSPageSize: true`.
Puppeteer RSS is the peak process-tree RSS (node driver + headless Chrome children) sampled from a `ps` snapshot every 0.02 s; gowkhtmltopdf RSS is `%M`.

- gowkhtmltopdf: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/bin/gowkhtmltopdf` (generic CLI)
- Puppeteer: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/scripts/puppeteer/print.sh` (puppeteer-core 24.43.1 + Google Chrome 143.0.7499.40)
- Reproduce: `./scripts/bench-external.sh --engines=puppeteer` (or `make bench`)

| Pages | Gowk time | Puppeteer time | Speedup | Gowk RSS | Puppeteer RSS | Gowk PDF bytes | Puppeteer PDF bytes |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 16 ms | 1.445 s | 92.85x | 19,200 KiB | 940,600 KiB | 34,210 | 134,319 |
| 10 | 26 ms | 1.488 s | 57.16x | 24,192 KiB | 1,019,952 KiB | 57,239 | 450,799 |
| 50 | 72 ms | 1.801 s | 25.10x | 29,760 KiB | 1,119,360 KiB | 167,525 | 1,981,892 |
| 100 | 127 ms | 2.178 s | 17.16x | 35,136 KiB | 1,240,920 KiB | 306,321 | 3,936,067 |
