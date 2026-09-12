# Direct CLI comparison: gowkhtmltopdf vs wkhtmltopdf

Process-level measurement. Each cell is the median of three timed runs after one warmup.
Wall time is Go `time.Since` around `/usr/bin/time`; RSS is peak resident set from `%M` (KiB).
gowkhtmltopdf used `--quiet --allow-local-files -o OUTPUT INPUT`; wkhtmltopdf used its native local-file flags. on the same generated report fixture.

- gowkhtmltopdf: `../../bin/gowkhtmltopdf` (generic CLI)
- wkhtmltopdf: `/usr/local/bin/wkhtmltopdf` (wkhtmltopdf 0.12.6.1 (with patched qt))
- Reproduce: `make bench-cli-compare`

| Pages | Gowk time | wkhtmltopdf time | Speedup | Gowk RSS | wkhtmltopdf RSS | Gowk PDF bytes | wkhtmltopdf PDF bytes |
|---:|---:|---:|---:|---:|---:|---:|---:|
| 2 | 14 ms | 260 ms | 18.50x | 19,200 KiB | 44,720 KiB | 34,210 | 18,486 |
| 5 | 19 ms | 269 ms | 14.37x | 21,696 KiB | 45,168 KiB | 42,795 | 30,584 |
| 10 | 26 ms | 286 ms | 11.18x | 25,152 KiB | 46,016 KiB | 57,239 | 50,994 |
| 20 | 36 ms | 315 ms | 8.74x | 25,920 KiB | 47,620 KiB | 84,680 | 90,742 |
| 50 | 69 ms | 403 ms | 5.85x | 29,760 KiB | 52,128 KiB | 167,525 | 210,678 |
| 100 | 126 ms | 546 ms | 4.35x | 35,904 KiB | 59,460 KiB | 306,321 | 411,260 |
| 200 | 234 ms | 852 ms | 3.64x | 44,928 KiB | 74,308 KiB | 583,670 | 816,285 |
| 250 | 288 ms | 1.008 s | 3.50x | 51,264 KiB | 81,820 KiB | 722,322 | 1,019,315 |
| 500 | 562 ms | 1.760 s | 3.13x | 79,296 KiB | 123,076 KiB | 1,420,537 | 2,036,776 |
