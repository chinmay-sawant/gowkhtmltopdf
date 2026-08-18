# Direct CLI comparison: gowkhtmltopdf vs WeasyPrint

Process-level measurement. Each cell is the median of 3 timed runs after one warmup.
Wall time is measured around `/usr/bin/time`; RSS is peak resident set from `%M` (KiB).
Fixture: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/testdata/golden/benchmarks/templates/report.html.tmpl` (20 invoice rows per requested page).
Host: Linux 6.6.87.2-microsoft-standard-WSL2 x86_64 (24 CPUs); toolchain: go version go1.26.4 linux/amd64; gowkhtmltopdf: 0.2.4.
Ghostscript `gs` was present; rendered page counts were checked against the requested size.
gowkhtmltopdf used `--quiet --allow-local-files -o OUTPUT INPUT`; weasyprint used `-q` (quiet).
weasyprint RSS is the peak of the weasyprint CLI process from `%M`; gowkhtmltopdf RSS is `%M`.

- gowkhtmltopdf: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/bin/gowkhtmltopdf` (generic CLI)
- WeasyPrint: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/scripts/weasyprint/print.sh` (WeasyPrint version 69.0)
- Reproduce: `./scripts/bench-external.sh --engines=weasyprint` (or `make bench`)

| Pages | Gowk time | WeasyPrint time | Speedup | Gowk RSS | WeasyPrint RSS | Gowk PDF bytes | WeasyPrint PDF bytes |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 19 ms | 616 ms | 32.15x | 24,576 KiB | 77,420 KiB | 34,068 | 15,587 |
| 10 | 31 ms | 1.352 s | 43.65x | 26,880 KiB | 106,000 KiB | 56,607 | 45,174 |
| 50 | 100 ms | 5.217 s | 52.01x | 42,624 KiB | 246,876 KiB | 164,461 | 190,546 |
| 100 | 186 ms | 10.528 s | 56.62x | 58,560 KiB | 423,004 KiB | 300,249 | 372,868 |
