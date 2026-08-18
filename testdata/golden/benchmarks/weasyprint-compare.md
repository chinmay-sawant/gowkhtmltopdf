# Direct CLI comparison: gowkhtmltopdf vs WeasyPrint

Process-level measurement. Each cell is the median of 3 timed runs after one warmup.
Wall time is measured around `/usr/bin/time`; RSS is peak resident set from `%M` (KiB).
gowkhtmltopdf used `--quiet --enable-local-file-access`; weasyprint used `-q` (quiet).
weasyprint RSS is the peak of the weasyprint CLI process from `%M`; gowkhtmltopdf RSS is `%M`.

- gowkhtmltopdf: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/bin/gowkhtmltopdf` (generic CLI)
- WeasyPrint: `/home/chinmay/ChinmayPersonalProjects/gowkhtmltopdf/scripts/weasyprint/print.sh` (WeasyPrint version 69.0)
- Reproduce: `./scripts/bench-external.sh --engines=weasyprint` (or `make bench-external`)

| Pages | Gowk time | WeasyPrint time | Speedup | Gowk RSS | WeasyPrint RSS | Gowk PDF bytes | WeasyPrint PDF bytes |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 24 ms | 796 ms | 33.37x | 23,808 KiB | 77,620 KiB | 34,068 | 15,585 |
| 10 | 42 ms | 1.743 s | 41.40x | 26,496 KiB | 105,500 KiB | 56,607 | 45,174 |
| 50 | 121 ms | 7.151 s | 59.26x | 42,240 KiB | 247,176 KiB | 164,461 | 190,548 |
| 100 | 225 ms | 13.464 s | 59.82x | 61,248 KiB | 422,584 KiB | 300,249 | 372,868 |
