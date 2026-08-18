.PHONY: test lint lint-frontend build fmt golden golden-update samples weasyprint clean claim-scan bench bench-cli-compare bench-external

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
	go run ./cmd/gowkhtmltopdf --enable-local-file-access \
		"testdata/golden/$$fixture" "$$output"; \
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
		go run ./cmd/gowkhtmltopdf --enable-local-file-access $$FONT_FLAGS $$HF_FLAGS "$$f" "output/$$name.pdf"; \
	done
	# Version / compliance smokes (unreleased 0.2.2): same two fixtures in four dirs.
	mkdir -p output/pdf-1.7 output/pdf-1.7-compliance output/pdf-2.0 output/pdf-2.0-compliance
	rm -f output/pdf-1.7/*.pdf output/pdf-1.7-compliance/*.pdf output/pdf-2.0/*.pdf output/pdf-2.0-compliance/*.pdf
	for f in testdata/golden/fixture-21-detailed-report.html testdata/golden/fixture-56-architecture-diagram.html; do \
		name=$$(basename "$$f" .html); \
		go run ./cmd/gowkhtmltopdf --pdf-version 1.7 --enable-local-file-access "$$f" "output/pdf-1.7/$$name.pdf"; \
		go run ./cmd/gowkhtmltopdf --pdf-profile a3a-ua1 --enable-local-file-access "$$f" "output/pdf-1.7-compliance/$$name.pdf"; \
		go run ./cmd/gowkhtmltopdf --pdf-version 2.0 --enable-local-file-access "$$f" "output/pdf-2.0/$$name.pdf"; \
		go run ./cmd/gowkhtmltopdf --pdf-profile a4-ua2 --enable-local-file-access "$$f" "output/pdf-2.0-compliance/$$name.pdf"; \
	done
	go run ./cmd/gowkhtmltopdf --enable-local-file-access --outline --outline-depth 2 --header-left "gowkhtmltopdf demo - [title]" --header-right "page [page]/[topage]" --footer-center "[section]" toc testdata/golden/fixture-16-invoice-with-css.html output/showcase-toc-hf-outline.pdf
	# Library-API architecture diagram → output/architecture-diagram.pdf only.
	# testdata/golden HTML (corpus fixture and api/ template) is not rewritten.
	go run ./testdata/golden/api
	go run ./cmd/gowkhtmltoimage --enable-local-file-access testdata/golden/fixture-01-simple-invoice.html output/fixture-01-simple-invoice.png
	go run ./examples/image --enable-local-file-access --width 1024 testdata/golden/fixture-21-detailed-report.html output/fixture-21-detailed-report.png
	# Live Wikipedia smoke (network, raw — no --simplify-dom). Soft-fail so offline/CI hosts still get fixture samples.
	# Operator recipe (not CSS fidelity): --use-system-fonts for IPA fallback; optional --zoom 2/3 densifies
	# author p{font-size:12pt} toward ~8pt; optional --print-link-underline / --simplify-dom-profile=mediawiki.
	go run ./cmd/gowkhtmltopdf --use-system-fonts --zoom 0.666667 \
		'https://en.wikipedia.org/wiki/Ana_de_Armas' \
		output/wiki-ana-de-armas.pdf \
		|| echo "warning: wiki-ana-de-armas.pdf live smoke skipped (network/fetch failed)"
	ls -la output/ | awk '{print $$5, $$9}' | tail -30

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

# Process-level CLI comparison against the installed WeasyPrint and Puppeteer
# engines (same generated report fixture and median-of-3 methodology as
# bench-cli-compare). The bench never runs engine commands itself: each engine
# is driven through its print script (scripts/weasyprint/print.sh,
# scripts/puppeteer/print.sh), so the measured command is exactly what that
# script runs. Engines that are not installed are skipped.
# Requires `make build`, /usr/bin/time and gs; Puppeteer additionally needs
# node + google-chrome and scripts/puppeteer/node_modules (npm ci --prefix
# scripts/puppeteer on first use), WeasyPrint needs the weasyprint CLI.
# Writes testdata/golden/benchmarks/{weasyprint,puppeteer}-compare.md and
# -results.csv. Default page matrix is 2/10/50/100; override with
# --sizes=2,5,10,20,50,100,200,250,500 or select one engine with
# --engines=weasyprint.
bench-external: build
	./scripts/bench-external.sh

# In-process Go benchmark matrix (generic + certified-islands, images).
bench:
	go test ./internal/convert -run '^$$' \
		-bench 'Benchmark(PDFPages|TemplatePages|WebFetchImage|ImageAssets)$$' \
		-benchmem -benchtime=1x -count=1

# Process-level CLI comparison against installed wkhtmltopdf. Requires
# `make build` and wkhtmltopdf on PATH. Writes testdata/golden/benchmarks/cli-compare*.
bench-cli-compare: build
	GOWKHTMLTOPDF_CLI_COMPARE=1 go test ./internal/convert \
		-run '^TestCompareWithWkhtmltopdfBinary$$' -count=1 -timeout 20m -v

clean:
	rm -rf testdata/golden/out
