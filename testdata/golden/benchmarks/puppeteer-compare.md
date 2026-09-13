# Direct CLI comparison: gowkhtmltopdf vs Puppeteer

Process-level measurement. Each cell is the median of 3 timed runs after one warmup.
Wall time is measured around `/usr/bin/time`; RSS is peak resident set from `%M` (KiB).
Fixture: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/testdata/golden/benchmarks/templates/report.html.tmpl` (20 invoice rows per requested page).
Host: Linux 6.6.87.2-microsoft-standard-WSL2 x86_64 (24 CPUs); toolchain: go version go1.26.4 linux/amd64; gowkhtmltopdf: 0.2.6.
Ghostscript `gs` was present; rendered page counts were checked against the requested size.
gowkhtmltopdf used `--quiet --allow-local-files -o OUTPUT INPUT`; Puppeteer printed via headless Chrome (`/usr/bin/google-chrome`) with `format: A4`, `printBackground: true`, `preferCSSPageSize: true`.
Gowk source: baseline rows from testdata/golden/benchmarks/cli-compare-results.csv (the same session's make bench-cli-compare run); engine rows are measured in this session.
Puppeteer RSS is the peak process-tree RSS (node driver + headless Chrome children) sampled from a `ps` snapshot every 0.02 s; gowkhtmltopdf RSS is `%M`.

- gowkhtmltopdf: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/bin/gowkhtmltopdf` (generic CLI)
- Puppeteer: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/scripts/puppeteer/print.sh` (puppeteer-core 24.43.1 + Google Chrome 143.0.7499.40)
- Reproduce: `./scripts/bench-external.sh --engines=puppeteer` (or `make bench`)

| Pages | Gowk time | Puppeteer time | Speedup | Gowk RSS | Puppeteer RSS | Gowk PDF bytes | Puppeteer PDF bytes |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 13 ms | 1.452 s | 111.73x | 19,584 KiB | 942,964 KiB | 34,210 | 134,319 |
| 10 | 24 ms | 1.479 s | 61.63x | 24,384 KiB | 1,022,844 KiB | 57,239 | 450,799 |
| 50 | 67 ms | 1.785 s | 26.65x | 29,376 KiB | 1,114,080 KiB | 167,525 | 1,981,892 |
| 100 | 124 ms | 2.158 s | 17.40x | 35,520 KiB | 1,241,028 KiB | 306,321 | 3,936,067 |
