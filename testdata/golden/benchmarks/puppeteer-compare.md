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
| 2 | 21 ms | 1.470 s | 68.85x | 24,960 KiB | 973,260 KiB | 34,209 | 134,319 |
| 10 | 37 ms | 1.550 s | 42.45x | 28,800 KiB | 1,021,160 KiB | 57,231 | 450,799 |
| 50 | 120 ms | 1.815 s | 15.08x | 48,576 KiB | 1,114,380 KiB | 167,442 | 1,981,892 |
| 100 | 249 ms | 2.179 s | 8.74x | 68,928 KiB | 1,241,476 KiB | 306,144 | 3,936,067 |
