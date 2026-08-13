.PHONY: test lint build fmt golden golden-update samples clean claim-scan bench bench-cli-compare

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

test:
	go test ./...

# Runs every linter enabled in .golangci.yml (enable-all). Installs the pinned
# binary into $(go env GOPATH)/bin when missing. Always builds with GOTOOLCHAIN=local
# so the binary matches go.mod's go1.26 toolchain.
lint:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not found; installing $(GOLANGCI_LINT_VERSION) with local Go toolchain..."; \
		GOTOOLCHAIN=local go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	}
	golangci-lint version
	golangci-lint run ./...

CLI_VERSION_LDFLAGS := -X gowkhtmltopdf/internal/cli.Version=$(shell cat VERSION)

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
# diagram (sample PDF + golden HTML mirror), image PNGs, and the optional live
# Wikipedia smoke (needs network; failure does not fail the target).
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
	go run ./cmd/gowkhtmltopdf --enable-local-file-access --outline --outline-depth 2 --header-left "gowkhtmltopdf demo - [title]" --header-right "page [page]/[topage]" --footer-center "[section]" toc testdata/golden/fixture-16-invoice-with-css.html output/showcase-toc-hf-outline.pdf
	# Library-API architecture diagram: golden PDF beside the template, sample
	# PDF in output/, and HTML mirror at testdata/golden/architecture-diagram.html.
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
