# Direct CLI comparison: gowkhtmltopdf vs wkhtmltopdf

Snapshot F, **2026-08-14**. Host: Linux amd64, 13th Gen Intel Core i7-13700HX
(WSL2, 24 CPUs). Freshly built generic `gowkhtmltopdf` 0.2.1.

Process-level measurement. Each cell is the median of three timed runs after one warmup.
Wall time is Go `time.Since` around `/usr/bin/time`; RSS is peak resident set from `%M` (KiB).
Both binaries used `--quiet --enable-local-file-access` on the same generated report fixture.

- gowkhtmltopdf: `bin/gowkhtmltopdf` (generic CLI)
- wkhtmltopdf: `/usr/local/bin/wkhtmltopdf` (wkhtmltopdf 0.12.6.1 (with patched qt))
- Reproduce: `make bench-cli-compare`

| Pages | Gowk time | wkhtmltopdf time | Speedup | Gowk RSS | wkhtmltopdf RSS | Gowk PDF bytes | wkhtmltopdf PDF bytes |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 16 ms | 254 ms | 15.95x | 24,192 KiB | 44,852 KiB | 42,501 | 18,486 |
| 5 | 22 ms | 265 ms | 12.23x | 24,768 KiB | 45,396 KiB | 50,919 | 30,584 |
| 10 | 30 ms | 278 ms | 9.41x | 26,496 KiB | 46,200 KiB | 65,072 | 50,994 |
| 20 | 44 ms | 304 ms | 6.84x | 29,760 KiB | 47,156 KiB | 91,899 | 90,742 |
| 50 | 88 ms | 387 ms | 4.40x | 41,472 KiB | 51,824 KiB | 172,926 | 210,678 |
| 100 | 184 ms | 530 ms | 2.89x | 58,752 KiB | 58,976 KiB | 308,714 | 411,260 |
| 200 | 353 ms | 812 ms | 2.30x | 90,048 KiB | 74,336 KiB | 579,862 | 816,285 |
| 250 | 433 ms | 942 ms | 2.18x | 112,704 KiB | 81,636 KiB | 715,476 | 1,019,315 |
| 500 | 1.045 s | 1.641 s | 1.57x | 199,872 KiB | 123,264 KiB | 1,398,479 | 2,036,776 |
