.PHONY: test lint build fmt golden golden-update samples clean

# Phase 00 scaffold: stdlib + allowlisted go-text/typesetting only.
# Direct third-party requires must stay ⊆ {github.com/go-text/typesetting}
# (enforced by internal/pdf.TestDirectModuleAllowlist).

test:
	go test ./...

lint:
	go vet ./...
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

build:
	mkdir -p bin
	go build -o bin/gowkhtmltopdf ./cmd/gowkhtmltopdf
	go build -o bin/gowkhtmltoimage ./cmd/gowkhtmltoimage

fmt:
	gofmt -w .

# Phase 9.1: golden corpus. TestGoldenCorpusAllFixtures walks every
# testdata/golden/*.html fixture, converts it through the full pipeline
# (load -> parse -> style -> layout -> paint -> write) and asserts the PDF
# structure (%PDF-, %%EOF, xref offset), the embedded font, the per-fixture
# page envelope and the feature expectations (images, URI annotations).
# TestGoldenCorpus covers the first three fixtures plus the fixture-03
# layout+paint performance budget.
golden:
	go test ./internal/convert/ -run 'TestGoldenCorpus' -v

golden-update:
	@echo "golden-update: implemented in Phase 3 (PDF writer)"; true

# Regenerate the sample outputs in output/: one PDF per golden fixture, a
# showcase PDF (TOC + headers/footers + outline), image PNGs, and the optional
# live Wikipedia smoke (needs network; failure does not fail the target).
samples:
	# Wipe regenerable fixture samples only (wiki-*.pdf is rewritten below).
	rm -f output/fixture-*.pdf output/fixture-*.png output/showcase-*.pdf
	# Opt-in CJK/system faces when present (fixture-27 and font-family lists).
	FONT_FLAGS=""; \
	if [ -d /usr/share/fonts/truetype/droid ]; then FONT_FLAGS="$$FONT_FLAGS --font-path /usr/share/fonts/truetype/droid"; fi; \
	if [ -d testdata/fonts ]; then FONT_FLAGS="$$FONT_FLAGS --font-path testdata/fonts"; fi; \
	for f in testdata/golden/fixture-*.html; do \
		case "$$f" in *-header.html|*-footer.html) continue;; esac; \
		name=$$(basename "$$f" .html); \
		id=$$(printf '%s\n' "$$name" | sed -n 's/^\(fixture-[0-9][0-9]*\).*/\1/p'); \
		HF_FLAGS=""; \
		if [ -n "$$id" ] && [ -f "testdata/golden/$$id-header.html" ]; then \
			HF_FLAGS="$$HF_FLAGS --header-html testdata/golden/$$id-header.html --margin-top -1"; \
		fi; \
		if [ -n "$$id" ] && [ -f "testdata/golden/$$id-footer.html" ]; then \
			HF_FLAGS="$$HF_FLAGS --footer-html testdata/golden/$$id-footer.html --margin-bottom -1"; \
		fi; \
		go run ./cmd/gowkhtmltopdf --enable-local-file-access $$FONT_FLAGS $$HF_FLAGS "$$f" "output/$$name.pdf"; \
	done
	go run ./cmd/gowkhtmltopdf --enable-local-file-access --outline --outline-depth 2 --header-left "gowkhtmltopdf demo - [title]" --header-right "page [page]/[topage]" --footer-center "[section]" toc testdata/golden/fixture-16-invoice-with-css.html output/showcase-toc-hf-outline.pdf
	go run ./cmd/gowkhtmltoimage --enable-local-file-access testdata/golden/fixture-01-simple-invoice.html output/fixture-01-simple-invoice.png
	go run ./examples/image --enable-local-file-access --width 1024 testdata/golden/fixture-21-detailed-report.html output/fixture-21-detailed-report.png
	# Live Wikipedia smoke (network, raw — no --simplify-dom). Soft-fail so offline/CI hosts still get fixture samples.
	# --use-system-fonts enables Unicode/IPA glyph fallback (DejaVu/Noto when present).
	go run ./cmd/gowkhtmltopdf --use-system-fonts \
		'https://en.wikipedia.org/wiki/Ana_de_Armas' \
		output/wiki-ana-de-armas.pdf \
		|| echo "warning: wiki-ana-de-armas.pdf live smoke skipped (network/fetch failed)"
	ls -la output/ | awk '{print $$5, $$9}' | tail -30

clean:
	rm -rf testdata/golden/out
