# Direct CLI comparison: gowkhtmltopdf vs wkhtmltopdf

Process-level measurement. Each cell is the median of three timed runs after one warmup.
Wall time is Go `time.Since` around `/usr/bin/time`; RSS is peak resident set from `%M` (KiB).
gowkhtmltopdf used `--quiet --allow-local-files -o OUTPUT INPUT`; wkhtmltopdf used its native local-file flags. on the same generated report fixture.

- gowkhtmltopdf: `../../bin/gowkhtmltopdf` (generic CLI)
- wkhtmltopdf: `/usr/local/bin/wkhtmltopdf` (wkhtmltopdf 0.12.6.1 (with patched qt))
- Reproduce: `make bench-cli-compare`

| Pages | Gowk time | wkhtmltopdf time | Speedup | Gowk RSS | wkhtmltopdf RSS | Gowk PDF bytes | wkhtmltopdf PDF bytes |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 13 ms | 258 ms | 19.68x | 19,584 KiB | 44,528 KiB | 34,210 | 18,486 |
| 5 | 18 ms | 269 ms | 14.68x | 22,080 KiB | 44,784 KiB | 42,795 | 30,584 |
| 10 | 24 ms | 279 ms | 11.65x | 24,384 KiB | 45,888 KiB | 57,239 | 50,994 |
| 20 | 35 ms | 310 ms | 8.75x | 26,304 KiB | 47,300 KiB | 84,680 | 90,742 |
| 50 | 67 ms | 393 ms | 5.84x | 29,376 KiB | 51,652 KiB | 167,525 | 210,678 |
| 100 | 124 ms | 532 ms | 4.30x | 35,520 KiB | 59,172 KiB | 306,321 | 411,260 |
| 200 | 240 ms | 814 ms | 3.39x | 45,888 KiB | 74,356 KiB | 583,670 | 816,285 |
| 250 | 279 ms | 973 ms | 3.49x | 52,032 KiB | 81,632 KiB | 722,322 | 1,019,315 |
| 500 | 573 ms | 1.718 s | 3.00x | 80,448 KiB | 123,068 KiB | 1,420,537 | 2,036,776 |
