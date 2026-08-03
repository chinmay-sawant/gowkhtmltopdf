.PHONY: test lint build fmt golden-update clean

# Phase 00 scaffold: stdlib-only module. No external deps expected.

test:
	go test ./...

lint:
	go vet ./...
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

build:
	go build ./...

fmt:
	gofmt -w .

golden-update:
	@echo "golden-update: implemented in Phase 3 (PDF writer)"; true

clean:
	rm -rf testdata/golden/out
