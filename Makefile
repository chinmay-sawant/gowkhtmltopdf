.PHONY: test lint build fmt golden golden-update samples clean

# Phase 00 scaffold: stdlib-only module. No external deps expected.

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
# showcase PDF (TOC + headers/footers + outline) and a PNG from the image
# converter. Sample files under output/ are committed as viewer smoke artifacts.
samples:
	# Wipe regenerable fixture samples only (keep optional URL smokes like wiki-*.pdf)
	rm -f output/fixture-*.pdf output/fixture-*.png output/showcase-*.pdf
	for f in testdata/golden/fixture-*.html; do \
		name=$$(basename "$$f" .html); \
		go run ./cmd/gowkhtmltopdf --enable-local-file-access "$$f" "output/$$name.pdf"; \
	done
	go run ./cmd/gowkhtmltopdf --enable-local-file-access --outline --outline-depth 2 --header-left "gowkhtmltopdf demo - [title]" --header-right "page [page]/[topage]" --footer-center "[section]" toc testdata/golden/fixture-16-invoice-with-css.html output/showcase-toc-hf-outline.pdf
	go run ./cmd/gowkhtmltoimage --enable-local-file-access testdata/golden/fixture-01-simple-invoice.html output/fixture-01-simple-invoice.png
	go run ./examples/image --enable-local-file-access --width 1024 testdata/golden/fixture-21-detailed-report.html output/fixture-21-detailed-report.png
	ls -la output/ | awk '{print $$5, $$9}' | tail -30

clean:
	rm -rf testdata/golden/out
