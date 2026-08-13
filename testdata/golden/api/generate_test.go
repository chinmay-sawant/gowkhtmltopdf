package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTemplatePathsFromAPIDirectory(t *testing.T) {
	t.Parallel()

	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	input, output, repoRoot, err := resolveTemplatePaths()
	if err != nil {
		t.Fatal(err)
	}

	wantInput := filepath.Join(workingDir, inputName)
	wantOutput := filepath.Join(workingDir, outputName)
	// go test runs with package directory as CWD: .../testdata/golden/api
	wantRoot := filepath.Dir(filepath.Dir(filepath.Dir(workingDir)))

	if input != wantInput || output != wantOutput {
		t.Fatalf("resolved paths = %q, %q; want %q, %q", input, output, wantInput, wantOutput)
	}

	if repoRoot != wantRoot {
		t.Fatalf("repo root = %q; want %q", repoRoot, wantRoot)
	}
}

func TestUniquePathsDropsDuplicatesAndEmpty(t *testing.T) {
	t.Parallel()

	first := filepath.Join(t.TempDir(), "a.pdf")
	second := filepath.Join(t.TempDir(), "b.pdf")

	got := uniquePaths(first, "", first, second)
	if len(got) != 2 {
		t.Fatalf("uniquePaths = %v; want 2 entries", got)
	}

	if got[0] != first && filepath.Base(got[0]) != "a.pdf" {
		t.Fatalf("first path = %q; want %q", got[0], first)
	}
}

func TestSamePath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	a := filepath.Join(dir, "file.html")
	b := filepath.Join(dir, ".", "file.html")

	if !samePath(a, b) {
		t.Fatalf("samePath(%q, %q) = false; want true", a, b)
	}

	if samePath(a, filepath.Join(dir, "other.html")) {
		t.Fatal("samePath reported distinct files as equal")
	}
}
