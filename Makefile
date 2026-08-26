.PHONY: test lint lint-frontend build fmt golden golden-update samples screenshots weasyprint clean claim-scan bench bench-engine bench-lib bench-inprocess bench-cli-compare c-shared bindings-clean check-versions python-binding-test python-benchmarks python-api

# Pure-Go runtime: the standard library plus the allowlisted direct modules
# below. No cgo, browser, or native converter process is required.
# Direct third-party requires must stay ⊆ {
#   github.com/go-text/typesetting,  # OpenType shaping
#   github.com/tdewolff/canvas,      # SVG-as-image rasterization
# }
# (enforced by internal/pdf.TestDirectModuleAllowlist).

# Pin golangci-lint for local + CI reproducibility. Override: make lint GOLANGCI_LINT_VERSION=vX.Y.Z
# Build with the local toolchain (go1.26.4): golangci-lint refuses to run when the
# binary's Go version is lower than the module's targeted Go version.
GOLANGCI_LINT_VERSION ?= v1.64.8

# WeasyPrint CLI used by the `weasyprint` samples target.
WEASYPRINT ?= weasyprint

test:
	go test ./...

# Runs every linter enabled in .golangci.yml (enable-all), then frontend
# `npm run lint` (ESLint plus src/data content/config checks). Installs the
# pinned golangci-lint binary into $(go env GOPATH)/bin when missing. Always
# builds with GOTOOLCHAIN=local so the binary matches go.mod's go1.26 toolchain.
lint:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not found; installing $(GOLANGCI_LINT_VERSION) with local Go toolchain..."; \
		GOTOOLCHAIN=local go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	}
	golangci-lint version
	golangci-lint run ./...
	$(MAKE) lint-frontend

lint-frontend:
	@command -v npm >/dev/null 2>&1 || { echo "npm is required for frontend lint" >&2; exit 1; }
	@if [ ! -d frontend/node_modules ]; then \
		echo "frontend/node_modules missing; running npm ci..."; \
		npm ci --prefix frontend; \
	fi
	npm --prefix frontend run lint

CLI_VERSION_LDFLAGS := -X github.com/chinmay-sawant/gowkhtmltopdf/internal/cli.Version=$(shell cat VERSION)

# Stamps the c-shared library (bindings/c) with the repo VERSION. Kept separate
# from CLI_VERSION_LDFLAGS so the opt-in cgo build never touches the pure-Go
# default targets. bindings/c is package main, so X must target main.libVersion.
BINDINGS_VERSION_LDFLAGS := -X main.libVersion=$(shell cat VERSION)

build:
	mkdir -p bin
	go build -ldflags "$(CLI_VERSION_LDFLAGS)" -o bin/gowkhtmltopdf ./cmd/gowkhtmltopdf
	go build -ldflags "$(CLI_VERSION_LDFLAGS)" -o bin/gowkhtmltoimage ./cmd/gowkhtmltoimage

# Scan live user-facing surfaces for stale product claims.
claim-scan:
	@if rg -n -S \
		-e 'using only the standard library' \
		-e 'pure-Go, stdlib-only' \
		-e 'zero third-party' \
		-e 'Qt WebKit engine' \
		-e 'identical input bytes produce identical PDF bytes' \
		doc.go README.md documentation/*.md \
		frontend/src/data/content internal/cli/help.go; then \
		echo "claim-scan: forbidden phrase found" >&2; exit 1; \
	fi
	@echo "claim-scan: clean"

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
	@set -eu; \
	if [ "$(GOLDEN_APPROVE)" != "1" ]; then \
		echo "Refusing golden-update: set GOLDEN_APPROVE=1 after reviewing the fixture" >&2; \
		exit 2; \
	fi; \
	fixture="$(GOLDEN_FIXTURE)"; \
	case "$$fixture" in \
		""|*/*|*-header.html|*-footer.html|*.html.html) \
			echo "Usage: make golden-update GOLDEN_FIXTURE=fixture-NN-name.html GOLDEN_APPROVE=1" >&2; \
			exit 2;; \
		*.html) ;; \
		*) \
			echo "GOLDEN_FIXTURE must be a body .html fixture basename" >&2; \
			exit 2;; \
	esac; \
	if [ ! -f "testdata/golden/$$fixture" ]; then \
		echo "Golden fixture not found: testdata/golden/$$fixture" >&2; \
		exit 2; \
	fi; \
	mkdir -p testdata/golden/out; \
	output="testdata/golden/out/$${fixture%.html}.pdf"; \
	echo "Generating $$output from testdata/golden/$$fixture"; \
	go run ./cmd/gowkhtmltopdf --allow-local-files \
		-o "$$output" "testdata/golden/$$fixture"; \
	echo "Review $$output manually; this target never rewrites committed fixtures."

# Regenerate the sample outputs in output/: one PDF per golden fixture, a
# showcase PDF (TOC + headers/footers + outline), the library-API architecture
# diagram PDF (output/architecture-diagram.pdf only; does not rewrite testdata/golden HTML),
# image PNGs, version/compliance smokes under output/pdf-{1.7,2.0}{,-compliance}/
# (fixture-21 and fixture-56), and the optional live Wikipedia smoke (needs
# network; failure does not fail the target).
samples:
	# Wipe regenerable fixture samples only (wiki-*.pdf is rewritten below).
	rm -f output/fixture-*.pdf output/fixture-*.png output/showcase-*.pdf output/architecture-diagram.pdf
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
		go run ./cmd/gowkhtmltopdf --allow-local-files $$FONT_FLAGS $$HF_FLAGS -o "output/$$name.pdf" "$$f"; \
	done
	# Version / compliance smokes (unreleased 0.2.2): same two fixtures in four dirs.
	mkdir -p output/pdf-1.7 output/pdf-1.7-compliance output/pdf-2.0 output/pdf-2.0-compliance
	rm -f output/pdf-1.7/*.pdf output/pdf-1.7-compliance/*.pdf output/pdf-2.0/*.pdf output/pdf-2.0-compliance/*.pdf
	for f in testdata/golden/fixture-21-detailed-report.html testdata/golden/fixture-56-architecture-diagram.html; do \
		name=$$(basename "$$f" .html); \
		go run ./cmd/gowkhtmltopdf --pdf-version 1.7 --allow-local-files -o "output/pdf-1.7/$$name.pdf" "$$f"; \
		go run ./cmd/gowkhtmltopdf --pdf-profile a3a-ua1 --allow-local-files -o "output/pdf-1.7-compliance/$$name.pdf" "$$f"; \
		go run ./cmd/gowkhtmltopdf --pdf-version 2.0 --allow-local-files -o "output/pdf-2.0/$$name.pdf" "$$f"; \
		go run ./cmd/gowkhtmltopdf --pdf-profile a4-ua2 --allow-local-files -o "output/pdf-2.0-compliance/$$name.pdf" "$$f"; \
	done
	go run ./cmd/gowkhtmltopdf --allow-local-files --outline --outline-depth 2 --header-left "gowkhtmltopdf demo - [title]" --header-right "page [page]/[topage]" --footer-center "[section]" --toc -o output/showcase-toc-hf-outline.pdf testdata/golden/fixture-16-invoice-with-css.html
	# Library-API architecture diagram → output/architecture-diagram.pdf only.
	# testdata/golden HTML (corpus fixture and api/ template) is not rewritten.
	go run ./testdata/golden/api
	go run ./cmd/gowkhtmltoimage --allow-local-files -o output/fixture-01-simple-invoice.png testdata/golden/fixture-01-simple-invoice.html
	go run ./examples/image --allow-local-files --width 1024 testdata/golden/fixture-21-detailed-report.html output/fixture-21-detailed-report.png
	# Live Wikipedia smoke (network, raw — no --simplify-dom). Soft-fail so offline/CI hosts still get fixture samples.
	# Operator recipe (not CSS fidelity): --use-system-fonts for IPA fallback; optional --zoom 2/3 densifies
	# author p{font-size:12pt} toward ~8pt; optional --print-link-underline / --simplify-dom-profile=mediawiki.
	go run ./cmd/gowkhtmltopdf --use-system-fonts --zoom 0.666667 \
		'https://en.wikipedia.org/wiki/Ana_de_Armas' \
		-o output/wiki-ana-de-armas.pdf \
		|| echo "warning: wiki-ana-de-armas.pdf live smoke skipped (network/fetch failed)"
	ls -la output/ | awk '{print $$5, $$9}' | tail -30

# Regenerate the committed frontend showcase screenshots and WebP thumbnails
# from the PDFs currently present in output/. Use `make samples` first when the
# source fixture PDFs themselves need to be refreshed.
screenshots:
	python3 scripts/screenshot_showcase.py

# Regenerate the WeasyPrint comparison samples in output/weasyprint/ from the
# same golden fixtures as `make samples` (one PDF per body fixture, excluding
# -header.html/-footer.html companions). WeasyPrint has no
# --header-html/--footer-html; fixture-36's companion header/footer child docs
# are approximated with @page margin boxes via a user stylesheet
# (scripts/weasyprint/fixture-36-hf.css). Version / compliance smokes mirror
# the samples target: fixture-21 and fixture-56 under
# output/weasyprint/pdf-{1.7,2.0}{,-compliance}/ using --pdf-version and
# --pdf-variant. Requires the weasyprint CLI: pip3 install --user weasyprint.
weasyprint:
	@command -v $(WEASYPRINT) >/dev/null 2>&1 || { \
		echo "weasyprint not found; install with: pip3 install --user weasyprint" >&2; \
		exit 1; \
	}
	# Wipe regenerable fixture samples only (like `make samples`).
	mkdir -p output/weasyprint
	rm -f output/weasyprint/fixture-*.pdf
	for f in testdata/golden/fixture-*.html; do \
		case "$$f" in *-header.html|*-footer.html) continue;; esac; \
		name=$$(basename "$$f" .html); \
		STYLE_FLAGS=""; \
		if [ "$$name" = "fixture-36-hf-nested-flex" ]; then \
			STYLE_FLAGS="-s scripts/weasyprint/fixture-36-hf.css"; \
		fi; \
		$(WEASYPRINT) $$STYLE_FLAGS "$$f" "output/weasyprint/$$name.pdf" \
			|| echo "warning: $$name.pdf failed"; \
	done
	# Version / compliance smokes: same two fixtures in four dirs.
	mkdir -p output/weasyprint/pdf-1.7 output/weasyprint/pdf-1.7-compliance \
		output/weasyprint/pdf-2.0 output/weasyprint/pdf-2.0-compliance
	rm -f output/weasyprint/pdf-1.7/*.pdf output/weasyprint/pdf-1.7-compliance/*.pdf \
		output/weasyprint/pdf-2.0/*.pdf output/weasyprint/pdf-2.0-compliance/*.pdf
	for f in testdata/golden/fixture-21-detailed-report.html testdata/golden/fixture-56-architecture-diagram.html; do \
		name=$$(basename "$$f" .html); \
		$(WEASYPRINT) --pdf-version 1.7 "$$f" "output/weasyprint/pdf-1.7/$$name.pdf"; \
		$(WEASYPRINT) --pdf-variant pdf/a-2b "$$f" "output/weasyprint/pdf-1.7-compliance/$$name.pdf"; \
		$(WEASYPRINT) --pdf-version 2.0 "$$f" "output/weasyprint/pdf-2.0/$$name.pdf"; \
		$(WEASYPRINT) --pdf-variant pdf/ua-2 "$$f" "output/weasyprint/pdf-2.0-compliance/$$name.pdf"; \
	done
	ls -la output/weasyprint/ | awk '{print $$5, $$9}' | tail -30

# External process benchmarks against the actual binary. `make bench` builds
# the CLI, times bin/gowkhtmltopdf against WeasyPrint and Puppeteer through
# scripts/bench-external.sh, then runs the dedicated wkhtmltopdf comparison.
# The numbers include process and disk overhead; they are release/operator
# evidence, not a default `make test` gate. Missing external engines are
# skipped, but the target fails when none are available. Writes
# testdata/golden/benchmarks/{weasyprint,puppeteer,cli}-compare.md and
# -results.csv. Default external page matrix is 2/10/50/100; override with
# --sizes=2,5,10,20,50,100,200,250,500 or select one engine with
# --engines=weasyprint. `bench-cli-compare` remains available standalone.
bench: build
	./scripts/bench-external.sh
	$(MAKE) bench-cli-compare

# Internal engine allocation matrix (generic + certified-islands, images).
# Measures the internal conversion pipeline directly; it is independent of
# the public library API and external process targets. See
# testdata/golden/benchmarks/README.md.
bench-engine:
	go test ./internal/convert -run '^$$' \
		-bench 'Benchmark(PDFPages|TemplatePages|WebFetchImage|ImageAssets)$$' \
		-benchmem -benchtime=1x -count=1

# Compatibility alias for the former target name.
bench-inprocess: bench-engine

# Public library allocation matrix. Measures Document.WritePDF and
# ImageDocument.WriteImage without starting a CLI or reading an HTML file from
# disk. The benchmark constructs the public documents before the timer; public
# validation, mapping, and the full renderer remain inside the timed calls.
bench-lib:
	go test . -run '^$$' -bench '^BenchmarkLibrary(PDF|Image)$$' \
		-benchmem -benchtime=10x -count=1

# The only Makefile entry for the process-level comparison against installed
# wkhtmltopdf. Requires `make build` and wkhtmltopdf on PATH. Writes
# testdata/golden/benchmarks/cli-compare*; a missing wkhtmltopdf is documented
# as a skipped Go test.
bench-cli-compare: build
	GOWKHTMLTOPDF_CLI_COMPARE=1 go test ./internal/convert \
		-run '^TestCompareWithWkhtmltopdfBinary$$' -count=1 -timeout 20m -v

clean:
	rm -rf testdata/golden/out
	rm -rf dist

# --- Python cgo bindings (opt-in only; never part of build/test/golden) -----

# Builds the C ABI shared library for the Python bindings. Requires an
# explicit CGO_ENABLED=1; the guard refuses to run otherwise so the default
# pure-Go targets can never drift into a cgo build. Emits dist/libgowkhtmltopdf.so
# plus the generated header, then smokes exports via nm (grep -c fails on 0).
c-shared:
	[ "$(CGO_ENABLED)" = "1" ] || { echo "refusing: c-shared needs CGO_ENABLED=1 (pure-Go default stays CGO_ENABLED=0)" >&2; exit 2; }
	mkdir -p dist && CGO_ENABLED=1 go build -buildmode=c-shared -ldflags "$(BINDINGS_VERSION_LDFLAGS) -s -w" -o dist/libgowkhtmltopdf.so ./bindings/c && file dist/libgowkhtmltopdf.so && nm -D dist/libgowkhtmltopdf.so | grep -c gowkhtmltopdf_

bindings-clean:
	rm -rf dist

# VERSION vs bindings/python/pyproject.toml alignment gate (Phase 46).
check-versions:
	bash scripts/check_versions.sh

# Convenience: rebuild the shared library, then run the Python stdlib unittest
# suite against it. Needs a working C toolchain (CGO_ENABLED=1) and python3;
# the tests load the dist/libgowkhtmltopdf.* artifact produced by c-shared.
python-binding-test:
	CGO_ENABLED=1 $(MAKE) c-shared
	python3 -m unittest discover -s bindings/python/tests -t . -v

# Public Python API library matrix. Same dirty report.html.tmpl fixture as
# `make bench-lib` (20 invoice rows per requested page). Template expansion
# happens before the timer; Document.pdf / ImageDocument.image stay inside
# the timed calls. Rebuilds the c-shared library first. Optional overrides:
# GOWKHTMLTOPDF_BENCH_SIZES=2,10,50 GOWKHTMLTOPDF_BENCH_RUNS=10
python-benchmarks:
	CGO_ENABLED=1 $(MAKE) c-shared
	PYTHONPATH=bindings/python/src \
		python3 bindings/python/tests/bench_library.py

# Python-API samples under output/python/:
#   generate.py            -> architecture-diagram.pdf (HTML file on disk)
#   generate_inline.py     -> invoice-inline.pdf (inline HTML + Document options)
#   generate_compliance.py -> pdf-{1.7,2.0}{,-compliance}/architecture-diagram.pdf
# Requires the c-shared library.
python-api:
	python3 testdata/golden/python_api/test_generate.py
	CGO_ENABLED=1 $(MAKE) c-shared
	PYTHONPATH=bindings/python/src \
		python3 testdata/golden/python_api/generate.py
	PYTHONPATH=bindings/python/src \
		python3 testdata/golden/python_api/generate_inline.py
	PYTHONPATH=bindings/python/src \
		python3 testdata/golden/python_api/generate_compliance.py
