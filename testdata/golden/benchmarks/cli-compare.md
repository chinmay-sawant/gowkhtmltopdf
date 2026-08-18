# Direct CLI comparison: gowkhtmltopdf vs wkhtmltopdf

Process-level measurement. Each cell is the median of three timed runs after one warmup.
Wall time is Go `time.Since` around `/usr/bin/time`; RSS is peak resident set from `%M` (KiB).
gowkhtmltopdf used `--quiet --allow-local-files -o OUTPUT INPUT`; wkhtmltopdf used its native local-file flags. on the same generated report fixture.

- gowkhtmltopdf: `../../bin/gowkhtmltopdf` (generic CLI)
- wkhtmltopdf: `/usr/local/bin/wkhtmltopdf` (wkhtmltopdf 0.12.6.1 (with patched qt))
- Reproduce: `make bench-cli-compare`

| Pages | Gowk time | wkhtmltopdf time | Speedup | Gowk RSS | wkhtmltopdf RSS | Gowk PDF bytes | wkhtmltopdf PDF bytes |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 21 ms | 274 ms | 12.83x | 24,384 KiB | 44,304 KiB | 34,068 | 18,486 |
| 5 | 26 ms | 279 ms | 10.68x | 24,384 KiB | 44,492 KiB | 42,467 | 30,584 |
| 10 | 37 ms | 300 ms | 8.04x | 26,880 KiB | 45,424 KiB | 56,607 | 50,994 |
| 20 | 55 ms | 335 ms | 6.06x | 30,720 KiB | 46,772 KiB | 83,434 | 90,742 |
| 50 | 113 ms | 433 ms | 3.82x | 43,008 KiB | 51,540 KiB | 164,461 | 210,678 |
| 100 | 223 ms | 613 ms | 2.75x | 62,784 KiB | 58,628 KiB | 300,249 | 411,260 |
| 200 | 469 ms | 939 ms | 2.00x | 98,496 KiB | 73,968 KiB | 571,397 | 816,285 |
| 250 | 609 ms | 1.175 s | 1.93x | 113,088 KiB | 81,484 KiB | 707,011 | 1,019,315 |
| 500 | 1.243 s | 2.169 s | 1.75x | 207,552 KiB | 122,640 KiB | 1,390,014 | 2,036,776 |
