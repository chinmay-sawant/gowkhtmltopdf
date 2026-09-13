// Package profiling is a manual heap and stack profiling harness for the two
// supported output features: PDF (Document) and image (ImageDocument).
//
// It is not part of the normal test suite. The harness only runs when
// GOWK_PROFILE_DIR is set, so make test stays fast.
//
// Usage:
//
//	GOWK_PROFILE_DIR=output/profiles/2026-09-10 \
//	GOWK_PROFILE_MODE=all \
//	go test ./internal/profiling -run TestProfileGoldenCorpus -count=1 -v
//
// For pprof symbolization, build the test binary once and run it directly:
//
//	go test -c -o output/profiles/2026-09-10/profiling.test ./internal/profiling
//	GOWK_PROFILE_DIR=output/profiles/2026-09-10 \
//	GOWK_PROFILE_MODE=pdf \
//	output/profiles/2026-09-10/profiling.test -test.run TestProfileGoldenCorpus
//
// Modes: pdf, image-png, image-jpeg, all. The fixture directory defaults to
// testdata/golden (or ../../testdata/golden under go test) and can be
// overridden with GOWK_PROFILE_FIXTURES.
package profiling
